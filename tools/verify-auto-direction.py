"""Exercise real model analysis, complete settings, undo and solar time in both editors."""
from pathlib import Path
import time
import re
exec(Path('tools/verify-atmosphere.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
out=Path('tmp/auto-direction');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=browser.new_page(viewport=dict(width=1440,height=960));page.set_default_timeout(180000)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
 page.add_init_script(init)
 page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
 page.goto('http://127.0.0.1:5202',wait_until='networkidle');page.get_by_role('button',name='Open workspace',exact=True).click()
 metrics=[]
 for cinematic in [False,True]:
  name='cinematic' if cinematic else 'render';page.get_by_role('button',name='Cinematic editor' if cinematic else 'Render editor',exact=True).click()
  page.get_by_role('button',name='Auto-style scene',exact=True).wait_for(state='visible')
  page.wait_for_function('!document.querySelector(\'button[aria-label="Auto-style scene"]\').disabled')
  before=time.monotonic();page.get_by_role('button',name='Auto-style scene',exact=True).click()
  page.get_by_text('Applied to the render view',exact=True).wait_for()
  assert page.locator('.auto-direction-palette span').count()>0
  elapsed=time.monotonic()-before;page.screenshot(path=str(out/(name+'-analysis.png')))
  report=page.get_by_role('region',name='Model analysis').inner_text() if page.get_by_role('region',name='Model analysis').count() else page.locator('.auto-direction-report').inner_text()
  assert 'Scene looks' in report,report
  assert 'Surface samples' in report
  page.locator('.auto-direction-audit summary').click()
  rows=page.locator('.auto-direction-audit tbody tr').count();assert rows>=90,rows
  assert rows==int(re.search('All (\\d+)',page.locator('.auto-direction-audit summary').inner_text()).group(1))
  page.get_by_label('Automatic creative intent',exact=True).select_option('cinematic')
  page.get_by_label('Automatic time of day',exact=True).select_option('fixed');page.get_by_label('Specified time',exact=True).fill('18:30')
  page.get_by_label('Automatic atmosphere',exact=True).select_option('clear');page.get_by_label('Automatic output size',exact=True).select_option('1920')
  page.get_by_role('button',name='Restyle',exact=True).click();page.get_by_text('Applied to the render view',exact=True).wait_for()
  page.get_by_role('button',name='Close',exact=True).click()
  if cinematic:
   page.get_by_role('tab',name='Export',exact=True).click();assert page.get_by_label('MP4 resolution',exact=True).input_value()=='1920'
   page.get_by_role('button',name='Edit lighting',exact=True).click()
  else:page.get_by_role('tab',name='Lighting',exact=True).click()
  assert page.get_by_label('Time of day',exact=True).input_value()=='18:30'
  page.get_by_label('Time of day',exact=True).fill('22:00')
  assert page.get_by_label('Skybox',exact=True).input_value()=='night'
  page.wait_for_function('!document.querySelector(\'button[aria-label="Auto-style scene"]\').disabled')
  page.screenshot(path=str(out/(name+'-night.png')))
  page.get_by_label('Time of day',exact=True).fill('12:00')
  assert page.get_by_label('Skybox',exact=True).input_value()=='daylight'
  page.wait_for_function('!document.querySelector(\'button[aria-label="Auto-style scene"]\').disabled')
  page.screenshot(path=str(out/(name+'-day.png')))
  page.get_by_role('button',name='Advanced auto styling',exact=True).click()
  page.get_by_role('button',name='Restore previous settings',exact=True).click()
  page.get_by_role('button',name='Close',exact=True).click()
  metrics.append(dict(editor=name,seconds=elapsed,settings=rows,report=report))
 assert not errors,errors
 (out/'result.json').write_text(json.dumps(dict(passed=True,metrics=metrics,errors=errors),indent=2),encoding='utf-8')
 print(json.dumps(dict(passed=True,metrics=metrics,errors=errors),indent=2),flush=True);browser.close()
