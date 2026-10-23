"""Render the three reported island looks with fog in the actual terrain pipeline."""
from pathlib import Path
fixture=Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0],__file__,'exec'))
out=Path('tmp/cloud-island-look-fix');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as pw:
 browser=pw.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=browser.new_page(viewport=dict(width=1280,height=800));errors=[]
 page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and any(t in m.text.lower() for t in ['shader','webgl','three']) else None)
 page.add_init_script(init)
 page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
 page.goto(args.url,wait_until='networkidle');page.get_by_role('button',name='Open workspace',exact=True).click();page.get_by_role('button',name='Render editor',exact=True).click()
 page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay')",timeout=120000)
 images=page.evaluate('''async()=>{
 const host=document.querySelector('.render-canvas');let fiber=host[Object.keys(host).find(k=>k.startsWith('__reactFiber'))],engine;
 while(fiber&&!engine){let h=fiber.memoizedState;while(h){if(h.memoizedState?.current?.capturePanorama)engine=h.memoizedState.current;h=h.next;}fiber=fiber.return;}
 if(!engine)throw Error('Missing terrain renderer');
 const {applyRenderPreset}=await import('/src/render-settings.ts'),images={};
 for(const id of ['cloudIsland','cloudGolden','cloudSoft']){
   const s=applyRenderPreset(engine.options,id);engine.apply({...s,edge:720,atmosphere:{...s.atmosphere,animate:false}});
   images[id]=await engine.capture();
 }
 return images;
 }''')
 for name,data in images.items():
  raw=base64.b64decode(data.split(',')[1]);(out/(name+'.png')).write_bytes(raw)
  with Image.open(io.BytesIO(raw)) as image:assert image.width==720
 assert not errors,errors
 print('PASS actual terrain PNGs for cloudIsland/cloudGolden/cloudSoft, fog enabled; no GPU errors',flush=True)
 browser.close()
