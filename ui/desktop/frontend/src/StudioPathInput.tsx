import StudioInput,{type StudioInputProps} from "./StudioInput";
import {useEffect,useRef} from "react";
import {useLocalPathChecks} from "./path-validation";
export default function StudioPathInput({kind,onValidityChange,error,...props}:StudioInputProps&{value:string;kind:"file"|"directory";onValidityChange?:(valid:boolean)=>void}){
 const callback=useRef(onValidityChange);callback.current=onValidityChange;
 const {pending,error:pathError}=useLocalPathChecks([{id:"field",value:props.value,kind}],!props.disabled&&!props.readOnly).field;
 const message=error||pathError;
 useEffect(()=>{callback.current?.(!pending&&!message)},[pending,message]);
 return <StudioInput {...props} pending={pending} error={message}/>;
}
