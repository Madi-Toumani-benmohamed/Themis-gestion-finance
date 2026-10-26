import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { '/api': process.env.THEMIS_API_PROXY_TARGET || 'http://localhost:8080' }
  }
})
