package installer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/venlax/c_build/internal/config"
	"github.com/venlax/c_build/internal/docker"
)

type commandArgsPassWay int

const (
	Tail commandArgsPassWay = iota
	Xargs 
	NoArgs
)

type command struct {
	strs []string
	passWay commandArgsPassWay
}

type pkgMgr struct{
	name string
	versionTmpl string
	updateCommand command
	installCommand command	
}

var pkgMgrs map[string]pkgMgr = map[string]pkgMgr {
	"apt" : {
		"apt", 
		`{{.Name}}={{.Version}}`,
		command {
			[]string{"apt", "update"},
			NoArgs,
		}, 
		command {
			[]string{"apt", "install", "-y", "--allow-downgrades"},
			Tail,
		},
	},
	"apk" : {
		"apk",
		`{{.Name}}={{.Version}}`,
		command {
			[]string{"apk", "update"},
			NoArgs,
		},
		command {
			[]string{"apk", "add", "-y"},
			Tail,
		},
	},
	"dnf" : {
		"dnf",
		`{{.Name}}-{{.Version}}`,
		command {
			[]string{"dnf", "makecache"},
			NoArgs,
		},
		command {
			[]string{"dnf", "install", "-y", "--allowerasing"},
			Tail,
		},
	},
	"yum" : {
		"yum",
		`{{.Name}}-{{.Version}}`,
		command {
			[]string{"yum", "makecache"},
			NoArgs,
		},
		command {
			[]string{"yum", "install", "-y"},
			Tail,
		},
	},
	"pacman": {
		"pacman",
		`{{.Name}}`,
		command {
			[]string{"pacman", "-Syu"},
			NoArgs,
		},
		command {
			[]string{"pacman", "-U", "--noconfirm"},
			Tail,
		},
	},
}

func GetPkgMgr(name string) pkgMgr {
	return pkgMgrs[name]
}

func (p *pkgMgr) runUpdate() {
	runCommand(p.updateCommand, []string{}, os.Stdout)
}

func (p *pkgMgr) RunInstall(libInfo config.LibInfo) {
	tpl, err := template.New("lib_full_name").Parse(p.versionTmpl)
	if err != nil {
		panic(err)
	}
	var arg string
	if libInfo.Version == "" {
		arg = libInfo.Name
	} else {
		var buf bytes.Buffer
		err := tpl.Execute(&buf, libInfo)
		if err != nil {
			panic(err)
		}
		arg = buf.String()
	}
	runCommand(p.installCommand, []string{arg}, os.Stdout)
	// argvs := make([]string, 0, len(p.installCommand.strs) + 1)
	// argvs = append(argvs, p.installCommand.strs...)
	// argvs = append(argvs, libInfo.Name + "=" + libInfo.Version)
	// err := docker.Run(argvs)
	// if (err != nil) {
	// 	panic(err)
	// }

	// go to checker please
} 

func (p *pkgMgr) runInstallAll() {
	tmp := make([]string, len(config.Libs))
	tpl, err := template.New("lib_full_name").Parse(p.versionTmpl)
	if err != nil {
		panic(err)
	}
	for i, libInfo := range config.Libs {
		if libInfo.Origin == "custom" {
			continue
		}
		var arg string
		if libInfo.Version == "" {
			arg = libInfo.Name
		} else {
			var buf bytes.Buffer
			err := tpl.Execute(&buf, libInfo)
			if err != nil {
				panic(err)
			}
			arg = buf.String()
		}
		tmp[i] = arg
	} 
	runCommand(p.installCommand, tmp, os.Stdout)
} 

// func (p *pkgMgr) runGetLibVersion(libInfo config.LibInfo) string {
// 	var sb strings.Builder 
// 	runCommand(p.getLibVersionCommand, []string{libInfo.Name}, &sb)
// 	versionRawStr := sb.String()
// 	return strings.TrimSpace(strings.TrimPrefix(versionRawStr, "Version:"))
// }

func runCommand(c command, args []string, writer io.Writer) {
	cmd := make([]string, 0)
	switch c.passWay {
	case NoArgs:
		cmd = append(cmd, c.strs...)
	case Tail:
		cmd = append(cmd, c.strs...)
		cmd = append(cmd, args...)
	case Xargs:
		cmd = append(cmd, "sh", "-c")
		realCmdStr := fmt.Sprintf(`echo %s | xargs %s`, strings.Join(args, ", "), strings.Join(c.strs, " "))
		cmd = append(cmd, realCmdStr)		
		// cmd = append(cmd, "xargs")
		// cmd = append(cmd, args...)
		// cmd = append(cmd)
	default:
		panic("Something wrong.")
	}

	err := docker.Run(cmd, writer)
	if (err != nil) {
		panic(err)
	}
}

func commandStr(c command, args []string) string {
	cmd := make([]string, 0)
	switch c.passWay {
	case NoArgs:
		cmd = append(cmd, c.strs...)
	case Tail:
		cmd = append(cmd, c.strs...)
		cmd = append(cmd, args...)
	case Xargs:
		cmd = append(cmd, "sh", "-c")
		realCmdStr := fmt.Sprintf(`echo %s | xargs %s`, strings.Join(args, ", "), strings.Join(c.strs, " "))
		cmd = append(cmd, realCmdStr)		
		// cmd = append(cmd, "xargs")
		// cmd = append(cmd, args...)
		// cmd = append(cmd)
	default:
		panic("Something wrong.")
	}
	return strings.Join(cmd, " ")
}