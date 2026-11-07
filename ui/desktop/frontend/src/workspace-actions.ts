import { useDocumentScope } from "./document-scope";
import { useEffect, useRef } from "react";
export type WorkspaceAction =
  | "new"
  | "import-files"
  | "import-folder"
  | "browse-minecraft"
  | "export"
  | "refresh"
  | "search"
  | "settings"
  | "rules"
  | "list"
  | "grid"
  | "files"
  | "collections"
  | "favorites";
const event = "craftbox:workspace-action";
let pending: { page: string; action: WorkspaceAction } | undefined;
export function dispatchWorkspaceAction(page: string, action: WorkspaceAction) {
  pending = { page, action };
  window.dispatchEvent(new Event(event));
}
export function useWorkspaceActions(
  page: string,
  handler: (action: WorkspaceAction) => void,
) {
  const scope = useDocumentScope();
  const active = useRef(scope.active);
  active.current = scope.active;
  const current = useRef(handler);
  current.current = handler;
  useEffect(() => {
    const consume = () => {
      if (active.current && pending?.page === page) {
        const action = pending.action;
        pending = undefined;
        current.current(action);
      }
    };
    window.addEventListener(event, consume);
    consume();
    return () => window.removeEventListener(event, consume);
  }, [page, scope.active]);
}
