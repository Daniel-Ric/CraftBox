package messages

import "sync"

var deletedCaptures sync.Map
var captureWrites sync.Map
var captureDiscarders sync.Map
var captureFolders sync.Map

func RecordCaptureFolder(id, folder string) { captureFolders.Store(id, folder) }
func CaptureFolder(id string) string {
	if v, ok := captureFolders.Load(id); ok {
		return v.(string)
	}
	return ""
}

func WorldDeleted(id string) bool { _, ok := deletedCaptures.Load(id); return id != "" && ok }

// MarkCaptureDiscarded suppresses queued updates when the lifecycle owner is
// already closing a world. It deliberately does not invoke the discard callback
// (which may acquire the owner's lock) or remove any files.
func MarkCaptureDiscarded(id string) {
	if id != "" {
		deletedCaptures.Store(id, true)
	}
}
func captureMutex(id string) *sync.Mutex {
	v, _ := captureWrites.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}
func RegisterCaptureDiscarder(id string, fn func() error) { captureDiscarders.Store(id, fn) }
func UnregisterCaptureDiscarder(id string)                { captureDiscarders.Delete(id) }

func WithCaptureWrite(id string, fn func() error) error {
	mu := captureMutex(id)
	mu.Lock()
	defer mu.Unlock()
	if WorldDeleted(id) {
		return nil
	}
	return fn()
}
func DiscardCapture(id string) error {
	mu := captureMutex(id)
	mu.Lock()
	defer mu.Unlock()
	deletedCaptures.Store(id, true)
	if v, ok := captureDiscarders.Load(id); ok {
		return v.(func() error)()
	}
	return nil
}
