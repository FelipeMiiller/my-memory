import * as React from "react";
import { Folder, FileText, ChevronRight, ChevronDown, Database, Star } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/utils/tailwind";

/**
 * FileExplorerStub — Central Vault + federated repositories (ADR-053).
 *
 * Mock layout that reflects the real my-memory vault topology:
 *   1. Central Vault at the TOP (with star icon) — federation hub
 *   2. Current repository (highlighted, "active" badge)
 *   3. Other federated repositories
 *
 * M3 will replace the mocks with real `mem:fs:list` IPC results. The shape
 * (Central + repos + tree) stays.
 */

interface VaultMock {
  id: string;
  name: string;
  kind: "central" | "current" | "federated";
  /** Show small tree of files inside the vault. */
  tree: MockNode[];
}

interface MockNode {
  name: string;
  type: "file" | "dir";
  children?: MockNode[];
}

const CENTRAL_VAULT: VaultMock = {
  id: "central",
  name: "Central · ~/KnowledgeVault",
  kind: "central",
  tree: [
    {
      name: "go-patterns.md",
      type: "file",
    },
    {
      name: "concepts",
      type: "dir",
      children: [
        { name: "federated-wikilinks.md", type: "file" },
        { name: "compile-not-retrieve.md", type: "file" },
      ],
    },
    {
      name: "books",
      type: "dir",
      children: [
        { name: "clean-architecture.md", type: "file" },
        { name: "designing-data-intensive.md", type: "file" },
      ],
    },
  ],
};

const CURRENT_REPO: VaultMock = {
  id: "my-memory",
  name: "FelipeMiiller/my-memory",
  kind: "current",
  tree: [
    {
      name: "docs",
      type: "dir",
      children: [
        { name: "ARCHITECTURE.md", type: "file" },
        { name: "CLI_GUIDE.md", type: "file" },
        { name: "VIEWER.md", type: "file" },
      ],
    },
    {
      name: ".specs",
      type: "dir",
      children: [
        { name: "STATE.md", type: "file" },
        { name: "ROADMAP-v3-workspace-architecture.md", type: "file" },
      ],
    },
    {
      name: "src",
      type: "dir",
      children: [
        { name: "app.tsx", type: "file" },
        { name: "lib", type: "dir" },
        { name: "components", type: "dir" },
      ],
    },
    {
      name: "electron",
      type: "dir",
      children: [
        { name: "main.ts", type: "file" },
        { name: "preload.ts", type: "file" },
      ],
    },
    {
      name: ".memory",
      type: "dir",
      children: [{ name: "config.yaml", type: "file" }],
    },
    { name: "README.md", type: "file" },
  ],
};

const FEDERATED_REPOS: VaultMock[] = [
  {
    id: "federated-resume",
    name: "FelipeMiiller/resume",
    kind: "federated",
    tree: [
      { name: "README.md", type: "file" },
      { name: "experiences.md", type: "file" },
    ],
  },
];

const ALL_VAULTS: VaultMock[] = [CENTRAL_VAULT, CURRENT_REPO, ...FEDERATED_REPOS];

function VaultHeader({
  vault,
  title,
}: {
  vault: VaultMock;
  title: string;
}): React.JSX.Element {
  const isCentral = vault.kind === "central";
  const isCurrent = vault.kind === "current";
  const Icon = isCentral ? Star : Database;
  const iconColor = isCentral ? "text-yellow-500" : isCurrent ? "text-emerald-500" : "text-blue-400";
  return (
    <div
      className={cn(
        "sticky top-0 z-10 flex items-center gap-1.5 border-b border-border bg-card/60 px-2 py-1 backdrop-blur",
        "text-[10px] font-semibold uppercase tracking-wider",
      )}
    >
      <Icon className={cn("h-3 w-3 shrink-0", iconColor)} />
      <span className="flex-1 truncate text-foreground/80">{title}</span>
      {isCurrent && (
        <span
          className="rounded bg-emerald-500/15 px-1.5 py-0.5 text-[9px] font-medium normal-case tracking-normal text-emerald-500"
          data-testid="explorer-current-badge"
        >
          ● {title === "Explorador" ? "" : "ativo"}
        </span>
      )}
    </div>
  );
}

function VaultSection({ vault, label }: { vault: VaultMock; label: string }): React.JSX.Element {
  const [open, setOpen] = React.useState(true);
  return (
    <div className="border-b border-border/40">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className={cn(
          "flex w-full items-center gap-1 px-2 py-1.5 text-left text-[11px] font-medium",
          "hover:bg-accent/40",
        )}
        data-testid={`explorer-vault-${vault.kind}`}
      >
        {open ? (
          <ChevronDown className="h-3 w-3 shrink-0 opacity-60" />
        ) : (
          <ChevronRight className="h-3 w-3 shrink-0 opacity-60" />
        )}
        <span className="flex-1 truncate">{vault.name}</span>
        <span className="text-[9px] text-muted-foreground">{vault.tree.length}</span>
      </button>
      {open && (
        <div className="pb-1">
          {vault.tree.map((node) => (
            <Node key={node.name} node={node} depth={1} />
          ))}
        </div>
      )}
    </div>
  );
}

function Node({ node, depth }: { node: MockNode; depth: number }): React.JSX.Element {
  const [open, setOpen] = React.useState(depth < 2);
  const isDir = node.type === "dir";
  return (
    <div>
      <button
        type="button"
        onClick={() => isDir && setOpen(!open)}
        className={cn(
          "flex w-full items-center gap-1 rounded px-1 py-0.5 text-left text-[12px]",
          "hover:bg-accent/40",
        )}
        style={{ paddingLeft: `${depth * 12 + 4}px` }}
      >
        {isDir ? (
          open ? (
            <ChevronDown className="h-3 w-3 shrink-0 opacity-60" />
          ) : (
            <ChevronRight className="h-3 w-3 shrink-0 opacity-60" />
          )
        ) : (
          <span className="inline-block h-3 w-3 shrink-0" />
        )}
        {isDir ? (
          <Folder className="h-3.5 w-3.5 shrink-0 text-amber-500" />
        ) : (
          <FileText className="h-3.5 w-3.5 shrink-0 text-blue-400" />
        )}
        <span className="truncate">{node.name}</span>
      </button>
      {isDir && open && node.children && (
        <div>
          {node.children.map((child) => (
            <Node key={child.name} node={child} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  );
}

export function FileExplorerStub(): React.JSX.Element {
  const { t } = useTranslation();

  return (
    <div className="flex h-full flex-col overflow-hidden" data-testid="file-explorer-stub">
      <VaultHeader vault={CURRENT_REPO} title={t("workspace.explorer.title")} />
      <div className="flex-1 overflow-auto">
        {ALL_VAULTS.map((vault) => {
          const label =
            vault.kind === "central"
              ? t("workspace.explorer.centralVault")
              : vault.kind === "current"
                ? t("workspace.explorer.currentRepo")
                : t("workspace.explorer.federatedRepos");
          return <VaultSection key={vault.id} vault={vault} label={label} />;
        })}
      </div>
      <div className="border-t border-border bg-card/30 px-2 py-1 text-[10px] text-muted-foreground italic">
        {t("workspace.explorer.comingInM3")}
      </div>
    </div>
  );
}
