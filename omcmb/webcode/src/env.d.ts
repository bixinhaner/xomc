/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string;
  readonly VITE_USE_MOCK: string;
  readonly VITE_MAP_TILE_URL?: string;
  readonly VITE_TILES_PROXY_TARGET?: string;
  readonly VITE_AGENT_RUNTIME_STREAM_URL?: string;
  readonly VITE_AGENT_RUNTIME_BASE_URL?: string;
  readonly VITE_AGENT_ACTION_CONNECTOR_ID?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
