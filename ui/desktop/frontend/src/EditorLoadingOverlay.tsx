import { LoaderCircle } from "lucide-react";
import "./editor-loading.css";

export default function EditorLoadingOverlay({
  label,
  loaded = 0,
  total = 0,
}: {
  label: string;
  loaded?: number;
  total?: number;
}) {
  return (
    <div className="editor-loading-overlay" role="status" aria-live="polite">
      <div className="editor-loading-content">
        <LoaderCircle className="editor-loading-spinner" aria-hidden="true" />
        <strong>{label}</strong>
        {total > 0 ? (
          <>
            <progress
              aria-label="Load world"
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
      </div>
    </div>
  );
}
