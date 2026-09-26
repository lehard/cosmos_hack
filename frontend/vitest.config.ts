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
      // Под нагрузкой машины сборки (много агентов) тест с живой картой идёт 4–5 с:
      // стандартных 5 с не хватает — ложные падения по таймауту.
      testTimeout: 20000,
      include: ['src/**/__tests__/*.spec.ts'],
    },
  }),
)
