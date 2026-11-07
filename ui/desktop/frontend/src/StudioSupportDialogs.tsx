import { useState } from "react";
import { Command } from "lucide-react";
import { CaptureIcon, RenderIcon, CinematicIcon } from "./studio-icons";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import StudioBrand from "./StudioBrand";
const topics = [
  { id: "capture", name: "Capture", icon: CaptureIcon },
  { id: "render", name: "Still renders", icon: RenderIcon },
  { id: "cinematic", name: "Camera sequences", icon: CinematicIcon },
  { id: "shortcuts", name: "Keyboard shortcuts", icon: Command },
];
export default function StudioSupportDialogs(p: {
  helpOpen: boolean;
  onHelpOpen: (v: boolean) => void;
  aboutOpen: boolean;
  onAboutOpen: (v: boolean) => void;
  version?: string;
  minecraft?: string;
}) {
  const [topic, setTopic] = useState("render");
  return (
    <>
      <Dialog open={p.helpOpen} onOpenChange={p.onHelpOpen}>
        <DialogContent className="studio-help-dialog">
          <DialogHeader>
            <DialogTitle>Controls &amp; shortcuts</DialogTitle>
            <DialogDescription>
              Workspaces and tools in CraftBox.
            </DialogDescription>
          </DialogHeader>
          <div className="support-body">
            <nav
              role="tablist"
              aria-label="Help topics"
              aria-orientation="vertical"
            >
              {topics.map((t) => (
                <button
                  key={t.id}
                  role="tab"
                  aria-selected={topic === t.id}
                  onClick={() => setTopic(t.id)}
                >
                  <t.icon size={15} />
                  {t.name}
                </button>
              ))}
            </nav>
            <section
              className="studio-dialog-body"
              role="tabpanel"
              aria-label={topics.find((t) => t.id === topic)!.name}
            >
              {topic === "capture" ? (
                <>
                  <h2>Capture worlds and resources</h2>
                  <ol className="workflow-steps">
                    <li>
                      <strong>Select source</strong>
                      <p>Choose a server or world. Sign in to start a capture.</p>
                    </li>
                    <li>
                      <strong>Configure capture</strong>
                      <p>Choose capture settings, start the capture, then connect Minecraft.</p>
                    </li>
                    <li>
                      <strong>Use your results</strong>
                      <p>Open a captured world in Render or Cinematic.</p>
                    </li>
                  </ol>
                </>
              ) : topic === "render" ? (
                <>
                  <h2>Compose a still render</h2>
                  <ol className="workflow-steps">
                    <li>
                      <strong>Open a world</strong>
                      <p>Open a world, then use View to position the camera.</p>
                    </li>
                    <li>
                      <strong>Adjust lighting and color</strong>
                      <p>Choose a look, then adjust lighting and color.</p>
                    </li>
                    <li>
                      <strong>Export the image</strong>
                      <p>Export the image. Enable transparency to remove the background.</p>
                    </li>
                  </ol>
                </>
              ) : topic === "cinematic" ? (
                <>
                  <h2>Compose a camera sequence</h2>
                  <ol className="workflow-steps">
                    <li>
                      <strong>Set camera positions</strong>
                      <p>Frame the subject and add camera keyframes.</p>
                    </li>
                    <li>
                      <strong>Check camera motion</strong>
                      <p>Preview the sequence and adjust keyframes in Timeline or Path.</p>
                    </li>
                    <li>
                      <strong>Save and export the sequence</strong>
                      <p>Save your project, then choose an export format.</p>
                    </li>
                  </ol>
                </>
              ) : (
                <>
                  <h2>Keyboard shortcuts</h2>
                  <p>
                    Use <kbd>Ctrl + K</kbd> to find menu commands and
                    workspaces.
                  </p>
                  <dl className="support-shortcuts">
                    {[
                      ["New project", "Ctrl + N"],
                      ["Switch workspaces", "Ctrl + Shift + 1–5"],
                      ["Preferences", "Ctrl + ,"],
                      ["Open world file", "Ctrl + O"],
                      ["Open world folder", "Ctrl + Shift + O"],
                      ["Export image", "Ctrl + Shift + E"],
                      ["Save cinematic project", "Ctrl + S"],
                      ["Undo / Redo", "Ctrl + Z / Ctrl + Shift + Z"],
                      ["Playback", "Space"],
                      ["Previous / Next frame", "← / →"],
                    ].map(([label, key]) => (
                      <div key={label}>
                        <dt>{label}</dt>
                        <dd>
                          <kbd>{key}</kbd>
                        </dd>
                      </div>
                    ))}
                  </dl>
                </>
              )}
            </section>
          </div>
          <DialogFooter>
            <span className="dialog-footer-note">Find commands: Ctrl + K</span>
            <button
              className="studio-button primary"
              onClick={() => p.onHelpOpen(false)}
            >
              Close
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={p.aboutOpen} onOpenChange={p.onAboutOpen}>
        <DialogContent resizable={false} className="studio-about-dialog">
          <DialogHeader>
            <DialogTitle>About CraftBox</DialogTitle>
            <DialogDescription>Application information.</DialogDescription>
          </DialogHeader>
          <div className="studio-dialog-body">
            <div className="about-identity">
              <StudioBrand size={76} />
              <div>
                <h2>CraftBox</h2>
                <p>Minecraft Bedrock studio</p>
              </div>
            </div>
            <dl className="properties-list">
              <div>
                <dt>Version</dt>
                <dd>{p.version || "—"}</dd>
              </div>
              <div>
                <dt>Minecraft</dt>
                <dd>{p.minecraft || "—"}</dd>
              </div>
              <div>
                <dt>Workspaces</dt>
                <dd>Capture · Render · Cinematic</dd>
              </div>
            </dl>
          </div>
          <DialogFooter>
            <button
              className="studio-button primary"
              onClick={() => p.onAboutOpen(false)}
            >
              Close
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
