                                                                           
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
const source=await(await fetch('/src/DesktopChrome.tsx')).text();const suffix=source.split('"/node_modules/.vite/deps/react.js')[1].split('"')[0];const React=(await import('/node_modules/.vite/deps/react.js'+suffix)).default;const {createRoot}=(await import('/node_modules/.vite/deps/react-dom_client.js')).default;const {default:Flyouts,notify,notifySaved}=await import('/src/StudioFlyouts.tsx');const {default:Chrome}=await import('/src/DesktopChrome.tsx');const {Dialog,DialogContent,DialogHeader,DialogTitle}=await import('/src/components/ui/dialog.tsx');
localStorage.clear();document.documentElement.classList.add('dark');document.body.innerHTML='<div id="flyout-check" class="desktop-workspace"></div>';window.__folder=0;
const h=React.createElement;createRoot(document.getElementById('flyout-check')).render(h(React.Fragment,null,h(Chrome,{page:'render',onPage:()=>{},onCommand:()=>{},onAccount:()=>{},onSettings:()=>{},onLogs:()=>{},onHelp:()=>{},onTheme:()=>{},theme:'dark',account:'',running:false,status:'Ready',logsOpen:false,editor:{ready:true,busy:false,undo:false,redo:false}}),h(Flyouts),h(Dialog,{open:true},h(DialogContent,null,h(DialogHeader,null,h(DialogTitle,null,'Export complete')),h('p',null,'The model is ready.')))));
await new Promise(r=>setTimeout(r,200));notifySaved('C:/Users/danie/Downloads/Hive Games Lobby Render.png',async()=>{window.__folder++});await new Promise(r=>setTimeout(r,250));
const card=document.querySelector('.studio-flyout');const hidden=!!card.closest('[inert],[aria-hidden=true]');const action=[...card.querySelectorAll('button')].find(b=>b.textContent==='Open folder');action.click();await new Promise(r=>setTimeout(r,150));if(window.__folder!==1)throw Error('Flyout action failed');
notify({tone:'success',title:'Render saved',description:'Hive Games Lobby Render.png',duration:Infinity,action:{label:'Open folder',onClick:()=>{}}});notify({tone:'warning',title:'Capture is still running',description:'End the current session before installing an update.',duration:Infinity});notify({tone:'error',title:'Export failed',description:'The destination is unavailable. Choose another folder and try again.',duration:Infinity});await new Promise(r=>setTimeout(r,500));return {actionWorks:true,hiddenByDialog:hidden,tones:[...document.querySelectorAll('.studio-flyout')].map(e=>e.dataset.tone)};})()`});
if(result.exceptionDetails)throw Error(JSON.stringify(result.exceptionDetails));console.log(JSON.stringify(result.result.value));const darkStyle=await send('Runtime.evaluate',{expression:"getComputedStyle(document.querySelector('.studio-flyout')).backgroundColor",returnByValue:true});const shot=await send('Page.captureScreenshot',{format:'png'});await writeFile('tools/light-audit/action-flyouts.png',Buffer.from(shot.data,'base64'));await send('Runtime.evaluate',{expression:"document.documentElement.classList.remove('dark');document.documentElement.dataset.theme='light'"});await new Promise(r=>setTimeout(r,100));const lightStyle=await send('Runtime.evaluate',{expression:"getComputedStyle(document.querySelector('.studio-flyout')).backgroundColor",returnByValue:true});if(darkStyle.result.value===lightStyle.result.value)throw Error('Flyout did not follow theme');console.log(JSON.stringify({dark:darkStyle.result.value,light:lightStyle.result.value}));const lightShot=await send('Page.captureScreenshot',{format:'png'});await writeFile('tools/light-audit/action-flyouts-light.png',Buffer.from(lightShot.data,'base64'));
} finally {ws?.close();browser.kill();await new Promise(r=>setTimeout(r,500));await rm(profile,{recursive:true,force:true}).catch(()=>{});}

