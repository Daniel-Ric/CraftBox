"""Keep the empty cinematic workspace usable before its lazy studio loads."""
from pathlib import Path
exec(Path('tools/verify-studio-redesign.py').read_text(encoding='utf-8').split('with sync_playwright() as p:')[0])
original_bootstrap = json.dumps(bootstrap)
state['worlds'] = []
init = init.replace(original_bootstrap, json.dumps({**bootstrap, 'state': state}))
out = Path('tmp/cloud-bodies/empty'); out.mkdir(parents=True, exist_ok=True)
with sync_playwright() as p:
    browser = p.chromium.launch(channel='msedge', headless=True)
    page = browser.new_page(viewport=dict(width=1440, height=960))
    page.add_init_script(init)
    page.goto('http://127.0.0.1:5204', wait_until='networkidle')
    page.get_by_role('button', name='Open workspace', exact=True).click()
    page.get_by_role('button', name='Cinematic editor', exact=True).click()
    for width, height in [(1440,960),(1182,695)]:
        page.set_viewport_size(dict(width=width,height=height))
        for console in [False, True]:
            if console:
                page.keyboard.press('Control+k')
                page.get_by_label('Search commands').fill('Console')
                page.get_by_role('button',name='Window Toggle console',exact=True).click()
            for lighting in [False, True]:
                if lighting: page.get_by_role('button',name='Scene styling',exact=True).click()
                page.wait_for_timeout(150)
                stage = page.locator('.render-stage').bounding_box()
                inspector = page.get_by_role('complementary',name='Render settings')
                assert inspector.is_visible() == lighting, (width, console, lighting, 'inspector visibility')
                assert stage['height'] >= 160, (width,console,lighting,stage)
                if lighting:
                    box = inspector.bounding_box()
                    assert box['x'] >= stage['x'] + stage['width'] - 1, (stage,box)
                    assert abs(box['y']-stage['y']) < 2, (stage,box)
                page.screenshot(path=str(out/f'{width}-console-{console}-lighting-{lighting}.png'))
            page.get_by_role('button',name='Scene styling',exact=True).click()
            if console: page.get_by_role('button',name='Close console',exact=True).click()
    browser.close()
print('Empty cinematic layout: 8 states passed')
