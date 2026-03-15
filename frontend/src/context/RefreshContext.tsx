import React, { createContext, useCallback, useContext, useState } from 'react'

interface RefreshContextType {
    isRefreshing: boolean
    triggerRefresh: () => void
    registerRefreshCallback: (callback: () => Promise<void>) => void
    unregisterRefreshCallback: (callback: () => Promise<void>) => void
}

const RefreshContext = createContext<RefreshContextType | undefined>(undefined)

export const RefreshProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [isRefreshing, setIsRefreshing] = useState(false)
    const [refreshCallbacks, setRefreshCallbacks] = useState<Array<() => Promise<void>>>([])

    const registerRefreshCallback = useCallback((callback: () => Promise<void>) => {
        setRefreshCallbacks(prev => {
            if (!prev.includes(callback)) {
                return [...prev, callback]
            }
            return prev
        })
    }, [])

    const unregisterRefreshCallback = useCallback((callback: () => Promise<void>) => {
        setRefreshCallbacks(prev => prev.filter(cb => cb !== callback))
    }, [])

    const triggerRefresh = useCallback(async () => {
        if (isRefreshing) return

        setIsRefreshing(true)
        try {
            // Execute all registered refresh callbacks in parallel
            await Promise.all(refreshCallbacks.map(callback => callback().catch(() => {
                return Promise.resolve() // Don't let one failure stop others
            })))
        } finally {
            setIsRefreshing(false)
        }
    }, [isRefreshing, refreshCallbacks])

    return (
        <RefreshContext.Provider value={{ isRefreshing, triggerRefresh, registerRefreshCallback, unregisterRefreshCallback }}>
            {children}
        </RefreshContext.Provider>
    )
}

export const useRefresh = () => {
    const context = useContext(RefreshContext)
    if (!context) {
        throw new Error('useRefresh must be used within RefreshProvider')
    }
    return context
}
