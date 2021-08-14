module github.com/jonhadfield/dotfiles-sn

go 1.16

require (
	github.com/asdine/storm/v3 v3.2.1
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/danieljoos/wincred v1.1.1 // indirect
	github.com/fatih/color v1.12.0
	github.com/fatih/set v0.2.1
	github.com/google/uuid v1.3.0 // indirect
	github.com/jonhadfield/findexec v0.0.0-20190902195615-78db24cd4e77
	github.com/jonhadfield/gosn-v2 v0.0.0-20210719172924-55b0cdb35616
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/columnize v2.1.2+incompatible
	github.com/spf13/cast v1.4.0 // indirect
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	golang.org/x/crypto v0.0.0-20210813211128-0a44fdfbc16e // indirect
	golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
)

//replace github.com/jonhadfield/gosn-v2 => ../gosn-v2
