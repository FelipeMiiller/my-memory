import { createStore, del, get, keys, set } from "idb-keyval";
import type { ChatConversation } from "@/lib/chat/types";

/**
 * IndexedDB persistence layer (ADR-051).
 *
 * `idb-keyval` is a tiny wrapper (~600B) over the native IndexedDB API.
 * We use a single object store `mem-chat-v1` keyed by conversation id. A
 * future schema bump can read `mem-chat-v2` and migrate.
 *
 * All functions are async because IndexedDB is async. Failures are logged
 * and surfaced as no-ops; the chat UI degrades to in-memory-only mode.
 */

const STORE_NAME = "mem-chat-v1";

let store: ReturnType<typeof createStore> | null = null;

function db(): ReturnType<typeof createStore> {
  if (!store) {
    store = createStore("mem-chat-db", STORE_NAME);
  }
  return store;
}

export async function listConversations(): Promise<ChatConversation[]> {
  try {
    const allKeys = await keys(db());
    const ids = allKeys
      .map((k) => (typeof k === "string" ? k : null))
      .filter((k): k is string => k !== null && k !== "__config__");
    const conversations: ChatConversation[] = [];
    for (const id of ids) {
      const c = await get<ChatConversation>(id, db());
      if (c) conversations.push(c);
    }
    conversations.sort((a, b) => b.updatedAt - a.updatedAt);
    return conversations;
  } catch (err) {
    console.warn("[chat/persistence] listConversations failed:", err);
    return [];
  }
}

export async function saveConversation(c: ChatConversation): Promise<void> {
  try {
    await set(c.id, c, db());
  } catch (err) {
    console.warn("[chat/persistence] saveConversation failed:", err);
  }
}

export async function deleteConversation(id: string): Promise<void> {
  try {
    await del(id, db());
  } catch (err) {
    console.warn("[chat/persistence] deleteConversation failed:", err);
  }
}

export async function getConversation(
  id: string,
): Promise<ChatConversation | null> {
  try {
    const c = await get<ChatConversation>(id, db());
    return c ?? null;
  } catch (err) {
    console.warn("[chat/persistence] getConversation failed:", err);
    return null;
  }
}

const CONFIG_KEY = "__config__";

export async function loadPersistedConfig(): Promise<unknown> {
  try {
    return await get(CONFIG_KEY, db());
  } catch {
    return null;
  }
}

export async function savePersistedConfig(value: unknown): Promise<void> {
  try {
    await set(CONFIG_KEY, value, db());
  } catch (err) {
    console.warn("[chat/persistence] savePersistedConfig failed:", err);
  }
}
