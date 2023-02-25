module github.com/jonhadfield/dotfiles-sn

go 1.16

require (
	github.com/asdine/storm/v3 v3.2.1
	github.com/briandowns/spinner v1.12.0
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/fatih/color v1.13.0
	github.com/fatih/set v0.2.1
	github.com/godbus/dbus/v5 v5.0.6 // indirect
	github.com/jonhadfield/findexec v0.0.0-20190902195615-78db24cd4e77
	github.com/jonhadfield/gosn-v2 v0.0.0-20211123204812-5a0242ddbf0d
	github.com/kr/text v0.2.0 // indirect
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/pkg/errors v0.9.1
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/columnize v2.1.2+incompatible
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e // indirect
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/ini.v1 v1.66.2 // indirect
)

//replace github.com/jonhadfield/gosn-v2 => ../gosn-v2
