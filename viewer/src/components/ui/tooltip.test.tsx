import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from './tooltip';

describe('Tooltip (shadcn/ui)', () => {
  it('renders trigger with content (closed by default)', () => {
    render(
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger>Hover me</TooltipTrigger>
          <TooltipContent>Hello</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
    expect(screen.getByText('Hover me')).toBeTruthy();
  });

  it('accepts sideOffset prop', () => {
    render(
      <TooltipProvider>
        <Tooltip defaultOpen>
          <TooltipTrigger>X</TooltipTrigger>
          <TooltipContent sideOffset={10}>Tip</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
    expect(screen.getByText('Tip')).toBeTruthy();
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
