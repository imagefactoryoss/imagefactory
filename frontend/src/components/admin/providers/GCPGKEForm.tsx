import { HelpCircle } from 'lucide-react'
import React, { useState } from 'react'
import { CommonKubernetesFields } from './CommonKubernetesFields'
import { TabbedForm } from './TabbedForm'
import { CopyableCodeBlock, TooltipDrawer } from './TooltipDrawer'
import { ProviderConfigFormProps, ProviderFormComponent } from './types'

const GCPGKEFormComponent: React.FC<ProviderConfigFormProps> = ({
    formData,
    setFormData
}) => {
    const [showAuthMethodDrawer, setShowAuthMethodDrawer] = useState(false)
    const tabs = [
        {
            id: 'auth',
            label: 'Authentication',
            content: (
                <div className="space-y-4">
                    <div>
                        <div className="flex items-center gap-2 mb-1">
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Authentication Method
                            </label>
                            <HelpCircle
                                className="h-4 w-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer"
                                onClick={(e) => { e.stopPropagation(); setShowAuthMethodDrawer(true); }}
                            />
                        </div>
                        <select
                            value={formData.config.auth_method || 'workload-identity'}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, auth_method: e.target.value }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                            required
                        >
                            <option value="workload-identity">Workload Identity</option>
                            <option value="service-account">Service Account Key</option>
                            <option value="kubeconfig">Kubeconfig File</option>
                        </select>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            GCP GKE supports Workload Identity for authentication (recommended)
                        </p>
                    </div>
                    {(formData.config.auth_method === 'workload-identity' || (!formData.config.auth_method)) && (
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                GCP Project ID
                            </label>
                            <input
                                type="text"
                                value={formData.config.gcp_project_id || ''}
                                onChange={(e) => setFormData({
                                    ...formData,
                                    config: { ...formData.config, gcp_project_id: e.target.value }
                                })}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                placeholder="my-gcp-project"
                            />
                            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                GCP project containing the GKE cluster
                            </p>
                        </div>
                    )}
                    {formData.config.auth_method === 'service-account' && (
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Service Account Key (JSON)
                            </label>
                            <textarea
                                value={formData.config.service_account_key || ''}
                                onChange={(e) => setFormData({
                                    ...formData,
                                    config: { ...formData.config, service_account_key: e.target.value }
                                })}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                rows={6}
                                placeholder='{"type": "service_account", "project_id": "..."}'
                            />
                            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                GCP service account key with GKE cluster access
                            </p>
                        </div>
                    )}
                </div>
            )
        },
        {
            id: 'config',
            label: 'Configuration',
            content: (
                <CommonKubernetesFields formData={formData} setFormData={setFormData} />
            )
        }
    ]

    return (
        <>
            <TabbedForm tabs={tabs} defaultActiveTab="auth" />

            {/* Authentication Method Setup Drawer */}
            <TooltipDrawer
                isOpen={showAuthMethodDrawer}
                onClose={() => setShowAuthMethodDrawer(false)}
                title={`${formData.config.auth_method || 'workload-identity'} Authentication Setup`}
            >
                <div className="space-y-4">
                    {(formData.config.auth_method || 'workload-identity') === 'workload-identity' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Workload Identity provides secure, GCP-managed authentication for GKE clusters. This is the recommended method for production environments.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses GCP Workload Identity for authentication</li>
                                    <li>No service account keys stored in the application</li>
                                    <li>Automatic credential rotation</li>
                                    <li>Integrates with GCP IAM and service accounts</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Requirements:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Application must run on GCP infrastructure (GKE, GCE, Cloud Run)</li>
                                    <li>GCP project ID must be specified</li>
                                    <li>Workload Identity must be enabled on the cluster</li>
                                    <li>Service account must have appropriate GKE permissions</li>
                                </ul>
                            </div>
                        </>
                    )}

                    {(formData.config.auth_method || 'workload-identity') === 'service-account' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use a GCP service account key for authentication. This method uses traditional service account keys stored in the application.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses GCP service account JSON key files</li>
                                    <li>Key content is stored in the application configuration</li>
                                    <li>Requires secure key management</li>
                                    <li>Works from any environment</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    Create a service account in GCP IAM, download the JSON key, and provide the key content below.
                                </p>
                                <CopyableCodeBlock
                                    title="Create Service Account Key"
                                    code={`# Create service account
gcloud iam service-accounts create image-factory-sa

# Grant necessary permissions
gcloud projects add-iam-policy-binding PROJECT_ID \\
  --member="serviceAccount:image-factory-sa@PROJECT_ID.iam.gserviceaccount.com" \\
  --role="roles/container.clusterViewer"

# Create and download key
gcloud iam service-accounts keys create key.json \\
  --iam-account=image-factory-sa@PROJECT_ID.iam.gserviceaccount.com`}
                                    language="bash"
                                />
                            </div>
                        </>
                    )}

                    {(formData.config.auth_method || 'workload-identity') === 'kubeconfig' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use a kubeconfig file for authentication. This method works with GKE clusters that have been added to your kubeconfig.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Reads authentication credentials from kubeconfig file</li>
                                    <li>Supports GKE-generated kubeconfig files</li>
                                    <li>Can reference multiple GKE clusters</li>
                                    <li>Standard Kubernetes authentication method</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    Use <code>gcloud</code> to get cluster credentials and update your kubeconfig.
                                </p>
                                <CopyableCodeBlock
                                    title="Get GKE cluster credentials"
                                    code={`# Authenticate with GCP
gcloud auth login

# Get cluster credentials
gcloud container clusters get-credentials CLUSTER_NAME \\
  --region REGION \\
  --project PROJECT_ID`}
                                    language="bash"
                                />
                            </div>
                        </>
                    )}

                    {/* Default case - show workload-identity */}
                    {(!formData.config.auth_method || (formData.config.auth_method !== 'service-account' && formData.config.auth_method !== 'kubeconfig')) && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Workload Identity provides secure, GCP-managed authentication for GKE clusters. This is the recommended method for production environments.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses GCP Workload Identity for authentication</li>
                                    <li>No service account keys stored in the application</li>
                                    <li>Automatic credential rotation</li>
                                    <li>Integrates with GCP IAM and service accounts</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Requirements:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Application must run on GCP infrastructure (GKE, GCE, Cloud Run)</li>
                                    <li>GCP project ID must be specified</li>
                                    <li>Workload Identity must be enabled on the cluster</li>
                                    <li>Service account must have appropriate GKE permissions</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    Enable Workload Identity on your GKE cluster and configure the Kubernetes service account.
                                </p>
                                <CopyableCodeBlock
                                    title="Enable Workload Identity on GKE cluster"
                                    code={`# Enable Workload Identity on the cluster
gcloud container clusters update CLUSTER_NAME \\
  --region REGION \\
  --workload-pool=PROJECT_ID.svc.id.goog

# Create Kubernetes service account
kubectl create serviceaccount image-factory-sa \\
  --namespace NAMESPACE

# Create IAM service account
gcloud iam service-accounts create image-factory-gsa \\
  --project PROJECT_ID

# Grant necessary permissions to the IAM service account
gcloud projects add-iam-policy-binding PROJECT_ID \\
  --member="serviceAccount:image-factory-gsa@PROJECT_ID.iam.gserviceaccount.com" \\
  --role="roles/container.clusterViewer"

# Annotate the Kubernetes service account
kubectl annotate serviceaccount image-factory-sa \\
  --namespace NAMESPACE \\
  iam.gke.io/gcp-service-account=image-factory-gsa@PROJECT_ID.iam.gserviceaccount.com`}
                                    language="bash"
                                />
                            </div>
                        </>
                    )}
                </div>
            </TooltipDrawer>
        </>
    )
}

export const GCPGKEForm: ProviderFormComponent = {
    component: GCPGKEFormComponent,
    displayName: 'Google GKE',
    description: 'Google Kubernetes Engine with Workload Identity'
}