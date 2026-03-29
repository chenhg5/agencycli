import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
const devPort = 27891

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: devPort,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:27892',
        changeOrigin: true,
      },
    },
  },
  preview: {
    port: devPort,
    strictPort: true,
  },
})
