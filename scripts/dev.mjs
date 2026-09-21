#!/usr/bin/env node
/**
 * Cross-platform dev orchestrator: spawns Vite dev server + Electron in parallel.
 *
 * Replaces electron-vite / concurrently deps with vanilla child_process spawn.
 * Used by `npm run dev` from workspace root.
 */
import { spawn } from 'node:child_process';
import process from 'node:process';

const isWindows = process.platform === 'win32';
const ELECTRON_BIN = isWindows ? 'node_modules/.bin/electron.cmd' : 'node_modules/.bin/electron';

const procs = [];

function spawnChild(label, cmd, args, opts = {}) {
  const child = spawn(cmd, args, {
    stdio: 'inherit',
    shell: isWindows,
    ...opts,
  });
  child.on('exit', (code) => {
    console.log(`[dev] ${label} exited with code ${code}`);
    procs.forEach((p) => {
      if (p !== child && !p.killed) p.kill();
    });
    process.exit(code ?? 0);
  });
  procs.push(child);
  return child;
}

// Start Vite dev server
spawnChild('vite', 'vite', ['--config', 'viewer/vite.config.ts']);

// Wait for Vite to be ready, then launch Electron
const waitForServer = async (url, timeoutMs = 30000) => {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url);
      if (res.status < 500) return true;
    } catch {
      // not ready yet
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`Vite dev server did not start within ${timeoutMs}ms`);
};

await waitForServer('http://localhost:5173');
console.log('[dev] Vite ready — launching Electron');
spawnChild('electron', ELECTRON_BIN, ['.']);

// Cleanup on Ctrl+C
process.on('SIGINT', () => {
  procs.forEach((p) => p.kill());
  process.exit(0);
});
