"""Verify deterministic WebM plus transparent cinematic PNG with temporal blur."""
from pathlib import Path
from PIL import ImageStat

fixture = Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0], str(Path(__file__).with_name('verify-studio-lights.py')), 'exec'))
args.output = Path('tmp/cinematic-video-motion')
args.output.mkdir(parents=True, exist_ok=True)
init += "\nwindow.go.desktop.App.SaveCinematicFile=async(name,kind,data)=>{window.__movie={name,kind,data};return 'C:/test/motion.'+kind;}"
sequence = dict(version=1, name='Temporal video', duration=0.25, fps=24, keys=[
    dict(id='start', time=0, position=[0.5,0.2,0.8], target=[0,0,0], fov=55, roll=0, exposure=1, ease='linear'),
    dict(id='end', time=0.25, position=[-0.5,0.2,0.8], target=[0,0,0], fov=55, roll=0, exposure=1, ease='linear')],
    motionBlur=dict(enabled=True, angle=360, phase=-180, samples=4, preview=False))

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
        page.locator('.cinema-project input[type=file]').set_input_files(dict(name='video-motion.json', mimeType='application/json', buffer=json.dumps(sequence).encode()))
        page.get_by_role('button', name='WebM', exact=True).click()
        page.get_by_label('Longest edge', exact=True).select_option('720')
        page.get_by_role('button', name='Export …', exact=True).click()
        page.wait_for_function('!!window.__movie', timeout=300000)
        movie = page.evaluate('window.__movie')
        assert movie['kind'] == 'webm'
        (args.output / 'motion.webm').write_bytes(base64.b64decode(movie['data']))
        metadata = page.evaluate('''async()=>{
          const raw=atob(window.__movie.data),data=Uint8Array.from(raw,c=>c.charCodeAt(0));
          const video=document.createElement('video');video.muted=true;video.src=URL.createObjectURL(new Blob([data],{type:'video/webm'}));
          await new Promise((ok,no)=>{video.onloadedmetadata=ok;video.onerror=()=>no(Error('Video decode failed'));});
          const info={width:video.videoWidth,height:video.videoHeight,duration:video.duration};
          // HTMLVideo.drawImage can return black in headless SwiftShader even
          // for valid files. Decode and copy the actual WebCodecs frame instead.
          const {Input,BufferSource,ALL_FORMATS,VideoSampleSink}=await import('/node_modules/mediabunny/dist/modules/src/index.js');
          const input=new Input({source:new BufferSource(data),formats:ALL_FORMATS});
          const sink=new VideoSampleSink(await input.getPrimaryVideoTrack()),sample=await sink.getSample(.125),frame=sample.toVideoFrame();
          const rect=frame.visibleRect,rgba=new Uint8Array(rect.width*rect.height*4);
          await frame.copyTo(rgba,{format:'RGBA',rect});
          info.brightness=Array.from(rgba).filter((_,i)=>i%4!==3).reduce((a,b)=>a+b,0)/(rect.width*rect.height*3);
          const canvas=document.createElement('canvas');canvas.width=rect.width;canvas.height=rect.height;canvas.getContext('2d').putImageData(new ImageData(new Uint8ClampedArray(rgba),rect.width,rect.height),0,0);
          info.image=canvas.toDataURL();frame.close();sample.close();input.dispose();
          URL.revokeObjectURL(video.src);return info;
        }''')
        (args.output / 'webm-decoded.png').write_bytes(base64.b64decode(metadata.pop('image').split(',')[1]))
        assert metadata['width'] == 720, metadata
        assert abs(metadata['duration'] - 0.25) < 0.05, metadata
        assert metadata['brightness'] > 20, metadata
        print('PASS deterministic WebM with preview blur disabled', metadata, flush=True)
        page.get_by_label('Playhead position', exact=True).fill('0.125')
        page.get_by_role('button', name='Export PNG', exact=True).click()
        page.get_by_label('Longest edge', exact=True).select_option('720')
        page.get_by_role('dialog').get_by_label('Transparent background', exact=True).check()
        page.get_by_role('button', name='Export …', exact=True).click()
        page.wait_for_function('!!window.__png', timeout=180000)
        data = base64.b64decode(page.evaluate('window.__png').split(',')[1])
        (args.output / 'transparent-motion.png').write_bytes(data)
        with Image.open(io.BytesIO(data)) as image:
            assert image.width == 720
            alpha = image.convert('RGBA').getchannel('A')
            assert alpha.getextrema() == (0, 255)
            assert sum(alpha.histogram()[1:255]) > 100
            assert sum(ImageStat.Stat(image.convert('RGB')).mean) > 30
        assert not errors, errors
        page.screenshot(path=str(args.output / 'restored.png'))
        print('PASS transparent temporal PNG and viewport restored; no WebGL errors', flush=True)
    except Exception:
        page.screenshot(path=str(args.output / 'failure.png'))
        print('ERRORS', errors, flush=True)
        print(page.locator('body').inner_text()[-12000:].encode('ascii', errors='backslashreplace').decode(), flush=True)
        raise
    finally:
        browser.close()
