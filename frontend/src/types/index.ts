// Common types
export interface ApiResponse<T> {
    data: T
    message?: string
    success: boolean
}

export interface PaginatedResponse<T> {
    data: T[]
    pagination: {
        page: number
        limit: number
        total: number
        totalPages: number
    }
}

export interface User {
    id: string
    email: string
    name: string
    role: UserRole
    tenantId?: string
    createdAt: string
    updatedAt: string
}

export type UserRole = 'admin' | 'tenant_admin' | 'developer' | 'viewer'

// Extended authentication types
export interface UserWithRoles extends User {
    roles: UserRoleWithPermissions[]
    tenant?: Tenant
    permissions: string[]
    isMFAEnabled: boolean
    lastLoginAt?: string
    loginCount: number
    status?: 'active' | 'inactive' | 'suspended' | 'pending'
    rolesByTenant?: Record<string, UserRoleWithPermissions[]> // Per-tenant role assignments
    auth_method?: string // Authentication method: 'ldap', 'credentials', 'oidc', 'api_key'
}

export interface UserRoleWithPermissions {
    id: string
    name: string
    permissions: Permission[]
    description?: string
    is_system?: boolean
}

export interface RoleAssignment {
    tenantId: string
    roleId: string
    roleName?: string
}

export interface Permission {
    id: string
    name: string
    resource: string
    action: string
    description?: string
    category?: string
    isSystemPermission?: boolean
    createdAt?: string
    updatedAt?: string
}

export interface LoginResponse {
    user: UserResponse
    access_token: string
    refresh_token: string
    access_token_ttl?: number // TTL in seconds
    refresh_token_ttl?: number // TTL in seconds
    access_token_expiry?: number // Unix timestamp when token expires
    requires_mfa?: boolean
    setup_required?: boolean
    requires_password_change?: boolean
}

export interface UserResponse {
    id: string
    email: string
    first_name: string
    last_name: string
    name?: string
    role?: string
    roles?: any[]
    status: string
    is_active: boolean
    auth_method: string
}

export interface RefreshTokenRequest {
    refresh_token: string
}

// Image Catalog Types
export interface Image {
    id: string
    tenant_id: string
    name: string
    description: string
    visibility: ImageVisibility
    status: ImageLifecycleStatus
    repository_url?: string
    registry_provider?: string
    architecture?: string
    os?: string
    language?: string
    framework?: string
    version?: string
    tags: string[]
    metadata: Record<string, any>
    size_bytes?: number
    pull_count: number
    created_by: string
    updated_by: string
    created_at: string
    updated_at: string
    deprecated_at?: string
    archived_at?: string
}

export type ImageVisibility = 'public' | 'tenant' | 'private'
export type ImageLifecycleStatus = 'draft' | 'published' | 'deprecated' | 'archived'

export interface ImageVersion {
    id: string
    image_id: string
    version: string
    description?: string
    digest?: string
    size_bytes?: number
    manifest: Record<string, any>
    config: Record<string, any>
    layers: Record<string, any>[]
    tags: string[]
    metadata: Record<string, any>
    created_by: string
    created_at: string
    published_at: string
    deprecated_at?: string
}

export interface ImageTag {
    id: string
    image_id: string
    tag: string
    category: string
    created_by: string
    created_at: string
}

export interface CreateImageRequest {
    name: string
    description?: string
    visibility: ImageVisibility
    tenant_id?: string // For system admins to specify which tenant
    repository_url?: string
    registry_provider?: string
    architecture?: string
    os?: string
    language?: string
    framework?: string
    version?: string
    tags?: string[]
    metadata?: Record<string, any>
}

export interface UpdateImageRequest {
    description?: string
    visibility?: ImageVisibility
    status?: ImageLifecycleStatus
    repository_url?: string
    registry_provider?: string
    architecture?: string
    os?: string
    language?: string
    framework?: string
    version?: string
    tags?: string[]
    metadata?: Record<string, any>
}

export interface SearchImagesRequest {
    query?: string
    status?: ImageLifecycleStatus
    registry_provider?: string
    architecture?: string
    os?: string
    language?: string
    framework?: string
    tags?: string[]
    sort_by?: string
    sort_order?: 'asc' | 'desc'
    limit?: number
    offset?: number
}

export interface ImageSearchFilters {
    query?: string
    visibility?: ImageVisibility[]
    status?: ImageLifecycleStatus[]
    registry_provider?: string[]
    architecture?: string[]
    os?: string[]
    language?: string[]
    framework?: string[]
    tags?: string[]
    tenant_id?: string
}

export interface ImageListResponse {
    images: Image[]
    pagination?: {
        page: number
        limit: number
        total: number
        totalPages: number
    }
}

export interface ImageVersionsResponse {
    versions: ImageVersion[]
}

export interface ImageTagListResponse {
    tags: string[]
}

export interface ImageDetailTag {
    tag: string
    category?: string
}

export interface CatalogImageMetadata {
    docker_config_digest?: string
    docker_manifest_digest?: string
    total_layer_count?: number
    compressed_size_bytes?: number
    uncompressed_size_bytes?: number
    packages_count?: number
    vulnerabilities_high_count?: number
    vulnerabilities_medium_count?: number
    vulnerabilities_low_count?: number
    entrypoint?: string
    cmd?: string
    env_vars?: string
    working_dir?: string
    labels?: string
    last_scanned_at?: string
    scan_tool?: string
    layers_evidence_status?: 'fresh' | 'stale' | 'unavailable'
    layers_evidence_build_id?: string
    layers_evidence_updated_at?: string
    sbom_evidence_status?: 'fresh' | 'stale' | 'unavailable'
    sbom_evidence_build_id?: string
    sbom_evidence_updated_at?: string
    vulnerability_evidence_status?: 'fresh' | 'stale' | 'unavailable'
    vulnerability_evidence_build_id?: string
    vulnerability_evidence_updated_at?: string
}

export interface CatalogImageLayer {
    layer_number: number
    layer_digest: string
    layer_size_bytes?: number
    media_type?: string
    is_base_layer: boolean
    base_image_name?: string
    base_image_tag?: string
    used_in_builds_count?: number
    last_used_in_build_at?: string
    history_created_by?: string
    source_command?: string
    diff_id?: string
    package_count?: number
    vulnerability_count?: number
    packages?: CatalogImageLayerPackage[]
    vulnerabilities?: CatalogImageLayerVulnerability[]
}

export interface CatalogImageLayerPackage {
    package_name: string
    package_version?: string
    package_type?: string
    package_path?: string
    source_command?: string
    known_vulnerabilities_count?: number
    critical_vulnerabilities_count?: number
}

export interface CatalogImageLayerVulnerability {
    cve_id: string
    severity: string
    cvss_v3_score?: number
    package_name?: string
    package_version?: string
    reference_url?: string
}

export interface CatalogImageSBOMPackage {
    package_name: string
    package_version?: string
    package_type?: string
    layer_digest?: string
    package_path?: string
    source_command?: string
    known_vulnerabilities_count?: number
    critical_vulnerabilities_count?: number
    high_severity_vulnerabilities?: CatalogImageSBOMVulnerability[]
}

export interface CatalogImageSBOMVulnerability {
    cve_id: string
    severity: string
    cvss_v3_score?: number
    description?: string
    published_date?: string
    reference_url?: string
}

export interface CatalogImageSBOM {
    format: string
    version?: string
    status?: string
    generated_by_tool?: string
    tool_version?: string
    scan_timestamp?: string
    scan_duration_seconds?: number
    packages: CatalogImageSBOMPackage[]
}

export interface CatalogImageVulnerabilityScan {
    id: string
    build_id?: string
    scan_tool: string
    tool_version?: string
    scan_status: string
    started_at?: string
    completed_at?: string
    duration_seconds?: number
    vulnerabilities_critical?: number
    vulnerabilities_high?: number
    vulnerabilities_medium?: number
    vulnerabilities_low?: number
    vulnerabilities_negligible?: number
    vulnerabilities_unknown?: number
    pass_fail_result?: string
    compliance_check_passed?: boolean
    scan_report_location?: string
    error_message?: string
}

export interface ImageStats {
    pull_count: number
    version_count: number
    layer_count: number
    vulnerability_scan_count: number
    last_updated: string
    latest_scan_status?: string
    latest_high_vulnerabilities?: number
    latest_medium_vulnerabilities?: number
}

export interface ImageDetailsResponse {
    image: Image
    versions: ImageVersion[]
    tags: {
        inline: string[]
        normalized: ImageDetailTag[]
        merged: string[]
    }
    metadata?: CatalogImageMetadata
    layers: CatalogImageLayer[]
    sbom?: CatalogImageSBOM
    vulnerability_scans: CatalogImageVulnerabilityScan[]
    stats: ImageStats
}

export type ImageImportStatus = 'pending' | 'approved' | 'importing' | 'success' | 'failed' | 'quarantined'
export type ImageImportSyncState = 'awaiting_approval' | 'awaiting_dispatch' | 'pipeline_running' | 'catalog_sync_pending' | 'completed' | 'failed' | 'dispatch_failed'
export type ImageImportExecutionState = 'awaiting_approval' | 'awaiting_dispatch' | 'pipeline_running' | 'evidence_pending' | 'ready_for_release' | 'completed'
export type ImageImportRequestType = 'quarantine' | 'scan'
export type ImageImportReleaseState = 'not_ready' | 'ready_for_release' | 'release_approved' | 'released' | 'release_blocked' | 'unknown'

export interface ImageImportDecisionTimeline {
    decision_status?: string
    decision_reason?: string
    decided_by_user_id?: string
    decided_at?: string
    workflow_step_status?: string
}

export interface ImageImportNotificationReconciliation {
    decision_event_type?: string
    idempotency_key?: string
    expected_recipients: number
    receipt_count: number
    in_app_notification_count: number
    delivery_state?: 'pending' | 'partial' | 'delivered'
}

export interface ImageImportRequest {
    id: string
    tenant_id: string
    requested_by_user_id: string
    request_type: ImageImportRequestType
    epr_record_id?: string
    source_registry: string
    source_image_ref: string
    registry_auth_id?: string
    status: ImageImportStatus
    error_message?: string
    internal_image_ref?: string
    pipeline_run_name?: string
    pipeline_namespace?: string
    policy_decision?: string
    policy_reasons_json?: string
    policy_snapshot_json?: string
    scan_summary_json?: string
    sbom_summary_json?: string
    sbom_evidence_json?: string
    source_image_digest?: string
    decision_timeline?: ImageImportDecisionTimeline
    notification_reconciliation?: ImageImportNotificationReconciliation
    sync_state?: ImageImportSyncState
    execution_state?: ImageImportExecutionState
    execution_state_updated_at?: string
    dispatch_queued_at?: string
    pipeline_started_at?: string
    evidence_ready_at?: string
    release_ready_at?: string
    failure_class?: 'dispatch' | 'auth' | 'connectivity' | 'policy' | 'runtime'
    failure_code?: string
    release_state?: ImageImportReleaseState
    release_eligible?: boolean
    release_blocker_reason?: string
    release_reason?: string
    retryable?: boolean
    created_at: string
    updated_at: string
}

export interface ReleasedArtifact {
    id: string
    tenant_id: string
    requested_by_user_id: string
    epr_record_id: string
    source_registry: string
    source_image_ref: string
    internal_image_ref?: string
    source_image_digest?: string
    policy_decision?: string
    policy_snapshot_json?: string
    release_state: ImageImportReleaseState
    release_reason?: string
    release_actor_user_id?: string
    release_requested_at?: string
    released_at?: string
    consumption_ready: boolean
    consumption_blocker_reason?: string
    created_at: string
    updated_at: string
}

export type EPRRegistrationStatus = 'pending' | 'approved' | 'rejected' | 'withdrawn'
export type EPRLifecycleStatus = 'active' | 'expiring' | 'expired' | 'suspended'

export interface EPRRegistrationRequest {
    id: string
    tenant_id: string
    epr_record_id: string
    product_name: string
    technology_name: string
    business_justification?: string
    source_registry?: string
    source_image_example?: string
    additional_notes?: string
    requested_by_user_id: string
    status: EPRRegistrationStatus
    lifecycle_status?: EPRLifecycleStatus
    approved_at?: string
    expires_at?: string
    suspension_reason?: string
    last_reviewed_at?: string
    decided_by_user_id?: string
    decision_reason?: string
    decided_at?: string
    created_at: string
    updated_at: string
}

export interface PopularImagesResponse {
    images: Image[]
}

export interface RecentImagesResponse {
    images: Image[]
}

// SSO Types
export interface SSOProvider {
    id: string
    type: 'saml' | 'oidc'
    name: string
    enabled: boolean
    config: SSOConfig
    metadata?: SSOProviderMetadata
    createdAt: string
    updatedAt: string
}

export interface SSOConfig {
    entityId?: string
    singleSignOnURL: string
    singleLogoutURL?: string
    certificate?: string
    attributeMapping: Record<string, string>
    nameIdFormat?: string
}

export interface SSOProviderMetadata {
    entityDescriptor?: string
    certificate?: string
    signingCertificate?: string
    encryptionCertificate?: string
}

export interface CreateSSOProviderRequest {
    type: 'saml' | 'oidc'
    name: string
    config: SSOConfig
}

export interface UpdateSSOProviderRequest {
    name?: string
    enabled?: boolean
    config?: Partial<SSOConfig>
}

// MFA Types
export type MFAMethod = 'totp' | 'sms' | 'email'

export interface MFASetup {
    secret: string
    qrCode: string
    backupCodes: string[]
}

export interface MFASetupRequest {
    method: MFAMethod
    phoneNumber?: string
    email?: string
}

export interface MFAChallenge {
    id: string
    method: MFAMethod
    expiresAt: string
}

export interface MFAChallengeResponse {
    challenge: MFAChallenge
    verificationToken: string
}

export interface VerifyMFARequest {
    challengeId: string
    verificationCode: string
    method: MFAMethod
}

export interface VerifyMFAResponse {
    success: boolean
    needsSetup: boolean
    setupMethod?: MFAMethod
}

// Authentication state types
export interface AuthState {
    user: UserWithRoles | null
    token: string | null
    refreshToken: string | null
    isAuthenticated: boolean
    isLoading: boolean
    requiresMFA: boolean
    mfaChallenge: MFAChallenge | null
}

// System Configuration types
export interface SystemConfig {
    id: string
    key: string
    value: any
    description?: string
    type: ConfigType
    category: ConfigCategory
    isEncrypted: boolean
    isEditable: boolean
    createdAt: string
    updatedAt: string
}

export type ConfigType = 'string' | 'number' | 'boolean' | 'json' | 'email' | 'url'

export type ConfigCategory = 'auth' | 'ldap' | 'smtp' | 'mfa' | 'storage' | 'notifications' | 'security'

export interface CreateSystemConfigRequest {
    key: string
    value: any
    description?: string
    type: ConfigType
    category: ConfigCategory
    isEncrypted?: boolean
}

export interface UpdateSystemConfigRequest {
    value?: any
    description?: string
    isEditable?: boolean
}

export interface TestConnectionRequest {
    type: 'ldap' | 'smtp' | 'sso'
    config: Record<string, any>
}

export interface TektonTaskImagesConfig {
    git_clone: string
    kaniko_executor: string
    buildkit: string
    skopeo: string
    trivy: string
    syft: string
    cosign: string
    packer: string
    python_alpine: string
    alpine: string
    cleanup_kubectl: string
}

// Onboarding types
export interface OnboardingWorkflow {
    id: string
    tenantId: string
    steps: OnboardingStep[]
    status: OnboardingStatus
    createdAt: string
    updatedAt: string
}

export interface OnboardingStep {
    id: string
    name: string
    description: string
    status: StepStatus
    completedAt?: string
    config?: Record<string, any>
    dependencies?: string[]
}

export type OnboardingStatus = 'pending' | 'in_progress' | 'completed' | 'failed'

export type StepStatus = 'pending' | 'in_progress' | 'completed' | 'failed' | 'skipped'

export interface StartOnboardingRequest {
    tenantId: string
    templateId?: string
    customSteps?: OnboardingStep[]
}

// Admin Dashboard types
export interface SystemStats {
    total_users: number
    active_users: number
    total_tenants: number
    active_tenants: number
    total_builds: number
    running_builds: number
    total_images: number
    storage_used_gb: number
    critical_vulnerabilities: number
    system_health: 'healthy' | 'warning' | 'critical'
    uptime: string
    last_backup?: string
    denial_metrics?: DenialMetricsRow[]
    release_metrics?: ReleaseMetricsSnapshot
    epr_lifecycle_metrics?: Record<string, number>
}

export interface ReleaseMetricsSnapshot {
    requested: number
    released: number
    failed: number
    consumed?: number
    total: number
}

export interface ReleaseGovernancePolicyConfig {
    enabled: boolean
    failure_ratio_threshold: number
    consecutive_failures_threshold: number
    minimum_samples: number
    window_minutes: number
}

export interface ProductInfoMetadataConfig {
    last_backlog_sync?: string
}

export type SREIncidentSeverity = 'info' | 'warning' | 'critical'
export type SREIncidentConfidence = 'low' | 'medium' | 'high'
export type SREIncidentStatus = 'observed' | 'triaged' | 'contained' | 'recovering' | 'resolved' | 'suppressed' | 'escalated'

export interface SREIncident {
    id: string
    tenant_id?: string | null
    correlation_key: string
    domain: string
    incident_type: string
    display_name: string
    summary: string
    severity: SREIncidentSeverity
    confidence: SREIncidentConfidence
    status: SREIncidentStatus
    source: string
    first_observed_at: string
    last_observed_at: string
    contained_at?: string | null
    resolved_at?: string | null
    suppressed_until?: string | null
    metadata?: Record<string, any> | null
    created_at: string
    updated_at: string
}

export interface SREFinding {
    id: string
    incident_id?: string | null
    source: string
    signal_type: string
    signal_key: string
    severity: SREIncidentSeverity
    confidence: SREIncidentConfidence
    title: string
    message: string
    raw_payload?: Record<string, any> | null
    occurred_at: string
    created_at: string
}

export interface SREEvidence {
    id: string
    incident_id: string
    evidence_type: string
    summary: string
    payload?: Record<string, any> | null
    captured_at: string
    created_at: string
}

export interface SREActionAttempt {
    id: string
    incident_id: string
    action_key: string
    action_class: string
    target_kind: string
    target_ref: string
    status: string
    actor_type: string
    actor_id?: string
    approval_required: boolean
    requested_at: string
    started_at?: string | null
    completed_at?: string | null
    error_message?: string
    result_payload?: Record<string, any> | null
    created_at: string
    updated_at: string
}

export interface SREApproval {
    id: string
    incident_id: string
    action_attempt_id?: string | null
    channel_provider_id: string
    status: string
    request_message: string
    requested_by?: string
    decided_by?: string
    decision_comment?: string
    requested_at: string
    decided_at?: string | null
    expires_at?: string | null
    created_at: string
    updated_at: string
}

export interface SREIncidentListResponse {
    incidents: SREIncident[]
    total: number
    limit: number
    offset: number
}

export interface SREIncidentDetailResponse {
    incident: SREIncident
    findings: SREFinding[]
    evidence: SREEvidence[]
    action_attempts: SREActionAttempt[]
    approvals: SREApproval[]
}

export interface SREIncidentWorkspaceResponse {
    incident: SREIncident
    executive_summary: string[]
    recommended_questions: string[]
    suggested_tooling: string[]
    default_tool_bundle?: string[]
    async_pressure_summary?: {
        backlog?: {
            incident_type: string
            display_name: string
            queue_kind: string
            subsystem: string
            count: number
            threshold: number
            threshold_delta: number
            threshold_ratio_percent: number
            trend: string
            operator_status: string
            latest_summary: string
            latest_captured_at?: string
            recent_observations?: Record<string, number>
            correlation_hints?: Record<string, string>
        }
        messaging_transport?: {
            status: string
            reconnects: number
            disconnects: number
            reconnect_threshold: number
            operator_status: string
            latest_summary: string
            latest_captured_at?: string
        }
        messaging_consumer?: {
            incident_type: string
            display_name: string
            kind: string
            stream: string
            consumer: string
            target_ref: string
            count: number
            threshold: number
            threshold_delta: number
            threshold_ratio_percent: number
            pending_count: number
            ack_pending_count: number
            waiting_count: number
            redelivered_count: number
            trend: string
            operator_status: string
            latest_summary: string
            latest_captured_at?: string
            last_active?: string
            correlation_hints?: Record<string, string>
        }
    }
    enabled_mcp_servers: SRESmartBotMCPServer[]
    agent_runtime: SRESmartBotAgentRuntimeConfig
}

export interface SREMCPToolDescriptor {
    server_id: string
    server_name: string
    server_kind: string
    tool_name: string
    display_name: string
    description: string
    read_only: boolean
    incident_required: boolean
}

export interface SREMCPToolListResponse {
    tools: SREMCPToolDescriptor[]
}

export interface SREMCPToolInvocationRequest {
    server_id: string
    tool_name: string
}

export interface SREMCPToolInvocationResponse {
    server_id: string
    server_name: string
    server_kind: string
    tool_name: string
    executed_at: string
    payload: Record<string, any>
}

export interface SREAgentDraftHypothesis {
    title: string
    confidence: string
    rationale: string
    signals_used: string[]
    evidence_refs?: SREAgentDraftEvidenceRef[]
}

export interface SREAgentDraftPlanStep {
    title: string
    description: string
    evidence_refs?: SREAgentDraftEvidenceRef[]
}

export interface SREAgentDraftEvidenceRef {
    tool_name: string
    server_name: string
    summary: string
}

export interface SREAgentDraftResponse {
    incident_id: string
    mode: string
    summary: string
    hypotheses: SREAgentDraftHypothesis[]
    investigation_plan: SREAgentDraftPlanStep[]
    tool_runs: SREMCPToolInvocationResponse[]
    human_confirmation_required: boolean
}

export interface SREAgentTriageResponse {
    incident_id: string
    mode: string
    summary: string
    probable_cause: string
    confidence: string
    next_checks: string[]
    recommended_action: string
    evidence_refs?: SREAgentDraftEvidenceRef[]
    human_confirmation_required: boolean
}

export interface SREAgentSeverityFactor {
    key: string
    label: string
    contribution: number
    reason: string
}

export interface SREAgentSeverityResponse {
    incident_id: string
    mode: string
    score: number
    level: string
    summary: string
    factors: SREAgentSeverityFactor[]
    human_confirmation_required: boolean
}

export interface SREAgentIncidentScorecardResponse {
    incident_id: string
    mode: string
    summary: string
    probable_cause: string
    confidence: string
    severity_score: number
    severity_level: string
    why_severe_cards: SREAgentSeverityFactor[]
    recommended_action: string
    action_key: string
    blast_radius: 'low' | 'medium' | 'high' | string
    execution_requires_approval: boolean
    human_confirmation_required: boolean
}

export interface SREAgentIncidentSnapshotResponse {
    incident_id: string
    mode: string
    summary: string
    triage?: SREAgentTriageResponse
    severity?: SREAgentSeverityResponse
    scorecard?: SREAgentIncidentScorecardResponse
    suggested_action?: SREAgentSuggestedActionResponse
    operator_handoff_note: string
    policy_guardrails?: string[]
    evidence_signals_expected?: string[]
    evidence_signals_observed?: string[]
    evidence_coverage_percent: number
    evidence_health_note: string
    human_confirmation_required: boolean
}

export interface SREAgentSuggestedActionResponse {
    incident_id: string
    mode: string
    action_key: string
    action_summary: string
    justification: string
    blast_radius: 'low' | 'medium' | 'high' | string
    advisory_only: boolean
    execution_requires_approval: boolean
    execution_guardrail: string
    evidence_refs?: SREAgentDraftEvidenceRef[]
    human_confirmation_required: boolean
}

export interface SREAgentInterpretationResponse {
    draft?: SREAgentDraftResponse
    provider: string
    model: string
    generated: boolean
    cache_hit?: boolean
    evidence_hash?: string
    summary_mode?: string
    timeline_summary?: string
    change_detection_15m?: string
    operator_handoff_note?: string
    fallback_reason?: string
    operator_summary?: string
    likely_root_cause?: string
    watchouts?: string[]
    citations?: SREAgentCitation[]
    operator_message_draft?: string
    raw_response?: string
}

export interface SREAgentCitation {
    kind: 'runbook' | 'evidence' | string
    source: string
    section?: string
    note: string
}

export interface SREAgentRuntimeProbeResponse {
    provider: string
    model: string
    base_url?: string
    healthy: boolean
    status: string
    message: string
    latency_ms?: number
    sample_response?: string
    model_installed: boolean
    installed_models?: string[]
    guidance?: string[]
}

export interface SREApprovalQueueItem {
    approval: SREApproval
    incident?: SREIncident
    action?: SREActionAttempt
}

export interface SREApprovalListResponse {
    approvals: SREApprovalQueueItem[]
    total: number
    limit: number
    offset: number
}

export interface SRESmartBotOperatorRule {
    id: string
    name: string
    domain: string
    incident_type: string
    severity: string
    enabled: boolean
    source: string
    match_labels?: Record<string, string>
    threshold?: number
    for_duration_seconds?: number
    suggested_action?: string
    auto_allowed?: boolean
}

export interface SRESmartBotDetectorRule {
    id: string
    name: string
    enabled: boolean
    source?: string
    query: string
    threshold?: number
    domain: string
    incident_type: string
    severity: string
    confidence?: string
    signal_key?: string
    suggested_action?: string
    auto_created?: boolean
}

export interface SRESmartBotChannelProvider {
    id: string
    name: string
    kind: string
    enabled: boolean
    supports_interactive_approval?: boolean
    config_ref?: string
    settings?: Record<string, string>
}

export interface SRESmartBotMCPServer {
    id: string
    name: string
    kind: string
    enabled: boolean
    transport: string
    endpoint?: string
    config_ref?: string
    allowed_tools?: string[]
    read_only?: boolean
    approval_required?: boolean
    settings?: Record<string, string>
}

export interface SRESmartBotAgentRuntimeConfig {
    enabled: boolean
    provider?: string
    model?: string
    base_url?: string
    system_prompt_ref?: string
    operator_summary_enabled: boolean
    hypothesis_ranking_enabled: boolean
    draft_action_plans_enabled: boolean
    conversational_approval_support: boolean
    max_tool_calls_per_turn: number
    max_incidents_per_summary: number
    require_human_confirmation_for_message: boolean
}

export interface SRESmartBotPolicyConfig {
    display_name: string
    enabled: boolean
    environment_mode: string
    detector_learning_mode?: 'disabled' | 'suggest_only' | 'training_auto_create'
    default_channel?: string
    default_channel_provider_id?: string
    auto_observe_enabled: boolean
    auto_notify_enabled: boolean
    auto_contain_enabled: boolean
    auto_recover_enabled: boolean
    require_approval_for_recover: boolean
    require_approval_for_disruptive: boolean
    duplicate_alert_suppression_seconds: number
    action_cooldown_seconds: number
    enabled_domains: string[]
    channel_providers: SRESmartBotChannelProvider[]
    mcp_servers: SRESmartBotMCPServer[]
    agent_runtime: SRESmartBotAgentRuntimeConfig
    detector_rules: SRESmartBotDetectorRule[]
    operator_rules: SRESmartBotOperatorRule[]
}

export interface SRESmartBotMutationResponse {
    incident?: SREIncident
    action?: SREActionAttempt
    approval?: SREApproval
}

export interface SRERemediationPack {
    key: string
    version: string
    name: string
    summary: string
    risk_tier: string
    action_class: string
    requires_approval: boolean
    incident_types: string[]
}

export interface SRERemediationPackListResponse {
    packs: SRERemediationPack[]
}

export interface SRERemediationPackRun {
    id: string
    tenant_id?: string | null
    incident_id: string
    pack_key: string
    pack_version: string
    run_kind: string
    status: string
    requested_by?: string
    request_id?: string
    approval_id?: string | null
    action_attempt_id?: string | null
    summary?: string
    result_payload?: Record<string, any> | null
    started_at?: string | null
    completed_at?: string | null
    created_at: string
    updated_at: string
}

export interface SRERemediationPackRunListResponse {
    runs: SRERemediationPackRun[]
}

export interface SRERemediationPackRunResponse {
    run: SRERemediationPackRun
}

export type SREDetectorRuleSuggestionStatus = 'pending' | 'accepted' | 'rejected'

export interface SREDetectorRuleSuggestion {
    id: string
    tenant_id?: string | null
    incident_id?: string | null
    fingerprint: string
    name: string
    description: string
    query: string
    threshold: number
    domain: string
    incident_type: string
    severity: SREIncidentSeverity
    confidence: SREIncidentConfidence
    signal_key?: string
    source: string
    status: SREDetectorRuleSuggestionStatus
    auto_created: boolean
    reason?: string
    evidence_payload?: Record<string, any> | null
    proposed_by?: string
    reviewed_by?: string
    reviewed_at?: string | null
    activated_rule_id?: string
    created_at: string
    updated_at: string
}

export interface SREDetectorRuleSuggestionListResponse {
    suggestions: SREDetectorRuleSuggestion[]
    limit: number
    offset: number
}

export interface SREDemoScenario {
    id: string
    name: string
    summary: string
    recommended_walkthrough: string
}

export interface SREDemoScenarioListResponse {
    scenarios: SREDemoScenario[]
}

export interface DenialMetricsRow {
    tenant_id?: string
    capability_key: string
    reason: string
    labels?: Record<string, string>
    count: number
}

export interface DispatcherMetrics {
    claims: number
    dispatches: number
    claim_errors: number
    dispatch_errors: number
    requeues: number
    skipped_for_limit: number
    claim_count: number
    claim_min_ms: number
    claim_max_ms: number
    claim_avg_ms: number
    dispatch_count: number
    dispatch_min_ms: number
    dispatch_max_ms: number
    dispatch_avg_ms: number
    available?: boolean
    mode?: 'embedded' | 'external'
    source?: string
    last_heartbeat?: string
    stale?: boolean
}

export interface DispatcherStatus {
    running: boolean
    available?: boolean
    mode?: 'embedded' | 'external'
    source?: string
    last_heartbeat?: string
    stale?: boolean
    managed_externally?: boolean
    message?: string
}

export interface SystemComponentStatus {
    name: string
    status: 'healthy' | 'warning' | 'critical'
    last_check: string
    message?: string
    endpoint?: string
    http_status?: number
    latency_ms?: number
    configured: boolean
}

export interface SystemComponentsStatusResponse {
    status: 'healthy' | 'warning' | 'critical'
    checked_at: string
    components: Record<string, SystemComponentStatus>
}

export interface ExecutionPipelineComponentHealth {
    enabled: boolean
    running: boolean
    available: boolean
    mode?: string
    source?: string
    last_activity?: string
    message?: string
    metrics?: Record<string, number>
}

export interface ExecutionPipelineHealthResponse {
    checked_at: string
    components: Record<string, ExecutionPipelineComponentHealth>
    workflow_control_plane?: {
        subject_type: string
        blocked_step_count: number
        dispatch_blocked: number
        monitor_blocked: number
        finalize_blocked: number
        oldest_blocked_at?: string
        oldest_blocked_age_seconds?: number
        recovery_action?: string
        recovery_hint?: string
    }
}

export interface UserManagementFilters {
    role?: UserRole[]
    tenantId?: string
    mfaEnabled?: boolean
    status?: 'active' | 'inactive'
    search?: string
    page?: number
    limit?: number
}

export interface TenantManagementFilters {
    status?: TenantStatus[]
    search?: string
    page?: number
    limit?: number
}

// Tenant types
export interface Tenant {
    id: string
    numericId: string | number
    tenantCode: string
    name: string
    slug: string
    description?: string
    contactEmail?: string
    industry?: string
    country?: string
    status: TenantStatus
    quota?: ResourceQuota
    config?: TenantConfig
    createdAt: string
    updatedAt: string
    version: number
}

export type TenantStatus = 'active' | 'suspended' | 'pending' | 'deleted'

export interface ResourceQuota {
    maxBuilds: number
    maxImages: number
    maxStorageGB: number
    maxConcurrentJobs: number
}

export interface TenantConfig {
    buildTimeout: string // duration string
    allowedImageTypes: string[]
    securityPolicies: Record<string, any>
    notificationSettings: Record<string, any>
}

export interface CreateTenantRequest {
    name: string
    slug: string
    description?: string
}

export interface UpdateTenantRequest {
    name?: string
    description?: string
    quota?: ResourceQuota
    config?: TenantConfig
}

// Build types - Updated for multi-tool build system
export interface Build {
    id: string
    tenantId: string
    projectId?: string
    buildNumber?: number
    gitBranch?: string
    gitCommit?: string
    failureReason?: string | null
    manifest: BuildManifest
    status: BuildStatus
    result?: BuildResult
    errorMessage?: string
    createdAt: string
    startedAt?: string
    completedAt?: string
    updatedAt: string
    version: number
}

export type BuildStatus =
    | 'pending'
    | 'queued'
    | 'running'
    | 'completed'
    | 'failed'
    | 'cancelled'

export type WorkflowStepStatus =
    | 'pending'
    | 'running'
    | 'succeeded'
    | 'failed'
    | 'blocked'

export interface BuildWorkflowStep {
    stepKey: string
    status: WorkflowStepStatus
    attempts: number
    lastError?: string | null
    startedAt?: string | null
    completedAt?: string | null
    createdAt: string
    updatedAt: string
}

export interface BuildWorkflowResponse {
    instanceId?: string
    executionId?: string
    status?: string
    steps: BuildWorkflowStep[]
}

export interface BuildTraceRuntimeComponent {
    enabled: boolean
    running: boolean
    last_activity?: string
    message?: string
}

export interface BuildTraceResponse {
    build: Build
    executions: BuildExecutionAttempt[]
    selectedExecutionId?: string
    workflow: BuildWorkflowResponse
    diagnostics?: {
        repoConfig?: {
            applied: boolean
            path?: string
            ref?: string
            stage?: string
            error?: string
            errorCode?: string
            updatedAt?: string
        }
    }
    runtime?: Record<string, BuildTraceRuntimeComponent>
    correlation?: {
        workflowInstanceId?: string
        executionId?: string
        activeStepKey?: string
    }
}

export interface BuildExecutionAttempt {
    id: string
    status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled'
    createdAt: string
    startedAt?: string
    completedAt?: string
    durationSeconds?: number
    errorMessage?: string
}

export type BuildType =
    | 'container'
    | 'vm'
    | 'cloud'
    | 'packer'
    | 'paketo'
    | 'kaniko'
    | 'buildx'
    | 'nix'

export type SBOMTool = 'syft' | 'grype' | 'trivy'

export type ScanTool = 'trivy' | 'clair' | 'grype' | 'snyk'

export type RegistryType = 's3' | 'harbor' | 'quay' | 'artifactory'

export type SecretManagerType = 'vault' | 'aws_secretsmanager' | 'azure_keyvault' | 'gcp_secretmanager'

export interface ToolAvailabilityConfig {
    build_methods: BuildMethodAvailability
    sbom_tools: SBOMToolAvailability
    scan_tools: ScanToolAvailability
    registry_types: RegistryTypeAvailability
    secret_managers: SecretManagerAvailability
    trivy_runtime?: TrivyRuntimeConfig
}

export interface TrivyRuntimeConfig {
    cache_mode: 'shared' | 'direct' | ''
    db_repository: string
    java_db_repository: string
}

export interface BuildManifest {
    name: string
    type: BuildType
    baseImage: string
    instructions: string[]
    environment: Record<string, string>
    tags: string[]
    metadata: Record<string, any>
    infrastructureType?: InfrastructureType
    infrastructureProviderId?: string
    infrastructure_type?: InfrastructureType
    infrastructure_provider_id?: string
    vmConfig?: VMBuildConfig
    buildConfig?: BuildConfig
}

export interface VMBuildConfig {
    packerTemplate?: string
    packerVariables?: Record<string, any>
    cloudProvider?: string
    region?: string
    availabilityZone?: string
    instanceType?: string
    vmSize?: string
    storageConfig?: VMStorageConfig
    networkConfig?: VMNetworkConfig
    outputFormat?: string
    outputName?: string
    outputDescription?: string
    provisioners?: VMProvisioner[]
    postProcessors?: VMPostProcessor[]
    buildTimeout?: number
    maxRetries?: number
    securityConfig?: VMSecurityConfig
}

export interface VMStorageConfig {
    rootVolumeSizeGB?: number
    rootVolumeType?: string
    dataVolumes?: VMDataVolume[]
    encrypted?: boolean
    kmsKeyID?: string
}

export interface VMDataVolume {
    sizeGB: number
    volumeType: string
    mountPoint?: string
    deviceName?: string
}

export interface VMNetworkConfig {
    vpcID?: string
    subnetID?: string
    securityGroups?: string[]
    associatePublicIP?: boolean
}

export interface VMProvisioner {
    type: string
    config: Record<string, any>
    only?: string[]
    except?: string[]
}

export interface VMPostProcessor {
    type: string
    config: Record<string, any>
    only?: string[]
    except?: string[]
}

export interface VMSecurityConfig {
    enableSELinux?: boolean
    enableFirewall?: boolean
    allowedPorts?: string[]
    sshKeys?: string[]
    users?: VMUser[]
}

export interface VMUser {
    username: string
    password?: string
    sshKeys?: string[]
    groups?: string[]
    sudo?: boolean
}

export interface BuildConfig {
    buildType: BuildType
    sbomTool: SBOMTool
    scanTool: ScanTool
    registryType: RegistryType
    secretManagerType: SecretManagerType
    // Basic build fields
    baseImage?: string
    instructions?: string[]
    tags?: string[]
    environment?: Record<string, string>
    // Packer-specific fields
    packerTemplate?: string
    packerTargetProfileId?: string
    buildVars?: Record<string, string>
    onError?: string
    parallel?: boolean
    // Buildpack-specific fields
    paketoConfig?: PaketoConfig
    // Nix-specific fields
    nixExpression?: string
    flakeUri?: string
    nixAttributes?: string[]
    nixOutputs?: Record<string, string>
    nixCacheDir?: string
    nixPure?: boolean
    nixShowTrace?: boolean
    // Dockerfile-based fields
    dockerfile?: string | {
        source: 'path' | 'content' | 'upload'
        path?: string
        content?: string
        filename?: string
    }
    buildContext?: string
    buildArgs?: Record<string, string>
    target?: string
    cache?: boolean
    cacheRepo?: string
    registryRepo?: string
    registryAuthId?: string
    skipUnusedStages?: boolean
    // Buildx-specific fields
    platforms?: string[]
    cacheTo?: string
    cacheFrom?: string[]
    secrets?: Record<string, string>
    // Common fields
    variables?: Record<string, any>
    builders?: PackerBuilder[]
    provisioners?: VMProvisioner[]
    postProcessors?: VMPostProcessor[]
    // Infrastructure selection
    infrastructure?: InfrastructureType
    // Source binding
    sourceId?: string
    refPolicy?: 'source_default' | 'fixed' | 'event_ref'
    fixedRef?: string
}

export interface PackerBuilder {
    type: string
    config: Record<string, any>
}

export interface PaketoConfig {
    builder: string
    buildpacks?: string[]
    env?: Record<string, string>
    buildArgs?: Record<string, string>
}

export interface BuildResult {
    imageId: string
    imageDigest: string
    size: number
    duration: string // duration string
    logs: string[]
    artifacts: string[]
    sbom: Record<string, any>
    scanResults: Record<string, any>
}

export interface CreateBuildRequest {
    tenantId: string
    projectId?: string
    manifest: BuildManifest
}

// Image types (Build artifacts)
export interface BuildArtifact {
    id: string
    tenantId: string
    buildId: string
    name: string
    tag: string
    digest: string
    size: number
    type: BuildType
    status: ImageStatus
    metadata: Record<string, any>
    sbom: Record<string, any>
    scanResults: ImageScanResults
    createdAt: string
    updatedAt: string
}

export type ImageStatus = 'building' | 'ready' | 'quarantined' | 'deleted'

export interface ImageScanResults {
    vulnerabilities: Vulnerability[]
    totalCount: number
    criticalCount: number
    highCount: number
    mediumCount: number
    lowCount: number
    scanDate: string
}

export interface Vulnerability {
    id: string
    severity: 'critical' | 'high' | 'medium' | 'low'
    title: string
    description: string
    package: string
    installedVersion: string
    fixedVersion?: string
    cveIds: string[]
}

// Notification types
export interface Notification {
    id: string
    type: NotificationType
    title: string
    message: string
    read: boolean
    createdAt: string
    metadata?: Record<string, any>
}

export type NotificationType =
    | 'build_completed'
    | 'build_failed'
    | 'image_ready'
    | 'security_alert'
    | 'system_alert'

// API Error types
export interface ApiError {
    message: string
    code?: string
    details?: Record<string, any>
}

// Form types
export interface LoginForm {
    email: string
    password: string
    use_ldap?: boolean
    tenant_id?: string
}

export interface BuildForm {
    name: string
    type: BuildType
    baseImage: string
    instructions: string
    environment: string
    tags: string
    metadata: string
}

// User Invitation types
export interface UserInvitation {
    id: string
    email: string
    tenantId: string
    tenantName?: string
    roleId?: string
    roleName?: string
    status: 'pending' | 'accepted' | 'expired' | 'revoked'
    token: string
    expiresAt: string
    acceptedAt?: string
    createdAt: string
    updatedAt: string
    invitedBy: string
    invitedByName?: string
}

export interface CreateInvitationRequest {
    email: string
    tenantId: string
    roleId?: string
    message?: string
}

export interface AcceptInvitationRequest {
    token: string
    password: string
    firstName: string
    lastName: string
}

export interface InvitationFilters {
    status?: string[]
    tenantId?: string
    invitedBy?: string
    search?: string
    page?: number
    limit?: number
}

// Audit types
export interface AuditEvent {
    id: string
    tenant_id: string
    user_id?: string
    user_name?: string
    event_type: string
    severity: string
    resource: string
    action: string
    ip_address?: string
    user_agent?: string
    details?: Record<string, any>
    message: string
    timestamp: string
}

export interface AuditEventsResponse {
    events: AuditEvent[]
    total: number
    limit: number
    offset: number
}

export interface AuditEventsFilters {
    user_id?: string
    event_type?: string
    severity?: string
    resource?: string
    action?: string
    start_time?: string
    end_time?: string
    search?: string
    limit?: number
    offset?: number
}

// Bulk Operation types
export interface BulkOperation {
    id: string
    operationType: 'import' | 'deactivate' | 'assign_roles'
    status: 'pending' | 'processing' | 'completed' | 'failed'
    totalCount: number
    succeededCount: number
    failedCount: number
    startedAt: string
    completedAt?: string
    createdBy: string
    createdByName?: string
    tenantId?: string
    tenantName?: string
    metadata?: Record<string, any>
    results?: BulkOperationResult[]
}

export interface BulkOperationResult {
    index: number
    success: boolean
    error?: string
    data?: any
}

export interface BulkImportRequest {
    tenantId: string
    file: File
    roleId?: string
}

export interface BulkDeactivateRequest {
    userIds: string[]
    tenantId: string
    reason: string
}

export interface BulkAssignRolesRequest {
    assignments: Array<{
        userId: string
        roleId: string
        tenantId: string
    }>
}

// Password Reset types
export interface PasswordResetRequest {
    email: string
}

export interface PasswordResetConfirmRequest {
    token: string
    password: string
}

export interface PasswordResetTokenValidation {
    valid: boolean
    email?: string
    expiresAt?: string
}

// User Activity types
export interface UserActivity {
    id: string
    userId: string
    action: string
    resource: string
    details?: Record<string, any>
    ipAddress?: string
    userAgent?: string
    timestamp: string
}

export interface LoginHistory {
    id: string
    userId: string
    ipAddress: string
    userAgent: string
    success: boolean
    failureReason?: string
    timestamp: string
}

export interface LoginHistory {
    id: string
    userId: string
    ipAddress: string
    userAgent: string
    success: boolean
    failureReason?: string
    timestamp: string
}

export interface UserSession {
    id: string
    userId: string
    ipAddress: string
    userAgent: string
    createdAt: string
    expiresAt: string
    isActive: boolean
}

// Dashboard types
export interface DashboardStats {
    totalBuilds: number
    runningBuilds: number
    completedBuilds: number
    failedBuilds: number
    totalImages: number
    totalVulnerabilities: number
    activeTenants: number
    storageUsage: number
}

export interface RecentActivity {
    type: 'build' | 'image' | 'tenant'
    action: string
    resource: string
    timestamp: string
    user: string
    tenantId?: string
}

// Filter types
export interface BuildFilters {
    status?: BuildStatus[]
    type?: BuildType[]
    tenantId?: string
    startDate?: string
    endDate?: string
    search?: string
}

export interface ImageFilters {
    status?: ImageStatus[]
    type?: BuildType[]
    tenantId?: string
    hasVulnerabilities?: boolean
    minSize?: number
    maxSize?: number
    search?: string
}

export interface TenantFilters {
    status?: TenantStatus[]
    search?: string
}

// Administrative types for tool availability management
export interface ToolAvailabilityConfig {
    build_methods: BuildMethodAvailability
    sbom_tools: SBOMToolAvailability
    scan_tools: ScanToolAvailability
    registry_types: RegistryTypeAvailability
    secret_managers: SecretManagerAvailability
    trivy_runtime?: TrivyRuntimeConfig
}

export interface BuildMethodAvailability {
    container: boolean
    packer: boolean
    paketo: boolean
    kaniko: boolean
    buildx: boolean
    nix: boolean
}

export interface SBOMToolAvailability {
    syft: boolean
    grype: boolean
    trivy: boolean
}

export interface ScanToolAvailability {
    trivy: boolean
    clair: boolean
    grype: boolean
    snyk: boolean
}

export interface RegistryTypeAvailability {
    s3: boolean
    harbor: boolean
    quay: boolean
    artifactory: boolean
}

export interface SecretManagerAvailability {
    vault: boolean
    aws_secretsmanager: boolean
    azure_keyvault: boolean
    gcp_secretmanager: boolean
}

export interface BuildCapabilitiesConfig {
    gpu: boolean
    privileged: boolean
    multi_arch: boolean
    high_memory: boolean
    host_networking: boolean
    premium: boolean
}

export interface OperationCapabilitiesConfig {
    build: boolean
    quarantine_request: boolean
    quarantine_release: boolean
    ondemand_image_scanning?: boolean
}

export interface CapabilitySurfaceSet {
    nav_keys: string[]
    route_keys: string[]
    action_keys: string[]
}

export interface CapabilitySurfaceDenial {
    reason_code: string
    capability: keyof OperationCapabilitiesConfig | string
    message: string
}

export interface CapabilitySurfacesResponse {
    tenant_id: string
    version: string
    capabilities: OperationCapabilitiesConfig
    surfaces: CapabilitySurfaceSet
    denials: Record<string, CapabilitySurfaceDenial>
}

// Project types - Added for build management system
export interface Project {
    id: string
    tenantId: string
    name: string
    slug?: string
    description?: string
    repositoryUrl?: string
    branch?: string
    gitProvider?: string
    repositoryAuthId?: string
    status: ProjectStatus
    visibility?: ProjectVisibility
    isDraft?: boolean
    buildCount: number
    lastBuildAt?: string
    createdAt: string
    updatedAt: string
    version: number
}

export type ProjectBuildConfigMode = 'ui_managed' | 'repo_managed'
export type ProjectBuildConfigOnError = 'strict' | 'fallback_to_ui'

export interface ProjectBuildSettings {
    projectId: string
    buildConfigMode: ProjectBuildConfigMode
    buildConfigFile: string
    buildConfigOnError: ProjectBuildConfigOnError
    updatedAt?: string
}

export interface ProjectSource {
    id: string
    projectId: string
    tenantId: string
    name: string
    provider: string
    repositoryUrl: string
    defaultBranch: string
    repositoryAuthId?: string
    isDefault: boolean
    isActive: boolean
    createdAt: string
    updatedAt: string
}

export interface CreateProjectSourceRequest {
    name: string
    provider?: string
    repositoryUrl: string
    defaultBranch?: string
    repositoryAuthId?: string
    isDefault?: boolean
    isActive?: boolean
}

export interface UpdateProjectSourceRequest extends CreateProjectSourceRequest { }

export type ProjectStatus = 'active' | 'archived' | 'suspended'

export type ProjectVisibility = 'private' | 'internal' | 'public'

export interface CreateProjectRequest {
    tenantId: string
    name: string
    slug?: string
    description?: string
    repositoryUrl?: string
    branch?: string
    gitProvider?: string
    repositoryAuthId?: string
    visibility?: ProjectVisibility
    isDraft?: boolean
}

export interface UpdateProjectRequest {
    name?: string
    slug?: string
    description?: string
    repositoryUrl?: string
    branch?: string
    status?: ProjectStatus
    gitProvider?: string
    repositoryAuthId?: string
    visibility?: ProjectVisibility
    isDraft?: boolean
}

// Build Configuration types - Reusable build templates within projects
export interface BuildConfiguration {
    id: string
    projectId: string
    tenantId: string
    name: string
    description?: string
    buildConfig: BuildConfig
    isActive: boolean
    createdAt: string
    updatedAt: string
    version: number
    lastUsedAt?: string
    usageCount: number
}

export interface BuildContextSuggestion {
    path: string
    reason: string
    score: number
}

export interface BuildDockerfileSuggestion {
    path: string
    context: string
    score: number
}

export interface BuildContextSuggestionsResponse {
    project_id: string
    repo_url: string
    ref?: string
    contexts: BuildContextSuggestion[]
    dockerfiles: BuildDockerfileSuggestion[]
    note?: string
}

export interface CreateBuildConfigurationRequest {
    projectId: string
    tenantId: string
    name: string
    description?: string
    buildConfig: BuildConfig
}

export interface UpdateBuildConfigurationRequest {
    name?: string
    description?: string
    buildConfig?: BuildConfig
    isActive?: boolean
}

// Wizard state for build creation
export interface WizardState {
    currentStep: number
    selectedProject?: Project
    buildName: string
    buildDescription: string
    buildMethod?: BuildType
    selectedTools: {
        sbom?: SBOMTool
        scan?: ScanTool
        registry?: RegistryType
        secrets?: SecretManagerType
    }
    buildConfig: Partial<BuildConfig>
    infrastructureType?: InfrastructureType
    infrastructureProviderId?: string | null
    validationErrors: Record<string, string>
    isSubmitting: boolean
}

// ============================================================================
// Infrastructure Types (Phase 3)
// ============================================================================

export type InfrastructureType = 'kubernetes' | 'aws-eks' | 'gcp-gke' | 'azure-aks' | 'oci-oke' | 'vmware-vks' | 'openshift' | 'rancher' | 'build_nodes'

export interface InfrastructureRecommendationRequest {
    build_method: string
    project_id: string
    config?: Record<string, any>
}

export interface InfrastructureRecommendation {
    recommended_infrastructure: InfrastructureType
    reason: string
    confidence: number
    requirements?: BuildRequirements
    alternatives?: AlternativeOption[]
    timestamp: string
}

export interface AlternativeOption {
    infrastructure: InfrastructureType
    reason: string
    confidence: number
}

export interface BuildRequirements {
    method: string
    resources: BuildResources
    timeout: number
    environment: Record<string, string>
    capabilities: string[]
}

export interface BuildResources {
    cpu: number
    memory_gb: number
    disk_gb: number
}

export interface InfrastructureUsageResponse {
    total_builds: number
    infrastructure_usage: InfrastructureUsage[]
    time_range: string
    timestamp: string
}

export interface InfrastructureUsage {
    infrastructure_type: InfrastructureType
    build_count: number
    percentage: number
    avg_duration_seconds: number
    success_rate: number
}

// ============================================================================
// Infrastructure Provider Management Types (Admin)
// ============================================================================

export type InfrastructureProviderType = 'kubernetes' | 'aws-eks' | 'gcp-gke' | 'azure-aks' | 'oci-oke' | 'vmware-vks' | 'openshift' | 'rancher' | 'build_nodes'

export interface InfrastructureProvider {
    id: string
    tenant_id: string
    is_global: boolean
    provider_type: InfrastructureProviderType
    name: string
    display_name: string
    config: Record<string, any> // Encrypted in backend
    status: 'online' | 'offline' | 'maintenance' | 'pending'
    capabilities: string[]
    created_by: string
    created_at: string
    updated_at: string
    last_health_check?: string
    health_status?: 'healthy' | 'warning' | 'critical' | 'unhealthy'
    readiness_status?: 'ready' | 'not_ready' | string
    readiness_last_checked?: string
    readiness_missing_prereqs?: string[]
    bootstrap_mode: 'image_factory_managed' | 'self_managed' | string
    credential_scope: 'cluster_admin' | 'namespace_admin' | 'read_only' | 'unknown' | string
    target_namespace?: string
    is_schedulable: boolean
    schedulable_reason?: string
    blocked_by?: string[]
    latest_prepare_run_id?: string
    latest_prepare_status?: ProviderPrepareRunStatus
    latest_prepare_updated_at?: string
    latest_prepare_error?: string
    latest_prepare_check_category?: string
    latest_prepare_check_severity?: 'info' | 'warn' | 'error' | string
    latest_prepare_remediation_hint?: string
}

export interface CreateInfrastructureProviderRequest {
    provider_type: InfrastructureProviderType
    name: string
    display_name: string
    config: Record<string, any>
    capabilities?: string[]
    is_global?: boolean
    bootstrap_mode?: 'image_factory_managed' | 'self_managed' | string
    credential_scope?: 'cluster_admin' | 'namespace_admin' | 'read_only' | 'unknown' | string
    target_namespace?: string
}

export interface UpdateInfrastructureProviderRequest {
    display_name?: string
    config?: Record<string, any>
    capabilities?: string[]
    status?: 'online' | 'offline' | 'maintenance'
    is_global?: boolean
    bootstrap_mode?: 'image_factory_managed' | 'self_managed' | string
    credential_scope?: 'cluster_admin' | 'namespace_admin' | 'read_only' | 'unknown' | string
    target_namespace?: string
}

export interface InfrastructureProviderHealth {
    provider_id: string
    status: 'healthy' | 'warning' | 'critical' | 'unhealthy'
    last_check: string
    details?: Record<string, any>
    metrics?: {
        total_nodes?: number
        healthy_nodes?: number
        total_cpu_capacity?: number
        used_cpu_cores?: number
        total_memory_capacity_gb?: number
        used_memory_gb?: number
    }
}

export interface TestProviderConnectionResponse {
    success: boolean
    message: string
    details?: Record<string, any>
}

export type PackerTargetProvider = 'vmware' | 'aws' | 'azure' | 'gcp'
export type PackerTargetValidationStatus = 'untested' | 'valid' | 'invalid'

export interface PackerTargetProfile {
    id: string
    tenant_id: string
    is_global: boolean
    name: string
    provider: PackerTargetProvider
    description?: string
    secret_ref: string
    options: Record<string, any>
    validation_status: PackerTargetValidationStatus
    last_validated_at?: string
    last_validation_message?: string
    last_remediation_hints?: string[]
    created_by: string
    created_at: string
    updated_at: string
}

export interface PackerTargetValidationCheck {
    name: string
    ok: boolean
    message?: string
    remediation_hint?: string
}

export interface PackerTargetValidationResult {
    profile_id: string
    provider: PackerTargetProvider
    status: PackerTargetValidationStatus
    checked_at: string
    checks: PackerTargetValidationCheck[]
    message: string
    remediation_hints: string[]
}

export type TektonInstallMode = 'gitops' | 'image_factory_installer'

export type TektonInstallerJobStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'cancelled'

export interface TektonInstallerJob {
    id: string
    provider_id: string
    tenant_id: string
    requested_by: string
    install_mode: TektonInstallMode
    asset_version: string
    status: TektonInstallerJobStatus
    error_message?: string
    started_at?: string
    completed_at?: string
    created_at: string
    updated_at: string
}

export interface TektonInstallerJobEvent {
    id: string
    job_id: string
    event_type: string
    message: string
    details?: Record<string, any>
    created_by?: string
    created_at: string
}

export interface TektonProviderStatus {
    provider_id: string
    readiness_status?: string
    readiness_last_checked?: string
    readiness_missing_prereqs?: string[]
    required_tasks?: string[]
    required_pipelines?: string[]
    active_job?: TektonInstallerJob
    recent_jobs: TektonInstallerJob[]
    active_job_events?: TektonInstallerJobEvent[]
}

export type ProviderPrepareRunStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'cancelled'

export interface ProviderPrepareRun {
    id: string
    provider_id: string
    tenant_id: string
    requested_by: string
    status: ProviderPrepareRunStatus
    requested_actions?: Record<string, any>
    result_summary?: Record<string, any>
    error_message?: string
    started_at?: string
    completed_at?: string
    created_at: string
    updated_at: string
}

export interface ProviderPrepareRunCheck {
    id: string
    run_id: string
    check_key: string
    category: string
    severity: 'info' | 'warn' | 'error' | string
    ok: boolean
    message: string
    details?: Record<string, any>
    created_at: string
}

export interface ProviderPrepareStatus {
    provider_id: string
    active_run?: ProviderPrepareRun
    checks?: ProviderPrepareRunCheck[]
}

export interface ProviderPrepareSummary {
    provider_id: string
    run_id?: string
    status?: ProviderPrepareRunStatus
    updated_at?: string
    error_message?: string
    latest_prepare_check_category?: string
    latest_prepare_check_severity?: 'info' | 'warn' | 'error' | string
    latest_prepare_remediation_hint?: string
}

export interface ProviderPrepareSummaryBatchMetrics {
    batch_count: number
    batch_total_ms: number
    batch_min_ms: number
    batch_max_ms: number
    batch_avg_ms: number
    batch_errors: number
    providers_total: number
    repository_batches: number
    fallback_batches: number
}

export type ProviderTenantNamespacePrepareStatus =
    | 'pending'
    | 'running'
    | 'succeeded'
    | 'failed'
    | 'cancelled'

export type TenantAssetDriftStatus = 'current' | 'stale' | 'unknown'

export interface ProviderTenantNamespacePrepare {
    id: string
    provider_id: string
    tenant_id: string
    namespace: string
    requested_by?: string
    status: ProviderTenantNamespacePrepareStatus
    result_summary?: Record<string, any>
    error_message?: string
    desired_asset_version?: string
    installed_asset_version?: string
    asset_drift_status: TenantAssetDriftStatus
    started_at?: string
    completed_at?: string
    created_at?: string
    updated_at?: string
}

export interface TenantNamespaceReconcileResult {
    tenant_id: string
    status: 'applied' | 'failed' | string
    message?: string
}

export interface TenantNamespaceReconcileSummary {
    provider_id: string
    mode: 'stale_only' | 'selected' | string
    targeted: number
    applied: number
    failed: number
    skipped: number
    stale_only_filter: boolean
    results: TenantNamespaceReconcileResult[]
}
