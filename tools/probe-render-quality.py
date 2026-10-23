"""Exercise the real render editor with recorded Go terrain-page responses."""
import argparse
import base64
import json
import time
from pathlib import Path
from urllib.parse import parse_qs, urlparse
from PIL import Image
from playwright.sync_api import sync_playwright

p = argparse.ArgumentParser()
p.add_argument('--pages', type=Path, required=True)
p.add_argument('--url', default='http://127.0.0.1:5201')
p.add_argument('--output', type=Path, default=Path('tmp/Render editor/browser'))
p.add_argument('--quick', action='store_true')
p.add_argument('--skies', action='store_true')
p.add_argument('--orbit-check', action='store_true')
args = p.parse_args()
pages = json.loads(args.pages.read_text(encoding='utf-8'))
args.output.mkdir(parents=True, exist_ok=True)
total = sum(len(x['chunks']) for x in pages)
indices = {'first': 0}
for i, page in enumerate(pages[:-1]):
    indices[','.join(map(str, page['next']))] = i + 1
state = dict(revision=1, command='worlds', running=False, connection=0,
             worlds=[dict(id='render-fixture', name='CubeCraft Render', chunks=total, state='finished')],
             packs=[], skins=[], player=[0,0,0], hasPlayer=False, loggedIn=False)
bootstrap = dict(version='Render verification', minecraft='1.26.52', commands=[], state=state)
init = '''window.runtime={EventsOn:()=>()=>{}};
window.go={desktop:{App:{
GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[],
GetRenderScenePage:async(id,r)=>fetch('/__render-page?cursor='+encodeURIComponent(r.cursor?r.cursor.join(','):'first')).then(r=>r.json()),
SaveRenderPNG:async(name,data)=>fetch('/__save-render',{method:'POST',body:JSON.stringify({name,data})}).then(r=>r.json()),
OpenRenderWorld:async()=>({id:'imported-fixture',name:'Imported world',chunks:TOTAL}),
CancelRenderImport:async()=>{},CloseRenderWorld:async()=>{},OpenResultFolder:async()=>{}
}}};'''.replace('BOOTSTRAP', json.dumps(bootstrap)).replace('TOTAL', str(total))
with sync_playwright() as pw:
    browser = pw.chromium.launch(channel='msedge', headless=True,
        args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
    page = browser.new_page(viewport=dict(width=1520,height=980),device_scale_factor=1)
    errors = []
    page.on('framenavigated', lambda frame: print('NAV',frame.url,flush=True) if frame==page.main_frame else None)
    page.on('response', lambda response: print('SKY',response.status,response.url,flush=True) if '.hdr' in response.url else None)
    page.on('pageerror', lambda e: errors.append(str(e)))
    page.on('console', lambda m: errors.append(m.text) if m.type == 'error' and ('WebGL' in m.text or 'THREE.' in m.text or 'shader' in m.text.lower()) else None)
    page.add_init_script(init)
    page.route('**/__render-page?*', lambda route: route.fulfill(status=200,content_type='application/json',
        body=json.dumps(pages[indices[parse_qs(urlparse(route.request.url).query)['cursor'][0]]])))
    saved = []
    def save(route):
        data = json.loads(route.request.post_data)
        path = args.output / f'render-{len(saved)+1}.png'
        path.write_bytes(base64.b64decode(data['data'].split(',',1)[1]))
        saved.append(path)
        route.fulfill(status=200,content_type='application/json',body=json.dumps(str(path)))
    page.route('**/__save-render', save)
    page.goto(args.url,wait_until='networkidle')
    page.set_viewport_size(dict(width=1600,height=1000))
    page.evaluate('''async()=>{
      const {RenderEngine}=await import('/src/RenderCanvas.tsx');
      const {renderDefaults}=await import('/src/render-settings.ts');
      const host=document.createElement('div');host.style.cssText='position:fixed;inset:0;background:#222;z-index:9999;display:flex;align-items:center;justify-content:center';document.body.append(host);
      window.engine=new RenderEngine(host,{current:{worldId:'render-fixture',settings:{...renderDefaults,aspect:1.6},onProgress:()=>{},onReady:()=>{},onError:e=>console.error(e),onSkyLoading:()=>{}}});
    }''')
    page.wait_for_function('window.engine && !window.engine.loading',timeout=120000)
    page.wait_for_timeout(500)
    print(page.evaluate('''()=>{let e=window.engine;return{bounds:e.bounds.min.toArray().concat(e.bounds.max.toArray()),samples:e.composer.renderTarget1.samples,shadowBias:e.light.shadow.bias,normalBias:e.light.shadow.normalBias,shadowBounds:[e.light.shadow.camera.left,e.light.shadow.camera.right,e.light.shadow.camera.near,e.light.shadow.camera.far],atlas:e.texture.image.width};}'''),flush=True)
    page.evaluate('()=>{window.engine.camera.zoom=3;window.engine.camera.updateProjectionMatrix();window.engine.dirty=true;}')
    page.wait_for_timeout(500)
    page.screenshot(path=str(args.output/'01-baseline.png'))
    flags=page.evaluate('()=>{let count=0;window.engine.terrain.traverse(m=>{const f=m.geometry?.getAttribute("foliage");if(f)for(let i=0;i<f.count;i++)count+=f.getX(i)>0.5?1:0;});return count;}')
    assert flags>0,'no crossed vegetation found in the fixture'
    page.evaluate('()=>{let e=window.engine;e.apply({...e.options,pitchLocked:true});}')
    before_angles=page.evaluate('()=>[window.engine.controls.getPolarAngle(),window.engine.controls.getAzimuthalAngle()]')
    page.mouse.move(800,500);page.mouse.down();page.mouse.move(920,575,steps=8);page.mouse.up();page.wait_for_timeout(250)
    after_angles=page.evaluate('()=>[window.engine.controls.getPolarAngle(),window.engine.controls.getAzimuthalAngle()]')
    assert abs(before_angles[0]-after_angles[0])<0.00001,(before_angles,after_angles)
    assert abs(before_angles[1]-after_angles[1])>0.05,(before_angles,after_angles)
    page.evaluate('()=>{let e=window.engine;e.view(0,-89.5);}')
    assert page.evaluate('window.engine.controls.getPolarAngle()>Math.PI/2'),'bottom perspective is clamped'
    page.evaluate('()=>{let e=window.engine;e.apply({...e.options,pitchLocked:false});e.view(45,35);}')
    print('PASS vegetation attributes, vertical inclination lock with horizontal orbit, bottom perspective',flags,flush=True)
    page.evaluate('()=>{let e=window.engine;e.options={...e.options,ao:0};e.ao.enabled=false;e.normals.enabled=false;e.draw();}')
    page.screenshot(path=str(args.output/'02-no-ao.png'))
    page.evaluate('()=>{let e=window.engine;e.light.castShadow=false;e.renderer.shadowMap.needsUpdate=true;e.draw();}')
    page.screenshot(path=str(args.output/'03-no-shadows.png'))
    page.evaluate('()=>{let e=window.engine;for(let m of Object.values(e.materials)){m.map=null;m.needsUpdate=true;}e.draw();}')
    page.screenshot(path=str(args.output/'04-no-textures.png'))
    page.evaluate('()=>{let e=window.engine;e.light.castShadow=true;for(let m of Object.values(e.materials)){m.map=e.texture;m.needsUpdate=true;}e.light.shadow.normalBias=0.4;e.light.shadow.bias=-0.0003;e.renderer.shadowMap.needsUpdate=true;e.draw();}')
    page.screenshot(path=str(args.output/'05-larger-shadow-bias.png'))
    print('Quality isolation finished',errors,flush=True)
    browser.close()
