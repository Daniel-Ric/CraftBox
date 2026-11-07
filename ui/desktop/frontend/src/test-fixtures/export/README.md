# Export importer fixtures

These files are generated from CraftBox's own three-triangle scene with opaque,
masked and blended materials and a half opaque / half transparent texture.
They contain no external world content. Three.js independently imports FBX, OBJ
and STL in export-importers.test.ts. The Go test compares committed file bytes
with fresh converter output to prevent stale fixtures silently passing.

From the repository root, regenerate after intentional writer changes:

```powershell
$env:CRAFTBOX_EXPORT_FIXTURE_OUT = Join-Path $pwd 'ui/desktop/frontend/src/test-fixtures/export'
go test ./internal/modelexport -run TestTransparentExchangeAndImporterFixtures -count=1
Remove-Item Env:CRAFTBOX_EXPORT_FIXTURE_OUT
```