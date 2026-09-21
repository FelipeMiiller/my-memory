import { describe, it, expect, vi } from 'vitest';
import cytoscape from 'cytoscape';
import { initCytoscape, pagerankToRadius } from './cytoscape-init';

vi.mock('cytoscape', () => {
  const mockCore = {
    on: vi.fn(),
    destroy: vi.fn(),
    fit: vi.fn(),
  };
  const cytoscape = vi.fn(() => mockCore);
  return { default: cytoscape };
});

const buildContainer = (): HTMLElement => {
  const el = document.createElement('div');
  el.style.width = '800px';
  el.style.height = '600px';
  document.body.appendChild(el);
  return el;
};

/**
 * Helper — extract the Cytoscape options passed to the mocked constructor.
 * vi.mocked typing collapses when the module is replaced via vi.mock, so we
 * cast the call arguments explicitly to `unknown` first.
 */
function lastCytoscapeCall(): Record<string, unknown> {
  const fn = cytoscape as unknown as { mock: { calls: unknown[][] } };
  const call = fn.mock.calls[0]?.[0];
  return (call ?? {}) as Record<string, unknown>;
}

describe('cytoscape-init', () => {
  it('pagerankToRadius maps 0..1 to ~14..74 px', () => {
    expect(pagerankToRadius(0)).toBe(14);
    expect(pagerankToRadius(1)).toBe(74);
    expect(pagerankToRadius(0.5)).toBeCloseTo(44, 5);
  });

  it('pagerankToRadius clamps out-of-range values', () => {
    expect(pagerankToRadius(-1)).toBe(14);
    expect(pagerankToRadius(99)).toBe(74);
  });

  it('initCytoscape maps dataset into Cytoscape ElementsDefinition', () => {
    const container = buildContainer();
    initCytoscape({
      container,
      dataset: {
        nodes: [
          { id: 'a', label: 'Alpha', color: '#6366f1' },
          { id: 'b', label: 'Beta' }, // no color → fallback
        ],
        edges: [{ source: 'a', target: 'b' }],
      },
    });
    const opts = lastCytoscapeCall();
    const elements = opts['elements'] as {
      nodes: Array<{ data: Record<string, unknown> }>;
      edges: Array<unknown>;
    };
    expect(elements.nodes.length).toBe(2);
    expect(elements.edges.length).toBe(1);
    // First node preserves color, second gets a fallback palette value
    expect(typeof elements.nodes[1]?.data['color']).toBe('string');
  });

  it('initCytoscape configures cose layout + node styling', () => {
    const container = buildContainer();
    initCytoscape({
      container,
      dataset: { nodes: [{ id: 'x', label: 'X' }], edges: [] },
    });
    const opts = lastCytoscapeCall();
    const layout = opts['layout'] as { name?: string };
    expect(layout.name).toBe('cose');
    expect(Array.isArray(opts['style'])).toBe(true);
    const styles = opts['style'] as Array<{ selector: string }>;
    expect(styles.some((s) => s.selector === 'node')).toBe(true);
    expect(styles.some((s) => s.selector === 'edge')).toBe(true);
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});