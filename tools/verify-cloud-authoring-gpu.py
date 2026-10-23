"""Render real overlapping authored cloud volumes, opening masks and depth moments."""
import argparse, base64, json
from pathlib import Path
from playwright.sync_api import sync_playwright
p=argparse.ArgumentParser();p.add_argument('--url',required=True);args=p.parse_args()
out=Path('tmp/cloud-authoring-gpu');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as pw:
 browser=pw.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=browser.new_page();errors=[]
 page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and any(s in m.text.lower() for s in ['shader','webgl','three']) else None)
 page.goto(args.url,wait_until='networkidle')
 result=page.evaluate('''async()=>{
  const T=await import('/node_modules/three/build/three.module.js'),{AtmospherePass}=await import('/src/atmosphere-volume.ts'),{renderDefaults}=await import('/src/render-settings.ts'),{createCloudLayer,keyCloudValue}=await import('/src/cloud-layers.ts');
  const r=new T.WebGLRenderer({preserveDrawingBuffer:true});r.setSize(256,192);r.outputColorSpace=T.LinearSRGBColorSpace;
  const scene=new T.Scene();scene.background=new T.Color(.04,.08,.15);
  const beauty=new T.WebGLRenderTarget(256,192,{depthTexture:new T.DepthTexture(256,192)}),target=new T.WebGLRenderTarget(256,192),pass=new AtmospherePass();pass.setSize(256,192);
  const sun=new T.DirectionalLight(0xffe1bb,3);sun.position.set(-300,300,400);
  const bounds=new T.Box3(new T.Vector3(-100,-100,-100),new T.Vector3(100,100,100));
  const settings={...renderDefaults,quality:'fine',atmosphere:{...renderDefaults.atmosphere,clouds:false,fog:false,animate:false,cloudAvoidance:0,cloudPreviewQuality:'medium',cloudExportQuality:'ultra'}};
  const camera=new T.PerspectiveCamera(46,256/192,1,5000);camera.position.set(0,30,550);camera.lookAt(0,0,0);camera.updateMatrixWorld();
  const images={};const render=(name,time=0,exporting=false)=>{r.setRenderTarget(beauty);r.render(scene,camera);pass.configure(settings,camera,beauty.depthTexture,sun,time,400,bounds,{timeline:true,exporting});pass.render(r,target,beauty);const pixels=new Uint8Array(256*192*4);r.readRenderTargetPixels(target,0,0,256,192,pixels);const canvas=document.createElement('canvas');canvas.width=256;canvas.height=192;const ctx=canvas.getContext('2d'),data=ctx.createImageData(256,192);for(let y=0;y<192;y++)data.data.set(pixels.subarray(y*256*4,(y+1)*256*4),(191-y)*256*4);ctx.putImageData(data,0,0);images[name]=canvas.toDataURL();return pixels;};
  const diff=(a,b)=>a.reduce((s,v,i)=>s+Math.abs(v-b[i]),0)/a.length;
  const clear=render('clear');
  const foreground={...createCloudLayer('volume'),position:[-.25,0,.3],size:[1.8,1.1,1.2],color:'#a8dcff',seed:42};
  settings.atmosphere.cloudLayers=[foreground];const one=render('one'),repeat=render('repeat');
  settings.atmosphere.cloudLayers.push({...createCloudLayer('volume'),size:[80,80,80],density:0});const empty=render('empty-volume');settings.atmosphere.cloudLayers=[foreground];
  const back={...createCloudLayer('bank'),position:[0,-.45,-.5],size:[3.5,.9,2.5],color:'#fff1d0',seed:16};settings.atmosphere.cloudLayers.push(back);const two=render('two');
  settings.atmosphere.cloudOpening={strength:1,center:[0,0],radius:[.6,.6],floor:-.7,softness:.25,irregularity:.4};const opened=render('opening');
  settings.atmosphere.cloudOpening.strength=0;
  let animated=keyCloudValue(foreground,'dissolve',0,0,24);animated=keyCloudValue(animated,'dissolve',1,1,24);settings.atmosphere.cloudLayers=[animated];const dissolved=render('dissolved',1);
  settings.atmosphere.cloudLayers=[foreground,back];const exported=render('ultra',0,true);
  const depth=new Uint16Array(pass.target.width*pass.target.height*4);r.readRenderTargetPixels(pass.target,0,0,pass.target.width,pass.target.height,depth,0,1);let depthPixels=0,meanDepth=0;for(let i=0;i<depth.length;i+=4){const z=T.DataUtils.fromHalfFloat(depth[i]),alpha=T.DataUtils.fromHalfFloat(depth[i+1]);if(alpha>.01&&z>0){depthPixels++;meanDepth+=z;}}
  const {DepthOfFieldPass}=await import('/src/render-depth-of-field.ts');
  const dof=new DepthOfFieldPass();dof.setSize(256,192);const blurredTarget=new T.WebGLRenderTarget(256,192);
  const optical={...settings,dofEnabled:true,dofDistance:meanDepth/Math.max(1,depthPixels),dofAperture:.7,dofWorldScale:1000,dofMaxBlur:16};
  const blur=(name,useCloudDepth)=>{dof.configure(optical,camera,beauty.depthTexture,false,useCloudDepth?pass.cloudDepthTexture:undefined);dof.render(r,blurredTarget,target,0,false);const pixels=new Uint8Array(256*192*4);r.readRenderTargetPixels(blurredTarget,0,0,256,192,pixels);const c=document.createElement('canvas');c.width=256;c.height=192;const ctx=c.getContext('2d'),data=ctx.createImageData(256,192);for(let y=0;y<192;y++)data.data.set(pixels.subarray(y*256*4,(y+1)*256*4),(191-y)*256*4);ctx.putImageData(data,0,0);images[name]=c.toDataURL();return pixels;};
  const terrainOnlyDepth=blur('dof-terrain-depth',false),volumeDepth=blur('dof-cloud-depth',true);
  const exportSteps=pass.uniforms.steps.value,exportResolution=pass.target.width/256;
  scene.background=null;r.setClearColor(0,0);const transparent=render('transparent');let translucentPixels=0,clearPixels=0;for(let i=3;i<transparent.length;i+=4){if(transparent[i]>0&&transparent[i]<255)translucentPixels++;if(transparent[i]===0)clearPixels++;}
  // An opaque near wall must occlude every cloud behind it, including authored volumes.
  const wall=new T.Mesh(new T.PlaneGeometry(5000,5000),new T.MeshBasicMaterial({color:0x426735}));wall.position.z=450;scene.add(wall);const occluded=render('occluded');settings.atmosphere.cloudLayers=[];const wallOnly=render('wall-only');
  const metrics={one:diff(clear,one),emptyVolumeDelta:diff(one,empty),overlap:diff(one,two),opening:diff(two,opened),repeat:diff(one,repeat),dissolved:diff(clear,dissolved),depthPixels,meanDepth:meanDepth/Math.max(1,depthPixels),dofDepthDifference:diff(terrainOnlyDepth,volumeDepth),exportSteps,exportResolution,translucentPixels,clearPixels,occlusionDelta:diff(occluded,wallOnly)};
  dof.dispose();blurredTarget.dispose();
  pass.dispose();beauty.dispose();target.dispose();r.dispose();return {images,metrics};
 }''')
 for name,data in result['images'].items():(out/(name+'.png')).write_bytes(base64.b64decode(data.split(',')[1]))
 (out/'metrics.json').write_text(json.dumps(result['metrics'],indent=2))
 print(json.dumps({'metrics':result['metrics'],'errors':errors}),flush=True)
 assert not errors,errors
 m=result['metrics'];assert m['one']>2 and m['overlap']>.3 and m['opening']>.2,m
 assert m['repeat']==0 and m['dissolved']<.01,m
 assert m['emptyVolumeDelta']==0,m
 assert m['depthPixels']>100 and 100<m['meanDepth']<900,m
 assert m['exportSteps']==192 and m['exportResolution']==1,m
 assert m['dofDepthDifference']>.01,m
 assert m['translucentPixels']>100 and m['clearPixels']>100 and m['occlusionDelta']<.01,m
 browser.close()
