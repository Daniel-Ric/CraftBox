package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
)

const (
	AppName           = "CraftBox"
	AppID             = "io.github.danielric.craftbox"
	AndroidAPILevel   = "21"
	GitTagExcludeExpr = "r*"
)

var (
	versionRegexp = regexp.MustCompile(`v(\d\.\d+\.\d+)(?:-(\d+)-(\w+))?`)
)

var gogioCmd string

func init() {
	var exeExt = ""
	if runtime.GOOS == "windows" {
		exeExt = ".exe"
	}
	gogioCmd = "./gogio" + exeExt
}

type Build struct {
	OS   string
	Arch string
	Type string
}

func (b Build) ExeExt() string {
	if b.OS == "windows" {
		return ".exe"
	}
	if b.OS == "android" {
		return ".apk"
	}
	if b.OS == "darwin" {
		return ".app"
	}
	return ""
}

func (b Build) LibExt() string {
	if b.OS == "windows" {
		return ".dll"
	}
	return ".so"
}

type BuildConfig struct {
	AppVersion         string
	BuildTag           string
	IsLatest           bool
	PackSupportEnabled bool
	GitHubOutputFile   *os.File
}

func (c *BuildConfig) writeGitHubOutput(name, value string) {
	if c.GitHubOutputFile != nil {
		fmt.Fprintf(c.GitHubOutputFile, "%s=%s\n", name, value)
		c.GitHubOutputFile.Sync()
	} else {
		log.Printf("::notice file=build.go::GITHUB_OUTPUT %s=%s", name, value)
	}
}

func findAndroidToolchain(ndkHome, arch, apiLevel string) (string, error) {
	hostSystem := runtime.GOOS
	hostArch := runtime.GOARCH

	hostTag := ""
	switch hostSystem {
	case "linux":
		hostTag = "linux-x86_64"
	case "darwin":
		if hostArch == "arm64" {
			hostTag = "darwin-aarch64"
		} else {
			hostTag = "darwin-x86_64"
		}
	case "windows":

		hostTag = "windows-x86_64"
	default:
		return "", fmt.Errorf("unsupported host system for Android NDK: %s", hostSystem)
	}

	compilerPrefixMap := map[string]string{
		"arm":   "armv7a-linux-androideabi",
		"arm64": "aarch64-linux-android",
		"386":   "i686-linux-android",
		"amd64": "x86_64-linux-android",
	}
	compilerPrefix, ok := compilerPrefixMap[arch]
	if !ok {
		return "", fmt.Errorf("unsupported Android architecture: %s", arch)
	}

	clangPath := filepath.Join(ndkHome, "toolchains", "llvm", "prebuilt", hostTag, "bin", fmt.Sprintf("%s%s-clang", compilerPrefix, apiLevel))

	if hostSystem == "windows" {
		clangPathCmd := clangPath + ".cmd"
		if _, err := os.Stat(clangPathCmd); err == nil {
			return clangPathCmd, nil
		}
	}

	if _, err := os.Stat(clangPath); err == nil {
		return clangPath, nil
	}

	return "", fmt.Errorf("android NDK clang compiler not found for arch %s at %s", arch, clangPath)
}

func generateChangelog(tagName string) (string, error) {

	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("git command not found: %w", err)
	}

	cmdRevParse := exec.Command("git", "rev-parse", tagName)
	tagCommitHashBytes, errRevParse := cmdRevParse.Output()
	if errRevParse != nil {

		log.Printf("Warning: Tag %s does not exist. Generating changelog from recent commits.", tagName)
		cmdLog := exec.Command("git", "log", "--pretty=format:%h %s", "-n", "20")
		logOutputBytes, errLog := cmdLog.Output()
		if errLog != nil {
			return "", fmt.Errorf("error fetching recent commits: %w", errLog)
		}
		return "No tag found, showing recent commits:\n" + strings.TrimSpace(string(logOutputBytes)), nil
	}

	tagCommitHash := strings.TrimSpace(string(tagCommitHashBytes))

	cmdLogRange := exec.Command("git", "log", "--pretty=format:%h %s", fmt.Sprintf("%s..HEAD", tagCommitHash))
	commitsOutputBytes, errLogRange := cmdLogRange.Output()
	if errLogRange != nil {
		return "", fmt.Errorf("error fetching commits since tag %s: %w", tagName, errLogRange)
	}

	commitsOutput := strings.TrimSpace(string(commitsOutputBytes))
	if commitsOutput == "" {
		return fmt.Sprintf("No new commits found since tag %s", tagName), nil
	}

	lines := strings.Split(commitsOutput, "\n")
	var changelogLines []string
	for _, line := range lines {
		if line != "" {
			changelogLines = append(changelogLines, "- "+line)
		}
	}

	return strings.Join(changelogLines, "\n"), nil
}

func (c *BuildConfig) getVersion() error {
	if version := os.Getenv("CRAFTBOX_VERSION"); version != "" {
		if !regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`).MatchString(version) {
			return fmt.Errorf("invalid CRAFTBOX_VERSION: %q", version)
		}
		buildTag := os.Getenv("CRAFTBOX_BUILD_TAG")
		if buildTag == "" {
			buildTag = version
		}
		if (buildTag != version && !strings.HasPrefix(buildTag, version+"-")) || !regexp.MustCompile(`^[0-9A-Za-z.-]+$`).MatchString(buildTag) {
			return fmt.Errorf("invalid CRAFTBOX_BUILD_TAG: %q", buildTag)
		}
		c.AppVersion = version
		c.BuildTag = buildTag
		c.IsLatest = !strings.HasSuffix(buildTag, "-dev")
		c.writeGitHubOutput("release_tag", "r"+c.BuildTag)
		c.writeGitHubOutput("build_tag", c.BuildTag)
		c.writeGitHubOutput("is_latest", fmt.Sprintf("%t", c.IsLatest))
		return nil
	}

	if _, err := exec.LookPath("git"); err != nil {
		log.Printf("Warning: git command not found. Using default version: 0.0.0")
		c.AppVersion = "0.0.0"
		c.BuildTag = "0.0.0-0"
		c.IsLatest = false
		c.writeGitHubOutput("release_tag", "r0.0.0")
		c.writeGitHubOutput("is_latest", "false")
		return nil
	}

	cmdDescribe := exec.Command("git", "describe", "--tags", "--always", "--exclude", GitTagExcludeExpr)
	gitDescribeBytes, err := cmdDescribe.Output()
	if err != nil {
		log.Printf("Error running git describe: %v. Using default version.", err)
		c.AppVersion = "0.0.0"
		c.BuildTag = "0.0.0-0"
		c.IsLatest = false
		c.writeGitHubOutput("release_tag", "r0.0.0")
		c.writeGitHubOutput("is_latest", "false")
		return nil
	}
	gitTag := strings.TrimSpace(string(gitDescribeBytes))

	verMatch := versionRegexp.FindStringSubmatch(gitTag)
	if verMatch == nil {
		log.Printf("Warning: git describe output '%s' does not match version regex. Using default version.", gitTag)
		c.AppVersion = "0.0.0"
		patch := "0"

		if regexp.MustCompile(`^[0-9a-f]{7,40}$`).MatchString(gitTag) {
			c.BuildTag = fmt.Sprintf("0.0.0-%s-%s", patch, gitTag[:min(7, len(gitTag))])
		} else {
			c.BuildTag = fmt.Sprintf("0.0.0-%s", patch)
		}
	} else {
		c.AppVersion = verMatch[1]
		patch := verMatch[2]
		if patch == "" {
			patch = "0"
		}
		commitHash := verMatch[3]
		c.BuildTag = fmt.Sprintf("%s-%s", c.AppVersion, patch)
		if commitHash != "" {
			c.BuildTag += "-" + commitHash
		}
	}

	cmdBranch := exec.Command("git", "branch", "--show-current")
	branchBytes, err := cmdBranch.Output()
	if err != nil {
		log.Printf("Warning: Could not determine active branch: %v", err)
		c.IsLatest = false
	} else {
		activeBranch := strings.TrimSpace(string(branchBytes))
		c.IsLatest = (activeBranch == "master" || activeBranch == "main")
	}

	c.writeGitHubOutput("release_tag", "r"+c.AppVersion)
	c.writeGitHubOutput("build_tag", c.BuildTag)
	c.writeGitHubOutput("is_latest", fmt.Sprintf("%t", c.IsLatest))

	return nil
}

func checkPackSupport() (bool, error) {
	filePath := filepath.Join("subcommands", "resourcepack-d", "resourcepack-d.go")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Warning: %s not found. Pack support disabled.", filePath)
			return false, nil
		}
		return false, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	return bytes.Contains(content[:min(len(content), 1024)], []byte("package ")), nil
}

func prepareWindowsResources(arch string) error {
	if arch != "amd64" && arch != "arm64" && arch != "386" {
		return fmt.Errorf("unsupported Windows resource architecture: %s", arch)
	}
	path := filepath.Join("cmd", AppName, "craft_box_windows_"+arch+".syso")
	if err := runCommand([]string{"go", "run", "github.com/akavel/rsrc@v0.10.2", "-arch", arch, "-manifest", "assets/craft-box.manifest", "-ico", "assets/craft-box.ico", "-o", path}, os.Environ(), ""); err != nil {
		return err
	}
	return setWailsIconResourceID(path)
}

func cleanSyso() {
	sysoDir := filepath.Join("cmd", AppName)
	files, err := os.ReadDir(sysoDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading directory %s: %v", sysoDir, err)
		}
		return
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".syso") {
			filePath := filepath.Join(sysoDir, file.Name())
			log.Printf("Removing %s", filePath)
			if err := os.Remove(filePath); err != nil {
				log.Printf("Error removing %s: %v", filePath, err)
			}
		}
	}
}

func setWailsIconResourceID(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 20 {
		return fmt.Errorf("invalid COFF resource header")
	}
	sections := int(binary.LittleEndian.Uint16(data[2:4]))
	start := 20 + int(binary.LittleEndian.Uint16(data[16:18]))
	for index := 0; index < sections; index++ {
		offset := start + index*40
		if offset+40 > len(data) {
			return fmt.Errorf("invalid COFF section table")
		}
		section := data[offset : offset+40]
		if string(bytes.TrimRight(section[:8], "\x00")) != ".rsrc" {
			continue
		}
		position := int(binary.LittleEndian.Uint32(section[20:24]))
		size := int(binary.LittleEndian.Uint32(section[16:20]))
		if position < 0 || size < 16 || position > len(data)-size {
			return fmt.Errorf("invalid resource section")
		}
		resources := data[position : position+size]
		count := int(binary.LittleEndian.Uint16(resources[12:14])) + int(binary.LittleEndian.Uint16(resources[14:16]))
		for entry := 0; entry < count; entry++ {
			at := 16 + entry*8
			if at+8 > len(resources) {
				return fmt.Errorf("invalid resource directory")
			}
			if binary.LittleEndian.Uint32(resources[at:at+4]) != 14 {
				continue
			}
			child := binary.LittleEndian.Uint32(resources[at+4 : at+8])
			if child&0x80000000 == 0 {
				return fmt.Errorf("icon group directory missing")
			}
			group := int(child & 0x7fffffff)
			if group > len(resources)-24 {
				return fmt.Errorf("invalid icon group directory")
			}
			if binary.LittleEndian.Uint16(resources[group+12:group+14]) != 0 || binary.LittleEndian.Uint16(resources[group+14:group+16]) != 1 {
				return fmt.Errorf("expected one numeric icon group")
			}
			binary.LittleEndian.PutUint32(resources[group+16:group+20], 3)
			return os.WriteFile(path, data, 0644)
		}
	}
	return fmt.Errorf("icon group resource missing")
}

func libBuildCmd(buildTag string, build Build, ldflags, tags []string) (string, []string) {
	outputFilename := fmt.Sprintf("libCraftBox-%s-%s-%s", build.OS, build.Arch, buildTag)
	outputFilename += build.LibExt()
	outputPath := filepath.Join("builds", outputFilename)

	return outputPath, []string{
		"go", "build",
		"-buildmode", "c-shared",
		"-ldflags", strings.Join(ldflags, " "),
		"-tags", strings.Join(tags, ","),
		"-o", outputPath,
		"-v",
		"-trimpath",
		"./cmd/libCraftBox",
	}
}

func cliBuildCmd(buildTag string, build Build, ldflags, tags []string) (string, []string) {
	outputFilename := fmt.Sprintf("CraftBox-%s-%s-%s", build.OS, build.Arch, buildTag)
	outputFilename += build.ExeExt()
	outputPath := filepath.Join("builds", outputFilename)

	return outputPath, []string{
		"go", "build",
		"-ldflags", strings.Join(ldflags, " "),
		"-tags", strings.Join(tags, ","),
		"-o", outputPath,
		"-v",
		"-trimpath",
		"./cmd/CraftBox",
	}
}

func guiBuildCmd(buildTag string, build Build, ldflags, tags []string) (string, []string) {
	outputFilename := fmt.Sprintf("CraftBox-gui-%s-%s-%s", build.OS, build.Arch, buildTag)
	outputFilename += build.ExeExt()
	outputPath := filepath.Join("builds", outputFilename)

	if build.OS != "android" {
		if build.OS == "linux" {
			tags = append(tags, "webkit2_41")
		}
		return outputPath, []string{
			"go", "build",
			"-ldflags", strings.Join(ldflags, " "),
			"-tags", strings.Join(tags, ","),
			"-o", outputPath,
			"-v",
			"-trimpath",
			"./cmd/CraftBox",
		}
	}

	buildCmd := []string{
		gogioCmd,
	}
	if build.OS == "android" {
		buildCmd = append(buildCmd, "-target", build.OS)
		buildCmd = append(buildCmd, "-appid", AppID)
		buildCmd = append(buildCmd, "-resources", "./android-resources")
	}
	tagSplit := strings.Split(strings.ReplaceAll(buildTag, "-", "."), ".")
	tagSplit = tagSplit[:len(tagSplit)-1]
	if len(tagSplit) < 4 {
		tagSplit = append(tagSplit, "0")
	}
	if build.OS == "android" {
		tagSplit[3] = strings.Join(tagSplit, "")
	}
	gioVersion := strings.Join(tagSplit, ".")
	gioTarget := build.OS
	if build.OS == "darwin" {
		gioTarget = "macos"
	}

	buildCmd = append(buildCmd,
		"-arch", build.Arch,
		"-target", gioTarget,
		"-version", gioVersion,
		"-icon", "icon.png",
		"-tags", strings.Join(tags, ","),
		"-ldflags", strings.Join(ldflags, " "),
		"-o", outputPath,
		"-x",
		"./cmd/CraftBox",
	)

	return outputPath, buildCmd
}

func runCommand(buildCmd, env []string, cwd string) error {
	log.Printf("Executing: %s", strings.Join(buildCmd, " "))
	cmd := exec.Command(buildCmd[0], buildCmd[1:]...)
	cmd.Env = env
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build command failed: %w", err)
	}
	return nil
}

func androidEnv(build Build, env *[]string) error {
	ndkHome := os.Getenv("ANDROID_NDK_HOME")
	if ndkHome == "" {
		log.Println("Error: ANDROID_NDK_HOME environment variable not set. Skipping Android build.")
		return nil
	}
	clangCompilerPath, err := findAndroidToolchain(ndkHome, build.Arch, AndroidAPILevel)
	if err != nil {
		log.Printf("Error finding Android NDK toolchain: %v. Skipping Android build.", err)
		return nil
	}
	log.Printf("Using Android NDK compiler: %s", clangCompilerPath)
	*env = append(*env, "CC="+clangCompilerPath)
	*env = append(*env, "CXX="+clangCompilerPath)
	*env = append(*env, "CGO_ENABLED=1")
	return nil
}

func (c *BuildConfig) doBuild(build Build) error {
	buildName := fmt.Sprintf("%s-%s-%s", build.OS, build.Arch, build.Type)
	log.Printf("\n--- Building %s ---", buildName)

	env := os.Environ()
	env = append(env, "GOVCS=*:off")
	env = append(env, "GOOS="+build.OS)
	env = append(env, "GOARCH="+build.Arch)

	var tags []string
	if c.PackSupportEnabled {
		tags = append(tags, "packs")
	}
	if build.Type == "gui" {
		tags = append(tags, "gui")
		if build.OS != "android" {
			tags = append(tags, "production")
		}
	}

	ldflags := []string{"-s", "-w"}
	ldflags = append(ldflags, fmt.Sprintf("-X github.com/Daniel-Ric/%s/utils.Version=%s", AppName, c.BuildTag))

	cmdName := AppName
	if build.Type != "cli" {
		cmdName += "-" + build.Type
	}
	ldflags = append(ldflags, fmt.Sprintf("-X github.com/Daniel-Ric/%s/utils.CmdName=%s", AppName, cmdName))

	var outputPath string
	var buildCmd []string
	switch build.OS {
	case "android":
		err := androidEnv(build, &env)
		if err != nil {
			return err
		}
		switch build.Type {
		case "lib":
			outputPath, buildCmd = libBuildCmd(c.BuildTag, build, ldflags, tags)
		case "gui":
			outputPath, buildCmd = guiBuildCmd(c.BuildTag, build, ldflags, tags)
		default:
			return fmt.Errorf("%s not supported on android", build.Type)
		}

	case "windows", "linux", "darwin":
		switch build.Type {
		case "lib":
			outputPath, buildCmd = libBuildCmd(c.BuildTag, build, ldflags, tags)
		case "gui":
			if build.OS == "windows" {
				cleanSyso()
				if err := prepareWindowsResources(build.Arch); err != nil {
					return err
				}
				ldflags = append(ldflags, "-H=windows")
			}
			outputPath, buildCmd = guiBuildCmd(c.BuildTag, build, ldflags, tags)
		case "cli":
			outputPath, buildCmd = cliBuildCmd(c.BuildTag, build, ldflags, tags)
		default:
			return fmt.Errorf("unknown build type %s", build.Type)
		}

	default:
		return fmt.Errorf("unknown os %s", build.OS)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	err := runCommand(buildCmd, env, "")
	if err != nil {
		return err
	}

	log.Printf("Successfully built: %s", outputPath)

	if build.Type == "gui" && build.OS == "windows" {
		cleanSyso()
	}

	return nil
}

func buildGogio() error {
	var env []string
	env = append(env, os.Environ()...)
	err := runCommand([]string{
		"go", "build",
		"-o", gogioCmd,
		"-v",
		"-trimpath",
		"./gio-cmd/gogio",
	}, env, "")
	if err != nil {
		return err
	}
	return nil
}

func main() {
	log.SetFlags(0)

	if len(os.Args) > 1 && os.Args[1] == "--windows-resources" {
		arch := "amd64"
		if len(os.Args) > 2 {
			arch = os.Args[2]
		}
		if err := prepareWindowsResources(arch); err != nil {
			log.Fatal(err)
		}
		return
	}
	log.Println("--- Starting Build Process ---")

	if false {
		if err := buildGogio(); err != nil {
			log.Fatalf("building gogio %s", err)
		}
	}
	gogioCmd = "gogio"

	buildCfg := &BuildConfig{}

	githubOutputPath := os.Getenv("GITHUB_OUTPUT")
	if githubOutputPath != "" {
		var err error
		buildCfg.GitHubOutputFile, err = os.OpenFile(githubOutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Warning: Could not open GITHUB_OUTPUT file %s: %v", githubOutputPath, err)
		} else {
			defer buildCfg.GitHubOutputFile.Close()
		}
	} else {
		log.Println("GITHUB_OUTPUT environment variable not set. Writing outputs to stderr.")
	}

	log.Println("Cleaning existing builds directory...")
	if err := os.RemoveAll("builds"); err != nil && !os.IsNotExist(err) {
		log.Printf("Error cleaning builds directory: %v", err)
	}
	if err := os.MkdirAll("builds", 0755); err != nil {
		log.Fatalf("Error creating builds directory: %v", err)
	}

	if err := buildCfg.getVersion(); err != nil {
		log.Fatalf("Failed to get version information: %v", err)
	}
	log.Printf("App Version: %s", buildCfg.AppVersion)
	log.Printf("Build Tag: %s", buildCfg.BuildTag)

	var err error
	buildCfg.PackSupportEnabled, err = checkPackSupport()
	if err != nil {
		log.Fatalf("Failed to check pack support: %v", err)
	}
	log.Printf("Pack Support Enabled: %t", buildCfg.PackSupportEnabled)

	allBuilds := []Build{

		{"windows", "amd64", "gui"},
		{"linux", "amd64", "gui"},

		{"darwin", "amd64", "gui"},
		{"darwin", "arm64", "gui"},

		{"windows", "amd64", "cli"},
		{"linux", "amd64", "cli"},
		{"linux", "arm64", "cli"},
		{"linux", "arm", "cli"},
		{"darwin", "amd64", "cli"},
		{"darwin", "arm64", "cli"},
	}

	buildsToRun := allBuilds
	var selectedOSes []string
	var selectedTypes []string
	if len(os.Args) > 1 {
		selectedOSes = strings.Split(strings.ToLower(os.Args[1]), ",")
		if len(os.Args) > 2 {
			selectedTypes = strings.Split(strings.ToLower(os.Args[2]), ",")
		}
	}

	shouldRunBuild := func(build Build) bool {
		if len(selectedOSes) > 0 && !slices.Contains(selectedOSes, build.OS) {
			return false
		}
		if len(selectedTypes) > 0 && !slices.Contains(selectedTypes, build.Type) {
			return false
		}
		return true
	}

	if len(selectedOSes)+len(selectedTypes) == 0 {
		log.Println("Building all configurations...")
	}

	frontendBuilt := false
	for _, build := range buildsToRun {
		if !shouldRunBuild(build) {
			continue
		}
		if build.OS == "linux" && build.Type != "cli" && runtime.GOOS == "windows" {
			log.Printf("Skipping Linux GUI build on Windows host.")
			continue
		}
		if build.OS == "android" && os.Getenv("ANDROID_NDK_HOME") == "" {
			log.Printf("Skipping Android build for %s because ANDROID_NDK_HOME is not set.", build.Arch)
			continue
		}
		if build.Type == "gui" && build.OS != "android" && !frontendBuilt {
			npm := "npm"
			if runtime.GOOS == "windows" {
				npm = "npm.cmd"
			}
			frontendDir := filepath.Join("ui", "desktop", "frontend")
			for _, args := range [][]string{{"ci"}, {"run", "build"}} {
				if err := runCommand(append([]string{npm}, args...), os.Environ(), frontendDir); err != nil {
					log.Fatalf("Frontend build failed: %v", err)
				}
			}
			frontendBuilt = true
		}
		if err := buildCfg.doBuild(build); err != nil {
			log.Fatalf("Build failed for %s-%s-%s: %v", build.OS, build.Arch, build.Type, err)
		}
	}

	changelogTagName := "v" + strings.Split(buildCfg.AppVersion, "-")[0]
	changelog, err := generateChangelog(changelogTagName)
	if err != nil {
		log.Printf("Warning: Failed to generate changelog: %v", err)
		changelog = fmt.Sprintf("Failed to generate changelog: %v", err)
	}

	log.Println("\n--- Changelog ---")
	fmt.Println(changelog)

	changelogFilePath := "changelog.txt"
	header := fmt.Sprintf("## Commits since %s\n", changelogTagName)
	if err := os.WriteFile(changelogFilePath, []byte(header+changelog), 0644); err != nil {
		log.Printf("Error writing changelog to file %s: %v", changelogFilePath, err)
	} else {
		log.Printf("Changelog saved to %s", changelogFilePath)
	}

	log.Println("\n--- Build Process Finished ---")
}
