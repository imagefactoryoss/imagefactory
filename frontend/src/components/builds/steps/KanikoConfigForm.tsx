import DockerfileInput from '@/components/common/DockerfileInput';
import HelpTooltip from '@/components/common/HelpTooltip';
import KeyValueForm from '@/components/common/KeyValueForm';
import { CreateKanikoConfigRequest, KanikoConfig } from '@/types/buildConfig';
import React, { useEffect, useRef, useState } from 'react';

interface KanikoConfigFormProps {
    buildId?: string;
    onSubmit?: (config: CreateKanikoConfigRequest) => void;
    isLoading?: boolean;
    error?: string;
    initialConfig?: KanikoConfig;
    initialValue?: Partial<CreateKanikoConfigRequest>;
    onChange?: (config: CreateKanikoConfigRequest) => void; // For wizard mode
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
    cacheRepo?: string;
    buildArgs: Array<{ key: string; value: string }>;
    skipUnusedStages: boolean;
}

export const KanikoConfigForm: React.FC<KanikoConfigFormProps> = ({
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
        cacheRepo: initialValue?.cache_repo || initialConfig?.config.cache_repo,
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
        skipUnusedStages: initialConfig?.config.skip_unused_stages || false,
    });

    const [errors, setErrors] = useState<Record<string, string>>({});
    const [registryStatus, setRegistryStatus] = useState<'idle' | 'checking' | 'valid' | 'invalid'>(
        'idle'
    );
    const [authTestStatus, setAuthTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle');
    const [authTestMessage, setAuthTestMessage] = useState<string | null>(null);
    const isWizardMode = !!onChange && !onSubmit;

    const validateRegistryRepo = (repo: string): boolean => {
        // Basic registry repo validation
        // ECR: 123456789.dkr.ecr.us-east-1.amazonaws.com/repo-name
        // Docker Hub: docker.io/username/repo or just username/repo
        // GCR: gcr.io/project-id/repo-name
        const registryPattern = /^([a-z0-9-]+\.)*[a-z0-9-]+\.[a-z]{2,}\/[a-z0-9\-_/]+$/;
        return registryPattern.test(repo.toLowerCase());
    };

    const handleRegistryRepoChange = (value: string) => {
        setFormState({ ...formState, registryRepo: value });
        setAuthTestStatus('idle');
        setAuthTestMessage(null);
        if (value.trim()) {
            if (validateRegistryRepo(value)) {
                setRegistryStatus('valid');
            } else {
                setRegistryStatus('invalid');
            }
        } else {
            setRegistryStatus('idle');
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

        if (!formState.registryRepo.trim()) {
            newErrors.registryRepo = 'Registry repository is required';
        } else if (!validateRegistryRepo(formState.registryRepo)) {
            newErrors.registryRepo = 'Invalid registry repository format';
        }

        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const buildRequestFromState = (): CreateKanikoConfigRequest => {
        const buildArgs = formState.buildArgs.reduce(
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
            registry_repo: formState.registryRepo,
            cache_repo: formState.cacheRepo,
            build_args: Object.keys(buildArgs).length > 0 ? buildArgs : undefined,
            skip_unused_stages: formState.skipUnusedStages,
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
                cacheRepo: initialValue.cache_repo ?? prev.cacheRepo,
                buildArgs: initialValue.build_args
                    ? Object.entries(initialValue.build_args).map(([k, v]) => ({ key: k, value: v }))
                    : prev.buildArgs,
                skipUnusedStages: prev.skipUnusedStages,
            }

            const unchanged =
                JSON.stringify(prev.dockerfile) === JSON.stringify(next.dockerfile) &&
                prev.buildContext === next.buildContext &&
                prev.registryRepo === next.registryRepo &&
                prev.cacheRepo === next.cacheRepo &&
                JSON.stringify(prev.buildArgs) === JSON.stringify(next.buildArgs)

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
            <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100">
                <p className="font-semibold">Kaniko Dockerfile precedence</p>
                <p className="mt-1">
                    For Tekton Kaniko builds, Dockerfile resolution is: inline content first, then repository path, then default
                    <code className="ml-1 rounded bg-amber-100 px-1 py-0.5 text-xs dark:bg-amber-900/60">Dockerfile</code>.
                </p>
            </div>

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

            {/* Registry Repository */}
            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Registry Repository *
                </label>
                <div className="relative">
                    <input
                        type="text"
                        value={formState.registryRepo}
                        onChange={(e) => handleRegistryRepoChange(e.target.value)}
                        placeholder="123456789.dkr.ecr.us-east-1.amazonaws.com/my-app"
                        className={`
              w-full px-3 py-2 border rounded-md text-sm
              focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400
              ${errors.registryRepo ? 'border-red-300 dark:border-red-600' : 'border-gray-300 dark:border-gray-600'}
            `}
                    />
                    {registryStatus === 'valid' && (
                        <span className="absolute right-3 top-2.5 text-green-600 dark:text-green-400 text-lg">✓</span>
                    )}
                    {registryStatus === 'invalid' && (
                        <span className="absolute right-3 top-2.5 text-red-600 dark:text-red-400 text-lg">✗</span>
                    )}
                </div>
                {errors.registryRepo && (
                    <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.registryRepo}</p>
                )}
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Format: registry.com/project/repository (e.g., registry.gitlab.com/group/my-app).
                </p>
            </div>

            {/* Image Tags */}
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

            {/* Registry Auth + Permission Test */}
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
                            registryStatus !== 'valid'
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
                        No registry auth found for this project/tenant. Create one to push images.
                    </p>
                )}
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Use the auth UUID from this list in YAML as `build_config.registry_auth_id`.
                </p>
                {authTestMessage && (
                    <p
                        className={`mt-1 text-sm ${authTestStatus === 'success'
                            ? 'text-green-700 dark:text-green-300'
                            : 'text-red-700 dark:text-red-300'
                            }`}
                    >
                        {authTestMessage}
                    </p>
                )}
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Test checks if selected credentials can request push access for this repository.
                </p>
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    If you do not select one, runtime falls back to project default registry auth, then tenant default.
                </p>
            </div>

            {/* Cache Repository */}
            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Cache Repository (Optional)
                </label>
                <input
                    type="text"
                    value={formState.cacheRepo || ''}
                    onChange={(e) => setFormState({ ...formState, cacheRepo: e.target.value })}
                    placeholder="123456789.dkr.ecr.us-east-1.amazonaws.com/cache"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Use separate repo for layer caching (recommended for multi-stage builds)
                </p>
            </div>

            {/* Build Args */}
            <KeyValueForm
                title="Build Arguments"
                items={formState.buildArgs}
                onItemsChange={(buildArgs) => setFormState({ ...formState, buildArgs })}
                keyPlaceholder="ARG name"
                valuePlaceholder="Value"
            />

            {/* Skip Unused Stages */}
            <div className="flex items-center gap-3">
                <input
                    type="checkbox"
                    id="skipUnusedStages"
                    checked={formState.skipUnusedStages}
                    onChange={(e) =>
                        setFormState({ ...formState, skipUnusedStages: e.target.checked })
                    }
                    className="h-4 w-4 text-blue-600 border border-gray-300 dark:border-gray-600 rounded focus:ring-2 focus:ring-blue-500 dark:bg-gray-800"
                />
                <label htmlFor="skipUnusedStages" className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Skip Unused Stages
                </label>
            </div>

            {/* Info Box */}
            <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <h4 className="text-sm font-semibold text-blue-900 dark:text-blue-200 mb-2">Kaniko Information</h4>
                <ul className="text-xs text-blue-800 dark:text-blue-300 space-y-1 list-disc list-inside">
                    <li>Builds container images without requiring Docker daemon</li>
                    <li>Executes build instructions in order, layer-by-layer</li>
                    <li>Supports multi-stage builds with optional cache repository</li>
                    <li>Registry credentials must be provided separately</li>
                </ul>
            </div>

            {isWizardMode ? (
                <div className="text-xs text-gray-500 dark:text-gray-400">
                    Configuration is saved automatically as you edit. Use the Next button to continue.
                </div>
            ) : (
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
                    {isLoading ? 'Creating Configuration...' : 'Create Kaniko Configuration'}
                </button>
            )}
        </form>
    );
};

export default KanikoConfigForm;
