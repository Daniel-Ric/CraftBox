"""Real WebGL checks for cinematic focus picking and animated target constraints."""
from pathlib import Path

fixture = Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0], str(Path(__file__).with_name('verify-studio-lights.py')), 'exec'))
args.output = Path('tmp/cinematic-tracking')
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
    page.route('**/__page?*', lambda route: route.fulfill(status=200, content_type='application/json', body=json.dumps(pages[indices[parse_qs(urlparse(route.request.url).query)['cursor'][0]]])))
    try:
        page.goto(args.url, wait_until='networkidle')
        page.get_by_role('button', name='Open workspace', exact=True).click()
        page.get_by_role('button', name='Cinematic editor', exact=True).click()
        page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay')")
        page.get_by_label('Playhead position', exact=True).fill('6')
        page.get_by_role('button', name='Pick camera focus', exact=True).click()
        canvas = page.locator('.render-canvas canvas')
        box = canvas.bounding_box()
        page.mouse.click(box['x'] + box['width'] * 0.5, box['y'] + box['height'] * 0.6)
        page.get_by_role('button', name='Pick camera focus', exact=True).wait_for()
        assert page.get_by_label('Camera depth of field', exact=True).is_checked()
        focus = float(page.get_by_label('Camera focus distance (blocks)', exact=True).input_value())
        assert focus > 0
        print('PASS cinematic click focus authors a lens key', flush=True)
        page.get_by_role('tab', name='Target', exact=True).click()
        page.get_by_role('button', name='Add tracking target', exact=True).click()
        page.get_by_label('Target name', exact=True).fill('Turmspitze')
        page.get_by_role('button', name='Click to place target', exact=True).click()
        page.mouse.click(box['x'] + box['width'] * 0.5, box['y'] + box['height'] * 0.6)
        page.get_by_role('button', name='Click to place target', exact=True).wait_for()
        page.get_by_label('Playhead position', exact=True).fill('0')
        page.get_by_label('Target position X', exact=True).fill('-0.3')
        page.get_by_label('Playhead position', exact=True).fill('12')
        page.get_by_label('Target position X', exact=True).fill('0.3')
        page.get_by_label('Enable target tracking', exact=True).check()
        page.get_by_label('Focus follows target', exact=True).check()
        page.get_by_label('Playhead position', exact=True).fill('3')
        page.get_by_role('button', name='Edit target tracks in the curve editor', exact=True).click()
        assert page.get_by_label('Active animation track', exact=True).locator('option').count() == 3
        page.get_by_role('button', name='Curve editor', exact=True).click()
        page.get_by_label('Value curve Position X', exact=True).wait_for()
        page.screenshot(path=str(args.output / 'target-curves.png'))
        page.get_by_role('button', name='Edit 3D path', exact=True).click()
        page.screenshot(path=str(args.output / 'target-guides.png'))
        page.get_by_role('button', name='Save project', exact=True).click()
        page.wait_for_function('window.__saved.length>0')
        document = page.evaluate('window.__saved.at(-1)')
        sequence = document['cinematic']['sequence']
        assert sequence['tracking']['focus'] and sequence['tracking']['enabled']
        assert sequence['targets'][0]['name'] == 'Turmspitze'
        assert len(sequence['targets'][0]['tracks']) == 3
        assert any(t['channel'] == 'focusDistance' and len(t['keys']) == 1 for t in sequence['tracks'])
        (args.output / 'saved.json').write_text(json.dumps(document, indent=2), encoding='utf-8')
        assert not errors, errors
        print('PASS target surface placement, independent target curves, camera constraint/focus and saved project; no WebGL errors', flush=True)
    except Exception:
        page.screenshot(path=str(args.output / 'failure.png'))
        print('ERRORS', errors, flush=True)
        print(page.locator('body').inner_text()[-12000:].encode('ascii', errors='backslashreplace').decode(), flush=True)
        raise
    finally:
        browser.close()
