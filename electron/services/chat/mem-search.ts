import { execFile } from "node:child_process";
import fs from "node:fs/promises";
import path from "node:path";
import { promisify } from "node:util";
import { app } from "electron";

const execFileP = promisify(execFile);

/**
 * RAG via subprocess (ADR-051).
 *
 * Spawns the local `mem` CLI binary and asks for the top-K results for a
 * query. Returns the parsed JSON. If `mem` is not in the PATH, returns an
 * empty result list (fail-soft) so the chat flow continues without context.
 *
 * The viewer bundle ships without `mem.exe` in production (only `package.json`
 * references it); in dev we look up the binary next to the rendered preload
 * (`out/bin/mem.exe` from `npm run build` or `bin/mem.exe` from `go build`).
 */

export interface MemSearchOptions {
  query: string;
  topK?: number;
  cwd?: string;
}

const DEFAULT_TOP_K = 5;
const EXEC_TIMEOUT_MS = 10_000;

export async function memSearch(
  options: MemSearchOptions,
): Promise<MemSearchResponse> {
  const topK = options.topK ?? DEFAULT_TOP_K;
  if (!options.query || options.query.trim().length === 0) {
    return { query: options.query, results: [] };
  }

  const binary = await resolveMemBinary();
  if (!binary) {
    // Fail-soft — caller decides UX.
    return { query: options.query, results: [] };
  }

  const args = [
    "search",
    options.query,
    "--json",
    "--limit",
    String(topK),
    "--mode",
    "hybrid",
  ];

  try {
    const { stdout } = await execFileP(binary, args, {
      cwd: options.cwd,
      timeout: EXEC_TIMEOUT_MS,
      windowsHide: true,
      maxBuffer: 4 * 1024 * 1024,
    });
    return parseMemSearchOutput(stdout, options.query);
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    console.warn("[chat/mem-search] subprocess failed:", msg);
    return { query: options.query, results: [] };
  }
}

function parseMemSearchOutput(
  stdout: string,
  query: string,
): MemSearchResponse {
  try {
    // The CLI emits JSON either as a bare array or wrapped in {results: [...]}.
    // We handle both shapes defensively.
    const parsed: unknown = JSON.parse(stdout);
    if (Array.isArray(parsed)) {
      return { query, results: normalizeResults(parsed) };
    }
    if (typeof parsed === "object" && parsed !== null && "results" in parsed) {
      const arr = (parsed as { results: unknown }).results;
      if (Array.isArray(arr)) {
        return { query, results: normalizeResults(arr) };
      }
    }
    return { query, results: [] };
  } catch (err: unknown) {
    console.warn("[chat/mem-search] failed to parse output:", err);
    return { query, results: [] };
  }
}

function normalizeResults(
  arr: ReadonlyArray<unknown>,
): ReadonlyArray<MemSearchResult> {
  const out: MemSearchResult[] = [];
  for (const item of arr) {
    if (typeof item !== "object" || item === null) continue;
    const o = item as Record<string, unknown>;
    const path = typeof o.path === "string" ? o.path : "";
    const title = typeof o.title === "string" ? o.title : path;
    const snippet = typeof o.snippet === "string" ? o.snippet : "";
    const score = typeof o.score === "number" ? o.score : 0;
    if (path.length > 0) {
      out.push({ path, title, snippet, score });
    }
  }
  return out;
}

/**
 * Locates the `mem` binary in this order:
 *   1. `MEM_CLI_PATH` env var (set by installer for production)
 *   2. `<userData>/bin/mem.exe` (copied by installer post-install)
 *   3. `<appPath>/../../bin/mem.exe` (dev: go build output alongside source)
 *   4. `<appPath>/mem.exe` (Windows packaged layout)
 *   5. `mem` / `mem.exe` on PATH (last resort)
 */
async function resolveMemBinary(): Promise<string | null> {
  const env = process.env.MEM_CLI_PATH;
  if (env && (await fileExists(env))) return env;

  const userData = app.getPath("userData");
  const candidates = [
    path.join(
      userData,
      "bin",
      process.platform === "win32" ? "mem.exe" : "mem",
    ),
    path.join(
      app.getAppPath(),
      "..",
      "..",
      "bin",
      process.platform === "win32" ? "mem.exe" : "mem",
    ),
    path.join(
      app.getAppPath(),
      process.platform === "win32" ? "mem.exe" : "mem",
    ),
  ];

  for (const c of candidates) {
    if (await fileExists(c)) return c;
  }

  // PATH lookup — only useful in dev environments that installed `mem` globally.
  return process.platform === "win32" ? "mem.exe" : "mem";
}

async function fileExists(p: string): Promise<boolean> {
  try {
    await fs.access(p);
    return true;
  } catch {
    return false;
  }
}
