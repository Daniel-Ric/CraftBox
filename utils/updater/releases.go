package updater

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils"
)

const RepositoryURL = "https://github.com/Daniel-Ric/CraftBox"

type ReleaseCheck struct {
	CurrentVersion string  `json:"currentVersion"`
	Channel        string  `json:"channel"`
	CheckedAt      string  `json:"checkedAt"`
	Available      bool    `json:"available"`
	Release        *Update `json:"release,omitempty"`
}

func CheckRelease(ctx context.Context, channel, current string) (*ReleaseCheck, error) {
	endpoint := githubLatestReleaseURL
	if channel == "preview" {
		endpoint = "https://api.github.com/repos/Daniel-Ric/CraftBox/releases?per_page=100"
	} else if channel != "stable" {
		return nil, fmt.Errorf("unknown update channel %q", channel)
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "CraftBox/"+utils.Version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer resp.Body.Close()
	result := &ReleaseCheck{CurrentVersion: current, Channel: channel, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	if resp.StatusCode == http.StatusNotFound {
		return result, nil
	}
	if resp.StatusCode == 429 || (resp.StatusCode == 403 && resp.Header.Get("X-RateLimit-Remaining") == "0") {
		return nil, fmt.Errorf("GitHub API rate limit reached; try again later or open the releases page")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub update check returned HTTP %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 4<<20))
	var releases []githubRelease
	if channel == "preview" {
		err = decoder.Decode(&releases)
	} else {
		var release githubRelease
		err = decoder.Decode(&release)
		releases = []githubRelease{release}
	}
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub release response: %w", err)
	}
	var selected *githubRelease
	for i := range releases {
		r := &releases[i]
		if r.Draft || (channel == "stable" && r.Prerelease) || !validVersion(r.TagName) {
			continue
		}
		if selected == nil || compareVersions(r.TagName, selected.TagName) > 0 {
			selected = r
		}
	}
	if selected == nil {
		return result, nil
	}
	release := selected
	update := &Update{Version: cleanVersion(release.TagName), Name: release.Name, Notes: release.Body, PublishedAt: release.PublishedAt, ReleaseURL: release.HTMLURL, Prerelease: release.Prerelease, UnavailableReason: "No compatible download is published for this operating system and architecture."}
	if update.ReleaseURL == "" || !repositoryURL(update.ReleaseURL, "/releases/") {
		update.ReleaseURL = RepositoryURL + "/releases"
	}
	prefix := fmt.Sprintf("%s-%s-%s-", utils.CmdName, runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if !strings.HasPrefix(asset.Name, prefix) || (runtime.GOOS == "windows" && !strings.HasSuffix(asset.Name, ".exe")) {
			continue
		}
		update.AssetName = asset.Name
		update.Size = asset.Size
		update.UnavailableReason = "This release asset has no valid SHA-256 digest. Open its GitHub release for details."
		digest, err := hex.DecodeString(strings.TrimPrefix(asset.Digest, "sha256:"))
		if !strings.HasPrefix(asset.Digest, "sha256:") || err != nil || len(digest) != 32 {
			continue
		}
		if !repositoryURL(asset.BrowserDownloadURL, "/releases/download/") {
			update.UnavailableReason = "The download URL is outside the CraftBox repository."
			continue
		}
		update.DownloadURL = asset.BrowserDownloadURL
		update.Sha256 = base64.StdEncoding.EncodeToString(digest)
		update.Installable = true
		update.UnavailableReason = ""
		break
	}
	result.Release = update
	result.Available = newerVersion(current, update.Version)
	return result, nil
}

func repositoryURL(value, path string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && u.Host == "github.com" && u.User == nil && strings.HasPrefix(u.Path, "/Daniel-Ric/CraftBox"+path)
}
func cleanVersion(v string) string {
	return strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(v), "r"), "v")
}
func versionParts(v string) ([]uint64, string, bool) {
	v = strings.SplitN(cleanVersion(v), "+", 2)[0]
	parts := strings.SplitN(v, "-", 2)
	core := strings.Split(parts[0], ".")
	if len(core) != 3 {
		return nil, "", false
	}
	numbers := make([]uint64, 3)
	for i, p := range core {
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil || p == "" || (len(p) > 1 && p[0] == '0') {
			return nil, "", false
		}
		numbers[i] = n
	}
	pre := ""
	if len(parts) == 2 {
		pre = parts[1]
		if pre == "" {
			return nil, "", false
		}
		for _, c := range pre {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-') {
				return nil, "", false
			}
		}
	}
	return numbers, pre, true
}
func validVersion(v string) bool { _, _, ok := versionParts(v); return ok }
func compareVersions(a, b string) int {
	an, ap, aok := versionParts(a)
	bn, bp, bok := versionParts(b)
	if !aok || !bok {
		return 0
	}
	for i := range an {
		if an[i] > bn[i] {
			return 1
		}
		if an[i] < bn[i] {
			return -1
		}
	}
	if ap == bp {
		return 0
	}
	if ap == "" {
		return 1
	}
	if bp == "" {
		return -1
	}
	aa, bb := strings.Split(ap, "."), strings.Split(bp, ".")
	for i := 0; i < len(aa) && i < len(bb); i++ {
		if aa[i] == bb[i] {
			continue
		}
		ai, ae := strconv.ParseUint(aa[i], 10, 64)
		bi, be := strconv.ParseUint(bb[i], 10, 64)
		if ae == nil && be == nil {
			if ai > bi {
				return 1
			}
			return -1
		}
		if ae == nil {
			return -1
		}
		if be == nil {
			return 1
		}
		return strings.Compare(aa[i], bb[i])
	}
	if len(aa) > len(bb) {
		return 1
	}
	return -1
}
func newerVersion(current, next string) bool {

	if strings.HasSuffix(current, "-dev") && strings.SplitN(cleanVersion(current), "-", 2)[0] == cleanVersion(next) {
		return false
	}
	return validVersion(current) && validVersion(next) && compareVersions(next, current) > 0
}
