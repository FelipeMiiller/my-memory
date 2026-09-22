import i18n from "i18next";
import { initReactI18next } from "react-i18next";

// Per-namespace bundles — keeps each translation file small, makes diffs
// easier to review, and lets a translator copy an entire language folder
// without touching unrelated strings.
import commonEN from "./locales/en-US/common.json";
import graphEN from "./locales/en-US/graph.json";
import topbarEN from "./locales/en-US/topbar.json";
import workspaceEN from "./locales/en-US/workspace.json";
import commonPT from "./locales/pt-BR/common.json";
import graphPT from "./locales/pt-BR/graph.json";
import topbarPT from "./locales/pt-BR/topbar.json";
import workspacePT from "./locales/pt-BR/workspace.json";

/**
 * i18n bootstrapper for the viewer renderer.
 *
 * Locale storage: `localStorage['mem.locale']` (set by `LocaleSwitcher`).
 * Default is `pt-BR` (per VE-16), fallback chain ends at `en-US`.
 *
 * ─── Adding a new language ───────────────────────────────────────────────
 * 1. Copy the whole `pt-BR/` folder:
 *      cp -r src/localization/locales/pt-BR src/localization/locales/<new>
 *    (same shape for every namespace: common / topbar / graph / workspace)
 * 2. Translate each `*.json` file under `locales/<new>/`. Keep the key paths
 *    identical — only the string values change.
 * 3. Register the new locale in three places (all in `src/localization/`):
 *      • this file — add imports, add an entry to `RESOURCES` below, and add
 *        the code to `readStoredLocale`'s allowlist + `SUPPORTED_LOCALES`.
 *      • `components/locale-switcher.tsx` — append `{ code: "<new>",
 *        labelKey: "locale.<new>" }` to the `OPTIONS` array.
 * 4. Add `"locale.<new>": "<Native name>"` to **every** language's
 *    `common.json` so the locale-switcher dropdown can show the native label.
 *
 * Adding a new namespace (e.g. `settings`):
 *  • Create `locales/<locale>/settings.json` for every locale — only do
 *    this once you actually have keys to put in it (no empty stubs).
 *  • Add the import + `RESOURCES[locale].settings = ...` wiring here.
 *  • List the namespace in `NAMESPACES` below so i18next pre-loads it.
 *
 * Phase 2 may swap these static imports for a lazy-loaded backend (e.g.
 * fetch + i18next-fs-backend) once locales grow beyond a handful — until
 * then everything is bundled so the renderer stays offline-capable
 * (ADR-050 LLM03).
 */

const STORAGE_KEY = "mem.locale";
const DEFAULT_LOCALE = "pt-BR";
const FALLBACK_LOCALE = "en-US";

export const SUPPORTED_LOCALES = ["pt-BR", "en-US"] as const;
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

const NAMESPACES = ["common", "topbar", "graph", "workspace"] as const;
export type Namespace = (typeof NAMESPACES)[number];

const RESOURCES = {
  "pt-BR": {
    common: commonPT,
    topbar: topbarPT,
    graph: graphPT,
    workspace: workspacePT,
  },
  "en-US": {
    common: commonEN,
    topbar: topbarEN,
    graph: graphEN,
    workspace: workspaceEN,
  },
} as const satisfies Record<SupportedLocale, Record<Namespace, unknown>>;

function readStoredLocale(): string {
  try {
    if (typeof localStorage === "undefined") return DEFAULT_LOCALE;
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && (SUPPORTED_LOCALES as readonly string[]).includes(stored))
      return stored;
    return DEFAULT_LOCALE;
  } catch {
    return DEFAULT_LOCALE;
  }
}

const lng = readStoredLocale();

void i18n.use(initReactI18next).init({
  resources: RESOURCES,
  lng,
  fallbackLng: FALLBACK_LOCALE,
  supportedLngs: SUPPORTED_LOCALES as unknown as string[],
  ns: NAMESPACES as unknown as string[],
  defaultNS: "common",
  interpolation: {
    escapeValue: false,
  },
  returnNull: false,
});

export default i18n;
export { DEFAULT_LOCALE, FALLBACK_LOCALE, NAMESPACES, STORAGE_KEY };
