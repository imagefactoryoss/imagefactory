import { Project } from '@/types'
import { projectService } from '@/services/projectService'
import { RepositoryAuthList, RepositoryAuthModal } from '@/components/projects/repository-auth'
import { RegistryAuthList, RegistryAuthModal } from '@/components/projects/registry-auth'
import { ProjectMembersUI } from '@/components/projects/ProjectMembersUI'
import { repositoryAuthClient } from '@/api/repositoryAuthClient'
import { gitProviderClient } from '@/api/gitProviderClient'
import { repositoryBranchClient } from '@/api/repositoryBranchClient'
import { GitProvider } from '@/types/gitProvider'
import { RegistryAuth } from '@/types/registryAuth'
import { RepositoryAuth, RepositoryAuthSummary } from '@/types/repositoryAuth'
import { useAuthStore } from '@/store/auth'
import { Check, Info, Loader, Pencil, X } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import toast from 'react-hot-toast'

interface EditProjectWizardModalProps {
    isOpen: boolean
    project: Project | null
    projectId: string
    canManageMembers: boolean
    onClose: () => void
    onProjectUpdated: (project: Project) => void
    mode?: 'edit' | 'create'
    restrictToBasics?: boolean
    onComplete?: () => void
}

interface EditProjectFormData {
    name: string
    slug: string
    description?: string
    sourceName?: string
    repositoryUrl?: string
    branch?: string
    visibility?: 'private' | 'internal' | 'public'
    gitProvider?: string
    repositoryAuthId?: string
}

type WizardStep = 'basics' | 'members' | 'repository-auth' | 'registry-auth'
type BasicsTab = 'project' | 'source'

const stepOrder: WizardStep[] = ['basics', 'members', 'repository-auth', 'registry-auth']

const stepLabels: Record<WizardStep, string> = {
    basics: 'Basics',
    members: 'Members',
    'repository-auth': 'Repository Auth',
    'registry-auth': 'Registry Auth',
}

const slugify = (value: string): string =>
    value
        .toLowerCase()
        .trim()
        .replace(/[^a-z0-9\s_-]/g, '')
        .replace(/[\s_]+/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-+|-+$/g, '')
        .slice(0, 100)

export default function EditProjectWizardModal({
    isOpen,
    project,
    projectId,
    canManageMembers,
    onClose,
    onProjectUpdated,
    mode = 'edit',
    restrictToBasics = false,
    onComplete,
}: EditProjectWizardModalProps) {
    const [activeStep, setActiveStep] = useState<WizardStep>('basics')
    const [formData, setFormData] = useState<EditProjectFormData>({
        name: '',
        slug: '',
        description: '',
        sourceName: 'primary',
        repositoryUrl: '',
        branch: 'main',
        visibility: 'private',
        gitProvider: 'generic',
        repositoryAuthId: '',
    })
    const [errors, setErrors] = useState<Record<string, string>>({})
    const [slugEdited, setSlugEdited] = useState(false)
    const [loading, setLoading] = useState(false)
    const [showAuthModal, setShowAuthModal] = useState(false)
    const [authRefreshKey, setAuthRefreshKey] = useState(0)
    const [showRegistryAuthModal, setShowRegistryAuthModal] = useState(false)
    const [registryAuthRefreshKey, setRegistryAuthRefreshKey] = useState(0)
    const [registryAuthToEdit, setRegistryAuthToEdit] = useState<RegistryAuth | undefined>(undefined)
    const [basicsSaved, setBasicsSaved] = useState(false)
    const [gitProviders, setGitProviders] = useState<GitProvider[]>([])
    const [providersLoading, setProvidersLoading] = useState(false)
    const [providersError, setProvidersError] = useState<string | null>(null)
    const [repositoryAuths, setRepositoryAuths] = useState<RepositoryAuth[]>([])
    const [authsLoading, setAuthsLoading] = useState(false)
    const [authsError, setAuthsError] = useState<string | null>(null)
    const [availableAuths, setAvailableAuths] = useState<RepositoryAuthSummary[]>([])
    const [availableAuthsLoading, setAvailableAuthsLoading] = useState(false)
    const [availableAuthsError, setAvailableAuthsError] = useState<string | null>(null)
    const [selectedAvailableAuthId, setSelectedAvailableAuthId] = useState('')
    const [cloningAuth, setCloningAuth] = useState(false)
    const [branches, setBranches] = useState<string[]>([])
    const [branchesLoading, setBranchesLoading] = useState(false)
    const [branchesError, setBranchesError] = useState<string | null>(null)
    const currentUserId = useAuthStore(state => state.user?.id || '')
    const [cloneTestStatus, setCloneTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')
    const [cloneTestMessage, setCloneTestMessage] = useState<string | null>(null)
    const [basicsTab, setBasicsTab] = useState<BasicsTab>('project')
    const initializedProjectIdRef = useRef<string | null>(null)
    const needsRepositoryAuth =
        mode === 'create' &&
        !!formData.repositoryUrl?.trim() &&
        !authsLoading &&
        !availableAuthsLoading &&
        repositoryAuths.length === 0 &&
        availableAuths.length === 0

    const effectiveStepOrder = useMemo(() => (restrictToBasics ? (['basics'] as WizardStep[]) : stepOrder), [restrictToBasics])
    const stepIndex = useMemo(() => effectiveStepOrder.indexOf(activeStep), [activeStep, effectiveStepOrder])
    const filteredAvailableAuths = useMemo(() => {
        if (!formData.gitProvider) {
            return availableAuths
        }
        return availableAuths.filter(auth => !auth.git_provider_key || auth.git_provider_key === formData.gitProvider)
    }, [availableAuths, formData.gitProvider])
    const isBasicsDirty = useMemo(() => {
        if (!project) return false
        return (
            formData.name !== project.name ||
            (formData.slug || '') !== ((project.slug || slugify(project.name)) || '') ||
            (formData.description || '') !== (project.description || '') ||
            (mode === 'create' && (formData.sourceName || 'primary') !== 'primary') ||
            (formData.repositoryUrl || '') !== (project.repositoryUrl || '') ||
            (formData.branch || '') !== (project.branch || '') ||
            (formData.visibility || 'private') !== (project.visibility || 'private') ||
            (formData.gitProvider || 'generic') !== (project.gitProvider || 'generic') ||
            (formData.repositoryAuthId || '') !== (project.repositoryAuthId || '')
        )
    }, [formData, project])

    useEffect(() => {
        if (!project || !isOpen) return

        // Initialize once per project session. This prevents losing unsaved tab state
        // when parent updates the same project object after intermediate saves.
        if (initializedProjectIdRef.current === project.id) return

        setFormData({
            name: project.name,
            slug: project.slug || slugify(project.name),
            description: project.description || '',
            sourceName: 'primary',
            repositoryUrl: project.repositoryUrl || '',
            branch: project.branch || 'main',
            visibility: project.visibility || 'private',
            gitProvider: project.gitProvider || 'generic',
            repositoryAuthId: project.repositoryAuthId || '',
        })
        setErrors({})
        setActiveStep('basics')
        setBasicsTab('project')
        setBasicsSaved(false)
        setSlugEdited(false)
        initializedProjectIdRef.current = project.id
    }, [project, isOpen])

    useEffect(() => {
        if (!isOpen) {
            initializedProjectIdRef.current = null
        }
    }, [isOpen])

    useEffect(() => {
        if (basicsSaved && isBasicsDirty) {
            setBasicsSaved(false)
        }
    }, [basicsSaved, isBasicsDirty])

    const loadGitProviders = async () => {
        try {
            setProvidersLoading(true)
            setProvidersError(null)
            const response = await gitProviderClient.getGitProviders()
            setGitProviders(response.providers || [])
            if (!formData.gitProvider && response.providers?.length) {
                setFormData(prev => ({
                    ...prev,
                    gitProvider: response.providers[0].key,
                }))
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to load git providers'
            setProvidersError(message)
        } finally {
            setProvidersLoading(false)
        }
    }

    const loadRepositoryAuths = async () => {
        try {
            setAuthsLoading(true)
            setAuthsError(null)
            const response = await repositoryAuthClient.getRepositoryAuths(projectId)
            setRepositoryAuths(response.repository_auths || [])
            if (!formData.repositoryAuthId && response.repository_auths?.length) {
                setFormData(prev => ({
                    ...prev,
                    repositoryAuthId: response.repository_auths[0].id,
                }))
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to load repository auth'
            setAuthsError(message)
        } finally {
            setAuthsLoading(false)
        }
    }

    const loadAvailableAuths = async () => {
        try {
            setAvailableAuthsLoading(true)
            setAvailableAuthsError(null)
            const response = await repositoryAuthClient.getAvailableRepositoryAuths(projectId)
            setAvailableAuths(response.repository_auths || [])
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to load available repository auth'
            setAvailableAuthsError(message)
        } finally {
            setAvailableAuthsLoading(false)
        }
    }

    useEffect(() => {
        if (!isOpen) return
        loadGitProviders()
        loadRepositoryAuths()
        loadAvailableAuths()
    }, [isOpen])

    useEffect(() => {
        if (!isOpen) return
        loadRepositoryAuths()
        loadAvailableAuths()
    }, [authRefreshKey, isOpen])


    useEffect(() => {
        setBranches([])
        setBranchesError(null)
    }, [formData.repositoryUrl, formData.repositoryAuthId, formData.gitProvider])

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {}

        if (!formData.name.trim()) {
            newErrors.name = 'Project name is required'
        } else if (formData.name.length < 3) {
            newErrors.name = 'Project name must be at least 3 characters'
        } else if (formData.name.length > 100) {
            newErrors.name = 'Project name must not exceed 100 characters'
        }

        if (formData.description && formData.description.length > 500) {
            newErrors.description = 'Description must not exceed 500 characters'
        }

        if (!formData.slug.trim()) {
            newErrors.slug = 'Project slug is required'
        } else if (!/^[a-z0-9-]+$/.test(formData.slug)) {
            newErrors.slug = 'Slug must contain only lowercase letters, numbers, and hyphens'
        } else if (formData.slug.length > 100) {
            newErrors.slug = 'Slug must not exceed 100 characters'
        }

        if (mode === 'create' && formData.repositoryUrl?.trim() && !formData.sourceName?.trim()) {
            newErrors.sourceName = 'Source name is required when repository URL is set'
        }

        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleFetchBranches = async () => {
        if (!formData.repositoryUrl) {
            setBranchesError('Repository URL is required to load branches')
            return
        }
        if (!formData.repositoryAuthId) {
            setBranchesError('Repository auth is required to load branches')
            return
        }

        try {
            setBranchesLoading(true)
            setBranchesError(null)
            const response = await repositoryBranchClient.listBranches(projectId, {
                repository_url: formData.repositoryUrl,
                auth_id: formData.repositoryAuthId,
                provider_key: formData.gitProvider,
            })
            setBranches(response.branches || [])
        } catch (error) {
            const responseError = (error as any)?.response?.data?.error as string | undefined
            const message = responseError || (error instanceof Error ? error.message : 'Failed to load branches')
            const lower = message.toLowerCase()
            if (lower.includes('decrypt') || lower.includes('ciphertext')) {
                setBranchesError('Unable to decrypt repository credentials. Please recreate the repository auth and try again.')
            } else {
                setBranchesError(message)
            }
        } finally {
            setBranchesLoading(false)
        }
    }

    const handleSaveBasics = async () => {
        if (!validateForm()) {
            return false
        }

        try {
            setLoading(true)
            const updatedProject = await projectService.updateProject(projectId, {
                name: formData.name,
                slug: formData.slug,
                description: formData.description,
                repositoryUrl: undefined,
                branch: undefined,
                gitProvider: undefined,
                repositoryAuthId: undefined,
                visibility: formData.visibility,
                isDraft: mode === 'create' ? false : undefined,
            })

            if (mode === 'create' && formData.repositoryUrl?.trim()) {
                const sourcePayload = {
                    name: formData.sourceName?.trim() || 'primary',
                    provider: formData.gitProvider || 'generic',
                    repositoryUrl: formData.repositoryUrl.trim(),
                    defaultBranch: formData.branch?.trim() || 'main',
                    repositoryAuthId: formData.repositoryAuthId || undefined,
                    isDefault: true,
                    isActive: true,
                }
                const existingSources = await projectService.listProjectSources(projectId)
                const existingSource = existingSources.find((source) => source.isDefault) || existingSources[0]
                if (existingSource) {
                    await projectService.updateProjectSource(projectId, existingSource.id, sourcePayload)
                } else {
                    await projectService.createProjectSource(projectId, sourcePayload)
                }
            }

            onProjectUpdated(updatedProject)
            toast.success(mode === 'create' ? 'Project basics saved successfully!' : 'Project updated successfully!')
            setBasicsSaved(true)
            return true
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to update project'
            setErrors({ submit: message })
            return false
        } finally {
            setLoading(false)
        }
    }

    const handleCloneAuth = async () => {
        if (!selectedAvailableAuthId) {
            return
        }

        try {
            setCloningAuth(true)
            setCloneTestStatus('idle')
            setCloneTestMessage(null)
            const cloned = await repositoryAuthClient.cloneRepositoryAuth(projectId, selectedAvailableAuthId)
            await loadRepositoryAuths()
            setFormData(prev => ({
                ...prev,
                repositoryAuthId: cloned.id,
            }))
            setSelectedAvailableAuthId('')
            toast.success('Repository authentication cloned successfully')

            setCloneTestStatus('testing')
            const testResult = await repositoryAuthClient.testRepositoryAuth(projectId, cloned.id)
            if (testResult.success) {
                setCloneTestStatus('success')
                setCloneTestMessage(testResult.message || 'Connection successful')
                toast.success('Repository authentication test successful!')
            } else {
                setCloneTestStatus('error')
                setCloneTestMessage(testResult.message || 'Authentication test failed')
                toast.error(testResult.message || 'Authentication test failed')
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to clone repository auth'
            toast.error(message)
            setCloneTestStatus('error')
            setCloneTestMessage(message)
        } finally {
            setCloningAuth(false)
        }
    }

    const handleClose = () => {
        if (!loading) {
            setErrors({})
            onClose()
        }
    }

    const handleDone = async () => {
        if (isBasicsDirty) {
            const saved = await handleSaveBasics()
            if (!saved) {
                return
            }
        }
        if (onComplete) {
            onComplete()
            return
        }
        handleClose()
    }

    const goToStep = (step: WizardStep) => {
        setActiveStep(step)
    }

    const goNext = async () => {
        if (activeStep === 'basics') {
            if (mode === 'create' && basicsTab === 'project') {
                setBasicsTab('source')
                return
            }
            if (formData.repositoryUrl?.trim() && repositoryAuths.length === 0) {
                setActiveStep('repository-auth')
                setShowAuthModal(true)
                return
            }

            if (isBasicsDirty) {
                const saved = await handleSaveBasics()
                if (!saved) {
                    return
                }
            }
        }

        const nextStep = effectiveStepOrder[stepIndex + 1]
        if (nextStep) {
            setActiveStep(nextStep)
        }
    }

    const goBack = () => {
        const prevStep = effectiveStepOrder[stepIndex - 1]
        if (prevStep) {
            setActiveStep(prevStep)
        }
    }

    if (!isOpen || !project) return null

    return (
        <div
            className="fixed inset-0 z-50 overflow-y-auto bg-gray-500 bg-opacity-75 dark:bg-gray-900 dark:bg-opacity-75"
            onClick={(e) => {
                if (e.target === e.currentTarget) {
                    handleClose()
                }
            }}
        >
            <div
                className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0"
                onClick={(e) => e.stopPropagation()}
            >
                <div className="inline-block align-bottom bg-white dark:bg-gray-800 rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-4xl sm:w-full">
                    <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                        <div className="flex items-center gap-3">
                            <Pencil className="h-6 w-6 text-blue-600 dark:text-blue-400" />
                            <div>
                                <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                                    {mode === 'create' ? 'Create Project' : 'Edit Project'}
                                </h3>
                                <p className="text-sm text-gray-500 dark:text-gray-400">Step {stepIndex + 1} of {effectiveStepOrder.length}</p>
                            </div>
                        </div>
                        <button
                            onClick={handleClose}
                            disabled={loading}
                            className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300 disabled:opacity-50"
                        >
                            <X className="h-5 w-5" />
                        </button>
                    </div>

                    {!restrictToBasics && (
                        <div className="border-b border-gray-200 dark:border-gray-700 px-6">
                            <div className="flex space-x-6">
                                {effectiveStepOrder.map((step) => (
                                <button
                                    key={step}
                                    onClick={() => goToStep(step)}
                                    className={`py-4 px-1 border-b-2 text-sm font-medium ${activeStep === step
                                        ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                        : step === 'basics' && basicsSaved
                                            ? 'border-green-500 text-green-600 dark:text-green-400'
                                        : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'
                                        }`}
                                >
                                    <span className="inline-flex items-center gap-2">
                                        {stepLabels[step]}
                                        {step === 'basics' && basicsSaved && (
                                            <Check className="h-4 w-4 text-green-600 dark:text-green-400" />
                                        )}
                                    </span>
                                </button>
                                ))}
                            </div>
                        </div>
                    )}

                    <div className="px-6 py-6">
                        {mode === 'create' && (
                            <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-200">
                                <div className="flex items-start gap-2">
                                    <Info className="mt-0.5 h-4 w-4 flex-shrink-0" />
                                    <p>
                                        A draft project is created first so source settings and credentials can be validated.
                                        If you provide repository details below, an initial default project source is created automatically.
                                    </p>
                                </div>
                            </div>
                        )}
                        {activeStep === 'basics' && (
                            <div className="space-y-4">
                                {errors.submit && (
                                    <div className="rounded-md bg-red-50 dark:bg-red-900/20 p-3">
                                        <p className="text-sm font-medium text-red-800 dark:text-red-300">
                                            {errors.submit}
                                        </p>
                                    </div>
                                )}
                                {!restrictToBasics && (
                                    <div className="border-b border-gray-200 dark:border-gray-700">
                                        <div className="flex items-center gap-2">
                                            <button
                                                type="button"
                                                onClick={() => setBasicsTab('project')}
                                                className={`px-3 py-2 text-sm font-medium border-b-2 ${basicsTab === 'project'
                                                    ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                                    : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
                                                    }`}
                                            >
                                                Project
                                            </button>
                                            {mode === 'create' && (
                                                <button
                                                    type="button"
                                                    onClick={() => setBasicsTab('source')}
                                                    className={`px-3 py-2 text-sm font-medium border-b-2 ${basicsTab === 'source'
                                                        ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                                        : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
                                                        }`}
                                                >
                                                    Initial Source
                                                </button>
                                            )}
                                        </div>
                                    </div>
                                )}

                                {(restrictToBasics || basicsTab === 'project') && (
                                    <>
                                        <div>
                                            <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                Project Name *
                                            </label>
                                            <input
                                                type="text"
                                                id="name"
                                                value={formData.name}
                                                onChange={(e) => {
                                                    const nextName = e.target.value
                                                    setFormData({
                                                        ...formData,
                                                        name: nextName,
                                                        slug: slugEdited ? formData.slug : slugify(nextName),
                                                    })
                                                }}
                                                placeholder="e.g., Mobile App Backend"
                                                className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 ${errors.name
                                                    ? 'border-red-500'
                                                    : 'border-gray-300 dark:border-gray-600'
                                                    }`}
                                            />
                                            {errors.name && (
                                                <p className="mt-1 text-sm text-red-500 dark:text-red-400">{errors.name}</p>
                                            )}
                                        </div>

                                        <div>
                                            <label htmlFor="slug" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                Project Slug *
                                            </label>
                                            <input
                                                type="text"
                                                id="slug"
                                                value={formData.slug}
                                                onChange={(e) => {
                                                    setSlugEdited(true)
                                                    setFormData({ ...formData, slug: slugify(e.target.value) })
                                                }}
                                                placeholder="e.g., mobile-app-backend"
                                                className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 ${errors.slug
                                                    ? 'border-red-500'
                                                    : 'border-gray-300 dark:border-gray-600'
                                                    }`}
                                            />
                                            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                                                URL-friendly identifier. You can customize it before finalizing.
                                            </p>
                                            {errors.slug && (
                                                <p className="mt-1 text-sm text-red-500 dark:text-red-400">{errors.slug}</p>
                                            )}
                                        </div>

                                        <div>
                                            <label htmlFor="description" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                Description (Optional)
                                            </label>
                                            <textarea
                                                id="description"
                                                value={formData.description}
                                                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                                placeholder="Describe your project..."
                                                rows={4}
                                                className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 ${errors.description
                                                    ? 'border-red-500'
                                                    : 'border-gray-300 dark:border-gray-600'
                                                    }`}
                                            />
                                            {errors.description && (
                                                <p className="mt-1 text-sm text-red-500 dark:text-red-400">{errors.description}</p>
                                            )}
                                        </div>

                                        <div>
                                            <label htmlFor="visibility" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                Visibility
                                            </label>
                                            <select
                                                id="visibility"
                                                value={formData.visibility}
                                                onChange={(e) => setFormData({ ...formData, visibility: e.target.value as 'private' | 'internal' | 'public' })}
                                                className="w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 dark:border-gray-600"
                                            >
                                                <option value="private">Private</option>
                                                <option value="internal">Internal</option>
                                                <option value="public">Public</option>
                                            </select>
                                            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                                Private: Only team members can access. Internal: All tenant users can access. Public: Anyone can view.
                                            </p>
                                        </div>
                                    </>
                                )}

                                {!restrictToBasics && mode === 'create' && basicsTab === 'source' && (
                                    <>
                                        <div className="rounded-lg border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 p-3">
                                            <p className="text-sm text-blue-800 dark:text-blue-200">
                                                A source is a repository + default branch + credentials. Builds bind to a source and can still choose different refs (default, fixed, or webhook event ref).
                                            </p>
                                        </div>

                                        <div>
                                    {mode === 'create' && (
                                        <div className="mb-3">
                                            <label htmlFor="sourceName" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                <span className="inline-flex items-center gap-2">
                                                    Initial Source Name
                                                    <span title="A friendly label for the source used in build source selection.">
                                                        <Info className="h-4 w-4 text-gray-500 dark:text-gray-400" aria-hidden="true" />
                                                    </span>
                                                </span>
                                            </label>
                                            <input
                                                type="text"
                                                id="sourceName"
                                                value={formData.sourceName}
                                                onChange={(e) => setFormData({ ...formData, sourceName: e.target.value })}
                                                placeholder="primary"
                                                className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 ${errors.sourceName
                                                    ? 'border-red-500'
                                                    : 'border-gray-300 dark:border-gray-600'
                                                    }`}
                                            />
                                            {errors.sourceName && (
                                                <p className="mt-1 text-sm text-red-500 dark:text-red-400">{errors.sourceName}</p>
                                            )}
                                        </div>
                                    )}
                                    <label htmlFor="repositoryUrl" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        {mode === 'create' ? 'Initial Source Repository URL' : 'Repository URL'}
                                    </label>
                                    <input
                                        type="url"
                                        id="repositoryUrl"
                                        value={formData.repositoryUrl}
                                        onChange={(e) => setFormData({ ...formData, repositoryUrl: e.target.value })}
                                        placeholder="https://github.com/username/repo.git or git@github.com:username/repo.git"
                                        className="w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 dark:border-gray-600"
                                    />
                                    <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                        {mode === 'create'
                                            ? 'Optional during project creation. If provided, this becomes the default project source.'
                                            : 'Git repository URL (HTTPS or SSH).'}
                                    </p>
                                </div>

                                <div>
                                    <label htmlFor="gitProvider" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        Git Provider
                                    </label>
                                    <select
                                        id="gitProvider"
                                        value={formData.gitProvider}
                                        onChange={(e) => setFormData({ ...formData, gitProvider: e.target.value })}
                                        className="w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 dark:border-gray-600"
                                    >
                                        <option value="">Select a provider</option>
                                        {gitProviders.map((provider) => (
                                            <option key={provider.key} value={provider.key}>
                                                {provider.display_name}
                                            </option>
                                        ))}
                                    </select>
                                    {providersLoading && (
                                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Loading providers...</p>
                                    )}
                                    {providersError && (
                                        <p className="mt-1 text-sm text-red-500 dark:text-red-400">{providersError}</p>
                                    )}
                                </div>

                                <div>
                                    <label htmlFor="repositoryAuthId" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        <span className="flex items-center gap-2">
                                            Repository Auth
                                            {!formData.repositoryAuthId && (
                                                <span className="rounded-full bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200 px-2 py-0.5 text-xs font-semibold">
                                                    Required to load branches
                                                </span>
                                            )}
                                        </span>
                                    </label>
                                    <select
                                        id="repositoryAuthId"
                                        value={formData.repositoryAuthId}
                                        onChange={(e) => setFormData({ ...formData, repositoryAuthId: e.target.value })}
                                        disabled={authsLoading || repositoryAuths.length === 0}
                                        className="w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 dark:border-gray-600"
                                    >
                                        <option value="">
                                            {repositoryAuths.length === 0 ? 'No auths in this project yet' : 'Select authentication'}
                                        </option>
                                        {repositoryAuths.map((auth) => (
                                            <option key={auth.id} value={auth.id}>
                                                {auth.name} — {auth.id}
                                            </option>
                                        ))}
                                    </select>
                                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                                        UUIDs are shown for direct mapping to config references.
                                    </p>
                                    {authsLoading && (
                                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Loading repository auth...</p>
                                    )}
                                    {authsError && (
                                        <p className="mt-1 text-sm text-red-500 dark:text-red-400">{authsError}</p>
                                    )}
                                    {!!formData.repositoryUrl?.trim() && !authsLoading && !availableAuthsLoading && repositoryAuths.length === 0 && availableAuths.length === 0 && (
                                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                            No repository auth methods yet.
                                        </p>
                                    )}
                                    {!!formData.repositoryUrl?.trim() && !authsLoading && !availableAuthsLoading && repositoryAuths.length === 0 && availableAuths.length > 0 && (
                                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                            No auths in this project yet. Choose one from “Use Existing Auth (Clone)” or create a new auth.
                                        </p>
                                    )}
                                </div>

                                {!!formData.repositoryUrl?.trim() && !authsLoading && !availableAuthsLoading && repositoryAuths.length === 0 && availableAuths.length === 0 && (
                                    <div className="rounded-lg border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20 p-4">
                                        <div className="flex items-start justify-between gap-3">
                                            <div>
                                                <h4 className="text-sm font-semibold text-amber-900 dark:text-amber-200">
                                                    Create repository authentication
                                                </h4>
                                                <p className="mt-1 text-sm text-amber-800 dark:text-amber-300">
                                                    You need at least one auth method before you can fetch branches.
                                                </p>
                                            </div>
                                            <button
                                                type="button"
                                                onClick={() => setShowAuthModal(true)}
                                                className="px-3 py-2 text-sm font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700"
                                            >
                                                Create Auth
                                            </button>
                                        </div>
                                    </div>
                                )}

                                {!!formData.repositoryUrl?.trim() && !authsLoading && !availableAuthsLoading && repositoryAuths.length === 0 && (
                                    <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white/70 dark:bg-slate-800/40 p-4">
                                        <div className="flex items-start justify-between gap-3">
                                            <div>
                                                <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                                                    No repository auth selected
                                                </h4>
                                                <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
                                                    You can create a new auth or clone an existing one, then come back to Basics.
                                                </p>
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <button
                                                    type="button"
                                                    onClick={() => setShowAuthModal(true)}
                                                    className="px-3 py-2 text-sm font-medium text-white bg-slate-900 rounded-lg hover:bg-slate-800 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-white"
                                                >
                                                    Add Auth
                                                </button>
                                                <button
                                                    type="button"
                                                    onClick={() => setActiveStep('repository-auth')}
                                                    className="px-3 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-600 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                                                >
                                                    Go to Auth Step
                                                </button>
                                            </div>
                                        </div>
                                    </div>
                                )}

                                {!!formData.repositoryUrl?.trim() && !authsLoading && !availableAuthsLoading && repositoryAuths.length === 0 && availableAuths.length === 0 && (
                                    <div className="rounded-lg border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 p-4">
                                        <p className="text-sm text-blue-800 dark:text-blue-300">
                                            Authentication is required to continue to the next step. Create or clone a repository auth to proceed.
                                        </p>
                                    </div>
                                )}

                                <div>
                                    <label htmlFor="availableRepositoryAuthId" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        Use Existing Auth (Clone)
                                    </label>
                                    <div className="flex items-center gap-2">
                                        <select
                                            id="availableRepositoryAuthId"
                                            value={selectedAvailableAuthId}
                                            onChange={(e) => setSelectedAvailableAuthId(e.target.value)}
                                            className="flex-1 px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 dark:border-gray-600"
                                        >
                                            <option value="">Select an auth to clone</option>
                                            {filteredAvailableAuths.map((auth) => (
                                                <option key={auth.id} value={auth.id}>
                                                    {auth.name} — {auth.id} — {auth.project_name}{auth.git_provider_key ? ` (${auth.git_provider_key})` : ''}{auth.auth_type ? ` — ${auth.auth_type.replace('_', ' ')}` : ''}{auth.created_by ? ` — Created by ${auth.created_by === currentUserId ? 'you' : (auth.created_by_email || 'unknown')}` : ''}
                                                </option>
                                            ))}
                                        </select>
                                        <button
                                            type="button"
                                            onClick={handleCloneAuth}
                                            disabled={!selectedAvailableAuthId || cloningAuth}
                                            className="px-3 py-2 text-sm font-medium text-white bg-emerald-600 rounded-lg hover:bg-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed"
                                        >
                                            {cloningAuth ? 'Cloning...' : 'Use'}
                                        </button>
                                    </div>
                                    {selectedAvailableAuthId && formData.gitProvider && filteredAvailableAuths.some(auth => auth.id === selectedAvailableAuthId && auth.git_provider_key && auth.git_provider_key !== formData.gitProvider) && (
                                        <p className="mt-1 text-sm text-amber-600 dark:text-amber-400">
                                            Warning: This auth was created for a different provider than the current selection.
                                        </p>
                                    )}
                                    {availableAuthsLoading && (
                                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Loading available auths...</p>
                                    )}
                                    {availableAuthsError && (
                                        <p className="mt-1 text-sm text-red-500 dark:text-red-400">{availableAuthsError}</p>
                                    )}
                                    {!availableAuthsLoading && filteredAvailableAuths.length === 0 && (
                                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                            No existing auths available to clone.
                                        </p>
                                    )}
                                    {cloneTestStatus !== 'idle' && (
                                        <p className={`mt-1 text-sm ${cloneTestStatus === 'success' ? 'text-green-600 dark:text-green-400' : cloneTestStatus === 'testing' ? 'text-blue-600 dark:text-blue-400' : 'text-red-500 dark:text-red-400'}`}>
                                            {cloneTestStatus === 'testing' ? 'Testing cloned authentication...' : cloneTestMessage}
                                        </p>
                                    )}
                                </div>

                                <div>
                                    <label htmlFor="branch" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        {mode === 'create' ? 'Initial Source Default Branch' : 'Default Branch'}
                                    </label>
                                    <div className="flex items-center gap-2">
                                        {branches.length > 0 ? (
                                            <div className="flex-1 space-y-2">
                                                <select
                                                    id="branch"
                                                    value={branches.includes(formData.branch) ? formData.branch : '__custom__'}
                                                    onChange={(e) => {
                                                        const next = e.target.value
                                                        setFormData({ ...formData, branch: next === '__custom__' ? '' : next })
                                                    }}
                                                    className="w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 dark:border-gray-600"
                                                >
                                                    <option value="" disabled>Select a branch</option>
                                                    {branches.map((branch) => (
                                                        <option key={branch} value={branch}>
                                                            {branch}
                                                        </option>
                                                    ))}
                                                    <option value="__custom__">Custom branch…</option>
                                                </select>
                                                {(!formData.branch || !branches.includes(formData.branch)) && (
                                                    <input
                                                        type="text"
                                                        value={formData.branch}
                                                        onChange={(e) => setFormData({ ...formData, branch: e.target.value })}
                                                        placeholder="Enter custom branch"
                                                        className="w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 dark:border-gray-600"
                                                    />
                                                )}
                                            </div>
                                        ) : (
                                            <input
                                                type="text"
                                                id="branch"
                                                list="branch-options"
                                                value={formData.branch}
                                                onChange={(e) => setFormData({ ...formData, branch: e.target.value })}
                                                placeholder="main"
                                                className="flex-1 px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 dark:border-gray-600"
                                            />
                                        )}
                                        <button
                                            type="button"
                                            onClick={handleFetchBranches}
                                            disabled={branchesLoading || !formData.repositoryUrl || !formData.repositoryAuthId}
                                            className="px-3 py-2 text-sm font-medium text-white bg-slate-900 dark:bg-slate-100 dark:text-slate-900 rounded-lg hover:bg-slate-800 dark:hover:bg-white disabled:opacity-50 disabled:cursor-not-allowed"
                                        >
                                            {branchesLoading ? 'Loading...' : 'Load Branches'}
                                        </button>
                                    </div>
                                    {branches.length > 0 && (
                                        <datalist id="branch-options">
                                            {branches.map((branch) => (
                                                <option key={branch} value={branch} />
                                            ))}
                                        </datalist>
                                    )}
                                    {branchesError && (
                                        <p className="mt-1 text-sm text-red-500 dark:text-red-400">{branchesError}</p>
                                    )}
                                    <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                        {mode === 'create'
                                            ? 'Default branch for the initial project source (usually main or master).'
                                            : 'Default branch for builds (usually main or master).'}
                                    </p>
                                </div>
                                    </>
                                )}
                            </div>
                        )}

                        {activeStep === 'members' && (
                            <div className="space-y-4">
                                <p className="text-sm text-gray-600 dark:text-gray-400">
                                    Add or remove project members and adjust their roles.
                                </p>
                                <ProjectMembersUI projectId={projectId} canManageMembers={canManageMembers} />
                            </div>
                        )}

                        {activeStep === 'repository-auth' && (
                            <div className="space-y-6">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <h4 className="text-lg font-semibold text-gray-900 dark:text-white">Repository Authentication</h4>
                                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                            Manage authentication methods for accessing your Git repository.
                                        </p>
                                    </div>
                                    <button
                                        onClick={() => setShowAuthModal(true)}
                                        className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
                                    >
                                        Add Authentication
                                    </button>
                                </div>
                                <RepositoryAuthList
                                    projectId={projectId}
                                    repoUrl={formData.repositoryUrl || project?.repositoryUrl}
                                    refreshKey={authRefreshKey}
                                />
                            </div>
                        )}

                        {activeStep === 'registry-auth' && (
                            <div className="space-y-6">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <h4 className="text-lg font-semibold text-gray-900 dark:text-white">Registry Authentication</h4>
                                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                            Manage registry credentials used for image push and pull.
                                        </p>
                                    </div>
                                    <button
                                        onClick={() => {
                                            setRegistryAuthToEdit(undefined)
                                            setShowRegistryAuthModal(true)
                                        }}
                                        className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
                                    >
                                        Add Registry Auth
                                    </button>
                                </div>
                                <RegistryAuthList
                                    projectId={projectId}
                                    refreshKey={registryAuthRefreshKey}
                                    onEdit={(auth) => {
                                        setRegistryAuthToEdit(auth)
                                        setShowRegistryAuthModal(true)
                                    }}
                                />
                            </div>
                        )}
                    </div>

                    <div className="flex items-center justify-between gap-3 px-6 py-4 bg-gray-50 dark:bg-gray-700 border-t border-gray-200 dark:border-gray-600">
                        <div className="text-sm text-gray-500 dark:text-gray-400">
                            Step {stepIndex + 1} of {effectiveStepOrder.length}
                        </div>
                        <div className="flex items-center gap-3">
                            <button
                                onClick={handleClose}
                                disabled={loading}
                                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-600 border border-gray-300 dark:border-gray-500 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-500 disabled:opacity-50"
                            >
                                Close
                            </button>
                            {!restrictToBasics && activeStep !== 'basics' && (
                                <button
                                    onClick={goBack}
                                    className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-600 border border-gray-300 dark:border-gray-500 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-500"
                                >
                                    Back
                                </button>
                            )}
                            {activeStep === 'basics' && (
                                <button
                                    onClick={handleSaveBasics}
                                    disabled={loading}
                                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    {loading && <Loader className="h-4 w-4 animate-spin" />}
                                    {loading ? 'Saving...' : 'Save Changes'}
                                </button>
                            )}
                            {activeStep === 'basics' && basicsSaved && !isBasicsDirty && (
                                <span className="text-sm text-green-600 dark:text-green-400 font-medium">
                                    Saved
                                </span>
                            )}
                            {!restrictToBasics && activeStep !== 'registry-auth' && (
                                <button
                                    onClick={goNext}
                                    className="px-4 py-2 text-sm font-medium text-white bg-slate-900 dark:bg-slate-100 dark:text-slate-900 rounded-lg hover:bg-slate-800 dark:hover:bg-white"
                                >
                                    Next
                                </button>
                            )}
                            {!restrictToBasics && activeStep === 'registry-auth' && (
                                <button
                                    onClick={handleDone}
                                    className="px-4 py-2 text-sm font-medium text-white bg-slate-900 dark:bg-slate-100 dark:text-slate-900 rounded-lg hover:bg-slate-800 dark:hover:bg-white"
                                >
                                    Done
                                </button>
                            )}
                        </div>
                    </div>
                </div>
            </div>

            <RepositoryAuthModal
                projectId={projectId}
                isOpen={showAuthModal}
                onClose={() => setShowAuthModal(false)}
                helperText={needsRepositoryAuth ? 'Repository authentication is required to connect a project source and load branches. Add one now to continue creating your project.' : undefined}
                onSuccess={() => {
                    setAuthRefreshKey((prev) => prev + 1)
                    if (needsRepositoryAuth) {
                        setActiveStep('basics')
                        toast.success('Repository auth saved. Continue in Basics.')
                    }
                }}
            />

            <RegistryAuthModal
                projectId={projectId}
                isOpen={showRegistryAuthModal}
                authToEdit={registryAuthToEdit}
                onClose={() => {
                    setShowRegistryAuthModal(false)
                    setRegistryAuthToEdit(undefined)
                }}
                onSuccess={() => {
                    setRegistryAuthToEdit(undefined)
                    setRegistryAuthRefreshKey(prev => prev + 1)
                }}
            />
        </div>
    )
}
