# Lossless GLB size reduction

All native export paths share the streaming writer: captures, imported worlds
and saved-project pages. The writer now welds only bit-identical float32
position/normal/UV tuples within one primitive, and selects unsigned 16-bit
indices where the highest index is below 65535. Larger indices remain 32-bit.
The all-ones index is reserved by the
[glTF 2.0 specification](https://registry.khronos.org/glTF/specs/2.0/glTF-2.0.html#meshes).

Triangle order, UV seams, hard edges, signed zero, chunk transforms, material
boundaries, full float32 attributes and exact embedded PNG bytes remain intact.
No texture resizing, simplification, quantization or new decoder is introduced.
Welding skips primitives above 128K vertices to bound temporary hashing memory;
this is an optimization budget, not an export limit. Cancellation and streaming
remain supported.

The real Hive fixture changed from **470,444,448 to 431,490,044 bytes**, saving
**8.28%**. Both files have 1,755 chunks and 6,139,178 triangles. Vertices decreased
from 12,278,356 to 12,198,074. Expanded triangle attributes have identical SHA256
`34b12df6e5ee9baca1b6eb0f4d838c3c394754da4711d389482f593b5faeb52b`;
PNG bytes and chunk translations also match. The final standalone roundtrip
sampled 78 MiB peak Go heap. Savings depend on the model.

`tmp/glb-check/main.go` reproduces the streaming comparison. Regression tests
cover exact attributes, seams, input immutability, triangle order, index-width
boundaries, alignment, cancellation and material separation.
