import { RegistryAuth } from '@/types/registryAuth'
import React from 'react'
import RegistryAuthForm from './RegistryAuthForm'

interface RegistryAuthModalProps {
    projectId?: string
    isOpen: boolean
    authToEdit?: RegistryAuth
    onClose: () => void
    onSuccess: () => void
}

const RegistryAuthModal: React.FC<RegistryAuthModalProps> = ({
    projectId,
    isOpen,
    authToEdit,
    onClose,
    onSuccess,
}) => {
    if (!isOpen) return null

    const handleSuccess = () => {
        onSuccess()
        onClose()
    }

    return (
        <div className="fixed inset-0 z-50 overflow-y-auto">
            <div className="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
                <div
                    className="fixed inset-0 bg-slate-500 bg-opacity-75 transition-opacity"
                ></div>

                <div className="inline-block align-bottom bg-white dark:bg-slate-800 rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-2xl sm:w-full">
                    <div className="bg-white dark:bg-slate-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                        <RegistryAuthForm
                            projectId={projectId}
                            authToEdit={authToEdit}
                            onSuccess={handleSuccess}
                            onCancel={onClose}
                        />
                    </div>
                </div>
            </div>
        </div>
    )
}

export default RegistryAuthModal
