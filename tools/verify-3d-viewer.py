"""Visual check using a real BDS-derived scene snapshot and headless WebGL.

Start the frontend dev server, then run with --scene tmp/bds-3d-fixture/scene.json.
The desktop transport is supplied by a test bridge; the mesh/atlas are real.
"""
import argparse
import json
import pathlib
import base64
import io
import struct
from PIL import Image
from playwright.sync_api import sync_playwright

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--scene', required=True, type=pathlib.Path)
parser.add_argument('--url', default='http://127.0.0.1:5182')
parser.add_argument('--output', type=pathlib.Path, default=pathlib.Path('tmp/scene-visual-check'))
args = parser.parse_args()
args.output.mkdir(parents=True, exist_ok=True)
snapshot = json.loads(args.scene.read_text(encoding='utf8'))
atlas = Image.open(io.BytesIO(base64.b64decode(snapshot['atlas'].split(',')[1]))).convert('RGBA')
cells = {(0, 0)}
for chunk in snapshot['chunks']:
    for part in chunk['parts']:
        for u, v in zip(part['uvs'][::2], part['uvs'][1::2]):
            cells.add((int(u*2048)//18*18, int(v*2048)//18*18))
        vertices = len(part['positions'])//3
        raw = struct.pack('<'+'f'*len(part['positions']), *part['positions'])
        raw += struct.pack('<'+'b'*len(part['normals']), *(round(n*127) for n in part['normals']))
        raw += bytes((-len(raw))%4)
        raw += struct.pack('<'+'f'*len(part['uvs']), *part['uvs'])
        raw += struct.pack('<'+'I'*len(part['indices']), *part['indices'])
        part.update(vertices=vertices, indices=len(part['indices']), data=base64.b64encode(raw).decode())
        for key in ('positions','normals','uvs'): del part[key]
snapshot['patches'] = [dict(x=x,y=y,size=18,data=base64.b64encode(atlas.crop((x,y,x+18,y+18)).tobytes()).decode()) for x,y in sorted(cells)]
snapshot['atlasSize'] = 2048
del snapshot['atlas']
state = dict(revision=1, command='worlds', running=True, stopping=False, connection=5,
             activeWorldId='bds-fixture', worlds=[dict(id='bds-fixture', name='BDS-Testwelt',
             state='Capturing', player=[8, -58, 8], hasPlayer=True, chunks=2)], packs=[], skins=[],
             player=[8, -58, 8], hasPlayer=True, loggedIn=False, authenticating=False)
bootstrap = dict(mapWindow=True, version='3D verification', minecraft='1.26.52', theme='dark',
                 commands=[], state=state)
init = '''window.callbacks={};window.runtime={EventsOn:(name,fn)=>{window.callbacks[name]=fn;return()=>delete window.callbacks[name]}};
window.fixture=SCENE;window.exported='';window.requested=[];
window.go={desktop:{MapWindowApp:{GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[],
GetScenePage:async(id,request)=>{
window.requested.push({id,...request});
if(window.failNext){window.failNext=false;throw new Error('context deadline exceeded')}
const s=window.fixture,first=request.after===0&&!request.cursor;
const start=request.cursor?s.chunks.findIndex(c=>c.x===request.cursor[0]&&c.z===request.cursor[1])+1:0;
const chunks=request.after===s.sequence?[]:s.chunks.slice(start,start+1);
const complete=chunks.length===0||start+1>=s.chunks.length;
if(start>0)await new Promise(resolve=>setTimeout(resolve,250));
return {...s,scope:0,reset:first,complete,next:complete?null:[chunks[0].x,chunks[0].z],total:s.chunks.length,chunks,patches:request.atlasRevision===s.atlasRevision?[]:s.patches}
},
ExportScene:async(id)=>{window.exported=id;return 'models/bds-fixture.glb'},OpenResultFolder:async()=>{},EditWorld:async()=>{}}}};'''.replace('SCENE', json.dumps(snapshot)).replace('BOOTSTRAP', json.dumps(bootstrap))
with sync_playwright() as p:
    browser = p.chromium.launch(channel='msedge', headless=True,
                                args=['--use-angle=swiftshader', '--enable-unsafe-swiftshader'])
    page = browser.new_page(viewport=dict(width=1440, height=920), device_scale_factor=1)
    errors = []
    page.on('pageerror', lambda error: errors.append(str(error)))
    page.add_init_script(init)
    page.goto(args.url, wait_until='networkidle')
    page.screenshot(path=str(args.output/'bootstrap.png'), full_page=True)
    page.get_by_role('button', name='3D', exact=True).click()
    page.get_by_role('button', name='Export GLB').wait_for(state='visible')
    page.wait_for_function("document.querySelector('.scene-loading-progress') && document.querySelector('.scene-caption')?.textContent.includes('1 Chunks')")
    page.wait_for_function("!document.querySelector('.scene-message') && document.querySelector('.scene-caption')?.textContent.includes('2 Chunks')")
    page.locator('.scene-canvas canvas').wait_for()
    assert not page.get_by_role('alert').count(), page.locator('body').inner_text()
    assert '0 blocks ohne Textur' not in page.locator('.scene-caption').inner_text()
    page.screenshot(path=str(args.output/'bds-world-3d.png'), full_page=True)
    assert page.locator('html').evaluate("e=>e.classList.contains('dark')")
    dark = page.locator('.scene-viewer').evaluate("e=>getComputedStyle(e).backgroundColor")
    assert dark == 'rgb(31, 31, 31)', dark
    for theme,expected in [('gray','rgb(59, 59, 59)'),('graphite','rgb(20, 20, 20)')]:
        page.evaluate("theme=>window.callbacks['desktop:theme'](theme)",theme)
        page.wait_for_timeout(150)
        actual=page.locator('.scene-viewer').evaluate("e=>getComputedStyle(e).backgroundColor")
        assert actual==expected,(theme,actual)
        pixel=Image.open(io.BytesIO(page.locator('.scene-canvas canvas').screenshot())).convert('RGB').getpixel((0,0))
        rgb=tuple(int(v.strip()) for v in expected[4:-1].split(','))
        assert all(abs(a-b)<=1 for a,b in zip(pixel,rgb)),(theme,pixel,rgb)
        page.screenshot(path=str(args.output/(theme+'-3d.png')),full_page=True)
    page.evaluate("window.callbacks['desktop:theme']('light')")
    page.wait_for_function("document.documentElement.dataset.theme==='light'&&!document.documentElement.classList.contains('dark')")
    page.wait_for_timeout(100)
    light = page.locator('.scene-viewer').evaluate("e=>getComputedStyle(e).backgroundColor")
    assert light == 'rgb(255, 255, 255)', light
    page.screenshot(path=str(args.output/'light-mode-3d.png'), full_page=True)
    page.evaluate("window.callbacks['desktop:theme']('dark')")
    page.wait_for_function("document.documentElement.dataset.theme==='dark'")

    before = page.locator('.scene-canvas canvas').screenshot()
    page.locator('.scene-canvas canvas').focus()
    page.keyboard.down('KeyW')
    page.wait_for_timeout(250)
    page.keyboard.up('KeyW')
    after = page.locator('.scene-canvas canvas').screenshot()
    assert before != after, 'Camera navigation did not change rendered pixels'
    page.get_by_role('button', name='Export GLB').click()
    page.get_by_role('button', name='Open folder').wait_for()
    assert page.evaluate('window.exported') == 'bds-fixture'
    page.evaluate('window.failNext=true')
    page.get_by_role('alert').wait_for()
    assert page.locator('.scene-caption').inner_text().startswith('2 Chunks')
    page.get_by_role('button', name='Try again').click()
    page.get_by_role('alert').wait_for(state='hidden')

    page.evaluate("window.fixture={...window.fixture,sequence:window.fixture.sequence+1,reset:false,chunks:[{...window.fixture.chunks[1],parts:[],missingTextures:0,approximateBlocks:0}]}")
    page.wait_for_function('window.requested.some(r=>r.after>0)')
    page.wait_for_timeout(1300)
    page.get_by_role('button', name='Overview', exact=True).click()
    page.screenshot(path=str(args.output/'live-chunk-update.png'), full_page=True)
    page.get_by_role('button', name='Select region', exact=True).click()
    assert page.get_by_role('button', name='2D', exact=True).get_attribute('aria-pressed') == 'true'
    page.set_viewport_size(dict(width=760, height=600))
    page.get_by_role('button', name='3D', exact=True).click()
    page.wait_for_timeout(1000)
    assert not page.get_by_role('alert').count()
    page.screenshot(path=str(args.output/'compact-3d.png'), full_page=True)
    assert not errors, errors
    print(json.dumps(dict(browserErrors=errors, source=args.scene.as_posix(), screenshots=str(args.output)), indent=2))
    browser.close()
