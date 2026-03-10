import { buildService } from '@/services/buildService'
import { RegistryType, SBOMTool, ScanTool, SecretManagerType, ToolAvailabilityConfig, WizardState } from '@/types'
import React, { useEffect, useState } from 'react'

interface ToolSelectionStepProps {
    wizardState: WizardState
    onUpdate: (updates: Partial<WizardState>) => void
}

const SBOM_TOOLS = [
    {
        type: 'syft' as SBOMTool,
        title: 'Syft',
        description: 'Comprehensive package detection with SPDX/CycloneDX output',
        recommended: true
    },
    {
        type: 'grype' as SBOMTool,
        title: 'Grype',
        description: 'SBOM generation integrated with vulnerability matching',
        recommended: false
    },
    {
        type: 'trivy' as SBOMTool,
        title: 'Trivy SBOM',
        description: 'SBOM generation with Trivy ecosystem integration',
        recommended: false
    }
]

const SCAN_TOOLS = [
    {
        type: 'trivy' as ScanTool,
        title: 'Trivy',
        description: 'Fast, comprehensive scanning with broad OS/package support',
        recommended: true
    },
    {
        type: 'clair' as ScanTool,
        title: 'Clair',
        description: 'Deep container analysis with database-driven matching',
        recommended: false
    },
    {
        type: 'grype' as ScanTool,
        title: 'Grype',
        description: 'Vulnerability matching with offline database support',
        recommended: false
    },
    {
        type: 'snyk' as ScanTool,
        title: 'Snyk',
        description: 'Cloud-based scanning with policy and compliance features',
        recommended: false
    }
]

const REGISTRY_BACKENDS = [
    {
        type: 's3' as RegistryType,
        title: 'S3',
        description: 'Direct S3-compatible object storage for simple deployments',
        recommended: true
    },
    {
        type: 'harbor' as RegistryType,
        title: 'Harbor',
        description: 'Full-featured registry with security scanning and replication',
        recommended: false
    },
    {
        type: 'quay' as RegistryType,
        title: 'Quay',
        description: 'Enterprise registry with geo-replication and vulnerability scanning',
        recommended: false
    },
    {
        type: 'artifactory' as RegistryType,
        title: 'Artifactory',
        description: 'Universal artifact repository with advanced access controls',
        recommended: false
    }
]

const SECRET_MANAGERS = [
    {
        type: 'vault' as SecretManagerType,
        title: 'HashiCorp Vault',
        description: 'Enterprise-grade secret management with multiple auth methods',
        recommended: true
    },
    {
        type: 'aws_secretsmanager' as SecretManagerType,
        title: 'AWS Secrets Manager',
        description: 'Cloud-native secret management for AWS environments',
        recommended: false
    },
    {
        type: 'azure_keyvault' as SecretManagerType,
        title: 'Azure Key Vault',
        description: 'Microsoft\'s cloud secret management service',
        recommended: false
    },
    {
        type: 'gcp_secretmanager' as SecretManagerType,
        title: 'GCP Secret Manager',
        description: 'Google\'s cloud-native secret storage',
        recommended: false
    }
]

const ToolSelectionStep: React.FC<ToolSelectionStepProps> = ({
    wizardState,
    onUpdate
}) => {
    const [toolAvailability, setToolAvailability] = useState<ToolAvailabilityConfig | null>(null)
    const [loadingAvailability, setLoadingAvailability] = useState(false)
    const [availabilityError, setAvailabilityError] = useState<string | null>(null)

    useEffect(() => {
        loadToolAvailability()
    }, [])

    const loadToolAvailability = async () => {
        try {
            setLoadingAvailability(true)
            setAvailabilityError(null)
            const availability = await buildService.getBuildToolAvailability()
            setToolAvailability(availability)
        } catch (error) {
            setToolAvailability(null)
            setAvailabilityError('Failed to load tenant tool availability. Please retry.')
        } finally {
            setLoadingAvailability(false)
        }
    }

    const handleToolSelection = (category: keyof WizardState['selectedTools'], tool: string) => {
        onUpdate({
            selectedTools: {
                ...wizardState.selectedTools,
                [category]: tool
            }
        })
    }

    const isToolAvailable = (category: string, tool: string): boolean => {
        if (!toolAvailability) return false

        switch (category) {
            case 'sbom':
                return toolAvailability.sbom_tools?.[tool as SBOMTool] ?? false
            case 'scan':
                return toolAvailability.scan_tools?.[tool as ScanTool] ?? false
            case 'registry':
                return toolAvailability.registry_types?.[tool as RegistryType] ?? false
            case 'secrets':
                return toolAvailability.secret_managers?.[tool as SecretManagerType] ?? false
            default:
                return false
        }
    }

    useEffect(() => {
        if (!toolAvailability) return

        const nextSelected = { ...wizardState.selectedTools }
        let changed = false

        const pickFirstAvailable = (
            category: 'sbom' | 'scan' | 'registry' | 'secrets',
            tools: Array<{ type: string }>
        ) => tools.find(t => isToolAvailable(category, t.type))?.type

        const ensureAvailable = (
            category: 'sbom' | 'scan' | 'registry' | 'secrets',
            tools: Array<{ type: string }>
        ) => {
            const current = nextSelected[category]
            if (!current || !isToolAvailable(category, current)) {
                const fallback = pickFirstAvailable(category, tools)
                if (current !== fallback) {
                    ;(nextSelected as any)[category] = fallback
                    changed = true
                }
            }
        }

        ensureAvailable('sbom', SBOM_TOOLS)
        ensureAvailable('scan', SCAN_TOOLS)
        ensureAvailable('registry', REGISTRY_BACKENDS)
        ensureAvailable('secrets', SECRET_MANAGERS)

        if (changed) {
            onUpdate({ selectedTools: nextSelected })
        }
    }, [toolAvailability])

    const renderToolSection = (
        title: string,
        description: string,
        tools: Array<{ type: string; title: string; description: string; recommended: boolean }>,
        category: keyof WizardState['selectedTools'],
        selectedTool?: string
    ) => (
        <div className="space-y-4">
            <div>
                <h3 className="text-lg font-medium text-slate-900 dark:text-white">
                    {title}
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                    {description}
                </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {tools.map((tool) => {
                    const available = isToolAvailable(String(category), tool.type)
                    return (
                        <div
                            key={tool.type}
                            onClick={() => available && handleToolSelection(category, tool.type)}
                            className={`relative p-4 border-2 rounded-lg cursor-pointer transition-all ${selectedTool === tool.type
                                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                                    : available
                                        ? 'border-slate-200 dark:border-slate-600 hover:border-slate-300 dark:hover:border-slate-500'
                                        : 'border-slate-200 dark:border-slate-600 opacity-50 cursor-not-allowed'
                                }`}
                        >
                            {tool.recommended && available && (
                                <div className="absolute -top-2 -right-2 bg-green-500 text-white text-xs px-2 py-1 rounded-full">
                                    Recommended
                                </div>
                            )}

                            {!available && (
                                <div className="absolute -top-2 -right-2 bg-red-500 text-white text-xs px-2 py-1 rounded-full">
                                    Disabled
                                </div>
                            )}

                            <div className="flex items-start space-x-3">
                                <input
                                    type="radio"
                                    checked={selectedTool === tool.type}
                                    onChange={() => available && handleToolSelection(category, tool.type)}
                                    disabled={!available}
                                    className="mt-1 text-blue-600 focus:ring-blue-500 disabled:opacity-50"
                                />
                                <div className="flex-1">
                                    <h4 className="text-sm font-medium text-slate-900 dark:text-white">
                                        {tool.title}
                                        {!available && (
                                            <span className="ml-2 text-xs text-red-600 dark:text-red-400">
                                                (Not Available)
                                            </span>
                                        )}
                                    </h4>
                                    <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                                        {tool.description}
                                    </p>
                                </div>
                            </div>
                        </div>
                    )
                })}
            </div>

            {!wizardState.selectedTools[category] && (
                <p className="text-sm text-red-600 dark:text-red-400">
                    Please select a {title.toLowerCase().split(' ')[0]} tool
                </p>
            )}
        </div>
    )

    return (
        <div className="space-y-8">
            {loadingAvailability && (
                <div className="text-center py-4">
                    <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
                    <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                        Loading tool availability...
                    </p>
                </div>
            )}

            {availabilityError && (
                <div className="rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    <div className="flex items-center justify-between gap-3">
                        <p className="text-sm font-medium">{availabilityError}</p>
                        <button
                            type="button"
                            onClick={loadToolAvailability}
                            className="rounded border border-amber-400 px-2 py-1 text-xs font-medium hover:bg-amber-100 dark:border-amber-600 dark:hover:bg-amber-900/40"
                        >
                            Retry
                        </button>
                    </div>
                </div>
            )}

            {/* SBOM Tools */}
            {renderToolSection(
                '🔍 SBOM Generation',
                'Generate Software Bill of Materials for supply chain security',
                SBOM_TOOLS,
                'sbom',
                wizardState.selectedTools.sbom
            )}

            {/* Security Scanners */}
            {renderToolSection(
                '🛡️ Security Scanning',
                'Scan for vulnerabilities and compliance issues',
                SCAN_TOOLS,
                'scan',
                wizardState.selectedTools.scan
            )}

            {/* Registry Backends */}
            {renderToolSection(
                '📦 Registry Backend',
                'Where to store built container images',
                REGISTRY_BACKENDS,
                'registry',
                wizardState.selectedTools.registry
            )}

            {/* Secret Managers */}
            {renderToolSection(
                '🔐 Secret Management',
                'How to handle build secrets and credentials',
                SECRET_MANAGERS,
                'secrets',
                wizardState.selectedTools.secrets
            )}

            {/* Tool Availability Note */}
            <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <div className="flex">
                    <div className="flex-shrink-0">
                        <svg className="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                            <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                        </svg>
                    </div>
                    <div className="ml-3">
                        <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200">
                            Tool Availability
                        </h3>
                        <div className="mt-2 text-sm text-blue-700 dark:text-blue-300">
                            <p>
                                Some tools may be disabled by your administrator. If a tool you need is not available,
                                contact your system administrator to enable it.
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default ToolSelectionStep
