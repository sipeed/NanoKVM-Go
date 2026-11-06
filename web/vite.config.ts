import { rm } from 'node:fs/promises';
import { resolve } from 'node:path';
import react from '@vitejs/plugin-react';
import { defineConfig, type Plugin } from 'vite';

const mpegtsDistSuffix = '/mpegts.js/dist/mpegts.js';
const upstreamMpegtsWorkerWrapperRegex = String.raw`/^function\s?\w*\(\w+,\s*\w+,\s*(\w+)\)/`;
const patchedMpegtsWorkerWrapperRegex = String.raw`/^function\s?[\w$]*\([\w$]+,\s*[\w$]+,\s*([\w$]+)\)/`;

function countOccurrences(source: string, target: string) {
  return source.split(target).length - 1;
}

function patchMpegtsWorkerWrapperRegex(): Plugin {
  let transformed = false;

  return {
    // mpegts.js@1.8.0 embeds webworkify-webpack, whose parser rejects the `$`
    // parameter names that the minifier may emit in the final production chunk.
    name: 'patch-mpegts-worker-wrapper-regex',
    enforce: 'pre',
    transform(code, id) {
      if (id.startsWith('\0') || id.includes('?')) return null;

      const normalizedId = id.replaceAll('\\', '/');
      if (!normalizedId.endsWith(mpegtsDistSuffix)) return null;

      const occurrences = countOccurrences(code, upstreamMpegtsWorkerWrapperRegex);
      if (occurrences !== 1) {
        this.error(`Expected one mpegts Worker wrapper regex, found ${occurrences}`);
      }

      transformed = true;
      return {
        code: code.replace(upstreamMpegtsWorkerWrapperRegex, patchedMpegtsWorkerWrapperRegex),
        map: null
      };
    },
    generateBundle(_options, bundle) {
      if (!transformed) {
        this.error('mpegts.js was not transformed during the production build');
      }

      let upstreamOccurrences = 0;
      let patchedOccurrences = 0;
      for (const output of Object.values(bundle)) {
        if (output.type !== 'chunk') continue;
        upstreamOccurrences += countOccurrences(output.code, upstreamMpegtsWorkerWrapperRegex);
        patchedOccurrences += countOccurrences(output.code, patchedMpegtsWorkerWrapperRegex);
      }

      if (upstreamOccurrences !== 0 || patchedOccurrences !== 1) {
        this.error(
          `Invalid mpegts Worker regex output: upstream=${upstreamOccurrences}, patched=${patchedOccurrences}`
        );
      }
    }
  };
}

function excludeMockServiceWorker(): Plugin {
  return {
    name: 'exclude-mock-service-worker',
    apply: 'build',
    async writeBundle({ dir }) {
      // Vite copies public files without tree shaking; this worker is dev-only.
      if (dir) await rm(resolve(dir, 'mockServiceWorker.js'), { force: true });
    }
  };
}

const localeChunkGroups = [
  ['th', 'hu', 'pl', 'nb', 'zh'],
  ['ru', 'de', 'es', 'da', 'en'],
  ['uk', 'fr', 'it', 'cz', 'zh_tw'],
  ['ja', 'vi', 'ko', 'nl', 'id']
] as const;

function getLocaleChunkName(id: string) {
  for (const [index, languages] of localeChunkGroups.entries()) {
    if (languages.some((language) => id.endsWith(`/locales/${language}.ts`))) {
      return `i18n-locales-${index + 1}`;
    }
  }
}

function getChunkName(id: string) {
  const localeChunkName = getLocaleChunkName(id);
  if (localeChunkName) return localeChunkName;

  if (
    id.includes('/src/i18n/') ||
    id.includes('/node_modules/i18next/') ||
    id.includes('/node_modules/react-i18next/')
  ) {
    return 'i18n-runtime';
  }

  if (
    id.includes('/node_modules/react/') ||
    id.includes('/node_modules/react-dom/') ||
    id.includes('/node_modules/scheduler/') ||
    id.includes('/node_modules/react-router') ||
    id.includes('/node_modules/@remix-run/router/')
  ) {
    return 'react';
  }

  if (
    id.includes('/node_modules/antd/') ||
    id.includes('/node_modules/@ant-design/') ||
    id.includes('/node_modules/rc-') ||
    id.includes('/node_modules/@rc-component/')
  ) {
    return 'antd';
  }

  if (
    id.includes('/node_modules/react-simple-keyboard/') ||
    id.includes('/node_modules/react-timezone-select/') ||
    id.includes('/node_modules/timezone-soft/') ||
    id.includes('/node_modules/spacetime/')
  ) {
    return 'desktop-tools';
  }

  if (
    id.includes('/src/pages/desktop/') &&
    !id.includes('/screen-timelapse-modal.tsx') &&
    !id.includes('/screen-timelapse-retention.tsx')
  ) {
    return 'desktop';
  }
}

export default defineConfig({
  plugins: [patchMpegtsWorkerWrapperRegex(), excludeMockServiceWorker(), react()],
  resolve: {
    tsconfigPaths: true
  },
  server: {
    port: 3001
  },
  build: {
    // Preserve the browser targets used before the Vite 8 upgrade.
    target: ['chrome107', 'edge107', 'firefox104', 'safari16'],
    chunkSizeWarningLimit: 1536,
    rolldownOptions: {
      output: {
        strictExecutionOrder: true,
        codeSplitting: {
          // Keep shared dependencies in their own groups instead of pulling
          // them recursively into the desktop chunk.
          includeDependenciesRecursively: false,
          groups: [{ name: getChunkName, debugName: 'app' }]
        }
      }
    }
  }
});
