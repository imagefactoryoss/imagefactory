# Admin Route Tenant Scope Checklist

## Scope Contract
- Default scope: current `AuthContext.TenantID`.
- Cross-tenant scope: system admin only, and only when explicitly requested via `all_tenants=true` or `X-Tenant-Scope: all`.
- Tenant override: `tenant_id` query param; system admin can target any tenant, non-admin must match own tenant.
- Global routes: no tenant data model (system/runtime only), intentionally not tenant-scoped.

## Route Checklist

| Route | Handler | Scope Status | Notes |
|---|---|---|---|
| `POST /api/v1/admin/projects/purge-deleted` | `ProjectHandler.PurgeDeletedProjects` | `GLOBAL` | System operation; not tenant-dimensioned. |
| `GET /api/v1/admin/infrastructure/usage` | `http/handlers.InfrastructureHandler.GetInfrastructureUsage` | `DONE` | Tenant/all-tenant scope enforcement added; response payload still mock data. |
| `GET /api/v1/admin/infrastructure/providers` | `InfrastructureProviderHandler.ListProviders` | `DONE` | Explicit all-tenant + tenant override implemented; supports `include_prepare_summary=true|false` (default `true`) for optional prepare metadata enrichment; summary enrichment is skipped above large-list cap to protect latency. |
| `POST /api/v1/admin/infrastructure/providers` | `InfrastructureProviderHandler.CreateProvider` | `DONE` | Scoped to request tenant from auth context. |
| `GET /api/v1/admin/infrastructure/providers/prepare/summary` | `InfrastructureProviderHandler.GetProviderPrepareSummaries` | `DONE` | Explicit tenant scope enforcement + scoped prepare-run summary aggregation. |
| `GET /api/v1/admin/infrastructure/providers/{id}` | `InfrastructureProviderHandler.GetProvider` | `DONE` | Scoped provider-access guard enforces tenant/all-tenant contract. |
| `PUT /api/v1/admin/infrastructure/providers/{id}` | `InfrastructureProviderHandler.UpdateProvider` | `DONE` | Scoped provider-access guard enforces tenant/all-tenant contract. |
| `DELETE /api/v1/admin/infrastructure/providers/{id}` | `InfrastructureProviderHandler.DeleteProvider` | `DONE` | Scoped provider-access guard enforces tenant/all-tenant contract. |
| `POST /api/v1/admin/infrastructure/providers/{id}/test-connection` | `InfrastructureProviderHandler.TestProviderConnection` | `DONE` | Scoped provider-access guard enforces tenant/all-tenant contract. |
| `GET /api/v1/admin/infrastructure/providers/{id}/health` | `InfrastructureProviderHandler.GetProviderHealth` | `DONE` | Scoped provider-access guard enforces tenant/all-tenant contract. |
| `GET /api/v1/admin/infrastructure/providers/{id}/readiness` | `InfrastructureProviderHandler.GetProviderReadiness` | `DONE` | Explicit auth + tenant override validation added. |
| `GET /api/v1/admin/infrastructure/providers/{id}/quarantine-dispatch-readiness` | `InfrastructureProviderHandler.GetProviderQuarantineDispatchReadiness` | `DONE` | Lightweight quarantine dispatch eligibility contract with explicit blocker reasons (`tekton_enabled`, `quarantine_dispatch_enabled`, provider readiness/schedulable posture, runtime auth config). |
| `POST /api/v1/admin/infrastructure/providers/{id}/prepare` | `InfrastructureProviderHandler.PrepareProvider` | `DONE` | Scoped provider-access guard + tenant/all-tenant contract for prepare-run start. |
| `GET /api/v1/admin/infrastructure/providers/{id}/prepare/status` | `InfrastructureProviderHandler.GetProviderPrepareStatus` | `DONE` | Scoped provider-access guard + tenant/all-tenant contract for active run status. |
| `GET /api/v1/admin/infrastructure/providers/{id}/prepare/runs` | `InfrastructureProviderHandler.ListProviderPrepareRuns` | `DONE` | Scoped provider-access guard + tenant/all-tenant contract for prepare run history. |
| `GET /api/v1/admin/infrastructure/providers/{id}/prepare/runs/{run_id}` | `InfrastructureProviderHandler.GetProviderPrepareRun` | `DONE` | Scoped provider-access guard + tenant/all-tenant contract for read-only prepare run details and checks. |
| `GET /api/v1/admin/infrastructure/providers/{id}/prepare/stream` | `InfrastructureProviderHandler.StreamProviderPrepareStatus` | `DONE` | Scoped provider-access guard + tenant/all-tenant contract for live prepare status stream (WebSocket). |
| `POST /api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/provision-namespace` | `InfrastructureProviderHandler.ProvisionTenantNamespace` | `DONE` | System-admin only; scoped provider-access guard; explicit `tenant_id` path param validated; uses bootstrap credentials to provision per-tenant namespace. |
| `POST /api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/prepare-namespace` | `InfrastructureProviderHandler.PrepareTenantNamespace` | `DONE` | Legacy alias for backward compatibility; prefer `provision-namespace`. |
| `POST /api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/deprovision-namespace` | `InfrastructureProviderHandler.DeprovisionTenantNamespace` | `DONE` | System-admin only; scoped provider-access guard; explicit `tenant_id` path param validated; safely tears down managed tenant namespace with ownership checks. |
| `GET /api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/provision-namespace/status` | `InfrastructureProviderHandler.GetTenantNamespacePrepareStatus` | `DONE` | System-admin only; scoped provider-access guard; explicit `tenant_id` path param validated; returns latest per-tenant provisioning status. |
| `GET /api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/provision-namespace/stream` | `InfrastructureProviderHandler.StreamTenantNamespacePrepareStatus` | `DONE` | System-admin only; scoped provider-access guard; explicit `tenant_id` path param validated; streams live per-tenant provisioning status over WebSocket. |
| `GET /api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/prepare-namespace/status` | `InfrastructureProviderHandler.GetTenantNamespacePrepareStatus` | `DONE` | Legacy alias for backward compatibility; prefer `provision-namespace/status`. |
| `GET /api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/prepare-namespace/stream` | `InfrastructureProviderHandler.StreamTenantNamespacePrepareStatus` | `DONE` | Legacy alias for backward compatibility; prefer `provision-namespace/stream`. |
| `POST /api/v1/admin/infrastructure/providers/{id}/tenants/reconcile-stale` | `InfrastructureProviderHandler.ReconcileStaleTenantNamespaces` | `DONE` | System-admin only; scoped provider-access guard; targeted reconcile for stale tenant namespaces only. |
| `POST /api/v1/admin/infrastructure/providers/{id}/tenants/reconcile-selected` | `InfrastructureProviderHandler.ReconcileSelectedTenantNamespaces` | `DONE` | System-admin only; scoped provider-access guard; request body `tenant_ids` explicitly scopes targeted tenant namespace reconcile. |
| `POST /api/v1/admin/infrastructure/providers/{id}/tekton/install` | `InfrastructureProviderHandler.InstallTekton` | `DONE` | Scoped provider-access guard enforced via shared installer entrypoint. |
| `POST /api/v1/admin/infrastructure/providers/{id}/tekton/upgrade` | `InfrastructureProviderHandler.UpgradeTekton` | `DONE` | Scoped provider-access guard enforced via shared installer entrypoint. |
| `POST /api/v1/admin/infrastructure/providers/{id}/tekton/retry` | `InfrastructureProviderHandler.RetryTektonJob` | `DONE` | Scoped provider-access guard + scoped tenant ID for retry operation. |
| `GET /api/v1/admin/infrastructure/providers/{id}/tekton/status` | `InfrastructureProviderHandler.GetTektonStatus` | `DONE` | Scoped provider-access guard enforces tenant/all-tenant contract. |
| `POST /api/v1/admin/infrastructure/providers/{id}/tekton/validate` | `InfrastructureProviderHandler.ValidateTekton` | `DONE` | Scoped provider-access guard enforced via shared installer entrypoint. |
| `PATCH /api/v1/admin/infrastructure/providers/{id}/status` | `InfrastructureProviderHandler.ToggleProviderStatus` | `DONE` | Scoped provider-access guard enforces tenant/all-tenant contract. |
| `GET /api/v1/admin/infrastructure/providers/{id}/permissions` | `InfrastructureProviderHandler.ListProviderPermissions` | `DONE` | Scoped provider-access guard enforces tenant/all-tenant contract. |
| `POST /api/v1/admin/infrastructure/providers/{id}/permissions` | `InfrastructureProviderHandler.GrantProviderPermission` | `DONE` | Scoped provider-access guard + tenant override checks. |
| `DELETE /api/v1/admin/infrastructure/providers/{id}/permissions` | `InfrastructureProviderHandler.RevokeProviderPermission` | `DONE` | Scoped provider-access guard + tenant override checks. |
| `GET /api/v1/admin/packer-target-profiles` | `PackerTargetProfileHandler.ListProfiles` | `DONE` | Uses shared tenant scope resolver with explicit all-tenant support and optional provider filter. |
| `POST /api/v1/admin/packer-target-profiles` | `PackerTargetProfileHandler.CreateProfile` | `DONE` | Creates tenant-scoped profile from auth scope; system admins can provide explicit `tenant_id` in request body. |
| `GET /api/v1/admin/packer-target-profiles/{id}` | `PackerTargetProfileHandler.GetProfile` | `DONE` | Reads profile within resolved tenant scope. |
| `PUT /api/v1/admin/packer-target-profiles/{id}` | `PackerTargetProfileHandler.UpdateProfile` | `DONE` | Updates profile within resolved tenant scope and resets validation state to `untested`. |
| `DELETE /api/v1/admin/packer-target-profiles/{id}` | `PackerTargetProfileHandler.DeleteProfile` | `DONE` | Deletes profile within resolved tenant scope. |
| `POST /api/v1/admin/packer-target-profiles/{id}/validate` | `PackerTargetProfileHandler.ValidateProfile` | `DONE` | Runs deterministic preflight validation and persists `valid|invalid` status, timestamp, and remediation hints. |
| `POST /api/v1/admin/system/reboot` | `SystemConfigHandler.RebootServer` | `GLOBAL` | System runtime action. |
| `POST /api/v1/system-configs` | `SystemConfigHandler.CreateConfig` | `DONE` | Tenant-scoped by default; explicit tenant override enforced via auth scope checks. |
| `GET /api/v1/system-configs` | `SystemConfigHandler.ListConfigs` | `DONE` | Supports explicit all-tenant scope for system admins via resolver contract. |
| `POST /api/v1/system-configs/category` | `SystemConfigHandler.UpdateCategoryConfig` | `DONE` | Category write path enforces tenant/all-tenant scope via shared resolver. |
| `POST /api/v1/system-configs/test-connection` | `SystemConfigHandler.TestConnection` | `DONE` | Runtime test path remains permission-protected and tenant-scoped where applicable. |
| `GET /api/v1/admin/settings/tools` | `SystemConfigHandler.GetToolAvailability` | `DONE` | Tenant scope resolver + explicit all-tenants support. |
| `PUT /api/v1/admin/settings/tools` | `SystemConfigHandler.UpdateToolAvailability` | `DONE` | Tenant scope resolver + explicit all-tenants support. |
| `GET /api/v1/admin/settings/tekton-task-images` | `SystemConfigHandler.GetTektonTaskImages` | `GLOBAL` | Global Tekton task image defaults; no tenant dimension today. |
| `PUT /api/v1/admin/settings/tekton-task-images` | `SystemConfigHandler.UpdateTektonTaskImages` | `GLOBAL` | Global Tekton task image defaults; no tenant dimension today. |
| `GET /api/v1/admin/settings/product-info-metadata` | `SystemConfigHandler.GetProductInfoMetadata` | `GLOBAL` | Global runtime metadata for Product Info page (for example `last_backlog_sync`). |
| `GET /api/v1/admin/settings/build-capabilities` | `SystemConfigHandler.GetBuildCapabilities` | `DONE` | Tenant scope resolver + explicit all-tenants support for admin capability view. |
| `PUT /api/v1/admin/settings/build-capabilities` | `SystemConfigHandler.UpdateBuildCapabilities` | `DONE` | Tenant scope resolver + explicit all-tenants support for capability updates. |
| `GET /api/v1/admin/settings/operation-capabilities` | `SystemConfigHandler.GetOperationCapabilities` | `DONE` | Tenant scope resolver + explicit all-tenants support for operation capability view. |
| `PUT /api/v1/admin/settings/operation-capabilities` | `SystemConfigHandler.UpdateOperationCapabilities` | `DONE` | Tenant scope resolver + explicit all-tenants support for operation capability updates. |
| `GET /api/v1/admin/settings/capability-surfaces` | `SystemConfigHandler.GetCapabilitySurfaces` | `GLOBAL` | Global capability surface metadata for UX rendering. |
| `GET /api/v1/admin/settings/quarantine-policy` | `SystemConfigHandler.GetQuarantinePolicy` | `GLOBAL` | Global quarantine policy defaults and controls. |
| `PUT /api/v1/admin/settings/quarantine-policy` | `SystemConfigHandler.UpdateQuarantinePolicy` | `GLOBAL` | Global quarantine policy update action. |
| `POST /api/v1/admin/settings/quarantine-policy/validate` | `SystemConfigHandler.ValidateQuarantinePolicy` | `GLOBAL` | Global quarantine policy validation operation. |
| `POST /api/v1/admin/settings/quarantine-policy/simulate` | `SystemConfigHandler.SimulateQuarantinePolicy` | `GLOBAL` | Global quarantine policy simulation operation. |
| `GET /api/v1/admin/settings/sor-registration` | `SystemConfigHandler.GetSORRegistration` | `GLOBAL` | Global EPR registration policy settings. |
| `PUT /api/v1/admin/settings/sor-registration` | `SystemConfigHandler.UpdateSORRegistration` | `GLOBAL` | Global EPR registration policy update action. |
| `GET /api/v1/admin/settings/epr-registration` | `SystemConfigHandler.GetSORRegistration` | `GLOBAL` | Global EPR registration policy settings. |
| `PUT /api/v1/admin/settings/epr-registration` | `SystemConfigHandler.UpdateSORRegistration` | `GLOBAL` | Global EPR registration policy update action. |
| `GET /api/v1/admin/settings/release-governance-policy` | `SystemConfigHandler.GetReleaseGovernancePolicy` | `GLOBAL` | Global release governance policy settings. |
| `PUT /api/v1/admin/settings/release-governance-policy` | `SystemConfigHandler.UpdateReleaseGovernancePolicy` | `GLOBAL` | Global release governance policy update action. |
| `GET /api/v1/admin/settings/robot-sre` | `SystemConfigHandler.GetRobotSREPolicy` | `GLOBAL` | SRE Smart Bot global policy and channel-provider settings. |
| `GET /api/v1/admin/settings/robot-sre/defaults` | `SystemConfigHandler.GetRobotSREPolicyDefaults` | `GLOBAL` | SRE Smart Bot global default policy template and bootstrap values for admin UX initialization. |
| `PUT /api/v1/admin/settings/robot-sre` | `SystemConfigHandler.UpdateRobotSREPolicy` | `GLOBAL` | SRE Smart Bot global policy update action. |
| `GET /api/v1/admin/sre/demo/scenarios` | `SRESmartBotHandler.ListDemoScenarios` | `GLOBAL` | Demo catalog for synthetic incident generation in admin-only SRE workflows. |
| `POST /api/v1/admin/sre/demo/incidents` | `SRESmartBotHandler.GenerateDemoIncident` | `GLOBAL` | Admin-only synthetic incident creation for demos and operator training. |
| `GET /api/v1/admin/sre/remediation-packs` | `SRESmartBotHandler.ListRemediationPacks` | `GLOBAL` | Lists available guided remediation packs for the current runtime contract. |
| `GET /api/v1/admin/sre/incidents` | `SRESmartBotHandler.ListIncidents` | `DONE` | Incident ledger list with tenant scoping by auth context unless all-tenants scope is explicitly requested. |
| `GET /api/v1/admin/sre/incidents/{id}` | `SRESmartBotHandler.GetIncident` | `DONE` | Incident detail endpoint; record itself is read under `system:read` admin scope. |
| `GET /api/v1/admin/sre/incidents/{id}/remediation-packs` | `SRESmartBotHandler.ListIncidentRemediationPacks` | `DONE` | Returns remediation packs that match the selected incident type for operator action planning. |
| `GET /api/v1/admin/sre/incidents/{id}/remediation-packs/runs` | `SRESmartBotHandler.ListIncidentRemediationPackRuns` | `DONE` | Lists remediation pack dry-run and execute records for the selected incident. |
| `POST /api/v1/admin/sre/incidents/{id}/remediation-packs/{packKey}/dry-run` | `SRESmartBotHandler.DryRunIncidentRemediationPack` | `DONE` | Runs deterministic remediation-pack precondition checks and persists dry-run output for the incident. |
| `POST /api/v1/admin/sre/incidents/{id}/remediation-packs/{packKey}/execute` | `SRESmartBotHandler.ExecuteIncidentRemediationPack` | `DONE` | Records approval-gated remediation-pack execution attempts and persists execution outcome payloads. |
| `GET /api/v1/admin/sre/detector-rule-suggestions` | `SRESmartBotHandler.ListDetectorRuleSuggestions` | `GLOBAL` | Global reviewer queue for pending detector rule suggestions. |
| `POST /api/v1/admin/sre/incidents/{id}/detector-rule-suggestions` | `SRESmartBotHandler.ProposeDetectorRuleSuggestion` | `DONE` | Creates a detector rule suggestion from the selected incident context. |
| `POST /api/v1/admin/sre/detector-rule-suggestions/{suggestionId}/accept` | `SRESmartBotHandler.AcceptDetectorRuleSuggestion` | `GLOBAL` | Admin-only approval path for promoting a suggested detector rule into policy. |
| `POST /api/v1/admin/sre/detector-rule-suggestions/{suggestionId}/reject` | `SRESmartBotHandler.RejectDetectorRuleSuggestion` | `GLOBAL` | Admin-only rejection path for dismissing a suggested detector rule. |
| `GET /api/v1/admin/sre/incidents/{id}/workspace` | `SRESmartBotHandler.GetIncidentWorkspace` | `DONE` | Incident workspace snapshot with tenant-scoped incident context. |
| `GET /api/v1/admin/sre/incidents/{id}/mcp/tools` | `SRESmartBotHandler.ListMCPTools` | `DONE` | Lists incident workspace tools available for the selected incident context. |
| `POST /api/v1/admin/sre/incidents/{id}/mcp/invoke` | `SRESmartBotHandler.InvokeMCPTool` | `DONE` | Invokes an incident workspace tool against the selected incident context. |
| `POST /api/v1/admin/sre/agent/probe` | `SRESmartBotHandler.ProbeAgentRuntime` | `GLOBAL` | Global runtime probe for the SRE Smart Bot agent environment. |
| `GET /api/v1/admin/sre/incidents/{id}/agent/draft` | `SRESmartBotHandler.GetAgentDraft` | `DONE` | Returns the current draft remediation/actions generated for an incident. |
| `GET /api/v1/admin/sre/incidents/{id}/agent/interpretation` | `SRESmartBotHandler.GetAgentInterpretation` | `DONE` | Returns the latest AI interpretation for the selected incident. |
| `GET /api/v1/admin/sre/approvals` | `SRESmartBotHandler.ListApprovals` | `GLOBAL` | Admin approval queue for pending SRE Smart Bot actions. |
| `POST /api/v1/admin/sre/incidents/{id}/email-summary` | `SRESmartBotHandler.EmailIncidentSummary` | `DONE` | Sends an incident summary for the selected incident through configured channels. |
| `POST /api/v1/admin/sre/incidents/{id}/actions/{actionId}/request-approval` | `SRESmartBotHandler.RequestApproval` | `DONE` | Requests approval for an incident action generated in the selected incident context. |
| `POST /api/v1/admin/sre/incidents/{id}/actions/{actionId}/execute` | `SRESmartBotHandler.ExecuteAction` | `DONE` | Executes an approved incident action in the selected incident context. |
| `POST /api/v1/admin/sre/incidents/{id}/approvals/{approvalId}/decide` | `SRESmartBotHandler.DecideApproval` | `DONE` | Records an approval decision for an incident action request. |
| `GET /api/v1/admin/epr/registration-requests` | `EPRRegistrationHandler.ListAllRequests` | `GLOBAL` | Central reviewer/system-admin queue for EPR registration review. |
| `GET /api/v1/admin/images/import-requests` | `ImageImportHandler.ListAllImportRequests` | `GLOBAL` | Central reviewer/system-admin queue for cross-tenant quarantine import review. |
| `GET /api/v1/admin/images/import-requests/{id}/workflow` | `ImageImportHandler.GetImportRequestWorkflowAdmin` | `GLOBAL` | Central reviewer/system-admin read-only workflow progression for cross-tenant quarantine import requests. |
| `GET /api/v1/admin/images/import-requests/{id}/logs` | `ImageImportHandler.GetImportRequestLogsAdmin` | `GLOBAL` | Central reviewer/system-admin read-only request and pipeline logs for cross-tenant quarantine import requests. |
| `GET /api/v1/admin/images/import-requests/{id}/logs/stream` | `ImageImportHandler.StreamImportRequestLogsAdmin` | `GLOBAL` | Central reviewer/system-admin live log stream for cross-tenant quarantine import requests. |
| `POST /api/v1/admin/images/import-requests/{id}/approve` | `ImageImportHandler.ApproveImportRequestAdmin` | `GLOBAL` | Central reviewer/system-admin approval action for cross-tenant quarantine import requests. |
| `POST /api/v1/admin/images/import-requests/{id}/reject` | `ImageImportHandler.RejectImportRequestAdmin` | `GLOBAL` | Central reviewer/system-admin rejection action for cross-tenant quarantine import requests. |
| `POST /api/v1/admin/epr/registration-requests/{id}/approve` | `EPRRegistrationHandler.ApproveRequest` | `GLOBAL` | Central reviewer/system-admin approval action. |
| `POST /api/v1/admin/epr/registration-requests/{id}/reject` | `EPRRegistrationHandler.RejectRequest` | `GLOBAL` | Central reviewer/system-admin rejection action. |
| `POST /api/v1/admin/epr/registration-requests/{id}/suspend` | `EPRRegistrationHandler.SuspendRequest` | `GLOBAL` | Central reviewer/system-admin lifecycle suspension action. |
| `POST /api/v1/admin/epr/registration-requests/{id}/reactivate` | `EPRRegistrationHandler.ReactivateRequest` | `GLOBAL` | Central reviewer/system-admin lifecycle reactivation action. |
| `POST /api/v1/admin/epr/registration-requests/{id}/revalidate` | `EPRRegistrationHandler.RevalidateRequest` | `GLOBAL` | Central reviewer/system-admin lifecycle revalidation action. |
| `POST /api/v1/admin/epr/registration-requests/bulk/suspend` | `EPRRegistrationHandler.BulkSuspendRequests` | `GLOBAL` | Central reviewer/system-admin bulk lifecycle suspension action. |
| `POST /api/v1/admin/epr/registration-requests/bulk/reactivate` | `EPRRegistrationHandler.BulkReactivateRequests` | `GLOBAL` | Central reviewer/system-admin bulk lifecycle reactivation action. |
| `POST /api/v1/admin/epr/registration-requests/bulk/revalidate` | `EPRRegistrationHandler.BulkRevalidateRequests` | `GLOBAL` | Central reviewer/system-admin bulk lifecycle revalidation action. |
| `GET /api/v1/admin/external-services/{name}` | `SystemConfigHandler.GetExternalService` | `DONE` | Tenant scope resolver applied. |
| `PUT /api/v1/admin/external-services/{name}` | `SystemConfigHandler.UpdateExternalService` | `DONE` | Tenant scope resolver applied. |
| `DELETE /api/v1/admin/external-services/{name}` | `SystemConfigHandler.DeleteExternalService` | `DONE` | Tenant scope resolver applied. |
| `GET /api/v1/admin/external-services` | `SystemConfigHandler.GetExternalServices` | `DONE` | Tenant scope resolver + explicit all-tenants support. |
| `POST /api/v1/admin/external-services` | `SystemConfigHandler.CreateExternalService` | `DONE` | Tenant scope resolver applied. |
| `GET /api/v1/admin/ldap` | `SystemConfigHandler.GetLDAPConfigs` | `GLOBAL` | Corporate LDAP is global-only; `tenant_id`/`all_tenants` query params are rejected. |
| `POST /api/v1/admin/ldap` | `SystemConfigHandler.CreateLDAPConfig` | `GLOBAL` | Corporate LDAP is global-only; tenant-scoped writes are rejected. |
| `GET /api/v1/admin/ldap/{id}` | `SystemConfigHandler.GetLDAPConfig` | `GLOBAL` | Corporate LDAP is global-only; tenant-scoped reads are rejected. |
| `PUT /api/v1/admin/ldap/{id}` | `SystemConfigHandler.UpdateLDAPConfig` | `GLOBAL` | Corporate LDAP is global-only; tenant-scoped updates are rejected. |
| `DELETE /api/v1/admin/ldap/{id}` | `SystemConfigHandler.DeleteLDAPConfig` | `GLOBAL` | Corporate LDAP is global-only; tenant-scoped deletes are rejected. |
| `GET /api/v1/admin/stats` | `SystemStatsHandler.GetSystemStats` | `GLOBAL` | Global platform aggregates; no tenant dimension today. |
| `GET /api/v1/admin/system/components-status` | `SystemComponentsStatusHandler.GetStatus` | `GLOBAL` | Runtime/system component health endpoint. |
| `GET /api/v1/admin/dispatcher/metrics` | `DispatcherMetricsHandler.GetMetrics` | `GLOBAL` | Runtime/dispatcher process metrics. |
| `GET /api/v1/admin/dispatcher/status` | `DispatcherControlHandler.GetDispatcherStatus` | `GLOBAL` | Runtime process state. |
| `GET /api/v1/admin/execution-pipeline/health` | `ExecutionPipelineHealthHandler.GetHealth` | `GLOBAL` | Runtime pipeline health + workflow blockage diagnostics. |
| `GET /api/v1/admin/execution-pipeline/metrics` | `ExecutionPipelineMetricsHandler.GetMetrics` | `GLOBAL` | Runtime execution-pipeline counters for dashboard/alert ingestion. |
| `POST /api/v1/admin/dispatcher/start` | `DispatcherControlHandler.StartDispatcher` | `GLOBAL` | Runtime control action. |
| `POST /api/v1/admin/dispatcher/stop` | `DispatcherControlHandler.StopDispatcher` | `GLOBAL` | Runtime control action. |
| `POST /api/v1/admin/orchestrator/start` | `OrchestratorControlHandler.StartOrchestrator` | `GLOBAL` | Runtime control action for workflow orchestrator. |
| `POST /api/v1/admin/orchestrator/stop` | `OrchestratorControlHandler.StopOrchestrator` | `GLOBAL` | Runtime control action for workflow orchestrator. |
| `GET /api/v1/admin/builds/analytics` | `BuildAnalyticsHandler.GetAnalytics` | `DONE` | Tenant-dimensioned view + explicit all-tenant scope. |
| `GET /api/v1/admin/builds/performance` | `BuildAnalyticsHandler.GetPerformance` | `DONE` | Tenant-dimensioned view + explicit all-tenant scope. |
| `GET /api/v1/admin/builds/failures` | `BuildAnalyticsHandler.GetFailures` | `DONE` | Tenant-dimensioned view + explicit all-tenant scope. |
| `GET /api/v1/admin/builds/policies` | `BuildPolicyHandler.GetPolicies` | `DONE` | Uses shared tenant scope resolver. |
| `POST /api/v1/admin/builds/policies` | `BuildPolicyHandler.CreatePolicy` | `DONE` | Uses shared tenant scope resolver. |
| `GET /api/v1/admin/builds/policies/{id}` | `BuildPolicyHandler.GetPolicy` | `DONE` | Uses shared tenant scope resolver + tenant ownership check. |
| `PUT /api/v1/admin/builds/policies/{id}` | `BuildPolicyHandler.UpdatePolicy` | `DONE` | Uses shared tenant scope resolver + tenant ownership check. |
| `DELETE /api/v1/admin/builds/policies/{id}` | `BuildPolicyHandler.DeletePolicy` | `DONE` | Uses shared tenant scope resolver + tenant ownership check. |
| `GET /api/v1/admin/infrastructure/nodes` | `rest.InfrastructureHandler.GetNodes` | `DONE` | Tenant-dimensioned schema/view + explicit all-tenant scope. |
| `POST /api/v1/admin/infrastructure/nodes` | `rest.InfrastructureHandler.CreateNode` | `DONE` | Writes tenant-bound node rows. |
| `GET /api/v1/admin/infrastructure/nodes/{id}` | `rest.InfrastructureHandler.GetNode` | `DONE` | Tenant/all-tenant scope enforcement. |
| `PUT /api/v1/admin/infrastructure/nodes/{id}` | `rest.InfrastructureHandler.UpdateNode` | `DONE` | Tenant/all-tenant scope enforcement. |
| `DELETE /api/v1/admin/infrastructure/nodes/{id}` | `rest.InfrastructureHandler.DeleteNode` | `DONE` | Tenant/all-tenant scope enforcement. |
| `GET /api/v1/admin/infrastructure/health` | `rest.InfrastructureHandler.GetInfrastructureHealth` | `DONE` | Tenant-dimensioned view + explicit all-tenant scope. |
| `GET /api/v1/admin/users/check-email` | `UserHandler.CheckUserEmail` | `DONE` | Requires auth + tenant/all-tenant scope; prevents cross-tenant enumeration. |
| `GET /api/v1/admin/tenants/{tenant_id}/notification-triggers` | `ProjectNotificationTriggerHandler.GetTenantNotificationTriggers` | `DONE` | Tenant-scoped trigger default read path; system admin can target any tenant. |
| `PUT /api/v1/admin/tenants/{tenant_id}/notification-triggers` | `ProjectNotificationTriggerHandler.UpdateTenantNotificationTriggers` | `DONE` | Tenant-scoped trigger default write path; system admin can target any tenant. |
| `GET /api/v1/admin/tenants/{tenant_id}/notification-replay/status` | `BuildNotificationReplayHandler.GetTenantBuildNotificationReplayStatus` | `DONE` | Tenant-scoped replay readiness/status view for build-notification email recovery. |
| `POST /api/v1/admin/tenants/{tenant_id}/notification-replay` | `BuildNotificationReplayHandler.ReplayTenantBuildNotificationFailures` | `DONE` | Tenant-scoped replay action for failed build-notification emails with bounded batch limit. |
| `GET /api/v1/audit-events` | `AuditHandler.ListAuditEvents` | `DONE` | Tenant-scoped by default with explicit all-tenant scope for system admins. |

## Migration Coverage
- `033_build_analytics_views.up.sql`
  - Build analytics views are tenant-aware:
    - `v_build_analytics`
    - `v_build_performance_trends`
    - `v_build_failures`
    - `v_build_slowest_builds`
    - `v_build_failure_reasons`
    - `v_build_failure_rate_by_project`
- `034_infrastructure_management.up.sql`
  - `infrastructure_nodes` includes required `tenant_id`.
  - `v_infrastructure_nodes` and `v_infrastructure_health` are tenant-aware.

## Current Platform Constraint
- Tenant isolation is implemented; company isolation is not fully implemented.
- Current runtime model is effectively single-company:
  - `company_id` in tenant creation can be omitted and defaults to the authenticated admin tenant context.
  - There is no separate company-scope auth context or cross-company enforcement layer yet.
- Multi-company support is a separate future scope and should be treated as an explicit epic.

## Strict Sweep Results (2026-02-14)

Route-level hardening applied during strict checklist sweep:
- `GET /api/v1/roles` now requires `role:read` permission middleware.
- `GET /api/v1/roles/{id}` now requires `role:read` permission middleware.
- `GET /api/v1/roles/{id}/permissions` now requires `role:read` permission middleware.
- `GET /api/v1/permissions` now requires `permissions:manage` permission middleware.
- `GET /api/v1/permissions/{id}` now requires `permissions:manage` permission middleware.
- `GET /api/v1/admin/settings/tools` now requires `system:read` permission middleware.
- `GET /api/v1/admin/users/check-email` now requires `user:read` permission middleware.

Rationale:
- These routes previously relied on handler-level behavior or bare authentication.
- The strict policy is route-level enforcement for authentication + permission checks before entering handlers.
