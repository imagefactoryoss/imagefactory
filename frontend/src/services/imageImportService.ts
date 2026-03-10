import type {
  BuildWorkflowResponse,
  ImageImportRequest,
  ReleasedArtifact,
} from "../types";
import { api } from "./api";

type ListImageImportRequestsResponse = {
  data: ImageImportRequest[];
  pagination?: {
    page: number;
    limit: number;
  };
};

type ListReleasedArtifactsResponse = {
  data: ReleasedArtifact[];
  pagination?: {
    page: number;
    limit: number;
    total: number;
  };
};

type CreateImageImportRequestResponse = {
  data: ImageImportRequest;
};

type RetryImageImportRequestResponse = {
  data: ImageImportRequest;
};

type WithdrawImageImportRequestResponse = {
  data: ImageImportRequest;
};

type ReleaseImageImportRequestResponse = {
  data: ImageImportRequest;
};

export interface ReleaseImageImportRequestInput {
  destinationImageRef: string;
  destinationRegistryAuthId?: string;
}

type ConsumeReleasedArtifactResponse = {
  data: {
    id: string;
    consumed: boolean;
    internal_image_ref: string;
    project_id?: string;
  };
};

type ApproveImageImportRequestResponse = {
  data: {
    import_request_id: string;
    approved: boolean;
    status: string;
  };
};

export interface ImageImportLogEntry {
  timestamp?: string;
  level?: string;
  message?: string;
  metadata?: Record<string, any>;
}

export interface ImageImportLifecycleStage {
  key: string;
  label: string;
  description: string;
  state: "pending" | "current" | "complete" | "failed";
  timestamp?: string;
}

export interface ImageImportWorkflowResponse extends BuildWorkflowResponse {
  lifecycleStages: ImageImportLifecycleStage[];
}

type ImageImportLogsResponse = {
  import_request_id: string;
  logs: ImageImportLogEntry[];
  total: number;
  has_more: boolean;
};

export interface CreateImageImportRequestInput {
  eprRecordId: string;
  sourceRegistry: string;
  sourceImageRef: string;
  registryAuthId?: string;
}

export interface CreateOnDemandScanRequestInput {
  sourceRegistry: string;
  sourceImageRef: string;
  registryAuthId?: string;
}

export type ImageImportApiErrorCode =
  | "tenant_capability_not_entitled"
  | "epr_registration_required"
  | "import_not_retryable"
  | "retry_backoff_active"
  | "retry_attempt_limit_reached"
  | "import_not_withdrawable"
  | "approval_already_decided"
  | "approval_decision_in_progress"
  | "approval_step_missing"
  | "approval_step_invalid_state"
  | "tenant_context_mismatch"
  | "release_not_eligible"
  | "not_found"
  | "validation_failed"
  | "invalid_request"
  | "invalid_registry_auth_id"
  | "internal_error"
  | "unknown_error";

export class ImageImportApiError extends Error {
  readonly code: ImageImportApiErrorCode;
  readonly status?: number;
  readonly details?: Record<string, unknown>;

  constructor(params: {
    code: ImageImportApiErrorCode;
    message: string;
    status?: number;
    details?: Record<string, unknown>;
  }) {
    super(params.message);
    this.name = "ImageImportApiError";
    this.code = params.code;
    this.status = params.status;
    this.details = params.details;
  }
}

const mapImageImportApiError = (err: any): ImageImportApiError => {
  const payload = err?.response?.data?.error;
  const code =
    (payload?.code as ImageImportApiErrorCode | undefined) || "unknown_error";
  const message =
    (payload?.message as string | undefined) ||
    (typeof err?.message === "string"
      ? err.message
      : "Failed to create import request");
  const status = err?.response?.status as number | undefined;
  const details = payload?.details as Record<string, unknown> | undefined;

  return new ImageImportApiError({
    code,
    message,
    status,
    details,
  });
};

class ImageImportService {
  async listImportRequests(
    limit: number = 5,
    page: number = 1,
  ): Promise<ImageImportRequest[]> {
    try {
      const response = await api.get<ListImageImportRequestsResponse>(
        `/images/import-requests?limit=${limit}&page=${page}`,
      );
      return response.data.data || [];
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async listAdminImportRequests(
    limit: number = 50,
    page: number = 1,
  ): Promise<ImageImportRequest[]> {
    try {
      const response = await api.get<ListImageImportRequestsResponse>(
        `/admin/images/import-requests?limit=${limit}&page=${page}`,
      );
      return response.data.data || [];
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async getImportRequest(id: string): Promise<ImageImportRequest> {
    try {
      const response = await api.get<CreateImageImportRequestResponse>(
        `/images/import-requests/${id}`,
      );
      return response.data.data;
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async getImportRequestWorkflow(
    id: string,
  ): Promise<ImageImportWorkflowResponse> {
    try {
      const response = await api.get(`/images/import-requests/${id}/workflow`);
      return {
        instanceId: response.data?.instance_id,
        executionId: response.data?.execution_id,
        status: response.data?.status,
        steps: (response.data?.steps || []).map((step: any) => ({
          stepKey: step.step_key,
          status: step.status,
          attempts: step.attempts || 0,
          lastError: step.last_error ?? null,
          startedAt: step.started_at ?? null,
          completedAt: step.completed_at ?? null,
          createdAt: step.created_at,
          updatedAt: step.updated_at,
        })),
        lifecycleStages: (response.data?.lifecycle_stages || []).map(
          (stage: any) => ({
            key: stage.key,
            label: stage.label,
            description: stage.description,
            state: stage.state,
            timestamp: stage.timestamp || undefined,
          }),
        ),
      };
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async getAdminImportRequestWorkflow(
    id: string,
  ): Promise<ImageImportWorkflowResponse> {
    try {
      const response = await api.get(
        `/admin/images/import-requests/${id}/workflow`,
      );
      return {
        instanceId: response.data?.instance_id,
        executionId: response.data?.execution_id,
        status: response.data?.status,
        steps: (response.data?.steps || []).map((step: any) => ({
          stepKey: step.step_key,
          status: step.status,
          attempts: step.attempts || 0,
          lastError: step.last_error ?? null,
          startedAt: step.started_at ?? null,
          completedAt: step.completed_at ?? null,
          createdAt: step.created_at,
          updatedAt: step.updated_at,
        })),
        lifecycleStages: (response.data?.lifecycle_stages || []).map(
          (stage: any) => ({
            key: stage.key,
            label: stage.label,
            description: stage.description,
            state: stage.state,
            timestamp: stage.timestamp || undefined,
          }),
        ),
      };
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async getImportRequestLogs(
    id: string,
    options?: {
      source?: "all" | "tekton" | "lifecycle";
      minLevel?: "debug" | "info" | "warn" | "error";
      limit?: number;
      offset?: number;
    },
  ): Promise<{
    importRequestId: string;
    entries: ImageImportLogEntry[];
    total: number;
    hasMore: boolean;
  }> {
    try {
      const params: Record<string, any> = {};
      if (options?.source) params.source = options.source;
      if (options?.minLevel) params.min_level = options.minLevel;
      if (options?.limit) params.limit = options.limit;
      if (options?.offset) params.offset = options.offset;
      const response = await api.get<ImageImportLogsResponse>(
        `/images/import-requests/${id}/logs`,
        { params },
      );
      return {
        importRequestId: response.data?.import_request_id || id,
        entries: response.data?.logs || [],
        total: response.data?.total || 0,
        hasMore: response.data?.has_more || false,
      };
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async getAdminImportRequestLogs(
    id: string,
    options?: {
      source?: "all" | "tekton" | "lifecycle";
      minLevel?: "debug" | "info" | "warn" | "error";
      limit?: number;
      offset?: number;
    },
  ): Promise<{
    importRequestId: string;
    entries: ImageImportLogEntry[];
    total: number;
    hasMore: boolean;
  }> {
    try {
      const params: Record<string, any> = {};
      if (options?.source) params.source = options.source;
      if (options?.minLevel) params.min_level = options.minLevel;
      if (options?.limit) params.limit = options.limit;
      if (options?.offset) params.offset = options.offset;
      const response = await api.get<ImageImportLogsResponse>(
        `/admin/images/import-requests/${id}/logs`,
        { params },
      );
      return {
        importRequestId: response.data?.import_request_id || id,
        entries: response.data?.logs || [],
        total: response.data?.total || 0,
        hasMore: response.data?.has_more || false,
      };
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async getAdminImportRequest(id: string): Promise<ImageImportRequest> {
    const limit = 100;
    for (let page = 1; page <= 20; page += 1) {
      const items = await this.listAdminImportRequests(limit, page);
      const found = items.find((item) => item.id === id);
      if (found) {
        return found;
      }
      if (items.length < limit) {
        break;
      }
    }

    throw new ImageImportApiError({
      code: "not_found",
      message: "import request not found",
      status: 404,
    });
  }

  async createImportRequest(
    input: CreateImageImportRequestInput,
  ): Promise<ImageImportRequest> {
    try {
      const response = await api.post<CreateImageImportRequestResponse>(
        "/images/import-requests",
        {
          epr_record_id: input.eprRecordId,
          source_registry: input.sourceRegistry,
          source_image_ref: input.sourceImageRef,
          registry_auth_id: input.registryAuthId || undefined,
        },
      );
      return response.data.data;
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async retryImportRequest(id: string): Promise<ImageImportRequest> {
    try {
      const response = await api.post<RetryImageImportRequestResponse>(
        `/images/import-requests/${id}/retry`,
      );
      return response.data.data;
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async withdrawImportRequest(
    id: string,
    reason?: string,
  ): Promise<ImageImportRequest> {
    try {
      const response = await api.post<WithdrawImageImportRequestResponse>(
        `/images/import-requests/${id}/withdraw`,
        {
          reason: reason?.trim() || undefined,
        },
      );
      return response.data.data;
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async approveImportRequest(id: string): Promise<void> {
    try {
      await api.post<ApproveImageImportRequestResponse>(
        `/images/import-requests/${id}/approve`,
      );
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async approveAdminImportRequest(id: string): Promise<void> {
    try {
      await api.post<ApproveImageImportRequestResponse>(
        `/admin/images/import-requests/${id}/approve`,
      );
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async rejectImportRequest(id: string, reason?: string): Promise<void> {
    try {
      await api.post<ApproveImageImportRequestResponse>(
        `/images/import-requests/${id}/reject`,
        {
          reason: reason?.trim() || undefined,
        },
      );
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async rejectAdminImportRequest(id: string, reason?: string): Promise<void> {
    try {
      await api.post<ApproveImageImportRequestResponse>(
        `/admin/images/import-requests/${id}/reject`,
        {
          reason: reason?.trim() || undefined,
        },
      );
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async releaseImportRequest(
    id: string,
    input: ReleaseImageImportRequestInput,
  ): Promise<ImageImportRequest> {
    try {
      const response = await api.post<ReleaseImageImportRequestResponse>(
        `/images/import-requests/${id}/release`,
        {
          destination_image_ref: input.destinationImageRef.trim(),
          destination_registry_auth_id:
            input.destinationRegistryAuthId?.trim() || undefined,
        },
      );
      return response.data.data;
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async listScanRequests(limit: number = 25): Promise<ImageImportRequest[]> {
    try {
      const response = await api.get<ListImageImportRequestsResponse>(
        `/images/scan-requests?limit=${limit}&page=1`,
      );
      return response.data.data || [];
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async createScanRequest(
    input: CreateOnDemandScanRequestInput,
  ): Promise<ImageImportRequest> {
    try {
      const response = await api.post<CreateImageImportRequestResponse>(
        "/images/scan-requests",
        {
          source_registry: input.sourceRegistry,
          source_image_ref: input.sourceImageRef,
          registry_auth_id: input.registryAuthId || undefined,
        },
      );
      return response.data.data;
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async retryScanRequest(id: string): Promise<ImageImportRequest> {
    try {
      const response = await api.post<RetryImageImportRequestResponse>(
        `/images/scan-requests/${id}/retry`,
      );
      return response.data.data;
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async listReleasedArtifacts(params?: {
    limit?: number;
    page?: number;
    search?: string;
  }): Promise<{
    items: ReleasedArtifact[];
    pagination: { page: number; limit: number; total: number };
  }> {
    const limit = params?.limit ?? 25;
    const page = params?.page ?? 1;
    const search = (params?.search || "").trim();
    const query = new URLSearchParams({
      limit: String(limit),
      page: String(page),
    });
    if (search) {
      query.set("search", search);
    }

    try {
      const response = await api.get<ListReleasedArtifactsResponse>(
        `/images/released-artifacts?${query.toString()}`,
      );
      const pagination = response.data.pagination || { page, limit, total: 0 };
      return {
        items: response.data.data || [],
        pagination,
      };
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }

  async consumeReleasedArtifact(
    id: string,
    projectId?: string,
    notes?: string,
  ): Promise<void> {
    try {
      await api.post<ConsumeReleasedArtifactResponse>(
        `/images/released-artifacts/${id}/consume`,
        {
          project_id: projectId?.trim() || undefined,
          notes: notes?.trim() || undefined,
        },
      );
    } catch (err) {
      throw mapImageImportApiError(err);
    }
  }
}

export const imageImportService = new ImageImportService();
