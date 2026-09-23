import "@testing-library/jest-dom/vitest";
import "../src/localization/i18n";

/**
 * Vitest setup — runs once before any test file.
 *
 * 1. Registers `@testing-library/jest-dom` matchers (`toBeInTheDocument`, etc).
 * 2. Side-effect imports the renderer's `i18n` bootstrapper, which already
 *    wires up all namespaces (`common`, `topbar`, `graph`, `workspace`,
 *    `chat`) for `pt-BR` and `en-US`. Importing the bootstrapper (instead of
 *    duplicating the bundle wiring here) keeps tests in sync with the
 *    renderer whenever a new locale or namespace lands.
 */
