import { useSettingsStore } from '@/lib/store'

export function SettingsIdp() {
  const { idpProvider, idpIssuer, idpClientId, idpClientSecret, idpRedirectUrl, idpGroupClaim, setIdpConfig } = useSettingsStore()
  return (
    <section className="card p-4 space-y-4">
      <h2 className="text-lg font-semibold">IDP</h2>
      <div className="grid md:grid-cols-2 gap-3">
        <div>
          <label className="label">Provider</label>
          <select className="input w-full" value={idpProvider ?? ''} onChange={(e) => setIdpConfig({ provider: (e.target.value || undefined) as 'OIDC' | 'SAML' })}>
            <option value="">Select…</option>
            <option value="OIDC">OIDC</option>
            <option value="SAML">SAML</option>
          </select>
        </div>
        <div>
          <label className="label">Issuer</label>
          <input className="input w-full" value={idpIssuer ?? ''} onChange={(e) => setIdpConfig({ issuer: e.target.value })} />
        </div>
        <div>
          <label className="label">Client ID</label>
          <input className="input w-full" value={idpClientId ?? ''} onChange={(e) => setIdpConfig({ clientId: e.target.value })} />
        </div>
        <div>
          <label className="label">Client Secret</label>
          <input className="input w-full" value={idpClientSecret ?? ''} onChange={(e) => setIdpConfig({ clientSecret: e.target.value })} />
        </div>
        <div>
          <label className="label">Redirect URL</label>
          <input className="input w-full" value={idpRedirectUrl ?? ''} onChange={(e) => setIdpConfig({ redirectUrl: e.target.value })} />
        </div>
        <div>
          <label className="label">Group Claim</label>
          <input className="input w-full" value={idpGroupClaim ?? ''} onChange={(e) => setIdpConfig({ groupClaim: e.target.value })} />
        </div>
      </div>
    </section>
  )
}
