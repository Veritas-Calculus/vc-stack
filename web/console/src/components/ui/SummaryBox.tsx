import React from 'react'

interface SummaryBoxProps {
  label: string
  value: string | number
  icon?: React.ReactNode
}

/**
 * A unified, theme-aware summary card for metrics.
 * Uses semantic tokens (surface/content) to ensure Apple-style aesthetic
 * in both Light and Dark modes, replacing hardcoded rainbow colors.
 */
export function SummaryBox({ label, value, icon }: SummaryBoxProps) {
  return (
    <div className="rounded-xl border border-border bg-surface-secondary p-4 flex items-center justify-between shadow-sm transition-colors hover:border-border-strong">
      <div>
        <div className="text-2xl font-bold text-content-primary tracking-tight">{value}</div>
        <div className="text-xs font-medium text-content-secondary uppercase tracking-wider mt-1">
          {label}
        </div>
      </div>
      {icon && (
        <div className="w-10 h-10 rounded-full bg-surface-tertiary border border-border flex items-center justify-center text-content-secondary">
          {icon}
        </div>
      )}
    </div>
  )
}
