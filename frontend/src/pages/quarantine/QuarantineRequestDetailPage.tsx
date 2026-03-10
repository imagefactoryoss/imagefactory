import {
  imageImportService,
  type ImageImportLifecycleStage,
  type ImageImportLogEntry,
  type ImageImportWorkflowResponse,
} from "@/services/imageImportService";
import { useAuthStore } from "@/store/auth";
import { useTenantStore } from "@/store/tenant";
import type { ImageImportRequest } from "@/types";
import {
  getImportProgressLabel,
  getImportSyncStateLabel,
  hasMeaningfulJSONEvidence,
} from "@/utils/imageImportDiagnostics";
import {
  AlertTriangle,
  ArrowLeft,
  Check,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Circle,
  CircleDot,
  Copy,
  FileText,
  RefreshCw,
} from "lucide-react";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation, useParams } from "react-router-dom";

interface QuarantineRequestDetailPageProps {
  scope: "tenant" | "admin";
}

const lifecycleStageCircleClasses = (
  state: ImageImportLifecycleStage["state"],
) => {
  switch (state) {
    case "complete":
      return "border-emerald-300 bg-emerald-100 text-emerald-700 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300";
    case "current":
      return "border-blue-300 bg-blue-100 text-blue-700 dark:border-blue-700 dark:bg-blue-950/40 dark:text-blue-300";
    case "failed":
      return "border-rose-300 bg-rose-100 text-rose-700 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-300";
    default:
      return "border-slate-300 bg-slate-100 text-slate-500 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-400";
  }
};

const STATUS_BADGE: Record<string, string> = {
  pending:
    "bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-200",
  approved: "bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-200",
  importing:
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/50 dark:text-indigo-200",
  success:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200",
  quarantined:
    "bg-purple-100 text-purple-800 dark:bg-purple-900/50 dark:text-purple-200",
  failed: "bg-rose-100 text-rose-800 dark:bg-rose-900/50 dark:text-rose-200",
};

const RELEASE_BADGE: Record<string, string> = {
  not_ready:
    "bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-200",
  ready_for_release:
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/50 dark:text-indigo-200",
  release_approved:
    "bg-sky-100 text-sky-800 dark:bg-sky-900/50 dark:text-sky-200",
  released:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200",
  release_blocked:
    "bg-rose-100 text-rose-800 dark:bg-rose-900/50 dark:text-rose-200",
  unknown: "bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-200",
};

const parseJSONSummary = (raw?: string) => {
  if (!raw) return "";
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
};

const formatDateTime = (value?: string) => {
  if (!value) return "n/a";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString();
};

const isTerminal = (request: ImageImportRequest) =>
  request.status === "success" ||
  request.status === "failed" ||
  request.status === "quarantined";

const QuarantineRequestDetailPage: React.FC<
  QuarantineRequestDetailPageProps
> = ({ scope }) => {
  const { requestId } = useParams<{ requestId: string }>();
  const location = useLocation();
  const { token } = useAuthStore();
  const { selectedTenantId } = useTenantStore();
  const [request, setRequest] = useState<ImageImportRequest | null>(null);
  const [workflow, setWorkflow] = useState<ImageImportWorkflowResponse | null>(
    null,
  );
  const [workflowError, setWorkflowError] = useState<string | null>(null);
  const [lifecycleLogs, setLifecycleLogs] = useState<ImageImportLogEntry[]>([]);
  const [executionLogs, setExecutionLogs] = useState<ImageImportLogEntry[]>([]);
  const [logsError, setLogsError] = useState<string | null>(null);
  const [activeLogTab, setActiveLogTab] = useState<"lifecycle" | "execution">(
    "execution",
  );
  const [isLogStreamConnected, setIsLogStreamConnected] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [openLifecycleErrorKey, setOpenLifecycleErrorKey] = useState<
    string | null
  >(null);
  const [copiedLifecycleErrorKey, setCopiedLifecycleErrorKey] = useState<
    string | null
  >(null);
  const [expandedExecutionTasks, setExpandedExecutionTasks] = useState<
    Record<string, boolean>
  >({});
  const logStreamSocketRef = useRef<WebSocket | null>(null);
  const logStreamReconnectTimerRef = useRef<number | undefined>(undefined);

  const loadRequest = useCallback(
    async (silent: boolean = false) => {
      if (!requestId) {
        setError("Missing request id");
        setLoading(false);
        return;
      }

      if (silent) {
        setRefreshing(true);
      } else {
        setLoading(true);
        setError(null);
      }

      try {
        const row =
          scope === "admin"
            ? await imageImportService.getAdminImportRequest(requestId)
            : await imageImportService.getImportRequest(requestId);
        setRequest(row);

        try {
          const workflowData =
            scope === "admin"
              ? await imageImportService.getAdminImportRequestWorkflow(
                  requestId,
                )
              : await imageImportService.getImportRequestWorkflow(requestId);
          setWorkflow(workflowData);
          setWorkflowError(null);
        } catch (workflowErr: any) {
          setWorkflow(null);
          setWorkflowError(
            workflowErr?.message || "Failed to load workflow details",
          );
        }
      } catch (err: any) {
        setError(err?.message || "Failed to load quarantine request");
        setRequest(null);
        setWorkflow(null);
      } finally {
        if (silent) {
          setRefreshing(false);
        } else {
          setLoading(false);
        }
      }
    },
    [requestId, scope],
  );

  const loadLogs = useCallback(async () => {
    if (!requestId) return;
    try {
      const [lifecycleResponse, executionResponse] =
        scope === "admin"
          ? await Promise.all([
              imageImportService.getAdminImportRequestLogs(requestId, {
                source: "lifecycle",
                limit: 500,
              }),
              imageImportService.getAdminImportRequestLogs(requestId, {
                source: "tekton",
                limit: 500,
              }),
            ])
          : await Promise.all([
              imageImportService.getImportRequestLogs(requestId, {
                source: "lifecycle",
                limit: 500,
              }),
              imageImportService.getImportRequestLogs(requestId, {
                source: "tekton",
                limit: 500,
              }),
            ]);
      setLifecycleLogs(lifecycleResponse.entries || []);
      setExecutionLogs(executionResponse.entries || []);
      setLogsError(null);
    } catch (err: any) {
      setLifecycleLogs([]);
      setExecutionLogs([]);
      setLogsError(err?.message || "Failed to load request logs");
    }
  }, [requestId, scope]);

  useEffect(() => {
    void loadRequest();
  }, [loadRequest]);

  useEffect(() => {
    void loadLogs();
  }, [loadLogs]);

  useEffect(() => {
    if (!request || !autoRefresh || isTerminal(request)) {
      return;
    }
    const timer = window.setInterval(() => {
      void loadRequest(true);
      void loadLogs();
    }, 8000);
    return () => {
      window.clearInterval(timer);
    };
  }, [autoRefresh, loadLogs, loadRequest, request]);

  useEffect(() => {
    if (!requestId || !token) {
      setIsLogStreamConnected(false);
      return;
    }

    const effectiveTenantID =
      scope === "admin"
        ? request?.tenant_id || selectedTenantId
        : selectedTenantId || request?.tenant_id;
    if (!effectiveTenantID) {
      setIsLogStreamConnected(false);
      return;
    }

    let active = true;
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const basePath =
      scope === "admin"
        ? `/api/v1/admin/images/import-requests/${requestId}/logs/stream`
        : `/api/v1/images/import-requests/${requestId}/logs/stream`;
    const wsUrl = `${protocol}//${window.location.host}${basePath}?token=${encodeURIComponent(token)}&tenant_id=${encodeURIComponent(effectiveTenantID)}`;

    const appendDedupedEntry = (
      current: ImageImportLogEntry[],
      entry: ImageImportLogEntry,
    ): ImageImportLogEntry[] => {
      const last = current[current.length - 1];
      if (
        last &&
        last.timestamp === entry.timestamp &&
        last.message === entry.message &&
        last.metadata?.task_run === entry.metadata?.task_run &&
        last.metadata?.step === entry.metadata?.step
      ) {
        return current;
      }
      return [...current, entry];
    };

    const connectLogStream = () => {
      if (!active) return;
      const existing = logStreamSocketRef.current;
      if (
        existing &&
        (existing.readyState === WebSocket.OPEN ||
          existing.readyState === WebSocket.CONNECTING)
      ) {
        return;
      }

      const socket = new WebSocket(wsUrl);
      logStreamSocketRef.current = socket;

      socket.onopen = () => {
        if (!active) return;
        setIsLogStreamConnected(true);
      };

      socket.onclose = () => {
        if (!active) return;
        if (logStreamSocketRef.current === socket) {
          logStreamSocketRef.current = null;
        }
        setIsLogStreamConnected(false);
        if (!request || isTerminal(request)) {
          return;
        }
        logStreamReconnectTimerRef.current = window.setTimeout(
          connectLogStream,
          2500,
        );
      };

      socket.onerror = () => {
        setIsLogStreamConnected(false);
      };

      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data || "{}");
          const entry: ImageImportLogEntry = {
            timestamp: payload.timestamp,
            level: payload.level || "INFO",
            message: payload.message || "",
            metadata: payload.metadata || {},
          };
          const source = String(entry.metadata?.source || "").toLowerCase();
          if (source === "tekton") {
            setExecutionLogs((current) => appendDedupedEntry(current, entry));
          } else {
            setLifecycleLogs((current) => appendDedupedEntry(current, entry));
          }
        } catch {
          // Ignore malformed payloads from stream.
        }
      };
    };

    connectLogStream();

    return () => {
      active = false;
      setIsLogStreamConnected(false);
      if (logStreamReconnectTimerRef.current) {
        window.clearTimeout(logStreamReconnectTimerRef.current);
        logStreamReconnectTimerRef.current = undefined;
      }
      if (logStreamSocketRef.current) {
        logStreamSocketRef.current.close();
        logStreamSocketRef.current = null;
      }
    };
  }, [request, requestId, scope, selectedTenantId, token]);

  const backPath = useMemo(() => {
    if (location.pathname.startsWith("/reviewer/")) {
      return "/reviewer/quarantine/requests";
    }
    if (scope === "admin") {
      return "/admin/quarantine/requests";
    }
    return "/quarantine/requests";
  }, [location.pathname, scope]);

  const visibleLogs = useMemo(
    () => (activeLogTab === "execution" ? executionLogs : lifecycleLogs),
    [activeLogTab, executionLogs, lifecycleLogs],
  );
  const executionTaskGroups = useMemo(() => {
    const taskMap: Map<string, Map<string, ImageImportLogEntry[]>> = new Map();
    executionLogs.forEach((entry) => {
      const taskRun = entry.metadata?.task_run
        ? String(entry.metadata.task_run)
        : "__ungrouped__";
      const step = entry.metadata?.step ? String(entry.metadata.step) : "__other__";
      if (!taskMap.has(taskRun)) taskMap.set(taskRun, new Map());
      const stepMap = taskMap.get(taskRun)!;
      if (!stepMap.has(step)) stepMap.set(step, []);
      stepMap.get(step)!.push(entry);
    });
    return Array.from(taskMap.entries());
  }, [executionLogs]);

  const latestExecutionSignal = useMemo(() => {
    const prioritized = [...executionLogs].sort((a, b) => {
      const ta = a.timestamp ? Date.parse(a.timestamp) : 0;
      const tb = b.timestamp ? Date.parse(b.timestamp) : 0;
      return tb - ta;
    });
    if (prioritized.length === 0) return "";

    const statusSignal = prioritized.find((entry) => {
      const signalType = String(entry.metadata?.signal_type || "").toLowerCase();
      return signalType === "taskrun_status" || signalType === "step_status";
    });
    if (statusSignal) {
      const taskRun = statusSignal.metadata?.task_run
        ? String(statusSignal.metadata.task_run)
        : "";
      const step = statusSignal.metadata?.step
        ? String(statusSignal.metadata.step)
        : "";
      const reason = statusSignal.metadata?.reason
        ? String(statusSignal.metadata.reason)
        : "";
      const msg = String(statusSignal.message || "").trim();
      const prefix = [taskRun, step].filter(Boolean).join(" / ");
      const body = reason && msg ? `${reason}: ${msg}` : reason || msg;
      if (!body) return "";
      return prefix ? `${prefix}: ${body}` : body;
    }

    const isSignalEntry = (entry: ImageImportLogEntry) => {
      const level = String(entry.level || "").toLowerCase();
      const msg = String(entry.message || "").toLowerCase();
      return (
        level === "error" ||
        level === "warn" ||
        msg.includes("error") ||
        msg.includes("failed") ||
        msg.includes("timeout")
      );
    };

    const picked = prioritized.find(isSignalEntry) || prioritized[0];
    const task = picked.metadata?.task_run
      ? String(picked.metadata.task_run)
      : "";
    const step = picked.metadata?.step ? String(picked.metadata.step) : "";
    const prefix = [task, step].filter(Boolean).join(" / ");
    const msg = String(picked.message || "").trim();
    if (!msg) return "";
    return prefix ? `${prefix}: ${msg}` : msg;
  }, [executionLogs]);

  useEffect(() => {
    setExpandedExecutionTasks((current) => {
      const next: Record<string, boolean> = {};
      executionTaskGroups.forEach(([taskRun], index) => {
        if (Object.prototype.hasOwnProperty.call(current, taskRun)) {
          next[taskRun] = current[taskRun];
          return;
        }
        next[taskRun] = index < 3;
      });
      return next;
    });
  }, [executionTaskGroups]);

  const toggleExecutionTask = useCallback((taskRun: string) => {
    setExpandedExecutionTasks((current) => ({
      ...current,
      [taskRun]: !current[taskRun],
    }));
  }, []);
  const expandAllExecutionTasks = useCallback(() => {
    const next: Record<string, boolean> = {};
    executionTaskGroups.forEach(([taskRun]) => {
      next[taskRun] = true;
    });
    setExpandedExecutionTasks(next);
  }, [executionTaskGroups]);
  const collapseAllExecutionTasks = useCallback(() => {
    const next: Record<string, boolean> = {};
    executionTaskGroups.forEach(([taskRun]) => {
      next[taskRun] = false;
    });
    setExpandedExecutionTasks(next);
  }, [executionTaskGroups]);

  const lifecycleStageErrors = useMemo(() => {
    const byStage = new Map<string, string>();
    const addError = (stageKey: string, errorText?: string | null) => {
      const normalized = (errorText || "").trim();
      if (!normalized || byStage.has(stageKey)) return;
      byStage.set(stageKey, normalized);
    };

    const maybeAppendExecutionSignal = (stageKey: string) => {
      if (!latestExecutionSignal) return;
      const current = byStage.get(stageKey);
      if (!current) return;
      if (current.includes(latestExecutionSignal)) return;
      byStage.set(
        stageKey,
        `${current}\n\nLatest pipeline signal: ${latestExecutionSignal}`,
      );
    };

    (workflow?.steps || []).forEach((step) => {
      const status = (step.status || "").toLowerCase();
      if (status !== "failed" && status !== "blocked") return;
      const errorText = step.lastError;
      switch (step.stepKey) {
        case "approval.request":
        case "approval.decision":
          addError("awaiting_approval", errorText);
          break;
        case "import.dispatch":
          addError("awaiting_dispatch", errorText);
          break;
        case "import.monitor":
          addError("pipeline_running", errorText);
          addError("evidence_pending", errorText);
          addError("ready_for_release", errorText);
          maybeAppendExecutionSignal("pipeline_running");
          maybeAppendExecutionSignal("evidence_pending");
          maybeAppendExecutionSignal("ready_for_release");
          break;
        default:
          break;
      }
    });

    const failedStage = (workflow?.lifecycleStages || []).find(
      (stage) => stage.state === "failed",
    );
    if (failedStage && request?.error_message) {
      addError(failedStage.key, request.error_message);
    }
    return byStage;
  }, [
    latestExecutionSignal,
    request?.error_message,
    workflow?.lifecycleStages,
    workflow?.steps,
  ]);

  const copyLifecycleStageError = useCallback(
    async (stageKey: string, errorText: string) => {
      try {
        await navigator.clipboard.writeText(errorText);
        setCopiedLifecycleErrorKey(stageKey);
        window.setTimeout(() => {
          setCopiedLifecycleErrorKey((current) =>
            current === stageKey ? null : current,
          );
        }, 1200);
      } catch {
        // ignore clipboard errors
      }
    },
    [],
  );

  return (
    <div className="w-full space-y-4 px-4 py-6 sm:space-y-6 sm:px-6 lg:px-8">
      <div className="flex flex-col gap-3 rounded-lg border border-slate-200 bg-white p-3 shadow-sm sm:p-4 dark:border-slate-700 dark:bg-slate-900 md:flex-row md:items-center md:justify-between">
        <div className="min-w-0 space-y-1">
          <h1 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
            Quarantine Request Details
          </h1>
          <p className="text-xs text-slate-600 dark:text-slate-300">
            Execution timeline, diagnostics, evidence, and release-readiness
            context.
          </p>
        </div>
        <div className="flex w-full flex-wrap items-center gap-2 md:w-auto md:justify-end">
          <Link
            to={backPath}
            className="inline-flex flex-1 items-center justify-center gap-1 rounded-md border border-slate-300 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 sm:flex-none dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Back to queue
          </Link>
          <button
            type="button"
            onClick={() => setAutoRefresh((prev) => !prev)}
            className={`inline-flex flex-1 items-center justify-center gap-1 rounded-md border px-3 py-2 text-xs font-medium sm:flex-none ${
              autoRefresh
                ? "border-emerald-300 bg-emerald-50 text-emerald-800 hover:bg-emerald-100 dark:border-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-200 dark:hover:bg-emerald-900/40"
                : "border-slate-300 text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
            }`}
          >
            Auto-refresh {autoRefresh ? "On" : "Off"}
          </button>
          <button
            type="button"
            onClick={() => {
              void loadRequest(true);
              void loadLogs();
            }}
            className="inline-flex flex-1 items-center justify-center gap-1 rounded-md border border-blue-300 bg-blue-50 px-3 py-2 text-xs font-medium text-blue-800 hover:bg-blue-100 sm:flex-none dark:border-blue-700 dark:bg-blue-950/30 dark:text-blue-200 dark:hover:bg-blue-900/40"
          >
            <RefreshCw
              className={`h-3.5 w-3.5 ${refreshing ? "animate-spin" : ""}`}
            />
            {refreshing ? "Refreshing..." : "Refresh"}
          </button>
        </div>
      </div>

      {loading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-400">
          Loading request details...
        </div>
      ) : error ? (
        <div className="rounded-lg border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-200">
          {error}
        </div>
      ) : request ? (
        <div className="space-y-4">
          <div className="rounded-lg border border-slate-200 bg-white p-3 shadow-sm sm:p-4 dark:border-slate-700 dark:bg-slate-900">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
                  Request
                </p>
                <p className="mt-1 break-all text-sm font-semibold text-slate-900 dark:text-slate-100">
                  {request.source_image_ref}
                </p>
                <p className="mt-1 break-all text-xs text-slate-600 dark:text-slate-300">
                  {request.id}
                </p>
              </div>
              <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">
                <span
                  className={`rounded-full px-2.5 py-1 text-xs font-semibold ${STATUS_BADGE[request.status] || STATUS_BADGE.pending}`}
                >
                  {request.status}
                </span>
                <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-200">
                  {getImportProgressLabel(request)}
                </span>
                <span
                  className={`rounded-full px-2.5 py-1 text-xs font-medium ${RELEASE_BADGE[request.release_state || "unknown"] || RELEASE_BADGE.unknown}`}
                >
                  release: {request.release_state || "unknown"}
                </span>
              </div>
            </div>
            <div className="mt-3 grid grid-cols-1 gap-2 text-xs sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-md border border-slate-200 bg-slate-50 p-2 dark:border-slate-700 dark:bg-slate-800/50">
                <p className="text-slate-500 dark:text-slate-400">Sync State</p>
                <p className="mt-1 font-medium text-slate-900 dark:text-slate-100">
                  {getImportSyncStateLabel(request.sync_state)}
                </p>
              </div>
              <div className="rounded-md border border-slate-200 bg-slate-50 p-2 dark:border-slate-700 dark:bg-slate-800/50">
                <p className="text-slate-500 dark:text-slate-400">Pipeline</p>
                <p className="mt-1 font-medium text-slate-900 dark:text-slate-100">
                  {request.pipeline_run_name || "n/a"}
                </p>
              </div>
              <div className="rounded-md border border-slate-200 bg-slate-50 p-2 dark:border-slate-700 dark:bg-slate-800/50">
                <p className="text-slate-500 dark:text-slate-400">Retryable</p>
                <p className="mt-1 font-medium text-slate-900 dark:text-slate-100">
                  {request.retryable ? "Yes" : "No"}
                </p>
              </div>
              <div className="rounded-md border border-slate-200 bg-slate-50 p-2 dark:border-slate-700 dark:bg-slate-800/50">
                <p className="text-slate-500 dark:text-slate-400">Updated</p>
                <p className="mt-1 font-medium text-slate-900 dark:text-slate-100">
                  {formatDateTime(request.updated_at)}
                </p>
              </div>
            </div>
          </div>

          <section className="rounded-lg border border-slate-200 bg-white p-3 shadow-sm sm:p-4 dark:border-slate-700 dark:bg-slate-900">
            <div className="flex items-center justify-between gap-2">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">
                Overview
              </p>
              <p className="text-[11px] text-slate-500 dark:text-slate-400">
                Compact request + workflow context
              </p>
            </div>
            <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Execution
                </p>
                <div className="mt-2 space-y-1 text-xs text-slate-600 dark:text-slate-300">
                  <p>Created: {formatDateTime(request.created_at)}</p>
                  <p>Updated: {formatDateTime(request.updated_at)}</p>
                  <p>Sync: {getImportSyncStateLabel(request.sync_state)}</p>
                  <p>State: {getImportProgressLabel(request)}</p>
                  {request.dispatch_queued_at ? (
                    <p>Queued: {formatDateTime(request.dispatch_queued_at)}</p>
                  ) : null}
                  {request.pipeline_started_at ? (
                    <p>
                      Started: {formatDateTime(request.pipeline_started_at)}
                    </p>
                  ) : null}
                  {request.evidence_ready_at ? (
                    <p>Evidence: {formatDateTime(request.evidence_ready_at)}</p>
                  ) : null}
                  {request.release_ready_at ? (
                    <p>
                      Release-ready: {formatDateTime(request.release_ready_at)}
                    </p>
                  ) : null}
                </div>
              </div>

              <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Request Metadata
                </p>
                <div className="mt-2 space-y-1 text-xs text-slate-600 dark:text-slate-300">
                  <p className="break-all">ID: {request.id}</p>
                  <p>Type: {request.request_type}</p>
                  <p className="break-all">EPR: {request.epr_record_id}</p>
                  <p className="break-all">
                    Registry: {request.source_registry}
                  </p>
                  <p className="break-all">
                    Retryable: {request.retryable ? "Yes" : "No"}
                  </p>
                  {request.pipeline_run_name ? (
                    <p className="break-all">
                      PipelineRun: {request.pipeline_namespace || "default"}/
                      {request.pipeline_run_name}
                    </p>
                  ) : null}
                  {request.release_blocker_reason ? (
                    <p className="break-all">
                      Release Blocker: {request.release_blocker_reason}
                    </p>
                  ) : null}
                </div>
              </div>

              <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Decision
                </p>
                {request.decision_timeline ? (
                  <div className="mt-2 space-y-1 text-xs text-slate-600 dark:text-slate-300">
                    <p>
                      Decision:{" "}
                      {request.decision_timeline.decision_status || "n/a"}
                    </p>
                    <p>
                      Workflow Step:{" "}
                      {request.decision_timeline.workflow_step_status || "n/a"}
                    </p>
                    {request.decision_timeline.decided_by_user_id ? (
                      <p className="break-all">
                        By: {request.decision_timeline.decided_by_user_id}
                      </p>
                    ) : null}
                    {request.decision_timeline.decided_at ? (
                      <p>
                        At:{" "}
                        {formatDateTime(request.decision_timeline.decided_at)}
                      </p>
                    ) : null}
                    {request.decision_timeline.decision_reason ? (
                      <p className="break-all">
                        Reason: {request.decision_timeline.decision_reason}
                      </p>
                    ) : null}
                  </div>
                ) : (
                  <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                    No decision metadata yet.
                  </p>
                )}
              </div>

              <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Notification
                </p>
                {request.notification_reconciliation ? (
                  <div className="mt-2 space-y-1 text-xs text-slate-600 dark:text-slate-300">
                    <p>
                      Event:{" "}
                      {request.notification_reconciliation
                        .decision_event_type || "n/a"}
                    </p>
                    <p>
                      State:{" "}
                      {request.notification_reconciliation.delivery_state ||
                        "pending"}
                    </p>
                    <p>
                      Receipts:{" "}
                      {request.notification_reconciliation.receipt_count}/
                      {request.notification_reconciliation.expected_recipients}
                    </p>
                    <p>
                      In-app:{" "}
                      {
                        request.notification_reconciliation
                          .in_app_notification_count
                      }
                    </p>
                    {request.notification_reconciliation.idempotency_key ? (
                      <p className="break-all">
                        Key:{" "}
                        {request.notification_reconciliation.idempotency_key}
                      </p>
                    ) : null}
                  </div>
                ) : (
                  <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                    No notification data yet.
                  </p>
                )}
              </div>
            </div>
          </section>

          <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">
                Lifecycle Stages
              </p>
              <div className="flex items-center gap-2 text-[11px] text-slate-500 dark:text-slate-400">
                {workflow?.status ? (
                  <span className="rounded border border-slate-300 px-2 py-0.5 dark:border-slate-600">
                    {workflow.status}
                  </span>
                ) : null}
                {workflow?.instanceId ? (
                  <span className="font-mono">
                    {workflow.instanceId.slice(0, 8)}
                  </span>
                ) : null}
              </div>
            </div>
            <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
              Server-reported stage model from quarantine workflow API.
            </p>
            {workflowError ? (
              <div className="mt-3 rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-950/30 dark:text-amber-200">
                {workflowError}
              </div>
            ) : null}
            {(workflow?.lifecycleStages || []).length === 0 ? (
              <div className="mt-3 rounded-md border border-slate-200 bg-slate-50 p-3 text-xs text-slate-500 dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-400">
                Lifecycle stages are not available for this request yet.
              </div>
            ) : (
              <>
                <div className="mt-3 hidden lg:block">
                  <div className="relative">
                    <div className="absolute left-4 right-4 top-4 h-px bg-slate-300 dark:bg-slate-700" />
                    <ol className="relative z-10 flex items-start gap-3">
                      {(workflow?.lifecycleStages || []).map((stage) => (
                        <li
                          key={stage.key}
                          className="min-w-0 flex-1 text-center"
                        >
                          <div
                            className={`mx-auto flex h-8 w-8 items-center justify-center rounded-full border ${lifecycleStageCircleClasses(stage.state)}`}
                          >
                            {stage.state === "complete" ? (
                              <CheckCircle2 className="h-4 w-4" />
                            ) : stage.state === "current" ? (
                              <CircleDot className="h-4 w-4" />
                            ) : stage.state === "failed" ? (
                              <AlertTriangle className="h-4 w-4" />
                            ) : (
                              <Circle className="h-4 w-4" />
                            )}
                          </div>
                          <p className="mt-2 text-xs font-semibold text-slate-900 dark:text-slate-100">
                            {stage.label}
                          </p>
                          <p className="mt-0.5 text-[11px] text-slate-600 dark:text-slate-300">
                            {stage.description}
                          </p>
                          <p className="mt-1 text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
                            {stage.state}
                          </p>
                          {stage.state === "failed" &&
                          lifecycleStageErrors.has(stage.key) ? (
                            <div className="relative mt-1 inline-flex">
                              <button
                                type="button"
                                onClick={() =>
                                  setOpenLifecycleErrorKey((current) =>
                                    current === stage.key ? null : stage.key,
                                  )
                                }
                                className="inline-flex items-center gap-1 rounded border border-rose-300 bg-rose-50 px-1.5 py-0.5 text-[10px] font-medium text-rose-700 hover:bg-rose-100 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-300 dark:hover:bg-rose-900/50"
                              >
                                <AlertTriangle className="h-3 w-3" />
                                Error
                              </button>
                              {openLifecycleErrorKey === stage.key ? (
                                <div className="absolute left-1/2 top-full z-20 mt-1 w-72 -translate-x-1/2 rounded-md border border-rose-300 bg-white p-2 text-left shadow-lg dark:border-rose-700 dark:bg-slate-900">
                                  <p className="max-h-24 overflow-auto break-words text-[11px] text-rose-700 dark:text-rose-300">
                                    {lifecycleStageErrors.get(stage.key)}
                                  </p>
                                  <button
                                    type="button"
                                    onClick={() =>
                                      void copyLifecycleStageError(
                                        stage.key,
                                        lifecycleStageErrors.get(stage.key) ||
                                          "",
                                      )
                                    }
                                    className="mt-2 inline-flex items-center gap-1 rounded border border-slate-300 px-2 py-0.5 text-[11px] text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                  >
                                    {copiedLifecycleErrorKey === stage.key ? (
                                      <Check className="h-3 w-3" />
                                    ) : (
                                      <Copy className="h-3 w-3" />
                                    )}
                                    {copiedLifecycleErrorKey === stage.key
                                      ? "Copied"
                                      : "Copy"}
                                  </button>
                                </div>
                              ) : null}
                            </div>
                          ) : null}
                          {stage.timestamp ? (
                            <p className="mt-1 text-[10px] text-slate-500 dark:text-slate-400">
                              {formatDateTime(stage.timestamp)}
                            </p>
                          ) : null}
                        </li>
                      ))}
                    </ol>
                  </div>
                </div>
                <div className="mt-3 space-y-2 lg:hidden">
                  {(workflow?.lifecycleStages || []).map((stage) => (
                    <div
                      key={stage.key}
                      className="flex items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/50"
                    >
                      <div
                        className={`flex h-7 w-7 items-center justify-center rounded-full border ${lifecycleStageCircleClasses(stage.state)}`}
                      >
                        {stage.state === "complete" ? (
                          <CheckCircle2 className="h-4 w-4" />
                        ) : stage.state === "current" ? (
                          <CircleDot className="h-4 w-4" />
                        ) : stage.state === "failed" ? (
                          <AlertTriangle className="h-4 w-4" />
                        ) : (
                          <Circle className="h-4 w-4" />
                        )}
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="text-xs font-semibold text-slate-900 dark:text-slate-100">
                            {stage.label}
                          </p>
                          <span className="text-[11px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
                            {stage.state}
                          </span>
                        </div>
                        <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-300">
                          {stage.description}
                        </p>
                        {stage.state === "failed" &&
                        lifecycleStageErrors.has(stage.key) ? (
                          <div className="relative mt-1 inline-flex">
                            <button
                              type="button"
                              onClick={() =>
                                setOpenLifecycleErrorKey((current) =>
                                  current === `mobile-${stage.key}`
                                    ? null
                                    : `mobile-${stage.key}`,
                                )
                              }
                              className="inline-flex items-center gap-1 rounded border border-rose-300 bg-rose-50 px-2 py-0.5 text-[11px] font-medium text-rose-700 hover:bg-rose-100 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-300 dark:hover:bg-rose-900/50"
                            >
                              <AlertTriangle className="h-3 w-3" />
                              Error
                            </button>
                            {openLifecycleErrorKey === `mobile-${stage.key}` ? (
                              <div className="absolute left-0 top-full z-20 mt-1 w-64 rounded-md border border-rose-300 bg-white p-2 text-left shadow-lg dark:border-rose-700 dark:bg-slate-900">
                                <p className="max-h-24 overflow-auto break-words text-[11px] text-rose-700 dark:text-rose-300">
                                  {lifecycleStageErrors.get(stage.key)}
                                </p>
                                <button
                                  type="button"
                                  onClick={() =>
                                    void copyLifecycleStageError(
                                      `mobile-${stage.key}`,
                                      lifecycleStageErrors.get(stage.key) || "",
                                    )
                                  }
                                  className="mt-2 inline-flex items-center gap-1 rounded border border-slate-300 px-2 py-0.5 text-[11px] text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                >
                                  {copiedLifecycleErrorKey ===
                                  `mobile-${stage.key}` ? (
                                    <Check className="h-3 w-3" />
                                  ) : (
                                    <Copy className="h-3 w-3" />
                                  )}
                                  {copiedLifecycleErrorKey ===
                                  `mobile-${stage.key}`
                                    ? "Copied"
                                    : "Copy"}
                                </button>
                              </div>
                            ) : null}
                          </div>
                        ) : null}
                        {stage.timestamp ? (
                          <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
                            {formatDateTime(stage.timestamp)}
                          </p>
                        ) : null}
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}
          </section>

          <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">
                Request Logs
              </p>
              <div className="inline-flex overflow-hidden rounded-md border border-slate-300 dark:border-slate-700">
                {(["lifecycle", "execution"] as const).map((tab) => (
                  <button
                    key={tab}
                    type="button"
                    onClick={() => setActiveLogTab(tab)}
                    className={`px-3 py-1.5 text-[11px] font-medium ${
                      activeLogTab === tab
                        ? "bg-slate-200 text-slate-900 dark:bg-slate-700 dark:text-white"
                        : "bg-white text-slate-600 hover:bg-slate-100 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800"
                    }`}
                  >
                    {tab === "execution"
                      ? `Execution (${executionLogs.length})`
                      : `Lifecycle (${lifecycleLogs.length})`}
                  </button>
                ))}
              </div>
            </div>
            <div className="mt-2 flex flex-wrap items-center gap-2 text-[11px]">
              <span
                className={`inline-flex items-center rounded-full px-2 py-1 font-medium ${
                  isLogStreamConnected
                    ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                    : "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
                }`}
              >
                {isLogStreamConnected ? "Live log stream on" : "Live log stream off"}
              </span>
              <span className="text-slate-500 dark:text-slate-400">
                {activeLogTab === "execution"
                  ? "Pipeline task logs"
                  : "Request lifecycle logs"}
              </span>
              {activeLogTab === "execution" && executionTaskGroups.length > 0 ? (
                <div className="inline-flex overflow-hidden rounded-md border border-slate-300 dark:border-slate-700">
                  <button
                    type="button"
                    onClick={expandAllExecutionTasks}
                    className="bg-white px-2.5 py-1 text-[11px] text-slate-700 hover:bg-slate-100 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
                  >
                    Expand all
                  </button>
                  <button
                    type="button"
                    onClick={collapseAllExecutionTasks}
                    className="border-l border-slate-300 bg-white px-2.5 py-1 text-[11px] text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
                  >
                    Collapse all
                  </button>
                </div>
              ) : null}
            </div>
            {logsError ? (
              <div className="mt-3 rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-950/30 dark:text-amber-200">
                {logsError}
              </div>
            ) : null}
            {visibleLogs.length === 0 ? (
              <div className="mt-3 rounded-md border border-slate-200 bg-slate-50 p-3 text-xs text-slate-500 dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-400">
                No {activeLogTab} logs available yet.
              </div>
            ) : (
              <div className="mt-3 max-h-[36rem] overflow-y-auto rounded-lg border border-slate-800 bg-slate-950 p-4 font-mono text-xs text-slate-200">
                {activeLogTab === "execution" ? (
                  executionTaskGroups.map(([taskRun, stepMap], taskIndex) => (
                    <div
                      key={`task-${taskRun}-${taskIndex}`}
                      className="mb-3 rounded border border-slate-800 last:mb-0"
                    >
                      <button
                        type="button"
                        onClick={() => toggleExecutionTask(taskRun)}
                        className="flex w-full items-center justify-between border-b border-slate-800 bg-slate-900 px-3 py-2 text-left text-[11px] font-semibold text-slate-200 hover:bg-slate-800"
                      >
                        <span className="truncate pr-3">
                          {taskRun === "__ungrouped__" ? "Other" : taskRun}
                        </span>
                        <span className="inline-flex items-center gap-2 text-slate-400">
                          {Array.from(stepMap.values()).reduce(
                            (sum, rows) => sum + rows.length,
                            0,
                          )}{" "}
                          lines
                          {expandedExecutionTasks[taskRun] ? (
                            <ChevronUp className="h-3.5 w-3.5" />
                          ) : (
                            <ChevronDown className="h-3.5 w-3.5" />
                          )}
                        </span>
                      </button>
                      {expandedExecutionTasks[taskRun] ? (
                        <div className="px-3 py-2">
                          {Array.from(stepMap.entries()).map(([step, entries]) => (
                            <div key={`${taskRun}-${step}`} className="mb-2 last:mb-0">
                              <div className="mb-1 text-[11px] font-medium uppercase text-slate-400">
                                {step === "__other__" ? "other" : step}
                              </div>
                              {entries.map((entry, index) => (
                                <div
                                  key={`${taskRun}-${step}-${entry.timestamp || "no-ts"}-${index}`}
                                  className="mb-1 border-b border-slate-800 pb-1 last:mb-0 last:border-b-0 last:pb-0"
                                >
                                  <div className="mb-1 flex flex-wrap items-center gap-2 text-[11px] text-slate-400">
                                    <span>{formatDateTime(entry.timestamp)}</span>
                                    <span className="rounded border border-slate-700 bg-slate-800 px-1.5 py-0.5 uppercase text-slate-300">
                                      {(entry.level || "info").toUpperCase()}
                                    </span>
                                  </div>
                                  <p className="break-words whitespace-pre-wrap text-xs leading-relaxed text-slate-100">
                                    {entry.message || (
                                      <FileText className="inline h-3 w-3" />
                                    )}
                                  </p>
                                </div>
                              ))}
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  ))
                ) : (
                  visibleLogs.map((entry, index) => (
                    <div
                      key={`${entry.timestamp || "no-ts"}-${index}`}
                      className="mb-2 border-b border-slate-800 pb-2 last:mb-0 last:border-b-0 last:pb-0"
                    >
                      <div className="mb-1 flex flex-wrap items-center gap-2 text-[11px] text-slate-400">
                        <span>{formatDateTime(entry.timestamp)}</span>
                        <span className="rounded border border-slate-700 bg-slate-800 px-1.5 py-0.5 uppercase text-slate-300">
                          {(entry.level || "info").toUpperCase()}
                        </span>
                        {entry.metadata?.source ? (
                          <span className="rounded border border-slate-700 bg-slate-800 px-1.5 py-0.5 text-slate-300">
                            {String(entry.metadata.source)}
                          </span>
                        ) : null}
                      </div>
                      <p className="break-words whitespace-pre-wrap text-xs leading-relaxed text-slate-100">
                        {entry.message || <FileText className="inline h-3 w-3" />}
                      </p>
                    </div>
                  ))
                )}
              </div>
            )}
          </section>

          {request.error_message ? (
            <section className="rounded-lg border border-rose-200 bg-rose-50 p-4 dark:border-rose-700 dark:bg-rose-950/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-rose-800 dark:text-rose-200">
                Latest Error
              </p>
              <p className="mt-1 text-xs text-rose-800 dark:text-rose-200">
                {request.error_message}
              </p>
            </section>
          ) : null}

          {hasMeaningfulJSONEvidence(request.scan_summary_json) ||
          hasMeaningfulJSONEvidence(request.sbom_summary_json) ||
          hasMeaningfulJSONEvidence(request.sbom_evidence_json) ||
          hasMeaningfulJSONEvidence(request.policy_reasons_json) ||
          hasMeaningfulJSONEvidence(request.policy_snapshot_json) ? (
            <section className="rounded-lg border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">
                Evidence
              </p>
              <div className="mt-3 grid grid-cols-1 gap-3 xl:grid-cols-2">
                {hasMeaningfulJSONEvidence(request.policy_reasons_json) ? (
                  <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                    <p className="mb-2 text-xs font-medium text-slate-700 dark:text-slate-200">
                      Policy Reasons
                    </p>
                    <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-all rounded-md bg-slate-900/90 p-3 text-[11px] text-slate-100">
                      <code>
                        {parseJSONSummary(request.policy_reasons_json)}
                      </code>
                    </pre>
                  </div>
                ) : null}
                {hasMeaningfulJSONEvidence(request.policy_snapshot_json) ? (
                  <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                    <p className="mb-2 text-xs font-medium text-slate-700 dark:text-slate-200">
                      Policy Snapshot
                    </p>
                    <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-all rounded-md bg-slate-900/90 p-3 text-[11px] text-slate-100">
                      <code>
                        {parseJSONSummary(request.policy_snapshot_json)}
                      </code>
                    </pre>
                  </div>
                ) : null}
                {hasMeaningfulJSONEvidence(request.scan_summary_json) ? (
                  <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                    <p className="mb-2 text-xs font-medium text-slate-700 dark:text-slate-200">
                      Scan Summary
                    </p>
                    <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-all rounded-md bg-slate-900/90 p-3 text-[11px] text-slate-100">
                      <code>{parseJSONSummary(request.scan_summary_json)}</code>
                    </pre>
                  </div>
                ) : null}
                {hasMeaningfulJSONEvidence(request.sbom_summary_json) ? (
                  <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                    <p className="mb-2 text-xs font-medium text-slate-700 dark:text-slate-200">
                      SBOM Summary
                    </p>
                    <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-all rounded-md bg-slate-900/90 p-3 text-[11px] text-slate-100">
                      <code>{parseJSONSummary(request.sbom_summary_json)}</code>
                    </pre>
                  </div>
                ) : null}
                {hasMeaningfulJSONEvidence(request.sbom_evidence_json) ? (
                  <div className="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/50">
                    <p className="mb-2 text-xs font-medium text-slate-700 dark:text-slate-200">
                      SBOM Evidence
                    </p>
                    <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-all rounded-md bg-slate-900/90 p-3 text-[11px] text-slate-100">
                      <code>
                        {parseJSONSummary(request.sbom_evidence_json)}
                      </code>
                    </pre>
                  </div>
                ) : null}
              </div>
            </section>
          ) : null}
        </div>
      ) : null}
    </div>
  );
};

export default QuarantineRequestDetailPage;
