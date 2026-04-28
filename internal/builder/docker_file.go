package builder

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/venlax/c_build/internal/config"
	"github.com/venlax/c_build/internal/installer"
)

const dockerfileTmpl string = `FROM {{.Image}}

{{- if .Env }}
ENV {{ join .Env " \\\n    " }}
{{- end }}

WORKDIR {{.WorkDir}}

{{- if .InstallCmds }}
{{- range .InstallCmds }}
RUN {{.}}
{{- end }}
{{- end }}

CMD ["/bin/sh", "-c", "{{ .BuildCmd }}"]
`

type DockerfileTmplData struct {
	Image       string
	Env         []string
	WorkDir     string
	InstallCmds []string
	BuildCmd    string
}

func RenderDockerfile(dstDir string, digest string) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(dockerfileTmpl)
	if err != nil {
		panic(err)
	}
	f, err := os.Create(dstDir + "/Dockerfile")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, genDockerfileData(digest))

	if err != nil {
		panic(err)
	}

	_, err = f.Write(buf.Bytes())

	if err != nil {
		panic(err)
	}
}

func genDockerfileData(digest string) DockerfileTmplData {
	var data DockerfileTmplData
	data.Image = digest
	data.WorkDir = config.WorkingDir
	data.Env = config.Env
	data.InstallCmds = installer.InstallStrs()
	cleanCmd := os.Getenv("C_BUILD_CLEAN_CMD")
	buildCmd := fmt.Sprintf("umask %s && export LD_PRELOAD=%s/libreprobuild_interceptor.so && %s",
		config.Cfg.MetaData.Umask, config.ReprobuildDir, config.BuildCmd)
	if config.HasCustom {
		buildCmd = fmt.Sprintf("umask %s && export LD_PRELOAD=%s/libreprobuild_interceptor.so && export LD_LIBRARY_PATH=\"%s/deps:$LD_LIBRARY_PATH\" && %s",
			config.Cfg.MetaData.Umask, config.ReprobuildDir, config.WorkingDir, config.BuildCmd)
	}
	if cleanCmd != "" {
		buildCmd = cleanCmd + " && " + buildCmd
	}
	data.BuildCmd = buildCmd
	return data
}
