/**
 * Standalone drawer components for AdminInfrastructureProviderDetailPage.
 * These drawers have a well-scoped prop surface and no cross-dependencies with
 * the rest of the page, making them safe to extract cleanly.
 */

import {
    CopyableCodeBlock,
    TooltipDrawer,
} from "@/components/admin/providers/TooltipDrawer";
import Drawer from "@/components/ui/Drawer.tsx";
import { InfrastructureProvider } from "@/types";
import { Check, RefreshCw, X } from "lucide-react";
import React from "react";
import {
    bootstrapClusterRoleTemplate,
    bootstrapNamespaceTemplate,
    kubernetesProviderTypes,
    runtimeNamespaceTemplate,
    runtimeRoleTemplate,
    runtimeServiceAccountTokenTemplate,
    serviceAccountTokenTemplate,
    statusColors,
} from "./adminInfraProviderDetailShared";

// ---------------------------------------------------------------------------
// TestConnectionDrawer
// ---------------------------------------------------------------------------

export const TestConnectionDrawer: React.FC<{
    isOpen: boolean;
    onClose: () => void;
    testProgress:
    | "initializing"
    | "connecting"
    | "receiving"
    | "completed"
    | "failed"
    | null;
    testError: string | null;
    provider: InfrastructureProvider;
    visibleAuthConfig?: Record<string, unknown>;
}> = ({ isOpen, onClose, testProgress, testError, provider, visibleAuthConfig }) => (
    <Drawer
        isOpen={isOpen}
        onClose={onClose}
        title="Testing Connection"
        description="Testing connectivity to the infrastructure provider"
    >
        <div className="space-y-6">
            {/* Progress Steps */}
            <div className="space-y-4">
                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
                    Test Progress
                </h3>

                <div className="space-y-3">
                    <div
                        className={`flex items-center space-x-3 ${testProgress === "initializing" ? "text-blue-600 dark:text-blue-400" : "text-gray-400 dark:text-gray-500"}`}
                    >
                        <div
                            className={`w-6 h-6 rounded-full flex items-center justify-center ${testProgress === "initializing" ? "bg-blue-100 dark:bg-blue-900" : "bg-gray-100 dark:bg-gray-800"}`}
                        >
                            {(testProgress === "initializing" ||
                                testProgress === "connecting" ||
                                testProgress === "receiving") && (
                                    <RefreshCw className="h-4 w-4 animate-spin" />
                                )}
                            {testProgress === "completed" && (
                                <Check className="h-4 w-4 text-green-600 dark:text-green-400" />
                            )}
                            {testProgress === "failed" && (
                                <X className="h-4 w-4 text-red-600 dark:text-red-400" />
                            )}
                        </div>
                        <span className="text-sm">
                            {testProgress === "initializing" && "Initializing test..."}
                            {testProgress === "connecting" && "Connecting to provider..."}
                            {testProgress === "receiving" && "Receiving response..."}
                            {testProgress === "completed" && "Connection test completed"}
                            {testProgress === "failed" && "Connection test failed"}
                        </span>
                    </div>
                </div>

                {/* Provider Info */}
                <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                        Provider Information
                    </h4>
                    <dl className="space-y-2 text-sm">
                        <div className="flex justify-between">
                            <dt className="text-gray-500 dark:text-gray-400">
                                Provider Type:
                            </dt>
                            <dd className="text-gray-900 dark:text-white font-medium">
                                {provider.provider_type}
                            </dd>
                        </div>
                        <div className="flex justify-between">
                            <dt className="text-gray-500 dark:text-gray-400">
                                Provider Name:
                            </dt>
                            <dd className="text-gray-900 dark:text-white font-medium">
                                {provider.display_name}
                            </dd>
                        </div>
                        <div className="flex justify-between">
                            <dt className="text-gray-500 dark:text-gray-400">Status:</dt>
                            <dd className="text-gray-900 dark:text-white font-medium">
                                <span
                                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColors[provider.status || "pending"]}`}
                                >
                                    {provider.status}
                                </span>
                            </dd>
                        </div>
                        {provider.health_status && (
                            <div className="flex justify-between">
                                <dt className="text-gray-500 dark:text-gray-400">
                                    Health Status:
                                </dt>
                                <dd className="text-gray-900 dark:text-white font-medium">
                                    <span
                                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${provider.health_status === "healthy"
                                            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                                            : provider.health_status === "warning"
                                                ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                                                : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                                            }`}
                                    >
                                        {provider.health_status}
                                    </span>
                                </dd>
                            </div>
                        )}
                        {provider.last_health_check && (
                            <div className="flex justify-between">
                                <dt className="text-gray-500 dark:text-gray-400">
                                    Last Health Check:
                                </dt>
                                <dd className="text-gray-900 dark:text-white font-medium">
                                    {new Date(provider.last_health_check).toLocaleString()}
                                </dd>
                            </div>
                        )}
                    </dl>
                </div>

                {/* Configuration Details */}
                <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                        Configuration
                    </h4>
                    <dl className="space-y-2 text-sm">
                        {provider.provider_type === "kubernetes" &&
                            (visibleAuthConfig?.auth_method ||
                                provider.config?.auth_method) && (
                                <div className="flex justify-between">
                                    <dt className="text-gray-500 dark:text-gray-400">
                                        Authentication Method:
                                    </dt>
                                    <dd className="text-gray-900 dark:text-white font-medium capitalize">
                                        {(
                                            (visibleAuthConfig?.auth_method as string | undefined) ||
                                            provider.config?.auth_method ||
                                            "unknown"
                                        ).replace("_", " ")}
                                    </dd>
                                </div>
                            )}
                        {provider.config?.apiServer && (
                            <div className="flex justify-between">
                                <dt className="text-gray-500 dark:text-gray-400">
                                    API Server Endpoint:
                                </dt>
                                <dd className="text-gray-900 dark:text-white font-medium font-mono text-xs">
                                    {provider.config.apiServer}
                                </dd>
                            </div>
                        )}
                        {provider.config?.endpoint &&
                            provider.provider_type !== "kubernetes" && (
                                <div className="flex justify-between">
                                    <dt className="text-gray-500 dark:text-gray-400">
                                        API Endpoint:
                                    </dt>
                                    <dd className="text-gray-900 dark:text-white font-medium font-mono text-xs">
                                        {provider.config.endpoint}
                                    </dd>
                                </div>
                            )}
                        {provider.config?.cluster_endpoint && (
                            <div className="flex justify-between">
                                <dt className="text-gray-500 dark:text-gray-400">
                                    Cluster Endpoint:
                                </dt>
                                <dd className="text-gray-900 dark:text-white font-medium font-mono text-xs">
                                    {provider.config.cluster_endpoint}
                                </dd>
                            </div>
                        )}
                        {provider.config?.namespace &&
                            !kubernetesProviderTypes.has(provider.provider_type) && (
                                <div className="flex justify-between">
                                    <dt className="text-gray-500 dark:text-gray-400">
                                        Namespace:
                                    </dt>
                                    <dd className="text-gray-900 dark:text-white font-medium">
                                        {provider.config.namespace}
                                    </dd>
                                </div>
                            )}
                        {provider.config?.region && (
                            <div className="flex justify-between">
                                <dt className="text-gray-500 dark:text-gray-400">Region:</dt>
                                <dd className="text-gray-900 dark:text-white font-medium">
                                    {provider.config.region}
                                </dd>
                            </div>
                        )}
                        {provider.config?.kubeconfig_path && (
                            <div className="flex justify-between">
                                <dt className="text-gray-500 dark:text-gray-400">
                                    Kubeconfig Path:
                                </dt>
                                <dd className="text-gray-900 dark:text-white font-medium font-mono text-xs">
                                    {provider.config.kubeconfig_path}
                                </dd>
                            </div>
                        )}
                        {provider.capabilities && provider.capabilities.length > 0 && (
                            <div className="flex justify-between">
                                <dt className="text-gray-500 dark:text-gray-400">
                                    Capabilities:
                                </dt>
                                <dd className="text-gray-900 dark:text-white font-medium">
                                    <div className="flex flex-wrap gap-1">
                                        {provider.capabilities.map((capability, index) => (
                                            <span
                                                key={index}
                                                className="inline-flex items-center px-2 py-1 rounded-md text-xs bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                                            >
                                                {capability}
                                            </span>
                                        ))}
                                    </div>
                                </dd>
                            </div>
                        )}
                    </dl>
                </div>

                {/* Test Details */}
                {testProgress === "completed" && (
                    <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                        <h4 className="text-sm font-medium text-green-900 dark:text-green-200 mb-2">
                            Test Results
                        </h4>
                        <div className="flex items-center space-x-2">
                            <Check className="h-5 w-5 text-green-600" />
                            <span className="text-sm text-green-800 dark:text-green-200">
                                Successfully connected to provider
                            </span>
                        </div>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                            The provider is now online and ready to use.
                        </p>
                    </div>
                )}

                {testProgress === "failed" && (
                    <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                        <h4 className="text-sm font-medium text-red-900 dark:text-red-200 mb-2">
                            Test Failed
                        </h4>
                        <div className="flex items-center space-x-2">
                            <X className="h-5 w-5 text-red-600" />
                            <span className="text-sm text-red-800 dark:text-red-200">
                                Connection test failed
                            </span>
                        </div>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                            {testError ||
                                "Please check your provider configuration and try again."}
                        </p>
                    </div>
                )}
            </div>

            {/* Close Button */}
            <div className="flex justify-end pt-4 border-t border-gray-200 dark:border-gray-700">
                <button
                    onClick={onClose}
                    className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 dark:bg-gray-700 dark:hover:bg-gray-600 font-medium"
                >
                    Close
                </button>
            </div>
        </div>
    </Drawer>
);

// ---------------------------------------------------------------------------
// BootstrapRBACDrawer
// ---------------------------------------------------------------------------

export const BootstrapRBACDrawer: React.FC<{
    isOpen: boolean;
    onClose: () => void;
    isManagedBootstrapProvider: boolean;
}> = ({ isOpen, onClose, isManagedBootstrapProvider }) => (
    <TooltipDrawer
        isOpen={isOpen}
        onClose={onClose}
        title={
            isManagedBootstrapProvider
                ? "Image Factory Managed Bootstrap RBAC"
                : "Self-Managed Runtime RBAC"
        }
    >
        <div className="space-y-4 text-sm text-gray-700 dark:text-gray-300">
            {isManagedBootstrapProvider ? (
                <p>
                    Image Factory managed mode only needs bootstrap credentials here.
                    During provider/tenant prepare, Image Factory creates and manages
                    runtime service accounts and RBAC automatically.
                </p>
            ) : (
                <p>
                    Self-managed mode assumes cluster bootstrap is handled outside of
                    Image Factory. You still need a runtime service account and
                    namespace-scoped RBAC so builds can run.
                </p>
            )}
            <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                Update namespace and object names before applying. These are secure
                defaults to get started, not mandatory final production policy.
            </div>

            {isManagedBootstrapProvider ? (
                <>
                    <CopyableCodeBlock
                        title="1) System Namespace + Service Accounts"
                        code={bootstrapNamespaceTemplate}
                        language="yaml"
                    />
                    <CopyableCodeBlock
                        title="2) Bootstrap ClusterRole + Binding (Managed Prepare)"
                        code={bootstrapClusterRoleTemplate}
                        language="yaml"
                    />
                    <CopyableCodeBlock
                        title="3) Bootstrap ServiceAccount Token (Recommended)"
                        code={serviceAccountTokenTemplate}
                        language="bash"
                    />
                </>
            ) : (
                <>
                    <CopyableCodeBlock
                        title="1) System Namespace + Runtime Service Account"
                        code={runtimeNamespaceTemplate}
                        language="yaml"
                    />
                    <CopyableCodeBlock
                        title="2) Tenant Namespace Runtime Role + Binding (Least Privilege)"
                        code={runtimeRoleTemplate}
                        language="yaml"
                    />
                    <CopyableCodeBlock
                        title="3) Runtime ServiceAccount Token Setup"
                        code={runtimeServiceAccountTokenTemplate}
                        language="bash"
                    />
                </>
            )}
        </div>
    </TooltipDrawer>
);
