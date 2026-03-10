import { useTenantStore } from '@/store/tenant';
import React from 'react';

const CurrentTenantDisplay: React.FC = () => {
    const { selectedTenantId, userTenants } = useTenantStore();

    // Don't show if user has no tenants
    if (!userTenants || userTenants.length === 0) {
        return null;
    }

    const selectedTenant = userTenants.find(t => t.id === selectedTenantId);

    return (
        <div className="flex items-center gap-2 px-3 py-2 bg-red-100 dark:bg-red-900 rounded-lg border border-red-300 dark:border-red-700">
            <span className="text-sm font-medium text-red-800 dark:text-red-200">Current Tenant:</span>
            <span className="text-sm font-semibold text-red-600 dark:text-red-400">
                {selectedTenant?.name || selectedTenantId || 'Unknown'}
            </span>
        </div>
    );
};

export default CurrentTenantDisplay;