import { useDocumentScope } from "./document-scope";
import { useEffect, useRef } from "react";
export type EditorCommand =
  | "inspect-view"
  | "open-archive"
  | "open-folder"
  | "reload"
  | "export-png"
  | "export-glb"
  | "fit"
  | "isometric"
  | "top"
  | "image-panel"
  | "reset-look"
  | "undo"
  | "redo"
  | "save-project"
  | "save-project-as"
  | "open-project"
  | "play"
  | "add-key"
  | "toggle-path"
  | "export-gif"
  | "export-mp4"
  | "export-webm";
export type EditorStatus = {
  ready: boolean;
  busy: boolean;
  undo: boolean;
  redo: boolean;
};
export const idleEditor: EditorStatus = {
  ready: false,
  busy: false,
  undo: false,
  redo: false,
};
const name = "craftbox:editor-command";
export function dispatchEditorCommand(command: EditorCommand) {
  window.dispatchEvent(new CustomEvent(name, { detail: command }));
}
export function useEditorCommands(handler: (command: EditorCommand) => void) {
  const scope = useDocumentScope();
  const active = useRef(scope.active);
  active.current = scope.active;
  const current = useRef(handler);
  current.current = handler;
  useEffect(() => {
    const listen = (e: Event) => {
      if (active.current)
        current.current((e as CustomEvent<EditorCommand>).detail);
    };
    window.addEventListener(name, listen);
    return () => window.removeEventListener(name, listen);
  }, []);
}
