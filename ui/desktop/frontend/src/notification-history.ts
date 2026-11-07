export type NotificationTone = "success" | "info" | "warning" | "error";
export type NotificationValue = {
  tone: NotificationTone;
  title: string;
  description?: string;
  duration?: number;
  id?: string;
  action?: { label: string; onClick: () => void | Promise<unknown> };
};
export type NotificationEntry = NotificationValue & {
  key: number;
  time: number;
  count: number;
  read: boolean;
};

// Deliberately session-local: old callbacks and potentially private error details
// must not be serialized to disk or restored against another project's state.
export class NotificationHistory {
  private entries: readonly NotificationEntry[] = [];
  private listeners = new Set<() => void>();
  private sequence = 0;
  private limit: number;
  private now: () => number;
  constructor(limit = 100, now = Date.now) {this.limit=limit;this.now=now;}
  snapshot = () => this.entries;
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  };
  private update(entries: readonly NotificationEntry[]) {
    this.entries = entries;
    for (const listener of this.listeners) listener();
  }
  record(value: NotificationValue) {
    const previous = value.id ? this.entries.find(e => e.id === value.id) : undefined;
    const same = previous && previous.tone === value.tone && previous.title === value.title && previous.description === value.description;
    const entry: NotificationEntry = {
      ...value,
      action: value.action ? { ...value.action } : undefined,
      key: same ? previous.key : ++this.sequence,
      time: this.now(), count: same ? previous.count + 1 : 1, read: false,
    };
    this.update([entry, ...this.entries.filter(e => e.key !== entry.key)].slice(0, this.limit));
    return entry;
  }
  markRead(key: number) {
    if (this.entries.some(e => e.key === key && !e.read)) this.update(this.entries.map(e => e.key === key ? { ...e, read: true } : e));
  }
  markAllRead() {
    if (this.entries.some(e => !e.read)) this.update(this.entries.map(e => ({ ...e, read: true })));
  }
  remove(key: number) { this.update(this.entries.filter(e => e.key !== key)); }
  clear() { if (this.entries.length) this.update([]); }
}
export const notificationHistory = new NotificationHistory();
