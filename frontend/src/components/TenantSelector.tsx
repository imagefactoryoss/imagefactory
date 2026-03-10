import { useTenantStore } from '@/store/tenant';
import React from 'react';

const TenantSelector: React.FC = () => {
    const { selectedTenantId, userTenants, setSelectedTenant } = useTenantStore();

    // Don't show if user only has access to one tenant or none
    if (!userTenants || userTenants.length <= 1) {
        return null;
    }

    const selectedTenant = userTenants.find(t => t.id === selectedTenantId);

    return (
        <div className="flex items-center gap-3 px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
            <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-slate-700 dark:text-slate-300">Current Tenant:</span>
                <span className="text-sm font-semibold text-blue-600 dark:text-blue-400">
                    {selectedTenant?.name || 'Unknown'}
                </span>
            </div>
            <div className="flex items-center gap-2">
                <span className="text-xs text-slate-500 dark:text-slate-400">Switch to:</span>
                <select
                    value={selectedTenantId || ''}
                    onChange={(e) => setSelectedTenant(e.target.value)}
                    className="px-2 py-1 text-xs bg-white dark:bg-slate-700 border border-slate-300 dark:border-slate-600 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                >
                    {userTenants.map((tenant) => (
                        <option key={tenant.id} value={tenant.id}>
                            {tenant.name}
                        </option>
                    ))}
                </select>
            </div>
        </div>
    );
};

export default TenantSelector;