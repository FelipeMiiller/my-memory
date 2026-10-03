import path from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// my-memory vitest config — Phase 1 MVP.
//
// Covers:
//   - Electron main + preload unit tests (src/main.test.ts, src/preload.test.ts)
//   - Renderer component tests (src tests .ts .tsx)
//
// Path alias `@/*` mirrors tsconfig.json.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.{ts,tsx}"],
    exclude: [
      "node_modules",
      "dist",
      "out",
      ".vite",
      "**/dist/**",
      "**/out/**",
      "src/tests/e2e/**",
    ],
    globals: true,
    setupFiles: ["./scripts/vitest.setup.ts"],
  },
});
