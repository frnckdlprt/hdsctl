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
	"fmt"
	"math"
	"strconv"
)

type MockExecutor struct {
	values map[string][]byte
}

func NewMockExecutor() *MockExecutor {
	result := &MockExecutor{values: map[string][]byte{}}
	result.values[":CH1:OFFSet"] = []byte("0")
	result.values[":CH1:DISPlay"] = []byte("ON")
	result.values[":CH2:DISPlay"] = []byte("OFF")
	return result
}

func (me *MockExecutor) Execute(cmd Command) (result []byte, err error) {
	if cmd.Definition.Name == ":DATa:WAVe:SCReen:CH1" {
		result = []byte{0, 0, 0, 0}
		offsb, _ := me.values[":CH1:OFFSet"]
		var offs int64
		if s, err := strconv.ParseFloat(string(offsb), 32); err == nil {
			offs = int64(s * 25.0)
		}
		for i := 0; i < 300; i++ {
			j := offs + int64(100*math.Sin(float64(i)/50))
			if j <= -127 {
				j = -127
			}
			if j >= 127 {
				j = 127
			}
			result = append(result, byte(j))
		}
		return result, nil
	}
	//fmt.Printf("mock exec: %v %v\n", cmd.Definition.Name, cmd.Arguments)
	if len(cmd.Arguments) == 0 {
		v, ok := me.values[cmd.Definition.Name]
		if !ok {
			return nil, fmt.Errorf("no value for: %v", cmd.Definition)
		}
		return v, nil
	}
	me.values[cmd.Definition.Name] = []byte(cmd.Arguments[0])
	return nil, nil
}
