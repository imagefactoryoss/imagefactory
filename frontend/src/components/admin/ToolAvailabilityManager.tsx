import {
    CheckCircle,
    Database,
    Key,
    RefreshCw,
    Save,
    Search,
    Settings,
    Shield,
    XCircle
} from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { toast } from 'react-hot-toast'
import { adminService } from '../../services/adminService'
import { buildService } from '../../services/buildService'
import { BuildCapabilitiesConfig, ToolAvailabilityConfig } from '../../types'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Label } from '../ui/label'
import { Separator } from '../ui/separator'
import { Switch } from '../ui/switch'

const ToolAvailabilityManager: React.FC = () => {
    const [scope, setScope] = useState<'tenant' | 'global'>('tenant')
    const [tenants, setTenants] = useState<Array<{ id: string; name: string }>>([])
    const [selectedTenantId, setSelectedTenantId] = useState<string>('')
    const [isLoadingTenants, setIsLoadingTenants] = useState(false)
    const [toolAvailability, setToolAvailability] = useState<ToolAvailabilityConfig | null>(null)
    const [buildCapabilities, setBuildCapabilities] = useState<BuildCapabilitiesConfig | null>(null)
    const [isLoading, setIsLoading] = useState(true)
    const [isSaving, setIsSaving] = useState(false)
    const [hasChanges, setHasChanges] = useState(false)

    useEffect(() => {
        loadTenants()
    }, [])

    useEffect(() => {
        if (scope === 'tenant' && !selectedTenantId) {
            return
        }
        loadToolAvailability()
    }, [scope, selectedTenantId])

    const loadTenants = async () => {
        try {
            setIsLoadingTenants(true)
            const response = await adminService.getTenants({ page: 1, limit: 200 })
            const rows = (response.data || []).map((tenant) => ({
                id: tenant.id,
                name: tenant.name,
            }))
            setTenants(rows)
            if (rows.length > 0) {
                setSelectedTenantId((current) => current || rows[0].id)
            }
        } catch (error) {
            toast.error('Failed to load tenants')
        } finally {
            setIsLoadingTenants(false)
        }
    }

    const loadToolAvailability = async () => {
        try {
            setIsLoading(true)
            const [config, capabilities] = await Promise.all([
                buildService.getToolAvailability({
                    globalDefault: scope === 'global',
                    tenantId: scope === 'tenant' ? selectedTenantId : undefined,
                }),
                buildService.getBuildCapabilities({
                    globalDefault: scope === 'global',
                    tenantId: scope === 'tenant' ? selectedTenantId : undefined,
                })
            ])
            // Ensure all categories have default values if missing
            const defaultTrivyRuntime = {
                cache_mode: 'shared' as const,
                db_repository: 'mirror.gcr.io/aquasec/trivy-db:2',
                java_db_repository: 'mirror.gcr.io/aquasec/trivy-java-db:1',
            }
            const defaultConfig: ToolAvailabilityConfig = {
                // Strict build-method semantics: omitted keys are treated as disabled.
                build_methods: { container: false, packer: false, paketo: false, kaniko: false, buildx: false, nix: false },
                sbom_tools: { syft: true, grype: true, trivy: true },
                scan_tools: { trivy: true, grype: true, clair: true, snyk: true },
                registry_types: { s3: true, harbor: true, quay: true, artifactory: true },
                secret_managers: { vault: true, aws_secretsmanager: true, gcp_secretmanager: true, azure_keyvault: true },
                trivy_runtime: defaultTrivyRuntime,
            }
            const mergedConfig: ToolAvailabilityConfig = {
                build_methods: { ...defaultConfig.build_methods, ...(config.build_methods || {}) },
                sbom_tools: { ...defaultConfig.sbom_tools, ...(config.sbom_tools || {}) },
                scan_tools: { ...defaultConfig.scan_tools, ...(config.scan_tools || {}) },
                registry_types: { ...defaultConfig.registry_types, ...(config.registry_types || {}) },
                secret_managers: { ...defaultConfig.secret_managers, ...(config.secret_managers || {}) },
                trivy_runtime: { ...defaultTrivyRuntime, ...(config.trivy_runtime || {}) },
            }
            const defaultCapabilities: BuildCapabilitiesConfig = {
                gpu: false,
                privileged: false,
                multi_arch: false,
                high_memory: false,
                host_networking: false,
                premium: false,
            }
            const mergedCapabilities: BuildCapabilitiesConfig = {
                ...defaultCapabilities,
                ...(capabilities || {}),
            }
            setToolAvailability(mergedConfig)
            setBuildCapabilities(mergedCapabilities)
            setHasChanges(false)
        } catch (error) {
            toast.error('Failed to load build policy settings')
        } finally {
            setIsLoading(false)
        }
    }

    const updateToolAvailability = (category: keyof ToolAvailabilityConfig, tool: string, enabled: boolean) => {
        if (!toolAvailability) return

        const updatedConfig = {
            ...toolAvailability,
            [category]: {
                ...toolAvailability[category],
                [tool]: enabled
            }
        }

        setToolAvailability(updatedConfig)
        setHasChanges(true)
    }

    const saveChanges = async () => {
        if (!toolAvailability || !buildCapabilities) return

        try {
            setIsSaving(true)
            await Promise.all([
                buildService.updateToolAvailability(toolAvailability, {
                    globalDefault: scope === 'global',
                    tenantId: scope === 'tenant' ? selectedTenantId : undefined,
                }),
                buildService.updateBuildCapabilities(buildCapabilities, {
                    globalDefault: scope === 'global',
                    tenantId: scope === 'tenant' ? selectedTenantId : undefined,
                }),
            ])
            setHasChanges(false)
            toast.success('Build policy settings updated successfully')
        } catch (error) {
            toast.error('Failed to save build policy settings')
        } finally {
            setIsSaving(false)
        }
    }

    const updateBuildCapability = (capability: keyof BuildCapabilitiesConfig, enabled: boolean) => {
        if (!buildCapabilities) return
        setBuildCapabilities({
            ...buildCapabilities,
            [capability]: enabled,
        })
        setHasChanges(true)
    }

    const updateTrivyRuntime = (field: 'cache_mode' | 'db_repository' | 'java_db_repository', value: string) => {
        if (!toolAvailability) return
        setToolAvailability({
            ...toolAvailability,
            trivy_runtime: {
                cache_mode: toolAvailability.trivy_runtime?.cache_mode || 'shared',
                db_repository: toolAvailability.trivy_runtime?.db_repository || 'mirror.gcr.io/aquasec/trivy-db:2',
                java_db_repository: toolAvailability.trivy_runtime?.java_db_repository || 'mirror.gcr.io/aquasec/trivy-java-db:1',
                [field]: value,
            },
        })
        setHasChanges(true)
    }

    if (isLoading) {
        return (
            <div className="flex items-center justify-center p-8 text-slate-700 dark:text-slate-200">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                <span className="ml-2">Loading tool availability...</span>
            </div>
        )
    }

    if (!toolAvailability || !buildCapabilities) {
        return (
            <div className="text-center p-8">
                <p className="text-slate-500 dark:text-slate-400">Failed to load build policy settings</p>
                <Button onClick={loadToolAvailability} className="mt-4">
                    <RefreshCw className="w-4 h-4 mr-2" />
                    Retry
                </Button>
            </div>
        )
    }

    const toolCategories = [
        {
            key: 'build_methods' as const,
            title: 'Build Methods',
            icon: Settings,
            description: 'Container and VM build methods',
            tools: [
                { key: 'container', label: 'Container', description: 'Standard Docker/container builds' },
                { key: 'packer', label: 'Packer', description: 'Infrastructure as Code builds' },
                { key: 'paketo', label: 'Paketo', description: 'Cloud Native Buildpacks' },
                { key: 'kaniko', label: 'Kaniko', description: 'Kubernetes-native container builds' },
                { key: 'buildx', label: 'Buildx', description: 'Docker Buildx for advanced builds' },
                { key: 'nix', label: 'Nix', description: 'Reproducible builds from Nix expressions' }
            ]
        },
        {
            key: 'sbom_tools' as const,
            title: 'SBOM Generation',
            icon: Shield,
            description: 'Software Bill of Materials tools',
            tools: [
                { key: 'syft', label: 'Syft', description: 'Comprehensive package detection' },
                { key: 'grype', label: 'Grype', description: 'SBOM with vulnerability matching' },
                { key: 'trivy', label: 'Trivy', description: 'SBOM generation with Trivy ecosystem' }
            ]
        },
        {
            key: 'scan_tools' as const,
            title: 'Security Scanning',
            icon: Search,
            description: 'Container security scanning tools',
            tools: [
                { key: 'trivy', label: 'Trivy', description: 'Fast, comprehensive scanning' },
                { key: 'clair', label: 'Clair', description: 'Deep container analysis' },
                { key: 'grype', label: 'Grype', description: 'Vulnerability matching with offline DB' },
                { key: 'snyk', label: 'Snyk', description: 'Cloud-based scanning with policies' }
            ]
        },
        {
            key: 'registry_types' as const,
            title: 'Container Registries',
            icon: Database,
            description: 'Supported container registry types',
            tools: [
                { key: 's3', label: 'S3 Compatible', description: 'Direct S3 object storage' },
                { key: 'harbor', label: 'Harbor', description: 'Enterprise registry with scanning' },
                { key: 'quay', label: 'Quay', description: 'Geo-replicated enterprise registry' },
                { key: 'artifactory', label: 'Artifactory', description: 'Universal artifact repository' }
            ]
        },
        {
            key: 'secret_managers' as const,
            title: 'Secret Managers',
            icon: Key,
            description: 'Secret management and retrieval',
            tools: [
                { key: 'vault', label: 'HashiCorp Vault', description: 'Enterprise secret management' },
                { key: 'aws_secretsmanager', label: 'AWS Secrets Manager', description: 'AWS native secret storage' },
                { key: 'azure_keyvault', label: 'Azure Key Vault', description: 'Azure secret management' },
                { key: 'gcp_secretmanager', label: 'GCP Secret Manager', description: 'Google Cloud secrets' }
            ]
        }
    ]
    const capabilityItems: Array<{ key: keyof BuildCapabilitiesConfig; label: string; description: string }> = [
        { key: 'gpu', label: 'GPU', description: 'Allow builds that require GPU-enabled infrastructure.' },
        { key: 'privileged', label: 'Privileged', description: 'Allow privileged build execution paths.' },
        { key: 'multi_arch', label: 'Multi-Arch', description: 'Allow multi-platform image builds.' },
        { key: 'high_memory', label: 'High Memory', description: 'Allow high-memory build workloads.' },
        { key: 'host_networking', label: 'Host Networking', description: 'Allow host-networking dependent builds.' },
        { key: 'premium', label: 'Premium', description: 'Allow premium build capability features.' },
    ]

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
                        Tool Availability Management
                    </h2>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Configure which build tools and services are available to users
                    </p>
                </div>
                <div className="flex items-center space-x-3">
                    {hasChanges && (
                        <Badge variant="secondary" className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-200">
                            Unsaved Changes
                        </Badge>
                    )}
                    <Button
                        onClick={loadToolAvailability}
                        variant="outline"
                        disabled={isLoading}
                    >
                        <RefreshCw className="w-4 h-4 mr-2" />
                        Refresh
                    </Button>
                    <Button
                        onClick={saveChanges}
                        disabled={!hasChanges || isSaving}
                    >
                        <Save className="w-4 h-4 mr-2" />
                        {isSaving ? 'Saving...' : 'Save Changes'}
                    </Button>
                </div>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Scope</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <Label htmlFor="tool-scope">Configuration Scope</Label>
                            <select
                                id="tool-scope"
                                value={scope}
                                onChange={(e) => setScope(e.target.value as 'tenant' | 'global')}
                                className="mt-2 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                            >
                                <option value="tenant">Tenant override</option>
                                <option value="global">Global default (fallback)</option>
                            </select>
                        </div>
                        {scope === 'tenant' && (
                            <div>
                                <Label htmlFor="tool-tenant">Tenant</Label>
                                <select
                                    id="tool-tenant"
                                    value={selectedTenantId}
                                    disabled={isLoadingTenants || tenants.length === 0}
                                    onChange={(e) => setSelectedTenantId(e.target.value)}
                                    className="mt-2 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                                >
                                    {tenants.length === 0 ? (
                                        <option value="">No tenants found</option>
                                    ) : null}
                                    {tenants.map((tenant) => (
                                        <option key={tenant.id} value={tenant.id}>
                                            {tenant.name}
                                        </option>
                                    ))}
                                </select>
                            </div>
                        )}
                    </div>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                        Tenant override applies to selected tenant only. Global default is used when a tenant-specific override is not present.
                    </p>
                </CardContent>
            </Card>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center space-x-2">
                            <Shield className="w-5 h-5" />
                            <span>Build Capabilities</span>
                        </CardTitle>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                            Tenant entitlement gates for advanced execution capabilities
                        </p>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        {capabilityItems.map((capability) => {
                            const isEnabled = Boolean(buildCapabilities[capability.key])
                            return (
                                <div key={capability.key} className="flex items-center justify-between p-3 border border-slate-200 rounded-lg dark:border-slate-700">
                                    <div className="flex-1">
                                        <div className="flex items-center space-x-2">
                                            <Label htmlFor={`build-capability-${capability.key}`} className="font-medium">
                                                {capability.label}
                                            </Label>
                                            {isEnabled ? (
                                                <CheckCircle className="w-4 h-4 text-green-600" />
                                            ) : (
                                                <XCircle className="w-4 h-4 text-red-600" />
                                            )}
                                        </div>
                                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                            {capability.description}
                                        </p>
                                    </div>
                                    <Switch
                                        id={`build-capability-${capability.key}`}
                                        checked={isEnabled}
                                        onCheckedChange={(checked: boolean) =>
                                            updateBuildCapability(capability.key, checked)
                                        }
                                    />
                                </div>
                            )
                        })}
                    </CardContent>
                </Card>
                {toolCategories.map((category) => {
                    const IconComponent = category.icon

                    return (
                        <Card key={category.key}>
                            <CardHeader>
                                <CardTitle className="flex items-center space-x-2">
                                    <IconComponent className="w-5 h-5" />
                                    <span>{category.title}</span>
                                </CardTitle>
                                <p className="text-sm text-gray-600 dark:text-gray-400">
                                    {category.description}
                                </p>
                            </CardHeader>
                            <CardContent className="space-y-4">
                                {category.tools.map((tool) => {
                                    const categoryData = toolAvailability[category.key]
                                    const isEnabled = categoryData ? (categoryData[tool.key as keyof typeof categoryData] as boolean) : false

                                    return (
                                        <div key={tool.key} className="flex items-center justify-between p-3 border border-slate-200 rounded-lg dark:border-slate-700">
                                            <div className="flex-1">
                                                <div className="flex items-center space-x-2">
                                                    <Label htmlFor={`${category.key}-${tool.key}`} className="font-medium">
                                                        {tool.label}
                                                    </Label>
                                                    {isEnabled ? (
                                                        <CheckCircle className="w-4 h-4 text-green-600" />
                                                    ) : (
                                                        <XCircle className="w-4 h-4 text-red-600" />
                                                    )}
                                                </div>
                                                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                                    {tool.description}
                                                </p>
                                            </div>
                                            <Switch
                                                id={`${category.key}-${tool.key}`}
                                                checked={isEnabled}
                                                onCheckedChange={(checked: boolean) =>
                                                    updateToolAvailability(category.key, tool.key, checked)
                                                }
                                            />
                                        </div>
                                    )
                                })}
                            </CardContent>
                        </Card>
                    )
                })}
            </div>

            <Separator />

            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                        <Shield className="w-5 h-5" />
                        <span>Trivy Runtime Defaults</span>
                    </CardTitle>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                        Default scan runtime settings applied when build metadata does not override Trivy values
                    </p>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div className="space-y-2">
                            <Label htmlFor="trivy-cache-mode">Cache Mode</Label>
                            <select
                                id="trivy-cache-mode"
                                value={toolAvailability.trivy_runtime?.cache_mode || 'shared'}
                                onChange={(e) => updateTrivyRuntime('cache_mode', e.target.value)}
                                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                            >
                                <option value="shared">Shared</option>
                                <option value="direct">Direct</option>
                            </select>
                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                Shared uses the pre-warmed cluster cache; Direct downloads DBs during each scan.
                            </p>
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="trivy-db-repo">Trivy DB Repository</Label>
                            <input
                                id="trivy-db-repo"
                                type="text"
                                value={toolAvailability.trivy_runtime?.db_repository || ''}
                                onChange={(e) => updateTrivyRuntime('db_repository', e.target.value)}
                                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                            />
                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                OCI reference for the main Trivy vulnerability database.
                            </p>
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="trivy-java-db-repo">Trivy Java DB Repository</Label>
                            <input
                                id="trivy-java-db-repo"
                                type="text"
                                value={toolAvailability.trivy_runtime?.java_db_repository || ''}
                                onChange={(e) => updateTrivyRuntime('java_db_repository', e.target.value)}
                                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                            />
                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                OCI reference for Trivy&apos;s Java-specific vulnerability database.
                            </p>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Summary */}
            <Card>
                <CardHeader>
                    <CardTitle>Configuration Summary</CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                        {toolCategories.map((category) => {
                            const categoryData = toolAvailability[category.key]
                            const enabledCount = Object.values(categoryData).filter(Boolean).length
                            const totalCount = Object.keys(categoryData).length

                            return (
                                <div key={category.key} className="text-center">
                                    <div className="text-2xl font-bold text-blue-600">
                                        {enabledCount}/{totalCount}
                                    </div>
                                    <div className="text-sm text-gray-600 dark:text-gray-400">
                                        {category.title}
                                    </div>
                                    <div className="text-xs text-gray-500">
                                        {enabledCount === totalCount ? 'All Enabled' :
                                            enabledCount === 0 ? 'All Disabled' :
                                                `${enabledCount} of ${totalCount} enabled`}
                                    </div>
                                </div>
                            )
                        })}
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}

export default ToolAvailabilityManager
