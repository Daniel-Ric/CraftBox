import { useEffect, useState } from "react";
import { api, type AccountProfile } from "./bridge";

export function useAccountProfile(enabled: boolean, accountKey: string) {
  const [result, setResult] = useState<{
    key: string;
    revision: number;
    profile?: AccountProfile;
    error?: boolean;
  }>();
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    if (!enabled) {
      setResult(undefined);
      return;
    }
    let active = true;
    void api.GetAccountProfile().then(
      (profile) => {
        if (active) setResult({ key: accountKey, revision, profile });
      },
      () => {
        if (active) setResult({ key: accountKey, revision, error: true });
      },
    );
    return () => {
      active = false;
    };
  }, [enabled, accountKey, revision]);

  const current = enabled && result?.key === accountKey ? result : undefined;
  return {
    profile: current?.profile,
    error: current?.error,
    loading: enabled && current?.revision !== revision,
    refresh: () => setRevision((value) => value + 1),
  };
}

export type AccountIdentityState = ReturnType<typeof useAccountProfile>;
