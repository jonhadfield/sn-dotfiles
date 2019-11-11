module github.com/jonhadfield/dotfiles-sn

go 1.12

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/fatih/color v1.7.0
	github.com/fatih/set v0.2.1
	github.com/google/uuid v1.1.1 // indirect
	github.com/jonhadfield/findexec v0.0.0-20190902195615-78db24cd4e77
	github.com/jonhadfield/gosn v0.0.0-20191111215415-15b083b40a06
	github.com/jonhadfield/sn-cli v0.0.0-20191111220039-59bccb866f1b
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/ryanuber/columnize v2.1.0+incompatible
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.22.1
)

// replace github.com/jonhadfield/sn-cli => ../sn-cli
