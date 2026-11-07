import { LoaderCircle } from "lucide-react";
import { Button } from "./components/ui/button";
import "./editor-loading.css";

export default function EditorLoadingOverlay({
  label,
  loaded = 0,
  total = 0,
  onCancel,
}: {
  label: string;
  loaded?: number;
  total?: number;
  onCancel?: () => void;
}) {
  return (
    <div className="editor-loading-overlay" role="status" aria-live="polite">
      <div className="editor-loading-content">
        <LoaderCircle className="editor-loading-spinner" aria-hidden="true" />
        <strong>{label}</strong>
        {total > 0 ? (
          <>
            <progress
              aria-label={label}
              max={total}
              value={Math.min(loaded, total)}
            />
            <span>
              {loaded.toLocaleString("en-US")} / {total.toLocaleString("en-US")}{" "}
              Chunks
            </span>
          </>
        ) : (
          <span>Please wait …</span>
        )}
        {onCancel && <Button variant="outline" size="sm" onClick={onCancel}>Stop preview</Button>}
      </div>
    </div>
  );
}
