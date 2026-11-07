import type {Option} from "./bridge";
import type {WorkspacePreferences} from "./workspace-preferences";

export function connectionValues(options: Pick<Option,"flag"|"default">[], current: Record<string,string>, prefs: Pick<WorkspacePreferences,"destinationHostname"|"lanDiscovery"|"localConnectionPort">) {
  const values = {...current};
  if(options.some(o=>o.flag==="hostname-alias"))values["hostname-alias"]=String(prefs.destinationHostname);
  if(options.some(o=>o.flag==="lan-discovery"))values["lan-discovery"]=String(prefs.lanDiscovery&&!prefs.destinationHostname);
  const listen=options.find(o=>o.flag==="listen");
  if(listen&&(!current.listen||current.listen===listen.default))values.listen="0.0.0.0:"+prefs.localConnectionPort;
  return values;
}
