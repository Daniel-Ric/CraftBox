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
p.add_argument('--dpr',type=float,default=1)
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
    page = browser.new_page(viewport=dict(width=1520,height=980),device_scale_factor=args.dpr)
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
    page.get_by_role('button',name='Render editor',exact=True).click()
    page.get_by_role('button',name='Export PNG',exact=True).wait_for()
    page.wait_for_function("!document.querySelector('.render-build') && document.querySelector('.render-canvas canvas')",timeout=120000)
    page.wait_for_timeout(500)
    canvas=page.locator('.render-canvas canvas')
    results={}
    for label in ['Vibrant island','Flower garden','Warm film','Winter evening','Clear & neutral']:
        page.get_by_role('button',name=label,exact=False).click()
        page.wait_for_function("!document.querySelector('.render-build')",timeout=60000)
        page.wait_for_timeout(300)
        assert not errors,errors
        image=canvas.screenshot()
        results[label]=image
        page.screenshot(path=str(args.output/(label.replace(' ','-')+'.png')),full_page=True)
    assert len(set(results.values()))==5,'reference looks do not change the image'
    page.get_by_role('tab',name='Color',exact=True).click()
    page.get_by_label('Saturation Value',exact=True).fill('1.4')
    assert page.get_by_label('Saturation',exact=True).input_value()=='1.4'
    page.get_by_label('Temperature Value',exact=True).fill('-0.2')
    assert page.get_by_label('Temperature',exact=True).input_value()=='-0.2'
    page.wait_for_timeout(200)
    before=canvas.screenshot()
    page.get_by_label('Enable color grading',exact=True).uncheck()
    page.wait_for_timeout(200)
    assert canvas.screenshot()!=before,'grading bypass has no effect'
    page.get_by_label('Enable color grading',exact=True).check()
    page.screenshot(path=str(args.output/'color-inspector.png'),full_page=True)
    page.get_by_role('button',name='Cinematic editor',exact=True).click()
    page.wait_for_function("document.querySelector('.cinema-studio') && !document.querySelector('.cinema-loading')",timeout=120000)
    page.get_by_role('button',name='Scene styling',exact=True).click()
    page.get_by_role('button',name='Flower garden',exact=False).click()
    assert not page.locator('.cinema-inspector').is_visible(),'duplicate camera/image inspectors'
    page.wait_for_function("!document.querySelector('.render-build')",timeout=60000)
    page.set_viewport_size(dict(width=1240,height=820))
    page.screenshot(path=str(args.output/'cinematic-inspector.png'),full_page=True)
    inspector=page.locator('.render-inspector').bounding_box()
    assert inspector['width']>=260,inspector
    assert not errors,errors
    print('PASS five distinct looks, color numeric/range editing, grading comparison, shared cinematic inspector; no browser errors',flush=True)
    browser.close()
