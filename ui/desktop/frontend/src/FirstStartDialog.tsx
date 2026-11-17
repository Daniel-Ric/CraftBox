import { useRef } from "react";
import {
  FolderOpen,
  LogIn,
  ArrowRight,
  ShieldCheck,
  Check,
} from "lucide-react";
import "./first-start-dialog.css";
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
}: {
  open: boolean;
  onLink: () => void;
  onSkip: () => void;
  loggedIn?: boolean;
  account?: string;
}) {
  const titleRef = useRef<HTMLHeadingElement>(null);
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onSkip();
      }}
    >
      <DialogContent className="first-start-dialog" initialFocus={titleRef}>
        <DialogHeader>
          <DialogTitle ref={titleRef} tabIndex={-1}>
            Welcome to CraftBox
          </DialogTitle>
          <DialogDescription>
            Choose how you want to get started
          </DialogDescription>
        </DialogHeader>
        <div className="first-start-body studio-dialog-body">
          <div className="first-start-brand">
            <img src="/craft-box-icon.png" alt="" width={64} height={64} />
            <div>
              <h2>Your worlds. Your next creation.</h2>
              <p>Capture from Minecraft, or start with files on your device.</p>
            </div>
          </div>
          <div className="first-start-options">
            <section>
              <LogIn size={21} />
              <h3>Connect Xbox</h3>
              <p>
                Discover friends' worlds, join Realms and capture from online
                sessions.
              </p>
            </section>
            <section>
              <FolderOpen size={21} />
              <h3>Work with local files</h3>
              <p>
                Open saved worlds, edit packs and skins, or create renders and
                camera sequences.
              </p>
            </section>
          </div>
          <p className="first-start-note">
            <ShieldCheck size={16} />
            <span>
              Sign-in happens on Microsoft's website. You can manage your
              account later from the top bar.
            </span>
          </p>
          {loggedIn && (
            <p className="first-start-connected">
              <Check size={15} />
              Your Microsoft account is already connected.
            </p>
          )}
        </div>
        <DialogFooter>
          {loggedIn ? (
            <Button onClick={onSkip}>
              Open CraftBox
              <ArrowRight size={14} />
            </Button>
          ) : (
            <>
              <Button variant="ghost" onClick={onSkip}>
                Continue offline
              </Button>
              <Button onClick={onLink}>
                <LogIn size={14} />
                Sign in with Microsoft
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
