import * as React from "react";
import { useTranslation } from "react-i18next";
import { Check, Globe } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { STORAGE_KEY, DEFAULT_LOCALE, type SupportedLocale } from "@/localization/i18n";

/**
 * LocaleSwitcher.tsx — topbar control that:
 *   • Reads the current locale from i18next.
 *   • Lets the user pick `pt-BR` or `en-US`.
 *   • Persists the choice in `localStorage[mem.locale]` so future renders
 *     (page reload, app re-launch) honour it via `@/localization/i18n`.
 *
 * Built on shadcn/ui primitives (Button + DropdownMenu from Radix).
 */

interface LocaleOption {
  code: SupportedLocale;
  labelKey: string;
}

const OPTIONS: ReadonlyArray<LocaleOption> = [
  { code: "pt-BR", labelKey: "locale.pt-BR" },
  { code: "en-US", labelKey: "locale.en-US" },
];

export function LocaleSwitcher(): React.JSX.Element {
  const { t, i18n } = useTranslation();
  const [current, setCurrent] = React.useState<SupportedLocale>(
    (i18n.resolvedLanguage ?? i18n.language ?? DEFAULT_LOCALE) as SupportedLocale,
  );

  React.useEffect(() => {
    const handler = (lng: string): void => {
      if (lng === "pt-BR" || lng === "en-US") setCurrent(lng);
    };
    i18n.on("languageChanged", handler);
    return () => {
      i18n.off("languageChanged", handler);
    };
  }, [i18n]);

  const handleSelect = React.useCallback(
    (code: SupportedLocale) => {
      try {
        localStorage.setItem(STORAGE_KEY, code);
      } catch {
        // localStorage unavailable — best-effort only.
      }
      void i18n.changeLanguage(code);
      setCurrent(code);
    },
    [i18n],
  );

  const currentLabel =
    OPTIONS.find((o) => o.code === current)?.code ?? DEFAULT_LOCALE;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className="h-8 gap-1.5 px-2 text-xs font-medium"
          data-testid="locale-switcher"
          aria-label={t("locale.label")}
        >
          <Globe className="h-3.5 w-3.5" aria-hidden="true" />
          <span>{currentLabel}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        <DropdownMenuLabel className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {t("locale.label")}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {OPTIONS.map((opt) => {
          const active = opt.code === current;
          return (
            <DropdownMenuItem
              key={opt.code}
              data-testid={`locale-option-${opt.code}`}
              data-active={active ? "true" : "false"}
              onSelect={() => handleSelect(opt.code)}
              className="flex items-center justify-between"
            >
              <span className="text-sm">{t(opt.labelKey)}</span>
              {active && (
                <Check
                  className="h-3.5 w-3.5 text-primary"
                  aria-hidden="true"
                />
              )}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default LocaleSwitcher;