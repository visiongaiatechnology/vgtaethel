module go-aethel

go 1.26.5

require (
	github.com/dop251/goja v0.0.0-20240220182346-e401ed450204
	github.com/emersion/go-imap v1.2.1
	github.com/emersion/go-message v0.15.0
	github.com/k2-fsa/sherpa-onnx-go v1.13.3
	github.com/wailsapp/wails/v2 v2.12.0
	golang.org/x/net v0.55.0
	golang.org/x/sys v0.45.0
)

// Falls sherpa-onnx-go nicht per go get auffindbar:
// 1. Klone https://github.com/k2-fsa/sherpa-onnx
// 2. Nutze replace:
//    replace github.com/k2-fsa/sherpa-onnx-go => ../sherpa-onnx/go

require (
	git.sr.ht/~jackmordaunt/go-toast/v2 v2.0.3 // indirect
	github.com/bep/debounce v1.2.1 // indirect
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jchv/go-winloader v0.0.0-20210711035445-715c2860da7e // indirect
	github.com/k2-fsa/sherpa-onnx-go-linux v1.13.3 // indirect
	github.com/k2-fsa/sherpa-onnx-go-macos v1.13.3 // indirect
	github.com/k2-fsa/sherpa-onnx-go-windows v1.13.3 // indirect
	github.com/labstack/echo/v4 v4.13.3 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/leaanthony/go-ansi-parser v1.6.1 // indirect
	github.com/leaanthony/gosod v1.0.4 // indirect
	github.com/leaanthony/slicer v1.6.0 // indirect
	github.com/leaanthony/u v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/samber/lo v1.49.1 // indirect
	github.com/tkrajina/go-reflector v0.5.8 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/wailsapp/go-webview2 v1.0.22 // indirect
	github.com/wailsapp/mimetype v1.4.1 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)
