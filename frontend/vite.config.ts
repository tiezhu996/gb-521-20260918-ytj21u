import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 18521,
    proxy: {
      '/api': 'http://127.0.0.1:19521',
      '/healthz': 'http://127.0.0.1:19521',
      '/readyz': 'http://127.0.0.1:19521',
    },
  },
});
