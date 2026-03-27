// Build method types
export type BuildMethod = 'packer' | 'buildx' | 'kaniko' | 'docker' | 'paketo' | 'nix';

// Base configuration interface
export interface BuildMethodConfig {
    id: string;
    build_id: string;
    method: BuildMethod;
    config: Record<string, unknown>;
    created_at?: string;
    updated_at?: string;
}

// Packer-specific configuration
export interface PackerConfig extends BuildMethodConfig {
    config: {
        template: string;
        variables?: Record<string, unknown>;
        build_vars?: Record<string, string>;
        on_error?: string;
        parallel?: boolean;
    };
}

// Buildx-specific configuration
export interface BuildxConfig extends BuildMethodConfig {
    config: {
        dockerfile: string
        build_context: string
        registry_repo?: string
        platforms: string[]
        build_args?: Record<string, string>
        secrets?: Record<string, string>
        cache?: {
            from?: string
            to?: string
        }
        no_cache?: boolean
        outputs?: string[]
    }
}

// Kaniko-specific configuration
export interface KanikoConfig extends BuildMethodConfig {
    config: {
        dockerfile: string;
        build_context: string;
        cache_repo?: string;
        registry_repo: string;
        build_args?: Record<string, string>;
        skip_unused_stages?: boolean;
    };
}

// Docker-specific configuration
export interface DockerConfig extends BuildMethodConfig {
    config: {
        dockerfile: string;
        build_context: string;
        registry_repo?: string;
        target_stage?: string;
        build_args?: Record<string, string>;
        environment_vars?: Record<string, string>;
    };
}

// Paketo-specific configuration
export interface PaketoConfig extends BuildMethodConfig {
    config: {
        builder: string
        buildpacks?: string[]
        env?: Record<string, string>
        build_args?: Record<string, string>
    }
}

// Nix-specific configuration
export interface NixConfig extends BuildMethodConfig {
    config: {
        nix_expression?: string;
        flake_uri?: string;
        attributes: string[];
        outputs?: Record<string, string>;
        cache_dir?: string;
        pure?: boolean;
        show_trace?: boolean;
    };
}

// Union type for all config types
export type AnyBuildMethodConfig = PackerConfig | BuildxConfig | KanikoConfig | DockerConfig | PaketoConfig | NixConfig;

// Build method info
export interface BuildMethodInfo {
    id: BuildMethod;
    name: string;
    description: string;
    icon: string;
    requirements: string[];
    advantages: string[];
    bestFor: string;
    documentationUrl: string;
}

// Preset configuration
export interface PresetConfig {
    name: string;
    method: BuildMethod;
    description: string;
    parameters: Record<string, unknown>;
}

// API Request types
export interface CreatePackerConfigRequest {
    build_id: string;
    template: string;
    variables?: Record<string, unknown>;
    build_vars?: Record<string, string>;
    on_error?: string;
    parallel?: boolean;
}

export interface CreateBuildxConfigRequest {
    build_id: string
    dockerfile: {
        source: 'path' | 'content' | 'upload'
        path?: string
        content?: string
        filename?: string
    }
    build_context: string
    registry_repo: string
    platforms?: string[]
    build_args?: Record<string, string>
    secrets?: Record<string, string>
    cache?: {
        from?: string
        to?: string
    }
    no_cache?: boolean
    outputs?: string[]
}

export interface CreateKanikoConfigRequest {
    build_id: string
    dockerfile: string | {
        source: 'path' | 'content' | 'upload'
        path?: string
        content?: string
        filename?: string
    }
    build_context: string
    registry_repo: string
    cache_repo?: string
    build_args?: Record<string, string>
    skip_unused_stages?: boolean
}

export interface CreateDockerConfigRequest {
    build_id: string
    dockerfile: string | {
        source: 'path' | 'content' | 'upload'
        path?: string
        content?: string
        filename?: string
    }
    build_context: string
    registry_repo: string
    target_stage?: string
    build_args?: Record<string, string>
    environment_vars?: Record<string, string>
}

export interface CreatePaketoConfigRequest {
    build_id: string
    builder: string
    buildpacks?: string[]
    env?: Record<string, string>
    build_args?: Record<string, string>
}

export interface CreateNixConfigRequest {
    build_id: string
    nix_expression?: string
    flake_uri?: string
    attributes: string[]
    outputs?: Record<string, string>
    cache_dir?: string
    pure?: boolean
    show_trace?: boolean
}

// API Response types
export interface ConfigResponse {
    id: string;
    build_id: string;
    method: BuildMethod;
    config: Record<string, unknown>;
    created_at?: string;
    updated_at?: string;
}

export interface ListConfigsResponse {
    count: number;
    method: BuildMethod;
    configs: ConfigResponse[];
}

export interface PresetsResponse {
    [method: string]: PresetConfig[];
}

// Queue-related types
export interface QueuedBuild {
    id: string;
    build_id: string;
    project_id: string;
    tenant_id: string;
    method: BuildMethod;
    status: 'pending' | 'assigned' | 'processing' | 'completed' | 'failed';
    position: number;
    priority: number;
    assigned_to_worker_id?: string;
    assigned_at?: string;
    eta_minutes?: number;
    created_at: string;
    updated_at: string;
}

// Queue statistics
export interface QueueStats {
    total_pending?: number
    pending?: number
    total_assigned?: number
    assigned?: number
    total_processing?: number
    processing?: number
    by_method?: Record<BuildMethod, number>
    average_wait_time_minutes?: number
    average_wait_time?: number
    estimated_processing_time_minutes?: number
}

// Build history for analytics
export interface BuildHistory {
    id: string;
    build_id: string;
    method: BuildMethod;
    status: 'success' | 'failure' | 'timeout';
    duration_seconds: number;
    started_at: string;
    completed_at: string;
    worker_id?: string;
}

// Performance metrics
export interface PerformanceMetrics {
    method: BuildMethod;
    total_builds: number;
    successful_builds: number;
    failed_builds: number;
    success_rate: number;
    average_duration_seconds: number;
    min_duration_seconds: number;
    max_duration_seconds: number;
}

// ETA prediction
export interface ETAPrediction {
    project_id: string;
    method: BuildMethod;
    estimated_minutes: number;
    confidence: number;
    based_on_builds: number;
    last_updated: string;
}

// Config Template types
export interface ConfigTemplate {
    id: string;
    project_id: string;
    created_by_user_id: string;
    name: string;
    description?: string;
    method: BuildMethod;
    template_data: Record<string, unknown>;
    is_shared: boolean;
    is_public: boolean;
    created_at: string;
    updated_at: string;
}

export interface ConfigTemplateListResponse {
    templates: ConfigTemplate[];
    pagination: {
        total: number;
        limit: number;
        offset: number;
    };
}

export interface SaveTemplateRequest {
    project_id: string;
    name: string;
    description?: string;
    method: BuildMethod;
    template_data: Record<string, unknown>;
    is_shared?: boolean;
    is_public?: boolean;
}

export interface ConfigTemplateShare {
    id: string;
    template_id: string;
    shared_with_user_id: string;
    can_use: boolean;
    can_edit: boolean;
    can_delete: boolean;
    created_at: string;
    updated_at: string;
}
