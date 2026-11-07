# Export studio

Audience: CraftBox users preparing screenshots/maps and captured worlds for
Blender, general editing, sharing, and cinematic delivery. Five useful common
formats per output family; no unverifiable claim of a global popularity ranking.

- Images: PNG, JPEG, WebP, TIFF, BMP.
- Static 3D: GLB, glTF, OBJ, binary FBX, binary STL.
- Motion: MP4, WebM, MOV, MKV, animated GIF.

Use existing studio colors: workspace #1e1e1e, panels #262626, separators
#414141, foreground #eeeeee, secondary #aaaaaa, primary #1473e6. Keep Manrope
and the application's compact 12px controls; 13px section headings and 18px
preview figures. Left-align text and form controls. Large preview remains the
visual focus; no decorative gradients or unrelated dashboard metrics.

```
Export image / model / sequence                         [close]
Format list       Settings                  Preview and output
5 formats         Name / presets             Current view
use-case hints    Appropriate options        Exact size or source
                  Advanced settings          Files / compatibility
---------------------------------------------------------------
Status / progress                       Cancel     Export…
```

Images expose dimensions, alpha support, JPEG/WebP quality and matte color.
Maps reuse the studio but retain region, pixels-per-block and annotation options.
Models expose format capabilities, world/center origin, scale, up axis and
texture inclusion. Multi-file models are ZIP packages with relative references;
STL explicitly carries geometry only. FBX is binary so Blender can import it.
Cinematics expose five real containers, dimensions, frame rate, bitrate, audio
and a selected time range. Format-specific limitations remain visible.

Save dialogs choose destinations in the main window. Map-window HTTP export
keeps capability protection and server-selected destinations; no caller paths.
Progress, cancellation, validation, atomic completion and reopening saved output
are shared. Existing PNG and GLB API entrypoints remain compatible wrappers.

Implementation keeps the streaming GLB builder as the canonical scene source.
Other model writers convert a disk-spooled GLB one mesh part at a time; they do
not take a whole-world geometry snapshot. Encoders must produce actual format
contents, checked using independent decoders/loaders where available.
