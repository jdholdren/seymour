import { fileURLToPath, URL } from 'node:url'
import { execSync } from 'node:child_process'
import tailwindcss from '@tailwindcss/vite'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// Get git info for version display
function getGitInfo() {
  try {
    const gitHash = execSync('git rev-parse --short HEAD').toString().trim()
    const gitBranch = execSync('git rev-parse --abbrev-ref HEAD').toString().trim()
    return { gitHash, gitBranch }
  } catch {
    return { gitHash: 'unknown', gitBranch: 'unknown' }
  }
}

const gitInfo = getGitInfo()

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
    tailwindcss(),
  ],
  define: {
    'import.meta.env.VITE_GIT_HASH': JSON.stringify(gitInfo.gitHash),
    'import.meta.env.VITE_GIT_BRANCH': JSON.stringify(gitInfo.gitBranch),
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    port: 3000,
    proxy: {},
  },
})
