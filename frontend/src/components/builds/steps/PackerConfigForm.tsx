import KeyValueForm from '@/components/common/KeyValueForm';
import { PACKER_ON_ERROR_MODES } from '@/lib/buildMethods';
import { packerTargetProfileService } from '@/services/packerTargetProfileService';
import { PackerTargetProfile } from '@/types';
import { CreatePackerConfigRequest, PackerConfig } from '@/types/buildConfig';
import React, { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';

interface PackerConfigFormProps {
    buildId?: string;
    onSubmit?: (config: CreatePackerConfigRequest) => void;
    isLoading?: boolean;
    error?: string;
    initialConfig?: PackerConfig;
    onChange?: (config: CreatePackerConfigRequest) => void; // For wizard mode
}

interface FormState {
    template: string;
    packerTargetProfileId: string;
    variables: Array<{ key: string; value: string }>;
    buildVars: Array<{ key: string; value: string }>;
    onError: string;
    parallel: boolean;
}

export const PackerConfigForm: React.FC<PackerConfigFormProps> = ({
    buildId,
    onSubmit,
    isLoading = false,
    error,
    initialConfig,
    onChange,
}) => {
    const [profiles, setProfiles] = useState<PackerTargetProfile[]>([]);
    const [profilesLoading, setProfilesLoading] = useState(false);
    const [formState, setFormState] = useState<FormState>({
        template: initialConfig?.config.template || '',
        packerTargetProfileId: initialConfig?.config.packer_target_profile_id || '',
        variables: initialConfig?.config.variables
            ? Object.entries(initialConfig.config.variables).map(([k, v]) => ({
                key: k,
                value: String(v),
            }))
            : [],
        buildVars: initialConfig?.config.build_vars
            ? Object.entries(initialConfig.config.build_vars).map(([k, v]) => ({
                key: k,
                value: v,
            }))
            : [],
        onError: initialConfig?.config.on_error || 'cleanup',
        parallel: initialConfig?.config.parallel || false,
    });

    const [templateError, setTemplateError] = useState<string>('');
    const [profileError, setProfileError] = useState<string>('');

    const validProfiles = useMemo(
        () => profiles.filter((p) => p.validation_status === 'valid'),
        [profiles]
    );

    useEffect(() => {
        let isMounted = true;
        const loadProfiles = async () => {
            try {
                setProfilesLoading(true);
                const data = await packerTargetProfileService.list();
                if (!isMounted) return;
                setProfiles(data || []);
            } catch (error: any) {
                if (!isMounted) return;
                toast.error(error?.message || 'Failed to load Packer target profiles');
            } finally {
                if (isMounted) setProfilesLoading(false);
            }
        };
        loadProfiles();
        return () => {
            isMounted = false;
        };
    }, []);

    const handleTemplateChange = (value: string) => {
        setFormState({ ...formState, template: value });
        // Basic validation
        if (value.trim() && !value.trim().startsWith('{')) {
            setTemplateError('Template must be valid JSON starting with {');
        } else {
            setTemplateError('');
        }
    };

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();

        if (!formState.template.trim()) {
            setTemplateError('Template is required');
            return;
        }

        if (templateError) {
            return;
        }
        if (!formState.packerTargetProfileId.trim()) {
            setProfileError('Target profile is required');
            return;
        }
        setProfileError('');

        const variables = formState.variables.reduce(
            (acc, { key, value }) => {
                if (key) acc[key] = value;
                return acc;
            },
            {} as Record<string, unknown>
        );
        const buildVars = formState.buildVars.reduce(
            (acc, { key, value }) => {
                if (key) acc[key] = value;
                return acc;
            },
            {} as Record<string, string>
        );

        const request: CreatePackerConfigRequest = {
            build_id: buildId || '',
            template: formState.template,
            packer_target_profile_id: formState.packerTargetProfileId || undefined,
            variables: Object.keys(variables).length > 0 ? variables : undefined,
            build_vars: Object.keys(buildVars).length > 0 ? buildVars : undefined,
            on_error: formState.onError || 'cleanup',
            parallel: formState.parallel,
        };

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

            {/* Template */}
            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Packer Template *
                </label>
                <textarea
                    value={formState.template}
                    onChange={(e) => handleTemplateChange(e.target.value)}
                    placeholder={`{
  "builders": [{
    "type": "amazon-ebs",
    ...
  }],
  "provisioners": [...]
}`}
                    rows={10}
                    className={`
            w-full px-3 py-2 border rounded-md font-mono text-sm
            focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400
            ${templateError ? 'border-red-300 dark:border-red-600' : 'border-gray-300 dark:border-gray-600'}
          `}
                />
                {templateError && <p className="text-sm text-red-600 dark:text-red-400 mt-1">{templateError}</p>}
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Enter a valid Packer template in JSON format
                </p>
            </div>

            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Target Profile ID *
                </label>
                <select
                    value={formState.packerTargetProfileId}
                    onChange={(e) => {
                        setFormState({ ...formState, packerTargetProfileId: e.target.value });
                        if (e.target.value.trim()) setProfileError('');
                    }}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                    disabled={profilesLoading}
                >
                    <option value="">
                        {profilesLoading ? 'Loading profiles...' : 'Select a validated target profile'}
                    </option>
                    {validProfiles.map((profile) => (
                        <option key={profile.id} value={profile.id}>
                            {profile.name} ({profile.provider}) - {profile.id}
                        </option>
                    ))}
                </select>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Required: choose a tenant-entitled profile with `valid` status.
                </p>
                {!profilesLoading && validProfiles.length === 0 && (
                    <p className="text-xs text-amber-700 dark:text-amber-300 mt-1">
                        No validated target profiles found. Ask an admin to validate at least one profile.
                    </p>
                )}
                {profileError && <p className="text-sm text-red-600 dark:text-red-400 mt-1">{profileError}</p>}
            </div>

            {/* Template Variables */}
            <KeyValueForm
                title="Template Variables"
                items={formState.variables}
                onItemsChange={(variables) => setFormState({ ...formState, variables })}
                keyPlaceholder="Variable name"
                valuePlaceholder="Variable value"
            />

            {/* Build Variables */}
            <KeyValueForm
                title="Build Variables (-var flag)"
                items={formState.buildVars}
                onItemsChange={(buildVars) => setFormState({ ...formState, buildVars })}
                keyPlaceholder="Build variable"
                valuePlaceholder="Value"
            />

            {/* On Error Mode */}
            <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    On Error Behavior
                </label>
                <select
                    value={formState.onError}
                    onChange={(e) => setFormState({ ...formState, onError: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white dark:focus:ring-blue-400"
                >
                    {PACKER_ON_ERROR_MODES.map((mode) => (
                        <option key={mode.value} value={mode.value}>
                            {mode.label}
                        </option>
                    ))}
                </select>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Determines what Packer does if a build fails
                </p>
            </div>

            {/* Parallel Builds */}
            <div className="flex items-center gap-3">
                <input
                    type="checkbox"
                    id="parallel"
                    checked={formState.parallel}
                    onChange={(e) => setFormState({ ...formState, parallel: e.target.checked })}
                    className="h-4 w-4 text-blue-600 border border-gray-300 dark:border-gray-600 rounded focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:focus:ring-blue-400"
                />
                <label htmlFor="parallel" className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Enable Parallel Builds
                </label>
            </div>

            {/* Submit Button */}
            <button
                type="submit"
                disabled={isLoading || !!templateError}
                className={`
          w-full px-4 py-2 rounded-md font-medium text-white transition-colors
          ${isLoading || templateError
                        ? 'bg-gray-400 dark:bg-gray-600 cursor-not-allowed'
                        : 'bg-blue-600 hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-600'
                    }
        `}
            >
                {isLoading ? 'Creating Configuration...' : 'Create Packer Configuration'}
            </button>
        </form>
    );
};

export default PackerConfigForm;
