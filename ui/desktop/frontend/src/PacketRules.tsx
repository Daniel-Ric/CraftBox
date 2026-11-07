import { useState, useEffect } from "react";
import { Plus, Trash2 } from "lucide-react";
import { api } from "./bridge";
import type { PacketRule, PacketStatus } from "./packet-inspector-model";
export default function PacketRules({
  status,
  onChanged,
  onError,
}: {
  status: PacketStatus;
  onChanged: () => void;
  onError: (e: unknown) => void;
}) {
  const [rules, setRules] = useState<PacketRule[]>([]),
    [loading, setLoading] = useState(true),
    [selected, setSelected] = useState(""),
    [dirty, setDirty] = useState(false);
  useEffect(() => {
    let stopped = false;
    void api
      .GetPacketRules()
      .then((value) => {
        if (!stopped) {
          setRules(value);
          setLoading(false);
        }
      })
      .catch((e) => {
        if (!stopped) onError(e);
      });
    return () => {
      stopped = true;
    };
  }, []);
  const current = rules.find((r) => r.id === selected) ?? rules[0];
  const edit = (patch: Partial<PacketRule>) => {
    setRules((rows) =>
      rows.map((r) => (r.id === current?.id ? { ...r, ...patch } : r)),
    );
    setDirty(true);
  };
  return (
    <div className="pi-rule-workspace">
      <aside>
        <button
          disabled={loading}
          onClick={() => {
            const id = crypto.randomUUID();
            setRules((r) => [
              ...r,
              {
                id,
                label: "New watchpoint",
                enabled: false,
                name: "Text",
                direction: "",
                search: "",
                action: "bookmark",
                payload: "{}",
                delayMS: 0,
                preMS: 5000,
                postMS: 5000,
                cooldownMS: 5000,
                hits: 0,
              },
            ]);
            setSelected(id);
            setDirty(true);
          }}
        >
          <Plus size={14} />
          New rule
        </button>
        {rules.map((r) => (
          <button
            className={r.id === current?.id ? "selected" : ""}
            key={r.id}
            onClick={() => setSelected(r.id)}
          >
            <strong>{r.label || r.name}</strong>
            <small>
              {r.action} · {r.enabled ? "Enabled" : "Disabled"} ·{" "}
              {status.rules.find((v) => v.id === r.id)?.hits ?? r.hits} hits
            </small>
          </button>
        ))}
      </aside>
      <section>
        {current ? (
          <>
            <header>
              <h3>{current.label || current.name}</h3>
              <button
                aria-label="Delete rule"
                onClick={() => {
                  setRules((r) => r.filter((v) => v.id !== current.id));
                  setDirty(true);
                }}
              >
                <Trash2 size={14} />
              </button>
            </header>
            <label>
              Rule name
              <input
                value={current.label}
                onChange={(e) => edit({ label: e.target.value })}
              />
            </label>
            <label>
              Packet type
              <input
                value={current.name}
                onChange={(e) => edit({ name: e.target.value })}
                placeholder="Exact type, or use search below"
              />
            </label>
            <label>
              Direction
              <select
                value={current.direction}
                onChange={(e) => edit({ direction: e.target.value })}
              >
                <option value="">Both directions</option>
                <option value="serverbound">Serverbound</option>
                <option value="clientbound">Clientbound</option>
              </select>
            </label>
            <label>
              Condition
              <input
                value={current.search}
                onChange={(e) => edit({ search: e.target.value })}
                placeholder="actor:123, size>100, or text"
              />
            </label>
            <label>
              Action
              <select
                value={current.action}
                onChange={(e) => edit({ action: e.target.value })}
              >
                <option value="bookmark">
                  Bookmark matching received packet
                </option>
                <option value="trigger">Record a trigger window</option>
                <option value="block">Block forwarding</option>
                <option value="edit">Replace decoded packet fields</option>
                <option value="delay">Delay forwarding</option>
              </select>
            </label>
            {current.action === "edit" && (
              <label>
                Complete replacement JSON
                <textarea
                  rows={12}
                  value={current.payload}
                  onChange={(e) => edit({ payload: e.target.value })}
                />
              </label>
            )}
            {current.action === "delay" && (
              <label>
                Delay (milliseconds, up to 5,000)
                <input
                  type="number"
                  min={0}
                  max={5000}
                  value={current.delayMS}
                  onChange={(e) => edit({ delayMS: Number(e.target.value) })}
                />
              </label>
            )}
            <label className="pi-check">
              <input
                type="checkbox"
                checked={current.enabled}
                onChange={(e) => edit({ enabled: e.target.checked })}
              />
              Enabled
            </label>
            <p>
              Bookmarks and triggers observe received packets, including the
              handshake. Blocking, editing and delay apply to decoded forwarding
              after spawn and require intervention mode.
            </p>
            {current.action === "trigger" && (
              <>
                <label>
                  Pre-roll (milliseconds, up to 30,000)
                  <input
                    type="number"
                    min={0}
                    max={30000}
                    value={current.preMS ?? 5000}
                    onChange={(e) => edit({ preMS: Number(e.target.value) })}
                  />
                </label>
                <label>
                  Post-roll (milliseconds, up to 30,000)
                  <input
                    type="number"
                    min={0}
                    max={30000}
                    value={current.postMS ?? 5000}
                    onChange={(e) => edit({ postMS: Number(e.target.value) })}
                  />
                </label>
                <label>
                  Cooldown (milliseconds)
                  <input
                    type="number"
                    min={1000}
                    max={3600000}
                    value={current.cooldownMS ?? 5000}
                    onChange={(e) =>
                      edit({ cooldownMS: Number(e.target.value) })
                    }
                  />
                </label>
                <p>
                  Triggers save received traffic before and after a match,
                  including while continuous recording is paused. The pre-roll
                  buffer is limited to 16 MiB; up to eight windows can record
                  together.
                </p>
              </>
            )}
            <button
              disabled={!dirty}
              className="pi-primary"
              onClick={() => {
                void api
                  .SetPacketRules(rules)
                  .then(() => {
                    setDirty(false);
                    onChanged();
                  })
                  .catch(onError);
              }}
            >
              Save rules
            </button>
          </>
        ) : (
          <div className="pi-empty">
            <h3>No packet rules</h3>
            <p>Add a watchpoint or a forwarding rule.</p>
          </div>
        )}
      </section>
    </div>
  );
}
