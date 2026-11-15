package worldstate

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	formImageMaxSize    = 4 << 20
	formImageWorkers    = 8
	formImageCacheLimit = 64 << 20
	formImageMaxCount   = 256
)

var (
	formImageClient     = &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{DialContext: dialFormImage, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 10 * time.Second, IdleConnTimeout: 30 * time.Second}}
	formImageCache      = map[string]formImage{}
	formImageCacheMu    sync.Mutex
	formImageCacheBytes int
)

func dialFormImage(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	for _, address := range addresses {
		ip := address.IP
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return nil, fmt.Errorf("form images require a public network address")
		}
	}
	var dialer net.Dialer
	var last error
	for _, address := range addresses {
		connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(address.IP.String(), port))
		if err == nil {
			return connection, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("form image host has no public address")
	}
	return nil, last
}

func fetchFormImages(ctx context.Context, images map[string]string, log *logrus.Entry) map[string][]byte {
	urls := make([]string, 0, len(images))
	for u := range images {
		urls = append(urls, u)
	}
	sort.Strings(urls)
	if len(urls) > formImageMaxCount {
		urls = urls[:formImageMaxCount]
	}
	results := make(map[string]formImage)
	var resultsMu sync.Mutex
	resultBytes := 0

	jobs := make(chan string)
	var wg sync.WaitGroup
	for range min(formImageWorkers, len(urls)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				resultsMu.Lock()
				full := resultBytes >= formImageCacheLimit
				resultsMu.Unlock()
				if full {
					continue
				}
				formImageCacheMu.Lock()
				img, done := formImageCache[u]
				formImageCacheMu.Unlock()
				if !done {
					var err error
					img, err = fetchFormImage(ctx, u)
					if err != nil {
						if log != nil {
							log.Debugf("form image %s: %s", u, err)
						}
						continue
					}
					formImageCacheMu.Lock()
					if formImageCacheBytes+len(img.data) > formImageCacheLimit {
						formImageCache = map[string]formImage{}
						formImageCacheBytes = 0
					}
					if _, exists := formImageCache[u]; !exists {
						formImageCache[u] = img
						formImageCacheBytes += len(img.data)
					}
					formImageCacheMu.Unlock()
				}
				resultsMu.Lock()
				if resultBytes+len(img.data) <= formImageCacheLimit {
					results[u] = img
					resultBytes += len(img.data)
				}
				resultsMu.Unlock()
			}
		}()
	}
	for _, u := range urls {
		jobs <- u
	}
	close(jobs)
	wg.Wait()

	files := make(map[string][]byte)
	total := 0
	for _, u := range urls {
		p := images[u]
		if !fs.ValidPath(p) || strings.ContainsAny(p, "\\:") {
			continue
		}
		if img := results[u]; img.data != nil && total+len(img.data) <= formImageCacheLimit {
			files[p+img.ext] = img.data
			total += len(img.data)
		}
	}
	return files
}

type formImage struct {
	data []byte
	ext  string
}

func fetchFormImage(ctx context.Context, u string) (formImage, error) {
	parsed, err := url.Parse(u)
	if err != nil || parsed.User != nil || parsed.Hostname() == "" || parsed.Scheme != "https" && parsed.Scheme != "http" {
		return formImage{}, fmt.Errorf("invalid form image URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return formImage{}, err
	}
	req.Header.Set("User-Agent", "CraftBox")
	res, err := formImageClient.Do(req)
	if err != nil {
		return formImage{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return formImage{}, fmt.Errorf("status %s", res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, formImageMaxSize+1))
	if err != nil {
		return formImage{}, err
	}
	if len(data) > formImageMaxSize {
		return formImage{}, fmt.Errorf("larger than %d bytes", formImageMaxSize)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return formImage{}, err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > 4096 || config.Height > 4096 {
		return formImage{}, fmt.Errorf("form image dimensions exceed 4096 pixels")
	}
	switch format {
	case "png":
		return formImage{data: data, ext: ".png"}, nil
	case "jpeg":
		return formImage{data: data, ext: ".jpg"}, nil
	}
	return formImage{}, fmt.Errorf("unsupported image format %s", format)
}
