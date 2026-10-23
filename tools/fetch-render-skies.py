"""Fetch the four redistributable CC0 HDR panoramas used by CraftBox."""
from pathlib import Path
from urllib.request import Request, urlopen
import hashlib
import json

root = Path(__file__).resolve().parents[1] / 'ui/desktop/frontend/public/skies'
root.mkdir(parents=True, exist_ok=True)
assets = ['kloofendal_48d_partly_cloudy_puresky','kloppenheim_06_puresky','belfast_sunset_puresky','rogland_clear_night']
manifest = []
for asset in assets:
    name = asset + '_2k.hdr'
    url = 'https://dl.polyhaven.org/file/ph-assets/HDRIs/hdr/2k/' + name
    request = Request(url, headers={'User-Agent':'CraftBox-RenderAssets/1.0'})
    data = urlopen(request, timeout=90).read()
    if not data.startswith((b'#?RADIANCE', b'#?RGBE')):
        raise ValueError('Not a Radiance HDR: ' + asset)
    (root/name).write_bytes(data)
    manifest.append(dict(file=name,source='https://polyhaven.com/a/'+asset,download=url,license='CC0-1.0',sha256=hashlib.sha256(data).hexdigest()))
    print(name, len(data), flush=True)
(root/'sources.json').write_text(json.dumps(manifest, indent=2)+'\n', encoding='utf-8')
