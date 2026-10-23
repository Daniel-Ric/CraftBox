import { useState } from "react";
import { Aperture, Clapperboard, Globe2, Command } from "lucide-react";
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
  { id: "capture", name: "Capture", icon: Globe2 },
  { id: "render", name: "Still renders", icon: Aperture },
  { id: "cinematic", name: "Camera sequences", icon: Clapperboard },
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
                      <p>
                        Enter a server address or select a server, Realm or friend's world. Sign in with Microsoft to access private sources.
                      </p>
                    </li>
                    <li>
                      <strong>Configure capture</strong>
                      <p>
                        Choose what to save on the right, then start the session and connect Minecraft.
                      </p>
                    </li>
                    <li>
                      <strong>Use your results</strong>
                      <p>
                        Select a world in the library to create a still render or camera sequence. Packs and skins have their own libraries.
                      </p>
                    </li>
                  </ol>
                </>
              ) : topic === "render" ? (
                <>
                  <h2>Compose a still render</h2>
                  <ol className="workflow-steps">
                    <li>
                      <strong>Open a world</strong>
                      <p>
                        Open a .mcworld file, a local world folder or a captured world. Choose a perspective from the View menu.
                      </p>
                    </li>
                    <li>
                      <strong>Adjust lighting and color</strong>
                      <p>
                        Start with a look in the inspector, then adjust lighting, environment, color and detail. Controls update the view immediately.
                      </p>
                    </li>
                    <li>
                      <strong>Export the image</strong>
                      <p>
                        Open PNG export and check the preview, file name and pixel dimensions. Enable a transparent background for a cutout.
                      </p>
                    </li>
                  </ol>
                </>
              ) : topic === "cinematic" ? (
                <>
                  <h2>Compose a camera sequence</h2>
                  <ol className="workflow-steps">
                    <li>
                      <strong>Set camera positions</strong>
                      <p>
                        Frame your subject and save the view to a keyframe. Set its time, look-at target and focal length in the inspector.
                      </p>
                    </li>
                    <li>
                      <strong>Check camera motion</strong>
                      <p>
                        Play the sequence and edit points in the timeline and path view. Smooth transitions pass through keyframes without stopping.
                      </p>
                    </li>
                    <li>
                      <strong>Save and export the sequence</strong>
                      <p>
                        Save the cinematic project for later editing. MP4 and WebM render frame by frame; GIF is useful for short previews.
                      </p>
                    </li>
                  </ol>
                </>
              ) : (
                <>
                  <h2>Keyboard shortcuts</h2>
                  <p>
                    Use <kbd>Ctrl + K</kbd> to find menu commands and workspaces.
                  </p>
                  <dl className="support-shortcuts">
                    {[
                      ["New project", "Ctrl + N"],
                      ["Switch workspaces", "Ctrl + Shift + 1–5"],
                      ["Preferences", "Ctrl + ,"],
                      ["Open world file", "Ctrl + O"],
                      ["Open world folder", "Ctrl + Shift + O"],
                      ["Export PNG", "Ctrl + Shift + E"],
                      ["Save cinematic project", "Ctrl + S"],
                      [
                        "Undo / Redo",
                        "Ctrl + Z / Ctrl + Shift + Z",
                      ],
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
        <DialogContent className="studio-about-dialog">
          <DialogHeader>
            <DialogTitle>About CraftBox</DialogTitle>
            <DialogDescription>Application information.</DialogDescription>
          </DialogHeader>
          <div className="studio-dialog-body">
            <div className="about-identity">
              <StudioBrand size={76} />
              <div>
                <h2>CraftBox</h2>
                <p>Render and cinematic studio for Minecraft Bedrock</p>
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
