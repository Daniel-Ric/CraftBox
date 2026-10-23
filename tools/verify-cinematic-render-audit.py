"""Compare real GPU pixels at shot cuts and temporal map compositing with authored clouds."""
from pathlib import Path
fixture = Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0], str(Path(__file__).with_name('verify-studio-lights.py')), 'exec'))
args.output = Path('tmp/cinematic-render-audit')
args.output.mkdir(parents=True, exist_ok=True)
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
        result = page.evaluate('''async()=>{
          const host=document.querySelector('.render-canvas');let fiber=host[Object.keys(host).find(k=>k.startsWith('__reactFiber'))],engine;
          while(fiber&&!engine){let hook=fiber.memoizedState;while(hook){if(hook.memoizedState?.current?.capturePanorama)engine=hook.memoizedState.current;hook=hook.next;}fiber=fiber.return;}
          if(!engine)throw Error('Cannot inspect render engine');
          const {sampleSequence}=await import('/src/cinematic.ts'),{temporalSamples}=await import('/src/cinematic-motion.ts'),{mapDefaults}=await import('/src/map-presentation.ts');
          const {createCloudLayer,keyCloudValue}=await import('/src/cloud-layers.ts');
          let cloud={...createCloudLayer('volume'),position:[-.8,-.3,0],size:[1.5,.8,1.2],windSpeed:0};
          cloud=keyCloudValue(cloud,'dissolve',0,0,24);cloud=keyCloudValue(cloud,'dissolve',.125,.65,24);
          cloud.tracks[0].keys.forEach(k=>k.interpolation='hold');
          engine.options={...engine.options,atmosphere:{...engine.options.atmosphere,clouds:false,fog:false,animate:false,cloudLayers:[cloud],cloudPreviewQuality:'low',cloudExportQuality:'medium'}};
          const key=(id,position)=>({id,time:0,position,target:[0,0,0],fov:55,roll:0,exposure:1,ease:'linear'});
          const a=key('ka',[.6,.25,1]),b=key('kb',[-.6,.25,1]);
          const sequence={version:2,name:'Cut audit',duration:.25,fps:24,keys:[a],tracks:[],
            cameras:[{id:'a',name:'A',duration:.25,keys:[a],tracks:[]},{id:'b',name:'B',duration:.25,keys:[b],tracks:[]}],
            shots:[{id:'sa',name:'A',cameraId:'a',start:0,end:.125,sourceIn:0,rate:1},{id:'sb',name:'B',cameraId:'b',start:.125,end:.25,sourceIn:0,rate:1}],
            motionBlur:{enabled:true,angle:720,phase:-360,samples:8,preview:false}};
          const images={};
          const render=async(name,pose,time,samples)=>{const canvas=await engine.renderFrame(pose,320,180,time,samples);const copy=document.createElement('canvas');copy.width=320;copy.height=180;const ctx=copy.getContext('2d');ctx.drawImage(canvas,0,0);const pixels=ctx.getImageData(0,0,320,180).data;images[name]=copy.toDataURL();return pixels;};
          const difference=(a,b)=>{let sum=0;for(let i=0;i<a.length;i++)if(i%4!==3)sum+=Math.abs(a[i]-b[i]);return sum/(320*180*3);};
          const cut=.125,pose=sampleSequence(sequence,cut),samples=temporalSamples(sequence,cut,24).map(s=>({...s,pose:sampleSequence(sequence,s.time)}));
          const sharpB=await render('cut-sharp',pose,cut),blurB=await render('cut-temporal',pose,cut,samples),sharpA=await render('before-cut',sampleSequence(sequence,cut-.001),cut-.001);
          const oldMap=engine.callbacks.current.mapView;
          engine.callbacks.current.mapView={...mapDefaults,enabled:true,grid:true,gridOpacity:.9,gridSize:16,contours:true,contourStep:16,heightTint:.5};
          const mapSharp=await render('map-sharp',pose,cut),mapTemporal=await render('map-temporal',pose,cut,samples);
          engine.callbacks.current.mapView=oldMap;engine.finishExport();
          return {images,metrics:{cutDifference:difference(sharpB,blurB),cameraDifference:difference(sharpA,sharpB),overlayDifference:difference(sharpB,mapSharp),mapParity:difference(mapSharp,mapTemporal),sampleCount:samples.length,firstSample:Math.min(...samples.map(s=>s.time))}};
        }''')
        for name, data in result['images'].items():
            (args.output / (name + '.png')).write_bytes(base64.b64decode(data.split(',')[1]))
        metrics = result['metrics']
        (args.output / 'metrics.json').write_text(json.dumps(metrics, indent=2), encoding='utf-8')
        assert metrics['sampleCount'] == 8 and metrics['firstSample'] >= .125, metrics
        assert metrics['cameraDifference'] > 5 and metrics['cutDifference'] < 3, metrics
        assert metrics['overlayDifference'] > .3 and metrics['mapParity'] < 3, metrics
        assert not errors, errors
        print('PASS shot-cut image isolation and temporal map overlay parity', metrics, flush=True)
    except Exception:
        page.screenshot(path=str(args.output / 'failure.png'))
        print('ERRORS', errors, flush=True)
        raise
    finally:
        browser.close()
