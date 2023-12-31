# HDSCTL: CLI for Owon HDS200 series oscilloscope

A cross platform golang CLI to control the Owon HDS200 series oscilloscope with SCPI commands, with also a minimal web frontend.

## Installation

Prerequisites: 
- libusb (https://libusb.info/) 
- golang (https://go.dev/dl/)

Then
`go install github.com/frnckdlprt/hdsctl/cmd/hdsctl@latest`

## Usage

- Run SCPI commands with for example `hdsctl ":HOR:SCAL 50ns;CH1:SCAL 1.00V"` or `hdsctl :DATa:WAVe:SCReen:HEAD? | jq`
- Run a minimal web interface with `hdsctl serve`, accessible on `http://localhost:8080`

## Limitations / Known issues

- the web UI has a very low refresh rate, about once per second (it could be pushed a bit higher, but then very soon the scope itself becomes less responsive)
- may require root privilege, alternatively on fedora I have been using `sudo chown $USER:$USER /dev/bus/usb/$(lsusb | grep PDS6062T | awk '{print $2 "/" substr($4,1,length($4)-1)}')` to avoid permission issues
- `hdsctl serve` produces a lot of errors such as `2023/12/30 17:45:18 handle_events: error: libusb: interrupted [code -10]`, this could be suppressed with a fork of gousb but the "go mod replace" would break "go install" here
