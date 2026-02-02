module github.com/openstatusHQ/cli

go 1.24.0

require github.com/urfave/cli/v3 v3.0.0-alpha9.2 // direct

require (
	buf.build/gen/go/openstatus/api/connectrpc/gosimple v1.19.1-20260202165838-5bd92a1e5d53.2
	buf.build/gen/go/openstatus/api/protocolbuffers/go v1.36.11-20260202165838-5bd92a1e5d53.1
	connectrpc.com/connect v1.19.1
	github.com/fatih/color v1.18.0
	github.com/google/go-cmp v0.7.0
	github.com/knadh/koanf/parsers/yaml v0.1.0
	github.com/knadh/koanf/providers/file v1.1.2
	github.com/knadh/koanf/v2 v2.1.1
	github.com/logrusorgru/aurora/v4 v4.0.0
	github.com/olekukonko/tablewriter v1.0.7
	github.com/rodaine/table v1.3.0
	github.com/urfave/cli-docs/v3 v3.0.0-alpha6
	sigs.k8s.io/yaml v1.4.0
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/olekukonko/errors v0.0.0-20250405072817-4e6d85265da6 // indirect
	github.com/olekukonko/ll v0.0.8 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
