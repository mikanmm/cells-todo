import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// During npm run dev, proxy /api requests to the Go backend on port 8081.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8081',
    },
  },
})
