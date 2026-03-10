import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TenantAssetDriftBadge } from "../TenantAssetDriftBadge";

describe("TenantAssetDriftBadge", () => {
  it("renders current status with success classes", () => {
    render(<TenantAssetDriftBadge status="current" />);
    const badge = screen.getByTestId("tenant-asset-drift-badge");
    expect(badge).toHaveTextContent("current");
    expect(badge.className).toContain("bg-green-100");
    expect(badge.className).toContain("dark:bg-green-900");
  });

  it("renders stale status with warning classes", () => {
    render(<TenantAssetDriftBadge status="stale" />);
    const badge = screen.getByTestId("tenant-asset-drift-badge");
    expect(badge).toHaveTextContent("stale");
    expect(badge.className).toContain("bg-amber-100");
    expect(badge.className).toContain("dark:bg-amber-900");
  });

  it("defaults invalid/empty values to unknown", () => {
    render(<TenantAssetDriftBadge status={"not-a-valid-status"} />);
    const badge = screen.getByTestId("tenant-asset-drift-badge");
    expect(badge).toHaveTextContent("unknown");
    expect(badge.className).toContain("bg-gray-100");
    expect(badge.className).toContain("dark:bg-gray-700");
  });
});

