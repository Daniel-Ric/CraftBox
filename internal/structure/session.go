package structure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Summary struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Size          Vec3           `json:"size"`
	Origin        Vec3           `json:"origin"`
	Anchor        Vec3           `json:"anchor"`
	Revision      uint64         `json:"revision"`
	Dirty         bool           `json:"dirty"`
	Undo          bool           `json:"undo"`
	Redo          bool           `json:"redo"`
	Path          string         `json:"path"`
	Palette       []State        `json:"palette"`
	Regions       []Region       `json:"regions"`
	Packs         []Dependency   `json:"packs"`
	View          map[string]any `json:"view"`
	Volume        int64          `json:"volume"`
	MemoryBytes   int64          `json:"memoryBytes"`
	BlockEntities int            `json:"blockEntities"`
	Entities      int            `json:"entities"`
}
type Job struct {
	Busy    bool      `json:"busy"`
	Stage   string    `json:"stage"`
	Done    int64     `json:"done"`
	Total   int64     `json:"total"`
	Started time.Time `json:"started"`
}
type Session struct {
	Mu               sync.Mutex
	ID, Folder, Path string
	History          *History
	View             map[string]any
	Revision         uint64
	Saved            *Document
	SavedView        string
	jobMu            sync.Mutex
	job              Job
	cancel           context.CancelFunc
}

func (s *Session) Summary() Summary {
	d := s.History.Doc
	volume := int64(d.Size[0]) * int64(d.Size[1]) * int64(d.Size[2])
	return Summary{s.ID, d.Name, d.Size, d.Origin, d.Anchor, s.Revision, s.Dirty(), s.History.CanUndo(), s.History.CanRedo(), s.Path, append([]State{}, d.Palette...), append([]Region{}, d.Regions...), append([]Dependency{}, d.Packs...), s.View, volume, int64(len(d.Sections)) * 33280, len(d.BlockData), len(d.Entities)}
}
func (s *Session) Dirty() bool {
	raw, _ := json.Marshal(s.View)
	return s.Saved != s.History.Doc || s.SavedView != string(raw)
}
func (s *Session) MarkSaved() {
	s.Saved = s.History.Doc
	raw, _ := json.Marshal(s.View)
	s.SavedView = string(raw)
}
func (s *Session) Begin(parent context.Context, stage string) (context.Context, func(), error) {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if s.job.Busy {
		return nil, nil, fmt.Errorf("Another structure operation is still running")
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.job = Job{Busy: true, Stage: stage, Started: time.Now()}
	return ctx, func() { cancel(); s.jobMu.Lock(); s.job.Busy = false; s.cancel = nil; s.jobMu.Unlock() }, nil
}
func (s *Session) Progress(stage string, done, total int64) {
	s.jobMu.Lock()
	s.job.Stage = stage
	s.job.Done = done
	s.job.Total = total
	s.jobMu.Unlock()
}
func (s *Session) Job() Job { s.jobMu.Lock(); defer s.jobMu.Unlock(); return s.job }
func (s *Session) Cancel() {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}
func dependencyKey(p Dependency) string {
	h := sha256.Sum256([]byte(p.Kind + "|" + p.ID))
	return hex.EncodeToString(h[:16])
}
func CopyDirectory(ctx context.Context, source, target string) error {
	return filepath.WalkDir(source, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		to := filepath.Join(target, rel)
		if e.IsDir() {
			return os.MkdirAll(to, 0700)
		}
		if !e.Type().IsRegular() {
			return fmt.Errorf("Linked dependency files are unsupported")
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, &contextReader{ctx, in})
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}

// AdoptDependencies makes recovery independent of source worlds and import temp folders.
func (s *Session) AdoptDependencies(ctx context.Context) error {
	d := s.History.Doc
	for i, p := range d.Packs {
		if p.Missing || p.Source == "" {
			continue
		}
		dest := filepath.Join(s.Folder, "assets", dependencyKey(p))
		if p.Source != dest {
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				if err = CopyDirectory(ctx, p.Source, dest); err != nil {
					os.RemoveAll(dest)
					return err
				}
			}
			d.Packs[i].Source = dest
		}
	}
	return nil
}
func (s *Session) Autosave(ctx context.Context) error {
	if err := os.MkdirAll(s.Folder, 0700); err != nil {
		return err
	}
	view := map[string]any{}
	for key, v := range s.View {
		view[key] = v
	}
	view["_recovery"] = map[string]any{"path": s.Path, "saved": !s.Dirty()}
	return WriteAtomic(filepath.Join(s.Folder, "recovery.craftbox"), func(w io.Writer) error { return WriteProject(ctx, w, s.History.Doc, view, false) })
}
func NewSession(ctx context.Context, root string, d *Document, view map[string]any) (*Session, error) {
	id := uuid.NewString()
	s := &Session{ID: id, Folder: filepath.Join(root, id), History: NewHistory(d), View: view, Revision: 1}
	if s.View == nil {
		s.View = map[string]any{}
	}
	if err := s.AdoptDependencies(ctx); err != nil {
		_ = os.RemoveAll(s.Folder)
		return nil, err
	}
	if err := s.Autosave(ctx); err != nil {
		_ = os.RemoveAll(s.Folder)
		return nil, err
	}
	return s, nil
}
func RecoverSession(ctx context.Context, root, id string) (*Session, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("Invalid recovery identifier")
	}
	folder := filepath.Join(root, id)
	f, err := os.Open(filepath.Join(folder, "recovery.craftbox"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	temporary, err := os.MkdirTemp("", "craftbox-structure-recovery-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temporary)
	d, view, err := ReadProject(ctx, f, info.Size(), temporary)
	if err != nil {
		return nil, err
	}
	for i, p := range d.Packs {
		dest := filepath.Join(folder, "assets", dependencyKey(p))
		if _, e := os.Stat(filepath.Join(dest, "manifest.json")); e == nil {
			d.Packs[i].Source = dest
			d.Packs[i].Missing = false
		}
	}
	s := &Session{ID: id, Folder: folder, History: NewHistory(d), View: view, Revision: 1}
	var state struct {
		Path  string `json:"path"`
		Saved bool   `json:"saved"`
	}
	if raw, e := os.ReadFile(filepath.Join(folder, "state.json")); e == nil {
		_ = json.Unmarshal(raw, &state)
		s.Path = state.Path
		if state.Saved {
			s.Saved = d
		}
	}
	if raw, ok := view["_recovery"].(map[string]any); ok {
		s.Path, _ = raw["path"].(string)
		if saved, _ := raw["saved"].(bool); saved {
			s.Saved = d
		}
		delete(view, "_recovery")
	}
	if s.Saved != nil {
		s.MarkSaved()
	}
	return s, nil
}
