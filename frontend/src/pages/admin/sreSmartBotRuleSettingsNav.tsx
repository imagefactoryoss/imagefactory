import React from 'react'
import { Link } from 'react-router-dom'

type RuleSettingsNavItem = 'operator' | 'detector'

const baseLinkClass = 'rounded-xl border px-3 py-2 text-sm font-medium transition'

const inactiveLinkClass = 'border-slate-300 bg-white text-slate-700 hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300'

const activeLinkClass = 'border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-700 dark:bg-sky-950/30 dark:text-sky-200'

const SRESmartBotRuleSettingsNav: React.FC<{ active: RuleSettingsNavItem }> = ({ active }) => {
    return (
        <div className="flex flex-wrap items-center gap-2">
            <Link to="/admin/operations/sre-smart-bot/settings" className={`${baseLinkClass} ${inactiveLinkClass}`}>
                Settings Home
            </Link>
            <Link
                to="/admin/operations/sre-smart-bot/settings/operator-rules"
                className={`${baseLinkClass} ${active === 'operator' ? activeLinkClass : inactiveLinkClass}`}
                aria-current={active === 'operator' ? 'page' : undefined}
            >
                Operator Rules
            </Link>
            <Link
                to="/admin/operations/sre-smart-bot/settings/detector-rules"
                className={`${baseLinkClass} ${active === 'detector' ? activeLinkClass : inactiveLinkClass}`}
                aria-current={active === 'detector' ? 'page' : undefined}
            >
                Active Detector Rules
            </Link>
        </div>
    )
}

export default SRESmartBotRuleSettingsNav
