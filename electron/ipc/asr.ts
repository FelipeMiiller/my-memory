import { randomUUID } from "node:crypto";
import { IPC_CHANNELS } from "@electron/constants";
import { type BrowserWindow, ipcMain } from "electron";

/**
 * ASR IPC handlers (ADR-045 / spec nemotron-asr-streaming).
 *
 * This module is the scaffold for the renderer → main → Go (`internal/asr/*`)
 * audio pipeline. While the Nemotron 3.5 ONNX client isn't wired in
 * (`mem asr download` not yet run, or the Go side is still Pending per
 * spec T1–T9), every chunk is acknowledged with a synthetic partial so the
 * renderer can verify the pipeline is connected.
 *
 * Wiring target:
 *   start  → spawns Nemotron session (Go subprocess or in-process)
 *   chunk  → forwards Int16 PCM to Go via stdin pipe / RPC
 *   stop   → drains remaining audio, emits `mem:asr:final`
 *
 * Replace the synthetic emission with the real Go client once T4-T7 of the
 * spec land.
 */

interface SessionState {
  sessionId: string;
  locale: string;
  startedAt: number;
  chunkCount: number;
  window: BrowserWindow;
}

const sessions = new Map<string, SessionState>();

function emit(session: SessionState, channel: string, payload: unknown): void {
  if (session.window.isDestroyed()) return;
  session.window.webContents.send(channel, payload);
}

function validateStart(req: unknown): AsrStartRequest | { error: string } {
  if (req === undefined || req === null) return {};
  if (typeof req !== "object") return { error: "Payload must be an object." };
  const r = req as Partial<AsrStartRequest>;
  if (r.locale !== undefined && typeof r.locale !== "string") {
    return { error: "locale must be a string." };
  }
  if (
    r.conversationTurnId !== undefined &&
    typeof r.conversationTurnId !== "string"
  ) {
    return { error: "conversationTurnId must be a string." };
  }
  return r as AsrStartRequest;
}

function validateChunk(req: unknown): AsrChunkRequest | { error: string } {
  if (typeof req !== "object" || req === null) {
    return { error: "Payload must be an object." };
  }
  const r = req as Partial<AsrChunkRequest>;
  if (typeof r.sessionId !== "string" || r.sessionId.length === 0) {
    return { error: "sessionId missing." };
  }
  if (!(r.pcm instanceof Object) && !Array.isArray(r.pcm)) {
    return { error: "pcm must be Int16Array." };
  }
  if (typeof r.sampleRate !== "number" || r.sampleRate !== 16000) {
    return { error: "sampleRate must be 16000." };
  }
  return r as AsrChunkRequest;
}

function validateStop(req: unknown): AsrStopRequest | { error: string } {
  if (typeof req !== "object" || req === null) {
    return { error: "Payload must be an object." };
  }
  const r = req as Partial<AsrStopRequest>;
  if (typeof r.sessionId !== "string" || r.sessionId.length === 0) {
    return { error: "sessionId missing." };
  }
  return r as AsrStopRequest;
}

export function registerAsrHandlers(
  getWindow: () => BrowserWindow | null,
): void {
  ipcMain.handle(
    IPC_CHANNELS.MEM_ASR_START,
    async (_event, payload: unknown): Promise<AsrStartResponse> => {
      const window = getWindow();
      if (!window || window.isDestroyed()) {
        throw { kind: "unknown", message: "No active window." };
      }
      const validated = validateStart(payload);
      if ("error" in validated) {
        throw { kind: "invalid", message: validated.error };
      }
      const sessionId = randomUUID();
      sessions.set(sessionId, {
        sessionId,
        locale: validated.locale ?? "auto",
        startedAt: Date.now(),
        chunkCount: 0,
        window,
      });
      // Notify renderer that the session is live (lets it toggle its UI).
      // Real session-ready events come from the Go side once the model is
      // warm-loaded — see spec T7 / T9.
      return { sessionId };
    },
  );

  ipcMain.handle(
    IPC_CHANNELS.MEM_ASR_CHUNK,
    async (_event, payload: unknown): Promise<{ accepted: boolean }> => {
      const validated = validateChunk(payload);
      if ("error" in validated) {
        throw { kind: "invalid", message: validated.error };
      }
      const session = sessions.get(validated.sessionId);
      if (!session) {
        throw {
          kind: "aborted",
          sessionId: validated.sessionId,
          message: "Unknown session — call mem:asr:start first.",
        };
      }
      session.chunkCount += 1;

      // Synthetic partial echo while the real Nemotron client is Pending.
      // Replace this block with: forward chunk to Go Nemotron client.
      if (session.chunkCount % 5 === 0) {
        emit(session, IPC_CHANNELS.MEM_ASR_PARTIAL, {
          sessionId: session.sessionId,
          transcript: `[stub] captado ${session.chunkCount} chunks · modelo Nemotron ainda não conectado`,
          confidence: 0,
        } satisfies AsrPartialEvent);
      }
      return { accepted: true };
    },
  );

  ipcMain.handle(
    IPC_CHANNELS.MEM_ASR_STOP,
    async (_event, payload: unknown): Promise<AsrStopResponse> => {
      const validated = validateStop(payload);
      if ("error" in validated) {
        throw { kind: "invalid", message: validated.error };
      }
      const session = sessions.get(validated.sessionId);
      if (!session) {
        return { finalized: false };
      }
      const durationMs = Date.now() - session.startedAt;
      // Synthetic final echo. Replace with real Go Nemotron final.
      emit(session, IPC_CHANNELS.MEM_ASR_FINAL, {
        sessionId: session.sessionId,
        transcript: `[stub] sessão encerrada — ${session.chunkCount} chunks, ${durationMs}ms`,
        durationMs,
        locale: session.locale,
      } satisfies AsrFinalEvent);
      sessions.delete(validated.sessionId);
      return { finalized: true };
    },
  );
}
