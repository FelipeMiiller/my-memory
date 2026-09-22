export const LOCAL_STORAGE_KEYS = {
  LANGUAGE: "mem.locale",
  THEME: "mem.theme",
};

export const IPC_CHANNELS = {
  MEM_DATASET_LOAD: "mem:dataset:load",
  MEM_CLI_RUN: "mem:cli:run",
  MEM_REVEAL: "mem:reveal",
};

export const ENVIRONMENT_VARIABLES = {
  NODE_ENV: process.env.NODE_ENV,
};

export const inDevelopment = ENVIRONMENT_VARIABLES.NODE_ENV === "development";