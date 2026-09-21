import '@testing-library/jest-dom/vitest';
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import ptBR from '../viewer/src/i18n/locales/pt-BR.json';
import enUS from '../viewer/src/i18n/locales/en-US.json';

/**
 * Vitest setup — runs once before any test file.
 *
 * 1. Registers `@testing-library/jest-dom` matchers (`toBeInTheDocument`, etc).
 * 2. Initialises i18next with the same locale bundles as the renderer so
 *    `useTranslation()` works inside `Layout`, `GraphView`, `LocaleSwitcher`,
 *    and `App` without each test having to mock react-i18next individually.
 *
 * Tests that need a specific locale call `i18n.changeLanguage('en-US')`
 * inside a `beforeEach` (already done in `viewer/src/views/GraphView.test.tsx`).
 */

void i18n
  .use(initReactI18next)
  .init({
    resources: {
      'pt-BR': { translation: ptBR },
      'en-US': { translation: enUS },
    },
    lng: 'pt-BR',
    fallbackLng: 'en-US',
    supportedLngs: ['pt-BR', 'en-US'],
    interpolation: { escapeValue: false },
    returnNull: false,
  })
  .catch((err: unknown) => {
    // eslint-disable-next-line no-console
    console.warn('[vitest.setup] i18next init failed', err);
  });