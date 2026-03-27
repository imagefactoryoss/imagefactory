import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ProjectNotificationTriggerMatrix from "../ProjectNotificationTriggerMatrix";
import {
  ProjectNotificationTriggerPreference,
  projectNotificationTriggerService,
} from "@/services/projectNotificationTriggerService";
import { userService } from "@/services/userService";
import toast from "react-hot-toast";

vi.mock("@/store/tenant", () => ({
  useTenantStore: (selector: (state: { selectedTenantId: string }) => string) =>
    selector({ selectedTenantId: "tenant-1" }),
}));

vi.mock("@/services/projectNotificationTriggerService", () => ({
  projectNotificationTriggerService: {
    getProjectNotificationTriggers: vi.fn(),
    updateProjectNotificationTriggers: vi.fn(),
    deleteProjectNotificationTrigger: vi.fn(),
  },
}));

vi.mock("@/services/userService", () => ({
  userService: {
    listUsers: vi.fn(),
  },
}));

vi.mock("react-hot-toast", () => ({
  default: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

const triggerIDs = [
  "BN-001",
  "BN-002",
  "BN-003",
  "BN-004",
  "BN-005",
  "BN-006",
  "BN-007",
  "BN-008",
  "BN-009",
  "BN-010",
  "BN-011",
  "BN-012",
  "BN-013",
] as const;

const buildPrefs = (
  overrides: Partial<ProjectNotificationTriggerPreference> = {},
): ProjectNotificationTriggerPreference[] =>
  triggerIDs.map((id) => ({
    trigger_id: id,
    source: "project",
    enabled: true,
    channels: ["in_app"],
    recipient_policy: "initiator",
    custom_recipient_user_ids: [],
    ...overrides,
  }));

const buildPrefsWithBN004Override = (
  override: Partial<ProjectNotificationTriggerPreference>,
): ProjectNotificationTriggerPreference[] =>
  triggerIDs.map((id) => ({
    trigger_id: id,
    source: "project",
    enabled: true,
    channels: ["in_app"],
    recipient_policy: "initiator",
    custom_recipient_user_ids: [],
    ...(id === "BN-004" ? override : {}),
  }));

describe("ProjectNotificationTriggerMatrix save validation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(userService.listUsers).mockResolvedValue({
      users: [{ id: "11111111-1111-4111-8111-111111111111", email: "u1@test.local" }],
    } as any);
    vi.mocked(projectNotificationTriggerService.updateProjectNotificationTriggers).mockResolvedValue(
      {
        project_id: "project-1",
        preferences: buildPrefs(),
      },
    );
  });

  it("blocks save when an enabled trigger has no channels", async () => {
    vi.mocked(projectNotificationTriggerService.getProjectNotificationTriggers).mockResolvedValue({
      project_id: "project-1",
      preferences: buildPrefs(),
    });

    render(<ProjectNotificationTriggerMatrix projectId="project-1" canEdit />);

    await screen.findByText("Build Notification Trigger Matrix");
    const bn004 = screen.getByText(/BN-004 - Build failed/i).closest("tr");
    expect(bn004).toBeTruthy();

    const checkboxes = bn004!.querySelectorAll('input[type="checkbox"]');
    await userEvent.click(checkboxes[1] as HTMLInputElement); // in-app off; email already off

    await userEvent.click(
      screen.getByRole("button", { name: "Save Trigger Preferences" }),
    );

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        expect.stringMatching(/Select at least one channel for BN-004/i),
      );
    });
    expect(
      projectNotificationTriggerService.updateProjectNotificationTriggers,
    ).not.toHaveBeenCalled();
  });

  it("blocks save when custom user UUID is invalid", async () => {
    vi.mocked(projectNotificationTriggerService.getProjectNotificationTriggers).mockResolvedValue({
      project_id: "project-1",
      preferences: buildPrefsWithBN004Override({
        recipient_policy: "custom_users",
        custom_recipient_user_ids: ["not-a-uuid"],
      }),
    });

    render(<ProjectNotificationTriggerMatrix projectId="project-1" canEdit />);
    await screen.findByText("Build Notification Trigger Matrix");

    await userEvent.click(
      screen.getByRole("button", { name: "Save Trigger Preferences" }),
    );

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        expect.stringMatching(/Invalid custom user UUID in BN-004/i),
      );
    });
    expect(
      projectNotificationTriggerService.updateProjectNotificationTriggers,
    ).not.toHaveBeenCalled();
  });

  it("saves successfully when preferences are valid", async () => {
    const prefs = buildPrefs();
    vi.mocked(projectNotificationTriggerService.getProjectNotificationTriggers).mockResolvedValue({
      project_id: "project-1",
      preferences: prefs,
    });
    vi.mocked(projectNotificationTriggerService.updateProjectNotificationTriggers).mockResolvedValue(
      {
        project_id: "project-1",
        preferences: prefs,
      },
    );

    render(<ProjectNotificationTriggerMatrix projectId="project-1" canEdit />);
    await screen.findByText("Build Notification Trigger Matrix");

    await userEvent.click(
      screen.getByRole("button", { name: "Save Trigger Preferences" }),
    );

    await waitFor(() => {
      expect(
        projectNotificationTriggerService.updateProjectNotificationTriggers,
      ).toHaveBeenCalledWith("project-1", expect.any(Array));
    });
    expect(toast.success).toHaveBeenCalledWith(
      expect.stringMatching(/saved/i),
    );
  });
});
