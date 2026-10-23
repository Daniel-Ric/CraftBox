package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCIVersionOverridesTaglessHistory(t *testing.T) {
	t.Setenv("CRAFTBOX_VERSION", "10.2.3")
	t.Setenv("CRAFTBOX_BUILD_TAG", "10.2.3-17-abcdef0")
	output, err := os.Create(filepath.Join(t.TempDir(), "github-output"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { output.Close() })
	config := BuildConfig{GitHubOutputFile: output}
	if err := config.getVersion(); err != nil {
		t.Fatal(err)
	}
	if config.AppVersion != "10.2.3" || config.BuildTag != "10.2.3-17-abcdef0" || !config.IsLatest {
		t.Fatalf("unexpected CI version: %+v", config)
	}
	data, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"release_tag=r10.2.3-17-abcdef0\n", "build_tag=10.2.3-17-abcdef0\n", "is_latest=true\n"} {
		if !strings.Contains(string(data), value) {
			t.Fatalf("missing output %q in %q", value, data)
		}
	}
}

func TestCIVersionKeepsDevelopmentBuildsSeparate(t *testing.T) {
	t.Setenv("CRAFTBOX_VERSION", "1.0.0")
	t.Setenv("CRAFTBOX_BUILD_TAG", "1.0.0-17-abcdef0-dev")
	config := BuildConfig{}
	if err := config.getVersion(); err != nil {
		t.Fatal(err)
	}
	if config.BuildTag != "1.0.0-17-abcdef0-dev" || config.IsLatest {
		t.Fatalf("development build classified as latest: %+v", config)
	}
}

func TestCIVersionRejectsInvalidMetadata(t *testing.T) {
	for _, tc := range []struct{ version, build string }{
		{"1.0", ""},
		{"01.0.0", ""},
		{"1.0.0\nother=value", ""},
		{"1.0.0", "2.0.0-17-abcdef0"},
		{"1.0.0", "1.0.0-17\nother=value"},
	} {
		t.Run(tc.version+"/"+tc.build, func(t *testing.T) {
			t.Setenv("CRAFTBOX_VERSION", tc.version)
			t.Setenv("CRAFTBOX_BUILD_TAG", tc.build)
			if err := (&BuildConfig{}).getVersion(); err == nil {
				t.Fatal("invalid version metadata accepted")
			}
		})
	}
}

func TestWindowsResourcesRejectUnsupportedArchitecture(t *testing.T) {
	if err := prepareWindowsResources("invalid"); err == nil {
		t.Fatal("unsupported resource architecture accepted")
	}
}

func TestWailsIconResourceKeepsImageReferences(t *testing.T) {

	data := make([]byte, 132)
	binary.LittleEndian.PutUint16(data[2:4], 1)
	copy(data[20:28], ".rsrc")
	binary.LittleEndian.PutUint32(data[36:40], 72)
	binary.LittleEndian.PutUint32(data[40:44], 60)
	resources := data[60:]
	binary.LittleEndian.PutUint16(resources[14:16], 1)
	binary.LittleEndian.PutUint32(resources[16:20], 14)
	binary.LittleEndian.PutUint32(resources[20:24], 0x80000018)
	binary.LittleEndian.PutUint16(resources[38:40], 1)
	binary.LittleEndian.PutUint32(resources[40:44], 2)
	binary.LittleEndian.PutUint32(resources[44:48], 0x80000030)
	resources[64] = 7
	file := filepath.Join(t.TempDir(), "icon.syso")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := setWailsIconResourceID(file); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(actual[100:104]) != 3 || actual[124] != 7 {
		t.Fatal("Wails icon ID or image data incorrect")
	}
}
