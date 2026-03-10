import React from 'react'
import { RepositoryAuth } from '@/types/repositoryAuth'
import RepositoryAuthForm from './RepositoryAuthForm'

interface RepositoryAuthModalProps {
    projectId: string
    isOpen: boolean
    authToEdit?: RepositoryAuth
    helperText?: string
    onClose: () => void
    onSuccess: () => void
}

const RepositoryAuthModal: React.FC<RepositoryAuthModalProps> = ({
    projectId,
    isOpen,
    authToEdit,
    helperText,
    onClose,
    onSuccess
}) => {
    if (!isOpen) return null

    const handleSuccess = () => {
        onSuccess()
        onClose()
    }

    return (
        <div className="fixed inset-0 z-50 overflow-y-auto">
            <div className="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
                {/* Background overlay */}
                <div
                    className="fixed inset-0 bg-slate-500 bg-opacity-75 transition-opacity"
                ></div>

                {/* Modal panel */}
                <div className="inline-block align-bottom bg-white dark:bg-slate-800 rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-2xl sm:w-full">
                    <div className="bg-white dark:bg-slate-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                        {helperText && (
                            <div className="mb-4 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-900 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-100">
                                <div className="flex items-start gap-3">
                                    <span className="mt-0.5 inline-flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-white text-xs">
                                        !
                                    </span>
                                    <div className="space-y-1">
                                        <p className="font-medium">Repository auth required</p>
                                        <p className="text-sm text-blue-900/90 dark:text-blue-100/90">
                                            {helperText}
                                        </p>
                                        <p className="text-xs text-blue-800/80 dark:text-blue-100/70">
                                            Add one now to unlock branch selection and continue project setup.
                                        </p>
                                        <button
                                            type="button"
                                            onClick={() => {
                                                const el = document.getElementById('repository-auth-name') as HTMLInputElement | null
                                                el?.focus()
                                            }}
                                            className="mt-2 inline-flex items-center rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700"
                                        >
                                            Add auth now
                                        </button>
                                    </div>
                                </div>
                            </div>
                        )}
                        <RepositoryAuthForm
                            projectId={projectId}
                            initialAuth={authToEdit}
                            onSuccess={handleSuccess}
                            onCancel={onClose}
                        />
                    </div>
                </div>
            </div>
        </div>
    )
}

export default RepositoryAuthModal
