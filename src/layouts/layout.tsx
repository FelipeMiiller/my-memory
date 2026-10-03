import { Brain, Github, Moon } from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";
import { ChatView } from "@/components/chat";
import { GraphView } from "@/components/graph";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/utils/tailwind";

/**
 * Layout.tsx — my-memory viewer shell (Phase 1 MVP).
 *
 * Layout structure:
 *   ┌──────────────────────────────────────────────────────┐
 *   │ Topbar: logo + title + LocaleSwitcher + GitHub + Dark │
 *   ├─────────────┬────────────────────────────────────────┤
 *   │ Sidebar     │  Main (Tabs: Grafo / Code / Staleness)  │
 *   │ (Resize)    │                                        │
 *   └─────────────┴────────────────────────────────────────┘
 *
 * - LocaleSwitcher: topbar dropdown switching `pt-BR` ↔ `en-US`,
 *   persisted in `localStorage[mem.locale]`.
 * - Resizable sidebar (shadcn `Resizable`) — collapsible via the handle.
 * - Tabs (Radix + shadcn) with 3 triggers:
 *     • Graph (active, mounts `<GraphView />`)
 *     • Code (disabled — Phase 2)
 *     • Staleness (disabled — Phase 2)
 * - Phase 2 placeholders rendered as Card with "Coming in Phase 2" copy.
 */
export default function Layout(): React.JSX.Element {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = React.useState<string>("graph");

  return (
    <TooltipProvider delayDuration={200}>
      <div className="flex h-screen w-screen flex-col overflow-hidden bg-background text-foreground">
        {/* Topbar */}
        <header
          className={cn(
            "flex h-14 shrink-0 items-center justify-between border-b border-border",
            "bg-card/40 px-4 backdrop-blur-sm",
          )}
          data-testid="topbar"
        >
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary/10 text-primary">
              <Brain className="h-5 w-5" aria-hidden="true" />
            </div>
            <div className="flex flex-col leading-tight">
              <span className="text-sm font-semibold">{t("app.title")}</span>
              <span className="text-xs text-muted-foreground">
                {t("app.subtitle")}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Badge
              variant="secondary"
              className="gap-1"
              data-testid="theme-badge"
            >
              <Moon className="h-3 w-3" aria-hidden="true" />
              {t("badge.dark")}
            </Badge>
            <LocaleSwitcher />
            <a
              href="https://github.com/FelipeMiiller/my-memory"
              target="_blank"
              rel="noreferrer noopener"
              className="inline-flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
              aria-label={t("github.label")}
            >
              <Github className="h-4 w-4" aria-hidden="true" />
            </a>
          </div>
        </header>

        {/* Body: sidebar + main */}
        <ResizablePanelGroup
          direction="horizontal"
          className="flex-1"
          data-testid="layout-body"
        >
          <ResizablePanel
            defaultSize={22}
            minSize={14}
            maxSize={40}
            className="bg-card/20"
          >
            <aside className="flex h-full flex-col border-r border-border">
              <div className="border-b border-border px-4 py-3">
                <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {t("nav.title")}
                </h2>
              </div>
              <ScrollArea className="flex-1">
                <nav
                  className="flex flex-col gap-1 px-2 py-3"
                  data-testid="sidebar-nav"
                >
                  <SidebarLink
                    label={t("nav.graph")}
                    description={t("nav.graphDesc")}
                    active={activeTab === "graph"}
                    onClick={() => setActiveTab("graph")}
                    testId="sidebar-link-grafo"
                  />
                  <SidebarLink
                    label={t("nav.code")}
                    description={t("nav.codeDesc")}
                    active={activeTab === "code"}
                    onClick={() => setActiveTab("code")}
                    disabled
                    testId="sidebar-link-code"
                  />
                  <SidebarLink
                    label={t("nav.staleness")}
                    description={t("nav.stalenessDesc")}
                    active={activeTab === "staleness"}
                    onClick={() => setActiveTab("staleness")}
                    disabled
                    testId="sidebar-link-staleness"
                  />
                  <SidebarLink
                    label={t("nav.chat")}
                    description={t("nav.chatDesc")}
                    active={activeTab === "chat"}
                    onClick={() => setActiveTab("chat")}
                    testId="sidebar-link-chat"
                  />
                </nav>
              </ScrollArea>
              <div className="border-t border-border px-4 py-3 text-[11px] text-muted-foreground">
                <p>
                  {t("footer.dataset")}{" "}
                  <code className="rounded bg-muted px-1 py-0.5 font-mono text-[10px]">
                    data-central.json
                  </code>
                </p>
                <p className="mt-1">{t("footer.phase1")}</p>
              </div>
            </aside>
          </ResizablePanel>

          <ResizableHandle withHandle />

          <ResizablePanel defaultSize={78} minSize={40}>
            <main
              className="flex h-full flex-col overflow-hidden"
              data-testid="layout-main"
            >
              <Tabs
                value={activeTab}
                onValueChange={setActiveTab}
                className="flex h-full flex-col"
              >
                <div className="border-b border-border bg-card/30 px-4 py-3">
                  <TabsList className="inline-flex h-9" data-testid="tabs-list">
                    <TabsTrigger value="graph" data-testid="tab-trigger-graph">
                      {t("tab.graph")}
                    </TabsTrigger>

                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex">
                          <TabsTrigger
                            value="code"
                            disabled
                            data-testid="tab-trigger-code"
                          >
                            {t("tab.code")}
                          </TabsTrigger>
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>{t("phase2.codeTooltip")}</TooltipContent>
                    </Tooltip>

                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex">
                          <TabsTrigger
                            value="staleness"
                            disabled
                            data-testid="tab-trigger-staleness"
                          >
                            {t("tab.staleness")}
                          </TabsTrigger>
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>
                        {t("phase2.stalenessTooltip")}
                      </TooltipContent>
                    </Tooltip>

                    <TabsTrigger value="chat" data-testid="tab-trigger-chat">
                      {t("tab.chat")}
                    </TabsTrigger>
                  </TabsList>
                </div>

                <TabsContent
                  value="graph"
                  className="mt-0 flex-1 overflow-hidden"
                  data-testid="tab-content-graph"
                >
                  <GraphView />
                </TabsContent>

                <TabsContent
                  value="code"
                  className="mt-0 flex-1 overflow-auto p-6"
                  data-testid="tab-content-code"
                >
                  <PhasePlaceholder
                    title={t("phase2.code.title")}
                    description={t("phase2.code.description")}
                    placeholderKey="code"
                  />
                </TabsContent>

                <TabsContent
                  value="staleness"
                  className="mt-0 flex-1 overflow-auto p-6"
                  data-testid="tab-content-staleness"
                >
                  <PhasePlaceholder
                    title={t("phase2.staleness.title")}
                    description={t("phase2.staleness.description")}
                    placeholderKey="staleness"
                  />
                </TabsContent>

                <TabsContent
                  value="chat"
                  className="mt-0 flex-1 overflow-hidden"
                  data-testid="tab-content-chat"
                >
                  <ChatView />
                </TabsContent>
              </Tabs>
            </main>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
    </TooltipProvider>
  );
}

interface SidebarLinkProps {
  label: string;
  description: string;
  active: boolean;
  onClick: () => void;
  disabled?: boolean;
  testId: string;
}

function SidebarLink({
  label,
  description,
  active,
  onClick,
  disabled = false,
  testId,
}: SidebarLinkProps): React.JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      data-testid={testId}
      className={cn(
        "flex flex-col items-start gap-0.5 rounded-md px-3 py-2 text-left text-sm transition-colors",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        active && "bg-accent text-accent-foreground",
        !active &&
          !disabled &&
          "hover:bg-accent/60 hover:text-accent-foreground",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      <span className="font-medium">{label}</span>
      <span className="text-[11px] text-muted-foreground">{description}</span>
    </button>
  );
}

interface PhasePlaceholderProps {
  title: string;
  description: string;
  placeholderKey: string;
}

function PhasePlaceholder({
  title,
  description,
  placeholderKey,
}: PhasePlaceholderProps): React.JSX.Element {
  const { t } = useTranslation();
  return (
    <Card className="mx-auto max-w-2xl">
      <CardHeader>
        <CardTitle data-testid={`placeholder-title-${placeholderKey}`}>
          {title}
        </CardTitle>
        <CardDescription>{t("phase2.coming_soon")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-2 text-sm text-muted-foreground">
        <p>{description}</p>
        <p>{t("phase2.graphTabNotice", { tab: t("tab.graph") })}</p>
      </CardContent>
    </Card>
  );
}
