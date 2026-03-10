import { HelpCircle, X } from "lucide-react";
import React, { useState } from "react";
import { ProviderFormData } from "./types";

interface CommonKubernetesFieldsProps {
  formData: ProviderFormData;
  setFormData: (data: ProviderFormData) => void;
}

export const CommonKubernetesFields: React.FC<CommonKubernetesFieldsProps> = ({
  formData,
  setFormData,
}) => {
  const [showApiServerTooltip, setShowApiServerTooltip] = useState(false);

  return (
    <>
      <div>
        <div className="flex items-center gap-2 mb-1">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            API Server Endpoint
          </label>
          <div className="group relative">
            <HelpCircle
              className="h-4 w-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer"
              onClick={(e) => {
                e.stopPropagation();
                setShowApiServerTooltip(!showApiServerTooltip);
              }}
            />
            {showApiServerTooltip && (
              <div className="absolute left-0 top-6 w-80 p-3 bg-gray-900 dark:bg-gray-700 text-white text-xs rounded-md shadow-lg border border-gray-600 z-10">
                <div className="flex justify-between items-center mb-2">
                  <div className="font-semibold">API Server URL</div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setShowApiServerTooltip(false);
                    }}
                    className="text-gray-400 hover:text-white"
                    type="button"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
                <div className="mb-2">
                  The Kubernetes API server endpoint used for cluster management
                  operations:
                </div>
                <ul className="list-disc list-inside space-y-1 mb-2">
                  <li>Creating and managing pods, services, deployments</li>
                  <li>Reading cluster status and logs</li>
                  <li>Executing build jobs and workloads</li>
                  <li>Cluster administration and monitoring</li>
                </ul>
                <div className="text-xs text-gray-300">
                  <strong>Example:</strong> https://k8s-cluster.example.com:6443
                </div>
              </div>
            )}
          </div>
        </div>
        <input
          type="url"
          value={formData.config.endpoint || ""}
          onChange={(e) =>
            setFormData({
              ...formData,
              config: { ...formData.config, endpoint: e.target.value },
            })
          }
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
          placeholder="https://k8s-cluster.example.com:6443"
          required
        />
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
          The Kubernetes API server endpoint for cluster management
        </p>
      </div>
      <div>
        <div className="flex items-center gap-2 mb-1">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Cluster Endpoint (optional)
          </label>
        </div>
        <input
          type="url"
          value={formData.config.cluster_endpoint || ""}
          onChange={(e) =>
            setFormData({
              ...formData,
              config: { ...formData.config, cluster_endpoint: e.target.value },
            })
          }
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
          placeholder="https://my-cluster.example.com"
        />
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
          Load balancer or ingress endpoint for accessing applications deployed
          to the cluster
        </p>
      </div>
    </>
  );
};
