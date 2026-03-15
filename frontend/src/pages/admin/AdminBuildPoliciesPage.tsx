import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Clock, HardDrive, Settings, Users, RefreshCw, Plus } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { adminService } from '@/services/adminService'
import toast from 'react-hot-toast'

interface BuildPolicy {
    id: string
    tenant_id: string
    policy_type: string
    policy_key: string
    policy_value: any
    description?: string
    is_active: boolean
    created_at: string
    updated_at: string
}

const AdminBuildPoliciesPage: React.FC = () => {
    const [policies, setPolicies] = useState<BuildPolicy[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    const fetchPolicies = async () => {
        setLoading(true)
        setError(null)
        try {
            const response = await adminService.getBuildPolicies()
            setPolicies(response.policies)
        } catch (err: any) {
            setError(err.message || 'Failed to load build policies')
            toast.error('Failed to load build policies')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchPolicies()
    }, [])

    const getPolicyDisplayValue = (policy: BuildPolicy): string => {
        const value = policy.policy_value
        if (value.value !== undefined && value.unit) {
            return `${value.value} ${value.unit}`
        }
        if (value.value !== undefined) {
            return String(value.value)
        }
        if (value.data) {
            if (value.data.schedule) {
                return value.data.schedule
            }
            if (value.data.algorithm) {
                return value.data.algorithm
            }
            if (value.data.enabled !== undefined) {
                return value.data.enabled ? 'Enabled' : 'Disabled'
            }
            return JSON.stringify(value.data)
        }
        return 'N/A'
    }

    const getPolicyDescription = (policy: BuildPolicy): string => {
        const descriptions: Record<string, string> = {
            'max_build_duration': 'Limit how long a single build can run',
            'concurrent_builds_per_tenant': 'Maximum simultaneous builds allowed per tenant',
            'storage_quota_per_build': 'Maximum disk space a build can use',
            'maintenance_windows': 'Scheduled maintenance periods when builds are paused',
            'priority_queuing': 'How builds are prioritized in queue',
            'approval_required': 'Whether builds require approval before execution',
            'auto_approval_threshold': 'Criteria for automatic approval of builds',
        }
        return descriptions[policy.policy_key] || policy.description || ''
    }

    const getPolicyTitle = (policyKey: string): string => {
        const titles: Record<string, string> = {
            'max_build_duration': 'Maximum Build Duration',
            'concurrent_builds_per_tenant': 'Concurrent Builds per Tenant',
            'storage_quota_per_build': 'Storage Quota per Build',
            'maintenance_windows': 'Maintenance Windows',
            'priority_queuing': 'Priority Queuing',
            'approval_required': 'Approval Required',
            'auto_approval_threshold': 'Auto Approval Threshold',
        }
        return titles[policyKey] || policyKey
    }

    const groupedPolicies = policies.reduce((acc, policy) => {
        if (!acc[policy.policy_type]) {
            acc[policy.policy_type] = []
        }
        acc[policy.policy_type].push(policy)
        return acc
    }, {} as Record<string, BuildPolicy[]>)

    if (loading) {
        return (
            <div className="space-y-6">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                        Build Policies & Rules
                    </h1>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Configure resource limits, scheduling rules, and approval workflows for builds
                    </p>
                </div>
                <Card>
                    <CardContent className="flex items-center justify-center py-12">
                        <RefreshCw className="w-8 h-8 text-gray-400 animate-spin" />
                    </CardContent>
                </Card>
            </div>
        )
    }

    if (error) {
        return (
            <div className="space-y-6">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                        Build Policies & Rules
                    </h1>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Configure resource limits, scheduling rules, and approval workflows for builds
                    </p>
                </div>
                <Card>
                    <CardContent className="flex items-center justify-center py-12">
                        <div className="text-center">
                            <p className="text-red-600 dark:text-red-400 mb-4">{error}</p>
                            <Button onClick={fetchPolicies} variant="outline">
                                <RefreshCw className="w-4 h-4 mr-2" />
                                Retry
                            </Button>
                        </div>
                    </CardContent>
                </Card>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                        Build Policies & Rules
                    </h1>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Configure resource limits, scheduling rules, and approval workflows for builds
                    </p>
                </div>
                <Button onClick={fetchPolicies} variant="outline" size="sm">
                    <RefreshCw className="w-4 h-4 mr-2" />
                    Refresh
                </Button>
            </div>

            {/* Resource Limits */}
            {groupedPolicies['resource_limit'] && groupedPolicies['resource_limit'].length > 0 && (
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center space-x-2">
                            <HardDrive className="w-5 h-5" />
                            <span>Resource Limits</span>
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-4">
                            {groupedPolicies['resource_limit'].map((policy) => (
                                <div key={policy.id} className="flex items-center justify-between p-4 border border-gray-200 dark:border-gray-700 rounded-lg">
                                    <div>
                                        <h3 className="font-medium">{getPolicyTitle(policy.policy_key)}</h3>
                                        <p className="text-sm text-gray-600 dark:text-gray-400">{getPolicyDescription(policy)}</p>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                        <Badge variant={policy.is_active ? 'default' : 'secondary'}>
                                            {getPolicyDisplayValue(policy)}
                                        </Badge>
                                        <Button size="sm" variant="outline">
                                            <Settings className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* Scheduling Rules */}
            {groupedPolicies['scheduling_rule'] && groupedPolicies['scheduling_rule'].length > 0 && (
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center space-x-2">
                            <Clock className="w-5 h-5" />
                            <span>Scheduling Rules</span>
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-4">
                            {groupedPolicies['scheduling_rule'].map((policy) => (
                                <div key={policy.id} className="flex items-center justify-between p-4 border border-gray-200 dark:border-gray-700 rounded-lg">
                                    <div>
                                        <h3 className="font-medium">{getPolicyTitle(policy.policy_key)}</h3>
                                        <p className="text-sm text-gray-600 dark:text-gray-400">{getPolicyDescription(policy)}</p>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                        <Badge variant={policy.is_active ? 'default' : 'secondary'}>
                                            {getPolicyDisplayValue(policy)}
                                        </Badge>
                                        <Button size="sm" variant="outline">
                                            <Settings className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* Approval Workflows */}
            {groupedPolicies['approval_workflow'] && groupedPolicies['approval_workflow'].length > 0 && (
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center space-x-2">
                            <Users className="w-5 h-5" />
                            <span>Approval Workflows</span>
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-4">
                            {groupedPolicies['approval_workflow'].map((policy) => (
                                <div key={policy.id} className="flex items-center justify-between p-4 border border-gray-200 dark:border-gray-700 rounded-lg">
                                    <div>
                                        <h3 className="font-medium">{getPolicyTitle(policy.policy_key)}</h3>
                                        <p className="text-sm text-gray-600 dark:text-gray-400">{getPolicyDescription(policy)}</p>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                        <Badge variant={policy.is_active ? 'default' : 'secondary'}>
                                            {getPolicyDisplayValue(policy)}
                                        </Badge>
                                        <Button size="sm" variant="outline">
                                            <Settings className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* Empty State */}
            {policies.length === 0 && (
                <Card>
                    <CardContent className="text-center py-12">
                        <Settings className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">No Build Policies Configured</h3>
                        <p className="text-gray-600 dark:text-gray-400 mb-4">
                            Build policies have not been configured yet. Create policies to manage resource limits, scheduling rules, and approval workflows.
                        </p>
                        <Button variant="outline">
                            <Plus className="w-4 h-4 mr-2" />
                            Create First Policy
                        </Button>
                    </CardContent>
                </Card>
            )}
        </div>
    )
}

export default AdminBuildPoliciesPage
