import { useEffect, useState } from "react";
import { CheckCircle2, Copy, RefreshCw, UserRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AccountIdentityState } from "./use-account-profile";
import "./account-identity.css";

export default function AccountIdentity({
  loggedIn,
  accountKey,
  identity,
}: {
  loggedIn: boolean;
  accountKey: string;
  identity: AccountIdentityState;
}) {
  const { profile, loading, error, refresh } = identity;
  const [copied, setCopied] = useState("");
  const [imageFailed, setImageFailed] = useState(false);
  useEffect(() => {
    setImageFailed(false);
    setCopied("");
  }, [loggedIn, accountKey, profile?.avatarURL, loading]);
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
                (loading ? "Loading Xbox profile…" : "Xbox account")
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
            disabled={loading}
            onClick={refresh}
          >
            <RefreshCw size={15} className={loading ? "spin" : ""} />
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
      {(error || (profile && !profile.detailsAvailable)) && (
        <p className="account-profile-notice" role="status">
          <span>
            {error
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
