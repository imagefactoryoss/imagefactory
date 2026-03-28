/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_APP_VERSION?: string
    readonly VITE_APP_BUILD_DATE?: string
    readonly VITE_PRODUCT_INFO_LAST_SYNC?: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}