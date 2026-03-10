import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { infrastructureService } from '@/services/infrastructureService'
import { InfrastructureProvider, InfrastructureRecommendation, InfrastructureType } from '@/types'
import { AlertCircle, CheckCircle, Info, Loader2, Server, Zap } from 'lucide-react'
import React, { useEffect, useState } from 'react'

interface InfrastructureSelectorProps {
    recommendation: InfrastructureRecommendation | null
    value: InfrastructureType | ''
    onChange: (value: InfrastructureType) => void
    selectedProviderId?: string | null
    onProviderChange?: (value: string | null) => void
    disabled?: boolean
}

const InfrastructureSelector: React.FC<InfrastructureSelectorProps> = ({
    recommendation,
    value,
    onChange,
    selectedProviderId,
    onProviderChange,
    disabled = false
}) => {
    const [availableProviders, setAvailableProviders] = useState<InfrastructureProvider[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        loadAvailableProviders()
    }, [])

    const loadAvailableProviders = async () => {
        try {
            setLoading(true)
            setError(null)
            const providers = await infrastructureService.getAvailableOptions()
            setAvailableProviders(providers)
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load infrastructure options')
        } finally {
            setLoading(false)
        }
    }

    const getInfrastructureIcon = (type: InfrastructureType) => {
        switch (type) {
            case 'kubernetes':
            case 'aws-eks':
            case 'gcp-gke':
            case 'azure-aks':
            case 'oci-oke':
            case 'vmware-vks':
            case 'openshift':
            case 'rancher':
                return <Zap className="h-4 w-4" />
            case 'build_nodes':
                return <Server className="h-4 w-4" />
            default:
                return <Info className="h-4 w-4" />
        }
    }

    const isSelectableProvider = (provider: InfrastructureProvider) =>
        provider.status === 'online' && provider.is_schedulable

    const isTektonEnabledProvider = (provider: InfrastructureProvider) =>
        provider.config?.tekton_enabled === true

    const isBuildReadyProvider = (provider: InfrastructureProvider) =>
        isSelectableProvider(provider) && isTektonEnabledProvider(provider)

    const getInfrastructureLabel = (type: InfrastructureType) => {
        switch (type) {
            case 'kubernetes':
                return 'Kubernetes Cluster'
            case 'aws-eks':
                return 'AWS EKS'
            case 'gcp-gke':
                return 'GCP GKE'
            case 'azure-aks':
                return 'Azure AKS'
            case 'oci-oke':
                return 'OCI OKE'
            case 'vmware-vks':
                return 'VMware vKS'
            case 'openshift':
                return 'OpenShift'
            case 'rancher':
                return 'Rancher'
            case 'build_nodes':
                return 'Build Nodes'
            default:
                return type
        }
    }

    const getInfrastructureDescription = (type: InfrastructureType) => {
        switch (type) {
            case 'kubernetes':
                return 'Scalable container orchestration with advanced networking and storage capabilities'
            case 'aws-eks':
                return 'Managed Kubernetes on AWS with IAM integration'
            case 'gcp-gke':
                return 'Managed Kubernetes on Google Cloud with workload identity support'
            case 'azure-aks':
                return 'Managed Kubernetes on Azure with managed identity integration'
            case 'oci-oke':
                return 'Managed Kubernetes on Oracle Cloud Infrastructure'
            case 'vmware-vks':
                return 'VMware-managed Kubernetes for vSphere environments'
            case 'openshift':
                return 'OpenShift clusters with enterprise Kubernetes features'
            case 'rancher':
                return 'Rancher-managed Kubernetes clusters'
            case 'build_nodes':
                return 'Dedicated build servers optimized for consistent performance and cost efficiency'
            default:
                return ''
        }
    }

    const getConfidenceColor = (confidence: number) => {
        if (confidence >= 0.8) return 'text-green-600'
        if (confidence >= 0.6) return 'text-yellow-600'
        return 'text-red-600'
    }

    const getConfidenceBadge = (confidence: number) => {
        if (confidence >= 0.8) return <Badge variant="default" className="bg-green-100 text-green-800">High Confidence</Badge>
        if (confidence >= 0.6) return <Badge variant="default" className="bg-yellow-100 text-yellow-800">Medium Confidence</Badge>
        return <Badge variant="default" className="bg-red-100 text-red-800">Low Confidence</Badge>
    }

    // Get available infrastructure types from providers
    const getAvailableTypes = (): InfrastructureType[] => {
        const types = new Set<InfrastructureType>()
        availableProviders.forEach(provider => {
            if (isBuildReadyProvider(provider)) {
                types.add(provider.provider_type)
            }
        })
        return Array.from(types)
    }

    const availableTypes = getAvailableTypes()
    const hasOnlineProviders = availableTypes.length > 0
    const selectedProviders = availableProviders.filter(
        provider => provider.provider_type === value && isBuildReadyProvider(provider)
    )
    const blockedProvidersForSelectedType = availableProviders.filter(
        provider => provider.provider_type === value && provider.status === 'online' && !provider.is_schedulable
    )
    const tektonDisabledProvidersForSelectedType = availableProviders.filter(
        provider => provider.provider_type === value && isSelectableProvider(provider) && !isTektonEnabledProvider(provider)
    )

    const shouldShowProviderSelect = Boolean(value) && selectedProviders.length > 0

    return (
        <div className="space-y-6">
            {/* Error Display */}
            {error && (
                <Card className="border-red-200 bg-red-50 dark:bg-red-900/20">
                    <CardContent className="pt-6">
                        <div className="flex items-center">
                            <AlertCircle className="h-5 w-5 text-red-600 mr-2" />
                            <span className="text-red-800 dark:text-red-200">{error}</span>
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* Recommendation Display */}
            {recommendation && hasOnlineProviders && (
                <Card>
                    <CardHeader className="pb-3">
                        <CardTitle className="flex items-center gap-2 text-lg">
                            <CheckCircle className="h-5 w-5 text-green-600" />
                            Infrastructure Recommendation
                        </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                                {getInfrastructureIcon(recommendation.recommended_infrastructure)}
                                <span className="font-medium">
                                    {getInfrastructureLabel(recommendation.recommended_infrastructure)}
                                </span>
                            </div>
                            {getConfidenceBadge(recommendation.confidence)}
                        </div>

                        <p className="text-sm text-gray-600 dark:text-gray-400">
                            {recommendation.reason}
                        </p>

                        <div className="text-xs text-gray-500">
                            Confidence: <span className={getConfidenceColor(recommendation.confidence)}>
                                {Math.round(recommendation.confidence * 100)}%
                            </span>
                        </div>

                        {/* Alternatives */}
                        {recommendation.alternatives && recommendation.alternatives.length > 0 && (
                            <div className="border-t pt-3">
                                <h4 className="text-sm font-medium mb-2">Alternative Options:</h4>
                                <div className="space-y-2">
                                    {recommendation.alternatives.map((alt, index) => (
                                        <div key={index} className="flex items-center justify-between text-sm">
                                            <div className="flex items-center gap-2">
                                                {getInfrastructureIcon(alt.infrastructure)}
                                                <span>{getInfrastructureLabel(alt.infrastructure)}</span>
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <span className="text-xs text-gray-500">
                                                    {Math.round(alt.confidence * 100)}%
                                                </span>
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => onChange(alt.infrastructure)}
                                                    disabled={disabled}
                                                >
                                                    Use This
                                                </Button>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}
                    </CardContent>
                </Card>
            )}

            {/* Infrastructure Selection */}
            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-lg">Infrastructure Selection</CardTitle>
                </CardHeader>
                <CardContent>
                    {loading ? (
                        <div className="flex items-center justify-center py-8">
                            <Loader2 className="h-6 w-6 animate-spin text-gray-400 mr-2" />
                            <span className="text-gray-500">Loading available infrastructure...</span>
                        </div>
                    ) : !hasOnlineProviders ? (
                        <div className="text-center py-8">
                            <AlertCircle className="h-12 w-12 mx-auto text-yellow-400 mb-4" />
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                                No Build-Ready Infrastructure Available
                            </h3>
                            <p className="text-gray-600 dark:text-gray-400 mb-4">
                                No providers are currently ready for Kubernetes builds. Providers must be online, schedulable, and have Tekton enabled.
                            </p>
                            <Button
                                onClick={loadAvailableProviders}
                                variant="outline"
                                disabled={loading}
                            >
                                <Loader2 className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                                Retry
                            </Button>
                        </div>
                    ) : (
                        <>
                            <div className="mb-4">
                                <p className="text-sm text-gray-600 dark:text-gray-400">
                                    Choose how your build will be executed. Infrastructure options are configured by administrators.
                                </p>
                                <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                                    Provider labels show the configured platform (EKS, GKE, AKS, OpenShift, Rancher, etc.).
                                </p>
                            </div>

                            <RadioGroup
                                value={value}
                                onValueChange={onChange}
                                disabled={disabled}
                                className="space-y-4"
                            >
                                {/* Available provider types */}
                                {availableTypes.map((type) => {
                                    const providersOfType = availableProviders.filter(
                                        p => p.provider_type === type && isSelectableProvider(p)
                                    )
                                    const providerNames = providersOfType.map(p => p.display_name).join(', ')

                                    return (
                                        <div key={type} className="flex items-start space-x-3">
                                            <RadioGroupItem value={type} id={type} className="mt-1" />
                                            <div className="flex-1">
                                                <Label
                                                    htmlFor={type}
                                                    className="flex items-center gap-2 cursor-pointer font-medium"
                                                >
                                                    {getInfrastructureIcon(type)}
                                                    {getInfrastructureLabel(type)}
                                                    {recommendation?.recommended_infrastructure === type && (
                                                        <Badge variant="secondary" className="ml-2">
                                                            Recommended
                                                        </Badge>
                                                    )}
                                                </Label>
                                                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                                    {getInfrastructureDescription(type)}
                                                </p>
                                                {providersOfType.length > 0 && (
                                                    <p className="text-xs text-gray-500 mt-1">
                                                        Available providers: {providerNames}
                                                    </p>
                                                )}
                                            </div>
                                        </div>
                                    )
                                })}
                            </RadioGroup>

                            {shouldShowProviderSelect && (
                                <div className="mt-4 space-y-2">
                                    <Label className="text-sm text-gray-700 dark:text-gray-300">
                                        Provider Selection
                                    </Label>
                                    {selectedProviders.length <= 1 ? (
                                        <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-3 py-2 text-sm text-gray-700 dark:text-gray-200">
                                            {selectedProviders.length === 1
                                                ? selectedProviders[0].display_name
                                                : 'No provider available for this infrastructure type.'}
                                        </div>
                                    ) : (
                                        <select
                                            value={selectedProviderId || ''}
                                            onChange={(event) => onProviderChange?.(event.target.value || null)}
                                            className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                                            disabled={disabled}
                                        >
                                            {selectedProviders.map(provider => (
                                                <option key={provider.id} value={provider.id}>
                                                    {provider.display_name}
                                                </option>
                                            ))}
                                        </select>
                                    )}
                                    <p className="text-xs text-gray-500 dark:text-gray-400">
                                        Choose which configured provider to use for this build.
                                    </p>
                                </div>
                            )}
                            {blockedProvidersForSelectedType.length > 0 && (
                                <div className="mt-3 rounded-md border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20 p-3">
                                    <div className="flex items-start gap-2">
                                        <AlertCircle className="h-4 w-4 mt-0.5 text-amber-700 dark:text-amber-300" />
                                        <div className="space-y-1">
                                            <p className="text-xs font-semibold text-amber-800 dark:text-amber-200">
                                                Some providers are online but not schedulable
                                            </p>
                                            {blockedProvidersForSelectedType.map((provider) => (
                                                <p key={provider.id} className="text-xs text-amber-800 dark:text-amber-200">
                                                    {provider.display_name}: {provider.schedulable_reason || 'Provider is currently blocked by readiness or policy gates.'}
                                                </p>
                                            ))}
                                        </div>
                                    </div>
                                </div>
                            )}
                            {tektonDisabledProvidersForSelectedType.length > 0 && (
                                <div className="mt-3 rounded-md border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20 p-3">
                                    <div className="flex items-start gap-2">
                                        <AlertCircle className="h-4 w-4 mt-0.5 text-amber-700 dark:text-amber-300" />
                                        <div className="space-y-1">
                                            <p className="text-xs font-semibold text-amber-800 dark:text-amber-200">
                                                Some providers are excluded because Tekton is not enabled
                                            </p>
                                            {tektonDisabledProvidersForSelectedType.map((provider) => (
                                                <p key={provider.id} className="text-xs text-amber-800 dark:text-amber-200">
                                                    {provider.display_name}: configure <code>tekton_enabled=true</code> before selecting for builds.
                                                </p>
                                            ))}
                                        </div>
                                    </div>
                                </div>
                            )}

                            {value && availableTypes.includes(value) && (
                                <div className="mt-4 p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                                    <div className="flex items-start gap-2">
                                        <CheckCircle className="h-4 w-4 text-green-600 mt-0.5" />
                                        <div>
                                            <p className="text-sm font-medium text-green-800 dark:text-green-200">
                                                Infrastructure Selected
                                            </p>
                                            <p className="text-sm text-green-700 dark:text-green-300 mt-1">
                                                Your build will run on {getInfrastructureLabel(value).toLowerCase()}.
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            )}
                        </>
                    )}
                </CardContent>
            </Card>
        </div>
    )
}

export default InfrastructureSelector
