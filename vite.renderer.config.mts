import path from "node:path";
import babel from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import { defineConfig } from "vite";

/**
 * my-memory renderer Vite config — Phase 1 MVP.
 *
 * - React 19 + Vite plugin-react with React Compiler (babel preset).
 * - Tailwind 4 via `@tailwindcss/vite` (CSS-first config in src/styles/global.css).
 * - Path alias `@/*` → `./src/*` per tsconfig.json.
 *
 * NOTE: deliberately drops `@tanstack/router-plugin/vite` from the upstream
 * template — my-memory Phase 1 uses shadcn/ui Tabs (Radix) for tab switching,
 * not TanStack Router. Phase 2 may re-introduce Router for Phase 2 features.
 */
export default defineConfig({
  plugins: [
    tailwindcss(),
    react(),
    babel({ presets: [reactCompilerPreset()] }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
    preserveSymlinks: true,
  },
});