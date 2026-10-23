"""Render saved responses from TestCapturedScenePageReplay using headless WebGL.

The page bodies are real Go HTTP responses; only the Wails transport is replayed.
Start the frontend server, then pass --pages path/to/pages.json.
"""
import argparse
import json
from pathlib import Path
from urllib.parse import urlparse, parse_qs
from playwright.sync_api import sync_playwright

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--pages', type=Path, required=True)
parser.add_argument('--url', default='http://127.0.0.1:5191')
parser.add_argument('--output', type=Path, default=Path('tmp/cubecraft-repro/browser'))
parser.add_argument('--centered-controls', action='store_true')
parser.add_argument('--player', help='Optional recorded x,y,z position for a close inspection')
args = parser.parse_args()
pages = json.loads(args.pages.read_text(encoding='utf8'))
args.output.mkdir(parents=True, exist_ok=True)
cursor_index = {'first': 0}
for i, page in enumerate(pages[:-1]):
    cursor_index[','.join(map(str, page['next']))] = i+1
total = sum(len(page['chunks']) for page in pages)
assert total == pages[0]['total']
focus = pages[0]['chunks'][0]
player = [focus['x']*16+8, 80, focus['z']*16+8]
if args.player:
    player = list(map(float, args.player.split(',')))
    assert len(player) == 3
state = dict(revision=1, command='worlds', running=True, stopping=False, connection=5,
             activeWorldId='capture-replay', worlds=[dict(id='capture-replay', name='CubeCraft Replay',
             state='Capturing', player=player, hasPlayer=True, chunks=total)], packs=[], skins=[],
             player=player, hasPlayer=True, loggedIn=False, authenticating=False)
bootstrap = dict(mapWindow=True, version='Capture verification', minecraft='1.26.52', theme='dark', commands=[], state=state)
init = '''window.runtime={EventsOn:()=>()=>{}};window.requested=0;
window.go={desktop:{MapWindowApp:{GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[],
GetScenePage:async(id,r)=>{window.requested++;return fetch('/__capture-page?cursor='+encodeURIComponent(r.after?'done':r.cursor?r.cursor.join(','):'first')).then(r=>r.json())},
ExportScene:async()=>'fixture.glb',OpenResultFolder:async()=>{},EditWorld:async()=>{}}}};
'''.replace('BOOTSTRAP', json.dumps(bootstrap))
with sync_playwright() as p:
    browser = p.chromium.launch(channel='msedge', headless=True,
                                args=['--use-angle=swiftshader', '--enable-unsafe-swiftshader'])
    page = browser.new_page(viewport=dict(width=1440, height=920), device_scale_factor=1)
    errors = []
    page.on('pageerror', lambda error: errors.append(str(error)))
    page.on('console', lambda message: errors.append(message.text) if message.type == 'error' and ('WebGL' in message.text or 'THREE.' in message.text or 'shader' in message.text.lower()) else None)
    page.add_init_script(init)
    def respond(route):
        cursor = parse_qs(urlparse(route.request.url).query)['cursor'][0]
        if cursor == 'done':
            result = {**pages[-1], 'reset': False, 'chunks': [], 'patches': []}
        else:
            result = pages[cursor_index[cursor]]
        route.fulfill(status=200, content_type='application/json', body=json.dumps(result))
    page.route('**/__capture-page?*', respond)
    page.goto(args.url, wait_until='networkidle')
    page.get_by_role('button', name='3D', exact=True).click()
    page.wait_for_function("document.querySelector('.scene-caption')?.textContent.match(/[1-9][0-9.]* Chunks/)", timeout=30000)
    page.wait_for_function("!document.querySelector('.scene-message') && !document.querySelector('.scene-loading-progress') && document.querySelector('.scene-caption')?.textContent.startsWith('"+str(total)+" Chunks')", timeout=60000)
    assert not page.get_by_role('alert').count(), page.locator('body').inner_text()
    assert page.get_by_role('button', name='Export GLB').is_enabled()
    assert page.locator('.scene-viewer').evaluate('e=>getComputedStyle(e).backgroundColor') == 'rgb(24, 36, 44)'
    page.get_by_role('button', name='Overview', exact=True).click()
    if args.centered_controls:
        toggle = page.get_by_role('button', name='Centered orbit', exact=True)
        toggle.click()
        assert toggle.get_attribute('aria-pressed') == 'true'
        assert page.get_by_role('button', name='Player', exact=True).is_disabled()
        assert 'Drehpunkt in der Modellmitte fixiert' in page.locator('.scene-caption').inner_text()
        canvas = page.locator('.scene-canvas canvas')


        capture = lambda: canvas.screenshot(style='.scene-toolbar { visibility: hidden !important } .scene-canvas canvas { outline: none !important }')
        box = canvas.bounding_box()
        x, y = box['x'] + box['width'] * .5, box['y'] + box['height'] * .5
        page.mouse.click(x, y)
        page.wait_for_timeout(700)
        baseline = capture()
        (args.output/'centered-before-pan.png').write_bytes(baseline)
        def drag(button, dx=95, dy=45):
            page.mouse.move(x, y)
            page.mouse.down(button=button)
            page.mouse.move(x+dx, y+dy, steps=10)
            page.mouse.up(button=button)
            page.wait_for_timeout(700)
        drag('right')
        after_pan = capture()
        (args.output/'centered-after-pan.png').write_bytes(after_pan)
        assert after_pan == baseline, 'right mouse translated locked camera'
        canvas.focus()
        for key in ['w', 'a', 's', 'd', 'q', 'e', 'ArrowLeft', 'ArrowUp']:
            page.keyboard.down(key)
            page.wait_for_timeout(80)
            page.keyboard.up(key)
        page.wait_for_timeout(200)
        assert capture() == baseline, 'keyboard translated locked camera'
        page.keyboard.down('Control')
        drag('left')
        page.keyboard.up('Control')
        assert capture() == baseline, 'modified drag translated locked camera'
        drag('left')
        rotated = capture()
        assert rotated != baseline, 'rotation was disabled'
        page.wait_for_timeout(1000)
        before_zoom = capture()
        page.mouse.move(x, y)
        page.mouse.wheel(0, -180)
        page.wait_for_timeout(700)
        assert capture() != before_zoom, 'zoom was disabled'
        toggle.click()
        assert toggle.get_attribute('aria-pressed') == 'false'
        assert page.get_by_role('button', name='Player', exact=True).is_enabled()
        page.wait_for_timeout(1000)
        unlocked = capture()
        drag('right')
        assert capture() != unlocked, 'pan did not resume after unlocking'
        toggle.click()
        page.set_viewport_size(dict(width=680, height=980))
        page.wait_for_timeout(700)
        assert toggle.get_attribute('aria-pressed') == 'true'
        page.screenshot(path=str(args.output/'centered-portrait.png'), full_page=True)
        page.set_viewport_size(dict(width=1440, height=920))
        page.wait_for_timeout(700)
        page.get_by_role('button', name='2D', exact=True).click()
        page.get_by_role('button', name='3D', exact=True).click()
        toggle.wait_for(state='visible')
        assert toggle.get_attribute('aria-pressed') == 'true', 'centered preference lost on viewer remount'
        page.wait_for_function("document.querySelector('.scene-caption')?.textContent.startsWith('"+str(total)+" Chunks')", timeout=30000)
        print('PASS centered controls: right/modified drag and keyboard locked; orbit and zoom work; free pan restored; portrait fit; preference survives 2D/3D remount')
    page.wait_for_timeout(250)
    page.screenshot(path=str(args.output/'cubecraft-275-chunks.png'), full_page=True)
    if args.player:
        page.get_by_role('button', name='Player', exact=True).click()
        page.wait_for_timeout(500)
        page.screenshot(path=str(args.output/'recorded-player-view.png'), full_page=True)
    assert not errors, errors
    print(json.dumps(dict(chunks=total, pages=len(pages), browserErrors=errors,
                          caption=page.locator('.scene-caption').inner_text(), screenshot=str(args.output/'cubecraft-275-chunks.png')), indent=2))
    browser.close()
