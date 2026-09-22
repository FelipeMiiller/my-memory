import path from "node:path";
import { defineConfig } from "vite";

/**
 * Vite config — Electron main process.
 *
 * Path aliases:
 *   - `@`     → ./src    (renderer; not used here but kept for renderer consistency)
 *   - `@electron` → ./electron (main/preload/ipc — used by main.ts + preload.ts)
 */
export default defineConfig({
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
      "@electron": path.resolve(import.meta.dirname, "./electron"),
    },
  },
});