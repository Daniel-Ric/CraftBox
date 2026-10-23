# Bundled HDR skies

These original photographed 2K HDR panoramas are from Poly Haven and are
redistributed under CC0 1.0: https://polyhaven.com/license

- Kloofendal 48d Partly Cloudy (Pure Sky), Greg Zaal / Jarod Guest:
  https://polyhaven.com/a/kloofendal_48d_partly_cloudy_puresky
- Kloppenheim 06 (Pure Sky), Greg Zaal / Jarod Guest:
  https://polyhaven.com/a/kloppenheim_06_puresky
- Belfast Sunset (Pure Sky), Dimitrios Savva / Greg Zaal / Jarod Guest:
  https://polyhaven.com/a/belfast_sunset_puresky
- Rogland Clear Night:
  https://polyhaven.com/a/rogland_clear_night

Original download URLs and SHA-256 hashes are recorded in `sources.json`.
The application loads these local files and does not contact Poly Haven.
Run `python tools/fetch-render-skies.py` from the repository root to refresh
the bundled source files. Runtime lighting is exposure-normalized; the sun
disc is limited in the separate diffuse-lighting copy so that the shadowed
directional light supplies the direct sunlight.
