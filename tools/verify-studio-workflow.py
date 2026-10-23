"""Production UI: deferred loading, full overlay, 2D PNG markers and project guard."""
from pathlib import Path
exec(Path('tools/verify-atmosphere.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
out=Path('tmp/studio-workflow');out.mkdir(parents=True,exist_ok=True)
init=init.replace("window.runtime={EventsOn:()=>()=>{}};","window.__listeners={};window.__saved=[];window.__closed=0;window.runtime={EventsOn:(n,cb)=>{(window.__listeners[n]??=[]).push(cb);return ()=>window.__listeners[n]=window.__listeners[n].filter(x=>x!==cb)}};")
init=init.replace('GetBootstrap:',"SetStudioProjectDirty:async d=>{window.__dirty=d},ConfirmStudioProjectClose:async()=>{window.__closed++},SaveStudioProject:async(worldId,name,document,saveAs)=>{window.__saved.push({worldId,name,document,saveAs});if(window.__cancelSave){window.__cancelSave=false;return '';}return 'C:/test/scene.craftbox';},OpenStudioProject:async()=>window.__openProject,GetBootstrap:")
with sync_playwright() as p:
 b=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=b.new_page(viewport=dict(width=1182,height=780),device_scale_factor=.5);page.set_default_timeout(120000)
 errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
 page.add_init_script(init);pending=[];hold=[True];skies=[];saved=[]
 page.on('request',lambda r:skies.append(r.url) if '/skies/' in r.url else None)
 def route(r):
  index=indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]
  if hold[0] and index==0:pending.append(r)
  else:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[index]))
 page.route('**/__page?*',route)
 def save(r):
  path=out/('map-'+str(len(saved))+'.png');path.write_bytes(base64.b64decode(r.request.post_data_json['data'].split(',')[1]));saved.append(str(path));r.fulfill(status=200,content_type='application/json',body=json.dumps(str(path)))
 page.route('**/__save',save)
 page.goto('http://127.0.0.1:5202',wait_until='domcontentloaded');page.get_by_role('button',name='Open workspace',exact=True).click();page.get_by_role('button',name='Render editor',exact=True).click()
 page.locator('.editor-loading-overlay').wait_for()
 boxes=page.evaluate('''()=>{let a=document.querySelector('.editor-loading-overlay').getBoundingClientRect(),b=document.querySelector('.render-workspace').getBoundingClientRect(),c=document.querySelector('.editor-loading-content').getBoundingClientRect();return {w:a.width,h:a.height,bw:b.width,bh:b.height,dx:(c.x+c.width/2)-(a.x+a.width/2),dy:(c.y+c.height/2)-(a.y+a.height/2)}}''')
 assert abs(boxes['w']-boxes['bw'])<1 and abs(boxes['h']-boxes['bh'])<1 and abs(boxes['dx'])<1 and abs(boxes['dy'])<1,boxes
 assert not skies,skies
 page.screenshot(path=str(out/'loading-dark.png'));page.evaluate("document.documentElement.classList.remove('dark');document.documentElement.classList.add('light')");page.screenshot(path=str(out/'loading-light.png'));page.evaluate("document.documentElement.classList.remove('light');document.documentElement.classList.add('dark')")
 hold[0]=False
 for r in pending:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[0]))
 print('loading overlay checked; releasing terrain',flush=True)
 page.wait_for_function('document.querySelector(\'button[aria-label="Auto-style scene"]\')?.disabled===false')
 page.get_by_role('button',name='2D map',exact=True).click();page.get_by_label('Show grid',exact=True).check();page.get_by_label('Elevation tint strength',exact=True).fill('0.25')
 def export():
  page.get_by_role('button',name='Export PNG',exact=True).click();dialog=page.get_by_role('dialog');dialog.get_by_label('Longest edge',exact=True).select_option('720');dialog.get_by_role('button',name='Export …',exact=True).click();page.wait_for_function("[...document.querySelectorAll('button')].some(b=>b.textContent==='Export PNG'&&!b.disabled)")
 export();page.get_by_role('button',name='Place marker on map',exact=True).click()
 canvas=page.get_by_label('3D camera view',exact=True);canvas.click(position=dict(x=250,y=170));page.get_by_label('Marker label',exact=True).fill('Hub Spawn');page.get_by_label('Marker symbol',exact=True).fill('★')
 page.screenshot(path=str(out/'map-marker.png'));export()
 first,second=Image.open(saved[0]).convert('RGBA'),Image.open(saved[1]).convert('RGBA')
 assert second.getpixel((0,0))[3]==0,second.getpixel((0,0));assert ImageChops.difference(first,second).getbbox() is not None,'marker missing from PNG'
 page.get_by_role('button',name='Save project',exact=True).click();page.wait_for_function('window.__dirty===false')
 document=page.evaluate('JSON.parse(window.__saved.at(-1).document)');assert document['mapView']['markers'][0]['label']=='Hub Spawn';(out/'saved-document.json').write_text(json.dumps(document),encoding='utf-8')
 page.get_by_label('Marker label',exact=True).fill('Quest Master');page.wait_for_function('window.__dirty===true');page.evaluate("window.__listeners['studio:close-requested'].forEach(cb=>cb(true))")
 page.get_by_role('dialog',name='Save changes?').wait_for();page.get_by_role('button',name='Cancel',exact=True).click();assert page.evaluate('window.__closed')==0
 page.get_by_role('button',name='Worlds',exact=True).click();page.get_by_role('dialog',name='Save changes?').wait_for();page.evaluate('window.__cancelSave=true');page.get_by_role('button',name='Save',exact=True).click();page.get_by_role('alert').filter(has_text='not saved').wait_for();page.get_by_role('button',name='Save',exact=True).click();page.get_by_role('heading',name='Worlds',exact=True).wait_for()
 print('transparent PNG markers and save guard checked',flush=True)
 for name,rotation in [('Portable A',45),('Portable B',120)]:
  doc=json.loads(json.dumps(document));doc['name']=name;doc['mapView']['rotation']=rotation;page.evaluate('(v)=>window.__openProject=v',dict(world=dict(id=name,name=name,chunks=48),document=json.dumps(doc),path='C:/test/'+name+'.craftbox'))
  if name=='Portable A':page.get_by_role('menuitem',name='File',exact=True).click();page.get_by_role('menuitem',name='Open studio project …',exact=False).click()
  else:page.get_by_role('button',name='Open project',exact=True).click()
  page.get_by_label('Map rotation',exact=True).wait_for();page.wait_for_function("!document.querySelector('.editor-loading-overlay')");assert float(page.get_by_label('Map rotation',exact=True).input_value())==rotation;assert page.get_by_label('Marker label',exact=True).input_value()=='Hub Spawn';page.wait_for_function('window.__dirty===false')
 print('same-editor project A to B reopening checked',flush=True)
 page.get_by_role('button',name='Cinematic editor',exact=True).click();page.get_by_role('button',name='Edit scene looks',exact=True).click();page.wait_for_function("!document.querySelector('.editor-loading-overlay')");page.get_by_role('button',name='Open look library',exact=True).click();page.get_by_role('button',name='All scenes',exact=False).click();assert page.locator('.look-library-grid .inspector-look').count()==60;page.screenshot(path=str(out/'cinematic-scenes.png'))
 page.keyboard.press('Escape');page.get_by_role('button',name='Edit camera and sequence',exact=True).click();page.get_by_role('button',name='Free camera',exact=True).click()
 page.get_by_role('button',name='Save project',exact=True).click();page.wait_for_function('window.__dirty===false')
 camera=page.get_by_label('3D camera view',exact=True);camera.focus();page.keyboard.down('w');page.wait_for_function('window.__dirty===true');page.keyboard.up('w')
 page.get_by_role('button',name='Worlds',exact=True).click();page.get_by_role('dialog',name='Save changes?').wait_for();page.get_by_role('button',name='Cancel',exact=True).click();page.get_by_role('button',name='Free camera',exact=True).wait_for()
 print('free camera movement triggers project guard',flush=True)
 assert not errors,errors
 (out/'result.json').write_text(json.dumps(dict(passed=True,overlay=boxes,png=saved,errors=errors),indent=2),encoding='utf-8');print(json.dumps(dict(passed=True,overlay=boxes,png=saved,errors=errors)),flush=True);b.close()
