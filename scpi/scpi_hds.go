/*
Copyright 2023 frnckdlprt.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scpi

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unsafe"
)

/*
#cgo pkg-config: libusb-1.0
#include <libusb.h>
void hdsctl_libusb_set_debug(libusb_context *ctx, int level) {
	libusb_set_option(ctx, LIBUSB_OPTION_LOG_LEVEL, level);
}
*/
import "C"

const vendorID = 0x5345
const productID = 0x1234
const inEndpoint = 0x81
const outEndpoint = 0x01
const throttleDelay = 10 * time.Millisecond
const discardReadTimeout = 10 * time.Millisecond
const cacheTimeout = 500 * time.Millisecond

type HDSExecutor struct {
	usbCtx    *C.libusb_context
	usbDev    *C.libusb_device_handle
	lastCmdTs time.Time
	cache     map[string]CacheEntry
	execSync  sync.Mutex
}

type CacheEntry struct {
	Value     []byte
	Timestamp time.Time
}

func NewHDSExecutor() (h *HDSExecutor) {
	h = &HDSExecutor{}
	C.libusb_init(&h.usbCtx)
	C.hdsctl_libusb_set_debug(h.usbCtx, C.LIBUSB_LOG_LEVEL_DEBUG)
	h.usbDev = C.libusb_open_device_with_vid_pid(h.usbCtx, vendorID, productID)
	h.cache = map[string]CacheEntry{}
	h.lastCmdTs = time.Now()
	idn, err := h.Execute(Command{Definition: &CommandDefinition{Name: "*IDN"}})
	if err != nil {
		log.Fatalf("failed to retrieve IDN: %s", err)
	}
	if !strings.HasPrefix(strings.ToUpper(string(idn)), "OWON,HDS2") {
		log.Fatalf("unsupported device: %s", idn)
	}
	return h
}

func (hds *HDSExecutor) Close() {
	if hds.usbDev != nil {
		defer C.libusb_close(hds.usbDev)
	}
	C.libusb_exit(nil)
}

// wait for a minimum of throttle delay between usb commands
func (hds *HDSExecutor) throttle() {
	dt := hds.lastCmdTs.Add(throttleDelay).Sub(time.Now())
	if dt > 0 {
		time.Sleep(dt)
	}
	hds.lastCmdTs = time.Now()
}

// flush any previous responses
func (hds *HDSExecutor) discardReads() {
	for {

		buff := make([]byte, 10000)
		transferred := C.int(0)
		C.libusb_bulk_transfer(hds.usbDev, inEndpoint, (*C.uchar)(unsafe.Pointer(&buff[0])), C.int(len(buff)), &transferred, 1000)
		if transferred == C.int(0) {
			return
		}
	}
}

func (hds *HDSExecutor) Execute(cmd Command) (result []byte, err error) {
	hds.execSync.Lock()
	defer hds.execSync.Unlock()
	var c string
	if len(cmd.Arguments) == 0 {
		if ce, ok := hds.cache[cmd.Definition.Name]; ok {
			if !time.Now().After(ce.Timestamp.Add(cacheTimeout)) {
				return ce.Value, nil
			}
		}
		c = fmt.Sprintf("%s?", cmd.Definition.Name)
	} else {
		c = fmt.Sprintf("%s %s", cmd.Definition.Name, cmd.Arguments[0])
	}
	hds.throttle()
	//hds.discardReads()
	transferred := C.int(0)
	C.libusb_bulk_transfer(hds.usbDev, outEndpoint, (*C.uchar)(unsafe.Pointer(C.CString(c))), C.int(len(c)), &transferred, 1000)
	if transferred != C.int(len(c)) {
		return nil, fmt.Errorf("only %d bytes written: %w", transferred, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to write: %w", err)
	}
	if len(cmd.Arguments) == 0 {
		buff := make([]byte, 10000)
		C.libusb_bulk_transfer(hds.usbDev, inEndpoint, (*C.uchar)(unsafe.Pointer(&buff[0])), C.int(len(buff)), &transferred, 1000)
		result = buff[:transferred]
		if strings.HasPrefix(c, ":DATa:WAVe:SCReen:") && transferred < 100 {
			return nil, fmt.Errorf("unexpected length %v for %s", transferred, c)
		}
		hds.cache[cmd.Definition.Name] = CacheEntry{Value: result, Timestamp: time.Now()}
		if cmd.Definition.Name == ":DATa:WAVe:SCReen:HEAD" {
			result = result[4:]
			CacheHeader(result, hds.cache)
		}
		return result, nil
	}
	return nil, nil
}

func NewHDSClient(executor Executor) Client {
	client := Client{
		Scheme:        []*CommandDefinition{},
		commandByName: map[string]*CommandDefinition{},
		commandById:   map[string]*CommandDefinition{},
		Executor:      executor,
	}
	client.AddCommandDefinition("*IDN", ReadOnly, nil, "the ID character string of the instrument")
	// when scale is changed the offset automatically changes
	client.AddCommandDefinition(":HORizontal:SCALe", ReadWrite, []string{"5.0ns", "10ns", "20ns", "50ns", "100ns", "200ns", "500ns", "1.0us", "2.0us", "5.0us", "10us", "20us", "50us", "100us", "200us", "500us", "1.0ms", "2.0ms", "5.0ms", "10ms", "20ms", "50ms", "100ms", "200ms", "500ms", "1.0s", "2.0s", "5.0s", "10s", "20s", "50s", "100s", "200s", "500s", "1000s"}, "the scale of the main time base")
	// offset unit is division, the screen shows +6 / -6 horizontal divisions, but offset can be out of screen
	client.AddCommandDefinition(":HORizontal:OFFSet", ReadWrite, nil, "the horizontal offset of the time base")
	client.AddCommandDefinition(":ACQuire:MODe", ReadWrite, []string{"SAMPle", "PEAK"}, "the acquisition mode of the oscilloscope")
	client.AddCommandDefinition(":ACQuire:DEPMem", ReadWrite, []string{"4K", "8K"}, "the number of waveform points that the oscilloscope can store in a single trigger sample")
	client.AddCommandDefinition(":CH<n>:DISPlay", ReadWrite, []string{"ON", "OFF"}, "the display status of the channel")
	client.AddCommandDefinition(":CH<n>:COUPling", ReadWrite, []string{"AC", "DC", "GND"}, "the coupling mode of the channel")
	client.AddCommandDefinition(":CH<n>:PROBe", ReadWrite, []string{"1X", "10X", "100X", "1000X"}, "the attenuation ratio of the probe")
	// with 1X probe range is 10.0mV to 10V, for 10X it is 100mV to 100V, etc...
	client.AddCommandDefinition(":CH<n>:SCALe", ReadWrite, []string{"10.0mV", "20.0mV", "50.0mV", "100mV", "200mV", "500mV", "1.00V", "2.00V", "5.00V", "10.0V", "2.00V", "5.00V", "10.0V", "20.0V", "50.0V", "100V", "200V", "500V", "1.00kV", "2.00kV", "5.00kV", "10.0kV"}, "the vertical scale")
	client.AddCommandDefinition(":CH<n>:OFFSet", ReadWrite, nil, "the vertical offset")
	client.AddCommandDefinition(":DATa:WAVe:SCReen:HEAD", ReadOnly, nil, "the file header of the screen waveform data file")
	client.AddCommandDefinition(":DATa:WAVe:SCReen:CH<n>", ReadOnly, nil, "the screen waveform data of the specified channel")
	client.AddCommandDefinition(":TRIGger:STATus", ReadOnly, nil, "the trigger status")
	client.AddCommandDefinition(":TRIGger:SINGle:SOURce", ReadWrite, []string{"CH1", "CH2"}, "the trigger source")
	client.AddCommandDefinition(":TRIGger:SINGle:COUPling", ReadWrite, []string{"AC", "DC"}, "the trigger coupling")
	client.AddCommandDefinition(":TRIGger:SINGle:EDGe", ReadWrite, []string{"RISE", "FALL"}, "the slope of the trigger")
	client.AddCommandDefinition(":TRIGger:SINGle:EDGe:LEVel", ReadWrite, nil, "the trigger level")
	client.AddCommandDefinition(":TRIGger:SINGle:SWEep", ReadWrite, []string{"AUTO", "NORMal", "SINGle"}, "the trigger sweep mode")
	client.AddCommandDefinition(":MEASurement:DISPlay", ReadWrite, []string{"ON", "OFF"}, "the display status of measurements")
	client.AddCommandDefinition(":MEASurement:CH<n>:MAX", ReadOnly, nil, "the measured MAX for channel <n>")
	client.AddCommandDefinition(":MEASurement:CH<n>:MIN", ReadOnly, nil, "the measured MIN for channel <n>")
	client.AddCommandDefinition(":MEASurement:CH<n>:PKPK", ReadOnly, nil, "the measured Peak-to-Peak for channel <n>")
	client.AddCommandDefinition(":MEASurement:CH<n>:VAMP", ReadOnly, nil, "the measured vertical amplitude for channel <n>")
	client.AddCommandDefinition(":MEASurement:CH<n>:AVERage", ReadOnly, nil, "the measured average for channel <n>")
	client.AddCommandDefinition(":MEASurement:CH<n>:PERiod", ReadWrite, nil, "the measured period for channel <n>")
	client.AddCommandDefinition(":MEASurement:CH<n>:FREQuency", ReadWrite, nil, "the measured frequency for channel <n>")
	client.AddCommandDefinition(":MEASurement:CH<n>:MAX", ReadWrite, nil, "GetCommandDefinitionByName the value of the channel measurement item.")
	client.AddCommandDefinition(":FUNCtion", ReadWrite, []string{"SINE", "SQUare", "RAMP", "PULSe", "AmpALT", "AttALT", "StairDn", "StairUD", "StairUp", "Besselj", "Bessely", "Sinc"}, "the form of the function generated")
	client.AddCommandDefinition(":FUNCtion:FREQuency", ReadWrite, nil, "the output frequency of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:PERiod", ReadWrite, nil, "the output period of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:AMPLitude", ReadWrite, nil, "the amplitude Peak-to-Peak of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:OFFSet", ReadWrite, nil, "the offset of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:HIGHt", ReadWrite, nil, "the high level of the arbitrary function generator.")
	client.AddCommandDefinition(":FUNCtion:LOW", ReadWrite, nil, "the low level of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:SYMMetry", ReadWrite, nil, "the symmetry of ramp waveform as a percentage of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:WIDTh", ReadWrite, nil, "the pulse width of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:RISing", ReadWrite, nil, "the rising time of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:FALing", ReadWrite, nil, "the falling time for the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:DTYCycle", ReadWrite, nil, "the duty cycle of the pulse waveform as a percentage of the arbitrary function generator")
	client.AddCommandDefinition(":FUNCtion:LOAD", ReadWrite, []string{"ON", "OFF"}, "")
	client.AddCommandDefinition(":CHANnel", ReadWrite, []string{"ON", "OFF"}, "the status of the arbitrary function generator")
	client.AddCommandDefinition(":DMM:CONFigure", ReadWrite, []string{"R", "RS", "DIODE", "C"}, "the present measurement function of the multimeter")
	client.AddCommandDefinition(":DMM:CONFigure:VOLTage", ReadWrite, []string{"AC", "DC"}, "the voltage measurement type of the multimeter")
	client.AddCommandDefinition(":DMM:CONFigure:CURRent", ReadWrite, []string{"AC", "DC"}, "the current measurement type of the multimeter")
	client.AddCommandDefinition(":DMM:REL", ReadWrite, []string{"ON", "OFF"}, "the relative status of the multimeter")
	client.AddCommandDefinition(":DMM:RANGE", ReadWrite, []string{"ON", "OFF", "mV", "V"}, "the range of the multimeter")
	client.AddCommandDefinition(":DMM:AUTO", ReadWrite, []string{"ON"}, "the auto range status of the multimeter")
	client.AddCommandDefinition(":DMM:MEAS", ReadOnly, nil, "the measured value of the multimeter")
	return client
}

func (client *Client) GetWave(ch int) (result []byte, err error) {
	if ch != 1 && ch != 2 {
		return nil, fmt.Errorf("invalid channel number: %v", ch)
	}
	res, err := client.GetBytes(fmt.Sprintf(":DATa:WAVe:SCReen:CH%v?", ch))
	if err != nil {
		return nil, fmt.Errorf("failed to GetBytes wave: %w", err)
	}
	if len(res) > 4 {
		return res[4:], nil
	}
	return []byte{}, nil
}

func CacheHeader(header []byte, cache map[string]CacheEntry) {
	raw := map[string]interface{}{}
	json.Unmarshal(header, &raw)
	ts := time.Now()

	// TODO: sometimes we get a bogus HEADER data
	if raw["TIMEBASE"] == nil {
		log.Println("failed to process header")
		return
	}
	timeBase := raw["TIMEBASE"].(map[string]interface{})
	cache[":HORizontal:SCALe"] = CacheEntry{Value: []byte(timeBase["SCALE"].(string)), Timestamp: ts}
	cache[":HORizontal:OFFSet"] = CacheEntry{Value: []byte(fmt.Sprintf("%f", timeBase["HOFFSET"].(float64))), Timestamp: ts}
	sample := raw["SAMPLE"].(map[string]interface{})
	cache[":ACQuire:MODe"] = CacheEntry{Value: []byte(sample["TYPE"].(string)), Timestamp: ts}
	cache[":ACQuire:DEPMem"] = CacheEntry{Value: []byte(sample["DEPMEM"].(string)), Timestamp: ts}
	channel := raw["CHANNEL"].([]interface{})
	channel1 := channel[0].(map[string]interface{})
	cache[":CH1:DISPlay"] = CacheEntry{Value: []byte(channel1["DISPLAY"].(string)), Timestamp: ts}
	cache[":CH1:COUPling"] = CacheEntry{Value: []byte(channel1["COUPLING"].(string)), Timestamp: ts}
	cache[":CH1:PROBe"] = CacheEntry{Value: []byte(channel1["PROBE"].(string)), Timestamp: ts}
	cache[":CH1:SCALe"] = CacheEntry{Value: []byte(channel1["SCALE"].(string)), Timestamp: ts}
	cache[":CH1:OFFSet"] = CacheEntry{Value: []byte(fmt.Sprintf("%.2f", channel1["OFFSET"].(float64)/25)), Timestamp: ts}
	//fmt.Printf("CH1 OFFSET from HEADER: %v / %v\n", channel1["OFFSET"], channel1["OFFSET"].(float64)/25)
	channel2 := channel[1].(map[string]interface{})
	cache[":CH2:DISPlay"] = CacheEntry{Value: []byte(channel2["DISPLAY"].(string)), Timestamp: ts}
	cache[":CH2:COUPling"] = CacheEntry{Value: []byte(channel2["COUPLING"].(string)), Timestamp: ts}
	cache[":CH2:PROBe"] = CacheEntry{Value: []byte(channel2["PROBE"].(string)), Timestamp: ts}
	cache[":CH2:SCALe"] = CacheEntry{Value: []byte(channel2["SCALE"].(string)), Timestamp: ts}
	cache[":CH2:OFFSet"] = CacheEntry{Value: []byte(fmt.Sprintf("%.2f", channel2["OFFSET"].(float64)/25)), Timestamp: ts}
	trig := raw["Trig"].(map[string]interface{})
	trigItems := trig["Items"].(map[string]interface{})
	cache[":TRIGger:SINGle:SOURce"] = CacheEntry{Value: []byte(trigItems["Channel"].(string)), Timestamp: ts}
	cache[":TRIGger:SINGle:COUPling"] = CacheEntry{Value: []byte(trigItems["Coupling"].(string)), Timestamp: ts}
	cache[":TRIGger:SINGle:EDGe"] = CacheEntry{Value: []byte(trigItems["Edge"].(string)), Timestamp: ts}
	cache[":TRIGger:SINGle:EDGe:LEVel"] = CacheEntry{Value: []byte(trigItems["Level"].(string)), Timestamp: ts}
	cache[":TRIGger:SINGle:SWEep"] = CacheEntry{Value: []byte(trigItems["Sweep"].(string)), Timestamp: ts}
}
