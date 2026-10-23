"""Compare real GIF outputs with and without HDR temporal integration."""
from pathlib import Path
from PIL import ImageChops, ImageFilter, ImageStat

fixture = Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0], str(Path(__file__).with_name('verify-studio-lights.py')), 'exec'))
args.output = Path('tmp/cinematic-motion')
args.output.mkdir(parents=True, exist_ok=True)
init += "\nwindow.__movies=[];window.go.desktop.App.SaveCinematicFile=async(name,kind,data)=>{window.__movies.push({name,kind,data});return 'C:/test/motion.'+kind;}"
sequence = dict(version=1, name='Shutter comparison', duration=0.5, fps=24, keys=[
    dict(id='start', time=0, position=[0.5,0.2,0.8], target=[0,0,0], fov=55, roll=0, exposure=1, ease='linear'),
    dict(id='end', time=0.5, position=[-0.5,0.2,0.8], target=[0,0,0], fov=55, roll=0, exposure=1, ease='linear')])

with sync_playwright() as pw:
    browser = pw.chromium.launch(channel='msedge', headless=True, args=['--use-angle=swiftshader', '--enable-unsafe-swiftshader'])
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
        page.locator('.cinema-project input[type=file]').set_input_files(dict(name='motion.json', mimeType='application/json', buffer=json.dumps(sequence).encode()))
        page.get_by_role('tab', name='Export', exact=True).click()
        page.get_by_label('GIF size', exact=True).select_option('320')
        for index, enabled in enumerate((False, True)):
            page.get_by_role('tab', name='Motion', exact=True).click()
            page.get_by_label('Enable motion blur', exact=True).set_checked(enabled)
            if enabled:
                page.get_by_label('Shutter angle', exact=True).fill('360')
                page.get_by_label('Shutter phase', exact=True).fill('-180')
                page.get_by_label('Temporal samples per frame', exact=True).fill('8')
            page.get_by_role('button', name='GIF', exact=True).click()
            page.get_by_role('button', name='Export …', exact=True).click()
            page.wait_for_function(f'window.__movies.length>{index}', timeout=300000)
            movie = page.evaluate(f'window.__movies[{index}]')
            data = base64.b64decode(movie['data'])
            name = 'blur' if enabled else 'sharp'
            (args.output / f'{name}.gif').write_bytes(data)
            with Image.open(io.BytesIO(data)) as image:
                assert image.n_frames == 6, image.n_frames
                image.seek(3)
                image.convert('RGB').save(args.output / f'{name}.png')
            print('PASS', name, '6-frame actual GIF export', flush=True)
        with Image.open(args.output / 'sharp.png') as sharp, Image.open(args.output / 'blur.png') as blur:
            sharp_brightness = sum(ImageStat.Stat(sharp).mean) / 3
            blur_brightness = sum(ImageStat.Stat(blur).mean) / 3
            assert 0.7 < blur_brightness / sharp_brightness < 1.3, (sharp_brightness, blur_brightness)
            difference = sum(ImageStat.Stat(ImageChops.difference(sharp, blur)).mean) / 3
            assert difference > 0.15, difference
            sharp_edges = sum(ImageStat.Stat(sharp.filter(ImageFilter.FIND_EDGES)).mean)
            blur_edges = sum(ImageStat.Stat(blur.filter(ImageFilter.FIND_EDGES)).mean)
            print('DIFFERENCE', difference, 'EDGE_ENERGY', sharp_edges, blur_edges, flush=True)
            assert sharp_edges * 0.2 < blur_edges < sharp_edges, (sharp_edges, blur_edges)
        page.get_by_label('Motion blur preview', exact=True).check()
        page.get_by_label('Playhead position', exact=True).fill('0.25')
        page.screenshot(path=str(args.output / 'preview.png'))
        page.get_by_role('button', name='Save project', exact=True).click()
        page.wait_for_function('window.__saved.length>0')
        document = page.evaluate('window.__saved.at(-1)')
        assert document['cinematic']['sequence']['motionBlur']['samples'] == 8
        (args.output / 'saved.json').write_text(json.dumps(document, indent=2), encoding='utf-8')
        assert not errors, errors
        print('PASS HDR shutter output differs and smooths edges; preview/save; no WebGL errors', flush=True)
    except Exception:
        page.screenshot(path=str(args.output / 'failure.png'))
        print('ERRORS', errors, flush=True)
        print(page.locator('body').inner_text()[-12000:].encode('ascii', errors='backslashreplace').decode(), flush=True)
        raise
    finally:
        browser.close()
