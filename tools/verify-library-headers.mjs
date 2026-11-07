                                                                           
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
 await send('Emulation.setDeviceMetricsOverride',{width:1280,height:820,deviceScaleFactor:1,mobile:false});
 const result=await send('Runtime.evaluate',{awaitPromise:true,returnByValue:true,expression:`(async()=>{
const source=await (await fetch('/src/DesktopChrome.tsx')).text();const suffix=source.split('"/node_modules/.vite/deps/react.js')[1].split('"')[0];
const React=(await import('/node_modules/.vite/deps/react.js'+suffix)).default;const {createRoot}=(await import('/node_modules/.vite/deps/react-dom_client.js')).default;const {default:AssetLibrary}=await import('/src/AssetLibrary.tsx');
document.documentElement.classList.add('dark');document.body.innerHTML='<div id="library-check" class="desktop-workspace"></div>';
const h=React.createElement;createRoot(document.getElementById('library-check')).render(h('div',null,...['worlds','packs','skins'].map(page=>h('div',{className:'asset-workspace',style:{height:180},key:page},h(AssetLibrary,{page,worlds:[],packs:[],skins:[],onOpenEditor:()=>{},onFolder:()=>{},onMap:()=>{},onSaveSkin:()=>{},onSource:()=>{}}),h('aside',{className:'configuration-grid'},'Select source'))),h('div',{className:'render-header'},h('div',null,h('h1',null,'Render editor')))));
await new Promise(r=>setTimeout(r,150));const style=e=>{const s=getComputedStyle(e),r=e.getBoundingClientRect();return {height:r.height,background:s.backgroundColor,padding:s.padding,font:getComputedStyle(e.querySelector('h1,h2')).fontSize}};
const editor=style(document.querySelector('.render-header'));const libraries=[...document.querySelectorAll('.library-heading')].map(style);for(const item of libraries)if(JSON.stringify(item)!==JSON.stringify(editor))throw Error(JSON.stringify({editor,libraries}));
for(const el of document.querySelectorAll('.asset-workspace')){if(el.querySelector('.asset-library').getBoundingClientRect().top!==el.getBoundingClientRect().top)throw Error('Empty heading row remains');}
const app=await(await fetch('/src/App.tsx')).text();if(app.includes('Explore the server. Save your world.'))throw Error('Duplicate title remains');return {editor,libraries};})()`});
if(result.exceptionDetails)throw Error(JSON.stringify(result.exceptionDetails));console.log(JSON.stringify(result.result.value));const screenshot=await send('Page.captureScreenshot',{format:'png'});await writeFile('tools/light-audit/library-headers.png',Buffer.from(screenshot.data,'base64'));
} finally {ws?.close();browser.kill();await new Promise(r=>setTimeout(r,500));await rm(profile,{recursive:true,force:true}).catch(()=>{});}

