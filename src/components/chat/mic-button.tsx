import { Mic, MicOff } from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { asrApi } from "@/lib/chat/ipc";
import { cn } from "@/utils/tailwind";

/**
 * MicButton — chat input toolbar mic toggle (ADR-045 scaffold).
 *
 * Audio pipeline (renderer side):
 *   1. getUserMedia({audio: {sampleRate: 16000, channelCount: 1}})
 *   2. AudioContext at 16 kHz + ScriptProcessorNode (bufferSize 2048)
 *   3. onaudioprocess converts Float32 → Int16 PCM, batches 160 ms chunks
 *   4. Each chunk → `mem:asr:chunk` via IPC; main acknowledges & echoes
 *      synthetic partials until the real Nemotron client (Go backend,
 *      `internal/asr/nemotron_onnx.go`) lands per spec nemotron-asr-streaming.
 *
 * Transcripts emitted by main are forwarded up via `onTranscript` so the
 * parent (ChatInput) can append/replace text in the textarea without
 * coupling this component to the chat store.
 */

const SAMPLE_RATE = 16_000;
const CHUNK_MS = 160;
const SAMPLES_PER_CHUNK = (SAMPLE_RATE * CHUNK_MS) / 1000; // 2560
const BUFFER_SIZE = 2048; // ScriptProcessorNode power-of-two

export type AsrTranscriptMode = "partial" | "final";

export interface MicButtonProps {
  /** Append partial transcripts live; replace prior partial on each call. */
  onTranscript: (text: string, mode: AsrTranscriptMode) => void;
  /** Fired when capture aborts (mic permission denied, error, stop). */
  onError?: (message: string) => void;
  /** Disabled while chat streaming or no API key. */
  disabled?: boolean;
  /** Locale hint forwarded to the ASR start request. */
  locale?: string;
}

export function MicButton({
  onTranscript,
  onError,
  disabled = false,
  locale,
}: MicButtonProps): React.JSX.Element {
  const { t } = useTranslation();
  const [dictating, setDictating] = React.useState(false);
  const [unsupported, setUnsupported] = React.useState(false);
  const [elapsedMs, setElapsedMs] = React.useState(0);

  // Refs to live resources so cleanup can reach them on unmount.
  const sessionIdRef = React.useRef<string | null>(null);
  const streamRef = React.useRef<MediaStream | null>(null);
  const audioCtxRef = React.useRef<AudioContext | null>(null);
  const processorRef = React.useRef<ScriptProcessorNode | null>(null);
  const bufferRef = React.useRef<Int16Array>(new Int16Array(SAMPLES_PER_CHUNK));
  const bufferIdxRef = React.useRef(0);
  const startedAtRef = React.useRef<number>(0);
  const elapsedTimerRef = React.useRef<number | null>(null);
  const lastPartialRef = React.useRef<string>("");

  React.useEffect(() => {
    // Feature-detect getUserMedia + AudioContext at mount time.
    if (
      typeof navigator === "undefined" ||
      !navigator.mediaDevices?.getUserMedia ||
      typeof window === "undefined" ||
      typeof window.AudioContext === "undefined"
    ) {
      setUnsupported(true);
    }
  }, []);

  React.useEffect(() => {
    if (!dictating) return;
    const id = window.setInterval(() => {
      setElapsedMs(Date.now() - startedAtRef.current);
    }, 100);
    elapsedTimerRef.current = id;
    return () => {
      window.clearInterval(id);
    };
  }, [dictating]);

  React.useEffect(() => {
    return () => {
      void stopCapture();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Subscribe to ASR events while dictating.
  React.useEffect(() => {
    if (!dictating) return;
    const unsubPartial = asrApi.onPartial((e) => {
      if (e.sessionId !== sessionIdRef.current) return;
      lastPartialRef.current = e.transcript;
      onTranscript(e.transcript, "partial");
    });
    const unsubFinal = asrApi.onFinal((e) => {
      if (e.sessionId !== sessionIdRef.current) return;
      // Append the final transcript (not replace) — partials already live
      // in the textarea; final adds the committed tail.
      onTranscript(e.transcript, "final");
    });
    const unsubError = asrApi.onError((e) => {
      if (e.sessionId !== sessionIdRef.current) return;
      onError?.(e.message);
      void stopCapture();
    });
    return () => {
      unsubPartial();
      unsubFinal();
      unsubError();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dictating]);

  function floatToInt16(input: Float32Array): Int16Array {
    const out = new Int16Array(input.length);
    for (let i = 0; i < input.length; i++) {
      const s = Math.max(-1, Math.min(1, input[i] ?? 0));
      out[i] = s < 0 ? Math.round(s * 0x8000) : Math.round(s * 0x7fff);
    }
    return out;
  }

  async function startCapture(): Promise<void> {
    if (disabled || unsupported || dictating) return;
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          channelCount: 1,
          sampleRate: SAMPLE_RATE,
          echoCancellation: true,
          noiseSuppression: true,
        },
      });
      const audioCtx = new window.AudioContext({ sampleRate: SAMPLE_RATE });
      // Some Chromium builds ignore sampleRate in the constructor and fall
      // back to the device rate. Resample defensively via the buffer ratio.
      const effectiveRate = audioCtx.sampleRate;
      const processor = audioCtx.createScriptProcessor(BUFFER_SIZE, 1, 1);
      const source = audioCtx.createMediaStreamSource(stream);

      const startRes = await asrApi.start({ locale });
      sessionIdRef.current = startRes.sessionId;
      streamRef.current = stream;
      audioCtxRef.current = audioCtx;
      processorRef.current = processor;
      bufferRef.current = new Int16Array(SAMPLES_PER_CHUNK);
      bufferIdxRef.current = 0;
      startedAtRef.current = Date.now();
      lastPartialRef.current = "";
      setElapsedMs(0);

      processor.onaudioprocess = (ev: AudioProcessingEvent): void => {
        const input = ev.inputBuffer.getChannelData(0);
        // Resample to 16 kHz if the AudioContext landed on a different rate.
        const ratio = effectiveRate / SAMPLE_RATE;
        const outCount =
          ratio === 1 ? input.length : Math.floor(input.length / ratio);
        const chunk = new Int16Array(outCount);
        if (ratio === 1) {
          const ints = floatToInt16(input);
          chunk.set(ints);
        } else {
          for (let i = 0; i < outCount; i++) {
            const idx = Math.floor(i * ratio);
            const s = Math.max(-1, Math.min(1, input[idx] ?? 0));
            chunk[i] = s < 0 ? Math.round(s * 0x8000) : Math.round(s * 0x7fff);
          }
        }

        // Batch into 160 ms windows before IPC to match the model's chunk
        // size (ADR-045 §Configuration — default 160 ms).
        const session = sessionIdRef.current;
        if (!session) return;
        const buf = bufferRef.current;
        let bufIdx = bufferIdxRef.current;
        for (let i = 0; i < chunk.length; i++) {
          buf[bufIdx++] = chunk[i] ?? 0;
          if (bufIdx >= SAMPLES_PER_CHUNK) {
            // Copy before shipping — the buffer is reused.
            const ship = new Int16Array(buf);
            void asrApi
              .chunk({
                sessionId: session,
                pcm: ship,
                sampleRate: SAMPLE_RATE,
              })
              .catch((err: unknown) => {
                console.warn("[mic] chunk rejected:", err);
              });
            bufIdx = 0;
          }
        }
        bufferIdxRef.current = bufIdx;
      };

      source.connect(processor);
      processor.connect(audioCtx.destination); // required for some Chromium builds

      setDictating(true);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      onError?.(msg);
      await stopCapture();
    }
  }

  async function stopCapture(): Promise<void> {
    const session = sessionIdRef.current;
    if (processorRef.current) {
      processorRef.current.disconnect();
      processorRef.current.onaudioprocess = null;
      processorRef.current = null;
    }
    if (audioCtxRef.current) {
      try {
        await audioCtxRef.current.close();
      } catch {
        // AudioContext.close() throws if already closed — ignore.
      }
      audioCtxRef.current = null;
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => {
        t.stop();
      });
      streamRef.current = null;
    }
    if (session) {
      try {
        await asrApi.stop({ sessionId: session });
      } catch {
        // best-effort
      }
    }
    sessionIdRef.current = null;
    bufferIdxRef.current = 0;
    setDictating(false);
    setElapsedMs(0);
    lastPartialRef.current = "";
  }

  function handleClick(): void {
    if (dictating) {
      void stopCapture();
    } else {
      void startCapture();
    }
  }

  const title = unsupported
    ? t("chat.input.dictateUnsupported")
    : dictating
      ? t("chat.input.stopDictation", { seconds: Math.floor(elapsedMs / 1000) })
      : t("chat.input.startDictation");

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      onClick={handleClick}
      disabled={disabled || unsupported}
      aria-label={title}
      title={title}
      data-testid="chat-input-mic"
      data-dictating={dictating ? "true" : "false"}
      className={cn(
        "h-8 w-8",
        dictating && "text-red-500 animate-pulse",
        unsupported && "opacity-40",
      )}
    >
      {dictating ? (
        <MicOff className="h-4 w-4" aria-hidden />
      ) : (
        <Mic className="h-4 w-4" aria-hidden />
      )}
    </Button>
  );
}

export default MicButton;
