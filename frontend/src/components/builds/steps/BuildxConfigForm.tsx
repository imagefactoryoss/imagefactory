import DockerfileInput from '@/components/common/DockerfileInput';
import HelpTooltip from '@/components/common/HelpTooltip';
import KeyValueForm from '@/components/common/KeyValueForm';
import { BUILDX_COMMON_PLATFORMS } from '@/lib/buildMethods';
import { BuildxConfig, CreateBuildxConfigRequest } from '@/types/buildConfig';
import React, { useEffect, useRef, useState } from 'react';

interface BuildxConfigFormProps {
    buildId?: string;
    onSubmit?: (config: CreateBuildxConfigRequest) => void;
    isLoading?: boolean;
    error?: string;
    initialConfig?: BuildxConfig;
    initialValue?: Partial<CreateBuildxConfigRequest>;
    onChange?: (config: CreateBuildxConfigRequest) => void; // For wizard mode
    registryAuthOptions?: Array<{ id: string; label: string }>;
    registryAuthLoading?: boolean;
    registryAuthError?: string | null;
    selectedRegistryAuthId?: string;
    onRegistryAuthChange?: (registryAuthId?: string) => void;
    onTestRegistryAuth?: (registryAuthId: string, registryRepo: string) => Promise<{ success: boolean; message: string }>;
    registryAuthValidationError?: string;
    imageTags?: string[];
    onImageTagsChange?: (tags: string[]) => void;
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
    platforms: string[];
    buildArgs: Array<{ key: string; value: string }>;
    secrets: Array<{ key: string; value: string }>;
    cache: {
        from?: string;
        to?: string;
    };
    noCache: boolean;
    outputs: string[];
}

export const BuildxConfigForm: React.FC<BuildxConfigFormProps> = ({
    buildId,
    onSubmit,
    isLoading = false,
    error,
    initialConfig,
    initialValue,
    onChange,
    registryAuthOptions = [],
    registryAuthLoading = false,
    registryAuthError = null,
    selectedRegistryAuthId,
    onRegistryAuthChange,
    onTestRegistryAuth,
    registryAuthValidationError,
    imageTags = [],
    onImageTagsChange,
}) => {
    const onChangeRef = useRef(onChange)
    const [formState, setFormState] = useState<FormState>({
        dockerfile: initialValue?.dockerfile
            ? (typeof initialValue.dockerfile === 'string'
                ? { source: 'path', path: initialValue.dockerfile }
                : initialValue.dockerfile)
            : typeof initialConfig?.config.dockerfile === 'string'
                ? { source: 'path', path: initialConfig.config.dockerfile }
                : initialConfig?.config.dockerfile || { source: 'path', path: 'Dockerfile' },
        buildContext: initialValue?.build_context || initialConfig?.config.build_context || '.',
        registryRepo: initialValue?.registry_repo || initialConfig?.config.registry_repo || '',
        platforms: initialValue?.platforms || initialConfig?.config.platforms || [],
        buildArgs: initialValue?.build_args
            ? Object.entries(initialValue.build_args).map(([k, v]) => ({
                key: k,
                value: v,
            }))
            : initialConfig?.config.build_args
                ? Object.entries(initialConfig.config.build_args).map(([k, v]) => ({
                    key: k,
                    value: v,
                }))
                : [],
        secrets: initialValue?.secrets
            ? Object.entries(initialValue.secrets).map(([k, v]) => ({
                key: k,
                value: v,
            }))
            : initialConfig?.config.secrets
                ? Object.entries(initialConfig.config.secrets).map(([k, v]) => ({
                    key: k,
                    value: v,
                }))
                : [],
        cache: {
            from: initialConfig?.config.cache?.from,
            to: initialConfig?.config.cache?.to,
        },
        noCache: initialConfig?.config.no_cache || false,
        outputs: initialConfig?.config.outputs || [],
    });

    const [errors, setErrors] = useState<Record<string, string>>({});
    const [authTestStatus, setAuthTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle');
    const [authTestMessage, setAuthTestMessage] = useState<string | null>(null);
    const showRegistryAuthSection = !!onRegistryAuthChange
        || !!selectedRegistryAuthId
        || !!registryAuthValidationError
        || !!registryAuthError
        || registryAuthLoading
        || registryAuthOptions.length > 0
        || !!onTestRegistryAuth;
    const showImageTagsSection = !!onImageTagsChange || imageTags.length > 0;

    const togglePlatform = (platform: string) => {
        setFormState((prev) => ({
            ...prev,
            platforms: prev.platforms.includes(platform)
                ? prev.platforms.filter((p) => p !== platform)
                : [...prev.platforms, platform],
        }));
    };

    const handleAddOutput = () => {
        setFormState((prev) => ({
            ...prev,
            outputs: [...prev.outputs, ''],
        }));
    };

    const handleRemoveOutput = (index: number) => {
        setFormState((prev) => ({
            ...prev,
            outputs: prev.outputs.filter((_, i) => i !== index),
        }));
    };

    const handleOutputChange = (index: number, value: string) => {
        setFormState((prev) => ({
            ...prev,
            outputs: prev.outputs.map((o, i) => (i === index ? value : o)),
        }));
    };

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {};

        // Validate dockerfile based on source
        if (formState.dockerfile.source === 'path' && !formState.dockerfile.path?.trim()) {
            newErrors.dockerfile = 'Dockerfile path is required';
        } else if (formState.dockerfile.source === 'content' && !formState.dockerfile.content?.trim()) {
            newErrors.dockerfile = 'Dockerfile content is required';
        } else if (formState.dockerfile.source === 'upload' && !formState.dockerfile.content?.trim()) {
            newErrors.dockerfile = 'Please upload a Dockerfile';
        }

        if (formState.platforms.length === 0) {
            newErrors.platforms = 'At least one platform must be selected';
        }
        if (!formState.registryRepo.trim()) {
            newErrors.registryRepo = 'Registry repository is required';
        }

        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const buildRequestFromState = (): CreateBuildxConfigRequest => {
        const buildArgs = formState.buildArgs.reduce(
            (acc, { key, value }) => {
                if (key) acc[key] = value;
                return acc;
            },
            {} as Record<string, string>
        );

        const secrets = formState.secrets.reduce(
            (acc, { key, value }) => {
                if (key) acc[key] = value;
                return acc;
            },
            {} as Record<string, string>
        );

        return {
            build_id: buildId || '',
            dockerfile: formState.dockerfile,
            build_context: formState.buildContext,
            registry_repo: formState.registryRepo.trim(),
            platforms: formState.platforms,
            build_args: Object.keys(buildArgs).length > 0 ? buildArgs : undefined,
            secrets: Object.keys(secrets).length > 0 ? secrets : undefined,
            cache:
                formState.cache.from || formState.cache.to
                    ? formState.cache
                    : undefined,
            no_cache: formState.noCache,
            outputs: formState.outputs.filter((o) => o.trim()) || undefined,
        };
    };

    useEffect(() => {
        onChangeRef.current = onChange
    }, [onChange])

    useEffect(() => {
        if (onChangeRef.current) {
            onChangeRef.current(buildRequestFromState())
        }
    }, [formState, buildId])

    useEffect(() => {
        if (!initialValue) return
        setFormState((prev) => {
            const next: FormState = {
                dockerfile: initialValue.dockerfile
                    ? (typeof initialValue.dockerfile === 'string'
                        ? { source: 'path', path: initialValue.dockerfile }
                        : initialValue.dockerfile)
                    : prev.dockerfile,
                buildContext: initialValue.build_context ?? prev.buildContext,
                registryRepo: initialValue.registry_repo ?? prev.registryRepo,
                platforms: initialValue.platforms ?? prev.platforms,
                buildArgs: initialValue.build_args
                    ? Object.entries(initialValue.build_args).map(([k, v]) => ({ key: k, value: v }))
                    : prev.buildArgs,
                secrets: initialValue.secrets
                    ? Object.entries(initialValue.secrets).map(([k, v]) => ({ key: k, value: v }))
                    : prev.secrets,
                cache: prev.cache,
                noCache: prev.noCache,
                outputs: prev.outputs,
            }

            const unchanged =
                JSON.stringify(prev.dockerfile) === JSON.stringify(next.dockerfile) &&
                prev.buildContext === next.buildContext &&
                prev.registryRepo === next.registryRepo &&
                JSON.stringify(prev.platforms) === JSON.stringify(next.platforms) &&
                JSON.stringify(prev.buildArgs) === JSON.stringify(next.buildArgs) &&
                JSON.stringify(prev.secrets) === JSON.stringify(next.secrets)

            return unchanged ? prev : next
        })
    }, [initialValue])

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();

        if (!validateForm()) return;

        const request = buildRequestFromState();

        if (onChange) {
            onChange(request);
        } else if (onSubmit) {
            onSubmit(request);
        }
    };

    const handleTestRegistryAuth = async () => {
        if (!onTestRegistryAuth || !selectedRegistryAuthId || !formState.registryRepo.trim()) return;
        setAuthTestStatus('testing');
        setAuthTestMessage(null);
        try {
            const result = await onTestRegistryAuth(selectedRegistryAuthId, formState.registryRepo.trim());
            setAuthTestStatus(result.success ? 'success' : 'error');
            setAuthTestMessage(result.message);
        } catch (err) {
            setAuthTestStatus('error');
            setAuthTestMessage(err instanceof Error ? err.message : 'Failed to test registry authentication');
        }
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-6 max-w-2xl">
            {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                </div>
            )}

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
                        Build Context
                        <HelpTooltip text='Dockerfile paths are resolved from the build context root. Example: if Dockerfile uses "COPY cmd ./cmd", set Build Context to "examples/image-factory-user-docs" when "cmd/" exists there.' />
                    </span>
                </label>
                <input
                    type="text"
                    value={formState.buildContext}
                    onChange={(e) => setFormState({ ...formState, buildContext: e.target.value })}
                    placeholder="."
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Relative path, absolute path, git URL, or S3 URL. COPY/ADD paths in Dockerfile are resolved from this context root.
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
                        setFormState({ ...formState, registryRepo: e.target.value });
                        setAuthTestStatus('idle');
                        setAuthTestMessage(null);
                    }}
                    placeholder="registry.gitlab.com/group/my-app"
                    className={`
                        w-full px-3 py-2 border rounded-md text-sm
                        focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400
                        ${errors.registryRepo ? 'border-red-300 dark:border-red-600' : 'border-gray-300 dark:border-gray-600'}
                    `}
                />
                {errors.registryRepo && (
                    <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.registryRepo}</p>
                )}
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Full image reference repository used as the Buildx target (tag comes from build tags or defaults).
                </p>
            </div>

            {showImageTagsSection && (
                <div>
                    <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                        Image Tags
                    </label>
                    <input
                        type="text"
                        value={imageTags.join(', ')}
                        onChange={(e) =>
                            onImageTagsChange?.(
                                e.target.value
                                    .split(',')
                                    .map((tag) => tag.trim())
                                    .filter((tag) => tag.length > 0)
                            )
                        }
                        placeholder="latest, v1.0.0, production"
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        Comma-separated tags. The first tag is used as the push tag for the registry repository.
                    </p>
                </div>
            )}

            {showRegistryAuthSection && (
                <div>
                    <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                        <span className="inline-flex items-center gap-2">
                            Registry Authentication *
                            <HelpTooltip text="Registry auth precedence: selected auth, then project default registry auth, then tenant default." />
                        </span>
                    </label>
                    <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
                        <select
                            value={selectedRegistryAuthId || ''}
                            onChange={(e) => {
                                onRegistryAuthChange?.(e.target.value || undefined);
                                setAuthTestStatus('idle');
                                setAuthTestMessage(null);
                            }}
                            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white dark:focus:ring-blue-400"
                            disabled={registryAuthLoading}
                        >
                            <option value="">{registryAuthLoading ? 'Loading registry auth...' : 'Select registry authentication'}</option>
                            {registryAuthOptions.map((option) => (
                                <option key={option.id} value={option.id}>
                                    {option.label}
                                </option>
                            ))}
                        </select>
                        <button
                            type="button"
                            onClick={handleTestRegistryAuth}
                            disabled={
                                authTestStatus === 'testing' ||
                                !selectedRegistryAuthId ||
                                !formState.registryRepo.trim() ||
                                !onTestRegistryAuth
                            }
                            className="inline-flex shrink-0 items-center justify-center rounded-md border border-green-300 bg-green-50 px-3 py-2 text-sm font-medium text-green-800 hover:bg-green-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-green-700 dark:bg-green-900/20 dark:text-green-200 dark:hover:bg-green-900/30"
                        >
                            {authTestStatus === 'testing' ? 'Testing...' : 'Test Auth'}
                        </button>
                    </div>
                    {registryAuthValidationError && (
                        <p className="mt-1 text-sm text-red-600 dark:text-red-400">{registryAuthValidationError}</p>
                    )}
                    {registryAuthError && (
                        <p className="mt-1 text-sm text-red-600 dark:text-red-400">{registryAuthError}</p>
                    )}
                    {!registryAuthLoading && registryAuthOptions.length === 0 && (
                        <p className="mt-1 text-sm text-amber-700 dark:text-amber-300">
                            No registry authentication found for this project/tenant. Create one before creating builds.
                        </p>
                    )}
                    {authTestMessage && (
                        <p className={`mt-1 text-sm ${authTestStatus === 'success' ? 'text-green-700 dark:text-green-300' : 'text-red-600 dark:text-red-400'}`}>
                            {authTestMessage}
                        </p>
                    )}
                </div>
            )}

            {/* Platforms */}
            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">
                    Target Platforms *
                </label>
                {errors.platforms && (
                    <p className="text-sm text-red-600 dark:text-red-400 mb-2">{errors.platforms}</p>
                )}
                <div className="grid grid-cols-2 gap-3">
                    {BUILDX_COMMON_PLATFORMS.map((platform) => (
                        <label key={platform} className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="checkbox"
                                checked={formState.platforms.includes(platform)}
                                onChange={() => togglePlatform(platform)}
                                className="h-4 w-4 text-blue-600 border border-gray-300 dark:border-gray-600 rounded focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:focus:ring-blue-400"
                            />
                            <span className="text-sm text-gray-700 dark:text-gray-300">{platform}</span>
                        </label>
                    ))}
                </div>
            </div>

            {/* Build Args */}
            <KeyValueForm
                title="Build Arguments"
                items={formState.buildArgs}
                onItemsChange={(buildArgs) => setFormState({ ...formState, buildArgs })}
                keyPlaceholder="ARG name"
                valuePlaceholder="Value"
            />

            {/* Secrets */}
            <KeyValueForm
                title="Secrets (Build secrets)"
                items={formState.secrets}
                onItemsChange={(secrets) => setFormState({ ...formState, secrets })}
                keyPlaceholder="Secret name"
                valuePlaceholder="Secret value"
                valueType="password"
            />

            {/* Cache Settings */}
            <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 bg-gray-50 dark:bg-gray-800/50">
                <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">Cache Settings</h3>
                <div className="space-y-3">
                    <div>
                        <label className="block text-sm text-gray-700 dark:text-gray-300 mb-1">Cache From</label>
                        <input
                            type="text"
                            value={formState.cache.from || ''}
                            onChange={(e) =>
                                setFormState({
                                    ...formState,
                                    cache: { ...formState.cache, from: e.target.value },
                                })
                            }
                            placeholder="type=registry,ref=myregistry.com/myimage:latest"
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                        />
                    </div>
                    <div>
                        <label className="block text-sm text-gray-700 dark:text-gray-300 mb-1">Cache To</label>
                        <input
                            type="text"
                            value={formState.cache.to || ''}
                            onChange={(e) =>
                                setFormState({
                                    ...formState,
                                    cache: { ...formState.cache, to: e.target.value },
                                })
                            }
                            placeholder="type=registry,ref=myregistry.com/myimage:latest"
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                        />
                    </div>
                </div>
            </div>

            {/* No Cache */}
            <div className="flex items-center gap-3">
                <input
                    type="checkbox"
                    id="noCache"
                    checked={formState.noCache}
                    onChange={(e) => setFormState({ ...formState, noCache: e.target.checked })}
                    className="h-4 w-4 text-blue-600 border border-gray-300 dark:border-gray-600 rounded focus:ring-2 focus:ring-blue-500 dark:bg-gray-800"
                />
                <label htmlFor="noCache" className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Disable Cache
                </label>
            </div>

            {/* Outputs */}
            <div>
                <div className="flex items-center justify-between mb-2">
                    <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300">Outputs</label>
                    <button
                        type="button"
                        onClick={handleAddOutput}
                        className="text-xs text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
                    >
                        + Add Output
                    </button>
                </div>
                <div className="space-y-2">
                    {formState.outputs.map((output, index) => (
                        <div key={index} className="flex gap-2">
                            <input
                                type="text"
                                value={output}
                                onChange={(e) => handleOutputChange(index, e.target.value)}
                                placeholder="type=registry,push=true"
                                className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                            />
                            <button
                                type="button"
                                onClick={() => handleRemoveOutput(index)}
                                className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 font-medium text-sm px-2"
                            >
                                Remove
                            </button>
                        </div>
                    ))}
                </div>
            </div>

            {/* Submit Button */}
            <button
                type="submit"
                disabled={isLoading}
                className={`
          w-full px-4 py-2 rounded-md font-medium text-white transition-colors
          ${isLoading
                        ? 'bg-gray-400 dark:bg-gray-600 cursor-not-allowed'
                        : 'bg-blue-600 hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-600'
                    }
        `}
            >
                {isLoading ? 'Creating Configuration...' : 'Create Buildx Configuration'}
            </button>
        </form>
    );
};

export default BuildxConfigForm;
