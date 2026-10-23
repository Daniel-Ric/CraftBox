"""Verify native audio playback, persistent edits and actual encoded audio/video."""
from pathlib import Path
import math
import struct
import wave

fixture = Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0], str(Path(__file__).with_name('verify-studio-lights.py')), 'exec'))
args.output = Path('tmp/cinematic-audio')
args.output.mkdir(parents=True, exist_ok=True)
sound = io.BytesIO()
with wave.open(sound, 'wb') as wav:
    wav.setparams((1, 2, 48000, 0, 'NONE', 'not compressed'))
    wav.writeframes(b''.join(struct.pack('<h', round(.4 * 32767 * math.sin(2 * math.pi * (440 if i < 24000 else 880) * i / 48000))) for i in range(48000)))
init += '''
window.__movies=[];window.__audioStarts=[];
window.go.desktop.App.SaveCinematicFile=async(name,kind,data)=>{window.__movies.push({name,kind,data});return 'C:/test/audio.'+kind;};
const NativeAudioContext=window.AudioContext;
window.AudioContext=class extends NativeAudioContext {
  createBufferSource(){const source=super.createBufferSource(),start=source.start.bind(source),ctx=this;
    source.start=(when,offset,...rest)=>{window.__audioStarts.push({when,offset,rate:source.playbackRate.value,current:ctx.currentTime});return start(when,offset,...rest);};return source;}
};'''
sequence = dict(version=1, name='Audio verification', duration=.5, fps=24, keys=[
    dict(id='start', time=0, position=[.6,.25,1], target=[0,0,0], fov=55, roll=0, exposure=1, ease='linear'),
    dict(id='end', time=.5, position=[.5,.25,1], target=[0,0,0], fov=55, roll=0, exposure=1, ease='linear')])

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
        page.locator('.cinema-project input[type=file]').set_input_files(dict(name='audio.json', mimeType='application/json', buffer=json.dumps(sequence).encode()))
        page.get_by_role('tab', name='Audio', exact=True).click()
        page.get_by_label('Import audio file', exact=True).set_input_files(dict(name='two-tones.wav', mimeType='audio/wav', buffer=sound.getvalue()))
        page.get_by_label('Audio source in', exact=True).wait_for()
        for label, value in [('Clip position', '.125'), ('Audio source in', '.25'), ('Audio source out', '.75'), ('Clip level', '-6'), ('Audio track 1 Level', '-3'), ('Clip pan', '-1'), ('Fade in', '.05'), ('Fade out', '.05')]:
            page.get_by_label(label, exact=True).fill(value)
        page.get_by_role('button', name='Open audio timeline', exact=True).click()
        assert page.locator('.cinematic-Audio timeline svg').count() > 0
        page.get_by_role('button', name='Play', exact=True).click()
        page.wait_for_function('window.__audioStarts.length>0')
        page.get_by_role('button', name='Pause', exact=True).click()
        starts = page.evaluate('window.__audioStarts')
        assert abs(starts[0]['offset'] - .25) < .001 and starts[0]['rate'] == 1, starts
        assert starts[0]['when'] >= starts[0]['current'], starts
        print('PASS native playback scheduling', starts[:2], flush=True)
        page.get_by_label('Playhead position', exact=True).fill('0.25')
        page.get_by_role('button', name='Add sequence marker', exact=True).click()
        page.get_by_label('Marker name', exact=True).fill('Cue')
        page.get_by_label('Marker comment', exact=True).fill('Beat')
        page.get_by_role('button', name='Save project', exact=True).click()
        page.wait_for_function('window.__saved.length>0')
        saved = page.evaluate('window.__saved.at(-1)')
        saved_sequence = saved['cinematic']['sequence']
        assert saved_sequence['markers'][0]['name'] == 'Cue' and saved_sequence['markers'][0]['comment'] == 'Beat'
        assert saved_sequence['audio']['clips'][0]['in'] == .25
        assert len(saved_sequence['audio']['assets'][0]['data']) > 100000
        (args.output / 'saved.json').write_text(json.dumps(saved), encoding='utf-8')
        print('PASS native project saved with media and marker', flush=True)

        page.wait_for_function("localStorage.getItem('craftbox-cinematic:render-fixture')?.includes('craftbox-cinematic-draft')")
        page.reload(wait_until='networkidle')
        welcome = page.get_by_role('button', name='Open workspace', exact=True)
        if welcome.is_visible(): welcome.click()
        page.get_by_role('button', name='Cinematic editor', exact=True).click()
        page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay') && !document.querySelector('.cinematic-draft-loading')")
        page.get_by_role('tab', name='Audio', exact=True).click()
        assert page.get_by_label('Clip level', exact=True).input_value() == '-6'
        assert page.get_by_label('Clip position', exact=True).input_value() == '0.125'
        print('PASS IndexedDB audio draft restored', flush=True)
        for kind in ['webm', 'mp4']:
            page.get_by_role('button', name=kind.upper() if kind == 'mp4' else 'WebM', exact=True).click()
            page.get_by_label('Longest edge', exact=True).select_option('720')
            page.get_by_role('button', name='Export …', exact=True).click()
            page.wait_for_function('(kind)=>window.__movies.some(m=>m.kind===kind)', arg=kind, timeout=300000)
            movie = page.evaluate('(kind)=>window.__movies.find(m=>m.kind===kind)', kind)
            (args.output / ('mix.' + kind)).write_bytes(base64.b64decode(movie['data']))
            metrics = page.evaluate('''async(kind)=>{
              const movie=window.__movies.find(m=>m.kind===kind),data=Uint8Array.from(atob(movie.data),c=>c.charCodeAt(0));
              const {Input,BufferSource,ALL_FORMATS,AudioSampleSink,VideoSampleSink}=await import('/node_modules/mediabunny/dist/modules/src/index.js');
              const input=new Input({source:new BufferSource(data),formats:ALL_FORMATS}),track=await input.getPrimaryAudioTrack();
              if(!track)throw Error('Export has no audio');
              const pcm=[new Float32Array(30000),new Float32Array(30000)];
              for await(const sample of new AudioSampleSink(track).samples()){
                for(let ch=0;ch<2;ch++){const block=new Float32Array(sample.numberOfFrames);sample.copyTo(block,{planeIndex:ch,format:'f32-planar'});const offset=Math.round(sample.timestamp*48000);for(let i=0;i<block.length;i++)if(offset+i>=0&&offset+i<pcm[ch].length)pcm[ch][offset+i]=block[i];}sample.close();
              }
              const rms=(ch,a,b)=>Math.sqrt(pcm[ch].slice(Math.round(a*48000),Math.round(b*48000)).reduce((s,v)=>s+v*v,0)/Math.round((b-a)*48000));
              const power=(freq,a,b)=>{let re=0,im=0;for(let i=Math.round(a*48000);i<Math.round(b*48000);i++){re+=pcm[0][i]*Math.cos(2*Math.PI*freq*i/48000);im+=pcm[0][i]*Math.sin(2*Math.PI*freq*i/48000);}return re*re+im*im;};
              const sample=await new VideoSampleSink(await input.getPrimaryVideoTrack()).getSample(.25),frame=sample.toVideoFrame(),rect=frame.visibleRect,rgba=new Uint8Array(rect.width*rect.height*4);await frame.copyTo(rgba,{format:'RGBA',rect});
              const result={codec:await track.getCodec(),duration:await input.computeDuration(),width:rect.width,height:rect.height,leading:rms(0,.02,.07),left:rms(0,.22,.32),right:rms(1,.22,.32),firstTone:power(440,.22,.32)/Math.max(1e-10,power(880,.22,.32)),secondTone:power(880,.40,.48)/Math.max(1e-10,power(440,.40,.48)),brightness:rgba.reduce((s,v,i)=>s+(i%4===3?0:v),0)/(rect.width*rect.height*3)};
              frame.close();sample.close();input.dispose();return result;
            }''', kind)
            assert metrics['codec'] == ('opus' if kind == 'webm' else 'aac'), metrics
            assert abs(metrics['duration'] - .5) < .06 and metrics['width'] == 720, metrics
            assert metrics['leading'] < .002 and .07 < metrics['left'] < .13 and metrics['right'] < .005, metrics
            assert metrics['firstTone'] > 20 and metrics['secondTone'] > 20 and metrics['brightness'] > 20, metrics
            (args.output / (kind + '-metrics.json')).write_text(json.dumps(metrics, indent=2), encoding='utf-8')
            print('PASS decoded ' + kind, metrics, flush=True)
        assert not errors, errors
        page.screenshot(path=str(args.output / 'verified.png'))
    except Exception:
        page.screenshot(path=str(args.output / 'failure.png'))
        print('ERRORS', errors, flush=True)
        print(page.locator('body').inner_text()[-12000:].encode('ascii', errors='backslashreplace').decode(), flush=True)
        raise
    finally:
        browser.close()
