import { useCapabilitySurfacesStore } from '@/store/capabilitySurfaces'
import { useOperationCapabilitiesStore } from '@/store/operationCapabilities'
import { useTenantStore } from '@/store/tenant'
import type { OperationCapabilitiesConfig } from '@/types'
import React from 'react'
import { Link } from 'react-router-dom'

interface CapabilityProtectedRouteProps {
    children: React.ReactNode
    capability?: keyof OperationCapabilitiesConfig
    routeKey?: string
    title?: string
    description?: string
}

const capabilityLabelMap: Record<keyof OperationCapabilitiesConfig, string> = {
    build: 'Image Build',
    quarantine_request: 'Quarantine Request',
    quarantine_release: 'Quarantine Release (Admin)',
    ondemand_image_scanning: 'On-Demand Image Scanning',
}

const CapabilityProtectedRoute: React.FC<CapabilityProtectedRouteProps> = ({
    children,
    capability,
    routeKey,
    title,
    description,
}) => {
    const selectedTenantId = useTenantStore((state) => state.selectedTenantId)
    const loadedTenantId = useOperationCapabilitiesStore((state) => state.loadedTenantId)
    const isLoading = useOperationCapabilitiesStore((state) => state.isLoading)
    const isEnabledByCapability = useOperationCapabilitiesStore((state) =>
        capability ? Boolean(state.capabilities[capability]) : false
    )
    const surfacesLoadedTenantId = useCapabilitySurfacesStore((state) => state.loadedTenantId)
    const surfacesLoading = useCapabilitySurfacesStore((state) => state.isLoading)
    const canAccessRouteKey = useCapabilitySurfacesStore((state) => state.canAccessRouteKey)

    const isEnabled = routeKey ? canAccessRouteKey(routeKey) : isEnabledByCapability

    const isCapabilityStatePending = !!selectedTenantId && (
        isLoading ||
        loadedTenantId !== selectedTenantId ||
        surfacesLoading ||
        surfacesLoadedTenantId !== selectedTenantId
    )

    if (isCapabilityStatePending) {
        return (
            <div className="min-h-[40vh] flex items-center justify-center">
                <div className="text-center">
                    <div className="h-10 w-10 animate-spin rounded-full border-2 border-slate-300 border-t-blue-600 dark:border-slate-700 dark:border-t-blue-400 mx-auto" />
                    <p className="mt-3 text-sm text-slate-600 dark:text-slate-300">Checking tenant capabilities...</p>
                </div>
            </div>
        )
    }

    if (!isEnabled) {
        const capabilityLabel = capability ? capabilityLabelMap[capability] : 'this feature'
        return (
            <div className="min-h-[60vh] flex items-center justify-center px-4">
                <div className="w-full max-w-2xl rounded-xl border border-amber-200 bg-amber-50 p-6 shadow-sm dark:border-amber-800 dark:bg-amber-950/40">
                    <h1 className="text-2xl font-bold text-amber-900 dark:text-amber-200">{title || 'Capability Not Entitled'}</h1>
                    <p className="mt-2 text-sm text-amber-800 dark:text-amber-300">
                        {description || `This tenant is not currently entitled for ${capabilityLabel}.`}
                    </p>
                    <p className="mt-2 text-sm text-amber-800 dark:text-amber-300">
                        Contact your tenant administrator to request capability enablement.
                    </p>
                    <div className="mt-5">
                        <Link
                            to="/dashboard"
                            className="inline-flex items-center rounded-md border border-amber-300 bg-white px-4 py-2 text-sm font-medium text-amber-900 hover:bg-amber-100 dark:border-amber-700 dark:bg-amber-900/40 dark:text-amber-200 dark:hover:bg-amber-900/60"
                        >
                            Back to Dashboard
                        </Link>
                    </div>
                </div>
            </div>
        )
    }

    return <>{children}</>
}

export default CapabilityProtectedRoute
