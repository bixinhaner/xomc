/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_USE_MOCK: string
  readonly VITE_API_BASE_URL: string
  readonly VITE_AGENT_RUNTIME_STREAM_URL?: string
  readonly VITE_AGENT_RUNTIME_BASE_URL?: string
  readonly VITE_AGENT_ACTION_CONNECTOR_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare const __APP_VERSION__: string
