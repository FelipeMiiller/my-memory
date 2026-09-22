import * as React from "react";
import { FilePlus2, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

/**
 * EditorPane — placeholder for M3's CodeMirror 6 editor (ADR-053).
 */

export function EditorPane(): React.JSX.Element {
  const { t } = useTranslation();
  return (
    <div className="flex h-full flex-col" data-testid="editor-pane">
      <div className="flex items-center justify-between border-b border-border bg-card/30 px-3 py-1.5">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          {t("workspace.editor.title")}
        </div>
        <div className="text-[10px] text-muted-foreground italic">
          CodeMirror 6 + Markdown live preview → M3
        </div>
      </div>
      <div className="flex flex-1 items-center justify-center p-6">
        <div className="max-w-md text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-muted/40">
            <FilePlus2 className="h-6 w-6 text-muted-foreground" />
          </div>
          <h3 className="mb-1 text-sm font-medium">{t("workspace.editor.emptyTitle")}</h3>
          <p className="mb-4 text-xs text-muted-foreground">
            {t("workspace.editor.emptySubtitle")}
          </p>
          <div className="flex justify-center gap-2">
            <Button variant="outline" size="sm" disabled>
              <FilePlus2 className="mr-1 h-3 w-3" />
              {t("workspace.editor.openFile")}
            </Button>
            <Button variant="outline" size="sm" disabled>
              <Search className="mr-1 h-3 w-3" />
              {t("workspace.editor.searchVault")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
