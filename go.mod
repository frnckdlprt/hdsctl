module github.com/frnckdlprt/hdsctl

go 1.20

require (
	github.com/google/gousb v1.1.2
	github.com/gorilla/websocket v1.5.1
)

require golang.org/x/net v0.17.0 // indirect

// a fork of google/gousb that suppresses logging of some interruption error (https://github.com/google/gousb/issues/87)
// not doing the replace for now, as it breaks "go install..."
replace github.com/google/gousb => github.com/frnckdlprt/gousb v1.1.3-0.20231230224731-2f7c8c945b28
