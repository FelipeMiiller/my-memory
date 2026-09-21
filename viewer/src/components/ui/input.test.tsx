import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Input } from './input';

describe('Input (shadcn/ui)', () => {
  it('renders with placeholder', () => {
    render(<Input placeholder="Type here" />);
    expect(screen.getByPlaceholderText('Type here')).toBeTruthy();
  });

  it('captures user typing', async () => {
    const user = userEvent.setup();
    render(<Input aria-label="Search" />);
    const input = screen.getByLabelText('Search') as HTMLInputElement;
    await user.type(input, 'turboquant');
    expect(input.value).toBe('turboquant');
  });

  it('is disabled when disabled prop set', () => {
    render(<Input disabled aria-label="X" />);
    const input = screen.getByLabelText('X') as HTMLInputElement;
    expect(input.disabled).toBe(true);
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
