import { type ReactNode } from "react";
import { ShieldCheck } from "lucide-react";
import DiscordPresenceStatus from "./DiscordPresenceStatus";
import { type PreferenceKey } from "./workspace-preferences";

export default function DiscordPresenceSettings({
  field,
}: {
  field: (key: PreferenceKey) => ReactNode;
}) {
  return (
    <div className="discord-settings">
      <div className="discord-intro">
        <img src="/craft-box-icon.png" alt="" width={42} height={42} />
        <div>
          <h3>Your workspace, on Discord</h3>
          <p>Ready to use. Open Discord desktop and turn on sharing below.</p>
        </div>
      </div>
      {field("discordEnabled")}
      <DiscordPresenceStatus />
      <div className="discord-privacy">
        <ShieldCheck size={16} />
        <p>
          Only your chosen text and activity are shared. CraftBox keeps server
          addresses, world names and account details private.
        </p>
      </div>
      <details className="discord-settings-section" open>
        <summary>
          Text &amp; activity <small>Choose what friends see</small>
        </summary>
        {field("discordDetails")}
        {field("discordAutomatic")}
        {field("discordState")}
        {field("discordShowElapsed")}
        <p className="discord-help">
          Automatic activity follows your capture mode. Turn it off to use your
          own second line. Changes save automatically.
        </p>
      </details>
      <details className="discord-settings-section">
        <summary>
          Logo &amp; status badge <small>Make it your own</small>
        </summary>
        <p className="discord-help">
          The CraftBox logo is always included. Customize its hover text or add
          a small badge with a public image link.
        </p>
        {field("discordLargeText")}
        {field("discordSmallImage")}
        {field("discordSmallText")}
        <p className="discord-help">
          For the badge, use a direct https:// link to a square image. A link to
          a web page containing an image will not work. Leave the URL empty to
          hide the badge.
        </p>
      </details>
      <details className="discord-settings-section">
        <summary>
          Link buttons <small>CraftBox GitHub and one custom link</small>
        </summary>
        <p className="discord-help">
          The CraftBox GitHub button is on by default. You can hide it below.
          Add one custom button with your own label and link, such as Community
          and https://discord.gg/your-invite. Leave both custom fields empty to
          hide it.
        </p>
        {field("discordGitHubButton")}
        {field("discordButton1Label")}
        {field("discordButton1Url")}
        <p className="discord-help">
          Discord may show these buttons only to other people viewing your
          profile.
        </p>
      </details>
      <details className="discord-settings-section">
        <summary>Not showing up?</summary>
        <ol className="discord-help discord-troubleshooting">
          <li>
            Open the Discord desktop app and sign in. The browser version alone
            cannot show CraftBox activity.
          </li>
          <li>
            In Discord settings, open <b>Activity Privacy</b> and enable
            activity sharing.
          </li>
          <li>
            Turn on <b>Discord Rich Presence</b> above. CraftBox connects
            automatically; no application setup is needed.
          </li>
          <li>
            Allow a few seconds for Discord to update, or select{" "}
            <b>Try again</b>. CraftBox also retries automatically every 15
            seconds.
          </li>
        </ol>
      </details>
    </div>
  );
}
