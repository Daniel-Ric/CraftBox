"""Verify application menus, workspace navigation and dialog layouts in Edge."""
import json
from pathlib import Path
from playwright.sync_api import sync_playwright

out=Path('tmp/desktop-chrome');out.mkdir(parents=True,exist_ok=True)
state=dict(revision=1,command='worlds',running=False,connection=0,worlds=[dict(id='fixture',name='CubeCraft Palace',chunks=48,state='finished',path='D:/worlds/Palace',entities=824)],packs=[dict(id='pack',name='Palace Resources',version='1.2.0',done=True,path='D:/packs',size=18432100,received=18432100)],skins=[dict(name='Builder',image=''),dict(name='Architect',image='')],player=[0,0,0],hasPlayer=False,loggedIn=False)
bootstrap=dict(version='Studio verification',minecraft='1.26.52',commands=[],state=state)
init='''window.runtime={EventsOn:()=>()=>{}};window.go={desktop:{App:{GetBootstrap:async()=>(BOOTSTRAP),GetMapTiles:async()=>[],GetLogs:async()=>[{id:1,time:'14:04:00',level:'info',message:'Arbeitsbereich bereit',source:'studio'}],ListSources:async()=>[{id:'test',name:'Palace Server',description:'Minecraft Bedrock',address:'localhost:19132',available:true}],SetTheme:async()=>{},OpenResultFolder:async()=>{},CheckUpdate:async()=>{}}}};'''.replace('BOOTSTRAP',json.dumps(bootstrap))
with sync_playwright() as p:
    browser=p.chromium.launch(channel='msedge',headless=True)
    page=browser.new_page(viewport=dict(width=1440,height=960))
    errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
    page.add_init_script(init);page.goto('http://127.0.0.1:5202',wait_until='networkidle')
    assert page.locator('.sidebar').count()==0
    assert page.get_by_role('menubar',name='Application menu').is_visible()
    def shot(name):
        page.wait_for_timeout(250)
        page.screenshot(path=str(out/(name+'.png')),full_page=True)
    shot('start-dark')
    page.get_by_role('button',name='Worlds',exact=True).click()
    shot('worlds-dark')
    page.get_by_role('menuitem',name='Window',exact=True).click()
    page.get_by_role('menuitem',name='Resource packs',exact=False).click()
    page.get_by_role('heading',name='Resource packs',exact=True).wait_for();shot('packs-dark')
    page.keyboard.press('Control+Shift+Digit3')
    page.wait_for_function("document.querySelector('.workspace-tabs button[aria-current=page]').textContent==='Player skins'")
    shot('skins-dark')
    page.get_by_role('button',name='Settings',exact=True).click();shot('settings-dark')
    page.get_by_role('button',name='Mid-gray',exact=True).click();shot('settings-gray')
    page.get_by_role('button',name='Charcoal',exact=True).click();shot('settings-graphite')
    page.get_by_role('button',name='Done',exact=True).click()
    page.get_by_role('menuitem',name='View',exact=True).click()
    page.get_by_role('menuitem',name='Light interface',exact=True).click();shot('settings-light')
    page.get_by_role('menuitem',name='View',exact=True).click()
    page.get_by_role('menuitem',name='Dark interface',exact=True).click()
    page.get_by_role('button',name='Worlds',exact=True).click()
    page.get_by_role('button',name='Open logs',exact=True).click()
    assert page.get_by_role('textbox',name='Search logs').is_visible();shot('logs')
    page.get_by_role('button',name='Not signed in – Sign in with Microsoft',exact=True).click()
    page.get_by_role('dialog').wait_for();shot('account')
    page.keyboard.press('Escape')
    page.get_by_role('button',name='Realms',exact=True).click()
    page.get_by_role('dialog').wait_for();shot('sources')
    page.keyboard.press('Escape')
    page.keyboard.press('F1');page.get_by_role('dialog').wait_for();shot('help')
    page.set_viewport_size(dict(width=800,height=420))
    page.wait_for_timeout(250)
    dialog=page.get_by_role('dialog');b=dialog.bounding_box()
    assert b['y']>=10 and b['y']+b['height']<=410,b
    assert dialog.evaluate('(e)=>e.scrollHeight>e.clientHeight'),'short dialog does not scroll'
    page.get_by_role('button',name='Close',exact=True).click()
    page.set_viewport_size(dict(width=720,height=650))
    page.get_by_role('button',name='Player skins',exact=True).click();shot('compact')
    assert page.evaluate('document.documentElement.scrollWidth<=innerWidth'),'horizontal page overflow'
    assert not errors,errors
    browser.close()
print('PASS menus, physical workspace shortcut, five core views, themes, logs, source/account/help dialogs and compact/short windows')
