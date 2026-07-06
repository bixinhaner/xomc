/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_AGENT_RUNTIME_STREAM_URL?: string
  readonly VITE_AGENT_RUNTIME_BASE_URL?: string
  readonly VITE_AGENT_ACTION_CONNECTOR_ID?: string
}

declare const __APP_VERSION__: string
