package desktop

import (
	"context"
	"fmt"
)

// The caller holds a.mu. Documents are addressed by identity, never by the last
// editor that happened to open one. renderDocument remains a legacy pointer for
// private conversion helpers; user-facing reads must look up their explicit ID.
func (a *App) keepRenderDocumentLocked(doc *renderDocument) {
	if a.renderDocuments == nil {
		a.renderDocuments = map[string]*renderDocument{}
	}
	a.renderDocuments[doc.info.ID] = doc
	a.renderDocument = doc
}
func (a *App) renderDocumentLocked(id string) *renderDocument {
	if doc := a.renderDocuments[id]; doc != nil {
		return doc
	}
	if a.renderDocument != nil && a.renderDocument.info.ID == id {
		return a.renderDocument
	}
	return nil
}

func (a *App) importRenderWorldForTab(ctx context.Context, tabID, path string, dimension *int32) (RenderWorld, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	worker := &App{ctx: ctx}
	a.mu.Lock()
	if a.renderTabImports == nil {
		a.renderTabImports = map[string]*App{}
	}
	if a.renderTabImports[tabID] != nil {
		a.mu.Unlock()
		return RenderWorld{}, fmt.Errorf("This document is already importing a world")
	}
	if a.renderTabCancels == nil {
		a.renderTabCancels = map[string]context.CancelFunc{}
	}
	a.renderTabCancels[tabID] = cancel
	a.renderTabImports[tabID] = worker
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.renderTabImports, tabID)
		delete(a.renderTabCancels, tabID)
		a.mu.Unlock()
	}()
	info, err := worker.importRenderWorldDimension(ctx, path, dimension)
	if err != nil {
		return info, err
	}
	worker.mu.Lock()
	doc := worker.renderDocument
	worker.renderDocument = nil
	worker.renderDocuments = nil
	worker.mu.Unlock()
	a.mu.Lock()
	if err = ctx.Err(); err != nil {
		a.mu.Unlock()
		closeRenderDocument(doc)
		return RenderWorld{}, err
	}
	a.keepRenderDocumentLocked(doc)
	a.mu.Unlock()
	return info, nil
}
func (a *App) CancelRenderImportForTab(tabID string) {
	a.mu.Lock()
	cancel := a.renderTabCancels[tabID]
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
func (a *App) SwitchRenderDimensionForTab(tabID, id string, dimension int32) (RenderWorld, error) {
	a.mu.Lock()
	doc := a.renderDocumentLocked(id)
	if doc == nil || doc.project != nil {
		a.mu.Unlock()
		return RenderWorld{}, fmt.Errorf("The imported world is no longer open")
	}
	path := doc.sourcePath
	ctx := a.ctx
	a.mu.Unlock()
	return a.importRenderWorldForTab(ctx, tabID, path, &dimension)
}

func (a *App) ReleaseDocumentTab(tabID string) {
	a.CancelRenderImportForTab(tabID)
	a.mu.Lock()
	delete(a.studioTabPaths, tabID)
	a.mu.Unlock()
}

func (a *App) SetDocumentCloseManaged(managed bool) {
	a.mu.Lock()
	a.documentCloseManaged = managed
	a.mu.Unlock()
}
