# Export studio

Five suitable common formats are available for each output family. These are
workflow choices, not a claimed global usage ranking.

| Output | Formats | Notes |
|---|---|---|
| Render, map, 360 panorama | PNG, JPEG, WebP, TIFF, BMP | PNG/TIFF lossless; JPEG/BMP opaque; JPEG/WebP quality control |
| Static source model | GLB, glTF, OBJ, FBX, STL | GLB recommended for Blender; multi-file formats delivered in ZIP |
| Cinematic | MP4, WebM, MOV, MKV, GIF | Actual containers, not renamed files; video encoder availability depends on WebView |

The format navigation, settings, output summary and footer share one component.
Images show exact dimensions, alpha support and matte color where needed. Maps
retain region, pixel density, annotations and the 64 MP budget. Panoramas retain
2:1 projection and their own resolution options. Cinematics expose frame rate,
bitrate, audio and timeline range. Encoded timestamps start at zero for a range;
audio is sampled at its original timeline offset. GIF has no audio and retains
its frame/pixel budget. All cinematic output retains the 190 MiB limit.

Models expose a Blender preset, centered/world origin, positive scale, Y/Z up and
material/texture inclusion. GLB/glTF always use the glTF Y-up convention. Scale 1
is one unit per block. Geometry comes from the full source/selected retained
region, independently of the camera. Post effects and camera animation are not
exported in static models.

GLB and glTF retain the canonical glTF material properties. glTF ZIP contains
`scene.gltf` and `scene.bin`, with the PNG texture stored in the buffer. OBJ ZIP
contains `scene.obj`, `scene.mtl`, `atlas.png` and `opacity.png` when textures are
enabled. FBX ZIP contains a binary FBX 7.4 file and the two PNG textures. OBJ/FBX
preserve diffuse tint, scalar opacity, texture opacity, UVs and normals; advanced
glTF shading and alpha cutoff behavior are approximated. Extract the ZIP before
importing and keep the files together. STL stores triangle geometry only and
does not guarantee a watertight printable mesh. FBX 7.4 has 32-bit node offsets;
an output exceeding its 4 GiB offset limit fails cleanly.

The existing streamed GLB builder writes an intermediate file. Model converters
read metadata plus one mesh primitive at a time; they never snapshot all world
geometry into RAM. FBX patches binary offsets in a temporary file. Cancellation,
write failures and validation errors remove temporary files. An existing output
is replaced only after the completed output is flushed and closed. Image data
is decoded and checked against dimension/pixel limits before a save dialog is
opened; WebP is encoded by the WebView and independently validated in Go.

Wails adds `ExportModel(request)` and `SaveImage(request)` to main and map
windows. Legacy PNG/GLB methods remain available. The authenticated map bridge
adds format/options to `/scene-export` and GET progress / DELETE cancellation;
the host still chooses paths. The contract is in `map-window-api.yaml`.

Validation includes actual image decoders for all five formats; converter tests
for transforms, alpha and cancellation; independent Three.js FBX/OBJ/STL imports;
actual Mediabunny MOV/Matroska muxing and demuxing; local HTTP contract tests; and
frontend interaction/range tests. Native Blender import and WebView hardware
encoding require testing on the target application. The session's browser host
was unavailable and approval review blocked a standalone visual test browser,
so no rendered screenshot review was possible here.

Format references:
- [Blender pipeline support](https://www.blender.org/features/pipeline/)
- [MDN image format guide](https://developer.mozilla.org/en-US/docs/Web/Media/Guides/Formats/Image_types)
- [Mediabunny output formats](https://mediabunny.dev/guide/output-formats)

## Verified build (2026-11-02)

Version: 1.63.1-export-studio-wails.20261102-dev

- Frontend: 119 suites, 371 tests passed; TypeScript and Vite production build passed.
- Go: ui/desktop (gui), internal/scene, internal/modelexport, internal/exportimage and assets passed.
- Windows GUI subsystem: 2 (no companion console window).
- Embedded vanilla asset bundle: 164,311,378 bytes; included in the executable.
- Executable: builds/resourcepack-fix/Craft-Box.exe (250798080 bytes).
- SHA-256: 39FE4BEFD1113D552D7DE3ADFF8E78A94D80B2DD90484A03FE3F412D61FC0485

The Go build exited 0 with a non-fatal module stat-cache write warning outside
this workspace. Native UI/Blender and hardware encoding remain unverified as
noted above; actual file decoders, importers and container muxers were exercised.