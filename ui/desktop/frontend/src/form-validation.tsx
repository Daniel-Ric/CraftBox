import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
const Context = createContext<{
  invalid: boolean;
  field: (id: string, invalid: boolean) => void;
} | null>(null);
export function DialogValidation({ children }: { children: ReactNode }) {
  const [fields, setFields] = useState(new Set<string>());
  const parentField = useContext(Context)?.field;
  const field = useCallback(
    (id: string, invalid: boolean) => {
      parentField?.(id, invalid);
      setFields((old) => {
        if (old.has(id) === invalid) return old;
        const next = new Set(old);
        if (invalid) next.add(id);
        else next.delete(id);
        return next;
      });
    },
    [parentField],
  );
  const value = useMemo(
    () => ({ invalid: fields.size > 0, field }),
    [fields, field],
  );
  return <Context.Provider value={value}>{children}</Context.Provider>;
}
export function useDialogValidation() {
  return useContext(Context);
}
