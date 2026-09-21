import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import cytoscape from 'cytoscape';
import { GraphView } from './GraphView';

/**
 * Mock cytoscape with a controllable Core so we can drive `tap` events
 * and assert that GraphView populates the SelectedNode panel.
 *
 * Cytoscape's `on()` overloads:
 *   cy.on('tap', handler)              → handler is 2nd arg (function)
 *   cy.on('tap', 'node', handler)      → handler is 3rd arg, selector is 2nd
 *
 * We track BOTH variants so the test can fire node-tap events.
 */
const tapHandlers: Array<(evt: unknown) => void> = [];
const mockCore = {
  on: vi.fn((event: string, selectorOrHandler: unknown, handler?: unknown) => {
    if (event !== 'tap') return mockCore;
    if (typeof selectorOrHandler === 'function') {
      tapHandlers.push(selectorOrHandler as (evt: unknown) => void);
    } else if (typeof handler === 'function') {
      tapHandlers.push(handler as (evt: unknown) => void);
    }
    return mockCore;
  }),
  destroy: vi.fn(),
  fit: vi.fn(),
  resize: vi.fn(),
};

vi.mock('cytoscape', () => {
  const cytoscape = vi.fn(() => mockCore);
  return { default: cytoscape };
});

const sampleDataset = {
  nodes: [
    {
      id: 'alpha',
      title: 'alpha',
      label: 'alpha',
      color: '#6366f1',
      community_label: 'core',
      community_color: '#6366f1',
      radius: 22,
      in_degree: 5,
      out_degree: 1,
      pagerank: 0.45,
      is_hub: true,
    },
    {
      id: 'beta',
      title: 'beta',
      label: 'beta',
      color: '#ec4899',
      community_label: 'core',
      community_color: '#ec4899',
      radius: 18,
      in_degree: 2,
      out_degree: 3,
      pagerank: 0.12,
    },
  ],
  edges: [
    { source: 'alpha', target: 'beta', color: '#475569', relation: 'links_to' },
  ],
};

describe('GraphView (Phase 1)', () => {
  beforeEach(() => {
    tapHandlers.length = 0;
    vi.clearAllMocks();
    vi.mocked(cytoscape).mockClear();
    // jsdom fetch polyfill is missing — provide a minimal Response shim.
    globalThis.fetch = vi.fn(async () =>
      ({
        ok: true,
        status: 200,
        json: async () => sampleDataset,
      }) as unknown as Response
    );
  });

  it('renders the cytoscape container element', () => {
    render(<GraphView />);
    expect(screen.getByTestId('cytoscape-container')).toBeTruthy();
  });

  it('calls initCytoscape (cytoscape()) with the fetched dataset', async () => {
    render(<GraphView />);
    await waitFor(() => {
      expect(vi.mocked(cytoscape)).toHaveBeenCalled();
    });
    const fn = cytoscape as unknown as { mock: { calls: unknown[][] } };
    const call = (fn.mock.calls[0]?.[0] ?? {}) as Record<string, unknown>;
    expect(call).toBeDefined();
    const elements = call['elements'] as {
      nodes: unknown[];
      edges: unknown[];
    };
    expect(elements.nodes.length).toBe(2);
    expect(elements.edges.length).toBe(1);
    expect(call['style']).toBeTruthy();
    const layout = call['layout'] as { name?: string };
    expect(layout.name).toBe('cose');
  });

  it('shows loading overlay initially, then hides it on success', async () => {
    render(<GraphView />);
    // Loading appears synchronously
    expect(screen.getByTestId('graph-loading')).toBeTruthy();
    await waitFor(() => {
      expect(screen.queryByTestId('graph-loading')).toBeNull();
    });
  });

  it('shows error overlay when fetch fails', async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: false,
      status: 500,
      json: async () => ({}) as unknown,
    }) as unknown as Response);
    render(<GraphView />);
    await waitFor(() => {
      expect(screen.getByTestId('graph-error')).toBeTruthy();
    });
  });

  it('shows selected-node panel after a node tap event', async () => {
    render(<GraphView />);
    await waitFor(() => {
      expect(vi.mocked(cytoscape)).toHaveBeenCalled();
    });
    // Simulate tapping the alpha node
    tapHandlers[0]?.({
      target: {
        data: () => ({
          id: 'alpha',
          label: 'alpha',
          community_label: 'core',
          community_color: '#6366f1',
          in_degree: 5,
          out_degree: 1,
          pagerank: 0.45,
          is_hub: true,
          color: '#6366f1',
        }),
      },
    });
    await waitFor(() => {
      expect(screen.getByTestId('selected-node-panel')).toBeTruthy();
    });
    // Assert the panel shows the alpha label — scope the query to the panel
    // so we don't trip on the footer "selecionado: alpha" badge.
    const panel = screen.getByTestId('selected-node-panel');
    expect(panel.textContent).toContain('alpha');
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});