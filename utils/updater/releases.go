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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	CurrentRelease *Update `json:"currentRelease,omitempty"`
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
	result := &ReleaseCheck{CurrentVersion: current, Channel: channel, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	var selected *githubRelease
	for page := 1; ; page++ {
		releases, next, err := releasePage(ctx, endpoint, channel == "preview")
		if err != nil {
			return nil, err
		}
		for i := range releases {
			r := &releases[i]
			if !r.Draft && cleanVersion(r.TagName) == cleanVersion(current) {
				result.CurrentRelease = releaseIdentity(r)
			}
			if r.Draft || !validVersion(r.TagName) {
				continue
			}
			if channel == "stable" && r.Prerelease || channel == "preview" && (!r.Prerelease || !strings.Contains(cleanVersion(r.TagName), "-preview.")) {
				continue
			}
			if selected == nil || compareVersions(r.TagName, selected.TagName) > 0 {
				selected = r
			}
		}
		if !next || selected != nil || channel == "stable" {
			break
		}
		endpoint = fmt.Sprintf("https://api.github.com/repos/Daniel-Ric/CraftBox/releases?per_page=100&page=%d", page+1)
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
		if runtime.GOOS == "darwin" && strings.HasSuffix(asset.Name, ".app.zip") {
			update.UnavailableReason = "Download the macOS app from the release to install this update."
			break
		}
		update.Installable = true
		update.UnavailableReason = ""
		break
	}
	result.Release = update
	result.Available = newerVersion(current, update.Version)
	if cleanVersion(current) == update.Version {
		result.CurrentRelease = update
	} else if result.CurrentRelease == nil && validVersion(current) && !developmentVersion(current) {
		for _, prefix := range []string{"v", "r"} {
			installed := installedRelease(ctx, prefix+cleanVersion(current))
			if installed != nil {
				result.CurrentRelease = installed
				break
			}
		}
	}
	return result, nil
}

type cachedReleasePage struct {
	releases []githubRelease
	next     bool
	expires  time.Time
}

var releaseRequests = struct {
	sync.Mutex
	pages   map[string]cachedReleasePage
	retryAt time.Time
}{}

func releasePage(ctx context.Context, endpoint string, list bool) ([]githubRelease, bool, error) {
	releaseRequests.Lock()
	defer releaseRequests.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	now := time.Now()
	if cached, ok := releaseRequests.pages[endpoint]; ok && now.Before(cached.expires) {
		return cached.releases, cached.next, nil
	}
	if now.Before(releaseRequests.retryAt) {
		return nil, false, fmt.Errorf("GitHub API rate limit reached; try again after %s or open the releases page", releaseRequests.retryAt.Local().Format("15:04:05 MST"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "CraftBox/"+utils.Version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		cacheReleasePage(endpoint, nil, false)
		return nil, false, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden && (resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != "") {
		now = time.Now()
		retryAt := now.Add(time.Minute)
		if seconds, err := strconv.ParseInt(resp.Header.Get("Retry-After"), 10, 64); err == nil && seconds > 0 && seconds <= 86400 {
			retryAt = now.Add(time.Duration(seconds) * time.Second)
		} else if date, err := http.ParseTime(resp.Header.Get("Retry-After")); err == nil && date.After(now) {
			retryAt = date
		}
		if reset, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil && resp.Header.Get("X-RateLimit-Remaining") == "0" && time.Unix(reset, 0).After(retryAt) {
			retryAt = time.Unix(reset, 0)
		}
		releaseRequests.retryAt = retryAt
		return nil, false, fmt.Errorf("GitHub API rate limit reached; try again after %s or open the releases page", retryAt.Local().Format("15:04:05 MST"))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("GitHub update check returned HTTP %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 4<<20))
	var releases []githubRelease
	if list {
		err = decoder.Decode(&releases)
	} else {
		var release githubRelease
		err = decoder.Decode(&release)
		releases = []githubRelease{release}
	}
	if err != nil {
		return nil, false, fmt.Errorf("invalid GitHub release response: %w", err)
	}
	next := strings.Contains(resp.Header.Get("Link"), `rel="next"`)
	cacheReleasePage(endpoint, releases, next)
	return releases, next, nil
}

func cacheReleasePage(endpoint string, releases []githubRelease, next bool) {
	if releaseRequests.pages == nil {
		releaseRequests.pages = make(map[string]cachedReleasePage)
	}
	for key, cached := range releaseRequests.pages {
		if time.Now().After(cached.expires) {
			delete(releaseRequests.pages, key)
		}
	}
	releaseRequests.pages[endpoint] = cachedReleasePage{releases: releases, next: next, expires: time.Now().Add(time.Minute)}
}

func installedRelease(ctx context.Context, tag string) *Update {
	releases, _, err := releasePage(ctx, "https://api.github.com/repos/Daniel-Ric/CraftBox/releases/tags/"+url.PathEscape(tag), false)
	if err != nil || len(releases) == 0 || releases[0].Draft || cleanVersion(releases[0].TagName) != cleanVersion(tag) {
		return nil
	}
	return releaseIdentity(&releases[0])
}

func releaseIdentity(release *githubRelease) *Update {
	link := release.HTMLURL
	if !repositoryURL(link, "/releases/") {
		link = RepositoryURL + "/releases"
	}
	return &Update{Version: cleanVersion(release.TagName), Name: release.Name, ReleaseURL: link}
}

var legacyBuild = regexp.MustCompile(`^([0-9]+)-[a-fA-F0-9]{7,40}$`)

func developmentVersion(v string) bool {
	return strings.HasSuffix(v, "-dev") || strings.Contains(v, "-dev.") || strings.TrimSpace(v) == ""
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
	if aBuild, bBuild := legacyBuild.FindStringSubmatch(ap), legacyBuild.FindStringSubmatch(bp); aBuild != nil && bBuild != nil {
		aNumber, _ := strconv.ParseUint(aBuild[1], 10, 64)
		bNumber, _ := strconv.ParseUint(bBuild[1], 10, 64)
		if aNumber > bNumber {
			return 1
		}
		if aNumber < bNumber {
			return -1
		}
		return 0
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

	if developmentVersion(current) && validVersion(next) && !developmentVersion(next) {
		base := strings.SplitN(cleanVersion(current), "-", 2)[0]
		return current == "" || compareVersions(strings.SplitN(cleanVersion(next), "-", 2)[0], base) >= 0
	}
	return validVersion(current) && validVersion(next) && compareVersions(next, current) > 0
}
