package assetlibrary

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func InstalledMinecraft() ([]Installation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", `[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new(); $result=@(Get-AppxPackage | Where-Object {$_.Name -match '^Microsoft\.(MinecraftUWP|MinecraftWindowsBeta|MinecraftWindows|MinecraftPreview)$'} | ForEach-Object {$p=$_; $a=''; try {$m=$p | Get-AppxPackageManifest; $a=@($m.Package.Applications.Application)[0].Id} catch {}; [pscustomobject]@{Name=$p.Name;Version=$p.Version.ToString();InstallLocation=$p.InstallLocation;PackageFamilyName=$p.PackageFamilyName;Application=$a}}); ConvertTo-Json -InputObject $result -Compress`)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	raw, e := cmd.Output()
	var packages []MinecraftPackage
	if e == nil {
		e = json.Unmarshal(raw, &packages)
	}
	for drive := 'C'; drive <= 'Z'; drive++ {
		for _, game := range []string{"Minecraft for Windows", "Minecraft Preview for Windows"} {
			root := filepath.Join(string(drive)+":\\", "XboxGames", game, "Content")
			b, readErr := readLimit(filepath.Join(root, "MicrosoftGame.config"), 1<<20)
			if readErr != nil {
				continue
			}
			var config struct {
				Identity struct {
					Name    string `xml:"Name,attr"`
					Version string `xml:"Version,attr"`
				} `xml:"Identity"`
				Protocols []struct {
					Name string `xml:"Name,attr"`
				} `xml:"ProtocolList>Protocol"`
			}
			if xml.Unmarshal(b, &config) != nil {
				continue
			}
			found := false
			for _, p := range packages {
				if strings.EqualFold(p.InstallLocation, root) {
					found = true
				}
			}
			if found {
				continue
			}
			application := ""
			for _, p := range config.Protocols {
				if p.Name == "minecraft" || p.Name == "minecraft-preview" {
					application = "uri:" + p.Name
					break
				}
			}
			name := config.Identity.Name
			if strings.Contains(game, "Preview") {
				name = "Microsoft.MinecraftPreview"
			}
			packages = append(packages, MinecraftPackage{Name: name, Version: config.Identity.Version, InstallLocation: root, Application: application})
		}
	}
	items := DiscoverMinecraft(os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA"), packages)
	return items, e
}
