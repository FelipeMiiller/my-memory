import * as React from "react";
import { createRoot } from "react-dom/client";
import { WorkspaceShell } from "@/layouts/workspace-shell";
import "@/localization/i18n";
import "@/styles/global.css";

/**
 * App.tsx — my-memory viewer entry point (v3 workspace, ADR-053).
 *
 * M1 of the Roadmap v3 introduces the 3-pane workspace shell. Chat moved
 * from a top-level tab (Phase 1, v2.0.0) into the right pane.
 *
 * Phase 2 / Roadmap v3:
 *   - File Explorer stub → real `mem:fs:list` in M3
 *   - Editor pane placeholder → CodeMirror 6 in M3
 *   - Dock placeholder → xterm.js + node-pty in M4
 */
export default function App(): React.JSX.Element {
  return <WorkspaceShell />;
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
