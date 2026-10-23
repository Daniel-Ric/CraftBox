"""Actual terrain in both production editors: bank model, signed height and PNG export."""
from pathlib import Path
exec(Path('tools/verify-atmosphere.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
out=Path('tmp/cloud-banks/editors');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 b=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=b.new_page(viewport=dict(width=1280,height=900));page.set_default_timeout(180000)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
 page.add_init_script(init)
 page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
 saved=[]
 def save(r):
  target=out/('export-'+str(len(saved))+'.png');target.write_bytes(base64.b64decode(r.request.post_data_json['data'].split(',')[1]));saved.append(str(target));r.fulfill(status=200,content_type='application/json',body=json.dumps(str(target)))
 page.route('**/__save',save)
 page.goto('http://127.0.0.1:5202',wait_until='networkidle');page.get_by_role('button',name='Open workspace',exact=True).click()
 for cinematic in [False,True]:
  name='cinematic' if cinematic else 'render'
  page.get_by_role('button',name='Cinematic editor' if cinematic else 'Render editor',exact=True).click()
  page.wait_for_function('document.querySelector(\'button[aria-label="Auto-style scene"]\')?.disabled===false')
  if cinematic:page.get_by_role('button',name='Scene styling',exact=True).click()
  page.get_by_role('tab',name='Looks',exact=True).click()
  page.get_by_role('button',name='Cloud island · Daylight',exact=False).click()
  page.get_by_role('tab',name='Details',exact=True).click();page.get_by_role('button',name='Fast',exact=True).click()
  page.get_by_role('tab',name='Atmosphere',exact=True).click()
  assert page.get_by_label('Cloud model',exact=True).input_value()=='banks'
  page.get_by_label('Cloud height',exact=True).fill('-120');page.get_by_label('Cloud thickness',exact=True).fill('200');page.get_by_label('Cloud size',exact=True).fill('160')
  page.get_by_label('Animate atmosphere',exact=True).uncheck()
  page.wait_for_timeout(1800);page.screenshot(path=str(out/(name+'.png')))
  page.get_by_role('tab',name='Lighting',exact=True).click()
  assert page.get_by_label('Adapt to daylight',exact=True).is_checked()
  page.get_by_role('button',name='Export PNG',exact=True).click()
  page.get_by_role('dialog').get_by_label('Longest edge',exact=True).select_option('720')
  page.get_by_role('dialog').get_by_role('button',name='Export …',exact=True).click()
  page.wait_for_function("[...document.querySelectorAll('button')].some(b=>b.textContent==='Export PNG'&&!b.disabled)")
 assert len(saved)==2,saved
 assert not errors,errors
 for path in saved:assert max(Image.open(path).size)==720
 print(json.dumps(dict(passed=True,files=saved,errors=errors)),flush=True);b.close()
