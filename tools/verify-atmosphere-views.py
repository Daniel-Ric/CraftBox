"""Inspect the atmosphere from below, above and near the horizon, with seeded variation."""
from pathlib import Path
import base64,json
from PIL import Image
from playwright.sync_api import sync_playwright
out=Path('tmp/atmosphere-v2/views');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 b=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=b.new_page();errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text) else None)
 page.goto('http://127.0.0.1:5204')
 images=page.evaluate('''async()=>{
 const T=await import('/node_modules/three/build/three.module.js'),{AtmospherePass}=await import('/src/atmosphere-volume.ts'),{renderDefaults}=await import('/src/render-settings.ts');
 const r=new T.WebGLRenderer();r.setSize(512,320);r.outputColorSpace=T.LinearSRGBColorSpace;
 const scene=new T.Scene();scene.background=new T.Color(.24,.4,.67);
 const read=new T.WebGLRenderTarget(512,320,{depthTexture:new T.DepthTexture(512,320)}),write=new T.WebGLRenderTarget(512,320);
 const pass=new AtmospherePass();pass.setSize(512,320);const light=new T.DirectionalLight(0xffffff,3);light.position.set(-250,350,200);
 const settings={...renderDefaults,quality:'balanced',atmosphere:{...renderDefaults.atmosphere,clouds:true,cloudHeight:100,cloudThickness:90,cloudSize:100,coverage:.55,animate:false}};
 const output=[];for(const [name,pos,target] of [['below',[0,20,0],[0,150,0]],['above',[0,400,0],[0,140,0]],['horizon',[0,80,250],[0,90,0]],['inside',[0,145,250],[0,145,0]],['new-seed',[0,400,0],[0,140,0]],['bodies',[0,80,250],[0,90,0]],['classic',[0,80,250],[0,90,0]]]){
   if(name==='new-seed')settings.atmosphere.cloudSeed=3718;
   if(name==='bodies'||name==='classic'){settings.atmosphere.cloudSeed=42;settings.atmosphere.cloudModel=name;}
   const camera=new T.PerspectiveCamera(65,512/320,1,30000);camera.position.fromArray(pos);camera.lookAt(new T.Vector3(...target));camera.updateMatrixWorld();
   r.setRenderTarget(read);r.render(scene,camera);pass.configure(settings,camera,read.depthTexture,light,0,500);pass.render(r,write,read);
   const bytes=new Uint8Array(512*320*4);r.readRenderTargetPixels(write,0,0,512,320,bytes);let binary='';for(let i=0;i<bytes.length;i+=32768)binary+=String.fromCharCode(...bytes.subarray(i,i+32768));output.push({name,data:btoa(binary)});
 }
 pass.dispose();read.dispose();write.dispose();r.dispose();return output;
 }''')
 assert not errors,errors
 for v in images:Image.frombytes('RGBA',(512,320),base64.b64decode(v['data'])).transpose(Image.Transpose.FLIP_TOP_BOTTOM).save(out/(v['name']+'.png'))
 assert images[1]['data']!=images[4]['data'],'Seed did not change cloud distribution'
 assert len({images[i]['data'] for i in [2,5,6]})==3,'Cloud models did not produce distinct forms'
 b.close();print(json.dumps(dict(views=[v['name'] for v in images],errors=errors)))
