import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Saat `npm run dev`, permintaan /api diteruskan ke backend Go di port 8081.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8081',
    },
  },
})
