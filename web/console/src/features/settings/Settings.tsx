import { useSettingsStore } from '@/lib/store'

export function Settings() {
  const { apiBaseUrl, setApiBaseUrl, setLogoDataUrl } = useSettingsStore()

  const onLogoChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => setLogoDataUrl(String(reader.result))
    reader.readAsDataURL(file)
  }

  const onSave = () => {
    // api base is already in store; persist handled by middleware
  }

  return (
    <div className="space-y-6">
      <section className="card p-4">
        <h2 className="text-lg font-semibold mb-3">Branding</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <label className="label">Custom Logo</label>
            <input type="file" className="input" onChange={onLogoChange} accept="image/*" />
          </div>
          <div>
            <label className="label">Backend API URL</label>
            <input
              className="input"
              placeholder="https://api.example.com"
              value={apiBaseUrl}
              onChange={(e) => setApiBaseUrl(e.target.value)}
            />
          </div>
        </div>
        <div className="mt-4">
          <button className="btn-primary" onClick={onSave}>
            Save
          </button>
        </div>
      </section>
    </div>
  )
}
