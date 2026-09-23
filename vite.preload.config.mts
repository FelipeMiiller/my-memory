import path from "node:path";
import { defineConfig } from "vite";

/**
 * Vite config — Electron preload.
 * Mirrors vite.main.config.mts path aliases (preload imports from `@electron/...`).
 */
export default defineConfig({
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
      "@electron": path.resolve(import.meta.dirname, "./electron"),
    },
  },
});
