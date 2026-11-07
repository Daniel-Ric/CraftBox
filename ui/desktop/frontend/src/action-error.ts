export function actionErrorSummary(value:string):string {
 if (/hostname alias requires a DNS server name|Destination hostname needs a valid server domain/i.test(value)) return "Destination hostname is enabled, but the selected address is an IP address, a local alias or an invalid domain. Enter a server domain such as play.example.com, or turn off Destination hostname in Preferences > Connection and try again.";
 if (/disconnect code 55\b/i.test(value) && /No reason was provided/i.test(value)) return "The host or server removed you from the world (code 55), but provided no reason. CraftBox cannot determine why.";
 if (/no SRTP Protection Profile/i.test(value)) return "The connection could not agree on a required security setting. The secure connection was stopped before Minecraft could sign in.";
 if (/start DTLS|DTLS.*handshake|TLS handshake/i.test(value)) return "The secure connection could not be established. The endpoints could not complete their security handshake.";
 if (/no such host|ENOTFOUND|name resolution/i.test(value)) return "The server address could not be found. Check the address and your internet connection.";
 if (/connection refused/i.test(value)) return "The connection was refused. Check the server address and port, then try again.";
 if (/i\/o timeout|context deadline exceeded|connection timed out/i.test(value)) return "The connection took too long to respond. Check your connection and try again.";
 if (/ICE.*failed|no network path|no candidate pairs/i.test(value)) return "A network path could not be established. A firewall, router or network compatibility issue may be preventing the connection.";
 if (/Windows permission.*destination hostname|UAC prompt/i.test(value)) return "Windows permission for the temporary hostname redirect was not granted. Start the connection again and approve the Windows UAC prompt, or turn off Destination hostname in Preferences → Connection.";
 if (/hosts|hostname/.test(value)&&/access is denied|permission denied/i.test(value)) return "Windows blocked the local hostname redirect. Approve the Windows permission prompt, or turn off Destination hostname in Preferences → Connection.";
 if (/https?:|http |status|stack|exception|\n/i.test(value)||value.length>180) return "The action could not be completed. Review the details below.";
 return value;
}

export type ConnectionFailure = {leg:"client"|"server";target:string;localReady:boolean};
export function connectionErrorPresentation(message:string, connection?:ConnectionFailure) {
 const network=!!connection||/DTLS|SRTP|ICE.*failed|read packet minecraft|dial minecraft|dial nethernet|start transports/i.test(message);
 const local=connection?.leg==="client"?"failed":connection?.localReady?"connected":"unknown";
 const server=connection?.leg==="server"?"failed":"unknown";
 let target="Target server";
 if(connection?.target){try{target=new URL(connection.target.includes("://")?connection.target:`https://${connection.target}`).hostname||target}catch{/* Keep a neutral label for Realms and other source kinds. */}}
 const summary=/receive minecraft.*Error:/i.test(message)?(connection?.leg==="client"?"Minecraft ended its connection with an error. Its technical reference is included below.":connection?.leg==="server"?"The server ended the connection with an error. Its technical reference is included below.":"The connection ended with an error. Its technical reference is included below."):actionErrorSummary(message);
 return {network,local,server,target,summary,title:network?"Connection could not be completed":"Could not complete this action",hint:connection?.leg==="client"?"Reconnect Minecraft to the local CraftBox address. The failure is on the Minecraft side of the proxy.":connection?.leg==="server"?"The affected connection is between CraftBox and the target server. If it happens again, include the session log when reporting it.":"The log contains more detail about where the connection stopped."};
}
