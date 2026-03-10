import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import SystemConfigurationPage from "../SystemConfigurationPage";

const apiGetMock = vi.fn();
const adminGetTektonTaskImagesMock = vi.fn();
const adminUpdateTektonTaskImagesMock = vi.fn();
const toastSuccessMock = vi.fn();
const toastErrorMock = vi.fn();
const canManageAdminMock = vi.fn();

vi.mock("@/services/api", () => ({
  api: {
    get: (...args: any[]) => apiGetMock(...args),
    post: vi.fn(),
    put: vi.fn(),
  },
}));

vi.mock("@/services/adminService", () => ({
  adminService: {
    getTektonTaskImages: (...args: any[]) => adminGetTektonTaskImagesMock(...args),
    updateTektonTaskImages: (...args: any[]) =>
      adminUpdateTektonTaskImagesMock(...args),
  },
}));

vi.mock("react-hot-toast", () => ({
  __esModule: true,
  default: {
    success: (...args: any[]) => toastSuccessMock(...args),
    error: (...args: any[]) => toastErrorMock(...args),
  },
}));

vi.mock("@/hooks/useAccess", () => ({
  useCanManageAdmin: () => canManageAdminMock(),
}));

describe("SystemConfigurationPage Tekton Task Images", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    canManageAdminMock.mockReturnValue(true);
    localStorage.setItem("system-config-active-tab", "tekton");

    apiGetMock.mockResolvedValue({
      data: {
        configs: [],
      },
    });

    adminGetTektonTaskImagesMock.mockResolvedValue({
      git_clone: "registry.local/tools/alpine-git:2.45.2",
      kaniko_executor: "registry.local/tools/kaniko:v1.23.2",
      buildkit: "registry.local/tools/buildkit:v0.13.2",
      skopeo: "registry.local/tools/skopeo:v1.15.0",
      trivy: "registry.local/security/trivy:0.57.1",
      syft: "registry.local/security/syft:v1.18.1",
      cosign: "registry.local/security/cosign:v2.4.1",
      packer: "registry.local/tools/packer:1.10.2",
      python_alpine: "registry.local/base/python:3.12-alpine",
      alpine: "registry.local/base/alpine:3.20",
      cleanup_kubectl: "registry.local/tools/kubectl:latest",
    });

    adminUpdateTektonTaskImagesMock.mockResolvedValue({});
  });

  it("loads Tekton task image values from admin settings API", async () => {
    render(<SystemConfigurationPage />);

    expect(await screen.findByText("Task Runtime Images")).toBeInTheDocument();
    expect(adminGetTektonTaskImagesMock).toHaveBeenCalledTimes(1);

    await waitFor(() => {
      expect(
        screen.getByDisplayValue("registry.local/tools/buildkit:v0.13.2"),
      ).toBeInTheDocument();
    });
  });

  it("saves edited Tekton task image values", async () => {
    render(<SystemConfigurationPage />);

    expect(await screen.findByText("Task Runtime Images")).toBeInTheDocument();

    const buildkitInput = screen.getByDisplayValue(
      "registry.local/tools/buildkit:v0.13.2",
    );
    fireEvent.change(buildkitInput, {
      target: { value: "registry.local/tools/buildkit:v0.14.0" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save Task Images" }));

    await waitFor(() => {
      expect(adminUpdateTektonTaskImagesMock).toHaveBeenCalledWith(
        expect.objectContaining({
          buildkit: "registry.local/tools/buildkit:v0.14.0",
        }),
      );
      expect(toastSuccessMock).toHaveBeenCalledWith(
        "Tekton task images saved successfully",
      );
    });
  });

  it("shows backend validation error message when save fails", async () => {
    adminUpdateTektonTaskImagesMock.mockRejectedValueOnce(
      new Error("buildkit must be a valid image reference"),
    );

    render(<SystemConfigurationPage />);
    expect(await screen.findByText("Task Runtime Images")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Save Task Images" }));

    await waitFor(() => {
      expect(toastErrorMock).toHaveBeenCalledWith(
        "buildkit must be a valid image reference",
      );
    });
  });

  it("hides save action in read-only mode for system administrator viewer", async () => {
    canManageAdminMock.mockReturnValue(false);
    render(<SystemConfigurationPage />);

    expect(await screen.findByText("Task Runtime Images")).toBeInTheDocument();
    expect(
      screen.getByText(
        /Read-only mode: configuration save and destructive system actions are hidden/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Save Task Images" }),
    ).not.toBeInTheDocument();
  });
});
