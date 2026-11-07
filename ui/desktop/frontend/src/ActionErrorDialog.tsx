import FeedbackFlyout from "./FeedbackFlyout";
import { useState } from "react";
import { Box, Check, Copy, FileText, Monitor, Server, X } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "./components/ui/dialog";
import { Button } from "./components/ui/button";
import { api } from "./bridge";
import { connectionErrorPresentation, type ConnectionFailure } from "./action-error";
import "./action-error.css";

export default function ActionErrorDialog({message,connection,sessionID,onClose,onCopy}:{message:string;connection?:ConnectionFailure;sessionID?:string;onClose:()=>void;onCopy:()=>void}){
 const [opening,setOpening]=useState(false),[logError,setLogError]=useState("");
 const info=connectionErrorPresentation(message,connection);
 async function openLog(){if(!sessionID||opening)return;setOpening(true);setLogError("");try{await api.OpenSessionLog(sessionID)}catch(error){setLogError(`The session log could not be opened: ${String(error)}`)}finally{setOpening(false)}}
 const localLabel=info.local==="connected"?"Connected":info.local==="failed"?"Connection failed":"Not verified";
 const serverLabel=info.server==="failed"?"Connection failed":"Not verified";
 return <Dialog open={!!message} onOpenChange={open=>{if(!open)onClose()}}>
  <DialogContent resizable={false} className={info.network?"connection-error-dialog":undefined} tone={info.network?undefined:"error"}>
   <DialogHeader><DialogTitle>{info.title}</DialogTitle><DialogDescription>{info.network?"Connection status":"Error details"}</DialogDescription></DialogHeader>
   <div className="error-body connection-error-body">
    {info.network&&<div className="connection-route" role="img" aria-label={`Minecraft to CraftBox: ${localLabel}. CraftBox to ${info.target}: ${serverLabel}.`}>
     <div className="connection-node" data-state={info.local}><div className="connection-node-icon"><Monitor size={25}/>{info.local==="connected"?<Check className="connection-node-status" size={14}/>:info.local==="failed"?<X className="connection-node-status" size={14}/>:null}</div><strong>Minecraft</strong><span>{localLabel}</span></div>
     <div className="connection-link" data-state={info.local}>{info.local==="failed"&&<X size={14}/>}</div>
     <div className="connection-node" data-state="running"><div className="connection-node-icon"><Box size={25}/></div><strong>CraftBox</strong><span>Running</span></div>
     <div className="connection-link" data-state={info.server}>{info.server==="failed"&&<X size={14}/>}</div>
     <div className="connection-node" data-state={info.server}><div className="connection-node-icon"><Server size={25}/>{info.server==="failed"&&<X className="connection-node-status" size={14}/>}</div><strong title={info.target}>{info.target}</strong><span>{serverLabel}</span></div>
    </div>}
    <p className="connection-error-summary">{info.summary}</p>
    {info.network&&<p className="connection-error-hint">{info.hint}</p>}
    {info.summary!==message&&<details className="connection-error-technical"><summary>Show technical details</summary><pre className="error-details">{message}</pre></details>}
    {logError&&<FeedbackFlyout title="Session log">{logError}</FeedbackFlyout>}
   </div>
   <DialogFooter>
    <Button variant="outline" className="connection-open-log" disabled={!sessionID||opening} title={sessionID?"Open the log for this application session":"Session logging is unavailable"} onClick={()=>void openLog()}><FileText/>{opening?"Opening log…":"Open session log"}</Button>
    <Button variant="outline" onClick={onCopy}><Copy/>Copy details</Button><Button onClick={onClose}>Close</Button>
   </DialogFooter>
  </DialogContent>
 </Dialog>;
}
