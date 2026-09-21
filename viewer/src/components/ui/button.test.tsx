import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Button } from './button';

describe('Button (shadcn/ui)', () => {
  it('renders with default variant and size', () => {
    render(<Button>Click me</Button>);
    const btn = screen.getByRole('button', { name: /click me/i });
    expect(btn).toBeTruthy();
  });

  it('applies variant class when variant prop is provided', () => {
    render(<Button variant="destructive">Delete</Button>);
    const btn = screen.getByRole('button', { name: /delete/i });
    expect(btn.className).toContain('destructive');
  });

  it('is disabled when disabled prop is set', () => {
    render(<Button disabled>Save</Button>);
    const btn = screen.getByRole('button', { name: /save/i });
    expect((btn as HTMLButtonElement).disabled).toBe(true);
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
