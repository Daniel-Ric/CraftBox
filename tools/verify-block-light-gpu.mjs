                                                                           
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
 const T=await import('/node_modules/three/build/three.module.js');
 const {installBlockLight,installEmission}=await import('/src/render-lighting.ts');
 const {installFoliageLighting}=await import('/src/foliage-lighting.ts');
 const {WaterSurface}=await import('/src/render-water.ts');
 const {renderDefaults}=await import('/src/render-settings.ts');
 const {decodePart}=await import('/src/scene-wire.ts');
 const errors=[];const original=console.error;console.error=(...args)=>{errors.push(args.join(' '));original(...args)};
 const r=new T.WebGLRenderer({preserveDrawingBuffer:true});r.setSize(384,256);r.outputColorSpace=T.LinearSRGBColorSpace;r.toneMapping=T.NoToneMapping;
 const scene=new T.Scene();scene.background=new T.Color(0,0,0);
 const camera=new T.PerspectiveCamera(40,1.5,.1,100);camera.position.set(4,4,6);camera.lookAt(0,0,0);
 const geometry=new T.BoxGeometry(2,2,2);const count=geometry.attributes.position.count;
 geometry.setAttribute('blockLight',new T.Uint8BufferAttribute(new Uint8Array(count).fill(14),1));
 geometry.setAttribute('blockEmission',new T.Float32BufferAttribute(new Float32Array(count),1));
 geometry.setAttribute('foliage',new T.Float32BufferAttribute(new Float32Array(count),1));
 const mesh=new T.Mesh(geometry);scene.add(mesh);const power={value:0};const metrics={};
 const brightness=()=>{r.render(scene,camera);const pixels=new Uint8Array(384*256*4);r.getContext().readPixels(0,0,384,256,r.getContext().RGBA,r.getContext().UNSIGNED_BYTE,pixels);let sum=0;for(let i=0;i<pixels.length;i+=4)sum+=pixels[i]+pixels[i+1]+pixels[i+2];return sum/(384*256*3)};
 for(const mode of ['standard','physical','foliage','water']) {
  const water=mode==='water'?new WaterSurface():undefined;
  const material=water?.material??(mode==='physical'?new T.MeshPhysicalMaterial({color:0xcccccc}):new T.MeshStandardMaterial({color:0xcccccc}));
  installEmission(material,{value:1});if(mode==='foliage')installFoliageLighting(material);installBlockLight(material,power);if(water){water.install();const depth=new T.DataTexture(new Uint8Array([255]),1,1,T.RedFormat);depth.needsUpdate=true;water.configure({...renderDefaults,waterColorStrength:0,waterOpacity:1,waterTransparency:0,waterDeepLight:0},camera,depth,384,256);}mesh.material=material;
  power.value=0;const off=brightness();power.value=1;const on=brightness();metrics[mode]={off,on};if(!(on>off+5))throw new Error('Block light did not illuminate '+mode);
 }
 // Decode a page generated directly from the user's saved Hive world.
 const page=${JSON.stringify(hivePage)};
 scene.remove(mesh);const terrain=new T.Group();scene.add(terrain);
 const material=new T.MeshStandardMaterial({color:0xcccccc});installBlockLight(material,power);
 let litVertices=0;
 for(const chunk of page.chunks)for(const packed of chunk.parts){if(['billboard','unlit','unlit-blend'].includes(packed.mode))continue;const p=decodePart(packed);const g=new T.BufferGeometry();g.setAttribute('position',new T.BufferAttribute(p.positions,3));g.setAttribute('normal',new T.BufferAttribute(p.normals,3,true));g.setAttribute('blockLight',new T.BufferAttribute(p.blockLight,1));g.setIndex(new T.BufferAttribute(p.indices,1));litVertices+=p.blockLight.filter(l=>l>0).length;terrain.add(new T.Mesh(g,material));}
 const box=new T.Box3().setFromObject(terrain),center=box.getCenter(new T.Vector3());const size=box.getSize(new T.Vector3()).length();camera.position.copy(center).add(new T.Vector3(size*.5,size*.3,size*.6));camera.lookAt(center);camera.far=size*5;camera.updateProjectionMatrix();
 power.value=0;const off=brightness();power.value=1;const on=brightness();if(!litVertices||!(on>off))throw new Error('Hive world surface lighting missing');metrics.hive={litVertices,off,on};
 const image=r.domElement.toDataURL();console.error=original;r.dispose();if(errors.length)throw new Error(errors.join('\\n'));return {metrics,errors,image};
})()`});
 if(result.exceptionDetails)throw new Error(JSON.stringify(result.exceptionDetails));
 const {image,...report}=result.result.value;
 await writeFile('tools/light-audit/hive-block-light-gpu.png',Buffer.from(image.split(',')[1],'base64'));
 await writeFile('tools/light-audit/gpu-result.json',JSON.stringify(report,null,2));
 console.log(JSON.stringify(report));
} finally {ws?.close();browser.kill();await new Promise(r=>setTimeout(r,500));await rm(profile,{recursive:true,force:true}).catch(()=>{});}
