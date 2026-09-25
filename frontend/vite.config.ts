// Сборка интерфейса ant. Результат (dist/) встраивается в бинарник ant
// (deploy/Dockerfile → backend/internal/infrastructure/transport/webui/dist).
// Все ресурсы — шрифты, иконки — собираются локально, без обращений в интернет
// во время работы (NFR-SEC-1, NFR-UI-2).
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: false,
    // Шрифты не встраиваем в CSS: отдельные woff2 кэшируются браузером.
    assetsInlineLimit: 0,
  },
  server: {
    // Разработка: API — у запущенного ant (make up / make run).
    proxy: {
      '/api': process.env.ANT_API_URL ?? 'http://127.0.0.1:8480',
    },
  },
})
