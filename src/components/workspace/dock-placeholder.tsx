import * as React from "react";
import { Terminal } from "lucide-react";

/**
 * DockPlaceholder — Terminal placeholder for M4 (ADR-053).
 *
 * M1 ships the dock container + toggle (Ctrl+`) but the actual xterm.js +
 * node-pty terminal lands in M4. This placeholder tells the user what's
 * coming so the empty space isn't confusing.
 */

export function DockPlaceholder(): React.JSX.Element {
  return (
    <div className="flex h-full flex-col bg-black/80" data-testid="dock-placeholder">
      <div className="flex items-center gap-2 border-b border-border/40 px-3 py-1.5 text-[10px] text-muted-foreground">
        <Terminal className="h-3 w-3" />
        <span className="font-mono uppercase tracking-wider">Terminal · placeholder</span>
      </div>
      <div className="flex flex-1 items-center justify-center font-mono text-xs text-muted-foreground">
        <div className="text-center">
          <p className="mb-1">$ xterm.js + node-pty shell → M4</p>
          <p className="text-[10px] italic">Ctrl+` to toggle</p>
        </div>
      </div>
    </div>
  );
}
