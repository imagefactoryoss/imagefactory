export interface GitProvider {
    key: string
    display_name: string
    provider_type: string
    api_base_url?: string
    supports_api: boolean
}

export interface GitProviderListResponse {
    providers: GitProvider[]
}
