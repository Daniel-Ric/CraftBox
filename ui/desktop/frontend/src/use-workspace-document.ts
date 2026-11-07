import { useEffect, useRef, useState } from "react";
import { api } from "./bridge";
import {
  documentLabel,
  useDocumentScope,
  type WorkspaceDocument,
} from "./document-scope";
// Reads the latest state during save; cancellation and concurrent edits never establish a clean baseline.
export function useWorkspaceDocument(
  kind: WorkspaceDocument["kind"],
  source: { id: string; name: string },
  view: Record<string, unknown>,
  busy: boolean,
  ready = true,
) {
  const scope = useDocumentScope();
  const [saveRevision, setSaveRevision] = useState(0);
  const [savingState, setSavingState] = useState(false);
  const signature = JSON.stringify({ source: source.id, view });
  const latest = useRef({ signature, source, view });
  latest.current = { signature, source, view };
  const baseline = useRef<string | undefined>(undefined),
    saved = useRef<string | undefined>(undefined),
    saving = useRef(false);
  const savedPath = useRef(scope.path);
  const signatureRef = useRef(signature);
  signatureRef.current = signature;
  // A loaded document's asynchronous resource preparation is not a user edit.
  useEffect(() => {
    if (!ready || baseline.current !== undefined) return;
    let second = 0;
    const first = requestAnimationFrame(() => {
      second = requestAnimationFrame(() => {
        baseline.current = signature;
        setSaveRevision((n) => n + 1);
      });
    });
    return () => {
      cancelAnimationFrame(first);
      cancelAnimationFrame(second);
    };
  }, [ready, signature]);
  const dirty =
    !!source.id &&
    baseline.current !== undefined &&
    signature !== (saved.current ?? baseline.current);
  useEffect(() => {
    scope.register?.({
      dispose: () => void api.ReleaseDocumentTab(scope.id).catch(() => {}),
      name: documentLabel(source.name, savedPath.current),
      path: savedPath.current,
      get dirty() {
        return (
          !!latest.current.source.id &&
          baseline.current !== undefined &&
          signatureRef.current !== (saved.current ?? baseline.current)
        );
      },
      ready: ready && !!source.id,
      get busy() {
        return busy || saving.current;
      },
      save: async (saveAs = false) => {
        if (saving.current || busy)
          throw Error("Wait for the current operation to finish.");
        saving.current = true;
        setSavingState(true);
        const snapshot = latest.current;
        try {
          const document: WorkspaceDocument = {
            format: "craftbox-workspace",
            version: 1,
            kind,
            source: snapshot.source,
            view: snapshot.view,
          };
          const path = await api.SaveWorkspaceProject(
            scope.id,
            snapshot.source.name,
            JSON.stringify(document),
            saveAs,
          );
          if (!path) return false;
          saved.current = snapshot.signature;
          savedPath.current = path;
          scope.onSaved?.(JSON.parse(JSON.stringify(document)), path);
          setSaveRevision((n) => n + 1);
          return signatureRef.current === snapshot.signature;
        } finally {
          saving.current = false;
          setSavingState(false);
        }
      },
    });
  }, [
    scope.register,
    scope.id,
    source.name,
    signature,
    dirty,
    busy,
    kind,
    ready,
    saveRevision,
    savingState,
  ]);
  useEffect(() => () => scope.register?.(undefined), [scope.register]);
}
