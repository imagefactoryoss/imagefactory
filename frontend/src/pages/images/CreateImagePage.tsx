import { ArrowLeft, Package } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { imageService } from '../../services/imageService'
import { useTenantStore } from '../../store/tenant'

const CreateImagePage: React.FC = () => {
    const navigate = useNavigate()
    const location = useLocation()
    const { imageId } = useParams<{ imageId: string }>()
    const isEditMode = Boolean(imageId)
    const [loading, setLoading] = useState(false)
    const [loadingInitial, setLoadingInitial] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [editAccessDenied, setEditAccessDenied] = useState(false)
    const [formData, setFormData] = useState({
        name: '',
        description: '',
        visibility: 'tenant' as 'public' | 'tenant' | 'private'
    })

    const { selectedTenantId } = useTenantStore()

    // Determine base path for navigation (admin or regular user)
    const isAdminView = location.pathname.startsWith('/admin')
    const basePath = isAdminView ? '/admin/images' : '/images'

    useEffect(() => {
        if (!isEditMode || !imageId) return

        const loadImage = async () => {
            try {
                setLoadingInitial(true)
                setError(null)
                const image = await imageService.getImage(imageId)
                if (selectedTenantId && image.tenant_id && image.tenant_id !== selectedTenantId) {
                    setEditAccessDenied(true)
                    setError('This image belongs to another tenant. Switch tenant context to edit it.')
                    return
                }
                setEditAccessDenied(false)
                setFormData({
                    name: image.name || '',
                    description: image.description || '',
                    visibility: (image.visibility as 'public' | 'tenant' | 'private') || 'tenant'
                })
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load image')
            } finally {
                setLoadingInitial(false)
            }
        }

        void loadImage()
    }, [imageId, isEditMode, selectedTenantId])

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        setLoading(true)
        setError(null)

        try {
            if (isEditMode && editAccessDenied) {
                setError('This image belongs to another tenant. Switch tenant context to edit it.')
                return
            }
            const payload: any = {
                ...formData
            }

            // For admin create flow, include selected tenant_id when explicit tenant context is chosen.
            // This allows system admins to create images for specific tenants
            if (!isEditMode && isAdminView && selectedTenantId) {
                payload.tenant_id = selectedTenantId
            }

            if (isEditMode && imageId) {
                await imageService.updateImage(imageId, payload)
            } else {
                await imageService.createImage(payload)
            }

            navigate(basePath)
        } catch (err) {
            setError(err instanceof Error ? err.message : `Failed to ${isEditMode ? 'update' : 'create'} image`)
        } finally {
            setLoading(false)
        }
    }

    // Check if admin view requires tenant selection
    const needsTenantSelection = !isEditMode && isAdminView && !selectedTenantId

    if (needsTenantSelection) {
        return (
            <div className="max-w-2xl mx-auto p-6">
                <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
                    <div className="text-center">
                        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100 mb-4">
                            Select Tenant Context
                        </h1>
                        <p className="text-slate-600 dark:text-slate-400 mb-6">
                            As a system administrator, you need to select which tenant to create the image for.
                        </p>
                        <Link
                            to="/admin"
                            className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                        >
                            Go to Admin Dashboard
                        </Link>
                    </div>
                </div>
            </div>
        )
    }

    if (loadingInitial) {
        return (
            <div className="flex justify-center items-center min-h-96">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        )
    }

    if (isEditMode && editAccessDenied) {
        return (
            <div className="max-w-2xl mx-auto p-6">
                <div className="mb-6">
                    <Link
                        to={basePath}
                        className="inline-flex items-center text-sm text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200"
                    >
                        <ArrowLeft className="w-4 h-4 mr-2" />
                        Back to Images
                    </Link>
                </div>
                <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-amber-200 dark:border-amber-800 p-6">
                    <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100 mb-2">Edit Unavailable</h1>
                    <p className="text-sm text-amber-700 dark:text-amber-300">
                        This image belongs to another tenant. Switch tenant context to the owner tenant to edit.
                    </p>
                </div>
            </div>
        )
    }

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
        const { name, value } = e.target
        setFormData(prev => ({
            ...prev,
            [name]: value
        }))
    }

    return (
        <div className="max-w-2xl mx-auto p-6">
            <div className="mb-6">
                <Link
                    to={basePath}
                    className="inline-flex items-center text-sm text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200"
                >
                    <ArrowLeft className="w-4 h-4 mr-2" />
                    Back to Images
                </Link>
            </div>

            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
                <div className="flex items-center mb-6">
                    <Package className="w-6 h-6 text-blue-500 mr-3" />
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">
                        {isEditMode ? 'Edit Image' : 'Create New Image'}
                    </h1>
                </div>

                {error && (
                    <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
                    </div>
                )}

                <form onSubmit={handleSubmit} className="space-y-6">
                    <div>
                        <label htmlFor="name" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Image Name *
                        </label>
                        <input
                            type="text"
                            id="name"
                            name="name"
                            required
                            disabled={isEditMode}
                            value={formData.name}
                            onChange={handleInputChange}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-slate-100"
                            placeholder="my-image"
                        />
                        {isEditMode && (
                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                Image name is immutable.
                            </p>
                        )}
                    </div>

                    <div>
                        <label htmlFor="description" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Description
                        </label>
                        <textarea
                            id="description"
                            name="description"
                            rows={3}
                            value={formData.description}
                            onChange={handleInputChange}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-slate-100"
                            placeholder="A brief description of the image"
                        />
                    </div>

                    <div>
                        <label htmlFor="visibility" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Visibility
                        </label>
                        <select
                            id="visibility"
                            name="visibility"
                            value={formData.visibility}
                            onChange={handleInputChange}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-slate-100"
                        >
                            <option value="private">Private</option>
                            <option value="tenant">Tenant</option>
                            <option value="public">Public</option>
                        </select>
                    </div>

                    <div className="flex justify-end space-x-3">
                        <Link
                            to={basePath}
                            className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm hover:bg-slate-50 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                        >
                            Cancel
                        </Link>
                        <button
                            type="submit"
                            disabled={loading}
                            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {loading
                                ? (isEditMode ? 'Saving...' : 'Creating...')
                                : (isEditMode ? 'Save Changes' : 'Create Image')}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    )
}

export default CreateImagePage
