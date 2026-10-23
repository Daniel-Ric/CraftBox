"""Verify independent animation, curve gestures and focus in the real WebGL editor.

Uses the same recorded-world bridge fixture as verify-studio-lights.py.
"""
from pathlib import Path

fixture = Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0], str(Path(__file__).with_name('verify-studio-lights.py')), 'exec'))
args.output = Path('tmp/cinematic-tracks')
args.output.mkdir(parents=True, exist_ok=True)

with sync_playwright() as pw:
    browser = pw.chromium.launch(channel='msedge', headless=True,
        args=['--use-angle=swiftshader', '--enable-unsafe-swiftshader'])
    page = browser.new_page(viewport=dict(width=1520, height=1080))
    page.set_default_timeout(60000)
    errors = []
    page.on('pageerror', lambda error: errors.append(str(error)))
    page.on('console', lambda msg: errors.append(msg.text) if msg.type == 'error' and any(t in msg.text for t in ('WebGL', 'THREE', 'shader')) else None)
    page.add_init_script(init)
    page.add_init_script("window.go.desktop.App.SaveCinematicFile=async(name,kind,data)=>{window.__movie={name,kind,data};return 'C:/test/shots.'+kind;}")
    page.route('**/__page?*', lambda route: route.fulfill(status=200, content_type='application/json', body=json.dumps(pages[indices[parse_qs(urlparse(route.request.url).query)['cursor'][0]]])))
    try:
        page.goto(args.url, wait_until='networkidle')
        page.get_by_role('button', name='Open workspace', exact=True).click()
        page.get_by_role('button', name='Cinematic editor', exact=True).click()
        page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay')")
        page.get_by_role('separator', name='Timeline height', exact=True).press('End')
        page.get_by_label('Active animation track', exact=True).select_option('fov')
        page.get_by_label('Playhead position', exact=True).fill('2')
        page.get_by_role('button', name='Add track keyframe', exact=True).click()
        page.get_by_label('Track value', exact=True).fill('75')
        page.get_by_role('button', name='Curve editor', exact=True).click()
        page.get_by_label('Outgoing Tangent Value', exact=True).fill('8')
        handle = page.get_by_role('button', name='Outgoing Drag tangent', exact=True)
        box = handle.bounding_box()
        page.mouse.move(box['x'] + box['width']/2, box['y'] + box['height']/2)
        page.mouse.down()
        page.mouse.move(box['x'] + 45, box['y'] - 12, steps=8)
        page.mouse.up()
        assert float(page.get_by_label('Outgoing Tangent Value', exact=True).input_value()) != 8
        page.screenshot(path=str(args.output / 'value-curve.png'))
        page.get_by_label('Curve view', exact=True).select_option('speed')
        page.get_by_label('Speed curve Field of view', exact=True).wait_for()
        page.screenshot(path=str(args.output / 'speed-curve.png'))
        page.get_by_role('button', name='Tracks', exact=True).click()
        page.get_by_label('Active animation track', exact=True).select_option('position.x')
        page.get_by_label('Track value', exact=True).fill('8')
        page.get_by_label('position Y', exact=True).fill('3')
        assert float(page.get_by_label('Track value', exact=True).input_value()) == 8
        page.get_by_label('Active animation track', exact=True).select_option('focusDistance')
        page.get_by_label('Playhead position', exact=True).fill('0')
        page.get_by_label('Camera focus distance (blocks)', exact=True).fill('120')
        page.get_by_label('Playhead position', exact=True).fill('6')
        page.get_by_label('Camera focus distance (blocks)', exact=True).fill('500')
        page.get_by_label('Playhead position', exact=True).fill('3')
        focus = float(page.get_by_label('Camera focus distance (blocks)', exact=True).input_value())
        assert 120 < focus < 500, focus
        page.get_by_role('button', name='Save project', exact=True).click()
        page.wait_for_function('window.__saved.length>0')
        document = page.evaluate('window.__saved.at(-1)')
        (args.output / 'saved.json').write_text(json.dumps(document, indent=2), encoding='utf-8')
        page.screenshot(path=str(args.output / 'focus-tracks.png'))
        page.get_by_role('tab', name='Shots', exact=True).click()
        page.get_by_role('button', name='Create cameras and shots', exact=True).click()
        page.get_by_role('button', name='Duplicate camera', exact=True).click()
        second = page.get_by_label('Edit camera', exact=True).input_value()
        page.get_by_label('Active animation track', exact=True).select_option('fov')
        for index in range(page.get_by_label('Selected track keyframe', exact=True).locator('option').count()):
            page.get_by_label('Selected track keyframe', exact=True).select_option(index=index)
            page.get_by_label('Track value', exact=True).fill('100')
        page.get_by_label('Playhead position', exact=True).fill('6')
        page.get_by_role('button', name='Split shot at playhead', exact=True).click()
        page.get_by_label('Shot camera', exact=True).select_option(second)
        page.get_by_label('Source in', exact=True).fill('0')
        page.get_by_role('button', name='Show shot sequence', exact=True).click()
        page.wait_for_function("document.querySelector('.cinema-monitor-bar').textContent.includes('Program · Camera 2')")
        page.screenshot(path=str(args.output / 'camera-cut.png'))
        page.get_by_role('button', name='Save project', exact=True).click()
        page.wait_for_function('window.__saved.length>1')
        project = page.evaluate("JSON.parse(localStorage.getItem('craftbox-cinematic:render-fixture'))")
        assert len(project['cameras']) == 2 and len(project['shots']) == 2
        assert project['cameras'][0]['tracks'][6]['keys'][0]['value'] != 100
        (args.output / 'shots.json').write_text(json.dumps(project, indent=2), encoding='utf-8')

        def shrink_animation(animation):
            animation['duration'] /= 24
            for key in animation['keys']:
                key['time'] /= 24
            for track in animation['tracks']:
                for key in track['keys']:
                    key['time'] /= 24
                    key['incoming'][0] /= 24
                    key['outgoing'][0] /= 24
        shrink_animation(project)
        for camera in project['cameras']:
            shrink_animation(camera)
            camera['lens']['enabled'] = False
        for shot in project['shots']:
            for field in ('start', 'end', 'sourceIn'):
                shot[field] /= 24
        page.locator('.cinema-project input[type=file]').set_input_files(dict(name='shot-export.json', mimeType='application/json', buffer=json.dumps(project).encode()))
        page.get_by_role('tab', name='Export', exact=True).click()
        page.get_by_label('GIF size', exact=True).select_option('320')
        page.get_by_role('button', name='GIF', exact=True).click()
        page.get_by_role('button', name='Export …', exact=True).click()
        page.wait_for_function('!!window.__movie', timeout=180000)
        movie = page.evaluate('window.__movie')
        assert movie['kind'] == 'gif'
        data = base64.b64decode(movie['data'])
        (args.output / 'shots.gif').write_bytes(data)
        with Image.open(io.BytesIO(data)) as image:
            assert image.n_frames == 6, image.n_frames
            image.seek(0)
            first = image.convert('RGB').copy()
            image.seek(3)
            assert image.convert('RGB').tobytes() != first.tobytes()
        assert not errors, errors
        print('PASS independent axes, tangent drag, velocity graph, focus pull, independent cameras, cut, project save, actual 6-frame GIF; no WebGL errors', flush=True)
    except Exception:
        page.screenshot(path=str(args.output / 'failure.png'))
        print('ERRORS', errors, flush=True)
        print(page.locator('body').inner_text()[-12000:].encode('ascii', errors='backslashreplace').decode(), flush=True)
        raise
    finally:
        browser.close()
