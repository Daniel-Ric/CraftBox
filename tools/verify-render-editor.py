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
    if page.get_by_role('dialog',name='Welcome to CraftBox').count():
        page.get_by_role('button',name='Open workspace',exact=True).click()
    page.get_by_role('button',name='Render editor',exact=True).click()
    export = page.get_by_role('button',name='Export PNG',exact=True)
    try:
        export.wait_for()
    except Exception:
        print('BROWSER ERRORS', errors)
        print(page.locator('body').inner_text()[:5000])
        page.screenshot(path=str(args.output/'failure.png'),full_page=True)
        raise
    try:
        page.wait_for_function("document.querySelector('.Render editor [role=alert]') || (!document.querySelector('.render-build') && [...document.querySelectorAll('button')].some(b=>b.textContent==='Export PNG'&&!b.disabled))",timeout=120000)
        assert not page.locator('.Render editor [role=alert]').count(), page.locator('.Render editor [role=alert]').all_text_contents()
    except Exception:
        print('BROWSER ERRORS', errors, flush=True)
        print(page.locator('body').inner_text()[:5000], flush=True)
        page.screenshot(path=str(args.output/'failure.png'),full_page=True)
        raise
    page.wait_for_timeout(1500)
    assert not errors, errors
    canvas = page.locator('.render-canvas canvas')
    if args.orbit_check:
        from io import BytesIO
        def sky_crop():
            with Image.open(BytesIO(canvas.screenshot())) as screenshot:
                return screenshot.crop((0,0,screenshot.width//8,screenshot.height//3)).tobytes()
        box=canvas.bounding_box()
        initial_sky=sky_crop()
        page.mouse.move(box['x']+box['width']/2,box['y']+box['height']/2)
        page.mouse.down()
        page.mouse.move(box['x']+box['width']/2,box['y']+box['height']/2+80,steps=12)
        page.mouse.up()
        page.wait_for_timeout(1500)
        tilted_sky=sky_crop()
        assert initial_sky!=tilted_sky, 'Skybox ignores vertical orbit rotation'
        page.mouse.wheel(0,-300)
        page.wait_for_timeout(1500)
        assert tilted_sky!=sky_crop(), 'Skybox ignores model zoom'
        print('PASS sky follows vertical orbit and zoom',flush=True)
        page.get_by_role('button',name='Fit to view',exact=True).click()
        page.get_by_role('button',name='Isometrisch',exact=True).click()
        page.wait_for_timeout(1000)
    before = canvas.screenshot()
    page.get_by_text('Weitere Lichtstimmungen',exact=True).click()
    page.get_by_role('button',name='Evening sun',exact=True).click()
    page.wait_for_timeout(1200)
    assert before != canvas.screenshot(), 'lighting preset did not change rendered image'
    page.screenshot(path=str(args.output/'editor-preview.png'),full_page=True)
    from io import BytesIO
    with Image.open(BytesIO(canvas.screenshot())) as preview:
        sky_colors = preview.crop((0,0,preview.width//4,preview.height)).resize((64,64)).convert('RGB').getcolors(100000)
        assert len(sky_colors)>20, 'HDR background is flat instead of showing the panorama'
    if not args.quick:
        page.get_by_role('tab',name='Output',exact=True).click()
        page.get_by_label('Resolution',exact=True).select_option('3840')
        start = time.monotonic()
        export.click()
        page.get_by_role('dialog').wait_for()
        page.screenshot(path=str(args.output/'png-export-dialog.png'),full_page=True)
        page.get_by_role('button',name='Export …',exact=True).click()
        page.get_by_text('Saved:',exact=False).wait_for(timeout=120000)
        with Image.open(saved[0]) as image:
            assert image.size == (3840,2160), image.size
            assert len(image.resize((128,72)).getcolors(1000000)) > 100, 'blank or flat export'
        print(f'PASS 4K PNG {time.monotonic()-start:.2f}s: {saved[0]}')
        page.get_by_label('Resolution',exact=True).select_option('1920')
        page.get_by_label('Transparent background').check()
        page.get_by_label('Image format',exact=True).select_option('1')
        export.click()
        page.get_by_role('button',name='Export …',exact=True).click()
        page.wait_for_function("document.querySelector('.render-feedback:not(.error)')?.textContent.includes('render-2.png')",timeout=120000)
        with Image.open(saved[1]) as image:
            assert image.size == (1920,1920)
            assert image.mode == 'RGBA'
            assert image.getchannel('A').getextrema() == (0,255)
        print(f'PASS transparent square PNG: {saved[1]}')
    page.get_by_role('button',name='View',exact=True).click()
    page.get_by_role('menuitem',name='Top view',exact=True).click()
    page.wait_for_timeout(1000)
    assert before != canvas.screenshot(), 'camera view did not change'
    if args.skies:
        page.get_by_role('tab',name='Output',exact=True).click()
        page.get_by_label('Transparent background').uncheck()
        page.get_by_role('tab',name='Lighting',exact=True).click()
        for sky in ['daylight','dawn','sunset','night']:
            page.get_by_label('Skybox',exact=True).select_option(sky)
            page.wait_for_function("!document.querySelector('.render-build')",timeout=60000)
            page.wait_for_timeout(300)
            assert not page.locator('.Render editor [role=alert]').count(), page.locator('.Render editor [role=alert]').all_text_contents()
            canvas.screenshot(path=str(args.output/f'sky-{sky}.png'))
        page.get_by_label('Custom HDR skybox',exact=True).set_input_files('ui/desktop/frontend/public/skies/kloofendal_48d_partly_cloudy_puresky_2k.hdr')
        page.wait_for_function("!document.querySelector('.render-build')",timeout=60000)
        assert page.get_by_label('Skybox',exact=True).input_value()=='custom'
        for quality in ['Fast','Balanced','Fine']:
            page.get_by_role('tab',name='Details',exact=True).click()
            page.get_by_role('button',name=quality,exact=True).click()
            page.wait_for_timeout(500)
            assert page.get_by_role('button',name=quality,exact=True).get_attribute('aria-pressed')=='true'
            canvas.screenshot(path=str(args.output/f'quality-{quality}.png'))
        print('PASS four bundled HDR skies, custom HDR import and three quality levels',flush=True)
    page.set_viewport_size(dict(width=800,height=850))
    page.screenshot(path=str(args.output/'editor-small.png'),full_page=True)
    page.set_viewport_size(dict(width=1440,height=960))
    page.get_by_role('button',name='Open world file',exact=True).click()
    page.screenshot(path=str(args.output/'import-dark.png'),full_page=True)
    page.get_by_role('button',name='Cancel',exact=True).click()
    page.get_by_role('menuitem',name='View',exact=True).click()
    page.get_by_role('menuitem',name='Light interface',exact=True).click()
    page.screenshot(path=str(args.output/'editor-light.png'),full_page=True)
    page.get_by_role('button',name='Export PNG',exact=True).click()
    page.get_by_role('img',name='Current render view',exact=True).wait_for()
    page.wait_for_function("document.querySelector('.export-frame img')?.naturalWidth>0")
    previous=page.get_by_role('img',name='Current render view',exact=True).get_attribute('src')
    check=page.get_by_role('dialog').get_by_label('Transparent background',exact=True)
    check.set_checked(not check.is_checked())
    page.wait_for_function("before=>document.querySelector('.export-frame img')?.src!==before",arg=previous)
    page.screenshot(path=str(args.output/'png-export-light.png'),full_page=True)
    page.get_by_role('button',name='Cancel',exact=True).click()
    page.get_by_role('button',name='Open world file',exact=True).click()
    page.screenshot(path=str(args.output/'import-light.png'),full_page=True)
    page.get_by_role('button',name='Cancel',exact=True).click()
    assert not errors, errors
    print(json.dumps(dict(chunks=total,browserErrors=errors,caption=page.locator('.render-caption').inner_text()),indent=2))
    browser.close()
