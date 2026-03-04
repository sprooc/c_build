package builder

import (
	// "fmt"
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/venlax/c_build/internal/config"
	"github.com/venlax/c_build/internal/docker"
	"github.com/venlax/c_build/internal/installer"
	// "github.com/venlax/c_build/internal/installer"
)

var originGraph *BuildGraph

func Build() {
	// err := docker.Run([]string{"cd", config.WorkingDir}, os.Stdout)
	// if err != nil {
	// 	panic(err)
	// }

	// err := docker.Run([]string{"make", "deps"}, os.Stdout)
	// if err != nil {
	// 	panic(err)
	// }

	originDir, err := extractWorkingDir(config.HostBuildRootDir + "/build_graph.yaml")

	if err != nil || originDir == "" {
		slog.Error("can't get the working dir from the build_graph.yaml")
		panic(err)
	}

	originGraph, err = LoadGraph(config.HostBuildRootDir + "/build_graph.yaml", originDir)

	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}

	err = docker.Run([]string{"make", "clean"}, os.Stdout)
	if err != nil {
		panic(err)
	}


	var ld_path string
	if config.HasCustom {
		ld_path = fmt.Sprintf("env LD_PRELOAD=%s/libreprobuild_interceptor.so LD_LIBRARY_PATH=\"%s/deps:$LD_LIBRARY_PATH\"", config.ReprobuildDir, config.WorkingDir)
	} else {
		ld_path = fmt.Sprintf("env LD_PRELOAD=%s/libreprobuild_interceptor.so", config.ReprobuildDir)
	}

	if len(config.Cfg.GitCommitIDs) > 0 {
		filePath := filepath.Join(config.HostReprobuildDir, "commits.txt")
		file, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		for _, commit := range config.Cfg.GitCommitIDs {
			// 每行：Repo CommitID\n
			fmt.Fprintf(writer, "%s %s\n", commit.Repo, commit.CommitID)
		}
		writer.Flush()
	}

	pkgMgr := installer.GetPkgMgr(config.PkgMgrName)

	(&pkgMgr).RunInstall(config.LibInfo{
		Name : "wget",
	})
	(&pkgMgr).RunInstall(config.LibInfo{
		Name : "sudo",
	})

	if exist, err := docker.FileExists("/usr/bin/bpftrace0.24"); (!exist) || err != nil {
		MakeInstallCommand := fmt.Sprintf("cd %s && make install", config.ReprobuildDir)

		err = docker.Run([]string{"sh", "-c", MakeInstallCommand}, os.Stdout)

		if err != nil {
			slog.Error("install bpftrace in docker failed")
			panic(err)
		}
	}

	Reprobuild := config.ReprobuildDir + "/build/reprobuild"

	if exist, err := docker.FileExists(Reprobuild); (!exist) || err != nil {
		docker.Run([]string{"make", "-C", config.ReprobuildDir}, os.Stdout)
	} 

	docker.Run([]string {"cp", Reprobuild, "/"}, os.Stdout)

	var MakeCommand string

	if config.GraphOutputPath != "" {
		MakeCommand = fmt.Sprintf("umask %s && /reprobuild -g=%s %s", config.Cfg.MetaData.Umask, config.GraphOutputPath, config.BuildCmd)
		// umask 022 && ./A -g=B make
	} else {
		MakeCommand = fmt.Sprintf("umask %s && /reprobuild -g %s", config.Cfg.MetaData.Umask, config.BuildCmd)
	}


	MakeCommand = strings.ReplaceAll(MakeCommand, "&&", "&& " + ld_path)


	err = docker.Run([]string{"sh", "-c", MakeCommand}, os.Stdout)
	if err != nil {
		panic(err)
	}

	// fmt.Println(installer.Sha256File("/ws/lua"))

	// err = docker.Run([]string{"./hello"}, os.Stdout)
	// if err != nil {
	// 	panic(err)
	// }

}

func Check() {
	slog.Info("Verify the build graph")

	var buildGraph *BuildGraph
	var err error

	if config.GraphOutputPath == "" {
		buildGraph, err = LoadGraph(config.HostBuildRootDir + "/build_graph.yaml", config.WorkingDir)
	} else {
		dstPath := "/tmp/build_graph.yaml"

		err = docker.CopyFileFromContainer(config.GraphOutputPath, dstPath)
		if err != nil {
			slog.Error(err.Error())
			panic(err)
		}
		
		buildGraph, err = LoadGraph(dstPath, config.WorkingDir)
	}

	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}

	if !EqualGraph(originGraph, buildGraph) {
		slog.Error("the build graph not equal to the original one")
		os.Exit(1)
	} else {
		slog.Info("[OK]: the build graphs are equal")
	}

	slog.Info("Check the artifacts")

	for _, artifact := range config.Cfg.Artifacts {
		sha256sum, err := installer.Sha256File(config.WorkingDir + "/" + artifact.Path)
		if err != nil {
			panic(err)
		}
		if sha256sum != artifact.Hash {
			slog.Error(fmt.Sprintf("build result [%s] hash [%s] not match the artifact hash [%s]", artifact.Path, sha256sum[:8], artifact.Hash[:8]))
			os.Exit(1)
		}
		slog.Info(fmt.Sprintf("[OK]: %s=%s", artifact.Path, sha256sum[:8]))
	}
}