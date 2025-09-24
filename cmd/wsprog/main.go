package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.bug.st/serial"
)

const (
	BAUD = 115200

	CHUNK_SIZE = 1024
)

const (
	X_START = uint8(0x10)
	X_END   = uint8(0x20)
	X_ACK   = uint8(0x30)
	X_NACK  = uint8(0x40)
	X_NCRC  = uint8(0x50)
)

type CrcError struct {
	CRC         uint16
	ExpectedCRC uint16
}

func (c *CrcError) Error() string {
	return fmt.Sprintf("crc error %X (expected %X)", c.CRC, c.ExpectedCRC)
}

func crc16(crc0 uint16, data []byte) uint16 {
	crc := crc0
	for _, b := range data {
		a := (crc >> 8) ^ uint16(b)
		crc = (a << 2) ^ (a << 1) ^ a ^ (crc << 8)
	}
	return crc
}

func programChunk(port serial.Port, chunk []byte) error {
	if len(chunk) != CHUNK_SIZE {
		return fmt.Errorf("invalid chunk size %d", len(chunk))
	}

	crc := crc16(0, chunk)
	crc_bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(crc_bytes, crc)

	b := make([]byte, 1)
	b[0] = X_START

	_, err := port.Write(b)

	if err != nil {
		return err
	}

	// Write CRC
	n, err := port.Write(crc_bytes)
	if err != nil {
		return nil
	}
	if n != len(crc_bytes) {
		return fmt.Errorf("failed to write %d bytes to serial port (%d written)", len(crc_bytes), n)
	}

	// Write chunk
	n, err = port.Write(chunk)

	if err != nil {
		return err
	}

	if n != len(chunk) {
		return fmt.Errorf("failed to write %d bytes to serial port (%d written)", len(chunk), n)
	}

	// Read response
	res := make([]byte, 1)
	n, err = port.Read(res)

	if err != nil {
		return err
	}

	if n != len(res) {
		return fmt.Errorf("failed to read %d bytes from serial port (%d read)", len(res), n)
	}

	if res[0] == X_NCRC {
		received_crc_bytes := make([]byte, 2)
		_, err := port.Read(received_crc_bytes)
		if err != nil {
			return err
		}

		return &CrcError{
			CRC:         binary.LittleEndian.Uint16(received_crc_bytes),
			ExpectedCRC: crc,
		}
	}

	if res[0] == X_NACK {
		return fmt.Errorf("programming failed")
	}

	if res[0] != X_ACK {
		return fmt.Errorf("invalid response received from the device: %d", int(res[0]))
	}

	return nil
}

func program(port_name string, firmware_file string) error {
	mode := &serial.Mode{
		BaudRate: BAUD,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(port_name, mode)

	if err != nil {
		return err
	}

	defer port.Close()

	port.SetReadTimeout(5 * time.Second)

	// Get firmware file size
	fwFileInfo, err := os.Stat(firmware_file)

	if err != nil {
		return err
	}

	fwFileSize := fwFileInfo.Size()
	var bytesWritten int64 = 0

	// Open firmware file
	fw, err := os.Open(firmware_file)

	if err != nil {
		return err
	}

	defer fw.Close()

	reader := bufio.NewReader(fw)

	fmt.Printf("...\r")

	for bytesWritten < fwFileSize {
		var chunk []byte = make([]byte, CHUNK_SIZE)

		bytesRead, err := reader.Read(chunk)

		if err != nil {
			return err
		}

		err = programChunk(port, chunk)

		if err != nil {
			return err
		}

		bytesWritten += int64(bytesRead)

		fmt.Printf("[%d%%] %d bytes written\r", int(bytesWritten*100/fwFileSize), bytesWritten)
	}

	fmt.Println()

	// Send END byte
	end_byte := make([]byte, 1)
	end_byte[0] = X_END
	_, err = port.Write(end_byte)

	if err != nil {
		return err
	}

	fmt.Println("Done.")

	return nil
}

func main() {
	portPtr := flag.String("p", "", "Serial port name")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -p <serial port> <firmware file>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	err := program(*portPtr, flag.Arg(0))

	if err != nil {
		log.Fatal(err)
	}
}
