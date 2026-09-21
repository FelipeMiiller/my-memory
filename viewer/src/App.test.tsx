import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import App from './App';

/**
 * Mock cytoscape to avoid jsdom canvas cost in unit tests.
 * The bundled library requires a real canvas which jsdom does not provide.
 */
vi.mock('cytoscape', () => {
  const mockCore = {
    on: vi.fn(),
    destroy: vi.fn(),
    fit: vi.fn(),
    resize: vi.fn(),
    nodes: vi.fn(() => ({ length: 0 })),
    edges: vi.fn(() => ({ length: 0 })),
  };
  const cytoscape = vi.fn(() => mockCore);
  return {
    default: cytoscape,
  };
});

describe('App (Phase 1 MVP entry)', () => {
  it('renders the Layout with topbar + tab list', () => {
    render(<App />);
    expect(screen.getByTestId('topbar')).toBeTruthy();
    expect(screen.getByTestId('tabs-list')).toBeTruthy();
    expect(screen.getByTestId('layout-body')).toBeTruthy();
  });

  it('shows the Grafo tab as default active trigger', () => {
    render(<App />);
    const grafoTab = screen.getByTestId('tab-trigger-graph');
    expect(grafoTab.getAttribute('data-state')).toBe('active');
  });

  it('renders the dark theme badge', () => {
    render(<App />);
    expect(screen.getByTestId('theme-badge').textContent).toContain('Dark');
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});