package config

var Image string = ""

var ContainerName string = "unspecified" // container already exists just reuse it not remove it

// var ContainerName string = "unspecified-gcc" // container already exists just reuse it not remove it

var WorkingDir = "/ws"

var ReprobuildDir = "/opt/reprobuild"

var GraphOutputPath = ""

var Env []string = []string {
	// "http_proxy=${your own proxy}",
	// "https_proxy=${your own proxy}",
	"CC=/usr/bin/x86_64-linux-gnu-gcc-14",
	"CXX=/usr/bin/x86_64-linux-gnu-g++-14",
	// "CFLAGS=-ffile-prefix-map=/ws=.",
	// "CXXFLAGS=-ffile-prefix-map=/ws=.",
}

var BuildCmd string = "make"