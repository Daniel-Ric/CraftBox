"""Check the capture action stays reachable while settings and console scroll."""
from pathlib import Path
import json

fixture = {}
exec(Path('tools/verify-studio-redesign.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0], fixture)
bootstrap = fixture['bootstrap']
flags = ['capture', 'client-cache', 'void', 'image', 'save-entities', 'save-custom-entities', 'save-inventories']
options = [dict(flag='address', type='connectInfo', name='Server', default='localhost:19132')]
options += [dict(flag=f, type='bool', name=f, default='true', description='Aufnahmeoption') for f in flags]
bootstrap['commands'] = [dict(name='worlds', options=options)]
original = json.dumps(fixture['bootstrap'])

init = fixture['init'].replace(json.dumps({**bootstrap, 'commands': []}), original)
out = Path('tmp/studio-redesign/capture-footer')
out.mkdir(parents=True, exist_ok=True)
with fixture['sync_playwright']() as p:
    browser = p.chromium.launch(channel='msedge', headless=True)
    page = browser.new_page()
    errors = []
    page.on('pageerror', lambda e: errors.append(str(e)))
    page.add_init_script(init)
    page.goto('http://127.0.0.1:5202', wait_until='networkidle')
    page.get_by_role('button', name='Open workspace', exact=True).click()
    checks = []
    for width, height in [(1440, 960), (1182, 695), (1024, 640)]:
        page.set_viewport_size(dict(width=width, height=height))
        for console in [False, True]:
            if console:
                page.keyboard.press('Control+k')
                page.get_by_label('Search commands').fill('Console')
                page.get_by_role('button', name='Window Toggle console', exact=True).click()
            button = page.get_by_role('button', name='Start download', exact=True)
            bounds = button.bounding_box()
            workspace = page.locator('.asset-workspace').bounding_box()
            assert bounds and bounds['y'] >= workspace['y']
            assert bounds['y'] + bounds['height'] <= workspace['y'] + workspace['height'], (width, height, console, bounds, workspace)
            page.screenshot(path=str(out / f'{width}-console-{console}.png'))
            checks.append(dict(width=width, height=height, console=console, button=bounds))
            if console:
                page.locator('.logs-heading > button').click()
    assert not errors, errors
    (out / 'verification.json').write_text(json.dumps(checks, indent=2), encoding='utf-8')
    browser.close()
    print('Capture footer: six viewport/console states passed; no browser errors.')
