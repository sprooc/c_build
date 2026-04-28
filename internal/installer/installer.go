package installer

import (
	"bytes"
	"log/slog"
	"os"
	"text/template"

	"github.com/venlax/c_build/internal/config"
	"github.com/venlax/c_build/internal/docker"
)

func Init() {
	configureAptSource()
	pkgMgr := GetPkgMgr(config.PkgMgrName)
	(&pkgMgr).runUpdate()
}

func configureAptSource() {
	if config.PkgMgrName != "apt" {
		return
	}

	slog.Info("configure apt source", "mirror", "mirrors.ustc.edu.cn")
	err := docker.Run([]string{"sh", "-c", `
set -e
mirror="http://mirrors.ustc.edu.cn/ubuntu/"
for f in /etc/apt/sources.list /etc/apt/sources.list.d/ubuntu.sources; do
  if [ -f "$f" ]; then
    sed -i \
      -e "s#http://archive.ubuntu.com/ubuntu/#${mirror}#g" \
      -e "s#http://security.ubuntu.com/ubuntu/#${mirror}#g" \
      -e "s#https://archive.ubuntu.com/ubuntu/#${mirror}#g" \
      -e "s#https://security.ubuntu.com/ubuntu/#${mirror}#g" \
      "$f"
  fi
done
cat >/etc/apt/apt.conf.d/99reprobuild-network <<'EOF'
Acquire::Retries "5";
Acquire::http::Timeout "30";
Acquire::https::Timeout "30";
Acquire::ForceIPv4 "true";
EOF
`}, os.Stdout)
	if err != nil {
		panic(err)
	}
}

func Install() {
	pkgMgr := GetPkgMgr(config.PkgMgrName)

	(pkgMgr).runInstallAll()
	runtimePkgs := ReprobuildRuntimePackages()
	for _, libInfo := range runtimePkgs {
		slog.Info("install reprobuild runtime dependency", "package", libInfo.Name)
		(&pkgMgr).RunInstall(libInfo)
	}
	slog.Info("skip dependency checksum verification", "dependency_count", len(config.Libs))
}

func InstallStrs() []string {
	var res []string
	pkgMgr := GetPkgMgr(config.PkgMgrName)
	res = append(res, commandStr((&pkgMgr).updateCommand, []string{}))
	targetArgs, err := installArgsForLibs(pkgMgr, config.Libs)
	if err != nil {
		panic(err)
	}

	if len(targetArgs) > 0 {
		res = append(res, commandStr((&pkgMgr).installCommand, targetArgs))
	}

	runtimeArgs, err := installArgsForLibs(pkgMgr, ReprobuildRuntimePackages())
	if err != nil {
		panic(err)
	}
	if len(runtimeArgs) > 0 {
		res = append(res, commandStr((&pkgMgr).installCommand, runtimeArgs))
	}

	return res
}

func installArgsForLibs(pkgMgr pkgMgr, libs []config.LibInfo) ([]string, error) {
	tpl, err := template.New("lib_full_name").Parse(pkgMgr.versionTmpl)
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, len(libs))
	for _, libInfo := range libs {
		if libInfo.Origin == "custom" || libInfo.Name == "" {
			continue
		}

		if libInfo.Version == "" {
			args = append(args, libInfo.Name)
			continue
		}

		var buf bytes.Buffer
		err := tpl.Execute(&buf, libInfo)
		if err != nil {
			return nil, err
		}
		args = append(args, buf.String())
	}

	return args, nil
}
