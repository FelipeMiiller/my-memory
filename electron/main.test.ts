import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

/**
 * Drains the microtask queue twice so the `.then()` callback registered by
 * `app.whenReady().then(() => createMainWindow())` in main.ts has a chance
 * to run before we assert against the mocks. Without this drain the
 * `Promise.resolve()` settles, the `.then` callback is queued, but the
 * synchronous `expect()` lines run first — leaving BrowserWindow un-built.
 */
const drainMicrotasks = async (): Promise<void> => {
  await Promise.resolve();
  await Promise.resolve();
  await new Promise<void>((resolve) => setTimeout(resolve, 0));
};

// Mock electron module BEFORE importing main.ts
const mockOnHeadersReceived = vi.fn();
const mockSetWindowOpenHandler = vi.fn();
const mockLoadURL = vi.fn();
const mockLoadFile = vi.fn();
const mockOpenDevTools = vi.fn();
const mockOnce = vi.fn();
const mockWebContents = {
  session: { webRequest: { onHeadersReceived: mockOnHeadersReceived } },
  setWindowOpenHandler: mockSetWindowOpenHandler,
  openDevTools: mockOpenDevTools,
};

const mockBrowserWindowCtor = vi.fn().mockImplementation(() => ({
  loadURL: mockLoadURL,
  loadFile: mockLoadFile,
  webContents: mockWebContents,
  once: mockOnce,
}));

const mockApp = {
  isPackaged: false,
  whenReady: vi.fn(() => Promise.resolve()),
  on: vi.fn(),
  quit: vi.fn(),
};

const mockShell = { openExternal: vi.fn() };

vi.mock('electron', () => ({
  app: mockApp,
  BrowserWindow: mockBrowserWindowCtor,
  shell: mockShell,
}));

describe('Electron main process', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
    mockApp.isPackaged = false;
    mockApp.whenReady.mockImplementation(() => Promise.resolve());
    process.env.NODE_ENV = 'development';
  });

  afterEach(() => {
    delete process.env.NODE_ENV;
  });

  it('exposes contextIsolation + sandbox + nodeIntegration:false in webPreferences', async () => {
    await import('./main');
    await drainMicrotasks();
    expect(mockBrowserWindowCtor).toHaveBeenCalledTimes(1);
    const opts = mockBrowserWindowCtor.mock.calls[0][0];
    expect(opts.webPreferences.contextIsolation).toBe(true);
    expect(opts.webPreferences.sandbox).toBe(true);
    expect(opts.webPreferences.nodeIntegration).toBe(false);
    expect(opts.webPreferences.preload).toContain('preload');
  });

  it('loads http://localhost:5173 in dev mode', async () => {
    await import('./main');
    await drainMicrotasks();
    expect(mockLoadURL).toHaveBeenCalledWith('http://localhost:5173');
  });

  it('loads file:// viewer/dist/index.html in production mode', async () => {
    mockApp.isPackaged = true;
    process.env.NODE_ENV = 'production';
    await import('./main');
    await drainMicrotasks();
    expect(mockLoadFile).toHaveBeenCalled();
    const filePath = mockLoadFile.mock.calls[0][0];
    expect(filePath).toContain('viewer');
    expect(filePath).toContain('index.html');
  });

  it('registers CSP header via webRequest.onHeadersReceived', async () => {
    await import('./main');
    await drainMicrotasks();
    expect(mockOnHeadersReceived).toHaveBeenCalled();
  });

  it('blocks window.open (security hardening)', async () => {
    await import('./main');
    await drainMicrotasks();
    expect(mockSetWindowOpenHandler).toHaveBeenCalled();
  });

  it('quits on window-all-closed (non-darwin)', async () => {
    const origPlatform = process.platform;
    Object.defineProperty(process, 'platform', { value: 'win32', configurable: true });
    await import('./main');
    expect(mockApp.on).toHaveBeenCalledWith('window-all-closed', expect.any(Function));
    Object.defineProperty(process, 'platform', { value: origPlatform, configurable: true });
  });

  it('uses window dimensions 1200x800 minimum', async () => {
    await import('./main');
    await drainMicrotasks();
    const opts = mockBrowserWindowCtor.mock.calls[0][0];
    expect(opts.width).toBe(1200);
    expect(opts.height).toBe(800);
  });
});
