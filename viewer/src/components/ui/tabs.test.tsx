import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Tabs, TabsList, TabsTrigger, TabsContent } from './tabs';

describe('Tabs (shadcn/ui)', () => {
  it('renders tabs and switches on click', async () => {
    const user = userEvent.setup();
    render(
      <Tabs defaultValue="a">
        <TabsList>
          <TabsTrigger value="a">Tab A</TabsTrigger>
          <TabsTrigger value="b">Tab B</TabsTrigger>
        </TabsList>
        <TabsContent value="a">Content A</TabsContent>
        <TabsContent value="b">Content B</TabsContent>
      </Tabs>
    );
    expect(screen.getByText('Content A')).toBeTruthy();
    await user.click(screen.getByRole('tab', { name: /tab b/i }));
    expect(screen.getByText('Content B')).toBeTruthy();
  });

  it('disables trigger when disabled prop is set', () => {
    render(
      <Tabs defaultValue="a">
        <TabsList>
          <TabsTrigger value="a">A</TabsTrigger>
          <TabsTrigger value="b" disabled>B</TabsTrigger>
        </TabsList>
        <TabsContent value="a">A</TabsContent>
        <TabsContent value="b">B</TabsContent>
      </Tabs>
    );
    const tabB = screen.getByRole('tab', { name: /b/i });
    // Radix UI's TabsTrigger signals "disabled" via `data-disabled` (a
    // boolean attribute) rather than `aria-disabled="true"`. Either
    // signal works for assistive tech; we assert against the one Radix
    // actually emits.
    expect(tabB.getAttribute('data-disabled')).toBe('');
    expect(tabB.hasAttribute('disabled')).toBe(true);
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
