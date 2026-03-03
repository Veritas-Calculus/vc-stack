/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

interface Tariff {
  id: number
  name: string
  resource_type: string
  rate: number
  currency: string
  unit: string
  description: string
  effective_on: string
}

interface UsageSummaryItem {
  resource_type: string
  total_usage: number
  unit: string
}

interface QuotaSummary {
  id: number
  account_id: number
  period: string
  balance: number
  credit: number
  usage: number
  currency: string
  state: string
}

export function UsageBilling() {
  const [tab, setTab] = useState<'tariffs' | 'usage' | 'billing'>('tariffs')
  const [tariffs, setTariffs] = useState<Tariff[]>([])
  const [usageSummary, setUsageSummary] = useState<UsageSummaryItem[]>([])
  const [billingSummaries, setBillingSummaries] = useState<QuotaSummary[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [t, u, b] = await Promise.all([
        api.get<{ tariffs: Tariff[] }>('/v1/tariffs'),
        api.get<{ summary: UsageSummaryItem[] }>('/v1/usage/summary'),
        api.get<{ summaries: QuotaSummary[] }>('/v1/billing/summary')
      ])
      setTariffs(t.data.tariffs || [])
      setUsageSummary(u.data.summary || [])
      setBillingSummaries(b.data.summaries || [])
    } catch (err) {
      console.error('Failed to load usage data:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const tabs = [
    { key: 'tariffs' as const, label: 'Tariffs', count: tariffs.length },
    { key: 'usage' as const, label: 'Usage Summary', count: usageSummary.length },
    { key: 'billing' as const, label: 'Billing', count: billingSummaries.length }
  ]

  const unitLabel = (u: string) => {
    const map: Record<string, string> = {
      per_hour: '/hr',
      per_gb_month: '/GB·mo',
      per_count: '/unit',
      hours: 'hrs',
      gb: 'GB',
      count: ''
    }
    return map[u] || u
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">Usage & Billing</h1>
        <p className="text-sm text-gray-400 mt-1">
          Resource metering, tariffs, and billing management
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-oxide-800 pb-px">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${tab === t.key ? 'bg-oxide-800 text-white border-b-2 border-blue-500' : 'text-gray-400 hover:text-white hover:bg-oxide-800/50'}`}
          >
            {t.label}
            <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-oxide-700 text-gray-400">
              {t.count}
            </span>
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <>
          {/* Tariffs Tab */}
          {tab === 'tariffs' && (
            <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-oxide-800 text-gray-400 text-xs uppercase tracking-wider">
                    <th className="px-4 py-3 text-left">Resource</th>
                    <th className="px-4 py-3 text-left">Rate</th>
                    <th className="px-4 py-3 text-left">Unit</th>
                    <th className="px-4 py-3 text-left">Currency</th>
                    <th className="px-4 py-3 text-left">Effective</th>
                  </tr>
                </thead>
                <tbody>
                  {tariffs.map((t) => (
                    <tr
                      key={t.id}
                      className="border-b border-oxide-800/50 hover:bg-oxide-800/30 transition-colors"
                    >
                      <td className="px-4 py-3">
                        <div className="text-white font-medium">{t.name}</div>
                        <div className="text-xs text-gray-500 font-mono">{t.resource_type}</div>
                      </td>
                      <td className="px-4 py-3 text-emerald-400 font-mono font-semibold">
                        ${t.rate.toFixed(2)}
                      </td>
                      <td className="px-4 py-3 text-gray-400">{unitLabel(t.unit)}</td>
                      <td className="px-4 py-3 text-gray-400">{t.currency}</td>
                      <td className="px-4 py-3 text-gray-500 text-xs">
                        {new Date(t.effective_on).toLocaleDateString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {tariffs.length === 0 && (
                <div className="text-center py-8 text-gray-500">No tariffs configured</div>
              )}
            </div>
          )}

          {/* Usage Tab */}
          {tab === 'usage' &&
            (usageSummary.length === 0 ? (
              <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur p-12 text-center text-gray-500">
                <div className="text-4xl mb-3">📊</div>
                <p>No usage data yet</p>
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {usageSummary.map((u, i) => (
                  <div
                    key={i}
                    className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur p-5 hover:border-oxide-700 transition-colors"
                  >
                    <div className="text-xs text-gray-500 uppercase tracking-wider mb-1">
                      {u.resource_type.replace(/_/g, ' ')}
                    </div>
                    <div className="text-2xl font-bold text-white">{u.total_usage.toFixed(1)}</div>
                    <div className="text-xs text-gray-400">{unitLabel(u.unit)}</div>
                  </div>
                ))}
              </div>
            ))}

          {/* Billing Tab */}
          {tab === 'billing' &&
            (billingSummaries.length === 0 ? (
              <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur p-12 text-center text-gray-500">
                <div className="text-4xl mb-3">💰</div>
                <p>No billing data</p>
              </div>
            ) : (
              <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-oxide-800 text-gray-400 text-xs uppercase tracking-wider">
                      <th className="px-4 py-3 text-left">Period</th>
                      <th className="px-4 py-3 text-left">Account</th>
                      <th className="px-4 py-3 text-right">Credit</th>
                      <th className="px-4 py-3 text-right">Usage</th>
                      <th className="px-4 py-3 text-right">Balance</th>
                      <th className="px-4 py-3 text-left">State</th>
                    </tr>
                  </thead>
                  <tbody>
                    {billingSummaries.map((b) => (
                      <tr
                        key={b.id}
                        className="border-b border-oxide-800/50 hover:bg-oxide-800/30 transition-colors"
                      >
                        <td className="px-4 py-3 text-white font-medium">{b.period}</td>
                        <td className="px-4 py-3 text-gray-400">#{b.account_id}</td>
                        <td className="px-4 py-3 text-right text-emerald-400 font-mono">
                          ${b.credit.toFixed(2)}
                        </td>
                        <td className="px-4 py-3 text-right text-red-400 font-mono">
                          ${b.usage.toFixed(2)}
                        </td>
                        <td className="px-4 py-3 text-right text-white font-mono font-semibold">
                          ${b.balance.toFixed(2)}
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`px-2 py-0.5 rounded text-xs ${b.state === 'enabled' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400'}`}
                          >
                            {b.state}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ))}
        </>
      )}
    </div>
  )
}
