"""Exercise placeable lights in the real WebGL editors using recorded terrain."""
import argparse
import base64
import io
import json
from PIL import Image
from pathlib import Path
from urllib.parse import parse_qs, urlparse
from playwright.sync_api import sync_playwright

parser = argparse.ArgumentParser()
parser.add_argument('--url', default='http://127.0.0.1:5202')
parser.add_argument('--pages', type=Path, default=Path('tmp/Render editor/terrain-pages.json'))
parser.add_argument('--output', type=Path, default=Path('tmp/studio-lights'))
parser.add_argument('--only-render', action='store_true')
parser.add_argument('--debug-panorama', action='store_true')
parser.add_argument('--transparent-panorama', action='store_true')
parser.add_argument('--panorama-width', type=int, choices=[1024, 2048, 4096], default=1024)
parser.add_argument('--cancel-panorama', action='store_true')
parser.add_argument('--verify-stills', action='store_true')
args = parser.parse_args()
args.output.mkdir(parents=True, exist_ok=True)
pages = json.loads(args.pages.read_text(encoding='utf-8'))
indices = {'first': 0}
for i, item in enumerate(pages[:-1]):
    indices[','.join(map(str, item['next']))] = i + 1
state = dict(revision=1, command='worlds', running=False,
             worlds=[dict(id='render-fixture', name='Studio terrain', chunks=sum(len(p['chunks']) for p in pages), state='finished')],
             packs=[], skins=[], player=[0, 0, 0], loggedIn=False)
bootstrap = dict(version='Studio light verification', minecraft='1.26.52', commands=[], state=state)
init = '''window.runtime={EventsOn:()=>()=>{}};
window.__saved=[];window.go={desktop:{App:{
GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[],
GetRenderScenePage:async(id,r)=>fetch('/__page?cursor='+encodeURIComponent(r.cursor?r.cursor.join(','):'first')).then(r=>r.json()),
SaveStudioProject:async(worldId,name,document)=>{window.__saved.push(JSON.parse(document));return 'C:/test/lights.craftbox';},
SetStudioProjectDirty:async()=>{},ConfirmStudioProjectClose:async()=>{}
 ,SaveRenderPNG:async(name,data)=>{window.__png=data;return 'C:/test/panorama.png';}
}}};'''.replace('BOOTSTRAP', json.dumps(bootstrap))

with sync_playwright() as pw:
    browser = pw.chromium.launch(channel='msedge', headless=True,
        args=['--use-angle=swiftshader', '--enable-unsafe-swiftshader'])
    for cinematic in (False, True):
        if cinematic and args.only_render:
            continue
        page = browser.new_page(viewport=dict(width=1520, height=980), device_scale_factor=1)
        page.set_default_timeout(60000)
        errors = []
        page.on('pageerror', lambda error: errors.append(str(error)))
        page.on('console', lambda message: errors.append(message.text) if message.type == 'error' and any(t in message.text for t in ('WebGL', 'THREE', 'shader')) else None)
        page.add_init_script(init)
        page.route('**/__page?*', lambda route: route.fulfill(status=200, content_type='application/json', body=json.dumps(pages[indices[parse_qs(urlparse(route.request.url).query)['cursor'][0]]])))
        try:
            page.goto(args.url, wait_until='networkidle')
            if page.get_by_role('button', name='Open workspace', exact=True).count():
                page.get_by_role('button', name='Open workspace', exact=True).click()
            page.get_by_role('button', name='Cinematic editor' if cinematic else 'Render editor', exact=True).click()
            page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay')")
            if cinematic:
                page.get_by_role('button', name='Edit 3D path', exact=True).click()
                page.get_by_role('button', name='Edit lighting', exact=True).click()
            else:
                page.get_by_role('tab', name='Lighting', exact=True).click()
            for name in ('Point light', 'Spot light', 'Area light'):
                page.get_by_role('button', name=name + ' add', exact=True).click()
                page.get_by_label('Light output (lumens)', exact=True).fill('80000')
                page.get_by_label('Light position Y', exact=True).fill('0.5')
                page.get_by_label('Light position Y', exact=True).press('Tab')
                page.wait_for_timeout(300)
            page.get_by_role('button', name='Spot light', exact=True).click()
            page.get_by_label('Edit look-at target in view', exact=True).check()
            page.get_by_role('button', name='Point light', exact=True).click()
            page.get_by_role('button', name='Spot light', exact=True).click()
            assert not page.get_by_label('Edit look-at target in view', exact=True).is_checked()
            page.get_by_role('button', name='Place in view', exact=True).click()
            canvas = page.locator('.render-canvas canvas')
            box = canvas.bounding_box()

            page.mouse.click(box['x'] + box['width'] * 0.5, box['y'] + box['height'] * 0.6)
            page.get_by_role('button', name='Place in view', exact=True).wait_for()
            page.wait_for_timeout(500)
            page.screenshot(path=str(args.output / ('cinematic.png' if cinematic else 'render.png')))
            page.get_by_role('button', name='Duplicate light', exact=True).click()
            assert page.get_by_label('Light name', exact=True).input_value() == 'Spot light Copy'
            page.get_by_role('button', name='Delete light', exact=True).click()
            assert page.get_by_role('button', name='Spot light', exact=True).count() == 1
            page.get_by_role('tab', name='Details', exact=True).click()
            page.get_by_label('Aperture (f/)', exact=True).fill('1.4')
            page.get_by_label('Scene scale (blocks per meter)', exact=True).fill('100')
            page.get_by_role('button', name='Click to set focus', exact=True).click()
            page.mouse.click(box['x'] + box['width'] * 0.5, box['y'] + box['height'] * 0.6)
            page.get_by_role('button', name='Click to set focus', exact=True).wait_for()
            assert page.get_by_label('Enable depth of field', exact=True).is_checked()
            assert float(page.get_by_label('Focus distance (blocks)', exact=True).input_value()) > 0
            page.get_by_label('Light name', exact=True).count()
            page.wait_for_timeout(500)
            page.screenshot(path=str(args.output / ('cinematic-dof.png' if cinematic else 'render-dof.png')))
            if not cinematic:
                if args.verify_stills:
                    from PIL import ImageChops, ImageStat
                    def still(name, transparent=False):
                        page.evaluate('window.__png=undefined')
                        page.get_by_role('button', name='Export PNG', exact=True).click()
                        page.get_by_label('Longest edge', exact=True).select_option('720')
                        page.get_by_role('dialog').get_by_label('Transparent background', exact=True).set_checked(transparent)
                        page.get_by_role('button', name='Export …', exact=True).click()
                        page.wait_for_function('!!window.__png', timeout=180000)
                        raw = base64.b64decode(page.evaluate('window.__png').split(',')[1])
                        (args.output / (name + '.png')).write_bytes(raw)
                        return Image.open(io.BytesIO(raw)).copy()
                    blurred = still('lights-dof')
                    page.get_by_label('Enable depth of field', exact=True).uncheck()
                    sharp = still('lights-sharp')
                    dof_difference = sum(ImageStat.Stat(ImageChops.difference(sharp.convert('RGB'), blurred.convert('RGB'))).mean) / 3
                    assert dof_difference > .05, dof_difference
                    page.get_by_role('tab', name='Lighting', exact=True).click()
                    for name in ('Point light', 'Spot light', 'Area light'):
                        page.get_by_label(name + ' aktiv', exact=True).uncheck()
                    unlit = still('without-studio-lights')
                    light_difference = sum(ImageStat.Stat(ImageChops.difference(sharp.convert('RGB'), unlit.convert('RGB'))).mean) / 3
                    assert light_difference > .05, light_difference
                    for name in ('Point light', 'Spot light', 'Area light'):
                        page.get_by_label(name + ' aktiv', exact=True).check()
                    page.get_by_role('tab', name='Details', exact=True).click()
                    page.get_by_label('Enable depth of field', exact=True).check()
                    transparent = still('transparent-lights-dof', True)
                    alpha = transparent.convert('RGBA').getchannel('A')
                    assert alpha.getextrema() == (0, 255) and sum(alpha.histogram()[1:255]) > 100
                    print('PASS still exports contain studio lighting and DOF; transparent blur retained', {'lightDifference': light_difference, 'dofDifference': dof_difference}, flush=True)
                    page.evaluate('window.__png=undefined')
                if args.cancel_panorama:
                    page.get_by_role('button', name='360° panorama', exact=True).click()
                    page.get_by_label('Panorama resolution', exact=True).select_option('4096')
                    page.get_by_role('button', name='Render panorama', exact=True).click()
                    page.get_by_role('button', name='Cancel panorama', exact=True).click()
                    page.wait_for_function('!document.querySelector(\'[aria-label="Panorama progress"]\')')
                    assert not page.evaluate('!!window.__png')
                    assert abs(canvas.bounding_box()['width'] - box['width']) < 1
                    print('PASS cancelled panorama restores viewport and saves no file', flush=True)
                if args.transparent_panorama:
                    page.get_by_role('tab', name='Output', exact=True).click()
                    page.get_by_label('Transparent background', exact=True).check()
                if args.debug_panorama:
                    page.evaluate('''()=>{const host=document.querySelector('.render-canvas');let fiber=host[Object.keys(host).find(k=>k.startsWith('__reactFiber'))],engine;while(fiber&&!engine){let hook=fiber.memoizedState;while(hook){if(hook.memoizedState?.current?.capturePanorama)engine=hook.memoizedState.current;hook=hook.next;}fiber=fiber.return;}if(!engine)throw Error('Cannot inspect panorama renderer');const draw=engine.draw;window.__faces=[];engine.draw=function(...args){draw.apply(this,args);if(this.exporting&&this.camera.isPerspectiveCamera){window.__faces.push({image:this.renderer.domElement.toDataURL(),rotation:this.camera.quaternion.toArray(),up:this.camera.up.toArray(),options:this.options});}}}''')
                page.get_by_role('button', name='360° panorama', exact=True).click()
                page.get_by_label('Panorama resolution', exact=True).select_option(str(args.panorama_width))
                page.get_by_role('button', name='Render panorama', exact=True).click()
                page.wait_for_function('!!window.__png', timeout=300000)
                image_data = base64.b64decode(page.evaluate('window.__png').split(',')[1])
                (args.output / ('panorama-transparent.png' if args.transparent_panorama else 'panorama.png')).write_bytes(image_data)
                if args.debug_panorama:
                    for index, face in enumerate(page.evaluate('window.__faces')):
                        (args.output / f'face-{index}.png').write_bytes(base64.b64decode(face.pop('image').split(',')[1]))
                        (args.output / f'face-{index}.json').write_text(json.dumps(face, indent=2), encoding='utf-8')
                with Image.open(io.BytesIO(image_data)) as image:
                    assert image.size == (args.panorama_width, args.panorama_width // 2), image.size
                    if args.transparent_panorama:
                        assert image.convert('RGBA').getchannel('A').getextrema() == (0, 255)
                page.wait_for_function('!document.querySelector(\'[aria-label="Panorama progress"]\')')
                assert abs(canvas.bounding_box()['width'] - box['width']) < 1
                print(f'PASS panorama {args.panorama_width}x{args.panorama_width // 2} export and viewport restoration', flush=True)
                page.get_by_role('button', name='Save project', exact=True).click()
                page.wait_for_function('window.__saved.length>0')
                document = page.evaluate('window.__saved.at(-1)')
                assert len(document['settings']['studioLights']) == 3
                assert document['settings']['dofEnabled']
                print('PASS authored lights and focus saved in studio document', flush=True)
            assert not errors, errors
            print('PASS', 'cinematic' if cinematic else 'render', 'create/edit/select/place/duplicate/delete, WebGL error-free', flush=True)
        except Exception:
            page.screenshot(path=str(args.output / 'failure.png'))
            print('ERRORS', errors, flush=True)
            print(page.locator('body').inner_text()[-10000:], flush=True)
            raise
        finally:
            page.close()
    browser.close()
