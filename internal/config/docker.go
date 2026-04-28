package config

var Image string = ""

var ContainerName string = "unspecified" // container already exists just reuse it not remove it

// var ContainerName string = "unspecified-gcc" // container already exists just reuse it not remove it

// WorkingDir is overwritten from metadata.build_path during Init so the
// container sees the project at the same path as the host build.
var WorkingDir = "/ws"

var ReprobuildDir = "/opt/reprobuild"

var GraphOutputPath = ""

var Env []string = []string{
	// "http_proxy=${your own proxy}",
	// "https_proxy=${your own proxy}",
	// "CC=gcc",
	// "CXX=g++",
	// "CFLAGS=-ffile-prefix-map=/ws=.",
	// "CXXFLAGS=-ffile-prefix-map=/ws=.",
}

var BuildCmd string = "make"
