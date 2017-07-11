// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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
	if OS == "windows" {
		runGoVersionInfo(bin, arch)
		defer rmGoVersionInfo(arch)
	}
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

func rmGoVersionInfoJSON() {
	err := os.Remove("versioninfo.json")
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "Error removing versioninfo.json:", err)
		}
	}
}

// generateGoVersionInfoJSON will generate a default versioninfo.json file so
// that the goversioninfo command can work.
func generateGoVersionInfoJSON() {
	type VersionInfo struct {
		FixedFileInfo struct {
			FileVersion struct {
				Major int `json:"Major"`
				Minor int `json:"Minor"`
				Patch int `json:"Patch"`
				Build int `json:"Build"`
			} `json:"FileVersion"`
			ProductVersion struct {
				Major int `json:"Major"`
				Minor int `json:"Minor"`
				Patch int `json:"Patch"`
				Build int `json:"Build"`
			} `json:"ProductVersion"`
			FileFlagsMask string `json:"FileFlagsMask"`
			FileFlags     string `json:"FileFlags "`
			FileOS        string `json:"FileOS"`
			FileType      string `json:"FileType"`
			FileSubType   string `json:"FileSubType"`
		} `json:"FixedFileInfo"`
		StringFileInfo struct {
			Comments         string `json:"Comments"`
			CompanyName      string `json:"CompanyName"`
			FileDescription  string `json:"FileDescription"`
			FileVersion      string `json:"FileVersion"`
			InternalName     string `json:"InternalName"`
			LegalCopyright   string `json:"LegalCopyright"`
			LegalTrademarks  string `json:"LegalTrademarks"`
			OriginalFilename string `json:"OriginalFilename"`
			PrivateBuild     string `json:"PrivateBuild"`
			ProductName      string `json:"ProductName"`
			ProductVersion   string `json:"ProductVersion"`
			SpecialBuild     string `json:"SpecialBuild"`
		} `json:"StringFileInfo"`
		VarFileInfo struct {
			Translation struct {
				LangID    string `json:"LangID"`
				CharsetID string `json:"CharsetID"`
			} `json:"Translation"`
		} `json:"VarFileInfo"`
	}

	// Defaults found at:
	// https://github.com/josephspurrier/goversioninfo/blob/096c7bd04a78bdb9b1bd32f81243644544e86f5c/versioninfo.json
	vi := VersionInfo{}
	vi.FixedFileInfo.FileVersion.Major = 1
	vi.FixedFileInfo.ProductVersion.Major = 1
	vi.StringFileInfo.ProductVersion = "v1.0.0.0"
	vi.FixedFileInfo.FileFlagsMask = "3f"
	vi.FixedFileInfo.FileFlags = "00"
	vi.FixedFileInfo.FileOS = "040004"
	vi.FixedFileInfo.FileType = "01"
	vi.FixedFileInfo.FileSubType = "00"
	vi.VarFileInfo.Translation.LangID = "0409"
	vi.VarFileInfo.Translation.CharsetID = "04B0"
	f, err := os.Create("versioninfo.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating versioninfo.json:", err)
		return
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(vi); err != nil {
		fmt.Fprintln(os.Stderr, "error encoding version info to JSON:", err)
		return
	}
}

func runGoVersionInfo(bin binary, arch string) {
	generateGoVersionInfoJSON()
	defer rmGoVersionInfoJSON()
	major, minor, patch, err := parseVersion(bin.version)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error parsing version:", err)
	}
	args := []string{
		fmt.Sprintf("-company=%s", "Kusubooru Inc."),
		fmt.Sprintf("-copyright=%s", "Copyright (C) 2015 Kusubooru Inc."),
		fmt.Sprintf("-product-name=%s", "Tagaa"),
		fmt.Sprintf("-product-version=%s", bin.version),
		fmt.Sprintf("-description=%s", "Tagaa Local Image Tagging"),
		fmt.Sprintf("-original-name=%s", bin.Name("windows", arch)),
		fmt.Sprintf("-icon=generate/kusubooru.ico"),
		fmt.Sprintf("-o=resource_windows_%s.syso", arch),
		fmt.Sprintf("-ver-major=%d", major),
		fmt.Sprintf("-ver-minor=%d", minor),
		fmt.Sprintf("-ver-patch=%d", patch),
	}
	cmd := exec.Command("goversioninfo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = copyGoEnv()
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running goversioninfo:", err)
	}
}

func rmGoVersionInfo(arch string) {
	err := os.Remove(fmt.Sprintf("resource_windows_%s.syso", arch))
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "Error removing syso file:", err)
		}
	}
}

func parseVersion(version string) (major, minor, patch int, err error) {
	if strings.HasPrefix(version, "v") {
		version = strings.TrimPrefix(version, "v")
	}
	if strings.Contains(version, "-") {
		version = version[:strings.Index(version, "-")]
	}
	v := strings.Split(version, ".")
	major, err = strconv.Atoi(v[0])
	if err != nil {
		return
	}
	minor, err = strconv.Atoi(v[1])
	if err != nil {
		return
	}
	patch, err = strconv.Atoi(v[2])
	if err != nil {
		return
	}
	return
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
