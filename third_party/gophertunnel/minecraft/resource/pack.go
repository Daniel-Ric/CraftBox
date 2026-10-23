package resource

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/df-mc/jsonc"
	"github.com/google/uuid"
)

type Pack struct {
	manifest *Manifest

	downloadURL string

	content *bytes.Reader

	archive *packArchive

	contentKey string

	checksum [32]byte
}

func ReadPath(path string) (*Pack, error) {
	return compile(path)
}

func ReadURL(url string) (*Pack, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download resource pack: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download resource pack: %v (%d)", resp.Status, resp.StatusCode)
	}
	pack, err := Read(resp.Body)
	if err != nil {
		return nil, err
	}
	pack.downloadURL = url
	return pack, nil
}

func MustReadPath(path string) *Pack {
	pack, err := compile(path)
	if err != nil {
		panic(err)
	}
	return pack
}

func MustReadURL(url string) *Pack {
	pack, err := ReadURL(url)
	if err != nil {
		panic(err)
	}
	return pack
}

func Read(r io.Reader) (*Pack, error) {
	temp, err := createTempFile()
	if err != nil {
		return nil, fmt.Errorf("create temp zip archive: %w", err)
	}
	_, _ = io.Copy(temp, r)
	if err := temp.Close(); err != nil {
		return nil, fmt.Errorf("close temp zip archive: %w", err)
	}
	pack, parseErr := ReadPath(temp.Name())
	if err := os.Remove(temp.Name()); err != nil {
		return nil, fmt.Errorf("remove temp zip archive: %w", err)
	}
	return pack, parseErr
}

func (pack *Pack) Name() string {
	return pack.manifest.Header.Name
}

func (pack *Pack) UUID() uuid.UUID {
	return pack.manifest.Header.UUID
}

func (pack *Pack) Description() string {
	return pack.manifest.Header.Description
}

func (pack *Pack) Version() string {
	return strconv.Itoa(pack.manifest.Header.Version[0]) + "." + strconv.Itoa(pack.manifest.Header.Version[1]) + "." + strconv.Itoa(pack.manifest.Header.Version[2])
}

func (pack *Pack) Modules() []Module {
	return pack.manifest.Modules
}

func (pack *Pack) Dependencies() []Dependency {
	return pack.manifest.Dependencies
}

func (pack *Pack) HasScripts() bool {
	for _, module := range pack.manifest.Modules {
		if module.Type == "client_data" {

			return true
		}
	}
	return false
}

func (pack *Pack) HasBehaviours() bool {
	for _, module := range pack.manifest.Modules {
		if module.Type == "client_data" || module.Type == "data" {

			return true
		}
	}
	return false
}

func (pack *Pack) HasTextures() bool {
	for _, module := range pack.manifest.Modules {
		if module.Type == "resources" {

			return true
		}
	}
	return false
}

func (pack *Pack) HasWorldTemplate() bool {
	return pack.manifest.worldTemplate
}

func (pack *Pack) DownloadURL() string {
	return pack.downloadURL
}

func (pack *Pack) Checksum() [32]byte {
	return pack.checksum
}

func (pack *Pack) Len() int {
	return pack.content.Len()
}

func (pack *Pack) DataChunkCount(length int) int {
	count := pack.Len() / length
	if pack.Len()%length != 0 {
		count++
	}
	return count
}

func (pack *Pack) Encrypted() bool {
	return pack.contentKey != ""
}

func (pack *Pack) ContentKey() string {
	return pack.contentKey
}

func (pack *Pack) ReadAt(b []byte, off int64) (n int, err error) {
	return pack.content.ReadAt(b, off)
}

func (p *Pack) ReadFile(filePath string) ([]byte, error) {
	file, ok := p.archive.files[strings.ToLower(filePath)]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: filePath, Err: fs.ErrNotExist}
	}
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer rc.Close()
	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}
	return content, nil
}

func (pack Pack) WithContentKey(key string) *Pack {
	pack.contentKey = key
	return &pack
}

func (pack *Pack) Manifest() Manifest {
	return *pack.manifest
}

func (pack *Pack) String() string {
	return fmt.Sprintf("%v v%v (%v): %v", pack.Name(), pack.Version(), pack.UUID(), pack.Description())
}

func compile(path string) (*Pack, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("open resource pack path: %w", err)
	}
	if info.IsDir() {
		temp, err := createTempArchive(path)
		if err != nil {
			return nil, err
		}

		path = temp.Name()

		_ = temp.Close()
		defer func() {
			_ = os.Remove(temp.Name())
		}()
	}

	manifest, err := readManifest(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read resource pack file content: %w", err)
	}
	checksum := sha256.Sum256(content)
	contentReader := bytes.NewReader(content)

	archive, err := indexPackArchive(contentReader)
	if err != nil {
		return nil, fmt.Errorf("read resource pack archive: %w", err)
	}
	return &Pack{manifest: manifest, checksum: checksum, content: contentReader, archive: archive}, nil
}

func createTempArchive(path string) (*os.File, error) {
	temp, err := createTempFile()
	if err != nil {
		return nil, err
	}
	writer := zip.NewWriter(temp)
	defer writer.Close()
	if err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(path, filePath)
		if err != nil {
			return fmt.Errorf("find relative path: %w", err)
		}

		relPath = strings.ReplaceAll(relPath, `\`, "/")

		if relPath == "." {
			return nil
		}
		s, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("read stat of file path %v: %w", filePath, err)
		}
		if s.IsDir() {

			_, _ = writer.Create(relPath + "/")
			return nil
		}
		f, err := writer.Create(relPath)
		if err != nil {
			return fmt.Errorf("create new zip file: %w", err)
		}
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("open resource pack file %v: %w", filePath, err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)

		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("write file data to zip: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("build zip archive: %w", err)
	}
	return temp, nil
}

func createTempFile() (*os.File, error) {

	dir, _ := os.UserConfigDir()
	_ = os.MkdirAll(dir, os.ModePerm)

	temp, err := os.CreateTemp(dir, "temp_resource_pack-*.mcpack")
	if err != nil {
		return nil, fmt.Errorf("create temp resource pack file: %w", err)
	}
	return temp, nil
}

type packReader struct {
	*zip.ReadCloser
}

func (reader packReader) find(fileName string) (io.ReadCloser, error) {
	for _, file := range reader.File {
		base := filepath.Base(file.Name)
		if file.Name != fileName && base != fileName {
			continue
		}
		fileReader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip file %v: %w", file.Name, err)
		}
		return fileReader, nil
	}
	return nil, fmt.Errorf("'%v' not found in zip", fileName)
}

func readManifest(path string) (*Manifest, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open zip reader: %w", err)
	}
	reader := packReader{ReadCloser: r}
	defer func() {
		_ = r.Close()
	}()

	manifestFile, err := reader.find("manifest.json")
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}
	defer func() {
		_ = manifestFile.Close()
	}()

	allData, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("read manifest file: %w", err)
	}
	manifest := &Manifest{}
	if err := jsonc.UnmarshalLenient(allData, manifest); err != nil {
		return nil, fmt.Errorf("decode manifest JSON: %w (data: %v)", err, string(allData))
	}

	if _, err := reader.find("level.dat"); err == nil {
		manifest.worldTemplate = true
	}

	return manifest, nil
}
