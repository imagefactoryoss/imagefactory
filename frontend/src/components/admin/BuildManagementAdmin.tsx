import {
    AlertCircle,
    CheckCircle,
    Clock,
    Eye,
    Filter,
    Play,
    RefreshCw,
    Search,
    Square,
    XCircle
} from 'lucide-react'
import React, { useCallback, useEffect, useState } from 'react'
import { toast } from 'react-hot-toast'
import { buildService } from '../../services/buildService'
import { Build, BuildStatus, BuildType } from '../../types'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Input } from '../ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../ui/table'
import { DispatcherMetricsPanel } from './panels'

const BuildManagementAdmin: React.FC = () => {
    const [builds, setBuilds] = useState<Build[]>([])
    const [isLoading, setIsLoading] = useState(true)
    const [searchTerm, setSearchTerm] = useState('')
    const [appliedSearchTerm, setAppliedSearchTerm] = useState('')
    const [statusFilter, setStatusFilter] = useState<BuildStatus | 'all'>('all')
    const [typeFilter, setTypeFilter] = useState<BuildType | 'all'>('all')
    const [currentPage, setCurrentPage] = useState(1)
    const [totalPages, setTotalPages] = useState(1)
    const [totalBuilds, setTotalBuilds] = useState(0)

    const loadBuilds = useCallback(async (silent = false, pageOverride?: number) => {
        try {
            if (!silent) {
                setIsLoading(true)
            }
            const params: any = {
                page: pageOverride ?? currentPage,
                limit: 20,
                all_tenants: true,
            }

            if (statusFilter !== 'all') {
                params.status = [statusFilter]
            }

            if (typeFilter !== 'all') {
                params.type = [typeFilter]
            }

            if (appliedSearchTerm) {
                params.search = appliedSearchTerm
            }

            const response = await buildService.getBuilds(params)
            setBuilds(response.data)
            setTotalBuilds(response.pagination.total)
            setTotalPages(Math.ceil(response.pagination.total / response.pagination.limit))
        } catch (error) {
            if (!silent) {
                toast.error('Failed to load builds')
            }
        } finally {
            if (!silent) {
                setIsLoading(false)
            }
        }
    }, [appliedSearchTerm, currentPage, statusFilter, typeFilter])

    useEffect(() => {
        void loadBuilds()
    }, [loadBuilds])

    useEffect(() => {
        const interval = window.setInterval(() => {
            void loadBuilds(true)
        }, 10000)
        return () => window.clearInterval(interval)
    }, [loadBuilds])

    const handleSearch = () => {
        setAppliedSearchTerm(searchTerm.trim())
        setCurrentPage(1)
        void loadBuilds(false, 1)
    }

    const handleStartBuild = async (buildId: string) => {
        try {
            await buildService.startBuild(buildId)
            toast.success('Build started successfully')
            void loadBuilds(true)
        } catch (error) {
            toast.error('Failed to start build')
        }
    }

    const handleCancelBuild = async (buildId: string) => {
        try {
            await buildService.cancelBuild(buildId)
            toast.success('Build cancelled successfully')
            void loadBuilds(true)
        } catch (error) {
            toast.error('Failed to cancel build')
        }
    }

    const getStatusIcon = (status: BuildStatus) => {
        switch (status) {
            case 'completed':
                return <CheckCircle className="w-4 h-4 text-green-600" />
            case 'failed':
                return <XCircle className="w-4 h-4 text-red-600" />
            case 'running':
                return <Clock className="w-4 h-4 text-blue-600 animate-spin" />
            case 'pending':
            case 'queued':
                return <Clock className="w-4 h-4 text-yellow-600" />
            case 'cancelled':
                return <AlertCircle className="w-4 h-4 text-gray-600" />
            default:
                return <Clock className="w-4 h-4 text-gray-600" />
        }
    }

    const getStatusBadgeVariant = (status: BuildStatus) => {
        switch (status) {
            case 'completed':
                return 'default'
            case 'failed':
                return 'destructive'
            case 'running':
                return 'secondary'
            case 'pending':
            case 'queued':
                return 'outline'
            case 'cancelled':
                return 'secondary'
            default:
                return 'outline'
        }
    }

    const formatDuration = (startedAt?: string, completedAt?: string) => {
        if (!startedAt) return '-'
        if (!completedAt) return 'In progress'

        const start = new Date(startedAt)
        const end = new Date(completedAt)
        const duration = end.getTime() - start.getTime()

        const minutes = Math.floor(duration / 60000)
        const seconds = Math.floor((duration % 60000) / 1000)

        return `${minutes}m ${seconds}s`
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
                        Build Management
                    </h2>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Monitor and manage all builds across the system
                    </p>
                </div>
                <Button onClick={() => void loadBuilds()} disabled={isLoading}>
                    <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                    Refresh
                </Button>
            </div>

            <DispatcherMetricsPanel />

            {/* Filters */}
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                        <Filter className="w-5 h-5" />
                        <span>Filters</span>
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Search
                            </label>
                            <div className="flex space-x-2">
                                <Input
                                    placeholder="Search builds..."
                                    value={searchTerm}
                                    onChange={(e) => setSearchTerm(e.target.value)}
                                    onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                                />
                                <Button onClick={handleSearch} size="sm">
                                    <Search className="w-4 h-4" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Status
                            </label>
                            <Select value={statusFilter} onValueChange={(value) => setStatusFilter(value as BuildStatus | 'all')}>
                                <SelectTrigger>
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="all">All Statuses</SelectItem>
                                    <SelectItem value="pending">Pending</SelectItem>
                                    <SelectItem value="queued">Queued</SelectItem>
                                    <SelectItem value="running">Running</SelectItem>
                                    <SelectItem value="completed">Completed</SelectItem>
                                    <SelectItem value="failed">Failed</SelectItem>
                                    <SelectItem value="cancelled">Cancelled</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Build Type
                            </label>
                            <Select value={typeFilter} onValueChange={(value) => setTypeFilter(value as BuildType | 'all')}>
                                <SelectTrigger>
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="all">All Types</SelectItem>
                                    <SelectItem value="container">Container</SelectItem>
                                    <SelectItem value="vm">VM</SelectItem>
                                    <SelectItem value="cloud">Cloud</SelectItem>
                                    <SelectItem value="packer">Packer</SelectItem>
                                    <SelectItem value="paketo">Paketo</SelectItem>
                                    <SelectItem value="kaniko">Kaniko</SelectItem>
                                    <SelectItem value="buildx">Buildx</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>

                        <div className="flex items-end">
                            <Button
                                onClick={() => {
                                    setSearchTerm('')
                                    setAppliedSearchTerm('')
                                    setStatusFilter('all')
                                    setTypeFilter('all')
                                    setCurrentPage(1)
                                    void loadBuilds(false, 1)
                                }}
                                variant="outline"
                            >
                                Clear Filters
                            </Button>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Builds Table */}
            <Card>
                <CardHeader>
                    <CardTitle>Builds ({totalBuilds})</CardTitle>
                </CardHeader>
                <CardContent>
                    {isLoading ? (
                        <div className="flex items-center justify-center p-8">
                            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                            <span className="ml-2">Loading builds...</span>
                        </div>
                    ) : builds.length === 0 ? (
                        <div className="text-center p-8">
                            <p className="text-gray-500">No builds found</p>
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <Table>
                                <TableHeader>
                                    <TableRow>
                                        <TableHead>Build</TableHead>
                                        <TableHead>Type</TableHead>
                                        <TableHead>Status</TableHead>
                                        <TableHead>Tenant</TableHead>
                                        <TableHead>Duration</TableHead>
                                        <TableHead>Created</TableHead>
                                        <TableHead>Actions</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {builds.map((build) => (
                                        <TableRow key={build.id}>
                                            <TableCell>
                                                <div>
                                                    <div className="font-medium">{build.manifest.name}</div>
                                                    <div className="text-sm text-gray-500">{build.id.slice(0, 8)}</div>
                                                </div>
                                            </TableCell>
                                            <TableCell>
                                                <Badge variant="outline">
                                                    {build.manifest.type}
                                                </Badge>
                                            </TableCell>
                                            <TableCell>
                                                <div className="flex items-center space-x-2">
                                                    {getStatusIcon(build.status)}
                                                    <Badge variant={getStatusBadgeVariant(build.status)}>
                                                        {build.status}
                                                    </Badge>
                                                </div>
                                            </TableCell>
                                            <TableCell>
                                                <code className="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                                                    {build.tenantId.slice(0, 8)}
                                                </code>
                                            </TableCell>
                                            <TableCell>
                                                {formatDuration(build.startedAt, build.completedAt)}
                                            </TableCell>
                                            <TableCell>
                                                {new Date(build.createdAt).toLocaleDateString()}
                                            </TableCell>
                                            <TableCell>
                                                <div className="flex items-center space-x-2">
                                                    <Button
                                                        size="sm"
                                                        variant="outline"
                                                        onClick={() => {/* TODO: View build details */ }}
                                                    >
                                                        <Eye className="w-4 h-4" />
                                                    </Button>
                                                    {(build.status === 'pending' || build.status === 'queued') && (
                                                        <Button
                                                            size="sm"
                                                            onClick={() => handleStartBuild(build.id)}
                                                        >
                                                            <Play className="w-4 h-4" />
                                                        </Button>
                                                    )}
                                                    {build.status === 'running' && (
                                                        <Button
                                                            size="sm"
                                                            variant="destructive"
                                                            onClick={() => handleCancelBuild(build.id)}
                                                        >
                                                            <Square className="w-4 h-4" />
                                                        </Button>
                                                    )}
                                                </div>
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </div>
                    )}

                    {/* Pagination */}
                    {totalPages > 1 && (
                        <div className="flex items-center justify-between mt-4">
                            <div className="text-sm text-gray-500">
                                Page {currentPage} of {totalPages}
                            </div>
                            <div className="flex space-x-2">
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                                    disabled={currentPage === 1}
                                >
                                    Previous
                                </Button>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                                    disabled={currentPage === totalPages}
                                >
                                    Next
                                </Button>
                            </div>
                        </div>
                    )}
                </CardContent>
            </Card>
        </div>
    )
}

export default BuildManagementAdmin
