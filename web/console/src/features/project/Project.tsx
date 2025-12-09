import { Link, useParams } from 'react-router-dom'
import { useEffect } from 'react'
import { useDataStore } from '@/lib/dataStore'
import { useAppStore } from '@/lib/appStore'

export function Project() {
  const { projectId } = useParams()
  const setActiveProjectId = useAppStore((s) => s.setActiveProjectId)
  useEffect(() => {
    if (projectId) setActiveProjectId(projectId)
  }, [projectId, setActiveProjectId])
  const { vpcs, routes, lbs, sgRules, asns, clusters, snapshots, volumes } = useDataStore()
  const pid = projectId
  const counts = {
    vpcs: vpcs.filter((x) => x.projectId === pid).length,
    routes: routes.filter((x) => x.projectId === pid).length,
    lbs: lbs.filter((x) => x.projectId === pid).length,
    sg: sgRules.filter((x) => x.projectId === pid).length,
    asns: asns.filter((x) => x.projectId === pid).length,
    clusters: clusters.filter((x) => x.projectId === pid).length,
    snapshots: snapshots.filter((x) => x.projectId === pid).length,
    volumes: volumes.filter((x) => x.projectId === pid).length
  }
  return (
    <div className="space-y-3">
      <div className="card p-4">
        <h1 className="text-lg font-semibold">Project Overview</h1>
        <p className="text-gray-400">Project ID: {projectId}</p>
      </div>
      <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-3">
        <div className="card p-4">
          <div className="text-gray-400">VPCs</div>
          <div className="text-2xl font-semibold">{counts.vpcs}</div>
          <Link className="text-oxide-300 hover:underline" to={`/project/${projectId}/network/vpc`}>
            View
          </Link>
        </div>
        <div className="card p-4">
          <div className="text-gray-400">ASNs</div>
          <div className="text-2xl font-semibold">{counts.asns}</div>
          <Link
            className="text-oxide-300 hover:underline"
            to={`/project/${projectId}/network/asns`}
          >
            View
          </Link>
        </div>
        <div className="card p-4">
          <div className="text-gray-400">Clusters</div>
          <div className="text-2xl font-semibold">{counts.clusters}</div>
          <Link className="text-oxide-300 hover:underline" to={`/project/${projectId}/compute/k8s`}>
            View
          </Link>
        </div>
        <div className="card p-4">
          <div className="text-gray-400">Snapshots</div>
          <div className="text-2xl font-semibold">{counts.snapshots}</div>
          <Link
            className="text-oxide-300 hover:underline"
            to={`/project/${projectId}/compute/snapshots`}
          >
            View
          </Link>
        </div>
        <div className="card p-4">
          <div className="text-gray-400">Volumes</div>
          <div className="text-2xl font-semibold">{counts.volumes}</div>
          <Link className="text-oxide-300 hover:underline" to={`/project/${projectId}/storage`}>
            View
          </Link>
        </div>
      </div>
    </div>
  )
}
