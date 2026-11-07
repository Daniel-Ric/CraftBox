import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useState } from "react";
import { CheckCircle2, RefreshCw } from "lucide-react";
import { AccountIcon } from "./studio-icons";
import { Button } from "@/components/ui/button";
import { api, type AccountProfile } from "./bridge";
import "./account-identity.css";
import AccountCapabilities from "./AccountCapabilities";

export default function AccountIdentity({
  loggedIn,
  accountKey,
}: {
  loggedIn: boolean;
  accountKey: string;
}) {
  const [result, setResult] = useState<{
    key: string;
    profile?: AccountProfile;
    error?: boolean;
  }>();
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    let active = true;
    setResult(undefined);
    if (loggedIn) {
      void Promise.resolve()
        .then(() => api.GetAccountProfile())
        .then(
          (profile) => {
            if (active) setResult({ key: accountKey, profile });
          },
          () => {
            if (active) setResult({ key: accountKey, error: true });
          },
        );
    }
    return () => {
      active = false;
    };
  }, [loggedIn, accountKey, revision]);
  const current = loggedIn && result?.key === accountKey ? result : undefined;
  const profile = current?.profile;
  return (
    <section className="account-overview" aria-label="Microsoft account">
      <div className="account-person">
        <AccountIcon size={30} />
        <div aria-live="polite">
          <h3>
            {loggedIn
              ? profile?.gamertag ||
                (current ? "Microsoft account" : "Loading Xbox profile…")
              : "Microsoft account"}
          </h3>
          <span className={loggedIn ? "account-connected" : "secondary"}>
            {loggedIn && <CheckCircle2 size={13} aria-hidden="true" />}
            {loggedIn ? "Signed in" : "Signed out"}
          </span>
        </div>
      </div>
      {profile && (
        <dl className="account-claims">
          {profile.classicGamertag &&
            profile.classicGamertag !== profile.gamertag && (
              <>
                <dt>Classic gamertag</dt>
                <dd>{profile.classicGamertag}</dd>
              </>
            )}
          {profile.xuid && (
            <>
              <dt>Xbox ID</dt>
              <dd>{profile.xuid}</dd>
            </>
          )}
          {!profile.gamertag && (
            <>
              <dt>Gamertag</dt>
              <dd>Not provided</dd>
            </>
          )}
        </dl>
      )}
      {current?.error && (
        <FeedbackFlyout title="Microsoft account">
          <span>Xbox profile unavailable</span>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setRevision((v) => v + 1)}
          >
            <RefreshCw size={13} />
            Retry
          </Button>
        </FeedbackFlyout>
      )}
      <AccountCapabilities />
    </section>
  );
}
