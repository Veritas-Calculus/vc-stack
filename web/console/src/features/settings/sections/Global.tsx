import { useSettingsStore } from '@/lib/store'

export function SettingsGlobal() {
  const { apiBaseUrl, setApiBaseUrl } = useSettingsStore()
  return (
    <section className="card p-4 space-y-4">
      <h2 className="text-lg font-semibold">Global Parameters</h2>
      <div>
        <label className="label">Backend API URL</label>
        <input className="input w-full" value={apiBaseUrl} onChange={(e) => setApiBaseUrl(e.target.value)} placeholder="https://api.example.com" />
      </div>
    </section>
  )
}
