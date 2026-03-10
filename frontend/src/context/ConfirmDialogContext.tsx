import ConfirmDialog from '@/components/common/ConfirmDialog'
import React, { createContext, useCallback, useContext, useMemo, useState } from 'react'

type ConfirmOptions = {
    title: string
    message: string
    confirmLabel?: string
    cancelLabel?: string
    destructive?: boolean
}

type ConfirmDialogContextValue = {
    confirm: (options: ConfirmOptions) => Promise<boolean>
}

const ConfirmDialogContext = createContext<ConfirmDialogContextValue | null>(null)

export const ConfirmDialogProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [active, setActive] = useState<(ConfirmOptions & { resolve: (value: boolean) => void }) | null>(null)

    const confirm = useCallback((options: ConfirmOptions) => {
        return new Promise<boolean>((resolve) => {
            setActive({ ...options, resolve })
        })
    }, [])

    const handleCancel = useCallback(() => {
        if (!active) return
        active.resolve(false)
        setActive(null)
    }, [active])

    const handleConfirm = useCallback(() => {
        if (!active) return
        active.resolve(true)
        setActive(null)
    }, [active])

    const value = useMemo<ConfirmDialogContextValue>(() => ({ confirm }), [confirm])

    return (
        <ConfirmDialogContext.Provider value={value}>
            {children}
            <ConfirmDialog
                isOpen={!!active}
                title={active?.title || 'Confirm'}
                message={active?.message || ''}
                confirmLabel={active?.confirmLabel}
                cancelLabel={active?.cancelLabel}
                destructive={!!active?.destructive}
                onCancel={handleCancel}
                onConfirm={handleConfirm}
            />
        </ConfirmDialogContext.Provider>
    )
}

export const useConfirmDialog = (): ConfirmDialogContextValue['confirm'] => {
    const context = useContext(ConfirmDialogContext)
    if (!context) {
        return async (options: ConfirmOptions) => {
            const prompt = options.message || options.title || 'Are you sure?'
            return window.confirm(prompt)
        }
    }
    return context.confirm
}
