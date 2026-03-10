import { buildService } from '@/services/buildService'
import { BuildType, Project, ToolAvailabilityConfig, WizardState } from '@/types'
import React, { useCallback, useEffect, useMemo, useState } from 'react'

interface BuildMethodStepProps {
    wizardState: WizardState
    projects: Project[]
    loadingProjects: boolean
    onUpdate: (updates: Partial<WizardState>) => void
}

const BUILD_METHODS = [
    {
        type: 'packer' as BuildType,
        title: 'Packer - Infrastructure as Code',
        description: 'Build custom system images with full control over infrastructure',
        features: [
            'Multi-platform support',
            'Advanced provisioning',
            'Infrastructure as Code'
        ],
        limitations: ['Steeper learning curve'],
        recommended: false
    },
    {
        type: 'container' as BuildType,
        title: 'Docker - Container Builds',
        description: 'Standard Dockerfile builds with broad ecosystem compatibility',
        features: [
            'Familiar Dockerfile workflow',
            'Broad registry compatibility',
            'Simple local debugging'
        ],
        limitations: [],
        recommended: true
    },
    {
        type: 'kaniko' as BuildType,
        title: 'Kaniko - Dockerfile Builds',
        description: 'Build containers from Dockerfiles in Kubernetes without privileged access',
        features: [
            'Familiar Dockerfile syntax',
            'No privileged containers required',
            'Multi-stage build support'
        ],
        limitations: [],
        recommended: true
    },
    {
        type: 'buildx' as BuildType,
        title: 'Buildx - Advanced Docker Builds',
        description: 'Next-generation Docker builds with multi-platform and advanced caching',
        features: [
            'Multi-platform builds',
            'Advanced build caching',
            'Performance optimizations'
        ],
        limitations: [],
        recommended: true
    },
    {
        type: 'paketo' as BuildType,
        title: 'Paketo Buildpacks',
        description: 'Cloud-native builds using CNCF buildpacks',
        features: [
            'No Dockerfile required',
            'Opinionated secure defaults',
            'Language auto-detection'
        ],
        limitations: ['Requires compatible app source layout'],
        recommended: false
    },
    {
        type: 'nix' as BuildType,
        title: 'Nix',
        description: 'Reproducible builds from Nix expressions or flakes',
        features: [
            'Fully declarative inputs',
            'Strong reproducibility guarantees',
            'Hermetic dependency resolution'
        ],
        limitations: ['Higher setup complexity'],
        recommended: false
    }
]

const BuildMethodStep: React.FC<BuildMethodStepProps> = ({
    wizardState,
    projects,
    loadingProjects,
    onUpdate
}) => {
    const [toolAvailability, setToolAvailability] = useState<ToolAvailabilityConfig | null>(null)
    const [loadingAvailability, setLoadingAvailability] = useState(false)
    const [availabilityError, setAvailabilityError] = useState<string | null>(null)

    useEffect(() => {
        let mounted = true
        const run = async () => {
            try {
                setLoadingAvailability(true)
                setAvailabilityError(null)
                const availability = await buildService.getBuildToolAvailability()
                if (!mounted) return
                setToolAvailability(availability)
            } catch (error) {
                if (!mounted) return
                setToolAvailability(null)
                setAvailabilityError('Failed to load build method availability. Showing default methods.')
            } finally {
                if (mounted) setLoadingAvailability(false)
            }
        }
        run()
        return () => {
            mounted = false
        }
    }, [])

    const isBuildMethodAllowed = useCallback((method: BuildType): boolean => {
        if (!toolAvailability) return true
        switch (method) {
            case 'packer':
                return toolAvailability.build_methods?.packer ?? true
            case 'paketo':
                return toolAvailability.build_methods?.paketo ?? true
            case 'kaniko':
                return toolAvailability.build_methods?.kaniko ?? true
            case 'buildx':
                return toolAvailability.build_methods?.buildx ?? true
            case 'container':
                return toolAvailability.build_methods?.container ?? true
            case 'nix':
                return toolAvailability.build_methods?.nix ?? true
            default:
                return true
        }
    }, [toolAvailability])

    const availableBuildMethods = useMemo(
        () => BUILD_METHODS.filter((method) => isBuildMethodAllowed(method.type)),
        [isBuildMethodAllowed]
    )

    useEffect(() => {
        if (availableBuildMethods.length === 0) return
        const current = wizardState.buildMethod
        if (!current || !isBuildMethodAllowed(current)) {
            onUpdate({ buildMethod: availableBuildMethods[0].type })
        }
    }, [availableBuildMethods, wizardState.buildMethod, isBuildMethodAllowed, onUpdate])

    const handleProjectChange = (projectId: string) => {
        const project = projects.find(p => p.id === projectId)
        onUpdate({ selectedProject: project })
    }

    const handleBuildMethodChange = (buildMethod: BuildType) => {
        onUpdate({ buildMethod })
    }

    const handleInputChange = (field: keyof WizardState, value: string) => {
        onUpdate({ [field]: value })
    }

    return (
        <div className="space-y-6">
            {/* Project Selection */}
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Project *
                </label>
                <select
                    value={wizardState.selectedProject?.id || ''}
                    onChange={(e) => handleProjectChange(e.target.value)}
                    disabled={loadingProjects}
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                >
                    <option value="">
                        {loadingProjects ? 'Loading projects...' : 'Select a project'}
                    </option>
                    {projects.map((project) => (
                        <option key={project.id} value={project.id}>
                            {project.name}
                        </option>
                    ))}
                </select>
                {wizardState.validationErrors.project && (
                    <p className="mt-1 text-sm text-red-600 dark:text-red-400">
                        {wizardState.validationErrors.project}
                    </p>
                )}
            </div>

            {/* Build Name */}
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Build Name *
                </label>
                <input
                    type="text"
                    value={wizardState.buildName}
                    onChange={(e) => handleInputChange('buildName', e.target.value)}
                    placeholder="e.g., web-app-production-v1.2.3"
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
                {wizardState.validationErrors.buildName && (
                    <p className="mt-1 text-sm text-red-600 dark:text-red-400">
                        {wizardState.validationErrors.buildName}
                    </p>
                )}
            </div>

            {/* Build Description */}
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Description (Optional)
                </label>
                <textarea
                    value={wizardState.buildDescription}
                    onChange={(e) => handleInputChange('buildDescription', e.target.value)}
                    placeholder="Brief description of this build..."
                    rows={3}
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
            </div>

            {/* Build Method Selection */}
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-4">
                    Build Method *
                </label>
                {loadingAvailability && (
                    <p className="mb-3 text-sm text-slate-600 dark:text-slate-400">
                        Loading build method availability...
                    </p>
                )}
                {availabilityError && (
                    <div className="mb-3 rounded-md border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20 px-3 py-2 text-sm text-amber-900 dark:text-amber-200">
                        {availabilityError}
                    </div>
                )}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {availableBuildMethods.map((method) => (
                        <div
                            key={method.type}
                            onClick={() => handleBuildMethodChange(method.type)}
                            className={`relative p-4 border-2 rounded-lg cursor-pointer transition-all ${wizardState.buildMethod === method.type
                                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                                    : 'border-slate-200 dark:border-slate-600 hover:border-slate-300 dark:hover:border-slate-500'
                                }`}
                        >
                            {method.recommended && (
                                <div className="absolute -top-2 -right-2 bg-green-500 text-white text-xs px-2 py-1 rounded-full">
                                    Recommended
                                </div>
                            )}

                            <div className="flex items-start space-x-3">
                                <input
                                    type="radio"
                                    checked={wizardState.buildMethod === method.type}
                                    onChange={() => handleBuildMethodChange(method.type)}
                                    className="mt-1 text-blue-600 focus:ring-blue-500"
                                />
                                <div className="flex-1">
                                    <h3 className="text-sm font-medium text-slate-900 dark:text-white">
                                        {method.title}
                                    </h3>
                                    <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                                        {method.description}
                                    </p>

                                    {method.features.length > 0 && (
                                        <div className="mt-3">
                                            <p className="text-xs font-medium text-green-700 dark:text-green-400 uppercase tracking-wide">
                                                Features
                                            </p>
                                            <ul className="mt-1 text-xs text-slate-600 dark:text-slate-400">
                                                {method.features.map((feature, index) => (
                                                    <li key={index} className="flex items-center">
                                                        <span className="text-green-500 mr-1">✓</span>
                                                        {feature}
                                                    </li>
                                                ))}
                                            </ul>
                                        </div>
                                    )}

                                    {method.limitations.length > 0 && (
                                        <div className="mt-2">
                                            <p className="text-xs font-medium text-orange-700 dark:text-orange-400 uppercase tracking-wide">
                                                Considerations
                                            </p>
                                            <ul className="mt-1 text-xs text-slate-600 dark:text-slate-400">
                                                {method.limitations.map((limitation, index) => (
                                                    <li key={index} className="flex items-center">
                                                        <span className="text-orange-500 mr-1">⚠️</span>
                                                        {limitation}
                                                    </li>
                                                ))}
                                            </ul>
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
                {availableBuildMethods.length === 0 && (
                    <p className="mt-2 text-sm text-red-600 dark:text-red-400">
                        No build methods are currently available for this tenant.
                    </p>
                )}
                {wizardState.validationErrors.buildMethod && (
                    <p className="mt-2 text-sm text-red-600 dark:text-red-400">
                        {wizardState.validationErrors.buildMethod}
                    </p>
                )}
            </div>
        </div>
    )
}

export default BuildMethodStep
