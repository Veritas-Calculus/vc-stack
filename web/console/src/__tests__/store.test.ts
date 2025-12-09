import { describe, it, expect } from 'vitest'
import { useSettingsStore } from '@/lib/store'

describe('settings store', () => {
  it('sets api base url', () => {
    const { setApiBaseUrl } = useSettingsStore.getState()
    setApiBaseUrl('https://example.test')
    expect(useSettingsStore.getState().apiBaseUrl).toBe('https://example.test')
  })

  it('sets logo data url', () => {
    const { setLogoDataUrl } = useSettingsStore.getState()
    setLogoDataUrl('data:image/png;base64,abc')
    expect(useSettingsStore.getState().logoDataUrl).toBe('data:image/png;base64,abc')
  })
})
