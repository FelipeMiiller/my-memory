import { FuseV1Options, FuseVersion } from "@electron/fuses";
import { MakerDeb } from "@electron-forge/maker-deb";
import { MakerRpm } from "@electron-forge/maker-rpm";
import { MakerSquirrel } from "@electron-forge/maker-squirrel";
import { MakerZIP } from "@electron-forge/maker-zip";
import { FusesPlugin } from "@electron-forge/plugin-fuses";
import { VitePlugin } from "@electron-forge/plugin-vite";
import type { ForgeConfig } from "@electron-forge/shared-types";

/**
 * Forge config — my-memory viewer.
 *
 * Makers cover all 4 platforms (Windows squirrel + zip, macOS zip, Linux
 * deb + rpm). Phase 1 only builds Windows locally; cross-platform release
 * happens in Phase 2 once we wire `npm run make` + `npm run publish` via
 * GitHub Actions (Electron Forge auto-creates draft releases).
 */
const config: ForgeConfig = {
  makers: [
    new MakerSquirrel({}),
    new MakerZIP({}, ["darwin"]),
    new MakerRpm({}),
    new MakerDeb({}),
  ],
  packagerConfig: {
    asar: true,
  },
  plugins: [
    new VitePlugin({
      build: [
        {
          config: "vite.main.config.mts",
          entry: "electron/main.ts",
          target: "main",
        },
        {
          config: "vite.preload.config.mts",
          entry: "electron/preload.ts",
          target: "preload",
        },
      ],
      renderer: [
        {
          config: "vite.renderer.config.mts",
          name: "main_window",
        },
      ],
    }),

    new FusesPlugin({
      version: FuseVersion.V1,
      [FuseV1Options.RunAsNode]: false,
      [FuseV1Options.EnableCookieEncryption]: true,
      [FuseV1Options.EnableNodeOptionsEnvironmentVariable]: false,
      [FuseV1Options.EnableNodeCliInspectArguments]: false,
      [FuseV1Options.EnableEmbeddedAsarIntegrityValidation]: true,
      [FuseV1Options.OnlyLoadAppFromAsar]: true,
    }),
  ],
  publishers: [
    {
      config: {
        draft: true,
        prerelease: false,
        repository: {
          name: "my-memory",
          owner: "FelipeMiiller",
        },
      },
      name: "@electron-forge/publisher-github",
    },
  ],
  rebuildConfig: {},
};

export default config;
