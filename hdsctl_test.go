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

package hdsctl

import (
	"encoding/json"
	"fmt"
	"github.com/frnckdlprt/hdsctl/scpi"
	"github.com/google/gousb"
	"testing"
	"time"
)

func Test_Definitions(t *testing.T) {
	scd := scpi.NewHDSClient(scpi.NewHDSExecutor())
	fmt.Println(scd.GetCommandDefinitionByName("*IDN?"))

}

func assertNilErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("error unexpected: %v", err)
	}
}

func Test_basic(t *testing.T) {
	client := scpi.NewHDSClient(scpi.NewHDSExecutor())
	assertNilErr(t, client.Execute(`
	:CH2:DISP ON
`))
}

func Test_ramp(t *testing.T) {
	client := scpi.NewHDSClient(scpi.NewHDSExecutor())
	assertNilErr(t, client.Execute(`
	:CH1:DISP ON
	:CH2:DISP OFF
	:TRIG:SING:SOUR CH1
	:CH1:PROBe 1X
`))

	assertNilErr(t, client.Execute(`
	:HOR:SCAL 10ms
	:HOR:OFFS 0.0
	:CH1:SCAL 200mV
	:CH1:OFFS 0.0
	:FUNC RAMP
	:FUNC:FREQ 50
	:FUNC:OFFS 0
	:FUNC:HIGH 0.5
	:FUNC:LOW -0.5
	:TRIG:SING:EDG:LEV 0V
	:TRIG:SING:EDG RISE
`))

	time.Sleep(5 * time.Second)
	assertNilErr(t, client.Execute(`
	:HOR:SCAL 100us
	:HOR:OFFS 0.0
	:CH1:SCAL 500mV
	:CH1:OFFS -3.0
	:FUNC SINE
	:FUNC:FREQ 500
	:FUNC:OFFS 1
	:FUNC:HIGH 2
	:FUNC:LOW 0
	:TRIG:SING:EDG:LEV 1V
	:TRIG:SING:EDG FALL
`))
	time.Sleep(5 * time.Second)
	assertNilErr(t, client.Execute(`
	:HOR:SCAL 20ns
	:HOR:OFFS 0.0
	:CH1:SCAL 1V
	:CH1:OFFS -2.0
	:FUNC SINE
	:FUNC:FREQ 10000000
	:FUNC:OFFS 1
	:FUNC:HIGH 2
	:FUNC:LOW 0
	:TRIG:SING:EDG:LEV 1V
	:TRIG:SING:EDG RISE
`))
}

func Test_Wave(t *testing.T) {
	client := scpi.NewHDSClient(scpi.NewHDSExecutor())
	wav, err := client.GetWave(1)
	if err != nil {
		t.Errorf("failed getting wave: %v", err)
	}
	fmt.Printf(fmt.Sprint(wav))
}

func Test_Wave1(t *testing.T) {
	ctx := gousb.NewContext()
	defer ctx.Close()
	ctx.Debug(9)
	vendorID := gousb.ID(0x5345)
	productID := gousb.ID(0x1234)
	dev, err := ctx.OpenDeviceWithVIDPID(vendorID, productID)
	defer dev.Close()
	if err != nil {
		t.Errorf("failed to open device: %v", err)
	}
	intf, done, err := dev.DefaultInterface()
	if err != nil {
		t.Errorf("failed to retrieve default interface for device %v: %v", dev, err)
	}
	defer done()

	outep, err := intf.OutEndpoint(0x01)
	var cmd string
	var numBytes int
	var inep *gousb.InEndpoint
	var buff []byte
	var readBytes int

	//cmd := ":DATA:WAVE:SCREEN:HEAD?"
	//numBytes, err := outep.Write([]byte(cmd))
	//if numBytes != len(cmd) {
	//	t.Errorf("%s: only %d bytes written: %v", outep, numBytes, err)
	//}
	//if err != nil {
	//	t.Errorf("failed to write: %v", err)
	//}
	//
	//inep, err := intf.InEndpoint(0x81)
	//buff := make([]byte, 10000)
	//readBytes, err := inep.Read(buff)
	//if err != nil {
	//	t.Errorf("failed to read bytes: %v", err)
	//}
	//fmt.Println(string(buff[4:readBytes]))

	cmd = ":DATA:WAVE:SCREEN:CH1?"
	numBytes, err = outep.Write([]byte(cmd))
	if numBytes != len(cmd) {
		t.Errorf("%s: only %d bytes written: %v", outep, numBytes, err)
	}
	if err != nil {
		t.Errorf("failed to write: %v", err)
	}

	inep, err = intf.InEndpoint(0x81)
	buff = make([]byte, 10000)
	readBytes, err = inep.Read(buff)
	if err != nil {
		t.Errorf("failed to read bytes: %v", err)
	}
	result := ""
	for idx := 4; idx < readBytes; idx = idx + 2 {
		val := (int16(buff[idx+2]) << 8) + int16(buff[idx])
		result = result + fmt.Sprintf("%v\n", val)
	}

	fmt.Println(result)

}

func Test_JSON(t *testing.T) {
	data := map[string]interface{}{}
	data["field1"] = []string{"a", "b"}
	s, _ := json.Marshal(data)
	fmt.Println(string(s))
}

func Test_header(t *testing.T) {
	cache := map[string]scpi.CacheEntry{}
	header := `
{
    "TIMEBASE": {
        "SCALE": "20ns",
        "HOFFSET": 0
    },
    "SAMPLE": {
        "FULLSCREEN": 300,
        "SLOWMOVE": -1,
        "DATALEN": 300,
        "SAMPLERATE": "250MSa/s",
        "TYPE": "SAMPle",
        "DEPMEM": "4K"
    },
    "CHANNEL": [
        {
            "NAME": "CH1",
            "DISPLAY": "ON",
            "COUPLING": "DC",
            "PROBE": "10X",
            "SCALE": "50.0mV",
            "OFFSET": -25,
            "FREQUENCE": 10000000.00000
        },
        {
            "NAME": "CH2",
            "DISPLAY": "OFF",
            "COUPLING": "DC",
            "PROBE": "10X",
            "SCALE": "100mV",
            "OFFSET": 36,
            "FREQUENCE": 0.00000
        }
    ],
    "DATATYPE": "SCREEN",
    "RUNSTATUS": "TRIG",
    "IDN": "owon_v1.2",
    "MODEL": "HDS272S_1",
    "Trig": {
        "Mode": "SINGle",
        "Type": "Edge",
        "Items": {
            "Channel": "CH1",
            "Level": "1.00V",
            "Edge": "RISE",
            "Coupling": "DC",
            "Sweep": "AUTO"
        }
    }
}
`
	scpi.CacheHeader([]byte(header), cache)
	for k, v := range cache {
		fmt.Printf("%v = %v\n", k, string(v.Value))
	}
}
func Test_units(t *testing.T) {
	client := scpi.NewHDSClient(scpi.NewHDSExecutor())
	for _, cd := range client.Scheme {
		v, _ := client.GetString(cd.Name + "?")

		fmt.Printf("%s >> %v\n", cd.Name, v)
	}
	//fmt.Println(client.GetString(":CH1:SCALe?"))
	//fmt.Println(client.GetString(":MEASurement:CH1:PKPK?"))
	//fmt.Println(client.GetString(":MEASurement:CH1:VAMP?"))
}

func SetAndGet(t *testing.T, client scpi.Client, k, v string) {
	assertNilErr(t, client.Execute(fmt.Sprintf("%s %s", k, v)))
	res, err := client.GetString(fmt.Sprintf("%s?", k))
	assertNilErr(t, err)
	if res != v {
		t.Fatalf("mismatch for %s: %s != %s", k, v, res)
	}

}
func Test_set(t *testing.T) {
	client := scpi.NewHDSClient(scpi.NewHDSExecutor())
	tests := []struct {
		k string
		v string
	}{
		{":CH1:DISP", "ON"},
		{":CH2:DISP", "OFF"},
		{":TRIG:SING:SOUR", "CH1"},
		{":CH1:PROBe", "1X"},
	}
	for _, test := range tests {
		SetAndGet(t, client, test.k, test.v)
	}
}
