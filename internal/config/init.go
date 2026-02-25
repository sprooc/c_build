package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

func Init(configPath string) {
	// PkgMgrName = "apt"
	// Image = "ubuntu:22.04"
	// Image = "gcc:13"
	// Libs = append(Libs, LibInfo{
	// 	// Name : "tmux",
	// 	// Version: "3.2a-4ubuntu0.2",
	// 	Name : "build-essential",
	// 	Version: "12.9ubuntu3",
	// })

	Parse(configPath)

	Image = strings.ReplaceAll(strings.ToLower(Cfg.MetaData.Distribution), " ", ":") // Not stable
	slog.Info(fmt.Sprintf("Image: <%s>", Image))

	slog.Info(fmt.Sprintf("PkgMgr: <%s>", PkgMgrName))

	HostBuildRootDir = Cfg.MetaData.BuildPath

	slog.Info(fmt.Sprintf("Build root dir: <%s>", HostBuildRootDir))

	if len(Cfg.MetaData.BuildCmd) > 0 {
		BuildCmd = Cfg.MetaData.BuildCmd
	}

	slog.Info(fmt.Sprintf("Build command: <%s>", BuildCmd))

	// fmt.Println("Dependencies:")
	for _, dep := range Cfg.Dependencies {
		lib := LibInfo{
			Name: dep.Name,
			Path: dep.Path,
			Version: dep.Version,
			Sha256: dep.Hash,
			Origin: dep.Origin,
		}
		HasCustom = lib.Origin == "custom"
		Libs = append(Libs, lib)
		// fmt.Printf("%+v\n", lib)
	}

	LD_PRELOAD := fmt.Sprintf("LD_PRELOAD=%s/libreprobuild_interceptor.so", ReprobuildDir)
	slog.Info(fmt.Sprintf("set %s", LD_PRELOAD))
	// Env = append(Env, LD_PRELOAD)
	if HasCustom {
		LD_LIBRARY_PATH := fmt.Sprintf("LD_LIBRARY_PATH=%s/deps:$LD_LIBRARY_PATH", WorkingDir)
		slog.Info(fmt.Sprintf("set %s", LD_LIBRARY_PATH))
	}

	t, _ := time.Parse(time.RFC3339, Cfg.MetaData.BuildTimeStamp)
	slog.Info(fmt.Sprintf("set SOURCE_DATE_EPOCH=%d", t.Unix()))
	Env = append(Env, fmt.Sprintf("SOURCE_DATE_EPOCH=%d", t.Unix()))

	CFLAGS := fmt.Sprintf("CFLAGS=-ffile-prefix-map=%s=. -frandom-seed=%s",WorkingDir, Cfg.MetaData.RandomSeed)
	CXXFLAGS := fmt.Sprintf("CXXFLAGS=-ffile-prefix-map=%s=. -frandom-seed=%s",WorkingDir, Cfg.MetaData.RandomSeed)
	REPROBUILD_COMPILER_FLAGS := fmt.Sprintf("REPROBUILD_COMPILER_FLAGS=-ffile-prefix-map=%s=. -frandom-seed=%s",WorkingDir, Cfg.MetaData.RandomSeed)

	slog.Info(fmt.Sprintf("set %s", CFLAGS))
	slog.Info(fmt.Sprintf("set %s", CXXFLAGS))
	slog.Info(fmt.Sprintf("set %s", REPROBUILD_COMPILER_FLAGS))

	Env = append(Env, CFLAGS, CXXFLAGS, REPROBUILD_COMPILER_FLAGS)

	locales := strings.Split(Cfg.MetaData.Locale, "\n")

	Env = append(Env, locales...)

	slog.Info(fmt.Sprintf("set locale: %s", Cfg.MetaData.Locale))

	if len(Cfg.GitCommitIDs) > 0 {
		REPROBUILD_LOG_GIT_CLONESG := fmt.Sprintf("REPROBUILD_LOG_GIT_CLONESG=%s/commits.txt", ReprobuildDir)
		slog.Info(fmt.Sprintf("set %s", REPROBUILD_LOG_GIT_CLONESG))
		Env = append(Env, REPROBUILD_LOG_GIT_CLONESG)
	}

	slog.Info(fmt.Sprintf("Container Env: %v", Env))

}