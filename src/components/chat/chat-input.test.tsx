/**
 * ChatInput — vitest unit test.
 *
 * Closes T11 from M1 retro (`feat-viewer-v3-m1-3-m1-3-pane-layout/tasks.md`)
 * for the input side: pressing Enter on a non-empty textarea calls `onSend`
 * with the trimmed text and clears the textarea afterwards.
 *
 * `MicButton` (ASR) is mocked because getUserMedia isn't available in jsdom.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { ChatInput } from "./chat-input";

// ASR scaffold not testable in jsdom — getUserMedia + ScriptProcessor aren't
// available. Replace the whole MicButton with a passive stub.
vi.mock("./mic-button", () => ({
  MicButton: (): React.JSX.Element => (
    <button type="button" aria-label="stub-mic" disabled />
  ),
}));

describe("ChatInput", () => {
  test("Enter on non-empty textarea calls onSend with trimmed text and clears input", () => {
    const onSend = vi.fn();
    const onStop = vi.fn();

    render(
      <ChatInput streaming={false} hasApiKey onSend={onSend} onStop={onStop} />,
    );

    const textarea = screen.getByTestId(
      "chat-input-textarea",
    ) as HTMLTextAreaElement;

    // Type a value with surrounding whitespace.
    fireEvent.change(textarea, { target: { value: "  oi memória  " } });
    expect(textarea.value).toBe("  oi memória  ");

    // Press Enter (no shift) to submit.
    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false });

    expect(onSend).toHaveBeenCalledTimes(1);
    expect(onSend).toHaveBeenCalledWith("oi memória");

    // After submit, textarea is cleared so the user can type the next prompt.
    expect(textarea.value).toBe("");
  });
});
