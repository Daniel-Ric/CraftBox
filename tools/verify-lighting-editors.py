"""Run generated Go light data through both production editor pipelines."""
from pathlib import Path
exec(Path('tools/verify-atmosphere.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
pages=json.loads(Path('tmp/lighting/scene-pages.json').read_text(encoding='utf-8'));indices={'first':0}
out=Path('tmp/lighting/editors');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 b=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=b.new_page(viewport=dict(width=1440,height=960));page.set_default_timeout(120000)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text) else None)
 page.add_init_script(init)
 page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[0])))
 page.route('**/__save',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps('lighting.png')))
 page.goto('http://127.0.0.1:5202',wait_until='networkidle');page.get_by_role('button',name='Open workspace',exact=True).click()
 page.get_by_role('button',name='Render editor',exact=True).click()
 page.wait_for_function("[...document.querySelectorAll('button')].some(b=>b.textContent==='Export PNG'&&!b.disabled)")
 page.get_by_role('tab',name='Lighting',exact=True).click()
 page.get_by_label('Sunlight',exact=True).fill('0');page.get_by_label('Ambient light',exact=True).fill('0.1')
 page.get_by_label('Skybox',exact=True).select_option('none');page.get_by_label('Sky lighting',exact=True).fill('0')
 def capture(name):
  page.wait_for_timeout(500);raw=base64.b64decode(page.locator('.render-canvas canvas').evaluate("c=>c.toDataURL().split(',')[1]"));(out/(name+'.png')).write_bytes(raw);return Image.open(io.BytesIO(raw)).convert('RGB')
 page.get_by_label('Minecraft block lighting',exact=True).uncheck();off=capture('blocklight-off')
 page.get_by_label('Minecraft block lighting',exact=True).check();on=capture('blocklight-on')
 delta=sum(ImageStat.Stat(ImageChops.difference(off,on)).mean);assert delta>.05,delta
 page.get_by_label('Enable cinematic light',exact=True).check();page.get_by_role('button',name='Fog for visible light beams',exact=True).click()
 capture('render-film-light');page.screenshot(path=str(out/'render-light-controls.png'))
 page.get_by_role('button',name='Export PNG',exact=True).click()
 page.get_by_role('dialog').get_by_role('button',name='Export …',exact=True).click()
 page.get_by_role('button',name='Cinematic editor',exact=True).click()
 page.wait_for_function("document.querySelector('.cinema-studio')&&!document.querySelector('.cinema-loading')")
 page.get_by_role('button',name='Edit lighting',exact=True).click()
 page.get_by_label('Block light intensity',exact=True).fill('2')
 page.get_by_label('Enable cinematic light',exact=True).check();page.get_by_role('button',name='Fog for visible light beams',exact=True).click()
 page.get_by_label('Cinematic light Position X',exact=True).fill('0.5')
 capture('cinematic-film-light');page.screenshot(path=str(out/'cinematic-light-controls.png'))
 assert page.get_by_label('Beam angle',exact=True).is_visible()
 assert not errors,errors
 b.close();print(json.dumps(dict(passed=True,blockLightDelta=delta,errors=errors)))
