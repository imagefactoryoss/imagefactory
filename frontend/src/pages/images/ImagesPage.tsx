import { EmptyState } from '@/components/common/EmptyState'
import { useCapabilitySurfacesStore } from '@/store/capabilitySurfaces'
import { canCreateImages } from '@/utils/permissions'
import { Clock, Eye, EyeOff, Filter, Package, Plus, Search, Star } from 'lucide-react'
import React, { useCallback, useEffect, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { imageService } from '../../services/imageService'
import type { Image, ImageLifecycleStatus, ImageSearchFilters, ImageVisibility } from '../../types'

const ImagesPage: React.FC = () => {
    const location = useLocation()
    const canViewQuarantineRequests = useCapabilitySurfacesStore((state) => state.canViewNavKey('quarantine_requests'))
    const [images, setImages] = useState<Image[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [searchQuery, setSearchQuery] = useState('')
    const [filters, setFilters] = useState<ImageSearchFilters>({})
    const [sortBy, setSortBy] = useState<'updated_at' | 'created_at' | 'name' | 'pull_count'>('updated_at')
    const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
    const [showFilters, setShowFilters] = useState(false)
    const [popularImages, setPopularImages] = useState<Image[]>([])
    const [recentImages, setRecentImages] = useState<Image[]>([])

    // Determine base path for links (admin or regular user)
    const isAdminView = location.pathname.startsWith('/admin')
    const basePath = isAdminView ? '/admin/images' : '/images'
    const quarantineWorkspacePath = isAdminView ? '/admin/quarantine/requests' : '/quarantine/requests'

    // Load images based on current filters
    const loadImages = useCallback(async () => {
        try {
            setLoading(true)
            setError(null)

            const searchParams = imageService.buildSearchFilters({
                ...filters,
                query: searchQuery || undefined
            })
            searchParams.sort_by = sortBy
            searchParams.sort_order = sortOrder

            const response = await imageService.searchImages(searchParams)
            setImages(response.images)
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load images')
        } finally {
            setLoading(false)
        }
    }, [filters, searchQuery, sortBy, sortOrder])

    // Load popular and recent images for sidebar
    const loadSidebarData = useCallback(async () => {
        try {
            const [popular, recent] = await Promise.all([
                imageService.getPopularImages(5),
                imageService.getRecentImages(5)
            ])
            setPopularImages(popular || [])
            setRecentImages(recent || [])
        } catch (err) {
            setPopularImages([])
            setRecentImages([])
        }
    }, [])

    useEffect(() => {
        loadImages()
        loadSidebarData()
    }, [loadImages, loadSidebarData])

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault()
        loadImages()
    }

    const handleFilterChange = (key: keyof ImageSearchFilters, value: any) => {
        setFilters(prev => ({
            ...prev,
            [key]: value
        }))
    }

    const clearFilters = () => {
        setFilters({})
        setSearchQuery('')
        setSortBy('updated_at')
        setSortOrder('desc')
    }

    const getVisibilityIcon = (visibility: ImageVisibility) => {
        switch (visibility) {
            case 'public': return <Eye className="w-4 h-4 text-green-500" />
            case 'tenant': return <Eye className="w-4 h-4 text-blue-500" />
            case 'private': return <EyeOff className="w-4 h-4 text-gray-500" />
        }
    }

    const getStatusColor = (status: ImageLifecycleStatus) => {
        switch (status) {
            case 'published': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300'
            case 'draft': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300'
            case 'deprecated': return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300'
            case 'archived': return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300'
        }
    }

    if (loading && images.length === 0) {
        return (
            <div className="flex justify-center items-center min-h-96">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        )
    }

    return (
        <div className="px-4 py-6 sm:px-6 lg:px-8">
            <div className="sm:flex sm:items-center sm:justify-between">
                <div className="sm:flex-auto">
                    <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Image Catalog</h1>
                    <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
                        Discover and manage container images across your organization.
                    </p>
                </div>
                <div className="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
                    {canCreateImages() && (
                        <Link
                            to={`${basePath}/create`}
                            className="inline-flex items-center justify-center rounded-md border border-transparent bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                        >
                            <Plus className="w-4 h-4 mr-2" />
                            Add Image
                        </Link>
                    )}
                </div>
            </div>

            <div className="mt-8 grid grid-cols-1 lg:grid-cols-4 gap-8">
                {/* Main Content */}
                <div className="lg:col-span-3">
                    {/* Search and Filters */}
                    <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6 mb-6">
                        <form onSubmit={handleSearch} className="space-y-4">
                            <div className="flex gap-4">
                                <div className="flex-1">
                                    <div className="relative">
                                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                                        <input
                                            type="text"
                                            placeholder="Search images by name, description, or tags..."
                                            value={searchQuery}
                                            onChange={(e) => setSearchQuery(e.target.value)}
                                            className="w-full pl-10 pr-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                                        />
                                    </div>
                                </div>
                                <button
                                    type="button"
                                    onClick={() => setShowFilters(!showFilters)}
                                    className="inline-flex items-center px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-700 hover:bg-slate-50 dark:hover:bg-slate-600"
                                >
                                    <Filter className="w-4 h-4 mr-2" />
                                    Filters
                                </button>
                                <button
                                    type="submit"
                                    className="inline-flex items-center px-4 py-2 border border-transparent rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
                                >
                                    Search
                                </button>
                            </div>

                            <div className="flex flex-wrap items-center gap-3">
                                <label className="text-sm text-slate-600 dark:text-slate-300">Sort</label>
                                <select
                                    value={sortBy}
                                    onChange={(e) => setSortBy(e.target.value as 'updated_at' | 'created_at' | 'name' | 'pull_count')}
                                    className="border border-slate-300 dark:border-slate-600 rounded-md px-3 py-2 text-sm dark:bg-slate-700 dark:text-white"
                                >
                                    <option value="updated_at">Last Updated</option>
                                    <option value="created_at">Date Created</option>
                                    <option value="name">Name</option>
                                    <option value="pull_count">Pull Count</option>
                                </select>
                                <button
                                    type="button"
                                    onClick={() => setSortOrder((prev) => (prev === 'asc' ? 'desc' : 'asc'))}
                                    className="inline-flex items-center px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-700 hover:bg-slate-50 dark:hover:bg-slate-600"
                                >
                                    {sortOrder === 'desc' ? 'Descending' : 'Ascending'}
                                </button>
                            </div>

                            {showFilters && (
                                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4 border-t border-slate-200 dark:border-slate-600">
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                            Visibility
                                        </label>
                                        <select
                                            value={filters.visibility?.[0] || ''}
                                            onChange={(e) => handleFilterChange('visibility', e.target.value ? [e.target.value as ImageVisibility] : undefined)}
                                            className="w-full border border-slate-300 dark:border-slate-600 rounded-md px-3 py-2 dark:bg-slate-700 dark:text-white"
                                        >
                                            <option value="">All</option>
                                            <option value="public">Public</option>
                                            <option value="tenant">Tenant</option>
                                            <option value="private">Private</option>
                                        </select>
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                            Status
                                        </label>
                                        <select
                                            value={filters.status?.[0] || ''}
                                            onChange={(e) => handleFilterChange('status', e.target.value ? [e.target.value as ImageLifecycleStatus] : undefined)}
                                            className="w-full border border-slate-300 dark:border-slate-600 rounded-md px-3 py-2 dark:bg-slate-700 dark:text-white"
                                        >
                                            <option value="">All</option>
                                            <option value="published">Published</option>
                                            <option value="draft">Draft</option>
                                            <option value="deprecated">Deprecated</option>
                                            <option value="archived">Archived</option>
                                        </select>
                                    </div>
                                    <div className="flex items-end">
                                        <button
                                            type="button"
                                            onClick={clearFilters}
                                            className="w-full inline-flex justify-center items-center px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-700 hover:bg-slate-50 dark:hover:bg-slate-600"
                                        >
                                            Clear Filters
                                        </button>
                                    </div>
                                </div>
                            )}
                        </form>
                    </div>

                    {/* Error State */}
                    {error && (
                        <div className="bg-red-50 dark:bg-red-900 border border-red-200 dark:border-red-700 rounded-md p-4 mb-6">
                            <div className="text-red-800 dark:text-red-200">{error}</div>
                        </div>
                    )}

                    {/* Images Grid */}
                    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
                        {images && images.length > 0 ? (
                            images.map((image) => (
                                <Link
                                    key={image.id}
                                    to={`${basePath}/${image.id}`}
                                    className="bg-white dark:bg-slate-800 shadow rounded-lg p-6 hover:shadow-lg transition-shadow"
                                >
                                    <div className="flex items-start justify-between mb-4">
                                        <div className="flex items-center space-x-2">
                                            <Package className="w-5 h-5 text-slate-400" />
                                            <h3 className="text-lg font-medium text-slate-900 dark:text-white truncate">
                                                {image.name}
                                            </h3>
                                        </div>
                                        {getVisibilityIcon(image.visibility)}
                                    </div>

                                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-4 line-clamp-2">
                                        {image.description || 'No description available'}
                                    </p>

                                    <div className="flex items-center justify-between mb-4">
                                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(image.status)}`}>
                                            {image.status}
                                        </span>
                                        <div className="flex items-center text-sm text-slate-500 dark:text-slate-400">
                                            <Star className="w-4 h-4 mr-1" />
                                            {image.pull_count}
                                        </div>
                                    </div>

                                    {image.tags && image.tags.length > 0 && (
                                        <div className="flex flex-wrap gap-1 mb-4">
                                            {image.tags.slice(0, 3).map((tag) => (
                                                <span
                                                    key={tag}
                                                    className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-slate-100 dark:bg-slate-700 text-slate-800 dark:text-slate-200"
                                                >
                                                    {tag}
                                                </span>
                                            ))}
                                            {image.tags && image.tags.length > 3 && (
                                                <span className="text-xs text-slate-500 dark:text-slate-400">
                                                    +{image.tags.length - 3} more
                                                </span>
                                            )}
                                        </div>
                                    )}

                                    <div className="text-xs text-slate-500 dark:text-slate-400">
                                        Updated {new Date(image.updated_at).toLocaleDateString()}
                                    </div>
                                </Link>
                            ))
                        ) : (
                            !loading && (
                                <div className="col-span-1 md:col-span-2 xl:col-span-3">
                                    {searchQuery || Object.keys(filters).length > 0 ? (
                                        <EmptyState
                                            icon="🖼️"
                                            title="No images found"
                                            description="Try adjusting your search or filters to find images."
                                        />
                                    ) : (
                                        <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 overflow-hidden">
                                            <div className="px-5 py-4 border-b border-slate-200 dark:border-slate-700 bg-gradient-to-r from-blue-50 via-cyan-50 to-emerald-50 dark:from-slate-800 dark:via-slate-800 dark:to-slate-700">
                                                <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                                                    Build and publish to populate your image catalog
                                                </h3>
                                                <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
                                                    The catalog is automatically populated from successful builds. Once your first image is pushed, layers, SBOM, and vulnerability evidence appear here.
                                                </p>
                                            </div>
                                            <div className="px-5 py-4">
                                                <ol className="list-decimal pl-5 text-sm text-slate-700 dark:text-slate-300 space-y-1">
                                                    <li>Create a project and connect a source repository.</li>
                                                    <li>Run a build using Kaniko, Buildx, or another supported method.</li>
                                                    <li>Open the completed build and confirm image push + evidence capture.</li>
                                                </ol>
                                                <div className="mt-4 flex flex-wrap gap-2">
                                                    {canCreateImages() && (
                                                        <Link
                                                            to={`${basePath}/create`}
                                                            className="inline-flex items-center rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
                                                        >
                                                            Add Image
                                                        </Link>
                                                    )}
                                                    <Link
                                                        to="/builds"
                                                        className="inline-flex items-center rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-600"
                                                    >
                                                        Go to Builds
                                                    </Link>
                                                </div>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            )
                        )}
                    </div>
                </div>

                {/* Sidebar */}
                <div className="lg:col-span-1 space-y-6">
                    {canViewQuarantineRequests ? (
                        <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-2">Quarantine Requests</h3>
                            <p className="text-sm text-slate-600 dark:text-slate-400">
                                Manage quarantine submissions, retry failed requests, and review request timelines in the dedicated workspace.
                            </p>
                            <Link
                                to={quarantineWorkspacePath}
                                className="mt-3 inline-flex items-center rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-600"
                            >
                                Open Quarantine Requests
                            </Link>
                        </div>
                    ) : null}

                    {/* Popular Images */}
                    <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                        <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4 flex items-center">
                            <Star className="w-5 h-5 mr-2 text-yellow-500" />
                            Popular Images
                        </h3>
                        <div className="space-y-3">
                            {popularImages && popularImages.length > 0 ? (
                                popularImages.map((image, index) => (
                                    <Link
                                        key={image.id}
                                        to={`${basePath}/${image.id}`}
                                        className="flex items-center space-x-3 p-2 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700"
                                    >
                                        <div className="flex-shrink-0 w-8 h-8 bg-blue-100 dark:bg-blue-900 rounded-md flex items-center justify-center">
                                            <span className="text-sm font-medium text-blue-600 dark:text-blue-400">
                                                {index + 1}
                                            </span>
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
                                                {image.name}
                                            </p>
                                            <p className="text-sm text-slate-500 dark:text-slate-400">
                                                {image.pull_count} pulls
                                            </p>
                                        </div>
                                    </Link>
                                ))
                            ) : (
                                <p className="text-sm text-slate-500 dark:text-slate-400">No popular images</p>
                            )}
                        </div>
                    </div>

                    {/* Recent Images */}
                    <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                        <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-4 flex items-center">
                            <Clock className="w-5 h-5 mr-2 text-blue-500" />
                            Recent Images
                        </h3>
                        <div className="space-y-3">
                            {recentImages && recentImages.length > 0 ? (
                                recentImages.map((image) => (
                                    <Link
                                        key={image.id}
                                        to={`${basePath}/${image.id}`}
                                        className="flex items-center space-x-3 p-2 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700"
                                    >
                                        <div className="flex-shrink-0">
                                            <Package className="w-5 h-5 text-slate-400" />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
                                                {image.name}
                                            </p>
                                            <p className="text-sm text-slate-500 dark:text-slate-400">
                                                {new Date(image.created_at).toLocaleDateString()}
                                            </p>
                                        </div>
                                    </Link>
                                ))
                            ) : (
                                <p className="text-sm text-slate-500 dark:text-slate-400">No recent images</p>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default ImagesPage
