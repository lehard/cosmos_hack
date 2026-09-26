// Юнит-тесты оболочки (vitest + @vue/test-utils + happy-dom): столы из yaml
// собираются из реестра, SSE инвалидирует ключи, права выключают действия
// в воспроизведении. Конфигурация сборки — общая с vite.config.ts.
import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config.ts'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'happy-dom',
      include: ['src/**/__tests__/*.spec.ts'],
    },
  }),
)
