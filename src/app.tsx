import * as React from "react";
import { createRoot } from "react-dom/client";
import Layout from "@/layouts/layout";
import "@/localization/i18n";
import "@/styles/global.css";

/**
 * App.tsx — my-memory viewer entry point (Phase 1 MVP).
 *
 * Renders the full Layout (topbar + Resizable sidebar + Tabs).
 * Tab "Grafo" is the default active tab and mounts <GraphView /> which
 * loads ./data-central.json via fetch and renders Cytoscape.
 *
 * Phase 2 will replace fetch + state with `window.memAPI.loadDataset()` IPC.
 */
export default function App(): React.JSX.Element {
  return <Layout />;
}

const container = document.getElementById("app");
if (!container) {
  throw new Error('Root element with id "app" not found');
}
const root = createRoot(container);
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);