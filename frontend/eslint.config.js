// ESLint интерфейса ant.
//
// Главное правило (AD-20): ручные HTTP-вызовы запрещены. Серверное состояние
// приходит только через сгенерированный клиент orval + Vue Query
// (src/shared/api/generated). Единственное исключение — модуль живых
// обновлений src/shared/api/sse на сгенерированных типах.
import js from '@eslint/js'
import { defineConfig } from 'eslint/config'
import pluginVue from 'eslint-plugin-vue'
import globals from 'globals'
import tseslint from 'typescript-eslint'

const HTTP_MESSAGE =
  'AD-20: ручные HTTP-вызовы запрещены — используйте сгенерированный клиент (src/shared/api/generated, orval + Vue Query); живые обновления — только src/shared/api/sse.'

const HTTP_CLIENT_LIBS = [
  'axios',
  'ky',
  'ofetch',
  'redaxios',
  'node-fetch',
  'cross-fetch',
  'isomorphic-fetch',
  'whatwg-fetch',
  'undici',
  'superagent',
  'got',
  'wretch',
  '@vueuse/core',
]

const noManualHttp = {
  'no-restricted-globals': [
    'error',
    ...['fetch', 'XMLHttpRequest', 'EventSource', 'WebSocket'].map((name) => ({ name, message: HTTP_MESSAGE })),
  ],
  'no-restricted-properties': [
    'error',
    ...['window', 'globalThis', 'self'].flatMap((object) =>
      ['fetch', 'XMLHttpRequest', 'EventSource', 'WebSocket'].map((property) => ({
        object,
        property,
        message: HTTP_MESSAGE,
      })),
    ),
    { object: 'navigator', property: 'sendBeacon', message: HTTP_MESSAGE },
  ],
  'no-restricted-imports': [
    'error',
    {
      paths: HTTP_CLIENT_LIBS.map((name) => ({
        name,
        message: HTTP_MESSAGE,
        // из @vueuse/core запрещены только сетевые помощники
        ...(name === '@vueuse/core' ? { importNames: ['useFetch', 'createFetch', 'useEventSource', 'useWebSocket'] } : {}),
      })),
    },
  ],
}

export default defineConfig(
  { ignores: ['dist/**', '.npm-cache/**', 'node_modules/**', 'src/shared/api/generated/**'] },
  js.configs.recommended,
  tseslint.configs.recommended,
  // essential — правила корректности; оформление не навязываем линтером.
  pluginVue.configs['flat/essential'],
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: { parser: tseslint.parser, extraFileExtensions: ['.vue'], sourceType: 'module' },
    },
  },
  {
    files: ['src/**/*.{ts,vue}'],
    languageOptions: { globals: globals.browser },
    rules: noManualHttp,
  },
  {
    // Единственный модуль с прямым сетевым доступом — поток SSE (AD-20, AD-21).
    files: ['src/shared/api/sse/**'],
    rules: { 'no-restricted-globals': 'off', 'no-restricted-properties': 'off' },
  },
  {
    // Конфигурации и скрипты генерации и проверок (scripts/) выполняются в Node.
    files: ['*.config.{js,ts}', 'scripts/**/*.mjs'],
    languageOptions: { globals: globals.node },
  },
)
