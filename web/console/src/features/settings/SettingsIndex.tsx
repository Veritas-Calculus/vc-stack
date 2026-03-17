import { Link, Route, Routes, useLocation } from 'react-router-dom'
import { SettingsBranding } from './sections/Branding'
import { SettingsGlobal } from './sections/Global'
import { SettingsVersion } from './sections/Version'

const items = [
  { to: 'global', label: 'Global Parameters' },
  { to: 'branding', label: 'Branding' },
  { to: 'version', label: 'Version' }
]

export function SettingsIndex() {
  const { pathname } = useLocation()
  return (
    <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4">
      <aside className="card p-2 h-fit">
        <nav className="space-y-1">
          {items.map((it) => {
            const active =
              pathname.endsWith(`/settings/${it.to}`) || pathname.endsWith(`/settings/${it.to}/`)
            return (
              <Link
                key={it.to}
                to={it.to}
                className={`block rounded px-3 py-2 text-sm ${active ? 'bg-surface-tertiary text-content-primary' : 'text-content-secondary hover:bg-surface-tertiary'}`}
              >
                {it.label}
              </Link>
            )
          })}
        </nav>
      </aside>
      <section className="space-y-4">
        <Routes>
          <Route path="global" element={<SettingsGlobal />} />
          <Route path="branding" element={<SettingsBranding />} />
          <Route path="version" element={<SettingsVersion />} />
          <Route path="*" element={<SettingsGlobal />} />
        </Routes>
      </section>
    </div>
  )
}
