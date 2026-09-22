import * as React from "react";

/**
 * useShortcut — minimal keyboard shortcut hook (ADR-053, M1).
 *
 * Parses strings like "Ctrl+B", "Ctrl+`", "Ctrl+Shift+P" and binds them
 * globally on the document. Returns the matched shortcut for debugging.
 *
 * Why not `react-hotkeys-hook`? Saves a dep. ~30 LOC, no extra bundle weight.
 */

export interface ShortcutOptions {
  /** Called when the shortcut fires. Event is preventDefault'd before this. */
  onMatch: (e: KeyboardEvent) => void;
  /** Whether to listen at document or window level. Default: document. */
  scope?: "document" | "window";
  /** Ignore if user is typing in input/textarea/contenteditable. Default: true. */
  skipInInputs?: boolean;
}

function parseCombo(combo: string): {
  key: string;
  ctrl: boolean;
  shift: boolean;
  alt: boolean;
  meta: boolean;
} {
  const parts = combo
    .split("+")
    .map((p) => p.trim())
    .filter(Boolean);
  let ctrl = false;
  let shift = false;
  let alt = false;
  let meta = false;
  let key = "";
  for (const p of parts) {
    const lower = p.toLowerCase();
    if (lower === "ctrl" || lower === "control" || lower === "cmd") ctrl = true;
    else if (lower === "shift") shift = true;
    else if (lower === "alt" || lower === "option") alt = true;
    else if (lower === "meta" || lower === "super" || lower === "win") meta = true;
    else key = p.length === 1 ? p.toLowerCase() : p;
  }
  return { key: key.toLowerCase(), ctrl, shift, alt, meta };
}

function isInEditable(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return true;
  if (target.isContentEditable) return true;
  return false;
}

export function useShortcut(combo: string, options: ShortcutOptions): void {
  const { onMatch, scope = "document", skipInInputs = true } = options;
  const parsed = React.useMemo(() => parseCombo(combo), [combo]);

  React.useEffect(() => {
    const handler = (e: Event): void => {
      if (!(e instanceof KeyboardEvent)) return;
      if (skipInInputs && isInEditable(e.target)) return;
      if (parsed.ctrl !== (e.ctrlKey || e.metaKey)) return;
      if (parsed.shift !== e.shiftKey) return;
      if (parsed.alt !== e.altKey) return;
      const key = e.key.toLowerCase();
      if (parsed.key !== key) return;
      e.preventDefault();
      onMatch(e);
    };
    const target = scope === "window" ? window : document;
    target.addEventListener("keydown", handler);
    return () => target.removeEventListener("keydown", handler);
  }, [parsed, onMatch, scope, skipInInputs]);
}
