/**
 * IPC manager — placeholder.
 *
 * Phase 1 uses direct contextBridge.exposeInMainWorld('memAPI', {...}) in
 * preload.ts. Phase 2 ADR-049 may introduce typed channels via @orpc/server
 * + MessagePort (template default) or stay with simple invoke/handle pairs.
 */
export const ipc = {
  initialize(): void {
    /* no-op Phase 1 */
  },
};
