import {
    AlertTriangle,
    ArrowLeft,
    Calendar,
    ChevronDown,
    ChevronRight,
    CheckCircle2,
    Clock,
    Copy,
    Download,
    Edit,
    Eye,
    EyeOff,
    GitBranch,
    Layers,
    Package,
    Plus,
    Settings,
    Shield,
    Star,
    Tag,
    Trash2,
    X,
} from 'lucide-react'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { useCapabilitySurfacesStore } from '@/store/capabilitySurfaces'
import React, { useEffect, useMemo, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { adminService } from '../../services/adminService'
import { imageService } from '../../services/imageService'
import { useTenantStore } from '../../store/tenant'
import type { ImageDetailsResponse, ImageVersion } from '../../types'

type OwnerTenantInfo = {
    name?: string
    contactEmail?: string
    industry?: string
    country?: string
}

const ImageDetailPage: React.FC = () => {
    const { imageId } = useParams<{ imageId: string }>()
    const navigate = useNavigate()
    const location = useLocation()

    const [details, setDetails] = useState<ImageDetailsResponse | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [activeTab, setActiveTab] = useState<'overview' | 'versions' | 'tags' | 'security' | 'settings'>('overview')
    const [newTag, setNewTag] = useState('')
    const [tagBusy, setTagBusy] = useState(false)
    const [sbomQuery, setSbomQuery] = useState('')
    const [sbomTypeFilter, setSbomTypeFilter] = useState('all')
    const [sbomSort, setSbomSort] = useState<'vuln' | 'name'>('vuln')
    const [sbomOnlyVulnerable, setSbomOnlyVulnerable] = useState(false)
    const [expandedSbomPackages, setExpandedSbomPackages] = useState<Record<string, boolean>>({})
    const [expandedLayers, setExpandedLayers] = useState<Record<string, boolean>>({})
    const [resolvedOwnerTenant, setResolvedOwnerTenant] = useState<OwnerTenantInfo | null>(null)
    const [scanActionBusy, setScanActionBusy] = useState(false)
    const [scanActionMessage, setScanActionMessage] = useState<string | null>(null)
    const [scanActionError, setScanActionError] = useState<string | null>(null)
    const { userTenants, selectedTenantId } = useTenantStore()
    const canRunOnDemandScanAction = useCapabilitySurfacesStore((state) => state.canRunActionKey('images.scan.ondemand'))
    const confirmDialog = useConfirmDialog()

    const isAdminView = location.pathname.startsWith('/admin')
    const basePath = isAdminView ? '/admin/images' : '/images'

    useEffect(() => {
        if (imageId) {
            void loadImageData()
        }
    }, [imageId])

    const loadImageData = async () => {
        if (!imageId) return

        try {
            setLoading(true)
            setError(null)
            const data = await imageService.getImageDetails(imageId)
            setDetails(data)
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load image data')
        } finally {
            setLoading(false)
        }
    }

    const image = details?.image
    const versions = details?.versions ?? []

    const mergedTags = useMemo(() => details?.tags?.merged ?? image?.tags ?? [], [details, image])
    const latestScan = useMemo(
        () => (details?.vulnerability_scans && details.vulnerability_scans.length > 0 ? details.vulnerability_scans[0] : null),
        [details]
    )
    const totalLayerBytes = useMemo(
        () => (details?.layers || []).reduce((sum, layer) => sum + (layer.layer_size_bytes || 0), 0),
        [details]
    )
    const baseLayerCount = useMemo(
        () => (details?.layers || []).filter((layer) => layer.is_base_layer).length,
        [details]
    )
    const mappedLayerPackagesCount = useMemo(
        () => (details?.layers || []).reduce((sum, layer) => sum + (layer.package_count || layer.packages?.length || 0), 0),
        [details]
    )
    const mappedLayerVulnerabilitiesCount = useMemo(
        () => (details?.layers || []).reduce((sum, layer) => sum + (layer.vulnerability_count || layer.vulnerabilities?.length || 0), 0),
        [details]
    )
    const sbomPackageTypes = useMemo(() => {
        const types = new Set<string>()
        ;(details?.sbom?.packages || []).forEach((pkg) => {
            if (pkg.package_type) types.add(pkg.package_type)
        })
        return ['all', ...Array.from(types).sort()]
    }, [details])
    const filteredSbomPackages = useMemo(() => {
        const packages = [...(details?.sbom?.packages || [])]
            .filter((pkg) => (sbomTypeFilter === 'all' ? true : pkg.package_type === sbomTypeFilter))
            .filter((pkg) => pkg.package_name.toLowerCase().includes(sbomQuery.toLowerCase()))
            .filter((pkg) => (sbomOnlyVulnerable ? (pkg.known_vulnerabilities_count || 0) > 0 : true))

        if (sbomSort === 'vuln') {
            packages.sort((a, b) => (b.known_vulnerabilities_count || 0) - (a.known_vulnerabilities_count || 0))
        } else {
            packages.sort((a, b) => a.package_name.localeCompare(b.package_name))
        }
        return packages
    }, [details, sbomQuery, sbomTypeFilter, sbomSort, sbomOnlyVulnerable])
    const ownerTenant = useMemo(
        () => userTenants.find((tenant) => tenant.id === image?.tenant_id) || null,
        [userTenants, image?.tenant_id]
    )
    useEffect(() => {
        const tenantID = image?.tenant_id
        if (!tenantID) {
            setResolvedOwnerTenant(null)
            return
        }

        if (ownerTenant?.name) {
            setResolvedOwnerTenant({
                name: ownerTenant.name,
            })
        }

        let cancelled = false
        const resolveTenantName = async () => {
            try {
                const tenant = await adminService.getTenantById(tenantID)
                if (!cancelled) {
                    setResolvedOwnerTenant(tenant || null)
                }
            } catch {
                if (!cancelled && !ownerTenant?.name) {
                    setResolvedOwnerTenant(null)
                }
            }
        }

        void resolveTenantName()
        return () => {
            cancelled = true
        }
    }, [image?.tenant_id, ownerTenant?.name])
    const ownerTenantLabel = useMemo(() => {
        if (resolvedOwnerTenant?.name) return resolvedOwnerTenant.name
        if (ownerTenant?.name) return ownerTenant.name
        if (!image?.tenant_id) return 'N/A'
        if (selectedTenantId && image.tenant_id === selectedTenantId) return 'Current tenant'
        return 'External tenant'
    }, [resolvedOwnerTenant?.name, ownerTenant?.name, image?.tenant_id, selectedTenantId])
    const canManageImage = useMemo(
        () => Boolean(selectedTenantId && image?.tenant_id && selectedTenantId === image.tenant_id),
        [selectedTenantId, image?.tenant_id]
    )
    const canTriggerOnDemandScan = canRunOnDemandScanAction
    const evidenceStates = useMemo(() => {
        const metadata = details?.metadata
        return {
            layers: metadata?.layers_evidence_status || 'unavailable',
            sbom: metadata?.sbom_evidence_status || 'unavailable',
            vulnerability: metadata?.vulnerability_evidence_status || 'unavailable',
        } as const
    }, [details?.metadata])
    const staleEvidenceItems = useMemo(() => {
        const items: string[] = []
        if (evidenceStates.layers === 'stale') items.push('layers')
        if (evidenceStates.sbom === 'stale') items.push('SBOM')
        if (evidenceStates.vulnerability === 'stale') items.push('vulnerability scan')
        return items
    }, [evidenceStates])

    const copyToClipboard = async (value: string) => {
        try {
            await navigator.clipboard.writeText(value)
        } catch (e) {
            console.error('Clipboard write failed', e)
        }
    }

    const truncateDigest = (value?: string) => {
        const digest = (value || '').trim()
        if (!digest) return 'N/A'
        if (digest.length <= 20) return digest
        return `${digest.slice(0, 14)}...${digest.slice(-8)}`
    }

    const formatBytes = (bytes?: number) => {
        if (!bytes || bytes <= 0) return 'N/A'
        const mb = bytes / 1024 / 1024
        if (mb < 1) return `${(bytes / 1024).toFixed(1)} KB`
        if (mb < 1024) return `${mb.toFixed(2)} MB`
        return `${(mb / 1024).toFixed(2)} GB`
    }

    const timeAgo = (iso?: string) => {
        if (!iso) return 'N/A'
        const then = new Date(iso).getTime()
        const now = Date.now()
        const diffMins = Math.max(0, Math.floor((now - then) / 60000))
        if (diffMins < 1) return 'just now'
        if (diffMins < 60) return `${diffMins}m ago`
        const diffHrs = Math.floor(diffMins / 60)
        if (diffHrs < 24) return `${diffHrs}h ago`
        return `${Math.floor(diffHrs / 24)}d ago`
    }

    const sbomPackageKey = (pkg: { package_name: string; package_version?: string }) =>
        `${pkg.package_name}-${pkg.package_version || ''}`

    const evidenceBadge = (label: string, status?: string, updatedAt?: string) => {
        const normalized = status || 'unavailable'
        const classes = normalized === 'fresh'
            ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-200'
            : normalized === 'stale'
                ? 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200'
                : 'border-slate-300 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300'
        return (
            <div className={`rounded-md border px-2 py-1 text-xs ${classes}`}>
                <span className="font-semibold">{label}</span>: {normalized}
                {updatedAt ? <span className="ml-1 opacity-80">({new Date(updatedAt).toLocaleString()})</span> : null}
            </div>
        )
    }

    const toggleSbomPackageExpanded = (key: string) => {
        setExpandedSbomPackages((prev) => ({ ...prev, [key]: !prev[key] }))
    }
    const toggleLayerExpanded = (digest: string) => {
        setExpandedLayers((prev) => ({ ...prev, [digest]: !prev[digest] }))
    }

    const handleDelete = async () => {
        if (!image) {
            return
        }
        const confirmed = await confirmDialog({
            title: 'Delete Image',
            message: 'Are you sure you want to delete this image? This action cannot be undone.',
            confirmLabel: 'Delete Image',
            destructive: true,
        })
        if (!confirmed) {
            return
        }

        try {
            await imageService.deleteImage(image.id)
            navigate(basePath)
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to delete image')
        }
    }

    const handleAddTag = async () => {
        if (!imageId || !newTag.trim()) return
        try {
            setTagBusy(true)
            await imageService.addImageTags(imageId, [newTag.trim()])
            setNewTag('')
            await loadImageData()
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to add tag')
        } finally {
            setTagBusy(false)
        }
    }

    const handleRemoveTag = async (tag: string) => {
        if (!imageId) return
        try {
            setTagBusy(true)
            await imageService.removeImageTags(imageId, [tag])
            await loadImageData()
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to remove tag')
        } finally {
            setTagBusy(false)
        }
    }

    const handleTriggerOnDemandScan = async () => {
        if (!imageId) return
        setScanActionBusy(true)
        setScanActionMessage(null)
        setScanActionError(null)
        try {
            const response = await imageService.triggerOnDemandScan(imageId)
            setScanActionMessage(response.message || 'On-demand scan queued successfully.')
        } catch (err: any) {
            const code = err?.response?.data?.error?.code as string | undefined
            if (code === 'tenant_capability_not_entitled') {
                setScanActionError('On-demand image scanning is not entitled for this tenant.')
            } else if (code === 'not_found') {
                setScanActionError('Image was not found for the current tenant context. Refresh and retry.')
            } else {
                setScanActionError(err instanceof Error ? err.message : 'Failed to trigger on-demand scan.')
            }
        } finally {
            setScanActionBusy(false)
        }
    }

    const getVisibilityIcon = (visibility: string) => {
        switch (visibility) {
            case 'public':
                return <Eye className="w-5 h-5 text-green-500" />
            case 'tenant':
                return <Eye className="w-5 h-5 text-blue-500" />
            case 'private':
                return <EyeOff className="w-5 h-5 text-gray-500" />
            default:
                return null
        }
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'published':
                return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300'
            case 'draft':
                return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300'
            case 'deprecated':
                return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300'
            case 'archived':
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
            default:
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
        }
    }

    const renderVersionRow = (version: ImageVersion) => (
        <div key={version.id} className="border border-slate-200 dark:border-slate-700 rounded-lg p-4">
            <div className="flex items-center justify-between gap-4">
                <div>
                    <h4 className="text-sm font-medium text-slate-900 dark:text-white">Version {version.version}</h4>
                    {version.description && (
                        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{version.description}</p>
                    )}
                    {version.tags?.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-2">
                            {version.tags.map((tag) => (
                                <span
                                    key={`${version.id}-${tag}`}
                                    className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200"
                                >
                                    {tag}
                                </span>
                            ))}
                        </div>
                    )}
                </div>
                <div className="text-right">
                    <div className="text-xs text-slate-500 dark:text-slate-400">
                        {version.published_at ? new Date(version.published_at).toLocaleString() : 'N/A'}
                    </div>
                    <button className="mt-2 inline-flex items-center px-3 py-1 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-800 hover:bg-slate-50 dark:hover:bg-slate-700">
                        <Download className="w-4 h-4 mr-1" />
                        Pull
                    </button>
                </div>
            </div>
            {version.digest && <div className="mt-2 text-xs font-mono text-slate-500 dark:text-slate-400">{version.digest}</div>}
        </div>
    )

    if (loading) {
        return (
            <div className="flex justify-center items-center min-h-96">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        )
    }

    if (error || !details || !image) {
        return (
            <div className="px-4 py-6 sm:px-6 lg:px-8">
                <div className="text-center">
                    <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Image Not Found</h1>
                    <p className="mt-2 text-slate-600 dark:text-slate-400">{error || 'The requested image could not be found.'}</p>
                    <Link to={basePath} className="mt-4 inline-flex items-center text-blue-600 hover:text-blue-500">
                        <ArrowLeft className="w-4 h-4 mr-2" />
                        Back to Images
                    </Link>
                </div>
            </div>
        )
    }

    return (
        <div className="px-4 py-6 sm:px-6 lg:px-8 space-y-6">
            <div className="flex items-center justify-between">
                <div className="flex items-center space-x-4">
                    <Link
                        to={basePath}
                        className="inline-flex items-center text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100"
                    >
                        <ArrowLeft className="w-5 h-5 mr-2" />
                        Back to Images
                    </Link>
                    <div className="flex items-center space-x-2">
                        <Package className="w-6 h-6 text-slate-400" />
                        <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">{image.name}</h1>
                    </div>
                </div>
                <div className="flex items-center space-x-2">
                    {canManageImage ? (
                        <Link
                            to={`${basePath}/${image.id}/edit`}
                            className="inline-flex items-center px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-800 hover:bg-slate-50 dark:hover:bg-slate-700"
                        >
                            <Edit className="w-4 h-4 mr-2" />
                            Edit
                        </Link>
                    ) : (
                        <button
                            type="button"
                            disabled
                            title="Switch to the owner tenant context to edit this image."
                            className="inline-flex items-center px-4 py-2 border border-slate-200 dark:border-slate-700 rounded-md text-sm font-medium text-slate-400 dark:text-slate-500 bg-slate-50 dark:bg-slate-900 cursor-not-allowed"
                        >
                            <Edit className="w-4 h-4 mr-2" />
                            Edit
                        </button>
                    )}
                    <button
                        onClick={handleDelete}
                        disabled={!canManageImage}
                        title={canManageImage ? 'Delete image' : 'Switch to the owner tenant context to delete this image.'}
                        className={`inline-flex items-center px-4 py-2 border rounded-md text-sm font-medium ${
                            canManageImage
                                ? 'border-red-300 dark:border-red-600 text-red-700 dark:text-red-300 bg-white dark:bg-slate-800 hover:bg-red-50 dark:hover:bg-red-900'
                                : 'border-slate-200 dark:border-slate-700 text-slate-400 dark:text-slate-500 bg-slate-50 dark:bg-slate-900 cursor-not-allowed'
                        }`}
                    >
                        <Trash2 className="w-4 h-4 mr-2" />
                        Delete
                    </button>
                </div>
            </div>
            {!canManageImage && (
                <div className="rounded-md border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20 px-3 py-2 text-xs text-amber-800 dark:text-amber-300">
                    This image belongs to another tenant. Editing is disabled in the current tenant context.
                </div>
            )}

            <div className="flex flex-wrap items-center gap-4">
                <div className="flex items-center gap-2">
                    {getVisibilityIcon(image.visibility)}
                    <span className="text-sm text-slate-600 dark:text-slate-400 capitalize">{image.visibility}</span>
                </div>
                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(image.status)}`}>
                    {image.status}
                </span>
                <div className="flex items-center text-sm text-slate-500 dark:text-slate-400">
                    <Star className="w-4 h-4 mr-1" />
                    {details.stats.pull_count} pulls
                </div>
                <div className="flex items-center text-sm text-slate-500 dark:text-slate-400">
                    <Clock className="w-4 h-4 mr-1" />
                    Updated {new Date(details.stats.last_updated || image.updated_at).toLocaleDateString()}
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-5 gap-3">
                <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-3">
                    <div className="text-xs text-slate-500 dark:text-slate-400">Latest Scan</div>
                    <div className="mt-1 flex items-center gap-2 text-sm font-medium">
                        {latestScan?.pass_fail_result === 'PASS' ? (
                            <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                        ) : (
                            <AlertTriangle className="w-4 h-4 text-amber-500" />
                        )}
                        <span className="text-slate-900 dark:text-white">{latestScan?.pass_fail_result || latestScan?.scan_status || 'N/A'}</span>
                    </div>
                </div>
                <div className="rounded-lg border border-rose-200 dark:border-rose-900 bg-rose-50/70 dark:bg-rose-950/20 p-3">
                    <div className="text-xs text-rose-600 dark:text-rose-300">Critical</div>
                    <div className="mt-1 text-lg font-semibold text-rose-700 dark:text-rose-200">{latestScan?.vulnerabilities_critical ?? 0}</div>
                </div>
                <div className="rounded-lg border border-orange-200 dark:border-orange-900 bg-orange-50/70 dark:bg-orange-950/20 p-3">
                    <div className="text-xs text-orange-600 dark:text-orange-300">High</div>
                    <div className="mt-1 text-lg font-semibold text-orange-700 dark:text-orange-200">{latestScan?.vulnerabilities_high ?? 0}</div>
                </div>
                <div className="rounded-lg border border-amber-200 dark:border-amber-900 bg-amber-50/70 dark:bg-amber-950/20 p-3">
                    <div className="text-xs text-amber-700 dark:text-amber-300">Medium</div>
                    <div className="mt-1 text-lg font-semibold text-amber-800 dark:text-amber-200">{latestScan?.vulnerabilities_medium ?? 0}</div>
                </div>
                <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-3">
                    <div className="text-xs text-slate-500 dark:text-slate-400">Scan Age</div>
                    <div className="mt-1 text-sm font-medium text-slate-900 dark:text-white">
                        {timeAgo(latestScan?.completed_at || latestScan?.started_at)}
                    </div>
                </div>
            </div>

            <div className="border-b border-slate-200 dark:border-slate-700">
                <nav className="-mb-px flex space-x-8">
                    {[
                        { id: 'overview', label: 'Overview', icon: Package },
                        { id: 'versions', label: 'Versions', icon: GitBranch },
                        { id: 'tags', label: 'Tags', icon: Tag },
                        { id: 'security', label: 'Security', icon: Shield },
                        { id: 'settings', label: 'Settings', icon: Settings },
                    ].map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id as typeof activeTab)}
                            className={`flex items-center py-2 px-1 border-b-2 font-medium text-sm ${
                                activeTab === tab.id
                                    ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                    : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300'
                            }`}
                        >
                            <tab.icon className="w-4 h-4 mr-2" />
                            {tab.label}
                        </button>
                    ))}
                </nav>
            </div>

            {activeTab === 'overview' && (
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    <div className="lg:col-span-2 space-y-6">
                        <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-2">Description</h3>
                            <p className="text-slate-600 dark:text-slate-400">{image.description || 'No description provided.'}</p>
                        </div>

                        {details.metadata && (
                            <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                                <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4">Catalog Metadata</h3>
                                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                                    <div>
                                        <dt className="text-slate-500 dark:text-slate-400">Digest</dt>
                                        <dd className="font-mono text-slate-900 dark:text-white">
                                            {details.metadata.docker_manifest_digest ? (
                                                <div className="inline-flex items-center gap-2">
                                                    <span title={details.metadata.docker_manifest_digest}>
                                                        {truncateDigest(details.metadata.docker_manifest_digest)}
                                                    </span>
                                                    <button
                                                        type="button"
                                                        onClick={() => copyToClipboard(details.metadata?.docker_manifest_digest || '')}
                                                        title="Copy digest"
                                                        className="inline-flex items-center rounded-md border border-slate-300 dark:border-slate-600 p-1 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                                    >
                                                        <Copy className="w-3.5 h-3.5" />
                                                    </button>
                                                </div>
                                            ) : (
                                                'N/A'
                                            )}
                                        </dd>
                                    </div>
                                    <div>
                                        <dt className="text-slate-500 dark:text-slate-400">Compressed Size</dt>
                                        <dd className="text-slate-900 dark:text-white">{details.metadata.compressed_size_bytes ? `${(details.metadata.compressed_size_bytes / 1024 / 1024).toFixed(2)} MB` : 'N/A'}</dd>
                                    </div>
                                    <div>
                                        <dt className="text-slate-500 dark:text-slate-400">Packages</dt>
                                        <dd className="text-slate-900 dark:text-white">{details.metadata.packages_count ?? 0}</dd>
                                    </div>
                                    <div>
                                        <dt className="text-slate-500 dark:text-slate-400">Scan Tool</dt>
                                        <dd className="text-slate-900 dark:text-white">{details.metadata.scan_tool || 'N/A'}</dd>
                                    </div>
                                </div>
                                <div className="mt-4 space-y-2">
                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Evidence Freshness</div>
                                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                                        {evidenceBadge('Layers', details.metadata.layers_evidence_status, details.metadata.layers_evidence_updated_at)}
                                        {evidenceBadge('SBOM', details.metadata.sbom_evidence_status, details.metadata.sbom_evidence_updated_at)}
                                        {evidenceBadge('Vuln', details.metadata.vulnerability_evidence_status, details.metadata.vulnerability_evidence_updated_at)}
                                    </div>
                                    {staleEvidenceItems.length > 0 && (
                                        <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
                                            Showing stale {staleEvidenceItems.join(', ')} data from an earlier build because the latest successful push did not publish fresh evidence.
                                        </div>
                                    )}
                                </div>
                            </div>
                        )}

                        <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4">Layers</h3>
                            <div className="grid grid-cols-1 sm:grid-cols-5 gap-3 mb-4">
                                <div className="rounded-md border border-slate-200 dark:border-slate-700 p-3">
                                    <div className="text-xs text-slate-500 dark:text-slate-400">Total Layers</div>
                                    <div className="text-lg font-semibold text-slate-900 dark:text-white">{details.layers.length}</div>
                                </div>
                                <div className="rounded-md border border-slate-200 dark:border-slate-700 p-3">
                                    <div className="text-xs text-slate-500 dark:text-slate-400">Base Layers</div>
                                    <div className="text-lg font-semibold text-slate-900 dark:text-white">{baseLayerCount}</div>
                                </div>
                                <div className="rounded-md border border-slate-200 dark:border-slate-700 p-3">
                                    <div className="text-xs text-slate-500 dark:text-slate-400">Total Size</div>
                                    <div className="text-lg font-semibold text-slate-900 dark:text-white">{formatBytes(totalLayerBytes)}</div>
                                </div>
                                <div className="rounded-md border border-slate-200 dark:border-slate-700 p-3">
                                    <div className="text-xs text-slate-500 dark:text-slate-400">Mapped Packages</div>
                                    <div className="text-lg font-semibold text-slate-900 dark:text-white">{mappedLayerPackagesCount}</div>
                                </div>
                                <div className="rounded-md border border-slate-200 dark:border-slate-700 p-3">
                                    <div className="text-xs text-slate-500 dark:text-slate-400">Mapped Vulnerabilities</div>
                                    <div className="text-lg font-semibold text-slate-900 dark:text-white">{mappedLayerVulnerabilitiesCount}</div>
                                </div>
                            </div>
                            {details.layers.length === 0 ? (
                                <p className="text-sm text-slate-500 dark:text-slate-400">No layer data available.</p>
                            ) : (
                                <div className="max-h-80 overflow-auto rounded-md border border-slate-200 dark:border-slate-700">
                                    <div className="grid grid-cols-12 gap-2 px-3 py-2 text-xs font-medium text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-700">
                                        <div className="col-span-1"></div>
                                        <div className="col-span-1">#</div>
                                        <div className="col-span-6">Digest</div>
                                        <div className="col-span-2">Size</div>
                                        <div className="col-span-2">Evidence</div>
                                    </div>
                                    {details.layers.map((layer) => (
                                        <div key={layer.layer_digest} className="border-b border-slate-100 dark:border-slate-700 text-sm">
                                            <div className="grid grid-cols-12 gap-2 px-3 py-2 items-center">
                                                <div className="col-span-1">
                                                    <button
                                                        type="button"
                                                        onClick={() => toggleLayerExpanded(layer.layer_digest)}
                                                        className="inline-flex items-center rounded border border-slate-300 dark:border-slate-600 p-0.5 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                                        title={expandedLayers[layer.layer_digest] ? 'Collapse layer evidence' : 'Expand layer evidence'}
                                                    >
                                                        {expandedLayers[layer.layer_digest] ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                                                    </button>
                                                </div>
                                                <div className="col-span-1 text-slate-700 dark:text-slate-200">{layer.layer_number}</div>
                                                <div className="col-span-6 flex items-center gap-2 min-w-0">
                                                    <Layers className="w-4 h-4 text-slate-400 shrink-0" />
                                                    <span className="font-mono text-xs text-slate-600 dark:text-slate-300 truncate">
                                                        {layer.layer_digest}
                                                    </span>
                                                    <button
                                                        onClick={() => copyToClipboard(layer.layer_digest)}
                                                        className="text-slate-500 hover:text-slate-800 dark:hover:text-slate-200"
                                                        title="Copy digest"
                                                    >
                                                        <Copy className="w-3 h-3" />
                                                    </button>
                                                </div>
                                                <div className="col-span-2 text-slate-700 dark:text-slate-200">
                                                    {formatBytes(layer.layer_size_bytes)}
                                                </div>
                                                <div className="col-span-2 flex items-center gap-2">
                                                    {layer.is_base_layer ? (
                                                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200">
                                                            base
                                                        </span>
                                                    ) : (
                                                        <span className="text-xs text-slate-500 dark:text-slate-400">{layer.media_type || 'layer'}</span>
                                                    )}
                                                    <span className="text-[11px] text-slate-500 dark:text-slate-400">
                                                        {layer.package_count || layer.packages?.length || 0} pkg / {layer.vulnerability_count || layer.vulnerabilities?.length || 0} vuln
                                                    </span>
                                                </div>
                                            </div>
                                            {expandedLayers[layer.layer_digest] && (
                                                <div className="px-3 pb-3">
                                                    <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 p-3 space-y-3">
                                                        <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-xs text-slate-600 dark:text-slate-300">
                                                            <div>Command: <span className="font-mono text-slate-800 dark:text-slate-100">{layer.source_command || 'N/A'}</span></div>
                                                            <div>History: <span className="font-mono text-slate-800 dark:text-slate-100">{layer.history_created_by || 'N/A'}</span></div>
                                                        </div>
                                                        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                                                            <div className="rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-2">
                                                                <div className="text-xs font-medium text-slate-700 dark:text-slate-200 mb-1">Packages</div>
                                                                {(layer.packages || []).length === 0 ? (
                                                                    <div className="text-xs text-slate-500 dark:text-slate-400">No per-layer package mapping.</div>
                                                                ) : (
                                                                    <div className="max-h-32 overflow-auto space-y-1">
                                                                        {(layer.packages || []).slice(0, 20).map((pkg) => (
                                                                            <div key={`${layer.layer_digest}-${pkg.package_name}-${pkg.package_version || ''}`} className="text-xs text-slate-600 dark:text-slate-300">
                                                                                <span className="font-medium text-slate-800 dark:text-slate-100">{pkg.package_name}</span>
                                                                                {pkg.package_version ? `@${pkg.package_version}` : ''} ({pkg.known_vulnerabilities_count || 0} vuln)
                                                                            </div>
                                                                        ))}
                                                                    </div>
                                                                )}
                                                            </div>
                                                            <div className="rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-2">
                                                                <div className="text-xs font-medium text-slate-700 dark:text-slate-200 mb-1">Mapped Vulnerabilities</div>
                                                                {(layer.vulnerabilities || []).length === 0 ? (
                                                                    <div className="text-xs text-slate-500 dark:text-slate-400">No per-layer vulnerabilities mapped.</div>
                                                                ) : (
                                                                    <div className="max-h-32 overflow-auto space-y-1">
                                                                        {(layer.vulnerabilities || []).slice(0, 20).map((vuln) => (
                                                                            <div key={`${layer.layer_digest}-${vuln.cve_id}-${vuln.package_name || ''}`} className="text-xs text-slate-600 dark:text-slate-300">
                                                                                <span className="font-mono text-slate-800 dark:text-slate-100">{vuln.cve_id}</span> ({vuln.severity || 'UNKNOWN'})
                                                                                {vuln.package_name ? ` - ${vuln.package_name}` : ''}
                                                                            </div>
                                                                        ))}
                                                                    </div>
                                                                )}
                                                            </div>
                                                        </div>
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>

                    <div className="space-y-6">
                        <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4">Statistics</h3>
                            <div className="space-y-3 text-sm">
                                <div className="flex justify-between">
                                    <span className="text-slate-500 dark:text-slate-400">Versions</span>
                                    <span className="font-medium text-slate-900 dark:text-white">{details.stats.version_count}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-slate-500 dark:text-slate-400">Layers</span>
                                    <span className="font-medium text-slate-900 dark:text-white">{details.stats.layer_count}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-slate-500 dark:text-slate-400">Scans</span>
                                    <span className="font-medium text-slate-900 dark:text-white">{details.stats.vulnerability_scan_count}</span>
                                </div>
                            </div>
                        </div>

                        <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4">Ownership</h3>
                            <div className="space-y-3 text-sm">
                                <div className="flex justify-between items-start gap-3">
                                    <span className="text-slate-500 dark:text-slate-400">Owner Tenant</span>
                                    <span className="font-medium text-slate-900 dark:text-white text-right">
                                        {ownerTenantLabel}
                                    </span>
                                </div>
                                <div className="flex justify-between items-start gap-3">
                                    <span className="text-slate-500 dark:text-slate-400">Contact</span>
                                    <span className="font-medium text-slate-900 dark:text-white text-right break-all">
                                        {resolvedOwnerTenant?.contactEmail || 'Not configured'}
                                    </span>
                                </div>
                                {(resolvedOwnerTenant?.industry || resolvedOwnerTenant?.country) && (
                                    <div className="flex justify-between items-start gap-3">
                                        <span className="text-slate-500 dark:text-slate-400">Profile</span>
                                        <span className="font-medium text-slate-900 dark:text-white text-right">
                                            {[resolvedOwnerTenant.industry, resolvedOwnerTenant.country].filter(Boolean).join(' • ')}
                                        </span>
                                    </div>
                                )}
                            </div>
                        </div>

                        <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4">Timeline</h3>
                            <div className="space-y-3 text-sm">
                                <div className="flex items-center">
                                    <Calendar className="w-4 h-4 mr-2 text-slate-400" />
                                    <span className="text-slate-500 dark:text-slate-400">Created:</span>
                                    <span className="ml-2 text-slate-900 dark:text-white">{new Date(image.created_at).toLocaleDateString()}</span>
                                </div>
                                <div className="flex items-center">
                                    <Clock className="w-4 h-4 mr-2 text-slate-400" />
                                    <span className="text-slate-500 dark:text-slate-400">Updated:</span>
                                    <span className="ml-2 text-slate-900 dark:text-white">{new Date(image.updated_at).toLocaleDateString()}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {activeTab === 'versions' && (
                <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                    <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4">Version History</h3>
                    {versions.length === 0 ? (
                        <p className="text-sm text-slate-500 dark:text-slate-400">No versions found.</p>
                    ) : (
                        <div className="space-y-4">{versions.map(renderVersionRow)}</div>
                    )}
                </div>
            )}

            {activeTab === 'tags' && (
                <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6 space-y-6">
                    <div>
                        <h3 className="text-lg font-medium text-slate-900 dark:text-white">Tags</h3>
                        <p className="text-sm text-slate-500 dark:text-slate-400">
                            Merged view combines inline image tags and normalized catalog tags.
                        </p>
                    </div>

                    <div className="flex gap-2">
                        <input
                            value={newTag}
                            onChange={(e) => setNewTag(e.target.value)}
                            placeholder="Add tag"
                            className="flex-1 rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-sm"
                        />
                        <button
                            onClick={handleAddTag}
                            disabled={!canManageImage || tagBusy || !newTag.trim()}
                            title={canManageImage ? 'Add tag' : 'Switch to the owner tenant context to modify tags.'}
                            className="inline-flex items-center px-3 py-2 text-sm rounded-md bg-blue-600 text-white disabled:opacity-50"
                        >
                            <Plus className="w-4 h-4 mr-1" />
                            Add
                        </button>
                    </div>

                    <div className="space-y-3">
                        <h4 className="text-sm font-medium text-slate-900 dark:text-white">Merged</h4>
                        <div className="flex flex-wrap gap-2">
                            {mergedTags.map((tag) => (
                                <span
                                    key={`merged-${tag}`}
                                    className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200"
                                >
                                    {tag}
                                    <button
                                        onClick={() => handleRemoveTag(tag)}
                                        disabled={!canManageImage}
                                        title={canManageImage ? 'Remove tag' : 'Switch to the owner tenant context to modify tags.'}
                                        className="ml-2 text-blue-700 hover:text-blue-900 dark:text-blue-300 dark:hover:text-blue-100"
                                    >
                                        <X className="w-3 h-3" />
                                    </button>
                                </span>
                            ))}
                            {mergedTags.length === 0 && (
                                <p className="text-sm text-slate-500 dark:text-slate-400">No tags assigned.</p>
                            )}
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                        <div>
                            <h4 className="font-medium text-slate-900 dark:text-white mb-2">Inline Tags</h4>
                            <p className="text-slate-600 dark:text-slate-400">{(details.tags.inline || []).join(', ') || 'None'}</p>
                        </div>
                        <div>
                            <h4 className="font-medium text-slate-900 dark:text-white mb-2">Normalized Tags</h4>
                            <p className="text-slate-600 dark:text-slate-400">
                                {(details.tags.normalized || []).map((t) => (t.category ? `${t.tag} (${t.category})` : t.tag)).join(', ') || 'None'}
                            </p>
                        </div>
                    </div>
                </div>
            )}

            {activeTab === 'security' && (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {staleEvidenceItems.length > 0 && (
                        <div className="lg:col-span-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
                            <div className="flex items-start gap-2">
                                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                                <span>Security evidence is stale for {staleEvidenceItems.join(', ')}. Latest successful build push did not publish fresh evidence.</span>
                            </div>
                        </div>
                    )}
                    <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                        <div className="mb-4 flex items-start justify-between gap-3">
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white">Vulnerability Scans</h3>
                            {canTriggerOnDemandScan ? (
                                <button
                                    type="button"
                                    onClick={handleTriggerOnDemandScan}
                                    disabled={!canManageImage || scanActionBusy}
                                    title={canManageImage ? 'Run on-demand scan now' : 'Switch to owner tenant context to run an on-demand scan.'}
                                    className="inline-flex items-center rounded-md border border-blue-300 dark:border-blue-700 bg-blue-50 dark:bg-blue-900/30 px-3 py-1.5 text-xs font-medium text-blue-700 dark:text-blue-200 hover:bg-blue-100 dark:hover:bg-blue-900/50 disabled:cursor-not-allowed disabled:opacity-60"
                                >
                                    {scanActionBusy ? 'Queueing...' : 'Run On-Demand Scan'}
                                </button>
                            ) : (
                                <span className="inline-flex items-center rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 px-2 py-1 text-[11px] text-slate-600 dark:text-slate-300">
                                    On-demand scanning not entitled
                                </span>
                            )}
                        </div>
                        {scanActionMessage ? (
                            <div className="mb-3 rounded border border-emerald-200 dark:border-emerald-800 bg-emerald-50 dark:bg-emerald-900/30 px-3 py-2 text-xs text-emerald-800 dark:text-emerald-200">
                                {scanActionMessage}
                            </div>
                        ) : null}
                        {scanActionError ? (
                            <div className="mb-3 rounded border border-rose-200 dark:border-rose-800 bg-rose-50 dark:bg-rose-900/30 px-3 py-2 text-xs text-rose-800 dark:text-rose-200">
                                {scanActionError}
                            </div>
                        ) : null}
                        {details.vulnerability_scans.length === 0 ? (
                            <p className="text-sm text-slate-500 dark:text-slate-400">No scans recorded.</p>
                        ) : (
                            <div className="space-y-3">
                                {details.vulnerability_scans.map((scan) => (
                                    <div key={scan.id} className="border border-slate-200 dark:border-slate-700 rounded-md p-3">
                                        <div className="flex items-center justify-between">
                                            <span className="text-sm font-medium text-slate-900 dark:text-white">{scan.scan_tool}</span>
                                            <span className="text-xs text-slate-500 dark:text-slate-400">{scan.scan_status}</span>
                                        </div>
                                        <div className="mt-2 text-xs text-slate-600 dark:text-slate-300">
                                            High: {scan.vulnerabilities_high ?? 0} | Medium: {scan.vulnerabilities_medium ?? 0} | Low: {scan.vulnerabilities_low ?? 0}
                                        </div>
                                        {scan.error_message ? (
                                            <div className="mt-2 text-xs rounded border border-rose-200 bg-rose-50 px-2 py-1 text-rose-700 dark:border-rose-700 dark:bg-rose-900/20 dark:text-rose-300">
                                                {scan.error_message}
                                            </div>
                                        ) : null}
                                        {(scan.build_id || scan.scan_report_location) ? (
                                            <div className="mt-2 flex items-center gap-3 text-xs">
                                                {scan.build_id ? (
                                                    <Link
                                                        to={`/builds/${scan.build_id}`}
                                                        className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 underline"
                                                    >
                                                        View build logs
                                                    </Link>
                                                ) : null}
                                                {scan.scan_report_location ? (
                                                    <a
                                                        href={scan.scan_report_location}
                                                        target="_blank"
                                                        rel="noreferrer"
                                                        className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 underline"
                                                    >
                                                        Open scan report
                                                    </a>
                                                ) : null}
                                            </div>
                                        ) : null}
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                        <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4">SBOM</h3>
                        {!details.sbom ? (
                            <p className="text-sm text-slate-500 dark:text-slate-400">No SBOM data available.</p>
                        ) : (
                            <div className="space-y-3">
                                <div className="grid grid-cols-2 gap-2 text-xs">
                                    <div className="rounded-md bg-slate-50 dark:bg-slate-900 p-2">Format: <span className="font-semibold">{details.sbom.format}</span></div>
                                    <div className="rounded-md bg-slate-50 dark:bg-slate-900 p-2">Status: <span className="font-semibold">{details.sbom.status || 'unknown'}</span></div>
                                    <div className="rounded-md bg-slate-50 dark:bg-slate-900 p-2">Tool: <span className="font-semibold">{details.sbom.generated_by_tool || 'N/A'}</span></div>
                                    <div className="rounded-md bg-slate-50 dark:bg-slate-900 p-2">Packages: <span className="font-semibold">{details.sbom.packages.length}</span></div>
                                </div>
                                <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                                    <input
                                        value={sbomQuery}
                                        onChange={(e) => setSbomQuery(e.target.value)}
                                        placeholder="Search package"
                                        className="sm:col-span-2 rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-1.5 text-sm"
                                    />
                                    <select
                                        value={sbomTypeFilter}
                                        onChange={(e) => setSbomTypeFilter(e.target.value)}
                                        className="rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-1.5 text-sm"
                                    >
                                        {sbomPackageTypes.map((type) => (
                                            <option key={type} value={type}>
                                                {type}
                                            </option>
                                        ))}
                                    </select>
                                </div>
                                <div className="flex items-center justify-end">
                                    <div className="flex items-center gap-2">
                                        <button
                                            onClick={() => setSbomOnlyVulnerable((prev) => !prev)}
                                            className={`text-xs px-2 py-1 rounded border ${sbomOnlyVulnerable
                                                    ? 'border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
                                                    : 'border-slate-300 dark:border-slate-600 text-slate-600 dark:text-slate-300'
                                                }`}
                                        >
                                            {sbomOnlyVulnerable ? 'Showing vulnerable only' : 'Show vulnerable only'}
                                        </button>
                                        <button
                                            onClick={() => setSbomSort((prev) => (prev === 'vuln' ? 'name' : 'vuln'))}
                                            className="text-xs px-2 py-1 rounded border border-slate-300 dark:border-slate-600 text-slate-600 dark:text-slate-300"
                                        >
                                            Sort: {sbomSort === 'vuln' ? 'vulnerabilities' : 'name'}
                                        </button>
                                    </div>
                                </div>
                                <div className="max-h-64 overflow-auto border border-slate-200 dark:border-slate-700 rounded-md">
                                    {filteredSbomPackages.map((pkg) => (
                                        <div key={`${pkg.package_name}-${pkg.package_version || ''}`} className="px-3 py-2 text-xs border-b border-slate-100 dark:border-slate-700">
                                            <div className="flex items-center justify-between gap-2">
                                                <div className="font-medium text-slate-900 dark:text-white">{pkg.package_name}</div>
                                                <div className={`px-1.5 py-0.5 rounded text-[10px] ${((pkg.known_vulnerabilities_count || 0) > 0)
                                                        ? 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
                                                        : 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200'
                                                    }`}>
                                                    vulns: {pkg.known_vulnerabilities_count ?? 0}
                                                </div>
                                            </div>
                                            <div className="text-slate-500 dark:text-slate-400">
                                                {pkg.package_version || 'n/a'} | type: {pkg.package_type || 'unknown'}
                                            </div>
                                            {(pkg.layer_digest || pkg.package_path) && (
                                                <div className="text-slate-500 dark:text-slate-400 break-all">
                                                    layer: {pkg.layer_digest || 'n/a'}
                                                    {pkg.package_path ? ` | path: ${pkg.package_path}` : ''}
                                                </div>
                                            )}
                                            {(pkg.high_severity_vulnerabilities || []).length > 0 && (
                                                <div className="mt-2 rounded border border-rose-200 dark:border-rose-900 bg-rose-50/60 dark:bg-rose-950/20 p-2">
                                                    <button
                                                        onClick={() => toggleSbomPackageExpanded(sbomPackageKey(pkg))}
                                                        className="mb-1 w-full flex items-center justify-between text-[11px] font-medium text-rose-700 dark:text-rose-300"
                                                    >
                                                        <span>High/Critical CVEs</span>
                                                        <span>
                                                            {expandedSbomPackages[sbomPackageKey(pkg)] ? 'Hide' : `Show (${(pkg.high_severity_vulnerabilities || []).length})`}
                                                        </span>
                                                    </button>
                                                    <div className={`space-y-1 ${expandedSbomPackages[sbomPackageKey(pkg)] ? 'block' : 'hidden'}`}>
                                                        {(pkg.high_severity_vulnerabilities || []).map((vuln) => (
                                                            <div key={`${pkg.package_name}-${vuln.cve_id}`} className="flex items-center justify-between gap-2">
                                                                <div className="min-w-0">
                                                                    <div className="font-mono text-[11px] text-slate-700 dark:text-slate-200">
                                                                        {vuln.cve_id} ({vuln.severity}
                                                                        {typeof vuln.cvss_v3_score === 'number' ? ` ${vuln.cvss_v3_score.toFixed(1)}` : ''})
                                                                    </div>
                                                                    {vuln.description && (
                                                                        <div className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                                                                            {vuln.description}
                                                                        </div>
                                                                    )}
                                                                </div>
                                                                {vuln.reference_url && (
                                                                    <a
                                                                        href={vuln.reference_url}
                                                                        target="_blank"
                                                                        rel="noreferrer"
                                                                        className="shrink-0 text-[11px] text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200 underline"
                                                                    >
                                                                        CVE link
                                                                    </a>
                                                                )}
                                                            </div>
                                                        ))}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    ))}
                                    {filteredSbomPackages.length === 0 && (
                                        <div className="px-3 py-6 text-center text-xs text-slate-500 dark:text-slate-400">
                                            No packages match current filters.
                                        </div>
                                    )}
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            )}

            {activeTab === 'settings' && (
                <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6 space-y-3">
                    <h3 className="text-lg font-medium text-slate-900 dark:text-white">Image Settings</h3>
                    <div className="text-sm text-slate-600 dark:text-slate-300">Repository: {image.repository_url || 'N/A'}</div>
                    <div className="text-sm text-slate-600 dark:text-slate-300">Registry: {image.registry_provider || 'N/A'}</div>
                    <div className="text-sm text-slate-600 dark:text-slate-300">Architecture: {image.architecture || 'N/A'}</div>
                    <div className="text-sm text-slate-600 dark:text-slate-300">OS: {image.os || 'N/A'}</div>
                    <div className="rounded-md bg-slate-50 dark:bg-slate-900 p-3 text-xs text-slate-500 dark:text-slate-400 flex items-start gap-2">
                        {details.stats.latest_scan_status === 'completed' ? (
                            <CheckCircle2 className="w-4 h-4 text-emerald-500 mt-0.5" />
                        ) : (
                            <AlertTriangle className="w-4 h-4 text-amber-500 mt-0.5" />
                        )}
                        Latest scan status: {details.stats.latest_scan_status || 'unknown'}
                    </div>
                </div>
            )}
        </div>
    )
}

export default ImageDetailPage
