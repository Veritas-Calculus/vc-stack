import { useSettingsStore } from '@/lib/store'

export function SettingsBranding() {
  const { setLogoDataUrl } = useSettingsStore()

  const onLogoChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => setLogoDataUrl(String(reader.result))
    reader.readAsDataURL(file)
  }

  return (
    <section className="card p-4 space-y-4">
      <h2 className="text-lg font-semibold">Branding</h2>
      <div>
        <label className="label">Custom Logo</label>
        <input type="file" className="input" onChange={onLogoChange} accept="image/*" />
      </div>
    </section>
  )
}
