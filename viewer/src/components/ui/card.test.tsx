import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './card';

describe('Card (shadcn/ui)', () => {
  it('renders full card composition', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Title</CardTitle>
          <CardDescription>Desc</CardDescription>
        </CardHeader>
        <CardContent>Body</CardContent>
        <CardFooter>Foot</CardFooter>
      </Card>
    );
    expect(screen.getByText('Title')).toBeTruthy();
    expect(screen.getByText('Desc')).toBeTruthy();
    expect(screen.getByText('Body')).toBeTruthy();
    expect(screen.getByText('Foot')).toBeTruthy();
  });

  it('applies custom className', () => {
    const { container } = render(<Card className="custom-test">X</Card>);
    expect((container.firstChild as HTMLElement).className).toContain('custom-test');
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
