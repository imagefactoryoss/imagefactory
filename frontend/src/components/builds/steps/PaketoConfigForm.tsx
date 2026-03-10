import KeyValueForm from '@/components/common/KeyValueForm'
import { CreatePaketoConfigRequest } from '@/types/buildConfig'
import React, { useEffect, useRef, useState } from 'react'

interface PaketoConfigFormProps {
    buildId?: string
    onSubmit?: (config: CreatePaketoConfigRequest) => void
    onChange?: (config: CreatePaketoConfigRequest) => void
    isLoading?: boolean
    error?: string
    initialValue?: Partial<CreatePaketoConfigRequest>
}

const DEFAULT_BUILDER = 'paketobuildpacks/builder:base'

const PaketoConfigForm: React.FC<PaketoConfigFormProps> = ({
    buildId,
    onSubmit,
    onChange,
    isLoading = false,
    error,
    initialValue,
}) => {
    const onChangeRef = useRef(onChange)
    const [builder, setBuilder] = useState(initialValue?.builder || DEFAULT_BUILDER)
    const [buildpacks, setBuildpacks] = useState<string[]>(initialValue?.buildpacks || [])
    const [buildpackInput, setBuildpackInput] = useState((initialValue?.buildpacks || []).join(', '))
    const [envItems, setEnvItems] = useState<Array<{ key: string; value: string }>>(
        initialValue?.env ? Object.entries(initialValue.env).map(([key, value]) => ({ key, value })) : []
    )
    const [buildArgItems, setBuildArgItems] = useState<Array<{ key: string; value: string }>>(
        initialValue?.build_args ? Object.entries(initialValue.build_args).map(([key, value]) => ({ key, value })) : []
    )
    const [errors, setErrors] = useState<Record<string, string>>({})

    const parseBuildpacks = (raw: string): string[] => raw
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean)

    const toRecord = (items: Array<{ key: string; value: string }>): Record<string, string> => items
        .reduce((acc, item) => {
            if (item.key.trim()) {
                acc[item.key.trim()] = item.value
            }
            return acc
        }, {} as Record<string, string>)

    const buildRequest = (): CreatePaketoConfigRequest => {
        const env = toRecord(envItems)
        const buildArgs = toRecord(buildArgItems)

        return {
            build_id: buildId || '',
            builder: builder.trim(),
            buildpacks: buildpacks.length > 0 ? buildpacks : undefined,
            env: Object.keys(env).length > 0 ? env : undefined,
            build_args: Object.keys(buildArgs).length > 0 ? buildArgs : undefined,
        }
    }

    const validate = (): boolean => {
        const nextErrors: Record<string, string> = {}
        if (!builder.trim()) {
            nextErrors.builder = 'Builder is required'
        }
        setErrors(nextErrors)
        return Object.keys(nextErrors).length === 0
    }

    useEffect(() => {
        onChangeRef.current = onChange
    }, [onChange])

    useEffect(() => {
        if (onChangeRef.current) {
            onChangeRef.current(buildRequest())
        }
    }, [builder, buildpacks, envItems, buildArgItems])

    const handleBuildpacksBlur = () => {
        const parsed = parseBuildpacks(buildpackInput)
        setBuildpacks(parsed)
        setBuildpackInput(parsed.join(', '))
    }

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        if (!validate()) {
            return
        }
        const req = buildRequest()
        if (onChange) {
            onChange(req)
        } else if (onSubmit) {
            onSubmit(req)
        }
    }

    return (
        <form onSubmit={handleSubmit} className="space-y-6 max-w-2xl">
            {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                </div>
            )}

            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Paketo Builder *
                </label>
                <input
                    type="text"
                    value={builder}
                    onChange={(e) => setBuilder(e.target.value)}
                    placeholder={DEFAULT_BUILDER}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                />
                {errors.builder && (
                    <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.builder}</p>
                )}
            </div>

            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Buildpacks
                </label>
                <input
                    type="text"
                    value={buildpackInput}
                    onChange={(e) => setBuildpackInput(e.target.value)}
                    onBlur={handleBuildpacksBlur}
                    placeholder="paketobuildpacks/nodejs, paketobuildpacks/procfile"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Comma-separated buildpack IDs (optional)
                </p>
            </div>

            <KeyValueForm
                title="Environment Variables"
                items={envItems}
                onItemsChange={setEnvItems}
                keyPlaceholder="BP_NODE_RUN_SCRIPTS"
                valuePlaceholder="build"
            />

            <KeyValueForm
                title="Build Arguments"
                items={buildArgItems}
                onItemsChange={setBuildArgItems}
                keyPlaceholder="ARG_NAME"
                valuePlaceholder="value"
            />

            {onSubmit && (
                <div className="pt-4">
                    <button
                        type="submit"
                        disabled={isLoading}
                        className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {isLoading ? 'Saving...' : 'Save Paketo Configuration'}
                    </button>
                </div>
            )}
        </form>
    )
}

export default PaketoConfigForm
