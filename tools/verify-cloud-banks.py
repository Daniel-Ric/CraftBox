"""GPU regression: bounded fog banks and low clouds with a rolling 3D silhouette."""
from pathlib import Path
import json, argparse
from playwright.sync_api import sync_playwright
out=Path('tmp/cloud-banks');out.mkdir(parents=True,exist_ok=True)
p=argparse.ArgumentParser();p.add_argument('--url',default='http://127.0.0.1:5204');args=p.parse_args()
with sync_playwright() as pw:
 browser=pw.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=browser.new_page(viewport=dict(width=900,height=600));errors=[]
 page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
 page.goto(args.url,wait_until='networkidle')
 result=page.evaluate('''async()=>{
  const T=await import('/node_modules/three/build/three.module.js');const {AtmospherePass}=await import('/src/atmosphere-volume.ts');const {renderDefaults}=await import('/src/render-settings.ts');
  const r=new T.WebGLRenderer({preserveDrawingBuffer:true});r.setSize(512,256);r.outputColorSpace=T.LinearSRGBColorSpace;r.setClearColor(0x264c79);
  const scene=new T.Scene();scene.background=new T.Color(.08,.18,.32);
  const beauty=new T.WebGLRenderTarget(512,256,{depthTexture:new T.DepthTexture(512,256)}),target=new T.WebGLRenderTarget(512,256),pass=new AtmospherePass();pass.setSize(512,256);
  const sun=new T.DirectionalLight(0xffd39a,3);sun.position.set(-400,200,50);
  const bounds=new T.Box3(new T.Vector3(-180,0,-180),new T.Vector3(180,240,180));
  const settings={...renderDefaults,quality:'fine',atmosphere:{...renderDefaults.atmosphere,clouds:true,cloudModel:'banks',cloudType:'stratocumulus',cloudReference:'bottom',cloudHeight:-120,cloudThickness:180,cloudSize:240,coverage:.88,cloudDensity:1.3,cloudExtent:2.8,detail:.65,erosion:.28,turbulence:.6,fog:false,animate:false}};
  const camera=new T.PerspectiveCamera(46,2,1,12000);camera.position.set(0,120,1100);camera.lookAt(0,-20,0);camera.updateMatrixWorld();
  const render=()=>{r.setRenderTarget(beauty);r.render(scene,camera);pass.configure(settings,camera,beauty.depthTexture,sun,0,720,bounds);pass.render(r,target,beauty);const bytes=new Uint8Array(512*256*4);r.readRenderTargetPixels(target,0,0,512,256,bytes);return bytes;};
  const screen=bytes=>{const c=document.createElement('canvas');c.width=512;c.height=256;const ctx=c.getContext('2d'),data=ctx.createImageData(512,256);for(let y=0;y<256;y++)data.data.set(bytes.slice(y*512*4,(y+1)*512*4),(255-y)*512*4);ctx.putImageData(data,0,0);return c.toDataURL();};
  const side=render(),repeat=render();let repeatDelta=0;for(let i=0;i<side.length;i++)repeatDelta+=Math.abs(side[i]-repeat[i]);
  // Keep the fog interval enabled while its contribution rounds to zero.
  settings.atmosphere.fog=true;settings.atmosphere.fogDensity=1e-8;const zeroFog=render();settings.atmosphere.fog=false;
  let zeroFogCloudDelta=0;for(let i=0;i<side.length;i++)zeroFogCloudDelta+=Math.abs(side[i]-zeroFog[i]);zeroFogCloudDelta/=side.length;
  settings.atmosphere.clouds=false;const clear=render();
  const tops=[],heights=[];for(let x=96;x<416;x++){let top=-1,bottom=256;for(let y=0;y<256;y++){const i=(y*512+x)*4;const delta=Math.abs(side[i]-clear[i])+Math.abs(side[i+1]-clear[i+1])+Math.abs(side[i+2]-clear[i+2]);if(delta>35){top=Math.max(top,y);bottom=Math.min(bottom,y);}}if(top>=0){tops.push(top);heights.push(top-bottom);}}
  const mean=a=>a.reduce((s,x)=>s+x,0)/Math.max(1,a.length),sd=a=>Math.sqrt(mean(a.map(x=>(x-mean(a))**2)));
  camera.position.set(5000,-20,1000);camera.lookAt(5000,-20,0);camera.updateMatrixWorld();const farClear=render();
  Object.assign(settings.atmosphere,{fog:true,fogType:'banks',fogReference:'bottom',fogHeight:0,fogFalloff:45,fogDensity:.15});const farFog=render();
  let farFogDelta=0;for(let i=0;i<farFog.length;i++)farFogDelta+=Math.abs(farFog[i]-farClear[i]);farFogDelta/=farFog.length;
  camera.position.set(0,120,1100);camera.lookAt(0,-20,0);camera.updateMatrixWorld();settings.atmosphere.clouds=true;const combined=render();
  settings.atmosphere.clouds=false;const fogA=render();Object.assign(settings.atmosphere,{cloudHeight:400,cloudSize:900,cloudExtent:8});const fogB=render();let disabledCloudDelta=0;for(let i=0;i<fogA.length;i++)disabledCloudDelta+=Math.abs(fogA[i]-fogB[i]);
  return {side:screen(side),combined:screen(combined),zeroFog:screen(zeroFog),metrics:{zeroFogCloudDelta,farFogDelta,disabledCloudDelta,repeatDelta,silhouetteColumns:tops.length,topVariation:sd(tops),meanThickness:mean(heights)}};
 }''')
 import base64
 for key in ['side','combined','zeroFog']:(out/(key+'.png')).write_bytes(base64.b64decode(result[key].split(',')[1]))
 (out/'metrics.json').write_text(json.dumps(result['metrics'],indent=2))
 print(json.dumps(dict(metrics=result['metrics'],errors=errors)),flush=True)
 assert not errors,errors
 m=result['metrics'];assert m['farFogDelta']<.01,('Bank fog extended to infinity',m)
 assert m['zeroFogCloudDelta']<.01,('Effectively transparent fog changed cloud sampling',m)
 assert m['repeatDelta']==0,m
 assert m['disabledCloudDelta']==0,('Disabled clouds changed fog',m)
 assert m['silhouetteColumns']>220,m
 assert m['topVariation']>3,m
 assert m['meanThickness']>25,m
 browser.close()
