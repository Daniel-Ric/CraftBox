import { useEffect, useState } from "react";
import {
  ArrowUpRight,
  Check,
  Copy,
  LoaderCircle,
  LogIn,
  LogOut,
  ShieldCheck,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import AccountIdentity from "./AccountIdentity";
import type { AccountIdentityState } from "./use-account-profile";
import { api } from "./bridge";
import { profileNameError } from "./input-rules";

export default function AccountDialog({
  open,
  onOpenChange,
  loggedIn,
  account,
  identity,
  running,
  authenticating,
  authCode,
  authURI,
  authNotice,
  action,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  loggedIn: boolean;
  account: string;
  identity: AccountIdentityState;
  running: boolean;
  authenticating: boolean;
  authCode: string;
  authURI: string;
  authNotice?: string;
  action: (operation: () => Promise<unknown>) => Promise<unknown>;
}) {
  const [profiles, setProfiles] = useState<string[]>([]);
  const [profileError, setProfileError] = useState(false);
  const [name, setName] = useState("");
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState(false);
  const [busy, setBusy] = useState(false);
  const locked = running || authenticating || busy;
  useEffect(() => {
    let active = true;
    setName("");
    setCopied(false);
    setCopyError(false);
    setProfileError(false);
    if (open && !authCode)
      void api.GetAccountSignInProfiles().then(
        (values) => {
          if (active) setProfiles(values);
        },
        () => {
          if (active) {
            setProfiles([]);
            setProfileError(true);
          }
        },
      );
    return () => {
      active = false;
    };
  }, [open, account, loggedIn, authCode]);
  function close() {
    onOpenChange(false);
    if (authenticating) void action(() => api.CancelLogin());
  }
  async function switchProfile(value: string) {
    setBusy(true);
    try {
      await action(() => api.SwitchAccount(value));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) close();
      }}
    >
      <DialogContent className="account-dialog account-dialog-redesigned">
        <DialogHeader>
          <DialogTitle>
            {authCode ? "Sign in with Microsoft" : "Your account"}
          </DialogTitle>
          <DialogDescription>
            {authCode
              ? "Complete sign-in in your browser"
              : "Xbox profile and saved sign-ins"}
          </DialogDescription>
        </DialogHeader>
        {authCode ? (
          <div className="account-dialog-body account-device-flow">
            <div className="account-signin-brand">
              <img src="/craft-box-icon.png" alt="" width={42} height={42} />
              <div>
                <strong>Connect Xbox to CraftBox</strong>
                <p>Your password stays on Microsoft's website.</p>
              </div>
            </div>
            <ol className="account-signin-steps">
              <li>Open the Microsoft sign-in page.</li>
              <li>Enter this code and choose your Microsoft account.</li>
            </ol>
            <div className="account-device-code">
              <code aria-label="Microsoft sign-in code">{authCode}</code>
              <Button
                variant="ghost"
                size="icon"
                aria-label="Copy sign-in code"
                onClick={() => {
                  void navigator.clipboard.writeText(authCode).then(
                    () => {
                      setCopied(true);
                      setCopyError(false);
                    },
                    () => setCopyError(true),
                  );
                }}
              >
                {copied ? <Check /> : <Copy />}
              </Button>
            </div>
            <span className="account-copy-status" role="status">
              {copyError
                ? "Select the code to copy it manually."
                : copied
                  ? "Code copied"
                  : "Only use this code on Microsoft's sign-in page."}
            </span>
            <Button
              className="account-browser-button"
              onClick={() => void action(() => api.OpenURL(authURI))}
            >
              Open Microsoft sign-in
              <ArrowUpRight size={15} />
            </Button>
            <p className="account-signin-wait" role="status">
              <LoaderCircle className="spin" size={15} />
              Waiting for confirmation in your browser…
            </p>
            <p className="account-security-note">
              {authNotice ||
                "Some servers restrict proxy connections. Consider a separate account for captures."}
            </p>
          </div>
        ) : (
          <div className="account-dialog-body">
            <AccountIdentity loggedIn={loggedIn} accountKey={account} identity={identity} />
            {!loggedIn && (
              <div className="account-signin-prompt">
                <p>
                  Sign in to discover friends' worlds, access Realms and use
                  authenticated captures. Local worlds, renders and editing work
                  without an account.
                </p>
                <Button
                  disabled={locked}
                  onClick={() => void action(() => api.Login())}
                >
                  {authenticating ? (
                    <LoaderCircle className="spin" />
                  ) : (
                    <LogIn />
                  )}
                  {authenticating
                    ? "Preparing sign-in…"
                    : "Sign in with Microsoft"}
                </Button>
                <small>
                  <ShieldCheck size={14} />
                  CraftBox never asks for your Microsoft password.
                </small>
              </div>
            )}
            <details className="account-saved-signins">
              <summary>
                Manage saved sign-ins
                <span translate={account ? "no" : undefined}>{account || "Default account"}</span>
              </summary>
              <div>
                <p>
                  Keep separate sign-ins for different Microsoft accounts on
                  this device. These labels do not change your Xbox gamertag.
                </p>
                <label htmlFor="saved-account-profile">Saved sign-in</label>
                <select
                  id="saved-account-profile"
                  value={account}
                  disabled={locked}
                  onChange={(event) => void switchProfile(event.target.value)}
                >
                  {Array.from(new Set([account, ...profiles])).map((value) => (
                    <option key={value} value={value} translate={value ? "no" : undefined}>
                      {value || "Default account"}
                    </option>
                  ))}
                </select>
                {profileError && (
                  <small role="status">
                    Saved sign-ins could not be loaded. Reopen this dialog to
                    retry.
                  </small>
                )}
                <label htmlFor="new-account-profile">Add another sign-in</label>
                <div className="account-add-signin">
                  <Input
                    id="new-account-profile"
                    value={name}
                    maxLength={80}
                    validate={profileNameError}
                    placeholder="A label, e.g. Capture account"
                    disabled={locked}
                    onChange={(event) => setName(event.target.value)}
                  />
                  <Button
                    variant="outline"
                    disabled={
                      locked ||
                      !name.trim() ||
                      !!profileNameError(name) ||
                      profiles.includes(name.trim()) ||
                      name.trim() === account
                    }
                    onClick={() => void switchProfile(name.trim())}
                  >
                    Add sign-in
                  </Button>
                </div>
                <small>
                  Adding selects an empty sign-in. Then sign in with Microsoft
                  to connect it.
                </small>
              </div>
            </details>
            {running && (
              <p className="account-profile-notice">
                End the active capture before changing accounts or signing out.
              </p>
            )}
          </div>
        )}
        <DialogFooter>
          {authenticating ? (
            <Button
              variant="ghost"
              onClick={() => void action(() => api.CancelLogin())}
            >
              Cancel sign-in
            </Button>
          ) : loggedIn ? (
            <Button
              variant="ghost"
              disabled={locked}
              title="Remove this saved sign-in. Your worlds, assets and recordings are kept."
              onClick={() => void action(() => api.Logout())}
            >
              <LogOut size={14} />
              Sign out
            </Button>
          ) : null}
          <Button variant="outline" onClick={close}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
