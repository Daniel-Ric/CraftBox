"""Reopen the cloud projects produced by the real editor export verifier."""
from pathlib import Path
source=Path(__file__).with_name('verify-cloud-authoring-editors.py').read_text(encoding='utf-8')
exec(compile(source.split('\nwith sync_playwright() as pw:')[0],__file__,'exec'))
with sync_playwright() as pw:
 browser=pw.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 for name in ['render','cinematic']:
  document=json.loads((args.output/(name+'-saved.json')).read_text(encoding='utf-8'))
  page=browser.new_page(viewport=dict(width=1520,height=1080));page.set_default_timeout(90000)
  errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
  page.add_init_script(init+'\nwindow.__saved.push('+json.dumps(document)+');')
  page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
  try:
   page.goto(args.url,wait_until='networkidle')
   page.get_by_role('button',name='Open project',exact=True).click()
   page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay')")
   if name=='cinematic':
    if page.get_by_role('button',name='Edit 3D path',exact=True).count():page.get_by_role('button',name='Edit 3D path',exact=True).click()
    page.get_by_role('button',name='Edit clouds and fog',exact=True).click()
   else:page.get_by_role('tab',name='Atmosphere',exact=True).click()
   assert page.locator('.cloud-object-list>div').count()==3
   assert page.get_by_label('Cloud export quality',exact=True).input_value()=='medium'
   before=float(page.get_by_label('Cloud position X',exact=True).input_value())
   page.get_by_role('button',name='Place cloud in view',exact=True).click()
   box=page.locator('.render-canvas canvas').bounding_box();page.mouse.click(box['x']+box['width']*.55,box['y']+box['height']*.55)
   page.get_by_role('button',name='Place cloud in view',exact=True).wait_for()
   assert abs(float(page.get_by_label('Cloud position X',exact=True).input_value())-before)>.001
   page.get_by_role('button',name='Save project',exact=True).click()
   page.wait_for_function('window.__saved.length>1')
   saved=page.evaluate('window.__saved.at(-1)')
   assert saved['settings']['atmosphere']['cloudOpening']==document['settings']['atmosphere']['cloudOpening']
   assert [l['tracks'] for l in saved['settings']['atmosphere']['cloudLayers']]==[l['tracks'] for l in document['settings']['atmosphere']['cloudLayers']]
   assert not errors,errors
   print('PASS',name,'native project reopen, immediate viewport placement, clouds/curves/opening retained',flush=True)
  except Exception:
   page.screenshot(path=str(args.output/(name+'-reopen-failure.png')))
   print(page.locator('body').inner_text()[-7000:].encode('ascii',errors='backslashreplace').decode(),flush=True);raise
  finally:page.close()
 browser.close()
