package uirecord

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

var imagePathChars = regexp.MustCompile(`^[A-Za-z0-9_\-./]+$`)

func ImagePath(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		return "", false
	}
	p := path.Clean("/" + u.Path)[1:]
	p = strings.TrimSuffix(p, path.Ext(p))
	if p == "" || strings.Contains(p, "..") || !imagePathChars.MatchString(p) {
		return "", false
	}
	return p, true
}
func (r *Recorder) Images() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.imagesLocked()
}

func (r *Recorder) imagesLocked() map[string]string {
	images := make(map[string]string)
	for _, f := range r.forms {
		buttons, _ := f.Data["buttons"].([]any)
		elements, _ := f.Data["elements"].([]any)
		for _, b := range append(append([]any{}, buttons...), elements...) {
			bm, _ := b.(map[string]any)
			img, _ := bm["image"].(map[string]any)
			if img == nil || img["type"] != "url" {
				continue
			}
			u, _ := img["data"].(string)
			if p, ok := ImagePath(u); ok {
				images[u] = p
			}
		}
	}
	byPath := make(map[string][]string)
	for u, p := range images {
		byPath[p] = append(byPath[p], u)
	}
	for p, urls := range byPath {
		if len(urls) < 2 {
			continue
		}
		sort.Strings(urls)
		for _, u := range urls {
			sum := sha256.Sum256([]byte(u))
			images[u] = fmt.Sprintf("%s_%x", p, sum[:8])
		}
	}
	return images
}
