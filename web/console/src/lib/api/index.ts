// Barrel re-export — all domain modules re-exported for backward compatibility.
// New code should import directly from the domain module, e.g.:
//   import { fetchInstances } from '@/lib/api/compute'
import api, { resolveApiBase } from './client'
export { resolveApiBase }
export default api

export * from './compute'
export * from './identity'
export * from './storage'
export * from './infrastructure'
export * from './network'
export * from './monitoring'
export * from './services'
