import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockExposeInMainWorld = vi.fn();
const mockInvoke = vi.fn();

vi.mock('electron', () => ({
  contextBridge: {
    exposeInMainWorld: mockExposeInMainWorld,
  },
  ipcRenderer: {
    invoke: mockInvoke,
  },
}));

describe('Electron preload contextBridge', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Force fresh module load per test
    vi.resetModules();
  });

  it('exposes memAPI on the main world', async () => {
    await import('./preload');
    expect(mockExposeInMainWorld).toHaveBeenCalledWith('memAPI', expect.any(Object));
  });

  it('exposes platform from process.platform', async () => {
    const origPlatform = process.platform;
    Object.defineProperty(process, 'platform', { value: 'linux', configurable: true });
    await import('./preload');
    const exposed = mockExposeInMainWorld.mock.calls[0][1];
    expect(exposed.platform).toBe('linux');
    Object.defineProperty(process, 'platform', { value: origPlatform, configurable: true });
  });

  it('loadDataset invokes ipcRenderer.invoke("mem:dataset:load")', async () => {
    await import('./preload');
    const exposed = mockExposeInMainWorld.mock.calls[0][1];
    mockInvoke.mockResolvedValueOnce({ nodes: [], edges: [] });
    await exposed.loadDataset();
    expect(mockInvoke).toHaveBeenCalledWith('mem:dataset:load');
  });

  it('runCli forwards args to ipcRenderer.invoke("mem:cli:run", args)', async () => {
    await import('./preload');
    const exposed = mockExposeInMainWorld.mock.calls[0][1];
    mockInvoke.mockResolvedValueOnce({ stdout: '', stderr: '', exitCode: 0 });
    const args = ['search', 'turboquant'];
    await exposed.runCli(args);
    expect(mockInvoke).toHaveBeenCalledWith('mem:cli:run', args);
  });

  it('loadDataset rejects with Phase 2 not-implemented error', async () => {
    mockInvoke.mockRejectedValueOnce(new Error('no handler'));
    await import('./preload');
    const exposed = mockExposeInMainWorld.mock.calls[0][1];
    await expect(exposed.loadDataset()).rejects.toThrow(/Phase 2/);
  });

  it('runCli rejects with Phase 2 not-implemented error', async () => {
    mockInvoke.mockRejectedValueOnce(new Error('no handler'));
    await import('./preload');
    const exposed = mockExposeInMainWorld.mock.calls[0][1];
    await expect(exposed.runCli(['x'])).rejects.toThrow(/Phase 2/);
  });

  it('does NOT expose ipcRenderer directly to renderer (security)', async () => {
    await import('./preload');
    const exposed = mockExposeInMainWorld.mock.calls[0][1];
    expect(exposed.ipcRenderer).toBeUndefined();
    expect(exposed.contextBridge).toBeUndefined();
  });

  it('does NOT expose Node primitives (process, require, Buffer)', async () => {
    await import('./preload');
    const exposed = mockExposeInMainWorld.mock.calls[0][1];
    expect(exposed.process).toBeUndefined();
    expect(exposed.require).toBeUndefined();
    expect(exposed.Buffer).toBeUndefined();
  });
});
