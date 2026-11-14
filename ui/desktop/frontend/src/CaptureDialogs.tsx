import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export default function CaptureDialogs({
  realm,
  onStories,
  warning,
  onWarning,
  remember,
  onRemember,
  onDisable,
}: {
  realm?: string;
  onStories: (answer: number) => void;
  warning: boolean;
  onWarning: (open: boolean) => void;
  remember: boolean;
  onRemember: (value: boolean) => void;
  onDisable: () => void;
}) {
  return (
    <>
      <Dialog
        open={Boolean(realm)}
        onOpenChange={(open) => {
          if (!open) onStories(0);
        }}
      >
        <DialogContent resizable={false}>
          <DialogHeader>
            <DialogTitle>Realm Stories</DialogTitle>
            <DialogDescription>{realm}</DialogDescription>
          </DialogHeader>
          <p>Choose whether to participate in Realm Stories before joining.</p>
          <DialogFooter>
            <Button variant="outline" onClick={() => onStories(0)}>
              Cancel
            </Button>
            <Button variant="outline" onClick={() => onStories(2)}>
              Opt out
            </Button>
            <Button onClick={() => onStories(1)}>Opt in</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={warning} onOpenChange={onWarning}>
        <DialogContent resizable={false} tone="warning">
          <DialogHeader>
            <DialogTitle>Disable packet recording?</DialogTitle>
            <DialogDescription>
              Packet recordings allow replaying a capture later.
            </DialogDescription>
          </DialogHeader>
          <div className="option-row">
            <Label htmlFor="remember-recording-warning">
              Remember my choice
            </Label>
            <Switch
              id="remember-recording-warning"
              checked={remember}
              onCheckedChange={onRemember}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => onWarning(false)}>
              Cancel
            </Button>
            <Button onClick={onDisable}>Disable recording</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
