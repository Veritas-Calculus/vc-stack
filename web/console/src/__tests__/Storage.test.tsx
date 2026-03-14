import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, cleanup } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'

vi.mock('@/lib/api', async (importOriginal) => {
  const mod = await importOriginal<Record<string, unknown>>()
  const mocked: Record<string, unknown> = {}
  for (const [key, val] of Object.entries(mod)) {
    mocked[key] = typeof val === 'function' ? vi.fn().mockResolvedValue({ data: {} }) : val
  }
  mocked.default = {
    get: vi.fn().mockResolvedValue({ data: {} }),
    post: vi.fn().mockResolvedValue({ data: {} }),
    put: vi.fn().mockResolvedValue({ data: {} }),
    delete: vi.fn().mockResolvedValue({ data: {} }),
    defaults: { withCredentials: false }
  }
  return mocked
})

vi.mock('@/lib/dataStore', async (importOriginal) => {
  const mod = await importOriginal<Record<string, unknown>>()
  return {
    ...mod,
    useDataStore: () => ({
      volumes: [],
      snapshots: [],
      fetchVolumes: vi.fn(),
      fetchSnapshots: vi.fn(),
      setVolumes: vi.fn()
    })
  }
})

vi.mock('@/lib/appStore', () => ({
  useAppStore: () => ({
    isAuthenticated: true,
    user: { username: 'admin', is_admin: true },
    token: 'mock-token'
  })
}))

afterEach(() => cleanup())

describe('Storage', () => {
  it('renders without crashing', async () => {
    const { Storage } = await import('@/features/storage/Storage')
    const { container } = render(
      <BrowserRouter>
        <Storage />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.querySelector('div')).not.toBeNull()
  })
})
