import { useEffect, useRef, useState } from "react";
import {
  LogIn,
  ArrowRight,
  ShieldCheck,
  Check,
} from "lucide-react";
import "./first-start-dialog.css";
import AccountCapabilities from "./AccountCapabilities";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
const storageKey = "craftbox-account-onboarding-v1";
function onboardingKey(installationId: string) {
  return installationId ? `${storageKey}:${installationId}` : storageKey;
}
export function accountOnboardingComplete(installationId = "") {
  try {
    return localStorage.getItem(onboardingKey(installationId)) === "complete";
  } catch {
    return false;
  }
}
export function rememberAccountOnboarding(installationId = "") {
  try {
    localStorage.setItem(onboardingKey(installationId), "complete");
  } catch {
    /* The current launch can still continue if browser storage is unavailable. */
  }
}
export default function FirstStartDialog({
  open,
  onLink,
  onSkip,
  loggedIn = false,
  account = "",
}: {
  open: boolean;
  onLink: () => void;
  onSkip: () => void;
  loggedIn?: boolean;
  account?: string;
}) {
  const [skipping, setSkipping] = useState(false);
  const titleRef = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    if (!open) setSkipping(false);
  }, [open]);
  useEffect(() => {
    if (open) titleRef.current?.focus();
  }, [open, skipping]);
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !loggedIn) setSkipping(true);
      }}
    >
      <DialogContent
        className="first-start-dialog"
        initialFocus={titleRef}
        showCloseButton={false}
      >
        <DialogHeader>
          <DialogTitle ref={titleRef} tabIndex={-1}>
            {loggedIn
              ? "Welcome to CraftBox"
              : skipping
                ? "Continue without Xbox?"
                : "Connect your Xbox account"}
          </DialogTitle>
        </DialogHeader>
        <div className="first-start-body studio-dialog-body">
          <DialogDescription className="first-start-intro">
            {loggedIn
              ? "Your Xbox account is connected. You can capture from servers and worlds, or use the offline tools below."
              : skipping
                ? "You can work offline now and sign in later."
                : "Welcome to CraftBox. Connect your Xbox account to capture from servers, Realms and friends, record packets and browse the server Marketplace. You can also skip this and use the offline tools."}
          </DialogDescription>
          {loggedIn && (
            <div className="first-start-connected" role="status">
              <Check size={16} aria-hidden="true" />
              <span>
                Xbox account already connected{account ? ` · ${account}` : ""}
              </span>
            </div>
          )}
          <AccountCapabilities />
          <div className="first-start-note">
            <ShieldCheck size={17} aria-hidden="true" />
            <p>
              {loggedIn
                ? "You can manage the linked account using"
                : "Sign-in happens on Microsoft's website. CraftBox never asks for your Microsoft password. You can link your account later using"}
              <strong> Microsoft account</strong> in the top bar.
            </p>
          </div>
        </div>
        <DialogFooter>
          {loggedIn ? (
            <Button  onClick={onSkip}>
              Continue to workspace
              <ArrowRight size={14} />
            </Button>
          ) : skipping ? (
            <>
              <Button variant="secondary" onClick={onLink}>
                <LogIn size={14} />
                Link Xbox account
              </Button>
              <Button  onClick={onSkip}>
                Continue offline
                <ArrowRight size={14} />
              </Button>
            </>
          ) : (
            <>
              <Button
                variant="secondary"
                onClick={() => setSkipping(true)}
              >
                Skip for now
              </Button>
              <Button  onClick={onLink}>
                <LogIn size={14} />
                Link Xbox account
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
