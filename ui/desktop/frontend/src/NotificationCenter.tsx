import { useLayoutEffect, useRef, useState, useSyncExternalStore } from "react";
import { Popover } from "@base-ui/react/popover";
import { Bell, CheckCheck, Trash2, X, CircleCheck, CircleAlert, Info, TriangleAlert, Copy, RotateCcw, ArrowUpRight, LoaderCircle } from "lucide-react";
import { notificationHistory, type NotificationEntry, type NotificationTone } from "./notification-history";
import { notify } from "./StudioFlyouts";
import "./notification-center.css";

const symbols = { success: CircleCheck, info: Info, warning: TriangleAlert, error: CircleAlert };
export default function NotificationCenter() {
  const entries = useSyncExternalStore(notificationHistory.subscribe, notificationHistory.snapshot);
  const [open, setOpen] = useState(false), [filter, setFilter] = useState<"all" | "unread" | NotificationTone>("all"), [search, setSearch] = useState("");
  const trigger=useRef<HTMLButtonElement>(null),positioner=useRef<HTMLDivElement>(null);
  useLayoutEffect(()=>{
    if(!open)return;
    const update=()=>{
      if(!trigger.current||!positioner.current)return;
      const box=trigger.current.getBoundingClientRect();
      const scale=parseFloat(getComputedStyle(document.documentElement).zoom)||1;
      const below=box.top<160*scale;
      const style=positioner.current.style;
      style.setProperty("--notification-right",Math.max(12,innerWidth-box.right)/scale+"px");
      style.setProperty("--notification-top",below?(box.bottom/scale+8)+"px":"auto");
      style.setProperty("--notification-bottom",below?"auto":((innerHeight-box.top)/scale+8)+"px");
      style.setProperty("--notification-height",Math.max(100,Math.min(570,(below?innerHeight-box.bottom-24:box.top-24)/scale))+"px");
    };
    update();window.addEventListener("resize",update);
    const resize=new ResizeObserver(update);if(trigger.current)resize.observe(trigger.current);
    const mutation=new MutationObserver(update);mutation.observe(document.documentElement,{attributes:true,attributeFilter:["style"]});
    return ()=>{window.removeEventListener("resize",update);resize.disconnect();mutation.disconnect()};
  },[open]);
  const unread = entries.filter(e => !e.read).length;
  const visible = entries.filter(e => (filter === "all" || (filter === "unread" ? !e.read : e.tone === filter)) && (e.title + " " + (e.description ?? "")).toLocaleLowerCase().includes(search.toLocaleLowerCase().trim()));
  return <Popover.Root open={open} onOpenChange={setOpen}>
    <Popover.Trigger ref={trigger} className="notification-bell" aria-label={`Notifications (${unread} unread)`} title="Notification history">
      <Bell size={15}/>{unread > 0 && <span className="notification-badge">{unread > 99 ? "99+" : unread}</span>}
    </Popover.Trigger>
    <Popover.Portal>
      <Popover.Positioner ref={positioner} side="top" align="end" sideOffset={8} collisionPadding={12} className="notification-positioner">
        <Popover.Popup className="notification-center">
          <header className="notification-heading">
            <Popover.Title>Notifications</Popover.Title>
            <div>
              <button aria-label="Mark all notifications as read" title="Mark all as read" disabled={!unread} onClick={()=>notificationHistory.markAllRead()}><CheckCheck size={15}/></button>
              <button aria-label="Clear notification history" title="Clear history" disabled={!entries.length} onClick={()=>notificationHistory.clear()}><Trash2 size={15}/></button>
              <Popover.Close aria-label="Close notification history"><X size={15}/></Popover.Close>
            </div>
          </header>
          <Popover.Description className="notification-caption">This session · {entries.length} saved · {unread} unread</Popover.Description>
          <div className="notification-filters">
            <input type="search" aria-label="Search notifications" placeholder="Search notifications" value={search} onChange={e=>setSearch(e.target.value)}/>
            <select aria-label="Filter notifications" value={filter} onChange={e=>setFilter(e.target.value as typeof filter)}>
              <option value="all">All</option><option value="unread">Unread</option><option value="error">Errors</option><option value="warning">Warnings</option><option value="success">Success</option><option value="info">Information</option>
            </select>
          </div>
          <div className="notification-list" role="region" aria-label="Notification history" tabIndex={0}>
            {visible.length ? visible.map(entry=><HistoryEntry key={entry.key} entry={entry}/>) : <div className="notification-empty"><Bell size={24}/><strong>{entries.length ? "No matching notifications" : "No notifications yet"}</strong><p>{entries.length ? "Change the filter or search." : "Past flyouts will appear here."}</p></div>}
          </div>
          <footer className="notification-footer">Last 100 notifications · Cleared when the app closes</footer>
        </Popover.Popup>
      </Popover.Positioner>
    </Popover.Portal>
  </Popover.Root>;
}

function HistoryEntry({entry}: {entry: NotificationEntry}) {
  const [busy, setBusy] = useState(false);
  const Icon = symbols[entry.tone];
  const time = new Date(entry.time);
  async function invoke(operation:()=>unknown|Promise<unknown>) {
    setBusy(true); notificationHistory.markRead(entry.key);
    try { await operation(); } catch(error) { notify({tone:"error",title:"Action failed",description:error instanceof Error ? error.message : String(error)}); } finally { setBusy(false); }
  }
  return <article className="notification-entry" data-tone={entry.tone} data-unread={!entry.read}>
    <div className="notification-entry-top"><Icon size={15}/><strong>{entry.title}</strong>{entry.count > 1 && <span className="notification-repeat">×{entry.count}</span>}<time dateTime={time.toISOString()} title={time.toLocaleString()}>{time.toLocaleTimeString([], {hour:"2-digit",minute:"2-digit"})}</time><button aria-label={`Remove notification: ${entry.title}`} title="Remove" onClick={()=>notificationHistory.remove(entry.key)}><X size={13}/></button></div>
    {entry.description && <details onToggle={e=>{if(e.currentTarget.open)notificationHistory.markRead(entry.key)}}><summary>Details</summary><p>{entry.description}</p></details>}
    <div className="notification-entry-actions">
      {entry.action && <button disabled={busy} onClick={()=>void invoke(()=>entry.action!.onClick())}>{busy ? <LoaderCircle size={12} className="spin"/> : <ArrowUpRight size={12}/>} {entry.action.label}</button>}
      <button disabled={busy} onClick={()=>{notificationHistory.markRead(entry.key);notify(entry,{history:false})}}><RotateCcw size={12}/>Show again</button>
      <button disabled={busy} onClick={()=>void invoke(()=>navigator.clipboard.writeText(entry.title+(entry.description ? "\n"+entry.description : "")))}><Copy size={12}/>Copy</button>
      {!entry.read && <button disabled={busy} onClick={()=>notificationHistory.markRead(entry.key)}>Mark read</button>}
    </div>
  </article>;
}
