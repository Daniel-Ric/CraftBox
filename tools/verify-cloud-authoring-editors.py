"""Exercise cloud authoring in both real terrain editors and native project saves."""
from pathlib import Path
import re
from PIL import ImageStat,ImageChops
fixture=Path(__file__).with_name('verify-studio-lights.py').read_text(encoding='utf-8')
exec(compile(fixture.split('with sync_playwright() as pw:')[0],str(Path(__file__).with_name('verify-studio-lights.py')),'exec'))
args.output=Path('tmp/cloud-authoring-editors');args.output.mkdir(parents=True,exist_ok=True)
init+="\nwindow.__movies=[];window.go.desktop.App.SaveCinematicFile=async(name,kind,data)=>{window.__movies.push({name,kind,data});return 'C:/test/clouds.'+kind;}"
init+="\nwindow.go.desktop.App.OpenStudioProject=async()=>({path:'C:/test/clouds.craftbox',world:{id:'render-fixture',name:'Studio terrain',chunks:48},document:JSON.stringify(window.__saved.at(-1))});window.go.desktop.App.CloseRenderWorld=async()=>{};"
with sync_playwright() as pw:
 browser=pw.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 for cinematic in [False,True]:
  if cinematic and args.only_render:continue
  page=browser.new_page(viewport=dict(width=1520,height=1080));page.set_default_timeout(90000)
  errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
  page.on('console',lambda m:errors.append(m.text) if m.type=='error' and any(t in m.text.lower() for t in ['shader','webgl','three']) else None)
  page.add_init_script(init)
  page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
  name='cinematic' if cinematic else 'render'
  try:
   page.goto(args.url,wait_until='networkidle');page.get_by_role('button',name='Open workspace',exact=True).click()
   page.get_by_role('button',name='Cinematic editor' if cinematic else 'Render editor',exact=True).click()
   page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay')")
   if cinematic:
    sequence=dict(version=1,name='Cloud animation',duration=.5,fps=24,keys=[dict(id='a',time=0,position=[.6,.25,1],target=[0,0,0],fov=55,roll=0,exposure=1,ease='linear')])
    page.locator('.cinema-project input[type=file]').set_input_files(dict(name='clouds.json',mimeType='application/json',buffer=json.dumps(sequence).encode()))
    page.get_by_role('button',name='Edit 3D path',exact=True).click()
    page.get_by_role('button',name='Edit clouds and fog',exact=True).click()
   else:page.get_by_role('tab',name='Atmosphere',exact=True).click()
   page.get_by_role('button',name='Frame island with clouds',exact=True).click()
   assert page.locator('.cloud-object-list>div').count()==3
   page.get_by_label('Cloud preview quality',exact=True).select_option('low')
   page.get_by_label('Cloud export quality',exact=True).select_option('medium')
   page.get_by_label('Animate atmosphere',exact=True).uncheck()
   page.get_by_role('button',name='Foreground',exact=True).click()
   page.get_by_label('Cloud position X',exact=True).fill('0.7')
   page.get_by_role('button',name='Place cloud in view',exact=True).click()
   canvas=page.locator('.render-canvas canvas');box=canvas.bounding_box()
   page.mouse.click(box['x']+box['width']*.5,box['y']+box['height']*.6)
   page.get_by_role('button',name='Place cloud in view',exact=True).wait_for()
   placed=float(page.get_by_label('Cloud position X',exact=True).input_value());assert abs(placed-.7)>.001,placed
   page.get_by_label('Cloud tool',exact=True).select_option('rotate')
   page.get_by_label('Cloud rotation Y',exact=True).fill('35')
   page.get_by_label('Cloud position X',exact=True).fill('0.9')
   page.get_by_label('Cloud position Y',exact=True).fill('-0.7')
   page.get_by_label('Cloud position Z',exact=True).fill('0.8')
   page.get_by_role('button',name='Duplicate cloud region',exact=True).click()
   assert page.locator('.cloud-object-list>div').count()==4
   page.get_by_role('button',name='Delete cloud region',exact=True).click()
   assert page.locator('.cloud-object-list>div').count()==3
   if cinematic:
    page.get_by_role('button',name='Foreground',exact=True).click()
    page.get_by_label('Playhead position',exact=True).fill('0')
    page.get_by_label('Cloud animation channel',exact=True).select_option('dissolve')
    page.get_by_role('button',name='Add cloud keyframe',exact=True).click()
    page.get_by_label('Playhead position',exact=True).fill('0.5')
    page.get_by_label('Region breakup',exact=True).fill('1')
    assert page.get_by_role('button',name=re.compile('^Curve keyframe [12]$')).count()==2
   page.get_by_role('button',name='Save project',exact=True).click();page.wait_for_function('window.__saved.length>0')
   doc=page.evaluate('window.__saved.at(-1)');s=doc['settings']['atmosphere']
   assert len(s['cloudLayers'])==3 and s['cloudOpening']['strength']==1,s
   assert s['cloudPreviewQuality']=='low' and s['cloudExportQuality']=='medium',s
   if cinematic:assert any(t['channel']=='dissolve' and len(t['keys'])==2 for l in s['cloudLayers'] for t in l['tracks']),s
   (args.output/(name+'-saved.json')).write_text(json.dumps(doc),encoding='utf-8')
   page.screenshot(path=str(args.output/(name+'-authoring.png')))
   if cinematic:page.get_by_label('Playhead position',exact=True).fill('0.25')
   page.evaluate('window.__png=undefined')
   page.get_by_role('button',name='Export PNG',exact=True).click()
   page.get_by_role('dialog').get_by_label('Longest edge',exact=True).select_option('720')
   page.get_by_role('button',name='Export …',exact=True).click()
   page.wait_for_function('!!window.__png',timeout=300000)
   raw=base64.b64decode(page.evaluate('window.__png').split(',')[1]);(args.output/(name+'.png')).write_bytes(raw)
   with Image.open(io.BytesIO(raw)) as image:assert image.width==720 and sum(ImageStat.Stat(image.convert('RGB')).mean)>30
   if cinematic:
    page.get_by_role('button',name='GIF',exact=True).click()
    page.get_by_role('dialog').get_by_label('Longest edge',exact=True).select_option('320')
    page.get_by_role('button',name='Export …',exact=True).click()
    page.wait_for_function('window.__movies.length>0',timeout=300000)
    movie=page.evaluate('window.__movies.at(-1)');data=base64.b64decode(movie['data']);(args.output/'cloud-animation.gif').write_bytes(data)
    with Image.open(io.BytesIO(data)) as image:
     assert image.n_frames>1,image.n_frames
     first=image.convert('RGB').copy();image.seek(image.n_frames-1)
     delta=sum(ImageStat.Stat(ImageChops.difference(first,image.convert('RGB'))).mean)/3
     assert delta>.1,delta
     print('PASS actual cloud-animation GIF with static camera; pixel delta:',delta,flush=True)
   if not cinematic and args.transparent_panorama:
    page.get_by_role('tab',name='Output',exact=True).click()
    page.get_by_label('Transparent background',exact=True).check()
    page.evaluate('window.__png=undefined')
    page.get_by_role('button',name='360° panorama',exact=True).click()
    page.get_by_label('Panorama resolution',exact=True).select_option('1024')
    page.get_by_role('button',name='Render panorama',exact=True).click()
    page.wait_for_function('!!window.__png',timeout=300000)
    data=base64.b64decode(page.evaluate('window.__png').split(',')[1]);(args.output/'cloud-panorama-transparent.png').write_bytes(data)
    with Image.open(io.BytesIO(data)) as image:
     assert image.size==(1024,512),image.size
     assert image.getchannel('A').getextrema()[0]<255
    print('PASS actual transparent cloud panorama',flush=True)
   saved=page.evaluate('window.__saved.length')
   page.get_by_role('button',name='Save project',exact=True).click()
   page.wait_for_function('(n)=>window.__saved.length>n',arg=saved)
   page.get_by_role('button',name='Open project',exact=True).click()
   page.wait_for_function("document.querySelector('.render-canvas canvas') && !document.querySelector('.editor-loading-overlay')")
   if cinematic:
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
   print('PASS native cloud project reopen and immediate viewport placement',name,flush=True)
   assert not errors,errors
   print('PASS',name,'cloud layers, surface placement, rotation, duplication, opening, animation/project save and actual PNG; no GPU errors',flush=True)
  except Exception:
   page.screenshot(path=str(args.output/(name+'-failure.png')))
   print('ERRORS',errors,flush=True);print(page.locator('body').inner_text()[-12000:].encode('ascii',errors='backslashreplace').decode(),flush=True);raise
  finally:page.close()
 browser.close()
