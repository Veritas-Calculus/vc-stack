export function SettingsVersion() {
  const version = __APP_VERSION__ ?? '0.1.0'
  const commit = __APP_COMMIT__ ?? 'dev'
  const buildTime = __APP_BUILD_TIME__ ?? ''
  return (
    <section className="card p-4 space-y-2">
      <h2 className="text-lg font-semibold">Version</h2>
      <div className="text-sm text-content-secondary">Version: {version}</div>
      <div className="text-sm text-content-secondary">Commit: {commit}</div>
      <div className="text-sm text-content-secondary">Build Time: {buildTime}</div>
    </section>
  )
}

declare const __APP_VERSION__: string | undefined
declare const __APP_COMMIT__: string | undefined
declare const __APP_BUILD_TIME__: string | undefined
