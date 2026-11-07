import React, {useState} from 'react';
import {createRoot} from 'react-dom/client';
import StudioFlyouts from '../src/StudioFlyouts';
import Status from '../src/MinecraftJoinStatus';
import Feedback from '../src/FeedbackFlyout';
import {Dialog,DialogContent,DialogTitle} from '../src/components/ui/dialog';
import {NewProjectDialog} from '../src/StartScreen';
import ConsoleDock from '../src/ConsoleDock';
import NotificationCenter from '../src/NotificationCenter';
import AssetStudio from '../src/AssetStudio';
import {DocumentScope} from '../src/document-scope';
import '../src/index.css';
import '../src/workspace.css';
import '../src/studio-foundations.css';
import '../src/studio-documents.css';
import '../src/adobe-ui.css';
import '../src/desktop-chrome.css';
import '../src/studio-focus.css';
import '../src/workbench.css';
import '../src/studio-sidebars.css';
import '../src/view-headings.css';
const proof=window as any;
proof.retries=0;proof.copied='';
document.documentElement.classList.add('dark');
const root=createRoot(document.getElementById('root')!);
function Fixture(){
 const [modal,setModal]=useState(false),[error,setError]=useState(false),[consoleOpen,setConsoleOpen]=useState(false);
 return <><StudioFlyouts/><div className="desktop-workspace workspace">
 <header className="desktop-chrome" style={{padding:16}}>CraftBox / Worlds</header>
 <Status phase={modal?'idle':'delayed'} address="http://zerix.ml:19132" launching={false} onRetry={()=>proof.retries++} onCopy={value=>proof.copied=value}/>
 <main style={{flex:1,padding:24}}><button onClick={()=>setModal(true)}>Test modal</button><input aria-label="Focus input"/><select aria-label="Focus select"><option>Select</option></select></main>
 <ConsoleDock open={consoleOpen} onOpenChange={setConsoleOpen} logs={[]} sessionName="CraftBox" running={false}/>
 <footer className="application-statusbar"><span>CraftBox</span><span>0 Worlds<NotificationCenter/></span></footer>
 <Dialog open={modal} onOpenChange={setModal}><DialogContent><DialogTitle>Test export</DialogTitle><button onClick={()=>setError(true)}>Create test error</button><button onClick={()=>setModal(false)}>Close test dialog</button>{error&&<Feedback title="Export failed">Disk full<button onClick={()=>proof.recovered=true}>Choose another folder</button></Feedback>}</DialogContent></Dialog>
 </div></>;
}
root.render(<Fixture/>);
proof.showProjects=()=>root.render(<><StudioFlyouts/><NewProjectDialog open onOpenChange={()=>{}} worlds={[]} onCreate={()=>{}} welcome={{onOpen:()=>{},recent:[
 'Zerix Network » Halloween Update Release | EU-NA-AS-OCE | discord.gg/mcbepvp – Still render (2)','Hoplite – Still render','Hive Games – Still render (2)','Hive Games – Still render','Another very long project name that should stay in its own card','Hive-Hub-Halloween – Still render'
].map((name,i)=>({id:String(i),name,page:'render' as const,worldId:'',edge:1920,aspect:16/9,look:'island',duration:12,fps:24,address:'',updated:Date.now()-i}))}}/></>);
proof.showAssets=()=>{
 proof.opened=[];
 proof.go={desktop:{App:{GetAssetLibrary:async()=>({items:['Pack one','Pack two','Pack three'].map((name,index)=>({id:'pack'+index,kind:'packs',name,version:'1.0',created:new Date().toISOString(),bytes:100,files:2,tags:[],image:'',source:'Capture',width:16,height:16})),bytes:300,directory:'Proof fixture'})}}};
 root.render(<><StudioFlyouts/><div className="desktop-workspace workspace workbench-view"><DocumentScope.Provider value={{id:'sidebar-proof',active:true,onOpen:id=>proof.opened.push(id),register:()=>{}}}><AssetStudio kind="packs" onSource={()=>{}}/></DocumentScope.Provider></div></>);
};

proof.showLocalWorlds=async()=>{
 const {default:LocalWorldsDialog}=await import('../src/LocalWorldsDialog');
 const world={id:'local',name:'Singleplayer Adventure',folder:'C:/Minecraft/worlds/adventure',installation:'Minecraft',modified:Date.now(),template:false,marketplace:false};
 proof.go={desktop:{App:{GetLocalWorlds:async()=>({worlds:[world]}),InspectLocalWorld:async()=>({world,packs:[{id:'bp',name:'Adventure add-on',version:'1.0.0',kind:'behavior',status:'included',embedded:false,requiredBy:'World'}],issues:[],portable:true,files:247,bytes:150000000,fingerprint:'proof'})}}};
 root.render(<><StudioFlyouts/><LocalWorldsDialog open onOpenChange={()=>{}}/></>);
};
