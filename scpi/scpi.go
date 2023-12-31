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
	"strings"
	"unicode"
)

type Executor interface {
	Execute(cmd Command) (result []byte, err error)
}

type CommandType int

const (
	ReadOnly CommandType = iota
	ReadWrite
)

type Command struct {
	Definition *CommandDefinition
	Arguments  []string
}

type CommandDefinition struct {
	Name       string
	Id         string
	ValueRange []string
	Type       CommandType
	Comment    string
}

type Client struct {
	Scheme        []*CommandDefinition
	commandById   map[string]*CommandDefinition
	commandByName map[string]*CommandDefinition
	Executor      Executor
}

func (client *Client) GetCommandDefinitionByName(name string) *CommandDefinition {
	k := name
	if strings.HasSuffix(k, "?") {
		k = strings.TrimSuffix(k, "?")
	}
	return client.commandByName[k]
}

func (client *Client) GetCommandDefinitionById(id string) *CommandDefinition {
	return client.commandById[id]
}

func (client *Client) AddCommandDefinition(keyword string, typ CommandType, valueRange []string, comment string) {
	for i := 1; i <= 2; i++ {
		k := strings.ReplaceAll(keyword, "<n>", fmt.Sprint(i))
		c := strings.ReplaceAll(comment, "<n>", fmt.Sprint(i))
		id := scpi2camel(k)
		cd := &CommandDefinition{Id: id, Name: k, ValueRange: valueRange, Comment: c}
		client.Scheme = append(client.Scheme, cd)
		client.commandById[id] = cd
		client.commandByName[k] = cd
		client.commandByName[scpi2short(k)] = cd
		if k == keyword {
			break
		}
	}
}

func (client *Client) Parse(c string) (cmd Command, err error) {
	cmdparts := strings.Split(strings.TrimSpace(c), " ")
	cmd = Command{}
	cname := cmdparts[0]
	cd := client.GetCommandDefinitionByName(cname)
	if cd == nil {
		return cmd, fmt.Errorf("unknown scpi command: %s", cname)
	}
	cmd.Definition = cd
	if len(cmdparts) == 2 {
		if strings.HasSuffix(cname, "?") {
			return cmd, fmt.Errorf("scpi command unexpected arguments: %s", c)
		}
		cmd.Arguments = strings.Split(cmdparts[1], ",")
	}
	return cmd, nil
}

func (client *Client) ParseAll(cmdList string) (cmds []Command, err error) {
	cmds = []Command{}
	for _, c := range strings.Split(cmdList, ";") {
		cmd, err := client.Parse(c)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

func (client *Client) Set(c string) (err error) {
	cmd, err := client.Parse(c)
	if err != nil {
		return err
	}
	_, err = client.Executor.Execute(cmd)
	return err
}

func (client *Client) GetBytes(qry string) (result []byte, err error) {
	cmd, err := client.Parse(qry)
	if err != nil {
		return nil, err
	}
	return client.Executor.Execute(cmd)
}

func (client *Client) GetString(qry string) (result string, err error) {
	res, err := client.GetBytes(qry)
	if err != nil {
		return "", fmt.Errorf("failed to GetBytes string: %w", err)
	}
	return strings.TrimSpace(string(res)), nil
}

func (client *Client) Execute(cmds string) (err error) {
	for _, line := range strings.Split(cmds, "\n") {
		for _, cmd := range strings.Split(line, ";") {
			cmd = strings.TrimSpace(cmd)
			if cmd == "" {
				continue
			}
			if strings.HasSuffix(cmd, "?") {
				out, err := client.GetString(cmd)
				if err != nil {
					return fmt.Errorf("failed to get string for %s: %w", cmd, err)
				}
				fmt.Println(out)
			} else {
				err = client.Set(cmd)
				if err != nil {
					return fmt.Errorf("failed to set %s: %v", cmd, err)
				}
			}
		}
	}
	return nil
}

func scpi2camel(s string) string {
	result := ""
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if unicode.IsLower(runes[i]) || string(runes[i]) == ":" || string(runes[i]) == "*" {
			continue
		}
		if i > 1 && string(runes[i-1]) == ":" {
			result += string(unicode.ToUpper(runes[i]))
		} else {
			result += string(unicode.ToLower(runes[i]))
		}
	}
	return result
}

func scpi2short(s string) string {
	result := ""
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if unicode.IsLower(runes[i]) {
			continue
		}
		result += string(runes[i])
	}
	return result
}
