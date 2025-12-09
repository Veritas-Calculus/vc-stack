import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type SettingsState = {
  apiBaseUrl: string
  logoDataUrl?: string
  // IDP / SSO
  idpProvider?: 'OIDC' | 'SAML'
  idpIssuer?: string
  idpClientId?: string
  idpClientSecret?: string
  idpRedirectUrl?: string
  idpGroupClaim?: string
  setApiBaseUrl: (url: string) => void
  setLogoDataUrl: (dataUrl?: string) => void
  setIdpConfig: (cfg: {
    provider?: 'OIDC' | 'SAML'
    issuer?: string
    clientId?: string
    clientSecret?: string
    redirectUrl?: string
    groupClaim?: string
  }) => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      apiBaseUrl: import.meta.env.VITE_API_BASE_URL || '',
      logoDataUrl: undefined,
      idpProvider: undefined,
      idpIssuer: undefined,
      idpClientId: undefined,
      idpClientSecret: undefined,
      idpRedirectUrl: undefined,
      idpGroupClaim: undefined,
      setApiBaseUrl: (url) => set({ apiBaseUrl: url }),
      setLogoDataUrl: (dataUrl) => set({ logoDataUrl: dataUrl }),
      setIdpConfig: (cfg) =>
        set((s) => ({
          idpProvider: cfg.provider ?? s.idpProvider,
          idpIssuer: cfg.issuer ?? s.idpIssuer,
          idpClientId: cfg.clientId ?? s.idpClientId,
          idpClientSecret: cfg.clientSecret ?? s.idpClientSecret,
          idpRedirectUrl: cfg.redirectUrl ?? s.idpRedirectUrl,
          idpGroupClaim: cfg.groupClaim ?? s.idpGroupClaim
        }))
    }),
    { name: 'vc-console-settings' }
  )
)
