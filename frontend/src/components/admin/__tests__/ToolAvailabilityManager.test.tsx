import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ToolAvailabilityManager from "../ToolAvailabilityManager";
import { adminService } from "../../../services/adminService";
import { buildService } from "../../../services/buildService";

vi.mock("react-hot-toast", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("../../../services/adminService", () => ({
  adminService: {
    getTenants: vi.fn(),
  },
}));

vi.mock("../../../services/buildService", () => ({
  buildService: {
    getToolAvailability: vi.fn(),
    updateToolAvailability: vi.fn(),
    getBuildCapabilities: vi.fn(),
    updateBuildCapabilities: vi.fn(),
  },
}));

const toolConfig = {
  build_methods: {
    container: true,
    packer: true,
    paketo: true,
    kaniko: true,
    buildx: true,
    nix: true,
  },
  sbom_tools: { syft: true, grype: true, trivy: true },
  scan_tools: { trivy: true, grype: true, clair: true, snyk: true },
  registry_types: { s3: true, harbor: true, quay: true, artifactory: true },
  secret_managers: {
    vault: true,
    aws_secretsmanager: true,
    gcp_secretmanager: true,
    azure_keyvault: true,
  },
};

const capabilityConfig = {
  gpu: true,
  privileged: true,
  multi_arch: true,
  high_memory: true,
  host_networking: true,
  premium: true,
};

describe("ToolAvailabilityManager tenant/global scope behavior", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(adminService.getTenants).mockResolvedValue({
      data: [
        { id: "tenant-1", name: "Tenant One" },
        { id: "tenant-2", name: "Tenant Two" },
      ],
    } as any);
    vi.mocked(buildService.getToolAvailability).mockResolvedValue(toolConfig as any);
    vi.mocked(buildService.updateToolAvailability).mockResolvedValue(toolConfig as any);
    vi.mocked(buildService.getBuildCapabilities).mockResolvedValue(capabilityConfig as any);
    vi.mocked(buildService.updateBuildCapabilities).mockResolvedValue(capabilityConfig as any);
  });

  it("loads tenant scope by default, reloads on tenant change, and reloads global scope", async () => {
    const user = userEvent.setup();
    render(<ToolAvailabilityManager />);

    await waitFor(() => {
      expect(buildService.getToolAvailability).toHaveBeenCalledWith({
        globalDefault: false,
        tenantId: "tenant-1",
      });
    });

    await user.selectOptions(screen.getByLabelText("Tenant"), "tenant-2");
    await waitFor(() => {
      expect(buildService.getToolAvailability).toHaveBeenLastCalledWith({
        globalDefault: false,
        tenantId: "tenant-2",
      });
    });

    await user.selectOptions(screen.getByLabelText("Configuration Scope"), "global");
    await waitFor(() => {
      expect(buildService.getToolAvailability).toHaveBeenLastCalledWith({
        globalDefault: true,
        tenantId: undefined,
      });
    });
  });

  it("saves with tenant scope params and with global scope params", async () => {
    const user = userEvent.setup();
    render(<ToolAvailabilityManager />);

    await waitFor(() => {
      expect(buildService.getToolAvailability).toHaveBeenCalledWith({
        globalDefault: false,
        tenantId: "tenant-1",
      });
    });

    await user.click(screen.getByRole("switch", { name: "Container" }));
    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(buildService.updateToolAvailability).toHaveBeenCalledWith(
        expect.any(Object),
        { globalDefault: false, tenantId: "tenant-1" },
      );
      expect(buildService.updateBuildCapabilities).toHaveBeenCalledWith(
        expect.any(Object),
        { globalDefault: false, tenantId: "tenant-1" },
      );
    });

    await user.selectOptions(screen.getByLabelText("Configuration Scope"), "global");
    await waitFor(() => {
      expect(buildService.getToolAvailability).toHaveBeenLastCalledWith({
        globalDefault: true,
        tenantId: undefined,
      });
    });

    await user.click(screen.getByRole("switch", { name: "Container" }));
    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(buildService.updateToolAvailability).toHaveBeenLastCalledWith(
        expect.any(Object),
        { globalDefault: true, tenantId: undefined },
      );
      expect(buildService.updateBuildCapabilities).toHaveBeenLastCalledWith(
        expect.any(Object),
        { globalDefault: true, tenantId: undefined },
      );
    });
  });

  it("normalizes partial build_methods payload and saves explicit strict keys", async () => {
    const user = userEvent.setup();
    vi.mocked(buildService.getToolAvailability).mockResolvedValue({
      build_methods: { kaniko: true } as any,
      sbom_tools: { syft: true, grype: true, trivy: true } as any,
      scan_tools: { trivy: true, grype: true, clair: true, snyk: true } as any,
      registry_types: { s3: true, harbor: true, quay: true, artifactory: true } as any,
      secret_managers: {
        vault: true,
        aws_secretsmanager: true,
        gcp_secretmanager: true,
        azure_keyvault: true,
      } as any,
    } as any);

    render(<ToolAvailabilityManager />);

    const containerSwitch = await screen.findByRole("switch", { name: "Container" });
    expect(containerSwitch).toHaveAttribute("aria-checked", "false");

    await user.click(screen.getByRole("switch", { name: "Buildx" }));
    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(buildService.updateToolAvailability).toHaveBeenCalled();
    });

    const calls = vi.mocked(buildService.updateToolAvailability).mock.calls;
    const [payload] = calls[calls.length - 1]!;
    expect(payload.build_methods).toEqual({
      container: false,
      packer: false,
      paketo: false,
      kaniko: true,
      buildx: true,
      nix: false,
    });
  });
});
