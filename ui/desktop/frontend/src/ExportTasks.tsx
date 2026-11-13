import FeedbackFlyout from "./FeedbackFlyout";
import {useEffect,useRef,useState,useSyncExternalStore} from "react";
import {Activity,CheckCircle2,Clock,LoaderCircle,Minimize2,TriangleAlert,X} from "lucide-react";
import {Dialog,DialogContent,DialogHeader,DialogTitle,DialogDescription} from "./components/ui/dialog";
import {exportTasks,type ExportTask} from "./export-tasks";
import "./export-tasks.css";

const active=(task:ExportTask)=>task.status==="running"||task.status==="cancelling";
function duration(ms:number){const s=Math.max(0,Math.floor(ms/1000));return `${Math.floor(s/60)}:${String(s%60).padStart(2,"0")}`}
export default function ExportTasks(){
  const tasks=useSyncExternalStore(exportTasks.subscribe,exportTasks.snapshot);
  const running=tasks.filter(active),last=tasks[0];
  const [open,setOpen]=useState(false),[clock,setClock]=useState(Date.now());const shown=useRef(0);
  useEffect(()=>{if(last&&last.id>shown.current){shown.current=last.id;setOpen(true)}},[last?.id]);
  useEffect(()=>{if(!running.length)return;const id=setInterval(()=>setClock(Date.now()),1000);return()=>clearInterval(id)},[running.length]);
  const hasFailure=tasks.some(task=>task.status==="failed");
  const hasSuccessful=tasks.some(task=>task.status==="succeeded"||task.status==="cancelled");
  useEffect(()=>{if(running.length||!hasSuccessful)return;const id=setTimeout(()=>{if(!hasFailure)setOpen(false);exportTasks.clearSuccessful()},1600);return()=>clearTimeout(id)},[running.length,last?.id,last?.status,hasFailure,hasSuccessful]);
  if(!tasks.length)return null;
  const visible=running.length?running:tasks.slice(0,5);
  return <>
    <button className="export-task-trigger" aria-label="Show export tasks" title="Export tasks" onClick={()=>setOpen(true)}>
      {running.length?<LoaderCircle size={13} className="spin"/>:<Activity size={13}/>}
      <span>{running.length>1?`${running.length} exports`:running[0]?.title??"Exports"}</span>
      {running.length===1&&running[0].progress!==undefined&&<span className="export-task-mini"><i style={{width:`${running[0].progress*100}%`}}/></span>}
    </button>
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="export-progress-dialog" showCloseButton={false}>
        <DialogHeader><DialogTitle>{running.length>1?"Export tasks":visible[0]?.title??"Exports"}</DialogTitle><DialogDescription>{running.length?"Working in the background":"Recent exports"}</DialogDescription></DialogHeader>
        <div className="export-progress-list">
          {visible.map(task=><section className="export-progress-item" key={task.id} data-status={task.status}>
            <div className="export-progress-title">{active(task)?<LoaderCircle size={14} className="spin"/>:task.status==="succeeded"?<CheckCircle2 size={14}/>:task.status==="failed"?<TriangleAlert size={14}/>:<X size={14}/>}<strong>{task.title}</strong><span>{task.progress===undefined?"":`${Math.round(task.progress*100)}%`}</span></div>
            <p>{task.detail.replace(/[_-]/g," ")}</p>
            <div className={`export-progress-bar ${active(task)&&task.progress===undefined?"indeterminate":""}`} role="progressbar" aria-label={task.title} aria-valuemin={0} aria-valuemax={100} aria-valuenow={task.progress===undefined?undefined:Math.round(task.progress*100)}><i style={{width:task.progress===undefined?(active(task)?"30%":"0"): `${task.progress*100}%`}}/></div>
            <div className="export-progress-meta"><span><Clock size={11}/>{duration((task.finished??clock)-task.started)}{task.total!==undefined&&task.total>0?` · ${task.completed??0} / ${task.total}`:""}{task.bytes?` · ${(task.bytes/1048576).toFixed(1)} MB`:""}</span>{task.cancel&&<button className="studio-button" onClick={task.cancel}>Cancel</button>}</div>
            {task.error&&<FeedbackFlyout title={`${task.title} failed`}>{task.error}</FeedbackFlyout>}
          </section>)}
        </div>
        <footer className="export-progress-footer">{!running.length&&<button className="studio-button" onClick={()=>{exportTasks.clearCompleted();setOpen(false)}}>Clear completed</button>}<button className="studio-button" onClick={()=>setOpen(false)}>{running.length?<><Minimize2 size={12}/>Run in background</>:"Close"}</button></footer>
      </DialogContent>
    </Dialog>
  </>;
}
