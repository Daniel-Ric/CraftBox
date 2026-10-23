"""Verify modal start, return navigation and compact document setup in Edge."""
import json
from pathlib import Path
from playwright.sync_api import sync_playwright
out=Path('tmp/welcome-dialog');out.mkdir(parents=True,exist_ok=True)
state=dict(revision=1,command='worlds',running=False,connection=0,worlds=[],packs=[],skins=[],player=[0,0,0],hasPlayer=False,loggedIn=False)
bootstrap=dict(version='Welcome verification',minecraft='1.26.52',commands=[],state=state)
init='''window.runtime={EventsOn:()=>()=>{}};window.go={desktop:{App:{GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[],ListSources:async()=>[],SetTheme:async()=>{}}}};'''.replace('BOOTSTRAP',json.dumps(bootstrap))
with sync_playwright() as p:
    browser=p.chromium.launch(channel='msedge',headless=True)
    page=browser.new_page(viewport=dict(width=1182,height=695));errors=[]
    page.on('pageerror',lambda e:errors.append(str(e)))
    page.add_init_script(init);page.goto('http://127.0.0.1:5202',wait_until='networkidle')
    welcome=page.get_by_role('dialog',name='Welcome to CraftBox',exact=True)
    welcome.wait_for();page.wait_for_timeout(200)
    page.screenshot(path=str(out/'welcome.png'))
    page.get_by_role('button',name='Create new',exact=True).click()
    document=page.get_by_role('dialog',name='New project',exact=True)
    document.wait_for()
    for width,height in [(1182,695),(1280,720),(1024,640)]:
        page.set_viewport_size(dict(width=width,height=height));page.wait_for_timeout(200)
        for mode in ['Still render','Camera sequence','Capture']:
            page.get_by_role('tab',name=mode,exact=True).click();page.wait_for_timeout(100)
            metrics=document.evaluate('(e)=>({height:e.clientHeight,scroll:e.scrollHeight,top:e.getBoundingClientRect().top,bottom:e.getBoundingClientRect().bottom})')
            assert metrics['scroll']<=metrics['height']+1,(width,height,mode,metrics)
            assert metrics['top']>=8 and metrics['bottom']<=height-8,(width,height,mode,metrics)
            assert page.get_by_role('button',name='Create',exact=True).is_in_viewport() if hasattr(page.get_by_role('button',name='Create',exact=True),'is_in_viewport') else page.get_by_role('button',name='Create',exact=True).is_visible()
            page.screenshot(path=str(out/f'new-{mode}-{width}x{height}.png'))
    page.get_by_role('button',name='Cancel',exact=True).click();welcome.wait_for()
    page.get_by_role('button',name='Open workspace',exact=True).click()
    page.get_by_label('Server address',exact=True).fill('example.org:19132')
    page.get_by_role('button',name='Start',exact=True).click();welcome.wait_for()
    page.keyboard.press('Escape');welcome.wait_for(state='hidden')
    assert page.get_by_label('Server address',exact=True).input_value()=='example.org:19132'
    page.get_by_role('button',name='Realms',exact=True).click()
    source=page.get_by_role('dialog',name='Select source',exact=True);source.wait_for()
    page.get_by_role('button',name='Featured servers',exact=True).click()
    selected=page.locator('.source-dialog .source-tabs .bg-primary')
    assert selected.evaluate('(e)=>getComputedStyle(e).borderTopLeftRadius')=='0px'
    assert selected.evaluate('(e)=>getComputedStyle(e).borderBottomWidth')=='2px'
    page.wait_for_timeout(150);page.screenshot(path=str(out/'source-tabs.png'))
    page.keyboard.press('Escape')
    page.set_viewport_size(dict(width=760,height=560))
    page.get_by_role('button',name='Start',exact=True).click();welcome.wait_for()
    b=welcome.bounding_box();assert b['y']>=8 and b['y']+b['height']<=552,b
    assert not errors,errors
    browser.close()
print('PASS modal welcome, return from New Project, preserved workspace, flat source tabs, no document scrolling at three desktop sizes, compact welcome; no browser errors')
