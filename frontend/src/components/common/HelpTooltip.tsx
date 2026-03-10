import React, { useEffect, useRef, useState } from 'react'

interface HelpTooltipProps {
    text: string
    className?: string
    autoHideMs?: number
    sticky?: boolean
    trigger?: React.ReactNode
    buttonClassName?: string
}

const HelpTooltip: React.FC<HelpTooltipProps> = ({
    text,
    className = '',
    autoHideMs = 3000,
    sticky = false,
    trigger,
    buttonClassName = '',
}) => {
    const [visible, setVisible] = useState(false)
    const [pinnedByClick, setPinnedByClick] = useState(false)
    const hideTimerRef = useRef<number | null>(null)

    const clearHideTimer = () => {
        if (hideTimerRef.current !== null) {
            window.clearTimeout(hideTimerRef.current)
            hideTimerRef.current = null
        }
    }

    const startPinnedTimer = () => {
        clearHideTimer()
        hideTimerRef.current = window.setTimeout(() => {
            setVisible(false)
            setPinnedByClick(false)
            hideTimerRef.current = null
        }, autoHideMs)
    }

    const handleClick = () => {
        if (pinnedByClick) {
            clearHideTimer()
            setPinnedByClick(false)
            setVisible(false)
            return
        }
        setVisible(true)
        setPinnedByClick(true)
        if (!sticky) {
            startPinnedTimer()
        }
    }

    const handleMouseEnter = () => setVisible(true)
    const handleMouseLeave = () => {
        if (!pinnedByClick) setVisible(false)
    }

    useEffect(() => {
        return () => clearHideTimer()
    }, [])

    return (
        <span
            className={`relative inline-flex ${className}`}
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
        >
            <button
                type="button"
                onClick={handleClick}
                onFocus={handleMouseEnter}
                onBlur={handleMouseLeave}
                className={`inline-flex h-4 w-4 items-center justify-center rounded-full border border-gray-300 text-[10px] text-gray-600 dark:border-gray-600 dark:text-gray-300 ${buttonClassName}`}
                aria-label="More information"
            >
                {trigger ?? 'i'}
            </button>
            {visible && (
                <span
                    role="tooltip"
                    className="absolute left-1/2 top-full z-20 mt-2 w-80 -translate-x-1/2 rounded-md border border-slate-300 bg-white px-3 py-2 text-xs font-normal text-slate-700 shadow-lg dark:border-slate-600 dark:bg-slate-900 dark:text-slate-200"
                >
                    <div className="flex items-start justify-between gap-2">
                        <span className="leading-5">{text}</span>
                        {pinnedByClick && (
                            <button
                                type="button"
                                onClick={() => {
                                    clearHideTimer()
                                    setPinnedByClick(false)
                                    setVisible(false)
                                }}
                                className="shrink-0 rounded border border-slate-300 px-1.5 py-0.5 text-[10px] text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-800"
                                aria-label="Close tooltip"
                            >
                                x
                            </button>
                        )}
                    </div>
                </span>
            )}
        </span>
    )
}

export default HelpTooltip
