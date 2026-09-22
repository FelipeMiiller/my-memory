export const LOCAL_STORAGE_KEYS = {
  LANGUAGE: "mem.locale",
  THEME: "mem.theme",
};

export const IPC_CHANNELS = {
  MEM_DATASET_LOAD: "mem:dataset:load",
  MEM_CLI_RUN: "mem:cli:run",
  MEM_REVEAL: "mem:reveal",
  // Chat (ADR-051) — request/response via invoke, streaming via events
  MEM_CHAT_SEND: "mem:chat:send",
  MEM_CHAT_ABORT: "mem:chat:abort",
  MEM_CHAT_CONFIG_GET: "mem:chat:config:get",
  MEM_CHAT_CONFIG_SET: "mem:chat:config:set",
  MEM_CHAT_MEM_SEARCH: "mem:chat:mem-search",
  // Events (one-way, main → renderer)
  MEM_CHAT_DELTA: "mem:chat:delta",
  MEM_CHAT_DONE: "mem:chat:done",
  MEM_CHAT_ERROR: "mem:chat:error",
  // Window controls (custom title-bar buttons; titleBarStyle: hidden)
  MEM_WINDOW_MINIMIZE: "mem:window:minimize",
  MEM_WINDOW_MAXIMIZE_TOGGLE: "mem:window:maximize-toggle",
  MEM_WINDOW_CLOSE: "mem:window:close",
  MEM_WINDOW_IS_MAXIMIZED: "mem:window:is-maximized",
};

export const ENVIRONMENT_VARIABLES = {
  NODE_ENV: process.env.NODE_ENV,
};

export const inDevelopment = ENVIRONMENT_VARIABLES.NODE_ENV === "development";