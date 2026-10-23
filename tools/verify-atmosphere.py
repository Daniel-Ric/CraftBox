"""Render real terrain through the production atmosphere pipeline and save evidence."""
import sys
import io
import base64
import json
from pathlib import Path
from urllib.parse import parse_qs, urlparse
from PIL import Image, ImageChops, ImageStat
from playwright.sync_api import sync_playwright

pages=json.loads(Path('tmp/Render editor/terrain-pages.json').read_text(encoding='utf-8'))
indices={'first':0}
for i,p in enumerate(pages[:-1]): indices[','.join(map(str,p['next']))]=i+1
state=dict(revision=1,command='worlds',running=False,worlds=[dict(id='render-fixture',name='Atmosphere terrain',chunks=48,state='finished')],packs=[],skins=[],player=[0,0,0],loggedIn=False)
bootstrap=dict(version='Atmosphere verification',minecraft='1.26.52',commands=[],state=state)
init="""window.runtime={EventsOn:()=>()=>{}};window.go={desktop:{App:{GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[],GetRenderScenePage:async(id,r)=>fetch('/__page?cursor='+encodeURIComponent(r.cursor?r.cursor.join(','):'first')).then(r=>r.json()),SaveRenderPNG:async(name,data)=>fetch('/__save',{method:'POST',body:JSON.stringify({data})}).then(r=>r.json())}}};""".replace('BOOTSTRAP',json.dumps(bootstrap))
out=Path('tmp/atmosphere/terrain');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
  browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
  page=browser.new_page(viewport=dict(width=1182,height=780));page.set_default_timeout(120000)
  errors=[]
  page.on('pageerror',lambda e:errors.append(str(e)))
  page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
  page.add_init_script(init)
  page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
  saved=[]
  def save(r):
    path=out/'atmosphere-export.png';path.write_bytes(base64.b64decode(r.request.post_data_json['data'].split(',')[1]));saved.append(path)
    r.fulfill(status=200,content_type='application/json',body=json.dumps(str(path)))
  page.route('**/__save',save)
  page.goto('http://127.0.0.1:5202',wait_until='networkidle')
  page.get_by_role('button',name='Open workspace',exact=True).click()
  page.get_by_role('button',name='Render editor',exact=True).click()
  page.wait_for_function("[...document.querySelectorAll('button')].some(b=>b.textContent==='Export PNG'&&!b.disabled)",timeout=120000)
  page.get_by_role('tab',name='Details',exact=True).click()
  page.get_by_role('button',name='Fast',exact=True).click()
  page.get_by_role('tab',name='Atmosphere',exact=True).click()
  page.get_by_role('button',name='Open cloud library',exact=True).click()
  assert page.locator('.atmosphere-preset').count()==14
  page.screenshot(path=str(out/'cloud-library.png'))
  page.get_by_role('button',name='Small fair-weather clouds',exact=False).click()
  assert page.get_by_label('Cloud height reference',exact=True).input_value()=='top'
  page.get_by_label('Cloud height',exact=True).fill('-32')
  page.get_by_label('Cloud height',exact=True).press('Tab')
  assert page.get_by_label('Cloud height',exact=True).input_value()=='-32'
  page.get_by_label('Cloud height reference',exact=True).select_option('world')
  page.get_by_label('Volumetric clouds',exact=True).uncheck()
  page.get_by_role('button',name='Open fog library',exact=True).click()
  assert page.locator('.atmosphere-preset').count()==12
  page.screenshot(path=str(out/'fog-library.png'))
  page.get_by_role('button',name='Close',exact=True).click()
  page.get_by_label('Animate atmosphere',exact=True).uncheck()
  canvas=page.locator('.render-canvas canvas')
  def image(name):
    page.wait_for_timeout(1800)
    assert not errors,errors
    data=canvas.evaluate("e=>e.toDataURL('image/png')").split(',')[1]
    raw=base64.b64decode(data);(out/(name+'.png')).write_bytes(raw)
    print('rendered',name,flush=True)
    return Image.open(io.BytesIO(raw)).convert('RGB')
  baseline=image('clear')
  page.get_by_label('Volumetric clouds',exact=True).check()
  page.get_by_label('Cloud height',exact=True).fill('70')
  page.get_by_label('Cloud thickness',exact=True).fill('65')
  page.get_by_label('Cloud size',exact=True).fill('60')
  page.get_by_label('Cloud cover',exact=True).fill('0.8')
  images={}
  for kind in ['cumulus','stratus','storm','cirrus','stratocumulus','altocumulus','lenticular','anvil']:
    page.get_by_label('Cloud type',exact=True).select_option(kind)
    images[kind]=image(kind)
    assert sum(ImageStat.Stat(ImageChops.difference(baseline,images[kind])).mean)>2,kind
  assert len({im.tobytes() for im in images.values()})==8


  page.get_by_label('Cloud type',exact=True).select_option('cumulus')
  page.get_by_label('Cloud cover',exact=True).fill('0.55')
  page.get_by_label('Cloud height',exact=True).fill('250')
  page.get_by_label('Cloud shadows on terrain',exact=True).fill('0')
  no_shadow=image('cloud-shadow-off')
  page.get_by_label('Cloud shadows on terrain',exact=True).fill('1')
  shadow=image('cloud-shadow-on')
  delta=sum(ImageStat.Stat(ImageChops.difference(no_shadow,shadow)).mean)
  assert delta>0.01,('Cloud shadows did not affect real loaded terrain',delta)
  page.get_by_label('Cloud height',exact=True).fill('70')
  page.get_by_label('Volumetric clouds',exact=True).uncheck()
  page.get_by_label('Volumetric fog',exact=True).check()
  page.get_by_label('Fog density',exact=True).fill('1.1')
  page.get_by_label('Fog height',exact=True).fill('65')
  fog=image('fog')
  assert sum(ImageStat.Stat(ImageChops.difference(baseline,fog)).mean)>2
  page.get_by_label('Fog height',exact=True).fill('-100')
  low=image('low-fog')
  assert sum(ImageStat.Stat(ImageChops.difference(fog,low)).mean)>2
  page.get_by_label('Fog height',exact=True).fill('65')
  page.get_by_label('Volumetric clouds',exact=True).check()
  page.get_by_label('Cloud type',exact=True).select_option('cumulus')
  image('combined');page.screenshot(path=str(out/'inspector.png'))
  page.get_by_role('button',name='Export PNG',exact=True).click()
  page.get_by_role('dialog').get_by_label('Longest edge',exact=True).select_option('720')
  page.get_by_role('button',name='Export …',exact=True).click()
  page.get_by_text('Saved: tmp',exact=False).wait_for(timeout=120000)
  assert saved and Image.open(saved[0]).size==(720,405)
  assert not errors,errors
  (out/'verification.json').write_text(json.dumps(dict(types=list(images),export=str(saved[0]),browserErrors=errors),indent=2))
  browser.close()
  print('Atmosphere terrain and PNG export passed.',flush=True)
