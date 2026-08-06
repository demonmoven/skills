import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { readFileSync } from 'node:fs'

const rootPackage = JSON.parse(
  readFileSync(new URL('../package.json', import.meta.url), 'utf-8'),
) as { version?: string }

export default defineConfig({
  plugins: [react()],
  define: {
    __GLOOP_VERSION__: JSON.stringify(rootPackage.version ?? '0.0.0'),
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:37317',
    },
  },
})
