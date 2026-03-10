import { HelpCircle } from 'lucide-react'
import React, { useState } from 'react'
import { CommonKubernetesFields } from './CommonKubernetesFields'
import { TabbedForm } from './TabbedForm'
import { CopyableCodeBlock, TooltipDrawer } from './TooltipDrawer'
import { ProviderConfigFormProps, ProviderFormComponent } from './types'

const AWSEKSFormComponent: React.FC<ProviderConfigFormProps> = ({
    formData,
    setFormData
}) => {
    const [showIAMDrawer, setShowIAMDrawer] = useState(false)
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
                            value={formData.config.auth_method || 'iam'}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, auth_method: e.target.value }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                            required
                        >
                            <option value="iam">IAM Roles</option>
                            <option value="kubeconfig">Kubeconfig File</option>
                            <option value="token">Service Account Token</option>
                        </select>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            AWS EKS supports IAM roles for authentication (recommended)
                        </p>
                    </div>
                    {(formData.config.auth_method === 'iam' || (!formData.config.auth_method)) && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    AWS Region
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.region || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, region: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="us-east-1"
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    AWS region where the EKS cluster is located
                                </p>
                            </div>
                            <div>
                                <div className="flex items-center gap-2 mb-1">
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                        IAM Role ARN (optional)
                                    </label>
                                    <div className="group relative">
                                        <HelpCircle
                                            className="h-4 w-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer"
                                            onClick={(e) => { e.stopPropagation(); setShowIAMDrawer(true); }}
                                        />
                                    </div>
                                </div>
                                <input
                                    type="text"
                                    value={formData.config.iam_role_arn || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, iam_role_arn: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="arn:aws:iam::123456789012:role/eks-service-role"
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    ARN of the IAM role with EKS cluster access (leave empty to use instance role)
                                </p>
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'kubeconfig' && (
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Kubeconfig Path
                            </label>
                            <input
                                type="text"
                                value={formData.config.kubeconfig_path || ''}
                                onChange={(e) => setFormData({
                                    ...formData,
                                    config: { ...formData.config, kubeconfig_path: e.target.value }
                                })}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                placeholder="/path/to/kubeconfig"
                                required
                            />
                            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                Path to the kubeconfig file for EKS cluster access
                            </p>
                        </div>
                    )}
                    {formData.config.auth_method === 'token' && (
                        <div className="space-y-4">
                            <div>
                                <div className="flex items-center gap-2 mb-1">
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                        Service Account Token
                                    </label>
                                    <div className="group relative">
                                        <HelpCircle
                                            className="h-4 w-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer"
                                            onClick={(e) => { e.stopPropagation(); setShowIAMDrawer(true); }}
                                        />
                                    </div>
                                </div>
                                <textarea
                                    value={formData.config.token || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, token: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    rows={4}
                                    placeholder="eyJhbGciOiJSUzI1NiIsImtpZCI6..."
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    CA Certificate (optional)
                                </label>
                                <textarea
                                    value={formData.config.ca_cert || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, ca_cert: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    rows={3}
                                    placeholder="-----BEGIN CERTIFICATE-----..."
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

            {/* IAM Role Setup Drawer */}
            <TooltipDrawer
                isOpen={showIAMDrawer}
                onClose={() => setShowIAMDrawer(false)}
                title="IAM Role Setup for EKS Access"
            >
                <div className="space-y-4">
                    <p className="text-gray-600 dark:text-gray-400">
                        To create an IAM role with the required permissions for EKS cluster access, set up the following:
                    </p>

                    <CopyableCodeBlock
                        title="1. Create the IAM Policy"
                        code={`{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "eks:DescribeCluster",
                "eks:ListClusters",
                "eks:AccessKubernetesApi"
            ],
            "Resource": "*"
        }
    ]
}`}
                        language="json"
                    />

                    <div>
                        <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                            2. Create the IAM Role
                        </h3>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
                            Create an IAM role and attach the above policy. Use the following trust relationship:
                        </p>

                        <CopyableCodeBlock
                            title="Trust Relationship Policy"
                            code={`{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::ACCOUNT-ID:root"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}`}
                            language="json"
                        />
                    </div>

                    <div>
                        <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                            3. Additional Permissions (if needed)
                        </h3>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                            For full cluster management, you may need additional permissions like EC2, VPC, and other AWS service access depending on your use case.
                        </p>
                    </div>
                </div>
            </TooltipDrawer>

            {/* Authentication Method Setup Drawer */}
            <TooltipDrawer
                isOpen={showAuthMethodDrawer}
                onClose={() => setShowAuthMethodDrawer(false)}
                title={`${formData.config.auth_method || 'iam'} Authentication Setup`}
            >
                <div className="space-y-4">
                    {(formData.config.auth_method || 'iam') === 'iam' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                IAM Roles provide secure, AWS-managed authentication for EKS clusters. This is the recommended method for production environments.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses AWS IAM roles for authentication</li>
                                    <li>No credentials stored in the application</li>
                                    <li>Automatic credential rotation</li>
                                    <li>Works with AWS STS and IAM policies</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Requirements:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Application must run on AWS infrastructure (EC2, ECS, EKS, Lambda)</li>
                                    <li>IAM role must have appropriate EKS permissions</li>
                                    <li>AWS region must be specified</li>
                                </ul>
                            </div>
                        </>
                    )}

                    {(formData.config.auth_method || 'iam') === 'kubeconfig' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use a kubeconfig file for authentication. This method is useful for development or when you have existing kubeconfig files.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Reads authentication credentials from kubeconfig file</li>
                                    <li>Supports various auth methods (certificates, tokens, etc.)</li>
                                    <li>Can reference external files or embedded credentials</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    Ensure your kubeconfig file is accessible at the specified path and contains valid EKS cluster credentials.
                                </p>
                                <CopyableCodeBlock
                                    title="Example kubeconfig path"
                                    code="/home/user/.kube/config"
                                    language="bash"
                                />
                            </div>
                        </>
                    )}

                    {(formData.config.auth_method || 'iam') === 'token' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use a Kubernetes service account token for authentication. This method provides direct API access using bearer tokens.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses Kubernetes service account tokens</li>
                                    <li>Requires API server endpoint and CA certificate</li>
                                    <li>Token must have appropriate RBAC permissions</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    You'll need to provide the cluster endpoint, service account token, and CA certificate. See the service account token field help for detailed setup instructions.
                                </p>
                            </div>
                        </>
                    )}
                </div>
            </TooltipDrawer>
        </>
    )
}

export const AWSEKSForm: ProviderFormComponent = {
    component: AWSEKSFormComponent,
    displayName: 'Amazon EKS',
    description: 'Amazon Elastic Kubernetes Service with IAM role authentication'
}