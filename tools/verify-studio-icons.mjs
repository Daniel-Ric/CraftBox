import { spawn } from 'node:child_process';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const profile=await mkdtemp(join(tmpdir(),'craftbox-icons-'));
const browser=spawn('C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',[
 '--headless=new','--no-sandbox','--no-first-run','--disable-background-timer-throttling',
 '--remote-debugging-port=9342',`--user-data-dir=${profile}`,'http://127.0.0.1:5204'
],{windowsHide:true,stdio:'ignore'});
let ws;
try {
 let tabs;for(let i=0;i<100;i++){try{tabs=await(await fetch('http://127.0.0.1:9342/json')).json();if(tabs.some(t=>t.type==='page'))break}catch{}await new Promise(r=>setTimeout(r,100))}
 ws=new WebSocket(tabs.find(t=>t.type==='page').webSocketDebuggerUrl);
 await new Promise((resolve,reject)=>{ws.onopen=resolve;ws.onerror=reject});
 let id=0;const pending=new Map();
 ws.onmessage=({data})=>{const m=JSON.parse(data);if(m.id){const p=pending.get(m.id);pending.delete(m.id);m.error?p.reject(m.error):p.resolve(m.result)}};
 ws.onclose=()=>{for(const p of pending.values())p.reject(Error('Icon verification browser closed'));pending.clear()};
 const send=(method,params={})=>new Promise((resolve,reject)=>{const key=++id;pending.set(key,{resolve,reject});ws.send(JSON.stringify({id:key,method,params}))});
 await send('Emulation.setDeviceMetricsOverride',{width:1280,height:820,deviceScaleFactor:1,mobile:false});
 await send('Page.navigate',{url:'http://127.0.0.1:5204'});
 await new Promise(r=>setTimeout(r,1200));
 const setup=await send('Runtime.evaluate',{awaitPromise:true,returnByValue:true,expression:`(async()=>{
 const source=await(await fetch('/src/StartScreen.tsx')).text();
 const reactUrl='/node_modules/.vite/deps/react.js'+source.split('"/node_modules/.vite/deps/react.js')[1].split('"')[0];
 const React=(await import(reactUrl)).default;
 const {createRoot}=(await import('/node_modules/.vite/deps/react-dom_client.js')).default;
 const {StartDialog}=await import('/src/StartScreen.tsx');
 const {default:DesktopChrome}=await import('/src/DesktopChrome.tsx');
 const {default:AssetLibrary}=await import('/src/AssetLibrary.tsx');
 const {projectDefaults}=await import('/src/start-project.ts');
 const {idleEditor}=await import('/src/editor-commands.ts');
 document.body.innerHTML='<div id="icons-test" class="desktop-workspace"></div>';document.documentElement.classList.add('dark');
 const root=createRoot(document.getElementById('icons-test'));const noop=()=>{};
 const recent=['worlds','render','cinematic','packs','skins'].map((page,i)=>({...projectDefaults,id:page,page,name:['Mineville Zeqa – World capture','Hive lobby – Still render','Castle tour – Camera sequence','Autumn textures – Resource packs','Player outfit – Player skins'][i],sourceName:'Hive Games',updated:Date.now()-i*1000}));
 const world={id:'world',name:'Hive Games',chunks:1755,entities:14,state:'finished',path:'D:/world'};
 const pack={id:'pack',name:'Autumn textures',version:'1.0.0',size:65536,received:65536,done:true,icon:'',path:'D:/pack'};
 const skin={name:'Player outfit',image:''};
 const chrome=page=>React.createElement(DesktopChrome,{page,onPage:noop,onCommand:noop,onAccount:noop,onSettings:noop,onLogs:noop,onHelp:noop,onTheme:noop,theme:'dark',account:'Microsoft account',running:false,status:'Ready',logsOpen:false,editor:idleEditor});
 window.__showIconView=async(view,theme='dark')=>{
  document.documentElement.classList.toggle('dark',theme==='dark');
  const library=React.createElement(AssetLibrary,{page:view==='welcome'?'worlds':view,worlds:[world],packs:[pack],skins:[skin],onOpenEditor:noop,onFolder:noop,onMap:noop,onSaveSkin:noop,onSource:noop,onWorldSettings:noop,onWorldCopy:noop});
  root.render(React.createElement(React.Fragment,null,chrome(view==='welcome'?'worlds':view),library,view==='welcome'?React.createElement(StartDialog,{open:true,onOpenChange:noop,recent,worlds:[world],onCreate:noop,onResume:noop,onPage:noop,onOpen:noop}):null));
  await new Promise(r=>setTimeout(r,300));
  if(view==='welcome'){
   const roles=[...document.querySelectorAll('.recent-preset-grid [data-icon-role]')].map(e=>e.dataset.iconRole);
   if(roles.join()!=='world,render,cinematic,pack,skin')throw Error('Project icons use the wrong subject: '+roles);
   if(document.querySelector('.recent-preset-grid .format-preview i'))throw Error('Legacy boxed icon placeholder remains');
  }else{
   document.querySelector('.library-items button')?.click();await new Promise(r=>setTimeout(r,100));
   const role={worlds:'world',packs:'pack',skins:'skin'}[view];
   if(!document.querySelector('.library-heading [data-icon-role="'+role+'"]'))throw Error('Wrong library header icon');
   if(!document.querySelector('.library-thumbnail [data-icon-role="'+role+'"]'))throw Error('Wrong library fallback icon');
  }
  const svg=document.querySelector('.workspace-tabs .studio-icon');
  return {view,theme,stroke:getComputedStyle(svg).strokeWidth,workspaceIcons:[...document.querySelectorAll('.workspace-tabs [data-icon-role]')].map(e=>e.dataset.iconRole)};
 };
 return true;
})()`});
 if(setup.exceptionDetails)throw Error(JSON.stringify(setup.exceptionDetails));
 const reports=[];
 for(const [view,theme]of [['welcome','dark'],['welcome','light'],['worlds','dark'],['packs','dark'],['skins','dark']]){
  const result=await send('Runtime.evaluate',{awaitPromise:true,returnByValue:true,expression:`window.__showIconView(${JSON.stringify(view)},${JSON.stringify(theme)})`});
  if(result.exceptionDetails)throw Error(JSON.stringify(result.exceptionDetails));
  reports.push(result.result.value);
  const shot=await send('Page.captureScreenshot',{format:'png'});
  await writeFile(`tools/light-audit/icons-${view}-${theme}.png`,Buffer.from(shot.data,'base64'));
 }
 await writeFile('tools/light-audit/icons-result.json',JSON.stringify(reports,null,2));console.log(JSON.stringify(reports));
}finally{ws?.close();browser.kill();await new Promise(r=>setTimeout(r,400));await rm(profile,{recursive:true,force:true}).catch(()=>{});}
