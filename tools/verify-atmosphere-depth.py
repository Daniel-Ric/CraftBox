"""GPU proof: geometry occludes volumes, orthographic/perspective work, time is deterministic."""
from pathlib import Path
import json
from playwright.sync_api import sync_playwright
out=Path('tmp/atmosphere/depth');out.mkdir(parents=True,exist_ok=True)
with sync_playwright() as p:
  browser=p.chromium.launch(channel='msedge',headless=True,args=['--use-angle=swiftshader','--enable-unsafe-swiftshader'])
  page=browser.new_page();errors=[]
  page.on('pageerror',lambda e:errors.append(str(e)))
  page.on('console',lambda m:errors.append(m.text) if m.type=='error' and ('shader' in m.text.lower() or 'WebGL' in m.text or 'THREE' in m.text) else None)
  page.goto('http://127.0.0.1:5204')
  result=page.evaluate('''async()=>{
    const T=await import('/node_modules/three/build/three.module.js');
    const {AtmospherePass}=await import('/src/atmosphere-volume.ts');
    const {renderDefaults}=await import('/src/render-settings.ts');
    const renderer=new T.WebGLRenderer({preserveDrawingBuffer:true});renderer.setSize(128,128);renderer.outputColorSpace=T.LinearSRGBColorSpace;
    const scene=new T.Scene();scene.background=new T.Color(.08,.1,.15);
    const cube=new T.Mesh(new T.BoxGeometry(90,90,90),new T.MeshBasicMaterial({color:new T.Color(.7,.12,.03)}));cube.position.y=110;scene.add(cube);
    const light=new T.DirectionalLight(0xffffff,3);light.position.set(300,500,300);
    const beauty=new T.WebGLRenderTarget(128,128,{depthTexture:new T.DepthTexture(128,128)}),write=new T.WebGLRenderTarget(128,128);
    const pass=new AtmospherePass();pass.setSize(128,128);
    const settings={...renderDefaults,quality:'fine',atmosphere:{...renderDefaults.atmosphere,clouds:true,coverage:.9,cloudHeight:0,cloudThickness:45,cloudSize:80,animate:true,windSpeed:20}};
    const bytes=()=>{const b=new Uint8Array(128*128*4);renderer.readRenderTargetPixels(write,0,0,128,128,b);return b;};
    const render=(camera,time)=>{renderer.setRenderTarget(beauty);renderer.render(scene,camera);pass.configure(settings,camera,beauty.depthTexture,light,time,500);pass.render(renderer,write,beauty);return bytes();};
    const metrics=[];
    for(const perspective of [false,true]){
      const camera=perspective?new T.PerspectiveCamera(48,1,1,2000):new T.OrthographicCamera(-150,150,150,-150,1,2000);
      camera.position.set(0,220,300);camera.lookAt(0,110,0);camera.updateMatrixWorld();
      const a=render(camera,0),b=render(camera,0),c=render(camera,3);
      let repeat=0,motion=0;for(let i=0;i<a.length;i++){repeat+=Math.abs(a[i]-b[i]);motion+=Math.abs(a[i]-c[i]);}
      renderer.setRenderTarget(beauty);renderer.render(scene,camera);const source=new Uint8Array(a.length);renderer.readRenderTargetPixels(beauty,0,0,128,128,source);
      const center=(64*128+64)*4;const occlusion=[...a.slice(center,center+3)].every((v,i)=>Math.abs(v-source[center+i])<=1);
      if(!occlusion)throw Error('Clouds leaked through foreground: '+[...a.slice(center,center+4)]+' vs '+[...source.slice(center,center+4)]);
      if(repeat!==0)throw Error('Same frame time was not deterministic: '+repeat);
      if(motion<100)throw Error('Wind did not move the volume: '+motion);
      settings.atmosphere.fog=true;settings.atmosphere.fogHeight=200;settings.atmosphere.fogDensity=1;
      const fog=render(camera,0);if(Math.abs(fog[center]-a[center])+Math.abs(fog[center+1]-a[center+1])<5)throw Error('Foreground fog missing');
      settings.atmosphere.fog=false;
      metrics.push({perspective,occlusion,repeat,motion,foreground:[...a.slice(center,center+4)],fog:[...fog.slice(center,center+4)]});
    }
    cube.visible=false;
    const upward=new T.OrthographicCamera(-80,80,80,-80,1,2000);upward.position.set(0,0,0);upward.lookAt(0,100,0);upward.updateMatrixWorld();
    Object.assign(settings.atmosphere,{cloudType:'stratus',cloudHeight:900,cloudThickness:2,coverage:1,fog:false,fogDensity:0});
    const thin=render(upward,0);settings.atmosphere.fog=true;const emptyFog=render(upward,0);
    let emptyEffectDelta=0;for(let i=0;i<thin.length;i++)emptyEffectDelta+=Math.abs(thin[i]-emptyFog[i]);
    if(emptyEffectDelta!==0)throw Error('Zero-density fog changed thin clouds: '+emptyEffectDelta);
    metrics.push({emptyEffectDelta});
    settings.atmosphere.fogDensity=.00001;
    const sparseFog=render(upward,0);let sparseDelta=0;for(let i=0;i<thin.length;i++)sparseDelta+=Math.abs(thin[i]-sparseFog[i]);
    if(sparseDelta>30000)throw Error('Separated fog slab erased thin cloud: '+sparseDelta);
    metrics.push({sparseDelta});
    Object.assign(settings.atmosphere,{cloudType:'cumulus',cloudHeight:100,cloudThickness:90,cloudSize:100,coverage:.75,fog:false});
    const grazing=new T.PerspectiveCamera(20,1,1,30000);
    const crossing=[];for(const y of [99.94,99.96]){grazing.position.set(0,y,0);grazing.lookAt(0,y+Math.sin(Math.PI/90)*300,-Math.cos(Math.PI/90)*300);grazing.updateMatrixWorld();crossing.push(render(grazing,0));}
    let boundaryDelta=0;for(let i=0;i<crossing[0].length;i++)boundaryDelta+=Math.abs(crossing[0][i]-crossing[1][i]);boundaryDelta/=crossing[0].length;
    if(boundaryDelta>.5)throw Error('Cloud layer boundary popped during camera movement: '+boundaryDelta);
    const seedA=render(grazing,0);settings.atmosphere.cloudSeed=819;const seedB=render(grazing,0);let seedDelta=0;for(let i=0;i<seedA.length;i++)seedDelta+=Math.abs(seedA[i]-seedB[i]);if(seedDelta<100)throw Error('Seed had no visual effect');
    metrics.push({boundaryDelta,seedDelta});
    const {foliageAttributes,installFoliageLighting}=await import('/src/foliage-lighting.ts');
    const plantScene=new T.Scene();const plantGeo=new T.PlaneGeometry(90,90);plantGeo.translate(0,110,1/64);
    plantGeo.setAttribute('foliage',new T.BufferAttribute(foliageAttributes(plantGeo.attributes.position.array),1));
    const plantMaterial=new T.MeshStandardMaterial({color:0x50aa20,side:T.DoubleSide,roughness:1});
    const plant=new T.Mesh(plantGeo,plantMaterial);plantScene.add(plant);const sun=new T.DirectionalLight(0xffffff,3);sun.position.set(0,500,0);plantScene.add(sun,new T.AmbientLight(0xffffff,.05));
    const plantCamera=new T.OrthographicCamera(-60,60,60,-60,1,1000);plantCamera.position.set(0,110,200);plantCamera.lookAt(0,110,0);
    const readPlant=()=>{renderer.setRenderTarget(write);renderer.render(plantScene,plantCamera);const rgba=bytes();return [...rgba.slice((64*128+64)*4,(64*128+64)*4+3)];};
    const vineBefore=readPlant();installFoliageLighting(plantMaterial);plantMaterial.needsUpdate=true;const vineAfter=readPlant();
    if(vineAfter.reduce((a,b)=>a+b,0)<=vineBefore.reduce((a,b)=>a+b,0)*2)throw Error('Native vine remained dark: '+vineBefore+' -> '+vineAfter);
    metrics.push({vineBefore,vineAfter});plantGeo.dispose();plantMaterial.dispose();
    pass.dispose();beauty.dispose();write.dispose();cube.geometry.dispose();cube.material.dispose();renderer.dispose();
    return metrics;
  }''')
  assert not errors,errors
  (out/'verification.json').write_text(json.dumps(result,indent=2))
  browser.close();print(json.dumps(result),flush=True)
