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

//
///*
//#cgo pkg-config: libusb-1.0
//#include <libusb.h>
//void hdsctl_libusb_set_debug(libusb_context *ctx, int level) {
//	libusb_set_option(ctx, LIBUSB_OPTION_LOG_LEVEL, level);
//}
//*/
//import "C"
//
//func main() {
//	var ctx *C.libusb_context
//	C.libusb_init(&ctx)
//	C.hdsctl_libusb_set_debug(ctx, C.LIBUSB_LOG_LEVEL_INFO)
//	//C.hdsctl_libusb_set_debug(ctx, C.LIBUSB_LOG_LEVEL_DEBUG)
//	handle := C.libusb_open_device_with_vid_pid(ctx, 0x5345, 0x1234)
//	if handle == nil {
//		log.Fatal("device not found")
//	}
//	C.libusb_claim_interface(handle, 0)
//	msg := "*IDN?"
//	transferred := C.int(0)
//	C.libusb_bulk_transfer(handle, 0x01, (*C.uchar)(unsafe.Pointer(C.CString(msg))), C.int(len(msg)), &transferred, 1000)
//	fmt.Printf("write transferred=%v\n", transferred)
//	buff := make([]byte, 10000)
//	C.libusb_bulk_transfer(handle, 0x81, (*C.uchar)(unsafe.Pointer(&buff[0])), C.int(len(buff)), &transferred, 1000)
//	fmt.Printf("read transferred=%v\n", transferred)
//	fmt.Println(string(buff[:transferred]))
//	C.libusb_close(handle)
//	C.libusb_exit(nil)
//	log.Println("done")
//}

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
