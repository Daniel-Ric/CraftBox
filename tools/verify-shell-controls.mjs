                                                                           
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
 const init=await send('Runtime.evaluate',{awaitPromise:true,returnByValue:true,expression:`(async()=>{
 const source=await (await fetch('/src/DesktopChrome.tsx')).text();const suffix=source.split('"/node_modules/.vite/deps/react.js')[1].split('"')[0];
 const React=(await import('/node_modules/.vite/deps/react.js'+suffix)).default;
 const {createRoot}=(await import('/node_modules/.vite/deps/react-dom_client.js')).default;
 const {default:DesktopChrome}=await import('/src/DesktopChrome.tsx');
 const {Dialog,DialogContent,DialogHeader,DialogTitle}=await import('/src/components/ui/dialog.tsx');
 const {Input}=await import('/src/components/ui/input.tsx');
 document.documentElement.classList.add('dark');document.body.innerHTML='<div id="shell-test"></div>';
 function Harness(){const [page,setPage]=React.useState('worlds'),[open,setOpen]=React.useState(false),[name,setName]=React.useState('Castle view');
 return React.createElement('div',{className:'desktop-workspace workspace'},
 React.createElement(DesktopChrome,{page,onPage:setPage,onCommand:()=>{},onAccount:()=>{},onSettings:()=>{},onLogs:()=>{},onHelp:()=>{},onTheme:()=>{},theme:'dark',account:'',running:false,status:'Ready',logsOpen:false,editor:{ready:true,busy:false,undo:false,redo:false}}),
 React.createElement('main',{style:{padding:24}},React.createElement('button',{className:'studio-button',onClick:()=>setOpen(true)},'Open panel')),
 React.createElement(Dialog,{open,onOpenChange:setOpen},React.createElement(DialogContent,null,
 React.createElement(DialogHeader,null,React.createElement(DialogTitle,null,'Move panel'),React.createElement('button',{'aria-label':'Header action',onClick:()=>window.__headerClicks=(window.__headerClicks||0)+1},'Action')),
 React.createElement('div',{style:{padding:24}},React.createElement('label',null,'Project name',React.createElement(Input,{'aria-label':'Project name',value:name,onChange:e=>setName(e.target.value)}))))));}
 createRoot(document.getElementById('shell-test')).render(React.createElement(Harness));
 const delay=()=>new Promise(r=>setTimeout(r,100));for(let i=0;i<100&&!document.querySelector('.workspace-home');i++)await delay();
 const home=document.querySelector('.workspace-home');const worlds=document.querySelector('.workspace-tabs button:nth-child(2)');
 if(home.textContent.trim()||home.getAttribute('aria-label')!=='Start'||home.getBoundingClientRect().width>40)throw new Error('Start is not compact/icon-only');
 home.click();await delay();if(home.getAttribute('aria-current')!=='page')throw new Error('Start navigation failed');worlds.click();await delay();
 if(getComputedStyle(worlds,'::after').height!=='2px')throw new Error('Selected tab indicator missing');
 [...document.querySelectorAll('button')].find(b=>b.textContent==='Open panel').click();
 for(let i=0;i<100&&!document.querySelector('[role=dialog]');i++)await delay();await delay();
 const input=document.querySelector('input[aria-label="Project name"]');input.focus();const focused=getComputedStyle(input);
 if(focused.outlineStyle!=='none'||focused.boxShadow!=='none'||focused.borderTopWidth!=='1px')throw new Error('Focus still has double lines');
 const color=focused.borderColor;input.setAttribute('aria-invalid','true');await new Promise(r=>setTimeout(r,200));if(getComputedStyle(input).borderColor===color)throw new Error('Error border lost');input.removeAttribute('aria-invalid');
 window.__shellRead=()=>{const dialog=document.querySelector('[role=dialog]');const rect=dialog?.getBoundingClientRect();const header=dialog?.querySelector('[data-slot=dialog-header]').getBoundingClientRect();return {rect:rect?.toJSON(),header:header?.toJSON(),translate:dialog?.style.translate,input:document.querySelector('input[aria-label="Project name"]')?.value,open:!!dialog}};
 return {homeWidth:home.getBoundingClientRect().width,focus:{outline:focused.outlineStyle,shadow:focused.boxShadow,border:focused.borderTopWidth},...window.__shellRead()};
 })()`});
 if(init.exceptionDetails)throw new Error(JSON.stringify(init.exceptionDetails));
 const read=async()=>{const r=await send('Runtime.evaluate',{expression:'window.__shellRead()',returnByValue:true});return r.result.value;};
 const mouse=async(type,x,y,pressed=false,count=1)=>{await send('Input.dispatchMouseEvent',{type,x,y,button:type==='mouseMoved'?'none':'left',buttons:pressed?1:0,clickCount:count});await new Promise(r=>setTimeout(r,60));};
 await send('Runtime.evaluate',{expression:`window.__events=[];for(const t of ['pointerdown','pointerup','pointermove','lostpointercapture','gotpointercapture','dblclick'])document.addEventListener(t,e=>window.__events.push([t,e.pointerId,e.buttons,e.defaultPrevented,document.querySelector('[role=dialog]')?.dataset.dragging,e.clientX,e.clientY,e.target.outerHTML.slice(0,80)]))`});
 const before=await read();let x=before.header.left+10,y=before.header.top+18;
 await mouse('mouseMoved',x,y);await mouse('mousePressed',x,y,true);await mouse('mouseMoved',x+100,y+70,true);await mouse('mouseReleased',x+100,y+70);
 const moved=await read();if(Math.abs(moved.rect.left-before.rect.left-100)>1||Math.abs(moved.rect.top-before.rect.top-70)>1)throw new Error('Native mouse drag failed '+JSON.stringify({before,moved}));
 x=moved.header.left+10;y=moved.header.top+18;await mouse('mouseMoved',x,y);await mouse('mousePressed',x,y,true);await mouse('mouseMoved',5,5,true);await mouse('mouseReleased',5,5);
 const bounded=await read();if(Math.abs(bounded.rect.left-8)>1||Math.abs(bounded.rect.top-8)>1)throw new Error('Viewport clamp failed '+JSON.stringify({bounded,events:(await send('Runtime.evaluate',{expression:'window.__events',returnByValue:true})).result.value}));
 x=bounded.header.left+10;y=bounded.header.top+18;await mouse('mouseMoved',x,y);for(const count of [1,2]){await mouse('mousePressed',x,y,true,count);await mouse('mouseReleased',x,y,false,count);}
 const centered=await read();if(centered.translate)throw new Error('Double-click did not recenter');if(centered.input!=='Castle view')throw new Error('Dragging changed the form');
 const dialogImage=await send('Page.captureScreenshot',{format:'png'});await writeFile('tools/light-audit/dialog-focus.png',Buffer.from(dialogImage.data,'base64'));
 await send('Input.dispatchKeyEvent',{type:'keyDown',key:'Escape',code:'Escape',windowsVirtualKeyCode:27});await send('Input.dispatchKeyEvent',{type:'keyUp',key:'Escape',code:'Escape',windowsVirtualKeyCode:27});
 await new Promise(r=>setTimeout(r,200));if((await read()).open)throw new Error('Escape no longer closes dialog');
 const report={homeWidth:init.result.value.homeWidth,focus:init.result.value.focus,drag:{dx:moved.rect.left-before.rect.left,dy:moved.rect.top-before.rect.top},bounded:{left:bounded.rect.left,top:bounded.rect.top},recentered:true,escapeCloses:true};
 const screenshot=await send('Page.captureScreenshot',{format:'png'});await writeFile('tools/light-audit/shell-controls.png',Buffer.from(screenshot.data,'base64'));
 await writeFile('tools/light-audit/shell-controls-result.json',JSON.stringify(report,null,2));console.log(JSON.stringify(report));
} finally {ws?.close();browser.kill();await new Promise(r=>setTimeout(r,500));await rm(profile,{recursive:true,force:true}).catch(()=>{});}

