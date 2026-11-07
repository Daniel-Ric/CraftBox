import {spawn} from 'node:child_process';
import {readFile,mkdtemp,rm,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
const galleryPath=process.argv[2]??'tools/light-audit/candle-gallery.json';
const outputPrefix=process.argv[3]??'tools/light-audit/candles';
const wide=process.argv[2]!==undefined;
const gallery=JSON.parse(await readFile(galleryPath,'utf8'));
const glb=(await readFile(galleryPath+'.glb')).toString('base64');
const profile=await mkdtemp(join(tmpdir(),'craftbox-candle-'));
const browser=spawn('C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',['--headless=new','--no-sandbox','--no-first-run','--use-angle=swiftshader','--enable-unsafe-swiftshader','--remote-debugging-port=9337',`--user-data-dir=${profile}`,'http://127.0.0.1:5204'],{windowsHide:true,stdio:'ignore'});
let ws;
try{
 let tabs;for(let i=0;i<100;i++){try{tabs=await(await fetch('http://127.0.0.1:9337/json')).json();if(tabs.some(t=>t.type==='page'))break}catch{}await new Promise(r=>setTimeout(r,100))}
 ws=new WebSocket(tabs.find(t=>t.type==='page').webSocketDebuggerUrl);await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});ws.addEventListener('error',reject,{once:true})});let id=0;const pending=new Map();ws.addEventListener('message',({data})=>{const m=JSON.parse(data);if(m.id){const p=pending.get(m.id);pending.delete(m.id);m.error?p.reject(m.error):p.resolve(m.result)}});const send=(method,params={})=>new Promise((resolve,reject)=>{const key=++id;pending.set(key,{resolve,reject});ws.send(JSON.stringify({id:key,method,params}))});
 await send('Page.enable');await send('Page.navigate',{url:'http://127.0.0.1:5204'});await new Promise(r=>setTimeout(r,1200));
 const result=await send('Runtime.evaluate',{awaitPromise:true,returnByValue:true,expression:`(async()=>{
 const T=await import('/node_modules/three/build/three.module.js');const {GLTFLoader}=await import('/node_modules/three/examples/jsm/loaders/GLTFLoader.js');const data=${JSON.stringify(gallery)};
 const renderer=new T.WebGLRenderer({antialias:false,preserveDrawingBuffer:true});renderer.setSize(1100,700);renderer.setPixelRatio(1);renderer.outputColorSpace=T.SRGBColorSpace;const scene=new T.Scene();scene.background=new T.Color(0x8b95a6);scene.add(new T.AmbientLight(0xffffff,2));const light=new T.DirectionalLight(0xffffff,2);light.position.set(-5,15,10);scene.add(light);
 const atlas=await new T.TextureLoader().loadAsync(data.atlas);atlas.flipY=false;atlas.colorSpace=T.SRGBColorSpace;atlas.magFilter=T.NearestFilter;atlas.minFilter=T.NearestFilter;atlas.generateMipmaps=false;
 const terrain=new T.Group();let candles=0;
 for(const chunk of data.chunks)for(const part of chunk.parts){const geometry=new T.BufferGeometry();geometry.setAttribute('position',new T.Float32BufferAttribute(part.positions,3));geometry.setAttribute('normal',new T.Float32BufferAttribute(part.normals,3));geometry.setAttribute('uv',new T.Float32BufferAttribute(part.uvs,2));geometry.setIndex(part.indices);const options={map:atlas,side:T.DoubleSide,alphaTest:part.mode==='opaque'?0:.5};const material=part.mode==='unlit'?new T.MeshBasicMaterial(options):new T.MeshStandardMaterial({...options,roughness:1});const mesh=new T.Mesh(geometry,material);mesh.position.set(chunk.x*16,0,chunk.z*16);terrain.add(mesh);candles+=part.positions.length/3;}scene.add(terrain);
 const camera=new T.OrthographicCamera(-7,7,4.45,-4.45,.01,100);camera.position.set(12,12,14);camera.lookAt(7,1,4);if(${wide}){camera.left=-11;camera.right=11;camera.top=7;camera.bottom=-7;camera.position.set(22,20,25);camera.lookAt(7,1,8);camera.updateProjectionMatrix();}renderer.render(scene,camera);const overview=renderer.domElement.toDataURL();
 if(!${wide}){camera.left=-1.4;camera.right=1.4;camera.top=.89;camera.bottom=-.89;camera.position.set(13,3.1,4.9);camera.lookAt(11.5,1.25,2.5);camera.updateProjectionMatrix();}renderer.render(scene,camera);const closeup=renderer.domElement.toDataURL();
 scene.remove(terrain);const exported=await new GLTFLoader().loadAsync('data:model/gltf-binary;base64,${glb}');scene.add(exported.scene);renderer.render(scene,camera);const exportedImage=renderer.domElement.toDataURL();const error=renderer.getContext().getError();if(error)throw Error('WebGL error '+error);renderer.dispose();return {overview,closeup,exportedImage,vertices:candles,meshes:terrain.children.length};})()`});
 if(result.exceptionDetails)throw Error(JSON.stringify(result.exceptionDetails));const {overview,closeup,exportedImage,...report}=result.result.value;for(const [key,value]of Object.entries({overview,closeup,exported:exportedImage}))await writeFile(`${outputPrefix}-${key}.png`,Buffer.from(value.split(',')[1],'base64'));console.log(JSON.stringify(report));
}finally{ws?.close();browser.kill();await new Promise(r=>setTimeout(r,400));await rm(profile,{recursive:true,force:true}).catch(()=>{})}
