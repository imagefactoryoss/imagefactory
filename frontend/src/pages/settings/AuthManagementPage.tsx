import { RegistryAuthForm, RegistryAuthList } from '@/components/projects/registry-auth'
import Drawer from '@/components/ui/Drawer'
import TenantRepositoryAuthPanel from '@/components/settings/auth/TenantRepositoryAuthPanel'
import { projectService } from '@/services/projectService'
import { Project } from '@/types'
import { RegistryAuth } from '@/types/registryAuth'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

type AuthSection = 'repository' | 'registry'
type ScopeFilter = 'tenant' | 'project'

const AuthManagementPage: React.FC = () => {
    const [activeSection, setActiveSection] = useState<AuthSection>('repository')
    const [scope, setScope] = useState<ScopeFilter>('tenant')
    const [searchQuery, setSearchQuery] = useState('')
    const [projects, setProjects] = useState<Project[]>([])
    const [selectedProjectId, setSelectedProjectId] = useState('')
    const [showRegistryModal, setShowRegistryModal] = useState(false)
    const [registryAuthToEdit, setRegistryAuthToEdit] = useState<RegistryAuth | undefined>(undefined)
    const [registryRefreshKey, setRegistryRefreshKey] = useState(0)

    useEffect(() => {
        const loadProjects = async () => {
            try {
                const response = await projectService.getProjects({ limit: 200 })
                const loaded = response.data || []
                setProjects(loaded)
                if (loaded.length > 0) {
                    setSelectedProjectId((current) => current || loaded[0].id)
                }
            } catch (error) {
                toast.error(error instanceof Error ? error.message : 'Failed to load projects for auth filtering')
            }
        }
        loadProjects()
    }, [])

    useEffect(() => {
        if (!selectedProjectId) return
        const exists = projects.some((project) => project.id === selectedProjectId)
        if (!exists) {
            setSelectedProjectId(projects.length > 0 ? projects[0].id : '')
        }
    }, [projects, selectedProjectId])

    const canUseProjectScope = projects.length > 0
    const scopeProjectId = scope === 'project' ? selectedProjectId : undefined

    const sectionTitle = useMemo(() => {
        if (activeSection === 'repository') return 'Repository Auth'
        return 'Registry Auth'
    }, [activeSection])

    return (
        <div className="px-4 py-6 sm:px-6 lg:px-8">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Auth Management</h1>
                <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                    Central place to manage repository and registry credentials.
                </p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
                <aside className="lg:col-span-3">
                    <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-3 space-y-2 sticky top-24">
                        <button
                            onClick={() => setActiveSection('repository')}
                            className={`w-full text-left px-3 py-2 rounded-md text-sm font-medium ${activeSection === 'repository'
                                ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-200'
                                : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700'
                                }`}
                        >
                            Repository Auth
                        </button>
                        <button
                            onClick={() => setActiveSection('registry')}
                            className={`w-full text-left px-3 py-2 rounded-md text-sm font-medium ${activeSection === 'registry'
                                ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-200'
                                : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700'
                                }`}
                        >
                            Registry Auth
                        </button>
                    </div>
                </aside>

                <section className="lg:col-span-9 space-y-4">
                    <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-4 space-y-4">
                        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">{sectionTitle} Filters</h2>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setScope('tenant')}
                                    className={`px-3 py-1.5 rounded-md text-sm font-medium ${scope === 'tenant'
                                        ? 'bg-blue-600 text-white'
                                        : 'bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200'
                                        }`}
                                >
                                    Tenant
                                </button>
                                <button
                                    onClick={() => setScope('project')}
                                    disabled={!canUseProjectScope}
                                    className={`px-3 py-1.5 rounded-md text-sm font-medium ${scope === 'project'
                                        ? 'bg-blue-600 text-white'
                                        : 'bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200'
                                        } disabled:opacity-50`}
                                >
                                    Project
                                </button>
                            </div>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                            <input
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                placeholder="Search by name, type, host or description"
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md dark:bg-slate-700 dark:text-white"
                            />

                            <select
                                value={selectedProjectId}
                                onChange={(e) => setSelectedProjectId(e.target.value)}
                                disabled={scope !== 'project'}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md dark:bg-slate-700 dark:text-white disabled:opacity-50"
                            >
                                <option value="">Select project</option>
                                {projects.map((project) => (
                                    <option key={project.id} value={project.id}>{project.name}</option>
                                ))}
                            </select>
                        </div>
                    </div>

                    {activeSection === 'repository' && (
                        <>
                            <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg px-4 py-3 text-sm text-amber-900 dark:text-amber-200">
                                Repository auth supports both tenant-scoped and project-scoped credentials. Use the scope chips above to switch context.
                            </div>
                            <TenantRepositoryAuthPanel
                                scope={scope}
                                selectedProjectId={scopeProjectId}
                                searchQuery={searchQuery}
                            />
                        </>
                    )}

                    {activeSection === 'registry' && (
                        <>
                            <div className="flex items-center justify-between">
                                <h2 className="text-xl font-semibold text-slate-900 dark:text-white">
                                    {scope === 'tenant' ? 'Tenant Registry Auth' : 'Project Registry Auth'}
                                </h2>
                                <button
                                    onClick={() => {
                                        setRegistryAuthToEdit(undefined)
                                        setShowRegistryModal(true)
                                    }}
                                    disabled={scope === 'project' && !selectedProjectId}
                                    className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm disabled:opacity-50"
                                >
                                    Add Registry Auth
                                </button>
                            </div>
                            <RegistryAuthList
                                projectId={scopeProjectId}
                                includeTenant={scope === 'tenant'}
                                searchQuery={searchQuery}
                                refreshKey={registryRefreshKey}
                                onEdit={(auth) => {
                                    setRegistryAuthToEdit(auth)
                                    setShowRegistryModal(true)
                                }}
                            />
                        </>
                    )}
                </section>
            </div>

            <Drawer
                isOpen={showRegistryModal}
                onClose={() => {
                    setShowRegistryModal(false)
                    setRegistryAuthToEdit(undefined)
                }}
                title={registryAuthToEdit ? 'Edit Registry Auth' : 'Add Registry Auth'}
                description={scope === 'tenant' ? 'Tenant-scoped registry credentials.' : 'Project-scoped registry credentials.'}
                width="xl"
            >
                <RegistryAuthForm
                    projectId={selectedProjectId || undefined}
                    allowProjectScope={projects.length > 0}
                    authToEdit={registryAuthToEdit}
                    onCancel={() => {
                        setShowRegistryModal(false)
                        setRegistryAuthToEdit(undefined)
                    }}
                    onSuccess={() => {
                        setShowRegistryModal(false)
                        setRegistryAuthToEdit(undefined)
                        setRegistryRefreshKey(prev => prev + 1)
                    }}
                />
            </Drawer>
        </div>
    )
}

export default AuthManagementPage
