package localworlds

import (
	"archive/zip"
	"compress/flate"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

type entry struct {
	Root, Rel, Name, Stage string
	Size, Modified         int64
	Data                   []byte
}

func ignored(name string) bool {
	value := strings.ToUpper(filepath.ToSlash(name))
	return value == "DB/LOCK" || value == "DB/LOG" || strings.HasPrefix(value, "DB/LOG.") || value == "SESSION.LOCK" || value == "CRAFTBOX-EXPORT-REPORT.JSON"
}
func walkFiles(ctx context.Context, root, prefix, stage string, files map[string]entry) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("worlds and packs containing filesystem links cannot be exported: %s", d.Name())
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported world file: %s", d.Name())
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if prefix == "" && ignored(rel) {
			return nil
		}
		name := filepath.ToSlash(filepath.Join(prefix, rel))
		key := strings.ToLower(name)
		if old, ok := files[key]; ok {
			if strings.EqualFold(filepath.Join(old.Root, old.Rel), path) {
				return nil
			}
			return fmt.Errorf("two world files have the same archive path: %s", name)
		}
		files[key] = entry{Root: root, Rel: rel, Name: name, Stage: stage, Size: info.Size(), Modified: info.ModTime().UnixNano()}
		return nil
	})
}
func Inspect(ctx context.Context, w World, o Options) (Plan, error) {
	p := Plan{World: w}
	var err error
	extra := w.ExtraPackRoots
	w, err = Describe(w.Folder, w.Storage, w.Installation)
	if err != nil {
		return p, err
	}
	w.ExtraPackRoots = extra
	p.World = w
	references, err := resolvePacks(ctx, w, o, &p)
	if err != nil {
		return p, err
	}
	if _, err = metadata(w.Folder); err != nil {
		p.Issues = append(p.Issues, Issue{Severity: "error", Code: "world_metadata", Message: "World metadata could not be decoded. Only an unchanged backup can be created."})
	}
	if w.Template || w.Marketplace {
		p.Issues = append(p.Issues, Issue{Severity: "warning", Code: "template", Message: "Created from a world template. Minecraft may require the recipient to own its Marketplace content; template settings are preserved."})
	}
	files := map[string]entry{}
	if err = walkFiles(ctx, w.Folder, "", "Copying world", files); err != nil {
		return p, err
	}
	for _, pack := range p.Packs {
		if pack.candidate == nil {
			continue
		}
		c := pack.candidate
		dest := pack.Kind + "_packs/" + pack.ID + "_" + pack.Version
		if inside(w.Folder, c.folder) {
			rel, e := filepath.Rel(w.Folder, c.folder)
			if e != nil {
				return p, e
			}
			dest = filepath.ToSlash(rel)
		} else {
			base := dest
			for suffix := 2; ; suffix++ {
				old, ok := files[strings.ToLower(dest+"/manifest.json")]
				if !ok || strings.EqualFold(filepath.Join(old.Root, old.Rel), filepath.Join(c.folder, "manifest.json")) {
					break
				}
				dest = fmt.Sprintf("%s_%d", base, suffix)
			}
		}
		if err = walkFiles(ctx, c.folder, dest, "Adding pack: "+pack.Name, files); err != nil {
			return p, err
		}
	}
	for _, kind := range []string{"resource", "behavior"} {
		refs := references[kind]
		if len(refs) == 0 {
			continue
		}
		values := make([]json.RawMessage, len(refs))
		for i, ref := range refs {
			values[i] = ref.Raw
		}
		data, e := json.MarshalIndent(values, "", "  ")
		if e != nil {
			return p, e
		}
		name := "world_" + kind + "_packs.json"
		files[name] = entry{Name: name, Stage: "Writing pack activation", Data: data, Size: int64(len(data))}
	}
	p.Portable = true
	for _, issue := range p.Issues {
		if issue.Severity == "error" {
			p.Portable = false
		}
	}
	for _, file := range files {
		p.entries = append(p.entries, file)
		p.Bytes += file.Size
	}
	sort.Slice(p.entries, func(i, j int) bool { return p.entries[i].Name < p.entries[j].Name })
	p.Files = len(p.entries)
	hash := sha256.New()
	encoder := json.NewEncoder(hash)
	for _, file := range p.entries {
		if err = encoder.Encode([]any{file.Root, file.Rel, file.Name, file.Size, file.Modified, file.Data}); err != nil {
			return p, err
		}
	}
	if err = encoder.Encode([]any{p.Portable, p.Packs, p.Issues, o.IncludeGlobal}); err != nil {
		return p, err
	}
	p.Fingerprint = hex.EncodeToString(hash.Sum(nil))
	return p, ctx.Err()
}

type contextWriter struct {
	ctx      context.Context
	out      io.Writer
	written  *int64
	progress func(int64)
}

func (w contextWriter) Write(data []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.out.Write(data)
	*w.written += int64(n)
	if w.progress != nil {
		w.progress(*w.written)
	}
	return n, err
}

func Export(ctx context.Context, w World, o Options, destination, expected string, progress func(Progress)) (result Result, err error) {
	if err = ctx.Err(); err != nil {
		return result, err
	}
	emit := func(stage string, done, total int64, files int) {
		if progress != nil {
			progress(Progress{Stage: stage, Completed: done, Total: total, Files: files, Running: true})
		}
	}
	emit("Checking world and packs", 0, 0, 0)
	if err = ctx.Err(); err != nil {
		return result, err
	}
	output, err := filepath.Abs(destination)
	if err != nil {
		return result, err
	}
	if !strings.EqualFold(filepath.Ext(output), ".mcworld") {
		return result, fmt.Errorf("choose a .mcworld destination")
	}
	parent, err := canonical(filepath.Dir(output))
	if err != nil {
		return result, err
	}
	output = filepath.Join(parent, filepath.Base(output))
	worldRoot, err := canonical(w.Folder)
	if err != nil {
		return result, err
	}
	if inside(worldRoot, output) {
		return result, fmt.Errorf("save the exported copy outside the original world")
	}
	db, err := leveldb.OpenFile(filepath.Join(worldRoot, "db"), &opt.Options{ReadOnly: true, ErrorIfMissing: true})
	if err != nil {
		return result, fmt.Errorf("close this world in Minecraft before exporting it; its database could not be locked: %w", err)
	}
	defer db.Close()
	plan, err := Inspect(ctx, w, o)
	if err != nil {
		return result, err
	}
	if expected != "" && expected != plan.Fingerprint {
		return result, fmt.Errorf("the world or its packs changed; refresh the export preview before trying again")
	}
	if !plan.Portable && !o.BackupOnly {
		return result, fmt.Errorf("this world has missing, ambiguous or protected dependencies; review the pack report or explicitly choose Backup only")
	}
	buffer := make([]byte, 128<<10)
	for _, file := range plan.entries {
		if file.Root != "" && inside(file.Root, output) {
			return result, fmt.Errorf("save the exported copy outside its source packs")
		}
	}
	temporary, err := os.CreateTemp(parent, ".craftbox-world-*.tmp")
	if err != nil {
		return result, err
	}
	tempName := temporary.Name()
	defer func() { temporary.Close(); os.Remove(tempName) }()
	writer := zip.NewWriter(temporary)
	writer.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) { return flate.NewWriter(out, flate.BestSpeed) })
	closed := false
	defer func() {
		if !closed {
			writer.Close()
		}
	}()
	roots := map[string]*os.Root{}
	defer func() {
		for _, root := range roots {
			root.Close()
		}
	}()
	var done int64
	var last time.Time
	currentStage := ""
	completed := 0
	reportProgress := func(n int64) {
		if progress != nil && (time.Since(last) > 100*time.Millisecond || n >= plan.Bytes) {
			last = time.Now()
			emit(currentStage, n, plan.Bytes, completed)
		}
	}
	for _, file := range plan.entries {
		if err = ctx.Err(); err != nil {
			return result, err
		}
		if currentStage != file.Stage {
			currentStage = file.Stage
			emit(currentStage, done, plan.Bytes, completed)
		}
		header := &zip.FileHeader{Name: file.Name, Method: zip.Deflate}
		if file.Modified > 0 {
			header.SetModTime(time.Unix(0, file.Modified))
		}
		header.SetMode(0644)
		out, e := writer.CreateHeader(header)
		if e != nil {
			return result, e
		}
		target := contextWriter{ctx: ctx, out: out, written: &done, progress: reportProgress}
		if file.Data != nil {
			if _, err = target.Write(file.Data); err != nil {
				return result, err
			}
		} else {
			root := roots[file.Root]
			if root == nil {
				root, err = os.OpenRoot(file.Root)
				if err != nil {
					return result, err
				}
				roots[file.Root] = root
			}
			input, e := root.Open(file.Rel)
			if e != nil {
				return result, e
			}
			before, e := input.Stat()
			if e != nil {
				input.Close()
				return result, e
			}
			if before.Size() != file.Size || before.ModTime().UnixNano() != file.Modified {
				input.Close()
				return result, fmt.Errorf("a world or pack file changed during export; close the world and try again")
			}
			copied, e := io.CopyBuffer(target, input, buffer)
			after, statErr := input.Stat()
			closeErr := input.Close()
			if e != nil {
				return result, e
			}
			if statErr != nil {
				return result, statErr
			}
			if closeErr != nil {
				return result, closeErr
			}
			if copied != file.Size || after.Size() != before.Size() || after.ModTime() != before.ModTime() {
				return result, fmt.Errorf("a world or pack file changed during export; try again after it finishes saving")
			}
		}
		completed++
	}
	mode := "share"
	if o.BackupOnly {
		mode = "backup"
	}
	report := struct {
		Format   int     `json:"format"`
		Mode     string  `json:"mode"`
		World    string  `json:"world"`
		Portable bool    `json:"portable"`
		Packs    []Pack  `json:"packs"`
		Issues   []Issue `json:"issues"`
	}{1, mode, w.Name, plan.Portable, plan.Packs, plan.Issues}
	data, e := json.MarshalIndent(report, "", "  ")
	if e != nil {
		return result, e
	}
	out, e := writer.Create("craftbox-export-report.json")
	if e != nil {
		return result, e
	}
	if _, err = out.Write(data); err != nil {
		return result, err
	}
	emit("Finishing .mcworld", done, plan.Bytes, completed)
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = writer.Close(); err != nil {
		return result, err
	}
	closed = true
	if err = temporary.Sync(); err != nil {
		return result, err
	}
	if err = temporary.Close(); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = os.Rename(tempName, output); err != nil {
		return result, err
	}
	info, err := os.Stat(output)
	if err != nil {
		return result, err
	}
	return Result{Path: output, Bytes: info.Size(), Files: completed + 1, Packs: len(plan.Packs), Portable: plan.Portable, Issues: plan.Issues}, nil
}
