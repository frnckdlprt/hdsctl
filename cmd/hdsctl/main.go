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

package main

import (
	"fmt"
	"github.com/frnckdlprt/hdsctl"
	"github.com/frnckdlprt/hdsctl/scpi"
	"github.com/frnckdlprt/hdsctl/version"
	"github.com/frnckdlprt/hdsctl/web"
	"os"
	"strings"
)

func main() {
	executor := scpi.NewHDSExecutor()
	defer executor.Close()
	//executor := scpi.NewMockExecutor()
	hds := hdsctl.NewHDS(scpi.NewHDSClient(executor))
	if os.Args[1] == "version" {
		fmt.Printf("hdsctl version %s (%s)\n", version.Version, version.BuildDate)
		return
	}
	if os.Args[1] == "serve" {
		web.StartServer(hds)
		return
	}
	hds.Client.Execute(strings.Join(os.Args[1:], " "))
}
