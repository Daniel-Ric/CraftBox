"""Capture representative new looks through the production renderer."""
from pathlib import Path
exec(Path('tools/verify-atmosphere.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
out=Path('tmp/scene-learning');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=browser.new_page(viewport=dict(width=1182,height=780),device_scale_factor=0.5);page.set_default_timeout(180000)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
 page.add_init_script(init)
 page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
 page.goto('http://127.0.0.1:5202',wait_until='networkidle')
 page.get_by_role('button',name='Open workspace',exact=True).click()
 page.get_by_role('button',name='Render editor',exact=True).click()
 page.get_by_role('button',name='Auto-style scene',exact=True).wait_for(state='visible')
 page.wait_for_function('!document.querySelector(\'button[aria-label="Auto-style scene"]\').disabled')
 page.get_by_label('Look category',exact=True).select_option('all')
 captures=[]
 for look in ['cloudPeach','winterClear','coastLagoon','interiorWarm']:
  page.locator('.inspector-looks .look-'+look).click()
  page.evaluate('()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))')
  page.screenshot(path=str(out/(look+'.png')))
  captures.append(look);print(look,flush=True)
 assert not errors,errors
 (out/'previews.json').write_text(json.dumps(dict(passed=True,captures=captures,errors=errors)),encoding='utf-8')
 print(json.dumps(dict(passed=True,captures=captures,errors=errors)),flush=True)
 browser.close()
