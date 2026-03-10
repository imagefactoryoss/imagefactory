import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'

import { projectService } from '@/services/projectService'
import { ProjectBuildConfigMode, ProjectBuildConfigOnError } from '@/types'

interface ProjectBuildSettingsPanelProps {
    projectId: string
    canEdit: boolean
}

const ProjectBuildSettingsPanel: React.FC<ProjectBuildSettingsPanelProps> = ({ projectId, canEdit }) => {
    const [mode, setMode] = useState<ProjectBuildConfigMode>('repo_managed')
    const [configFile, setConfigFile] = useState('image-factory.yaml')
    const [onErrorPolicy, setOnErrorPolicy] = useState<ProjectBuildConfigOnError>('strict')
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)

    useEffect(() => {
        const load = async () => {
            try {
                setLoading(true)
                const settings = await projectService.getProjectBuildSettings(projectId)
                setMode(settings.buildConfigMode)
                setConfigFile(settings.buildConfigFile || 'image-factory.yaml')
                setOnErrorPolicy(settings.buildConfigOnError || 'strict')
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : 'Failed to load build settings'
                toast.error(errorMessage)
            } finally {
                setLoading(false)
            }
        }

        void load()
    }, [projectId])

    const handleSave = async () => {
        try {
            setSaving(true)
            const saved = await projectService.updateProjectBuildSettings(projectId, {
                buildConfigMode: mode,
                buildConfigFile: configFile.trim() || 'image-factory.yaml',
                buildConfigOnError: onErrorPolicy,
            })
            setMode(saved.buildConfigMode)
            setConfigFile(saved.buildConfigFile)
            setOnErrorPolicy(saved.buildConfigOnError)
            toast.success('Build settings updated')
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to update build settings'
            toast.error(errorMessage)
        } finally {
            setSaving(false)
        }
    }

    if (loading) {
        return (
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow border border-slate-200 dark:border-slate-700 p-6">
                <p className="text-slate-600 dark:text-slate-300">Loading build settings...</p>
            </div>
        )
    }

    return (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow border border-slate-200 dark:border-slate-700 p-6 space-y-6">
            <div>
                <h2 className="text-xl font-semibold text-slate-900 dark:text-white">Build Configuration Source</h2>
                <p className="text-sm text-slate-600 dark:text-slate-300 mt-1">
                    Choose whether build configuration comes from UI state or repository `image-factory.yaml`.
                </p>
            </div>

            <div className="space-y-3">
                <label className="flex items-start gap-3 p-3 border border-slate-200 dark:border-slate-700 rounded-md bg-slate-50 dark:bg-slate-900/60">
                    <input
                        type="radio"
                        name="build-config-mode"
                        value="repo_managed"
                        checked={mode === 'repo_managed'}
                        onChange={() => setMode('repo_managed')}
                        disabled={!canEdit || saving}
                        className="mt-1"
                    />
                    <span>
                        <span className="block text-sm font-medium text-slate-900 dark:text-white">Repository managed</span>
                        <span className="block text-xs text-slate-600 dark:text-slate-300">
                            Build start/retry reads config from repository YAML file.
                        </span>
                    </span>
                </label>

                <label className="flex items-start gap-3 p-3 border border-slate-200 dark:border-slate-700 rounded-md bg-slate-50 dark:bg-slate-900/60">
                    <input
                        type="radio"
                        name="build-config-mode"
                        value="ui_managed"
                        checked={mode === 'ui_managed'}
                        onChange={() => setMode('ui_managed')}
                        disabled={!canEdit || saving}
                        className="mt-1"
                    />
                    <span>
                        <span className="block text-sm font-medium text-slate-900 dark:text-white">UI managed</span>
                        <span className="block text-xs text-slate-600 dark:text-slate-300">
                            Build start/retry skips repository YAML and uses saved UI/API manifest configuration.
                        </span>
                    </span>
                </label>
            </div>

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Repository build config file</label>
                <input
                    type="text"
                    value={configFile}
                    onChange={(e) => setConfigFile(e.target.value)}
                    disabled={!canEdit || saving || mode !== 'repo_managed'}
                    placeholder="image-factory.yaml"
                    className="w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-60"
                />
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Default is `image-factory.yaml`. Relative paths are allowed (for example: `.ci/image-factory.yaml`).
                </p>
            </div>

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Repository config error handling</label>
                <select
                    value={onErrorPolicy}
                    onChange={(e) => setOnErrorPolicy(e.target.value as ProjectBuildConfigOnError)}
                    disabled={!canEdit || saving || mode !== 'repo_managed'}
                    className="w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-60"
                >
                    <option value="strict">Strict: fail build when repo YAML is invalid</option>
                    <option value="fallback_to_ui">Fallback: ignore invalid repo YAML and use saved UI config</option>
                </select>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Recommended: <span className="font-medium">Strict</span>. Fallback can hide runtime/config issues and should be used only for temporary mitigation.
                </p>
            </div>

            {canEdit ? (
                <div className="flex justify-end">
                    <button
                        type="button"
                        onClick={handleSave}
                        disabled={saving}
                        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-60"
                    >
                        {saving ? 'Saving...' : 'Save Build Settings'}
                    </button>
                </div>
            ) : (
                <p className="text-xs text-slate-500 dark:text-slate-400">You do not have permission to edit build settings.</p>
            )}
        </div>
    )
}

export default ProjectBuildSettingsPanel
