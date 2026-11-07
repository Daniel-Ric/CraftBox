import {useEffect,useMemo,useState} from "react";
import {api} from "./bridge";
export type LocalPathField={id:string;value:string;kind:"file"|"directory"};
export function useLocalPathChecks(paths:LocalPathField[],enabled=true){
 const signature=JSON.stringify(paths);
 const request=useMemo(()=>({paths,enabled}),[signature,enabled]);
 const [result,setResult]=useState<{request:typeof request;errors:Record<string,string>}>();
 useEffect(()=>{
  if(!request.enabled)return;
  let cancelled=false;
  const timer=setTimeout(()=>{
   for(const p of request.paths){
    if(!p.value)continue;
    const finish=(error:string)=>{if(!cancelled)setResult(old=>({request,errors:{...(old?.request===request?old.errors:{}),[p.id]:error}}))};
    void Promise.resolve().then(()=>api.ValidateLocalPath(p.value,p.kind)).then(()=>finish(""),e=>finish(e instanceof Error?e.message:String(e)));
   }
  },250);
  return()=>{cancelled=true;clearTimeout(timer)};
 },[request]);
 const checks:Record<string,{pending:boolean;error:string}>={};
 for(const p of request.paths){
  const error=result?.request===request?result.errors[p.id]:undefined;
  checks[p.id]=!enabled||!p.value?{pending:false,error:""}:{pending:error===undefined,error:error??""};
 }
 return checks;
}
