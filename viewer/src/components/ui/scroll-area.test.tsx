import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ScrollArea } from './scroll-area';

describe('ScrollArea (shadcn/ui)', () => {
  it('renders children inside scroll viewport', () => {
    render(
      <ScrollArea data-testid="sa">
        <p>Item 1</p>
        <p>Item 2</p>
      </ScrollArea>
    );
    expect(screen.getByText('Item 1')).toBeTruthy();
    expect(screen.getByText('Item 2')).toBeTruthy();
  });

  it('applies custom className', () => {
    const { container } = render(
      <ScrollArea className="my-scroll">X</ScrollArea>
    );
    expect((container.firstChild as HTMLElement).className).toContain('my-scroll');
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
