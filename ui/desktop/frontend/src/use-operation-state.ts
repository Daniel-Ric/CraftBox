import { useCallback, useRef, useState } from "react";

// Completion promises may resolve before React publishes the final render.
// Session guards must read the operation's current value synchronously.
export function useOperationState() {
  const [value, setValue] = useState(false);
  const current = useRef(false);
  const set = useCallback((next: boolean) => {
    current.current = next;
    setValue(next);
  }, []);
  return [value, set, current] as const;
}
