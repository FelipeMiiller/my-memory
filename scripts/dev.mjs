#!/usr/bin/env node
/**
 * Dev orchestrator — Electron Forge `start` wrapper.
 *
 * Replaces the old electron-vite + concurrently approach.
 * `npm run dev` from workspace root launches the full Electron + Vite dev stack.
 */
import { spawn } from "node:child_process";
import process from "node:process";

const isWindows = process.platform === "win32";
const FORGE_BIN = isWindows
  ? "node_modules/.bin/electron-forge.cmd"
  : "node_modules/.bin/electron-forge";

const child = spawn(FORGE_BIN, ["start"], {
  stdio: "inherit",
  shell: isWindows,
});

child.on("exit", (code) => {
  process.exit(code ?? 0);
});

process.on("SIGINT", () => {
  child.kill();
  process.exit(0);
});
