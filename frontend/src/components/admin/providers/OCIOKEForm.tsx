import { HelpCircle } from 'lucide-react'
import React, { useState } from 'react'
import { CommonKubernetesFields } from './CommonKubernetesFields'
import { TabbedForm } from './TabbedForm'
import { CopyableCodeBlock, TooltipDrawer } from './TooltipDrawer'
import { ProviderConfigFormProps, ProviderFormComponent } from './types'

const OCIOKEFormComponent: React.FC<ProviderConfigFormProps> = ({
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
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Authentication Method
                            <button
                                type="button"
                                onClick={() => setShowAuthMethodDrawer(true)}
                                className="ml-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                                title="Help with authentication methods"
                            >
                                <HelpCircle size={16} />
                            </button>
                        </label>
                        <select
                            value={formData.config.auth_method || 'instance-principal'}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, auth_method: e.target.value }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                            required
                        >
                            <option value="instance-principal">Instance Principal</option>
                            <option value="api-key">API Key</option>
                            <option value="kubeconfig">Kubeconfig File</option>
                        </select>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Oracle OKE supports Instance Principal for authentication (recommended)
                        </p>
                    </div>
                    {(formData.config.auth_method === 'instance-principal' || (!formData.config.auth_method)) && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Tenancy OCID
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.tenancy_ocid || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, tenancy_ocid: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="ocid1.tenancy.oc1.."
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    OCI tenancy OCID where the instance principal is configured
                                </p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    OCI Region
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.oci_region || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, oci_region: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="us-ashburn-1"
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    OCI region where the OKE cluster is located
                                </p>
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'api-key' && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    User OCID
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.user_ocid || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, user_ocid: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="ocid1.user.oc1.."
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Tenancy OCID
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.tenancy_ocid || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, tenancy_ocid: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="ocid1.tenancy.oc1.."
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Private Key
                                </label>
                                <textarea
                                    value={formData.config.private_key || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, private_key: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    rows={6}
                                    placeholder="-----BEGIN PRIVATE KEY-----..."
                                />
                            </div>
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
            {showAuthMethodDrawer && (
                <TooltipDrawer
                    isOpen={showAuthMethodDrawer}
                    title={`Setting up ${formData.config.auth_method || 'instance-principal'} authentication for Oracle OKE`}
                    onClose={() => setShowAuthMethodDrawer(false)}
                >
                    {formData.config.auth_method === 'instance-principal' && (
                        <div className="space-y-6">
                            <div>
                                <h3 className="text-lg font-semibold mb-2">Instance Principal Setup</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                    Use Oracle Cloud Infrastructure's instance principal for secure, automatic authentication. This is the recommended method for applications running on OCI compute instances.
                                </p>
                                <div className="space-y-4">
                                    <div>
                                        <h4 className="font-medium mb-2">1. Enable Instance Principal</h4>
                                        <CopyableCodeBlock
                                            title="Enable Instance Principal"
                                            code={`# For compute instances, instance principal is enabled by default
# For dynamic groups, create a dynamic group rule:
# ALL {instance.compartment.id = 'ocid1.compartment.oc1..example'}

# Create a policy for the dynamic group
oci iam policy create \\
  --compartment-id <tenancy-ocid> \\
  --name "image-factory-policy" \\
  --statements '[
    "Allow dynamic-group image-factory-dg to read clusters in compartment id <compartment-ocid>",
    "Allow dynamic-group image-factory-dg to use cluster-node-pools in compartment id <compartment-ocid>"
  ]'`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">2. Configure Provider</h4>
                                        <p className="text-sm mb-2">Set the following values in the provider configuration:</p>
                                        <ul className="list-disc list-inside text-sm space-y-1">
                                            <li><strong>Tenancy OCID:</strong> Your Oracle Cloud tenancy OCID</li>
                                            <li><strong>Compartment OCID:</strong> The compartment containing your OKE cluster</li>
                                            <li><strong>Cluster OCID:</strong> Your OKE cluster OCID</li>
                                            <li><strong>Auth Method:</strong> instance-principal</li>
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'api-key' && (
                        <div className="space-y-6">
                            <div>
                                <h3 className="text-lg font-semibold mb-2">API Key Setup</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                    Use OCI API keys for authentication. This method requires generating and securely storing API key files.
                                </p>
                                <div className="space-y-4">
                                    <div>
                                        <h4 className="font-medium mb-2">1. Generate API Key Pair</h4>
                                        <CopyableCodeBlock
                                            title="Generate API Key Pair"
                                            code={`# Generate private key
openssl genrsa -out ~/.oci/oci_api_key.pem 2048

# Generate public key
openssl rsa -pubout -in ~/.oci/oci_api_key.pem -out ~/.oci/oci_api_key_public.pem

# Get the public key content
cat ~/.oci/oci_api_key_public.pem`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">2. Add API Key to OCI Console</h4>
                                        <p className="text-sm mb-2">In the OCI Console:</p>
                                        <ol className="list-decimal list-inside text-sm space-y-1 mb-4">
                                            <li>Go to Identity → Users → Your User</li>
                                            <li>Click "API Keys" in the left sidebar</li>
                                            <li>Click "Add API Keys"</li>
                                            <li>Paste the public key content</li>
                                            <li>Click "Add"</li>
                                        </ol>
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">3. Configure Provider</h4>
                                        <p className="text-sm mb-2">Use the API key credentials in the provider configuration:</p>
                                        <ul className="list-disc list-inside text-sm space-y-1">
                                            <li><strong>Tenancy OCID:</strong> Your Oracle Cloud tenancy OCID</li>
                                            <li><strong>User OCID:</strong> Your OCI user OCID</li>
                                            <li><strong>Private Key:</strong> Content of the private key file</li>
                                            <li><strong>Fingerprint:</strong> The fingerprint shown in OCI console after adding the key</li>
                                            <li><strong>Auth Method:</strong> api-key</li>
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'kubeconfig' && (
                        <div className="space-y-6">
                            <div>
                                <h3 className="text-lg font-semibold mb-2">Kubeconfig Setup</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                    Use a kubeconfig file for direct cluster access. This method works with OKE clusters that have been configured in your kubeconfig.
                                </p>
                                <div className="space-y-4">
                                    <div>
                                        <h4 className="font-medium mb-2">1. Get OKE Cluster Access</h4>
                                        <CopyableCodeBlock
                                            title="Get OKE Cluster Access"
                                            code={`# Set up OCI CLI authentication first
oci setup config

# Get cluster kubeconfig
oci ce cluster create-kubeconfig \\
  --cluster-id <cluster-ocid> \\
  --file ~/.kube/config \\
  --region <region> \\
  --token-version 2.0.0

# Verify access
kubectl cluster-info`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">2. Configure Provider</h4>
                                        <p className="text-sm mb-2">Upload the kubeconfig file content in the provider configuration:</p>
                                        <ul className="list-disc list-inside text-sm space-y-1">
                                            <li><strong>Kubeconfig:</strong> Paste the entire content of your kubeconfig file</li>
                                            <li><strong>Auth Method:</strong> kubeconfig</li>
                                        </ul>
                                        <p className="text-sm text-amber-600 dark:text-amber-400 mt-2">
                                            ⚠️ Store the kubeconfig securely and rotate it regularly for security.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                    {(!formData.config.auth_method || formData.config.auth_method === 'instance-principal') && (
                        <div className="space-y-6">
                            <div>
                                <h3 className="text-lg font-semibold mb-2">Default: Instance Principal Setup</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                    Use Oracle Cloud Infrastructure's instance principal for secure, automatic authentication. This is the recommended method for applications running on OCI compute instances.
                                </p>
                                <div className="space-y-4">
                                    <div>
                                        <h4 className="font-medium mb-2">1. Enable Instance Principal</h4>
                                        <CopyableCodeBlock
                                            title="Enable Instance Principal"
                                            code={`# For compute instances, instance principal is enabled by default
# For dynamic groups, create a dynamic group rule:
# ALL {instance.compartment.id = 'ocid1.compartment.oc1..example'}

# Create a policy for the dynamic group
oci iam policy create \\
  --compartment-id <tenancy-ocid> \\
  --name "image-factory-policy" \\
  --statements '[
    "Allow dynamic-group image-factory-dg to read clusters in compartment id <compartment-ocid>",
    "Allow dynamic-group image-factory-dg to use cluster-node-pools in compartment id <compartment-ocid>"
  ]'`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">2. Configure Provider</h4>
                                        <p className="text-sm mb-2">Set the following values in the provider configuration:</p>
                                        <ul className="list-disc list-inside text-sm space-y-1">
                                            <li><strong>Tenancy OCID:</strong> Your Oracle Cloud tenancy OCID</li>
                                            <li><strong>Compartment OCID:</strong> The compartment containing your OKE cluster</li>
                                            <li><strong>Cluster OCID:</strong> Your OKE cluster OCID</li>
                                            <li><strong>Auth Method:</strong> instance-principal</li>
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                </TooltipDrawer>
            )}
        </>
    )
}

export const OCIOKEForm: ProviderFormComponent = {
    component: OCIOKEFormComponent,
    displayName: 'Oracle OKE',
    description: 'Oracle Container Engine for Kubernetes'
}