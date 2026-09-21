/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly DEV: boolean;
  readonly PROD: boolean;
  readonly MODE: string;
  readonly BASE_URL: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

// Phase 2: extend with `memAPI` types from electron/preload.ts
declare global {
  interface Window {
    memAPI?: {
      platform: NodeJS.Platform;
      loadDataset: () => Promise<unknown>;
      runCli: (args: string[]) => Promise<unknown>;
    };
  }
}

export {};
