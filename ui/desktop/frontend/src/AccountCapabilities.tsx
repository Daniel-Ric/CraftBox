import { CheckCircle2, Gamepad2, Globe } from "lucide-react";
import "./account-identity.css";

export default function AccountCapabilities() {
  return <div className="account-capabilities">
    <section><h3><Gamepad2 size={15} /> Xbox connection</h3><small>Microsoft sign-in required</small><ul>
      <li>Xbox gamertag and account identity</li>
      <li>Discover and connect to friends' shared Minecraft sessions</li>
      <li>Authenticated NetherNet connections for live captures</li>
    </ul></section>
    <section><h3><Globe size={15} /> Minecraft online</h3><small>Linked account and internet required</small><ul>
      <li>Capture worlds, resource packs and player skins</li>
      <li>Browse Featured Servers, Experiences and accessible Realms</li>
      <li>Record live traffic in Packet Inspector</li>
      <li>Browse server Marketplace offers and collections in Production or Staging</li>
    </ul></section>
    <section><h3><CheckCircle2 size={15} /> Without an account</h3><small>Local tools remain available offline</small><ul>
      <li>Import worlds; create still renders and camera sequences</li>
      <li>Inspect, edit and export local packs, models and skins</li>
      <li>Pose skins, adjust cosmetics and preview animations</li>
      <li>Play and arrange local audio with waveform tracks</li>
      <li>Open saved recordings and analyze packets, entities and chunks</li>
    </ul></section>
  </div>;
}
