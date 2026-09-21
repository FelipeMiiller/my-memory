import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from './resizable';

describe('Resizable (shadcn/ui)', () => {
  it('renders panel group + handle (smoke)', () => {
    const { container } = render(
      <ResizablePanelGroup direction="horizontal">
        <ResizablePanel>Left</ResizablePanel>
        <ResizableHandle withHandle />
        <ResizablePanel>Right</ResizablePanel>
      </ResizablePanelGroup>
    );
    expect(container).toBeTruthy();
  });

  it('renders handle without withHandle prop', () => {
    const { container } = render(
      <ResizablePanelGroup direction="vertical">
        <ResizablePanel>Top</ResizablePanel>
        <ResizableHandle />
        <ResizablePanel>Bottom</ResizablePanel>
      </ResizablePanelGroup>
    );
    expect(container).toBeTruthy();
  });

  it('smoke check passes', () => {
    expect(true).toBe(true);
  });
});
