import { WizardState } from '@/types'
import React from 'react'

interface ValidationStepProps {
    wizardState: WizardState
    onCreateBuild: () => Promise<void>
}

const ValidationStep: React.FC<ValidationStepProps> = ({
    wizardState,
    onCreateBuild
}) => {
    const handleCreateBuild = async () => {
        try {
            await onCreateBuild()
        } catch (error) {
        }
    }

    return (
        <div className="space-y-6">
            <div>
                <h3 className="text-lg font-medium text-slate-900 dark:text-white">
                    Review & Create Build
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                    Review your build configuration and create the build.
                </p>
            </div>

            {/* Build Summary */}
            <div className="bg-slate-50 dark:bg-slate-800 rounded-lg p-6 space-y-4">
                <h4 className="text-md font-medium text-slate-900 dark:text-white">
                    Build Summary
                </h4>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                        <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            Project:
                        </span>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {wizardState.selectedProject?.name || 'Not selected'}
                        </p>
                    </div>

                    <div>
                        <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            Build Name:
                        </span>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {wizardState.buildName || 'Not specified'}
                        </p>
                    </div>

                    <div>
                        <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            Build Method:
                        </span>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {wizardState.buildMethod || 'Not selected'}
                        </p>
                    </div>

                    <div>
                        <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            SBOM Tool:
                        </span>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {wizardState.selectedTools.sbom || 'Not selected'}
                        </p>
                    </div>

                    <div>
                        <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            Security Scanner:
                        </span>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {wizardState.selectedTools.scan || 'Not selected'}
                        </p>
                    </div>

                    <div>
                        <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            Registry Backend:
                        </span>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {wizardState.selectedTools.registry || 'Not selected'}
                        </p>
                    </div>

                    <div>
                        <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            Secret Manager:
                        </span>
                        <p className="text-sm text-slate-900 dark:text-white">
                            {wizardState.selectedTools.secrets || 'Not selected'}
                        </p>
                    </div>
                </div>

                {wizardState.buildDescription && (
                    <div>
                        <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            Description:
                        </span>
                        <p className="text-sm text-slate-900 dark:text-white mt-1">
                            {wizardState.buildDescription}
                        </p>
                    </div>
                )}
            </div>

            {/* Validation Messages */}
            {Object.keys(wizardState.validationErrors).length > 0 && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <div className="flex">
                        <div className="flex-shrink-0">
                            <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                            </svg>
                        </div>
                        <div className="ml-3">
                            <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
                                Validation Errors
                            </h3>
                            <div className="mt-2 text-sm text-red-700 dark:text-red-300">
                                <ul className="list-disc pl-5 space-y-1">
                                    {Object.entries(wizardState.validationErrors).map(([key, error]) => (
                                        <li key={key}>{error}</li>
                                    ))}
                                </ul>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* Create Build Button */}
            <div className="flex justify-end">
                <button
                    onClick={handleCreateBuild}
                    disabled={wizardState.isSubmitting || Object.keys(wizardState.validationErrors).length > 0}
                    className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    {wizardState.isSubmitting ? (
                        <>
                            <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                            </svg>
                            Creating Build...
                        </>
                    ) : (
                        'Create Build'
                    )}
                </button>
            </div>
        </div>
    )
}

export default ValidationStep