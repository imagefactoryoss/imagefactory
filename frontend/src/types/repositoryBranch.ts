export interface RepositoryBranchRequest {
    repository_url?: string
    auth_id?: string
    provider_key?: string
}

export interface RepositoryBranchResponse {
    branches: string[]
}
