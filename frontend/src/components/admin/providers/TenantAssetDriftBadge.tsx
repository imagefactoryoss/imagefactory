import { TenantAssetDriftStatus } from "@/types";

type TenantAssetDriftBadgeProps = {
  status?: TenantAssetDriftStatus | string | null;
};

const tenantAssetDriftColors: Record<string, string> = {
  current: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  stale: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200",
  unknown: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200",
};

export function TenantAssetDriftBadge({ status }: TenantAssetDriftBadgeProps) {
  const normalizedStatus =
    status === "current" || status === "stale" || status === "unknown"
      ? status
      : "unknown";

  return (
    <span
      data-testid="tenant-asset-drift-badge"
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${tenantAssetDriftColors[normalizedStatus]}`}
    >
      {normalizedStatus}
    </span>
  );
}

