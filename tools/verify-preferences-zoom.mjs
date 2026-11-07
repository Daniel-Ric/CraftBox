                                                                           
import { spawn } from 'node:child_process';
import { createWriteStream } from 'node:fs';
import { mkdtemp, rm, writeFile, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const profile=await mkdtemp(join(tmpdir(),'craftbox-block-light-'));
const browser=spawn('C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',[
 '--headless=new','--no-sandbox','--no-first-run','--no-default-browser-check','--use-angle=swiftshader',
 '--enable-unsafe-swiftshader','--remote-debugging-port=9334',`--user-data-dir=${profile}`,'http://127.0.0.1:5204'
],{windowsHide:true,stdio:['ignore','ignore','pipe']});
const browserLog=createWriteStream('tools/light-audit/shell-browser.log');
browser.stderr.pipe(browserLog);
let ws;
try {
 let tabs;for(let i=0;i<100;i++){try{tabs=await (await fetch('http://127.0.0.1:9334/json')).json();if(tabs.some(t=>t.type==='page'))break;}catch{}await new Promise(r=>setTimeout(r,100));}
 await new Promise(r=>setTimeout(r,2000));tabs=await (await fetch('http://127.0.0.1:9334/json')).json();
 const tab=tabs.find(t=>t.type==='page'&&t.url.includes('5204'))??tabs.find(t=>t.type==='page');ws=new WebSocket(tab.webSocketDebuggerUrl);
 await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});ws.addEventListener('error',reject,{once:true});});
 let id=0;const pending=new Map();
 ws.addEventListener('close',e=>{for(const p of pending.values())p.reject(new Error('DevTools closed '+e.code+' '+e.reason));});
 ws.addEventListener('message',({data})=>{const m=JSON.parse(data);if(m.id){const p=pending.get(m.id);pending.delete(m.id);m.error?p.reject(new Error(JSON.stringify(m.error))):p.resolve(m.result);}});
 const send=(method,params={})=>new Promise((resolve,reject)=>{const key=++id;pending.set(key,{resolve,reject});ws.send(JSON.stringify({id:key,method,params}));});
 await send('Page.enable');await send('Page.navigate',{url:'http://127.0.0.1:5204'});
 await new Promise(r=>setTimeout(r,2000));
 await send('Emulation.setDeviceMetricsOverride',{width:760,height:600,deviceScaleFactor:1,mobile:false});
 const result=await send('Runtime.evaluate',{awaitPromise:true,returnByValue:true,expression:`(async()=>{
const source=await(await fetch('/src/DesktopChrome.tsx')).text();const suffix=source.split('"/node_modules/.vite/deps/react.js')[1].split('"')[0];const React=(await import('/node_modules/.vite/deps/react.js'+suffix)).default;const {createRoot}=(await import('/node_modules/.vite/deps/react-dom_client.js')).default;const {default:Preferences}=await import('/src/PreferencesDialog.tsx');
localStorage.clear();const {saveWorkspacePreferences,preferenceDefaults}=await import('/src/workspace-preferences.ts');saveWorkspacePreferences({...preferenceDefaults,uiScale:125});document.documentElement.classList.add('dark');document.body.innerHTML='<div id="preferences-check" class="desktop-workspace"></div>';
window.go={desktop:{App:{OpenURL:async()=>{}}}};
const info={available:true,currentVersion:'1.63.1',checkedAt:new Date().toISOString(),channel:'stable',release:{version:'1.64.0',name:'CraftBox 1.64',notes:'Improved capture and editor performance. Fixed rendering issues.',releaseURL:'https://github.com/Daniel-Ric/CraftBox/releases',publishedAt:'2026-11-01T10:00:00Z',prerelease:false,assetName:'CraftBox-gui-windows-amd64-1.64.0.exe',size:84937216,installable:true}};
createRoot(document.getElementById('preferences-check')).render(React.createElement(Preferences,{open:true,onOpenChange:()=>{},theme:'dark',onTheme:()=>{},version:'1.63.1',minecraft:'1.26.60',updateInfo:info,updateBusy:false,onCheckUpdate:()=>{},onInstallUpdate:()=>{}}));
await new Promise(r=>setTimeout(r,300));const dialog=document.querySelector('[role=dialog]');const nav=[...dialog.querySelectorAll('[role=tab]')];if(nav.length!==6)throw Error('Missing categories');for(const button of nav){button.click();await new Promise(r=>setTimeout(r,70));if(!dialog.querySelector('[role=tabpanel] h2'))throw Error('Missing panel');}
const panel=dialog.querySelector('.preferences-content');const rect=dialog.getBoundingClientRect();if(rect.top<0||rect.bottom>innerHeight||rect.width>innerWidth)throw Error('Dialog clipped '+JSON.stringify(rect));
if(!dialog.textContent.includes('CraftBox 1.64')||!dialog.textContent.includes('Last checked'))throw Error('Release metadata missing');const summary=dialog.querySelector('summary');summary.click();panel.scrollTop=panel.scrollHeight;return {categories:nav.map(b=>b.textContent),dialog:{width:rect.width,height:rect.height},scrolls:panel.scrollHeight>panel.clientHeight};})()`});
if(result.exceptionDetails)throw Error(JSON.stringify(result.exceptionDetails));console.log(JSON.stringify(result.result.value));const shot=await send('Page.captureScreenshot',{format:'png'});await writeFile('tools/light-audit/preferences-zoom.png',Buffer.from(shot.data,'base64'));
} finally {ws?.close();browser.kill();await new Promise(r=>setTimeout(r,500));await rm(profile,{recursive:true,force:true}).catch(()=>{});}

