                                                                           
import { spawn } from 'node:child_process';
import { mkdtemp, rm, writeFile, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const profile=await mkdtemp(join(tmpdir(),'craftbox-block-light-'));
const hivePage=JSON.parse(await readFile('tools/light-audit/hive-render-page.json','utf8'));
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
 const result=await send('Runtime.evaluate',{awaitPromise:true,returnByValue:true,expression:`(async()=>{
 const React=(await import('/node_modules/.vite/deps/react.js')).default;const {createRoot}=(await import('/node_modules/.vite/deps/react-dom_client.js')).default;
 const {default:RenderEditor}=await import('/src/RenderEditor.tsx');
 const editorSource=await (await fetch('/src/RenderEditor.tsx')).text();
 const canvasUrl='/src/RenderCanvas.tsx'+editorSource.split('"/src/RenderCanvas.tsx')[1].split('"')[0];
 const {RenderEngine}=await import(canvasUrl);
 const {renderDefaults}=await import('/src/render-settings.ts');
 const page=${JSON.stringify(hivePage)};page.complete=true;page.next=null;page.total=page.chunks.length;
 const errors=[],engines=[];const load=RenderEngine.prototype.load;RenderEngine.prototype.load=function(){engines.push(this);return load.call(this)};
 window.go={desktop:{App:{GetRenderScenePage:async()=>structuredClone(page),GetMapPage:async()=>({tiles:[],complete:true,next:null,sequence:1,total:0,coverage:[]})}}};
 document.body.innerHTML='<div id="switch-test" style="height:500px;width:900px;display:flex;flex-direction:column"></div>';
 const host=document.getElementById('switch-test');
 const settings={...renderDefaults,sky:'none',shadow:256,quality:'fast',ao:0,environment:0,bloom:0,atmosphere:{...renderDefaults.atmosphere,clouds:false,fog:false}};
 const root=createRoot(host);root.render(React.createElement(RenderEditor,{worlds:[{id:'test',name:'Hive',chunks:1,state:'finished'}],activeWorldId:'test',loadedProject:{world:{id:'test',name:'Hive',chunks:1},path:'test',document:{name:'Test',settings,mapView:{enabled:false,rotation:0,zoom:1,markers:[]}}}}));
 const wait=async(fn)=>{for(let i=0;i<200;i++){if(fn())return;await new Promise(r=>setTimeout(r,50))}throw new Error('Timed out; '+JSON.stringify({errors,engines:engines.map(e=>({loading:e.loading,disposed:e.disposed,groups:e.groups.size})),text:host.innerText.slice(-500)}))};
 await wait(()=>engines.length&&engines.at(-1).loading===false);await new Promise(r=>setTimeout(r,100));
 const read=()=>{const e=engines.at(-1);e.draw();const c=e.renderer.domElement;return {engines:engines.length,width:c.width,height:c.height,rect:[c.clientWidth,c.clientHeight],camera:e.snapshot(),host:[e.host.clientWidth,e.host.clientHeight],groups:e.groups.size,disposed:e.disposed}};
 const first=read();const firstImage=engines.at(-1).renderer.domElement.toDataURL();const button=(text)=>[...host.querySelectorAll('button')].find(b=>b.textContent.trim()===text);
 button('2D map').click();await new Promise(r=>setTimeout(r,200));const map={oldDisposed:engines[0].disposed,canvas:host.querySelectorAll('.render-canvas').length};
 button('3D view').click();await wait(()=>!engines.at(-1).loading&&engines.at(-1).renderer.domElement.clientWidth>0);await new Promise(r=>setTimeout(r,100));const restored=read();
 const image=engines.at(-1).renderer.domElement.toDataURL();const identicalImage=firstImage===image;
 if(engines.length!==1||map.oldDisposed||!identicalImage)throw new Error(JSON.stringify({first,map,restored,identicalImage}));
 root.unmount();return {first,map,restored,identicalImage,errors,image};
 })()`});
 if(result.exceptionDetails)throw new Error(JSON.stringify(result.exceptionDetails));
 const {image,...report}=result.result.value;
 await writeFile('tools/light-audit/editor-view-switch-result.json',JSON.stringify(report,null,2));
 await writeFile('tools/light-audit/editor-view-switch.png',Buffer.from(image.split(',')[1],'base64'));
 console.log(JSON.stringify(report));
} finally {ws?.close();browser.kill();await new Promise(r=>setTimeout(r,500));await rm(profile,{recursive:true,force:true}).catch(()=>{});}


