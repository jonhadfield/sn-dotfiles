module github.com/jonhadfield/dotfiles-sn

go 1.16

require (
	github.com/asdine/storm/v3 v3.2.1
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/fatih/color v1.11.0
	github.com/fatih/set v0.2.1
	github.com/jonhadfield/findexec v0.0.0-20190902195615-78db24cd4e77
	github.com/jonhadfield/gosn-v2 v0.0.0-20210515192549-4d2e4afd096e
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/pelletier/go-toml v1.9.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/columnize v2.1.2+incompatible
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/sys v0.0.0-20210514084401-e8d321eab015 // indirect
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
)

//replace github.com/jonhadfield/gosn-v2 => ../gosn-v2
