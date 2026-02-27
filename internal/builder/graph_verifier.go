package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type BuildGraph struct {
    Nodes []Node `yaml:"nodes"`
    Edges []Edge `yaml:"edges"`
}

type Node struct {
    Path string `yaml:"path"`
    Type string `yaml:"type"`
    Hash string `yaml:"hash"`
}

type Edge struct {
    Command     string   `yaml:"command"`
    CommandPath string   `yaml:"command_path"`
    PID         int      `yaml:"pid"`
    Inputs      []string `yaml:"inputs"`
    Output      string   `yaml:"output"`
    Args        string   `yaml:"args"`
}

var filePrefixMapRe = regexp.MustCompile(
	`-ffile-prefix-map=([^=\s]+)=\.(\s|$|")`,
)

func LoadGraph(path string, workingDir string) (*BuildGraph, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var g BuildGraph
    if err := yaml.Unmarshal(data, &g); err != nil {
        return nil, err
    }

	normalizeGraph(&g, workingDir)

    return &g, nil
}

func normalizeGraph(g *BuildGraph, workingDir string) {
    for i := range g.Nodes {
        g.Nodes[i].Path = normalizePath(g.Nodes[i].Path, workingDir)
    }

    for i := range g.Edges {
        for j := range g.Edges[i].Inputs {
            g.Edges[i].Inputs[j] = normalizePath(
                g.Edges[i].Inputs[j],
				workingDir,
            )
        }

        g.Edges[i].Output = normalizePath(
            g.Edges[i].Output,
			workingDir,
        )
    }
}

func normalizePath(p string, workingDir string) string {
    absWorkingDir, err := filepath.Abs(workingDir)
    if err != nil {
        return p
    }

    absPath, err := filepath.Abs(p)
    if err != nil {
        return p
    }

    rel, err := filepath.Rel(absWorkingDir, absPath)
    if err != nil {
        return p
    }

    if !strings.HasPrefix(rel, "..") {
        return rel
    }

    return p
}


func EqualGraph(g1, g2 *BuildGraph) bool {
    if !equalNodes(g1.Nodes, g2.Nodes) {
		fmt.Println("node not equal")
        return false
    }

    if !equalEdges(g1.Edges, g2.Edges) {
		fmt.Println("edge not equal")
        return false
    }

    return true
}

func equalNodes(a, b []Node) bool {
    if len(a) != len(b) {
        return false
    }

    ma := make(map[string]Node)
    mb := make(map[string]Node)

    for _, n := range a {
        ma[n.Path] = n
    }
    for _, n := range b {
        mb[n.Path] = n
    }

    for path, nodeA := range ma {
        nodeB, ok := mb[path]
        if !ok {
			fmt.Println(path, nodeA)
            return false
        }

        if nodeA.Type != nodeB.Type {
            return false
        }

		if nodeA.Hash == "" || nodeB.Hash == "" {
			continue
		}

        if nodeA.Hash != nodeB.Hash {
            return false
        }
    }

    return true
}

func edgeSignature(e Edge) string {
    inputs := append([]string(nil), e.Inputs...)
    sort.Strings(inputs)

    return strings.Join([]string{
        strings.Join(inputs, ","),
        e.Output,
    }, "|")
}

func equalEdges(a, b []Edge) bool {
    if len(a) != len(b) {
        return false
    }

    count := make(map[string]int)

    for _, e := range a {
        sig := edgeSignature(e)
        count[sig]++
    }

    for _, e := range b {
        sig := edgeSignature(e)
        if count[sig] == 0 {
            return false
        }
        count[sig]--
    }

    return true
}

func extractWorkingDir(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	m := filePrefixMapRe.FindStringSubmatch(string(data))
	if len(m) > 1 {
		return m[1], nil
	}

	return "", nil
}