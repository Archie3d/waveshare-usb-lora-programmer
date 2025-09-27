# Waveshare custom programmer CLI

This is a CLI programmer that flashed the [Waveshare USB LoRa](https://github.com/Archie3d/waveshare-usb-lora) that has [the custom bootloader](https://github.com/Archie3d/waveshare-usb-lora-bootloader) installed.

This can be used to flash [the custom LoRa firmware](https://github.com/Archie3d/waveshare-usb-lora-firmware) via USB serial interface (instead of using ST-LINK).

## Build
Install [Go compiler](https://go.dev/).

```bash
go build -o . ./...
```

This will produce `wsprog` executable.

## Usage
Assuming Waveshare LoRa device appears on USB-serial port `COM4`:
```bash
wsprog -p COM4 firmware.bin
```
