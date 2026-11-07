# Portable first start

`GetBootstrap.installationId` remains an opaque string. Its identity now includes
a random nonce stored next to the executable at `.craftbox/installation-id`, in
addition to the normalized executable directory. This corrects the old path-only
scope: clearing a folder and copying only the EXE into it must not inherit a
completed introduction from WebView's shared local storage.

The marker is created exclusively, with bounded retries for concurrent startup.
Renaming or updating the EXE retains the identity. A new or cleaned folder gets a
new scope. Bootstrap caches the scope for the current app. If the marker cannot
be persisted/read, startup uses a transient scope and records
`installation.identity.unavailable` instead of hiding the introduction or
preventing startup. The marker contains no account information or credentials.

The first-run introduction is independent of both the shared `showWelcome`
preference and initial login state. Signed-out users can link Xbox or explicitly
continue offline. Signed-in users see their existing connection and continue
explicitly; no second login or sign-out is required. Only a successful login
initiated from the introduction completes it automatically. Cancelling login
keeps the introduction pending. Completing it opens the workspace welcome for
that launch. Later launches honor the normal welcome preference.

Account tokens, workspace preferences, captures and recent projects are not
deleted or migrated. Existing folders without the new marker see the
introduction once after updating. Keep `.craftbox` alongside the EXE to retain
its first-start state. Copying only the EXE starts a fresh introduction.

Regression coverage includes a cleaned folder at the same path, EXE updates,
concurrent startup, bootstrap serialization, signed-in first start with welcome
disabled, optional linking/cancellation, offline continuation and configured
installations honoring the welcome preference.

## Build verification (2026-11-02)

Version: 1.63.1-portable-welcome-wails.20261102-dev.

- Go GUI desktop package tests passed, including the portable identity tests.
- TypeScript and Vite production build passed.
- First-start and application workflow tests: 29 passed, 0 failed.
- EXE: builds/resourcepack-fix/Craft-Box.exe, 250806272 bytes.
- Windows GUI subsystem: 2; embedded vanilla assets retained.
- SHA-256: 0AB55FAF2663B3338B1A5312E0F93EA38967A3F33415BD5CB3B8E95E0717C929.

The full frontend run had 373 passing tests and one failure in the newly added
restart test's DOM text selector. That selector was corrected and the complete
App/FirstStartDialog test files were rerun as reported above. No native UI session
was launched for this verification.