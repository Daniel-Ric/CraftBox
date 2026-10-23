"""GPU proof for night moon alignment and world-space cloud clearance."""
from pathlib import Path
import json,base64
from PIL import Image,ImageChops,ImageStat
from playwright.sync_api import sync_playwright
out=Path('tmp/studio-atmosphere');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 b=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=b.new_page();page.set_default_timeout(180000);errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
 page.goto('http://127.0.0.1:5204')
 result=page.evaluate('''async()=>{
 const T=await import('/node_modules/.vite/deps/three.js'),{SkyBackgroundPass}=await import('/src/render-sky-background.ts'),{createNightSky}=await import('/src/night-sky.ts'),{AtmospherePass}=await import('/src/atmosphere-volume.ts'),{buildCloudClearance}=await import('/src/cloud-clearance.ts'),{renderDefaults}=await import('/src/render-settings.ts');
 const width=384,height=256,r=new T.WebGLRenderer({alpha:true});r.setSize(width,height);r.outputColorSpace=T.LinearSRGBColorSpace;
 const target=new T.WebGLRenderTarget(width,height,{depthTexture:new T.DepthTexture(width,height)}),write=new T.WebGLRenderTarget(width,height),sky=createNightSky(),pass=new SkyBackgroundPass();pass.setSize(width,height);
 const camera=new T.PerspectiveCamera(65,width/height,.1,2000),moon=new T.Vector3(.58,.74,.34).normalize();camera.lookAt(moon);camera.updateMatrixWorld();
 function bytes(rt){const data=new Uint8Array(width*height*4);r.readRenderTargetPixels(rt,0,0,width,height,data);let text='';for(let i=0;i<data.length;i+=32768)text+=String.fromCharCode(...data.subarray(i,i+32768));return btoa(text);}
 const images=[];for(const elevation of [0,30]){pass.configure(sky.background,camera,37,elevation,.28,moon);pass.render(r,write,target);images.push({name:'night-'+elevation,data:bytes(target)});}
 const terrain=new T.Group(),mesh=new T.Mesh(new T.BoxGeometry(30,40,30),new T.MeshStandardMaterial({color:0x55aa88}));mesh.position.y=20;terrain.add(mesh);
 const field=await buildCloudClearance(terrain),atmosphere=new AtmospherePass();atmosphere.setClearance(field);atmosphere.setSize(width,height);
 const view=new T.OrthographicCamera(-100,100,67,-67,.1,1000);view.position.set(0,220,0);view.up.set(0,0,-1);view.lookAt(0,0,0);view.updateMatrixWorld();
 const scene=new T.Scene();scene.background=new T.Color(.02,.025,.035);
 const light=new T.DirectionalLight(0xffffff,3);light.position.set(30,80,40);
 const settings={...renderDefaults,quality:'fast',atmosphere:{...renderDefaults.atmosphere,clouds:true,cloudModel:'banks',cloudDensity:2,cloudHeight:-20,cloudThickness:90,cloudSize:70,coverage:.95,cloudClearance:24,cloudAvoidance:0,animate:false,fog:false}};
 for(const strength of [0,1]){settings.atmosphere.cloudAvoidance=strength;r.setRenderTarget(target);r.render(scene,view);atmosphere.configure(settings,view,target.depthTexture,light,0,100,new T.Box3().setFromObject(terrain));atmosphere.render(r,write,target);images.push({name:'clearance-'+strength,data:bytes(write)});}
 return {width,height,images,field:{distance:field.distanceAt(0,0),min:field.min.toArray(),extent:field.extent,bottom:field.bottom},avoid:atmosphere.uniforms.aAvoidance.value.toArray(),region:atmosphere.uniforms.aClearanceRegion.value.toArray()};
 }''')
 print(json.dumps({k:v for k,v in result.items() if k!='images'}),flush=True)
 images={}
 for item in result['images']:
  img=Image.frombytes('RGBA',(result['width'],result['height']),base64.b64decode(item['data'])).transpose(Image.Transpose.FLIP_TOP_BOTTOM);img.save(out/(item['name']+'.png'));images[item['name']]=img
 w,h=result['width'],result['height'];moon=(w//2-2,h//2-2,w//2+2,h//2+2)
 delta=ImageStat.Stat(ImageChops.difference(images['night-0'].crop(moon),images['night-30'].crop(moon))).mean
 assert max(delta)<1,delta
 stars=sum(1 for y in range(h) for x in range(w) if abs(x-w/2)>10 and max(images['night-0'].getpixel((x,y))[:3])>35)
 assert stars>10,stars
 nightMean=ImageStat.Stat(images['night-0'].convert('RGB')).mean;assert max(nightMean)<15,nightMean
 difference=ImageChops.difference(images['clearance-0'],images['clearance-1']).convert('RGB')
 center=ImageStat.Stat(difference.crop((w//2-20,h//2-20,w//2+20,h//2+20))).mean
 outside=ImageStat.Stat(difference.crop((0,0,35,h))).mean
 print(json.dumps(dict(center=center,globalDelta=ImageStat.Stat(difference).mean,box=difference.getbbox())),flush=True)
 assert max(center)>2,center
 assert max(outside)<.5,outside
 assert not errors,errors
 metrics=dict(passed=True,moonAlignmentDelta=delta,stars=stars,nightMean=nightMean,clearanceCenterDelta=center,foregroundOutsideDelta=outside,errors=errors)
 (out/'result.json').write_text(json.dumps(metrics,indent=2),encoding='utf-8');print(json.dumps(metrics),flush=True);b.close()
