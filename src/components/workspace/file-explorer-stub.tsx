import * as React from "react";
import { Folder, FileText, ChevronRight, ChevronDown } from "lucide-react";
import { cn } from "@/utils/tailwind";

/**
 * FileExplorerStub — placeholder for M3's real file explorer (ADR-053).
 *
 * Shows a mock directory tree so the user can see the layout structure.
 * In M3 this will be replaced by a real implementation backed by the
 * `mem:fs:list` IPC channel.
 */

interface MockNode {
  name: string;
  type: "file" | "dir";
  children?: MockNode[];
}

const MOCK_TREE: MockNode[] = [
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
    name: ".memory",
    type: "dir",
    children: [{ name: "config.yaml", type: "file" }],
  },
  { name: "README.md", type: "file" },
];

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
  return (
    <div className="flex h-full flex-col" data-testid="file-explorer-stub">
      <div className="border-b border-border bg-card/30 px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
        Explorer
      </div>
      <div className="flex-1 overflow-auto py-1">
        {MOCK_TREE.map((node) => (
          <Node key={node.name} node={node} depth={0} />
        ))}
      </div>
      <div className="border-t border-border bg-card/30 px-3 py-1 text-[10px] text-muted-foreground italic">
        Mock tree · real impl em M3
      </div>
    </div>
  );
}
