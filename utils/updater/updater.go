package updater

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/minio/selfupdate"
	"github.com/sirupsen/logrus"
)

type progressWriter struct {
	OnProgress func(percent int)
	percent    int
	done       int
	Total      int
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.done += len(b)
	percent := -1
	if p.Total > 0 {
		percent = min(100, (p.done*100)/p.Total)
	}
	if p.percent != percent {
		p.OnProgress(percent)
		p.percent = percent
	}
	return len(b), nil
}

const githubLatestReleaseURL = "https://api.github.com/repos/Daniel-Ric/CraftBox/releases/latest"

func fetchHttp(url string) (io.ReadCloser, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		cancel()
		return nil, 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		return nil, 0, err
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		cancel()
		return nil, 0, fmt.Errorf("bad http status from %s: %v", url, resp.Status)
	}
	return &cancelReadCloser{ReadCloser: resp.Body, cancel: cancel}, int(resp.ContentLength), nil
}

type cancelReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelReadCloser) Close() error { defer r.cancel(); return r.ReadCloser.Close() }

func CheckUpdate() {

	if developmentVersion(utils.Version) {
		return
	}
	err := func() error {
		update, err := GetLatest()
		if err != nil {
			return err
		}
		isNew := isNewVersion(update.Version)
		if isNew {
			logrus.Info(locale.Loc("update_available", locale.Strmap{"Version": update.Version}))
			messages.SendEvent(&messages.EventUpdateAvailable{
				Version: update.Version,
			})
		}
		return nil
	}()
	if err != nil {
		logrus.Error(err)
	}
}

func GetLatest() (*Update, error) {
	if runtime.GOOS == "android" || runtime.GOOS == "js" {
		return &Update{
			Version: utils.Version,
		}, nil
	}

	update, err := latestUpdate()
	if err != nil {
		return nil, err
	}
	return update, nil
}

func DoUpdate(update *Update) error {
	if update == nil || !update.Installable || !repositoryURL(update.DownloadURL, "/releases/download/") || !update.IsNew() {
		return fmt.Errorf("no verified newer CraftBox release is selected")
	}
	checksum, err := base64.StdEncoding.DecodeString(update.Sha256)
	if err != nil || len(checksum) != sha256.Size {
		return fmt.Errorf("update has no valid SHA-256 checksum")
	}

	r, size, err := fetchHttp(update.DownloadURL)
	if err != nil {
		return err
	}
	defer r.Close()

	f, err := os.CreateTemp(utils.PathCache(""), "CraftBox-update-*.bin")
	if err != nil {
		return err
	}
	updatePath := f.Name()
	defer func() {
		f.Close()
		os.Remove(updatePath)
	}()

	sum := sha256.New()
	mw := io.MultiWriter(f, sum, &progressWriter{
		OnProgress: func(percent int) {
			messages.SendEvent(&messages.EventUpdateDownloadProgress{
				Progress: percent,
			})
		},
		Total: size,
	})
	_, err = io.Copy(mw, r)
	if err != nil {
		return err
	}

	if !bytes.Equal(checksum, sum.Sum(nil)) {
		return fmt.Errorf("update checksum mismatch")
	}
	if err := f.Close(); err != nil {
		return err
	}

	messages.SendEvent(&messages.EventUpdateDoInstall{
		Filepath: updatePath,
	})

	if runtime.GOOS != "android" {
		f, err := os.Open(updatePath)
		if err != nil {
			return err
		}
		defer os.Remove(f.Name())
		defer f.Close()

		checksum, err := base64.StdEncoding.DecodeString(update.Sha256)
		if err != nil {
			return err
		}

		if err = selfupdate.Apply(f, selfupdate.Options{
			Checksum: checksum,
			Hash:     crypto.SHA256,
		}); err != nil {
			return err
		}
	}
	return nil
}

func Restart() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	process := exec.Command(executable, os.Args[1:]...)
	process.Dir, err = os.Getwd()
	if err != nil {
		return err
	}
	return process.Start()
}

type Update struct {
	Version           string `json:"version"`
	Sha256            string `json:"-"`
	DownloadURL       string `json:"downloadURL,omitempty"`
	Name              string `json:"name"`
	Notes             string `json:"notes"`
	ReleaseURL        string `json:"releaseURL"`
	PublishedAt       string `json:"publishedAt"`
	Prerelease        bool   `json:"prerelease"`
	AssetName         string `json:"assetName"`
	Size              int64  `json:"size"`
	Installable       bool   `json:"installable"`
	UnavailableReason string `json:"unavailableReason,omitempty"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Assets      []githubAsset `json:"assets"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt string        `json:"published_at"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
}

type githubAsset struct {
	Name               string `json:"name"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func isNewVersion(version string) bool {
	return newerVersion(utils.Version, version)
}

func (u *Update) IsNew() bool {
	return isNewVersion(u.Version)
}

func latestUpdate() (*Update, error) {
	result, err := CheckRelease(context.Background(), "stable", utils.Version)
	if err != nil {
		return nil, err
	}
	if result.Release == nil {
		return nil, fmt.Errorf("no published CraftBox release is available")
	}
	return result.Release, nil
}
