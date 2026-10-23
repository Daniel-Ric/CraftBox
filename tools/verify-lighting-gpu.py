"""Actual WebGL checks for emitter lighting, shadow occlusion and local scattering."""
from pathlib import Path
import base64,json
from PIL import Image,ImageChops,ImageStat
from playwright.sync_api import sync_playwright
out=Path('tmp/lighting/gpu');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
 b=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
 page=b.new_page();errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
 page.goto('http://127.0.0.1:5204')
 result=page.evaluate('''async()=>{
 const T=await import('/node_modules/three/build/three.module.js'),{LightingRig,installEmission}=await import('/src/render-lighting.ts'),{AtmospherePass}=await import('/src/atmosphere-volume.ts'),{renderDefaults}=await import('/src/render-settings.ts');
 const r=new T.WebGLRenderer();r.setSize(512,320);r.outputColorSpace=T.LinearSRGBColorSpace;r.shadowMap.enabled=true;r.shadowMap.type=T.PCFShadowMap;r.shadowMap.autoUpdate=false;
 const scene=new T.Scene();scene.background=new T.Color(.015,.02,.03);scene.add(new T.AmbientLight(0xffffff,.15));
 const camera=new T.PerspectiveCamera(50,512/320,.1,200);camera.position.set(11,10,13);camera.lookAt(0,1,0);camera.updateMatrixWorld();
 const rig=new LightingRig();scene.add(rig.group);rig.setChunk('0,0',[{position:[0,2,0],level:15,color:[1,.6,.2]}],new T.Vector3());
 const make=(geometry,pos,emission=0)=>{geometry.setAttribute('blockEmission',new T.BufferAttribute(new Float32Array(geometry.attributes.position.count).fill(emission),1));const material=new T.MeshStandardMaterial({color:0xcccccc,roughness:1,side:T.DoubleSide});installEmission(material,rig.emission);const mesh=new T.Mesh(geometry,material);mesh.position.fromArray(pos);mesh.castShadow=true;mesh.receiveShadow=true;mesh.customDistanceMaterial=rig.depth;scene.add(mesh);return mesh;};
 make(new T.BoxGeometry(18,.2,16),[0,-.1,0]);const wall=make(new T.BoxGeometry(.25,5,8),[2,2.5,0]);make(new T.BoxGeometry(1,1,1),[0,2,0],1);
 const bounds=new T.Box3(new T.Vector3(-9,0,-8),new T.Vector3(9,5,8)),sun=new T.DirectionalLight(0xffffff,0);sun.position.set(10,20,10);
 const read=new T.WebGLRenderTarget(512,320,{depthTexture:new T.DepthTexture(512,320)}),write=new T.WebGLRenderTarget(512,320),pass=new AtmospherePass();pass.setSize(512,320);pass.localLighting.rig=rig;
 const settings={...renderDefaults,sky:'none',sun:0,ambient:.15,blockLightShadows:1,atmosphere:{...renderDefaults.atmosphere,animate:false,fog:true,fogDensity:.8,fogHeight:3,fogFalloff:3,fogVariation:0}};
 const output=[];
 function render(name,fog=false){rig.update(settings,camera,bounds);r.shadowMap.needsUpdate=true;r.setRenderTarget(read);r.render(scene,camera);if(fog){pass.localLighting.power=settings.volumeLightPower;pass.configure(settings,camera,read.depthTexture,sun,0,40,bounds);pass.render(r,write,read);}const bytes=new Uint8Array(512*320*4);r.readRenderTargetPixels(fog?write:read,0,0,512,320,bytes);let binary='';for(let i=0;i<bytes.length;i+=32768)binary+=String.fromCharCode(...bytes.subarray(i,i+32768));output.push({name,data:btoa(binary)});}
 settings.blockLights=false;render('off');settings.blockLights=true;render('point-shadow');wall.castShadow=false;render('point-no-wall-shadow');wall.castShadow=true;
 const checker=new Uint8Array(4*4*4);for(let i=0;i<16;i++){checker[i*4]=checker[i*4+1]=checker[i*4+2]=255;checker[i*4+3]=(i%4<2)?0:255;}const texture=new T.DataTexture(checker,4,4);texture.needsUpdate=true;wall.material.map=texture;wall.material.alphaTest=.5;wall.material.needsUpdate=true;render('cutout-shadow');wall.material.map=null;wall.material.alphaTest=0;wall.material.needsUpdate=true;
 settings.volumeLightPower=0;render('fog-unlit',true);settings.volumeLightPower=1;render('fog-point-lit',true);
 settings.blockLights=false;settings.studioSpot=true;settings.spotPower=150;settings.spotX=-.7;settings.spotY=.8;settings.spotZ=.1;settings.spotTargetX=.5;settings.spotTargetY=-.15;settings.spotTargetZ=0;settings.volumeLightPower=0;render('fog-spot-unlit',true);settings.volumeLightPower=1;render('fog-spot-lit',true);
 settings.studioSpot=false;settings.sun=3.5;settings.elevation=55;settings.blockLights=false;render('day-off');settings.blockLights=true;render('day-on');settings.blockDaylightAdaptation=false;render('day-manual-glow');
 pass.dispose();rig.dispose();r.dispose();return output;
 }''')
 images={}
 for value in result:
  im=Image.frombytes('RGBA',(512,320),base64.b64decode(value['data'])).transpose(Image.Transpose.FLIP_TOP_BOTTOM).convert('RGB');im.save(out/(value['name']+'.png'));images[value['name']]=im
 delta=lambda a,b:sum(ImageStat.Stat(ImageChops.difference(images[a],images[b])).mean)
 metrics=dict(illumination=delta('off','point-shadow'),wallShadow=delta('point-shadow','point-no-wall-shadow'),cutout=delta('point-shadow','cutout-shadow'),pointScattering=delta('fog-unlit','fog-point-lit'),spotScattering=delta('fog-spot-unlit','fog-spot-lit'))
 assert metrics['illumination']>2,metrics
 assert metrics['wallShadow']>.2,metrics
 assert metrics['cutout']>.2,metrics
 assert metrics['pointScattering']>.1,metrics
 assert metrics['spotScattering']>.1,metrics
 metrics['daylightGlow']=delta('day-off','day-on');metrics['manualDaylightGlow']=delta('day-off','day-manual-glow')
 assert metrics['daylightGlow']<.01,metrics
 assert metrics['manualDaylightGlow']>2,metrics
 assert not errors,errors
 b.close();print(json.dumps(dict(metrics=metrics,errors=errors)))
