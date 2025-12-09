import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuthStore } from '@/lib/auth'

export function OidcCallback() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const login = useAuthStore((s) => s.login)

  useEffect(() => {
    const code = params.get('code')
    const state = params.get('state')
    const stored = sessionStorage.getItem('oidc_state')
    if (!code || !state || !stored || state !== stored) {
      navigate('/login', { replace: true })
      return
    }
    // In real flow, exchange code for token via backend. Here we stub it.
    sessionStorage.removeItem('oidc_state')
    login('oidc-token')
    navigate('/projects', { replace: true })
  }, [params, navigate, login])

  return (
    <div className="min-h-screen grid place-items-center bg-oxide-950 text-gray-300">
      <div className="p-4">Completing sign-in…</div>
    </div>
  )
}
