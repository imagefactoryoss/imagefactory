import DockerfileInput from '@/components/common/DockerfileInput'
import HelpTooltip from '@/components/common/HelpTooltip'
import KeyValueForm from '@/components/common/KeyValueForm'
import { CreateDockerConfigRequest, DockerConfig } from '@/types/buildConfig'
import React, { useEffect, useState } from 'react'

interface DockerConfigFormProps {
    buildId: string
    onSubmit?: (config: CreateDockerConfigRequest) => void
    isLoading?: boolean
    error?: string
    initialConfig?: DockerConfig
    initialValue?: Partial<CreateDockerConfigRequest>
    onChange?: (config: CreateDockerConfigRequest) => void // For wizard mode
}

interface FormState {
    dockerfile: {
        source: 'path' | 'content' | 'upload'
        path?: string
        content?: string
        filename?: string
    };
    buildContext: string;
    registryRepo: string;
    targetStage?: string;
    buildArgs: Array<{ key: string; value: string }>;
    environmentVars: Array<{ key: string; value: string }>;
}

export const DockerConfigForm: React.FC<DockerConfigFormProps> = ({
    buildId,
    onSubmit,
    isLoading = false,
    error,
    initialConfig,
    initialValue,
    onChange
}) => {
    const toDockerfileState = (value: any) => {
        if (!value) return { source: 'path' as const, path: 'Dockerfile' }
        if (typeof value === 'string') return { source: 'path' as const, path: value }
        return value
    }

    const toKeyValueArray = (value: Record<string, string> | undefined) =>
        value
            ? Object.entries(value).map(([key, val]) => ({ key, value: val }))
            : []

    const [formState, setFormState] = useState<FormState>({
        dockerfile: toDockerfileState(initialValue?.dockerfile ?? initialConfig?.config.dockerfile),
        buildContext: initialValue?.build_context ?? initialConfig?.config.build_context ?? '.',
        registryRepo: initialValue?.registry_repo ?? initialConfig?.config.registry_repo ?? '',
        targetStage: initialValue?.target_stage ?? initialConfig?.config.target_stage,
        buildArgs: toKeyValueArray(initialValue?.build_args ?? initialConfig?.config.build_args),
        environmentVars: toKeyValueArray(initialValue?.environment_vars ?? initialConfig?.config.environment_vars),
    })

    const [errors, setErrors] = useState<Record<string, string>>({})

    useEffect(() => {
        if (!initialValue) return
        setFormState((prev) => {
            const next: FormState = {
                ...prev,
                dockerfile: toDockerfileState(initialValue.dockerfile),
                buildContext: initialValue.build_context ?? prev.buildContext,
                registryRepo: initialValue.registry_repo ?? prev.registryRepo,
                targetStage: initialValue.target_stage ?? prev.targetStage,
                buildArgs: toKeyValueArray(initialValue.build_args),
                environmentVars: toKeyValueArray(initialValue.environment_vars),
            }

            const unchanged =
                JSON.stringify(prev.dockerfile) === JSON.stringify(next.dockerfile) &&
                prev.buildContext === next.buildContext &&
                prev.registryRepo === next.registryRepo &&
                prev.targetStage === next.targetStage &&
                JSON.stringify(prev.buildArgs) === JSON.stringify(next.buildArgs) &&
                JSON.stringify(prev.environmentVars) === JSON.stringify(next.environmentVars)

            return unchanged ? prev : next
        })
    }, [initialValue])

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {}

        // Validate dockerfile based on source
        if (formState.dockerfile.source === 'path' && !formState.dockerfile.path?.trim()) {
            newErrors.dockerfile = 'Dockerfile path is required';
        } else if (formState.dockerfile.source === 'content' && !formState.dockerfile.content?.trim()) {
            newErrors.dockerfile = 'Dockerfile content is required';
        } else if (formState.dockerfile.source === 'upload' && !formState.dockerfile.content?.trim()) {
            newErrors.dockerfile = 'Please upload a Dockerfile';
        }

        if (!formState.buildContext.trim()) {
            newErrors.buildContext = 'Build context is required'
        }
        if (!formState.registryRepo.trim()) {
            newErrors.registryRepo = 'Registry repository is required'
        }

        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()

        if (!validateForm()) {
            return
        }

        const buildArgs = formState.buildArgs.reduce(
            (acc, { key, value }) => {
                if (key) acc[key] = value
                return acc
            },
            {} as Record<string, string>
        )

        const environmentVars = formState.environmentVars.reduce(
            (acc, { key, value }) => {
                if (key) acc[key] = value
                return acc
            },
            {} as Record<string, string>
        )

        const request: CreateDockerConfigRequest = {
            build_id: buildId || '',
            dockerfile: formState.dockerfile,
            build_context: formState.buildContext,
            registry_repo: formState.registryRepo.trim(),
            target_stage: formState.targetStage,
            build_args: Object.keys(buildArgs).length > 0 ? buildArgs : undefined,
            environment_vars: Object.keys(environmentVars).length > 0 ? environmentVars : undefined,
        }

        if (onChange) {
            onChange(request)
        } else if (onSubmit) {
            onSubmit(request)
        }
    }

    return (
        <form onSubmit={handleSubmit} className="space-y-6 max-w-2xl">
            {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                </div>
            )}

            {/* Basic Info */}
            <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <h3 className="text-sm font-semibold text-blue-900 dark:text-blue-200 mb-2">ℹ️ Docker Configuration</h3>
                <p className="text-sm text-blue-700 dark:text-blue-300">
                    Standard Docker build from a Dockerfile. Supports multi-stage builds and custom build arguments.
                </p>
            </div>

            {/* Dockerfile */}
            <DockerfileInput
                value={formState.dockerfile}
                onChange={(dockerfile) => setFormState({ ...formState, dockerfile })}
                error={errors.dockerfile}
            />

            {/* Build Context */}
            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    <span className="inline-flex items-center gap-2">
                        Build Context *
                        <HelpTooltip text='Dockerfile paths are resolved from the build context root. Example: if Dockerfile uses "COPY cmd ./cmd", set Build Context to "examples/image-factory-user-docs" when "cmd/" exists there.' />
                    </span>
                </label>
                <input
                    type="text"
                    value={formState.buildContext}
                    onChange={(e) => {
                        setFormState({ ...formState, buildContext: e.target.value })
                        setErrors({ ...errors, buildContext: '' })
                    }}
                    placeholder="."
                    className={`
                        w-full px-3 py-2 border rounded-md text-sm
                        focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400
                        ${errors.buildContext ? 'border-red-300 dark:border-red-600' : 'border-gray-300 dark:border-gray-600'}
                    `}
                />
                {errors.buildContext && <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.buildContext}</p>}
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    The directory used as build context root (usually "."). Dockerfile COPY/ADD paths are resolved from this root.
                </p>
            </div>

            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Registry Repository *
                </label>
                <input
                    type="text"
                    value={formState.registryRepo}
                    onChange={(e) => {
                        setFormState({ ...formState, registryRepo: e.target.value })
                        setErrors({ ...errors, registryRepo: '' })
                    }}
                    placeholder="registry.gitlab.com/group/my-app"
                    className={`
                        w-full px-3 py-2 border rounded-md text-sm
                        focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400
                        ${errors.registryRepo ? 'border-red-300 dark:border-red-600' : 'border-gray-300 dark:border-gray-600'}
                    `}
                />
                {errors.registryRepo && <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.registryRepo}</p>}
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Full image repository to tag and push after build.
                </p>
            </div>

            {/* Target Stage (for multi-stage builds) */}
            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Target Stage
                </label>
                <input
                    type="text"
                    value={formState.targetStage || ''}
                    onChange={(e) => setFormState({ ...formState, targetStage: e.target.value || undefined })}
                    placeholder="e.g., builder, runtime"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    For multi-stage builds, specify which stage to build to. Leave empty to build all stages.
                </p>
            </div>

            {/* Build Arguments */}
            <KeyValueForm
                title="Build Arguments (--build-arg)"
                items={formState.buildArgs}
                onItemsChange={(buildArgs) => setFormState({ ...formState, buildArgs })}
                keyPlaceholder="Argument name"
                valuePlaceholder="Argument value"
            />

            {/* Environment Variables */}
            <KeyValueForm
                title="Environment Variables"
                items={formState.environmentVars}
                onItemsChange={(environmentVars) => setFormState({ ...formState, environmentVars })}
                keyPlaceholder="Variable name"
                valuePlaceholder="Variable value"
            />

            {/* Submit Button */}
            <div className="flex gap-3 pt-4">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="
                        px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium
                        hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed
                        transition-colors
                    "
                >
                    {isLoading ? 'Creating Configuration...' : 'Create Configuration'}
                </button>
            </div>
        </form>
    )
}

export default DockerConfigForm
