import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import ptBR from "./locales/pt-BR.json";
import enUS from "./locales/en-US.json";

/**
 * i18n bootstrapper for the viewer renderer.
 *
 * Reads the persisted locale from `localStorage['mem.locale']` (set by the
 * Locale switcher in the topbar). Falls back to `pt-BR` (per VE-16) and
 * ultimately `en-US` if a key is missing in the active locale.
 *
 * Phase 2 may add lazy-loaded namespaces + a remote translation pipeline,
 * but for Phase 1 both locale bundles are statically imported so the
 * renderer never hits the network (zero-CDN, ADR-050 LLM03).
 */

const STORAGE_KEY = "mem.locale";
const DEFAULT_LOCALE = "pt-BR";
const FALLBACK_LOCALE = "en-US";

function readStoredLocale(): string {
  try {
    if (typeof localStorage === "undefined") return DEFAULT_LOCALE;
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "pt-BR" || stored === "en-US") return stored;
    return DEFAULT_LOCALE;
  } catch {
    return DEFAULT_LOCALE;
  }
}

const lng = readStoredLocale();

void i18n.use(initReactI18next).init({
  resources: {
    "pt-BR": { translation: ptBR },
    "en-US": { translation: enUS },
  },
  lng,
  fallbackLng: FALLBACK_LOCALE,
  supportedLngs: ["pt-BR", "en-US"],
  interpolation: {
    escapeValue: false,
  },
  returnNull: false,
});

export default i18n;
export { DEFAULT_LOCALE, FALLBACK_LOCALE, STORAGE_KEY };
export type SupportedLocale = "pt-BR" | "en-US";