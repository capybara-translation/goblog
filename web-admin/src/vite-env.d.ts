/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly MODE: string
  readonly VITE_BLOG_TIMEZONE?: string
  readonly VITE_BLOG_TITLE?: string
  // more env variables...
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
