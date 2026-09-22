import cytoscape, { type Core, type ElementsDefinition, type LayoutOptions } from "cytoscape";

/**
 * cytoscape-init.ts — minimal bundled Cytoscape bootstrapper.
 *
 * Phase 1 (MVP) — uses Cytoscape v3 with a `cose` force-directed layout.
 * No external layout extensions (no `cytoscape-cose-bilkent`, no edgehandles)
 * — keeps bundle lean and zero-CDN per ADR-048 / ADR-050 LLM03.
 *
 * Phase 2: replace `cose` with `cola` (convergence) + add pan/zoom helpers
 * and community-cluster coloring from `data-central.communities`.
 */

export interface GraphNodeDatum {
  id: string;
  /** Display label — maps from `data-central.nodes[i].title`. */
  label: string;
  /** Optional fill color — maps from `data-central.nodes[i].color` (fallback palette). */
  color?: string;
  /** Optional community color override. */
  community_color?: string;
  /** Community display label (e.g. "architecture", "My-Memory"). */
  community_label?: string;
  /** Node radius in px (mapped from pagerank-scaled `radius`). */
  radius?: number;
  in_degree?: number;
  out_degree?: number;
  pagerank?: number;
  is_hub?: boolean;
  [extra: string]: unknown;
}

export interface GraphEdgeDatum {
  id?: string;
  source: string;
  target: string;
  /** Optional edge color from dataset. */
  color?: string;
  relation?: string;
  weight?: number;
  [extra: string]: unknown;
}

export interface GraphDataset {
  nodes: GraphNodeDatum[];
  edges: GraphEdgeDatum[];
}

export interface CytoscapeInitOptions {
  container: HTMLElement;
  dataset: GraphDataset;
}

const FALLBACK_PALETTE = [
  "#6366f1",
  "#ec4899",
  "#14b8a6",
  "#f59e0b",
  "#06b6d4",
  "#8b5cf6",
  "#10b981",
  "#ef4444",
  "#3b82f6",
  "#f97316",
];

function paletteColor(index: number): string {
  return FALLBACK_PALETTE[index % FALLBACK_PALETTE.length] ?? "#64748b";
}

/**
 * Initialize a Cytoscape Core inside the given container with the supplied dataset.
 *
 * Caller is responsible for:
 *   • `cy.destroy()` on unmount (GraphView useEffect cleanup).
 *   • Container sizing (must have non-zero width/height before init).
 */
export function initCytoscape({
  container,
  dataset,
}: CytoscapeInitOptions): Core {
  const elements: ElementsDefinition = {
    nodes: dataset.nodes.map((node, idx) => ({
      data: {
        id: node.id,
        label: node.label,
        color: node.color ?? node.community_color ?? paletteColor(idx),
        community_color: node.community_color,
        community_label: node.community_label,
        radius: node.radius ?? 18,
        in_degree: node.in_degree ?? 0,
        out_degree: node.out_degree ?? 0,
        pagerank: node.pagerank ?? 0,
        is_hub: node.is_hub ?? false,
      },
    })),
    edges: dataset.edges.map((edge, idx) => ({
      data: {
        id: edge.id ?? `e-${idx}-${edge.source}-${edge.target}`,
        source: edge.source,
        target: edge.target,
        color: edge.color ?? "#475569",
        relation: edge.relation,
        weight: edge.weight ?? 1,
      },
    })),
  };

  const layoutOptions: LayoutOptions = {
    name: "cose",
    animate: false,
    fit: true,
    padding: 30,
    nodeRepulsion: () => 8000,
    idealEdgeLength: () => 80,
    gravity: 0.25,
    numIter: 1000,
    randomize: false,
  };

  const cytoscapeOptions = {
    container,
    elements,
    style: [
      {
        selector: "node",
        style: {
          "background-color": "data(color)",
          label: "data(label)",
          color: "#e5e7eb",
          "font-size": 10,
          "text-valign": "center",
          "text-halign": "center",
          "text-outline-color": "#0a0a0a",
          "text-outline-width": 2,
          "text-wrap": "wrap",
          "text-max-width": 120,
          width: "data(radius)",
          height: "data(radius)",
          "border-width": 1.5,
          "border-color": "#0f172a",
          opacity: 0.95,
        },
      },
      {
        selector: "node:selected",
        style: {
          "border-width": 3,
          "border-color": "#fbbf24",
          "background-blacken": -0.15,
        },
      },
      {
        selector: "node[?is_hub]",
        style: {
          "border-width": 2.5,
          "border-color": "#fbbf24",
        },
      },
      {
        selector: "edge",
        style: {
          width: 1.2,
          "line-color": "data(color)",
          "target-arrow-color": "data(color)",
          "target-arrow-shape": "triangle",
          "curve-style": "bezier",
          "arrow-scale": 0.8,
          opacity: 0.6,
        },
      },
      {
        selector: "edge[?selected]",
        style: {
          width: 2,
          opacity: 0.95,
        },
      },
    ],
    layout: layoutOptions,
    wheelSensitivity: 0.2,
    minZoom: 0.2,
    maxZoom: 3.5,
  } as unknown as cytoscape.CytoscapeOptions;

  const cy = cytoscape(cytoscapeOptions);

  return cy;
}

/**
 * Estimate a node radius from pagerank (used when dataset has no `radius`).
 * Phase 1 defensive helper — data-central.json already supplies `radius`.
 */
export function pagerankToRadius(pagerank: number): number {
  const clamped = Math.max(0, Math.min(1, pagerank));
  return 14 + clamped * 60;
}