import React from 'react'

type Participant = {
  id: string
  label: string
}

type Step =
  | { type: 'message'; from: string; to: string; text: string; async: boolean }
  | { type: 'branch'; label: string }

type Props = {
  chart: string
}

const MermaidSequencePreview: React.FC<Props> = ({ chart }) => {
  const lines = chart
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)

  const participants: Participant[] = []
  const steps: Step[] = []

  const participantRe = /^(actor|participant)\s+([A-Za-z0-9_]+)(?:\s+as\s+(.+))?$/i
  const messageRe = /^([A-Za-z0-9_]+)\s*(-+>>?)\s*([A-Za-z0-9_]+)\s*:\s*(.+)$/

  for (const line of lines) {
    if (line === 'sequenceDiagram' || line === 'autonumber') continue

    const participantMatch = line.match(participantRe)
    if (participantMatch) {
      const id = participantMatch[2]
      const label = (participantMatch[3] || id).trim()
      if (!participants.find((item) => item.id === id)) {
        participants.push({ id, label })
      }
      continue
    }

    if (/^(alt|else)\b/i.test(line)) {
      const label = line.replace(/^(alt|else)\s*/i, '').trim() || 'Branch'
      steps.push({ type: 'branch', label })
      continue
    }

    if (/^end$/i.test(line)) continue

    const messageMatch = line.match(messageRe)
    if (messageMatch) {
      const from = messageMatch[1]
      const arrow = messageMatch[2]
      const to = messageMatch[3]
      const text = messageMatch[4].trim()
      const async = arrow.includes('--')
      steps.push({ type: 'message', from, to, text, async })
    }
  }

  if (participants.length === 0 || steps.length === 0) {
    return (
      <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-950/30 dark:text-amber-200">
        Unable to render sequence preview for this diagram.
      </div>
    )
  }

  const participantLabel = (id: string) => participants.find((item) => item.id === id)?.label || id

  return (
    <div className="rounded-md border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900">
      <div className="flex flex-wrap gap-2">
        {participants.map((participant) => (
          <span
            key={participant.id}
            className="rounded-full border border-slate-300 bg-slate-50 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          >
            {participant.label}
          </span>
        ))}
      </div>
      <ol className="mt-3 space-y-2">
        {steps.map((step, index) => {
          if (step.type === 'branch') {
            return (
              <li
                key={`branch-${index}`}
                className="rounded-md border border-sky-200 bg-sky-50 px-2 py-1 text-[11px] font-medium text-sky-800 dark:border-sky-700 dark:bg-sky-950/30 dark:text-sky-200"
              >
                Branch: {step.label}
              </li>
            )
          }
          return (
            <li
              key={`step-${index}`}
              className="rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-[11px] text-slate-700 dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-200"
            >
              <span className="font-semibold">{participantLabel(step.from)}</span>
              <span className="mx-2 text-slate-500 dark:text-slate-400">{step.async ? '-->>' : '->>'}</span>
              <span className="font-semibold">{participantLabel(step.to)}</span>
              <span className="mx-2 text-slate-500 dark:text-slate-400">:</span>
              <span>{step.text}</span>
            </li>
          )
        })}
      </ol>
    </div>
  )
}

export default MermaidSequencePreview
