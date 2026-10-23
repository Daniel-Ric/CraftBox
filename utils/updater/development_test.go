package updater

import (
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/Daniel-Ric/CraftBox/utils"
)

type countingTransport struct{ requests int }

func (t *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	t.requests++
	return nil, errors.New("unexpected release request from local test build")
}

func TestDevelopmentBuildDoesNotCheckOfficialRelease(t *testing.T) {
	previousVersion, previousClient := utils.Version, http.DefaultClient
	t.Cleanup(func() { utils.Version = previousVersion; http.DefaultClient = previousClient })
	utils.Version = "1.62.0-nethernet-dev"
	transport := &countingTransport{}
	http.DefaultClient = &http.Client{Transport: transport}
	CheckUpdate()
	if transport.requests != 0 {
		t.Fatal("local test build queried official releases")
	}
}

type releaseTransport func(*http.Request) (*http.Response, error)

func (f releaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLatestUpdateUsesOwnRepository(t *testing.T) {
	previousClient, previousName := http.DefaultClient, utils.CmdName
	defer func() { http.DefaultClient = previousClient; utils.CmdName = previousName }()
	utils.CmdName = "CraftBox-gui"
	http.DefaultClient = &http.Client{Transport: releaseTransport(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.String(); got != "https://api.github.com/repos/Daniel-Ric/CraftBox/releases/latest" {
			t.Fatalf("unexpected update repository: %s", got)
		}
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		body := `{"tag_name":"r1.62.0","assets":[{"name":"CraftBox-gui-` + runtime.GOOS + `-` + runtime.GOARCH + `-1.62.0` + suffix + `","digest":"sha256:` + strings.Repeat("ab", 32) + `","browser_download_url":"https://github.com/Daniel-Ric/CraftBox/releases/download/r1.62.0/test.exe"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	update, err := latestUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if update.Version != "1.62.0" || !strings.HasPrefix(update.DownloadURL, "https://github.com/Daniel-Ric/CraftBox/") {
		t.Fatalf("wrong update: %#v", update)
	}
}
