import * as React from 'react';
import { Layout } from '@components/Layout';

/**
 * App.tsx — Phase 1 MVP entry.
 *
 * Renders the full Layout (topbar + Resizable sidebar + Tabs).
 * Tab "Grafo" is the default active tab and mounts `<GraphView />`
 * which loads `./data-central.json` via fetch and renders Cytoscape.
 *
 * Phase 2 will replace this with router-aware Layout + IPC-backed
 * dataset loading via `window.memAPI.loadDataset()`.
 */
export default function App(): React.JSX.Element {
  return <Layout />;
}