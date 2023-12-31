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

package web

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/frnckdlprt/hdsctl"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func apiEndpoint(hds *hdsctl.HDS, w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	cname := parts[len(parts)-1]
	v, err := hds.GetField(cname)
	if err != nil {
		log.Println(err)
	}
	w.Write([]byte(v))
}

func wsEndpoint(hds *hdsctl.HDS, w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	mx := sync.Mutex{}

	go func() {

		lastdata := map[string]interface{}{}
		for {
			time.Sleep(150 * time.Millisecond)
			data := map[string]interface{}{}
			hds.GetField("datWavScrHead")
			for i := 1; i <= 2; i++ {
				chDisp, _ := hds.GetField(fmt.Sprintf("ch%vDisp", i))
				if chDisp == "ON" {
					wav, _ := hds.Client.GetWave(i)
					vals := ""
					for _, w := range wav {
						vals += fmt.Sprintf("%v ", int8(w))
					}
					data[fmt.Sprintf("wave%v", i)] = vals
				}
			}

			fields := []string{
				"ch1Disp", "ch1Scal", "ch1Offs", "ch1Prob", "ch1Coup",
				"ch2Disp", "ch2Scal", "ch2Offs", "ch2Prob", "ch2Coup",
				"horScal", "horOffs", "acqMod", "acqDepm",
				"func", "funcOffs", "chan", "funcFreq", "funcAmpl", "funcLow", "funcHigh",
				"trigSingSour", "trigSingCoup", "trigSingEdg", "trigSingSwe", "trigSingEdgLev",
				"dmmMeas"}
			for _, f := range fields {
				cd := hds.Client.GetCommandDefinitionById(f)
				if cd == nil {
					log.Printf("unknown field: %s\n", f)
					continue
				}
				data[f], _ = hds.GetField(f)

				if cd.ValueRange != nil && !strings.HasPrefix(f, "dmm") {
					data[fmt.Sprintf("%s.range", f)] = cd.ValueRange
				}
			}
			dataUpdate := map[string]interface{}{}
			for k, v := range data {
				if k == "wave1" || k == "wave2" {
					dataUpdate[k] = v
					continue
				}
				if fmt.Sprintf("%v", lastdata[k]) != fmt.Sprintf("%v", v) {
					dataUpdate[k] = v
				}
			}
			if len(dataUpdate) > 0 {
				msg, err := json.Marshal(dataUpdate)
				if err != nil {
					log.Println(err)
				}
				mx.Lock()
				ws.WriteMessage(websocket.TextMessage, msg)
				mx.Unlock()
			}
			lastdata = data
		}
	}()
	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		//log.Println("Received ", p)
		parts := strings.Split(string(p), ":")
		param := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		err = hds.SetField(param, value)
		if err != nil {
			log.Println(err)
		}
		realv, err := hds.GetField(param)
		if err != nil {
			log.Println(err)
		}
		//log.Printf("ui: %v real: %v\n", value, realv)

		data := map[string]interface{}{}
		data[param] = realv
		msg, err := json.Marshal(data)
		if err != nil {
			log.Println(err)
		}
		mx.Lock()
		if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println(err)
			mx.Unlock()
			return
		}
		mx.Unlock()
	}
}

func homePage(hds *hdsctl.HDS, w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/hdsctl.css" {
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(cssTxt))
		return
	}
	if r.URL.Path == "/hdsctl.js" {
		w.Header().Set("Content-Type", "text/javascript")
		data := map[string]any{}
		data["wsEndpoint"] = "ws://" + r.Host + "/ws"
		data["hds"] = hds
		err := jsTemplate.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
		return
	}
	w.Write([]byte(htmlTxt))
}

type Server struct {
	hds *hdsctl.HDS
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		apiEndpoint(s.hds, w, r)
	} else if strings.HasPrefix(r.URL.Path, "/ws") {
		wsEndpoint(s.hds, w, r)
	} else {
		homePage(s.hds, w, r)
	}
}

func StartServer(hds *hdsctl.HDS) {
	server := &Server{hds: hds}
	log.Fatal(http.ListenAndServe(":8080", server))
}

//go:embed index.html
var htmlTxt string

//go:embed hdsctl.js
var jsTemplateTxt string

var jsTemplate = template.Must(template.New("").Parse(jsTemplateTxt))

//go:embed hdsctl.css
var cssTxt string
