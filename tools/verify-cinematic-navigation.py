"""Exercise free camera placement and the complete shared inspector on real terrain."""
from pathlib import Path
import math
exec(Path('tools/verify-atmosphere.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
out=Path('tmp/cinematic-navigation');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=browser.new_page(viewport=dict(width=1440,height=960));page.set_default_timeout(120000)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text) else None)
 page.add_init_script(init)
 page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
 page.goto('http://127.0.0.1:5202',wait_until='networkidle')
 page.get_by_role('button',name='Open workspace',exact=True).click()
 page.get_by_role('button',name='Cinematic editor',exact=True).click()
 page.wait_for_function("document.querySelector('.cinema-studio') && !document.querySelector('.cinema-loading')",timeout=120000)
 page.get_by_role('button',name='Edit clouds and fog',exact=True).click()
 page.get_by_label('Volumetric clouds',exact=True).check()
 page.get_by_label('Cloud height reference',exact=True).select_option('bottom')
 page.get_by_label('Cloud height',exact=True).fill('-50')
 page.get_by_label('Volumetric fog',exact=True).check()
 page.get_by_label('Fog density',exact=True).fill('0.1')
 page.screenshot(path=str(out/'atmosphere.png'))
 page.get_by_role('button',name='Edit colors',exact=True).click()
 page.get_by_label('Saturation Value',exact=True).fill('1.25');page.get_by_label('Saturation Value',exact=True).press('Tab')
 page.screenshot(path=str(out/'color.png'))
 page.get_by_role('button',name='Scene styling',exact=True).click()
 page.get_by_role('button',name='Edit lighting',exact=True).click()
 assert page.get_by_label('Exposure Value',exact=True).is_visible()
 page.get_by_role('button',name='Edit camera and sequence',exact=True).click()
 page.get_by_role('button',name='Free camera',exact=True).click()
 page.get_by_role('button',name='Update keyframe',exact=True).click()
 def key(): return page.evaluate("JSON.parse(localStorage.getItem('craftbox-cinematic:render-fixture')).keys[0]")
 before=key()
 canvas=page.get_by_label('3D camera view',exact=True)
 canvas.click(position=dict(x=200,y=180));page.keyboard.down('w');page.wait_for_timeout(700);page.keyboard.up('w')
 page.get_by_role('button',name='Update keyframe',exact=True).click();moved=key()
 dist=lambda a,b:math.sqrt(sum((x-y)**2 for x,y in zip(a,b)))
 assert dist(before['position'],moved['position'])>.001,(before,moved)
 assert abs(dist(before['target'],moved['target'])-dist(before['position'],moved['position']))<.00001
 box=canvas.bounding_box();x=box['x']+200;y=box['y']+180
 page.mouse.move(x,y);page.mouse.down(button='right');page.mouse.move(x+90,y+30,steps=5);page.mouse.up(button='right')
 page.get_by_role('button',name='Update keyframe',exact=True).click();turned=key()
 assert dist(moved['position'],turned['position'])<.00001
 assert dist(moved['target'],turned['target'])>.01

 canvas.click(position=dict(x=200,y=180));page.keyboard.down('w')
 page.get_by_label('Flight speed',exact=True).focus();page.keyboard.up('w')
 page.get_by_role('button',name='Update keyframe',exact=True).click();blurred=key()
 page.wait_for_timeout(300);page.get_by_role('button',name='Update keyframe',exact=True).click()
 assert dist(blurred['position'],key()['position'])<.00001
 canvas.focus();page.keyboard.press('Escape')
 assert page.get_by_role('button',name='Free camera',exact=True).get_attribute('aria-pressed')=='false'
 page.get_by_role('button',name='Free camera',exact=True).click()
 page.screenshot(path=str(out/'free-camera.png'))
 for width,height in [(1182,695),(1024,640)]:
  page.set_viewport_size(dict(width=width,height=height));page.wait_for_timeout(100)
  assert page.evaluate('document.documentElement.scrollWidth<=innerWidth')
  page.screenshot(path=str(out/f'free-camera-{width}.png'))
 assert not errors,errors
 browser.close()
print('Cinematic: shared atmosphere/color/light, view-relative movement, look, keyframe placement, blur, Escape and compact layouts passed')
