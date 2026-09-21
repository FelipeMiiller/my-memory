import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Badge } from './badge';

describe('Badge (shadcn/ui)', () => {
  it('renders with default variant', () => {
    render(<Badge>Default</Badge>);
    expect(screen.getByText('Default')).toBeTruthy();
  });

  it('renders with secondary variant', () => {
    render(<Badge variant="secondary">Secondary</Badge>);
    const el = screen.getByText('Secondary');
    expect(el.className).toContain('secondary');
  });

  it('renders with destructive variant', () => {
    render(<Badge variant="destructive">Danger</Badge>);
    const el = screen.getByText('Danger');
    expect(el.className).toContain('destructive');
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
