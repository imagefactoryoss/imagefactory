export type RegistryAuthType = 'basic_auth' | 'token' | 'dockerconfigjson'

export interface RegistryAuth {
    id: string
    tenant_id: string
    project_id?: string
    scope: 'tenant' | 'project'
    name: string
    description?: string
    registry_type: string
    auth_type: RegistryAuthType
    registry_host: string
    is_active: boolean
    is_default: boolean
    created_by: string
    created_at: string
    updated_at: string
}

export interface RegistryAuthListResponse {
    registry_auth: RegistryAuth[]
    total_count: number
}

export interface CreateRegistryAuthRequest {
    project_id?: string
    name: string
    description?: string
    registry_type: string
    auth_type: RegistryAuthType
    registry_host: string
    is_default?: boolean
    credentials: Record<string, string>
}

export interface UpdateRegistryAuthRequest {
    name: string
    description?: string
    registry_type: string
    auth_type: RegistryAuthType
    registry_host: string
    is_default?: boolean
    credentials?: Record<string, string>
}

export interface TestRegistryAuthPermissionsRequest {
    registry_repo: string
}

export interface TestRegistryAuthPermissionsResponse {
    success: boolean
    message: string
    registry_host?: string
    registry_repo?: string
}
