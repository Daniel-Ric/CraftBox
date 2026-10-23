"""Verify shared lights in actual motion-blurred media and mid-frame cancellation."""
from pathlib import Path
from PIL import ImageChops, ImageStat
fixture = Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0], str(Path(__file__).with_name('verify-studio-lights.py')), 'exec'))
args.output = Path('tmp/cinematic-light-exports')
args.output.mkdir(parents=True, exist_ok=True)
init += "\nwindow.__movies=[];window.go.desktop.App.SaveCinematicFile=async(name,kind,data)=>{window.__movies.push({name,kind,data});return 'C:/test/lights.'+kind;}"
sequence = dict(version=1, name='Light export', duration=.25, fps=24, keys=[
    dict(id='start', time=0, position=[.6,.25,1], target=[0,0,0], fov=55, roll=0, exposure=1, ease='linear'),
    dict(id='end', time=.25, position=[.5,.25,1], target=[0,0,0], fov=55, roll=0, exposure=1, ease='linear')],
    motionBlur=dict(enabled=True, angle=180, phase=-90, samples=4, preview=False))
with sync_playwright() as pw:
    browser = pw.chromium.launch(channel='msedge', headless=True, args=['--use-angle=swiftshader', '--enable-unsafe-swiftshader'])
    page = browser.new_page(viewport=dict(width=1520, height=1080))
    page.set_default_timeout(60000)
    errors = []
    page.on('pageerror', lambda error: errors.append(str(error)))
    page.add_init_script(init)
    page.route('**/__page?*', lambda route: route.fulfill(status=200, content_type='application/json', body=json.dumps(pages[indices[parse_qs(urlparse(route.request.url).query)['cursor'][0]]])))
    try:
        page.goto(args.url, wait_until='networkidle')
        page.get_by_role('button', name='Open workspace', exact=True).click()
        page.get_by_role('button', name='Cinematic editor', exact=True).click()
        page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay')")
        page.locator('.cinema-project input[type=file]').set_input_files(dict(name='lights.json', mimeType='application/json', buffer=json.dumps(sequence).encode()))
        page.get_by_role('button', name='Edit 3D path', exact=True).click()
        page.get_by_role('button', name='Edit lighting', exact=True).click()
        page.get_by_role('button', name='Add point light', exact=True).click()
        page.get_by_label('Light output (lumens)', exact=True).fill('1000000')
        page.get_by_label('Light position Y', exact=True).fill('0.5')
        def export_gif(name):
            count = page.evaluate('window.__movies.length')
            page.get_by_role('button', name='GIF', exact=True).click()
            page.get_by_label('Longest edge', exact=True).select_option('320')
            page.get_by_role('button', name='Export …', exact=True).click()
            page.wait_for_function('(count)=>window.__movies.length>count', arg=count, timeout=300000)
            movie = page.evaluate('window.__movies.at(-1)')
            raw = base64.b64decode(movie['data'])
            (args.output / (name + '.gif')).write_bytes(raw)
            image = Image.open(io.BytesIO(raw)); image.seek(1)
            return image.convert('RGB').copy()
        lit = export_gif('lit-motion')
        page.get_by_label('Point light aktiv', exact=True).uncheck()
        unlit = export_gif('unlit-motion')
        difference = sum(ImageStat.Stat(ImageChops.difference(lit, unlit)).mean) / 3
        assert difference > .1, difference
        print('PASS cinematic motion export contains placeable lights', difference, flush=True)
        page.get_by_label('Point light aktiv', exact=True).check()


        sequence['motionBlur']['samples'] = 64
        page.locator('.cinema-project input[type=file]').set_input_files(dict(name='cancel.json', mimeType='application/json', buffer=json.dumps(sequence).encode()))
        before = page.evaluate('window.__movies.length')
        page.evaluate('''()=>{const host=document.querySelector('.render-canvas');let fiber=host[Object.keys(host).find(k=>k.startsWith('__reactFiber'))],engine;while(fiber&&!engine){let hook=fiber.memoizedState;while(hook){if(hook.memoizedState?.current?.capturePanorama)engine=hook.memoizedState.current;hook=hook.next;}fiber=fiber.return;}if(!engine)throw Error('Cannot inspect temporal renderer');const draw=engine.draw;window.__rawSamples=0;engine.draw=function(...args){if(this.exporting&&args[2])window.__rawSamples++;return draw.apply(this,args);};}''')
        page.get_by_role('button', name='WebM', exact=True).click()
        page.get_by_label('Longest edge', exact=True).select_option('720')
        page.get_by_role('button', name='Export …', exact=True).click()
        page.wait_for_function('window.__rawSamples>0')
        page.locator('.cinema-record button').click()
        page.wait_for_function("document.body.innerText.includes('Export canceled')")
        assert page.evaluate('window.__movies.length') == before
        samples = page.evaluate('window.__rawSamples')
        assert 0 < samples < 64, samples
        assert page.get_by_role('button', name='WebM', exact=True).is_enabled()
        assert not errors, errors
        page.screenshot(path=str(args.output / 'cancel-restored.png'))
        print('PASS 64-sample video cancellation restores editor without saving a movie; samples rendered:', samples, flush=True)
    except Exception:
        page.screenshot(path=str(args.output / 'failure.png'))
        print('ERRORS', errors, flush=True)
        print(page.locator('body').inner_text()[-12000:].encode('ascii', errors='backslashreplace').decode(), flush=True)
        raise
    finally:
        browser.close()
