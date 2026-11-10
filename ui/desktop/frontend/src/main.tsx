import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./index.css";
import "./workspace.css";
import "./studio-foundations.css";
import "./studio-shell.css";
import "./studio-documents.css";
import "./studio-libraries.css";
import "./studio-editors.css";
import "./adobe-ui.css";
import "./workbench.css";
import "./workbench-dialogs.css";
import "./workspace-content.css";
import "./collection-layouts.css";
import StudioTooltipLayer from "./StudioTooltipLayer";
import StudioFlyouts from "./StudioFlyouts";
import StudioRangeLayer from "./StudioRangeLayer";
import "./studio-focus.css";
import "./studio-sidebars.css";
import "./view-headings.css";
import "./workspace-responsive.css";
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
    <StudioTooltipLayer />
    <StudioFlyouts />
    <StudioRangeLayer />
  </React.StrictMode>,
);
