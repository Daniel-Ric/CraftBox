                                                                           
import { spawn } from 'node:child_process';
import { mkdtemp, rm, writeFile, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const profile=await mkdtemp(join(tmpdir(),'craftbox-block-light-'));
const browser=spawn('C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',[
 '--headless=new','--no-sandbox','--no-first-run','--no-default-browser-check','--use-angle=swiftshader',
 '--enable-unsafe-swiftshader','--remote-debugging-port=9334',`--user-data-dir=${profile}`,'http://127.0.0.1:5204'
],{windowsHide:true,stdio:['ignore','ignore','pipe']});
browser.stderr.on('data',data=>process.stderr.write(data));
browser.on('exit',code=>console.error('Headless Edge exit:',code));
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
 const moduleSource=await (await fetch('/src/StartScreen.tsx')).text();
 const reactUrl='/node_modules/.vite/deps/react.js'+moduleSource.split('"/node_modules/.vite/deps/react.js')[1].split('"')[0];
 const React=(await import(reactUrl)).default;
 const {createRoot}=(await import('/node_modules/.vite/deps/react-dom_client.js')).default;
 const {StartDialog}=await import('/src/StartScreen.tsx');
 const {projectDefaults,recentKey,loadRecent,manageRecentProject}=await import('/src/start-project.ts');
 const errors=[];window.addEventListener('error',e=>errors.push(e.message));
 document.documentElement.classList.add('dark');document.body.innerHTML='<div id="project-test"></div>';
 const base={...projectDefaults,worldId:'hive',sourceName:'Hive Games',updated:Date.now()};
 localStorage.setItem(recentKey,JSON.stringify([
  {...base,id:'hive',name:'Untitled',nameMode:'auto'},
  {...base,id:'castle',name:'Castle trailer',page:'cinematic',duration:30,fps:30,updated:Date.now()-60000},
  {...base,id:'capture',page:'worlds',name:'Untitled',nameMode:'auto',address:'play.example.net',sourceName:'Hive Games',updated:Date.now()-120000},
  {...base,id:'island',name:'Sky island cover',worldId:'island',sourceName:'Sky island',updated:Date.now()-180000},
 ]));
 let created;
 function Harness(){const [recent,setRecent]=React.useState(loadRecent);return React.createElement(StartDialog,{open:true,onOpenChange:()=>{},recent,worlds:[{id:'hive',name:'Hive Games',chunks:1755},{id:'island',name:'Sky island',chunks:24}],onCreate:()=>{},onResume:p=>created=p,onPage:()=>{},onOpen:()=>{},onRecentAction:(id,action)=>setRecent(manageRecentProject(id,action))});}
 const root=createRoot(document.getElementById('project-test'));root.render(React.createElement(Harness));
 const delay=()=>new Promise(r=>setTimeout(r,75));
 const wait=async fn=>{for(let i=0;i<100;i++){if(fn())return;await delay()}throw new Error('Project controls timed out '+JSON.stringify({errors,recent:loadRecent(),text:document.body.innerText.slice(-800)}))};
 const button=text=>[...document.querySelectorAll('button')].find(b=>b.getAttribute('aria-label')===text||b.textContent.trim()===text);
 const input=(el,value)=>{Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(el,value);el.dispatchEvent(new Event('input',{bubbles:true}));};
 const select=(label,value)=>{const el=document.querySelector('select[aria-label="'+label+'"]');Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype,'value').set.call(el,value);el.dispatchEvent(new Event('change',{bubbles:true}));};
 await wait(()=>document.querySelector('.recent-name-form input'));
 await wait(()=>document.activeElement===document.querySelector('[role="dialog"]'));
 const recentTab=[...document.querySelectorAll('[role="tab"]')].find(b=>b.textContent.trim()==='Recent');
 if(getComputedStyle(recentTab).outlineStyle!=='none')throw new Error('Recent has an automatic startup focus outline');
 window.__startupFocus={role:document.activeElement.getAttribute('role'),tabFocused:document.activeElement===recentTab,dialogOutline:getComputedStyle(document.activeElement).outlineStyle};
 input(document.querySelector('.recent-name-form input'),'Hive castle cover');await delay();button('Save name').click();await delay();
 if(loadRecent().find(p=>p.id==='hive').name!=='Hive castle cover')throw new Error('Rename did not persist');
 button('Pin project').click();await delay();if(!loadRecent()[0].pinned)throw new Error('Pin did not persist');
 button('Duplicate project setup').click();await delay();if(!loadRecent().some(p=>p.name==='Hive castle cover copy'))throw new Error('Copy missing');
 input(document.querySelector('input[aria-label="Search recent projects"]'),'Castle trailer');await delay();
 if(document.querySelectorAll('.recent-preset-grid button').length!==1)throw new Error('Search failed');
 input(document.querySelector('input[aria-label="Search recent projects"]'),'');await delay();
 select('Filter recent projects','capture');await delay();if(document.querySelectorAll('.recent-preset-grid button').length!==1)throw new Error('Capture filter failed');
 select('Filter recent projects','all');select('Sort recent projects','name');await delay();
 button('Hive castle cover').click();await delay();button('Remove from recent').click();await delay();
 if(loadRecent().some(p=>p.id==='hive'))throw new Error('Removal did not persist');
 button('Hive castle cover copy').click();await delay();
 const bounds=document.querySelector('[role="dialog"]').getBoundingClientRect();
 if(bounds.left<0||bounds.top<0||bounds.right>1281||bounds.bottom>821)throw new Error('Dialog overflow');
 const recentImage=await new Promise(resolve=>requestAnimationFrame(()=>resolve(document.querySelector('[role="dialog"]').outerHTML)));
 window.__projectRecentReady=true;
 return {count:loadRecent().length,names:loadRecent().map(p=>p.name),bounds:{width:bounds.width,height:bounds.height},startupFocus:window.__startupFocus,recentImage};
 })()`});
 if(result.exceptionDetails)throw new Error(JSON.stringify(result.exceptionDetails));
 const {recentImage,...report}=result.result.value;
 await writeFile('tools/light-audit/project-controls-result.json',JSON.stringify(report,null,2));
 const screenshot=await send('Page.captureScreenshot',{format:'png'});
 await writeFile('tools/light-audit/project-controls.png',Buffer.from(screenshot.data,'base64'));
 const naming=await send('Runtime.evaluate',{awaitPromise:true,returnByValue:true,expression:`(async()=>{
  const delay=()=>new Promise(r=>setTimeout(r,100));
  const tab=text=>[...document.querySelectorAll('[role="tab"]')].find(b=>b.textContent.trim()===text);
  const change=(el,value)=>{Object.getOwnPropertyDescriptor(el instanceof HTMLSelectElement?HTMLSelectElement.prototype:HTMLInputElement.prototype,'value').set.call(el,value);el.dispatchEvent(new Event(el instanceof HTMLSelectElement?'change':'input',{bubbles:true}));};
  tab('Still render').click();await delay();change(document.querySelector('select[aria-label="World"]'),'hive');await delay();
  const input=document.querySelector('.document-properties input:not([type="checkbox"])');
  input.focus();
  const focusStyle=getComputedStyle(input);
  if(focusStyle.outlineStyle!=='none'||focusStyle.boxShadow!=='none'||focusStyle.borderTopWidth!=='1px')throw new Error('Input has stacked focus decoration');
  window.__focusStyle={outline:focusStyle.outlineStyle,shadow:focusStyle.boxShadow,border:focusStyle.borderTopWidth,color:focusStyle.borderTopColor};
  if(input.value!=='Hive Games – Still render')throw new Error('Automatic creation name failed: '+input.value);
  if(getComputedStyle(document.querySelector('.auto-project-name')).display!=='flex')throw new Error('Automatic name control layout failed');
  window.__autoProjectName=input.value;
  return {automaticName:input.value,focusStyle:window.__focusStyle};
 })()`});
 if(naming.exceptionDetails)throw new Error(JSON.stringify(naming.exceptionDetails));
 const newProjectScreenshot=await send('Page.captureScreenshot',{format:'png'});
 await writeFile('tools/light-audit/project-auto-name.png',Buffer.from(newProjectScreenshot.data,'base64'));
 report.automaticName=naming.result.value.automaticName;
 report.focusStyle=naming.result.value.focusStyle;
 await writeFile('tools/light-audit/project-controls-result.json',JSON.stringify(report,null,2));
 console.log(JSON.stringify(report));
} finally {ws?.close();browser.kill();await new Promise(r=>setTimeout(r,500));await rm(profile,{recursive:true,force:true}).catch(()=>{});}

