// +build ignore

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const commandName = "tagaa"

type platform struct {
	os   string
	arch string
}

type binary struct {
	name    string
	version string
	targets []platform
}

func (bin binary) Name(os, arch string) string {
	s := fmt.Sprintf("%s_%s-%s_%s", bin.name, os, arch, bin.version)
	if os == "windows" {
		s = s + ".exe"
	}
	return s
}

func (bin binary) Names() []string {
	names := make([]string, len(bin.targets))
	for i, t := range bin.targets {
		names[i] = bin.Name(t.os, t.arch)
	}
	return names
}

var (
	release   = flag.Bool("release", false, "Build binaries for all target platforms.")
	clean     = flag.Bool("clean", false, "Remove all created binaries from current directory.")
	buildARCH = flag.String("arch", runtime.GOARCH, "Architecture to build for.")
	buildOS   = flag.String("os", runtime.GOOS, "Operating system to build for.")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: go run make.go [OPTIONS]\n\n")
	fmt.Fprintln(os.Stderr, "OPTIONS:")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	bin := binary{
		name: commandName,
		targets: []platform{
			{os: "linux", arch: "386"}, {os: "linux", arch: "amd64"},
			{os: "windows", arch: "386"}, {os: "windows", arch: "amd64"},
			{os: "darwin", arch: "386"}, {os: "darwin", arch: "amd64"},
		},
	}
	bin.version = getVersion()

	if *release {
		fmt.Println("CPUs:", runtime.NumCPU())
		fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0))
		start := time.Now()
		forEachBinary(bin, buildBinary)
		fmt.Println("Time elapsed:", time.Since(start))
		os.Exit(0)
	}

	if *clean {
		forEachBinary(bin, rmBinary)
		os.Exit(0)
	}

	buildBinary(bin, *buildOS, *buildARCH)
}

func getVersion() string {
	cmd := exec.Command("git", "describe", "--tags")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error running git describe: %v", err)
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
}

type binaryFunc func(bin binary, OS, arch string)

func forEachBinary(bin binary, fn binaryFunc) {
	var wg sync.WaitGroup
	for _, t := range bin.targets {
		wg.Add(1)
		go func(bin binary, os, arch string) {
			defer wg.Done()
			fn(bin, os, arch)
		}(bin, t.os, t.arch)
	}
	wg.Wait()
}

func buildBinary(bin binary, OS, arch string) {
	ldflags := fmt.Sprintf("--ldflags=-X main.theVersion=%s", bin.version)
	cmd := exec.Command("go", "build", ldflags, "-o", bin.Name(OS, arch))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = copyGoEnv()
	cmd.Env = setEnv(cmd.Env, "GOOS", OS)
	cmd.Env = setEnv(cmd.Env, "GOARCH", arch)
	fmt.Println("Building binary:", bin.Name(OS, arch))
	if err := cmd.Run(); err != nil {
		log.Fatalln("Error running go build:", err)
	}
}

func rmBinary(bin binary, OS, arch string) {
	err := os.Remove(bin.Name(OS, arch))
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "Error removing binary:", err)
		}
	}
}

func copyGoEnv() (environ []string) {
	for _, env := range os.Environ() {
		environ = append(environ, env)
	}
	return
}

func setEnv(env []string, key, value string) []string {
	for i, e := range env {
		if strings.HasPrefix(e, fmt.Sprintf("%s=", key)) {
			env[i] = fmt.Sprintf("%s=%s", key, value)
			return env
		}
	}
	env = append(env, fmt.Sprintf("%s=%s", key, value))
	return env
}
