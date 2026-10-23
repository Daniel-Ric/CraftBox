"""Save, transfer, reload and import a render style through the actual editor."""
from pathlib import Path
exec(Path('tools/verify-atmosphere.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
original_bootstrap = json.dumps(bootstrap)
state['worlds'].append(dict(id='other-world',name='Other world',chunks=48,state='finished'))
init = init.replace(original_bootstrap,json.dumps(bootstrap))
init += "window.go.desktop.App.SaveCinematicFile=async(name,kind,data)=>fetch('/__style-export',{method:'POST',body:JSON.stringify({name,data,kind})}).then(r=>r.json());"
out=Path('tmp/cloud-bodies/styles');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
    browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
    page=browser.new_page(viewport=dict(width=1182,height=780));page.set_default_timeout(90000)
    errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
    page.add_init_script(init)
    page.route('**/__page?*',lambda r:r.fulfill(status=200,content_type='application/json',body=json.dumps(pages[indices[parse_qs(urlparse(r.request.url).query)['cursor'][0]]])))
    exported=[]
    def save(r):
        data=r.request.post_data_json
        raw=base64.b64decode(data['data'].split(',')[-1]);exported.append(raw)
        (out/'palace.render-style.json').write_bytes(raw)
        r.fulfill(status=200,content_type='application/json',body=json.dumps('palace.render-style.json'))
    page.route('**/__style-export',save)
    page.goto('http://127.0.0.1:5204',wait_until='networkidle')
    page.get_by_role('button',name='Open workspace',exact=True).click()
    page.get_by_role('button',name='Render editor',exact=True).click()
    page.get_by_role('tab',name='Details',exact=True).click()
    page.get_by_role('button',name='Fast',exact=True).click()
    page.get_by_role('tab',name='Lighting',exact=True).click()
    page.get_by_label('Exposure Value',exact=True).fill('1.25');page.get_by_label('Exposure Value',exact=True).press('Tab')
    page.get_by_role('tab',name='Atmosphere',exact=True).click()
    page.get_by_label('Cloud height reference',exact=True).select_option('bottom')
    page.get_by_label('Cloud height',exact=True).fill('-20');page.get_by_label('Cloud height',exact=True).press('Tab')
    launcher=page.get_by_role('button',name='Save / load presets …',exact=True)
    launcher.click()
    page.get_by_label('Preset name').fill('Palast-Clouds')
    page.get_by_role('button',name='Save current style',exact=True).click()
    page.get_by_text('Current scene style saved.',exact=True).wait_for()
    page.screenshot(path=str(out/'style-dialog.png'))
    page.get_by_role('button',name='Export JSON …',exact=True).click()
    page.get_by_text('Saved: palace.render-style.json',exact=True).wait_for()
    assert len(exported)==1
    settings=json.loads(exported[0])['settings']
    assert settings['exposure']==1.25 and settings['atmosphere']['cloudHeight']==-20 and settings['atmosphere']['cloudReference']=='bottom'
    page.get_by_role('button',name='Close',exact=True).click()
    page.get_by_label('Map to render',exact=True).select_option('other-world')
    page.get_by_role('tab',name='Lighting',exact=True).click()
    page.get_by_label('Exposure Value',exact=True).fill('0.7');page.get_by_label('Exposure Value',exact=True).press('Tab')
    launcher.click();page.get_by_role('button',name='Palast-Clouds',exact=True).click()
    page.get_by_role('button',name='Apply',exact=True).click()
    assert page.get_by_label('Map to render',exact=True).input_value()=='other-world'
    assert page.get_by_label('Exposure Value',exact=True).input_value()=='1.25'
    page.reload(wait_until='networkidle')
    page.screenshot(path=str(out/'after-reload.png'))
    welcome=page.get_by_role('button',name='Open workspace',exact=True)
    if welcome.is_visible(): welcome.click()

    page.get_by_role('button',name='Render editor',exact=True).click()
    launcher=page.get_by_role('button',name='Save / load presets …',exact=True)
    launcher.click();page.get_by_role('button',name='Palast-Clouds',exact=True).click()
    page.get_by_role('button',name='Remove',exact=True).click()
    page.get_by_text('No saved presets yet.',exact=True).wait_for()
    page.get_by_label('Import render preset',exact=True).set_input_files(str(out/'palace.render-style.json'))
    page.get_by_text('Imported. Select “Apply” to use it for this world.',exact=True).wait_for()
    page.get_by_role('button',name='Apply',exact=True).click()
    page.get_by_role('tab',name='Atmosphere',exact=True).click()
    assert page.get_by_label('Cloud height',exact=True).input_value()=='-20'
    assert page.get_by_label('Cloud height reference',exact=True).input_value()=='bottom'
    assert not errors, errors
    browser.close()
print('Render styles: save, JSON export, second world transfer, IndexedDB reload, delete and JSON import passed')
