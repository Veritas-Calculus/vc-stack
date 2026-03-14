import { Navigate, Route, Routes } from 'react-router-dom'
import { Overview } from './Overview'
import { Zones } from './Zones'
import { Clusters } from './Clusters'
import { Hosts } from './Hosts'
import { PrimaryStorage, SecondaryStorage } from './StoragePools'
import { DBUsage, Alarms } from './SystemHealth'

export function Infrastructure() {
  return (
    <div className="space-y-4">
      <Routes>
        <Route path="" element={<Navigate to="overview" replace />} />
        <Route path="overview" element={<Overview />} />
        <Route path="zones" element={<Zones />} />
        <Route path="clusters" element={<Clusters />} />
        <Route path="hosts" element={<Hosts />} />
        <Route path="primary-storage" element={<PrimaryStorage />} />
        <Route path="secondary-storage" element={<SecondaryStorage />} />
        <Route path="db-usage" element={<DBUsage />} />
        <Route path="alarms" element={<Alarms />} />
        <Route path="*" element={<Overview />} />
      </Routes>
    </div>
  )
}
