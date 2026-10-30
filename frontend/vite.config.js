import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  // Discover the Worker dependency before the first calculation, without a dev-page reload.
  optimizeDeps: { include: ['mathjs'] },
  server: {
    proxy: { '/api': process.env.THEMIS_API_PROXY_TARGET || 'http://localhost:8080' }
  }
})
