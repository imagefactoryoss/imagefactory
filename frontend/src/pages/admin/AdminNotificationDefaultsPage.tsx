import React from "react";

import TenantNotificationDefaultsPanel from "@/components/admin/TenantNotificationDefaultsPanel";

const AdminNotificationDefaultsPage: React.FC = () => {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
          Notification Defaults
        </h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
          Manage tenant-level build notification defaults from a central admin
          surface.
        </p>
      </div>
      <TenantNotificationDefaultsPanel />
    </div>
  );
};

export default AdminNotificationDefaultsPage;
