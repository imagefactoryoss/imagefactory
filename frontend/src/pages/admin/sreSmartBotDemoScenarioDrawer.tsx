import Drawer from '@/components/ui/Drawer'
import type { SREDemoScenario } from '@/types'
import React from 'react'

import { EmptyState } from './sreSmartBotIncidentPageShared'

type SRESmartBotDemoScenarioDrawerProps = {
    isOpen: boolean
    onClose: () => void
    demoLoading: boolean
    generatingDemo: boolean
    selectedDemoScenarioId: string
    setSelectedDemoScenarioId: (id: string) => void
    demoScenarios: SREDemoScenario[]
    selectedDemoScenario: SREDemoScenario | null
    onGenerateDemoIncident: () => Promise<void>
}

const SRESmartBotDemoScenarioDrawer: React.FC<SRESmartBotDemoScenarioDrawerProps> = ({
    isOpen,
    onClose,
    demoLoading,
    generatingDemo,
    selectedDemoScenarioId,
    setSelectedDemoScenarioId,
    demoScenarios,
    selectedDemoScenario,
    onGenerateDemoIncident,
}) => {
    return (
        <Drawer
            isOpen={isOpen}
            onClose={onClose}
            title="Demo Scenarios"
            description="Generate realistic SRE Smart Bot incidents on demand so you can demo grounded investigation, AI interpretation, and approval-safe action flow."
            width="60vw"
        >
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.9fr)]">
                <div className="space-y-2">
                    <label className="space-y-2">
                        <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Scenario</span>
                        <select
                            value={selectedDemoScenarioId}
                            onChange={(e) => setSelectedDemoScenarioId(e.target.value)}
                            disabled={demoLoading || generatingDemo}
                            className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 disabled:cursor-not-allowed disabled:opacity-70 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900"
                        >
                            {demoScenarios.map((scenario) => (
                                <option key={scenario.id} value={scenario.id}>
                                    {scenario.name}
                                </option>
                            ))}
                        </select>
                    </label>
                    <div className="flex flex-wrap items-center gap-3">
                        <button
                            onClick={() => void onGenerateDemoIncident()}
                            disabled={demoLoading || generatingDemo || !selectedDemoScenarioId}
                            className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-70 dark:bg-sky-500 dark:hover:bg-sky-400"
                        >
                            {generatingDemo ? 'Generating...' : 'Generate Demo Incident'}
                        </button>
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                            Best flow: generate, open the incident, run MCP tools, then show draft and local interpretation.
                        </p>
                    </div>
                </div>
                <div className="rounded-2xl border border-slate-200 bg-slate-50/90 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                    {selectedDemoScenario ? (
                        <>
                            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{selectedDemoScenario.name}</h3>
                            <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">{selectedDemoScenario.summary}</p>
                            <div className="mt-4 rounded-xl border border-slate-200 bg-white/80 p-3 dark:border-slate-800 dark:bg-slate-900/70">
                                <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">Suggested Walkthrough</p>
                                <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{selectedDemoScenario.recommended_walkthrough}</p>
                            </div>
                        </>
                    ) : (
                        <EmptyState title="No demo scenarios available" description="The backend demo generator is not exposing any scenarios yet." />
                    )}
                </div>
            </div>
        </Drawer>
    )
}

export default SRESmartBotDemoScenarioDrawer
