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
	"fmt"
	"github.com/frnckdlprt/hdsctl/scpi"
)

type HDS struct {
	Client scpi.Client
	Data   *HDSData
}

func NewHDS(client scpi.Client) (result *HDS) {
	return &HDS{Client: client, Data: NewHDSData(client)}
}

func (hds *HDS) SetField(k, value string) (err error) {
	for _, f := range hds.Data.Fields {
		if f.Id == k {
			return hds.Client.Set(fmt.Sprintf("%s %s", f.SCPI, value))
		}
	}
	return fmt.Errorf("invalid field: %s", k)
}

func (hds *HDS) GetField(k string) (v string, err error) {
	for _, f := range hds.Data.Fields {
		if f.Id == k {
			v, err := hds.Client.GetString(fmt.Sprintf("%s?", f.SCPI))
			if err != nil {
				return "", fmt.Errorf("failed to get %s: %w", v, err)
			}
			return v, nil
		}
	}
	return "", fmt.Errorf("invalid field: %s", k)
}

type HDSField struct {
	Id   string
	SCPI string
	//Range     func(scope *HDSData) []string
	Range     []string
	Validator func(value interface{}, scope HDSData) bool
	Value     interface{}
}

type HDSData struct {
	Fields []*HDSField
}

func NewHDSData(client scpi.Client) *HDSData {
	fields := []*HDSField{}
	for _, cd := range client.Scheme {
		f := &HDSField{
			Id:    cd.Id,
			SCPI:  cd.Name,
			Range: cd.ValueRange,
		}
		//if cd.ValueRange != nil {
		//	a := cd.ValueRange
		//	f.Range = func(scope *HDSData) []string {
		//		return scope.Get
		//	}
		//}
		fields = append(fields, f)
	}
	return &HDSData{Fields: fields}
}
