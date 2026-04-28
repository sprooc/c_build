package builder

import (
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

func Build() {
	// err := docker.Run([]string{"cd", config.WorkingDir}, os.Stdout)
	// if err != nil {
	// 	panic(err)
	// }

	// err := docker.Run([]string{"make", "deps"}, os.Stdout)
	// if err != nil {
	// 	panic(err)
	// }

	cleanCmd := os.Getenv("C_BUILD_CLEAN_CMD")
	if cleanCmd == "" {
		slog.Info("skip c_build internal clean step")
	} else {
		slog.Info("run c_build clean command", "cmd", cleanCmd)
		err := docker.Run([]string{"sh", "-c", cleanCmd}, os.Stdout)
		if err != nil {
			panic(err)
		}
	}

	makeCommand := fmt.Sprintf("umask %s && export LD_PRELOAD=%s/libreprobuild_interceptor.so && %s",
		config.Cfg.MetaData.Umask, config.ReprobuildDir, config.BuildCmd)
	if config.HasCustom {
		makeCommand = fmt.Sprintf("umask %s && export LD_PRELOAD=%s/libreprobuild_interceptor.so && export LD_LIBRARY_PATH=\"%s/deps:$LD_LIBRARY_PATH\" && %s",
			config.Cfg.MetaData.Umask, config.ReprobuildDir, config.WorkingDir, config.BuildCmd)
	}
	slog.Info("run original build command", "cmd", config.BuildCmd)

	err := docker.Run([]string{"sh", "-c", makeCommand}, os.Stdout)
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
	slog.Info("Check the artifacts")

	for _, artifact := range config.Cfg.Artifacts {
		path, ok := artifactPathInContainer(artifact.Path)
		if !ok {
			continue
		}
		sha256sum, err := installer.Sha256File(path)
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

func artifactPathInContainer(path string) (string, bool) {
	if !filepath.IsAbs(path) {
		return filepath.Join(config.WorkingDir, path), true
	}

	cleanPath := filepath.Clean(path)
	cleanWorkingDir := filepath.Clean(config.WorkingDir)
	if cleanPath == cleanWorkingDir || strings.HasPrefix(cleanPath, cleanWorkingDir+string(os.PathSeparator)) {
		return cleanPath, true
	}

	slog.Warn("skip external artifact hash check", "path", path)
	return "", false
}
