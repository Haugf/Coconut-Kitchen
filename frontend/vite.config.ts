import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// In dev, the Go backend runs on :8080 and serves /api.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { '/api': 'http://localhost:8080' },
  },
})
