import { configTemplateService } from '@/services/configTemplateService';
import React, { useEffect, useState } from 'react';
import { toast } from 'react-hot-toast';
import { BuildMethod, ConfigTemplate } from '../../types/buildConfig';

type UUID = string;

interface ConfigTemplatesPanelProps {
    projectId: UUID;
    onLoadTemplate: (template: ConfigTemplate) => void;
    currentBuildMethod?: string;
}

interface SaveTemplateModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (name: string, description: string) => void;
    isLoading: boolean;
}

const SaveTemplateModal: React.FC<SaveTemplateModalProps> = ({
    isOpen,
    onClose,
    onSave,
    isLoading,
}) => {
    const [name, setName] = useState('');
    const [description, setDescription] = useState('');

    const handleSave = () => {
        if (!name.trim()) {
            toast.error('Template name is required');
            return;
        }
        onSave(name, description);
        setName('');
        setDescription('');
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg shadow-lg p-6 w-96">
                <h2 className="text-xl font-semibold mb-4">Save as Template</h2>

                <div className="mb-4">
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                        Template Name *
                    </label>
                    <input
                        type="text"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        placeholder="e.g., Python FastAPI Build"
                        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                        disabled={isLoading}
                        maxLength={255}
                    />
                </div>

                <div className="mb-6">
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                        Description (optional)
                    </label>
                    <textarea
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        placeholder="Brief description of this template..."
                        rows={3}
                        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                        disabled={isLoading}
                        maxLength={1000}
                    />
                </div>

                <div className="flex gap-3 justify-end">
                    <button
                        onClick={onClose}
                        disabled={isLoading}
                        className="px-4 py-2 text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={handleSave}
                        disabled={isLoading || !name.trim()}
                        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {isLoading ? 'Saving...' : 'Save Template'}
                    </button>
                </div>
            </div>
        </div>
    );
};

interface TemplateItemProps {
    template: ConfigTemplate;
    onLoad: (template: ConfigTemplate) => void;
    onDelete: (id: UUID) => void;
    isDeleting: boolean;
}

const TemplateItem: React.FC<TemplateItemProps> = ({
    template,
    onLoad,
    onDelete,
    isDeleting,
}) => {
    return (
        <div className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
            <div className="flex items-start justify-between mb-2">
                <div className="flex-1">
                    <h3 className="font-semibold text-gray-900">{template.name}</h3>
                    {template.description && (
                        <p className="text-sm text-gray-600 mt-1">{template.description}</p>
                    )}
                </div>
                <span className="inline-block px-2 py-1 bg-blue-100 text-blue-700 text-xs font-semibold rounded">
                    {template.method}
                </span>
            </div>

            <div className="flex gap-2 mt-4">
                <button
                    onClick={() => onLoad(template)}
                    className="flex-1 px-3 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 transition-colors"
                >
                    Load
                </button>
                <button
                    onClick={() => onDelete(template.id)}
                    disabled={isDeleting}
                    className="px-3 py-2 border border-red-300 text-red-600 text-sm rounded-md hover:bg-red-50 disabled:opacity-50"
                >
                    {isDeleting ? 'Deleting...' : 'Delete'}
                </button>
            </div>

            <div className="text-xs text-gray-500 mt-2">
                Created: {new Date(template.created_at).toLocaleDateString()}
            </div>
        </div>
    );
};

export const ConfigTemplatesPanel: React.FC<ConfigTemplatesPanelProps> = ({
    projectId,
    onLoadTemplate,
    currentBuildMethod,
}) => {
    const [templates, setTemplates] = useState<ConfigTemplate[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isSaveModalOpen, setIsSaveModalOpen] = useState(false);
    const [isSaving, setIsSaving] = useState(false);
    const [isDeletingId, setIsDeletingId] = useState<UUID | null>(null);

    useEffect(() => {
        loadTemplates();
    }, [projectId]);

    const loadTemplates = async () => {
        try {
            setIsLoading(true);
            const data = await configTemplateService.listTemplates(projectId, 20, 0);
            setTemplates(data.templates || []);
        } catch (error) {
            console.error('Error loading templates:', error);
            toast.error('Failed to load templates');
        } finally {
            setIsLoading(false);
        }
    };

    const handleSaveTemplate = async (name: string, description: string) => {
        try {
            setIsSaving(true);

            // Get current build config from the page
            // This should be injected or retrieved from context
            const currentConfig = JSON.parse(
                localStorage.getItem('currentBuildConfig') || '{}'
            );

            const method = (currentBuildMethod || 'docker') as BuildMethod;

            await configTemplateService.saveTemplate({
                project_id: projectId,
                name,
                description,
                method,
                template_data: currentConfig,
                is_shared: false,
                is_public: false,
            });

            toast.success('Template saved successfully');
            setIsSaveModalOpen(false);
            await loadTemplates();
        } catch (error) {
            console.error('Error saving template:', error);
            toast.error(error instanceof Error ? error.message : 'Failed to save template');
        } finally {
            setIsSaving(false);
        }
    };

    const handleDeleteTemplate = async (templateId: UUID) => {
        if (!confirm('Are you sure you want to delete this template?')) {
            return;
        }

        try {
            setIsDeletingId(templateId);
            await configTemplateService.deleteTemplate(templateId);

            toast.success('Template deleted');
            await loadTemplates();
        } catch (error) {
            console.error('Error deleting template:', error);
            toast.error('Failed to delete template');
        } finally {
            setIsDeletingId(null);
        }
    };

    return (
        <div className="flex flex-col h-full">
            {/* Header */}
            <div className="flex items-center justify-between mb-4 pb-4 border-b border-gray-200">
                <h2 className="text-lg font-semibold text-gray-900">Build Templates</h2>
                <button
                    onClick={() => setIsSaveModalOpen(true)}
                    className="px-3 py-2 bg-green-600 text-white text-sm rounded-md hover:bg-green-700 transition-colors"
                >
                    Save Current as Template
                </button>
            </div>

            {/* Templates List */}
            <div className="flex-1 overflow-y-auto">
                {isLoading ? (
                    <div className="flex items-center justify-center h-full">
                        <p className="text-gray-500">Loading templates...</p>
                    </div>
                ) : templates.length === 0 ? (
                    <div className="flex items-center justify-center h-full">
                        <div className="text-center">
                            <p className="text-gray-500 mb-2">No templates yet</p>
                            <p className="text-sm text-gray-400">
                                Save your build configurations as templates for quick reuse
                            </p>
                        </div>
                    </div>
                ) : (
                    <div className="grid grid-cols-1 gap-3">
                        {templates.map((template) => (
                            <TemplateItem
                                key={template.id}
                                template={template}
                                onLoad={onLoadTemplate}
                                onDelete={handleDeleteTemplate}
                                isDeleting={isDeletingId === template.id}
                            />
                        ))}
                    </div>
                )}
            </div>

            {/* Save Modal */}
            <SaveTemplateModal
                isOpen={isSaveModalOpen}
                onClose={() => setIsSaveModalOpen(false)}
                onSave={handleSaveTemplate}
                isLoading={isSaving}
            />
        </div>
    );
};
