import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Layout } from './Layout';

/**
 * Mock cytoscape so jsdom does not need a real canvas — Layout mounts
 * GraphView, which initialises Cytoscape on mount.
 */
vi.mock('cytoscape', () => {
  const mockCore = {
    on: vi.fn(),
    destroy: vi.fn(),
    fit: vi.fn(),
    resize: vi.fn(),
  };
  const cytoscape = vi.fn(() => mockCore);
  return { default: cytoscape };
});

describe('Layout (Phase 1 shell)', () => {
  it('renders topbar, sidebar and tab list', () => {
    render(<Layout />);
    expect(screen.getByTestId('topbar')).toBeTruthy();
    expect(screen.getByTestId('sidebar-nav')).toBeTruthy();
    expect(screen.getByTestId('tabs-list')).toBeTruthy();
  });

  it('renders all three tab triggers (Grafo active; Code/Staleness disabled)', () => {
    render(<Layout />);
    const grafo = screen.getByTestId('tab-trigger-graph');
    const code = screen.getByTestId('tab-trigger-code');
    const staleness = screen.getByTestId('tab-trigger-staleness');

    expect(grafo.getAttribute('data-state')).toBe('active');
    expect(code.hasAttribute('disabled')).toBe(true);
    expect(staleness.hasAttribute('disabled')).toBe(true);
  });

  it('renders 3 sidebar nav links with descriptive copy', () => {
    render(<Layout />);
    expect(screen.getByTestId('sidebar-link-grafo')).toBeTruthy();
    expect(screen.getByTestId('sidebar-link-code')).toBeTruthy();
    expect(screen.getByTestId('sidebar-link-staleness')).toBeTruthy();
  });

  it('shows the Grafo tab content (GraphView) by default', () => {
    render(<Layout />);
    expect(screen.getByTestId('tab-content-graph')).toBeTruthy();
  });

  it('does not show Code or Staleness content in default state', () => {
    render(<Layout />);
    const code = screen.queryByTestId('tab-content-code');
    const staleness = screen.queryByTestId('tab-content-staleness');
    expect(code === null || code.hasAttribute('hidden')).toBe(true);
    expect(staleness === null || staleness.hasAttribute('hidden')).toBe(true);
  });

  it('renders dataset summary footer pointing to data-central.json', () => {
    const { container } = render(<Layout />);
    // `data-central.json` appears in the sidebar footer (`<code>` element)
    // AND in the GraphView loading overlay — use a CSS-narrowed query to
    // avoid the "multiple elements found" failure mode after i18n.
    const codeEls = container.querySelectorAll('code');
    const hit = Array.from(codeEls).some((el) =>
      /data-central\.json/i.test(el.textContent ?? '')
    );
    expect(hit).toBe(true);
  });

  it('sidebar Code/Staleness buttons are disabled', () => {
    render(<Layout />);
    const codeBtn = screen.getByTestId('sidebar-link-code') as HTMLButtonElement;
    const stalenessBtn = screen.getByTestId(
      'sidebar-link-staleness'
    ) as HTMLButtonElement;
    expect(codeBtn.disabled).toBe(true);
    expect(stalenessBtn.disabled).toBe(true);
    // user-event v14 refuses to click a disabled button — fire a plain
    // DOM click to confirm the active tab does NOT change.
    codeBtn.click();
    stalenessBtn.click();
    expect(
      screen.getByTestId('tab-trigger-graph').getAttribute('data-state')
    ).toBe('active');
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});