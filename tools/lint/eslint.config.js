// Konfigurasi ESLint frontend Claim PNC.
//
// Berkas ini tinggal di tools/lint/, bukan di akar frontend, karena typescript-eslint belum
// mendukung TypeScript 7 yang dipakai proyek: TypeScript 7 tidak lagi menyediakan API JS
// yang dibutuhkan parser. Folder ini membawa TypeScript 5.9 miliknya sendiri HANYA untuk
// parsing lint; kompilasi dan pemeriksaan tipe tetap memakai TypeScript 7 di akar.
//
// Jalankan dari akar frontend:  npm run lint
import { fileURLToPath } from 'node:url'

import js from '@eslint/js'
import reactHooks from 'eslint-plugin-react-hooks'
import sonarjs from 'eslint-plugin-sonarjs'
import globals from 'globals'
import tseslint from 'typescript-eslint'

const frontendRoot = fileURLToPath(new URL('../..', import.meta.url))

// Pembagian keparahan, diambil dari metadata aturan sonarjs sendiri:
//   meta.type === 'problem'     bug dan keamanan  → error   (memblokir pipeline)
//   meta.type === 'suggestion'  maintainability    → warn    (dilaporkan, tidak memblokir)
// Dengan begitu tidak ada daftar aturan yang dipelihara tangan di sini.
function severityByType(rules) {
  return Object.fromEntries(
    Object.entries(rules).map(([name, value]) => {
      const current = Array.isArray(value) ? value[0] : value
      if (current === 'off' || current === 0) return [name, 'off']
      const rule = sonarjs.rules[name.replace('sonarjs/', '')]
      const level = rule?.meta?.type === 'problem' ? 'error' : 'warn'
      const options = Array.isArray(value) ? value.slice(1) : []
      return [name, [level, ...options]]
    }),
  )
}

export default tseslint.config(
  {
    basePath: frontendRoot,
    ignores: ['node_modules/**', 'tools/**', 'coverage/**', 'dist/**'],
  },
  {
    basePath: frontendRoot,
    files: ['src/**/*.{ts,tsx}', 'vite.config.ts'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended, sonarjs.configs.recommended],
    languageOptions: {
      ecmaVersion: 2022,
      globals: { ...globals.browser, ...globals.node },
      parserOptions: {
        projectService: true,
        tsconfigRootDir: frontendRoot,
      },
    },
    plugins: { 'react-hooks': reactHooks },
    rules: {
      ...severityByType(sonarjs.configs.recommended.rules),
      // rules-of-hooks adalah bug (hook dipanggil bersyarat); sisanya saran.
      ...Object.fromEntries(
        Object.keys(reactHooks.configs.recommended.rules).map((name) => [name, 'warn']),
      ),
      'react-hooks/rules-of-hooks': 'error',
      // Variabel tak terpakai sudah ditegakkan tsc (noUnusedLocals/noUnusedParameters).
      '@typescript-eslint/no-unused-vars': 'warn',
    },
  },
)
