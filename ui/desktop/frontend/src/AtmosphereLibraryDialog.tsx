import { useState } from "react";
import { Cloud, Haze, Search } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  cloudLibrary,
  fogLibrary,
  type AtmospherePreset,
} from "./atmosphere-library";
export default function AtmosphereLibraryDialog({
  kind,
  onClose,
  onApply,
}: {
  kind: "cloud" | "fog" | null;
  onClose: () => void;
  onApply: (preset: AtmospherePreset) => void;
}) {
  const [query, setQuery] = useState("");
  const library = kind === "cloud" ? cloudLibrary : fogLibrary;
  return (
    <Dialog
      open={!!kind}
      onOpenChange={(open) => {
        if (!open) {
          setQuery("");
          onClose();
        }
      }}
    >
      <DialogContent className="atmosphere-library-window">
        <DialogHeader>
          <DialogTitle>
            {kind === "cloud" ? "Cloud library" : "Fog library"}
          </DialogTitle>
          <DialogDescription>
            Choose a preset, then refine its shape, color and placement in the inspector.
          </DialogDescription>
        </DialogHeader>
        <div className="studio-dialog-body atmosphere-library-body">
          <label className="atmosphere-search">
            <Search size={14} />
            <input
              aria-label="Browse atmosphere presets"
              placeholder="Search presets …"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </label>
          <div className="atmosphere-library-grid">
            {library
              .filter((p) =>
                (p.name + " " + p.description)
                  .toLowerCase()
                  .includes(query.toLowerCase()),
              )
              .map((p) => (
                <button
                  key={p.id}
                  className="atmosphere-preset"
                  onClick={() => {
                    onApply(p);
                    setQuery("");
                    onClose();
                  }}
                >
                  <span
                    className={`atmosphere-preset-icon atmosphere-preset-${p.id}`}
                  >
                    {kind === "cloud" ? (
                      <Cloud size={30} />
                    ) : (
                      <Haze size={30} />
                    )}
                  </span>
                  <strong>{p.name}</strong>
                  <span>{p.description}</span>
                </button>
              ))}
          </div>
        </div>
        <DialogFooter>
          <span>
            {library.length} presets · existing settings for the other effect are preserved
          </span>
          <button className="studio-button" onClick={()=>{setQuery('');onClose();}}>
            Close
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
