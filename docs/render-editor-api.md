# Native render editor contract

The editor runs inside the main desktop window. These Wails methods are not
exposed by the map-window HTTP host.

- `SaveCinematicFile(name, kind, base64) -> path`: kind is `gif`, `mp4`,
  `webm`, or `json`. Accepts base64 bytes up to 256 MiB encoded, verifies the
  container signature (or valid JSON), and opens a native Save dialog with the
  matching extension. Cancel returns an empty path. Frontend paths are not used.

- `OpenRenderWorld(kind: "archive" | "folder") -> {id,name,chunks}` opens a native
  file/directory dialog. Cancel returns an empty document. Imports publish only
  after the registry and chunk scan are complete. A successful import replaces
  the previous imported document; capture scenes are independent.
- `CancelRenderImport()` cancels import work. `CloseRenderWorld(id)` releases the
  matching imported store and its private extracted temporary files.
- `GetRenderScenePage(id, PageRequest) -> Page` uses the existing packed paging
  format and always requests terrain-only geometry. Imported IDs and retained
  capture/region IDs are accepted. BlockEntities such as sign text remain part
  of terrain; actors and holograms are omitted. Requests use a separate cache
  scope. New clients/documents start with after=0, target=0, cursor=null.
- `SaveRenderPNG(name, dataURL) -> path` accepts only PNG data URLs, validates the
  decoded image and caps either dimension at 4096. A native Save dialog chooses
  the destination; no filesystem path is accepted from frontend parameters.
  Cancel returns an empty path.

`PageRequest.terrainOnly` is an optional boolean (default false) on the shared
scene contract. Clients switching this mode must restart their paging cycle.

Archives reject absolute paths, traversal, alternate streams and symlinks.
Imports are read-only, bounded to 4096 Overworld chunks and 8 GiB extracted
archive data. Packs follow the recorded world resource/behavior stack order.
Render quality is real-time raster lighting, shadow maps and optional GTAO,
with a static, entity-free model and PNG output up to 4K.
