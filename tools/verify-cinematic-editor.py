"""Exercise the real render editor with recorded Go terrain-page responses."""
import argparse
import base64
import json
import time
from pathlib import Path
from urllib.parse import parse_qs, urlparse
from PIL import Image
from playwright.sync_api import sync_playwright

p = argparse.ArgumentParser()
p.add_argument('--pages', type=Path, required=True)
p.add_argument('--url', default='http://127.0.0.1:5201')
p.add_argument('--output', type=Path, default=Path('tmp/Render editor/browser'))
p.add_argument('--quick', action='store_true')
p.add_argument('--skies', action='store_true')
p.add_argument('--orbit-check', action='store_true')
p.add_argument('--atmosphere', action='store_true')
p.add_argument('--lighting', action='store_true')
args = p.parse_args()
pages = json.loads(args.pages.read_text(encoding='utf-8'))
args.output.mkdir(parents=True, exist_ok=True)
total = sum(len(x['chunks']) for x in pages)
indices = {'first': 0}
for i, page in enumerate(pages[:-1]):
    indices[','.join(map(str, page['next']))] = i + 1
state = dict(revision=1, command='worlds', running=False, connection=0,
             worlds=[dict(id='render-fixture', name='CubeCraft Render', chunks=total, state='finished')],
             packs=[], skins=[], player=[0,0,0], hasPlayer=False, loggedIn=False)
bootstrap = dict(version='Render verification', minecraft='1.26.52', commands=[], state=state)
init = '''window.runtime={EventsOn:()=>()=>{}};
window.go={desktop:{App:{
GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[],
GetRenderScenePage:async(id,r)=>fetch('/__render-page?cursor='+encodeURIComponent(r.cursor?r.cursor.join(','):'first')).then(r=>r.json()),
SaveRenderPNG:async(name,data)=>fetch('/__save-render',{method:'POST',body:JSON.stringify({name,data})}).then(r=>r.json()),
SaveCinematicFile:async(name,kind,data)=>fetch('/__save-cinematic',{method:'POST',body:JSON.stringify({name,kind,data})}).then(r=>r.json()),OpenRenderWorld:async()=>({id:'imported-fixture',name:'Imported world',chunks:TOTAL}),
CancelRenderImport:async()=>{},CloseRenderWorld:async()=>{},OpenResultFolder:async()=>{}
}}};'''.replace('BOOTSTRAP', json.dumps(bootstrap)).replace('TOTAL', str(total))
with sync_playwright() as pw:
    browser = pw.chromium.launch(channel='msedge', headless=True,
        args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
    page = browser.new_page(viewport=dict(width=1520,height=980),device_scale_factor=1)
    errors = []
    page.on('framenavigated', lambda frame: print('NAV',frame.url,flush=True) if frame==page.main_frame else None)
    page.on('response', lambda response: print('SKY',response.status,response.url,flush=True) if '.hdr' in response.url else None)
    page.on('pageerror', lambda e: errors.append(str(e)))
    page.on('console', lambda m: errors.append(m.text) if m.type == 'error' and ('WebGL' in m.text or 'THREE.' in m.text or 'shader' in m.text.lower()) else None)
    page.add_init_script(init)
    page.route('**/__render-page?*', lambda route: route.fulfill(status=200,content_type='application/json',
        body=json.dumps(pages[indices[parse_qs(urlparse(route.request.url).query)['cursor'][0]]])))
    saved = []
    def save(route):
        data = json.loads(route.request.post_data)
        path = args.output / f'render-{len(saved)+1}.png'
        path.write_bytes(base64.b64decode(data['data'].split(',',1)[1]))
        saved.append(path)
        route.fulfill(status=200,content_type='application/json',body=json.dumps(str(path)))
    page.route('**/__save-render', save)
    page.goto(args.url,wait_until='networkidle')
    page.screenshot(path=str(args.output/'workspace-dark.png'),full_page=True)
    saved_videos=[]
    def save_video(route):
        payload=route.request.post_data_json
        path=args.output/('export.'+payload['kind'])
        path.write_bytes(base64.b64decode(payload['data']))
        saved_videos.append(path)
        route.fulfill(status=200,content_type='application/json',body=json.dumps(str(path)))
    page.route('**/__save-cinematic',save_video)
    page.get_by_role('button',name='Create new',exact=True).click()
    page.get_by_role('tab',name='Camera sequence',exact=True).click()
    page.get_by_label('Project name',exact=True).fill('Studio Browser Test')
    page.get_by_label('Duration (seconds)',exact=True).fill('20')
    page.get_by_label('Frame rate',exact=True).select_option('50')
    page.get_by_label('World',exact=True).select_option('render-fixture')
    page.screenshot(path=str(args.output/'new-project.png'),full_page=True)
    page.get_by_role('button',name='Create',exact=True).click()
    page.get_by_role('button',name='MP4',exact=True).wait_for()
    page.wait_for_function("document.querySelector('.Render editor [role=alert]') || !document.querySelector('.cinema-loading')",timeout=120000)
    assert not page.locator('.Render editor [role=alert]').count(), page.locator('.Render editor [role=alert]').all_text_contents()
    assert not errors,errors
    page.get_by_role('tab',name='Sequence',exact=True).click()
    assert page.get_by_label('Sequence duration',exact=True).input_value()=='20'
    assert page.get_by_label('Frame rate',exact=True).input_value()=='50'
    page.screenshot(path=str(args.output/'cinematic-dark.png'),full_page=True)
    page.get_by_label('Playhead position',exact=True).fill('3')
    page.get_by_role('button',name='Add keyframe',exact=True).click()
    assert page.get_by_text('4 Keyframes',exact=True).count()==1
    page.get_by_role('menuitem',name='Edit',exact=True).click()
    page.get_by_role('menuitem',name='Undo',exact=False).click()
    assert page.get_by_text('3 Keyframes',exact=True).count()==1
    page.get_by_role('button',name='Edit 3D path',exact=True).click()
    page.wait_for_timeout(400)
    page.screenshot(path=str(args.output/'cinematic-path.png'),full_page=True)
    page.get_by_role('button',name='Scene styling',exact=True).click()
    page.get_by_role('tab',name='Details',exact=True).click()
    assert page.get_by_label('HDR reflections',exact=True).is_visible()
    page.get_by_label('HDR reflections',exact=True).fill('1.5')
    if args.atmosphere:
        page.get_by_role('tab',name='Atmosphere',exact=True).click()
        page.get_by_label('Volumetric clouds',exact=True).check()
        page.get_by_label('Cloud height',exact=True).fill('95')
        page.get_by_label('Cloud cover',exact=True).fill('0.55')
        page.get_by_label('Wind speed',exact=True).fill('15')
        page.get_by_label('Volumetric fog',exact=True).check()
        page.get_by_label('Fog density',exact=True).fill('0.2')
        page.get_by_label('Fog height',exact=True).fill('50')
        page.screenshot(path=str(args.output/'atmosphere-inspector.png'),full_page=True)
    if args.lighting:
        page.get_by_role('tab',name='Lighting',exact=True).click()
        page.get_by_label('Sunlight',exact=True).fill('0')
        page.get_by_label('Ambient light',exact=True).fill('0.1')
        page.get_by_label('Skybox',exact=True).select_option('none')
        page.get_by_label('Sky lighting',exact=True).fill('0')
        page.get_by_label('Enable cinematic light',exact=True).check()
        page.get_by_role('button',name='Fog for visible light beams',exact=True).click()
    page.get_by_role('button',name='Scene styling',exact=True).click()
    sequence=dict(version=1,name='Export test',duration=0.25,fps=24,keys=[
      dict(id='a',time=0,position=[1.8,.8,1.8],target=[0,0,0],fov=45,roll=0,exposure=1,ease='smooth'),
      dict(id='middle',time=.125,position=[0,.5,2],target=[0,.05,0],fov=42,roll=2,exposure=1,ease='smooth'),
      dict(id='b',time=.25,position=[-1.8,.8,1.8],target=[0,.1,0],fov=40,roll=4,exposure=1,ease='smooth')])
    page.locator('.cinema-project input[type=file]').set_input_files(dict(name='test.json',mimeType='application/json',buffer=json.dumps(sequence).encode()))
    page.get_by_role('tab',name='Export',exact=True).click()
    page.get_by_label('GIF size',exact=True).select_option('320')
    page.get_by_role('menuitem',name='File',exact=True).click()
    page.get_by_role('menuitem',name='Animation als GIF export',exact=False).click()
    page.get_by_role('dialog').wait_for()
    page.screenshot(path=str(args.output/'gif-export-dialog.png'),full_page=True)
    page.get_by_role('button',name='Export …',exact=True).click()
    page.wait_for_function("document.querySelector('.cinema-status')?.textContent.includes('Saved:') || document.querySelector('.cinema-status')?.textContent.includes('Error')",timeout=180000)
    assert saved_videos and saved_videos[-1].suffix=='.gif',page.locator('.cinema-status').inner_text()
    with Image.open(saved_videos[-1]) as image:
        assert image.size==(320,180),image.size
        assert image.n_frames>=2,image.n_frames
        assert len(image.convert('RGB').getcolors(1000000))>100,'flat GIF'
        image.seek(image.n_frames-1)
        image.convert('RGB').save(args.output/'gif-last-frame.png')
    page.get_by_label('MP4 resolution',exact=True).select_option('720')
    page.get_by_role('menuitem',name='File',exact=True).click()
    page.get_by_role('menuitem',name='Video als MP4 export',exact=False).click()
    page.get_by_role('button',name='Export …',exact=True).click()
    page.wait_for_function("document.querySelector('.cinema-status')?.textContent.includes('export.mp4') || document.querySelector('.cinema-status')?.textContent.includes('Error')",timeout=180000)
    assert saved_videos[-1].suffix=='.mp4',page.locator('.cinema-status').inner_text()
    assert saved_videos[-1].read_bytes()[4:8]==b'ftyp'
    encoded=base64.b64encode(saved_videos[-1].read_bytes()).decode()
    metadata=page.evaluate('''async b64=>{const video=document.createElement('video');video.src='data:video/mp4;base64,'+b64;await new Promise((r,j)=>{video.onloadedmetadata=r;video.onerror=j;});return {width:video.videoWidth,height:video.videoHeight,duration:video.duration};}''',encoded)
    assert metadata['width']==720 and metadata['height']==406,metadata
    assert .2<=metadata['duration']<=.4,metadata
    page.get_by_role('button',name='WebM',exact=True).click()
    page.get_by_role('button',name='Export …',exact=True).click()
    page.wait_for_function("document.querySelector('.cinema-status')?.textContent.includes('export.webm') || document.querySelector('.cinema-status')?.textContent.includes('Error')",timeout=120000)
    assert saved_videos[-1].suffix=='.webm',page.locator('.cinema-status').inner_text()
    assert saved_videos[-1].read_bytes().startswith(bytes([0x1a,0x45,0xdf,0xa3]))
    webm64=base64.b64encode(saved_videos[-1].read_bytes()).decode()
    webm_metadata=page.evaluate('''async b64=>{const video=document.createElement('video');video.src='data:video/webm;base64,'+b64;await new Promise((r,j)=>{video.onloadeddata=r;video.onerror=()=>j(Error('WebM contains no decodable video frames'));setTimeout(()=>j(Error('WebM decode timeout')),8000);});return {width:video.videoWidth,height:video.videoHeight};}''',webm64)
    assert webm_metadata['width']>0 and webm_metadata['height']>0,webm_metadata
    page.set_viewport_size(dict(width=1240,height=820))
    page.screenshot(path=str(args.output/'cinematic-desktop.png'),full_page=True)
    bounds=page.locator('.cinema-status').bounding_box()
    assert bounds['y']+bounds['height']<=825,bounds
    transport=page.locator('.cinema-transport').bounding_box()
    timeline=page.locator('.cinema-timeline').bounding_box()
    assert transport['y']+transport['height']<=timeline['y']+1,(transport,timeline)
    page.get_by_role('button',name='Settings',exact=True).click()
    page.get_by_role('button',name='Light',exact=True).click()
    page.get_by_role('button',name='Done',exact=True).click()
    page.screenshot(path=str(args.output/'settings-light.png'),full_page=True)
    page.get_by_role('button',name='Cinematic editor',exact=True).click()
    page.wait_for_function("!document.querySelector('.cinema-loading')",timeout=120000)
    page.screenshot(path=str(args.output/'cinematic-light.png'),full_page=True)
    page.get_by_role('button',name='MP4',exact=True).click()
    page.get_by_role('img',name='Current render view',exact=True).wait_for()
    page.screenshot(path=str(args.output/'mp4-export-light.png'),full_page=True)
    page.locator('.export-format-navigation button').filter(has_text='WEBM').click()
    page.screenshot(path=str(args.output/'webm-export-light.png'),full_page=True)
    page.get_by_role('button',name='Cancel',exact=True).click()
    assert not errors,errors
    print(json.dumps(dict(passed=True,browserErrors=errors,mp4=metadata,webm=webm_metadata,files=[str(p) for p in saved_videos]),indent=2),flush=True)
    browser.close()
