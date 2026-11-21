import { useEffect, useState } from "react";
import { CheckCircle2, Copy, RefreshCw, UserRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import { api, type AccountProfile } from "./bridge";
import "./account-identity.css";

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
  const [copied, setCopied] = useState("");
  const [imageFailed, setImageFailed] = useState(false);
  useEffect(() => {
    let active = true;
    setResult(undefined);
    setImageFailed(false);
    setCopied("");
    if (loggedIn)
      void api.GetAccountProfile().then(
        (profile) => {
          if (active) setResult({ key: accountKey, profile });
        },
        () => {
          if (active) setResult({ key: accountKey, error: true });
        },
      );
    return () => {
      active = false;
    };
  }, [loggedIn, accountKey, revision]);
  const current = loggedIn && result?.key === accountKey ? result : undefined;
  const profile = current?.profile;
  async function copyID(label: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(label);
    } catch {
      setCopied("Could not copy. Select the ID to copy it manually.");
    }
  }
  return (
    <section className="account-overview" aria-label="Xbox profile">
      <div className="account-person">
        <div className="account-avatar">
          {profile?.avatarURL && !imageFailed ? (
            <img
              src={profile.avatarURL}
              alt="Xbox gamer picture"
              referrerPolicy="no-referrer"
              onError={() => setImageFailed(true)}
            />
          ) : (
            <UserRound size={30} />
          )}
        </div>
        <div className="account-person-name" aria-live="polite">
          <h2 translate={profile?.gamertag ? "no" : undefined}>
            {loggedIn
              ? profile?.gamertag ||
                (current ? "Xbox account" : "Loading Xbox profile…")
              : "Connect your Xbox account"}
          </h2>
          <span className={loggedIn ? "account-connected" : "secondary"}>
            {loggedIn && <CheckCircle2 size={13} />}
            {loggedIn
              ? "Signed in with Microsoft"
              : "Capture from servers, Realms and friends"}
          </span>
          {profile?.presence && <small translate="no">{profile.presence}</small>}
        </div>
        {loggedIn && (
          <Button
            variant="ghost"
            size="icon"
            aria-label="Refresh Xbox profile"
            disabled={!current}
            onClick={() => setRevision((v) => v + 1)}
          >
            <RefreshCw size={15} className={!current ? "spin" : ""} />
          </Button>
        )}
      </div>
      {profile?.bio && <p className="account-bio" translate="no">{profile.bio}</p>}
      {profile?.detailsAvailable && (
        <dl className="account-statistics">
          {profile.gamerscore !== undefined && profile.gamerscore !== "" && (
            <div>
              <dt>Gamerscore</dt>
              <dd>{profile.gamerscore}</dd>
            </div>
          )}
          <div>
            <dt>Friends</dt>
            <dd>{profile.friends}</dd>
          </div>
          <div>
            <dt>Followers</dt>
            <dd>{profile.followers}</dd>
          </div>
          <div>
            <dt>Following</dt>
            <dd>{profile.following}</dd>
          </div>
        </dl>
      )}
      {(current?.error || (profile && !profile.detailsAvailable)) && (
        <p className="account-profile-notice" role="status">
          <span>
            {current?.error
              ? "Xbox profile could not be loaded."
              : "Additional Xbox profile details are temporarily unavailable."}
          </span>{" "}
          Your saved sign-in is unchanged. Use refresh to try again.
        </p>
      )}
      {profile && (
        <dl className="account-claims">
          {profile.classicGamertag &&
            profile.classicGamertag !== profile.gamertag && (
              <>
                <dt>Classic gamertag</dt>
                <dd translate="no">{profile.classicGamertag}</dd>
              </>
            )}
          {profile.accountTier && (
            <>
              <dt>Xbox account tier</dt>
              <dd translate="no">{profile.accountTier}</dd>
            </>
          )}
          <dt>Local sign-in</dt>
          <dd translate={accountKey ? "no" : undefined}>{accountKey || "Default account"}</dd>
          {profile.xuid && (
            <>
              <dt>Xbox ID</dt>
              <dd>
                <span>{profile.xuid}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Copy Xbox ID"
                  onClick={() => void copyID("Xbox ID", profile.xuid)}
                >
                  <Copy size={13} />
                </Button>
              </dd>
            </>
          )}
          {profile.playFabID && (
            <>
              <dt>Minecraft PlayFab ID</dt>
              <dd>
                <span>{profile.playFabID}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Copy PlayFab ID"
                  onClick={() => void copyID("PlayFab ID", profile.playFabID!)}
                >
                  <Copy size={13} />
                </Button>
              </dd>
            </>
          )}
        </dl>
      )}
      {copied && (
        <small role="status">
          {copied.includes("Could not") ? copied : `${copied} copied`}
        </small>
      )}
    </section>
  );
}
