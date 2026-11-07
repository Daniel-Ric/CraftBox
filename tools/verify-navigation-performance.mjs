import { spawn } from 'node:child_process';
import { mkdtemp, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const page=JSON.parse(await readFile('tools/light-audit/hive-render-page.json','utf8'));
const profile=await mkdtemp(join(tmpdir(),'craftbox-navigation-'));
const browser=spawn('C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',[
  '--headless=new','--no-sandbox','--no-first-run','--use-angle=swiftshader','--enable-unsafe-swiftshader',
  '--disable-background-timer-throttling','--disable-renderer-backgrounding','--disable-backgrounding-occluded-windows',
  '--remote-debugging-port=9341',`--user-data-dir=${profile}`,'http://127.0.0.1:5204'
],{windowsHide:true,stdio:'ignore'});
let ws;
try {
 let tabs;
 for(let i=0;i<100;i++){try{tabs=await(await fetch('http://127.0.0.1:9341/json')).json();if(tabs.some(t=>t.type==='page'))break}catch{}await new Promise(r=>setTimeout(r,100))}
 ws=new WebSocket(tabs.find(t=>t.type==='page').webSocketDebuggerUrl);
 await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});ws.addEventListener('error',reject,{once:true})});
 let id=0;const pending=new Map();
 ws.addEventListener('close',()=>{for(const p of pending.values())p.reject(Error('Verification browser connection closed'));pending.clear()});
 ws.addEventListener('message',({data})=>{const m=JSON.parse(data);if(m.id){const p=pending.get(m.id);pending.delete(m.id);m.error?p.reject(m.error):p.resolve(m.result)}});
 const send=(method,params={})=>new Promise((resolve,reject)=>{const key=++id;pending.set(key,{resolve,reject});ws.send(JSON.stringify({id:key,method,params}))});
 await send('Emulation.setDeviceMetricsOverride',{width:1000,height:700,deviceScaleFactor:2,mobile:false});
 await send('Page.navigate',{url:'http://127.0.0.1:5204'});
 await new Promise(r=>setTimeout(r,1500));
                                                                               
                                                               
 await send('Page.setWebLifecycleState',{state:'active'});
 await send('Emulation.setFocusEmulationEnabled',{enabled:true});
 await send('Page.bringToFront');
 await send('Runtime.evaluate',{expression:`window.__navigationCheck={state:'running',stage:'import'};(async()=>{const {runNavigationCheck}=await import('/perf/navigation-fixture.ts');return runNavigationCheck(${JSON.stringify(page)},stage=>window.__navigationCheck.stage=stage);})().then(reports=>window.__navigationCheck={state:'done',reports},error=>window.__navigationCheck={state:'error',message:String(error?.stack??error)});true`});
 let reports,lastStage='';const deadline=Date.now()+360000;
 while(Date.now()<deadline){
   const result=await send('Runtime.evaluate',{returnByValue:true,expression:'window.__navigationCheck'});
   if(result.exceptionDetails)throw Error(JSON.stringify(result.exceptionDetails));
   const job=result.result.value;
   if(!job)throw Error('Verification page was replaced before the job completed');
   if(job.state==='error')throw Error(job.message);
   if(job.state==='done'){reports=job.reports;break;}
   if(job.stage!==lastStage){lastStage=job.stage;console.log('Verification:',lastStage);}
   await new Promise(r=>setTimeout(r,500));
 }
 if(!reports)throw Error('Verification exceeded its six-minute work budget');
 for(const report of reports){await writeFile(`tools/light-audit/navigation-${report.editor}-export.png`,Buffer.from(report.exportProofImage.split(',')[1],'base64'));delete report.exportProofImage;}
 await writeFile('tools/light-audit/navigation-performance.json',JSON.stringify(reports,null,2));
 console.log(JSON.stringify(reports));
} finally {
 ws?.close();browser.kill();await new Promise(r=>setTimeout(r,400));await rm(profile,{recursive:true,force:true}).catch(()=>{});
}
