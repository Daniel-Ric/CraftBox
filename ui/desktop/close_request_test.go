package desktop

import "testing"

func TestQuitAsksAboutSessionBeforeUnsavedProjects(t *testing.T) {
	var events []string
	a := &App{documentCloseManaged: true, studioProjectDirty: true}
	a.state.Running = true
	a.emit = func(name string, _ interface{}) { events = append(events, name) }
	if !a.shouldPreventWindowClose() || len(events) != 1 || events[0] != "desktop:close-requested" {
		t.Fatalf("first close should ask about the active session, got %v", events)
	}
	// The frontend next resolves projects, then stops the session and approves
	// both gates. Discard leaves dirty set: it must still close without reasking.
	a.sessionCloseApproved, a.studioCloseApproved = true, true
	a.state.Running = false
	if a.shouldPreventWindowClose() {
		t.Fatal("approved quit reopened a dialog")
	}
	if len(events) != 1 {
		t.Fatalf("extra close dialogs: %v", events)
	}
}

func TestQuitWithoutActiveSessionStillProtectsProjects(t *testing.T) {
	var events []string
	a := &App{documentCloseManaged: true, studioProjectDirty: true}
	a.emit = func(name string, _ interface{}) { events = append(events, name) }
	if !a.shouldPreventWindowClose() || len(events) != 1 || events[0] != "studio:close-requested" {
		t.Fatalf("projects were not protected: %v", events)
	}
	a.studioCloseApproved = true
	if a.shouldPreventWindowClose() {
		t.Fatal("project approval was ignored")
	}
}
