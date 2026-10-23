"""Exercise task-specific studio windows and libraries against a controlled backend."""
import base64
import json
from pathlib import Path
from playwright.sync_api import sync_playwright

out=Path('tmp/studio-redesign/current');out.mkdir(parents=True,exist_ok=True)
icon='data:image/png;base64,'+base64.b64encode(Path('assets/craft-box-icon-refined.png').read_bytes()).decode()
state=dict(revision=1,command='worlds',running=False,stopping=False,connection=0,worlds=[dict(id='palace',name='CubeCraft Palace',chunks=48,entities=824,state='finished',path='D:/worlds/Palace',spawn=[32,70,16]),dict(id='island',name='Tropical Island',chunks=256,entities=128,state='finished',path='D:/worlds/Island')],packs=[dict(id='pack',name='Palace Resources',version='1.2.0',done=True,path='D:/packs',size=18432100,received=18432100,icon=icon),dict(id='pack2',name='Island Textures',version='2.0.1',done=False,size=8211100,received=4211100)],skins=[dict(name='Builder',image=icon),dict(name='Architect',image=icon)],player=[0,70,0],hasPlayer=False,loggedIn=False,account='',error='',authCode='',updateProgress=0)
bootstrap=dict(version='1.59.0-studio-verification',minecraft='1.26.52',commands=[],state=state)
init='''window.studioEvents={};window.nativeCalls=[];
window.runtime={EventsOn:(name,fn)=>{window.studioEvents[name]=fn;return()=>{};}};
window.go={desktop:{App:{GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[{id:1,time:'2026-10-19T10:00:00Z',level:'info',message:'Arbeitsbereich bereit',source:'studio'},{id:2,time:'2026-10-19T10:00:02Z',level:'warning',message:'Welt enthält noch nicht besuchte Chunks',source:'capture'}],ListSources:async(kind)=>[{id:'test',name:'Palace Server',description:'Minecraft Bedrock',address:'localhost:19132',available:true},{id:'offline',name:'Garden Realm',description:'Derzeit offline',address:'',available:false}],OpenResultFolder:async(p)=>{window.nativeCalls.push(['folder',p]);},SaveRenderPNG:async(n,d)=>{window.nativeCalls.push(['png',n,d.slice(0,22)]);return 'D:/exports/'+n+'.png';},CheckUpdate:async()=>{throw new Error('Update-Server nicht erreichbar: HTTP 404');},DismissError:async()=>{},CancelLogin:async()=>{},Login:async()=>{},Logout:async()=>{},SwitchAccount:async()=>{},OpenURL:async()=>{},OpenMapWindow:async()=>{}}}};'''.replace('BOOTSTRAP',json.dumps(bootstrap))
with sync_playwright() as p:
    browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
    page=browser.new_page(viewport=dict(width=1440,height=960))
    errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
    page.add_init_script(init);page.goto('http://127.0.0.1:5202',wait_until='networkidle')
    shots=[]
    def shot(name):
        page.wait_for_timeout(150)
        for dialog in page.get_by_role('dialog').all():
            if not dialog.is_visible():continue
            b=dialog.bounding_box()
            assert b['x']>=6 and b['y']>=6 and b['x']+b['width']<=page.viewport_size['width']-5 and b['y']+b['height']<=page.viewport_size['height']-5,(name,b)
            assert not dialog.evaluate('(e)=>e.scrollHeight>e.clientHeight+2'),(name,'outer dialog scrolls')
        assert page.evaluate('document.documentElement.scrollWidth<=innerWidth'),name
        page.screenshot(path=str(out/(name+'.png')),full_page=False);shots.append(name)
    shot('welcome-dark')
    page.get_by_role('button',name='Create new',exact=True).click()
    shot('new-render-dark')
    page.get_by_role('tab',name='Camera sequence',exact=True).click();shot('new-cinematic-dark')
    page.get_by_role('tab',name='Capture',exact=True).click();shot('new-capture-dark')
    page.get_by_role('button',name='Cancel',exact=True).click()
    page.get_by_role('button',name='Open workspace',exact=True).click()
    page.get_by_role('button',name='CubeCraft Palace',exact=True).click();shot('worlds-detail-dark')
    page.get_by_role('button',name='Grid view',exact=True).click();shot('worlds-grid-dark')
    page.get_by_role('button',name='Resource packs',exact=True).click()
    page.get_by_role('button',name='Palace Resources',exact=True).click();shot('packs-detail-dark')
    page.get_by_role('button',name='Open pack folder',exact=True).click()
    assert page.evaluate("nativeCalls.some(c=>c[0]==='folder'&&c[1]==='D:/packs')")
    page.get_by_role('button',name='Player skins',exact=True).click()
    page.get_by_role('button',name='Builder',exact=True).click();shot('skins-detail-dark')
    page.get_by_role('button',name='Save skin as PNG',exact=True).click()
    assert page.evaluate("nativeCalls.some(c=>c[0]==='png'&&c[1]==='Builder')")
    page.get_by_role('button',name='Settings',exact=True).click();shot('preferences-dark')
    for name in ['Light','Mid-gray','Charcoal','Dark']:
        page.get_by_role('button',name=name,exact=True).click();shot('preferences-'+name)
    page.get_by_role('tab',name='Workspace',exact=True).click();shot('preferences-workspace')
    page.get_by_role('tab',name='Version & updates',exact=True).click();shot('preferences-version')
    page.get_by_role('button',name='Check for updates',exact=True).click()
    page.get_by_text('Show technical details',exact=True).click();shot('error-details')
    page.get_by_role('button',name='Close',exact=True).click()
    page.get_by_role('button',name='Done',exact=True).click()
    page.get_by_role('button',name='Worlds',exact=True).click()
    page.get_by_role('button',name='Realms',exact=True).click();shot('source-picker')
    page.get_by_role('tab',name='Featured servers',exact=True).click()
    page.get_by_role('button',name='Cancel',exact=True).click()
    page.get_by_role('button',name='Not signed in – Sign in with Microsoft',exact=True).click();shot('account-signed-out')
    page.keyboard.press('Escape')
    page.keyboard.press('Control+k');page.get_by_label('Search commands').fill('Console');shot('command-search')
    page.get_by_role('button',name='Window Toggle console',exact=True).click()
    page.get_by_label('Search logs',exact=True).fill('Ready');shot('console-filtered')
    handle=page.get_by_role('separator',name='Console height',exact=True)
    before=int(handle.get_attribute('aria-valuenow'));handle.focus();page.keyboard.press('ArrowUp')
    assert int(handle.get_attribute('aria-valuenow'))==before+20
    page.get_by_role('button',name='Close console',exact=True).click()
    page.keyboard.press('F1');shot('help');page.keyboard.press('Escape')
    page.get_by_role('menuitem',name='Help',exact=True).click()
    page.get_by_role('menuitem',name='About CraftBox',exact=True).click();shot('about');page.keyboard.press('Escape')
    page.get_by_role('button',name='Not signed in – Sign in with Microsoft',exact=True).click()
    signed_in={**state,'revision':10,'loggedIn':True,'account':'Studio Builder'}
    page.evaluate("s=>window.studioEvents['desktop:state'](s)",signed_in);shot('account-signed-in')
    device={**signed_in,'revision':11,'authCode':'ABCD-1234','authURI':'https://www.microsoft.com/link','authenticating':True}
    page.evaluate("s=>window.studioEvents['desktop:state'](s)",device);shot('account-device-code')
    page.evaluate("s=>window.studioEvents['desktop:state'](s)",{**state,'revision':12})
    page.keyboard.press('Escape')
    empty={**state,'revision':13,'worlds':[],'packs':[],'skins':[]}
    page.evaluate("s=>window.studioEvents['desktop:state'](s)",empty)
    for theme in ['Dark interface','Light interface']:
        page.get_by_role('menuitem',name='View',exact=True).click()
        page.get_by_role('menuitem',name=theme,exact=True).click()
        for workspace in ['Worlds','Resource packs','Player skins']:
            page.get_by_role('button',name=workspace,exact=True).click();shot('empty-'+workspace+'-'+theme)
    page.get_by_role('menuitem',name='View',exact=True).click();page.get_by_role('menuitem',name='Dark interface',exact=True).click()
    page.evaluate("s=>window.studioEvents['desktop:state'](s)",{**state,'revision':14})
    for width,height in [(1182,695),(1024,640),(760,560)]:
        page.set_viewport_size(dict(width=width,height=height))
        page.keyboard.press('Control+n')
        for mode in ['Still render','Camera sequence','Capture']:
            page.get_by_role('tab',name=mode,exact=True).click()
            shot(f'new-{mode}-{width}')
            assert page.get_by_role('button',name='Create',exact=True).is_visible()
        page.get_by_role('button',name='Cancel',exact=True).click()
    page.set_viewport_size(dict(width=1440,height=960))
    for menu in ['File','Scene','Sequence','Window']:
        page.get_by_role('menuitem',name=menu,exact=True).click();shot('menu-'+menu);page.keyboard.press('Escape')
    page.get_by_role('menuitem',name='View',exact=True).click();page.get_by_role('menuitem',name='Light interface',exact=True).click()
    for workspace,item in [('Worlds','CubeCraft Palace'),('Resource packs','Palace Resources'),('Player skins','Builder')]:
        page.get_by_role('button',name=workspace,exact=True).click();page.get_by_role('button',name=item,exact=True).click();shot('populated-light-'+workspace)
    page.get_by_role('button',name='Start',exact=True).click();shot('welcome-light')
    page.get_by_role('button',name='Create new',exact=True).click()
    for mode in ['Still render','Camera sequence','Capture']:
        page.get_by_role('tab',name=mode,exact=True).click();shot('new-light-'+mode)
    page.get_by_role('button',name='Cancel',exact=True).click();page.get_by_role('button',name='Open workspace',exact=True).click()
    page.get_by_role('button',name='Settings',exact=True).click();shot('preferences-light')
    page.get_by_role('tab',name='Version & updates',exact=True).click();page.get_by_role('button',name='Check for updates',exact=True).click();shot('error-light')
    page.get_by_role('button',name='Close',exact=True).click();page.get_by_role('button',name='Done',exact=True).click()
    page.keyboard.press('F1')
    for topic in ['Capture','Still renders','Camera sequences','Keyboard shortcuts']:
        page.get_by_role('tab',name=topic,exact=True).click();shot('help-light-'+topic)
    page.get_by_role('button',name='Close',exact=True).click()
    page.get_by_role('menuitem',name='Help',exact=True).click();page.get_by_role('menuitem',name='About CraftBox',exact=True).click();shot('about-light');page.keyboard.press('Escape')
    page.get_by_role('button',name='Not signed in – Sign in with Microsoft',exact=True).click();shot('account-light');page.keyboard.press('Escape')
    page.get_by_role('button',name='Realms',exact=True).click();shot('source-light');page.keyboard.press('Escape')
    page.evaluate('window.originalSources=window.go.desktop.App.ListSources;window.go.desktop.App.ListSources=async()=>[]')
    page.get_by_role('button',name='Realms',exact=True).click();page.get_by_text('No sources are currently available here.',exact=True).wait_for();shot('source-empty-light');page.keyboard.press('Escape')
    page.evaluate('()=>{window.go.desktop.App.ListSources=()=>new Promise(()=>{});}')
    page.get_by_role('button',name='Realms',exact=True).click();shot('source-loading-light');page.keyboard.press('Escape')
    page.evaluate('window.go.desktop.App.ListSources=window.originalSources')
    map_bootstrap={**bootstrap,'mapWindow':True,'theme':'dark','state':{**state,'activeWorldId':'palace','running':True}}
    map_page=browser.new_page(viewport=dict(width=1440,height=960))
    map_page.on('pageerror',lambda e:errors.append(str(e)))
    map_page.add_init_script(init.replace(json.dumps(bootstrap),json.dumps(map_bootstrap)))
    map_page.goto('http://127.0.0.1:5202',wait_until='networkidle')
    map_page.get_by_role('region',name='World maps',exact=True).wait_for()
    original=page;page=map_page
    shot('map-2d-dark')
    page.get_by_role('button',name='Select region',exact=True).click();shot('map-region-dark')
    page.get_by_role('button',name='Cancel',exact=True).click()
    page.get_by_role('button',name='Spawn point',exact=True).click();shot('map-spawn-dark')
    page.get_by_role('button',name='Cancel',exact=True).click()
    page.get_by_role('tab',name='CubeCraft Palace · 1',exact=True).dblclick();shot('map-rename-dark')
    page.get_by_role('button',name='Cancel',exact=True).click()
    page.evaluate("window.studioEvents['desktop:theme']('light')");shot('map-2d-light')
    page.get_by_role('button',name='Spawn point',exact=True).click();shot('map-spawn-light');page.get_by_role('button',name='Cancel',exact=True).click()
    page.get_by_role('button',name='Select region',exact=True).click();shot('map-region-light');page.get_by_role('button',name='Cancel',exact=True).click()
    page.get_by_role('tab',name='CubeCraft Palace · 1',exact=True).dblclick();shot('map-rename-light');page.get_by_role('button',name='Cancel',exact=True).click()
    page=original
    assert not errors,errors
    (out/'verification.json').write_text(json.dumps(dict(passed=True,shots=shots,browserErrors=errors),indent=2),encoding='utf-8')
    browser.close()
print('PASS studio windows, libraries, commands, themes, source/account/error dialogs, console and document sizes',len(shots))
