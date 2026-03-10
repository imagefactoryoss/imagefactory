import EditProjectWizardModal from '@/pages/projects/EditProjectWizardModal'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { projectService } from '@/services/projectService'
import { useTenantStore } from '@/store/tenant'
import { Project } from '@/types'
import { canManageMembers } from '@/utils/permissions'
import { Loader } from 'lucide-react'
import { useEffect, useState } from 'react'
import toast from 'react-hot-toast'

interface CreateProjectModalProps {
    isOpen: boolean
    onClose: () => void
    onCreated?: (project: Project) => void
}

export default function CreateProjectModal({ isOpen, onClose, onCreated }: CreateProjectModalProps) {
    const { selectedTenantId } = useTenantStore()
    const [draftProject, setDraftProject] = useState<Project | null>(null)
    const [finalProject, setFinalProject] = useState<Project | null>(null)
    const [creatingDraft, setCreatingDraft] = useState(false)
    const [finalized, setFinalized] = useState(false)
    const [draftName, setDraftName] = useState('')
    const confirmDialog = useConfirmDialog()

    useEffect(() => {
        if (!isOpen) {
            setDraftProject(null)
            setFinalProject(null)
            setFinalized(false)
            setCreatingDraft(false)
        }
    }, [isOpen])

    useEffect(() => {
        if (!isOpen || draftProject || creatingDraft) {
            return
        }

        const createDraft = async () => {
            if (!selectedTenantId) {
                toast.error('No tenant selected')
                onClose()
                return
            }

            try {
                setCreatingDraft(true)
                const uniqueSuffix = Date.now().toString().slice(-6)
                const name = `Untitled Project ${uniqueSuffix}`
                setDraftName(name)
                const project = await projectService.createProject(selectedTenantId, {
                    name,
                    description: '',
                    branch: 'main',
                    visibility: 'private',
                    isDraft: true,
                })
                setDraftProject(project)
                setFinalProject(project)
            } catch (error: any) {
                const message = error.response?.data?.error || (error instanceof Error ? error.message : 'Failed to create project')
                toast.error(message)
                onClose()
            } finally {
                setCreatingDraft(false)
            }
        }

        createDraft()
    }, [isOpen, draftProject, creatingDraft, selectedTenantId, onClose])

    const handleClose = async () => {
        if (draftProject && !finalized) {
            const confirmed = await confirmDialog({
                title: 'Discard Draft Project',
                message: 'Discard this draft project? Any repository auth you created here will be removed.',
                confirmLabel: 'Discard Draft',
                destructive: true,
            })
            if (!confirmed) {
                return
            }

            try {
                await projectService.deleteProject(draftProject.id, { source: 'draft_cancel' })
            } catch (error) {
                // Best-effort cleanup; ignore errors
            }
        }

        setDraftProject(null)
        setFinalProject(null)
        setFinalized(false)
        setDraftName('')
        onClose()
    }

    const handleComplete = () => {
        setFinalized(true)
        if (finalProject && onCreated) {
            onCreated(finalProject)
        }
        onClose()
    }

    if (!isOpen) return null

    if (!draftProject) {
        return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-gray-500/75 dark:bg-gray-900/75">
                <div className="flex items-center gap-3 rounded-lg bg-white dark:bg-gray-800 px-6 py-4 shadow-lg">
                    <Loader className="h-5 w-5 animate-spin text-blue-600 dark:text-blue-400" />
                    <div>
                        <p className="text-sm font-medium text-gray-700 dark:text-gray-200">
                            Creating draft {draftName ? `"${draftName}"` : 'project'}...
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                            We validate settings first, then finalize the real project.
                        </p>
                    </div>
                </div>
            </div>
        )
    }

    return (
        <EditProjectWizardModal
            isOpen={isOpen}
            projectId={draftProject.id}
            project={draftProject}
            canManageMembers={canManageMembers()}
            onClose={handleClose}
            onComplete={handleComplete}
            mode="create"
            onProjectUpdated={(updatedProject) => {
                setDraftProject(updatedProject)
                setFinalProject(updatedProject)
            }}
        />
    )
}
