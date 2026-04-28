package installer

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/venlax/c_build/internal/config"
)

func ReprobuildRuntimePackages() []config.LibInfo {
	candidates := reprobuildRuntimeCandidates()
	if len(candidates) == 0 {
		slog.Warn("skip reprobuild runtime dependency detection: no host binaries found",
			"reprobuild_dir", config.HostReprobuildDir)
		return nil
	}

	existing := make(map[string]struct{}, len(config.Libs))
	for _, lib := range config.Libs {
		if lib.Name != "" {
			existing[lib.Name] = struct{}{}
		}
	}

	packages := make(map[string]config.LibInfo)
	for _, candidate := range candidates {
		libs, err := linkedLibraries(candidate)
		if err != nil {
			slog.Warn("failed to inspect reprobuild runtime libraries",
				"path", candidate,
				"error", err)
			continue
		}

		for _, libPath := range libs {
			pkg, err := packageForLibrary(libPath)
			if err != nil {
				slog.Warn("failed to resolve runtime package for library",
					"library", libPath,
					"error", err)
				continue
			}
			if pkg.Name == "" {
				continue
			}
			if _, ok := existing[pkg.Name]; ok {
				continue
			}
			packages[pkg.Name] = pkg
		}
	}

	res := make([]config.LibInfo, 0, len(packages))
	for _, pkg := range packages {
		res = append(res, pkg)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})

	return res
}

func reprobuildRuntimeCandidates() []string {
	rawCandidates := []string{
		filepath.Join(config.HostReprobuildDir, "build", "reprobuild"),
		filepath.Join(config.HostReprobuildDir, "libreprobuild_interceptor.so"),
		filepath.Join(config.HostReprobuildDir, "build", "libreprobuild_interceptor.so"),
	}

	seen := make(map[string]struct{}, len(rawCandidates))
	var candidates []string
	for _, candidate := range rawCandidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	return candidates
}

func linkedLibraries(binaryPath string) ([]string, error) {
	out, err := exec.Command("ldd", binaryPath).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ldd %s failed: %w: %s", binaryPath, err, strings.TrimSpace(string(out)))
	}

	return parseLddOutput(string(out)), nil
}

func parseLddOutput(output string) []string {
	seen := make(map[string]struct{})
	var libs []string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.Contains(line, "=> not found") {
			continue
		}

		var path string
		if idx := strings.Index(line, "=>"); idx >= 0 {
			fields := strings.Fields(strings.TrimSpace(line[idx+2:]))
			if len(fields) > 0 && strings.HasPrefix(fields[0], "/") {
				path = fields[0]
			}
		} else {
			fields := strings.Fields(line)
			if len(fields) > 0 && strings.HasPrefix(fields[0], "/") {
				path = fields[0]
			}
		}

		if path == "" {
			continue
		}

		if resolved, err := filepath.EvalSymlinks(path); err == nil && resolved != "" {
			path = resolved
		}

		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		libs = append(libs, path)
	}

	return libs
}

func packageForLibrary(libPath string) (config.LibInfo, error) {
	switch config.PkgMgrName {
	case "apt":
		return aptPackageForLibrary(libPath)
	case "dnf", "yum":
		return rpmPackageForLibrary(libPath)
	case "pacman":
		return pacmanPackageForLibrary(libPath)
	default:
		return config.LibInfo{}, fmt.Errorf("unsupported package manager for reprobuild runtime dependency detection: %s", config.PkgMgrName)
	}
}

func aptPackageForLibrary(libPath string) (config.LibInfo, error) {
	out, err := exec.Command("dpkg", "-S", libPath).CombinedOutput()
	if err != nil {
		return config.LibInfo{}, fmt.Errorf("dpkg -S %s failed: %w: %s", libPath, err, strings.TrimSpace(string(out)))
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "diversion by") {
			continue
		}
		pkgName, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		return config.LibInfo{
			Name: strings.TrimSpace(pkgName),
		}, nil
	}

	return config.LibInfo{}, fmt.Errorf("unable to parse dpkg owner for %s", libPath)
}

func rpmPackageForLibrary(libPath string) (config.LibInfo, error) {
	out, err := exec.Command("rpm", "-qf", libPath).CombinedOutput()
	if err != nil {
		return config.LibInfo{}, fmt.Errorf("rpm -qf %s failed: %w: %s", libPath, err, strings.TrimSpace(string(out)))
	}

	fullName := strings.TrimSpace(string(out))
	if fullName == "" {
		return config.LibInfo{}, fmt.Errorf("rpm -qf returned empty package for %s", libPath)
	}

	nameOut, err := exec.Command("rpm", "-q", "--qf", "%{NAME}\n", fullName).CombinedOutput()
	if err != nil {
		return config.LibInfo{}, fmt.Errorf("rpm query name for %s failed: %w: %s", fullName, err, strings.TrimSpace(string(nameOut)))
	}

	return config.LibInfo{
		Name: strings.TrimSpace(string(nameOut)),
	}, nil
}

func pacmanPackageForLibrary(libPath string) (config.LibInfo, error) {
	out, err := exec.Command("pacman", "-Qo", libPath).CombinedOutput()
	if err != nil {
		return config.LibInfo{}, fmt.Errorf("pacman -Qo %s failed: %w: %s", libPath, err, strings.TrimSpace(string(out)))
	}

	line := strings.TrimSpace(string(out))
	parts := strings.Split(line, " is owned by ")
	if len(parts) != 2 {
		return config.LibInfo{}, fmt.Errorf("unable to parse pacman owner for %s: %s", libPath, line)
	}

	fields := strings.Fields(parts[1])
	if len(fields) == 0 {
		return config.LibInfo{}, fmt.Errorf("empty pacman owner for %s", libPath)
	}

	return config.LibInfo{
		Name: fields[0],
	}, nil
}
