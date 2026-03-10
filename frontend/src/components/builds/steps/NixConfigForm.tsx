import KeyValueForm from '@/components/common/KeyValueForm'
import { CreateNixConfigRequest, NixConfig } from '@/types/buildConfig'
import React, { useEffect, useRef, useState } from 'react'

interface NixConfigFormProps {
    buildId?: string
    onSubmit?: (config: CreateNixConfigRequest) => void
    isLoading?: boolean
    error?: string
    initialConfig?: NixConfig
    initialValue?: Partial<CreateNixConfigRequest>
    onChange?: (config: CreateNixConfigRequest) => void // For wizard mode
}

interface FormState {
    nixExpression?: string
    flakeUri?: string
    useFlake: boolean
    attributes: string[]
    outputs: Array<{ key: string; value: string }>
    cacheDir?: string
    pure: boolean
    showTrace: boolean
}

export const NixConfigForm: React.FC<NixConfigFormProps> = ({
    buildId,
    onSubmit,
    isLoading = false,
    error,
    initialConfig,
    initialValue,
    onChange,
}) => {
    const onChangeRef = useRef(onChange)
    const [formState, setFormState] = useState<FormState>({
        nixExpression: initialValue?.nix_expression ?? initialConfig?.config.nix_expression,
        flakeUri: initialValue?.flake_uri ?? initialConfig?.config.flake_uri,
        useFlake: !!(initialValue?.flake_uri ?? initialConfig?.config.flake_uri),
        attributes: initialValue?.attributes || initialConfig?.config.attributes || [],
        outputs: initialValue?.outputs
            ? Object.entries(initialValue.outputs).map(([k, v]) => ({
                key: k,
                value: v,
            }))
            : initialConfig?.config.outputs
                ? Object.entries(initialConfig.config.outputs).map(([k, v]) => ({
                key: k,
                value: v,
            }))
            : [],
        cacheDir: initialValue?.cache_dir ?? initialConfig?.config.cache_dir,
        pure: initialValue?.pure ?? initialConfig?.config.pure ?? true,
        showTrace: initialValue?.show_trace ?? initialConfig?.config.show_trace ?? false,
    })

    const [errors, setErrors] = useState<Record<string, string>>({})

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {}

        if (formState.useFlake) {
            if (!formState.flakeUri?.trim()) {
                newErrors.flakeUri = 'Flake URI is required when using Flake'
            }
        } else {
            if (!formState.nixExpression?.trim()) {
                newErrors.nixExpression = 'Nix expression is required'
            }
        }

        if (formState.attributes.length === 0) {
            newErrors.attributes = 'At least one attribute must be specified'
        }

        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleAddAttribute = () => {
        setFormState((prev) => ({
            ...prev,
            attributes: [...prev.attributes, ''],
        }))
    }

    const handleRemoveAttribute = (index: number) => {
        setFormState((prev) => ({
            ...prev,
            attributes: prev.attributes.filter((_, i) => i !== index),
        }))
    }

    const handleAttributeChange = (index: number, value: string) => {
        setFormState((prev) => ({
            ...prev,
            attributes: prev.attributes.map((a, i) => (i === index ? value : a)),
        }))
    }

    const buildRequest = (): CreateNixConfigRequest => {
        const outputs = formState.outputs.reduce(
            (acc, { key, value }) => {
                if (key) acc[key] = value
                return acc
            },
            {} as Record<string, string>
        )

        return {
            build_id: buildId || '',
            nix_expression: formState.useFlake ? undefined : formState.nixExpression,
            flake_uri: formState.useFlake ? formState.flakeUri : undefined,
            attributes: formState.attributes.filter((a) => a.trim()),
            outputs: Object.keys(outputs).length > 0 ? outputs : undefined,
            cache_dir: formState.cacheDir,
            pure: formState.pure,
            show_trace: formState.showTrace,
        }
    }

    useEffect(() => {
        onChangeRef.current = onChange
    }, [onChange])

    useEffect(() => {
        if (!initialValue) {
            return
        }
        setFormState({
            nixExpression: initialValue.nix_expression ?? initialConfig?.config.nix_expression,
            flakeUri: initialValue.flake_uri ?? initialConfig?.config.flake_uri,
            useFlake: !!(initialValue.flake_uri ?? initialConfig?.config.flake_uri),
            attributes: initialValue.attributes || initialConfig?.config.attributes || [],
            outputs: initialValue.outputs
                ? Object.entries(initialValue.outputs).map(([k, v]) => ({
                    key: k,
                    value: v,
                }))
                : initialConfig?.config.outputs
                    ? Object.entries(initialConfig.config.outputs).map(([k, v]) => ({
                        key: k,
                        value: v,
                    }))
                    : [],
            cacheDir: initialValue.cache_dir ?? initialConfig?.config.cache_dir,
            pure: initialValue.pure ?? initialConfig?.config.pure ?? true,
            showTrace: initialValue.show_trace ?? initialConfig?.config.show_trace ?? false,
        })
    }, [initialValue, initialConfig])

    useEffect(() => {
        if (onChangeRef.current) {
            onChangeRef.current(buildRequest())
        }
    }, [formState, buildId])

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()

        if (!validateForm()) {
            return
        }

        const request = buildRequest()

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
                <h3 className="text-sm font-semibold text-blue-900 dark:text-blue-200 mb-2">ℹ️ Nix Configuration</h3>
                <p className="text-sm text-blue-700 dark:text-blue-300">
                    Build using Nix expressions or Flakes. Define reproducible, declarative builds with full dependency management.
                </p>
            </div>

            {/* Nix Source Selection */}
            <div className="border border-gray-300 dark:border-gray-600 rounded-lg p-4 space-y-3">
                <p className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">Nix Source *</p>
                <div className="space-y-3">
                    <label className="flex items-center gap-3 cursor-pointer">
                        <input
                            type="radio"
                            name="nixSource"
                            checked={!formState.useFlake}
                            onChange={() => setFormState({ ...formState, useFlake: false })}
                            className="text-blue-600 focus:ring-blue-500 dark:bg-gray-700 dark:focus:ring-blue-400"
                        />
                        <span className="text-sm text-gray-700 dark:text-gray-300">Traditional Nix Expression</span>
                    </label>
                    <label className="flex items-center gap-3 cursor-pointer">
                        <input
                            type="radio"
                            name="nixSource"
                            checked={formState.useFlake}
                            onChange={() => setFormState({ ...formState, useFlake: true })}
                            className="text-blue-600 focus:ring-blue-500 dark:bg-gray-700 dark:focus:ring-blue-400"
                        />
                        <span className="text-sm text-gray-700 dark:text-gray-300">Nix Flake</span>
                    </label>
                </div>
            </div>

            {/* Nix Expression */}
            {!formState.useFlake && (
                <div>
                    <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                        Nix Expression *
                    </label>
                    <textarea
                        value={formState.nixExpression || ''}
                        onChange={(e) => {
                            setFormState({ ...formState, nixExpression: e.target.value })
                            setErrors({ ...errors, nixExpression: '' })
                        }}
                        placeholder={`let
  pkgs = import <nixpkgs> {};
in
pkgs.stdenv.mkDerivation {
  name = "myapp";
  src = ./.;
  buildInputs = [ pkgs.nodejs pkgs.yarn ];
  buildPhase = "yarn build";
}`}
                        rows={10}
                        className={`
                            w-full px-3 py-2 border rounded-md font-mono text-sm
                            focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400
                            ${errors.nixExpression ? 'border-red-300 dark:border-red-600' : 'border-gray-300 dark:border-gray-600'}
                        `}
                    />
                    {errors.nixExpression && (
                        <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.nixExpression}</p>
                    )}
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        Enter your Nix expression that defines how to build your project
                    </p>
                </div>
            )}

            {/* Flake URI */}
            {formState.useFlake && (
                <div>
                    <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                        Flake URI *
                    </label>
                    <input
                        type="text"
                        value={formState.flakeUri || ''}
                        onChange={(e) => {
                            setFormState({ ...formState, flakeUri: e.target.value })
                            setErrors({ ...errors, flakeUri: '' })
                        }}
                        placeholder="github:owner/repo"
                        className={`
                            w-full px-3 py-2 border rounded-md text-sm
                            focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400
                            ${errors.flakeUri ? 'border-red-300 dark:border-red-600' : 'border-gray-300 dark:border-gray-600'}
                        `}
                    />
                    {errors.flakeUri && <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.flakeUri}</p>}
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        Flake URI (e.g., github:owner/repo, path:./flake.nix)
                    </p>
                </div>
            )}

            {/* Attributes */}
            <div>
                <div className="flex items-center justify-between mb-3">
                    <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300">
                        Build Attributes *
                    </label>
                    <button
                        type="button"
                        onClick={handleAddAttribute}
                        className="text-xs px-2 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 rounded hover:bg-blue-200 dark:hover:bg-blue-800 transition-colors"
                    >
                        + Add Attribute
                    </button>
                </div>
                <div className="space-y-2">
                    {formState.attributes.map((attr, index) => (
                        <div key={index} className="flex gap-2">
                            <input
                                type="text"
                                value={attr}
                                onChange={(e) => handleAttributeChange(index, e.target.value)}
                                placeholder="e.g., packages.x86_64-linux.default"
                                className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                            />
                            <button
                                type="button"
                                onClick={() => handleRemoveAttribute(index)}
                                className="px-3 py-2 bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300 rounded-md text-sm hover:bg-red-200 dark:hover:bg-red-800 transition-colors"
                            >
                                Remove
                            </button>
                        </div>
                    ))}
                </div>
                {errors.attributes && <p className="text-sm text-red-600 dark:text-red-400 mt-2">{errors.attributes}</p>}
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                    Attributes to build (e.g., packages.x86_64-linux.myapp)
                </p>
            </div>

            {/* Outputs */}
            <KeyValueForm
                title="Build Outputs (Optional)"
                items={formState.outputs}
                onItemsChange={(outputs) => setFormState({ ...formState, outputs })}
                keyPlaceholder="Output name"
                valuePlaceholder="Output path"
            />

            {/* Cache Directory */}
            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Cache Directory
                </label>
                <input
                    type="text"
                    value={formState.cacheDir || ''}
                    onChange={(e) => setFormState({ ...formState, cacheDir: e.target.value || undefined })}
                    placeholder="/tmp/nix-cache"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Directory to cache Nix store outputs (optional)
                </p>
            </div>

            {/* Options */}
            <div className="space-y-3 border border-gray-300 dark:border-gray-600 rounded-lg p-4 dark:bg-gray-800">
                <p className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">Build Options</p>
                <label className="flex items-center gap-3 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={formState.pure}
                        onChange={(e) => setFormState({ ...formState, pure: e.target.checked })}
                        className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 dark:border-gray-600 rounded dark:bg-gray-700"
                    />
                    <span className="text-sm text-gray-700 dark:text-gray-300">Pure evaluation (reproducible builds)</span>
                </label>
                <label className="flex items-center gap-3 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={formState.showTrace}
                        onChange={(e) => setFormState({ ...formState, showTrace: e.target.checked })}
                        className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 dark:border-gray-600 rounded dark:bg-gray-700"
                    />
                    <span className="text-sm text-gray-700 dark:text-gray-300">Show trace (detailed error messages)</span>
                </label>
            </div>

            {/* Submit Button */}
            <div className="flex gap-3 pt-4">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="
                        px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium
                        hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed
                        dark:bg-blue-700 dark:hover:bg-blue-600 dark:disabled:bg-gray-600
                        transition-colors
                    "
                >
                    {isLoading ? 'Creating Configuration...' : 'Create Configuration'}
                </button>
            </div>
        </form>
    )
}

export default NixConfigForm
