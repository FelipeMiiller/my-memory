import * as React from 'react';
import { useTranslation } from 'react-i18next';
import type { Core as CytoscapeCore, EventObject } from 'cytoscape';
import { AlertCircle, Loader2 } from 'lucide-react';
import { Badge } from '@ui/badge';
import { Card, CardContent } from '@ui/card';
import {
  initCytoscape,
  pagerankToRadius,
  type GraphDataset,
  type GraphNodeDatum,
} from '@lib/cytoscape-init';

/**
 * GraphView.tsx — Phase 1 MVP interactive graph.
 *
 * On mount:
 *   1. fetch('./data-central.json') — static asset bundled by Vite from viewer/public/.
 *   2. Map nodes/edges into Cytoscape ElementsDefinition via `initCytoscape()`.
 *   3. Apply `cose` force-directed layout + fit to viewport.
 *   4. Wire `tap` event on nodes → set `selectedNode` state → render shadcn
 *      Tooltip-like overlay panel (right side) with label/community/path/degree.
 *
 * Phase 2:
 *   • Replace fetch with `window.memAPI.loadDataset()` (IPC `mem:dataset:load`).
 *   • Triptych Inspector (ADR-024) for the selected node.
 *   • Community-cluster concentric layouts + side panel filters.
 */

interface SelectedNode {
  id: string;
  label: string;
  community_label?: string;
  community_color?: string;
  in_degree: number;
  out_degree: number;
  pagerank: number;
  is_hub: boolean;
  color: string;
}

type LoadState =
  | { kind: 'idle' }
  | { kind: 'loading' }
  | { kind: 'ready'; dataset: GraphDataset; selected: SelectedNode | null }
  | { kind: 'error'; message: string };

export function GraphView(): React.JSX.Element {
  const { t } = useTranslation();
  const containerRef = React.useRef<HTMLDivElement | null>(null);
  const cyRef = React.useRef<CytoscapeCore | null>(null);
  const [state, setState] = React.useState<LoadState>({ kind: 'idle' });

  React.useEffect(() => {
    let cancelled = false;

    setState({ kind: 'loading' });

    fetch('./data-central.json')
      .then((res) => {
        if (!res.ok) {
          throw new Error(
            `Failed to fetch data-central.json (HTTP ${res.status})`
          );
        }
        return res.json() as Promise<GraphDataset>;
      })
      .then((dataset) => {
        if (cancelled) return;
        const container = containerRef.current;
        if (!container) {
          setState({ kind: 'error', message: 'Container ref not attached' });
          return;
        }

        // Tear down any previous instance (StrictMode double-invoke safety).
        if (cyRef.current) {
          cyRef.current.destroy();
          cyRef.current = null;
        }

        const instance = initCytoscape({ container, dataset });

        const handleTap = (event: EventObject): void => {
          const node = event.target;
          const data = node.data() as Record<string, unknown>;
          const selected: SelectedNode = {
            id: String(data['id'] ?? ''),
            label: String(data['label'] ?? ''),
            community_label:
              typeof data['community_label'] === 'string'
                ? (data['community_label'] as string)
                : undefined,
            community_color:
              typeof data['community_color'] === 'string'
                ? (data['community_color'] as string)
                : undefined,
            in_degree: Number(data['in_degree'] ?? 0),
            out_degree: Number(data['out_degree'] ?? 0),
            pagerank: Number(data['pagerank'] ?? 0),
            is_hub: Boolean(data['is_hub']),
            color: String(data['color'] ?? '#64748b'),
          };
          setState({ kind: 'ready', dataset, selected });
        };

        const handleBackground = (): void => {
          setState((prev) =>
            prev.kind === 'ready' ? { ...prev, selected: null } : prev
          );
        };

        instance.on('tap', 'node', handleTap);
        instance.on('tap', handleBackground);

        cyRef.current = instance;
        setState({ kind: 'ready', dataset, selected: null });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        const message =
          err instanceof Error ? err.message : 'Unknown dataset error';
        setState({ kind: 'error', message });
      });

    return () => {
      cancelled = true;
      if (cyRef.current) {
        cyRef.current.destroy();
        cyRef.current = null;
      }
    };
  }, []);

  return (
    <div
      className="relative flex h-full w-full overflow-hidden bg-background"
      data-testid="graph-view"
    >
      {/* Cytoscape container */}
      <div
        ref={containerRef}
        className="h-full w-full"
        data-testid="cytoscape-container"
        aria-label="Knowledge graph canvas"
      />

      {/* Loading overlay */}
      {state.kind === 'loading' && (
        <div
          className="pointer-events-none absolute inset-0 flex items-center justify-center bg-background/60 backdrop-blur-sm"
          data-testid="graph-loading"
        >
          <div className="flex items-center gap-2 rounded-md border border-border bg-card px-4 py-2 text-sm shadow">
            <Loader2 className="h-4 w-4 animate-spin text-primary" />
            {t('graph.loading')}
          </div>
        </div>
      )}

      {/* Error overlay */}
      {state.kind === 'error' && (
        <div
          className="absolute inset-0 flex items-center justify-center bg-background/70 p-6"
          data-testid="graph-error"
        >
          <Card className="max-w-md border-destructive/40">
            <CardContent className="flex items-start gap-3 p-4">
              <AlertCircle className="mt-0.5 h-5 w-5 text-destructive" />
              <div className="space-y-1 text-sm">
                <p className="font-medium">{t('graph.errorTitle')}</p>
                <p className="text-muted-foreground">{state.message}</p>
                <p className="text-xs text-muted-foreground">
                  {t('graph.errorHint')}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Selected-node panel (right) */}
      {state.kind === 'ready' && state.selected && (
        <SelectedNodePanel node={state.selected} />
      )}

      {/* Dataset summary footer */}
      {state.kind === 'ready' && (
        <footer className="pointer-events-none absolute bottom-2 left-2 flex items-center gap-2 rounded-md border border-border bg-card/80 px-3 py-1.5 text-xs text-muted-foreground shadow-sm backdrop-blur">
          <span>
            <strong className="text-foreground">
              {state.dataset.nodes.length}
            </strong>{' '}
            {t('graph.summaryNodes', { count: state.dataset.nodes.length })}
          </span>
          <span aria-hidden="true">·</span>
          <span>
            <strong className="text-foreground">
              {state.dataset.edges.length}
            </strong>{' '}
            {t('graph.summaryEdges', { count: state.dataset.edges.length })}
          </span>
          {state.selected && (
            <>
              <span aria-hidden="true">·</span>
              <Badge variant="secondary" className="text-[10px]">
                {t('graph.selected', { label: state.selected.label })}
              </Badge>
            </>
          )}
        </footer>
      )}
    </div>
  );
}

function SelectedNodePanel({
  node,
}: {
  node: SelectedNode;
}): React.JSX.Element {
  const { t } = useTranslation();
  return (
    <aside
      className="pointer-events-auto absolute right-3 top-3 w-72 max-w-[80%] rounded-lg border border-border bg-card/95 p-4 shadow-lg backdrop-blur"
      data-testid="selected-node-panel"
    >
      <header className="mb-3 flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="truncate text-sm font-semibold" title={node.label}>
            {node.label}
          </h3>
          {node.community_label && (
            <p className="mt-0.5 text-xs text-muted-foreground">
              {node.community_label}
            </p>
          )}
        </div>
        {node.is_hub && (
          <Badge variant="default" className="shrink-0 text-[10px]">
            {t('badge.hub')}
          </Badge>
        )}
      </header>

      <dl className="space-y-1.5 text-xs">
        <Stat label={t('inspector.id')} value={node.id} mono />
        <Stat
          label={t('inspector.color')}
          value={
            <span className="inline-flex items-center gap-1.5">
              <span
                aria-hidden="true"
                className="inline-block h-3 w-3 rounded-sm border border-border"
                style={{ backgroundColor: node.color }}
              />
              {node.color}
            </span>
          }
        />
        <Stat label={t('inspector.inDegree')} value={String(node.in_degree)} />
        <Stat label={t('inspector.outDegree')} value={String(node.out_degree)} />
        <Stat
          label={t('inspector.pageRank')}
          value={node.pagerank.toFixed(4)}
          mono
        />
        <Stat
          label={t('inspector.radius')}
          value={String(Math.round(pagerankToRadius(node.pagerank)))}
        />
      </dl>
    </aside>
  );
}

function Stat({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}): React.JSX.Element {
  return (
    <div className="flex items-center justify-between gap-2">
      <dt className="text-muted-foreground">{label}</dt>
      <dd
        className={
          mono
            ? 'max-w-[60%] truncate font-mono text-[11px]'
            : 'max-w-[60%] truncate text-foreground'
        }
      >
        {value}
      </dd>
    </div>
  );
}

export default GraphView;

// Re-export the dataset type so consumers (tests) can cast JSON correctly.
export type { GraphDataset, GraphNodeDatum };