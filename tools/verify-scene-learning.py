"""Production UI proof: scene library and local corrections on real terrain."""
from pathlib import Path
exec(Path('tools/verify-atmosphere.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
out=Path('tmp/scene-learning');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=browser.new_page(viewport=dict(width=1182,height=780),device_scale_factor=0.5);page.set_default_timeout(180000)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.add_init_script(init)
 page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
 page.goto('http://127.0.0.1:5202',wait_until='networkidle')
 page.get_by_role('button',name='Open workspace',exact=True).click()
 results=[]
 for cinematic in [False,True]:
  editor='cinematic' if cinematic else 'render'
  print(editor+' loading',flush=True)
  page.get_by_role('button',name='Cinematic editor' if cinematic else 'Render editor',exact=True).click()
  page.get_by_role('button',name='Auto-style scene',exact=True).wait_for(state='visible')
  page.wait_for_function('!document.querySelector(\'button[aria-label="Auto-style scene"]\').disabled')
  if cinematic:page.get_by_role('button',name='Scene styling',exact=True).click()
  page.get_by_role('tab',name='Looks',exact=True).click()
  page.get_by_role('button',name='Open look library',exact=True).click()
  page.get_by_role('button',name='All scenes',exact=False).click()
  assert page.locator('.look-library-grid .inspector-look').count()==60
  page.screenshot(path=str(out/(editor+'-library.png')))
  page.get_by_label('Browse library',exact=True).fill('cyberpunk')
  assert page.locator('.look-library-grid .inspector-look').count()==1
  page.locator('.look-library-grid .look-cityNeon').click()
  print(editor+' remembering neon',flush=True)
  page.get_by_role('button',name='Remember current look for auto styling',exact=True).click()
  page.get_by_text('Als Auto-Referenz Saved.',exact=False).wait_for()
  assert page.evaluate("JSON.parse(localStorage.getItem('craftbox-direction-examples')).examples[0].lookId")=='cityNeon'
  page.get_by_role('button',name='Auto-style scene',exact=True).click()
  page.get_by_text('Applied to the render view',exact=True).wait_for()
  title=page.locator('.auto-direction-report h2').inner_text()
  assert 'Neon' in title,title
  assert 'gemerkte lokale Referenz' in page.locator('.auto-direction-report').inner_text()
  page.screenshot(path=str(out/(editor+'-learned.png')))
  page.get_by_role('button',name='Close',exact=True).click()
  print(editor+' correcting reference',flush=True)

  page.get_by_label('Look category',exact=True).select_option('all')
  page.get_by_label('Search looks',exact=True).fill('Goldenes Gegenlicht')
  page.locator('.inspector-looks .look-cloudGolden').click()
  page.get_by_role('button',name='Remember current look for auto styling',exact=True).click()
  page.wait_for_function("JSON.parse(localStorage.getItem('craftbox-direction-examples')).examples[0].lookId==='cloudGolden'")
  assert page.evaluate("JSON.parse(localStorage.getItem('craftbox-direction-examples')).examples.length")==1
  page.get_by_role('button',name='Auto-style scene',exact=True).click()
  print(editor+' running corrected auto',flush=True)
  page.get_by_text('Applied to the render view',exact=True).wait_for()
  assert 'Goldenes Gegenlicht' in page.locator('.auto-direction-report h2').inner_text()
  page.screenshot(path=str(out/(editor+'-corrected.png')))
  page.get_by_role('button',name='Close',exact=True).click()
  results.append(dict(editor=editor,looks=60,learned=title,correction='cloudGolden'))
 assert not errors,errors
 (out/'result.json').write_text(json.dumps(dict(passed=True,results=results,errors=errors),indent=2),encoding='utf-8')
 print(json.dumps(dict(passed=True,results=results,errors=errors)),flush=True)
 browser.close()
