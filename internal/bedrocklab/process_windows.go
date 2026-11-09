package bedrocklab

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
)

func hideCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}

func CopyReadableExecutable(ctx context.Context, source, root string) (string, error) {
	info, e := os.Stat(source)
	if e != nil {
		return "", e
	}
	if !info.Mode().IsRegular() || info.Size() > 2<<30 {
		return "", fmt.Errorf("unsupported executable size")
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		return "", e
	}
	work, e := os.MkdirTemp(root, "readable-*")
	if e != nil {
		return "", e
	}
	target := filepath.Join(work, "Minecraft.Windows.exe")
	pending := target + ".pending"
	errorFile := filepath.Join(work, "error.txt")
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := "$ErrorActionPreference='Stop'; try { Copy-Item -LiteralPath " + quote(source) + " -Destination " + quote(pending) + "; Move-Item -LiteralPath " + quote(pending) + " -Destination " + quote(target) + " } catch { [IO.File]::WriteAllText(" + quote(errorFile) + ", $_.Exception.Message) }"
	wide := utf16.Encode([]rune(script))
	encoded := make([]byte, len(wide)*2)
	for i, v := range wide {
		binary.LittleEndian.PutUint16(encoded[i*2:], v)
	}
	command := `$ErrorActionPreference='Stop'; $p=Get-AppxPackage | Where-Object {$_.InstallLocation -eq $env:CRAFTBOX_GAME_LOCATION} | Select-Object -First 1; if (!$p) {throw 'Minecraft package identity is unavailable. Choose an existing readable executable instead.'}; $m=$p | Get-AppxPackageManifest; $id=@($m.Package.Applications.Application)[0].Id; Invoke-CommandInDesktopPackage -PackageFamilyName $p.PackageFamilyName -AppId $id -Command "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe" -Args ('-NoLogo -NoProfile -NonInteractive -WindowStyle Hidden -EncodedCommand '+$env:CRAFTBOX_COPY_SCRIPT)`
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command)
	hideCommand(cmd)
	cmd.Env = append(os.Environ(), "CRAFTBOX_GAME_LOCATION="+filepath.Dir(source), "CRAFTBOX_COPY_SCRIPT="+base64.StdEncoding.EncodeToString(encoded))
	if b, e := cmd.CombinedOutput(); e != nil {
		return "", fmt.Errorf("readable copy failed: %s (%w)", strings.TrimSpace(string(b)), e)
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if b, e := os.ReadFile(errorFile); e == nil {
				return "", fmt.Errorf("Minecraft could not copy its executable: %s", string(b))
			}
			if copied, e := os.Stat(target); e == nil {
				if copied.Size() != info.Size() {
					return "", fmt.Errorf("readable copy size mismatch")
				}
				f, e := os.Open(target)
				if e != nil {
					return "", e
				}
				signature := make([]byte, 2)
				_, e = f.Read(signature)
				f.Close()
				if e != nil || string(signature) != "MZ" {
					return "", fmt.Errorf("readable copy is not a Windows executable")
				}
				return target, nil
			}
		}
	}
}
