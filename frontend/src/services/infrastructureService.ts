import {
  CreateInfrastructureProviderRequest,
  InfrastructureProvider,
  InfrastructureProviderHealth,
  PaginatedResponse,
  TektonProviderStatus,
  ProviderPrepareRun,
  ProviderPrepareSummary,
  ProviderPrepareSummaryBatchMetrics,
  ProviderPrepareStatus,
  ProviderTenantNamespacePrepare,
  TenantNamespaceReconcileSummary,
  TestProviderConnectionResponse,
  TektonInstallMode,
  TektonInstallerJob,
  UpdateInfrastructureProviderRequest,
} from "@/types";
import api from "./api";

// Infrastructure service for managing infrastructure providers
export const infrastructureService = {
  // Get all infrastructure providers (admin only)
  async getProviders(filters?: {
    provider_type?: string;
    status?: string;
    page?: number;
    limit?: number;
  }): Promise<PaginatedResponse<InfrastructureProvider>> {
    try {
      const params = new URLSearchParams();
      if (filters?.provider_type)
        params.append("provider_type", filters.provider_type);
      if (filters?.status) params.append("status", filters.status);
      if (filters?.page) params.append("page", filters.page.toString());
      if (filters?.limit) params.append("limit", filters.limit.toString());

      const queryString = params.toString();
      const url = queryString
        ? `/admin/infrastructure/providers?${queryString}`
        : "/admin/infrastructure/providers";
      const response = await api.get(url);

      return {
        data: response.data.providers || response.data.data || [],
        pagination: response.data.pagination || {
          page: filters?.page || 1,
          limit: filters?.limit || 20,
          total: response.data.total || 0,
          totalPages: Math.ceil(
            (response.data.total || 0) / (filters?.limit || 20),
          ),
        },
      };
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to fetch infrastructure providers",
      );
    }
  },

  // Get a specific infrastructure provider
  async getProvider(id: string): Promise<InfrastructureProvider> {
    try {
      const response = await api.get(`/admin/infrastructure/providers/${id}`);
      return response.data.provider || response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to fetch infrastructure provider",
      );
    }
  },

  // Create a new infrastructure provider
  async createProvider(
    data: CreateInfrastructureProviderRequest,
  ): Promise<InfrastructureProvider> {
    try {
      const response = await api.post("/admin/infrastructure/providers", data);
      return response.data.provider || response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to create infrastructure provider",
      );
    }
  },

  // Update an infrastructure provider
  async updateProvider(
    id: string,
    data: UpdateInfrastructureProviderRequest,
  ): Promise<InfrastructureProvider> {
    try {
      const response = await api.put(
        `/admin/infrastructure/providers/${id}`,
        data,
      );
      return response.data.provider || response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to update infrastructure provider",
      );
    }
  },

  // Delete an infrastructure provider
  async deleteProvider(id: string): Promise<void> {
    try {
      await api.delete(`/admin/infrastructure/providers/${id}`);
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to delete infrastructure provider",
      );
    }
  },

  // Test connection to an infrastructure provider
  async testProviderConnection(
    id: string,
  ): Promise<TestProviderConnectionResponse> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${id}/test-connection`,
      );
      return response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to test provider connection",
      );
    }
  },

  // Get health status of an infrastructure provider
  async getProviderHealth(id: string): Promise<InfrastructureProviderHealth> {
    try {
      const response = await api.get(
        `/admin/infrastructure/providers/${id}/health`,
      );
      return response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to get provider health",
      );
    }
  },

  async getTektonStatus(id: string, limit = 20): Promise<TektonProviderStatus> {
    try {
      const response = await api.get(
        `/admin/infrastructure/providers/${id}/tekton/status`,
        {
          params: { limit },
        },
      );
      return response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to get Tekton status",
      );
    }
  },

  async installTekton(
    id: string,
    payload?: {
      install_mode?: TektonInstallMode;
      asset_version?: string;
      idempotency_key?: string;
    },
  ): Promise<TektonInstallerJob> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${id}/tekton/install`,
        payload || {},
      );
      return response.data?.job;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to start Tekton install",
      );
    }
  },

  async upgradeTekton(
    id: string,
    payload?: {
      install_mode?: TektonInstallMode;
      asset_version?: string;
      idempotency_key?: string;
    },
  ): Promise<TektonInstallerJob> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${id}/tekton/upgrade`,
        payload || {},
      );
      return response.data?.job;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to start Tekton upgrade",
      );
    }
  },

  async validateTekton(
    id: string,
    payload?: {
      install_mode?: TektonInstallMode;
      asset_version?: string;
      idempotency_key?: string;
    },
  ): Promise<TektonInstallerJob> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${id}/tekton/validate`,
        payload || {},
      );
      return response.data?.job;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to start Tekton validation",
      );
    }
  },

  async retryTektonJob(id: string, jobId: string): Promise<TektonInstallerJob> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${id}/tekton/retry`,
        {
          job_id: jobId,
        },
      );
      return response.data?.job;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to retry Tekton job",
      );
    }
  },

  async prepareProvider(
    id: string,
    payload?: { requested_actions?: Record<string, any> },
  ): Promise<ProviderPrepareRun> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${id}/prepare`,
        payload || {},
      );
      return response.data?.run;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to start provider prepare run",
      );
    }
  },

  async getProviderPrepareStatus(id: string): Promise<ProviderPrepareStatus> {
    try {
      const response = await api.get(
        `/admin/infrastructure/providers/${id}/prepare/status`,
      );
      return response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to get provider prepare status",
      );
    }
  },

  async listProviderPrepareRuns(
    id: string,
    params?: { limit?: number; offset?: number },
  ): Promise<ProviderPrepareRun[]> {
    try {
      const response = await api.get(
        `/admin/infrastructure/providers/${id}/prepare/runs`,
        {
          params: {
            limit: params?.limit,
            offset: params?.offset,
          },
        },
      );
      return response.data?.runs || [];
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to list provider prepare runs",
      );
    }
  },

  async getProviderPrepareRun(
    id: string,
    runId: string,
    params?: { limit?: number; offset?: number },
  ): Promise<ProviderPrepareStatus> {
    try {
      const response = await api.get(
        `/admin/infrastructure/providers/${id}/prepare/runs/${runId}`,
        {
          params: {
            limit: params?.limit,
            offset: params?.offset,
          },
        },
      );
      return response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to get provider prepare run details",
      );
    }
  },

  async provisionTenantNamespace(
    providerId: string,
    tenantId: string,
  ): Promise<ProviderTenantNamespacePrepare> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${providerId}/tenants/${tenantId}/provision-namespace`,
      );
      return response.data?.prepare;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to provision tenant namespace",
      );
    }
  },

  async deprovisionTenantNamespace(
    providerId: string,
    tenantId: string,
  ): Promise<ProviderTenantNamespacePrepare> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${providerId}/tenants/${tenantId}/deprovision-namespace`,
      );
      return response.data?.prepare;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to deprovision tenant namespace",
      );
    }
  },

  async getTenantNamespaceProvisionStatus(
    providerId: string,
    tenantId: string,
  ): Promise<ProviderTenantNamespacePrepare | null> {
    try {
      const response = await api.get(
        `/admin/infrastructure/providers/${providerId}/tenants/${tenantId}/provision-namespace/status`,
      );
      return response.data?.prepare || null;
    } catch (error: any) {
      // A missing row is not an error; backend returns prepare=null in that case.
      if (error?.response?.status === 404) return null;
      throw new Error(
        error?.response?.data?.error ||
          "Failed to load tenant namespace provision status",
      );
    }
  },

  async reconcileStaleTenantNamespaces(
    providerId: string,
  ): Promise<TenantNamespaceReconcileSummary> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${providerId}/tenants/reconcile-stale`,
      );
      return response.data?.summary;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to reconcile stale tenant namespaces",
      );
    }
  },

  async reconcileSelectedTenantNamespaces(
    providerId: string,
    tenantIds: string[],
  ): Promise<TenantNamespaceReconcileSummary> {
    try {
      const response = await api.post(
        `/admin/infrastructure/providers/${providerId}/tenants/reconcile-selected`,
        { tenant_ids: tenantIds },
      );
      return response.data?.summary;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to reconcile selected tenant namespaces",
      );
    }
  },

  async getProviderPrepareSummaries(
    providerIds: string[],
  ): Promise<ProviderPrepareSummary[]> {
    const result = await this.getProviderPrepareSummariesWithMetrics(
      providerIds,
      false,
    );
    return result.summaries;
  },

  async getProviderPrepareSummariesWithMetrics(
    providerIds: string[],
    includeBatchMetrics = true,
  ): Promise<{
    summaries: ProviderPrepareSummary[];
    batch_metrics?: ProviderPrepareSummaryBatchMetrics;
  }> {
    if (!providerIds.length) {
      return { summaries: [] };
    }
    try {
      const response = await api.get(
        "/admin/infrastructure/providers/prepare/summary",
        {
          params: {
            provider_ids: providerIds.join(","),
            include_batch_metrics: includeBatchMetrics,
          },
        },
      );
      return {
        summaries: response.data?.summaries || [],
        batch_metrics: response.data?.batch_metrics,
      };
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to fetch provider prepare summaries",
      );
    }
  },

  // Enable/disable an infrastructure provider
  async toggleProviderStatus(
    id: string,
    enabled: boolean,
  ): Promise<InfrastructureProvider> {
    try {
      const response = await api.patch(
        `/admin/infrastructure/providers/${id}/status`,
        {
          enabled,
        },
      );
      return response.data.provider || response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to update provider status",
      );
    }
  },

  // Get available infrastructure options for users (read-only)
  async getAvailableOptions(): Promise<InfrastructureProvider[]> {
    try {
      const response = await api.get("/infrastructure/providers/available");
      return response.data.providers || response.data.data || [];
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to fetch available infrastructure options",
      );
    }
  },

  // Permissions (admin)
  async getProviderPermissions(
    id: string,
  ): Promise<Array<{ tenant_id: string; permission: string }>> {
    try {
      const response = await api.get(
        `/admin/infrastructure/providers/${id}/permissions`,
      );
      return response.data.permissions || [];
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to fetch provider permissions",
      );
    }
  },

  async grantProviderPermission(
    id: string,
    tenantId: string,
    permission = "infrastructure:select",
  ): Promise<void> {
    try {
      await api.post(`/admin/infrastructure/providers/${id}/permissions`, {
        tenant_id: tenantId,
        permission,
      });
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to grant provider permission",
      );
    }
  },

  async revokeProviderPermission(
    id: string,
    tenantId: string,
    permission = "infrastructure:select",
  ): Promise<void> {
    try {
      await api.delete(`/admin/infrastructure/providers/${id}/permissions`, {
        data: { tenant_id: tenantId, permission },
      });
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error || "Failed to revoke provider permission",
      );
    }
  },

  // Get infrastructure recommendation for a build
  async getRecommendation(data: {
    build_method: string;
    project_id: string;
    config?: Record<string, any>;
  }): Promise<any> {
    try {
      const response = await api.post(
        "/builds/infrastructure-recommendation",
        data,
      );
      return response.data;
    } catch (error: any) {
      throw new Error(
        error?.response?.data?.error ||
          "Failed to get infrastructure recommendation",
      );
    }
  },
};
