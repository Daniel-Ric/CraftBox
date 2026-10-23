"""Read-only visual inventory of the user's local Cape Editor reference."""
import sys
sys.stdout.reconfigure(encoding='utf-8')
from pathlib import Path
from playwright.sync_api import sync_playwright
out=Path('tmp/studio-redesign/reference');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
    browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
    page=browser.new_page(viewport=dict(width=1440,height=960))
    page.goto('http://127.0.0.1:5203',wait_until='networkidle')
    page.wait_for_timeout(700)
    page.screenshot(path=str(out/'cape-initial.png'))
    print(page.locator('body').inner_text()[:6500],flush=True)
    print('DIALOGS',page.get_by_role('dialog').all_text_contents(),flush=True)
    print('BUTTONS',page.get_by_role('button').all_text_contents()[:50],flush=True)
    page.get_by_role('button',name='Create',exact=True).click()
    page.wait_for_timeout(250)
    page.screenshot(path=str(out/'cape-workspace.png'))
    page.get_by_role('button',name='Export',exact=True).click()
    page.wait_for_timeout(200)
    page.screenshot(path=str(out/'cape-export.png'))
    print('EXPORT',page.get_by_role('dialog').all_text_contents(),flush=True)
    browser.close()
