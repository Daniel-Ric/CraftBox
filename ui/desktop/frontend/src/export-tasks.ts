export type ExportTask = {
  id:number;title:string;detail:string;status:"running"|"cancelling"|"succeeded"|"cancelled"|"failed";
  started:number;finished?:number;progress?:number;completed?:number;total?:number;bytes?:number;error?:string;cancel?:()=>void;
};
export const isExportCancellation=(error:unknown)=>error instanceof Error&&error.name==="AbortError"||/context cancel|cancelled|canceled|aborted/i.test(String(error));
export class ExportTaskStore {
  private entries:readonly ExportTask[]=[];
  private listeners=new Set<()=>void>();
  private next=0;
  snapshot=()=>this.entries;
  subscribe=(listener:()=>void)=>{this.listeners.add(listener);return ()=>{this.listeners.delete(listener)}};
  private publish(entries:readonly ExportTask[]){this.entries=entries;for(const listener of this.listeners)listener()}
  private patch(id:number,patch:Partial<ExportTask>){this.publish(this.entries.map(t=>t.id===id?{...t,...patch}:t))}
  clearCompleted(){this.publish(this.entries.filter(t=>t.status==="running"||t.status==="cancelling"))}
  clearSuccessful(){this.publish(this.entries.filter(t=>t.status!=="succeeded"&&t.status!=="cancelled"))}
  begin(title:string,options:{cancelable?:boolean}={}){
    const id=++this.next,controller=new AbortController();let ended=false,callback:(()=>unknown|Promise<unknown>)|undefined;
    const cancel=()=>{
      if(ended||controller.signal.aborted||(!callback&&!options.cancelable))return;
      this.patch(id,{status:"cancelling",detail:"Cancelling…",cancel:undefined});controller.abort(new DOMException("Export cancelled","AbortError"));
      if(callback)Promise.resolve().then(callback).catch(()=>{if(!ended)this.patch(id,{status:"running",detail:"Could not cancel. Export is still running.",cancel:undefined})});
    };
    this.publish([{id,title,detail:"Preparing…",status:"running",started:Date.now(),cancel:options.cancelable?cancel:undefined},...this.entries.filter(t=>t.status==="running"||t.status==="cancelling"||t.id>this.next-30)]);
    const finish=(status:ExportTask["status"],error?:string)=>{if(ended)return;ended=true;this.patch(id,{status,error,finished:Date.now(),cancel:undefined,detail:status==="succeeded"?"Completed":status==="cancelled"?"Cancelled":"Export failed",progress:status==="succeeded"?1:undefined})};
    return {id,signal:controller.signal,cancel,
      update:(detail:string,completed?:number,total?:number,bytes?:number)=>{
        if(ended||controller.signal.aborted)return;
        const progress=typeof total==="number"&&Number.isFinite(total)&&total>0&&typeof completed==="number"&&Number.isFinite(completed)?Math.max(0,Math.min(1,completed/total)):undefined;
        const old=this.entries.find(t=>t.id===id);
        if(old&&old.detail===detail&&Math.floor((old.progress??-1)*100)===Math.floor((progress??-1)*100)&&old.bytes===bytes)return;
        this.patch(id,{detail,completed,total,bytes,progress});
      },
      setCancel:(value:()=>unknown|Promise<unknown>)=>{if(!ended&&!controller.signal.aborted){callback=value;this.patch(id,{cancel})}},
      complete:()=>finish("succeeded"),cancelled:()=>finish("cancelled"),
      fail:(error:unknown)=>finish(isExportCancellation(error)?"cancelled":"failed",isExportCancellation(error)?undefined:String(error)),
    };
  }
}
export type ExportTaskHandle=ReturnType<ExportTaskStore["begin"]>;
export const exportTasks=new ExportTaskStore();
export async function runExportTask<T>(title:string,work:(task:ExportTaskHandle)=>Promise<T>,options:{cancelable?:boolean}={}){
  const task=exportTasks.begin(title,options);
  try{const result=await work(task);task.complete();return result}catch(error){task.fail(error);throw error}
}

const saveExports=new Set(["SaveImage","SaveAssetModel","SaveRenderPNG","SaveCinematicFile","SaveMapPNG","SaveConsoleLog","SaveWorldComposition","SaveStudioProject","SaveStudioProjectForTab","SaveWorkspaceProject"]);
export function isExportMethod(name:string){return name.startsWith("Export")||saveExports.has(name)}
type RawBackend=Record<string,((...args:unknown[])=>Promise<unknown>)|undefined>;
const nativeExportFamilies=new Set<string>();
export async function trackBackendExport(name:string,invoke:()=>Promise<unknown>,backend:RawBackend,jobID?:string){
  const titles:Record<string,string>={ExportModel:"Export 3D model",ExportRenderGLB:"Export 3D model",ExportWorldCopy:"Export world",ExportAssets:"Export assets",ExportPacketCapture:"Export packets",SaveImage:"Save image",SaveRenderPNG:"Save render",SaveCinematicFile:"Save video or sequence",SaveMapPNG:"Save map",SaveConsoleLog:"Export console",SaveAssetModel:"Export asset model",ExportMarketplaceItems:"Export marketplace data",ExportScene:"Export scene",SaveWorldComposition:"Save combined world"};
  const model=["ExportModel","ExportRenderGLB"].includes(name),world=name==="ExportWorldCopy",localWorld=name==="ExportLocalWorld";
  const family=model?"model":world?"world":localWorld?"local-world":undefined;
  if(family&&nativeExportFamilies.has(family))throw new Error("An export of this type is already running. Wait for it to finish.");
  if(family)nativeExportFamilies.add(family);
  const project=["SaveStudioProject","SaveStudioProjectForTab","SaveWorkspaceProject"].includes(name);
  const task=exportTasks.begin(project?"Save project":localWorld?"Copy singleplayer world":titles[name]??"Export file");
  const progress=jobID?(model?backend.GetModelExportProgress:world?backend.GetWorldExportProgress:localWorld?backend.GetLocalWorldExportProgress:undefined):undefined;
  const cancel=model?backend.CancelModelExportJob:world?backend.CancelWorldExportJob:localWorld?backend.CancelLocalWorldExportJob:undefined;
  let active=true,polling=false;
  const poll=async()=>{if(!active||polling||!progress)return;polling=true;try{const value=await progress.call(backend) as {jobId?:string;running?:boolean;stage?:string;completed?:number;done?:number;total?:number;bytes?:number};if(active&&value.running&&value.jobId===jobID){task.update(value.stage||"Exporting…",value.completed??value.done,value.total,value.bytes);if(cancel)task.setCancel(()=>cancel.call(backend,jobID))}}catch{/* Progress must not fail a successful export. */}finally{polling=false}};
  let timer:ReturnType<typeof setInterval>|undefined;
  try{
    // The native progress endpoints describe one job per family. Reserve the
    // family before opening a destination dialog, and never adopt an older job.
    if(progress){const previous=await progress.call(backend) as {running?:boolean};if(previous.running)throw new Error("An export of this type is already running. Wait for it to finish.")}
    const promise=invoke();if(progress)timer=setInterval(()=>void poll(),400);void poll();const result=await promise;if(result===""||result===null)task.cancelled();else task.complete();return result;
  }catch(error){task.fail(error);throw error}finally{active=false;if(timer!==undefined)clearInterval(timer);if(family)nativeExportFamilies.delete(family)}
}
