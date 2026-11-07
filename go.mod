module github.com/Daniel-Ric/CraftBox

go 1.26.10

replace github.com/brentp/intintmap => github.com/brentp/intintmap v0.0.0-20190211203843-30dc0ade9af9

replace github.com/df-mc/dragonfly => ./third_party/dragonfly

replace github.com/sandertv/gophertunnel => ./third_party/gophertunnel

replace github.com/df-mc/go-nethernet => ./third_party/nethernet

replace github.com/pion/webrtc/v4 => ./third_party/webrtc

replace github.com/df-mc/go-xsapi/v2 => ./third_party/xsapi

replace github.com/df-mc/go-playfab/v2 => ./third_party/playfab

require (
	gioui.org v0.9.0
	gioui.org/x v0.9.0
	git.wow.st/gmp/jni v0.0.0-20210610011705-34026c7e22d0
	github.com/OneOfOne/xxhash v1.2.8
	github.com/chzyer/readline v1.5.1
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21
	github.com/coder/websocket v1.8.15
	github.com/dblezek/tga v0.0.0-20150626111426-80720cbc1017
	github.com/df-mc/dragonfly v0.11.7-0.20261104184903-106fdc28d09a
	github.com/df-mc/go-nethernet v1.0.27
	github.com/df-mc/go-playfab/v2 v2.0.4-0.20261011110813-e01ef1ec605c
	github.com/df-mc/goleveldb v1.1.9
	github.com/dop251/goja v0.0.0-20251201205617-2bb4c724c0f9
	github.com/dop251/goja_nodejs v0.0.0-20251015164255-5e94316bedaf
	github.com/fatih/color v1.18.0
	github.com/gioui-plugins/gio-plugins v0.8.0
	github.com/go-gl/mathgl v1.2.0
	github.com/go-jose/go-jose/v4 v4.1.5
	github.com/google/uuid v1.6.0
	github.com/klauspost/compress v1.18.4
	github.com/minio/selfupdate v0.6.0
	github.com/nicksnyder/go-i18n/v2 v2.6.0
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/sandertv/gophertunnel v1.63.1-0.20261030102901-0e179fc4bb08
	github.com/sirupsen/logrus v1.9.3
	github.com/tailscale/hujson v0.0.0-20250605163823-992244df8c5a
	github.com/wailsapp/wails/v2 v2.15.0
	golang.org/x/exp v0.0.0-20251209150349-8475f28825e9
	golang.org/x/exp/shiny v0.0.0-20251209150349-8475f28825e9
	golang.org/x/oauth2 v0.36.0
	golang.org/x/sys v0.47.0
	golang.org/x/text v0.41.0
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/cenkalti/backoff/v7 v7.0.0 // indirect

require (
	git.sr.ht/~jackmordaunt/go-toast/v2 v2.0.3 // indirect
	github.com/UserExistsError/conpty v0.1.4
	github.com/andreburgaud/crypt2go v1.8.0 // indirect
	github.com/bep/debounce v1.2.1 // indirect
	github.com/coreos/go-oidc/v3 v3.17.0 // indirect
	github.com/creachadair/mds v0.26.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jchv/go-winloader v0.0.0-20210711035445-715c2860da7e // indirect
	github.com/labstack/echo/v4 v4.13.3 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/leaanthony/go-ansi-parser v1.6.1 // indirect
	github.com/leaanthony/gosod v1.0.4 // indirect
	github.com/leaanthony/slicer v1.6.0 // indirect
	github.com/leaanthony/u v1.1.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/pion/stun/v4 v4.0.1 // indirect
	github.com/pion/transport/v5 v5.1.4 // indirect
	github.com/pion/turn/v5 v5.1.3 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/samber/lo v1.49.1 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/tkrajina/go-reflector v0.5.8 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/wailsapp/go-webview2 v1.0.22 // indirect
	github.com/wailsapp/mimetype v1.4.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/time v0.14.0 // indirect
)

require (
	aead.dev/minisign v0.3.0 // indirect
	gioui.org/shader v1.0.8 // indirect
	github.com/brentp/intintmap v0.0.0-20251106190759-56907b1f8479 // indirect
	github.com/creachadair/jrpc2 v1.3.5
	github.com/df-mc/go-xsapi/v2 v2.0.4-0.20261025033432-d43cca94551a
	github.com/df-mc/jsonc v1.0.5 // indirect
	github.com/df-mc/worldupgrader v1.0.22 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/ftrvxmtrx/tga v0.0.0-20150524081124-bd8e8d5be13a // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/go-text/typesetting v0.3.1 // indirect
	github.com/godbus/dbus/v5 v5.2.0 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/pprof v0.0.0-20251208000136-3d256cb9ff16 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pion/datachannel v1.6.3 // indirect
	github.com/pion/dtls/v3 v3.1.15 // indirect
	github.com/pion/ice/v4 v4.4.8 // indirect
	github.com/pion/interceptor v0.1.51 // indirect
	github.com/pion/logging v0.2.4 // indirect
	github.com/pion/mdns/v2 v2.2.2 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.20 // indirect
	github.com/pion/rtp v1.10.6 // indirect
	github.com/pion/sctp v1.12.0 // indirect
	github.com/pion/sdp/v3 v3.0.20 // indirect
	github.com/pion/srtp/v3 v3.1.3 // indirect
	github.com/pion/webrtc/v4 v4.2.26 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	github.com/zaataylor/cartesian v0.0.0-20221028053253-3b3244d82727
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/image v0.41.0
	golang.org/x/net v0.57.0 // indirect
)
