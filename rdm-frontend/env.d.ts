/// <reference types="vite/client" />

// `npm create vue` puts a file like this at the project root. If yours already
// exists, keep its first line and paste the interface below into it.

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>
  export default component
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_DEBUG_BYPASS_AUTH?: string
  readonly VITE_DEBUG_MOCK_API?: string
  readonly VITE_DEBUG_FORCE_ADMIN?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
