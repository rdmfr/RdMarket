import tailwindcss from '@tailwindcss/vite';
import vue from '@vitejs/plugin-vue';
import path from 'path';
import { spawn } from 'child_process';
import { defineConfig, type Plugin } from 'vite';

function goBackendPlugin(): Plugin {
  return {
    name: 'go-backend',
    configureServer() {
      if (process.platform === 'win32') return;

      try {
        const goProc = spawn(path.resolve(__dirname, '../backend/bin/server'), [], {
          env: { ...process.env, PORT: '8088', DATABASE_URL: '' },
          stdio: 'ignore',
          detached: true,
        });
        goProc.on('error', (error) => {
          console.warn('Note: Go server spawn skipped or already running:', error.message);
        });
        goProc.unref();
      } catch (e) {
        console.warn('Note: Go server spawn skipped or already running:', e);
      }
    },
  };
}

export default defineConfig(() => {
  return {
    plugins: [vue(), tailwindcss(), goBackendPlugin()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      port: 3000,
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:8088',
          changeOrigin: true,
        },
      },
      // HMR is disabled in AI Studio via DISABLE_HMR env var.
      hmr: process.env.DISABLE_HMR !== 'true',
      watch: process.env.DISABLE_HMR === 'true' ? null : {},
    },
  };
});
