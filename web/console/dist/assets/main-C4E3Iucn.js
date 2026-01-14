const __vite__mapDeps = (
  i,
  m = __vite__mapDeps,
  d = m.f ||
    (m.f = [
      'assets/ui-vendor-CJfbT-UK.js',
      'assets/react-vendor-B-2j4S7D.js',
      'assets/addon-web-links-D-NCOiOE.js',
      'assets/xterm-C0BCwPpi.css'
    ])
) => i.map((i) => d[i])
import {
  r as k,
  a as ts,
  u as at,
  b as Ze,
  L as Oe,
  N as ut,
  c as Ve,
  d as te,
  e as xe,
  f as Js,
  g as Ys,
  h as St,
  R as Zs,
  B as Qs
} from './react-vendor-B-2j4S7D.js'
import { c as Qe, a as wt } from './utils-vendor-CVg5kGBW.js'
function er(m, y) {
  for (var g = 0; g < y.length; g++) {
    const w = y[g]
    if (typeof w != 'string' && !Array.isArray(w)) {
      for (const T in w)
        if (T !== 'default' && !(T in m)) {
          const R = Object.getOwnPropertyDescriptor(w, T)
          R && Object.defineProperty(m, T, R.get ? R : { enumerable: !0, get: () => w[T] })
        }
    }
  }
  return Object.freeze(Object.defineProperty(m, Symbol.toStringTag, { value: 'Module' }))
}
;(function () {
  const y = document.createElement('link').relList
  if (y && y.supports && y.supports('modulepreload')) return
  for (const T of document.querySelectorAll('link[rel="modulepreload"]')) w(T)
  new MutationObserver((T) => {
    for (const R of T)
      if (R.type === 'childList')
        for (const I of R.addedNodes) I.tagName === 'LINK' && I.rel === 'modulepreload' && w(I)
  }).observe(document, { childList: !0, subtree: !0 })
  function g(T) {
    const R = {}
    return (
      T.integrity && (R.integrity = T.integrity),
      T.referrerPolicy && (R.referrerPolicy = T.referrerPolicy),
      T.crossOrigin === 'use-credentials'
        ? (R.credentials = 'include')
        : T.crossOrigin === 'anonymous'
          ? (R.credentials = 'omit')
          : (R.credentials = 'same-origin'),
      R
    )
  }
  function w(T) {
    if (T.ep) return
    T.ep = !0
    const R = g(T)
    fetch(T.href, R)
  }
})()
var ss = { exports: {} },
  ot = {}
/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */ var tr = k,
  sr = Symbol.for('react.element'),
  rr = Symbol.for('react.fragment'),
  ir = Object.prototype.hasOwnProperty,
  nr = tr.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner,
  ar = { key: !0, ref: !0, __self: !0, __source: !0 }
function rs(m, y, g) {
  var w,
    T = {},
    R = null,
    I = null
  ;(g !== void 0 && (R = '' + g),
    y.key !== void 0 && (R = '' + y.key),
    y.ref !== void 0 && (I = y.ref))
  for (w in y) ir.call(y, w) && !ar.hasOwnProperty(w) && (T[w] = y[w])
  if (m && m.defaultProps) for (w in ((y = m.defaultProps), y)) T[w] === void 0 && (T[w] = y[w])
  return { $$typeof: sr, type: m, key: R, ref: I, props: T, _owner: nr.current }
}
ot.Fragment = rr
ot.jsx = rs
ot.jsxs = rs
ss.exports = ot
var e = ss.exports,
  pt = {},
  Mt = ts
;((pt.createRoot = Mt.createRoot), (pt.hydrateRoot = Mt.hydrateRoot))
function is(m, y) {
  let g
  try {
    g = m()
  } catch {
    return
  }
  return {
    getItem: (T) => {
      var R
      const I = (o) => (o === null ? null : JSON.parse(o, void 0)),
        i = (R = g.getItem(T)) != null ? R : null
      return i instanceof Promise ? i.then(I) : I(i)
    },
    setItem: (T, R) => g.setItem(T, JSON.stringify(R, void 0)),
    removeItem: (T) => g.removeItem(T)
  }
}
const mt = (m) => (y) => {
    try {
      const g = m(y)
      return g instanceof Promise
        ? g
        : {
            then(w) {
              return mt(w)(g)
            },
            catch(w) {
              return this
            }
          }
    } catch (g) {
      return {
        then(w) {
          return this
        },
        catch(w) {
          return mt(w)(g)
        }
      }
    }
  },
  or = (m, y) => (g, w, T) => {
    let R = {
        storage: is(() => localStorage),
        partialize: (c) => c,
        version: 0,
        merge: (c, t) => ({ ...t, ...c }),
        ...y
      },
      I = !1
    const i = new Set(),
      o = new Set()
    let l = R.storage
    if (!l)
      return m(
        (...c) => {
          ;(console.warn(
            `[zustand persist middleware] Unable to update item '${R.name}', the given storage is currently unavailable.`
          ),
            g(...c))
        },
        w,
        T
      )
    const u = () => {
        const c = R.partialize({ ...w() })
        return l.setItem(R.name, { state: c, version: R.version })
      },
      a = T.setState
    T.setState = (c, t) => (a(c, t), u())
    const h = m((...c) => (g(...c), u()), w, T)
    T.getInitialState = () => h
    let f
    const x = () => {
      var c, t
      if (!l) return
      ;((I = !1),
        i.forEach((s) => {
          var r
          return s((r = w()) != null ? r : h)
        }))
      const n =
        ((t = R.onRehydrateStorage) == null ? void 0 : t.call(R, (c = w()) != null ? c : h)) ||
        void 0
      return mt(l.getItem.bind(l))(R.name)
        .then((s) => {
          if (s)
            if (typeof s.version == 'number' && s.version !== R.version) {
              if (R.migrate) {
                const r = R.migrate(s.state, s.version)
                return r instanceof Promise ? r.then((d) => [!0, d]) : [!0, r]
              }
              console.error(
                "State loaded from storage couldn't be migrated since no migrate function was provided"
              )
            } else return [!1, s.state]
          return [!1, void 0]
        })
        .then((s) => {
          var r
          const [d, v] = s
          if (((f = R.merge(v, (r = w()) != null ? r : h)), g(f, !0), d)) return u()
        })
        .then(() => {
          ;(n?.(f, void 0), (f = w()), (I = !0), o.forEach((s) => s(f)))
        })
        .catch((s) => {
          n?.(void 0, s)
        })
    }
    return (
      (T.persist = {
        setOptions: (c) => {
          ;((R = { ...R, ...c }), c.storage && (l = c.storage))
        },
        clearStorage: () => {
          l?.removeItem(R.name)
        },
        getOptions: () => R,
        rehydrate: () => x(),
        hasHydrated: () => I,
        onHydrate: (c) => (
          i.add(c),
          () => {
            i.delete(c)
          }
        ),
        onFinishHydration: (c) => (
          o.add(c),
          () => {
            o.delete(c)
          }
        )
      }),
      R.skipHydration || x(),
      f || h
    )
  },
  Ct = or,
  He = Qe()(
    Ct(
      (m) => ({
        apiBaseUrl: '',
        logoDataUrl: void 0,
        idpProvider: void 0,
        idpIssuer: void 0,
        idpClientId: void 0,
        idpClientSecret: void 0,
        idpRedirectUrl: void 0,
        idpGroupClaim: void 0,
        setApiBaseUrl: (y) => m({ apiBaseUrl: y }),
        setLogoDataUrl: (y) => m({ logoDataUrl: y }),
        setIdpConfig: (y) =>
          m((g) => ({
            idpProvider: y.provider ?? g.idpProvider,
            idpIssuer: y.issuer ?? g.idpIssuer,
            idpClientId: y.clientId ?? g.idpClientId,
            idpClientSecret: y.clientSecret ?? g.idpClientSecret,
            idpRedirectUrl: y.redirectUrl ?? g.idpRedirectUrl,
            idpGroupClaim: y.groupClaim ?? g.idpGroupClaim
          }))
      }),
      { name: 'vc-console-settings' }
    )
  ),
  lt = Qe()(
    Ct((m) => ({ token: null, login: (y) => m({ token: y }), logout: () => m({ token: null }) }), {
      name: 'auth'
    })
  ),
  Te = Qe()(
    Ct(
      (m, y) => ({
        activeProjectId: null,
        setActiveProjectId: (g) => m({ activeProjectId: g }),
        sidebarCollapsed: !1,
        toggleSidebar: () => m({ sidebarCollapsed: !y().sidebarCollapsed }),
        setSidebarCollapsed: (g) => m({ sidebarCollapsed: g }),
        projectContext: !1,
        setProjectContext: (g) => m({ projectContext: g })
      }),
      {
        name: 'vc-console-app',
        storage: is(() => localStorage),
        partialize: (m) => ({
          activeProjectId: m.activeProjectId,
          sidebarCollapsed: m.sidebarCollapsed
        }),
        version: 2,
        migrate: (m, y) => {
          const g = m ?? {}
          if (y < 2) {
            if ('projectContext' in g) {
              const w = { ...g }
              return (delete w.projectContext, w)
            }
            if ('state' in g && typeof g.state == 'object' && g.state) {
              const w = { ...g.state }
              return (delete w.projectContext, { ...g, state: w })
            }
          }
          if ('projectContext' in g) {
            const w = { ...g }
            return (delete w.projectContext, w)
          }
          if ('state' in g && typeof g.state == 'object' && g.state) {
            const w = { ...g.state }
            if ('projectContext' in w) return (delete w.projectContext, { ...g, state: w })
          }
          return g
        },
        onRehydrateStorage: () => (m) => {
          m?.setProjectContext(!1)
        }
      }
    )
  )
function we() {
  return Math.random().toString(36).slice(2, 9)
}
const lr = () => new Date().toISOString(),
  Ce = Qe((m) => ({
    projects: [
      { id: '1', name: 'admin' },
      { id: '2', name: 'demo' }
    ],
    capacity: { vcpu: 200, memGiB: 1024, storageGiB: 5e3 },
    utilization: [
      {
        projectId: '1',
        points: Array.from({ length: 24 }).map((y, g) => ({
          t: Date.now() - (24 - g) * 36e5,
          vcpu: 40 + (g % 5),
          memGiB: 300 + (g % 7) * 5,
          storageGiB: 1200 + g * 10
        }))
      },
      {
        projectId: '2',
        points: Array.from({ length: 24 }).map((y, g) => ({
          t: Date.now() - (24 - g) * 36e5,
          vcpu: 15 + (g % 3),
          memGiB: 120 + (g % 6) * 3,
          storageGiB: 600 + g * 7
        }))
      }
    ],
    vpcs: [
      { id: 'vpc1', projectId: '1', name: 'prod-vpc', cidr: '10.0.0.0/16', state: 'available' },
      { id: 'vpc2', projectId: '2', name: 'dev-vpc', cidr: '10.1.0.0/16', state: 'available' }
    ],
    routes: [{ id: 'r1', projectId: '1', destination: '0.0.0.0/0', target: 'igw-123' }],
    lbs: [{ id: 'lb1', projectId: '1', name: 'web-lb', type: 'L7', status: 'active' }],
    sgRules: [
      {
        id: 'sg1',
        projectId: '1',
        direction: 'ingress',
        protocol: 'tcp',
        ports: '80,443',
        cidr: '0.0.0.0/0'
      }
    ],
    asns: [{ id: 'asn1', projectId: '1', number: 65001, description: 'Primary ASN' }],
    addVpc: (y) =>
      m((g) => ({
        vpcs: [
          ...g.vpcs,
          {
            id: we(),
            name: y.name,
            projectId: y.projectId,
            cidr: y.cidr,
            state: y.state ?? 'available'
          }
        ]
      })),
    removeVpc: (y) => m((g) => ({ vpcs: g.vpcs.filter((w) => w.id !== y) })),
    addAsn: (y) => m((g) => ({ asns: [...g.asns, { id: we(), ...y }] })),
    removeAsn: (y) => m((g) => ({ asns: g.asns.filter((w) => w.id !== y) })),
    clusters: [{ id: 'c1', projectId: '1', name: 'prod-k8s', version: '1.29', status: 'running' }],
    images: [],
    flavors: [],
    setFlavors: (y) => m(() => ({ flavors: y })),
    snapshots: [
      { id: 'snp1', projectId: '1', sourceId: 'vm-1', kind: 'vm', status: 'ready' },
      { id: 'snp2', projectId: '1', sourceId: 'vol1', kind: 'volume', status: 'ready' }
    ],
    setSnapshots: (y) => m(() => ({ snapshots: y })),
    hypervisors: [
      { id: 'hv1', name: 'hv-a', virtType: 'KVM', status: 'online' },
      { id: 'hv2', name: 'hv-b', virtType: 'QEMU', status: 'online' }
    ],
    sshKeys: [],
    addCluster: (y) =>
      m((g) => ({
        clusters: [...g.clusters, { id: we(), status: y.status ?? 'provisioning', ...y }]
      })),
    addImage: (y) => m((g) => ({ images: [...g.images, { id: we(), ...y, status: 'available' }] })),
    addFlavor: (y) => m((g) => ({ flavors: [...g.flavors, { id: we(), ...y }] })),
    addSnapshot: (y) =>
      m((g) => ({ snapshots: [...g.snapshots, { id: we(), ...y, status: 'creating' }] })),
    addSSHKey: (y) => m((g) => ({ sshKeys: [...g.sshKeys, { id: we(), ...y }] })),
    removeSSHKey: (y) => m((g) => ({ sshKeys: g.sshKeys.filter((w) => w.id !== y) })),
    instances: [
      { id: 'vm-1', projectId: '1', name: 'web-app', ip: '10.0.0.21', state: 'running' },
      { id: 'vm-2', projectId: '2', name: 'dev-vm', ip: '10.1.0.30', state: 'stopped' }
    ],
    setInstances: (y) => m(() => ({ instances: y })),
    addInstance: (y) => m((g) => ({ instances: [...g.instances, { id: we(), ...y }] })),
    publicIPs: [{ id: 'pip1', projectId: '1', address: '203.0.113.10', status: 'allocated' }],
    addPublicIP: (y) => m((g) => ({ publicIPs: [...g.publicIPs, { id: we(), ...y }] })),
    volumes: [{ id: 'vol1', projectId: '1', name: 'data-1', sizeGiB: 100, status: 'available' }],
    addVolume: (y) =>
      m((g) => ({ volumes: [...g.volumes, { id: we(), status: y.status ?? 'available', ...y }] })),
    backups: [{ id: 'b1', projectId: '1', sourceId: 'vol1', status: 'ready' }],
    addBackup: (y) =>
      m((g) => ({ backups: [...g.backups, { id: we(), ...y, status: 'creating' }] })),
    backends: [
      {
        id: 'bk1',
        name: 's3-primary',
        category: 'Primary',
        type: 'S3',
        endpoint: 's3.example.com',
        status: 'connected'
      },
      {
        id: 'bk2',
        name: 'ceph-rbd',
        category: 'Secondary',
        type: 'CephRBD',
        endpoint: 'ceph-mon01:6789',
        status: 'connected'
      }
    ],
    addBackend: (y) =>
      m((g) => ({
        backends: [...g.backends, { id: we(), status: y.status ?? 'connected', ...y }]
      })),
    removeBackend: (y) => m((g) => ({ backends: g.backends.filter((w) => w.id !== y) })),
    notices: [{ id: 'n1', time: lr(), resource: 'prod-vpc', type: 'created', status: 'unread' }],
    markNotice: (y, g) =>
      m((w) => ({ notices: w.notices.map((T) => (T.id === y ? { ...T, status: g } : T)) })),
    cmdb: [
      {
        id: 'grp-root',
        name: 'Datacenter',
        kind: 'group',
        children: [
          {
            id: 'grp-prod',
            name: 'Production',
            kind: 'group',
            children: [
              {
                id: 'h1',
                name: 'web-01',
                kind: 'host',
                address: '10.0.0.11',
                defaultUser: 'ubuntu',
                tags: ['vm']
              },
              {
                id: 'h2',
                name: 'web-02',
                kind: 'host',
                address: '10.0.0.12',
                defaultUser: 'ubuntu',
                tags: ['vm']
              },
              {
                id: 'h3',
                name: 'db-01',
                kind: 'host',
                address: '10.0.1.10',
                defaultUser: 'postgres',
                tags: ['bm']
              }
            ]
          },
          {
            id: 'grp-dev',
            name: 'Development',
            kind: 'group',
            children: [
              {
                id: 'h4',
                name: 'dev-app',
                kind: 'host',
                address: '10.1.0.20',
                defaultUser: 'dev',
                tags: ['vm']
              }
            ]
          }
        ]
      }
    ],
    isos: [],
    k8sIsos: [],
    roles: [
      { id: 'r1', name: 'Administrator', roleType: 'system' },
      { id: 'r2', name: 'Viewer', roleType: 'system' }
    ],
    addRole: (y) => m((g) => ({ roles: [...g.roles, { id: we(), ...y }] })),
    updateRole: (y, g) =>
      m((w) => ({ roles: w.roles.map((T) => (T.id === y ? { ...T, ...g } : T)) })),
    removeRole: (y) => m((g) => ({ roles: g.roles.filter((w) => w.id !== y) })),
    policies: [
      {
        id: 'p1',
        name: 'AdministratorAccess',
        type: 'system',
        document:
          '{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}]}'
      },
      {
        id: 'p2',
        name: 'ReadOnlyAccess',
        type: 'system',
        document:
          '{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["read", "list"], "Resource": "*"}]}'
      }
    ],
    addPolicy: (y) => m((g) => ({ policies: [...g.policies, { id: we(), ...y }] })),
    removePolicy: (y) => m((g) => ({ policies: g.policies.filter((w) => w.id !== y) })),
    accounts: [
      {
        id: 'u1',
        name: 'admin',
        status: 'active',
        role: 'Administrator',
        roleType: 'system',
        source: 'Local'
      }
    ],
    updateAccount: (y, g) =>
      m((w) => ({ accounts: w.accounts.map((T) => (T.id === y ? { ...T, ...g } : T)) }))
  }))
function cr() {
  return Math.random().toString(36).slice(2, 9)
}
const Je = Qe((m, y) => ({
    toasts: [],
    push: (g) => {
      const w = cr(),
        T = { id: w, variant: 'info', timeoutMs: 3500, ...g }
      m((I) => ({ toasts: [...I.toasts, T] }))
      const R = T.timeoutMs ?? 3500
      return (R > 0 && window.setTimeout(() => y().remove(w), R), w)
    },
    remove: (g) => m((w) => ({ toasts: w.toasts.filter((T) => T.id !== g) })),
    clear: () => m({ toasts: [] })
  })),
  me = {
    success(m, y) {
      return Je.getState().push({
        message: m,
        variant: 'success',
        title: y?.title,
        timeoutMs: y?.timeoutMs
      })
    },
    error(m, y) {
      return Je.getState().push({
        message: m,
        variant: 'error',
        title: y?.title,
        timeoutMs: y?.timeoutMs ?? 5e3
      })
    },
    info(m, y) {
      return Je.getState().push({
        message: m,
        variant: 'info',
        title: y?.title,
        timeoutMs: y?.timeoutMs
      })
    }
  }
function hr() {
  const m = Je((g) => g.toasts),
    y = Je((g) => g.remove)
  return m.length === 0
    ? null
    : e.jsx('div', {
        className: 'fixed bottom-4 right-4 z-50 space-y-2 w-[min(360px,calc(100vw-2rem))]',
        children: m.map((g) =>
          e.jsxs(
            'div',
            {
              className: `rounded-md border shadow-card p-3 text-sm flex items-start gap-2 ${g.variant === 'success' ? 'border-emerald-700 bg-emerald-900/40 text-emerald-100' : g.variant === 'error' ? 'border-rose-700 bg-rose-900/40 text-rose-100' : 'border-oxide-700 bg-oxide-800 text-gray-100'}`,
              children: [
                e.jsxs('div', {
                  className: 'mt-0.5',
                  children: [
                    g.variant === 'success' &&
                      e.jsx('svg', {
                        width: '16',
                        height: '16',
                        viewBox: '0 0 24 24',
                        fill: 'currentColor',
                        children: e.jsx('path', {
                          d: 'M9 16.2l-3.5-3.5L4 14.2 9 19l12-12-1.5-1.5z'
                        })
                      }),
                    g.variant === 'error' &&
                      e.jsx('svg', {
                        width: '16',
                        height: '16',
                        viewBox: '0 0 24 24',
                        fill: 'currentColor',
                        children: e.jsx('path', {
                          d: 'M12 2L1 21h22L12 2zm0 14h-1v-1h1v1zm0-3h-1V8h1v5z'
                        })
                      }),
                    g.variant === 'info' &&
                      e.jsx('svg', {
                        width: '16',
                        height: '16',
                        viewBox: '0 0 24 24',
                        fill: 'currentColor',
                        children: e.jsx('path', {
                          d: 'M11 9h2V7h-2v2zm0 8h2v-6h-2v6zm1-16C6.48 1 2 5.48 2 11s4.48 10 10 10 10-4.48 10-10S17.52 1 12 1z'
                        })
                      })
                  ]
                }),
                e.jsxs('div', {
                  className: 'flex-1',
                  children: [
                    g.title &&
                      e.jsx('div', { className: 'font-medium text-sm mb-0.5', children: g.title }),
                    e.jsx('div', { className: 'leading-snug', children: g.message })
                  ]
                }),
                e.jsx('button', {
                  className: 'opacity-70 hover:opacity-100',
                  onClick: () => y(g.id),
                  'aria-label': 'Close',
                  children: e.jsx('svg', {
                    width: '16',
                    height: '16',
                    viewBox: '0 0 24 24',
                    fill: 'currentColor',
                    children: e.jsx('path', {
                      d: 'M18.3 5.71L12 12.01l-6.3-6.3-1.41 1.41 6.3 6.3-6.3 6.3 1.41 1.41 6.3-6.3 6.3 6.3 1.41-1.41-6.3-6.3 6.3-6.3z'
                    })
                  })
                })
              ]
            },
            g.id
          )
        )
      })
}
function dr(m) {
  const y = m.match(/^\/project\/([^/]+)/)
  return y ? decodeURIComponent(y[1]) : null
}
function ur({ children: m }) {
  const y = He((D) => D.logoDataUrl),
    g = at(),
    w = Ze(),
    T = lt((D) => D.logout),
    [R, I] = k.useState({}),
    [i, o] = k.useState(!1),
    [l, u] = k.useState(!1),
    a = k.useRef(null),
    [h, f] = k.useState(null),
    x = k.useRef(null),
    c = k.useMemo(() => dr(g.pathname), [g.pathname]),
    t = Te((D) => D.activeProjectId),
    n = Te((D) => D.setActiveProjectId),
    s = Te((D) => D.projectContext),
    r = Te((D) => D.setProjectContext),
    d = Te((D) => D.sidebarCollapsed),
    v = Te((D) => D.toggleSidebar),
    _ = c ?? t,
    { projects: b, notices: p } = Ce(),
    S = k.useMemo(() => p.filter((D) => D.status === 'unread').length, [p]),
    [L, M] = k.useState(!1),
    P = k.useRef(null)
  k.useEffect(() => {
    c && (c !== t && n(c), r(!0))
  }, [c, t, n, r])
  const j = k.useMemo(() => {
    if (!_ || !s)
      return [
        { type: 'link', to: '/docs', label: 'Docs' },
        { type: 'link', to: '/projects', label: 'Project' },
        { type: 'link', to: '/images', label: 'Images' },
        { type: 'link', to: '/utilization', label: 'Utilization' }
      ]
    const D = `/project/${encodeURIComponent(_)}`
    return [
      { type: 'link', to: '/docs', label: 'Docs' },
      { type: 'link', to: `${D}/images`, label: 'Images' },
      { type: 'link', to: `${D}/utilization`, label: 'Utilization' },
      {
        type: 'group',
        label: 'Compute',
        base: `${D}/compute`,
        children: [
          { to: `${D}/compute/instances`, label: 'Instances' },
          { to: `${D}/compute/firecracker`, label: 'Firecracker' },
          { to: `${D}/compute/flavors`, label: 'Flavors' },
          { to: `${D}/compute/vm-snapshots`, label: 'VM Snapshots' },
          { to: `${D}/compute/k8s`, label: 'Kubernetes' },
          { to: `${D}/compute/kms`, label: 'SSH Keypairs' }
        ]
      },
      {
        type: 'group',
        label: 'Storage',
        base: `${D}/storage`,
        children: [
          { to: `${D}/storage/volumes`, label: 'Volumes' },
          { to: `${D}/storage/snapshots`, label: 'Snapshots' },
          { to: `${D}/storage/backups`, label: 'Backups' }
        ]
      },
      {
        type: 'group',
        label: 'Network',
        base: `${D}/network`,
        children: [
          { to: `${D}/network/vpc`, label: 'VPC' },
          { to: `${D}/network/routers`, label: 'Routers' },
          { to: `${D}/network/sg`, label: 'Security Groups' },
          { to: `${D}/network/topology`, label: 'Topology' },
          { to: `${D}/network/public-ips`, label: 'Public IPs' },
          { to: `${D}/network/asns`, label: 'ASNs' },
          { to: `${D}/network/vpn`, label: 'VPN' },
          { to: `${D}/network/acl`, label: 'Network ACL' }
        ]
      },
      {
        type: 'group',
        label: 'Images',
        base: `${D}/images`,
        children: [
          { to: `${D}/images/templates`, label: 'Templates' },
          { to: `${D}/images/iso`, label: 'ISOs' },
          { to: `${D}/images/k8s-iso`, label: 'Kubernetes ISO' }
        ]
      },
      {
        type: 'group',
        label: 'IAM',
        base: '/iam',
        children: [
          { to: '/iam/roles', label: 'Roles' },
          { to: '/iam/policies', label: 'Policies' }
        ]
      },
      { type: 'link', to: '/accounts', label: 'Accounts' },
      {
        type: 'group',
        label: 'Infrastructure',
        base: `${D}/infrastructure`,
        children: [
          { to: `${D}/infrastructure/overview`, label: 'Overview' },
          { to: `${D}/infrastructure/zones`, label: 'Zones' },
          { to: `${D}/infrastructure/clusters`, label: 'Clusters' },
          { to: `${D}/infrastructure/hosts`, label: 'Hosts' },
          { to: `${D}/infrastructure/primary-storage`, label: 'Primary Storage' },
          { to: `${D}/infrastructure/secondary-storage`, label: 'Secondary Storage' },
          { to: `${D}/infrastructure/db-usage`, label: 'DB / Usage' },
          { to: `${D}/infrastructure/alarms`, label: 'Alarms' }
        ]
      },
      { type: 'link', to: '/notifications', label: 'Notifications' }
    ]
  }, [_, s])
  return (
    k.useMemo(() => {
      const D = { ...R }
      ;(j.forEach((O) => {
        O.type === 'group' && (D[O.base] = g.pathname.startsWith(O.base))
      }),
        I(D))
    }, [g.pathname, j]),
    k.useEffect(() => {
      o(!1)
    }, [g.pathname]),
    k.useEffect(() => {
      function D(O) {
        P.current && (P.current.contains(O.target) || M(!1))
      }
      return (document.addEventListener('click', D), () => document.removeEventListener('click', D))
    }, []),
    k.useEffect(() => {
      const D = g.pathname
      ;(D === '/' || D === '/projects' || D.startsWith('/projects')) && r(!1)
    }, [g.pathname, r]),
    e.jsxs('div', {
      className: `min-h-screen grid ${d ? 'grid-cols-[64px_1fr]' : 'grid-cols-[248px_1fr]'} grid-rows-[56px_1fr]`,
      children: [
        e.jsxs('aside', {
          className: 'row-span-2 bg-oxide-900 border-r border-oxide-800',
          children: [
            e.jsxs('div', {
              className: 'h-14 flex items-center px-4 gap-2 border-b border-oxide-800',
              children: [
                y
                  ? e.jsx('img', {
                      src: y,
                      alt: 'logo',
                      className: 'h-6 w-6 rounded object-contain'
                    })
                  : e.jsx('img', {
                      src: '/logo-42.svg',
                      alt: 'logo',
                      className: 'h-6 w-6 rounded object-contain'
                    }),
                !d && e.jsx(Oe, { to: '/', className: 'font-semibold', children: 'VC Console' })
              ]
            }),
            e.jsx('nav', {
              className: `p-2 space-y-1 ${d ? 'px-1' : ''}`,
              children: j.map((D, O) => {
                if (D.type === 'link')
                  return e.jsxs(
                    ut,
                    {
                      to: D.to,
                      className: ({ isActive: F }) =>
                        `flex items-center gap-2 rounded-md ${d ? 'px-2 py-2 justify-center' : 'px-3 py-2 text-sm'} hover:bg-oxide-800 ${F ? 'bg-oxide-800 text-white' : 'text-gray-300'}`,
                      children: [
                        e.jsx(et, { name: D.label }),
                        !d && e.jsx('span', { children: D.label })
                      ]
                    },
                    O
                  )
                const $ = R[D.base]
                return e.jsxs(
                  'div',
                  {
                    className: 'relative',
                    onMouseEnter: () => {
                      d &&
                        (x.current && (window.clearTimeout(x.current), (x.current = null)),
                        f(D.base))
                    },
                    onMouseLeave: () => {
                      d &&
                        (x.current && window.clearTimeout(x.current),
                        (x.current = window.setTimeout(
                          () => f((F) => (F === D.base ? null : F)),
                          160
                        )))
                    },
                    children: [
                      e.jsxs('button', {
                        type: 'button',
                        onClick: () => {
                          d
                            ? f((F) => (F === D.base ? null : D.base))
                            : I((F) => ({ ...F, [D.base]: !F[D.base] }))
                        },
                        className: `w-full flex items-center ${d ? 'justify-center px-2 py-2' : 'justify-between px-3 py-2 text-sm'} rounded-md hover:bg-oxide-800 ${g.pathname.startsWith(D.base) ? 'bg-oxide-800 text-white' : 'text-gray-300'}`,
                        children: [
                          e.jsxs('span', {
                            className: 'flex items-center gap-2',
                            children: [
                              e.jsx(et, { name: D.label }),
                              !d && e.jsx('span', { children: D.label })
                            ]
                          }),
                          !d &&
                            e.jsx('svg', {
                              width: '14',
                              height: '14',
                              viewBox: '0 0 24 24',
                              className: `transition-transform ${$ ? 'rotate-90' : ''}`,
                              'aria-hidden': 'true',
                              fill: 'currentColor',
                              children: e.jsx('path', { d: 'M9 6l6 6-6 6' })
                            })
                        ]
                      }),
                      d &&
                        h === D.base &&
                        e.jsx('div', {
                          className:
                            'absolute left-full top-0 z-50 ml-2 min-w-44 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1',
                          onMouseEnter: () => {
                            ;(x.current && (window.clearTimeout(x.current), (x.current = null)),
                              f(D.base))
                          },
                          onMouseLeave: () => {
                            ;(x.current && window.clearTimeout(x.current),
                              (x.current = window.setTimeout(
                                () => f((F) => (F === D.base ? null : F)),
                                160
                              )))
                          },
                          children: D.children.map((F) =>
                            e.jsxs(
                              ut,
                              {
                                to: F.to,
                                className: ({ isActive: W }) =>
                                  `flex items-center gap-2 rounded-md px-3 py-1.5 text-sm hover:bg-oxide-800 ${W ? 'bg-oxide-800 text-white' : 'text-gray-200'}`,
                                onClick: () => f(null),
                                children: [
                                  e.jsx(et, { name: F.label, small: !0 }),
                                  e.jsx('span', { children: F.label })
                                ]
                              },
                              F.to
                            )
                          )
                        }),
                      !d &&
                        $ &&
                        e.jsx('div', {
                          className: 'mt-1 space-y-1',
                          children: D.children.map((F) =>
                            e.jsxs(
                              ut,
                              {
                                to: F.to,
                                className: ({ isActive: W }) =>
                                  `flex items-center gap-2 rounded-md ml-4 px-3 py-1.5 text-sm hover:bg-oxide-800 ${W ? 'bg-oxide-800 text-white' : 'text-gray-300'}`,
                                children: [
                                  e.jsx(et, { name: F.label, small: !0 }),
                                  e.jsx('span', { children: F.label })
                                ]
                              },
                              F.to
                            )
                          )
                        })
                    ]
                  },
                  O
                )
              })
            })
          ]
        }),
        e.jsxs('header', {
          className:
            'h-14 flex items-center justify-between px-4 border-b border-oxide-800 bg-oxide-900/80 backdrop-blur',
          children: [
            e.jsxs('div', {
              className: 'flex items-center gap-2',
              children: [
                e.jsx('button', {
                  type: 'button',
                  className:
                    'h-8 w-8 grid place-items-center rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200',
                  'aria-label': 'Toggle sidebar',
                  onClick: v,
                  title: d ? 'Expand sidebar' : 'Collapse sidebar',
                  children: d
                    ? e.jsxs('svg', {
                        width: '16',
                        height: '16',
                        viewBox: '0 0 24 24',
                        fill: 'none',
                        stroke: 'currentColor',
                        strokeWidth: '2',
                        strokeLinecap: 'round',
                        strokeLinejoin: 'round',
                        children: [
                          e.jsx('path', { d: 'M6 4l6 8-6 8' }),
                          e.jsx('path', { d: 'M12 4l6 8-6 8' })
                        ]
                      })
                    : e.jsxs('svg', {
                        width: '16',
                        height: '16',
                        viewBox: '0 0 24 24',
                        fill: 'none',
                        stroke: 'currentColor',
                        strokeWidth: '2',
                        strokeLinecap: 'round',
                        strokeLinejoin: 'round',
                        children: [
                          e.jsx('path', { d: 'M18 4l-6 8 6 8' }),
                          e.jsx('path', { d: 'M12 4l-6 8 6 8' })
                        ]
                      })
                }),
                _ &&
                  e.jsxs('div', {
                    className: 'relative',
                    ref: P,
                    children: [
                      e.jsxs('button', {
                        className:
                          'h-8 px-3 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 text-sm',
                        onClick: () => M((D) => !D),
                        children: ['Project: ', _]
                      }),
                      L &&
                        e.jsx('div', {
                          className:
                            'absolute z-40 mt-2 w-56 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1',
                          children: b.map((D) =>
                            e.jsx(
                              'button',
                              {
                                className:
                                  'w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800',
                                onClick: () => {
                                  ;(M(!1),
                                    n(D.id),
                                    r(!0),
                                    w(`/project/${encodeURIComponent(D.id)}`))
                                },
                                children: D.name
                              },
                              D.id
                            )
                          )
                        })
                    ]
                  })
              ]
            }),
            e.jsxs('div', {
              className: 'flex items-center gap-3',
              children: [
                e.jsxs('div', {
                  className: 'relative',
                  onMouseEnter: () => {
                    ;(a.current && (window.clearTimeout(a.current), (a.current = null)), u(!0))
                  },
                  onMouseLeave: () => {
                    ;(a.current && window.clearTimeout(a.current),
                      (a.current = window.setTimeout(() => u(!1), 180)))
                  },
                  children: [
                    e.jsx('button', {
                      className:
                        'h-8 px-3 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 text-sm',
                      onClick: () => u((D) => !D),
                      'aria-haspopup': 'menu',
                      'aria-expanded': l,
                      children: 'Create'
                    }),
                    l &&
                      e.jsxs('div', {
                        className:
                          'absolute right-0 top-full z-40 w-44 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1',
                        onMouseEnter: () => {
                          ;(a.current && (window.clearTimeout(a.current), (a.current = null)),
                            u(!0))
                        },
                        children: [
                          e.jsx('button', {
                            className:
                              'w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800',
                            onClick: () =>
                              w(s && _ ? `/project/${_}/compute/instances` : '/projects'),
                            children: 'Instance'
                          }),
                          e.jsx('button', {
                            className:
                              'w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800',
                            onClick: () => w(s && _ ? `/project/${_}/compute/k8s` : '/projects'),
                            children: 'Kubernetes'
                          }),
                          e.jsx('button', {
                            className:
                              'w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800',
                            onClick: () =>
                              w(s && _ ? `/project/${_}/storage/volumes` : '/projects'),
                            children: 'Volume'
                          }),
                          e.jsx('button', {
                            className:
                              'w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800',
                            onClick: () => w(s && _ ? `/project/${_}/network/vpc` : '/projects'),
                            children: 'VPC'
                          })
                        ]
                      })
                  ]
                }),
                e.jsx('button', {
                  className:
                    'h-8 w-8 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 grid place-items-center',
                  'aria-label': 'WebShell',
                  onClick: () => w('/webshell'),
                  title: 'WebShell',
                  children: e.jsxs('svg', {
                    width: '16',
                    height: '16',
                    viewBox: '0 0 24 24',
                    fill: 'none',
                    stroke: 'currentColor',
                    strokeWidth: '2',
                    strokeLinecap: 'round',
                    strokeLinejoin: 'round',
                    children: [
                      e.jsx('path', { d: 'M4 4h16v16H4z' }),
                      e.jsx('path', { d: 'M7 9l3 3-3 3' }),
                      e.jsx('path', { d: 'M12 16h5' })
                    ]
                  })
                }),
                e.jsxs('button', {
                  className:
                    'relative h-8 w-8 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 grid place-items-center',
                  'aria-label': 'Notifications',
                  title: 'Notifications',
                  onClick: () => w('/notifications'),
                  children: [
                    e.jsxs('svg', {
                      width: '16',
                      height: '16',
                      viewBox: '0 0 24 24',
                      fill: 'none',
                      stroke: 'currentColor',
                      strokeWidth: '2',
                      strokeLinecap: 'round',
                      strokeLinejoin: 'round',
                      children: [
                        e.jsx('path', { d: 'M18 8a6 6 0 10-12 0c0 7-3 8-3 8h18s-3-1-3-8' }),
                        e.jsx('path', { d: 'M13.73 21a2 2 0 01-3.46 0' })
                      ]
                    }),
                    S > 0 &&
                      e.jsx('span', {
                        className:
                          'absolute -top-1 -right-1 min-w-[16px] h-4 px-1 rounded-full bg-red-600 text-[10px] leading-4 text-white grid place-items-center',
                        children: S > 9 ? '9+' : S
                      })
                  ]
                }),
                e.jsxs('div', {
                  className: 'relative',
                  children: [
                    e.jsx('button', {
                      type: 'button',
                      'aria-label': 'User menu',
                      className: 'h-8 w-8 rounded-full bg-oxide-700',
                      onClick: () => o((D) => !D)
                    }),
                    i &&
                      e.jsxs('div', {
                        className:
                          'absolute right-0 z-40 mt-2 w-44 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1',
                        children: [
                          e.jsx(Oe, {
                            to: '/settings',
                            className: 'block px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800',
                            onClick: () => o(!1),
                            children: 'Settings'
                          }),
                          e.jsx('button', {
                            className:
                              'w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800',
                            onClick: () => {
                              ;(o(!1), T(), n(null), r(!1), w('/login', { replace: !0 }))
                            },
                            children: 'Sign out'
                          })
                        ]
                      })
                  ]
                })
              ]
            })
          ]
        }),
        e.jsx('main', { className: 'p-6 space-y-6 bg-oxide-950', children: m }),
        e.jsx(hr, {})
      ]
    })
  )
}
function et({ name: m, small: y }) {
  const g = y ? 14 : 16,
    w = 'currentColor'
  return (
    {
      Docs: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('path', { d: 'M4 19.5V4a2 2 0 0 1 2-2h7l5 5v12.5a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2z' }),
          e.jsx('path', { d: 'M13 2v6h6' })
        ]
      }),
      Project: e.jsx('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: e.jsx('path', { d: 'M3 7h5l2 2h11v11H3z' })
      }),
      Images: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '3', width: '18', height: '18', rx: '2' }),
          e.jsx('circle', { cx: '8.5', cy: '8.5', r: '1.5' }),
          e.jsx('path', { d: 'M21 15l-5-5L5 21' })
        ]
      }),
      Utilization: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [e.jsx('path', { d: 'M3 3v18h18' }), e.jsx('path', { d: 'M19 9l-5 5-4-4-3 3' })]
      }),
      Compute: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '3', width: '7', height: '7' }),
          e.jsx('rect', { x: '14', y: '3', width: '7', height: '7' }),
          e.jsx('rect', { x: '14', y: '14', width: '7', height: '7' }),
          e.jsx('rect', { x: '3', y: '14', width: '7', height: '7' })
        ]
      }),
      Storage: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('ellipse', { cx: '12', cy: '5', rx: '9', ry: '3' }),
          e.jsx('path', { d: 'M3 5v6c0 1.7 4 3 9 3s9-1.3 9-3V5' }),
          e.jsx('path', { d: 'M3 11v6c0 1.7 4 3 9 3s9-1.3 9-3v-6' })
        ]
      }),
      Network: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '6', cy: '12', r: '3' }),
          e.jsx('circle', { cx: '18', cy: '6', r: '3' }),
          e.jsx('circle', { cx: '18', cy: '18', r: '3' }),
          e.jsx('path', { d: 'M8.7 10.7 15.3 8.3' }),
          e.jsx('path', { d: 'M8.7 13.3 15.3 15.7' })
        ]
      }),
      Templates: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '3', width: '7', height: '7', rx: '1' }),
          e.jsx('rect', { x: '14', y: '3', width: '7', height: '7', rx: '1' }),
          e.jsx('rect', { x: '3', y: '14', width: '7', height: '7', rx: '1' }),
          e.jsx('rect', { x: '14', y: '14', width: '7', height: '7', rx: '1' })
        ]
      }),
      IAM: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '12', cy: '7', r: '4' }),
          e.jsx('path', { d: 'M5.5 21a6.5 6.5 0 0 1 13 0' })
        ]
      }),
      Accounts: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '8', cy: '8', r: '3' }),
          e.jsx('circle', { cx: '16', cy: '8', r: '3' }),
          e.jsx('path', { d: 'M2 21a6 6 0 0 1 6-6h0' }),
          e.jsx('path', { d: 'M22 21a6 6 0 0 0-6-6h0' })
        ]
      }),
      Infrastructure: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '4', width: '18', height: '8', rx: '2' }),
          e.jsx('rect', { x: '7', y: '16', width: '10', height: '4', rx: '1' })
        ]
      }),
      Notifications: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('path', { d: 'M18 8a6 6 0 10-12 0c0 7-3 8-3 8h18s-3-1-3-8' }),
          e.jsx('path', { d: 'M13.73 21a2 2 0 01-3.46 0' })
        ]
      }),
      Instances: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '3', width: '18', height: '14', rx: '2' }),
          e.jsx('path', { d: 'M8 21h8' })
        ]
      }),
      Flavors: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('path', { d: 'M4 14h16' }),
          e.jsx('path', { d: 'M4 10h16' }),
          e.jsx('path', { d: 'M4 6h16' })
        ]
      }),
      'VM Snapshots': e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '7', width: '18', height: '14', rx: '2' }),
          e.jsx('path', { d: 'M8 7l2-3h4l2 3' })
        ]
      }),
      Kubernetes: e.jsx('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: e.jsx('polygon', { points: '12 2 19 7 19 17 12 22 5 17 5 7' })
      }),
      'SSH Keypairs': e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '7.5', cy: '15.5', r: '5.5' }),
          e.jsx('path', { d: 'M14 12l7-7' }),
          e.jsx('path', { d: 'M13 7h8v8' })
        ]
      }),
      Volumes: e.jsx('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: e.jsx('rect', { x: '4', y: '4', width: '16', height: '16', rx: '2' })
      }),
      Snapshots: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '12', cy: '12', r: '3' }),
          e.jsx('path', { d: 'M5 7h3l2-2h4l2 2h3v10H5z' })
        ]
      }),
      Backups: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [e.jsx('path', { d: 'M12 20v-6' }), e.jsx('path', { d: 'M6 14l6-6 6 6' })]
      }),
      VPC: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [e.jsx('path', { d: 'M3 7h18v10H3z' }), e.jsx('path', { d: 'M7 7V3h10v4' })]
      }),
      'Security Groups': e.jsx('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: e.jsx('path', { d: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z' })
      }),
      'Public IPs': e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '12', cy: '12', r: '10' }),
          e.jsx('path', { d: 'M2 12h20' }),
          e.jsx('path', { d: 'M12 2a15.3 15.3 0 0 1 0 20' })
        ]
      }),
      ASNs: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('path', { d: 'M4 12h16' }),
          e.jsx('path', { d: 'M4 6h16' }),
          e.jsx('path', { d: 'M4 18h16' })
        ]
      }),
      VPN: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '11', width: '18', height: '10', rx: '2' }),
          e.jsx('path', { d: 'M7 11V7a5 5 0 0 1 10 0v4' })
        ]
      }),
      'Network ACL': e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '4', width: '18', height: '16', rx: '2' }),
          e.jsx('path', { d: 'M7 8h10' }),
          e.jsx('path', { d: 'M7 12h10' }),
          e.jsx('path', { d: 'M7 16h6' })
        ]
      }),
      Topology: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '12', cy: '7', r: '3' }),
          e.jsx('circle', { cx: '5', cy: '17', r: '3' }),
          e.jsx('circle', { cx: '19', cy: '17', r: '3' }),
          e.jsx('path', { d: 'M12 10v4' }),
          e.jsx('path', { d: 'M9 17h6' })
        ]
      }),
      ISO: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '12', cy: '12', r: '8' }),
          e.jsx('circle', { cx: '12', cy: '12', r: '2' })
        ]
      }),
      'K8s ISO': e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('polygon', { points: '12 2 19 7 19 17 12 22 5 17 5 7' }),
          e.jsx('circle', { cx: '12', cy: '12', r: '2' })
        ]
      }),
      Overview: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [e.jsx('path', { d: 'M3 12l9-9 9 9' }), e.jsx('path', { d: 'M9 21V9h6v12' })]
      }),
      Zones: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '3', width: '7', height: '7' }),
          e.jsx('rect', { x: '14', y: '3', width: '7', height: '7' }),
          e.jsx('rect', { x: '3', y: '14', width: '7', height: '7' }),
          e.jsx('rect', { x: '14', y: '14', width: '7', height: '7' })
        ]
      }),
      Clusters: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('circle', { cx: '8', cy: '8', r: '3' }),
          e.jsx('circle', { cx: '16', cy: '8', r: '3' }),
          e.jsx('circle', { cx: '12', cy: '16', r: '3' })
        ]
      }),
      Hosts: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('rect', { x: '3', y: '4', width: '18', height: '8', rx: '2' }),
          e.jsx('rect', { x: '7', y: '16', width: '10', height: '4', rx: '1' })
        ]
      }),
      'Primary Storage': e.jsx('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: e.jsx('ellipse', { cx: '12', cy: '5', rx: '9', ry: '3' })
      }),
      'Secondary Storage': e.jsx('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: e.jsx('ellipse', { cx: '12', cy: '5', rx: '9', ry: '3' })
      }),
      'DB / Usage': e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('ellipse', { cx: '12', cy: '5', rx: '9', ry: '3' }),
          e.jsx('path', { d: 'M3 11c0 1.7 4 3 9 3s9-1.3 9-3' })
        ]
      }),
      Alarms: e.jsxs('svg', {
        width: g,
        height: g,
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: w,
        strokeWidth: '2',
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
        children: [
          e.jsx('path', { d: 'M22 5l-5-3' }),
          e.jsx('path', { d: 'M2 5l5-3' }),
          e.jsx('circle', { cx: '12', cy: '13', r: '7' }),
          e.jsx('path', { d: 'M12 10v4l2 2' })
        ]
      })
    }[m] ??
    e.jsx('svg', {
      width: g,
      height: g,
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: w,
      strokeWidth: '2',
      strokeLinecap: 'round',
      strokeLinejoin: 'round',
      children: e.jsx('circle', { cx: '12', cy: '12', r: '2' })
    })
  )
}
function fr() {
  const { setLogoDataUrl: m } = He(),
    y = async (g) => {
      const w = g.target.files?.[0]
      if (!w) return
      const T = new FileReader()
      ;((T.onload = () => m(String(T.result))), T.readAsDataURL(w))
    }
  return e.jsxs('section', {
    className: 'card p-4 space-y-4',
    children: [
      e.jsx('h2', { className: 'text-lg font-semibold', children: 'Branding' }),
      e.jsxs('div', {
        children: [
          e.jsx('label', { className: 'label', children: 'Custom Logo' }),
          e.jsx('input', { type: 'file', className: 'input', onChange: y, accept: 'image/*' })
        ]
      })
    ]
  })
}
function Pt() {
  const { apiBaseUrl: m, setApiBaseUrl: y } = He()
  return e.jsxs('section', {
    className: 'card p-4 space-y-4',
    children: [
      e.jsx('h2', { className: 'text-lg font-semibold', children: 'Global Parameters' }),
      e.jsxs('div', {
        children: [
          e.jsx('label', { className: 'label', children: 'Backend API URL' }),
          e.jsx('input', {
            className: 'input w-full',
            value: m,
            onChange: (g) => y(g.target.value),
            placeholder: 'https://api.example.com'
          })
        ]
      })
    ]
  })
}
function pr() {
  const {
    idpProvider: m,
    idpIssuer: y,
    idpClientId: g,
    idpClientSecret: w,
    idpRedirectUrl: T,
    idpGroupClaim: R,
    setIdpConfig: I
  } = He()
  return e.jsxs('section', {
    className: 'card p-4 space-y-4',
    children: [
      e.jsx('h2', { className: 'text-lg font-semibold', children: 'IDP' }),
      e.jsxs('div', {
        className: 'grid md:grid-cols-2 gap-3',
        children: [
          e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'Provider' }),
              e.jsxs('select', {
                className: 'input w-full',
                value: m ?? '',
                onChange: (i) => I({ provider: i.target.value || void 0 }),
                children: [
                  e.jsx('option', { value: '', children: 'Select…' }),
                  e.jsx('option', { value: 'OIDC', children: 'OIDC' }),
                  e.jsx('option', { value: 'SAML', children: 'SAML' })
                ]
              })
            ]
          }),
          e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'Issuer' }),
              e.jsx('input', {
                className: 'input w-full',
                value: y ?? '',
                onChange: (i) => I({ issuer: i.target.value })
              })
            ]
          }),
          e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'Client ID' }),
              e.jsx('input', {
                className: 'input w-full',
                value: g ?? '',
                onChange: (i) => I({ clientId: i.target.value })
              })
            ]
          }),
          e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'Client Secret' }),
              e.jsx('input', {
                className: 'input w-full',
                value: w ?? '',
                onChange: (i) => I({ clientSecret: i.target.value })
              })
            ]
          }),
          e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'Redirect URL' }),
              e.jsx('input', {
                className: 'input w-full',
                value: T ?? '',
                onChange: (i) => I({ redirectUrl: i.target.value })
              })
            ]
          }),
          e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'Group Claim' }),
              e.jsx('input', {
                className: 'input w-full',
                value: R ?? '',
                onChange: (i) => I({ groupClaim: i.target.value })
              })
            ]
          })
        ]
      })
    ]
  })
}
function mr() {
  return e.jsxs('section', {
    className: 'card p-4 space-y-2',
    children: [
      e.jsx('h2', { className: 'text-lg font-semibold', children: 'Version' }),
      e.jsxs('div', { className: 'text-sm text-gray-300', children: ['Version: ', '0.1.0'] }),
      e.jsxs('div', { className: 'text-sm text-gray-300', children: ['Commit: ', ''] }),
      e.jsxs('div', {
        className: 'text-sm text-gray-300',
        children: ['Build Time: ', '2025-12-10T02:12:13.619Z']
      })
    ]
  })
}
const _r = [
  { to: 'global', label: 'Global Parameters' },
  { to: 'idp', label: 'IDP' },
  { to: 'branding', label: 'Branding' },
  { to: 'version', label: 'Version' }
]
function gr() {
  const { pathname: m } = at()
  return e.jsxs('div', {
    className: 'grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4',
    children: [
      e.jsx('aside', {
        className: 'card p-2 h-fit',
        children: e.jsx('nav', {
          className: 'space-y-1',
          children: _r.map((y) => {
            const g = m.endsWith(`/settings/${y.to}`) || m.endsWith(`/settings/${y.to}/`)
            return e.jsx(
              Oe,
              {
                to: y.to,
                className: `block rounded px-3 py-2 text-sm ${g ? 'bg-oxide-800 text-white' : 'text-gray-300 hover:bg-oxide-800'}`,
                children: y.label
              },
              y.to
            )
          })
        })
      }),
      e.jsx('section', {
        className: 'space-y-4',
        children: e.jsxs(Ve, {
          children: [
            e.jsx(te, { path: 'global', element: e.jsx(Pt, {}) }),
            e.jsx(te, { path: 'idp', element: e.jsx(pr, {}) }),
            e.jsx(te, { path: 'branding', element: e.jsx(fr, {}) }),
            e.jsx(te, { path: 'version', element: e.jsx(mr, {}) }),
            e.jsx(te, { path: '*', element: e.jsx(Pt, {}) })
          ]
        })
      })
    ]
  })
}
function oe({ title: m, subtitle: y, actions: g }) {
  return e.jsxs('div', {
    className: 'flex items-start justify-between gap-4',
    children: [
      e.jsxs('div', {
        children: [
          e.jsx('h1', {
            className: 'text-xl font-semibold tracking-tight text-white',
            children: m
          }),
          y && e.jsx('p', { className: 'mt-1 text-sm text-gray-400', children: y })
        ]
      }),
      g && e.jsx('div', { className: 'flex items-center gap-2', children: g })
    ]
  })
}
function de({ columns: m, data: y, empty: g = 'No data', onRowClick: w, isRowSelected: T }) {
  const [R, I] = k.useState(null),
    [i, o] = k.useState('asc'),
    l = k.useMemo(() => {
      if (!R) return y
      const a = [...y]
      return (
        a.sort((h, f) => {
          const x = h[R],
            c = f[R],
            t = String(x ?? ''),
            n = String(c ?? '')
          return i === 'asc' ? t.localeCompare(n) : n.localeCompare(t)
        }),
        a
      )
    }, [y, i, R]),
    u = (a) => {
      R === a ? o((h) => (h === 'asc' ? 'desc' : 'asc')) : (I(a), o('asc'))
    }
  return e.jsx('div', {
    className: 'overflow-hidden rounded-lg border border-oxide-800',
    children: e.jsxs('table', {
      className: 'w-full text-left text-sm',
      children: [
        e.jsx('thead', {
          className: 'bg-oxide-900/70',
          children: e.jsx('tr', {
            children: m.map((a) =>
              e.jsx(
                'th',
                {
                  className: `px-3 py-2 font-medium text-gray-300 select-none ${a.className ?? ''}`,
                  onClick: () => (a.sortable === !1 ? void 0 : u(String(a.key))),
                  children: e.jsxs('span', {
                    className: 'inline-flex items-center gap-1 cursor-pointer',
                    children: [
                      a.headerRender ?? a.header,
                      a.sortable === !1
                        ? null
                        : R === a.key &&
                          e.jsx('span', {
                            className: 'text-xs text-gray-500',
                            children: i === 'asc' ? '▲' : '▼'
                          })
                    ]
                  })
                },
                String(a.key)
              )
            )
          })
        }),
        e.jsx('tbody', {
          children:
            l.length === 0
              ? e.jsx('tr', {
                  children: e.jsx('td', {
                    className: 'px-3 py-6 text-center text-gray-500',
                    colSpan: m.length,
                    children: g
                  })
                })
              : l.map((a, h) => {
                  const f = T ? T(a) : !1
                  return e.jsx(
                    'tr',
                    {
                      className: `border-t border-oxide-800 hover:bg-oxide-900/40 ${f ? 'bg-oxide-900/70' : ''}`,
                      onClick: w ? () => w(a) : void 0,
                      children: m.map((x) =>
                        e.jsx(
                          'td',
                          {
                            className: `px-3 py-2 text-gray-200 ${x.className ?? ''}`,
                            children: x.render ? x.render(a) : String(a[x.key])
                          },
                          String(x.key)
                        )
                      )
                    },
                    h
                  )
                })
        })
      ]
    })
  })
}
function Ne({ placeholder: m = 'Search…', onSearch: y, children: g }) {
  return e.jsxs('div', {
    className:
      'flex items-center justify-between gap-3 flex-nowrap overflow-x-auto whitespace-nowrap',
    children: [
      e.jsx('div', {
        className: 'flex items-center gap-2 whitespace-nowrap',
        children:
          y &&
          e.jsx('input', {
            className: 'input w-72',
            placeholder: m,
            onChange: (w) => y?.(w.target.value)
          })
      }),
      e.jsx('div', { className: 'flex items-center gap-2 whitespace-nowrap', children: g })
    ]
  })
}
function Ge({ actions: m }) {
  const [y, g] = k.useState(!1),
    w = k.useRef(null),
    [T, R] = k.useState(null)
  return (
    k.useEffect(() => {
      const I = (i) => {
        w.current && (w.current.contains(i.target) || g(!1))
      }
      return (document.addEventListener('click', I), () => document.removeEventListener('click', I))
    }, []),
    e.jsxs('div', {
      className: 'relative',
      ref: w,
      children: [
        e.jsx('button', {
          type: 'button',
          className:
            'px-2 h-7 inline-flex items-center rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200',
          'aria-label': 'Actions',
          onClick: (I) => {
            const i = I.currentTarget.getBoundingClientRect()
            ;(R({ x: i.right, y: i.bottom }), g((o) => !o))
          },
          children: e.jsxs('svg', {
            width: '14',
            height: '14',
            viewBox: '0 0 24 24',
            fill: 'currentColor',
            'aria-hidden': 'true',
            children: [
              e.jsx('circle', { cx: '5', cy: '12', r: '2' }),
              e.jsx('circle', { cx: '12', cy: '12', r: '2' }),
              e.jsx('circle', { cx: '19', cy: '12', r: '2' })
            ]
          })
        }),
        y &&
          T &&
          ts.createPortal(
            e.jsx('div', {
              style: { position: 'fixed', top: T.y + 4, left: T.x - 144 },
              className:
                'z-50 w-36 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1',
              children: m.map((I, i) =>
                e.jsx(
                  'button',
                  {
                    type: 'button',
                    onClick: () => {
                      ;(g(!1), I.onClick())
                    },
                    className: `w-full text-left px-3 py-1.5 text-sm hover:bg-oxide-800 ${I.danger ? 'text-rose-300' : 'text-gray-200'}`,
                    children: I.label
                  },
                  i
                )
              )
            }),
            document.body
          )
      ]
    })
  )
}
function le({ title: m, open: y, onClose: g, children: w, footer: T }) {
  return y
    ? e.jsxs('div', {
        className: 'fixed inset-0 z-50 grid place-items-center',
        children: [
          e.jsx('div', { className: 'absolute inset-0 bg-black/50', onClick: g }),
          e.jsxs('div', {
            className:
              'relative w-full max-w-2xl rounded-lg border border-oxide-800 bg-oxide-900 shadow-card',
            children: [
              e.jsxs('div', {
                className: 'px-4 py-3 border-b border-oxide-800 flex items-center justify-between',
                children: [
                  e.jsx('h3', { className: 'font-semibold', children: m }),
                  e.jsx('button', {
                    className: 'text-gray-400 hover:text-gray-200',
                    onClick: g,
                    'aria-label': 'Close',
                    children: '×'
                  })
                ]
              }),
              e.jsx('div', { className: 'p-4 space-y-3', children: w }),
              T &&
                e.jsx('div', {
                  className: 'px-4 py-3 border-t border-oxide-800 flex justify-end gap-2',
                  children: T
                })
            ]
          })
        ]
      })
    : null
}
function qe() {
  return (typeof window < 'u' ? window.__VC_CONFIG__?.apiBase : void 0) || void 0 || '/api'
}
const ee = wt.create({ baseURL: qe(), withCredentials: !1 })
ee.interceptors.request.use((m) => {
  const y = localStorage.getItem('auth')
  let g = null
  if (y)
    try {
      ;((g = JSON.parse(y)?.state?.token || null),
        console.log('[API] Token from localStorage:', g ? 'Found' : 'Not found'))
    } catch {
      console.log('[API] Failed to parse auth data')
    }
  else console.log('[API] No auth data in localStorage')
  return (g && ((m.headers = m.headers ?? {}), (m.headers.Authorization = `Bearer ${g}`)), m)
})
ee.interceptors.response.use(
  (m) => m,
  (m) => {
    if (m.response?.status === 401) {
      const g = `[API] 401 Unauthorized from: ${m.config?.url || 'unknown'}`
      console.error(g)
      try {
        const w = JSON.parse(localStorage.getItem('debug_logs') || '[]')
        ;(w.push({ time: new Date().toISOString(), msg: g }),
          w.length > 50 && w.shift(),
          localStorage.setItem('debug_logs', JSON.stringify(w)))
      } catch {}
      ;(localStorage.removeItem('auth'),
        typeof window < 'u' &&
          !window.location.pathname.startsWith('/login') &&
          (console.error('[API] Redirecting to /login due to 401'),
          (window.location.href = '/login')))
    }
    return Promise.reject(m)
  }
)
function ye(m) {
  return m ? { headers: { 'X-Project-ID': m } } : void 0
}
const vr = (m) => ({
    id: String(m.id),
    name: m.name,
    vcpu: m.vcpus,
    memoryGiB: Math.round((m.ram || 0) / 1024)
  }),
  xr = (m) => ({
    id: String(m.id),
    projectId: String(m.project_id ?? ''),
    sourceId: String(m.volume_id ?? ''),
    kind: 'vm',
    status: m.status === 'available' ? 'ready' : 'creating'
  })
async function ns(m, y) {
  return (await ee.post('/v1/auth/login', { username: m, password: y })).data
}
async function Ye(m) {
  return (await ee.get('/v1/instances', ye(m))).data.instances ?? []
}
async function kt() {
  return ((await ee.get('/v1/flavors')).data.flavors ?? []).map(vr)
}
async function as(m) {
  const g = (await ee.post('/v1/flavors', m)).data.flavor
  return {
    id: String(g.id),
    name: g.name,
    vcpu: g.vcpus,
    memoryGiB: Math.round((g.ram || 0) / 1024)
  }
}
async function os(m) {
  await ee.delete(`/v1/flavors/${m}`)
}
async function ge(m) {
  const y = await ee.get('/v1/images', ye(m)),
    g = (R) => Math.max(1, Math.ceil((R ?? 0) / 1024 ** 3)),
    w = (R) =>
      R === 'available' || R === 'uploading' || R === 'queued' || R === 'active' ? R : 'available',
    T = (R) => R
  return (y.data.images ?? []).map((R) => ({
    id: String(R.id),
    name: R.name,
    sizeGiB: g(R.size),
    minDiskGiB: R.min_disk ? Math.max(1, R.min_disk) : void 0,
    status: w(R.status),
    disk_format: T(R.disk_format),
    owner: R.owner_id ? String(R.owner_id) : void 0
  }))
}
async function ls(m) {
  const y = await ee.post('/v1/images/register', m)
  return { id: String(y.data.image.id) }
}
async function cs(m, y) {
  await ee.post(`/v1/images/${m}/import`, {})
}
async function ct(m, y) {
  const g = new FormData()
  ;(g.append('file', m), y?.name && g.append('name', y.name))
  const w = await ee.post('/v1/images/upload', g, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return { id: String(w.data.image.id) }
}
async function ht(m) {
  await ee.delete(`/v1/images/${m}`)
}
async function hs(m, y) {
  return (await ee.post('/v1/instances', y, ye(m))).data.instance
}
async function ds() {
  return ((await ee.get('/v1/snapshots')).data.snapshots ?? []).map(xr)
}
async function _t(m) {
  return ((await ee.get('/v1/volumes', ye(m))).data.volumes ?? []).map((g) => ({
    id: String(g.id),
    name: g.name,
    sizeGiB: g.size_gb,
    status: g.status ?? 'available',
    projectId: g.project_id ? String(g.project_id) : void 0,
    rbd: [g.rbd_pool, g.rbd_image].filter(Boolean).join('/') || void 0
  }))
}
async function us(m, y) {
  const w = (await ee.post('/v1/volumes', y, ye(m))).data.volume
  return {
    id: String(w.id),
    name: w.name,
    sizeGiB: w.size_gb,
    status: w.status ?? 'available',
    projectId: w.project_id ? String(w.project_id) : m,
    rbd: [w.rbd_pool, w.rbd_image].filter(Boolean).join('/') || void 0
  }
}
async function fs(m) {
  await ee.delete(`/v1/volumes/${m}`)
}
async function ps(m, y) {
  const w = (await ee.post(`/v1/volumes/${m}/resize`, { new_size_gb: y })).data.volume
  return {
    id: String(w.id),
    name: w.name,
    sizeGiB: w.size_gb,
    status: w.status ?? 'available',
    projectId: w.project_id ? String(w.project_id) : void 0,
    rbd: [w.rbd_pool, w.rbd_image].filter(Boolean).join('/') || void 0
  }
}
async function gt(m) {
  return ((await ee.get(`/v1/instances/${m}/volumes`)).data.volumes ?? []).map((g) => ({
    id: String(g.id),
    name: g.name,
    sizeGiB: g.size_gb,
    status: g.status ?? 'in-use',
    projectId: g.project_id ? String(g.project_id) : void 0,
    rbd: [g.rbd_pool, g.rbd_image].filter(Boolean).join('/') || void 0
  }))
}
async function ms(m, y, g) {
  await ee.post(`/v1/instances/${m}/volumes`, { volume_id: Number(y), device: g })
}
async function _s(m, y) {
  await ee.delete(`/v1/instances/${m}/volumes/${y}`)
}
async function tt(m) {
  return ((await ee.get('/v1/snapshots', ye(m))).data.snapshots ?? []).map((g) => ({
    id: String(g.id),
    name: g.name,
    volumeId: String(g.volume_id),
    status: g.status ?? 'available',
    backup: g.backup_pool && g.backup_image ? `${g.backup_pool}/${g.backup_image}` : void 0
  }))
}
async function gs(m, y) {
  const w = (await ee.post('/v1/snapshots', y, ye(m))).data.snapshot
  return {
    id: String(w.id),
    name: w.name,
    volumeId: String(w.volume_id),
    status: w.status ?? 'available',
    backup: w.backup_pool && w.backup_image ? `${w.backup_pool}/${w.backup_image}` : void 0
  }
}
async function vt(m, y) {
  return (await ee.get('/v1/audit', { params: y, ...(ye(m) ?? {}) })).data.audit ?? []
}
async function vs() {
  return (await ee.get('/v1/nodes')).data.nodes ?? []
}
async function xs(m) {
  await ee.delete(`/v1/nodes/${m}`)
}
async function jt(m) {
  const g = (await ee.post(`/v1/instances/${m}/console`)).data.ws,
    w = qe()
  let T
  if (w.startsWith('http://') || w.startsWith('https://')) {
    const R = new URL(w)
    T = `${R.protocol === 'https:' ? 'wss:' : 'ws:'}//${R.host}${g}`
  } else
    T = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}${g}`
  return T
}
async function bs(m) {
  return (await ee.post(`/v1/instances/${m}/start`)).data.instance
}
async function ys(m) {
  return (await ee.post(`/v1/instances/${m}/stop`)).data.instance
}
async function Ss(m) {
  return (await ee.post(`/v1/instances/${m}/reboot`)).data.instance
}
async function ws(m) {
  await ee.delete(`/v1/instances/${m}`)
}
async function Cs(m) {
  await ee.post(`/v1/instances/${m}/force-delete`)
}
async function ks(m) {
  return (await ee.get(`/v1/instances/${m}/deletion-status`)).data
}
async function We(m) {
  return (
    (await ee.get('/v1/networks', { params: m ? { tenant_id: m } : void 0, ...(ye(m) ?? {}) })).data
      .networks ?? []
  ).map((g) => ({
    id: String(g.id),
    name: g.name,
    cidr: g.cidr,
    description: g.description,
    zone: g.zone,
    tenant_id: g.tenant_id,
    status: g.status,
    network_type: g.network_type,
    physical_network: g.physical_network,
    segmentation_id: g.segmentation_id,
    shared: g.shared,
    external: g.external,
    mtu: g.mtu
  }))
}
async function Nt(m, y) {
  const w = (
    await ee.post(
      '/v1/networks',
      {
        name: y.name,
        cidr: y.cidr,
        description: y.description ?? '',
        zone: y.zone,
        dns1: y.dns1,
        dns2: y.dns2,
        start: y.start ?? !0,
        tenant_id: m,
        enable_dhcp: y.enable_dhcp,
        dhcp_lease_time: y.dhcp_lease_time,
        gateway: y.gateway,
        allocation_start: y.allocation_start,
        allocation_end: y.allocation_end,
        network_type: y.network_type,
        physical_network: y.physical_network,
        segmentation_id: y.segmentation_id,
        shared: y.shared,
        external: y.external,
        mtu: y.mtu
      },
      ye(m)
    )
  ).data.network
  return {
    id: String(w.id),
    name: w.name,
    cidr: w.cidr,
    description: w.description,
    zone: w.zone,
    tenant_id: w.tenant_id,
    status: w.status,
    network_type: w.network_type,
    physical_network: w.physical_network,
    segmentation_id: w.segmentation_id,
    shared: w.shared,
    external: w.external,
    mtu: w.mtu
  }
}
async function st(m) {
  const y = await ee.get('/v1/subnets', { params: m ? { tenant_id: m } : void 0 })
  return (Array.isArray(y.data) ? y.data : (y.data.subnets ?? [])).map((w) => ({
    id: String(w.id),
    name: w.name,
    network_id: w.network_id,
    cidr: w.cidr,
    gateway: w.gateway,
    allocation_start: w.allocation_start,
    allocation_end: w.allocation_end,
    dns_nameservers: w.dns_nameservers,
    enable_dhcp: w.enable_dhcp,
    dhcp_lease_time: w.dhcp_lease_time,
    tenant_id: w.tenant_id,
    status: w.status
  }))
}
async function js(m) {
  await ee.post(`/v1/networks/${m}/restart`, {})
}
async function xt(m) {
  await ee.delete(`/v1/networks/${m}`)
}
async function Et(m) {
  return ((await ee.get('/v1/ports', { params: m })).data.ports ?? []).map((g) => ({
    id: String(g.id),
    name: g.name,
    network_id: g.network_id,
    subnet_id: g.subnet_id,
    mac_address: g.mac_address,
    fixed_ips: g.fixed_ips,
    security_groups: g.security_groups,
    device_id: g.device_id,
    device_owner: g.device_owner,
    status: g.status
  }))
}
async function rt(m) {
  return ((await ee.get('/v1/ssh-keys', ye(m))).data.ssh_keys ?? []).map((g) => ({
    id: String(g.id),
    name: g.name,
    public_key: g.public_key,
    project_id: g.project_id,
    user_id: g.user_id
  }))
}
async function Lt(m, y) {
  const w = (await ee.post('/v1/ssh-keys', y, ye(m))).data.ssh_key
  return { id: String(w.id), name: w.name, public_key: w.public_key, project_id: w.project_id }
}
async function Ns(m, y) {
  await ee.delete(`/v1/ssh-keys/${y}`, ye(m))
}
async function Es(m) {
  return (
    (await ee.get('/v1/floating-ips', { params: m ? { tenant_id: m } : void 0, ...(ye(m) ?? {}) }))
      .data.floating_ips ?? []
  ).map((g) => ({
    id: String(g.id),
    address: g.floating_ip,
    status: g.status === 'associated' ? 'associated' : 'available',
    network_id: g.network_id,
    fixed_ip: g.fixed_ip,
    port_id: g.port_id
  }))
}
async function Ls(m, y) {
  const w = (
    await ee.post(
      '/v1/floating-ips',
      {
        tenant_id: m,
        network_id: y.network_id,
        subnet_id: y.subnet_id,
        port_id: y.port_id,
        fixed_ip: y.fixed_ip
      },
      ye(m)
    )
  ).data.floating_ip
  return {
    id: String(w.id),
    address: w.floating_ip,
    status: w.status === 'associated' ? 'associated' : 'available',
    network_id: w.network_id,
    fixed_ip: w.fixed_ip,
    port_id: w.port_id
  }
}
async function Rs(m) {
  await ee.delete(`/v1/floating-ips/${m}`)
}
async function bt(m, y) {
  const w = (await ee.put(`/v1/floating-ips/${m}`, y)).data.floating_ip
  return {
    id: String(w.id),
    address: w.floating_ip,
    status: w.status === 'associated' ? 'associated' : 'available',
    network_id: w.network_id,
    fixed_ip: w.fixed_ip,
    port_id: w.port_id
  }
}
async function it() {
  return ((await ee.get('/v1/zones')).data.zones ?? []).map((y) => ({
    id: String(y.id),
    name: y.name,
    allocation: y.allocation === 'disabled' ? 'disabled' : 'enabled',
    type: y.type === 'edge' ? 'edge' : 'core',
    network_type: y.network_type === 'Basic' ? 'Basic' : 'Advanced'
  }))
}
async function Ds(m) {
  const g = (
    await ee.post('/v1/zones', {
      name: m.name,
      allocation: m.allocation ?? 'enabled',
      type: m.type,
      network_type: m.network_type ?? 'Advanced'
    })
  ).data.zone
  return {
    id: String(g.id),
    name: g.name,
    allocation: g.allocation === 'disabled' ? 'disabled' : 'enabled',
    type: g.type === 'edge' ? 'edge' : 'core',
    network_type: g.network_type === 'Basic' ? 'Basic' : 'Advanced'
  }
}
async function As() {
  return ((await ee.get('/v1/projects')).data.projects ?? []).map((y) => ({
    id: String(y.id),
    name: y.name,
    description: y.description,
    user_id: y.user_id ? String(y.user_id) : void 0
  }))
}
async function Bs() {
  return ((await ee.get('/v1/users')).data.users ?? []).map((y) => ({
    id: String(y.id),
    username: y.username,
    email: y.email,
    first_name: y.first_name,
    last_name: y.last_name
  }))
}
async function Is(m) {
  const y = await ee.get('/v1/routers', { params: m ? { tenant_id: m } : void 0 })
  return (Array.isArray(y.data) ? y.data : (y.data.routers ?? [])).map((w) => ({
    id: String(w.id),
    name: w.name,
    description: w.description,
    tenant_id: w.tenant_id,
    external_gateway_network_id: w.external_gateway_network_id,
    external_gateway_ip: w.external_gateway_ip,
    enable_snat: w.enable_snat,
    admin_up: w.admin_up,
    status: w.status,
    created_at: w.created_at,
    updated_at: w.updated_at
  }))
}
async function Ms(m) {
  return (await ee.post('/v1/routers', m)).data
}
async function Ps(m, y) {
  return (await ee.put(`/v1/routers/${m}`, y)).data
}
async function Ts(m) {
  await ee.delete(`/v1/routers/${m}`)
}
async function Os(m) {
  const y = await ee.get(`/v1/routers/${m}/interfaces`)
  return Array.isArray(y.data) ? y.data : []
}
async function Rt(m, y) {
  return (await ee.post(`/v1/routers/${m}/add-interface`, { subnet_id: y })).data
}
async function Dt(m, y) {
  await ee.post(`/v1/routers/${m}/remove-interface`, { subnet_id: y })
}
async function nt(m, y) {
  return (await ee.post(`/v1/routers/${m}/set-gateway`, { external_network_id: y })).data
}
async function At(m) {
  return (await ee.post(`/v1/routers/${m}/clear-gateway`)).data
}
async function yt(m) {
  const y = await ee.get('/v1/topology', {
    params: m ? { tenant_id: m } : void 0,
    ...(ye(m) ?? {})
  })
  return { nodes: y.data.nodes || [], edges: y.data.edges || [] }
}
const Hs = Object.freeze(
  Object.defineProperty(
    {
      __proto__: null,
      addRouterInterface: Rt,
      allocateFloatingIP: Ls,
      attachVolumeToInstance: ms,
      clearRouterGateway: At,
      createFlavor: as,
      createInstance: hs,
      createNetwork: Nt,
      createRouter: Ms,
      createSSHKey: Lt,
      createVolume: us,
      createVolumeSnapshot: gs,
      createZone: Ds,
      default: ee,
      deleteFlavor: os,
      deleteFloatingIP: Rs,
      deleteImage: ht,
      deleteNetwork: xt,
      deleteNode: xs,
      deleteRouter: Ts,
      deleteSSHKey: Ns,
      deleteVolume: fs,
      destroyInstance: ws,
      detachVolumeFromInstance: _s,
      fetchAudit: vt,
      fetchDeletionStatus: ks,
      fetchFlavors: kt,
      fetchFloatingIPs: Es,
      fetchImages: ge,
      fetchInstanceVolumes: gt,
      fetchInstancesRaw: Ye,
      fetchNetworks: We,
      fetchNodes: vs,
      fetchPorts: Et,
      fetchProjects: As,
      fetchRouterInterfaces: Os,
      fetchRouters: Is,
      fetchSSHKeys: rt,
      fetchSnapshots: ds,
      fetchSubnets: st,
      fetchTopology: yt,
      fetchUsers: Bs,
      fetchVolumeSnapshots: tt,
      fetchVolumes: _t,
      fetchZones: it,
      forceDeleteInstance: Cs,
      importImage: cs,
      loginApi: ns,
      rebootInstance: Ss,
      registerImage: ls,
      removeRouterInterface: Dt,
      resizeVolume: ps,
      resolveApiBase: qe,
      restartNetwork: js,
      setRouterGateway: nt,
      startConsole: jt,
      startInstance: bs,
      stopInstance: ys,
      updateFloatingIP: bt,
      updateRouter: Ps,
      uploadImage: ct
    },
    Symbol.toStringTag,
    { value: 'Module' }
  )
)
function br() {
  const { projectId: m } = xe(),
    [y, g] = k.useState([]),
    [w, T] = k.useState(!1),
    [R, I] = k.useState([]),
    [i, o] = k.useState(''),
    [l, u] = k.useState([]),
    [a, h] = k.useState([])
  k.useEffect(() => {
    let L = !0
    return (
      T(!0),
      Promise.all([Es(m), We(m), Et({ tenant_id: m }), Ye(m)])
        .then(([M, P, j, D]) => {
          if (!L) return
          g(
            M.map(($) => ({
              id: $.id,
              address: $.address,
              status: $.status,
              port_id: $.port_id,
              fixed_ip: $.fixed_ip
            }))
          )
          const O = P.filter(($) => $.external)
          ;(I(O), O.length > 0 && o(O[0].id), u(j), h(D))
        })
        .finally(() => L && T(!1)),
      () => {
        L = !1
      }
    )
  }, [m])
  const f = k.useMemo(() => {
      const L = new Map()
      for (const M of a) L.set(String(M.id), M.name)
      return (M) => (M ? L.get(String(M)) || M : '')
    }, [a]),
    x = k.useMemo(
      () => (L) => {
        if (!L.port_id) return ''
        const M = l.find((D) => D.id === L.port_id),
          P = f(M?.device_id),
          j = L.fixed_ip || M?.fixed_ips?.[0]?.ip
        return [P, j].filter(Boolean).join(' @ ')
      },
      [l, f]
    ),
    c = k.useMemo(
      () => [
        { key: 'address', header: 'Public IP' },
        { key: 'status', header: 'Status' },
        {
          key: 'attached',
          header: 'Attached To',
          render: (L) => e.jsx('span', { className: 'text-xs text-gray-300', children: x(L) })
        },
        {
          key: 'id',
          header: '',
          className: 'w-10 text-right',
          render: (L) =>
            e.jsx('div', {
              className: 'flex justify-end',
              children: e.jsxs('div', {
                className: 'flex gap-3 items-center',
                children: [
                  L.status === 'available'
                    ? e.jsx('button', {
                        className: 'text-blue-400 hover:underline',
                        onClick: () => p(L.id),
                        children: 'Associate'
                      })
                    : e.jsx('button', {
                        className: 'text-yellow-300 hover:underline',
                        onClick: () => S(L.id),
                        children: 'Disassociate'
                      }),
                  e.jsx('button', {
                    className: 'text-red-400 hover:underline',
                    onClick: async () => {
                      ;(await Rs(L.id), g((M) => M.filter((P) => P.id !== L.id)))
                    },
                    children: 'Delete'
                  })
                ]
              })
            })
        }
      ],
      [x]
    ),
    [t, n] = k.useState(!1),
    [s, r] = k.useState(!1),
    [d, v] = k.useState(''),
    [_, b] = k.useState(''),
    p = (L) => {
      ;(v(L), b(''), r(!0))
    },
    S = async (L) => {
      const M = await bt(L, { fixed_ip: '', port_id: '' })
      g((P) =>
        P.map((j) =>
          j.id === M.id
            ? {
                id: M.id,
                address: M.address,
                status: M.status,
                port_id: M.port_id,
                fixed_ip: M.fixed_ip
              }
            : j
        )
      )
    }
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Public IPs',
        subtitle: 'Elastic IP addresses',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => n(!0),
          children: 'Allocate'
        })
      }),
      e.jsx(de, { columns: c, data: y, empty: w ? 'Loading…' : 'No public IPs' }),
      e.jsx(le, {
        title: 'Allocate Public IP',
        open: t,
        onClose: () => n(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => n(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: async () => {
                if (!m || !i) return
                const L = await Ls(m, { network_id: i })
                ;(g((M) => [...M, { id: L.id, address: L.address, status: L.status }]), n(!1))
              },
              children: 'Allocate'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Public Network' }),
                e.jsx('select', {
                  className: 'input w-full',
                  value: i,
                  onChange: (L) => o(L.target.value),
                  children: R.map((L) =>
                    e.jsxs(
                      'option',
                      { value: L.id, children: [L.name, ' ', L.cidr ? `(${L.cidr})` : ''] },
                      L.id
                    )
                  )
                })
              ]
            }),
            e.jsx('p', {
              className: 'text-xs text-gray-400',
              children: "IP will be auto-allocated from the selected network's first subnet pool."
            })
          ]
        })
      }),
      e.jsx(le, {
        title: 'Associate Public IP',
        open: s,
        onClose: () => r(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => r(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: !_,
              onClick: async () => {
                const L = l.find((j) => j.id === _)
                if (!d || !L || !L.fixed_ips || L.fixed_ips.length === 0) return
                const M = L.fixed_ips[0]?.ip,
                  P = await bt(d, { port_id: _, fixed_ip: M })
                ;(g((j) =>
                  j.map((D) =>
                    D.id === P.id
                      ? {
                          id: P.id,
                          address: P.address,
                          status: P.status,
                          port_id: P.port_id,
                          fixed_ip: P.fixed_ip
                        }
                      : D
                  )
                ),
                  r(!1))
              },
              children: 'Associate'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Select Instance Port' }),
                e.jsxs('select', {
                  className: 'input w-full',
                  value: _,
                  onChange: (L) => b(L.target.value),
                  children: [
                    e.jsx('option', { value: '', disabled: !0, children: 'Select a port' }),
                    l
                      .filter((L) => L.device_id)
                      .map((L) =>
                        e.jsxs(
                          'option',
                          {
                            value: L.id,
                            children: [
                              f(L.device_id),
                              ' •',
                              ' ',
                              L.fixed_ips && L.fixed_ips[0]?.ip ? L.fixed_ips[0].ip : L.id
                            ]
                          },
                          L.id
                        )
                      )
                  ]
                })
              ]
            }),
            e.jsx('p', {
              className: 'text-xs text-gray-400',
              children:
                'Select a VM port to map this Public IP to. The first fixed IP on the port will be used.'
            })
          ]
        })
      })
    ]
  })
}
function yr() {
  const { projectId: m } = xe(),
    [y, g] = k.useState([]),
    [w, T] = k.useState([]),
    [R, I] = k.useState([]),
    [i, o] = k.useState(!1),
    [l, u] = k.useState(null),
    [a, h] = k.useState(!1),
    [f, x] = k.useState(!1),
    [c, t] = k.useState(!1),
    [n, s] = k.useState(null),
    [r, d] = k.useState([]),
    [v, _] = k.useState(''),
    [b, p] = k.useState(''),
    [S, L] = k.useState(!0),
    [M, P] = k.useState(!0),
    [j, D] = k.useState(''),
    [O, $] = k.useState(''),
    [F, W] = k.useState(''),
    C = k.useMemo(() => w.filter((V) => V.external), [w]),
    A = k.useMemo(() => R.filter((V) => !r.some((se) => se.subnet_id === V.id)), [R, r])
  k.useEffect(() => {
    ;(N(), B(), z())
  }, [m])
  const N = async () => {
      try {
        const V = await Is(m)
        g(V)
      } catch (V) {
        u(V.message || 'Failed to load routers')
      }
    },
    B = async () => {
      try {
        const V = await We(m)
        T(V)
      } catch {
        T([])
      }
    },
    z = async () => {
      try {
        const V = await st(m)
        I(V)
      } catch {
        I([])
      }
    },
    K = async (V) => {
      try {
        const se = await Os(V)
        d(se)
      } catch (se) {
        u(se.message || 'Failed to load router interfaces')
      }
    },
    J = async () => {
      if (!v.trim()) {
        u('Router name is required')
        return
      }
      if (!m) {
        u('Project ID is required')
        return
      }
      ;(o(!0), u(null))
      try {
        const V = await Ms({ name: v, description: b, tenant_id: m, enable_snat: S, admin_up: M })
        ;(j && (await nt(V.id, j)), await N(), h(!1), _(''), p(''), L(!0), P(!0), D(''))
      } catch (V) {
        const se = V
        u(se.response?.data?.error || se.message || 'Failed to create router')
      } finally {
        o(!1)
      }
    },
    Q = async (V) => {
      if (confirm(`Are you sure you want to delete router "${V.name}"?`)) {
        ;(o(!0), u(null))
        try {
          ;(await Ts(V.id), await N())
        } catch (se) {
          const ne = se
          u(ne.response?.data?.error || ne.message || 'Failed to delete router')
        } finally {
          o(!1)
        }
      }
    },
    H = async () => {
      if (!(!n || !O)) {
        ;(o(!0), u(null))
        try {
          ;(await Rt(n.id, O), await K(n.id), await N(), $(''))
        } catch (V) {
          const se = V
          u(se.response?.data?.error || se.message || 'Failed to add interface')
        } finally {
          o(!1)
        }
      }
    },
    E = async (V) => {
      if (n && confirm('Are you sure you want to remove this interface?')) {
        ;(o(!0), u(null))
        try {
          ;(await Dt(n.id, V.subnet_id), await K(n.id), await N())
        } catch (se) {
          const ne = se
          u(ne.response?.data?.error || ne.message || 'Failed to remove interface')
        } finally {
          o(!1)
        }
      }
    },
    G = async () => {
      if (!n || !F) {
        u('Please select an external network')
        return
      }
      ;(o(!0), u(null))
      try {
        ;(await nt(n.id, F), await N(), t(!1), W(''), s(null))
      } catch (V) {
        const se = V
        u(se.response?.data?.error || se.message || 'Failed to set gateway')
      } finally {
        o(!1)
      }
    },
    q = async (V) => {
      if (confirm(`Are you sure you want to clear the external gateway for router "${V.name}"?`)) {
        ;(o(!0), u(null))
        try {
          ;(await At(V.id), await N())
        } catch (se) {
          const ne = se
          u(ne.response?.data?.error || ne.message || 'Failed to clear gateway')
        } finally {
          o(!1)
        }
      }
    },
    Z = (V) => {
      if (!V) return 'Not set'
      const se = w.find((ne) => ne.id === V)
      return se ? se.name : V
    },
    Y = [
      { key: 'name', header: 'Name' },
      { key: 'description', header: 'Description', render: (V) => V.description || '-' },
      {
        key: 'external_gateway',
        header: 'External Gateway',
        render: (V) =>
          e.jsx('span', {
            className: V.external_gateway_network_id ? 'text-green-400' : 'text-gray-500',
            children: Z(V.external_gateway_network_id)
          })
      },
      {
        key: 'snat',
        header: 'SNAT',
        render: (V) =>
          e.jsx('span', {
            className: V.enable_snat ? 'text-green-400' : 'text-gray-500',
            children: V.enable_snat ? 'Enabled' : 'Disabled'
          })
      },
      {
        key: 'status',
        header: 'Status',
        render: (V) =>
          e.jsx('span', { className: 'text-xs text-gray-300', children: V.status || 'ACTIVE' })
      },
      {
        key: 'actions',
        header: '',
        className: 'w-10 text-right',
        render: (V) =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx(Ge, {
              actions: [
                {
                  label: 'Interfaces',
                  onClick: async () => {
                    ;(s(V), await K(V.id), x(!0))
                  }
                },
                ...(V.external_gateway_network_id
                  ? [{ label: 'Clear Gateway', onClick: () => q(V), danger: !1 }]
                  : [
                      {
                        label: 'Set Gateway',
                        onClick: () => {
                          ;(s(V), t(!0))
                        }
                      }
                    ]),
                { label: 'Delete', onClick: () => Q(V), danger: !0 }
              ]
            })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Routers',
        subtitle: 'L3 routing for connecting networks',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => h(!0),
          children: 'Create Router'
        })
      }),
      l &&
        e.jsx('div', {
          className: 'p-3 bg-red-900/30 border border-red-800/50 rounded text-red-400 text-sm',
          children: l
        }),
      e.jsx(de, { columns: Y, data: y, empty: i ? 'Loading...' : 'No routers' }),
      e.jsx(le, {
        title: 'Create Router',
        open: a,
        onClose: () => {
          ;(h(!1), _(''), p(''), L(!0), P(!0), D(''))
        },
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => h(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: J,
              disabled: i || !v.trim(),
              children: i ? 'Creating...' : 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-4',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name *' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: v,
                  onChange: (V) => _(V.target.value),
                  placeholder: 'e.g.: main-router'
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Description' }),
                e.jsx('textarea', {
                  className: 'input w-full',
                  value: b,
                  onChange: (V) => p(V.target.value),
                  rows: 2,
                  placeholder: 'Optional'
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'External Gateway (Optional)' }),
                e.jsxs('select', {
                  className: 'input w-full',
                  value: j,
                  onChange: (V) => D(V.target.value),
                  children: [
                    e.jsx('option', { value: '', children: '-- Select external network --' }),
                    C.map((V) =>
                      e.jsxs('option', { value: V.id, children: [V.name, ' (', V.cidr, ')'] }, V.id)
                    )
                  ]
                }),
                e.jsx('p', {
                  className: 'text-xs text-gray-400 mt-1',
                  children:
                    'Select an external network to enable internet access for connected tenant networks'
                })
              ]
            }),
            e.jsxs('div', {
              className: 'flex items-center gap-2',
              children: [
                e.jsx('input', {
                  type: 'checkbox',
                  id: 'enable-snat',
                  checked: S,
                  onChange: (V) => L(V.target.checked)
                }),
                e.jsxs('label', {
                  htmlFor: 'enable-snat',
                  className: 'label m-0 cursor-pointer',
                  children: [
                    'Enable SNAT',
                    e.jsx('span', {
                      className: 'text-xs text-gray-400 ml-2',
                      children: '(for private network internet access)'
                    })
                  ]
                })
              ]
            }),
            e.jsxs('div', {
              className: 'flex items-center gap-2',
              children: [
                e.jsx('input', {
                  type: 'checkbox',
                  id: 'admin-up',
                  checked: M,
                  onChange: (V) => P(V.target.checked)
                }),
                e.jsx('label', {
                  htmlFor: 'admin-up',
                  className: 'label m-0 cursor-pointer',
                  children: 'Admin State Up'
                })
              ]
            })
          ]
        })
      }),
      e.jsx(le, {
        title: `Set External Gateway - ${n?.name || ''}`,
        open: c,
        onClose: () => {
          ;(t(!1), W(''), s(null))
        },
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => t(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: G,
              disabled: i || !F,
              children: i ? 'Setting...' : 'Set Gateway'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-4',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'External Network *' }),
                e.jsxs('select', {
                  className: 'input w-full',
                  value: F,
                  onChange: (V) => W(V.target.value),
                  children: [
                    e.jsx('option', { value: '', children: '-- Select external network --' }),
                    C.map((V) =>
                      e.jsxs('option', { value: V.id, children: [V.name, ' (', V.cidr, ')'] }, V.id)
                    )
                  ]
                })
              ]
            }),
            e.jsxs('div', {
              className:
                'p-3 bg-blue-900/20 border border-blue-800/30 rounded text-sm text-gray-300',
              children: [
                e.jsx('p', {
                  children:
                    'After setting gateway, internal networks connected to this router can access the internet.'
                }),
                n?.enable_snat &&
                  e.jsx('p', {
                    className: 'mt-2',
                    children:
                      'SNAT is enabled, internal IPs will be translated to external gateway IP.'
                  })
              ]
            })
          ]
        })
      }),
      e.jsx(le, {
        title: `Router Interfaces - ${n?.name || ''}`,
        open: f,
        onClose: () => {
          ;(x(!1), s(null), $(''))
        },
        footer: e.jsx('button', {
          className: 'btn-secondary',
          onClick: () => x(!1),
          children: 'Close'
        }),
        children: e.jsxs('div', {
          className: 'space-y-4',
          children: [
            e.jsxs('div', {
              className: 'p-4 bg-oxide-900 rounded border border-oxide-800',
              children: [
                e.jsx('h3', {
                  className: 'text-sm font-semibold text-gray-200 mb-3',
                  children: 'Add Interface'
                }),
                e.jsxs('div', {
                  className: 'flex gap-2',
                  children: [
                    e.jsxs('select', {
                      className: 'input flex-1',
                      value: O,
                      onChange: (V) => $(V.target.value),
                      children: [
                        e.jsx('option', { value: '', children: '-- Select subnet --' }),
                        A.map((V) => {
                          const se = w.find((ne) => ne.id === V.network_id)
                          return e.jsxs(
                            'option',
                            { value: V.id, children: [se?.name || V.network_id, ' - ', V.cidr] },
                            V.id
                          )
                        })
                      ]
                    }),
                    e.jsx('button', {
                      className: 'btn-primary',
                      onClick: H,
                      disabled: i || !O,
                      children: 'Add'
                    })
                  ]
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('h3', {
                  className: 'text-sm font-semibold text-gray-200 mb-3',
                  children: 'Connected Interfaces'
                }),
                r.length === 0
                  ? e.jsx('p', {
                      className: 'text-gray-500 text-sm',
                      children: 'No interfaces connected'
                    })
                  : e.jsx('div', {
                      className: 'space-y-2',
                      children: r.map((V) => {
                        const se = R.find((ue) => ue.id === V.subnet_id),
                          ne = se ? w.find((ue) => ue.id === se.network_id) : null
                        return e.jsxs(
                          'div',
                          {
                            className:
                              'flex items-center justify-between p-3 bg-oxide-900 rounded border border-oxide-800',
                            children: [
                              e.jsxs('div', {
                                children: [
                                  e.jsx('div', {
                                    className: 'text-sm font-medium text-gray-200',
                                    children: se?.cidr || V.subnet_id
                                  }),
                                  e.jsxs('div', {
                                    className: 'text-xs text-gray-400',
                                    children: [
                                      'Network: ',
                                      ne?.name || 'Unknown',
                                      ' • IP: ',
                                      V.ip_address
                                    ]
                                  })
                                ]
                              }),
                              e.jsx('button', {
                                onClick: () => E(V),
                                className: 'btn-danger text-sm',
                                children: 'Remove'
                              })
                            ]
                          },
                          V.id
                        )
                      })
                    })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Tt() {
  const { projectId: m } = xe(),
    { projects: y } = Ce(),
    [g, w] = k.useState([]),
    [T, R] = k.useState([]),
    [I, i] = k.useState(!1),
    [o, l] = k.useState(!1),
    [u, a] = k.useState(''),
    [h, f] = k.useState(''),
    [x, c] = k.useState(''),
    [t, n] = k.useState(m ?? ''),
    [s, r] = k.useState(''),
    [d, v] = k.useState('8.8.8.8'),
    [_, b] = k.useState('8.8.4.4'),
    [p, S] = k.useState(!0),
    [L, M] = k.useState(''),
    [P, j] = k.useState([]),
    [D, O] = k.useState(!0),
    [$, F] = k.useState(''),
    [W, C] = k.useState(''),
    [A, N] = k.useState(''),
    [B, z] = k.useState('86400'),
    [K, J] = k.useState('vxlan'),
    [Q, H] = k.useState(''),
    [E, G] = k.useState(''),
    [q, Z] = k.useState(!1),
    [Y, V] = k.useState(!1),
    [se, ne] = k.useState('1450'),
    ue = async () => {
      i(!0)
      try {
        const [X, ae] = await Promise.all([We(m), it()])
        ;(w(X), R(ae))
      } finally {
        i(!1)
      }
    }
  k.useEffect(() => {
    let X = !0
    return (
      i(!0),
      Promise.all([We(m), it()])
        .then(([ae, he]) => {
          X && (w(ae), R(he))
        })
        .finally(() => X && i(!1)),
      () => {
        X = !1
      }
    )
  }, [m])
  const re = k.useMemo(() => {
      const X = L.trim().toLowerCase()
      return X
        ? g.filter((ae) =>
            [ae.name, ae.cidr, ae.description, ae.zone].some((he) =>
              (he ?? '').toLowerCase().includes(X)
            )
          )
        : g
    }, [L, g]),
    Ee = k.useMemo(() => T.map((X) => X.name).filter(Boolean), [T]),
    Fe = k.useMemo(() => re.map((X) => X.id), [re]),
    Le = P.length > 0 && Fe.every((X) => P.includes(X)),
    Ae = (X) => {
      j(X ? Fe : [])
    },
    Be = (X, ae) => {
      j((he) => (ae ? Array.from(new Set([...he, X])) : he.filter((Ie) => Ie !== X)))
    },
    ve = [
      {
        key: 'select',
        header: '',
        sortable: !1,
        className: 'w-8',
        headerRender: e.jsx('input', {
          type: 'checkbox',
          checked: Le,
          onChange: (X) => Ae(X.target.checked)
        }),
        render: (X) =>
          e.jsx('input', {
            type: 'checkbox',
            checked: P.includes(X.id),
            onChange: (ae) => Be(X.id, ae.target.checked)
          })
      },
      { key: 'name', header: 'Name' },
      {
        key: 'network_type',
        header: 'Type',
        render: (X) => {
          const ae = X.network_type || 'vxlan',
            he = {
              vxlan: 'VXLAN (Overlay)',
              vlan: 'VLAN (Provider)',
              flat: 'Flat (Provider)',
              gre: 'GRE (Tunnel)',
              geneve: 'Geneve (Tunnel)',
              local: 'Local'
            }
          return e.jsxs('span', {
            className: 'text-xs',
            children: [
              e.jsx('span', {
                className: 'font-medium text-blue-400',
                children: he[ae] || ae.toUpperCase()
              }),
              X.segmentation_id &&
                e.jsxs('span', {
                  className: 'text-gray-400 ml-1',
                  children: ['(', X.segmentation_id, ')']
                })
            ]
          })
        }
      },
      {
        key: 'status',
        header: 'State',
        render: (X) =>
          e.jsx('span', { className: 'text-xs text-gray-300', children: X.status ?? 'active' })
      },
      { key: 'description', header: 'Description' },
      { key: 'cidr', header: 'CIDR' },
      {
        key: 'flags',
        header: 'Flags',
        render: (X) =>
          e.jsxs('div', {
            className: 'flex gap-1',
            children: [
              X.shared &&
                e.jsx('span', {
                  className: 'px-1.5 py-0.5 text-xs bg-green-900/30 text-green-400 rounded',
                  children: 'Shared'
                }),
              X.external &&
                e.jsx('span', {
                  className: 'px-1.5 py-0.5 text-xs bg-purple-900/30 text-purple-400 rounded',
                  children: 'External'
                })
            ]
          })
      },
      { key: 'tenant_id', header: 'Account' },
      { key: 'zone', header: 'Zone' },
      {
        key: 'actions',
        header: '',
        className: 'w-10 text-right',
        render: (X) =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx(Ge, {
              actions: [
                {
                  label: 'Delete',
                  danger: !0,
                  onClick: async () => {
                    ;(await xt(X.id), await ue(), j((ae) => ae.filter((he) => he !== X.id)))
                  }
                }
              ]
            })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'VPCs',
        subtitle: 'Virtual Private Clouds',
        actions: e.jsxs('div', {
          className: 'flex items-center gap-2',
          children: [
            e.jsx('button', { className: 'btn-secondary', onClick: ue, children: 'Refresh' }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: () => l(!0),
              children: 'Create VPC'
            }),
            P.length > 0 &&
              e.jsxs(e.Fragment, {
                children: [
                  e.jsx('button', {
                    className: 'btn-secondary',
                    onClick: async () => {
                      for (const X of P)
                        try {
                          await js(X)
                        } catch {}
                      await ue()
                    },
                    children: 'Restart VPC'
                  }),
                  e.jsx('button', {
                    className: 'btn-danger',
                    onClick: async () => {
                      const X = [...P]
                      for (const ae of X)
                        try {
                          await xt(ae)
                        } catch {}
                      ;(j([]), await ue())
                    },
                    children: 'Remove VPC'
                  })
                ]
              })
          ]
        })
      }),
      e.jsx(Ne, { placeholder: 'Search VPCs', onSearch: M }),
      e.jsx(de, { columns: ve, data: re, empty: I ? 'Loading…' : 'No VPCs' }),
      e.jsx(le, {
        title: 'Create VPC',
        open: o,
        onClose: () => l(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => l(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: async () => {
                if (!t || !u || !h || !x) return
                const X = {
                    name: u,
                    cidr: h,
                    zone: x,
                    description: s || void 0,
                    dns1: d || void 0,
                    dns2: _ || void 0,
                    start: p,
                    enable_dhcp: D,
                    dhcp_lease_time: parseInt(B) || 86400,
                    gateway: $ || void 0,
                    allocation_start: W || void 0,
                    allocation_end: A || void 0,
                    network_type: K,
                    physical_network: Q || void 0,
                    segmentation_id: E ? parseInt(E) : void 0,
                    shared: q,
                    external: Y,
                    mtu: se ? parseInt(se) : void 0
                  },
                  ae = await Nt(t, X)
                ;(w((he) => [...he, ae]),
                  a(''),
                  f(''),
                  c(''),
                  r(''),
                  v('8.8.8.8'),
                  b('8.8.4.4'),
                  S(!0),
                  O(!0),
                  F(''),
                  C(''),
                  N(''),
                  z('86400'),
                  n(m ?? ''),
                  J('vxlan'),
                  H(''),
                  G(''),
                  Z(!1),
                  V(!1),
                  ne('1450'),
                  l(!1))
              },
              children: 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-4',
          children: [
            e.jsxs('div', {
              className: 'space-y-3 border-b border-gray-700 pb-4',
              children: [
                e.jsx('h3', {
                  className: 'text-sm font-semibold text-gray-200',
                  children: 'Network Information'
                }),
                e.jsxs('div', {
                  className: 'grid grid-cols-2 gap-3',
                  children: [
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'Name *' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          value: u,
                          onChange: (X) => a(X.target.value)
                        })
                      ]
                    }),
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'Zone *' }),
                        e.jsxs('select', {
                          className: 'input w-full',
                          value: x,
                          onChange: (X) => c(X.target.value),
                          children: [
                            e.jsx('option', { value: '', disabled: !0, children: 'Select a zone' }),
                            Ee.map((X) => e.jsx('option', { value: X, children: X }, X))
                          ]
                        })
                      ]
                    })
                  ]
                }),
                e.jsxs('div', {
                  className: 'grid grid-cols-2 gap-3',
                  children: [
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'Network Type *' }),
                        e.jsxs('select', {
                          className: 'input w-full',
                          value: K,
                          onChange: (X) => {
                            const ae = X.target.value
                            ;(J(ae),
                              ne(
                                ae === 'vxlan' || ae === 'gre' || ae === 'geneve' ? '1450' : '1500'
                              ))
                          },
                          children: [
                            e.jsx('option', {
                              value: 'vxlan',
                              children: 'VXLAN (Overlay) - Recommended'
                            }),
                            e.jsx('option', { value: 'vlan', children: 'VLAN (Provider)' }),
                            e.jsx('option', { value: 'flat', children: 'Flat (Provider)' }),
                            e.jsx('option', { value: 'gre', children: 'GRE (Tunnel)' }),
                            e.jsx('option', { value: 'geneve', children: 'Geneve (Tunnel)' }),
                            e.jsx('option', { value: 'local', children: 'Local' })
                          ]
                        }),
                        e.jsxs('p', {
                          className: 'text-xs text-gray-400 mt-1',
                          children: [
                            K === 'vxlan' && 'Self-service overlay network, supports multi-node',
                            K === 'vlan' && 'Requires physical network and VLAN ID (1-4094)',
                            K === 'flat' && 'Direct connection to physical network',
                            (K === 'gre' || K === 'geneve') && 'Tunnel-based overlay network',
                            K === 'local' && 'Single node only'
                          ]
                        })
                      ]
                    }),
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'MTU' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          type: 'number',
                          placeholder: '1450',
                          value: se,
                          onChange: (X) => ne(X.target.value)
                        }),
                        e.jsx('p', {
                          className: 'text-xs text-gray-400 mt-1',
                          children: '1450 for overlay, 1500 for provider networks'
                        })
                      ]
                    })
                  ]
                }),
                (K === 'vlan' || K === 'flat') &&
                  e.jsxs('div', {
                    className:
                      'grid grid-cols-2 gap-3 p-3 bg-blue-900/10 border border-blue-800/30 rounded',
                    children: [
                      e.jsxs('div', {
                        children: [
                          e.jsx('label', { className: 'label', children: 'Physical Network *' }),
                          e.jsx('input', {
                            className: 'input w-full',
                            placeholder: 'provider',
                            value: Q,
                            onChange: (X) => H(X.target.value)
                          }),
                          e.jsx('p', {
                            className: 'text-xs text-gray-400 mt-1',
                            children:
                              'Must match bridge_mappings config (e.g., "provider", "external")'
                          })
                        ]
                      }),
                      K === 'vlan' &&
                        e.jsxs('div', {
                          children: [
                            e.jsx('label', { className: 'label', children: 'VLAN ID *' }),
                            e.jsx('input', {
                              className: 'input w-full',
                              type: 'number',
                              min: '1',
                              max: '4094',
                              placeholder: '100',
                              value: E,
                              onChange: (X) => G(X.target.value)
                            }),
                            e.jsx('p', {
                              className: 'text-xs text-gray-400 mt-1',
                              children: 'VLAN tag (1-4094)'
                            })
                          ]
                        })
                    ]
                  }),
                (K === 'vxlan' || K === 'gre') &&
                  e.jsxs('div', {
                    children: [
                      e.jsx('label', {
                        className: 'label',
                        children: 'Segmentation ID (Optional)'
                      }),
                      e.jsx('input', {
                        className: 'input w-full',
                        type: 'number',
                        placeholder: 'Auto-assigned',
                        value: E,
                        onChange: (X) => G(X.target.value)
                      }),
                      e.jsxs('p', {
                        className: 'text-xs text-gray-400 mt-1',
                        children: [
                          K === 'vxlan' ? 'VNI (VXLAN Network Identifier)' : 'Tunnel key',
                          ' - leave empty for auto'
                        ]
                      })
                    ]
                  }),
                e.jsxs('div', {
                  className: 'grid grid-cols-2 gap-3',
                  children: [
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'Account *' }),
                        e.jsxs('select', {
                          className: 'input w-full',
                          value: t,
                          onChange: (X) => n(X.target.value),
                          children: [
                            e.jsx('option', {
                              value: '',
                              disabled: !0,
                              children: 'Select an account'
                            }),
                            y.map((X) =>
                              e.jsxs(
                                'option',
                                { value: X.id, children: [X.name, ' (', X.id, ')'] },
                                X.id
                              )
                            )
                          ]
                        })
                      ]
                    }),
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'CIDR *' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          placeholder: '10.0.0.0/16',
                          value: h,
                          onChange: (X) => f(X.target.value)
                        })
                      ]
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Description' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      value: s,
                      onChange: (X) => r(X.target.value)
                    })
                  ]
                }),
                e.jsxs('div', {
                  className: 'flex gap-6',
                  children: [
                    e.jsxs('div', {
                      className: 'flex items-center gap-2',
                      children: [
                        e.jsx('input', {
                          type: 'checkbox',
                          id: 'network-shared',
                          checked: q,
                          onChange: (X) => Z(X.target.checked)
                        }),
                        e.jsxs('label', {
                          htmlFor: 'network-shared',
                          className: 'label m-0 cursor-pointer',
                          children: [
                            'Shared Network',
                            e.jsx('span', {
                              className: 'text-xs text-gray-400 ml-2',
                              children: '(accessible by multiple tenants)'
                            })
                          ]
                        })
                      ]
                    }),
                    e.jsxs('div', {
                      className: 'flex items-center gap-2',
                      children: [
                        e.jsx('input', {
                          type: 'checkbox',
                          id: 'network-external',
                          checked: Y,
                          onChange: (X) => V(X.target.checked)
                        }),
                        e.jsxs('label', {
                          htmlFor: 'network-external',
                          className: 'label m-0 cursor-pointer',
                          children: [
                            'External Network',
                            e.jsx('span', {
                              className: 'text-xs text-gray-400 ml-2',
                              children: '(for floating IPs)'
                            })
                          ]
                        })
                      ]
                    })
                  ]
                })
              ]
            }),
            e.jsxs('div', {
              className: 'space-y-3 border-b border-gray-700 pb-4',
              children: [
                e.jsxs('div', {
                  className: 'flex items-center gap-2',
                  children: [
                    e.jsx('input', {
                      type: 'checkbox',
                      checked: D,
                      onChange: (X) => O(X.target.checked)
                    }),
                    e.jsx('h3', {
                      className: 'text-sm font-semibold text-gray-200',
                      children: 'Enable DHCP'
                    })
                  ]
                }),
                D &&
                  e.jsxs(e.Fragment, {
                    children: [
                      e.jsxs('div', {
                        className: 'grid grid-cols-2 gap-3',
                        children: [
                          e.jsxs('div', {
                            children: [
                              e.jsx('label', { className: 'label', children: 'Gateway IP' }),
                              e.jsx('input', {
                                className: 'input w-full',
                                placeholder: 'Auto (e.g., 10.0.0.1)',
                                value: $,
                                onChange: (X) => F(X.target.value)
                              }),
                              e.jsx('p', {
                                className: 'text-xs text-gray-400 mt-1',
                                children: 'Leave empty for auto-calculation (.1 of subnet)'
                              })
                            ]
                          }),
                          e.jsxs('div', {
                            children: [
                              e.jsx('label', {
                                className: 'label',
                                children: 'DHCP Lease Time (seconds)'
                              }),
                              e.jsx('input', {
                                className: 'input w-full',
                                type: 'number',
                                placeholder: '86400',
                                value: B,
                                onChange: (X) => z(X.target.value)
                              }),
                              e.jsx('p', {
                                className: 'text-xs text-gray-400 mt-1',
                                children: 'Default: 86400 (24 hours)'
                              })
                            ]
                          })
                        ]
                      }),
                      e.jsxs('div', {
                        className: 'grid grid-cols-2 gap-3',
                        children: [
                          e.jsxs('div', {
                            children: [
                              e.jsx('label', {
                                className: 'label',
                                children: 'Allocation Pool Start'
                              }),
                              e.jsx('input', {
                                className: 'input w-full',
                                placeholder: 'Auto (e.g., 10.0.0.2)',
                                value: W,
                                onChange: (X) => C(X.target.value)
                              }),
                              e.jsx('p', {
                                className: 'text-xs text-gray-400 mt-1',
                                children: 'First IP in DHCP pool'
                              })
                            ]
                          }),
                          e.jsxs('div', {
                            children: [
                              e.jsx('label', {
                                className: 'label',
                                children: 'Allocation Pool End'
                              }),
                              e.jsx('input', {
                                className: 'input w-full',
                                placeholder: 'Auto (last usable IP)',
                                value: A,
                                onChange: (X) => N(X.target.value)
                              }),
                              e.jsx('p', {
                                className: 'text-xs text-gray-400 mt-1',
                                children: 'Last IP in DHCP pool'
                              })
                            ]
                          })
                        ]
                      }),
                      e.jsxs('div', {
                        className: 'grid grid-cols-2 gap-3',
                        children: [
                          e.jsxs('div', {
                            children: [
                              e.jsx('label', { className: 'label', children: 'Primary DNS' }),
                              e.jsx('input', {
                                className: 'input w-full',
                                placeholder: '8.8.8.8',
                                value: d,
                                onChange: (X) => v(X.target.value)
                              })
                            ]
                          }),
                          e.jsxs('div', {
                            children: [
                              e.jsx('label', { className: 'label', children: 'Secondary DNS' }),
                              e.jsx('input', {
                                className: 'input w-full',
                                placeholder: '8.8.4.4',
                                value: _,
                                onChange: (X) => b(X.target.value)
                              })
                            ]
                          })
                        ]
                      })
                    ]
                  })
              ]
            }),
            e.jsxs('div', {
              className: 'flex items-center gap-2',
              children: [
                e.jsx('input', {
                  type: 'checkbox',
                  checked: p,
                  onChange: (X) => S(X.target.checked)
                }),
                e.jsx('label', {
                  className: 'label m-0',
                  children: 'Activate Network Immediately'
                }),
                e.jsx('span', {
                  className: 'text-xs text-gray-400',
                  children: '(create in SDN backend)'
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Sr() {
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'VPN',
        subtitle: 'Site-to-site and client VPN',
        actions: e.jsx('button', { className: 'btn-primary', children: 'Create VPN' })
      }),
      e.jsx('div', { className: 'card p-4 text-gray-300', children: 'No VPNs' })
    ]
  })
}
function wr() {
  const m = [],
    y = [
      { key: 'direction', header: 'Direction' },
      { key: 'protocol', header: 'Protocol' },
      { key: 'ports', header: 'Ports' },
      { key: 'cidr', header: 'CIDR' },
      {
        key: 'actions',
        header: '',
        className: 'w-10 text-right',
        render: () =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx(Ge, { actions: [{ label: 'Delete', onClick: () => {}, danger: !0 }] })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Security Groups',
        subtitle: 'Ingress/Egress rules',
        actions: e.jsx('button', { className: 'btn-primary', children: 'Add Rule' })
      }),
      e.jsx(de, { columns: y, data: m, empty: 'No rules' })
    ]
  })
}
function Cr() {
  return e.jsx('div', {
    className: 'space-y-4',
    children: e.jsxs(Ve, {
      children: [
        e.jsx(te, { path: 'vpc', element: e.jsx(Tt, {}) }),
        e.jsx(te, { path: 'routers', element: e.jsx(yr, {}) }),
        e.jsx(te, { path: 'sg', element: e.jsx(wr, {}) }),
        e.jsx(te, { path: 'topology', element: e.jsx(Nr, {}) }),
        e.jsx(te, { path: 'public-ips', element: e.jsx(br, {}) }),
        e.jsx(te, { path: 'asns', element: e.jsx(kr, {}) }),
        e.jsx(te, { path: 'vpn', element: e.jsx(Sr, {}) }),
        e.jsx(te, { path: 'acl', element: e.jsx(jr, {}) }),
        e.jsx(te, { path: '*', element: e.jsx(Tt, {}) })
      ]
    })
  })
}
function kr() {
  const { projectId: m } = xe(),
    { asns: y, addAsn: g, removeAsn: w } = Ce(),
    T = k.useMemo(() => y.filter((h) => h.projectId === m), [y, m]),
    [R, I] = k.useState(!1),
    [i, o] = k.useState(''),
    [l, u] = k.useState(''),
    a = [
      { key: 'number', header: 'ASN' },
      { key: 'description', header: 'Description' },
      {
        key: 'actions',
        header: '',
        className: 'w-10 text-right',
        render: (h) =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx(Ge, {
              actions: [{ label: 'Delete', onClick: () => w(h.id), danger: !0 }]
            })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'ASNs',
        subtitle: 'Autonomous System Numbers',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => I(!0),
          children: 'Add ASN'
        })
      }),
      e.jsx(de, { columns: a, data: T, empty: 'No ASNs' }),
      e.jsx(le, {
        title: 'Add ASN',
        open: R,
        onClose: () => I(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => I(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: () => {
                m &&
                  i &&
                  (g({ projectId: m, number: Number(i), description: l || void 0 }),
                  o(''),
                  u(''),
                  I(!1))
              },
              children: 'Save'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'ASN' }),
                e.jsx('input', {
                  className: 'input w-full',
                  type: 'number',
                  value: i,
                  onChange: (h) => o(h.target.value ? Number(h.target.value) : '')
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Description' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: l,
                  onChange: (h) => u(h.target.value)
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function jr() {
  const m = [],
    y = [
      { key: 'id', header: 'ID' },
      { key: 'rule', header: 'Rule' },
      { key: 'action', header: 'Action' },
      {
        key: 'actions',
        header: '',
        className: 'w-10 text-right',
        render: () =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx(Ge, { actions: [{ label: 'Delete', onClick: () => {}, danger: !0 }] })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Network ACL',
        subtitle: 'ACL rules',
        actions: e.jsx('button', { className: 'btn-primary', children: 'Add Rule' })
      }),
      e.jsx(de, { columns: y, data: m, empty: 'No ACL rules' })
    ]
  })
}
function Nr() {
  const { projectId: m } = xe(),
    [y, g] = k.useState('graph'),
    [w, T] = k.useState(!1),
    [R, I] = k.useState({ nodes: [], edges: [] })
  k.useEffect(() => {
    if (!m) return
    let o = !0
    return (
      T(!0),
      yt(m)
        .then((l) => {
          o && I(l)
        })
        .finally(() => o && T(!1)),
      () => {
        o = !1
      }
    )
  }, [m])
  const i = async () => {
    if (m) {
      T(!0)
      try {
        const o = await yt(m)
        I(o)
      } finally {
        T(!1)
      }
    }
  }
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Network Topology',
        subtitle: 'Visualize network architecture and resources',
        actions: e.jsxs('div', {
          className: 'flex gap-2',
          children: [
            e.jsx('button', {
              className: `px-4 py-2 rounded ${y === 'graph' ? 'bg-blue-600 text-white' : 'bg-oxide-800 text-gray-300'}`,
              onClick: () => g('graph'),
              children: e.jsxs('span', {
                className: 'flex items-center gap-2',
                children: [
                  e.jsx('svg', {
                    className: 'w-4 h-4',
                    fill: 'none',
                    stroke: 'currentColor',
                    viewBox: '0 0 24 24',
                    children: e.jsx('path', {
                      strokeLinecap: 'round',
                      strokeLinejoin: 'round',
                      strokeWidth: 2,
                      d: 'M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7'
                    })
                  }),
                  'Topology Graph'
                ]
              })
            }),
            e.jsx('button', {
              className: `px-4 py-2 rounded ${y === 'list' ? 'bg-blue-600 text-white' : 'bg-oxide-800 text-gray-300'}`,
              onClick: () => g('list'),
              children: e.jsxs('span', {
                className: 'flex items-center gap-2',
                children: [
                  e.jsx('svg', {
                    className: 'w-4 h-4',
                    fill: 'none',
                    stroke: 'currentColor',
                    viewBox: '0 0 24 24',
                    children: e.jsx('path', {
                      strokeLinecap: 'round',
                      strokeLinejoin: 'round',
                      strokeWidth: 2,
                      d: 'M4 6h16M4 10h16M4 14h16M4 18h16'
                    })
                  }),
                  'Network Diagram'
                ]
              })
            })
          ]
        })
      }),
      w
        ? e.jsx('div', {
            className: 'card p-8 text-center text-gray-400',
            children: 'Loading topology...'
          })
        : y === 'graph'
          ? e.jsx(Er, { topology: R, onRefresh: i })
          : e.jsx(Lr, { topology: R })
    ]
  })
}
function Er({ topology: m, onRefresh: y }) {
  const { projectId: g } = xe(),
    [w, T] = k.useState(null),
    [R, I] = k.useState({}),
    [i, o] = k.useState(null),
    [l, u] = k.useState(null),
    [a, h] = k.useState(!1),
    [f, x] = k.useState(!1),
    [c, t] = k.useState(''),
    [n, s] = k.useState(!1),
    [r, d] = k.useState(''),
    [v, _] = k.useState(!1),
    [b, p] = k.useState('')
  k.useEffect(() => {
    if (m.nodes.length === 0) return
    const O = {}
    let $ = 150,
      F = 150,
      W = 150,
      C = 150
    for (const A of m.nodes)
      A.type === 'network' && A.external && ((O[A.id] = { x: $, y: 80 }), ($ += 220))
    for (const A of m.nodes) A.type === 'router' && ((O[A.id] = { x: F, y: 250 }), (F += 260))
    for (const A of m.nodes) A.type === 'subnet' && ((O[A.id] = { x: W, y: 420 }), (W += 200))
    for (const A of m.nodes) A.type === 'instance' && ((O[A.id] = { x: C, y: 540 }), (C += 180))
    I((A) => ({ ...O, ...A }))
  }, [m.nodes])
  const S = (O, $, F) => {
    I((W) => {
      const C = W[O] || { x: 0, y: 0 }
      return { ...W, [O]: { x: C.x + $, y: C.y + F } }
    })
  }
  if (m.nodes.length === 0)
    return e.jsxs('div', {
      className: 'card p-8 text-center',
      children: [
        e.jsx('div', { className: 'text-gray-400 mb-4', children: 'No network resources found' }),
        e.jsx('div', {
          className: 'text-sm text-gray-500',
          children: 'Create a network or router to begin building your topology'
        })
      ]
    })
  const L = m.nodes.filter((O) => O.type === 'network' && O.external),
    M = m.nodes.filter((O) => O.type === 'router'),
    P = m.nodes.filter((O) => O.type === 'subnet'),
    j = m.nodes.filter((O) => O.type === 'instance'),
    D = {
      onMouseDown: (O, $) => {
        const F = O.clientX,
          W = O.clientY,
          C = (N) => S($, N.clientX - F, N.clientY - W),
          A = () => {
            ;(window.removeEventListener('mousemove', C), window.removeEventListener('mouseup', A))
          }
        ;(window.addEventListener('mousemove', C), window.addEventListener('mouseup', A))
      }
    }
  return e.jsxs('div', {
    className: 'card p-6',
    children: [
      e.jsxs('div', {
        className: 'mb-4 flex items-center justify-between',
        children: [
          e.jsxs('div', {
            className: 'flex items-center gap-4 text-xs',
            children: [
              e.jsxs('div', {
                className: 'flex items-center gap-2',
                children: [
                  e.jsx('div', { className: 'w-3 h-3 rounded-full bg-purple-500' }),
                  e.jsx('span', { className: 'text-gray-400', children: 'External Network' })
                ]
              }),
              e.jsxs('div', {
                className: 'flex items-center gap-2',
                children: [
                  e.jsx('div', { className: 'w-3 h-3 rounded-full bg-blue-500' }),
                  e.jsx('span', { className: 'text-gray-400', children: 'Internal Network' })
                ]
              }),
              e.jsxs('div', {
                className: 'flex items-center gap-2',
                children: [
                  e.jsx('div', { className: 'w-3 h-3 rounded-full bg-green-500' }),
                  e.jsx('span', { className: 'text-gray-400', children: 'Router' })
                ]
              })
            ]
          }),
          w &&
            e.jsx('button', {
              className: 'text-xs text-gray-400 hover:text-gray-200',
              onClick: () => T(null),
              children: 'Clear Selection'
            })
        ]
      }),
      e.jsxs('svg', {
        viewBox: '0 0 1200 680',
        className: 'w-full h-[600px] bg-oxide-950 rounded border border-oxide-800',
        children: [
          e.jsx('defs', {
            children: e.jsx('marker', {
              id: 'arrowhead',
              markerWidth: '10',
              markerHeight: '10',
              refX: '9',
              refY: '3',
              orient: 'auto',
              children: e.jsx('polygon', { points: '0 0, 10 3, 0 6', fill: '#64748b' })
            })
          }),
          L.map((O, $) => {
            const F = O.id,
              W = R[F] || { x: 150 + $ * 250, y: 80 },
              C = w === F
            return e.jsxs(
              'g',
              {
                onClick: () => T(F),
                className: 'cursor-move',
                onMouseDown: (A) => D.onMouseDown(A, F),
                children: [
                  e.jsx('rect', {
                    x: W.x - 60,
                    y: W.y - 30,
                    width: '120',
                    height: '60',
                    rx: '8',
                    fill: C ? '#7c3aed' : '#6d28d9',
                    stroke: C ? '#a78bfa' : '#8b5cf6',
                    strokeWidth: '2'
                  }),
                  e.jsx('text', {
                    x: W.x,
                    y: W.y - 5,
                    textAnchor: 'middle',
                    fill: '#e5e7eb',
                    fontSize: '13',
                    fontWeight: '600',
                    children:
                      (O.name ?? '').length > 15
                        ? (O.name ?? '').substring(0, 15) + '...'
                        : (O.name ?? '')
                  }),
                  e.jsxs('text', {
                    x: W.x,
                    y: W.y + 12,
                    textAnchor: 'middle',
                    fill: '#d1d5db',
                    fontSize: '11',
                    children: ['External • ', O.cidr]
                  })
                ]
              },
              F
            )
          }),
          m.edges.map((O, $) => {
            const F = R[O.source] || { x: 0, y: 0 },
              W = R[O.target] || { x: 0, y: 0 },
              C =
                O.type === 'l3-gateway'
                  ? '#64748b'
                  : O.type === 'l3'
                    ? '#5eead4'
                    : O.type === 'l2'
                      ? '#60a5fa'
                      : '#a3e635',
              A = O.type === 'l3' ? '3,3' : void 0
            return e.jsx(
              'line',
              {
                x1: F.x,
                y1: F.y,
                x2: W.x,
                y2: W.y,
                stroke: C,
                strokeWidth: '2',
                strokeDasharray: A,
                markerEnd: 'url(#arrowhead)'
              },
              $
            )
          }),
          M.map((O, $) => {
            const F = O.id,
              W = R[F] || { x: 200 + $ * 300, y: 250 },
              C = w === F,
              A = !!m.nodes.find((N) => N.id === F)?.external_gateway_network_id
            return e.jsxs(
              'g',
              {
                onClick: () => T(F),
                className: 'cursor-move',
                onMouseDown: (N) => D.onMouseDown(N, F),
                children: [
                  e.jsx('circle', {
                    cx: W.x,
                    cy: W.y,
                    r: '35',
                    fill: C ? '#10b981' : '#059669',
                    stroke: C ? '#34d399' : '#10b981',
                    strokeWidth: '2'
                  }),
                  e.jsx('text', {
                    x: W.x,
                    y: W.y + 5,
                    textAnchor: 'middle',
                    fill: '#e5e7eb',
                    fontSize: '12',
                    fontWeight: '600',
                    children:
                      (O.name ?? '').length > 10
                        ? (O.name ?? '').substring(0, 10) + '...'
                        : (O.name ?? '')
                  }),
                  e.jsx('text', {
                    x: W.x,
                    y: W.y + 55,
                    textAnchor: 'middle',
                    fill: '#d1d5db',
                    fontSize: '10',
                    children: 'Router'
                  }),
                  A &&
                    e.jsx('text', {
                      x: W.x,
                      y: W.y + 68,
                      textAnchor: 'middle',
                      fill: '#34d399',
                      fontSize: '9',
                      children: '✓ Gateway'
                    }),
                  m.nodes.find((N) => N.id === F)?.enable_snat &&
                    e.jsx('text', {
                      x: W.x,
                      y: W.y + 81,
                      textAnchor: 'middle',
                      fill: '#60a5fa',
                      fontSize: '9',
                      children: 'SNAT'
                    })
                ]
              },
              F
            )
          }),
          P.map((O, $) => {
            const F = O.id,
              W = R[F] || { x: 150 + $ * 200, y: 420 },
              C = w === F
            return e.jsxs(
              'g',
              {
                onClick: () => T(F),
                className: 'cursor-move',
                onMouseDown: (A) => D.onMouseDown(A, F),
                children: [
                  e.jsx('rect', {
                    x: W.x - 60,
                    y: W.y - 30,
                    width: '120',
                    height: '60',
                    rx: '8',
                    fill: C ? '#3b82f6' : '#2563eb',
                    stroke: C ? '#60a5fa' : '#3b82f6',
                    strokeWidth: '2'
                  }),
                  e.jsx('text', {
                    x: W.x,
                    y: W.y - 5,
                    textAnchor: 'middle',
                    fill: '#e5e7eb',
                    fontSize: '13',
                    fontWeight: '600',
                    children:
                      (O.name ?? '').length > 15
                        ? (O.name ?? '').substring(0, 15) + '...'
                        : (O.name ?? '')
                  }),
                  e.jsx('text', {
                    x: W.x,
                    y: W.y + 12,
                    textAnchor: 'middle',
                    fill: '#d1d5db',
                    fontSize: '11',
                    children: O.cidr
                  })
                ]
              },
              F
            )
          }),
          j.map((O, $) => {
            const F = O.id,
              W = R[F] || { x: 150 + $ * 180, y: 560 },
              C = w === F
            return e.jsxs(
              'g',
              {
                onClick: () => T(F),
                className: 'cursor-move',
                onMouseDown: (A) => D.onMouseDown(A, F),
                children: [
                  e.jsx('rect', {
                    x: W.x - 50,
                    y: W.y - 20,
                    width: '100',
                    height: '40',
                    rx: '6',
                    fill: C ? '#0ea5e9' : '#0284c7',
                    stroke: C ? '#38bdf8' : '#0ea5e9',
                    strokeWidth: '2'
                  }),
                  e.jsx('text', {
                    x: W.x,
                    y: W.y - 2,
                    textAnchor: 'middle',
                    fill: '#e5e7eb',
                    fontSize: '12',
                    fontWeight: '600',
                    children: O.name ?? 'VM'
                  }),
                  e.jsx('text', {
                    x: W.x,
                    y: W.y + 12,
                    textAnchor: 'middle',
                    fill: '#d1d5db',
                    fontSize: '10',
                    children: O.state ?? ''
                  })
                ]
              },
              F
            )
          })
        ]
      }),
      w &&
        e.jsxs('div', {
          className: 'mt-4 p-4 bg-oxide-900 rounded border border-oxide-700',
          children: [
            e.jsx('div', {
              className: 'text-sm font-semibold text-gray-200 mb-2',
              children: 'Selected Resource'
            }),
            e.jsx('div', {
              className: 'text-xs text-gray-400',
              children: (() => {
                const O = m.nodes.find(($) => $.id === w)
                return O
                  ? O.type === 'network'
                    ? `Network: ${O.name}`
                    : O.type === 'subnet'
                      ? `Subnet: ${O.name} (${O.cidr})`
                      : O.type === 'router'
                        ? `Router: ${O.name}`
                        : O.type === 'instance'
                          ? `Instance: ${O.name} (${O.state ?? ''})`
                          : 'Unknown'
                  : 'Unknown'
              })()
            }),
            (() => {
              const O = m.nodes.find((K) => K.id === w)
              if (!O || O.type !== 'router') return null
              const $ = O.resource_id || '',
                F = !!O.external_gateway_network_id,
                W = !!O.enable_snat,
                C = async () => {
                  if (g) {
                    if (!i) {
                      const K = await We(g)
                      o(K.filter((J) => J.external))
                    }
                    ;(t(''), x(!0))
                  }
                },
                A = async () => {
                  if ($) {
                    h(!0)
                    try {
                      ;(await At($), y?.())
                    } finally {
                      h(!1)
                    }
                  }
                },
                N = async () => {
                  if ($) {
                    h(!0)
                    try {
                      ;(await Ps($, { enable_snat: !W }), y?.())
                    } finally {
                      h(!1)
                    }
                  }
                },
                B = async () => {
                  if (g) {
                    if (!l) {
                      const K = await st(g)
                      u(K)
                    }
                    ;(d(''), s(!0))
                  }
                },
                z = async () => {
                  if (g) {
                    if (!l) {
                      const K = await st(g)
                      u(K)
                    }
                    ;(p(''), _(!0))
                  }
                }
              return e.jsxs('div', {
                className: 'mt-3 flex flex-wrap gap-2',
                children: [
                  F
                    ? e.jsx('button', {
                        className: 'btn-secondary btn-xs',
                        onClick: A,
                        disabled: a,
                        children: 'Clear Gateway'
                      })
                    : e.jsx('button', {
                        className: 'btn-primary btn-xs',
                        onClick: C,
                        children: 'Set Gateway'
                      }),
                  e.jsx('button', {
                    className: 'btn-secondary btn-xs',
                    onClick: N,
                    disabled: a,
                    children: W ? 'Disable SNAT' : 'Enable SNAT'
                  }),
                  e.jsx('button', {
                    className: 'btn-secondary btn-xs',
                    onClick: B,
                    children: 'Add Interface'
                  }),
                  e.jsx('button', {
                    className: 'btn-secondary btn-xs',
                    onClick: z,
                    children: 'Remove Interface'
                  })
                ]
              })
            })(),
            f &&
              e.jsx(le, {
                title: 'Set Router Gateway',
                open: f,
                onClose: () => x(!1),
                footer: e.jsxs(e.Fragment, {
                  children: [
                    e.jsx('button', {
                      className: 'btn-secondary',
                      onClick: () => x(!1),
                      children: 'Cancel'
                    }),
                    e.jsx('button', {
                      className: 'btn-primary',
                      disabled: !c || a,
                      onClick: async () => {
                        const $ = m.nodes.find((F) => F.id === w)?.resource_id || ''
                        if (!(!$ || !c)) {
                          h(!0)
                          try {
                            ;(await nt($, c), y?.(), x(!1))
                          } finally {
                            h(!1)
                          }
                        }
                      },
                      children: 'Set Gateway'
                    })
                  ]
                }),
                children: e.jsx('div', {
                  className: 'space-y-3',
                  children: e.jsxs('div', {
                    children: [
                      e.jsx('label', { className: 'label', children: 'External Network' }),
                      e.jsxs('select', {
                        className: 'input w-full',
                        value: c,
                        onChange: (O) => t(O.target.value),
                        children: [
                          e.jsx('option', {
                            value: '',
                            disabled: !0,
                            children: 'Select external network'
                          }),
                          (i ?? []).map((O) =>
                            e.jsxs(
                              'option',
                              { value: O.id, children: [O.name, ' (', O.cidr, ')'] },
                              O.id
                            )
                          )
                        ]
                      }),
                      i !== null &&
                        i.length === 0 &&
                        e.jsx('p', {
                          className: 'text-xs text-gray-400 mt-2',
                          children:
                            'No external networks available. Create a flat/VLAN network and mark it External.'
                        })
                    ]
                  })
                })
              }),
            n &&
              e.jsx(le, {
                title: 'Add Router Interface',
                open: n,
                onClose: () => s(!1),
                footer: e.jsxs(e.Fragment, {
                  children: [
                    e.jsx('button', {
                      className: 'btn-secondary',
                      onClick: () => s(!1),
                      children: 'Cancel'
                    }),
                    e.jsx('button', {
                      className: 'btn-primary',
                      disabled: !r || a,
                      onClick: async () => {
                        const $ = m.nodes.find((F) => F.id === w)?.resource_id || ''
                        if (!(!$ || !r)) {
                          h(!0)
                          try {
                            ;(await Rt($, r), y?.(), s(!1))
                          } finally {
                            h(!1)
                          }
                        }
                      },
                      children: 'Add Interface'
                    })
                  ]
                }),
                children: e.jsx('div', {
                  className: 'space-y-3',
                  children: e.jsxs('div', {
                    children: [
                      e.jsx('label', { className: 'label', children: 'Subnet' }),
                      e.jsxs('select', {
                        className: 'input w-full',
                        value: r,
                        onChange: (O) => d(O.target.value),
                        children: [
                          e.jsx('option', { value: '', disabled: !0, children: 'Select subnet' }),
                          (() => {
                            const O = m.nodes.find((F) => F.id === w),
                              $ = new Set(O?.interfaces ?? [])
                            return (l ?? [])
                              .filter((F) => !$.has(F.id))
                              .map((F) =>
                                e.jsxs(
                                  'option',
                                  { value: F.id, children: [F.name, ' (', F.cidr, ')'] },
                                  F.id
                                )
                              )
                          })()
                        ]
                      })
                    ]
                  })
                })
              }),
            v &&
              e.jsx(le, {
                title: 'Remove Router Interface',
                open: v,
                onClose: () => _(!1),
                footer: e.jsxs(e.Fragment, {
                  children: [
                    e.jsx('button', {
                      className: 'btn-secondary',
                      onClick: () => _(!1),
                      children: 'Cancel'
                    }),
                    e.jsx('button', {
                      className: 'btn-danger',
                      disabled: !b || a,
                      onClick: async () => {
                        const $ = m.nodes.find((F) => F.id === w)?.resource_id || ''
                        if (!(!$ || !b)) {
                          h(!0)
                          try {
                            ;(await Dt($, b), y?.(), _(!1))
                          } finally {
                            h(!1)
                          }
                        }
                      },
                      children: 'Remove Interface'
                    })
                  ]
                }),
                children: e.jsx('div', {
                  className: 'space-y-3',
                  children: e.jsxs('div', {
                    children: [
                      e.jsx('label', { className: 'label', children: 'Subnet' }),
                      e.jsxs('select', {
                        className: 'input w-full',
                        value: b,
                        onChange: (O) => p(O.target.value),
                        children: [
                          e.jsx('option', { value: '', disabled: !0, children: 'Select subnet' }),
                          (() => {
                            const O = m.nodes.find((W) => W.id === w),
                              $ = new Set(O?.interfaces ?? []),
                              F = (l ?? []).filter((W) => $.has(W.id))
                            return F.length > 0
                              ? F.map((W) =>
                                  e.jsxs(
                                    'option',
                                    { value: W.id, children: [W.name, ' (', W.cidr, ')'] },
                                    W.id
                                  )
                                )
                              : e.jsx('option', {
                                  value: '',
                                  disabled: !0,
                                  children: 'No interfaces to remove'
                                })
                          })()
                        ]
                      })
                    ]
                  })
                })
              })
          ]
        })
    ]
  })
}
function Lr({ topology: m }) {
  const y = m.nodes.filter((R) => R.type === 'network'),
    g = m.nodes.filter((R) => R.type === 'router'),
    w = m.nodes.filter((R) => R.type === 'subnet'),
    T = m.nodes.filter((R) => R.type === 'instance')
  return y.length === 0 && g.length === 0 && w.length === 0 && T.length === 0
    ? e.jsxs('div', {
        className: 'card p-8 text-center',
        children: [
          e.jsx('div', { className: 'text-gray-400 mb-4', children: 'No network resources found' }),
          e.jsx('div', {
            className: 'text-sm text-gray-500',
            children: 'Create a network or router to begin'
          })
        ]
      })
    : e.jsxs('div', {
        className: 'space-y-4',
        children: [
          g.length > 0 &&
            e.jsxs('div', {
              className: 'card',
              children: [
                e.jsx('div', {
                  className: 'p-4 border-b border-oxide-800',
                  children: e.jsxs('h3', {
                    className: 'text-sm font-semibold text-gray-200',
                    children: ['Routers (', g.length, ')']
                  })
                }),
                e.jsx('div', {
                  className: 'divide-y divide-oxide-800',
                  children: g.map((R) =>
                    e.jsx(
                      'div',
                      {
                        className: 'p-4 hover:bg-oxide-900/50',
                        children: e.jsx('div', {
                          className: 'flex items-center justify-between mb-2',
                          children: e.jsxs('div', {
                            className: 'flex items-center gap-3',
                            children: [
                              e.jsx('div', {
                                className:
                                  'w-10 h-10 rounded-full bg-green-600 flex items-center justify-center text-white text-xs font-bold',
                                children: 'R'
                              }),
                              e.jsxs('div', {
                                children: [
                                  e.jsx('div', {
                                    className: 'text-sm font-medium text-gray-200',
                                    children: R.name
                                  }),
                                  e.jsxs('div', {
                                    className: 'text-xs text-gray-400',
                                    children: ['Router • ', R.id]
                                  })
                                ]
                              })
                            ]
                          })
                        })
                      },
                      R.id
                    )
                  )
                })
              ]
            }),
          y.filter((R) => R.external).length > 0 &&
            e.jsxs('div', {
              className: 'card',
              children: [
                e.jsx('div', {
                  className: 'p-4 border-b border-oxide-800',
                  children: e.jsxs('h3', {
                    className: 'text-sm font-semibold text-gray-200',
                    children: ['External Networks (', y.filter((R) => R.external).length, ')']
                  })
                }),
                e.jsx('div', {
                  className: 'divide-y divide-oxide-800',
                  children: y
                    .filter((R) => R.external)
                    .map((R) =>
                      e.jsx(
                        'div',
                        {
                          className: 'p-4 hover:bg-oxide-900/50',
                          children: e.jsxs('div', {
                            className: 'flex items-center justify-between mb-2',
                            children: [
                              e.jsxs('div', {
                                className: 'flex items-center gap-3',
                                children: [
                                  e.jsx('div', {
                                    className:
                                      'w-10 h-10 rounded bg-purple-600 flex items-center justify-center text-white text-xs font-bold',
                                    children: 'EXT'
                                  }),
                                  e.jsxs('div', {
                                    children: [
                                      e.jsx('div', {
                                        className: 'text-sm font-medium text-gray-200',
                                        children: R.name
                                      }),
                                      e.jsx('div', {
                                        className: 'text-xs text-gray-400',
                                        children: R.cidr
                                      })
                                    ]
                                  })
                                ]
                              }),
                              e.jsx('div', {
                                className: 'flex items-center gap-2',
                                children: e.jsx('span', {
                                  className:
                                    'px-2 py-1 text-xs bg-purple-900/30 text-purple-400 rounded',
                                  children: 'External'
                                })
                              })
                            ]
                          })
                        },
                        R.id
                      )
                    )
                })
              ]
            }),
          y.filter((R) => !R.external).length > 0 &&
            e.jsxs('div', {
              className: 'card',
              children: [
                e.jsx('div', {
                  className: 'p-4 border-b border-oxide-800',
                  children: e.jsxs('h3', {
                    className: 'text-sm font-semibold text-gray-200',
                    children: ['Internal Networks (', y.filter((R) => !R.external).length, ')']
                  })
                }),
                e.jsx('div', {
                  className: 'divide-y divide-oxide-800',
                  children: y
                    .filter((R) => !R.external)
                    .map((R) =>
                      e.jsx(
                        'div',
                        {
                          className: 'p-4 hover:bg-oxide-900/50',
                          children: e.jsx('div', {
                            className: 'flex items-center justify-between mb-2',
                            children: e.jsxs('div', {
                              className: 'flex items-center gap-3',
                              children: [
                                e.jsx('div', {
                                  className:
                                    'w-10 h-10 rounded bg-blue-600 flex items-center justify-center text-white text-xs font-bold',
                                  children: 'NET'
                                }),
                                e.jsxs('div', {
                                  children: [
                                    e.jsx('div', {
                                      className: 'text-sm font-medium text-gray-200',
                                      children: R.name
                                    }),
                                    e.jsx('div', {
                                      className: 'text-xs text-gray-400',
                                      children: R.cidr
                                    })
                                  ]
                                })
                              ]
                            })
                          })
                        },
                        R.id
                      )
                    )
                })
              ]
            }),
          w.length > 0 &&
            e.jsxs('div', {
              className: 'card',
              children: [
                e.jsx('div', {
                  className: 'p-4 border-b border-oxide-800',
                  children: e.jsxs('h3', {
                    className: 'text-sm font-semibold text-gray-200',
                    children: ['Subnets (', w.length, ')']
                  })
                }),
                e.jsx('div', {
                  className: 'divide-y divide-oxide-800',
                  children: w.map((R) =>
                    e.jsx(
                      'div',
                      {
                        className: 'p-4 hover:bg-oxide-900/50',
                        children: e.jsx('div', {
                          className: 'flex items-center justify-between mb-2',
                          children: e.jsxs('div', {
                            className: 'flex items-center gap-3',
                            children: [
                              e.jsx('div', {
                                className:
                                  'w-10 h-10 rounded bg-cyan-600 flex items-center justify-center text-white text-xs font-bold',
                                children: 'S'
                              }),
                              e.jsxs('div', {
                                children: [
                                  e.jsx('div', {
                                    className: 'text-sm font-medium text-gray-200',
                                    children: R.name
                                  }),
                                  e.jsx('div', {
                                    className: 'text-xs text-gray-400',
                                    children: R.cidr
                                  })
                                ]
                              })
                            ]
                          })
                        })
                      },
                      R.id
                    )
                  )
                })
              ]
            }),
          T.length > 0 &&
            e.jsxs('div', {
              className: 'card',
              children: [
                e.jsx('div', {
                  className: 'p-4 border-b border-oxide-800',
                  children: e.jsxs('h3', {
                    className: 'text-sm font-semibold text-gray-200',
                    children: ['Instances (', T.length, ')']
                  })
                }),
                e.jsx('div', {
                  className: 'divide-y divide-oxide-800',
                  children: T.map((R) =>
                    e.jsx(
                      'div',
                      {
                        className: 'p-4 hover:bg-oxide-900/50',
                        children: e.jsx('div', {
                          className: 'flex items-center justify-between mb-2',
                          children: e.jsxs('div', {
                            className: 'flex items-center gap-3',
                            children: [
                              e.jsx('div', {
                                className:
                                  'w-10 h-10 rounded bg-sky-600 flex items-center justify-center text-white text-xs font-bold',
                                children: 'VM'
                              }),
                              e.jsxs('div', {
                                children: [
                                  e.jsx('div', {
                                    className: 'text-sm font-medium text-gray-200',
                                    children: R.name
                                  }),
                                  e.jsx('div', {
                                    className: 'text-xs text-gray-400',
                                    children: R.state ?? ''
                                  })
                                ]
                              })
                            ]
                          })
                        })
                      },
                      R.id
                    )
                  )
                })
              ]
            })
        ]
      })
}
const Rr = {
  default: 'bg-oxide-800 text-gray-200 border-oxide-700',
  success: 'bg-emerald-900/40 text-emerald-300 border-emerald-800',
  warning: 'bg-amber-900/40 text-amber-300 border-amber-800',
  danger: 'bg-rose-900/40 text-rose-300 border-rose-800',
  info: 'bg-sky-900/40 text-sky-300 border-sky-800'
}
function pe({ children: m, variant: y = 'default' }) {
  return e.jsx('span', {
    className: `inline-flex items-center rounded-full border px-2 py-0.5 text-xs ${Rr[y]}`,
    children: m
  })
}
function Dr({ instanceIds: m, onComplete: y, onClose: g }) {
  const [w, T] = k.useState(new Map(m.map((f) => [f, { instanceId: f, status: 'checking' }]))),
    [R, I] = k.useState(!0),
    i = k.useCallback(async () => {
      const f = new Map(w)
      let x = !0
      for (const c of m)
        try {
          const t = await ks(c)
          ;(f.set(c, {
            instanceId: c,
            status: t.status,
            message: t.last_error,
            retryCount: t.retry_count,
            maxRetries: t.max_retries
          }),
            t.status !== 'completed' && t.status !== 'failed' && (x = !1))
        } catch {
          f.get(c)?.status === 'checking' && (x = !1)
        }
      ;(T(f),
        x &&
          (I(!1),
          setTimeout(() => {
            y()
          }, 2e3)))
    }, [m, w, y])
  k.useEffect(() => {
    if (!R) return
    i()
    const f = setInterval(i, 2e3)
    return () => clearInterval(f)
  }, [R, i])
  const o = (f) => {
      switch (f) {
        case 'completed':
          return 'text-green-400'
        case 'failed':
          return 'text-red-400'
        case 'processing':
          return 'text-blue-400'
        case 'pending':
          return 'text-yellow-400'
        default:
          return 'text-gray-400'
      }
    },
    l = (f) => {
      switch (f) {
        case 'completed':
          return e.jsx('svg', {
            className: 'w-5 h-5 text-green-400',
            fill: 'none',
            stroke: 'currentColor',
            viewBox: '0 0 24 24',
            children: e.jsx('path', {
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
              strokeWidth: 2,
              d: 'M5 13l4 4L19 7'
            })
          })
        case 'failed':
          return e.jsx('svg', {
            className: 'w-5 h-5 text-red-400',
            fill: 'none',
            stroke: 'currentColor',
            viewBox: '0 0 24 24',
            children: e.jsx('path', {
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
              strokeWidth: 2,
              d: 'M6 18L18 6M6 6l12 12'
            })
          })
        case 'processing':
          return e.jsxs('svg', {
            className: 'w-5 h-5 text-blue-400 animate-spin',
            fill: 'none',
            viewBox: '0 0 24 24',
            children: [
              e.jsx('circle', {
                className: 'opacity-25',
                cx: '12',
                cy: '12',
                r: '10',
                stroke: 'currentColor',
                strokeWidth: '4'
              }),
              e.jsx('path', {
                className: 'opacity-75',
                fill: 'currentColor',
                d: 'M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z'
              })
            ]
          })
        case 'pending':
          return e.jsx('svg', {
            className: 'w-5 h-5 text-yellow-400',
            fill: 'none',
            stroke: 'currentColor',
            viewBox: '0 0 24 24',
            children: e.jsx('path', {
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
              strokeWidth: 2,
              d: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z'
            })
          })
        default:
          return e.jsx('svg', {
            className: 'w-5 h-5 text-gray-400 animate-pulse',
            fill: 'none',
            stroke: 'currentColor',
            viewBox: '0 0 24 24',
            children: e.jsx('path', {
              strokeLinecap: 'round',
              strokeLinejoin: 'round',
              strokeWidth: 2,
              d: 'M12 6v6m0 0v6m0-6h6m-6 0H6'
            })
          })
      }
    },
    u = Array.from(w.values()).filter((f) => f.status === 'completed').length,
    a = Array.from(w.values()).filter((f) => f.status === 'failed').length,
    h = ((u + a) / m.length) * 100
  return e.jsx(le, {
    open: !0,
    onClose: R ? () => {} : g,
    title: 'Deleting Instances',
    children: e.jsxs('div', {
      className: 'space-y-4',
      children: [
        e.jsxs('div', {
          className: 'space-y-2',
          children: [
            e.jsxs('div', {
              className: 'flex justify-between text-sm text-gray-400',
              children: [
                e.jsx('span', { children: 'Progress' }),
                e.jsxs('span', { children: [u + a, ' / ', m.length] })
              ]
            }),
            e.jsx('div', {
              className: 'w-full bg-gray-700 rounded-full h-2',
              children: e.jsx('div', {
                className: 'bg-blue-500 h-2 rounded-full transition-all duration-300',
                style: { width: `${h}%` }
              })
            })
          ]
        }),
        e.jsx('div', {
          className: 'space-y-2 max-h-96 overflow-y-auto',
          children: Array.from(w.values()).map((f) =>
            e.jsxs(
              'div',
              {
                className: 'flex items-start gap-3 p-3 bg-gray-800 rounded-lg',
                children: [
                  e.jsx('div', { className: 'flex-shrink-0 mt-0.5', children: l(f.status) }),
                  e.jsxs('div', {
                    className: 'flex-1 min-w-0',
                    children: [
                      e.jsxs('div', {
                        className: 'flex items-center gap-2',
                        children: [
                          e.jsxs('span', {
                            className: 'text-sm font-medium text-gray-200',
                            children: ['Instance ', f.instanceId]
                          }),
                          e.jsx('span', {
                            className: `text-xs font-semibold ${o(f.status)}`,
                            children: f.status.toUpperCase()
                          })
                        ]
                      }),
                      f.message &&
                        e.jsx('p', { className: 'text-xs text-red-400 mt-1', children: f.message }),
                      f.retryCount !== void 0 &&
                        f.retryCount > 0 &&
                        e.jsxs('p', {
                          className: 'text-xs text-yellow-400 mt-1',
                          children: ['Retry ', f.retryCount, '/', f.maxRetries]
                        })
                    ]
                  })
                ]
              },
              f.instanceId
            )
          )
        }),
        !R &&
          e.jsx('div', {
            className: 'pt-4 border-t border-gray-700',
            children: e.jsxs('div', {
              className: 'flex items-center justify-between text-sm',
              children: [
                e.jsxs('span', { className: 'text-green-400', children: ['✓ Completed: ', u] }),
                a > 0 && e.jsxs('span', { className: 'text-red-400', children: ['✗ Failed: ', a] })
              ]
            })
          }),
        !R &&
          e.jsx('button', {
            onClick: g,
            className:
              'w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors',
            children: 'Close'
          })
      ]
    })
  })
}
function Xe(m) {
  if (!m) return 'unknown error'
  if (typeof m == 'string') return m
  if (typeof m == 'object') {
    const y = m
    if (typeof y.message == 'string') return y.message
  }
  return 'unknown error'
}
function Ot() {
  const { projectId: m } = xe(),
    [y, g] = k.useState([]),
    [w, T] = k.useState(!1),
    [R, I] = k.useState('all'),
    [i, o] = k.useState(''),
    [l, u] = k.useState(new Set()),
    [a, h] = k.useState({}),
    [f, x] = k.useState({}),
    [c, t] = k.useState({}),
    [n, s] = k.useState(new Set()),
    [r, d] = k.useState([]),
    [v, _] = k.useState(!1),
    { flavors: b, setFlavors: p } = Ce(),
    [S, L] = k.useState([]),
    [M, P] = k.useState([]),
    [j, D] = k.useState([]),
    [O, $] = k.useState(!1),
    [F, W] = k.useState(''),
    [C, A] = k.useState(''),
    [N, B] = k.useState(''),
    [z, K] = k.useState(!1),
    [J, Q] = k.useState(''),
    [H, E] = k.useState(''),
    [G, q] = k.useState(''),
    [Z, Y] = k.useState(!1),
    [V, se] = k.useState(!1),
    [ne, ue] = k.useState(''),
    [re, Ee] = k.useState(''),
    [Fe, Le] = k.useState(!1),
    [Ae, Be] = k.useState(''),
    [ve, X] = k.useState('10.0.0.0/24'),
    ae = k.useMemo(() => {
      const ie = S.find((ce) => String(ce.id) === N)?.minDiskGiB ?? 0
      return Math.max(1, ie)
    }, [S, N]),
    he = k.useCallback(async () => {
      T(!0)
      try {
        const [U] = await Promise.all([Ye(m)])
        g(U)
      } finally {
        T(!1)
      }
    }, [m])
  k.useEffect(() => {
    let U = !0
    return (
      T(!0),
      Ye(m)
        .then((ie) => {
          U && g(ie)
        })
        .catch(() => {
          U && g([])
        })
        .finally(() => {
          U && T(!1)
        }),
      Promise.allSettled([kt(), ge(m), We(m), rt(m), As(), Bs()]).then((ie) => {
        if (!U) return
        const [ce, _e, be, Se, De, $e] = ie
        if (
          (ce.status === 'fulfilled' && p(ce.value),
          _e.status === 'fulfilled' && L(_e.value),
          be.status === 'fulfilled' && P(be.value),
          Se.status === 'fulfilled' && D(Se.value),
          De.status === 'fulfilled')
        ) {
          const ze = {}
          ;(De.value.forEach((Pe) => {
            ze[String(Pe.id)] = Pe.name
          }),
            x(ze))
        }
        if ($e.status === 'fulfilled') {
          const ze = {}
          ;($e.value.forEach((Pe) => {
            ze[String(Pe.id)] =
              Pe.username ||
              `${Pe.first_name ?? ''} ${Pe.last_name ?? ''}`.trim() ||
              Pe.email ||
              String(Pe.id)
          }),
            t(ze))
        }
      }),
      () => {
        U = !1
      }
    )
  }, [m])
  const [Ie, je] = k.useState(0)
  ;(k.useEffect(() => {
    const U = y.some((ie) => ie.status === 'building' || ie.status === 'spawning')
    if ((y.length === 0 || U) && Ie < 3) {
      const ie = setTimeout(async () => {
        ;(await he(), je((ce) => ce + 1))
      }, 2e3)
      return () => clearTimeout(ie)
    }
    y.length > 0 && !U && Ie !== 0 && je(0)
  }, [y, Ie, he]),
    k.useEffect(() => {
      let U = !1
      async function ie() {
        const ce = await Promise.all(
          y.map(async (_e) => {
            try {
              const Se =
                (await Et({ tenant_id: m, device_id: _e.uuid })).find(
                  (De) => De.fixed_ips && De.fixed_ips.length > 0
                )?.fixed_ips?.[0]?.ip || ''
              return [String(_e.id), Se]
            } catch {
              return [String(_e.id), '']
            }
          })
        )
        if (!U) {
          const _e = {}
          for (const [be, Se] of ce) _e[be] = Se
          h(_e)
        }
      }
      return (
        y.length > 0 ? ie() : h({}),
        () => {
          U = !0
        }
      )
    }, [y, m]))
  const Me = k.useMemo(() => {
      const U = y.filter((ce) => {
          if (R === 'all') return !0
          const _e =
            ce.power_state === 'running' || ce.status === 'active' || ce.status === 'running'
          return R === 'running' ? _e : !_e
        }),
        ie = i.trim().toLowerCase()
      return ie
        ? U.filter(
            (ce) =>
              ce.name.toLowerCase().includes(ie) ||
              String(ce.uuid || '')
                .toLowerCase()
                .includes(ie) ||
              String(ce.host_id || '')
                .toLowerCase()
                .includes(ie)
          )
        : U
    }, [y, R, i]),
    Re = Me.length > 0 && Me.every((U) => l.has(String(U.id))),
    Ue = (U) => {
      u(U ? new Set(Me.map((ie) => String(ie.id))) : new Set())
    },
    fe = (U, ie) => {
      u((ce) => {
        const _e = new Set(ce)
        return (ie ? _e.add(U) : _e.delete(U), _e)
      })
    },
    dt = [
      {
        key: '__sel__',
        header: '',
        headerRender: e.jsx('input', {
          type: 'checkbox',
          'aria-label': 'Select all',
          checked: Re,
          onChange: (U) => Ue(U.target.checked)
        }),
        render: (U) =>
          e.jsx('input', {
            type: 'checkbox',
            'aria-label': `Select ${U.name}`,
            checked: l.has(String(U.id)),
            onChange: (ie) => fe(String(U.id), ie.target.checked),
            onClick: (ie) => ie.stopPropagation()
          }),
        className: 'w-8'
      },
      {
        key: 'name',
        header: 'Name',
        render: (U) =>
          e.jsx('button', {
            className: 'text-primary-400 hover:underline',
            onClick: async (ie) => {
              ;(ie.stopPropagation(),
                await jt(String(U.id)),
                (window.location.href = `/project/${m}/compute/instances/${U.id}/console`))
            },
            children: U.name
          })
      },
      {
        key: 'status',
        header: 'state',
        render: (U) => {
          const ie = String(U.id)
          return n.has(ie)
            ? e.jsxs(pe, {
                variant: 'info',
                children: [
                  'provisioning',
                  e.jsx('span', { className: 'ml-1 inline-block animate-spin', children: '⏳' })
                ]
              })
            : U.status === 'building' || U.status === 'spawning'
              ? e.jsx(pe, { variant: 'warning', children: 'building' })
              : U.status === 'error'
                ? e.jsx(pe, { variant: 'danger', children: 'error' })
                : U.power_state === 'running' || U.status === 'active' || U.status === 'running'
                  ? e.jsx(pe, { variant: 'success', children: 'running' })
                  : e.jsx(pe, { children: 'stopped' })
        }
      },
      { key: 'vm_id', header: 'Internal name', render: (U) => U.vm_id || U.uuid },
      { key: 'ip', header: 'ip address', render: (U) => a[String(U.id)] || '' },
      { key: 'host_id', header: 'Host' },
      {
        key: 'user_id',
        header: 'Account',
        render: (U) => (U.user_id ? (c[String(U.user_id)] ?? String(U.user_id)) : '')
      },
      {
        key: 'project_id',
        header: 'Zone',
        render: (U) => (U.project_id ? (f[String(U.project_id)] ?? String(U.project_id)) : '')
      },
      {
        key: 'actions',
        header: 'disks',
        className: 'w-24 text-right',
        render: (U) =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx('button', {
              className: 'text-blue-400 hover:underline',
              title: 'View instance disks',
              onClick: (ie) => {
                ;(ie.stopPropagation(),
                  (window.location.href = `/project/${m}/compute/instances/${U.id}/volumes`))
              },
              children: 'View disks'
            })
          })
      }
    ]
  async function Xs() {
    if (!(!F || !C || !N) && H) {
      K(!0)
      try {
        const U = { name: F, flavor_id: Number(C), image_id: Number(N), enable_tpm: Z },
          ie = Number(J)
        if (
          (!Number.isNaN(ie) && ie > 0 && (U.root_disk_gb = ie), (U.networks = [{ uuid: H }]), G)
        ) {
          const be = j.find((Se) => Se.id === G)
          be && (U.ssh_key = be.public_key)
        }
        const ce = await hs(m, U)
        g((be) => [ce, ...be])
        const _e = String(ce.id)
        ;(s((be) => new Set(be).add(_e)),
          (async () => {
            try {
              const be = Date.now() + 2e4
              for (; Date.now() < be; ) {
                await new Promise(($e) => setTimeout($e, 1500))
                const Se = await Ye(m)
                g(Se)
                const De = Se.find(($e) => String($e.id) === _e)
                if (De) {
                  const $e = De.status === 'active' && De.power_state === 'running',
                    ze = De.status === 'error'
                  if ($e || ze) break
                }
              }
            } catch {
            } finally {
              s((be) => {
                const Se = new Set(be)
                return (Se.delete(_e), Se)
              })
            }
          })(),
          $(!1),
          W(''),
          A(''),
          B(''),
          Q(''),
          E(''),
          q(''),
          Y(!1))
      } finally {
        K(!1)
      }
    }
  }
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, { title: 'Instances', subtitle: 'Virtual machines' }),
      e.jsx(Ne, {
        placeholder: 'Search name/uuid/host',
        onSearch: o,
        children: e.jsxs('div', {
          className: 'flex items-center gap-2',
          children: [
            e.jsx('button', {
              className: 'btn-secondary h-9',
              onClick: he,
              disabled: w,
              children: 'Refresh'
            }),
            e.jsxs('select', {
              className: 'input h-9',
              value: R,
              onChange: (U) => I(U.target.value),
              children: [
                e.jsx('option', { value: 'all', children: 'All' }),
                e.jsx('option', { value: 'running', children: 'Running' }),
                e.jsx('option', { value: 'stopped', children: 'Stopped' })
              ]
            }),
            e.jsx('button', {
              className: 'btn-primary h-9 w-40',
              onClick: async () => {
                $(!0)
                try {
                  const [U, ie, ce] = await Promise.allSettled([ge(m), We(m), rt(m)])
                  ;(U.status === 'fulfilled' && L(U.value),
                    ie.status === 'fulfilled' && P(ie.value),
                    ce.status === 'fulfilled' && D(ce.value))
                } catch {}
              },
              children: 'Add Instance'
            }),
            l.size > 0 &&
              e.jsxs('div', {
                className: 'flex items-center gap-2 ml-2',
                children: [
                  e.jsx('button', {
                    className: 'icon-btn',
                    'aria-label': 'Start instance',
                    title: 'Start instance',
                    onClick: async () => {
                      try {
                        ;(await Promise.all(Array.from(l).map((U) => bs(U))),
                          me.success(`Started ${l.size} instance(s)`))
                      } catch (U) {
                        me.error(`Start failed: ${Xe(U)}`)
                      } finally {
                        await he()
                      }
                    },
                    children: e.jsx('svg', {
                      width: '18',
                      height: '18',
                      viewBox: '0 0 24 24',
                      fill: 'currentColor',
                      children: e.jsx('path', { d: 'M8 5v14l11-7z' })
                    })
                  }),
                  e.jsx('button', {
                    className: 'icon-btn',
                    'aria-label': 'Stop instance',
                    title: 'Stop instance',
                    onClick: async () => {
                      try {
                        ;(await Promise.all(Array.from(l).map((U) => ys(U))),
                          me.info(`Stopped ${l.size} instance(s)`))
                      } catch (U) {
                        me.error(`Stop failed: ${Xe(U)}`)
                      } finally {
                        await he()
                      }
                    },
                    children: e.jsx('svg', {
                      width: '18',
                      height: '18',
                      viewBox: '0 0 24 24',
                      fill: 'currentColor',
                      children: e.jsx('path', { d: 'M6 6h12v12H6z' })
                    })
                  }),
                  e.jsx('button', {
                    className: 'icon-btn',
                    'aria-label': 'Restart instance',
                    title: 'Restart instance',
                    onClick: async () => {
                      try {
                        ;(await Promise.all(Array.from(l).map((U) => Ss(U))),
                          me.info(`Restarted ${l.size} instance(s)`))
                      } catch (U) {
                        me.error(`Restart failed: ${Xe(U)}`)
                      } finally {
                        await he()
                      }
                    },
                    children: e.jsx('svg', {
                      width: '18',
                      height: '18',
                      viewBox: '0 0 24 24',
                      fill: 'currentColor',
                      children: e.jsx('path', {
                        d: 'M12 6V3L8 7l4 4V8a4 4 0 1 1-4 4H6a6 6 0 1 0 6-6z'
                      })
                    })
                  }),
                  e.jsx('button', {
                    className: 'icon-btn text-rose-300',
                    'aria-label': 'Destroy instance',
                    title: 'Destroy instance',
                    onClick: async () => {
                      if (confirm(`Destroy ${l.size} instance(s)?`))
                        try {
                          const U = Array.from(l)
                          ;(await Promise.all(U.map((ie) => ws(ie))), d(U), _(!0), u(new Set()))
                        } catch (U) {
                          ;(me.error(`Destroy failed: ${Xe(U)}`), await he())
                        }
                    },
                    children: e.jsx('svg', {
                      width: '18',
                      height: '18',
                      viewBox: '0 0 24 24',
                      fill: 'currentColor',
                      children: e.jsx('path', { d: 'M6 7h12l-1 14H7L6 7zm3-3h6l1 2H8l1-2z' })
                    })
                  }),
                  e.jsx('button', {
                    className: 'icon-btn text-rose-500',
                    'aria-label': 'Force delete (orphaned VMs)',
                    title:
                      "Force delete - removes database records for VMs stuck in 'deleting' state",
                    onClick: async () => {
                      const ie = Array.from(l)
                        .map((ce) => y.find((_e) => String(_e.id) === ce))
                        .filter(Boolean)
                        .filter((ce) => ce.status === 'deleting')
                      if (ie.length === 0) {
                        me.info('Force delete only works on instances stuck in "deleting" status')
                        return
                      }
                      if (
                        confirm(`Force delete ${ie.length} instance(s) stuck in deleting state?

This will remove database records but NOT delete VMs from hypervisor.`)
                      )
                        try {
                          ;(await Promise.all(ie.map((ce) => Cs(String(ce.id)))),
                            me.success(`Force deleted ${ie.length} instance(s)`),
                            u(new Set()),
                            await he())
                        } catch (ce) {
                          ;(me.error(`Force delete failed: ${Xe(ce)}`), await he())
                        }
                    },
                    children: e.jsx('svg', {
                      width: '18',
                      height: '18',
                      viewBox: '0 0 24 24',
                      fill: 'currentColor',
                      children: e.jsx('path', {
                        d: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z'
                      })
                    })
                  })
                ]
              })
          ]
        })
      }),
      e.jsx(de, {
        columns: dt,
        data: Me,
        empty: w ? 'Loading…' : 'No instances',
        onRowClick: (U) => {
          const ie = String(U.id)
          fe(ie, !l.has(ie))
        },
        isRowSelected: (U) => l.has(String(U.id))
      }),
      e.jsx(le, {
        title: 'Create Instance',
        open: O,
        onClose: () => {
          ;($(!1), he())
        },
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => $(!1),
              disabled: z,
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: Xs,
              disabled: z || !F || !C || !N || !H,
              children: 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: F,
                  onChange: (U) => W(U.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              className: 'grid grid-cols-2 gap-3',
              children: [
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Flavor' }),
                    e.jsxs('select', {
                      className: 'input w-full',
                      value: C,
                      onChange: (U) => A(U.target.value),
                      children: [
                        e.jsx('option', { value: '', children: 'Select' }),
                        b.map((U) =>
                          e.jsxs(
                            'option',
                            {
                              value: U.id,
                              children: [U.name, ' • ', U.vcpu, ' vCPU / ', U.memoryGiB, ' GiB']
                            },
                            U.id
                          )
                        )
                      ]
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Image' }),
                    e.jsxs('select', {
                      className: 'input w-full',
                      value: N,
                      onChange: (U) => B(U.target.value),
                      children: [
                        e.jsx('option', { value: '', children: 'Select' }),
                        S.map((U) =>
                          e.jsxs(
                            'option',
                            { value: U.id, children: [U.name, ' • ', U.sizeGiB, ' GiB'] },
                            U.id
                          )
                        )
                      ]
                    })
                  ]
                })
              ]
            }),
            e.jsxs('div', {
              className: 'grid grid-cols-3 gap-3',
              children: [
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Root Disk (GiB)' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      type: 'number',
                      min: ae,
                      placeholder: `${ae}+`,
                      value: J,
                      onChange: (U) => Q(U.target.value)
                    }),
                    e.jsxs('p', {
                      className: 'text-xs text-muted mt-1',
                      children: ['Minimum ', ae, ' GiB based on flavor/image']
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Network' }),
                    e.jsxs('select', {
                      className: 'input w-full',
                      value: H,
                      onChange: (U) => {
                        if (U.target.value === '__create__') {
                          Le(!0)
                          return
                        }
                        E(U.target.value)
                      },
                      children: [
                        e.jsx('option', { value: '', children: 'Select' }),
                        M.map((U) =>
                          e.jsxs(
                            'option',
                            { value: U.id, children: [U.name, U.cidr ? ` • ${U.cidr}` : ''] },
                            U.id
                          )
                        ),
                        e.jsx('option', { value: '__create__', children: '+ Create new network…' })
                      ]
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'SSH Key' }),
                    e.jsxs('select', {
                      className: 'input w-full',
                      value: G,
                      onChange: (U) => {
                        if (U.target.value === '__create__') {
                          se(!0)
                          return
                        }
                        q(U.target.value)
                      },
                      children: [
                        e.jsx('option', { value: '', children: 'None' }),
                        j.map((U) => e.jsx('option', { value: U.id, children: U.name }, U.id)),
                        e.jsx('option', { value: '__create__', children: '+ Add new SSH key…' })
                      ]
                    })
                  ]
                })
              ]
            }),
            e.jsxs('div', {
              className: 'flex items-center gap-2',
              children: [
                e.jsx('input', {
                  type: 'checkbox',
                  id: 'enableTPM',
                  checked: Z,
                  onChange: (U) => Y(U.target.checked)
                }),
                e.jsx('label', {
                  htmlFor: 'enableTPM',
                  className: 'label cursor-pointer',
                  children: 'Enable TPM (Trusted Platform Module)'
                })
              ]
            })
          ]
        })
      }),
      e.jsx(le, {
        title: 'Add SSH Key',
        open: V,
        onClose: () => se(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => se(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: async () => {
                if (m && ne && re) {
                  const U = await Lt(m, { name: ne, public_key: re })
                  ;(D((ie) => [...ie, U]), q(U.id), ue(''), Ee(''), se(!1))
                }
              },
              children: 'Add'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: ne,
                  onChange: (U) => ue(U.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Public Key' }),
                e.jsx('textarea', {
                  className: 'input w-full h-28',
                  value: re,
                  onChange: (U) => Ee(U.target.value)
                })
              ]
            })
          ]
        })
      }),
      e.jsx(le, {
        title: 'Create Network',
        open: Fe,
        onClose: () => Le(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => Le(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: async () => {
                if (m && Ae && ve) {
                  const U = await Nt(m, { name: Ae, cidr: ve })
                  ;(P((ie) => [...ie, U]), E(U.id), Be(''), X('10.0.0.0/24'), Le(!1))
                }
              },
              children: 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: Ae,
                  onChange: (U) => Be(U.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'CIDR' }),
                e.jsx('input', {
                  className: 'input w-full',
                  placeholder: '10.0.0.0/24',
                  value: ve,
                  onChange: (U) => X(U.target.value)
                })
              ]
            })
          ]
        })
      }),
      v &&
        e.jsx(Dr, {
          instanceIds: r,
          onComplete: () => {
            ;(_(!1), d([]), he(), me.success('Deletion process completed'))
          },
          onClose: () => {
            ;(_(!1), d([]), he())
          }
        })
    ]
  })
}
function Ar() {
  const { id: m } = xe(),
    y = Ze(),
    g = k.useRef(null),
    [w, T] = k.useState(null),
    [R, I] = k.useState(null)
  return (
    k.useEffect(() => {
      m &&
        jt(m)
          .then((i) => T(i))
          .catch(() => I('Failed to start console'))
    }, [m]),
    e.jsxs('div', {
      className: 'space-y-3',
      children: [
        e.jsx(oe, {
          title: 'Console',
          subtitle: `Instance ${m}`,
          actions: e.jsx('button', {
            className: 'btn-secondary',
            onClick: () => y(-1),
            children: 'Back'
          })
        }),
        !w &&
          !R &&
          e.jsx('div', { className: 'p-4 text-gray-400', children: 'Requesting console…' }),
        R && e.jsx('div', { className: 'p-4 text-red-400', children: R }),
        w &&
          e.jsx('div', {
            className: 'border border-oxide-800 rounded-lg overflow-hidden',
            children: e.jsx('iframe', {
              ref: g,
              title: 'console',
              className: 'w-full h-[70vh]',
              src: `/novnc.html?path=${encodeURIComponent(w)}`
            })
          })
      ]
    })
  )
}
function Br() {
  const { projectId: m } = xe(),
    { snapshots: y, addSnapshot: g, setSnapshots: w } = Ce(),
    [T, R] = k.useState(!1)
  k.useEffect(() => {
    let h = !0
    return (
      R(!0),
      ds()
        .then((f) => {
          h && w(f)
        })
        .finally(() => h && R(!1)),
      () => {
        h = !1
      }
    )
  }, [])
  const I = k.useMemo(() => y.filter((h) => h.projectId === m && h.kind === 'vm'), [y, m]),
    i = [
      { key: 'id', header: 'ID' },
      { key: 'sourceId', header: 'VM' },
      { key: 'status', header: 'Status' }
    ],
    [o, l] = k.useState(!1),
    [u, a] = k.useState('')
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'VM Snapshots',
        subtitle: 'Snapshots of instances',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => l(!0),
          children: 'Create Snapshot'
        })
      }),
      e.jsx(de, { columns: i, data: I, empty: T ? 'Loading…' : 'No snapshots' }),
      e.jsx(le, {
        title: 'Create VM Snapshot',
        open: o,
        onClose: () => l(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => l(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: () => {
                m && u && (g({ projectId: m, sourceId: u, kind: 'vm' }), a(''), l(!1))
              },
              children: 'Create'
            })
          ]
        }),
        children: e.jsx('div', {
          className: 'space-y-3',
          children: e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'VM ID' }),
              e.jsx('input', {
                className: 'input w-full',
                value: u,
                onChange: (h) => a(h.target.value)
              })
            ]
          })
        })
      })
    ]
  })
}
function Ir() {
  const { projectId: m } = xe(),
    [y, g] = k.useState([]),
    [w, T] = k.useState([]),
    [R, I] = k.useState(!1),
    [i, o] = k.useState('all'),
    [l, u] = k.useState(''),
    [a, h] = k.useState(!1),
    [f, x] = k.useState(''),
    [c, t] = k.useState('1'),
    [n, s] = k.useState('512'),
    [r, d] = k.useState('10'),
    [v, _] = k.useState(''),
    [b, p] = k.useState(''),
    [S, L] = k.useState('microvm'),
    [M, P] = k.useState(!1),
    j = k.useCallback(async () => {
      I(!0)
      try {
        const N = m ? { headers: { 'X-Project-ID': m } } : void 0,
          { data: B } = await ee.get('/v1/firecracker', N)
        g(B.instances || [])
      } catch {
        g([])
      } finally {
        I(!1)
      }
    }, [m]),
    D = k.useCallback(async () => {
      try {
        const N = m ? { headers: { 'X-Project-ID': m } } : void 0,
          { data: B } = await ee.get('/v1/images', N)
        T(B.images || [])
      } catch {
        T([])
      }
    }, [m])
  k.useEffect(() => {
    ;(j(), D())
  }, [j, D])
  const O = k.useMemo(() => {
      let N = y
      if (
        (i === 'running' && (N = N.filter((B) => B.power_state === 'running')),
        i === 'stopped' && (N = N.filter((B) => B.power_state === 'shutdown')),
        l)
      ) {
        const B = l.toLowerCase()
        N = N.filter((z) => z.name.toLowerCase().includes(B) || z.uuid.toLowerCase().includes(B))
      }
      return N
    }, [y, i, l]),
    $ = async () => {
      if (!f.trim()) {
        me.error('Name is required')
        return
      }
      if (!v) {
        me.error('Image selection is required')
        return
      }
      if (parseInt(c) < 1 || parseInt(n) < 128) {
        me.error('Invalid vCPUs or memory configuration')
        return
      }
      P(!0)
      try {
        const N = m ? { headers: { 'X-Project-ID': m } } : void 0
        ;(await ee.post(
          '/v1/firecracker',
          {
            name: f,
            vcpus: parseInt(c),
            memory_mb: parseInt(n),
            disk_gb: parseInt(r) || 10,
            image_id: parseInt(v),
            kernel_path: b || void 0,
            type: S
          },
          N
        ),
          me.success(`Firecracker ${S} "${f}" is being created`),
          h(!1),
          x(''),
          t('1'),
          s('512'),
          d('10'),
          _(''),
          p(''),
          L('microvm'),
          j())
      } catch (N) {
        const B = N instanceof Error ? N.message : 'Failed to create Firecracker instance'
        me.error(B)
      } finally {
        P(!1)
      }
    },
    F = k.useCallback(
      async (N) => {
        try {
          ;(await ee.post(`/v1/firecracker/${N}/start`),
            me.success('Firecracker instance started'),
            j())
        } catch (B) {
          const z = B instanceof Error ? B.message : 'Failed to start instance'
          me.error(z)
        }
      },
      [j]
    ),
    W = k.useCallback(
      async (N) => {
        try {
          ;(await ee.post(`/v1/firecracker/${N}/stop`),
            me.success('Firecracker instance stopped'),
            j())
        } catch (B) {
          const z = B instanceof Error ? B.message : 'Failed to stop instance'
          me.error(z)
        }
      },
      [j]
    ),
    C = k.useCallback(
      async (N) => {
        if (confirm('Are you sure you want to delete this Firecracker instance?'))
          try {
            ;(await ee.delete(`/v1/firecracker/${N}`),
              me.success('Firecracker instance deleted'),
              j())
          } catch (B) {
            const z = B instanceof Error ? B.message : 'Failed to delete instance'
            me.error(z)
          }
      },
      [j]
    ),
    A = k.useMemo(
      () => [
        {
          key: 'name',
          header: 'Name',
          render: (N) => e.jsx('span', { className: 'text-primary-400', children: N.name })
        },
        {
          key: 'type',
          header: 'Type',
          render: (N) =>
            e.jsx(pe, {
              variant: N.type === 'microvm' ? 'info' : 'warning',
              children: N.type === 'microvm' ? 'MicroVM' : 'Function'
            })
        },
        {
          key: 'status',
          header: 'State',
          render: (N) =>
            N.status === 'building'
              ? e.jsx(pe, { variant: 'warning', children: 'building' })
              : N.status === 'error'
                ? e.jsx(pe, { variant: 'danger', children: 'error' })
                : N.status === 'active' && N.power_state === 'running'
                  ? e.jsx(pe, { variant: 'success', children: 'running' })
                  : e.jsx(pe, { children: 'stopped' })
        },
        { key: 'vm_id', header: 'Internal name', render: (N) => N.vm_id || N.uuid },
        { key: 'vcpus', header: 'vCPUs', render: (N) => N.vcpus },
        { key: 'memory_mb', header: 'Memory', render: (N) => `${N.memory_mb} MB` },
        { key: 'disk_gb', header: 'Disk', render: (N) => `${N.disk_gb || '-'} GB` },
        {
          key: 'actions',
          header: 'Actions',
          render: (N) =>
            e.jsxs('div', {
              className: 'flex gap-1',
              children: [
                e.jsx('button', {
                  onClick: (B) => {
                    ;(B.stopPropagation(), F(N.id))
                  },
                  disabled: N.power_state === 'running',
                  className:
                    'icon-btn text-green-400 disabled:opacity-30 disabled:cursor-not-allowed',
                  title: 'Start',
                  children: e.jsx('svg', {
                    width: '18',
                    height: '18',
                    viewBox: '0 0 24 24',
                    fill: 'currentColor',
                    children: e.jsx('path', { d: 'M8 5v14l11-7z' })
                  })
                }),
                e.jsx('button', {
                  onClick: (B) => {
                    ;(B.stopPropagation(), W(N.id))
                  },
                  disabled: N.power_state === 'shutdown',
                  className:
                    'icon-btn text-yellow-400 disabled:opacity-30 disabled:cursor-not-allowed',
                  title: 'Stop',
                  children: e.jsx('svg', {
                    width: '18',
                    height: '18',
                    viewBox: '0 0 24 24',
                    fill: 'currentColor',
                    children: e.jsx('path', { d: 'M6 6h12v12H6z' })
                  })
                }),
                e.jsx('button', {
                  onClick: (B) => {
                    ;(B.stopPropagation(), C(N.id))
                  },
                  className: 'icon-btn text-rose-400',
                  title: 'Delete',
                  children: e.jsx('svg', {
                    width: '18',
                    height: '18',
                    viewBox: '0 0 24 24',
                    fill: 'currentColor',
                    children: e.jsx('path', { d: 'M6 7h12l-1 14H7L6 7zm3-3h6l1 2H8l1-2z' })
                  })
                })
              ]
            })
        }
      ],
      [F, W, C]
    )
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, { title: 'Firecracker', subtitle: 'Lightweight microVMs and function containers' }),
      e.jsx(Ne, {
        placeholder: 'Search name/uuid',
        onSearch: u,
        children: e.jsxs('div', {
          className: 'flex items-center gap-2',
          children: [
            e.jsx('button', {
              className: 'btn-secondary h-9',
              onClick: j,
              disabled: R,
              children: 'Refresh'
            }),
            e.jsxs('select', {
              className: 'input h-9',
              value: i,
              onChange: (N) => o(N.target.value),
              children: [
                e.jsx('option', { value: 'all', children: 'All' }),
                e.jsx('option', { value: 'running', children: 'Running' }),
                e.jsx('option', { value: 'stopped', children: 'Stopped' })
              ]
            }),
            e.jsx('button', {
              className: 'btn-primary h-9 w-40',
              onClick: () => h(!0),
              children: 'Add MicroVM'
            })
          ]
        })
      }),
      e.jsx(de, { data: O, columns: A, empty: 'No Firecracker instances' }),
      e.jsx(le, {
        open: a,
        onClose: () => h(!1),
        title: 'Create Firecracker Instance',
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => h(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: $,
              disabled: M,
              children: M ? 'Creating...' : 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: f,
                  onChange: (N) => x(N.target.value),
                  placeholder: 'my-microvm'
                })
              ]
            }),
            e.jsxs('div', {
              className: 'grid grid-cols-2 gap-3',
              children: [
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Type' }),
                    e.jsxs('select', {
                      className: 'input w-full',
                      value: S,
                      onChange: (N) => L(N.target.value),
                      children: [
                        e.jsx('option', { value: 'microvm', children: 'MicroVM' }),
                        e.jsx('option', { value: 'function', children: 'Function Container' })
                      ]
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'vCPUs' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      type: 'number',
                      min: '1',
                      max: '32',
                      value: c,
                      onChange: (N) => t(N.target.value)
                    })
                  ]
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Memory (MB)' }),
                e.jsx('input', {
                  className: 'input w-full',
                  type: 'number',
                  min: '128',
                  max: '32768',
                  step: '128',
                  value: n,
                  onChange: (N) => s(N.target.value)
                }),
                e.jsx('p', {
                  className: 'text-xs text-muted mt-1',
                  children: 'Minimum 128 MB recommended'
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Image' }),
                e.jsxs('select', {
                  className: 'input w-full',
                  value: v,
                  onChange: (N) => _(N.target.value),
                  children: [
                    e.jsx('option', { value: '', children: 'Select an image...' }),
                    w.map((N) =>
                      e.jsxs(
                        'option',
                        {
                          value: N.id,
                          children: [N.name, ' (', N.os_type, ', ', N.size_gb, 'GB)']
                        },
                        N.id
                      )
                    )
                  ]
                }),
                e.jsx('p', {
                  className: 'text-xs text-muted mt-1',
                  children: 'Boot image from Ceph storage'
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Disk Size (GB)' }),
                e.jsx('input', {
                  className: 'input w-full',
                  type: 'number',
                  min: '1',
                  max: '500',
                  value: r,
                  onChange: (N) => d(N.target.value)
                }),
                e.jsx('p', {
                  className: 'text-xs text-muted mt-1',
                  children: 'Root disk size (default: 10GB)'
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Kernel Path (optional)' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: b,
                  onChange: (N) => p(N.target.value),
                  placeholder: 'Leave empty to use default kernel'
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Mr() {
  const { projectId: m } = xe(),
    { clusters: y, addCluster: g } = Ce(),
    w = k.useMemo(() => y.filter((a) => a.projectId === m), [y, m]),
    T = [
      { key: 'name', header: 'Name' },
      { key: 'version', header: 'Version' },
      { key: 'status', header: 'Status' }
    ],
    [R, I] = k.useState(!1),
    [i, o] = k.useState(''),
    [l, u] = k.useState('1.29')
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Kubernetes',
        subtitle: 'Clusters',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => I(!0),
          children: 'Create Cluster'
        })
      }),
      e.jsx(de, { columns: T, data: w, empty: 'No clusters' }),
      e.jsx(le, {
        title: 'Create Cluster',
        open: R,
        onClose: () => I(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => I(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: () => {
                m && i && l && (g({ projectId: m, name: i, version: l }), o(''), u('1.29'), I(!1))
              },
              children: 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: i,
                  onChange: (a) => o(a.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Version' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: l,
                  onChange: (a) => u(a.target.value)
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Pr() {
  const { flavors: m, setFlavors: y } = Ce(),
    [g, w] = k.useState(!1)
  k.useEffect(() => {
    let t = !0
    return (
      w(!0),
      kt()
        .then((n) => {
          t && y(n)
        })
        .finally(() => t && w(!1)),
      () => {
        t = !1
      }
    )
  }, [])
  const T = [
      { key: 'name', header: 'Name' },
      { key: 'vcpu', header: 'vCPU' },
      { key: 'memoryGiB', header: 'Memory (GiB)' },
      {
        key: 'id',
        header: '',
        className: 'w-10 text-right',
        render: (t) =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx('button', {
              className: 'text-red-400 hover:underline',
              onClick: async () => {
                try {
                  ;(await os(t.id), y(m.filter((n) => n.id !== t.id)))
                } catch (n) {
                  if (wt.isAxiosError(n) && n.response?.status === 409)
                    alert('Flavor is in use and cannot be deleted')
                  else {
                    const s = n instanceof Error ? n.message : 'unknown error'
                    alert('Delete failed: ' + s)
                  }
                }
              },
              children: 'Delete'
            })
          })
      }
    ],
    [R, I] = k.useState(!1),
    [i, o] = k.useState(''),
    [l, u] = k.useState(''),
    [a, h] = k.useState(''),
    [f, x] = k.useState(''),
    c = async () => {
      if (!i || !l || !a) return
      const t = { name: i, vcpus: Number(l), ram: Number(a) * 1024, disk: f ? Number(f) : void 0 },
        n = await as(t)
      ;(y([...m, n]), o(''), u(''), h(''), x(''), I(!1))
    }
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Flavors',
        subtitle: 'Instance sizes',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => I(!0),
          children: 'Create Flavor'
        })
      }),
      e.jsx(de, { columns: T, data: m, empty: g ? 'Loading…' : 'No flavors' }),
      e.jsx(le, {
        title: 'Create Flavor',
        open: R,
        onClose: () => I(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => I(!1),
              children: 'Cancel'
            }),
            e.jsx('button', { className: 'btn-primary', onClick: c, children: 'Save' })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: i,
                  onChange: (t) => o(t.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              className: 'grid grid-cols-3 gap-3',
              children: [
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'vCPU' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      type: 'number',
                      value: l,
                      onChange: (t) => u(t.target.value ? Number(t.target.value) : '')
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Memory (GiB)' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      type: 'number',
                      value: a,
                      onChange: (t) => h(t.target.value ? Number(t.target.value) : '')
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Disk (GB)' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      type: 'number',
                      value: f,
                      onChange: (t) => x(t.target.value ? Number(t.target.value) : '')
                    })
                  ]
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Tr() {
  const { projectId: m } = xe(),
    [y, g] = k.useState([]),
    w = (a, h = 24, f = 12) =>
      a ? (a.length > h + f + 3 ? `${a.slice(0, h)}…${a.slice(-f)}` : a) : ''
  k.useEffect(() => {
    let a = !0
    return (
      rt(m).then((h) => {
        a && g(h.map((f) => ({ id: f.id, name: f.name, publicKey: f.public_key })))
      }),
      () => {
        a = !1
      }
    )
  }, [m])
  const T = [
      { key: 'name', header: 'Name' },
      {
        key: 'publicKey',
        header: 'Public Key',
        className: 'max-w-[420px]',
        render: (a) =>
          e.jsxs('div', {
            className: 'flex items-center gap-2 min-w-0',
            children: [
              e.jsx('span', {
                className: 'font-mono text-xs truncate min-w-0',
                title: a.publicKey,
                children: w(a.publicKey, 28, 16)
              }),
              e.jsx('button', {
                className: 'text-blue-400 hover:underline text-xs shrink-0',
                onClick: () => navigator.clipboard.writeText(a.publicKey),
                title: 'Copy full key',
                children: 'Copy'
              })
            ]
          })
      },
      {
        key: 'id',
        header: '',
        className: 'w-10 text-right',
        render: (a) =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx('button', {
              className: 'text-red-400 hover:underline',
              onClick: async () => {
                m && (await Ns(m, a.id), g((h) => h.filter((f) => f.id !== a.id)))
              },
              children: 'Delete'
            })
          })
      }
    ],
    [R, I] = k.useState(!1),
    [i, o] = k.useState(''),
    [l, u] = k.useState('')
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'SSH Keypairs',
        subtitle: 'Project SSH keys',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => I(!0),
          children: 'Add Key'
        })
      }),
      e.jsx(de, { columns: T, data: y, empty: 'No keys' }),
      e.jsx(le, {
        title: 'Add SSH Key',
        open: R,
        onClose: () => I(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => I(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: async () => {
                if (m && i && l) {
                  const a = await Lt(m, { name: i, public_key: l })
                  ;(g((h) => [...h, { id: a.id, name: a.name, publicKey: a.public_key }]),
                    o(''),
                    u(''),
                    I(!1))
                }
              },
              children: 'Add'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: i,
                  onChange: (a) => o(a.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Public Key' }),
                e.jsx('textarea', {
                  className: 'input w-full h-28',
                  value: l,
                  onChange: (a) => u(a.target.value)
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Or() {
  return e.jsx('div', {
    className: 'space-y-4',
    children: e.jsxs(Ve, {
      children: [
        e.jsx(te, { path: 'instances', element: e.jsx(Ot, {}) }),
        e.jsx(te, { path: 'instances/:id/console', element: e.jsx(Ar, {}) }),
        e.jsx(te, { path: 'instances/:id/volumes', element: e.jsx(Hr, {}) }),
        e.jsx(te, { path: 'firecracker', element: e.jsx(Ir, {}) }),
        e.jsx(te, { path: 'flavors', element: e.jsx(Pr, {}) }),
        e.jsx(te, { path: 'vm-snapshots', element: e.jsx(Br, {}) }),
        e.jsx(te, { path: 'k8s', element: e.jsx(Mr, {}) }),
        e.jsx(te, { path: 'kms', element: e.jsx(Tr, {}) }),
        e.jsx(te, { path: '*', element: e.jsx(Ot, {}) })
      ]
    })
  })
}
function Hr() {
  const { id: m } = xe(),
    [y, g] = k.useState([]),
    [w, T] = k.useState(!1),
    [R, I] = k.useState(''),
    [i, o] = k.useState(!1),
    l = async () => {
      m && g(await gt(m))
    }
  k.useEffect(() => {
    ;(async () => m && g(await gt(m)))()
  }, [m])
  const u = [
    { key: 'name', header: 'Name' },
    { key: 'sizeGiB', header: 'Size (GiB)' },
    {
      key: 'status',
      header: 'Status',
      render: (a) =>
        a.status === 'in-use'
          ? e.jsx(pe, { variant: 'success', children: 'in-use' })
          : e.jsx(pe, { children: a.status })
    },
    { key: 'rbd', header: 'RBD' },
    {
      key: 'id',
      header: '',
      className: 'w-10 text-right',
      render: (a) =>
        Number(a.id) > 0
          ? e.jsx('div', {
              className: 'flex justify-end',
              children: e.jsx('button', {
                className: 'text-red-400 hover:underline',
                onClick: async () => {
                  m && (await _s(m, a.id), await l())
                },
                children: 'Detach'
              })
            })
          : null
    }
  ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Instance Volumes',
        subtitle: `Volumes for instance ${m}`,
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => T(!0),
          children: 'Attach Volume'
        })
      }),
      e.jsx(de, { columns: u, data: y, empty: 'No volumes' }),
      e.jsx(le, {
        title: 'Attach Volume',
        open: w,
        onClose: () => T(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => T(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: i || !R,
              onClick: async () => {
                if (m && R)
                  try {
                    ;(o(!0), await ms(m, R), I(''), T(!1), await l())
                  } finally {
                    o(!1)
                  }
              },
              children: 'Attach'
            })
          ]
        }),
        children: e.jsx('div', {
          className: 'space-y-3',
          children: e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'Volume ID' }),
              e.jsx('input', {
                className: 'input w-full',
                value: R,
                onChange: (a) => I(a.target.value),
                placeholder: 'Enter available volume ID'
              }),
              e.jsx('p', {
                className: 'text-xs text-muted mt-1',
                children:
                  'Attach an existing available volume by its ID. You can find IDs on the Storage → Volumes page.'
              })
            ]
          })
        })
      })
    ]
  })
}
function Fr() {
  return e.jsx('div', {
    className: 'space-y-4',
    children: e.jsxs(Ve, {
      children: [
        e.jsx(te, { path: 'volumes', element: e.jsx(Ht, {}) }),
        e.jsx(te, { path: 'snapshots', element: e.jsx($r, {}) }),
        e.jsx(te, { path: 'backups', element: e.jsx(Wr, {}) }),
        e.jsx(te, { path: '*', element: e.jsx(Ht, {}) })
      ]
    })
  })
}
function Ht() {
  const { projectId: m } = xe(),
    [y, g] = k.useState([]),
    [w, T] = k.useState(!1),
    R = async () => {
      T(!0)
      try {
        g(await _t(m))
      } finally {
        T(!1)
      }
    }
  k.useEffect(() => {
    let S = !0
    return (
      T(!0),
      _t(m)
        .then((L) => {
          S && g(L)
        })
        .finally(() => {
          S && T(!1)
        }),
      () => {
        S = !1
      }
    )
  }, [m])
  const I = [
      {
        key: 'name',
        header: 'Name',
        render: (S) =>
          e.jsxs('div', {
            className: 'flex items-center gap-2',
            children: [
              e.jsx('span', { children: S.name }),
              S.id === '0' && e.jsx(pe, { children: 'Root Disk' })
            ]
          })
      },
      { key: 'sizeGiB', header: 'Size (GiB)' },
      {
        key: 'status',
        header: 'Status',
        render: (S) =>
          S.status === 'in-use'
            ? e.jsx(pe, { variant: 'success', children: 'in-use' })
            : e.jsx(pe, { children: S.status })
      },
      { key: 'rbd', header: 'RBD', render: (S) => S.rbd ?? '-' },
      {
        key: 'actions',
        header: 'Actions',
        sortable: !1,
        render: (S) =>
          e.jsxs('div', {
            className: 'flex gap-2',
            children: [
              e.jsx('button', {
                className: 'text-blue-400 hover:text-blue-300 text-sm',
                onClick: () => {
                  ;(s(S.id), d(S.sizeGiB), _(S.sizeGiB), t(!0))
                },
                children: 'Resize'
              }),
              e.jsx('button', {
                className: `text-sm ${S.status === 'in-use' ? 'text-gray-500 cursor-not-allowed' : 'text-red-400 hover:text-red-300'}`,
                disabled: S.status === 'in-use',
                title:
                  S.status === 'in-use'
                    ? 'Volume is in use by an instance; detach or delete the instance first'
                    : 'Delete volume',
                onClick: async () => {
                  if (S.status !== 'in-use' && confirm(`Delete volume "${S.name}"?`))
                    try {
                      ;(await fs(S.id), await R())
                    } catch (L) {
                      wt.isAxiosError(L) && L.response?.status === 409
                        ? alert(
                            'Volume is in use by an instance; please detach or delete the instance first.'
                          )
                        : alert('Failed to delete volume: ' + L.message)
                    }
                },
                children: 'Delete'
              })
            ]
          })
      }
    ],
    [i, o] = k.useState(!1),
    [l, u] = k.useState(''),
    [a, h] = k.useState(''),
    [f, x] = k.useState(!1),
    [c, t] = k.useState(!1),
    [n, s] = k.useState(''),
    [r, d] = k.useState(0),
    [v, _] = k.useState(0),
    [b, p] = k.useState(!1)
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Volumes',
        subtitle: 'Project volumes',
        actions: e.jsxs('div', {
          className: 'flex gap-2',
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: R,
              disabled: w,
              children: w ? 'Loading...' : 'Refresh'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: () => o(!0),
              children: 'Create Volume'
            })
          ]
        })
      }),
      e.jsx(Ne, { placeholder: 'Search volumes' }),
      e.jsx(de, { columns: I, data: y, empty: 'No volumes' }),
      e.jsx(le, {
        title: 'Create Volume',
        open: i,
        onClose: () => o(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => o(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: f,
              onClick: async () => {
                if (m && l && a) {
                  x(!0)
                  try {
                    ;(await us(m, { name: l, size_gb: Number(a) }), u(''), h(''), o(!1), await R())
                  } catch (S) {
                    alert('Failed to create volume: ' + S.message)
                  } finally {
                    x(!1)
                  }
                }
              },
              children: 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: l,
                  onChange: (S) => u(S.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Size (GiB)' }),
                e.jsx('input', {
                  className: 'input w-full',
                  type: 'number',
                  min: '1',
                  value: a,
                  onChange: (S) => h(S.target.value ? Number(S.target.value) : '')
                })
              ]
            })
          ]
        })
      }),
      e.jsx(le, {
        title: 'Resize Volume',
        open: c,
        onClose: () => t(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => t(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: b || v <= r,
              onClick: async () => {
                if (v > r) {
                  p(!0)
                  try {
                    ;(await ps(n, v), t(!1), await R())
                  } catch (S) {
                    alert('Failed to resize volume: ' + S.message)
                  } finally {
                    p(!1)
                  }
                }
              },
              children: 'Resize'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Current Size (GiB)' }),
                e.jsx('input', {
                  className: 'input w-full',
                  type: 'number',
                  value: r,
                  disabled: !0
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'New Size (GiB)' }),
                e.jsx('input', {
                  className: 'input w-full',
                  type: 'number',
                  min: r + 1,
                  value: v,
                  onChange: (S) => _(S.target.value ? Number(S.target.value) : r)
                }),
                e.jsx('p', {
                  className: 'text-xs text-gray-400 mt-1',
                  children: 'New size must be larger than current size'
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function $r() {
  const { projectId: m } = xe(),
    [y, g] = k.useState([])
  k.useEffect(() => {
    ;(async () => g(await tt(m)))()
  }, [m])
  const w = [
      { key: 'name', header: 'Name' },
      { key: 'volumeId', header: 'Volume' },
      { key: 'status', header: 'Status' },
      { key: 'backup', header: 'Backup' }
    ],
    [T, R] = k.useState(!1),
    [I, i] = k.useState(''),
    [o, l] = k.useState(''),
    [u, a] = k.useState(!1)
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Snapshots',
        subtitle: 'Volume snapshots',
        actions: e.jsxs('div', {
          className: 'flex gap-2',
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: async () => {
                g(await tt(m))
              },
              children: 'Refresh'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: () => R(!0),
              children: 'Create Snapshot'
            })
          ]
        })
      }),
      e.jsx(de, { columns: w, data: y, empty: 'No snapshots' }),
      e.jsx(le, {
        title: 'Create Volume Snapshot',
        open: T,
        onClose: () => R(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => R(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: u,
              onClick: async () => {
                m &&
                  I &&
                  (a(!0),
                  await gs(m, { name: o || `snap-${Date.now()}`, volume_id: Number(I) }),
                  l(''),
                  i(''),
                  R(!1),
                  g(await tt(m)),
                  a(!1))
              },
              children: 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: o,
                  onChange: (h) => l(h.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Volume ID' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: I,
                  onChange: (h) => i(h.target.value)
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Wr() {
  const { projectId: m } = xe(),
    [y, g] = k.useState([])
  k.useEffect(() => {
    ;(async () => g(await vt(m, { resource: 'snapshot' })))()
  }, [m])
  const w = [
    { key: 'id', header: 'ID' },
    { key: 'resource', header: 'Resource' },
    { key: 'resource_id', header: 'RID' },
    { key: 'action', header: 'Action' },
    { key: 'status', header: 'Status' },
    { key: 'message', header: 'Message' }
  ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Backups',
        subtitle: 'Recent backup operations (audit)',
        actions: e.jsx('button', {
          className: 'btn-secondary',
          onClick: async () => g(await vt(m, { resource: 'snapshot' })),
          children: 'Refresh'
        })
      }),
      e.jsx(de, { columns: w, data: y, empty: 'No records' })
    ]
  })
}
const Ur = [
  { id: '1', name: 'admin', role: 'Owner', status: 'active' },
  { id: '2', name: 'demo', role: 'Member', status: 'active' }
]
function zr() {
  const m = [
    {
      key: 'name',
      header: 'Name',
      render: (y) =>
        e.jsx(Oe, {
          className: 'text-oxide-300 hover:underline',
          to: `/project/${encodeURIComponent(y.id)}`,
          children: y.name
        })
    },
    { key: 'id', header: 'ID', className: 'text-gray-400' },
    { key: 'role', header: 'Role' },
    {
      key: 'status',
      header: 'Status',
      render: (y) =>
        e.jsx(pe, { variant: y.status === 'active' ? 'success' : 'warning', children: y.status })
    },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: () =>
        e.jsx('div', {
          className: 'flex justify-end',
          children: e.jsx(Ge, {
            actions: [
              { label: 'View', onClick: () => {} },
              { label: 'Edit', onClick: () => {} },
              { label: 'Delete', onClick: () => {}, danger: !0 }
            ]
          })
        })
    }
  ]
  return e.jsxs('div', {
    className: 'space-y-4',
    children: [
      e.jsx(oe, {
        title: 'Projects',
        subtitle: 'Manage console projects and access',
        actions: e.jsx('button', { className: 'btn-primary', children: 'Create Project' })
      }),
      e.jsx(Ne, { placeholder: 'Search projects' }),
      e.jsx(de, { columns: m, data: Ur, empty: 'No projects' })
    ]
  })
}
function Vr() {
  const { notices: m, markNotice: y } = Ce(),
    g = [
      { key: 'time', header: 'Time', className: 'text-gray-400' },
      { key: 'resource', header: 'Resource' },
      { key: 'type', header: 'Type' },
      {
        key: 'status',
        header: 'Status',
        render: (w) =>
          e.jsx(pe, { variant: w.status === 'unread' ? 'info' : 'default', children: w.status })
      },
      {
        key: 'actions',
        header: '',
        className: 'w-10 text-right',
        render: (w) =>
          e.jsx('div', {
            className: 'flex justify-end',
            children: e.jsx(Ge, {
              actions: [
                w.status === 'unread'
                  ? { label: 'Mark as read', onClick: () => y(w.id, 'read') }
                  : { label: 'Mark as unread', onClick: () => y(w.id, 'unread') }
              ]
            })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-4',
    children: [
      e.jsx(oe, { title: 'Notifications', subtitle: 'System and resource events' }),
      e.jsx(de, { columns: g, data: m, empty: 'No notifications' })
    ]
  })
}
function Gr() {
  const [m, y] = k.useState(''),
    [g, w] = k.useState(''),
    [T, R] = k.useState(!1),
    I = Ze(),
    i = at(),
    o = lt((_) => _.login),
    l = Te((_) => _.setActiveProjectId),
    u = Te((_) => _.setProjectContext),
    a = He((_) => _.logoDataUrl),
    h = He((_) => _.idpProvider),
    f = He((_) => _.idpIssuer),
    x = He((_) => _.idpClientId),
    c = He((_) => _.idpRedirectUrl),
    t = h === 'OIDC' && !!f && !!x,
    n = k.useMemo(() => c || `${window.location.origin}/auth/oidc/callback`, [c]),
    s = k.useMemo(() => (f ? `${f.replace(/\/$/, '')}/authorize` : ''), [f])
  function r() {
    if (!t) return
    const _ = Math.random().toString(36).slice(2)
    try {
      sessionStorage.setItem('oidc_state', _)
    } catch {}
    const b = new URL(s)
    ;(b.searchParams.set('client_id', x),
      b.searchParams.set('redirect_uri', n),
      b.searchParams.set('response_type', 'code'),
      b.searchParams.set('scope', 'openid profile email'),
      b.searchParams.set('state', _),
      (window.location.href = b.toString()))
  }
  const d = k.useRef(!1)
  k.useEffect(() => {
    if (!d.current) {
      d.current = !0
      try {
        ;(l(null), u(!1))
      } catch {}
    }
  }, [l, u])
  const v = async (_) => {
    ;(_.preventDefault(), R(!0))
    const b = (p) => {
      console.log(p)
      try {
        const S = JSON.parse(localStorage.getItem('debug_logs') || '[]')
        ;(S.push({ time: new Date().toISOString(), msg: p }),
          S.length > 50 && S.shift(),
          localStorage.setItem('debug_logs', JSON.stringify(S)))
      } catch {}
    }
    try {
      if (!m || !g) return
      b(`[Login] Attempting login for user: ${m}`)
      const S = (await ns(m, g)).access_token
      if (S) {
        ;(b('[Login] Login successful, token received'),
          o(S),
          await new Promise((j) => setTimeout(j, 100)))
        const L = localStorage.getItem('auth')
        ;(b(`[Login] Token saved to localStorage: ${L ? 'Yes' : 'No'}`),
          (!L || !L.includes(S)) &&
            (b('[Login] Warning: Token may not have been saved correctly'),
            alert('Warning: Token may not have been saved correctly')))
        const P = i.state?.from?.pathname || '/projects'
        ;(b(`[Login] Navigating to: ${P}`), I(P, { replace: !0 }))
      }
    } catch (p) {
      ;(b(`[Login] Login failed: ${p}`), alert('Login failed. Please check your credentials.'))
    } finally {
      R(!1)
    }
  }
  return e.jsx('div', {
    className: 'min-h-screen grid place-items-center bg-oxide-950 px-4',
    children: e.jsxs('form', {
      onSubmit: v,
      className:
        'w-full max-w-sm p-6 rounded-lg border border-oxide-700 bg-oxide-800 shadow-card space-y-4 text-gray-100',
      children: [
        e.jsxs('div', {
          className: 'flex items-center gap-2',
          children: [
            a
              ? e.jsx('img', { src: a, alt: 'logo', className: 'h-6 w-6 rounded object-contain' })
              : e.jsx('img', {
                  src: '/logo-42.svg',
                  alt: 'logo',
                  className: 'h-6 w-6 rounded object-contain'
                }),
            e.jsx('h1', { className: 'text-xl font-semibold', children: 'Sign in to VC Console' })
          ]
        }),
        e.jsx('p', {
          className: 'text-sm text-gray-400',
          children: 'Use your account to access the console.'
        }),
        t &&
          e.jsxs('div', {
            className: 'space-y-2',
            children: [
              e.jsx('button', {
                type: 'button',
                className:
                  'w-full h-9 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 text-sm',
                onClick: r,
                children: 'Continue with OpenID Connect'
              }),
              e.jsxs('div', {
                className: 'flex items-center gap-2 text-xs text-gray-500',
                children: [
                  e.jsx('div', { className: 'h-px flex-1 bg-oxide-800' }),
                  e.jsx('span', { children: 'or' }),
                  e.jsx('div', { className: 'h-px flex-1 bg-oxide-800' })
                ]
              })
            ]
          }),
        e.jsxs('div', {
          className: 'space-y-2',
          children: [
            e.jsx('label', {
              className: 'label text-gray-300',
              htmlFor: 'username',
              children: 'Username'
            }),
            e.jsx('input', {
              id: 'username',
              className:
                'input w-full rounded-md bg-oxide-900 border border-oxide-700 px-3 py-2 text-sm text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-oxide-600',
              value: m,
              onChange: (_) => y(_.target.value)
            })
          ]
        }),
        e.jsxs('div', {
          className: 'space-y-2',
          children: [
            e.jsx('label', {
              className: 'label text-gray-300',
              htmlFor: 'password',
              children: 'Password'
            }),
            e.jsx('input', {
              id: 'password',
              type: 'password',
              className:
                'input w-full rounded-md bg-oxide-900 border border-oxide-700 px-3 py-2 text-sm text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-oxide-600',
              value: g,
              onChange: (_) => w(_.target.value)
            })
          ]
        }),
        e.jsx('button', {
          type: 'submit',
          className:
            'btn-primary w-full inline-flex items-center justify-center rounded-md h-9 bg-oxide-600 hover:bg-oxide-500 text-white disabled:opacity-50',
          disabled: T,
          children: T ? 'Signing in…' : 'Sign in'
        })
      ]
    })
  })
}
function Kr() {
  const [m] = Js(),
    y = Ze(),
    g = lt((w) => w.login)
  return (
    k.useEffect(() => {
      const w = m.get('code'),
        T = m.get('state'),
        R = sessionStorage.getItem('oidc_state')
      if (!w || !T || !R || T !== R) {
        y('/login', { replace: !0 })
        return
      }
      ;(sessionStorage.removeItem('oidc_state'), g('oidc-token'), y('/projects', { replace: !0 }))
    }, [m, y, g]),
    e.jsx('div', {
      className: 'min-h-screen grid place-items-center bg-oxide-950 text-gray-300',
      children: e.jsx('div', { className: 'p-4', children: 'Completing sign-in…' })
    })
  )
}
const ft = [
  {
    name: 'Identity',
    description: 'Authentication, authorization, and user management',
    endpoints: [
      { method: 'POST', path: '/api/v1/auth/login', description: 'Authenticate user' },
      { method: 'POST', path: '/api/v1/auth/refresh', description: 'Refresh access token' },
      { method: 'POST', path: '/api/v1/auth/logout', description: 'Logout user' },
      { method: 'GET', path: '/api/v1/projects', description: 'List projects' },
      { method: 'POST', path: '/api/v1/projects', description: 'Create project' },
      { method: 'DELETE', path: '/api/v1/projects/:id', description: 'Delete project' },
      { method: 'GET', path: '/api/v1/users', description: 'List users' },
      { method: 'POST', path: '/api/v1/users', description: 'Create user' },
      { method: 'GET', path: '/api/v1/users/:id', description: 'Get user details' },
      { method: 'PUT', path: '/api/v1/users/:id', description: 'Update user' },
      { method: 'DELETE', path: '/api/v1/users/:id', description: 'Delete user' },
      { method: 'GET', path: '/api/v1/roles', description: 'List roles' },
      { method: 'POST', path: '/api/v1/roles', description: 'Create role' },
      { method: 'GET', path: '/api/v1/permissions', description: 'List permissions' },
      { method: 'GET', path: '/api/v1/profile', description: 'Get current user profile' }
    ]
  },
  {
    name: 'Compute',
    description: 'Virtual machine and resource management',
    endpoints: [
      { method: 'GET', path: '/api/v1/instances', description: 'List instances' },
      { method: 'POST', path: '/api/v1/instances', description: 'Create instance' },
      { method: 'GET', path: '/api/v1/instances/:id', description: 'Get instance details' },
      { method: 'DELETE', path: '/api/v1/instances/:id', description: 'Delete instance' },
      { method: 'POST', path: '/api/v1/instances/:id/start', description: 'Start instance' },
      { method: 'POST', path: '/api/v1/instances/:id/stop', description: 'Stop instance' },
      { method: 'POST', path: '/api/v1/instances/:id/reboot', description: 'Reboot instance' },
      { method: 'GET', path: '/api/v1/flavors', description: 'List flavors' },
      { method: 'POST', path: '/api/v1/flavors', description: 'Create flavor' },
      { method: 'GET', path: '/api/v1/images', description: 'List images' },
      { method: 'POST', path: '/api/v1/images', description: 'Create image' },
      { method: 'GET', path: '/api/v1/volumes', description: 'List volumes' },
      { method: 'POST', path: '/api/v1/volumes', description: 'Create volume' }
    ]
  },
  {
    name: 'Network',
    description: 'Network and VPC management',
    endpoints: [
      { method: 'GET', path: '/api/v1/networks', description: 'List networks' },
      { method: 'POST', path: '/api/v1/networks', description: 'Create network' },
      { method: 'GET', path: '/api/v1/networks/:id', description: 'Get network details' },
      { method: 'PUT', path: '/api/v1/networks/:id', description: 'Update network' },
      { method: 'DELETE', path: '/api/v1/networks/:id', description: 'Delete network' },
      {
        method: 'POST',
        path: '/api/v1/networks/:id/restart',
        description: 'Restart network services'
      },
      {
        method: 'GET',
        path: '/api/v1/networks/:id/diagnose',
        description: 'Diagnose network issues'
      }
    ]
  },
  {
    name: 'Host',
    description: 'Physical host management',
    endpoints: [
      { method: 'GET', path: '/api/v1/hosts', description: 'List hosts' },
      { method: 'POST', path: '/api/v1/hosts/register', description: 'Register new host' },
      { method: 'GET', path: '/api/v1/hosts/:id', description: 'Get host details' },
      { method: 'POST', path: '/api/v1/hosts/:id/enable', description: 'Enable host' },
      { method: 'POST', path: '/api/v1/hosts/:id/disable', description: 'Disable host' },
      {
        method: 'POST',
        path: '/api/v1/hosts/:id/maintenance',
        description: 'Toggle maintenance mode'
      }
    ]
  },
  {
    name: 'Scheduler',
    description: 'Workload scheduling and node management',
    endpoints: [
      { method: 'POST', path: '/api/v1/schedule', description: 'Schedule workload' },
      { method: 'GET', path: '/api/v1/nodes', description: 'List scheduler nodes' },
      { method: 'POST', path: '/api/v1/nodes/register', description: 'Register scheduler node' }
    ]
  },
  {
    name: 'Metadata',
    description: 'Instance metadata service',
    endpoints: [
      { method: 'GET', path: '/latest/meta-data', description: 'Get instance metadata' },
      { method: 'GET', path: '/latest/user-data', description: 'Get user data' },
      {
        method: 'GET',
        path: '/api/v1/metadata/instances/:id',
        description: 'Get metadata by instance ID'
      }
    ]
  },
  {
    name: 'Events',
    description: 'System event logging',
    endpoints: [
      { method: 'GET', path: '/api/v1/events', description: 'List events' },
      { method: 'GET', path: '/api/v1/events/:id', description: 'Get event details' },
      {
        method: 'GET',
        path: '/api/v1/events/resource/:type/:id',
        description: 'Get resource events'
      }
    ]
  },
  {
    name: 'Quota',
    description: 'Resource quota management',
    endpoints: [
      { method: 'GET', path: '/api/v1/quotas/tenants/:id', description: 'Get tenant quota' },
      { method: 'PUT', path: '/api/v1/quotas/tenants/:id', description: 'Update tenant quota' },
      { method: 'GET', path: '/api/v1/quotas/defaults', description: 'Get default quotas' }
    ]
  },
  {
    name: 'Monitoring',
    description: 'System health and metrics',
    endpoints: [
      { method: 'GET', path: '/health', description: 'System health check' },
      { method: 'GET', path: '/metrics', description: 'Prometheus metrics' },
      { method: 'GET', path: '/api/v1/monitoring/status', description: 'Component status' }
    ]
  }
]
function qr({ method: m }) {
  const y = { GET: 'info', POST: 'success', PUT: 'warning', PATCH: 'warning', DELETE: 'danger' }
  return e.jsx(pe, { variant: y[m], children: m })
}
function Xr() {
  const [m, y] = k.useState(ft[0].name),
    g = ft.find((w) => w.name === m)
  return e.jsxs('div', {
    className: 'space-y-6',
    children: [
      e.jsx(oe, { title: 'API Documentation', subtitle: 'Reference for platform REST APIs' }),
      e.jsxs('div', {
        className: 'grid grid-cols-12 gap-6',
        children: [
          e.jsx('div', {
            className: 'col-span-12 md:col-span-3 space-y-1',
            children: ft.map((w) =>
              e.jsx(
                'button',
                {
                  onClick: () => y(w.name),
                  className: `w-full text-left px-4 py-2 rounded-md transition-colors ${m === w.name ? 'bg-primary-500/10 text-primary-400 font-medium' : 'hover:bg-white/5 text-muted'}`,
                  children: w.name
                },
                w.name
              )
            )
          }),
          e.jsx('div', {
            className: 'col-span-12 md:col-span-9',
            children:
              g &&
              e.jsxs('div', {
                className: 'space-y-6',
                children: [
                  e.jsxs('div', {
                    children: [
                      e.jsxs('h2', {
                        className: 'text-xl font-semibold text-foreground',
                        children: [g.name, ' API']
                      }),
                      e.jsx('p', { className: 'text-muted mt-1', children: g.description })
                    ]
                  }),
                  e.jsx('div', {
                    className: 'space-y-4',
                    children: g.endpoints.map((w, T) =>
                      e.jsxs(
                        'div',
                        {
                          className:
                            'card p-4 flex items-start gap-4 group hover:border-primary-500/30 transition-colors',
                          children: [
                            e.jsx('div', {
                              className: 'w-20 shrink-0',
                              children: e.jsx(qr, { method: w.method })
                            }),
                            e.jsxs('div', {
                              className: 'flex-1 min-w-0',
                              children: [
                                e.jsx('div', {
                                  className: 'font-mono text-sm text-foreground break-all',
                                  children: w.path
                                }),
                                w.description &&
                                  e.jsx('div', {
                                    className: 'text-sm text-muted mt-1',
                                    children: w.description
                                  })
                              ]
                            }),
                            e.jsx('button', {
                              className:
                                'opacity-0 group-hover:opacity-100 p-1 hover:bg-white/10 rounded transition-all',
                              onClick: () => {
                                navigator.clipboard.writeText(w.path)
                              },
                              title: 'Copy path',
                              children: e.jsx('svg', {
                                width: '16',
                                height: '16',
                                viewBox: '0 0 24 24',
                                fill: 'currentColor',
                                className: 'text-muted',
                                children: e.jsx('path', {
                                  d: 'M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z'
                                })
                              })
                            })
                          ]
                        },
                        T
                      )
                    )
                  })
                ]
              })
          })
        ]
      })
    ]
  })
}
function Ft() {
  const [m, y] = k.useState([]),
    [g, w] = k.useState(!1),
    [T, R] = k.useState(''),
    [I, i] = k.useState('qcow2'),
    [o, l] = k.useState(''),
    [u, a] = k.useState(''),
    [h, f] = k.useState(''),
    [x, c] = k.useState(''),
    [t, n] = k.useState(''),
    [s, r] = k.useState(!1)
  k.useEffect(() => {
    ;(async () => y(await ge()))()
  }, [])
  const d = [
    { key: 'name', header: 'Name' },
    { key: 'sizeGiB', header: 'Size (GiB)' },
    {
      key: 'status',
      header: 'Status',
      render: (v) =>
        e.jsx(pe, {
          variant: v.status === 'available' || v.status === 'active' ? 'success' : 'info',
          children: v.status
        })
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (v) =>
        e.jsx('div', {
          className: 'flex gap-2',
          children: e.jsx('button', {
            className: 'btn-secondary btn-xs',
            onClick: async () => {
              r(!0)
              try {
                await cs(v.id)
              } finally {
                ;(r(!1), y(await ge()))
              }
            },
            children: 'Import'
          })
        })
    }
  ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Images',
        subtitle: 'Global images available for projects',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => w(!0),
          children: 'Register Image'
        })
      }),
      e.jsx(Ne, { placeholder: 'Search images' }),
      e.jsx(de, { columns: d, data: m, empty: 'No images' }),
      e.jsx(le, {
        title: 'Register Image',
        open: g,
        onClose: () => w(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => w(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: s,
              onClick: async () => {
                if (T) {
                  r(!0)
                  try {
                    const v = { name: T, disk_format: I }
                    ;(o && (v.rgw_url = o),
                      u && (v.file_path = u),
                      h && x && ((v.rbd_pool = h), (v.rbd_image = x), t && (v.rbd_snap = t)),
                      await ls(v),
                      y(await ge()),
                      w(!1),
                      R(''),
                      i('qcow2'),
                      l(''),
                      a(''),
                      f(''),
                      c(''),
                      n(''))
                  } finally {
                    r(!1)
                  }
                }
              },
              children: 'Register'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: T,
                  onChange: (v) => R(v.target.value)
                })
              ]
            }),
            e.jsx('div', {
              className: 'grid grid-cols-2 gap-3',
              children: e.jsxs('div', {
                children: [
                  e.jsx('label', { className: 'label', children: 'Disk Format' }),
                  e.jsxs('select', {
                    className: 'input w-full',
                    value: I,
                    onChange: (v) => i(v.target.value),
                    children: [
                      e.jsx('option', { value: 'qcow2', children: 'qcow2' }),
                      e.jsx('option', { value: 'raw', children: 'raw' }),
                      e.jsx('option', { value: 'iso', children: 'iso' })
                    ]
                  })
                ]
              })
            }),
            e.jsxs('div', {
              className: 'space-y-2',
              children: [
                e.jsx('div', { className: 'font-medium', children: 'Source' }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'RGW/HTTP URL' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      placeholder: 'https://rgw.example.com/bucket/key',
                      value: o,
                      onChange: (v) => l(v.target.value)
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'File Path (CephFS)' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      placeholder: '/cephfs/vc/images/foo.qcow2',
                      value: u,
                      onChange: (v) => a(v.target.value)
                    })
                  ]
                }),
                e.jsxs('div', {
                  className: 'grid grid-cols-3 gap-3',
                  children: [
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'RBD Pool' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          placeholder: 'vcpool',
                          value: h,
                          onChange: (v) => f(v.target.value)
                        })
                      ]
                    }),
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'RBD Image' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          placeholder: 'ubuntu-22.04',
                          value: x,
                          onChange: (v) => c(v.target.value)
                        })
                      ]
                    }),
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'RBD Snap (optional)' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          placeholder: 'base',
                          value: t,
                          onChange: (v) => n(v.target.value)
                        })
                      ]
                    })
                  ]
                }),
                e.jsx('p', {
                  className: 'text-xs text-muted-foreground',
                  children:
                    '至少提供一种来源（RGW URL 或 FilePath 或 RBD）；注册后可在列表中对含 RGW URL 的镜像点击 Import 执行落地。'
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function $t() {
  const { capacity: m, utilization: y, projects: g } = Ce(),
    [w, T] = k.useState('24h'),
    R = k.useMemo(() => {
      const i = Date.now(),
        o = w === '24h' ? 24 : w === '7d' ? 24 * 7 : 24 * 30
      return i - o * 36e5
    }, [w]),
    I = k.useMemo(() => {
      let i = 0,
        o = 0,
        l = 0
      for (const u of y) {
        const a = u.points.filter((f) => f.t >= R),
          h = a[a.length - 1]
        h && ((i += h.vcpu), (o += h.memGiB), (l += h.storageGiB))
      }
      return { vcpu: i, memGiB: o, storageGiB: l }
    }, [y, R])
  return e.jsxs('div', {
    className: 'space-y-4',
    children: [
      e.jsxs('div', {
        className: 'grid md:grid-cols-3 gap-3',
        children: [
          e.jsxs('div', {
            className: 'card p-4',
            children: [
              e.jsx('div', { className: 'text-gray-400', children: 'vCPU used' }),
              e.jsxs('div', {
                className: 'text-2xl font-semibold',
                children: [I.vcpu, ' / ', m.vcpu]
              }),
              e.jsx('div', {
                className: 'h-2 bg-oxide-800 rounded mt-2',
                children: e.jsx('div', {
                  className: 'h-2 bg-oxide-500 rounded',
                  style: { width: `${Math.min(100, (I.vcpu / m.vcpu) * 100).toFixed(0)}%` }
                })
              })
            ]
          }),
          e.jsxs('div', {
            className: 'card p-4',
            children: [
              e.jsx('div', { className: 'text-gray-400', children: 'Memory used (GiB)' }),
              e.jsxs('div', {
                className: 'text-2xl font-semibold',
                children: [I.memGiB, ' / ', m.memGiB]
              }),
              e.jsx('div', {
                className: 'h-2 bg-oxide-800 rounded mt-2',
                children: e.jsx('div', {
                  className: 'h-2 bg-oxide-500 rounded',
                  style: { width: `${Math.min(100, (I.memGiB / m.memGiB) * 100).toFixed(0)}%` }
                })
              })
            ]
          }),
          e.jsxs('div', {
            className: 'card p-4',
            children: [
              e.jsx('div', { className: 'text-gray-400', children: 'Storage used (GiB)' }),
              e.jsxs('div', {
                className: 'text-2xl font-semibold',
                children: [I.storageGiB, ' / ', m.storageGiB]
              }),
              e.jsx('div', {
                className: 'h-2 bg-oxide-800 rounded mt-2',
                children: e.jsx('div', {
                  className: 'h-2 bg-oxide-500 rounded',
                  style: {
                    width: `${Math.min(100, (I.storageGiB / m.storageGiB) * 100).toFixed(0)}%`
                  }
                })
              })
            ]
          })
        ]
      }),
      e.jsxs('div', {
        className: 'card p-4 flex items-center justify-between',
        children: [
          e.jsx('div', { className: 'text-sm text-gray-300', children: 'Utilization by project' }),
          e.jsxs('div', {
            className: 'flex items-center gap-2',
            children: [
              e.jsx('label', { className: 'label', children: 'Range' }),
              e.jsxs('select', {
                className: 'input',
                value: w,
                onChange: (i) => T(i.target.value),
                children: [
                  e.jsx('option', { value: '24h', children: 'Last 24h' }),
                  e.jsx('option', { value: '7d', children: 'Last 7d' }),
                  e.jsx('option', { value: '30d', children: 'Last 30d' })
                ]
              })
            ]
          })
        ]
      }),
      e.jsx('div', {
        className: 'grid md:grid-cols-2 lg:grid-cols-3 gap-3',
        children: g.map((i) => {
          const l = (y.find((h) => h.projectId === i.id)?.points ?? []).filter((h) => h.t >= R),
            a = l[l.length - 1] ?? { vcpu: 0, memGiB: 0, storageGiB: 0 }
          return e.jsxs(
            'div',
            {
              className: 'card p-4 space-y-2',
              children: [
                e.jsx('div', { className: 'font-medium', children: i.name }),
                e.jsxs('div', { className: 'text-sm text-gray-400', children: ['vCPU: ', a.vcpu] }),
                e.jsxs('div', {
                  className: 'text-sm text-gray-400',
                  children: ['Memory: ', a.memGiB, ' GiB']
                }),
                e.jsxs('div', {
                  className: 'text-sm text-gray-400',
                  children: ['Storage: ', a.storageGiB, ' GiB']
                })
              ]
            },
            i.id
          )
        })
      })
    ]
  })
}
const Jr = 'modulepreload',
  Yr = function (m) {
    return '/' + m
  },
  Wt = {},
  Ke = function (y, g, w) {
    let T = Promise.resolve()
    if (g && g.length > 0) {
      document.getElementsByTagName('link')
      const I = document.querySelector('meta[property=csp-nonce]'),
        i = I?.nonce || I?.getAttribute('nonce')
      T = Promise.allSettled(
        g.map((o) => {
          if (((o = Yr(o)), o in Wt)) return
          Wt[o] = !0
          const l = o.endsWith('.css'),
            u = l ? '[rel="stylesheet"]' : ''
          if (document.querySelector(`link[href="${o}"]${u}`)) return
          const a = document.createElement('link')
          if (
            ((a.rel = l ? 'stylesheet' : Jr),
            l || (a.as = 'script'),
            (a.crossOrigin = ''),
            (a.href = o),
            i && a.setAttribute('nonce', i),
            document.head.appendChild(a),
            l)
          )
            return new Promise((h, f) => {
              ;(a.addEventListener('load', h),
                a.addEventListener('error', () => f(new Error(`Unable to preload CSS for ${o}`))))
            })
        })
      )
    }
    function R(I) {
      const i = new Event('vite:preloadError', { cancelable: !0 })
      if (((i.payload = I), window.dispatchEvent(i), !i.defaultPrevented)) throw I
    }
    return T.then((I) => {
      for (const i of I || []) i.status === 'rejected' && R(i.reason)
      return y().catch(R)
    })
  }
function Fs({ node: m, onPick: y, openMap: g, setOpenMap: w, query: T }) {
  const R = g[m.id] ?? !0
  if (m.kind === 'group') {
    const i = m
    return e.jsxs('div', {
      className: 'ml-2',
      children: [
        e.jsxs('button', {
          type: 'button',
          className: 'text-left text-sm text-gray-200 hover:underline',
          onClick: () => w(i.id, !R),
          children: [R ? '▾' : '▸', ' ', i.name]
        }),
        R &&
          e.jsx('div', {
            className: 'ml-4 border-l border-oxide-800 pl-2',
            children: i.children
              .filter((o) => Bt(o, T))
              .map((o) =>
                e.jsx(Fs, { node: o, onPick: y, openMap: g, setOpenMap: w, query: T }, o.id)
              )
          })
      ]
    })
  }
  const I = m
  return e.jsxs('div', {
    className: 'ml-2 text-sm',
    children: [
      e.jsx('button', {
        type: 'button',
        className: 'text-blue-300 hover:underline',
        onClick: () => y(I),
        children: I.name
      }),
      e.jsxs('span', {
        className: 'text-gray-500',
        children: [' ', '— ', I.address, I.defaultUser ? ` (${I.defaultUser})` : '']
      })
    ]
  })
}
function Bt(m, y) {
  if (!y) return !0
  const g = y.toLowerCase()
  if (m.kind === 'group') {
    const T = m
    return T.name.toLowerCase().includes(g) || T.children.some((R) => Bt(R, y))
  }
  const w = m
  return (
    w.name.toLowerCase().includes(g) ||
    w.address.toLowerCase().includes(g) ||
    (w.defaultUser?.toLowerCase().includes(g) ?? !1) ||
    (w.tags?.some((T) => T.toLowerCase().includes(g)) ?? !1)
  )
}
function Zr() {
  const m = Ce((B) => B.cmdb),
    [y, g] = k.useState(''),
    [w, T] = k.useState(''),
    [R, I] = k.useState('root'),
    [i, o] = k.useState('password'),
    [l, u] = k.useState(''),
    [a, h] = k.useState(''),
    [f, x] = k.useState(''),
    [c, t] = k.useState(!1),
    [n, s] = k.useState(!1),
    [r, d] = k.useState(''),
    [v, _] = k.useState(''),
    b = k.useMemo(() => m.filter((B) => Bt(B, v)), [m, v]),
    [p, S] = k.useState(() => {
      try {
        const B = localStorage.getItem('webshell_cmdb_open')
        return B ? JSON.parse(B) : {}
      } catch {
        return {}
      }
    }),
    L = (B, z) => {
      S((K) => {
        const J = { ...K, [B]: z }
        return (localStorage.setItem('webshell_cmdb_open', JSON.stringify(J)), J)
      })
    },
    M = k.useRef(null),
    P = k.useRef(null),
    j = k.useRef(null),
    D = k.useRef(null),
    O = k.useRef(null),
    $ = k.useRef(null),
    F = async () => {
      if (M.current || !P.current) return
      const [{ Terminal: B }, { FitAddon: z }, { WebLinksAddon: K }] = await Promise.all([
          Ke(() => import('./ui-vendor-CJfbT-UK.js').then((G) => G.x), __vite__mapDeps([0, 1])),
          Ke(() => Promise.resolve().then(() => ki), void 0),
          Ke(
            () => import('./addon-web-links-D-NCOiOE.js').then((G) => G.a),
            __vite__mapDeps([2, 1])
          ),
          Ke(() => Promise.resolve({}), __vite__mapDeps([3]))
        ]),
        J = new B({
          theme: { background: '#0b0f14' },
          fontSize: 14,
          fontFamily: 'Menlo, Monaco, "Courier New", monospace',
          cursorBlink: !0,
          cursorStyle: 'block',
          allowProposedApi: !0
        }),
        Q = new z(),
        H = new K()
      ;(J.loadAddon(Q),
        J.loadAddon(H),
        J.open(P.current),
        Q.fit(),
        (M.current = J),
        (D.current = Q))
      const E = () => {
        D.current && D.current.fit()
      }
      return (
        window.addEventListener('resize', E),
        () => {
          window.removeEventListener('resize', E)
        }
      )
    },
    W = async () => {
      if (!(!y || c)) {
        if (i === 'password' && !l) {
          d('Password is required')
          return
        }
        if (i === 'key' && !f) {
          d('Private key is required')
          return
        }
        ;(t(!0), d(''))
        try {
          await F()
          const B = M.current
          if (!B) throw new Error('Terminal not initialized')
          ;(B.write('\x1B[2J\x1B[H'),
            B.writeln('Connecting to ' + R + '@' + y + (w ? ':' + w : '') + '...'))
          const z = qe()
          let K
          if (z.startsWith('http://') || z.startsWith('https://')) {
            const Q = new URL(z)
            K = `${Q.protocol === 'https:' ? 'wss:' : 'ws:'}//${Q.host}/ws/webshell`
          } else
            K = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/webshell`
          const J = new WebSocket(K)
          ;((j.current = J),
            (J.onopen = () => {
              const Q = {
                host: y,
                port: w || 22,
                user: R,
                auth_method: i,
                password: i === 'password' ? l : '',
                private_key: i === 'key' ? f : ''
              }
              J.send(JSON.stringify(Q))
            }),
            (J.onmessage = (Q) => {
              try {
                const H = JSON.parse(Q.data)
                if (H.type === 'connected') {
                  if (
                    (s(!0),
                    t(!1),
                    B.writeln(`\r
\x1B[32mConnected successfully!\x1B[0m\r
`),
                    O.current && O.current.dispose(),
                    (O.current = B.onData((E) => {
                      J.readyState === WebSocket.OPEN &&
                        J.send(JSON.stringify({ type: 'input', data: E }))
                    })),
                    $.current && $.current.dispose(),
                    ($.current = B.onResize((E) => {
                      J.readyState === WebSocket.OPEN &&
                        J.send(JSON.stringify({ type: 'resize', cols: E.cols, rows: E.rows }))
                    })),
                    D.current)
                  ) {
                    const E = D.current.proposeDimensions()
                    E &&
                      J.readyState === WebSocket.OPEN &&
                      J.send(JSON.stringify({ type: 'resize', cols: E.cols, rows: E.rows }))
                  }
                } else if (H.type === 'output') B.write(H.data)
                else if (H.type === 'error') {
                  const E = H.data
                  ;(B.writeln(`\r
\x1B[1;31m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1B[0m`),
                    B.writeln('\x1B[1;31m✗ Connection Failed\x1B[0m'),
                    B.writeln('\x1B[1;31m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1B[0m'),
                    B.writeln('\x1B[33m' + E + '\x1B[0m'),
                    B.writeln(''),
                    E.includes('unable to authenticate') ||
                    E.includes('authentication failed') ||
                    E.includes('password')
                      ? (B.writeln('\x1B[36mPossible reasons:\x1B[0m'),
                        B.writeln('  • \x1B[33mIncorrect password or SSH key\x1B[0m'),
                        B.writeln('  • \x1B[33mUser account does not exist\x1B[0m'),
                        B.writeln('  • \x1B[33mSSH key not authorized on server\x1B[0m'),
                        B.writeln(''),
                        B.writeln('\x1B[1;32mPlease check your credentials and try again.\x1B[0m'),
                        d(
                          '❌ Authentication failed. Please check your password or SSH key and try again.'
                        ))
                      : d('❌ ' + E),
                    B.writeln(`\x1B[1;31m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1B[0m\r
`),
                    t(!1),
                    s(!1),
                    J.close())
                }
              } catch (H) {
                console.error('Failed to parse WebSocket message:', H)
              }
            }),
            (J.onerror = (Q) => {
              ;(console.error('WebSocket error:', Q),
                B.writeln(`\r
\x1B[31mConnection error\x1B[0m`),
                d('❌ WebSocket connection error. Please check network connectivity.'),
                t(!1),
                s(!1))
            }),
            (J.onclose = () => {
              ;(B.writeln(`\r
\x1B[33mConnection closed\x1B[0m`),
                s(!1),
                t(!1),
                O.current && (O.current.dispose(), (O.current = null)),
                $.current && ($.current.dispose(), ($.current = null)))
            }))
        } catch (B) {
          ;(console.error('Connection error:', B),
            d(B instanceof Error ? B.message : 'Connection failed'),
            t(!1))
        }
      }
    },
    C = () => {
      ;(j.current && (j.current.close(), (j.current = null)), s(!1))
    },
    A = () => {
      ;(g(''), T(''), u(''), h(''), x(''), d(''))
    },
    N = (B) => {
      const z = B.target.files?.[0]
      if (!z) {
        ;(h(''), x(''))
        return
      }
      h(z.name)
      const K = new FileReader()
      ;((K.onload = () => {
        x(String(K.result || ''))
      }),
        K.readAsText(z))
    }
  return (
    k.useEffect(
      () => () => {
        ;(j.current && j.current.close(),
          M.current && M.current.dispose(),
          O.current && O.current.dispose(),
          $.current && $.current.dispose())
      },
      []
    ),
    e.jsxs('div', {
      className: 'space-y-4',
      children: [
        e.jsxs('div', {
          className: 'flex items-center justify-between',
          children: [
            e.jsx(oe, { title: 'WebShell', subtitle: 'SSH terminal access to remote hosts' }),
            e.jsxs('button', {
              onClick: () => (window.location.href = '/webshell/sessions'),
              className:
                'px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 flex items-center gap-2',
              children: [
                e.jsx('svg', {
                  className: 'w-4 h-4',
                  fill: 'none',
                  stroke: 'currentColor',
                  viewBox: '0 0 24 24',
                  children: e.jsx('path', {
                    strokeLinecap: 'round',
                    strokeLinejoin: 'round',
                    strokeWidth: 2,
                    d: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z'
                  })
                }),
                '会话记录'
              ]
            })
          ]
        }),
        e.jsxs('div', {
          className: 'grid gap-4 md:grid-cols-[360px_1fr]',
          children: [
            e.jsxs('div', {
              className: 'card p-4 space-y-3',
              children: [
                e.jsxs('div', {
                  children: [
                    e.jsxs('div', {
                      className: 'flex items-center justify-between mb-2',
                      children: [
                        e.jsx('div', {
                          className: 'text-sm font-medium text-gray-200',
                          children: 'CMDB'
                        }),
                        e.jsx('span', {
                          className: 'text-xs text-gray-500',
                          children: 'Select a host'
                        })
                      ]
                    }),
                    e.jsx('input', {
                      className: 'input w-full mb-2',
                      placeholder: 'Search hosts/groups...',
                      value: v,
                      onChange: (B) => _(B.target.value)
                    }),
                    e.jsx('div', {
                      className:
                        'max-h-56 overflow-auto rounded border border-oxide-800 p-2 bg-oxide-950',
                      children: b.map((B) =>
                        e.jsx(
                          Fs,
                          {
                            node: B,
                            openMap: p,
                            setOpenMap: L,
                            query: v,
                            onPick: (z) => {
                              ;(g(z.address), z.defaultUser && I(z.defaultUser))
                            }
                          },
                          B.id
                        )
                      )
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Host' }),
                    e.jsx('input', {
                      className: 'input w-full',
                      placeholder: '10.0.0.10 or hostname',
                      value: y,
                      onChange: (B) => g(B.target.value),
                      disabled: n
                    })
                  ]
                }),
                e.jsxs('div', {
                  className: 'grid grid-cols-2 gap-3',
                  children: [
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'Port' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          type: 'number',
                          placeholder: '22',
                          value: w,
                          onChange: (B) => T(B.target.value ? Number(B.target.value) : ''),
                          disabled: n
                        })
                      ]
                    }),
                    e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'User' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          value: R,
                          onChange: (B) => I(B.target.value),
                          disabled: n
                        })
                      ]
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Auth Method' }),
                    e.jsxs('div', {
                      className: 'flex items-center gap-4 text-sm text-gray-200',
                      children: [
                        e.jsxs('label', {
                          className: 'inline-flex items-center gap-2',
                          children: [
                            e.jsx('input', {
                              type: 'radio',
                              name: 'auth',
                              checked: i === 'password',
                              onChange: () => o('password'),
                              disabled: n
                            }),
                            'Password'
                          ]
                        }),
                        e.jsxs('label', {
                          className: 'inline-flex items-center gap-2',
                          children: [
                            e.jsx('input', {
                              type: 'radio',
                              name: 'auth',
                              checked: i === 'key',
                              onChange: () => o('key'),
                              disabled: n
                            }),
                            'SSH Key'
                          ]
                        })
                      ]
                    })
                  ]
                }),
                i === 'password'
                  ? e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'Password' }),
                        e.jsx('input', {
                          className: 'input w-full',
                          type: 'password',
                          value: l,
                          onChange: (B) => u(B.target.value),
                          disabled: n
                        })
                      ]
                    })
                  : e.jsxs('div', {
                      children: [
                        e.jsx('label', { className: 'label', children: 'SSH Private Key' }),
                        e.jsx('input', {
                          className:
                            'input w-full file:mr-4 file:py-2 file:px-3 file:rounded file:border-0 file:bg-oxide-700 file:text-gray-200',
                          type: 'file',
                          onChange: N,
                          disabled: n
                        }),
                        a &&
                          e.jsxs('div', {
                            className: 'text-xs text-gray-500 mt-1',
                            children: [
                              'Selected: ',
                              a,
                              ' ',
                              f ? `(${f.length} bytes)` : '(loading...)'
                            ]
                          })
                      ]
                    }),
                r &&
                  e.jsxs('div', {
                    className: 'bg-red-900/30 border-2 border-red-600 rounded-lg p-4 space-y-2',
                    children: [
                      e.jsxs('div', {
                        className: 'flex items-start gap-2',
                        children: [
                          e.jsx('span', {
                            className: 'text-red-500 text-xl mt-0.5',
                            children: '⚠️'
                          }),
                          e.jsxs('div', {
                            className: 'flex-1',
                            children: [
                              e.jsx('div', {
                                className: 'font-semibold text-red-400 mb-1',
                                children: 'Connection Error'
                              }),
                              e.jsx('div', { className: 'text-sm text-red-300', children: r })
                            ]
                          })
                        ]
                      }),
                      r.includes('Authentication') || r.includes('authentication')
                        ? e.jsxs('div', {
                            className: 'mt-3 pt-3 border-t border-red-700/50',
                            children: [
                              e.jsx('div', {
                                className: 'text-xs text-red-200 mb-2',
                                children: '💡 Quick fix:'
                              }),
                              e.jsxs('ul', {
                                className: 'text-xs text-red-300 space-y-1 ml-4',
                                children: [
                                  e.jsx('li', {
                                    children: '• Check if your password/key is correct'
                                  }),
                                  e.jsx('li', {
                                    children: '• Make sure the user account exists on the server'
                                  }),
                                  e.jsx('li', {
                                    children: '• Verify SSH key is in ~/.ssh/authorized_keys'
                                  })
                                ]
                              }),
                              e.jsx('button', {
                                className:
                                  'mt-3 text-xs bg-red-700 hover:bg-red-600 text-white px-3 py-1.5 rounded',
                                onClick: () => {
                                  ;(d(''), i === 'password' ? u('') : (h(''), x('')))
                                },
                                children: '🔄 Clear and Retry'
                              })
                            ]
                          })
                        : null
                    ]
                  }),
                e.jsx('div', {
                  className: 'flex gap-2',
                  children: n
                    ? e.jsx('button', {
                        className: 'btn-secondary',
                        onClick: C,
                        children: 'Disconnect'
                      })
                    : e.jsxs(e.Fragment, {
                        children: [
                          e.jsx('button', {
                            className: 'btn-primary',
                            disabled: !y || c || (i === 'password' ? !l : !f),
                            onClick: W,
                            children: c ? 'Connecting…' : r ? 'Retry Connection' : 'Connect'
                          }),
                          e.jsx('button', {
                            className: 'btn-secondary',
                            onClick: A,
                            disabled: c,
                            children: 'Clear'
                          })
                        ]
                      })
                })
              ]
            }),
            e.jsx('div', {
              className: 'card p-0 overflow-hidden',
              children: e.jsx('div', { ref: P, className: 'h-[600px] w-full bg-oxide-950 p-2' })
            })
          ]
        })
      ]
    })
  )
}
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const Qr = (m) => m.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase(),
  ei = (m) =>
    m.replace(/^([A-Z])|[\s-_]+(\w)/g, (y, g, w) => (w ? w.toUpperCase() : g.toLowerCase())),
  Ut = (m) => {
    const y = ei(m)
    return y.charAt(0).toUpperCase() + y.slice(1)
  },
  $s = (...m) =>
    m
      .filter((y, g, w) => !!y && y.trim() !== '' && w.indexOf(y) === g)
      .join(' ')
      .trim(),
  ti = (m) => {
    for (const y in m) if (y.startsWith('aria-') || y === 'role' || y === 'title') return !0
  }
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ var si = {
  xmlns: 'http://www.w3.org/2000/svg',
  width: 24,
  height: 24,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round',
  strokeLinejoin: 'round'
}
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const ri = k.forwardRef(
  (
    {
      color: m = 'currentColor',
      size: y = 24,
      strokeWidth: g = 2,
      absoluteStrokeWidth: w,
      className: T = '',
      children: R,
      iconNode: I,
      ...i
    },
    o
  ) =>
    k.createElement(
      'svg',
      {
        ref: o,
        ...si,
        width: y,
        height: y,
        stroke: m,
        strokeWidth: w ? (Number(g) * 24) / Number(y) : g,
        className: $s('lucide', T),
        ...(!R && !ti(i) && { 'aria-hidden': 'true' }),
        ...i
      },
      [...I.map(([l, u]) => k.createElement(l, u)), ...(Array.isArray(R) ? R : [R])]
    )
)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const ke = (m, y) => {
  const g = k.forwardRef(({ className: w, ...T }, R) =>
    k.createElement(ri, {
      ref: R,
      iconNode: y,
      className: $s(`lucide-${Qr(Ut(m))}`, `lucide-${m}`, w),
      ...T
    })
  )
  return ((g.displayName = Ut(m)), g)
}
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const ii = [
    ['path', { d: 'm12 19-7-7 7-7', key: '1l729n' }],
    ['path', { d: 'M19 12H5', key: 'x3x0zl' }]
  ],
  ni = ke('arrow-left', ii)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const ai = [
    ['path', { d: 'M12 6v6l4 2', key: 'mmk7yg' }],
    ['circle', { cx: '12', cy: '12', r: '10', key: '1mglay' }]
  ],
  oi = ke('clock', ai)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const li = [
    ['path', { d: 'M12 15V3', key: 'm9g1x1' }],
    ['path', { d: 'M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4', key: 'ih7n3h' }],
    ['path', { d: 'm7 10 5 5 5-5', key: 'brsn70' }]
  ],
  Ws = ke('download', li)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const ci = [
    [
      'path',
      {
        d: 'M10 20a1 1 0 0 0 .553.895l2 1A1 1 0 0 0 14 21v-7a2 2 0 0 1 .517-1.341L21.74 4.67A1 1 0 0 0 21 3H3a1 1 0 0 0-.742 1.67l7.225 7.989A2 2 0 0 1 10 14z',
        key: 'sc7q7i'
      }
    ]
  ],
  hi = ke('funnel', ci)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const di = [
    ['line', { x1: '22', x2: '2', y1: '12', y2: '12', key: '1y58io' }],
    [
      'path',
      {
        d: 'M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z',
        key: 'oot6mr'
      }
    ],
    ['line', { x1: '6', x2: '6.01', y1: '16', y2: '16', key: 'sgf278' }],
    ['line', { x1: '10', x2: '10.01', y1: '16', y2: '16', key: '1l4acy' }]
  ],
  zt = ke('hard-drive', di)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const ui = [
    ['rect', { x: '14', y: '3', width: '5', height: '18', rx: '1', key: 'kaeet6' }],
    ['rect', { x: '5', y: '3', width: '5', height: '18', rx: '1', key: '1wsw3u' }]
  ],
  fi = ke('pause', ui)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const pi = [
    [
      'path',
      {
        d: 'M5 5a2 2 0 0 1 3.008-1.728l11.997 6.998a2 2 0 0 1 .003 3.458l-12 7A2 2 0 0 1 5 19z',
        key: '10ikf1'
      }
    ]
  ],
  Us = ke('play', pi)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const mi = [
    ['rect', { width: '20', height: '8', x: '2', y: '2', rx: '2', ry: '2', key: 'ngkwjq' }],
    ['rect', { width: '20', height: '8', x: '2', y: '14', rx: '2', ry: '2', key: 'iecqi9' }],
    ['line', { x1: '6', x2: '6.01', y1: '6', y2: '6', key: '16zg32' }],
    ['line', { x1: '6', x2: '6.01', y1: '18', y2: '18', key: 'nzw8ys' }]
  ],
  Vt = ke('server', mi)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const _i = [
    [
      'path',
      {
        d: 'M17.971 4.285A2 2 0 0 1 21 6v12a2 2 0 0 1-3.029 1.715l-9.997-5.998a2 2 0 0 1-.003-3.432z',
        key: '15892j'
      }
    ],
    ['path', { d: 'M3 20V4', key: '1ptbpl' }]
  ],
  Gt = ke('skip-back', _i)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const gi = [
    ['path', { d: 'M21 4v16', key: '7j8fe9' }],
    [
      'path',
      {
        d: 'M6.029 4.285A2 2 0 0 0 3 6v12a2 2 0 0 0 3.029 1.715l9.997-5.998a2 2 0 0 0 .003-3.432z',
        key: 'zs4d6'
      }
    ]
  ],
  vi = ke('skip-forward', gi)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const xi = [
    ['path', { d: 'M12 19h8', key: 'baeox8' }],
    ['path', { d: 'm4 17 6-6-6-6', key: '1yngyt' }]
  ],
  bi = ke('terminal', xi)
/**
 * @license lucide-react v0.556.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */ const yi = [
    ['path', { d: 'M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2', key: '975kel' }],
    ['circle', { cx: '12', cy: '7', r: '4', key: '17ys0d' }]
  ],
  Kt = ke('user', yi),
  qt = qe()
function Si() {
  const [m, y] = k.useState([]),
    [g, w] = k.useState(0),
    [T, R] = k.useState(1),
    [I] = k.useState(20),
    [i, o] = k.useState(!1),
    [l, u] = k.useState(''),
    [a, h] = k.useState(''),
    [f, x] = k.useState('all'),
    [c, t] = k.useState(!1)
  k.useEffect(() => {
    ;(async () => {
      o(!0)
      try {
        const p = new URLSearchParams({ page: T.toString(), page_size: I.toString() })
        ;(l && p.append('username', l),
          a && p.append('remote_host', a),
          f !== 'all' && p.append('status', f))
        const S = await fetch(`${qt}/v1/webshell/sessions?${p}`, {
          headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
        })
        if (S.ok) {
          const L = await S.json()
          ;(y(L.data || []), w(L.total))
        }
      } catch {
      } finally {
        o(!1)
      }
    })()
  }, [T, I, l, a, f])
  const n = (b) => {
      if (!b) return 'N/A'
      const p = Math.floor(b / 3600),
        S = Math.floor((b % 3600) / 60),
        L = b % 60
      return p > 0 ? `${p}h ${S}m ${L}s` : S > 0 ? `${S}m ${L}s` : `${L}s`
    },
    s = (b) => {
      if (b === 0) return '0 B'
      const p = 1024,
        S = ['B', 'KB', 'MB', 'GB'],
        L = Math.floor(Math.log(b) / Math.log(p))
      return Math.round((b / Math.pow(p, L)) * 100) / 100 + ' ' + S[L]
    },
    r = (b) =>
      new Date(b).toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      }),
    d = async (b) => {
      try {
        const p = await fetch(`${qt}/v1/webshell/sessions/${b.session_id}/export`, {
          headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
        })
        if (p.ok) {
          const S = await p.blob(),
            L = window.URL.createObjectURL(S),
            M = document.createElement('a')
          ;((M.href = L),
            (M.download = `webshell-${b.session_id}-${b.started_at}.cast`),
            document.body.appendChild(M),
            M.click(),
            document.body.removeChild(M),
            window.URL.revokeObjectURL(L))
        }
      } catch {}
    },
    v = (b) => {
      switch (b) {
        case 'active':
          return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
        case 'closed':
          return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
        case 'connection_failed':
        case 'auth_failed':
          return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
        default:
          return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
      }
    },
    _ = Math.ceil(g / I)
  return e.jsxs('div', {
    className: 'p-6 space-y-6',
    children: [
      e.jsxs('div', {
        className: 'flex items-center justify-between',
        children: [
          e.jsxs('div', {
            children: [
              e.jsx('h1', {
                className: 'text-2xl font-bold text-gray-900 dark:text-white',
                children: 'WebShell 会话记录'
              }),
              e.jsx('p', {
                className: 'text-sm text-gray-500 dark:text-gray-400 mt-1',
                children: '查看和管理所有SSH会话记录'
              })
            ]
          }),
          e.jsxs('button', {
            onClick: () => t(!c),
            className:
              'flex items-center gap-2 px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700',
            children: [e.jsx(hi, { className: 'w-4 h-4' }), '筛选']
          })
        ]
      }),
      c &&
        e.jsxs('div', {
          className:
            'bg-white dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700 space-y-4',
          children: [
            e.jsxs('div', {
              className: 'grid grid-cols-1 md:grid-cols-3 gap-4',
              children: [
                e.jsxs('div', {
                  children: [
                    e.jsxs('label', {
                      className: 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1',
                      children: [e.jsx(Kt, { className: 'w-4 h-4 inline mr-1' }), '用户名']
                    }),
                    e.jsx('input', {
                      type: 'text',
                      value: l,
                      onChange: (b) => u(b.target.value),
                      placeholder: '搜索用户名...',
                      className:
                        'w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white'
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsxs('label', {
                      className: 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1',
                      children: [e.jsx(Vt, { className: 'w-4 h-4 inline mr-1' }), '远程主机']
                    }),
                    e.jsx('input', {
                      type: 'text',
                      value: a,
                      onChange: (b) => h(b.target.value),
                      placeholder: '搜索主机地址...',
                      className:
                        'w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white'
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', {
                      className: 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1',
                      children: '状态'
                    }),
                    e.jsxs('select', {
                      value: f,
                      onChange: (b) => x(b.target.value),
                      className:
                        'w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white',
                      children: [
                        e.jsx('option', { value: 'all', children: '全部' }),
                        e.jsx('option', { value: 'active', children: '活动中' }),
                        e.jsx('option', { value: 'closed', children: '已关闭' }),
                        e.jsx('option', { value: 'connecting', children: '连接中' }),
                        e.jsx('option', { value: 'connection_failed', children: '连接失败' }),
                        e.jsx('option', { value: 'auth_failed', children: '认证失败' })
                      ]
                    })
                  ]
                })
              ]
            }),
            e.jsx('div', {
              className: 'flex justify-end',
              children: e.jsx('button', {
                onClick: () => {
                  ;(u(''), h(''), x('all'))
                },
                className:
                  'px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white',
                children: '清空筛选'
              })
            })
          ]
        }),
      e.jsx('div', {
        className:
          'bg-white dark:bg-gray-800 p-4 rounded-lg border border-gray-200 dark:border-gray-700',
        children: e.jsxs('div', {
          className: 'flex items-center justify-between text-sm text-gray-600 dark:text-gray-400',
          children: [
            e.jsxs('span', { children: ['共 ', g, ' 条会话记录'] }),
            e.jsxs('span', { children: ['第 ', T, ' / ', _, ' 页'] })
          ]
        })
      }),
      e.jsx('div', {
        className:
          'bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden',
        children: i
          ? e.jsx('div', {
              className: 'p-8 text-center text-gray-500 dark:text-gray-400',
              children: '加载中...'
            })
          : m.length === 0
            ? e.jsxs('div', {
                className: 'p-8 text-center text-gray-500 dark:text-gray-400',
                children: [
                  e.jsx(bi, { className: 'w-12 h-12 mx-auto mb-3 opacity-50' }),
                  e.jsx('p', { children: '暂无会话记录' })
                ]
              })
            : e.jsx('div', {
                className: 'overflow-x-auto',
                children: e.jsxs('table', {
                  className: 'w-full',
                  children: [
                    e.jsx('thead', {
                      className:
                        'bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700',
                      children: e.jsxs('tr', {
                        children: [
                          e.jsx('th', {
                            className:
                              'px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider',
                            children: '会话信息'
                          }),
                          e.jsx('th', {
                            className:
                              'px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider',
                            children: '远程主机'
                          }),
                          e.jsx('th', {
                            className:
                              'px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider',
                            children: '时间'
                          }),
                          e.jsx('th', {
                            className:
                              'px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider',
                            children: '流量'
                          }),
                          e.jsx('th', {
                            className:
                              'px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider',
                            children: '状态'
                          }),
                          e.jsx('th', {
                            className:
                              'px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider',
                            children: '操作'
                          })
                        ]
                      })
                    }),
                    e.jsx('tbody', {
                      className: 'divide-y divide-gray-200 dark:divide-gray-700',
                      children: m.map((b) =>
                        e.jsxs(
                          'tr',
                          {
                            className: 'hover:bg-gray-50 dark:hover:bg-gray-900/50',
                            children: [
                              e.jsx('td', {
                                className: 'px-4 py-4',
                                children: e.jsxs('div', {
                                  className: 'space-y-1',
                                  children: [
                                    e.jsxs('div', {
                                      className: 'flex items-center gap-2',
                                      children: [
                                        e.jsx(Kt, { className: 'w-4 h-4 text-gray-400' }),
                                        e.jsx('span', {
                                          className: 'font-medium text-gray-900 dark:text-white',
                                          children: b.username
                                        })
                                      ]
                                    }),
                                    e.jsxs('div', {
                                      className:
                                        'text-xs text-gray-500 dark:text-gray-400 font-mono',
                                      children: [b.session_id.substring(0, 16), '...']
                                    }),
                                    e.jsx('div', {
                                      className: 'text-xs text-gray-500 dark:text-gray-400',
                                      children: b.client_ip && `来自 ${b.client_ip}`
                                    })
                                  ]
                                })
                              }),
                              e.jsx('td', {
                                className: 'px-4 py-4',
                                children: e.jsxs('div', {
                                  className: 'space-y-1',
                                  children: [
                                    e.jsxs('div', {
                                      className: 'flex items-center gap-2',
                                      children: [
                                        e.jsx(Vt, { className: 'w-4 h-4 text-gray-400' }),
                                        e.jsxs('span', {
                                          className:
                                            'text-sm font-medium text-gray-900 dark:text-white',
                                          children: [
                                            b.remote_user,
                                            '@',
                                            b.remote_host,
                                            ':',
                                            b.remote_port
                                          ]
                                        })
                                      ]
                                    }),
                                    e.jsx('div', {
                                      className: 'text-xs text-gray-500 dark:text-gray-400',
                                      children:
                                        b.auth_method === 'password' ? '密码认证' : 'SSH密钥认证'
                                    })
                                  ]
                                })
                              }),
                              e.jsx('td', {
                                className: 'px-4 py-4',
                                children: e.jsxs('div', {
                                  className: 'space-y-1 text-sm',
                                  children: [
                                    e.jsxs('div', {
                                      className:
                                        'flex items-center gap-2 text-gray-600 dark:text-gray-400',
                                      children: [
                                        e.jsx(oi, { className: 'w-4 h-4' }),
                                        r(b.started_at)
                                      ]
                                    }),
                                    b.ended_at &&
                                      e.jsxs('div', {
                                        className: 'text-xs text-gray-500 dark:text-gray-400',
                                        children: ['持续 ', n(b.duration_seconds)]
                                      })
                                  ]
                                })
                              }),
                              e.jsx('td', {
                                className: 'px-4 py-4',
                                children: e.jsxs('div', {
                                  className: 'space-y-1 text-xs',
                                  children: [
                                    e.jsxs('div', {
                                      className:
                                        'flex items-center gap-1 text-gray-600 dark:text-gray-400',
                                      children: [
                                        e.jsx(zt, { className: 'w-3 h-3' }),
                                        '↑ ',
                                        s(b.bytes_sent)
                                      ]
                                    }),
                                    e.jsxs('div', {
                                      className:
                                        'flex items-center gap-1 text-gray-600 dark:text-gray-400',
                                      children: [
                                        e.jsx(zt, { className: 'w-3 h-3' }),
                                        '↓ ',
                                        s(b.bytes_received)
                                      ]
                                    })
                                  ]
                                })
                              }),
                              e.jsx('td', {
                                className: 'px-4 py-4',
                                children: e.jsx('span', {
                                  className: `inline-flex px-2 py-1 text-xs font-medium rounded-full ${v(b.status)}`,
                                  children: b.status
                                })
                              }),
                              e.jsx('td', {
                                className: 'px-4 py-4',
                                children: e.jsxs('div', {
                                  className: 'flex items-center justify-end gap-2',
                                  children: [
                                    e.jsx('button', {
                                      onClick: () =>
                                        (window.location.href = `/webshell/replay/${b.session_id}`),
                                      className:
                                        'p-2 text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded',
                                      title: '回放',
                                      children: e.jsx(Us, { className: 'w-4 h-4' })
                                    }),
                                    e.jsx('button', {
                                      onClick: () => d(b),
                                      className:
                                        'p-2 text-gray-600 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 rounded',
                                      title: '导出',
                                      children: e.jsx(Ws, { className: 'w-4 h-4' })
                                    })
                                  ]
                                })
                              })
                            ]
                          },
                          b.id
                        )
                      )
                    })
                  ]
                })
              })
      }),
      _ > 1 &&
        e.jsxs('div', {
          className: 'flex items-center justify-between',
          children: [
            e.jsx('button', {
              onClick: () => R(Math.max(1, T - 1)),
              disabled: T === 1,
              className:
                'px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700',
              children: '上一页'
            }),
            e.jsx('div', {
              className: 'flex items-center gap-2',
              children: Array.from({ length: Math.min(5, _) }, (b, p) => {
                let S
                return (
                  _ <= 5 || T <= 3 ? (S = p + 1) : T >= _ - 2 ? (S = _ - 4 + p) : (S = T - 2 + p),
                  e.jsx(
                    'button',
                    {
                      onClick: () => R(S),
                      className: `px-3 py-1 text-sm font-medium rounded ${T === S ? 'bg-blue-600 text-white' : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700'}`,
                      children: S
                    },
                    S
                  )
                )
              })
            }),
            e.jsx('button', {
              onClick: () => R(Math.min(_, T + 1)),
              disabled: T === _,
              className:
                'px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700',
              children: '下一页'
            })
          ]
        })
    ]
  })
}
var zs = { exports: {} }
;(function (m, y) {
  ;(function (g, w) {
    m.exports = w()
  })(globalThis, () =>
    (() => {
      var g = {
          4567: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (s, r, d, v) {
                  var _,
                    b = arguments.length,
                    p = b < 3 ? r : v === null ? (v = Object.getOwnPropertyDescriptor(r, d)) : v
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    p = Reflect.decorate(s, r, d, v)
                  else
                    for (var S = s.length - 1; S >= 0; S--)
                      (_ = s[S]) && (p = (b < 3 ? _(p) : b > 3 ? _(r, d, p) : _(r, d)) || p)
                  return (b > 3 && p && Object.defineProperty(r, d, p), p)
                },
              u =
                (this && this.__param) ||
                function (s, r) {
                  return function (d, v) {
                    r(d, v, s)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.AccessibilityManager = void 0))
            const a = o(9042),
              h = o(9924),
              f = o(844),
              x = o(4725),
              c = o(2585),
              t = o(3656)
            let n = (i.AccessibilityManager = class extends f.Disposable {
              constructor(s, r, d, v) {
                ;(super(),
                  (this._terminal = s),
                  (this._coreBrowserService = d),
                  (this._renderService = v),
                  (this._rowColumns = new WeakMap()),
                  (this._liveRegionLineCount = 0),
                  (this._charsToConsume = []),
                  (this._charsToAnnounce = ''),
                  (this._accessibilityContainer =
                    this._coreBrowserService.mainDocument.createElement('div')),
                  this._accessibilityContainer.classList.add('xterm-accessibility'),
                  (this._rowContainer = this._coreBrowserService.mainDocument.createElement('div')),
                  this._rowContainer.setAttribute('role', 'list'),
                  this._rowContainer.classList.add('xterm-accessibility-tree'),
                  (this._rowElements = []))
                for (let _ = 0; _ < this._terminal.rows; _++)
                  ((this._rowElements[_] = this._createAccessibilityTreeNode()),
                    this._rowContainer.appendChild(this._rowElements[_]))
                if (
                  ((this._topBoundaryFocusListener = (_) => this._handleBoundaryFocus(_, 0)),
                  (this._bottomBoundaryFocusListener = (_) => this._handleBoundaryFocus(_, 1)),
                  this._rowElements[0].addEventListener('focus', this._topBoundaryFocusListener),
                  this._rowElements[this._rowElements.length - 1].addEventListener(
                    'focus',
                    this._bottomBoundaryFocusListener
                  ),
                  this._refreshRowsDimensions(),
                  this._accessibilityContainer.appendChild(this._rowContainer),
                  (this._liveRegion = this._coreBrowserService.mainDocument.createElement('div')),
                  this._liveRegion.classList.add('live-region'),
                  this._liveRegion.setAttribute('aria-live', 'assertive'),
                  this._accessibilityContainer.appendChild(this._liveRegion),
                  (this._liveRegionDebouncer = this.register(
                    new h.TimeBasedDebouncer(this._renderRows.bind(this))
                  )),
                  !this._terminal.element)
                )
                  throw new Error('Cannot enable accessibility before Terminal.open')
                ;(this._terminal.element.insertAdjacentElement(
                  'afterbegin',
                  this._accessibilityContainer
                ),
                  this.register(this._terminal.onResize((_) => this._handleResize(_.rows))),
                  this.register(this._terminal.onRender((_) => this._refreshRows(_.start, _.end))),
                  this.register(this._terminal.onScroll(() => this._refreshRows())),
                  this.register(this._terminal.onA11yChar((_) => this._handleChar(_))),
                  this.register(
                    this._terminal.onLineFeed(() =>
                      this._handleChar(`
`)
                    )
                  ),
                  this.register(this._terminal.onA11yTab((_) => this._handleTab(_))),
                  this.register(this._terminal.onKey((_) => this._handleKey(_.key))),
                  this.register(this._terminal.onBlur(() => this._clearLiveRegion())),
                  this.register(
                    this._renderService.onDimensionsChange(() => this._refreshRowsDimensions())
                  ),
                  this.register(
                    (0, t.addDisposableDomListener)(document, 'selectionchange', () =>
                      this._handleSelectionChange()
                    )
                  ),
                  this.register(
                    this._coreBrowserService.onDprChange(() => this._refreshRowsDimensions())
                  ),
                  this._refreshRows(),
                  this.register(
                    (0, f.toDisposable)(() => {
                      ;(this._accessibilityContainer.remove(), (this._rowElements.length = 0))
                    })
                  ))
              }
              _handleTab(s) {
                for (let r = 0; r < s; r++) this._handleChar(' ')
              }
              _handleChar(s) {
                this._liveRegionLineCount < 21 &&
                  (this._charsToConsume.length > 0
                    ? this._charsToConsume.shift() !== s && (this._charsToAnnounce += s)
                    : (this._charsToAnnounce += s),
                  s ===
                    `
` &&
                    (this._liveRegionLineCount++,
                    this._liveRegionLineCount === 21 &&
                      (this._liveRegion.textContent += a.tooMuchOutput)))
              }
              _clearLiveRegion() {
                ;((this._liveRegion.textContent = ''), (this._liveRegionLineCount = 0))
              }
              _handleKey(s) {
                ;(this._clearLiveRegion(), /\p{Control}/u.test(s) || this._charsToConsume.push(s))
              }
              _refreshRows(s, r) {
                this._liveRegionDebouncer.refresh(s, r, this._terminal.rows)
              }
              _renderRows(s, r) {
                const d = this._terminal.buffer,
                  v = d.lines.length.toString()
                for (let _ = s; _ <= r; _++) {
                  const b = d.lines.get(d.ydisp + _),
                    p = [],
                    S = b?.translateToString(!0, void 0, void 0, p) || '',
                    L = (d.ydisp + _ + 1).toString(),
                    M = this._rowElements[_]
                  M &&
                    (S.length === 0
                      ? ((M.innerText = ' '), this._rowColumns.set(M, [0, 1]))
                      : ((M.textContent = S), this._rowColumns.set(M, p)),
                    M.setAttribute('aria-posinset', L),
                    M.setAttribute('aria-setsize', v))
                }
                this._announceCharacters()
              }
              _announceCharacters() {
                this._charsToAnnounce.length !== 0 &&
                  ((this._liveRegion.textContent += this._charsToAnnounce),
                  (this._charsToAnnounce = ''))
              }
              _handleBoundaryFocus(s, r) {
                const d = s.target,
                  v = this._rowElements[r === 0 ? 1 : this._rowElements.length - 2]
                if (
                  d.getAttribute('aria-posinset') ===
                    (r === 0 ? '1' : `${this._terminal.buffer.lines.length}`) ||
                  s.relatedTarget !== v
                )
                  return
                let _, b
                if (
                  (r === 0
                    ? ((_ = d), (b = this._rowElements.pop()), this._rowContainer.removeChild(b))
                    : ((_ = this._rowElements.shift()), (b = d), this._rowContainer.removeChild(_)),
                  _.removeEventListener('focus', this._topBoundaryFocusListener),
                  b.removeEventListener('focus', this._bottomBoundaryFocusListener),
                  r === 0)
                ) {
                  const p = this._createAccessibilityTreeNode()
                  ;(this._rowElements.unshift(p),
                    this._rowContainer.insertAdjacentElement('afterbegin', p))
                } else {
                  const p = this._createAccessibilityTreeNode()
                  ;(this._rowElements.push(p), this._rowContainer.appendChild(p))
                }
                ;(this._rowElements[0].addEventListener('focus', this._topBoundaryFocusListener),
                  this._rowElements[this._rowElements.length - 1].addEventListener(
                    'focus',
                    this._bottomBoundaryFocusListener
                  ),
                  this._terminal.scrollLines(r === 0 ? -1 : 1),
                  this._rowElements[r === 0 ? 1 : this._rowElements.length - 2].focus(),
                  s.preventDefault(),
                  s.stopImmediatePropagation())
              }
              _handleSelectionChange() {
                if (this._rowElements.length === 0) return
                const s = document.getSelection()
                if (!s) return
                if (s.isCollapsed)
                  return void (
                    this._rowContainer.contains(s.anchorNode) && this._terminal.clearSelection()
                  )
                if (!s.anchorNode || !s.focusNode)
                  return void console.error('anchorNode and/or focusNode are null')
                let r = { node: s.anchorNode, offset: s.anchorOffset },
                  d = { node: s.focusNode, offset: s.focusOffset }
                if (
                  ((r.node.compareDocumentPosition(d.node) & Node.DOCUMENT_POSITION_PRECEDING ||
                    (r.node === d.node && r.offset > d.offset)) &&
                    ([r, d] = [d, r]),
                  r.node.compareDocumentPosition(this._rowElements[0]) &
                    (Node.DOCUMENT_POSITION_CONTAINED_BY | Node.DOCUMENT_POSITION_FOLLOWING) &&
                    (r = { node: this._rowElements[0].childNodes[0], offset: 0 }),
                  !this._rowContainer.contains(r.node))
                )
                  return
                const v = this._rowElements.slice(-1)[0]
                if (
                  (d.node.compareDocumentPosition(v) &
                    (Node.DOCUMENT_POSITION_CONTAINED_BY | Node.DOCUMENT_POSITION_PRECEDING) &&
                    (d = { node: v, offset: v.textContent?.length ?? 0 }),
                  !this._rowContainer.contains(d.node))
                )
                  return
                const _ = ({ node: S, offset: L }) => {
                    const M = S instanceof Text ? S.parentNode : S
                    let P = parseInt(M?.getAttribute('aria-posinset'), 10) - 1
                    if (isNaN(P)) return (console.warn('row is invalid. Race condition?'), null)
                    const j = this._rowColumns.get(M)
                    if (!j) return (console.warn('columns is null. Race condition?'), null)
                    let D = L < j.length ? j[L] : j.slice(-1)[0] + 1
                    return (D >= this._terminal.cols && (++P, (D = 0)), { row: P, column: D })
                  },
                  b = _(r),
                  p = _(d)
                if (b && p) {
                  if (b.row > p.row || (b.row === p.row && b.column >= p.column))
                    throw new Error('invalid range')
                  this._terminal.select(
                    b.column,
                    b.row,
                    (p.row - b.row) * this._terminal.cols - b.column + p.column
                  )
                }
              }
              _handleResize(s) {
                this._rowElements[this._rowElements.length - 1].removeEventListener(
                  'focus',
                  this._bottomBoundaryFocusListener
                )
                for (let r = this._rowContainer.children.length; r < this._terminal.rows; r++)
                  ((this._rowElements[r] = this._createAccessibilityTreeNode()),
                    this._rowContainer.appendChild(this._rowElements[r]))
                for (; this._rowElements.length > s; )
                  this._rowContainer.removeChild(this._rowElements.pop())
                ;(this._rowElements[this._rowElements.length - 1].addEventListener(
                  'focus',
                  this._bottomBoundaryFocusListener
                ),
                  this._refreshRowsDimensions())
              }
              _createAccessibilityTreeNode() {
                const s = this._coreBrowserService.mainDocument.createElement('div')
                return (
                  s.setAttribute('role', 'listitem'),
                  (s.tabIndex = -1),
                  this._refreshRowDimensions(s),
                  s
                )
              }
              _refreshRowsDimensions() {
                if (this._renderService.dimensions.css.cell.height) {
                  ;((this._accessibilityContainer.style.width = `${this._renderService.dimensions.css.canvas.width}px`),
                    this._rowElements.length !== this._terminal.rows &&
                      this._handleResize(this._terminal.rows))
                  for (let s = 0; s < this._terminal.rows; s++)
                    this._refreshRowDimensions(this._rowElements[s])
                }
              }
              _refreshRowDimensions(s) {
                s.style.height = `${this._renderService.dimensions.css.cell.height}px`
              }
            })
            i.AccessibilityManager = n = l(
              [u(1, c.IInstantiationService), u(2, x.ICoreBrowserService), u(3, x.IRenderService)],
              n
            )
          },
          3614: (I, i) => {
            function o(h) {
              return h.replace(/\r?\n/g, '\r')
            }
            function l(h, f) {
              return f ? '\x1B[200~' + h + '\x1B[201~' : h
            }
            function u(h, f, x, c) {
              ;((h = l(
                (h = o(h)),
                x.decPrivateModes.bracketedPasteMode && c.rawOptions.ignoreBracketedPasteMode !== !0
              )),
                x.triggerDataEvent(h, !0),
                (f.value = ''))
            }
            function a(h, f, x) {
              const c = x.getBoundingClientRect(),
                t = h.clientX - c.left - 10,
                n = h.clientY - c.top - 10
              ;((f.style.width = '20px'),
                (f.style.height = '20px'),
                (f.style.left = `${t}px`),
                (f.style.top = `${n}px`),
                (f.style.zIndex = '1000'),
                f.focus())
            }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.rightClickHandler =
                i.moveTextAreaUnderMouseCursor =
                i.paste =
                i.handlePasteEvent =
                i.copyHandler =
                i.bracketTextForPaste =
                i.prepareTextForTerminal =
                  void 0),
              (i.prepareTextForTerminal = o),
              (i.bracketTextForPaste = l),
              (i.copyHandler = function (h, f) {
                ;(h.clipboardData && h.clipboardData.setData('text/plain', f.selectionText),
                  h.preventDefault())
              }),
              (i.handlePasteEvent = function (h, f, x, c) {
                ;(h.stopPropagation(),
                  h.clipboardData && u(h.clipboardData.getData('text/plain'), f, x, c))
              }),
              (i.paste = u),
              (i.moveTextAreaUnderMouseCursor = a),
              (i.rightClickHandler = function (h, f, x, c, t) {
                ;(a(h, f, x), t && c.rightClickSelect(h), (f.value = c.selectionText), f.select())
              }))
          },
          7239: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.ColorContrastCache = void 0))
            const l = o(1505)
            i.ColorContrastCache = class {
              constructor() {
                ;((this._color = new l.TwoKeyMap()), (this._css = new l.TwoKeyMap()))
              }
              setCss(u, a, h) {
                this._css.set(u, a, h)
              }
              getCss(u, a) {
                return this._css.get(u, a)
              }
              setColor(u, a, h) {
                this._color.set(u, a, h)
              }
              getColor(u, a) {
                return this._color.get(u, a)
              }
              clear() {
                ;(this._color.clear(), this._css.clear())
              }
            }
          },
          3656: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.addDisposableDomListener = void 0),
              (i.addDisposableDomListener = function (o, l, u, a) {
                o.addEventListener(l, u, a)
                let h = !1
                return {
                  dispose: () => {
                    h || ((h = !0), o.removeEventListener(l, u, a))
                  }
                }
              }))
          },
          3551: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (n, s, r, d) {
                  var v,
                    _ = arguments.length,
                    b = _ < 3 ? s : d === null ? (d = Object.getOwnPropertyDescriptor(s, r)) : d
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    b = Reflect.decorate(n, s, r, d)
                  else
                    for (var p = n.length - 1; p >= 0; p--)
                      (v = n[p]) && (b = (_ < 3 ? v(b) : _ > 3 ? v(s, r, b) : v(s, r)) || b)
                  return (_ > 3 && b && Object.defineProperty(s, r, b), b)
                },
              u =
                (this && this.__param) ||
                function (n, s) {
                  return function (r, d) {
                    s(r, d, n)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.Linkifier = void 0))
            const a = o(3656),
              h = o(8460),
              f = o(844),
              x = o(2585),
              c = o(4725)
            let t = (i.Linkifier = class extends f.Disposable {
              get currentLink() {
                return this._currentLink
              }
              constructor(n, s, r, d, v) {
                ;(super(),
                  (this._element = n),
                  (this._mouseService = s),
                  (this._renderService = r),
                  (this._bufferService = d),
                  (this._linkProviderService = v),
                  (this._linkCacheDisposables = []),
                  (this._isMouseOut = !0),
                  (this._wasResized = !1),
                  (this._activeLine = -1),
                  (this._onShowLinkUnderline = this.register(new h.EventEmitter())),
                  (this.onShowLinkUnderline = this._onShowLinkUnderline.event),
                  (this._onHideLinkUnderline = this.register(new h.EventEmitter())),
                  (this.onHideLinkUnderline = this._onHideLinkUnderline.event),
                  this.register((0, f.getDisposeArrayDisposable)(this._linkCacheDisposables)),
                  this.register(
                    (0, f.toDisposable)(() => {
                      ;((this._lastMouseEvent = void 0), this._activeProviderReplies?.clear())
                    })
                  ),
                  this.register(
                    this._bufferService.onResize(() => {
                      ;(this._clearCurrentLink(), (this._wasResized = !0))
                    })
                  ),
                  this.register(
                    (0, a.addDisposableDomListener)(this._element, 'mouseleave', () => {
                      ;((this._isMouseOut = !0), this._clearCurrentLink())
                    })
                  ),
                  this.register(
                    (0, a.addDisposableDomListener)(
                      this._element,
                      'mousemove',
                      this._handleMouseMove.bind(this)
                    )
                  ),
                  this.register(
                    (0, a.addDisposableDomListener)(
                      this._element,
                      'mousedown',
                      this._handleMouseDown.bind(this)
                    )
                  ),
                  this.register(
                    (0, a.addDisposableDomListener)(
                      this._element,
                      'mouseup',
                      this._handleMouseUp.bind(this)
                    )
                  ))
              }
              _handleMouseMove(n) {
                this._lastMouseEvent = n
                const s = this._positionFromMouseEvent(n, this._element, this._mouseService)
                if (!s) return
                this._isMouseOut = !1
                const r = n.composedPath()
                for (let d = 0; d < r.length; d++) {
                  const v = r[d]
                  if (v.classList.contains('xterm')) break
                  if (v.classList.contains('xterm-hover')) return
                }
                ;(this._lastBufferCell &&
                  s.x === this._lastBufferCell.x &&
                  s.y === this._lastBufferCell.y) ||
                  (this._handleHover(s), (this._lastBufferCell = s))
              }
              _handleHover(n) {
                if (this._activeLine !== n.y || this._wasResized)
                  return (
                    this._clearCurrentLink(),
                    this._askForLink(n, !1),
                    void (this._wasResized = !1)
                  )
                ;(this._currentLink && this._linkAtPosition(this._currentLink.link, n)) ||
                  (this._clearCurrentLink(), this._askForLink(n, !0))
              }
              _askForLink(n, s) {
                ;(this._activeProviderReplies && s) ||
                  (this._activeProviderReplies?.forEach((d) => {
                    d?.forEach((v) => {
                      v.link.dispose && v.link.dispose()
                    })
                  }),
                  (this._activeProviderReplies = new Map()),
                  (this._activeLine = n.y))
                let r = !1
                for (const [d, v] of this._linkProviderService.linkProviders.entries())
                  s
                    ? this._activeProviderReplies?.get(d) &&
                      (r = this._checkLinkProviderResult(d, n, r))
                    : v.provideLinks(n.y, (_) => {
                        if (this._isMouseOut) return
                        const b = _?.map((p) => ({ link: p }))
                        ;(this._activeProviderReplies?.set(d, b),
                          (r = this._checkLinkProviderResult(d, n, r)),
                          this._activeProviderReplies?.size ===
                            this._linkProviderService.linkProviders.length &&
                            this._removeIntersectingLinks(n.y, this._activeProviderReplies))
                      })
              }
              _removeIntersectingLinks(n, s) {
                const r = new Set()
                for (let d = 0; d < s.size; d++) {
                  const v = s.get(d)
                  if (v)
                    for (let _ = 0; _ < v.length; _++) {
                      const b = v[_],
                        p = b.link.range.start.y < n ? 0 : b.link.range.start.x,
                        S = b.link.range.end.y > n ? this._bufferService.cols : b.link.range.end.x
                      for (let L = p; L <= S; L++) {
                        if (r.has(L)) {
                          v.splice(_--, 1)
                          break
                        }
                        r.add(L)
                      }
                    }
                }
              }
              _checkLinkProviderResult(n, s, r) {
                if (!this._activeProviderReplies) return r
                const d = this._activeProviderReplies.get(n)
                let v = !1
                for (let _ = 0; _ < n; _++)
                  (this._activeProviderReplies.has(_) && !this._activeProviderReplies.get(_)) ||
                    (v = !0)
                if (!v && d) {
                  const _ = d.find((b) => this._linkAtPosition(b.link, s))
                  _ && ((r = !0), this._handleNewLink(_))
                }
                if (
                  this._activeProviderReplies.size ===
                    this._linkProviderService.linkProviders.length &&
                  !r
                )
                  for (let _ = 0; _ < this._activeProviderReplies.size; _++) {
                    const b = this._activeProviderReplies
                      .get(_)
                      ?.find((p) => this._linkAtPosition(p.link, s))
                    if (b) {
                      ;((r = !0), this._handleNewLink(b))
                      break
                    }
                  }
                return r
              }
              _handleMouseDown() {
                this._mouseDownLink = this._currentLink
              }
              _handleMouseUp(n) {
                if (!this._currentLink) return
                const s = this._positionFromMouseEvent(n, this._element, this._mouseService)
                s &&
                  this._mouseDownLink === this._currentLink &&
                  this._linkAtPosition(this._currentLink.link, s) &&
                  this._currentLink.link.activate(n, this._currentLink.link.text)
              }
              _clearCurrentLink(n, s) {
                this._currentLink &&
                  this._lastMouseEvent &&
                  (!n ||
                    !s ||
                    (this._currentLink.link.range.start.y >= n &&
                      this._currentLink.link.range.end.y <= s)) &&
                  (this._linkLeave(this._element, this._currentLink.link, this._lastMouseEvent),
                  (this._currentLink = void 0),
                  (0, f.disposeArray)(this._linkCacheDisposables))
              }
              _handleNewLink(n) {
                if (!this._lastMouseEvent) return
                const s = this._positionFromMouseEvent(
                  this._lastMouseEvent,
                  this._element,
                  this._mouseService
                )
                s &&
                  this._linkAtPosition(n.link, s) &&
                  ((this._currentLink = n),
                  (this._currentLink.state = {
                    decorations: {
                      underline: n.link.decorations === void 0 || n.link.decorations.underline,
                      pointerCursor:
                        n.link.decorations === void 0 || n.link.decorations.pointerCursor
                    },
                    isHovered: !0
                  }),
                  this._linkHover(this._element, n.link, this._lastMouseEvent),
                  (n.link.decorations = {}),
                  Object.defineProperties(n.link.decorations, {
                    pointerCursor: {
                      get: () => this._currentLink?.state?.decorations.pointerCursor,
                      set: (r) => {
                        this._currentLink?.state &&
                          this._currentLink.state.decorations.pointerCursor !== r &&
                          ((this._currentLink.state.decorations.pointerCursor = r),
                          this._currentLink.state.isHovered &&
                            this._element.classList.toggle('xterm-cursor-pointer', r))
                      }
                    },
                    underline: {
                      get: () => this._currentLink?.state?.decorations.underline,
                      set: (r) => {
                        this._currentLink?.state &&
                          this._currentLink?.state?.decorations.underline !== r &&
                          ((this._currentLink.state.decorations.underline = r),
                          this._currentLink.state.isHovered && this._fireUnderlineEvent(n.link, r))
                      }
                    }
                  }),
                  this._linkCacheDisposables.push(
                    this._renderService.onRenderedViewportChange((r) => {
                      if (!this._currentLink) return
                      const d = r.start === 0 ? 0 : r.start + 1 + this._bufferService.buffer.ydisp,
                        v = this._bufferService.buffer.ydisp + 1 + r.end
                      if (
                        this._currentLink.link.range.start.y >= d &&
                        this._currentLink.link.range.end.y <= v &&
                        (this._clearCurrentLink(d, v), this._lastMouseEvent)
                      ) {
                        const _ = this._positionFromMouseEvent(
                          this._lastMouseEvent,
                          this._element,
                          this._mouseService
                        )
                        _ && this._askForLink(_, !1)
                      }
                    })
                  ))
              }
              _linkHover(n, s, r) {
                ;(this._currentLink?.state &&
                  ((this._currentLink.state.isHovered = !0),
                  this._currentLink.state.decorations.underline && this._fireUnderlineEvent(s, !0),
                  this._currentLink.state.decorations.pointerCursor &&
                    n.classList.add('xterm-cursor-pointer')),
                  s.hover && s.hover(r, s.text))
              }
              _fireUnderlineEvent(n, s) {
                const r = n.range,
                  d = this._bufferService.buffer.ydisp,
                  v = this._createLinkUnderlineEvent(
                    r.start.x - 1,
                    r.start.y - d - 1,
                    r.end.x,
                    r.end.y - d - 1,
                    void 0
                  )
                ;(s ? this._onShowLinkUnderline : this._onHideLinkUnderline).fire(v)
              }
              _linkLeave(n, s, r) {
                ;(this._currentLink?.state &&
                  ((this._currentLink.state.isHovered = !1),
                  this._currentLink.state.decorations.underline && this._fireUnderlineEvent(s, !1),
                  this._currentLink.state.decorations.pointerCursor &&
                    n.classList.remove('xterm-cursor-pointer')),
                  s.leave && s.leave(r, s.text))
              }
              _linkAtPosition(n, s) {
                const r = n.range.start.y * this._bufferService.cols + n.range.start.x,
                  d = n.range.end.y * this._bufferService.cols + n.range.end.x,
                  v = s.y * this._bufferService.cols + s.x
                return r <= v && v <= d
              }
              _positionFromMouseEvent(n, s, r) {
                const d = r.getCoords(n, s, this._bufferService.cols, this._bufferService.rows)
                if (d) return { x: d[0], y: d[1] + this._bufferService.buffer.ydisp }
              }
              _createLinkUnderlineEvent(n, s, r, d, v) {
                return { x1: n, y1: s, x2: r, y2: d, cols: this._bufferService.cols, fg: v }
              }
            })
            i.Linkifier = t = l(
              [
                u(1, c.IMouseService),
                u(2, c.IRenderService),
                u(3, x.IBufferService),
                u(4, c.ILinkProviderService)
              ],
              t
            )
          },
          9042: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.tooMuchOutput = i.promptLabel = void 0),
              (i.promptLabel = 'Terminal input'),
              (i.tooMuchOutput = 'Too much output to announce, navigate to rows manually to read'))
          },
          3730: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (c, t, n, s) {
                  var r,
                    d = arguments.length,
                    v = d < 3 ? t : s === null ? (s = Object.getOwnPropertyDescriptor(t, n)) : s
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    v = Reflect.decorate(c, t, n, s)
                  else
                    for (var _ = c.length - 1; _ >= 0; _--)
                      (r = c[_]) && (v = (d < 3 ? r(v) : d > 3 ? r(t, n, v) : r(t, n)) || v)
                  return (d > 3 && v && Object.defineProperty(t, n, v), v)
                },
              u =
                (this && this.__param) ||
                function (c, t) {
                  return function (n, s) {
                    t(n, s, c)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.OscLinkProvider = void 0))
            const a = o(511),
              h = o(2585)
            let f = (i.OscLinkProvider = class {
              constructor(c, t, n) {
                ;((this._bufferService = c), (this._optionsService = t), (this._oscLinkService = n))
              }
              provideLinks(c, t) {
                const n = this._bufferService.buffer.lines.get(c - 1)
                if (!n) return void t(void 0)
                const s = [],
                  r = this._optionsService.rawOptions.linkHandler,
                  d = new a.CellData(),
                  v = n.getTrimmedLength()
                let _ = -1,
                  b = -1,
                  p = !1
                for (let S = 0; S < v; S++)
                  if (b !== -1 || n.hasContent(S)) {
                    if ((n.loadCell(S, d), d.hasExtendedAttrs() && d.extended.urlId)) {
                      if (b === -1) {
                        ;((b = S), (_ = d.extended.urlId))
                        continue
                      }
                      p = d.extended.urlId !== _
                    } else b !== -1 && (p = !0)
                    if (p || (b !== -1 && S === v - 1)) {
                      const L = this._oscLinkService.getLinkData(_)?.uri
                      if (L) {
                        const M = {
                          start: { x: b + 1, y: c },
                          end: { x: S + (p || S !== v - 1 ? 0 : 1), y: c }
                        }
                        let P = !1
                        if (!r?.allowNonHttpProtocols)
                          try {
                            const j = new URL(L)
                            ;['http:', 'https:'].includes(j.protocol) || (P = !0)
                          } catch {
                            P = !0
                          }
                        P ||
                          s.push({
                            text: L,
                            range: M,
                            activate: (j, D) => (r ? r.activate(j, D, M) : x(0, D)),
                            hover: (j, D) => r?.hover?.(j, D, M),
                            leave: (j, D) => r?.leave?.(j, D, M)
                          })
                      }
                      ;((p = !1),
                        d.hasExtendedAttrs() && d.extended.urlId
                          ? ((b = S), (_ = d.extended.urlId))
                          : ((b = -1), (_ = -1)))
                    }
                  }
                t(s)
              }
            })
            function x(c, t) {
              if (
                confirm(`Do you want to navigate to ${t}?

WARNING: This link could potentially be dangerous`)
              ) {
                const n = window.open()
                if (n) {
                  try {
                    n.opener = null
                  } catch {}
                  n.location.href = t
                } else console.warn('Opening link blocked as opener could not be cleared')
              }
            }
            i.OscLinkProvider = f = l(
              [u(0, h.IBufferService), u(1, h.IOptionsService), u(2, h.IOscLinkService)],
              f
            )
          },
          6193: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.RenderDebouncer = void 0),
              (i.RenderDebouncer = class {
                constructor(o, l) {
                  ;((this._renderCallback = o),
                    (this._coreBrowserService = l),
                    (this._refreshCallbacks = []))
                }
                dispose() {
                  this._animationFrame &&
                    (this._coreBrowserService.window.cancelAnimationFrame(this._animationFrame),
                    (this._animationFrame = void 0))
                }
                addRefreshCallback(o) {
                  return (
                    this._refreshCallbacks.push(o),
                    this._animationFrame ||
                      (this._animationFrame = this._coreBrowserService.window.requestAnimationFrame(
                        () => this._innerRefresh()
                      )),
                    this._animationFrame
                  )
                }
                refresh(o, l, u) {
                  ;((this._rowCount = u),
                    (o = o !== void 0 ? o : 0),
                    (l = l !== void 0 ? l : this._rowCount - 1),
                    (this._rowStart = this._rowStart !== void 0 ? Math.min(this._rowStart, o) : o),
                    (this._rowEnd = this._rowEnd !== void 0 ? Math.max(this._rowEnd, l) : l),
                    this._animationFrame ||
                      (this._animationFrame = this._coreBrowserService.window.requestAnimationFrame(
                        () => this._innerRefresh()
                      )))
                }
                _innerRefresh() {
                  if (
                    ((this._animationFrame = void 0),
                    this._rowStart === void 0 ||
                      this._rowEnd === void 0 ||
                      this._rowCount === void 0)
                  )
                    return void this._runRefreshCallbacks()
                  const o = Math.max(this._rowStart, 0),
                    l = Math.min(this._rowEnd, this._rowCount - 1)
                  ;((this._rowStart = void 0),
                    (this._rowEnd = void 0),
                    this._renderCallback(o, l),
                    this._runRefreshCallbacks())
                }
                _runRefreshCallbacks() {
                  for (const o of this._refreshCallbacks) o(0)
                  this._refreshCallbacks = []
                }
              }))
          },
          3236: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.Terminal = void 0))
            const l = o(3614),
              u = o(3656),
              a = o(3551),
              h = o(9042),
              f = o(3730),
              x = o(1680),
              c = o(3107),
              t = o(5744),
              n = o(2950),
              s = o(1296),
              r = o(428),
              d = o(4269),
              v = o(5114),
              _ = o(8934),
              b = o(3230),
              p = o(9312),
              S = o(4725),
              L = o(6731),
              M = o(8055),
              P = o(8969),
              j = o(8460),
              D = o(844),
              O = o(6114),
              $ = o(8437),
              F = o(2584),
              W = o(7399),
              C = o(5941),
              A = o(9074),
              N = o(2585),
              B = o(5435),
              z = o(4567),
              K = o(779)
            class J extends P.CoreTerminal {
              get onFocus() {
                return this._onFocus.event
              }
              get onBlur() {
                return this._onBlur.event
              }
              get onA11yChar() {
                return this._onA11yCharEmitter.event
              }
              get onA11yTab() {
                return this._onA11yTabEmitter.event
              }
              get onWillOpen() {
                return this._onWillOpen.event
              }
              constructor(H = {}) {
                ;(super(H),
                  (this.browser = O),
                  (this._keyDownHandled = !1),
                  (this._keyDownSeen = !1),
                  (this._keyPressHandled = !1),
                  (this._unprocessedDeadKey = !1),
                  (this._accessibilityManager = this.register(new D.MutableDisposable())),
                  (this._onCursorMove = this.register(new j.EventEmitter())),
                  (this.onCursorMove = this._onCursorMove.event),
                  (this._onKey = this.register(new j.EventEmitter())),
                  (this.onKey = this._onKey.event),
                  (this._onRender = this.register(new j.EventEmitter())),
                  (this.onRender = this._onRender.event),
                  (this._onSelectionChange = this.register(new j.EventEmitter())),
                  (this.onSelectionChange = this._onSelectionChange.event),
                  (this._onTitleChange = this.register(new j.EventEmitter())),
                  (this.onTitleChange = this._onTitleChange.event),
                  (this._onBell = this.register(new j.EventEmitter())),
                  (this.onBell = this._onBell.event),
                  (this._onFocus = this.register(new j.EventEmitter())),
                  (this._onBlur = this.register(new j.EventEmitter())),
                  (this._onA11yCharEmitter = this.register(new j.EventEmitter())),
                  (this._onA11yTabEmitter = this.register(new j.EventEmitter())),
                  (this._onWillOpen = this.register(new j.EventEmitter())),
                  this._setup(),
                  (this._decorationService = this._instantiationService.createInstance(
                    A.DecorationService
                  )),
                  this._instantiationService.setService(
                    N.IDecorationService,
                    this._decorationService
                  ),
                  (this._linkProviderService = this._instantiationService.createInstance(
                    K.LinkProviderService
                  )),
                  this._instantiationService.setService(
                    S.ILinkProviderService,
                    this._linkProviderService
                  ),
                  this._linkProviderService.registerLinkProvider(
                    this._instantiationService.createInstance(f.OscLinkProvider)
                  ),
                  this.register(this._inputHandler.onRequestBell(() => this._onBell.fire())),
                  this.register(
                    this._inputHandler.onRequestRefreshRows((E, G) => this.refresh(E, G))
                  ),
                  this.register(this._inputHandler.onRequestSendFocus(() => this._reportFocus())),
                  this.register(this._inputHandler.onRequestReset(() => this.reset())),
                  this.register(
                    this._inputHandler.onRequestWindowsOptionsReport((E) =>
                      this._reportWindowsOptions(E)
                    )
                  ),
                  this.register(this._inputHandler.onColor((E) => this._handleColorEvent(E))),
                  this.register(
                    (0, j.forwardEvent)(this._inputHandler.onCursorMove, this._onCursorMove)
                  ),
                  this.register(
                    (0, j.forwardEvent)(this._inputHandler.onTitleChange, this._onTitleChange)
                  ),
                  this.register(
                    (0, j.forwardEvent)(this._inputHandler.onA11yChar, this._onA11yCharEmitter)
                  ),
                  this.register(
                    (0, j.forwardEvent)(this._inputHandler.onA11yTab, this._onA11yTabEmitter)
                  ),
                  this.register(
                    this._bufferService.onResize((E) => this._afterResize(E.cols, E.rows))
                  ),
                  this.register(
                    (0, D.toDisposable)(() => {
                      ;((this._customKeyEventHandler = void 0),
                        this.element?.parentNode?.removeChild(this.element))
                    })
                  ))
              }
              _handleColorEvent(H) {
                if (this._themeService)
                  for (const E of H) {
                    let G,
                      q = ''
                    switch (E.index) {
                      case 256:
                        ;((G = 'foreground'), (q = '10'))
                        break
                      case 257:
                        ;((G = 'background'), (q = '11'))
                        break
                      case 258:
                        ;((G = 'cursor'), (q = '12'))
                        break
                      default:
                        ;((G = 'ansi'), (q = '4;' + E.index))
                    }
                    switch (E.type) {
                      case 0:
                        const Z = M.color.toColorRGB(
                          G === 'ansi'
                            ? this._themeService.colors.ansi[E.index]
                            : this._themeService.colors[G]
                        )
                        this.coreService.triggerDataEvent(
                          `${F.C0.ESC}]${q};${(0, C.toRgbString)(Z)}${F.C1_ESCAPED.ST}`
                        )
                        break
                      case 1:
                        if (G === 'ansi')
                          this._themeService.modifyColors(
                            (Y) => (Y.ansi[E.index] = M.channels.toColor(...E.color))
                          )
                        else {
                          const Y = G
                          this._themeService.modifyColors(
                            (V) => (V[Y] = M.channels.toColor(...E.color))
                          )
                        }
                        break
                      case 2:
                        this._themeService.restoreColor(E.index)
                    }
                  }
              }
              _setup() {
                ;(super._setup(), (this._customKeyEventHandler = void 0))
              }
              get buffer() {
                return this.buffers.active
              }
              focus() {
                this.textarea && this.textarea.focus({ preventScroll: !0 })
              }
              _handleScreenReaderModeOptionChange(H) {
                H
                  ? !this._accessibilityManager.value &&
                    this._renderService &&
                    (this._accessibilityManager.value = this._instantiationService.createInstance(
                      z.AccessibilityManager,
                      this
                    ))
                  : this._accessibilityManager.clear()
              }
              _handleTextAreaFocus(H) {
                ;(this.coreService.decPrivateModes.sendFocus &&
                  this.coreService.triggerDataEvent(F.C0.ESC + '[I'),
                  this.element.classList.add('focus'),
                  this._showCursor(),
                  this._onFocus.fire())
              }
              blur() {
                return this.textarea?.blur()
              }
              _handleTextAreaBlur() {
                ;((this.textarea.value = ''),
                  this.refresh(this.buffer.y, this.buffer.y),
                  this.coreService.decPrivateModes.sendFocus &&
                    this.coreService.triggerDataEvent(F.C0.ESC + '[O'),
                  this.element.classList.remove('focus'),
                  this._onBlur.fire())
              }
              _syncTextArea() {
                if (
                  !this.textarea ||
                  !this.buffer.isCursorInViewport ||
                  this._compositionHelper.isComposing ||
                  !this._renderService
                )
                  return
                const H = this.buffer.ybase + this.buffer.y,
                  E = this.buffer.lines.get(H)
                if (!E) return
                const G = Math.min(this.buffer.x, this.cols - 1),
                  q = this._renderService.dimensions.css.cell.height,
                  Z = E.getWidth(G),
                  Y = this._renderService.dimensions.css.cell.width * Z,
                  V = this.buffer.y * this._renderService.dimensions.css.cell.height,
                  se = G * this._renderService.dimensions.css.cell.width
                ;((this.textarea.style.left = se + 'px'),
                  (this.textarea.style.top = V + 'px'),
                  (this.textarea.style.width = Y + 'px'),
                  (this.textarea.style.height = q + 'px'),
                  (this.textarea.style.lineHeight = q + 'px'),
                  (this.textarea.style.zIndex = '-5'))
              }
              _initGlobal() {
                ;(this._bindKeys(),
                  this.register(
                    (0, u.addDisposableDomListener)(this.element, 'copy', (E) => {
                      this.hasSelection() && (0, l.copyHandler)(E, this._selectionService)
                    })
                  ))
                const H = (E) =>
                  (0, l.handlePasteEvent)(E, this.textarea, this.coreService, this.optionsService)
                ;(this.register((0, u.addDisposableDomListener)(this.textarea, 'paste', H)),
                  this.register((0, u.addDisposableDomListener)(this.element, 'paste', H)),
                  O.isFirefox
                    ? this.register(
                        (0, u.addDisposableDomListener)(this.element, 'mousedown', (E) => {
                          E.button === 2 &&
                            (0, l.rightClickHandler)(
                              E,
                              this.textarea,
                              this.screenElement,
                              this._selectionService,
                              this.options.rightClickSelectsWord
                            )
                        })
                      )
                    : this.register(
                        (0, u.addDisposableDomListener)(this.element, 'contextmenu', (E) => {
                          ;(0, l.rightClickHandler)(
                            E,
                            this.textarea,
                            this.screenElement,
                            this._selectionService,
                            this.options.rightClickSelectsWord
                          )
                        })
                      ),
                  O.isLinux &&
                    this.register(
                      (0, u.addDisposableDomListener)(this.element, 'auxclick', (E) => {
                        E.button === 1 &&
                          (0, l.moveTextAreaUnderMouseCursor)(E, this.textarea, this.screenElement)
                      })
                    ))
              }
              _bindKeys() {
                ;(this.register(
                  (0, u.addDisposableDomListener)(this.textarea, 'keyup', (H) => this._keyUp(H), !0)
                ),
                  this.register(
                    (0, u.addDisposableDomListener)(
                      this.textarea,
                      'keydown',
                      (H) => this._keyDown(H),
                      !0
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(
                      this.textarea,
                      'keypress',
                      (H) => this._keyPress(H),
                      !0
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(this.textarea, 'compositionstart', () =>
                      this._compositionHelper.compositionstart()
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(this.textarea, 'compositionupdate', (H) =>
                      this._compositionHelper.compositionupdate(H)
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(this.textarea, 'compositionend', () =>
                      this._compositionHelper.compositionend()
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(
                      this.textarea,
                      'input',
                      (H) => this._inputEvent(H),
                      !0
                    )
                  ),
                  this.register(
                    this.onRender(() => this._compositionHelper.updateCompositionElements())
                  ))
              }
              open(H) {
                if (!H) throw new Error('Terminal requires a parent element.')
                if (
                  (H.isConnected ||
                    this._logService.debug(
                      'Terminal.open was called on an element that was not attached to the DOM'
                    ),
                  this.element?.ownerDocument.defaultView && this._coreBrowserService)
                )
                  return void (
                    this.element.ownerDocument.defaultView !== this._coreBrowserService.window &&
                    (this._coreBrowserService.window = this.element.ownerDocument.defaultView)
                  )
                ;((this._document = H.ownerDocument),
                  this.options.documentOverride &&
                    this.options.documentOverride instanceof Document &&
                    (this._document = this.optionsService.rawOptions.documentOverride),
                  (this.element = this._document.createElement('div')),
                  (this.element.dir = 'ltr'),
                  this.element.classList.add('terminal'),
                  this.element.classList.add('xterm'),
                  H.appendChild(this.element))
                const E = this._document.createDocumentFragment()
                ;((this._viewportElement = this._document.createElement('div')),
                  this._viewportElement.classList.add('xterm-viewport'),
                  E.appendChild(this._viewportElement),
                  (this._viewportScrollArea = this._document.createElement('div')),
                  this._viewportScrollArea.classList.add('xterm-scroll-area'),
                  this._viewportElement.appendChild(this._viewportScrollArea),
                  (this.screenElement = this._document.createElement('div')),
                  this.screenElement.classList.add('xterm-screen'),
                  this.register(
                    (0, u.addDisposableDomListener)(this.screenElement, 'mousemove', (G) =>
                      this.updateCursorStyle(G)
                    )
                  ),
                  (this._helperContainer = this._document.createElement('div')),
                  this._helperContainer.classList.add('xterm-helpers'),
                  this.screenElement.appendChild(this._helperContainer),
                  E.appendChild(this.screenElement),
                  (this.textarea = this._document.createElement('textarea')),
                  this.textarea.classList.add('xterm-helper-textarea'),
                  this.textarea.setAttribute('aria-label', h.promptLabel),
                  O.isChromeOS || this.textarea.setAttribute('aria-multiline', 'false'),
                  this.textarea.setAttribute('autocorrect', 'off'),
                  this.textarea.setAttribute('autocapitalize', 'off'),
                  this.textarea.setAttribute('spellcheck', 'false'),
                  (this.textarea.tabIndex = 0),
                  (this._coreBrowserService = this.register(
                    this._instantiationService.createInstance(
                      v.CoreBrowserService,
                      this.textarea,
                      H.ownerDocument.defaultView ?? window,
                      (this._document ?? typeof window < 'u') ? window.document : null
                    )
                  )),
                  this._instantiationService.setService(
                    S.ICoreBrowserService,
                    this._coreBrowserService
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(this.textarea, 'focus', (G) =>
                      this._handleTextAreaFocus(G)
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(this.textarea, 'blur', () =>
                      this._handleTextAreaBlur()
                    )
                  ),
                  this._helperContainer.appendChild(this.textarea),
                  (this._charSizeService = this._instantiationService.createInstance(
                    r.CharSizeService,
                    this._document,
                    this._helperContainer
                  )),
                  this._instantiationService.setService(S.ICharSizeService, this._charSizeService),
                  (this._themeService = this._instantiationService.createInstance(L.ThemeService)),
                  this._instantiationService.setService(S.IThemeService, this._themeService),
                  (this._characterJoinerService = this._instantiationService.createInstance(
                    d.CharacterJoinerService
                  )),
                  this._instantiationService.setService(
                    S.ICharacterJoinerService,
                    this._characterJoinerService
                  ),
                  (this._renderService = this.register(
                    this._instantiationService.createInstance(
                      b.RenderService,
                      this.rows,
                      this.screenElement
                    )
                  )),
                  this._instantiationService.setService(S.IRenderService, this._renderService),
                  this.register(
                    this._renderService.onRenderedViewportChange((G) => this._onRender.fire(G))
                  ),
                  this.onResize((G) => this._renderService.resize(G.cols, G.rows)),
                  (this._compositionView = this._document.createElement('div')),
                  this._compositionView.classList.add('composition-view'),
                  (this._compositionHelper = this._instantiationService.createInstance(
                    n.CompositionHelper,
                    this.textarea,
                    this._compositionView
                  )),
                  this._helperContainer.appendChild(this._compositionView),
                  (this._mouseService = this._instantiationService.createInstance(_.MouseService)),
                  this._instantiationService.setService(S.IMouseService, this._mouseService),
                  (this.linkifier = this.register(
                    this._instantiationService.createInstance(a.Linkifier, this.screenElement)
                  )),
                  this.element.appendChild(E))
                try {
                  this._onWillOpen.fire(this.element)
                } catch {}
                ;(this._renderService.hasRenderer() ||
                  this._renderService.setRenderer(this._createRenderer()),
                  (this.viewport = this._instantiationService.createInstance(
                    x.Viewport,
                    this._viewportElement,
                    this._viewportScrollArea
                  )),
                  this.viewport.onRequestScrollLines((G) =>
                    this.scrollLines(G.amount, G.suppressScrollEvent, 1)
                  ),
                  this.register(
                    this._inputHandler.onRequestSyncScrollBar(() => this.viewport.syncScrollArea())
                  ),
                  this.register(this.viewport),
                  this.register(
                    this.onCursorMove(() => {
                      ;(this._renderService.handleCursorMove(), this._syncTextArea())
                    })
                  ),
                  this.register(
                    this.onResize(() => this._renderService.handleResize(this.cols, this.rows))
                  ),
                  this.register(this.onBlur(() => this._renderService.handleBlur())),
                  this.register(this.onFocus(() => this._renderService.handleFocus())),
                  this.register(
                    this._renderService.onDimensionsChange(() => this.viewport.syncScrollArea())
                  ),
                  (this._selectionService = this.register(
                    this._instantiationService.createInstance(
                      p.SelectionService,
                      this.element,
                      this.screenElement,
                      this.linkifier
                    )
                  )),
                  this._instantiationService.setService(
                    S.ISelectionService,
                    this._selectionService
                  ),
                  this.register(
                    this._selectionService.onRequestScrollLines((G) =>
                      this.scrollLines(G.amount, G.suppressScrollEvent)
                    )
                  ),
                  this.register(
                    this._selectionService.onSelectionChange(() => this._onSelectionChange.fire())
                  ),
                  this.register(
                    this._selectionService.onRequestRedraw((G) =>
                      this._renderService.handleSelectionChanged(G.start, G.end, G.columnSelectMode)
                    )
                  ),
                  this.register(
                    this._selectionService.onLinuxMouseSelection((G) => {
                      ;((this.textarea.value = G), this.textarea.focus(), this.textarea.select())
                    })
                  ),
                  this.register(
                    this._onScroll.event((G) => {
                      ;(this.viewport.syncScrollArea(), this._selectionService.refresh())
                    })
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(this._viewportElement, 'scroll', () =>
                      this._selectionService.refresh()
                    )
                  ),
                  this.register(
                    this._instantiationService.createInstance(
                      c.BufferDecorationRenderer,
                      this.screenElement
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(this.element, 'mousedown', (G) =>
                      this._selectionService.handleMouseDown(G)
                    )
                  ),
                  this.coreMouseService.areMouseEventsActive
                    ? (this._selectionService.disable(),
                      this.element.classList.add('enable-mouse-events'))
                    : this._selectionService.enable(),
                  this.options.screenReaderMode &&
                    (this._accessibilityManager.value = this._instantiationService.createInstance(
                      z.AccessibilityManager,
                      this
                    )),
                  this.register(
                    this.optionsService.onSpecificOptionChange('screenReaderMode', (G) =>
                      this._handleScreenReaderModeOptionChange(G)
                    )
                  ),
                  this.options.overviewRulerWidth &&
                    (this._overviewRulerRenderer = this.register(
                      this._instantiationService.createInstance(
                        t.OverviewRulerRenderer,
                        this._viewportElement,
                        this.screenElement
                      )
                    )),
                  this.optionsService.onSpecificOptionChange('overviewRulerWidth', (G) => {
                    !this._overviewRulerRenderer &&
                      G &&
                      this._viewportElement &&
                      this.screenElement &&
                      (this._overviewRulerRenderer = this.register(
                        this._instantiationService.createInstance(
                          t.OverviewRulerRenderer,
                          this._viewportElement,
                          this.screenElement
                        )
                      ))
                  }),
                  this._charSizeService.measure(),
                  this.refresh(0, this.rows - 1),
                  this._initGlobal(),
                  this.bindMouse())
              }
              _createRenderer() {
                return this._instantiationService.createInstance(
                  s.DomRenderer,
                  this,
                  this._document,
                  this.element,
                  this.screenElement,
                  this._viewportElement,
                  this._helperContainer,
                  this.linkifier
                )
              }
              bindMouse() {
                const H = this,
                  E = this.element
                function G(Y) {
                  const V = H._mouseService.getMouseReportCoords(Y, H.screenElement)
                  if (!V) return !1
                  let se, ne
                  switch (Y.overrideType || Y.type) {
                    case 'mousemove':
                      ;((ne = 32),
                        Y.buttons === void 0
                          ? ((se = 3), Y.button !== void 0 && (se = Y.button < 3 ? Y.button : 3))
                          : (se = 1 & Y.buttons ? 0 : 4 & Y.buttons ? 1 : 2 & Y.buttons ? 2 : 3))
                      break
                    case 'mouseup':
                      ;((ne = 0), (se = Y.button < 3 ? Y.button : 3))
                      break
                    case 'mousedown':
                      ;((ne = 1), (se = Y.button < 3 ? Y.button : 3))
                      break
                    case 'wheel':
                      if (
                        (H._customWheelEventHandler && H._customWheelEventHandler(Y) === !1) ||
                        H.viewport.getLinesScrolled(Y) === 0
                      )
                        return !1
                      ;((ne = Y.deltaY < 0 ? 0 : 1), (se = 4))
                      break
                    default:
                      return !1
                  }
                  return (
                    !(ne === void 0 || se === void 0 || se > 4) &&
                    H.coreMouseService.triggerMouseEvent({
                      col: V.col,
                      row: V.row,
                      x: V.x,
                      y: V.y,
                      button: se,
                      action: ne,
                      ctrl: Y.ctrlKey,
                      alt: Y.altKey,
                      shift: Y.shiftKey
                    })
                  )
                }
                const q = { mouseup: null, wheel: null, mousedrag: null, mousemove: null },
                  Z = {
                    mouseup: (Y) => (
                      G(Y),
                      Y.buttons ||
                        (this._document.removeEventListener('mouseup', q.mouseup),
                        q.mousedrag &&
                          this._document.removeEventListener('mousemove', q.mousedrag)),
                      this.cancel(Y)
                    ),
                    wheel: (Y) => (G(Y), this.cancel(Y, !0)),
                    mousedrag: (Y) => {
                      Y.buttons && G(Y)
                    },
                    mousemove: (Y) => {
                      Y.buttons || G(Y)
                    }
                  }
                ;(this.register(
                  this.coreMouseService.onProtocolChange((Y) => {
                    ;(Y
                      ? (this.optionsService.rawOptions.logLevel === 'debug' &&
                          this._logService.debug(
                            'Binding to mouse events:',
                            this.coreMouseService.explainEvents(Y)
                          ),
                        this.element.classList.add('enable-mouse-events'),
                        this._selectionService.disable())
                      : (this._logService.debug('Unbinding from mouse events.'),
                        this.element.classList.remove('enable-mouse-events'),
                        this._selectionService.enable()),
                      8 & Y
                        ? q.mousemove ||
                          (E.addEventListener('mousemove', Z.mousemove),
                          (q.mousemove = Z.mousemove))
                        : (E.removeEventListener('mousemove', q.mousemove), (q.mousemove = null)),
                      16 & Y
                        ? q.wheel ||
                          (E.addEventListener('wheel', Z.wheel, { passive: !1 }),
                          (q.wheel = Z.wheel))
                        : (E.removeEventListener('wheel', q.wheel), (q.wheel = null)),
                      2 & Y
                        ? q.mouseup || (q.mouseup = Z.mouseup)
                        : (this._document.removeEventListener('mouseup', q.mouseup),
                          (q.mouseup = null)),
                      4 & Y
                        ? q.mousedrag || (q.mousedrag = Z.mousedrag)
                        : (this._document.removeEventListener('mousemove', q.mousedrag),
                          (q.mousedrag = null)))
                  })
                ),
                  (this.coreMouseService.activeProtocol = this.coreMouseService.activeProtocol),
                  this.register(
                    (0, u.addDisposableDomListener)(E, 'mousedown', (Y) => {
                      if (
                        (Y.preventDefault(),
                        this.focus(),
                        this.coreMouseService.areMouseEventsActive &&
                          !this._selectionService.shouldForceSelection(Y))
                      )
                        return (
                          G(Y),
                          q.mouseup && this._document.addEventListener('mouseup', q.mouseup),
                          q.mousedrag && this._document.addEventListener('mousemove', q.mousedrag),
                          this.cancel(Y)
                        )
                    })
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(
                      E,
                      'wheel',
                      (Y) => {
                        if (!q.wheel) {
                          if (
                            this._customWheelEventHandler &&
                            this._customWheelEventHandler(Y) === !1
                          )
                            return !1
                          if (!this.buffer.hasScrollback) {
                            const V = this.viewport.getLinesScrolled(Y)
                            if (V === 0) return
                            const se =
                              F.C0.ESC +
                              (this.coreService.decPrivateModes.applicationCursorKeys ? 'O' : '[') +
                              (Y.deltaY < 0 ? 'A' : 'B')
                            let ne = ''
                            for (let ue = 0; ue < Math.abs(V); ue++) ne += se
                            return (this.coreService.triggerDataEvent(ne, !0), this.cancel(Y, !0))
                          }
                          return this.viewport.handleWheel(Y) ? this.cancel(Y) : void 0
                        }
                      },
                      { passive: !1 }
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(
                      E,
                      'touchstart',
                      (Y) => {
                        if (!this.coreMouseService.areMouseEventsActive)
                          return (this.viewport.handleTouchStart(Y), this.cancel(Y))
                      },
                      { passive: !0 }
                    )
                  ),
                  this.register(
                    (0, u.addDisposableDomListener)(
                      E,
                      'touchmove',
                      (Y) => {
                        if (!this.coreMouseService.areMouseEventsActive)
                          return this.viewport.handleTouchMove(Y) ? void 0 : this.cancel(Y)
                      },
                      { passive: !1 }
                    )
                  ))
              }
              refresh(H, E) {
                this._renderService?.refreshRows(H, E)
              }
              updateCursorStyle(H) {
                this._selectionService?.shouldColumnSelect(H)
                  ? this.element.classList.add('column-select')
                  : this.element.classList.remove('column-select')
              }
              _showCursor() {
                this.coreService.isCursorInitialized ||
                  ((this.coreService.isCursorInitialized = !0),
                  this.refresh(this.buffer.y, this.buffer.y))
              }
              scrollLines(H, E, G = 0) {
                G === 1
                  ? (super.scrollLines(H, E, G), this.refresh(0, this.rows - 1))
                  : this.viewport?.scrollLines(H)
              }
              paste(H) {
                ;(0, l.paste)(H, this.textarea, this.coreService, this.optionsService)
              }
              attachCustomKeyEventHandler(H) {
                this._customKeyEventHandler = H
              }
              attachCustomWheelEventHandler(H) {
                this._customWheelEventHandler = H
              }
              registerLinkProvider(H) {
                return this._linkProviderService.registerLinkProvider(H)
              }
              registerCharacterJoiner(H) {
                if (!this._characterJoinerService) throw new Error('Terminal must be opened first')
                const E = this._characterJoinerService.register(H)
                return (this.refresh(0, this.rows - 1), E)
              }
              deregisterCharacterJoiner(H) {
                if (!this._characterJoinerService) throw new Error('Terminal must be opened first')
                this._characterJoinerService.deregister(H) && this.refresh(0, this.rows - 1)
              }
              get markers() {
                return this.buffer.markers
              }
              registerMarker(H) {
                return this.buffer.addMarker(this.buffer.ybase + this.buffer.y + H)
              }
              registerDecoration(H) {
                return this._decorationService.registerDecoration(H)
              }
              hasSelection() {
                return !!this._selectionService && this._selectionService.hasSelection
              }
              select(H, E, G) {
                this._selectionService.setSelection(H, E, G)
              }
              getSelection() {
                return this._selectionService ? this._selectionService.selectionText : ''
              }
              getSelectionPosition() {
                if (this._selectionService && this._selectionService.hasSelection)
                  return {
                    start: {
                      x: this._selectionService.selectionStart[0],
                      y: this._selectionService.selectionStart[1]
                    },
                    end: {
                      x: this._selectionService.selectionEnd[0],
                      y: this._selectionService.selectionEnd[1]
                    }
                  }
              }
              clearSelection() {
                this._selectionService?.clearSelection()
              }
              selectAll() {
                this._selectionService?.selectAll()
              }
              selectLines(H, E) {
                this._selectionService?.selectLines(H, E)
              }
              _keyDown(H) {
                if (
                  ((this._keyDownHandled = !1),
                  (this._keyDownSeen = !0),
                  this._customKeyEventHandler && this._customKeyEventHandler(H) === !1)
                )
                  return !1
                const E = this.browser.isMac && this.options.macOptionIsMeta && H.altKey
                if (!E && !this._compositionHelper.keydown(H))
                  return (
                    this.options.scrollOnUserInput &&
                      this.buffer.ybase !== this.buffer.ydisp &&
                      this.scrollToBottom(),
                    !1
                  )
                E || (H.key !== 'Dead' && H.key !== 'AltGraph') || (this._unprocessedDeadKey = !0)
                const G = (0, W.evaluateKeyboardEvent)(
                  H,
                  this.coreService.decPrivateModes.applicationCursorKeys,
                  this.browser.isMac,
                  this.options.macOptionIsMeta
                )
                if ((this.updateCursorStyle(H), G.type === 3 || G.type === 2)) {
                  const q = this.rows - 1
                  return (this.scrollLines(G.type === 2 ? -q : q), this.cancel(H, !0))
                }
                return (
                  G.type === 1 && this.selectAll(),
                  !!this._isThirdLevelShift(this.browser, H) ||
                    (G.cancel && this.cancel(H, !0),
                    !G.key ||
                      !!(
                        H.key &&
                        !H.ctrlKey &&
                        !H.altKey &&
                        !H.metaKey &&
                        H.key.length === 1 &&
                        H.key.charCodeAt(0) >= 65 &&
                        H.key.charCodeAt(0) <= 90
                      ) ||
                      (this._unprocessedDeadKey
                        ? ((this._unprocessedDeadKey = !1), !0)
                        : ((G.key !== F.C0.ETX && G.key !== F.C0.CR) || (this.textarea.value = ''),
                          this._onKey.fire({ key: G.key, domEvent: H }),
                          this._showCursor(),
                          this.coreService.triggerDataEvent(G.key, !0),
                          !this.optionsService.rawOptions.screenReaderMode || H.altKey || H.ctrlKey
                            ? this.cancel(H, !0)
                            : void (this._keyDownHandled = !0))))
                )
              }
              _isThirdLevelShift(H, E) {
                const G =
                  (H.isMac &&
                    !this.options.macOptionIsMeta &&
                    E.altKey &&
                    !E.ctrlKey &&
                    !E.metaKey) ||
                  (H.isWindows && E.altKey && E.ctrlKey && !E.metaKey) ||
                  (H.isWindows && E.getModifierState('AltGraph'))
                return E.type === 'keypress' ? G : G && (!E.keyCode || E.keyCode > 47)
              }
              _keyUp(H) {
                ;((this._keyDownSeen = !1),
                  (this._customKeyEventHandler && this._customKeyEventHandler(H) === !1) ||
                    ((function (E) {
                      return E.keyCode === 16 || E.keyCode === 17 || E.keyCode === 18
                    })(H) || this.focus(),
                    this.updateCursorStyle(H),
                    (this._keyPressHandled = !1)))
              }
              _keyPress(H) {
                let E
                if (
                  ((this._keyPressHandled = !1),
                  this._keyDownHandled ||
                    (this._customKeyEventHandler && this._customKeyEventHandler(H) === !1))
                )
                  return !1
                if ((this.cancel(H), H.charCode)) E = H.charCode
                else if (H.which === null || H.which === void 0) E = H.keyCode
                else {
                  if (H.which === 0 || H.charCode === 0) return !1
                  E = H.which
                }
                return !(
                  !E ||
                  ((H.altKey || H.ctrlKey || H.metaKey) &&
                    !this._isThirdLevelShift(this.browser, H)) ||
                  ((E = String.fromCharCode(E)),
                  this._onKey.fire({ key: E, domEvent: H }),
                  this._showCursor(),
                  this.coreService.triggerDataEvent(E, !0),
                  (this._keyPressHandled = !0),
                  (this._unprocessedDeadKey = !1),
                  0)
                )
              }
              _inputEvent(H) {
                if (
                  H.data &&
                  H.inputType === 'insertText' &&
                  (!H.composed || !this._keyDownSeen) &&
                  !this.optionsService.rawOptions.screenReaderMode
                ) {
                  if (this._keyPressHandled) return !1
                  this._unprocessedDeadKey = !1
                  const E = H.data
                  return (this.coreService.triggerDataEvent(E, !0), this.cancel(H), !0)
                }
                return !1
              }
              resize(H, E) {
                H !== this.cols || E !== this.rows
                  ? super.resize(H, E)
                  : this._charSizeService &&
                    !this._charSizeService.hasValidSize &&
                    this._charSizeService.measure()
              }
              _afterResize(H, E) {
                ;(this._charSizeService?.measure(), this.viewport?.syncScrollArea(!0))
              }
              clear() {
                if (this.buffer.ybase !== 0 || this.buffer.y !== 0) {
                  ;(this.buffer.clearAllMarkers(),
                    this.buffer.lines.set(
                      0,
                      this.buffer.lines.get(this.buffer.ybase + this.buffer.y)
                    ),
                    (this.buffer.lines.length = 1),
                    (this.buffer.ydisp = 0),
                    (this.buffer.ybase = 0),
                    (this.buffer.y = 0))
                  for (let H = 1; H < this.rows; H++)
                    this.buffer.lines.push(this.buffer.getBlankLine($.DEFAULT_ATTR_DATA))
                  ;(this._onScroll.fire({ position: this.buffer.ydisp, source: 0 }),
                    this.viewport?.reset(),
                    this.refresh(0, this.rows - 1))
                }
              }
              reset() {
                ;((this.options.rows = this.rows), (this.options.cols = this.cols))
                const H = this._customKeyEventHandler
                ;(this._setup(),
                  super.reset(),
                  this._selectionService?.reset(),
                  this._decorationService.reset(),
                  this.viewport?.reset(),
                  (this._customKeyEventHandler = H),
                  this.refresh(0, this.rows - 1))
              }
              clearTextureAtlas() {
                this._renderService?.clearTextureAtlas()
              }
              _reportFocus() {
                this.element?.classList.contains('focus')
                  ? this.coreService.triggerDataEvent(F.C0.ESC + '[I')
                  : this.coreService.triggerDataEvent(F.C0.ESC + '[O')
              }
              _reportWindowsOptions(H) {
                if (this._renderService)
                  switch (H) {
                    case B.WindowsOptionsReportType.GET_WIN_SIZE_PIXELS:
                      const E = this._renderService.dimensions.css.canvas.width.toFixed(0),
                        G = this._renderService.dimensions.css.canvas.height.toFixed(0)
                      this.coreService.triggerDataEvent(`${F.C0.ESC}[4;${G};${E}t`)
                      break
                    case B.WindowsOptionsReportType.GET_CELL_SIZE_PIXELS:
                      const q = this._renderService.dimensions.css.cell.width.toFixed(0),
                        Z = this._renderService.dimensions.css.cell.height.toFixed(0)
                      this.coreService.triggerDataEvent(`${F.C0.ESC}[6;${Z};${q}t`)
                  }
              }
              cancel(H, E) {
                if (this.options.cancelEvents || E)
                  return (H.preventDefault(), H.stopPropagation(), !1)
              }
            }
            i.Terminal = J
          },
          9924: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.TimeBasedDebouncer = void 0),
              (i.TimeBasedDebouncer = class {
                constructor(o, l = 1e3) {
                  ;((this._renderCallback = o),
                    (this._debounceThresholdMS = l),
                    (this._lastRefreshMs = 0),
                    (this._additionalRefreshRequested = !1))
                }
                dispose() {
                  this._refreshTimeoutID && clearTimeout(this._refreshTimeoutID)
                }
                refresh(o, l, u) {
                  ;((this._rowCount = u),
                    (o = o !== void 0 ? o : 0),
                    (l = l !== void 0 ? l : this._rowCount - 1),
                    (this._rowStart = this._rowStart !== void 0 ? Math.min(this._rowStart, o) : o),
                    (this._rowEnd = this._rowEnd !== void 0 ? Math.max(this._rowEnd, l) : l))
                  const a = Date.now()
                  if (a - this._lastRefreshMs >= this._debounceThresholdMS)
                    ((this._lastRefreshMs = a), this._innerRefresh())
                  else if (!this._additionalRefreshRequested) {
                    const h = a - this._lastRefreshMs,
                      f = this._debounceThresholdMS - h
                    ;((this._additionalRefreshRequested = !0),
                      (this._refreshTimeoutID = window.setTimeout(() => {
                        ;((this._lastRefreshMs = Date.now()),
                          this._innerRefresh(),
                          (this._additionalRefreshRequested = !1),
                          (this._refreshTimeoutID = void 0))
                      }, f)))
                  }
                }
                _innerRefresh() {
                  if (
                    this._rowStart === void 0 ||
                    this._rowEnd === void 0 ||
                    this._rowCount === void 0
                  )
                    return
                  const o = Math.max(this._rowStart, 0),
                    l = Math.min(this._rowEnd, this._rowCount - 1)
                  ;((this._rowStart = void 0), (this._rowEnd = void 0), this._renderCallback(o, l))
                }
              }))
          },
          1680: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (n, s, r, d) {
                  var v,
                    _ = arguments.length,
                    b = _ < 3 ? s : d === null ? (d = Object.getOwnPropertyDescriptor(s, r)) : d
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    b = Reflect.decorate(n, s, r, d)
                  else
                    for (var p = n.length - 1; p >= 0; p--)
                      (v = n[p]) && (b = (_ < 3 ? v(b) : _ > 3 ? v(s, r, b) : v(s, r)) || b)
                  return (_ > 3 && b && Object.defineProperty(s, r, b), b)
                },
              u =
                (this && this.__param) ||
                function (n, s) {
                  return function (r, d) {
                    s(r, d, n)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.Viewport = void 0))
            const a = o(3656),
              h = o(4725),
              f = o(8460),
              x = o(844),
              c = o(2585)
            let t = (i.Viewport = class extends x.Disposable {
              constructor(n, s, r, d, v, _, b, p) {
                ;(super(),
                  (this._viewportElement = n),
                  (this._scrollArea = s),
                  (this._bufferService = r),
                  (this._optionsService = d),
                  (this._charSizeService = v),
                  (this._renderService = _),
                  (this._coreBrowserService = b),
                  (this.scrollBarWidth = 0),
                  (this._currentRowHeight = 0),
                  (this._currentDeviceCellHeight = 0),
                  (this._lastRecordedBufferLength = 0),
                  (this._lastRecordedViewportHeight = 0),
                  (this._lastRecordedBufferHeight = 0),
                  (this._lastTouchY = 0),
                  (this._lastScrollTop = 0),
                  (this._wheelPartialScroll = 0),
                  (this._refreshAnimationFrame = null),
                  (this._ignoreNextScrollEvent = !1),
                  (this._smoothScrollState = { startTime: 0, origin: -1, target: -1 }),
                  (this._onRequestScrollLines = this.register(new f.EventEmitter())),
                  (this.onRequestScrollLines = this._onRequestScrollLines.event),
                  (this.scrollBarWidth =
                    this._viewportElement.offsetWidth - this._scrollArea.offsetWidth || 15),
                  this.register(
                    (0, a.addDisposableDomListener)(
                      this._viewportElement,
                      'scroll',
                      this._handleScroll.bind(this)
                    )
                  ),
                  (this._activeBuffer = this._bufferService.buffer),
                  this.register(
                    this._bufferService.buffers.onBufferActivate(
                      (S) => (this._activeBuffer = S.activeBuffer)
                    )
                  ),
                  (this._renderDimensions = this._renderService.dimensions),
                  this.register(
                    this._renderService.onDimensionsChange((S) => (this._renderDimensions = S))
                  ),
                  this._handleThemeChange(p.colors),
                  this.register(p.onChangeColors((S) => this._handleThemeChange(S))),
                  this.register(
                    this._optionsService.onSpecificOptionChange('scrollback', () =>
                      this.syncScrollArea()
                    )
                  ),
                  setTimeout(() => this.syncScrollArea()))
              }
              _handleThemeChange(n) {
                this._viewportElement.style.backgroundColor = n.background.css
              }
              reset() {
                ;((this._currentRowHeight = 0),
                  (this._currentDeviceCellHeight = 0),
                  (this._lastRecordedBufferLength = 0),
                  (this._lastRecordedViewportHeight = 0),
                  (this._lastRecordedBufferHeight = 0),
                  (this._lastTouchY = 0),
                  (this._lastScrollTop = 0),
                  this._coreBrowserService.window.requestAnimationFrame(() =>
                    this.syncScrollArea()
                  ))
              }
              _refresh(n) {
                if (n)
                  return (
                    this._innerRefresh(),
                    void (
                      this._refreshAnimationFrame !== null &&
                      this._coreBrowserService.window.cancelAnimationFrame(
                        this._refreshAnimationFrame
                      )
                    )
                  )
                this._refreshAnimationFrame === null &&
                  (this._refreshAnimationFrame =
                    this._coreBrowserService.window.requestAnimationFrame(() =>
                      this._innerRefresh()
                    ))
              }
              _innerRefresh() {
                if (this._charSizeService.height > 0) {
                  ;((this._currentRowHeight =
                    this._renderDimensions.device.cell.height / this._coreBrowserService.dpr),
                    (this._currentDeviceCellHeight = this._renderDimensions.device.cell.height),
                    (this._lastRecordedViewportHeight = this._viewportElement.offsetHeight))
                  const s =
                    Math.round(this._currentRowHeight * this._lastRecordedBufferLength) +
                    (this._lastRecordedViewportHeight - this._renderDimensions.css.canvas.height)
                  this._lastRecordedBufferHeight !== s &&
                    ((this._lastRecordedBufferHeight = s),
                    (this._scrollArea.style.height = this._lastRecordedBufferHeight + 'px'))
                }
                const n = this._bufferService.buffer.ydisp * this._currentRowHeight
                ;(this._viewportElement.scrollTop !== n &&
                  ((this._ignoreNextScrollEvent = !0), (this._viewportElement.scrollTop = n)),
                  (this._refreshAnimationFrame = null))
              }
              syncScrollArea(n = !1) {
                if (this._lastRecordedBufferLength !== this._bufferService.buffer.lines.length)
                  return (
                    (this._lastRecordedBufferLength = this._bufferService.buffer.lines.length),
                    void this._refresh(n)
                  )
                ;(this._lastRecordedViewportHeight ===
                  this._renderService.dimensions.css.canvas.height &&
                  this._lastScrollTop === this._activeBuffer.ydisp * this._currentRowHeight &&
                  this._renderDimensions.device.cell.height === this._currentDeviceCellHeight) ||
                  this._refresh(n)
              }
              _handleScroll(n) {
                if (
                  ((this._lastScrollTop = this._viewportElement.scrollTop),
                  !this._viewportElement.offsetParent)
                )
                  return
                if (this._ignoreNextScrollEvent)
                  return (
                    (this._ignoreNextScrollEvent = !1),
                    void this._onRequestScrollLines.fire({ amount: 0, suppressScrollEvent: !0 })
                  )
                const s =
                  Math.round(this._lastScrollTop / this._currentRowHeight) -
                  this._bufferService.buffer.ydisp
                this._onRequestScrollLines.fire({ amount: s, suppressScrollEvent: !0 })
              }
              _smoothScroll() {
                if (
                  this._isDisposed ||
                  this._smoothScrollState.origin === -1 ||
                  this._smoothScrollState.target === -1
                )
                  return
                const n = this._smoothScrollPercent()
                ;((this._viewportElement.scrollTop =
                  this._smoothScrollState.origin +
                  Math.round(
                    n * (this._smoothScrollState.target - this._smoothScrollState.origin)
                  )),
                  n < 1
                    ? this._coreBrowserService.window.requestAnimationFrame(() =>
                        this._smoothScroll()
                      )
                    : this._clearSmoothScrollState())
              }
              _smoothScrollPercent() {
                return this._optionsService.rawOptions.smoothScrollDuration &&
                  this._smoothScrollState.startTime
                  ? Math.max(
                      Math.min(
                        (Date.now() - this._smoothScrollState.startTime) /
                          this._optionsService.rawOptions.smoothScrollDuration,
                        1
                      ),
                      0
                    )
                  : 1
              }
              _clearSmoothScrollState() {
                ;((this._smoothScrollState.startTime = 0),
                  (this._smoothScrollState.origin = -1),
                  (this._smoothScrollState.target = -1))
              }
              _bubbleScroll(n, s) {
                const r = this._viewportElement.scrollTop + this._lastRecordedViewportHeight
                return (
                  !(
                    (s < 0 && this._viewportElement.scrollTop !== 0) ||
                    (s > 0 && r < this._lastRecordedBufferHeight)
                  ) || (n.cancelable && n.preventDefault(), !1)
                )
              }
              handleWheel(n) {
                const s = this._getPixelsScrolled(n)
                return (
                  s !== 0 &&
                  (this._optionsService.rawOptions.smoothScrollDuration
                    ? ((this._smoothScrollState.startTime = Date.now()),
                      this._smoothScrollPercent() < 1
                        ? ((this._smoothScrollState.origin = this._viewportElement.scrollTop),
                          this._smoothScrollState.target === -1
                            ? (this._smoothScrollState.target = this._viewportElement.scrollTop + s)
                            : (this._smoothScrollState.target += s),
                          (this._smoothScrollState.target = Math.max(
                            Math.min(
                              this._smoothScrollState.target,
                              this._viewportElement.scrollHeight
                            ),
                            0
                          )),
                          this._smoothScroll())
                        : this._clearSmoothScrollState())
                    : (this._viewportElement.scrollTop += s),
                  this._bubbleScroll(n, s))
                )
              }
              scrollLines(n) {
                if (n !== 0)
                  if (this._optionsService.rawOptions.smoothScrollDuration) {
                    const s = n * this._currentRowHeight
                    ;((this._smoothScrollState.startTime = Date.now()),
                      this._smoothScrollPercent() < 1
                        ? ((this._smoothScrollState.origin = this._viewportElement.scrollTop),
                          (this._smoothScrollState.target = this._smoothScrollState.origin + s),
                          (this._smoothScrollState.target = Math.max(
                            Math.min(
                              this._smoothScrollState.target,
                              this._viewportElement.scrollHeight
                            ),
                            0
                          )),
                          this._smoothScroll())
                        : this._clearSmoothScrollState())
                  } else this._onRequestScrollLines.fire({ amount: n, suppressScrollEvent: !1 })
              }
              _getPixelsScrolled(n) {
                if (n.deltaY === 0 || n.shiftKey) return 0
                let s = this._applyScrollModifier(n.deltaY, n)
                return (
                  n.deltaMode === WheelEvent.DOM_DELTA_LINE
                    ? (s *= this._currentRowHeight)
                    : n.deltaMode === WheelEvent.DOM_DELTA_PAGE &&
                      (s *= this._currentRowHeight * this._bufferService.rows),
                  s
                )
              }
              getBufferElements(n, s) {
                let r,
                  d = ''
                const v = [],
                  _ = s ?? this._bufferService.buffer.lines.length,
                  b = this._bufferService.buffer.lines
                for (let p = n; p < _; p++) {
                  const S = b.get(p)
                  if (!S) continue
                  const L = b.get(p + 1)?.isWrapped
                  if (((d += S.translateToString(!L)), !L || p === b.length - 1)) {
                    const M = document.createElement('div')
                    ;((M.textContent = d), v.push(M), d.length > 0 && (r = M), (d = ''))
                  }
                }
                return { bufferElements: v, cursorElement: r }
              }
              getLinesScrolled(n) {
                if (n.deltaY === 0 || n.shiftKey) return 0
                let s = this._applyScrollModifier(n.deltaY, n)
                return (
                  n.deltaMode === WheelEvent.DOM_DELTA_PIXEL
                    ? ((s /= this._currentRowHeight + 0),
                      (this._wheelPartialScroll += s),
                      (s =
                        Math.floor(Math.abs(this._wheelPartialScroll)) *
                        (this._wheelPartialScroll > 0 ? 1 : -1)),
                      (this._wheelPartialScroll %= 1))
                    : n.deltaMode === WheelEvent.DOM_DELTA_PAGE && (s *= this._bufferService.rows),
                  s
                )
              }
              _applyScrollModifier(n, s) {
                const r = this._optionsService.rawOptions.fastScrollModifier
                return (r === 'alt' && s.altKey) ||
                  (r === 'ctrl' && s.ctrlKey) ||
                  (r === 'shift' && s.shiftKey)
                  ? n *
                      this._optionsService.rawOptions.fastScrollSensitivity *
                      this._optionsService.rawOptions.scrollSensitivity
                  : n * this._optionsService.rawOptions.scrollSensitivity
              }
              handleTouchStart(n) {
                this._lastTouchY = n.touches[0].pageY
              }
              handleTouchMove(n) {
                const s = this._lastTouchY - n.touches[0].pageY
                return (
                  (this._lastTouchY = n.touches[0].pageY),
                  s !== 0 && ((this._viewportElement.scrollTop += s), this._bubbleScroll(n, s))
                )
              }
            })
            i.Viewport = t = l(
              [
                u(2, c.IBufferService),
                u(3, c.IOptionsService),
                u(4, h.ICharSizeService),
                u(5, h.IRenderService),
                u(6, h.ICoreBrowserService),
                u(7, h.IThemeService)
              ],
              t
            )
          },
          3107: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (c, t, n, s) {
                  var r,
                    d = arguments.length,
                    v = d < 3 ? t : s === null ? (s = Object.getOwnPropertyDescriptor(t, n)) : s
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    v = Reflect.decorate(c, t, n, s)
                  else
                    for (var _ = c.length - 1; _ >= 0; _--)
                      (r = c[_]) && (v = (d < 3 ? r(v) : d > 3 ? r(t, n, v) : r(t, n)) || v)
                  return (d > 3 && v && Object.defineProperty(t, n, v), v)
                },
              u =
                (this && this.__param) ||
                function (c, t) {
                  return function (n, s) {
                    t(n, s, c)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.BufferDecorationRenderer = void 0))
            const a = o(4725),
              h = o(844),
              f = o(2585)
            let x = (i.BufferDecorationRenderer = class extends h.Disposable {
              constructor(c, t, n, s, r) {
                ;(super(),
                  (this._screenElement = c),
                  (this._bufferService = t),
                  (this._coreBrowserService = n),
                  (this._decorationService = s),
                  (this._renderService = r),
                  (this._decorationElements = new Map()),
                  (this._altBufferIsActive = !1),
                  (this._dimensionsChanged = !1),
                  (this._container = document.createElement('div')),
                  this._container.classList.add('xterm-decoration-container'),
                  this._screenElement.appendChild(this._container),
                  this.register(
                    this._renderService.onRenderedViewportChange(() => this._doRefreshDecorations())
                  ),
                  this.register(
                    this._renderService.onDimensionsChange(() => {
                      ;((this._dimensionsChanged = !0), this._queueRefresh())
                    })
                  ),
                  this.register(this._coreBrowserService.onDprChange(() => this._queueRefresh())),
                  this.register(
                    this._bufferService.buffers.onBufferActivate(() => {
                      this._altBufferIsActive =
                        this._bufferService.buffer === this._bufferService.buffers.alt
                    })
                  ),
                  this.register(
                    this._decorationService.onDecorationRegistered(() => this._queueRefresh())
                  ),
                  this.register(
                    this._decorationService.onDecorationRemoved((d) => this._removeDecoration(d))
                  ),
                  this.register(
                    (0, h.toDisposable)(() => {
                      ;(this._container.remove(), this._decorationElements.clear())
                    })
                  ))
              }
              _queueRefresh() {
                this._animationFrame === void 0 &&
                  (this._animationFrame = this._renderService.addRefreshCallback(() => {
                    ;(this._doRefreshDecorations(), (this._animationFrame = void 0))
                  }))
              }
              _doRefreshDecorations() {
                for (const c of this._decorationService.decorations) this._renderDecoration(c)
                this._dimensionsChanged = !1
              }
              _renderDecoration(c) {
                ;(this._refreshStyle(c), this._dimensionsChanged && this._refreshXPosition(c))
              }
              _createElement(c) {
                const t = this._coreBrowserService.mainDocument.createElement('div')
                ;(t.classList.add('xterm-decoration'),
                  t.classList.toggle('xterm-decoration-top-layer', c?.options?.layer === 'top'),
                  (t.style.width = `${Math.round((c.options.width || 1) * this._renderService.dimensions.css.cell.width)}px`),
                  (t.style.height =
                    (c.options.height || 1) * this._renderService.dimensions.css.cell.height +
                    'px'),
                  (t.style.top =
                    (c.marker.line - this._bufferService.buffers.active.ydisp) *
                      this._renderService.dimensions.css.cell.height +
                    'px'),
                  (t.style.lineHeight = `${this._renderService.dimensions.css.cell.height}px`))
                const n = c.options.x ?? 0
                return (
                  n && n > this._bufferService.cols && (t.style.display = 'none'),
                  this._refreshXPosition(c, t),
                  t
                )
              }
              _refreshStyle(c) {
                const t = c.marker.line - this._bufferService.buffers.active.ydisp
                if (t < 0 || t >= this._bufferService.rows)
                  c.element &&
                    ((c.element.style.display = 'none'), c.onRenderEmitter.fire(c.element))
                else {
                  let n = this._decorationElements.get(c)
                  ;(n ||
                    ((n = this._createElement(c)),
                    (c.element = n),
                    this._decorationElements.set(c, n),
                    this._container.appendChild(n),
                    c.onDispose(() => {
                      ;(this._decorationElements.delete(c), n.remove())
                    })),
                    (n.style.top = t * this._renderService.dimensions.css.cell.height + 'px'),
                    (n.style.display = this._altBufferIsActive ? 'none' : 'block'),
                    c.onRenderEmitter.fire(n))
                }
              }
              _refreshXPosition(c, t = c.element) {
                if (!t) return
                const n = c.options.x ?? 0
                ;(c.options.anchor || 'left') === 'right'
                  ? (t.style.right = n
                      ? n * this._renderService.dimensions.css.cell.width + 'px'
                      : '')
                  : (t.style.left = n
                      ? n * this._renderService.dimensions.css.cell.width + 'px'
                      : '')
              }
              _removeDecoration(c) {
                ;(this._decorationElements.get(c)?.remove(),
                  this._decorationElements.delete(c),
                  c.dispose())
              }
            })
            i.BufferDecorationRenderer = x = l(
              [
                u(1, f.IBufferService),
                u(2, a.ICoreBrowserService),
                u(3, f.IDecorationService),
                u(4, a.IRenderService)
              ],
              x
            )
          },
          5871: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.ColorZoneStore = void 0),
              (i.ColorZoneStore = class {
                constructor() {
                  ;((this._zones = []),
                    (this._zonePool = []),
                    (this._zonePoolIndex = 0),
                    (this._linePadding = { full: 0, left: 0, center: 0, right: 0 }))
                }
                get zones() {
                  return (
                    (this._zonePool.length = Math.min(this._zonePool.length, this._zones.length)),
                    this._zones
                  )
                }
                clear() {
                  ;((this._zones.length = 0), (this._zonePoolIndex = 0))
                }
                addDecoration(o) {
                  if (o.options.overviewRulerOptions) {
                    for (const l of this._zones)
                      if (
                        l.color === o.options.overviewRulerOptions.color &&
                        l.position === o.options.overviewRulerOptions.position
                      ) {
                        if (this._lineIntersectsZone(l, o.marker.line)) return
                        if (
                          this._lineAdjacentToZone(
                            l,
                            o.marker.line,
                            o.options.overviewRulerOptions.position
                          )
                        )
                          return void this._addLineToZone(l, o.marker.line)
                      }
                    if (this._zonePoolIndex < this._zonePool.length)
                      return (
                        (this._zonePool[this._zonePoolIndex].color =
                          o.options.overviewRulerOptions.color),
                        (this._zonePool[this._zonePoolIndex].position =
                          o.options.overviewRulerOptions.position),
                        (this._zonePool[this._zonePoolIndex].startBufferLine = o.marker.line),
                        (this._zonePool[this._zonePoolIndex].endBufferLine = o.marker.line),
                        void this._zones.push(this._zonePool[this._zonePoolIndex++])
                      )
                    ;(this._zones.push({
                      color: o.options.overviewRulerOptions.color,
                      position: o.options.overviewRulerOptions.position,
                      startBufferLine: o.marker.line,
                      endBufferLine: o.marker.line
                    }),
                      this._zonePool.push(this._zones[this._zones.length - 1]),
                      this._zonePoolIndex++)
                  }
                }
                setPadding(o) {
                  this._linePadding = o
                }
                _lineIntersectsZone(o, l) {
                  return l >= o.startBufferLine && l <= o.endBufferLine
                }
                _lineAdjacentToZone(o, l, u) {
                  return (
                    l >= o.startBufferLine - this._linePadding[u || 'full'] &&
                    l <= o.endBufferLine + this._linePadding[u || 'full']
                  )
                }
                _addLineToZone(o, l) {
                  ;((o.startBufferLine = Math.min(o.startBufferLine, l)),
                    (o.endBufferLine = Math.max(o.endBufferLine, l)))
                }
              }))
          },
          5744: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (r, d, v, _) {
                  var b,
                    p = arguments.length,
                    S = p < 3 ? d : _ === null ? (_ = Object.getOwnPropertyDescriptor(d, v)) : _
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    S = Reflect.decorate(r, d, v, _)
                  else
                    for (var L = r.length - 1; L >= 0; L--)
                      (b = r[L]) && (S = (p < 3 ? b(S) : p > 3 ? b(d, v, S) : b(d, v)) || S)
                  return (p > 3 && S && Object.defineProperty(d, v, S), S)
                },
              u =
                (this && this.__param) ||
                function (r, d) {
                  return function (v, _) {
                    d(v, _, r)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.OverviewRulerRenderer = void 0))
            const a = o(5871),
              h = o(4725),
              f = o(844),
              x = o(2585),
              c = { full: 0, left: 0, center: 0, right: 0 },
              t = { full: 0, left: 0, center: 0, right: 0 },
              n = { full: 0, left: 0, center: 0, right: 0 }
            let s = (i.OverviewRulerRenderer = class extends f.Disposable {
              get _width() {
                return this._optionsService.options.overviewRulerWidth || 0
              }
              constructor(r, d, v, _, b, p, S) {
                ;(super(),
                  (this._viewportElement = r),
                  (this._screenElement = d),
                  (this._bufferService = v),
                  (this._decorationService = _),
                  (this._renderService = b),
                  (this._optionsService = p),
                  (this._coreBrowserService = S),
                  (this._colorZoneStore = new a.ColorZoneStore()),
                  (this._shouldUpdateDimensions = !0),
                  (this._shouldUpdateAnchor = !0),
                  (this._lastKnownBufferLength = 0),
                  (this._canvas = this._coreBrowserService.mainDocument.createElement('canvas')),
                  this._canvas.classList.add('xterm-decoration-overview-ruler'),
                  this._refreshCanvasDimensions(),
                  this._viewportElement.parentElement?.insertBefore(
                    this._canvas,
                    this._viewportElement
                  ))
                const L = this._canvas.getContext('2d')
                if (!L) throw new Error('Ctx cannot be null')
                ;((this._ctx = L),
                  this._registerDecorationListeners(),
                  this._registerBufferChangeListeners(),
                  this._registerDimensionChangeListeners(),
                  this.register(
                    (0, f.toDisposable)(() => {
                      this._canvas?.remove()
                    })
                  ))
              }
              _registerDecorationListeners() {
                ;(this.register(
                  this._decorationService.onDecorationRegistered(() =>
                    this._queueRefresh(void 0, !0)
                  )
                ),
                  this.register(
                    this._decorationService.onDecorationRemoved(() =>
                      this._queueRefresh(void 0, !0)
                    )
                  ))
              }
              _registerBufferChangeListeners() {
                ;(this.register(
                  this._renderService.onRenderedViewportChange(() => this._queueRefresh())
                ),
                  this.register(
                    this._bufferService.buffers.onBufferActivate(() => {
                      this._canvas.style.display =
                        this._bufferService.buffer === this._bufferService.buffers.alt
                          ? 'none'
                          : 'block'
                    })
                  ),
                  this.register(
                    this._bufferService.onScroll(() => {
                      this._lastKnownBufferLength !==
                        this._bufferService.buffers.normal.lines.length &&
                        (this._refreshDrawHeightConstants(), this._refreshColorZonePadding())
                    })
                  ))
              }
              _registerDimensionChangeListeners() {
                ;(this.register(
                  this._renderService.onRender(() => {
                    ;(this._containerHeight &&
                      this._containerHeight === this._screenElement.clientHeight) ||
                      (this._queueRefresh(!0),
                      (this._containerHeight = this._screenElement.clientHeight))
                  })
                ),
                  this.register(
                    this._optionsService.onSpecificOptionChange('overviewRulerWidth', () =>
                      this._queueRefresh(!0)
                    )
                  ),
                  this.register(this._coreBrowserService.onDprChange(() => this._queueRefresh(!0))),
                  this._queueRefresh(!0))
              }
              _refreshDrawConstants() {
                const r = Math.floor(this._canvas.width / 3),
                  d = Math.ceil(this._canvas.width / 3)
                ;((t.full = this._canvas.width),
                  (t.left = r),
                  (t.center = d),
                  (t.right = r),
                  this._refreshDrawHeightConstants(),
                  (n.full = 0),
                  (n.left = 0),
                  (n.center = t.left),
                  (n.right = t.left + t.center))
              }
              _refreshDrawHeightConstants() {
                c.full = Math.round(2 * this._coreBrowserService.dpr)
                const r = this._canvas.height / this._bufferService.buffer.lines.length,
                  d = Math.round(Math.max(Math.min(r, 12), 6) * this._coreBrowserService.dpr)
                ;((c.left = d), (c.center = d), (c.right = d))
              }
              _refreshColorZonePadding() {
                ;(this._colorZoneStore.setPadding({
                  full: Math.floor(
                    (this._bufferService.buffers.active.lines.length / (this._canvas.height - 1)) *
                      c.full
                  ),
                  left: Math.floor(
                    (this._bufferService.buffers.active.lines.length / (this._canvas.height - 1)) *
                      c.left
                  ),
                  center: Math.floor(
                    (this._bufferService.buffers.active.lines.length / (this._canvas.height - 1)) *
                      c.center
                  ),
                  right: Math.floor(
                    (this._bufferService.buffers.active.lines.length / (this._canvas.height - 1)) *
                      c.right
                  )
                }),
                  (this._lastKnownBufferLength = this._bufferService.buffers.normal.lines.length))
              }
              _refreshCanvasDimensions() {
                ;((this._canvas.style.width = `${this._width}px`),
                  (this._canvas.width = Math.round(this._width * this._coreBrowserService.dpr)),
                  (this._canvas.style.height = `${this._screenElement.clientHeight}px`),
                  (this._canvas.height = Math.round(
                    this._screenElement.clientHeight * this._coreBrowserService.dpr
                  )),
                  this._refreshDrawConstants(),
                  this._refreshColorZonePadding())
              }
              _refreshDecorations() {
                ;(this._shouldUpdateDimensions && this._refreshCanvasDimensions(),
                  this._ctx.clearRect(0, 0, this._canvas.width, this._canvas.height),
                  this._colorZoneStore.clear())
                for (const d of this._decorationService.decorations)
                  this._colorZoneStore.addDecoration(d)
                this._ctx.lineWidth = 1
                const r = this._colorZoneStore.zones
                for (const d of r) d.position !== 'full' && this._renderColorZone(d)
                for (const d of r) d.position === 'full' && this._renderColorZone(d)
                ;((this._shouldUpdateDimensions = !1), (this._shouldUpdateAnchor = !1))
              }
              _renderColorZone(r) {
                ;((this._ctx.fillStyle = r.color),
                  this._ctx.fillRect(
                    n[r.position || 'full'],
                    Math.round(
                      (this._canvas.height - 1) *
                        (r.startBufferLine / this._bufferService.buffers.active.lines.length) -
                        c[r.position || 'full'] / 2
                    ),
                    t[r.position || 'full'],
                    Math.round(
                      (this._canvas.height - 1) *
                        ((r.endBufferLine - r.startBufferLine) /
                          this._bufferService.buffers.active.lines.length) +
                        c[r.position || 'full']
                    )
                  ))
              }
              _queueRefresh(r, d) {
                ;((this._shouldUpdateDimensions = r || this._shouldUpdateDimensions),
                  (this._shouldUpdateAnchor = d || this._shouldUpdateAnchor),
                  this._animationFrame === void 0 &&
                    (this._animationFrame = this._coreBrowserService.window.requestAnimationFrame(
                      () => {
                        ;(this._refreshDecorations(), (this._animationFrame = void 0))
                      }
                    )))
              }
            })
            i.OverviewRulerRenderer = s = l(
              [
                u(2, x.IBufferService),
                u(3, x.IDecorationService),
                u(4, h.IRenderService),
                u(5, x.IOptionsService),
                u(6, h.ICoreBrowserService)
              ],
              s
            )
          },
          2950: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (c, t, n, s) {
                  var r,
                    d = arguments.length,
                    v = d < 3 ? t : s === null ? (s = Object.getOwnPropertyDescriptor(t, n)) : s
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    v = Reflect.decorate(c, t, n, s)
                  else
                    for (var _ = c.length - 1; _ >= 0; _--)
                      (r = c[_]) && (v = (d < 3 ? r(v) : d > 3 ? r(t, n, v) : r(t, n)) || v)
                  return (d > 3 && v && Object.defineProperty(t, n, v), v)
                },
              u =
                (this && this.__param) ||
                function (c, t) {
                  return function (n, s) {
                    t(n, s, c)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.CompositionHelper = void 0))
            const a = o(4725),
              h = o(2585),
              f = o(2584)
            let x = (i.CompositionHelper = class {
              get isComposing() {
                return this._isComposing
              }
              constructor(c, t, n, s, r, d) {
                ;((this._textarea = c),
                  (this._compositionView = t),
                  (this._bufferService = n),
                  (this._optionsService = s),
                  (this._coreService = r),
                  (this._renderService = d),
                  (this._isComposing = !1),
                  (this._isSendingComposition = !1),
                  (this._compositionPosition = { start: 0, end: 0 }),
                  (this._dataAlreadySent = ''))
              }
              compositionstart() {
                ;((this._isComposing = !0),
                  (this._compositionPosition.start = this._textarea.value.length),
                  (this._compositionView.textContent = ''),
                  (this._dataAlreadySent = ''),
                  this._compositionView.classList.add('active'))
              }
              compositionupdate(c) {
                ;((this._compositionView.textContent = c.data),
                  this.updateCompositionElements(),
                  setTimeout(() => {
                    this._compositionPosition.end = this._textarea.value.length
                  }, 0))
              }
              compositionend() {
                this._finalizeComposition(!0)
              }
              keydown(c) {
                if (this._isComposing || this._isSendingComposition) {
                  if (c.keyCode === 229 || c.keyCode === 16 || c.keyCode === 17 || c.keyCode === 18)
                    return !1
                  this._finalizeComposition(!1)
                }
                return c.keyCode !== 229 || (this._handleAnyTextareaChanges(), !1)
              }
              _finalizeComposition(c) {
                if (
                  (this._compositionView.classList.remove('active'), (this._isComposing = !1), c)
                ) {
                  const t = {
                    start: this._compositionPosition.start,
                    end: this._compositionPosition.end
                  }
                  ;((this._isSendingComposition = !0),
                    setTimeout(() => {
                      if (this._isSendingComposition) {
                        let n
                        ;((this._isSendingComposition = !1),
                          (t.start += this._dataAlreadySent.length),
                          (n = this._isComposing
                            ? this._textarea.value.substring(t.start, t.end)
                            : this._textarea.value.substring(t.start)),
                          n.length > 0 && this._coreService.triggerDataEvent(n, !0))
                      }
                    }, 0))
                } else {
                  this._isSendingComposition = !1
                  const t = this._textarea.value.substring(
                    this._compositionPosition.start,
                    this._compositionPosition.end
                  )
                  this._coreService.triggerDataEvent(t, !0)
                }
              }
              _handleAnyTextareaChanges() {
                const c = this._textarea.value
                setTimeout(() => {
                  if (!this._isComposing) {
                    const t = this._textarea.value,
                      n = t.replace(c, '')
                    ;((this._dataAlreadySent = n),
                      t.length > c.length
                        ? this._coreService.triggerDataEvent(n, !0)
                        : t.length < c.length
                          ? this._coreService.triggerDataEvent(`${f.C0.DEL}`, !0)
                          : t.length === c.length &&
                            t !== c &&
                            this._coreService.triggerDataEvent(t, !0))
                  }
                }, 0)
              }
              updateCompositionElements(c) {
                if (this._isComposing) {
                  if (this._bufferService.buffer.isCursorInViewport) {
                    const t = Math.min(this._bufferService.buffer.x, this._bufferService.cols - 1),
                      n = this._renderService.dimensions.css.cell.height,
                      s =
                        this._bufferService.buffer.y *
                        this._renderService.dimensions.css.cell.height,
                      r = t * this._renderService.dimensions.css.cell.width
                    ;((this._compositionView.style.left = r + 'px'),
                      (this._compositionView.style.top = s + 'px'),
                      (this._compositionView.style.height = n + 'px'),
                      (this._compositionView.style.lineHeight = n + 'px'),
                      (this._compositionView.style.fontFamily =
                        this._optionsService.rawOptions.fontFamily),
                      (this._compositionView.style.fontSize =
                        this._optionsService.rawOptions.fontSize + 'px'))
                    const d = this._compositionView.getBoundingClientRect()
                    ;((this._textarea.style.left = r + 'px'),
                      (this._textarea.style.top = s + 'px'),
                      (this._textarea.style.width = Math.max(d.width, 1) + 'px'),
                      (this._textarea.style.height = Math.max(d.height, 1) + 'px'),
                      (this._textarea.style.lineHeight = d.height + 'px'))
                  }
                  c || setTimeout(() => this.updateCompositionElements(!0), 0)
                }
              }
            })
            i.CompositionHelper = x = l(
              [
                u(2, h.IBufferService),
                u(3, h.IOptionsService),
                u(4, h.ICoreService),
                u(5, a.IRenderService)
              ],
              x
            )
          },
          9806: (I, i) => {
            function o(l, u, a) {
              const h = a.getBoundingClientRect(),
                f = l.getComputedStyle(a),
                x = parseInt(f.getPropertyValue('padding-left')),
                c = parseInt(f.getPropertyValue('padding-top'))
              return [u.clientX - h.left - x, u.clientY - h.top - c]
            }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.getCoords = i.getCoordsRelativeToElement = void 0),
              (i.getCoordsRelativeToElement = o),
              (i.getCoords = function (l, u, a, h, f, x, c, t, n) {
                if (!x) return
                const s = o(l, u, a)
                return s
                  ? ((s[0] = Math.ceil((s[0] + (n ? c / 2 : 0)) / c)),
                    (s[1] = Math.ceil(s[1] / t)),
                    (s[0] = Math.min(Math.max(s[0], 1), h + (n ? 1 : 0))),
                    (s[1] = Math.min(Math.max(s[1], 1), f)),
                    s)
                  : void 0
              }))
          },
          9504: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.moveToCellSequence = void 0))
            const l = o(2584)
            function u(t, n, s, r) {
              const d = t - a(t, s),
                v = n - a(n, s),
                _ =
                  Math.abs(d - v) -
                  (function (b, p, S) {
                    let L = 0
                    const M = b - a(b, S),
                      P = p - a(p, S)
                    for (let j = 0; j < Math.abs(M - P); j++) {
                      const D = h(b, p) === 'A' ? -1 : 1
                      S.buffer.lines.get(M + D * j)?.isWrapped && L++
                    }
                    return L
                  })(t, n, s)
              return c(_, x(h(t, n), r))
            }
            function a(t, n) {
              let s = 0,
                r = n.buffer.lines.get(t),
                d = r?.isWrapped
              for (; d && t >= 0 && t < n.rows; )
                (s++, (r = n.buffer.lines.get(--t)), (d = r?.isWrapped))
              return s
            }
            function h(t, n) {
              return t > n ? 'A' : 'B'
            }
            function f(t, n, s, r, d, v) {
              let _ = t,
                b = n,
                p = ''
              for (; _ !== s || b !== r; )
                ((_ += d ? 1 : -1),
                  d && _ > v.cols - 1
                    ? ((p += v.buffer.translateBufferLineToString(b, !1, t, _)),
                      (_ = 0),
                      (t = 0),
                      b++)
                    : !d &&
                      _ < 0 &&
                      ((p += v.buffer.translateBufferLineToString(b, !1, 0, t + 1)),
                      (_ = v.cols - 1),
                      (t = _),
                      b--))
              return p + v.buffer.translateBufferLineToString(b, !1, t, _)
            }
            function x(t, n) {
              const s = n ? 'O' : '['
              return l.C0.ESC + s + t
            }
            function c(t, n) {
              t = Math.floor(t)
              let s = ''
              for (let r = 0; r < t; r++) s += n
              return s
            }
            i.moveToCellSequence = function (t, n, s, r) {
              const d = s.buffer.x,
                v = s.buffer.y
              if (!s.buffer.hasScrollback)
                return (
                  (function (p, S, L, M, P, j) {
                    return u(S, M, P, j).length === 0
                      ? ''
                      : c(f(p, S, p, S - a(S, P), !1, P).length, x('D', j))
                  })(d, v, 0, n, s, r) +
                  u(v, n, s, r) +
                  (function (p, S, L, M, P, j) {
                    let D
                    D = u(S, M, P, j).length > 0 ? M - a(M, P) : S
                    const O = M,
                      $ = (function (F, W, C, A, N, B) {
                        let z
                        return (
                          (z = u(C, A, N, B).length > 0 ? A - a(A, N) : W),
                          (F < C && z <= A) || (F >= C && z < A) ? 'C' : 'D'
                        )
                      })(p, S, L, M, P, j)
                    return c(f(p, D, L, O, $ === 'C', P).length, x($, j))
                  })(d, v, t, n, s, r)
                )
              let _
              if (v === n) return ((_ = d > t ? 'D' : 'C'), c(Math.abs(d - t), x(_, r)))
              _ = v > n ? 'D' : 'C'
              const b = Math.abs(v - n)
              return c(
                (function (p, S) {
                  return S.cols - p
                })(v > n ? t : d, s) +
                  (b - 1) * s.cols +
                  1 +
                  ((v > n ? d : t) - 1),
                x(_, r)
              )
            }
          },
          1296: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (j, D, O, $) {
                  var F,
                    W = arguments.length,
                    C = W < 3 ? D : $ === null ? ($ = Object.getOwnPropertyDescriptor(D, O)) : $
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    C = Reflect.decorate(j, D, O, $)
                  else
                    for (var A = j.length - 1; A >= 0; A--)
                      (F = j[A]) && (C = (W < 3 ? F(C) : W > 3 ? F(D, O, C) : F(D, O)) || C)
                  return (W > 3 && C && Object.defineProperty(D, O, C), C)
                },
              u =
                (this && this.__param) ||
                function (j, D) {
                  return function (O, $) {
                    D(O, $, j)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.DomRenderer = void 0))
            const a = o(3787),
              h = o(2550),
              f = o(2223),
              x = o(6171),
              c = o(6052),
              t = o(4725),
              n = o(8055),
              s = o(8460),
              r = o(844),
              d = o(2585),
              v = 'xterm-dom-renderer-owner-',
              _ = 'xterm-rows',
              b = 'xterm-fg-',
              p = 'xterm-bg-',
              S = 'xterm-focus',
              L = 'xterm-selection'
            let M = 1,
              P = (i.DomRenderer = class extends r.Disposable {
                constructor(j, D, O, $, F, W, C, A, N, B, z, K, J) {
                  ;(super(),
                    (this._terminal = j),
                    (this._document = D),
                    (this._element = O),
                    (this._screenElement = $),
                    (this._viewportElement = F),
                    (this._helperContainer = W),
                    (this._linkifier2 = C),
                    (this._charSizeService = N),
                    (this._optionsService = B),
                    (this._bufferService = z),
                    (this._coreBrowserService = K),
                    (this._themeService = J),
                    (this._terminalClass = M++),
                    (this._rowElements = []),
                    (this._selectionRenderModel = (0, c.createSelectionRenderModel)()),
                    (this.onRequestRedraw = this.register(new s.EventEmitter()).event),
                    (this._rowContainer = this._document.createElement('div')),
                    this._rowContainer.classList.add(_),
                    (this._rowContainer.style.lineHeight = 'normal'),
                    this._rowContainer.setAttribute('aria-hidden', 'true'),
                    this._refreshRowElements(this._bufferService.cols, this._bufferService.rows),
                    (this._selectionContainer = this._document.createElement('div')),
                    this._selectionContainer.classList.add(L),
                    this._selectionContainer.setAttribute('aria-hidden', 'true'),
                    (this.dimensions = (0, x.createRenderDimensions)()),
                    this._updateDimensions(),
                    this.register(
                      this._optionsService.onOptionChange(() => this._handleOptionsChanged())
                    ),
                    this.register(this._themeService.onChangeColors((Q) => this._injectCss(Q))),
                    this._injectCss(this._themeService.colors),
                    (this._rowFactory = A.createInstance(a.DomRendererRowFactory, document)),
                    this._element.classList.add(v + this._terminalClass),
                    this._screenElement.appendChild(this._rowContainer),
                    this._screenElement.appendChild(this._selectionContainer),
                    this.register(
                      this._linkifier2.onShowLinkUnderline((Q) => this._handleLinkHover(Q))
                    ),
                    this.register(
                      this._linkifier2.onHideLinkUnderline((Q) => this._handleLinkLeave(Q))
                    ),
                    this.register(
                      (0, r.toDisposable)(() => {
                        ;(this._element.classList.remove(v + this._terminalClass),
                          this._rowContainer.remove(),
                          this._selectionContainer.remove(),
                          this._widthCache.dispose(),
                          this._themeStyleElement.remove(),
                          this._dimensionsStyleElement.remove())
                      })
                    ),
                    (this._widthCache = new h.WidthCache(this._document, this._helperContainer)),
                    this._widthCache.setFont(
                      this._optionsService.rawOptions.fontFamily,
                      this._optionsService.rawOptions.fontSize,
                      this._optionsService.rawOptions.fontWeight,
                      this._optionsService.rawOptions.fontWeightBold
                    ),
                    this._setDefaultSpacing())
                }
                _updateDimensions() {
                  const j = this._coreBrowserService.dpr
                  ;((this.dimensions.device.char.width = this._charSizeService.width * j),
                    (this.dimensions.device.char.height = Math.ceil(
                      this._charSizeService.height * j
                    )),
                    (this.dimensions.device.cell.width =
                      this.dimensions.device.char.width +
                      Math.round(this._optionsService.rawOptions.letterSpacing)),
                    (this.dimensions.device.cell.height = Math.floor(
                      this.dimensions.device.char.height *
                        this._optionsService.rawOptions.lineHeight
                    )),
                    (this.dimensions.device.char.left = 0),
                    (this.dimensions.device.char.top = 0),
                    (this.dimensions.device.canvas.width =
                      this.dimensions.device.cell.width * this._bufferService.cols),
                    (this.dimensions.device.canvas.height =
                      this.dimensions.device.cell.height * this._bufferService.rows),
                    (this.dimensions.css.canvas.width = Math.round(
                      this.dimensions.device.canvas.width / j
                    )),
                    (this.dimensions.css.canvas.height = Math.round(
                      this.dimensions.device.canvas.height / j
                    )),
                    (this.dimensions.css.cell.width =
                      this.dimensions.css.canvas.width / this._bufferService.cols),
                    (this.dimensions.css.cell.height =
                      this.dimensions.css.canvas.height / this._bufferService.rows))
                  for (const O of this._rowElements)
                    ((O.style.width = `${this.dimensions.css.canvas.width}px`),
                      (O.style.height = `${this.dimensions.css.cell.height}px`),
                      (O.style.lineHeight = `${this.dimensions.css.cell.height}px`),
                      (O.style.overflow = 'hidden'))
                  this._dimensionsStyleElement ||
                    ((this._dimensionsStyleElement = this._document.createElement('style')),
                    this._screenElement.appendChild(this._dimensionsStyleElement))
                  const D = `${this._terminalSelector} .${_} span { display: inline-block; height: 100%; vertical-align: top;}`
                  ;((this._dimensionsStyleElement.textContent = D),
                    (this._selectionContainer.style.height = this._viewportElement.style.height),
                    (this._screenElement.style.width = `${this.dimensions.css.canvas.width}px`),
                    (this._screenElement.style.height = `${this.dimensions.css.canvas.height}px`))
                }
                _injectCss(j) {
                  this._themeStyleElement ||
                    ((this._themeStyleElement = this._document.createElement('style')),
                    this._screenElement.appendChild(this._themeStyleElement))
                  let D = `${this._terminalSelector} .${_} { color: ${j.foreground.css}; font-family: ${this._optionsService.rawOptions.fontFamily}; font-size: ${this._optionsService.rawOptions.fontSize}px; font-kerning: none; white-space: pre}`
                  ;((D += `${this._terminalSelector} .${_} .xterm-dim { color: ${n.color.multiplyOpacity(j.foreground, 0.5).css};}`),
                    (D += `${this._terminalSelector} span:not(.xterm-bold) { font-weight: ${this._optionsService.rawOptions.fontWeight};}${this._terminalSelector} span.xterm-bold { font-weight: ${this._optionsService.rawOptions.fontWeightBold};}${this._terminalSelector} span.xterm-italic { font-style: italic;}`))
                  const O = `blink_underline_${this._terminalClass}`,
                    $ = `blink_bar_${this._terminalClass}`,
                    F = `blink_block_${this._terminalClass}`
                  ;((D += `@keyframes ${O} { 50% {  border-bottom-style: hidden; }}`),
                    (D += `@keyframes ${$} { 50% {  box-shadow: none; }}`),
                    (D += `@keyframes ${F} { 0% {  background-color: ${j.cursor.css};  color: ${j.cursorAccent.css}; } 50% {  background-color: inherit;  color: ${j.cursor.css}; }}`),
                    (D += `${this._terminalSelector} .${_}.${S} .xterm-cursor.xterm-cursor-blink.xterm-cursor-underline { animation: ${O} 1s step-end infinite;}${this._terminalSelector} .${_}.${S} .xterm-cursor.xterm-cursor-blink.xterm-cursor-bar { animation: ${$} 1s step-end infinite;}${this._terminalSelector} .${_}.${S} .xterm-cursor.xterm-cursor-blink.xterm-cursor-block { animation: ${F} 1s step-end infinite;}${this._terminalSelector} .${_} .xterm-cursor.xterm-cursor-block { background-color: ${j.cursor.css}; color: ${j.cursorAccent.css};}${this._terminalSelector} .${_} .xterm-cursor.xterm-cursor-block:not(.xterm-cursor-blink) { background-color: ${j.cursor.css} !important; color: ${j.cursorAccent.css} !important;}${this._terminalSelector} .${_} .xterm-cursor.xterm-cursor-outline { outline: 1px solid ${j.cursor.css}; outline-offset: -1px;}${this._terminalSelector} .${_} .xterm-cursor.xterm-cursor-bar { box-shadow: ${this._optionsService.rawOptions.cursorWidth}px 0 0 ${j.cursor.css} inset;}${this._terminalSelector} .${_} .xterm-cursor.xterm-cursor-underline { border-bottom: 1px ${j.cursor.css}; border-bottom-style: solid; height: calc(100% - 1px);}`),
                    (D += `${this._terminalSelector} .${L} { position: absolute; top: 0; left: 0; z-index: 1; pointer-events: none;}${this._terminalSelector}.focus .${L} div { position: absolute; background-color: ${j.selectionBackgroundOpaque.css};}${this._terminalSelector} .${L} div { position: absolute; background-color: ${j.selectionInactiveBackgroundOpaque.css};}`))
                  for (const [W, C] of j.ansi.entries())
                    D += `${this._terminalSelector} .${b}${W} { color: ${C.css}; }${this._terminalSelector} .${b}${W}.xterm-dim { color: ${n.color.multiplyOpacity(C, 0.5).css}; }${this._terminalSelector} .${p}${W} { background-color: ${C.css}; }`
                  ;((D += `${this._terminalSelector} .${b}${f.INVERTED_DEFAULT_COLOR} { color: ${n.color.opaque(j.background).css}; }${this._terminalSelector} .${b}${f.INVERTED_DEFAULT_COLOR}.xterm-dim { color: ${n.color.multiplyOpacity(n.color.opaque(j.background), 0.5).css}; }${this._terminalSelector} .${p}${f.INVERTED_DEFAULT_COLOR} { background-color: ${j.foreground.css}; }`),
                    (this._themeStyleElement.textContent = D))
                }
                _setDefaultSpacing() {
                  const j = this.dimensions.css.cell.width - this._widthCache.get('W', !1, !1)
                  ;((this._rowContainer.style.letterSpacing = `${j}px`),
                    (this._rowFactory.defaultSpacing = j))
                }
                handleDevicePixelRatioChange() {
                  ;(this._updateDimensions(), this._widthCache.clear(), this._setDefaultSpacing())
                }
                _refreshRowElements(j, D) {
                  for (let O = this._rowElements.length; O <= D; O++) {
                    const $ = this._document.createElement('div')
                    ;(this._rowContainer.appendChild($), this._rowElements.push($))
                  }
                  for (; this._rowElements.length > D; )
                    this._rowContainer.removeChild(this._rowElements.pop())
                }
                handleResize(j, D) {
                  ;(this._refreshRowElements(j, D),
                    this._updateDimensions(),
                    this.handleSelectionChanged(
                      this._selectionRenderModel.selectionStart,
                      this._selectionRenderModel.selectionEnd,
                      this._selectionRenderModel.columnSelectMode
                    ))
                }
                handleCharSizeChanged() {
                  ;(this._updateDimensions(), this._widthCache.clear(), this._setDefaultSpacing())
                }
                handleBlur() {
                  ;(this._rowContainer.classList.remove(S),
                    this.renderRows(0, this._bufferService.rows - 1))
                }
                handleFocus() {
                  ;(this._rowContainer.classList.add(S),
                    this.renderRows(this._bufferService.buffer.y, this._bufferService.buffer.y))
                }
                handleSelectionChanged(j, D, O) {
                  if (
                    (this._selectionContainer.replaceChildren(),
                    this._rowFactory.handleSelectionChanged(j, D, O),
                    this.renderRows(0, this._bufferService.rows - 1),
                    !j || !D)
                  )
                    return
                  this._selectionRenderModel.update(this._terminal, j, D, O)
                  const $ = this._selectionRenderModel.viewportStartRow,
                    F = this._selectionRenderModel.viewportEndRow,
                    W = this._selectionRenderModel.viewportCappedStartRow,
                    C = this._selectionRenderModel.viewportCappedEndRow
                  if (W >= this._bufferService.rows || C < 0) return
                  const A = this._document.createDocumentFragment()
                  if (O) {
                    const N = j[0] > D[0]
                    A.appendChild(
                      this._createSelectionElement(W, N ? D[0] : j[0], N ? j[0] : D[0], C - W + 1)
                    )
                  } else {
                    const N = $ === W ? j[0] : 0,
                      B = W === F ? D[0] : this._bufferService.cols
                    A.appendChild(this._createSelectionElement(W, N, B))
                    const z = C - W - 1
                    if (
                      (A.appendChild(
                        this._createSelectionElement(W + 1, 0, this._bufferService.cols, z)
                      ),
                      W !== C)
                    ) {
                      const K = F === C ? D[0] : this._bufferService.cols
                      A.appendChild(this._createSelectionElement(C, 0, K))
                    }
                  }
                  this._selectionContainer.appendChild(A)
                }
                _createSelectionElement(j, D, O, $ = 1) {
                  const F = this._document.createElement('div'),
                    W = D * this.dimensions.css.cell.width
                  let C = this.dimensions.css.cell.width * (O - D)
                  return (
                    W + C > this.dimensions.css.canvas.width &&
                      (C = this.dimensions.css.canvas.width - W),
                    (F.style.height = $ * this.dimensions.css.cell.height + 'px'),
                    (F.style.top = j * this.dimensions.css.cell.height + 'px'),
                    (F.style.left = `${W}px`),
                    (F.style.width = `${C}px`),
                    F
                  )
                }
                handleCursorMove() {}
                _handleOptionsChanged() {
                  ;(this._updateDimensions(),
                    this._injectCss(this._themeService.colors),
                    this._widthCache.setFont(
                      this._optionsService.rawOptions.fontFamily,
                      this._optionsService.rawOptions.fontSize,
                      this._optionsService.rawOptions.fontWeight,
                      this._optionsService.rawOptions.fontWeightBold
                    ),
                    this._setDefaultSpacing())
                }
                clear() {
                  for (const j of this._rowElements) j.replaceChildren()
                }
                renderRows(j, D) {
                  const O = this._bufferService.buffer,
                    $ = O.ybase + O.y,
                    F = Math.min(O.x, this._bufferService.cols - 1),
                    W = this._optionsService.rawOptions.cursorBlink,
                    C = this._optionsService.rawOptions.cursorStyle,
                    A = this._optionsService.rawOptions.cursorInactiveStyle
                  for (let N = j; N <= D; N++) {
                    const B = N + O.ydisp,
                      z = this._rowElements[N],
                      K = O.lines.get(B)
                    if (!z || !K) break
                    z.replaceChildren(
                      ...this._rowFactory.createRow(
                        K,
                        B,
                        B === $,
                        C,
                        A,
                        F,
                        W,
                        this.dimensions.css.cell.width,
                        this._widthCache,
                        -1,
                        -1
                      )
                    )
                  }
                }
                get _terminalSelector() {
                  return `.${v}${this._terminalClass}`
                }
                _handleLinkHover(j) {
                  this._setCellUnderline(j.x1, j.x2, j.y1, j.y2, j.cols, !0)
                }
                _handleLinkLeave(j) {
                  this._setCellUnderline(j.x1, j.x2, j.y1, j.y2, j.cols, !1)
                }
                _setCellUnderline(j, D, O, $, F, W) {
                  ;(O < 0 && (j = 0), $ < 0 && (D = 0))
                  const C = this._bufferService.rows - 1
                  ;((O = Math.max(Math.min(O, C), 0)),
                    ($ = Math.max(Math.min($, C), 0)),
                    (F = Math.min(F, this._bufferService.cols)))
                  const A = this._bufferService.buffer,
                    N = A.ybase + A.y,
                    B = Math.min(A.x, F - 1),
                    z = this._optionsService.rawOptions.cursorBlink,
                    K = this._optionsService.rawOptions.cursorStyle,
                    J = this._optionsService.rawOptions.cursorInactiveStyle
                  for (let Q = O; Q <= $; ++Q) {
                    const H = Q + A.ydisp,
                      E = this._rowElements[Q],
                      G = A.lines.get(H)
                    if (!E || !G) break
                    E.replaceChildren(
                      ...this._rowFactory.createRow(
                        G,
                        H,
                        H === N,
                        K,
                        J,
                        B,
                        z,
                        this.dimensions.css.cell.width,
                        this._widthCache,
                        W ? (Q === O ? j : 0) : -1,
                        W ? (Q === $ ? D : F) - 1 : -1
                      )
                    )
                  }
                }
              })
            i.DomRenderer = P = l(
              [
                u(7, d.IInstantiationService),
                u(8, t.ICharSizeService),
                u(9, d.IOptionsService),
                u(10, d.IBufferService),
                u(11, t.ICoreBrowserService),
                u(12, t.IThemeService)
              ],
              P
            )
          },
          3787: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (_, b, p, S) {
                  var L,
                    M = arguments.length,
                    P = M < 3 ? b : S === null ? (S = Object.getOwnPropertyDescriptor(b, p)) : S
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    P = Reflect.decorate(_, b, p, S)
                  else
                    for (var j = _.length - 1; j >= 0; j--)
                      (L = _[j]) && (P = (M < 3 ? L(P) : M > 3 ? L(b, p, P) : L(b, p)) || P)
                  return (M > 3 && P && Object.defineProperty(b, p, P), P)
                },
              u =
                (this && this.__param) ||
                function (_, b) {
                  return function (p, S) {
                    b(p, S, _)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.DomRendererRowFactory = void 0))
            const a = o(2223),
              h = o(643),
              f = o(511),
              x = o(2585),
              c = o(8055),
              t = o(4725),
              n = o(4269),
              s = o(6171),
              r = o(3734)
            let d = (i.DomRendererRowFactory = class {
              constructor(_, b, p, S, L, M, P) {
                ;((this._document = _),
                  (this._characterJoinerService = b),
                  (this._optionsService = p),
                  (this._coreBrowserService = S),
                  (this._coreService = L),
                  (this._decorationService = M),
                  (this._themeService = P),
                  (this._workCell = new f.CellData()),
                  (this._columnSelectMode = !1),
                  (this.defaultSpacing = 0))
              }
              handleSelectionChanged(_, b, p) {
                ;((this._selectionStart = _),
                  (this._selectionEnd = b),
                  (this._columnSelectMode = p))
              }
              createRow(_, b, p, S, L, M, P, j, D, O, $) {
                const F = [],
                  W = this._characterJoinerService.getJoinedCharacters(b),
                  C = this._themeService.colors
                let A,
                  N = _.getNoBgTrimmedLength()
                p && N < M + 1 && (N = M + 1)
                let B = 0,
                  z = '',
                  K = 0,
                  J = 0,
                  Q = 0,
                  H = !1,
                  E = 0,
                  G = !1,
                  q = 0
                const Z = [],
                  Y = O !== -1 && $ !== -1
                for (let V = 0; V < N; V++) {
                  _.loadCell(V, this._workCell)
                  let se = this._workCell.getWidth()
                  if (se === 0) continue
                  let ne = !1,
                    ue = V,
                    re = this._workCell
                  if (W.length > 0 && V === W[0][0]) {
                    ne = !0
                    const fe = W.shift()
                    ;((re = new n.JoinedCellData(
                      this._workCell,
                      _.translateToString(!0, fe[0], fe[1]),
                      fe[1] - fe[0]
                    )),
                      (ue = fe[1] - 1),
                      (se = re.getWidth()))
                  }
                  const Ee = this._isCellInSelection(V, b),
                    Fe = p && V === M,
                    Le = Y && V >= O && V <= $
                  let Ae = !1
                  this._decorationService.forEachDecorationAtCell(V, b, void 0, (fe) => {
                    Ae = !0
                  })
                  let Be = re.getChars() || h.WHITESPACE_CELL_CHAR
                  if (
                    (Be === ' ' && (re.isUnderline() || re.isOverline()) && (Be = ' '),
                    (q = se * j - D.get(Be, re.isBold(), re.isItalic())),
                    A)
                  ) {
                    if (
                      B &&
                      ((Ee && G) || (!Ee && !G && re.bg === K)) &&
                      ((Ee && G && C.selectionForeground) || re.fg === J) &&
                      re.extended.ext === Q &&
                      Le === H &&
                      q === E &&
                      !Fe &&
                      !ne &&
                      !Ae
                    ) {
                      ;(re.isInvisible() ? (z += h.WHITESPACE_CELL_CHAR) : (z += Be), B++)
                      continue
                    }
                    ;(B && (A.textContent = z),
                      (A = this._document.createElement('span')),
                      (B = 0),
                      (z = ''))
                  } else A = this._document.createElement('span')
                  if (
                    ((K = re.bg),
                    (J = re.fg),
                    (Q = re.extended.ext),
                    (H = Le),
                    (E = q),
                    (G = Ee),
                    ne && M >= V && M <= ue && (M = V),
                    !this._coreService.isCursorHidden &&
                      Fe &&
                      this._coreService.isCursorInitialized)
                  ) {
                    if ((Z.push('xterm-cursor'), this._coreBrowserService.isFocused))
                      (P && Z.push('xterm-cursor-blink'),
                        Z.push(
                          S === 'bar'
                            ? 'xterm-cursor-bar'
                            : S === 'underline'
                              ? 'xterm-cursor-underline'
                              : 'xterm-cursor-block'
                        ))
                    else if (L)
                      switch (L) {
                        case 'outline':
                          Z.push('xterm-cursor-outline')
                          break
                        case 'block':
                          Z.push('xterm-cursor-block')
                          break
                        case 'bar':
                          Z.push('xterm-cursor-bar')
                          break
                        case 'underline':
                          Z.push('xterm-cursor-underline')
                      }
                  }
                  if (
                    (re.isBold() && Z.push('xterm-bold'),
                    re.isItalic() && Z.push('xterm-italic'),
                    re.isDim() && Z.push('xterm-dim'),
                    (z = re.isInvisible()
                      ? h.WHITESPACE_CELL_CHAR
                      : re.getChars() || h.WHITESPACE_CELL_CHAR),
                    re.isUnderline() &&
                      (Z.push(`xterm-underline-${re.extended.underlineStyle}`),
                      z === ' ' && (z = ' '),
                      !re.isUnderlineColorDefault()))
                  )
                    if (re.isUnderlineColorRGB())
                      A.style.textDecorationColor = `rgb(${r.AttributeData.toColorRGB(re.getUnderlineColor()).join(',')})`
                    else {
                      let fe = re.getUnderlineColor()
                      ;(this._optionsService.rawOptions.drawBoldTextInBrightColors &&
                        re.isBold() &&
                        fe < 8 &&
                        (fe += 8),
                        (A.style.textDecorationColor = C.ansi[fe].css))
                    }
                  ;(re.isOverline() && (Z.push('xterm-overline'), z === ' ' && (z = ' ')),
                    re.isStrikethrough() && Z.push('xterm-strikethrough'),
                    Le && (A.style.textDecoration = 'underline'))
                  let ve = re.getFgColor(),
                    X = re.getFgColorMode(),
                    ae = re.getBgColor(),
                    he = re.getBgColorMode()
                  const Ie = !!re.isInverse()
                  if (Ie) {
                    const fe = ve
                    ;((ve = ae), (ae = fe))
                    const dt = X
                    ;((X = he), (he = dt))
                  }
                  let je,
                    Me,
                    Re,
                    Ue = !1
                  switch (
                    (this._decorationService.forEachDecorationAtCell(V, b, void 0, (fe) => {
                      ;(fe.options.layer !== 'top' && Ue) ||
                        (fe.backgroundColorRGB &&
                          ((he = 50331648),
                          (ae = (fe.backgroundColorRGB.rgba >> 8) & 16777215),
                          (je = fe.backgroundColorRGB)),
                        fe.foregroundColorRGB &&
                          ((X = 50331648),
                          (ve = (fe.foregroundColorRGB.rgba >> 8) & 16777215),
                          (Me = fe.foregroundColorRGB)),
                        (Ue = fe.options.layer === 'top'))
                    }),
                    !Ue &&
                      Ee &&
                      ((je = this._coreBrowserService.isFocused
                        ? C.selectionBackgroundOpaque
                        : C.selectionInactiveBackgroundOpaque),
                      (ae = (je.rgba >> 8) & 16777215),
                      (he = 50331648),
                      (Ue = !0),
                      C.selectionForeground &&
                        ((X = 50331648),
                        (ve = (C.selectionForeground.rgba >> 8) & 16777215),
                        (Me = C.selectionForeground))),
                    Ue && Z.push('xterm-decoration-top'),
                    he)
                  ) {
                    case 16777216:
                    case 33554432:
                      ;((Re = C.ansi[ae]), Z.push(`xterm-bg-${ae}`))
                      break
                    case 50331648:
                      ;((Re = c.channels.toColor(ae >> 16, (ae >> 8) & 255, 255 & ae)),
                        this._addStyle(
                          A,
                          `background-color:#${v((ae >>> 0).toString(16), '0', 6)}`
                        ))
                      break
                    default:
                      Ie
                        ? ((Re = C.foreground), Z.push(`xterm-bg-${a.INVERTED_DEFAULT_COLOR}`))
                        : (Re = C.background)
                  }
                  switch ((je || (re.isDim() && (je = c.color.multiplyOpacity(Re, 0.5))), X)) {
                    case 16777216:
                    case 33554432:
                      ;(re.isBold() &&
                        ve < 8 &&
                        this._optionsService.rawOptions.drawBoldTextInBrightColors &&
                        (ve += 8),
                        this._applyMinimumContrast(A, Re, C.ansi[ve], re, je, void 0) ||
                          Z.push(`xterm-fg-${ve}`))
                      break
                    case 50331648:
                      const fe = c.channels.toColor((ve >> 16) & 255, (ve >> 8) & 255, 255 & ve)
                      this._applyMinimumContrast(A, Re, fe, re, je, Me) ||
                        this._addStyle(A, `color:#${v(ve.toString(16), '0', 6)}`)
                      break
                    default:
                      this._applyMinimumContrast(A, Re, C.foreground, re, je, Me) ||
                        (Ie && Z.push(`xterm-fg-${a.INVERTED_DEFAULT_COLOR}`))
                  }
                  ;(Z.length && ((A.className = Z.join(' ')), (Z.length = 0)),
                    Fe || ne || Ae ? (A.textContent = z) : B++,
                    q !== this.defaultSpacing && (A.style.letterSpacing = `${q}px`),
                    F.push(A),
                    (V = ue))
                }
                return (A && B && (A.textContent = z), F)
              }
              _applyMinimumContrast(_, b, p, S, L, M) {
                if (
                  this._optionsService.rawOptions.minimumContrastRatio === 1 ||
                  (0, s.treatGlyphAsBackgroundColor)(S.getCode())
                )
                  return !1
                const P = this._getContrastCache(S)
                let j
                if ((L || M || (j = P.getColor(b.rgba, p.rgba)), j === void 0)) {
                  const D =
                    this._optionsService.rawOptions.minimumContrastRatio / (S.isDim() ? 2 : 1)
                  ;((j = c.color.ensureContrastRatio(L || b, M || p, D)),
                    P.setColor((L || b).rgba, (M || p).rgba, j ?? null))
                }
                return !!j && (this._addStyle(_, `color:${j.css}`), !0)
              }
              _getContrastCache(_) {
                return _.isDim()
                  ? this._themeService.colors.halfContrastCache
                  : this._themeService.colors.contrastCache
              }
              _addStyle(_, b) {
                _.setAttribute('style', `${_.getAttribute('style') || ''}${b};`)
              }
              _isCellInSelection(_, b) {
                const p = this._selectionStart,
                  S = this._selectionEnd
                return (
                  !(!p || !S) &&
                  (this._columnSelectMode
                    ? p[0] <= S[0]
                      ? _ >= p[0] && b >= p[1] && _ < S[0] && b <= S[1]
                      : _ < p[0] && b >= p[1] && _ >= S[0] && b <= S[1]
                    : (b > p[1] && b < S[1]) ||
                      (p[1] === S[1] && b === p[1] && _ >= p[0] && _ < S[0]) ||
                      (p[1] < S[1] && b === S[1] && _ < S[0]) ||
                      (p[1] < S[1] && b === p[1] && _ >= p[0]))
                )
              }
            })
            function v(_, b, p) {
              for (; _.length < p; ) _ = b + _
              return _
            }
            i.DomRendererRowFactory = d = l(
              [
                u(1, t.ICharacterJoinerService),
                u(2, x.IOptionsService),
                u(3, t.ICoreBrowserService),
                u(4, x.ICoreService),
                u(5, x.IDecorationService),
                u(6, t.IThemeService)
              ],
              d
            )
          },
          2550: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.WidthCache = void 0),
              (i.WidthCache = class {
                constructor(o, l) {
                  ;((this._flat = new Float32Array(256)),
                    (this._font = ''),
                    (this._fontSize = 0),
                    (this._weight = 'normal'),
                    (this._weightBold = 'bold'),
                    (this._measureElements = []),
                    (this._container = o.createElement('div')),
                    this._container.classList.add('xterm-width-cache-measure-container'),
                    this._container.setAttribute('aria-hidden', 'true'),
                    (this._container.style.whiteSpace = 'pre'),
                    (this._container.style.fontKerning = 'none'))
                  const u = o.createElement('span')
                  u.classList.add('xterm-char-measure-element')
                  const a = o.createElement('span')
                  ;(a.classList.add('xterm-char-measure-element'), (a.style.fontWeight = 'bold'))
                  const h = o.createElement('span')
                  ;(h.classList.add('xterm-char-measure-element'), (h.style.fontStyle = 'italic'))
                  const f = o.createElement('span')
                  ;(f.classList.add('xterm-char-measure-element'),
                    (f.style.fontWeight = 'bold'),
                    (f.style.fontStyle = 'italic'),
                    (this._measureElements = [u, a, h, f]),
                    this._container.appendChild(u),
                    this._container.appendChild(a),
                    this._container.appendChild(h),
                    this._container.appendChild(f),
                    l.appendChild(this._container),
                    this.clear())
                }
                dispose() {
                  ;(this._container.remove(),
                    (this._measureElements.length = 0),
                    (this._holey = void 0))
                }
                clear() {
                  ;(this._flat.fill(-9999), (this._holey = new Map()))
                }
                setFont(o, l, u, a) {
                  ;(o === this._font &&
                    l === this._fontSize &&
                    u === this._weight &&
                    a === this._weightBold) ||
                    ((this._font = o),
                    (this._fontSize = l),
                    (this._weight = u),
                    (this._weightBold = a),
                    (this._container.style.fontFamily = this._font),
                    (this._container.style.fontSize = `${this._fontSize}px`),
                    (this._measureElements[0].style.fontWeight = `${u}`),
                    (this._measureElements[1].style.fontWeight = `${a}`),
                    (this._measureElements[2].style.fontWeight = `${u}`),
                    (this._measureElements[3].style.fontWeight = `${a}`),
                    this.clear())
                }
                get(o, l, u) {
                  let a = 0
                  if (!l && !u && o.length === 1 && (a = o.charCodeAt(0)) < 256) {
                    if (this._flat[a] !== -9999) return this._flat[a]
                    const x = this._measure(o, 0)
                    return (x > 0 && (this._flat[a] = x), x)
                  }
                  let h = o
                  ;(l && (h += 'B'), u && (h += 'I'))
                  let f = this._holey.get(h)
                  if (f === void 0) {
                    let x = 0
                    ;(l && (x |= 1),
                      u && (x |= 2),
                      (f = this._measure(o, x)),
                      f > 0 && this._holey.set(h, f))
                  }
                  return f
                }
                _measure(o, l) {
                  const u = this._measureElements[l]
                  return ((u.textContent = o.repeat(32)), u.offsetWidth / 32)
                }
              }))
          },
          2223: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.TEXT_BASELINE = i.DIM_OPACITY = i.INVERTED_DEFAULT_COLOR = void 0))
            const l = o(6114)
            ;((i.INVERTED_DEFAULT_COLOR = 257),
              (i.DIM_OPACITY = 0.5),
              (i.TEXT_BASELINE = l.isFirefox || l.isLegacyEdge ? 'bottom' : 'ideographic'))
          },
          6171: (I, i) => {
            function o(u) {
              return 57508 <= u && u <= 57558
            }
            function l(u) {
              return (
                (u >= 128512 && u <= 128591) ||
                (u >= 127744 && u <= 128511) ||
                (u >= 128640 && u <= 128767) ||
                (u >= 9728 && u <= 9983) ||
                (u >= 9984 && u <= 10175) ||
                (u >= 65024 && u <= 65039) ||
                (u >= 129280 && u <= 129535) ||
                (u >= 127462 && u <= 127487)
              )
            }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.computeNextVariantOffset =
                i.createRenderDimensions =
                i.treatGlyphAsBackgroundColor =
                i.allowRescaling =
                i.isEmoji =
                i.isRestrictedPowerlineGlyph =
                i.isPowerlineGlyph =
                i.throwIfFalsy =
                  void 0),
              (i.throwIfFalsy = function (u) {
                if (!u) throw new Error('value must not be falsy')
                return u
              }),
              (i.isPowerlineGlyph = o),
              (i.isRestrictedPowerlineGlyph = function (u) {
                return 57520 <= u && u <= 57527
              }),
              (i.isEmoji = l),
              (i.allowRescaling = function (u, a, h, f) {
                return (
                  a === 1 &&
                  h > Math.ceil(1.5 * f) &&
                  u !== void 0 &&
                  u > 255 &&
                  !l(u) &&
                  !o(u) &&
                  !(function (x) {
                    return 57344 <= x && x <= 63743
                  })(u)
                )
              }),
              (i.treatGlyphAsBackgroundColor = function (u) {
                return (
                  o(u) ||
                  (function (a) {
                    return 9472 <= a && a <= 9631
                  })(u)
                )
              }),
              (i.createRenderDimensions = function () {
                return {
                  css: { canvas: { width: 0, height: 0 }, cell: { width: 0, height: 0 } },
                  device: {
                    canvas: { width: 0, height: 0 },
                    cell: { width: 0, height: 0 },
                    char: { width: 0, height: 0, left: 0, top: 0 }
                  }
                }
              }),
              (i.computeNextVariantOffset = function (u, a, h = 0) {
                return (u - (2 * Math.round(a) - h)) % (2 * Math.round(a))
              }))
          },
          6052: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.createSelectionRenderModel = void 0))
            class o {
              constructor() {
                this.clear()
              }
              clear() {
                ;((this.hasSelection = !1),
                  (this.columnSelectMode = !1),
                  (this.viewportStartRow = 0),
                  (this.viewportEndRow = 0),
                  (this.viewportCappedStartRow = 0),
                  (this.viewportCappedEndRow = 0),
                  (this.startCol = 0),
                  (this.endCol = 0),
                  (this.selectionStart = void 0),
                  (this.selectionEnd = void 0))
              }
              update(u, a, h, f = !1) {
                if (
                  ((this.selectionStart = a),
                  (this.selectionEnd = h),
                  !a || !h || (a[0] === h[0] && a[1] === h[1]))
                )
                  return void this.clear()
                const x = u.buffers.active.ydisp,
                  c = a[1] - x,
                  t = h[1] - x,
                  n = Math.max(c, 0),
                  s = Math.min(t, u.rows - 1)
                n >= u.rows || s < 0
                  ? this.clear()
                  : ((this.hasSelection = !0),
                    (this.columnSelectMode = f),
                    (this.viewportStartRow = c),
                    (this.viewportEndRow = t),
                    (this.viewportCappedStartRow = n),
                    (this.viewportCappedEndRow = s),
                    (this.startCol = a[0]),
                    (this.endCol = h[0]))
              }
              isCellSelected(u, a, h) {
                return (
                  !!this.hasSelection &&
                  ((h -= u.buffer.active.viewportY),
                  this.columnSelectMode
                    ? this.startCol <= this.endCol
                      ? a >= this.startCol &&
                        h >= this.viewportCappedStartRow &&
                        a < this.endCol &&
                        h <= this.viewportCappedEndRow
                      : a < this.startCol &&
                        h >= this.viewportCappedStartRow &&
                        a >= this.endCol &&
                        h <= this.viewportCappedEndRow
                    : (h > this.viewportStartRow && h < this.viewportEndRow) ||
                      (this.viewportStartRow === this.viewportEndRow &&
                        h === this.viewportStartRow &&
                        a >= this.startCol &&
                        a < this.endCol) ||
                      (this.viewportStartRow < this.viewportEndRow &&
                        h === this.viewportEndRow &&
                        a < this.endCol) ||
                      (this.viewportStartRow < this.viewportEndRow &&
                        h === this.viewportStartRow &&
                        a >= this.startCol))
                )
              }
            }
            i.createSelectionRenderModel = function () {
              return new o()
            }
          },
          456: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.SelectionModel = void 0),
              (i.SelectionModel = class {
                constructor(o) {
                  ;((this._bufferService = o),
                    (this.isSelectAllActive = !1),
                    (this.selectionStartLength = 0))
                }
                clearSelection() {
                  ;((this.selectionStart = void 0),
                    (this.selectionEnd = void 0),
                    (this.isSelectAllActive = !1),
                    (this.selectionStartLength = 0))
                }
                get finalSelectionStart() {
                  return this.isSelectAllActive
                    ? [0, 0]
                    : this.selectionEnd && this.selectionStart && this.areSelectionValuesReversed()
                      ? this.selectionEnd
                      : this.selectionStart
                }
                get finalSelectionEnd() {
                  if (this.isSelectAllActive)
                    return [
                      this._bufferService.cols,
                      this._bufferService.buffer.ybase + this._bufferService.rows - 1
                    ]
                  if (this.selectionStart) {
                    if (!this.selectionEnd || this.areSelectionValuesReversed()) {
                      const o = this.selectionStart[0] + this.selectionStartLength
                      return o > this._bufferService.cols
                        ? o % this._bufferService.cols == 0
                          ? [
                              this._bufferService.cols,
                              this.selectionStart[1] + Math.floor(o / this._bufferService.cols) - 1
                            ]
                          : [
                              o % this._bufferService.cols,
                              this.selectionStart[1] + Math.floor(o / this._bufferService.cols)
                            ]
                        : [o, this.selectionStart[1]]
                    }
                    if (
                      this.selectionStartLength &&
                      this.selectionEnd[1] === this.selectionStart[1]
                    ) {
                      const o = this.selectionStart[0] + this.selectionStartLength
                      return o > this._bufferService.cols
                        ? [
                            o % this._bufferService.cols,
                            this.selectionStart[1] + Math.floor(o / this._bufferService.cols)
                          ]
                        : [Math.max(o, this.selectionEnd[0]), this.selectionEnd[1]]
                    }
                    return this.selectionEnd
                  }
                }
                areSelectionValuesReversed() {
                  const o = this.selectionStart,
                    l = this.selectionEnd
                  return !(!o || !l) && (o[1] > l[1] || (o[1] === l[1] && o[0] > l[0]))
                }
                handleTrim(o) {
                  return (
                    this.selectionStart && (this.selectionStart[1] -= o),
                    this.selectionEnd && (this.selectionEnd[1] -= o),
                    this.selectionEnd && this.selectionEnd[1] < 0
                      ? (this.clearSelection(), !0)
                      : (this.selectionStart &&
                          this.selectionStart[1] < 0 &&
                          (this.selectionStart[1] = 0),
                        !1)
                  )
                }
              }))
          },
          428: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (s, r, d, v) {
                  var _,
                    b = arguments.length,
                    p = b < 3 ? r : v === null ? (v = Object.getOwnPropertyDescriptor(r, d)) : v
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    p = Reflect.decorate(s, r, d, v)
                  else
                    for (var S = s.length - 1; S >= 0; S--)
                      (_ = s[S]) && (p = (b < 3 ? _(p) : b > 3 ? _(r, d, p) : _(r, d)) || p)
                  return (b > 3 && p && Object.defineProperty(r, d, p), p)
                },
              u =
                (this && this.__param) ||
                function (s, r) {
                  return function (d, v) {
                    r(d, v, s)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.CharSizeService = void 0))
            const a = o(2585),
              h = o(8460),
              f = o(844)
            let x = (i.CharSizeService = class extends f.Disposable {
              get hasValidSize() {
                return this.width > 0 && this.height > 0
              }
              constructor(s, r, d) {
                ;(super(),
                  (this._optionsService = d),
                  (this.width = 0),
                  (this.height = 0),
                  (this._onCharSizeChange = this.register(new h.EventEmitter())),
                  (this.onCharSizeChange = this._onCharSizeChange.event))
                try {
                  this._measureStrategy = this.register(new n(this._optionsService))
                } catch {
                  this._measureStrategy = this.register(new t(s, r, this._optionsService))
                }
                this.register(
                  this._optionsService.onMultipleOptionChange(['fontFamily', 'fontSize'], () =>
                    this.measure()
                  )
                )
              }
              measure() {
                const s = this._measureStrategy.measure()
                ;(s.width === this.width && s.height === this.height) ||
                  ((this.width = s.width), (this.height = s.height), this._onCharSizeChange.fire())
              }
            })
            i.CharSizeService = x = l([u(2, a.IOptionsService)], x)
            class c extends f.Disposable {
              constructor() {
                ;(super(...arguments), (this._result = { width: 0, height: 0 }))
              }
              _validateAndSet(r, d) {
                r !== void 0 &&
                  r > 0 &&
                  d !== void 0 &&
                  d > 0 &&
                  ((this._result.width = r), (this._result.height = d))
              }
            }
            class t extends c {
              constructor(r, d, v) {
                ;(super(),
                  (this._document = r),
                  (this._parentElement = d),
                  (this._optionsService = v),
                  (this._measureElement = this._document.createElement('span')),
                  this._measureElement.classList.add('xterm-char-measure-element'),
                  (this._measureElement.textContent = 'W'.repeat(32)),
                  this._measureElement.setAttribute('aria-hidden', 'true'),
                  (this._measureElement.style.whiteSpace = 'pre'),
                  (this._measureElement.style.fontKerning = 'none'),
                  this._parentElement.appendChild(this._measureElement))
              }
              measure() {
                return (
                  (this._measureElement.style.fontFamily =
                    this._optionsService.rawOptions.fontFamily),
                  (this._measureElement.style.fontSize = `${this._optionsService.rawOptions.fontSize}px`),
                  this._validateAndSet(
                    Number(this._measureElement.offsetWidth) / 32,
                    Number(this._measureElement.offsetHeight)
                  ),
                  this._result
                )
              }
            }
            class n extends c {
              constructor(r) {
                ;(super(),
                  (this._optionsService = r),
                  (this._canvas = new OffscreenCanvas(100, 100)),
                  (this._ctx = this._canvas.getContext('2d')))
                const d = this._ctx.measureText('W')
                if (
                  !('width' in d && 'fontBoundingBoxAscent' in d && 'fontBoundingBoxDescent' in d)
                )
                  throw new Error('Required font metrics not supported')
              }
              measure() {
                this._ctx.font = `${this._optionsService.rawOptions.fontSize}px ${this._optionsService.rawOptions.fontFamily}`
                const r = this._ctx.measureText('W')
                return (
                  this._validateAndSet(r.width, r.fontBoundingBoxAscent + r.fontBoundingBoxDescent),
                  this._result
                )
              }
            }
          },
          4269: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (n, s, r, d) {
                  var v,
                    _ = arguments.length,
                    b = _ < 3 ? s : d === null ? (d = Object.getOwnPropertyDescriptor(s, r)) : d
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    b = Reflect.decorate(n, s, r, d)
                  else
                    for (var p = n.length - 1; p >= 0; p--)
                      (v = n[p]) && (b = (_ < 3 ? v(b) : _ > 3 ? v(s, r, b) : v(s, r)) || b)
                  return (_ > 3 && b && Object.defineProperty(s, r, b), b)
                },
              u =
                (this && this.__param) ||
                function (n, s) {
                  return function (r, d) {
                    s(r, d, n)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.CharacterJoinerService = i.JoinedCellData = void 0))
            const a = o(3734),
              h = o(643),
              f = o(511),
              x = o(2585)
            class c extends a.AttributeData {
              constructor(s, r, d) {
                ;(super(),
                  (this.content = 0),
                  (this.combinedData = ''),
                  (this.fg = s.fg),
                  (this.bg = s.bg),
                  (this.combinedData = r),
                  (this._width = d))
              }
              isCombined() {
                return 2097152
              }
              getWidth() {
                return this._width
              }
              getChars() {
                return this.combinedData
              }
              getCode() {
                return 2097151
              }
              setFromCharData(s) {
                throw new Error('not implemented')
              }
              getAsCharData() {
                return [this.fg, this.getChars(), this.getWidth(), this.getCode()]
              }
            }
            i.JoinedCellData = c
            let t = (i.CharacterJoinerService = class Vs {
              constructor(s) {
                ;((this._bufferService = s),
                  (this._characterJoiners = []),
                  (this._nextCharacterJoinerId = 0),
                  (this._workCell = new f.CellData()))
              }
              register(s) {
                const r = { id: this._nextCharacterJoinerId++, handler: s }
                return (this._characterJoiners.push(r), r.id)
              }
              deregister(s) {
                for (let r = 0; r < this._characterJoiners.length; r++)
                  if (this._characterJoiners[r].id === s)
                    return (this._characterJoiners.splice(r, 1), !0)
                return !1
              }
              getJoinedCharacters(s) {
                if (this._characterJoiners.length === 0) return []
                const r = this._bufferService.buffer.lines.get(s)
                if (!r || r.length === 0) return []
                const d = [],
                  v = r.translateToString(!0)
                let _ = 0,
                  b = 0,
                  p = 0,
                  S = r.getFg(0),
                  L = r.getBg(0)
                for (let M = 0; M < r.getTrimmedLength(); M++)
                  if ((r.loadCell(M, this._workCell), this._workCell.getWidth() !== 0)) {
                    if (this._workCell.fg !== S || this._workCell.bg !== L) {
                      if (M - _ > 1) {
                        const P = this._getJoinedRanges(v, p, b, r, _)
                        for (let j = 0; j < P.length; j++) d.push(P[j])
                      }
                      ;((_ = M), (p = b), (S = this._workCell.fg), (L = this._workCell.bg))
                    }
                    b += this._workCell.getChars().length || h.WHITESPACE_CELL_CHAR.length
                  }
                if (this._bufferService.cols - _ > 1) {
                  const M = this._getJoinedRanges(v, p, b, r, _)
                  for (let P = 0; P < M.length; P++) d.push(M[P])
                }
                return d
              }
              _getJoinedRanges(s, r, d, v, _) {
                const b = s.substring(r, d)
                let p = []
                try {
                  p = this._characterJoiners[0].handler(b)
                } catch (S) {
                  console.error(S)
                }
                for (let S = 1; S < this._characterJoiners.length; S++)
                  try {
                    const L = this._characterJoiners[S].handler(b)
                    for (let M = 0; M < L.length; M++) Vs._mergeRanges(p, L[M])
                  } catch (L) {
                    console.error(L)
                  }
                return (this._stringRangesToCellRanges(p, v, _), p)
              }
              _stringRangesToCellRanges(s, r, d) {
                let v = 0,
                  _ = !1,
                  b = 0,
                  p = s[v]
                if (p) {
                  for (let S = d; S < this._bufferService.cols; S++) {
                    const L = r.getWidth(S),
                      M = r.getString(S).length || h.WHITESPACE_CELL_CHAR.length
                    if (L !== 0) {
                      if ((!_ && p[0] <= b && ((p[0] = S), (_ = !0)), p[1] <= b)) {
                        if (((p[1] = S), (p = s[++v]), !p)) break
                        p[0] <= b ? ((p[0] = S), (_ = !0)) : (_ = !1)
                      }
                      b += M
                    }
                  }
                  p && (p[1] = this._bufferService.cols)
                }
              }
              static _mergeRanges(s, r) {
                let d = !1
                for (let v = 0; v < s.length; v++) {
                  const _ = s[v]
                  if (d) {
                    if (r[1] <= _[0]) return ((s[v - 1][1] = r[1]), s)
                    if (r[1] <= _[1])
                      return ((s[v - 1][1] = Math.max(r[1], _[1])), s.splice(v, 1), s)
                    ;(s.splice(v, 1), v--)
                  } else {
                    if (r[1] <= _[0]) return (s.splice(v, 0, r), s)
                    if (r[1] <= _[1]) return ((_[0] = Math.min(r[0], _[0])), s)
                    r[0] < _[1] && ((_[0] = Math.min(r[0], _[0])), (d = !0))
                  }
                }
                return (d ? (s[s.length - 1][1] = r[1]) : s.push(r), s)
              }
            })
            i.CharacterJoinerService = t = l([u(0, x.IBufferService)], t)
          },
          5114: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.CoreBrowserService = void 0))
            const l = o(844),
              u = o(8460),
              a = o(3656)
            class h extends l.Disposable {
              constructor(c, t, n) {
                ;(super(),
                  (this._textarea = c),
                  (this._window = t),
                  (this.mainDocument = n),
                  (this._isFocused = !1),
                  (this._cachedIsFocused = void 0),
                  (this._screenDprMonitor = new f(this._window)),
                  (this._onDprChange = this.register(new u.EventEmitter())),
                  (this.onDprChange = this._onDprChange.event),
                  (this._onWindowChange = this.register(new u.EventEmitter())),
                  (this.onWindowChange = this._onWindowChange.event),
                  this.register(this.onWindowChange((s) => this._screenDprMonitor.setWindow(s))),
                  this.register(
                    (0, u.forwardEvent)(this._screenDprMonitor.onDprChange, this._onDprChange)
                  ),
                  this._textarea.addEventListener('focus', () => (this._isFocused = !0)),
                  this._textarea.addEventListener('blur', () => (this._isFocused = !1)))
              }
              get window() {
                return this._window
              }
              set window(c) {
                this._window !== c && ((this._window = c), this._onWindowChange.fire(this._window))
              }
              get dpr() {
                return this.window.devicePixelRatio
              }
              get isFocused() {
                return (
                  this._cachedIsFocused === void 0 &&
                    ((this._cachedIsFocused =
                      this._isFocused && this._textarea.ownerDocument.hasFocus()),
                    queueMicrotask(() => (this._cachedIsFocused = void 0))),
                  this._cachedIsFocused
                )
              }
            }
            i.CoreBrowserService = h
            class f extends l.Disposable {
              constructor(c) {
                ;(super(),
                  (this._parentWindow = c),
                  (this._windowResizeListener = this.register(new l.MutableDisposable())),
                  (this._onDprChange = this.register(new u.EventEmitter())),
                  (this.onDprChange = this._onDprChange.event),
                  (this._outerListener = () => this._setDprAndFireIfDiffers()),
                  (this._currentDevicePixelRatio = this._parentWindow.devicePixelRatio),
                  this._updateDpr(),
                  this._setWindowResizeListener(),
                  this.register((0, l.toDisposable)(() => this.clearListener())))
              }
              setWindow(c) {
                ;((this._parentWindow = c),
                  this._setWindowResizeListener(),
                  this._setDprAndFireIfDiffers())
              }
              _setWindowResizeListener() {
                this._windowResizeListener.value = (0, a.addDisposableDomListener)(
                  this._parentWindow,
                  'resize',
                  () => this._setDprAndFireIfDiffers()
                )
              }
              _setDprAndFireIfDiffers() {
                ;(this._parentWindow.devicePixelRatio !== this._currentDevicePixelRatio &&
                  this._onDprChange.fire(this._parentWindow.devicePixelRatio),
                  this._updateDpr())
              }
              _updateDpr() {
                this._outerListener &&
                  (this._resolutionMediaMatchList?.removeListener(this._outerListener),
                  (this._currentDevicePixelRatio = this._parentWindow.devicePixelRatio),
                  (this._resolutionMediaMatchList = this._parentWindow.matchMedia(
                    `screen and (resolution: ${this._parentWindow.devicePixelRatio}dppx)`
                  )),
                  this._resolutionMediaMatchList.addListener(this._outerListener))
              }
              clearListener() {
                this._resolutionMediaMatchList &&
                  this._outerListener &&
                  (this._resolutionMediaMatchList.removeListener(this._outerListener),
                  (this._resolutionMediaMatchList = void 0),
                  (this._outerListener = void 0))
              }
            }
          },
          779: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.LinkProviderService = void 0))
            const l = o(844)
            class u extends l.Disposable {
              constructor() {
                ;(super(),
                  (this.linkProviders = []),
                  this.register((0, l.toDisposable)(() => (this.linkProviders.length = 0))))
              }
              registerLinkProvider(h) {
                return (
                  this.linkProviders.push(h),
                  {
                    dispose: () => {
                      const f = this.linkProviders.indexOf(h)
                      f !== -1 && this.linkProviders.splice(f, 1)
                    }
                  }
                )
              }
            }
            i.LinkProviderService = u
          },
          8934: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (x, c, t, n) {
                  var s,
                    r = arguments.length,
                    d = r < 3 ? c : n === null ? (n = Object.getOwnPropertyDescriptor(c, t)) : n
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    d = Reflect.decorate(x, c, t, n)
                  else
                    for (var v = x.length - 1; v >= 0; v--)
                      (s = x[v]) && (d = (r < 3 ? s(d) : r > 3 ? s(c, t, d) : s(c, t)) || d)
                  return (r > 3 && d && Object.defineProperty(c, t, d), d)
                },
              u =
                (this && this.__param) ||
                function (x, c) {
                  return function (t, n) {
                    c(t, n, x)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.MouseService = void 0))
            const a = o(4725),
              h = o(9806)
            let f = (i.MouseService = class {
              constructor(x, c) {
                ;((this._renderService = x), (this._charSizeService = c))
              }
              getCoords(x, c, t, n, s) {
                return (0, h.getCoords)(
                  window,
                  x,
                  c,
                  t,
                  n,
                  this._charSizeService.hasValidSize,
                  this._renderService.dimensions.css.cell.width,
                  this._renderService.dimensions.css.cell.height,
                  s
                )
              }
              getMouseReportCoords(x, c) {
                const t = (0, h.getCoordsRelativeToElement)(window, x, c)
                if (this._charSizeService.hasValidSize)
                  return (
                    (t[0] = Math.min(
                      Math.max(t[0], 0),
                      this._renderService.dimensions.css.canvas.width - 1
                    )),
                    (t[1] = Math.min(
                      Math.max(t[1], 0),
                      this._renderService.dimensions.css.canvas.height - 1
                    )),
                    {
                      col: Math.floor(t[0] / this._renderService.dimensions.css.cell.width),
                      row: Math.floor(t[1] / this._renderService.dimensions.css.cell.height),
                      x: Math.floor(t[0]),
                      y: Math.floor(t[1])
                    }
                  )
              }
            })
            i.MouseService = f = l([u(0, a.IRenderService), u(1, a.ICharSizeService)], f)
          },
          3230: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (s, r, d, v) {
                  var _,
                    b = arguments.length,
                    p = b < 3 ? r : v === null ? (v = Object.getOwnPropertyDescriptor(r, d)) : v
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    p = Reflect.decorate(s, r, d, v)
                  else
                    for (var S = s.length - 1; S >= 0; S--)
                      (_ = s[S]) && (p = (b < 3 ? _(p) : b > 3 ? _(r, d, p) : _(r, d)) || p)
                  return (b > 3 && p && Object.defineProperty(r, d, p), p)
                },
              u =
                (this && this.__param) ||
                function (s, r) {
                  return function (d, v) {
                    r(d, v, s)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.RenderService = void 0))
            const a = o(6193),
              h = o(4725),
              f = o(8460),
              x = o(844),
              c = o(7226),
              t = o(2585)
            let n = (i.RenderService = class extends x.Disposable {
              get dimensions() {
                return this._renderer.value.dimensions
              }
              constructor(s, r, d, v, _, b, p, S) {
                ;(super(),
                  (this._rowCount = s),
                  (this._charSizeService = v),
                  (this._renderer = this.register(new x.MutableDisposable())),
                  (this._pausedResizeTask = new c.DebouncedIdleTask()),
                  (this._observerDisposable = this.register(new x.MutableDisposable())),
                  (this._isPaused = !1),
                  (this._needsFullRefresh = !1),
                  (this._isNextRenderRedrawOnly = !0),
                  (this._needsSelectionRefresh = !1),
                  (this._canvasWidth = 0),
                  (this._canvasHeight = 0),
                  (this._selectionState = { start: void 0, end: void 0, columnSelectMode: !1 }),
                  (this._onDimensionsChange = this.register(new f.EventEmitter())),
                  (this.onDimensionsChange = this._onDimensionsChange.event),
                  (this._onRenderedViewportChange = this.register(new f.EventEmitter())),
                  (this.onRenderedViewportChange = this._onRenderedViewportChange.event),
                  (this._onRender = this.register(new f.EventEmitter())),
                  (this.onRender = this._onRender.event),
                  (this._onRefreshRequest = this.register(new f.EventEmitter())),
                  (this.onRefreshRequest = this._onRefreshRequest.event),
                  (this._renderDebouncer = new a.RenderDebouncer(
                    (L, M) => this._renderRows(L, M),
                    p
                  )),
                  this.register(this._renderDebouncer),
                  this.register(p.onDprChange(() => this.handleDevicePixelRatioChange())),
                  this.register(b.onResize(() => this._fullRefresh())),
                  this.register(b.buffers.onBufferActivate(() => this._renderer.value?.clear())),
                  this.register(d.onOptionChange(() => this._handleOptionsChanged())),
                  this.register(
                    this._charSizeService.onCharSizeChange(() => this.handleCharSizeChanged())
                  ),
                  this.register(_.onDecorationRegistered(() => this._fullRefresh())),
                  this.register(_.onDecorationRemoved(() => this._fullRefresh())),
                  this.register(
                    d.onMultipleOptionChange(
                      [
                        'customGlyphs',
                        'drawBoldTextInBrightColors',
                        'letterSpacing',
                        'lineHeight',
                        'fontFamily',
                        'fontSize',
                        'fontWeight',
                        'fontWeightBold',
                        'minimumContrastRatio',
                        'rescaleOverlappingGlyphs'
                      ],
                      () => {
                        ;(this.clear(), this.handleResize(b.cols, b.rows), this._fullRefresh())
                      }
                    )
                  ),
                  this.register(
                    d.onMultipleOptionChange(['cursorBlink', 'cursorStyle'], () =>
                      this.refreshRows(b.buffer.y, b.buffer.y, !0)
                    )
                  ),
                  this.register(S.onChangeColors(() => this._fullRefresh())),
                  this._registerIntersectionObserver(p.window, r),
                  this.register(p.onWindowChange((L) => this._registerIntersectionObserver(L, r))))
              }
              _registerIntersectionObserver(s, r) {
                if ('IntersectionObserver' in s) {
                  const d = new s.IntersectionObserver(
                    (v) => this._handleIntersectionChange(v[v.length - 1]),
                    { threshold: 0 }
                  )
                  ;(d.observe(r),
                    (this._observerDisposable.value = (0, x.toDisposable)(() => d.disconnect())))
                }
              }
              _handleIntersectionChange(s) {
                ;((this._isPaused =
                  s.isIntersecting === void 0 ? s.intersectionRatio === 0 : !s.isIntersecting),
                  this._isPaused ||
                    this._charSizeService.hasValidSize ||
                    this._charSizeService.measure(),
                  !this._isPaused &&
                    this._needsFullRefresh &&
                    (this._pausedResizeTask.flush(),
                    this.refreshRows(0, this._rowCount - 1),
                    (this._needsFullRefresh = !1)))
              }
              refreshRows(s, r, d = !1) {
                this._isPaused
                  ? (this._needsFullRefresh = !0)
                  : (d || (this._isNextRenderRedrawOnly = !1),
                    this._renderDebouncer.refresh(s, r, this._rowCount))
              }
              _renderRows(s, r) {
                this._renderer.value &&
                  ((s = Math.min(s, this._rowCount - 1)),
                  (r = Math.min(r, this._rowCount - 1)),
                  this._renderer.value.renderRows(s, r),
                  this._needsSelectionRefresh &&
                    (this._renderer.value.handleSelectionChanged(
                      this._selectionState.start,
                      this._selectionState.end,
                      this._selectionState.columnSelectMode
                    ),
                    (this._needsSelectionRefresh = !1)),
                  this._isNextRenderRedrawOnly ||
                    this._onRenderedViewportChange.fire({ start: s, end: r }),
                  this._onRender.fire({ start: s, end: r }),
                  (this._isNextRenderRedrawOnly = !0))
              }
              resize(s, r) {
                ;((this._rowCount = r), this._fireOnCanvasResize())
              }
              _handleOptionsChanged() {
                this._renderer.value &&
                  (this.refreshRows(0, this._rowCount - 1), this._fireOnCanvasResize())
              }
              _fireOnCanvasResize() {
                this._renderer.value &&
                  ((this._renderer.value.dimensions.css.canvas.width === this._canvasWidth &&
                    this._renderer.value.dimensions.css.canvas.height === this._canvasHeight) ||
                    this._onDimensionsChange.fire(this._renderer.value.dimensions))
              }
              hasRenderer() {
                return !!this._renderer.value
              }
              setRenderer(s) {
                ;((this._renderer.value = s),
                  this._renderer.value &&
                    (this._renderer.value.onRequestRedraw((r) =>
                      this.refreshRows(r.start, r.end, !0)
                    ),
                    (this._needsSelectionRefresh = !0),
                    this._fullRefresh()))
              }
              addRefreshCallback(s) {
                return this._renderDebouncer.addRefreshCallback(s)
              }
              _fullRefresh() {
                this._isPaused
                  ? (this._needsFullRefresh = !0)
                  : this.refreshRows(0, this._rowCount - 1)
              }
              clearTextureAtlas() {
                this._renderer.value &&
                  (this._renderer.value.clearTextureAtlas?.(), this._fullRefresh())
              }
              handleDevicePixelRatioChange() {
                ;(this._charSizeService.measure(),
                  this._renderer.value &&
                    (this._renderer.value.handleDevicePixelRatioChange(),
                    this.refreshRows(0, this._rowCount - 1)))
              }
              handleResize(s, r) {
                this._renderer.value &&
                  (this._isPaused
                    ? this._pausedResizeTask.set(() => this._renderer.value?.handleResize(s, r))
                    : this._renderer.value.handleResize(s, r),
                  this._fullRefresh())
              }
              handleCharSizeChanged() {
                this._renderer.value?.handleCharSizeChanged()
              }
              handleBlur() {
                this._renderer.value?.handleBlur()
              }
              handleFocus() {
                this._renderer.value?.handleFocus()
              }
              handleSelectionChanged(s, r, d) {
                ;((this._selectionState.start = s),
                  (this._selectionState.end = r),
                  (this._selectionState.columnSelectMode = d),
                  this._renderer.value?.handleSelectionChanged(s, r, d))
              }
              handleCursorMove() {
                this._renderer.value?.handleCursorMove()
              }
              clear() {
                this._renderer.value?.clear()
              }
            })
            i.RenderService = n = l(
              [
                u(2, t.IOptionsService),
                u(3, h.ICharSizeService),
                u(4, t.IDecorationService),
                u(5, t.IBufferService),
                u(6, h.ICoreBrowserService),
                u(7, h.IThemeService)
              ],
              n
            )
          },
          9312: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (p, S, L, M) {
                  var P,
                    j = arguments.length,
                    D = j < 3 ? S : M === null ? (M = Object.getOwnPropertyDescriptor(S, L)) : M
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    D = Reflect.decorate(p, S, L, M)
                  else
                    for (var O = p.length - 1; O >= 0; O--)
                      (P = p[O]) && (D = (j < 3 ? P(D) : j > 3 ? P(S, L, D) : P(S, L)) || D)
                  return (j > 3 && D && Object.defineProperty(S, L, D), D)
                },
              u =
                (this && this.__param) ||
                function (p, S) {
                  return function (L, M) {
                    S(L, M, p)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.SelectionService = void 0))
            const a = o(9806),
              h = o(9504),
              f = o(456),
              x = o(4725),
              c = o(8460),
              t = o(844),
              n = o(6114),
              s = o(4841),
              r = o(511),
              d = o(2585),
              v = ' ',
              _ = new RegExp(v, 'g')
            let b = (i.SelectionService = class extends t.Disposable {
              constructor(p, S, L, M, P, j, D, O, $) {
                ;(super(),
                  (this._element = p),
                  (this._screenElement = S),
                  (this._linkifier = L),
                  (this._bufferService = M),
                  (this._coreService = P),
                  (this._mouseService = j),
                  (this._optionsService = D),
                  (this._renderService = O),
                  (this._coreBrowserService = $),
                  (this._dragScrollAmount = 0),
                  (this._enabled = !0),
                  (this._workCell = new r.CellData()),
                  (this._mouseDownTimeStamp = 0),
                  (this._oldHasSelection = !1),
                  (this._oldSelectionStart = void 0),
                  (this._oldSelectionEnd = void 0),
                  (this._onLinuxMouseSelection = this.register(new c.EventEmitter())),
                  (this.onLinuxMouseSelection = this._onLinuxMouseSelection.event),
                  (this._onRedrawRequest = this.register(new c.EventEmitter())),
                  (this.onRequestRedraw = this._onRedrawRequest.event),
                  (this._onSelectionChange = this.register(new c.EventEmitter())),
                  (this.onSelectionChange = this._onSelectionChange.event),
                  (this._onRequestScrollLines = this.register(new c.EventEmitter())),
                  (this.onRequestScrollLines = this._onRequestScrollLines.event),
                  (this._mouseMoveListener = (F) => this._handleMouseMove(F)),
                  (this._mouseUpListener = (F) => this._handleMouseUp(F)),
                  this._coreService.onUserInput(() => {
                    this.hasSelection && this.clearSelection()
                  }),
                  (this._trimListener = this._bufferService.buffer.lines.onTrim((F) =>
                    this._handleTrim(F)
                  )),
                  this.register(
                    this._bufferService.buffers.onBufferActivate((F) =>
                      this._handleBufferActivate(F)
                    )
                  ),
                  this.enable(),
                  (this._model = new f.SelectionModel(this._bufferService)),
                  (this._activeSelectionMode = 0),
                  this.register(
                    (0, t.toDisposable)(() => {
                      this._removeMouseDownListeners()
                    })
                  ))
              }
              reset() {
                this.clearSelection()
              }
              disable() {
                ;(this.clearSelection(), (this._enabled = !1))
              }
              enable() {
                this._enabled = !0
              }
              get selectionStart() {
                return this._model.finalSelectionStart
              }
              get selectionEnd() {
                return this._model.finalSelectionEnd
              }
              get hasSelection() {
                const p = this._model.finalSelectionStart,
                  S = this._model.finalSelectionEnd
                return !(!p || !S || (p[0] === S[0] && p[1] === S[1]))
              }
              get selectionText() {
                const p = this._model.finalSelectionStart,
                  S = this._model.finalSelectionEnd
                if (!p || !S) return ''
                const L = this._bufferService.buffer,
                  M = []
                if (this._activeSelectionMode === 3) {
                  if (p[0] === S[0]) return ''
                  const P = p[0] < S[0] ? p[0] : S[0],
                    j = p[0] < S[0] ? S[0] : p[0]
                  for (let D = p[1]; D <= S[1]; D++) {
                    const O = L.translateBufferLineToString(D, !0, P, j)
                    M.push(O)
                  }
                } else {
                  const P = p[1] === S[1] ? S[0] : void 0
                  M.push(L.translateBufferLineToString(p[1], !0, p[0], P))
                  for (let j = p[1] + 1; j <= S[1] - 1; j++) {
                    const D = L.lines.get(j),
                      O = L.translateBufferLineToString(j, !0)
                    D?.isWrapped ? (M[M.length - 1] += O) : M.push(O)
                  }
                  if (p[1] !== S[1]) {
                    const j = L.lines.get(S[1]),
                      D = L.translateBufferLineToString(S[1], !0, 0, S[0])
                    j && j.isWrapped ? (M[M.length - 1] += D) : M.push(D)
                  }
                }
                return M.map((P) => P.replace(_, ' ')).join(
                  n.isWindows
                    ? `\r
`
                    : `
`
                )
              }
              clearSelection() {
                ;(this._model.clearSelection(),
                  this._removeMouseDownListeners(),
                  this.refresh(),
                  this._onSelectionChange.fire())
              }
              refresh(p) {
                ;(this._refreshAnimationFrame ||
                  (this._refreshAnimationFrame =
                    this._coreBrowserService.window.requestAnimationFrame(() => this._refresh())),
                  n.isLinux &&
                    p &&
                    this.selectionText.length &&
                    this._onLinuxMouseSelection.fire(this.selectionText))
              }
              _refresh() {
                ;((this._refreshAnimationFrame = void 0),
                  this._onRedrawRequest.fire({
                    start: this._model.finalSelectionStart,
                    end: this._model.finalSelectionEnd,
                    columnSelectMode: this._activeSelectionMode === 3
                  }))
              }
              _isClickInSelection(p) {
                const S = this._getMouseBufferCoords(p),
                  L = this._model.finalSelectionStart,
                  M = this._model.finalSelectionEnd
                return !!(L && M && S) && this._areCoordsInSelection(S, L, M)
              }
              isCellInSelection(p, S) {
                const L = this._model.finalSelectionStart,
                  M = this._model.finalSelectionEnd
                return !(!L || !M) && this._areCoordsInSelection([p, S], L, M)
              }
              _areCoordsInSelection(p, S, L) {
                return (
                  (p[1] > S[1] && p[1] < L[1]) ||
                  (S[1] === L[1] && p[1] === S[1] && p[0] >= S[0] && p[0] < L[0]) ||
                  (S[1] < L[1] && p[1] === L[1] && p[0] < L[0]) ||
                  (S[1] < L[1] && p[1] === S[1] && p[0] >= S[0])
                )
              }
              _selectWordAtCursor(p, S) {
                const L = this._linkifier.currentLink?.link?.range
                if (L)
                  return (
                    (this._model.selectionStart = [L.start.x - 1, L.start.y - 1]),
                    (this._model.selectionStartLength = (0, s.getRangeLength)(
                      L,
                      this._bufferService.cols
                    )),
                    (this._model.selectionEnd = void 0),
                    !0
                  )
                const M = this._getMouseBufferCoords(p)
                return !!M && (this._selectWordAt(M, S), (this._model.selectionEnd = void 0), !0)
              }
              selectAll() {
                ;((this._model.isSelectAllActive = !0),
                  this.refresh(),
                  this._onSelectionChange.fire())
              }
              selectLines(p, S) {
                ;(this._model.clearSelection(),
                  (p = Math.max(p, 0)),
                  (S = Math.min(S, this._bufferService.buffer.lines.length - 1)),
                  (this._model.selectionStart = [0, p]),
                  (this._model.selectionEnd = [this._bufferService.cols, S]),
                  this.refresh(),
                  this._onSelectionChange.fire())
              }
              _handleTrim(p) {
                this._model.handleTrim(p) && this.refresh()
              }
              _getMouseBufferCoords(p) {
                const S = this._mouseService.getCoords(
                  p,
                  this._screenElement,
                  this._bufferService.cols,
                  this._bufferService.rows,
                  !0
                )
                if (S) return (S[0]--, S[1]--, (S[1] += this._bufferService.buffer.ydisp), S)
              }
              _getMouseEventScrollAmount(p) {
                let S = (0, a.getCoordsRelativeToElement)(
                  this._coreBrowserService.window,
                  p,
                  this._screenElement
                )[1]
                const L = this._renderService.dimensions.css.canvas.height
                return S >= 0 && S <= L
                  ? 0
                  : (S > L && (S -= L),
                    (S = Math.min(Math.max(S, -50), 50)),
                    (S /= 50),
                    S / Math.abs(S) + Math.round(14 * S))
              }
              shouldForceSelection(p) {
                return n.isMac
                  ? p.altKey && this._optionsService.rawOptions.macOptionClickForcesSelection
                  : p.shiftKey
              }
              handleMouseDown(p) {
                if (
                  ((this._mouseDownTimeStamp = p.timeStamp),
                  (p.button !== 2 || !this.hasSelection) && p.button === 0)
                ) {
                  if (!this._enabled) {
                    if (!this.shouldForceSelection(p)) return
                    p.stopPropagation()
                  }
                  ;(p.preventDefault(),
                    (this._dragScrollAmount = 0),
                    this._enabled && p.shiftKey
                      ? this._handleIncrementalClick(p)
                      : p.detail === 1
                        ? this._handleSingleClick(p)
                        : p.detail === 2
                          ? this._handleDoubleClick(p)
                          : p.detail === 3 && this._handleTripleClick(p),
                    this._addMouseDownListeners(),
                    this.refresh(!0))
                }
              }
              _addMouseDownListeners() {
                ;(this._screenElement.ownerDocument &&
                  (this._screenElement.ownerDocument.addEventListener(
                    'mousemove',
                    this._mouseMoveListener
                  ),
                  this._screenElement.ownerDocument.addEventListener(
                    'mouseup',
                    this._mouseUpListener
                  )),
                  (this._dragScrollIntervalTimer = this._coreBrowserService.window.setInterval(
                    () => this._dragScroll(),
                    50
                  )))
              }
              _removeMouseDownListeners() {
                ;(this._screenElement.ownerDocument &&
                  (this._screenElement.ownerDocument.removeEventListener(
                    'mousemove',
                    this._mouseMoveListener
                  ),
                  this._screenElement.ownerDocument.removeEventListener(
                    'mouseup',
                    this._mouseUpListener
                  )),
                  this._coreBrowserService.window.clearInterval(this._dragScrollIntervalTimer),
                  (this._dragScrollIntervalTimer = void 0))
              }
              _handleIncrementalClick(p) {
                this._model.selectionStart &&
                  (this._model.selectionEnd = this._getMouseBufferCoords(p))
              }
              _handleSingleClick(p) {
                if (
                  ((this._model.selectionStartLength = 0),
                  (this._model.isSelectAllActive = !1),
                  (this._activeSelectionMode = this.shouldColumnSelect(p) ? 3 : 0),
                  (this._model.selectionStart = this._getMouseBufferCoords(p)),
                  !this._model.selectionStart)
                )
                  return
                this._model.selectionEnd = void 0
                const S = this._bufferService.buffer.lines.get(this._model.selectionStart[1])
                S &&
                  S.length !== this._model.selectionStart[0] &&
                  S.hasWidth(this._model.selectionStart[0]) === 0 &&
                  this._model.selectionStart[0]++
              }
              _handleDoubleClick(p) {
                this._selectWordAtCursor(p, !0) && (this._activeSelectionMode = 1)
              }
              _handleTripleClick(p) {
                const S = this._getMouseBufferCoords(p)
                S && ((this._activeSelectionMode = 2), this._selectLineAt(S[1]))
              }
              shouldColumnSelect(p) {
                return (
                  p.altKey &&
                  !(n.isMac && this._optionsService.rawOptions.macOptionClickForcesSelection)
                )
              }
              _handleMouseMove(p) {
                if ((p.stopImmediatePropagation(), !this._model.selectionStart)) return
                const S = this._model.selectionEnd
                  ? [this._model.selectionEnd[0], this._model.selectionEnd[1]]
                  : null
                if (
                  ((this._model.selectionEnd = this._getMouseBufferCoords(p)),
                  !this._model.selectionEnd)
                )
                  return void this.refresh(!0)
                ;(this._activeSelectionMode === 2
                  ? this._model.selectionEnd[1] < this._model.selectionStart[1]
                    ? (this._model.selectionEnd[0] = 0)
                    : (this._model.selectionEnd[0] = this._bufferService.cols)
                  : this._activeSelectionMode === 1 &&
                    this._selectToWordAt(this._model.selectionEnd),
                  (this._dragScrollAmount = this._getMouseEventScrollAmount(p)),
                  this._activeSelectionMode !== 3 &&
                    (this._dragScrollAmount > 0
                      ? (this._model.selectionEnd[0] = this._bufferService.cols)
                      : this._dragScrollAmount < 0 && (this._model.selectionEnd[0] = 0)))
                const L = this._bufferService.buffer
                if (this._model.selectionEnd[1] < L.lines.length) {
                  const M = L.lines.get(this._model.selectionEnd[1])
                  M &&
                    M.hasWidth(this._model.selectionEnd[0]) === 0 &&
                    this._model.selectionEnd[0] < this._bufferService.cols &&
                    this._model.selectionEnd[0]++
                }
                ;(S &&
                  S[0] === this._model.selectionEnd[0] &&
                  S[1] === this._model.selectionEnd[1]) ||
                  this.refresh(!0)
              }
              _dragScroll() {
                if (
                  this._model.selectionEnd &&
                  this._model.selectionStart &&
                  this._dragScrollAmount
                ) {
                  this._onRequestScrollLines.fire({
                    amount: this._dragScrollAmount,
                    suppressScrollEvent: !1
                  })
                  const p = this._bufferService.buffer
                  ;(this._dragScrollAmount > 0
                    ? (this._activeSelectionMode !== 3 &&
                        (this._model.selectionEnd[0] = this._bufferService.cols),
                      (this._model.selectionEnd[1] = Math.min(
                        p.ydisp + this._bufferService.rows,
                        p.lines.length - 1
                      )))
                    : (this._activeSelectionMode !== 3 && (this._model.selectionEnd[0] = 0),
                      (this._model.selectionEnd[1] = p.ydisp)),
                    this.refresh())
                }
              }
              _handleMouseUp(p) {
                const S = p.timeStamp - this._mouseDownTimeStamp
                if (
                  (this._removeMouseDownListeners(),
                  this.selectionText.length <= 1 &&
                    S < 500 &&
                    p.altKey &&
                    this._optionsService.rawOptions.altClickMovesCursor)
                ) {
                  if (this._bufferService.buffer.ybase === this._bufferService.buffer.ydisp) {
                    const L = this._mouseService.getCoords(
                      p,
                      this._element,
                      this._bufferService.cols,
                      this._bufferService.rows,
                      !1
                    )
                    if (L && L[0] !== void 0 && L[1] !== void 0) {
                      const M = (0, h.moveToCellSequence)(
                        L[0] - 1,
                        L[1] - 1,
                        this._bufferService,
                        this._coreService.decPrivateModes.applicationCursorKeys
                      )
                      this._coreService.triggerDataEvent(M, !0)
                    }
                  }
                } else this._fireEventIfSelectionChanged()
              }
              _fireEventIfSelectionChanged() {
                const p = this._model.finalSelectionStart,
                  S = this._model.finalSelectionEnd,
                  L = !(!p || !S || (p[0] === S[0] && p[1] === S[1]))
                L
                  ? p &&
                    S &&
                    ((this._oldSelectionStart &&
                      this._oldSelectionEnd &&
                      p[0] === this._oldSelectionStart[0] &&
                      p[1] === this._oldSelectionStart[1] &&
                      S[0] === this._oldSelectionEnd[0] &&
                      S[1] === this._oldSelectionEnd[1]) ||
                      this._fireOnSelectionChange(p, S, L))
                  : this._oldHasSelection && this._fireOnSelectionChange(p, S, L)
              }
              _fireOnSelectionChange(p, S, L) {
                ;((this._oldSelectionStart = p),
                  (this._oldSelectionEnd = S),
                  (this._oldHasSelection = L),
                  this._onSelectionChange.fire())
              }
              _handleBufferActivate(p) {
                ;(this.clearSelection(),
                  this._trimListener.dispose(),
                  (this._trimListener = p.activeBuffer.lines.onTrim((S) => this._handleTrim(S))))
              }
              _convertViewportColToCharacterIndex(p, S) {
                let L = S
                for (let M = 0; S >= M; M++) {
                  const P = p.loadCell(M, this._workCell).getChars().length
                  this._workCell.getWidth() === 0 ? L-- : P > 1 && S !== M && (L += P - 1)
                }
                return L
              }
              setSelection(p, S, L) {
                ;(this._model.clearSelection(),
                  this._removeMouseDownListeners(),
                  (this._model.selectionStart = [p, S]),
                  (this._model.selectionStartLength = L),
                  this.refresh(),
                  this._fireEventIfSelectionChanged())
              }
              rightClickSelect(p) {
                this._isClickInSelection(p) ||
                  (this._selectWordAtCursor(p, !1) && this.refresh(!0),
                  this._fireEventIfSelectionChanged())
              }
              _getWordAt(p, S, L = !0, M = !0) {
                if (p[0] >= this._bufferService.cols) return
                const P = this._bufferService.buffer,
                  j = P.lines.get(p[1])
                if (!j) return
                const D = P.translateBufferLineToString(p[1], !1)
                let O = this._convertViewportColToCharacterIndex(j, p[0]),
                  $ = O
                const F = p[0] - O
                let W = 0,
                  C = 0,
                  A = 0,
                  N = 0
                if (D.charAt(O) === ' ') {
                  for (; O > 0 && D.charAt(O - 1) === ' '; ) O--
                  for (; $ < D.length && D.charAt($ + 1) === ' '; ) $++
                } else {
                  let K = p[0],
                    J = p[0]
                  ;(j.getWidth(K) === 0 && (W++, K--), j.getWidth(J) === 2 && (C++, J++))
                  const Q = j.getString(J).length
                  for (
                    Q > 1 && ((N += Q - 1), ($ += Q - 1));
                    K > 0 && O > 0 && !this._isCharWordSeparator(j.loadCell(K - 1, this._workCell));

                  ) {
                    j.loadCell(K - 1, this._workCell)
                    const H = this._workCell.getChars().length
                    ;(this._workCell.getWidth() === 0
                      ? (W++, K--)
                      : H > 1 && ((A += H - 1), (O -= H - 1)),
                      O--,
                      K--)
                  }
                  for (
                    ;
                    J < j.length &&
                    $ + 1 < D.length &&
                    !this._isCharWordSeparator(j.loadCell(J + 1, this._workCell));

                  ) {
                    j.loadCell(J + 1, this._workCell)
                    const H = this._workCell.getChars().length
                    ;(this._workCell.getWidth() === 2
                      ? (C++, J++)
                      : H > 1 && ((N += H - 1), ($ += H - 1)),
                      $++,
                      J++)
                  }
                }
                $++
                let B = O + F - W + A,
                  z = Math.min(this._bufferService.cols, $ - O + W + C - A - N)
                if (S || D.slice(O, $).trim() !== '') {
                  if (L && B === 0 && j.getCodePoint(0) !== 32) {
                    const K = P.lines.get(p[1] - 1)
                    if (K && j.isWrapped && K.getCodePoint(this._bufferService.cols - 1) !== 32) {
                      const J = this._getWordAt(
                        [this._bufferService.cols - 1, p[1] - 1],
                        !1,
                        !0,
                        !1
                      )
                      if (J) {
                        const Q = this._bufferService.cols - J.start
                        ;((B -= Q), (z += Q))
                      }
                    }
                  }
                  if (
                    M &&
                    B + z === this._bufferService.cols &&
                    j.getCodePoint(this._bufferService.cols - 1) !== 32
                  ) {
                    const K = P.lines.get(p[1] + 1)
                    if (K?.isWrapped && K.getCodePoint(0) !== 32) {
                      const J = this._getWordAt([0, p[1] + 1], !1, !1, !0)
                      J && (z += J.length)
                    }
                  }
                  return { start: B, length: z }
                }
              }
              _selectWordAt(p, S) {
                const L = this._getWordAt(p, S)
                if (L) {
                  for (; L.start < 0; ) ((L.start += this._bufferService.cols), p[1]--)
                  ;((this._model.selectionStart = [L.start, p[1]]),
                    (this._model.selectionStartLength = L.length))
                }
              }
              _selectToWordAt(p) {
                const S = this._getWordAt(p, !0)
                if (S) {
                  let L = p[1]
                  for (; S.start < 0; ) ((S.start += this._bufferService.cols), L--)
                  if (!this._model.areSelectionValuesReversed())
                    for (; S.start + S.length > this._bufferService.cols; )
                      ((S.length -= this._bufferService.cols), L++)
                  this._model.selectionEnd = [
                    this._model.areSelectionValuesReversed() ? S.start : S.start + S.length,
                    L
                  ]
                }
              }
              _isCharWordSeparator(p) {
                return (
                  p.getWidth() !== 0 &&
                  this._optionsService.rawOptions.wordSeparator.indexOf(p.getChars()) >= 0
                )
              }
              _selectLineAt(p) {
                const S = this._bufferService.buffer.getWrappedRangeForLine(p),
                  L = {
                    start: { x: 0, y: S.first },
                    end: { x: this._bufferService.cols - 1, y: S.last }
                  }
                ;((this._model.selectionStart = [0, S.first]),
                  (this._model.selectionEnd = void 0),
                  (this._model.selectionStartLength = (0, s.getRangeLength)(
                    L,
                    this._bufferService.cols
                  )))
              }
            })
            i.SelectionService = b = l(
              [
                u(3, d.IBufferService),
                u(4, d.ICoreService),
                u(5, x.IMouseService),
                u(6, d.IOptionsService),
                u(7, x.IRenderService),
                u(8, x.ICoreBrowserService)
              ],
              b
            )
          },
          4725: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.ILinkProviderService =
                i.IThemeService =
                i.ICharacterJoinerService =
                i.ISelectionService =
                i.IRenderService =
                i.IMouseService =
                i.ICoreBrowserService =
                i.ICharSizeService =
                  void 0))
            const l = o(8343)
            ;((i.ICharSizeService = (0, l.createDecorator)('CharSizeService')),
              (i.ICoreBrowserService = (0, l.createDecorator)('CoreBrowserService')),
              (i.IMouseService = (0, l.createDecorator)('MouseService')),
              (i.IRenderService = (0, l.createDecorator)('RenderService')),
              (i.ISelectionService = (0, l.createDecorator)('SelectionService')),
              (i.ICharacterJoinerService = (0, l.createDecorator)('CharacterJoinerService')),
              (i.IThemeService = (0, l.createDecorator)('ThemeService')),
              (i.ILinkProviderService = (0, l.createDecorator)('LinkProviderService')))
          },
          6731: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (b, p, S, L) {
                  var M,
                    P = arguments.length,
                    j = P < 3 ? p : L === null ? (L = Object.getOwnPropertyDescriptor(p, S)) : L
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    j = Reflect.decorate(b, p, S, L)
                  else
                    for (var D = b.length - 1; D >= 0; D--)
                      (M = b[D]) && (j = (P < 3 ? M(j) : P > 3 ? M(p, S, j) : M(p, S)) || j)
                  return (P > 3 && j && Object.defineProperty(p, S, j), j)
                },
              u =
                (this && this.__param) ||
                function (b, p) {
                  return function (S, L) {
                    p(S, L, b)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.ThemeService = i.DEFAULT_ANSI_COLORS = void 0))
            const a = o(7239),
              h = o(8055),
              f = o(8460),
              x = o(844),
              c = o(2585),
              t = h.css.toColor('#ffffff'),
              n = h.css.toColor('#000000'),
              s = h.css.toColor('#ffffff'),
              r = h.css.toColor('#000000'),
              d = { css: 'rgba(255, 255, 255, 0.3)', rgba: 4294967117 }
            i.DEFAULT_ANSI_COLORS = Object.freeze(
              (() => {
                const b = [
                    h.css.toColor('#2e3436'),
                    h.css.toColor('#cc0000'),
                    h.css.toColor('#4e9a06'),
                    h.css.toColor('#c4a000'),
                    h.css.toColor('#3465a4'),
                    h.css.toColor('#75507b'),
                    h.css.toColor('#06989a'),
                    h.css.toColor('#d3d7cf'),
                    h.css.toColor('#555753'),
                    h.css.toColor('#ef2929'),
                    h.css.toColor('#8ae234'),
                    h.css.toColor('#fce94f'),
                    h.css.toColor('#729fcf'),
                    h.css.toColor('#ad7fa8'),
                    h.css.toColor('#34e2e2'),
                    h.css.toColor('#eeeeec')
                  ],
                  p = [0, 95, 135, 175, 215, 255]
                for (let S = 0; S < 216; S++) {
                  const L = p[(S / 36) % 6 | 0],
                    M = p[(S / 6) % 6 | 0],
                    P = p[S % 6]
                  b.push({ css: h.channels.toCss(L, M, P), rgba: h.channels.toRgba(L, M, P) })
                }
                for (let S = 0; S < 24; S++) {
                  const L = 8 + 10 * S
                  b.push({ css: h.channels.toCss(L, L, L), rgba: h.channels.toRgba(L, L, L) })
                }
                return b
              })()
            )
            let v = (i.ThemeService = class extends x.Disposable {
              get colors() {
                return this._colors
              }
              constructor(b) {
                ;(super(),
                  (this._optionsService = b),
                  (this._contrastCache = new a.ColorContrastCache()),
                  (this._halfContrastCache = new a.ColorContrastCache()),
                  (this._onChangeColors = this.register(new f.EventEmitter())),
                  (this.onChangeColors = this._onChangeColors.event),
                  (this._colors = {
                    foreground: t,
                    background: n,
                    cursor: s,
                    cursorAccent: r,
                    selectionForeground: void 0,
                    selectionBackgroundTransparent: d,
                    selectionBackgroundOpaque: h.color.blend(n, d),
                    selectionInactiveBackgroundTransparent: d,
                    selectionInactiveBackgroundOpaque: h.color.blend(n, d),
                    ansi: i.DEFAULT_ANSI_COLORS.slice(),
                    contrastCache: this._contrastCache,
                    halfContrastCache: this._halfContrastCache
                  }),
                  this._updateRestoreColors(),
                  this._setTheme(this._optionsService.rawOptions.theme),
                  this.register(
                    this._optionsService.onSpecificOptionChange('minimumContrastRatio', () =>
                      this._contrastCache.clear()
                    )
                  ),
                  this.register(
                    this._optionsService.onSpecificOptionChange('theme', () =>
                      this._setTheme(this._optionsService.rawOptions.theme)
                    )
                  ))
              }
              _setTheme(b = {}) {
                const p = this._colors
                if (
                  ((p.foreground = _(b.foreground, t)),
                  (p.background = _(b.background, n)),
                  (p.cursor = _(b.cursor, s)),
                  (p.cursorAccent = _(b.cursorAccent, r)),
                  (p.selectionBackgroundTransparent = _(b.selectionBackground, d)),
                  (p.selectionBackgroundOpaque = h.color.blend(
                    p.background,
                    p.selectionBackgroundTransparent
                  )),
                  (p.selectionInactiveBackgroundTransparent = _(
                    b.selectionInactiveBackground,
                    p.selectionBackgroundTransparent
                  )),
                  (p.selectionInactiveBackgroundOpaque = h.color.blend(
                    p.background,
                    p.selectionInactiveBackgroundTransparent
                  )),
                  (p.selectionForeground = b.selectionForeground
                    ? _(b.selectionForeground, h.NULL_COLOR)
                    : void 0),
                  p.selectionForeground === h.NULL_COLOR && (p.selectionForeground = void 0),
                  h.color.isOpaque(p.selectionBackgroundTransparent) &&
                    (p.selectionBackgroundTransparent = h.color.opacity(
                      p.selectionBackgroundTransparent,
                      0.3
                    )),
                  h.color.isOpaque(p.selectionInactiveBackgroundTransparent) &&
                    (p.selectionInactiveBackgroundTransparent = h.color.opacity(
                      p.selectionInactiveBackgroundTransparent,
                      0.3
                    )),
                  (p.ansi = i.DEFAULT_ANSI_COLORS.slice()),
                  (p.ansi[0] = _(b.black, i.DEFAULT_ANSI_COLORS[0])),
                  (p.ansi[1] = _(b.red, i.DEFAULT_ANSI_COLORS[1])),
                  (p.ansi[2] = _(b.green, i.DEFAULT_ANSI_COLORS[2])),
                  (p.ansi[3] = _(b.yellow, i.DEFAULT_ANSI_COLORS[3])),
                  (p.ansi[4] = _(b.blue, i.DEFAULT_ANSI_COLORS[4])),
                  (p.ansi[5] = _(b.magenta, i.DEFAULT_ANSI_COLORS[5])),
                  (p.ansi[6] = _(b.cyan, i.DEFAULT_ANSI_COLORS[6])),
                  (p.ansi[7] = _(b.white, i.DEFAULT_ANSI_COLORS[7])),
                  (p.ansi[8] = _(b.brightBlack, i.DEFAULT_ANSI_COLORS[8])),
                  (p.ansi[9] = _(b.brightRed, i.DEFAULT_ANSI_COLORS[9])),
                  (p.ansi[10] = _(b.brightGreen, i.DEFAULT_ANSI_COLORS[10])),
                  (p.ansi[11] = _(b.brightYellow, i.DEFAULT_ANSI_COLORS[11])),
                  (p.ansi[12] = _(b.brightBlue, i.DEFAULT_ANSI_COLORS[12])),
                  (p.ansi[13] = _(b.brightMagenta, i.DEFAULT_ANSI_COLORS[13])),
                  (p.ansi[14] = _(b.brightCyan, i.DEFAULT_ANSI_COLORS[14])),
                  (p.ansi[15] = _(b.brightWhite, i.DEFAULT_ANSI_COLORS[15])),
                  b.extendedAnsi)
                ) {
                  const S = Math.min(p.ansi.length - 16, b.extendedAnsi.length)
                  for (let L = 0; L < S; L++)
                    p.ansi[L + 16] = _(b.extendedAnsi[L], i.DEFAULT_ANSI_COLORS[L + 16])
                }
                ;(this._contrastCache.clear(),
                  this._halfContrastCache.clear(),
                  this._updateRestoreColors(),
                  this._onChangeColors.fire(this.colors))
              }
              restoreColor(b) {
                ;(this._restoreColor(b), this._onChangeColors.fire(this.colors))
              }
              _restoreColor(b) {
                if (b !== void 0)
                  switch (b) {
                    case 256:
                      this._colors.foreground = this._restoreColors.foreground
                      break
                    case 257:
                      this._colors.background = this._restoreColors.background
                      break
                    case 258:
                      this._colors.cursor = this._restoreColors.cursor
                      break
                    default:
                      this._colors.ansi[b] = this._restoreColors.ansi[b]
                  }
                else
                  for (let p = 0; p < this._restoreColors.ansi.length; ++p)
                    this._colors.ansi[p] = this._restoreColors.ansi[p]
              }
              modifyColors(b) {
                ;(b(this._colors), this._onChangeColors.fire(this.colors))
              }
              _updateRestoreColors() {
                this._restoreColors = {
                  foreground: this._colors.foreground,
                  background: this._colors.background,
                  cursor: this._colors.cursor,
                  ansi: this._colors.ansi.slice()
                }
              }
            })
            function _(b, p) {
              if (b !== void 0)
                try {
                  return h.css.toColor(b)
                } catch {}
              return p
            }
            i.ThemeService = v = l([u(0, c.IOptionsService)], v)
          },
          6349: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.CircularList = void 0))
            const l = o(8460),
              u = o(844)
            class a extends u.Disposable {
              constructor(f) {
                ;(super(),
                  (this._maxLength = f),
                  (this.onDeleteEmitter = this.register(new l.EventEmitter())),
                  (this.onDelete = this.onDeleteEmitter.event),
                  (this.onInsertEmitter = this.register(new l.EventEmitter())),
                  (this.onInsert = this.onInsertEmitter.event),
                  (this.onTrimEmitter = this.register(new l.EventEmitter())),
                  (this.onTrim = this.onTrimEmitter.event),
                  (this._array = new Array(this._maxLength)),
                  (this._startIndex = 0),
                  (this._length = 0))
              }
              get maxLength() {
                return this._maxLength
              }
              set maxLength(f) {
                if (this._maxLength === f) return
                const x = new Array(f)
                for (let c = 0; c < Math.min(f, this.length); c++)
                  x[c] = this._array[this._getCyclicIndex(c)]
                ;((this._array = x), (this._maxLength = f), (this._startIndex = 0))
              }
              get length() {
                return this._length
              }
              set length(f) {
                if (f > this._length) for (let x = this._length; x < f; x++) this._array[x] = void 0
                this._length = f
              }
              get(f) {
                return this._array[this._getCyclicIndex(f)]
              }
              set(f, x) {
                this._array[this._getCyclicIndex(f)] = x
              }
              push(f) {
                ;((this._array[this._getCyclicIndex(this._length)] = f),
                  this._length === this._maxLength
                    ? ((this._startIndex = ++this._startIndex % this._maxLength),
                      this.onTrimEmitter.fire(1))
                    : this._length++)
              }
              recycle() {
                if (this._length !== this._maxLength)
                  throw new Error('Can only recycle when the buffer is full')
                return (
                  (this._startIndex = ++this._startIndex % this._maxLength),
                  this.onTrimEmitter.fire(1),
                  this._array[this._getCyclicIndex(this._length - 1)]
                )
              }
              get isFull() {
                return this._length === this._maxLength
              }
              pop() {
                return this._array[this._getCyclicIndex(this._length-- - 1)]
              }
              splice(f, x, ...c) {
                if (x) {
                  for (let t = f; t < this._length - x; t++)
                    this._array[this._getCyclicIndex(t)] = this._array[this._getCyclicIndex(t + x)]
                  ;((this._length -= x), this.onDeleteEmitter.fire({ index: f, amount: x }))
                }
                for (let t = this._length - 1; t >= f; t--)
                  this._array[this._getCyclicIndex(t + c.length)] =
                    this._array[this._getCyclicIndex(t)]
                for (let t = 0; t < c.length; t++) this._array[this._getCyclicIndex(f + t)] = c[t]
                if (
                  (c.length && this.onInsertEmitter.fire({ index: f, amount: c.length }),
                  this._length + c.length > this._maxLength)
                ) {
                  const t = this._length + c.length - this._maxLength
                  ;((this._startIndex += t),
                    (this._length = this._maxLength),
                    this.onTrimEmitter.fire(t))
                } else this._length += c.length
              }
              trimStart(f) {
                ;(f > this._length && (f = this._length),
                  (this._startIndex += f),
                  (this._length -= f),
                  this.onTrimEmitter.fire(f))
              }
              shiftElements(f, x, c) {
                if (!(x <= 0)) {
                  if (f < 0 || f >= this._length) throw new Error('start argument out of range')
                  if (f + c < 0) throw new Error('Cannot shift elements in list beyond index 0')
                  if (c > 0) {
                    for (let n = x - 1; n >= 0; n--) this.set(f + n + c, this.get(f + n))
                    const t = f + x + c - this._length
                    if (t > 0)
                      for (this._length += t; this._length > this._maxLength; )
                        (this._length--, this._startIndex++, this.onTrimEmitter.fire(1))
                  } else for (let t = 0; t < x; t++) this.set(f + t + c, this.get(f + t))
                }
              }
              _getCyclicIndex(f) {
                return (this._startIndex + f) % this._maxLength
              }
            }
            i.CircularList = a
          },
          1439: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.clone = void 0),
              (i.clone = function o(l, u = 5) {
                if (typeof l != 'object') return l
                const a = Array.isArray(l) ? [] : {}
                for (const h in l) a[h] = u <= 1 ? l[h] : l[h] && o(l[h], u - 1)
                return a
              }))
          },
          8055: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.contrastRatio =
                i.toPaddedHex =
                i.rgba =
                i.rgb =
                i.css =
                i.color =
                i.channels =
                i.NULL_COLOR =
                  void 0))
            let o = 0,
              l = 0,
              u = 0,
              a = 0
            var h, f, x, c, t
            function n(r) {
              const d = r.toString(16)
              return d.length < 2 ? '0' + d : d
            }
            function s(r, d) {
              return r < d ? (d + 0.05) / (r + 0.05) : (r + 0.05) / (d + 0.05)
            }
            ;((i.NULL_COLOR = { css: '#00000000', rgba: 0 }),
              (function (r) {
                ;((r.toCss = function (d, v, _, b) {
                  return b !== void 0 ? `#${n(d)}${n(v)}${n(_)}${n(b)}` : `#${n(d)}${n(v)}${n(_)}`
                }),
                  (r.toRgba = function (d, v, _, b = 255) {
                    return ((d << 24) | (v << 16) | (_ << 8) | b) >>> 0
                  }),
                  (r.toColor = function (d, v, _, b) {
                    return { css: r.toCss(d, v, _, b), rgba: r.toRgba(d, v, _, b) }
                  }))
              })(h || (i.channels = h = {})),
              (function (r) {
                function d(v, _) {
                  return (
                    (a = Math.round(255 * _)),
                    ([o, l, u] = t.toChannels(v.rgba)),
                    { css: h.toCss(o, l, u, a), rgba: h.toRgba(o, l, u, a) }
                  )
                }
                ;((r.blend = function (v, _) {
                  if (((a = (255 & _.rgba) / 255), a === 1)) return { css: _.css, rgba: _.rgba }
                  const b = (_.rgba >> 24) & 255,
                    p = (_.rgba >> 16) & 255,
                    S = (_.rgba >> 8) & 255,
                    L = (v.rgba >> 24) & 255,
                    M = (v.rgba >> 16) & 255,
                    P = (v.rgba >> 8) & 255
                  return (
                    (o = L + Math.round((b - L) * a)),
                    (l = M + Math.round((p - M) * a)),
                    (u = P + Math.round((S - P) * a)),
                    { css: h.toCss(o, l, u), rgba: h.toRgba(o, l, u) }
                  )
                }),
                  (r.isOpaque = function (v) {
                    return (255 & v.rgba) == 255
                  }),
                  (r.ensureContrastRatio = function (v, _, b) {
                    const p = t.ensureContrastRatio(v.rgba, _.rgba, b)
                    if (p) return h.toColor((p >> 24) & 255, (p >> 16) & 255, (p >> 8) & 255)
                  }),
                  (r.opaque = function (v) {
                    const _ = (255 | v.rgba) >>> 0
                    return (([o, l, u] = t.toChannels(_)), { css: h.toCss(o, l, u), rgba: _ })
                  }),
                  (r.opacity = d),
                  (r.multiplyOpacity = function (v, _) {
                    return ((a = 255 & v.rgba), d(v, (a * _) / 255))
                  }),
                  (r.toColorRGB = function (v) {
                    return [(v.rgba >> 24) & 255, (v.rgba >> 16) & 255, (v.rgba >> 8) & 255]
                  }))
              })(f || (i.color = f = {})),
              (function (r) {
                let d, v
                try {
                  const _ = document.createElement('canvas')
                  ;((_.width = 1), (_.height = 1))
                  const b = _.getContext('2d', { willReadFrequently: !0 })
                  b &&
                    ((d = b),
                    (d.globalCompositeOperation = 'copy'),
                    (v = d.createLinearGradient(0, 0, 1, 1)))
                } catch {}
                r.toColor = function (_) {
                  if (_.match(/#[\da-f]{3,8}/i))
                    switch (_.length) {
                      case 4:
                        return (
                          (o = parseInt(_.slice(1, 2).repeat(2), 16)),
                          (l = parseInt(_.slice(2, 3).repeat(2), 16)),
                          (u = parseInt(_.slice(3, 4).repeat(2), 16)),
                          h.toColor(o, l, u)
                        )
                      case 5:
                        return (
                          (o = parseInt(_.slice(1, 2).repeat(2), 16)),
                          (l = parseInt(_.slice(2, 3).repeat(2), 16)),
                          (u = parseInt(_.slice(3, 4).repeat(2), 16)),
                          (a = parseInt(_.slice(4, 5).repeat(2), 16)),
                          h.toColor(o, l, u, a)
                        )
                      case 7:
                        return { css: _, rgba: ((parseInt(_.slice(1), 16) << 8) | 255) >>> 0 }
                      case 9:
                        return { css: _, rgba: parseInt(_.slice(1), 16) >>> 0 }
                    }
                  const b = _.match(
                    /rgba?\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(,\s*(0|1|\d?\.(\d+))\s*)?\)/
                  )
                  if (b)
                    return (
                      (o = parseInt(b[1])),
                      (l = parseInt(b[2])),
                      (u = parseInt(b[3])),
                      (a = Math.round(255 * (b[5] === void 0 ? 1 : parseFloat(b[5])))),
                      h.toColor(o, l, u, a)
                    )
                  if (!d || !v) throw new Error('css.toColor: Unsupported css format')
                  if (((d.fillStyle = v), (d.fillStyle = _), typeof d.fillStyle != 'string'))
                    throw new Error('css.toColor: Unsupported css format')
                  if (
                    (d.fillRect(0, 0, 1, 1),
                    ([o, l, u, a] = d.getImageData(0, 0, 1, 1).data),
                    a !== 255)
                  )
                    throw new Error('css.toColor: Unsupported css format')
                  return { rgba: h.toRgba(o, l, u, a), css: _ }
                }
              })(x || (i.css = x = {})),
              (function (r) {
                function d(v, _, b) {
                  const p = v / 255,
                    S = _ / 255,
                    L = b / 255
                  return (
                    0.2126 * (p <= 0.03928 ? p / 12.92 : Math.pow((p + 0.055) / 1.055, 2.4)) +
                    0.7152 * (S <= 0.03928 ? S / 12.92 : Math.pow((S + 0.055) / 1.055, 2.4)) +
                    0.0722 * (L <= 0.03928 ? L / 12.92 : Math.pow((L + 0.055) / 1.055, 2.4))
                  )
                }
                ;((r.relativeLuminance = function (v) {
                  return d((v >> 16) & 255, (v >> 8) & 255, 255 & v)
                }),
                  (r.relativeLuminance2 = d))
              })(c || (i.rgb = c = {})),
              (function (r) {
                function d(_, b, p) {
                  const S = (_ >> 24) & 255,
                    L = (_ >> 16) & 255,
                    M = (_ >> 8) & 255
                  let P = (b >> 24) & 255,
                    j = (b >> 16) & 255,
                    D = (b >> 8) & 255,
                    O = s(c.relativeLuminance2(P, j, D), c.relativeLuminance2(S, L, M))
                  for (; O < p && (P > 0 || j > 0 || D > 0); )
                    ((P -= Math.max(0, Math.ceil(0.1 * P))),
                      (j -= Math.max(0, Math.ceil(0.1 * j))),
                      (D -= Math.max(0, Math.ceil(0.1 * D))),
                      (O = s(c.relativeLuminance2(P, j, D), c.relativeLuminance2(S, L, M))))
                  return ((P << 24) | (j << 16) | (D << 8) | 255) >>> 0
                }
                function v(_, b, p) {
                  const S = (_ >> 24) & 255,
                    L = (_ >> 16) & 255,
                    M = (_ >> 8) & 255
                  let P = (b >> 24) & 255,
                    j = (b >> 16) & 255,
                    D = (b >> 8) & 255,
                    O = s(c.relativeLuminance2(P, j, D), c.relativeLuminance2(S, L, M))
                  for (; O < p && (P < 255 || j < 255 || D < 255); )
                    ((P = Math.min(255, P + Math.ceil(0.1 * (255 - P)))),
                      (j = Math.min(255, j + Math.ceil(0.1 * (255 - j)))),
                      (D = Math.min(255, D + Math.ceil(0.1 * (255 - D)))),
                      (O = s(c.relativeLuminance2(P, j, D), c.relativeLuminance2(S, L, M))))
                  return ((P << 24) | (j << 16) | (D << 8) | 255) >>> 0
                }
                ;((r.blend = function (_, b) {
                  if (((a = (255 & b) / 255), a === 1)) return b
                  const p = (b >> 24) & 255,
                    S = (b >> 16) & 255,
                    L = (b >> 8) & 255,
                    M = (_ >> 24) & 255,
                    P = (_ >> 16) & 255,
                    j = (_ >> 8) & 255
                  return (
                    (o = M + Math.round((p - M) * a)),
                    (l = P + Math.round((S - P) * a)),
                    (u = j + Math.round((L - j) * a)),
                    h.toRgba(o, l, u)
                  )
                }),
                  (r.ensureContrastRatio = function (_, b, p) {
                    const S = c.relativeLuminance(_ >> 8),
                      L = c.relativeLuminance(b >> 8)
                    if (s(S, L) < p) {
                      if (L < S) {
                        const j = d(_, b, p),
                          D = s(S, c.relativeLuminance(j >> 8))
                        if (D < p) {
                          const O = v(_, b, p)
                          return D > s(S, c.relativeLuminance(O >> 8)) ? j : O
                        }
                        return j
                      }
                      const M = v(_, b, p),
                        P = s(S, c.relativeLuminance(M >> 8))
                      if (P < p) {
                        const j = d(_, b, p)
                        return P > s(S, c.relativeLuminance(j >> 8)) ? M : j
                      }
                      return M
                    }
                  }),
                  (r.reduceLuminance = d),
                  (r.increaseLuminance = v),
                  (r.toChannels = function (_) {
                    return [(_ >> 24) & 255, (_ >> 16) & 255, (_ >> 8) & 255, 255 & _]
                  }))
              })(t || (i.rgba = t = {})),
              (i.toPaddedHex = n),
              (i.contrastRatio = s))
          },
          8969: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.CoreTerminal = void 0))
            const l = o(844),
              u = o(2585),
              a = o(4348),
              h = o(7866),
              f = o(744),
              x = o(7302),
              c = o(6975),
              t = o(8460),
              n = o(1753),
              s = o(1480),
              r = o(7994),
              d = o(9282),
              v = o(5435),
              _ = o(5981),
              b = o(2660)
            let p = !1
            class S extends l.Disposable {
              get onScroll() {
                return (
                  this._onScrollApi ||
                    ((this._onScrollApi = this.register(new t.EventEmitter())),
                    this._onScroll.event((M) => {
                      this._onScrollApi?.fire(M.position)
                    })),
                  this._onScrollApi.event
                )
              }
              get cols() {
                return this._bufferService.cols
              }
              get rows() {
                return this._bufferService.rows
              }
              get buffers() {
                return this._bufferService.buffers
              }
              get options() {
                return this.optionsService.options
              }
              set options(M) {
                for (const P in M) this.optionsService.options[P] = M[P]
              }
              constructor(M) {
                ;(super(),
                  (this._windowsWrappingHeuristics = this.register(new l.MutableDisposable())),
                  (this._onBinary = this.register(new t.EventEmitter())),
                  (this.onBinary = this._onBinary.event),
                  (this._onData = this.register(new t.EventEmitter())),
                  (this.onData = this._onData.event),
                  (this._onLineFeed = this.register(new t.EventEmitter())),
                  (this.onLineFeed = this._onLineFeed.event),
                  (this._onResize = this.register(new t.EventEmitter())),
                  (this.onResize = this._onResize.event),
                  (this._onWriteParsed = this.register(new t.EventEmitter())),
                  (this.onWriteParsed = this._onWriteParsed.event),
                  (this._onScroll = this.register(new t.EventEmitter())),
                  (this._instantiationService = new a.InstantiationService()),
                  (this.optionsService = this.register(new x.OptionsService(M))),
                  this._instantiationService.setService(u.IOptionsService, this.optionsService),
                  (this._bufferService = this.register(
                    this._instantiationService.createInstance(f.BufferService)
                  )),
                  this._instantiationService.setService(u.IBufferService, this._bufferService),
                  (this._logService = this.register(
                    this._instantiationService.createInstance(h.LogService)
                  )),
                  this._instantiationService.setService(u.ILogService, this._logService),
                  (this.coreService = this.register(
                    this._instantiationService.createInstance(c.CoreService)
                  )),
                  this._instantiationService.setService(u.ICoreService, this.coreService),
                  (this.coreMouseService = this.register(
                    this._instantiationService.createInstance(n.CoreMouseService)
                  )),
                  this._instantiationService.setService(u.ICoreMouseService, this.coreMouseService),
                  (this.unicodeService = this.register(
                    this._instantiationService.createInstance(s.UnicodeService)
                  )),
                  this._instantiationService.setService(u.IUnicodeService, this.unicodeService),
                  (this._charsetService = this._instantiationService.createInstance(
                    r.CharsetService
                  )),
                  this._instantiationService.setService(u.ICharsetService, this._charsetService),
                  (this._oscLinkService = this._instantiationService.createInstance(
                    b.OscLinkService
                  )),
                  this._instantiationService.setService(u.IOscLinkService, this._oscLinkService),
                  (this._inputHandler = this.register(
                    new v.InputHandler(
                      this._bufferService,
                      this._charsetService,
                      this.coreService,
                      this._logService,
                      this.optionsService,
                      this._oscLinkService,
                      this.coreMouseService,
                      this.unicodeService
                    )
                  )),
                  this.register(
                    (0, t.forwardEvent)(this._inputHandler.onLineFeed, this._onLineFeed)
                  ),
                  this.register(this._inputHandler),
                  this.register((0, t.forwardEvent)(this._bufferService.onResize, this._onResize)),
                  this.register((0, t.forwardEvent)(this.coreService.onData, this._onData)),
                  this.register((0, t.forwardEvent)(this.coreService.onBinary, this._onBinary)),
                  this.register(
                    this.coreService.onRequestScrollToBottom(() => this.scrollToBottom())
                  ),
                  this.register(
                    this.coreService.onUserInput(() => this._writeBuffer.handleUserInput())
                  ),
                  this.register(
                    this.optionsService.onMultipleOptionChange(['windowsMode', 'windowsPty'], () =>
                      this._handleWindowsPtyOptionChange()
                    )
                  ),
                  this.register(
                    this._bufferService.onScroll((P) => {
                      ;(this._onScroll.fire({
                        position: this._bufferService.buffer.ydisp,
                        source: 0
                      }),
                        this._inputHandler.markRangeDirty(
                          this._bufferService.buffer.scrollTop,
                          this._bufferService.buffer.scrollBottom
                        ))
                    })
                  ),
                  this.register(
                    this._inputHandler.onScroll((P) => {
                      ;(this._onScroll.fire({
                        position: this._bufferService.buffer.ydisp,
                        source: 0
                      }),
                        this._inputHandler.markRangeDirty(
                          this._bufferService.buffer.scrollTop,
                          this._bufferService.buffer.scrollBottom
                        ))
                    })
                  ),
                  (this._writeBuffer = this.register(
                    new _.WriteBuffer((P, j) => this._inputHandler.parse(P, j))
                  )),
                  this.register(
                    (0, t.forwardEvent)(this._writeBuffer.onWriteParsed, this._onWriteParsed)
                  ))
              }
              write(M, P) {
                this._writeBuffer.write(M, P)
              }
              writeSync(M, P) {
                ;(this._logService.logLevel <= u.LogLevelEnum.WARN &&
                  !p &&
                  (this._logService.warn('writeSync is unreliable and will be removed soon.'),
                  (p = !0)),
                  this._writeBuffer.writeSync(M, P))
              }
              input(M, P = !0) {
                this.coreService.triggerDataEvent(M, P)
              }
              resize(M, P) {
                isNaN(M) ||
                  isNaN(P) ||
                  ((M = Math.max(M, f.MINIMUM_COLS)),
                  (P = Math.max(P, f.MINIMUM_ROWS)),
                  this._bufferService.resize(M, P))
              }
              scroll(M, P = !1) {
                this._bufferService.scroll(M, P)
              }
              scrollLines(M, P, j) {
                this._bufferService.scrollLines(M, P, j)
              }
              scrollPages(M) {
                this.scrollLines(M * (this.rows - 1))
              }
              scrollToTop() {
                this.scrollLines(-this._bufferService.buffer.ydisp)
              }
              scrollToBottom() {
                this.scrollLines(
                  this._bufferService.buffer.ybase - this._bufferService.buffer.ydisp
                )
              }
              scrollToLine(M) {
                const P = M - this._bufferService.buffer.ydisp
                P !== 0 && this.scrollLines(P)
              }
              registerEscHandler(M, P) {
                return this._inputHandler.registerEscHandler(M, P)
              }
              registerDcsHandler(M, P) {
                return this._inputHandler.registerDcsHandler(M, P)
              }
              registerCsiHandler(M, P) {
                return this._inputHandler.registerCsiHandler(M, P)
              }
              registerOscHandler(M, P) {
                return this._inputHandler.registerOscHandler(M, P)
              }
              _setup() {
                this._handleWindowsPtyOptionChange()
              }
              reset() {
                ;(this._inputHandler.reset(),
                  this._bufferService.reset(),
                  this._charsetService.reset(),
                  this.coreService.reset(),
                  this.coreMouseService.reset())
              }
              _handleWindowsPtyOptionChange() {
                let M = !1
                const P = this.optionsService.rawOptions.windowsPty
                ;(P && P.buildNumber !== void 0 && P.buildNumber !== void 0
                  ? (M = P.backend === 'conpty' && P.buildNumber < 21376)
                  : this.optionsService.rawOptions.windowsMode && (M = !0),
                  M
                    ? this._enableWindowsWrappingHeuristics()
                    : this._windowsWrappingHeuristics.clear())
              }
              _enableWindowsWrappingHeuristics() {
                if (!this._windowsWrappingHeuristics.value) {
                  const M = []
                  ;(M.push(
                    this.onLineFeed(d.updateWindowsModeWrappedState.bind(null, this._bufferService))
                  ),
                    M.push(
                      this.registerCsiHandler(
                        { final: 'H' },
                        () => ((0, d.updateWindowsModeWrappedState)(this._bufferService), !1)
                      )
                    ),
                    (this._windowsWrappingHeuristics.value = (0, l.toDisposable)(() => {
                      for (const P of M) P.dispose()
                    })))
                }
              }
            }
            i.CoreTerminal = S
          },
          8460: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.runAndSubscribe = i.forwardEvent = i.EventEmitter = void 0),
              (i.EventEmitter = class {
                constructor() {
                  ;((this._listeners = []), (this._disposed = !1))
                }
                get event() {
                  return (
                    this._event ||
                      (this._event = (o) => (
                        this._listeners.push(o),
                        {
                          dispose: () => {
                            if (!this._disposed) {
                              for (let l = 0; l < this._listeners.length; l++)
                                if (this._listeners[l] === o)
                                  return void this._listeners.splice(l, 1)
                            }
                          }
                        }
                      )),
                    this._event
                  )
                }
                fire(o, l) {
                  const u = []
                  for (let a = 0; a < this._listeners.length; a++) u.push(this._listeners[a])
                  for (let a = 0; a < u.length; a++) u[a].call(void 0, o, l)
                }
                dispose() {
                  ;(this.clearListeners(), (this._disposed = !0))
                }
                clearListeners() {
                  this._listeners && (this._listeners.length = 0)
                }
              }),
              (i.forwardEvent = function (o, l) {
                return o((u) => l.fire(u))
              }),
              (i.runAndSubscribe = function (o, l) {
                return (l(void 0), o((u) => l(u)))
              }))
          },
          5435: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (W, C, A, N) {
                  var B,
                    z = arguments.length,
                    K = z < 3 ? C : N === null ? (N = Object.getOwnPropertyDescriptor(C, A)) : N
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    K = Reflect.decorate(W, C, A, N)
                  else
                    for (var J = W.length - 1; J >= 0; J--)
                      (B = W[J]) && (K = (z < 3 ? B(K) : z > 3 ? B(C, A, K) : B(C, A)) || K)
                  return (z > 3 && K && Object.defineProperty(C, A, K), K)
                },
              u =
                (this && this.__param) ||
                function (W, C) {
                  return function (A, N) {
                    C(A, N, W)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.InputHandler = i.WindowsOptionsReportType = void 0))
            const a = o(2584),
              h = o(7116),
              f = o(2015),
              x = o(844),
              c = o(482),
              t = o(8437),
              n = o(8460),
              s = o(643),
              r = o(511),
              d = o(3734),
              v = o(2585),
              _ = o(1480),
              b = o(6242),
              p = o(6351),
              S = o(5941),
              L = { '(': 0, ')': 1, '*': 2, '+': 3, '-': 1, '.': 2 },
              M = 131072
            function P(W, C) {
              if (W > 24) return C.setWinLines || !1
              switch (W) {
                case 1:
                  return !!C.restoreWin
                case 2:
                  return !!C.minimizeWin
                case 3:
                  return !!C.setWinPosition
                case 4:
                  return !!C.setWinSizePixels
                case 5:
                  return !!C.raiseWin
                case 6:
                  return !!C.lowerWin
                case 7:
                  return !!C.refreshWin
                case 8:
                  return !!C.setWinSizeChars
                case 9:
                  return !!C.maximizeWin
                case 10:
                  return !!C.fullscreenWin
                case 11:
                  return !!C.getWinState
                case 13:
                  return !!C.getWinPosition
                case 14:
                  return !!C.getWinSizePixels
                case 15:
                  return !!C.getScreenSizePixels
                case 16:
                  return !!C.getCellSizePixels
                case 18:
                  return !!C.getWinSizeChars
                case 19:
                  return !!C.getScreenSizeChars
                case 20:
                  return !!C.getIconTitle
                case 21:
                  return !!C.getWinTitle
                case 22:
                  return !!C.pushTitle
                case 23:
                  return !!C.popTitle
                case 24:
                  return !!C.setWinLines
              }
              return !1
            }
            var j
            ;(function (W) {
              ;((W[(W.GET_WIN_SIZE_PIXELS = 0)] = 'GET_WIN_SIZE_PIXELS'),
                (W[(W.GET_CELL_SIZE_PIXELS = 1)] = 'GET_CELL_SIZE_PIXELS'))
            })(j || (i.WindowsOptionsReportType = j = {}))
            let D = 0
            class O extends x.Disposable {
              getAttrData() {
                return this._curAttrData
              }
              constructor(C, A, N, B, z, K, J, Q, H = new f.EscapeSequenceParser()) {
                ;(super(),
                  (this._bufferService = C),
                  (this._charsetService = A),
                  (this._coreService = N),
                  (this._logService = B),
                  (this._optionsService = z),
                  (this._oscLinkService = K),
                  (this._coreMouseService = J),
                  (this._unicodeService = Q),
                  (this._parser = H),
                  (this._parseBuffer = new Uint32Array(4096)),
                  (this._stringDecoder = new c.StringToUtf32()),
                  (this._utf8Decoder = new c.Utf8ToUtf32()),
                  (this._workCell = new r.CellData()),
                  (this._windowTitle = ''),
                  (this._iconName = ''),
                  (this._windowTitleStack = []),
                  (this._iconNameStack = []),
                  (this._curAttrData = t.DEFAULT_ATTR_DATA.clone()),
                  (this._eraseAttrDataInternal = t.DEFAULT_ATTR_DATA.clone()),
                  (this._onRequestBell = this.register(new n.EventEmitter())),
                  (this.onRequestBell = this._onRequestBell.event),
                  (this._onRequestRefreshRows = this.register(new n.EventEmitter())),
                  (this.onRequestRefreshRows = this._onRequestRefreshRows.event),
                  (this._onRequestReset = this.register(new n.EventEmitter())),
                  (this.onRequestReset = this._onRequestReset.event),
                  (this._onRequestSendFocus = this.register(new n.EventEmitter())),
                  (this.onRequestSendFocus = this._onRequestSendFocus.event),
                  (this._onRequestSyncScrollBar = this.register(new n.EventEmitter())),
                  (this.onRequestSyncScrollBar = this._onRequestSyncScrollBar.event),
                  (this._onRequestWindowsOptionsReport = this.register(new n.EventEmitter())),
                  (this.onRequestWindowsOptionsReport = this._onRequestWindowsOptionsReport.event),
                  (this._onA11yChar = this.register(new n.EventEmitter())),
                  (this.onA11yChar = this._onA11yChar.event),
                  (this._onA11yTab = this.register(new n.EventEmitter())),
                  (this.onA11yTab = this._onA11yTab.event),
                  (this._onCursorMove = this.register(new n.EventEmitter())),
                  (this.onCursorMove = this._onCursorMove.event),
                  (this._onLineFeed = this.register(new n.EventEmitter())),
                  (this.onLineFeed = this._onLineFeed.event),
                  (this._onScroll = this.register(new n.EventEmitter())),
                  (this.onScroll = this._onScroll.event),
                  (this._onTitleChange = this.register(new n.EventEmitter())),
                  (this.onTitleChange = this._onTitleChange.event),
                  (this._onColor = this.register(new n.EventEmitter())),
                  (this.onColor = this._onColor.event),
                  (this._parseStack = {
                    paused: !1,
                    cursorStartX: 0,
                    cursorStartY: 0,
                    decodedLength: 0,
                    position: 0
                  }),
                  (this._specialColors = [256, 257, 258]),
                  this.register(this._parser),
                  (this._dirtyRowTracker = new $(this._bufferService)),
                  (this._activeBuffer = this._bufferService.buffer),
                  this.register(
                    this._bufferService.buffers.onBufferActivate(
                      (E) => (this._activeBuffer = E.activeBuffer)
                    )
                  ),
                  this._parser.setCsiHandlerFallback((E, G) => {
                    this._logService.debug('Unknown CSI code: ', {
                      identifier: this._parser.identToString(E),
                      params: G.toArray()
                    })
                  }),
                  this._parser.setEscHandlerFallback((E) => {
                    this._logService.debug('Unknown ESC code: ', {
                      identifier: this._parser.identToString(E)
                    })
                  }),
                  this._parser.setExecuteHandlerFallback((E) => {
                    this._logService.debug('Unknown EXECUTE code: ', { code: E })
                  }),
                  this._parser.setOscHandlerFallback((E, G, q) => {
                    this._logService.debug('Unknown OSC code: ', {
                      identifier: E,
                      action: G,
                      data: q
                    })
                  }),
                  this._parser.setDcsHandlerFallback((E, G, q) => {
                    ;(G === 'HOOK' && (q = q.toArray()),
                      this._logService.debug('Unknown DCS code: ', {
                        identifier: this._parser.identToString(E),
                        action: G,
                        payload: q
                      }))
                  }),
                  this._parser.setPrintHandler((E, G, q) => this.print(E, G, q)),
                  this._parser.registerCsiHandler({ final: '@' }, (E) => this.insertChars(E)),
                  this._parser.registerCsiHandler({ intermediates: ' ', final: '@' }, (E) =>
                    this.scrollLeft(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'A' }, (E) => this.cursorUp(E)),
                  this._parser.registerCsiHandler({ intermediates: ' ', final: 'A' }, (E) =>
                    this.scrollRight(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'B' }, (E) => this.cursorDown(E)),
                  this._parser.registerCsiHandler({ final: 'C' }, (E) => this.cursorForward(E)),
                  this._parser.registerCsiHandler({ final: 'D' }, (E) => this.cursorBackward(E)),
                  this._parser.registerCsiHandler({ final: 'E' }, (E) => this.cursorNextLine(E)),
                  this._parser.registerCsiHandler({ final: 'F' }, (E) =>
                    this.cursorPrecedingLine(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'G' }, (E) =>
                    this.cursorCharAbsolute(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'H' }, (E) => this.cursorPosition(E)),
                  this._parser.registerCsiHandler({ final: 'I' }, (E) => this.cursorForwardTab(E)),
                  this._parser.registerCsiHandler({ final: 'J' }, (E) =>
                    this.eraseInDisplay(E, !1)
                  ),
                  this._parser.registerCsiHandler({ prefix: '?', final: 'J' }, (E) =>
                    this.eraseInDisplay(E, !0)
                  ),
                  this._parser.registerCsiHandler({ final: 'K' }, (E) => this.eraseInLine(E, !1)),
                  this._parser.registerCsiHandler({ prefix: '?', final: 'K' }, (E) =>
                    this.eraseInLine(E, !0)
                  ),
                  this._parser.registerCsiHandler({ final: 'L' }, (E) => this.insertLines(E)),
                  this._parser.registerCsiHandler({ final: 'M' }, (E) => this.deleteLines(E)),
                  this._parser.registerCsiHandler({ final: 'P' }, (E) => this.deleteChars(E)),
                  this._parser.registerCsiHandler({ final: 'S' }, (E) => this.scrollUp(E)),
                  this._parser.registerCsiHandler({ final: 'T' }, (E) => this.scrollDown(E)),
                  this._parser.registerCsiHandler({ final: 'X' }, (E) => this.eraseChars(E)),
                  this._parser.registerCsiHandler({ final: 'Z' }, (E) => this.cursorBackwardTab(E)),
                  this._parser.registerCsiHandler({ final: '`' }, (E) => this.charPosAbsolute(E)),
                  this._parser.registerCsiHandler({ final: 'a' }, (E) => this.hPositionRelative(E)),
                  this._parser.registerCsiHandler({ final: 'b' }, (E) =>
                    this.repeatPrecedingCharacter(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'c' }, (E) =>
                    this.sendDeviceAttributesPrimary(E)
                  ),
                  this._parser.registerCsiHandler({ prefix: '>', final: 'c' }, (E) =>
                    this.sendDeviceAttributesSecondary(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'd' }, (E) => this.linePosAbsolute(E)),
                  this._parser.registerCsiHandler({ final: 'e' }, (E) => this.vPositionRelative(E)),
                  this._parser.registerCsiHandler({ final: 'f' }, (E) => this.hVPosition(E)),
                  this._parser.registerCsiHandler({ final: 'g' }, (E) => this.tabClear(E)),
                  this._parser.registerCsiHandler({ final: 'h' }, (E) => this.setMode(E)),
                  this._parser.registerCsiHandler({ prefix: '?', final: 'h' }, (E) =>
                    this.setModePrivate(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'l' }, (E) => this.resetMode(E)),
                  this._parser.registerCsiHandler({ prefix: '?', final: 'l' }, (E) =>
                    this.resetModePrivate(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'm' }, (E) => this.charAttributes(E)),
                  this._parser.registerCsiHandler({ final: 'n' }, (E) => this.deviceStatus(E)),
                  this._parser.registerCsiHandler({ prefix: '?', final: 'n' }, (E) =>
                    this.deviceStatusPrivate(E)
                  ),
                  this._parser.registerCsiHandler({ intermediates: '!', final: 'p' }, (E) =>
                    this.softReset(E)
                  ),
                  this._parser.registerCsiHandler({ intermediates: ' ', final: 'q' }, (E) =>
                    this.setCursorStyle(E)
                  ),
                  this._parser.registerCsiHandler({ final: 'r' }, (E) => this.setScrollRegion(E)),
                  this._parser.registerCsiHandler({ final: 's' }, (E) => this.saveCursor(E)),
                  this._parser.registerCsiHandler({ final: 't' }, (E) => this.windowOptions(E)),
                  this._parser.registerCsiHandler({ final: 'u' }, (E) => this.restoreCursor(E)),
                  this._parser.registerCsiHandler({ intermediates: "'", final: '}' }, (E) =>
                    this.insertColumns(E)
                  ),
                  this._parser.registerCsiHandler({ intermediates: "'", final: '~' }, (E) =>
                    this.deleteColumns(E)
                  ),
                  this._parser.registerCsiHandler({ intermediates: '"', final: 'q' }, (E) =>
                    this.selectProtected(E)
                  ),
                  this._parser.registerCsiHandler({ intermediates: '$', final: 'p' }, (E) =>
                    this.requestMode(E, !0)
                  ),
                  this._parser.registerCsiHandler(
                    { prefix: '?', intermediates: '$', final: 'p' },
                    (E) => this.requestMode(E, !1)
                  ),
                  this._parser.setExecuteHandler(a.C0.BEL, () => this.bell()),
                  this._parser.setExecuteHandler(a.C0.LF, () => this.lineFeed()),
                  this._parser.setExecuteHandler(a.C0.VT, () => this.lineFeed()),
                  this._parser.setExecuteHandler(a.C0.FF, () => this.lineFeed()),
                  this._parser.setExecuteHandler(a.C0.CR, () => this.carriageReturn()),
                  this._parser.setExecuteHandler(a.C0.BS, () => this.backspace()),
                  this._parser.setExecuteHandler(a.C0.HT, () => this.tab()),
                  this._parser.setExecuteHandler(a.C0.SO, () => this.shiftOut()),
                  this._parser.setExecuteHandler(a.C0.SI, () => this.shiftIn()),
                  this._parser.setExecuteHandler(a.C1.IND, () => this.index()),
                  this._parser.setExecuteHandler(a.C1.NEL, () => this.nextLine()),
                  this._parser.setExecuteHandler(a.C1.HTS, () => this.tabSet()),
                  this._parser.registerOscHandler(
                    0,
                    new b.OscHandler((E) => (this.setTitle(E), this.setIconName(E), !0))
                  ),
                  this._parser.registerOscHandler(1, new b.OscHandler((E) => this.setIconName(E))),
                  this._parser.registerOscHandler(2, new b.OscHandler((E) => this.setTitle(E))),
                  this._parser.registerOscHandler(
                    4,
                    new b.OscHandler((E) => this.setOrReportIndexedColor(E))
                  ),
                  this._parser.registerOscHandler(8, new b.OscHandler((E) => this.setHyperlink(E))),
                  this._parser.registerOscHandler(
                    10,
                    new b.OscHandler((E) => this.setOrReportFgColor(E))
                  ),
                  this._parser.registerOscHandler(
                    11,
                    new b.OscHandler((E) => this.setOrReportBgColor(E))
                  ),
                  this._parser.registerOscHandler(
                    12,
                    new b.OscHandler((E) => this.setOrReportCursorColor(E))
                  ),
                  this._parser.registerOscHandler(
                    104,
                    new b.OscHandler((E) => this.restoreIndexedColor(E))
                  ),
                  this._parser.registerOscHandler(
                    110,
                    new b.OscHandler((E) => this.restoreFgColor(E))
                  ),
                  this._parser.registerOscHandler(
                    111,
                    new b.OscHandler((E) => this.restoreBgColor(E))
                  ),
                  this._parser.registerOscHandler(
                    112,
                    new b.OscHandler((E) => this.restoreCursorColor(E))
                  ),
                  this._parser.registerEscHandler({ final: '7' }, () => this.saveCursor()),
                  this._parser.registerEscHandler({ final: '8' }, () => this.restoreCursor()),
                  this._parser.registerEscHandler({ final: 'D' }, () => this.index()),
                  this._parser.registerEscHandler({ final: 'E' }, () => this.nextLine()),
                  this._parser.registerEscHandler({ final: 'H' }, () => this.tabSet()),
                  this._parser.registerEscHandler({ final: 'M' }, () => this.reverseIndex()),
                  this._parser.registerEscHandler({ final: '=' }, () =>
                    this.keypadApplicationMode()
                  ),
                  this._parser.registerEscHandler({ final: '>' }, () => this.keypadNumericMode()),
                  this._parser.registerEscHandler({ final: 'c' }, () => this.fullReset()),
                  this._parser.registerEscHandler({ final: 'n' }, () => this.setgLevel(2)),
                  this._parser.registerEscHandler({ final: 'o' }, () => this.setgLevel(3)),
                  this._parser.registerEscHandler({ final: '|' }, () => this.setgLevel(3)),
                  this._parser.registerEscHandler({ final: '}' }, () => this.setgLevel(2)),
                  this._parser.registerEscHandler({ final: '~' }, () => this.setgLevel(1)),
                  this._parser.registerEscHandler({ intermediates: '%', final: '@' }, () =>
                    this.selectDefaultCharset()
                  ),
                  this._parser.registerEscHandler({ intermediates: '%', final: 'G' }, () =>
                    this.selectDefaultCharset()
                  ))
                for (const E in h.CHARSETS)
                  (this._parser.registerEscHandler({ intermediates: '(', final: E }, () =>
                    this.selectCharset('(' + E)
                  ),
                    this._parser.registerEscHandler({ intermediates: ')', final: E }, () =>
                      this.selectCharset(')' + E)
                    ),
                    this._parser.registerEscHandler({ intermediates: '*', final: E }, () =>
                      this.selectCharset('*' + E)
                    ),
                    this._parser.registerEscHandler({ intermediates: '+', final: E }, () =>
                      this.selectCharset('+' + E)
                    ),
                    this._parser.registerEscHandler({ intermediates: '-', final: E }, () =>
                      this.selectCharset('-' + E)
                    ),
                    this._parser.registerEscHandler({ intermediates: '.', final: E }, () =>
                      this.selectCharset('.' + E)
                    ),
                    this._parser.registerEscHandler({ intermediates: '/', final: E }, () =>
                      this.selectCharset('/' + E)
                    ))
                ;(this._parser.registerEscHandler({ intermediates: '#', final: '8' }, () =>
                  this.screenAlignmentPattern()
                ),
                  this._parser.setErrorHandler(
                    (E) => (this._logService.error('Parsing error: ', E), E)
                  ),
                  this._parser.registerDcsHandler(
                    { intermediates: '$', final: 'q' },
                    new p.DcsHandler((E, G) => this.requestStatusString(E, G))
                  ))
              }
              _preserveStack(C, A, N, B) {
                ;((this._parseStack.paused = !0),
                  (this._parseStack.cursorStartX = C),
                  (this._parseStack.cursorStartY = A),
                  (this._parseStack.decodedLength = N),
                  (this._parseStack.position = B))
              }
              _logSlowResolvingAsync(C) {
                this._logService.logLevel <= v.LogLevelEnum.WARN &&
                  Promise.race([
                    C,
                    new Promise((A, N) => setTimeout(() => N('#SLOW_TIMEOUT'), 5e3))
                  ]).catch((A) => {
                    if (A !== '#SLOW_TIMEOUT') throw A
                    console.warn('async parser handler taking longer than 5000 ms')
                  })
              }
              _getCurrentLinkId() {
                return this._curAttrData.extended.urlId
              }
              parse(C, A) {
                let N,
                  B = this._activeBuffer.x,
                  z = this._activeBuffer.y,
                  K = 0
                const J = this._parseStack.paused
                if (J) {
                  if (
                    (N = this._parser.parse(this._parseBuffer, this._parseStack.decodedLength, A))
                  )
                    return (this._logSlowResolvingAsync(N), N)
                  ;((B = this._parseStack.cursorStartX),
                    (z = this._parseStack.cursorStartY),
                    (this._parseStack.paused = !1),
                    C.length > M && (K = this._parseStack.position + M))
                }
                if (
                  (this._logService.logLevel <= v.LogLevelEnum.DEBUG &&
                    this._logService.debug(
                      'parsing data' +
                        (typeof C == 'string'
                          ? ` "${C}"`
                          : ` "${Array.prototype.map.call(C, (E) => String.fromCharCode(E)).join('')}"`),
                      typeof C == 'string' ? C.split('').map((E) => E.charCodeAt(0)) : C
                    ),
                  this._parseBuffer.length < C.length &&
                    this._parseBuffer.length < M &&
                    (this._parseBuffer = new Uint32Array(Math.min(C.length, M))),
                  J || this._dirtyRowTracker.clearRange(),
                  C.length > M)
                )
                  for (let E = K; E < C.length; E += M) {
                    const G = E + M < C.length ? E + M : C.length,
                      q =
                        typeof C == 'string'
                          ? this._stringDecoder.decode(C.substring(E, G), this._parseBuffer)
                          : this._utf8Decoder.decode(C.subarray(E, G), this._parseBuffer)
                    if ((N = this._parser.parse(this._parseBuffer, q)))
                      return (this._preserveStack(B, z, q, E), this._logSlowResolvingAsync(N), N)
                  }
                else if (!J) {
                  const E =
                    typeof C == 'string'
                      ? this._stringDecoder.decode(C, this._parseBuffer)
                      : this._utf8Decoder.decode(C, this._parseBuffer)
                  if ((N = this._parser.parse(this._parseBuffer, E)))
                    return (this._preserveStack(B, z, E, 0), this._logSlowResolvingAsync(N), N)
                }
                ;(this._activeBuffer.x === B && this._activeBuffer.y === z) ||
                  this._onCursorMove.fire()
                const Q =
                    this._dirtyRowTracker.end +
                    (this._bufferService.buffer.ybase - this._bufferService.buffer.ydisp),
                  H =
                    this._dirtyRowTracker.start +
                    (this._bufferService.buffer.ybase - this._bufferService.buffer.ydisp)
                H < this._bufferService.rows &&
                  this._onRequestRefreshRows.fire(
                    Math.min(H, this._bufferService.rows - 1),
                    Math.min(Q, this._bufferService.rows - 1)
                  )
              }
              print(C, A, N) {
                let B, z
                const K = this._charsetService.charset,
                  J = this._optionsService.rawOptions.screenReaderMode,
                  Q = this._bufferService.cols,
                  H = this._coreService.decPrivateModes.wraparound,
                  E = this._coreService.modes.insertMode,
                  G = this._curAttrData
                let q = this._activeBuffer.lines.get(
                  this._activeBuffer.ybase + this._activeBuffer.y
                )
                ;(this._dirtyRowTracker.markDirty(this._activeBuffer.y),
                  this._activeBuffer.x &&
                    N - A > 0 &&
                    q.getWidth(this._activeBuffer.x - 1) === 2 &&
                    q.setCellFromCodepoint(this._activeBuffer.x - 1, 0, 1, G))
                let Z = this._parser.precedingJoinState
                for (let Y = A; Y < N; ++Y) {
                  if (((B = C[Y]), B < 127 && K)) {
                    const ue = K[String.fromCharCode(B)]
                    ue && (B = ue.charCodeAt(0))
                  }
                  const V = this._unicodeService.charProperties(B, Z)
                  z = _.UnicodeService.extractWidth(V)
                  const se = _.UnicodeService.extractShouldJoin(V),
                    ne = se ? _.UnicodeService.extractWidth(Z) : 0
                  if (
                    ((Z = V),
                    J && this._onA11yChar.fire((0, c.stringFromCodePoint)(B)),
                    this._getCurrentLinkId() &&
                      this._oscLinkService.addLineToLink(
                        this._getCurrentLinkId(),
                        this._activeBuffer.ybase + this._activeBuffer.y
                      ),
                    this._activeBuffer.x + z - ne > Q)
                  ) {
                    if (H) {
                      const ue = q
                      let re = this._activeBuffer.x - ne
                      for (
                        this._activeBuffer.x = ne,
                          this._activeBuffer.y++,
                          this._activeBuffer.y === this._activeBuffer.scrollBottom + 1
                            ? (this._activeBuffer.y--,
                              this._bufferService.scroll(this._eraseAttrData(), !0))
                            : (this._activeBuffer.y >= this._bufferService.rows &&
                                (this._activeBuffer.y = this._bufferService.rows - 1),
                              (this._activeBuffer.lines.get(
                                this._activeBuffer.ybase + this._activeBuffer.y
                              ).isWrapped = !0)),
                          q = this._activeBuffer.lines.get(
                            this._activeBuffer.ybase + this._activeBuffer.y
                          ),
                          ne > 0 && q instanceof t.BufferLine && q.copyCellsFrom(ue, re, 0, ne, !1);
                        re < Q;

                      )
                        ue.setCellFromCodepoint(re++, 0, 1, G)
                    } else if (((this._activeBuffer.x = Q - 1), z === 2)) continue
                  }
                  if (se && this._activeBuffer.x) {
                    const ue = q.getWidth(this._activeBuffer.x - 1) ? 1 : 2
                    q.addCodepointToCell(this._activeBuffer.x - ue, B, z)
                    for (let re = z - ne; --re >= 0; )
                      q.setCellFromCodepoint(this._activeBuffer.x++, 0, 0, G)
                  } else if (
                    (E &&
                      (q.insertCells(
                        this._activeBuffer.x,
                        z - ne,
                        this._activeBuffer.getNullCell(G)
                      ),
                      q.getWidth(Q - 1) === 2 &&
                        q.setCellFromCodepoint(Q - 1, s.NULL_CELL_CODE, s.NULL_CELL_WIDTH, G)),
                    q.setCellFromCodepoint(this._activeBuffer.x++, B, z, G),
                    z > 0)
                  )
                    for (; --z; ) q.setCellFromCodepoint(this._activeBuffer.x++, 0, 0, G)
                }
                ;((this._parser.precedingJoinState = Z),
                  this._activeBuffer.x < Q &&
                    N - A > 0 &&
                    q.getWidth(this._activeBuffer.x) === 0 &&
                    !q.hasContent(this._activeBuffer.x) &&
                    q.setCellFromCodepoint(this._activeBuffer.x, 0, 1, G),
                  this._dirtyRowTracker.markDirty(this._activeBuffer.y))
              }
              registerCsiHandler(C, A) {
                return C.final !== 't' || C.prefix || C.intermediates
                  ? this._parser.registerCsiHandler(C, A)
                  : this._parser.registerCsiHandler(
                      C,
                      (N) => !P(N.params[0], this._optionsService.rawOptions.windowOptions) || A(N)
                    )
              }
              registerDcsHandler(C, A) {
                return this._parser.registerDcsHandler(C, new p.DcsHandler(A))
              }
              registerEscHandler(C, A) {
                return this._parser.registerEscHandler(C, A)
              }
              registerOscHandler(C, A) {
                return this._parser.registerOscHandler(C, new b.OscHandler(A))
              }
              bell() {
                return (this._onRequestBell.fire(), !0)
              }
              lineFeed() {
                return (
                  this._dirtyRowTracker.markDirty(this._activeBuffer.y),
                  this._optionsService.rawOptions.convertEol && (this._activeBuffer.x = 0),
                  this._activeBuffer.y++,
                  this._activeBuffer.y === this._activeBuffer.scrollBottom + 1
                    ? (this._activeBuffer.y--, this._bufferService.scroll(this._eraseAttrData()))
                    : this._activeBuffer.y >= this._bufferService.rows
                      ? (this._activeBuffer.y = this._bufferService.rows - 1)
                      : (this._activeBuffer.lines.get(
                          this._activeBuffer.ybase + this._activeBuffer.y
                        ).isWrapped = !1),
                  this._activeBuffer.x >= this._bufferService.cols && this._activeBuffer.x--,
                  this._dirtyRowTracker.markDirty(this._activeBuffer.y),
                  this._onLineFeed.fire(),
                  !0
                )
              }
              carriageReturn() {
                return ((this._activeBuffer.x = 0), !0)
              }
              backspace() {
                if (!this._coreService.decPrivateModes.reverseWraparound)
                  return (
                    this._restrictCursor(),
                    this._activeBuffer.x > 0 && this._activeBuffer.x--,
                    !0
                  )
                if ((this._restrictCursor(this._bufferService.cols), this._activeBuffer.x > 0))
                  this._activeBuffer.x--
                else if (
                  this._activeBuffer.x === 0 &&
                  this._activeBuffer.y > this._activeBuffer.scrollTop &&
                  this._activeBuffer.y <= this._activeBuffer.scrollBottom &&
                  this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y)
                    ?.isWrapped
                ) {
                  ;((this._activeBuffer.lines.get(
                    this._activeBuffer.ybase + this._activeBuffer.y
                  ).isWrapped = !1),
                    this._activeBuffer.y--,
                    (this._activeBuffer.x = this._bufferService.cols - 1))
                  const C = this._activeBuffer.lines.get(
                    this._activeBuffer.ybase + this._activeBuffer.y
                  )
                  C.hasWidth(this._activeBuffer.x) &&
                    !C.hasContent(this._activeBuffer.x) &&
                    this._activeBuffer.x--
                }
                return (this._restrictCursor(), !0)
              }
              tab() {
                if (this._activeBuffer.x >= this._bufferService.cols) return !0
                const C = this._activeBuffer.x
                return (
                  (this._activeBuffer.x = this._activeBuffer.nextStop()),
                  this._optionsService.rawOptions.screenReaderMode &&
                    this._onA11yTab.fire(this._activeBuffer.x - C),
                  !0
                )
              }
              shiftOut() {
                return (this._charsetService.setgLevel(1), !0)
              }
              shiftIn() {
                return (this._charsetService.setgLevel(0), !0)
              }
              _restrictCursor(C = this._bufferService.cols - 1) {
                ;((this._activeBuffer.x = Math.min(C, Math.max(0, this._activeBuffer.x))),
                  (this._activeBuffer.y = this._coreService.decPrivateModes.origin
                    ? Math.min(
                        this._activeBuffer.scrollBottom,
                        Math.max(this._activeBuffer.scrollTop, this._activeBuffer.y)
                      )
                    : Math.min(this._bufferService.rows - 1, Math.max(0, this._activeBuffer.y))),
                  this._dirtyRowTracker.markDirty(this._activeBuffer.y))
              }
              _setCursor(C, A) {
                ;(this._dirtyRowTracker.markDirty(this._activeBuffer.y),
                  this._coreService.decPrivateModes.origin
                    ? ((this._activeBuffer.x = C),
                      (this._activeBuffer.y = this._activeBuffer.scrollTop + A))
                    : ((this._activeBuffer.x = C), (this._activeBuffer.y = A)),
                  this._restrictCursor(),
                  this._dirtyRowTracker.markDirty(this._activeBuffer.y))
              }
              _moveCursor(C, A) {
                ;(this._restrictCursor(),
                  this._setCursor(this._activeBuffer.x + C, this._activeBuffer.y + A))
              }
              cursorUp(C) {
                const A = this._activeBuffer.y - this._activeBuffer.scrollTop
                return (
                  A >= 0
                    ? this._moveCursor(0, -Math.min(A, C.params[0] || 1))
                    : this._moveCursor(0, -(C.params[0] || 1)),
                  !0
                )
              }
              cursorDown(C) {
                const A = this._activeBuffer.scrollBottom - this._activeBuffer.y
                return (
                  A >= 0
                    ? this._moveCursor(0, Math.min(A, C.params[0] || 1))
                    : this._moveCursor(0, C.params[0] || 1),
                  !0
                )
              }
              cursorForward(C) {
                return (this._moveCursor(C.params[0] || 1, 0), !0)
              }
              cursorBackward(C) {
                return (this._moveCursor(-(C.params[0] || 1), 0), !0)
              }
              cursorNextLine(C) {
                return (this.cursorDown(C), (this._activeBuffer.x = 0), !0)
              }
              cursorPrecedingLine(C) {
                return (this.cursorUp(C), (this._activeBuffer.x = 0), !0)
              }
              cursorCharAbsolute(C) {
                return (this._setCursor((C.params[0] || 1) - 1, this._activeBuffer.y), !0)
              }
              cursorPosition(C) {
                return (
                  this._setCursor(
                    C.length >= 2 ? (C.params[1] || 1) - 1 : 0,
                    (C.params[0] || 1) - 1
                  ),
                  !0
                )
              }
              charPosAbsolute(C) {
                return (this._setCursor((C.params[0] || 1) - 1, this._activeBuffer.y), !0)
              }
              hPositionRelative(C) {
                return (this._moveCursor(C.params[0] || 1, 0), !0)
              }
              linePosAbsolute(C) {
                return (this._setCursor(this._activeBuffer.x, (C.params[0] || 1) - 1), !0)
              }
              vPositionRelative(C) {
                return (this._moveCursor(0, C.params[0] || 1), !0)
              }
              hVPosition(C) {
                return (this.cursorPosition(C), !0)
              }
              tabClear(C) {
                const A = C.params[0]
                return (
                  A === 0
                    ? delete this._activeBuffer.tabs[this._activeBuffer.x]
                    : A === 3 && (this._activeBuffer.tabs = {}),
                  !0
                )
              }
              cursorForwardTab(C) {
                if (this._activeBuffer.x >= this._bufferService.cols) return !0
                let A = C.params[0] || 1
                for (; A--; ) this._activeBuffer.x = this._activeBuffer.nextStop()
                return !0
              }
              cursorBackwardTab(C) {
                if (this._activeBuffer.x >= this._bufferService.cols) return !0
                let A = C.params[0] || 1
                for (; A--; ) this._activeBuffer.x = this._activeBuffer.prevStop()
                return !0
              }
              selectProtected(C) {
                const A = C.params[0]
                return (
                  A === 1 && (this._curAttrData.bg |= 536870912),
                  (A !== 2 && A !== 0) || (this._curAttrData.bg &= -536870913),
                  !0
                )
              }
              _eraseInBufferLine(C, A, N, B = !1, z = !1) {
                const K = this._activeBuffer.lines.get(this._activeBuffer.ybase + C)
                ;(K.replaceCells(A, N, this._activeBuffer.getNullCell(this._eraseAttrData()), z),
                  B && (K.isWrapped = !1))
              }
              _resetBufferLine(C, A = !1) {
                const N = this._activeBuffer.lines.get(this._activeBuffer.ybase + C)
                N &&
                  (N.fill(this._activeBuffer.getNullCell(this._eraseAttrData()), A),
                  this._bufferService.buffer.clearMarkers(this._activeBuffer.ybase + C),
                  (N.isWrapped = !1))
              }
              eraseInDisplay(C, A = !1) {
                let N
                switch ((this._restrictCursor(this._bufferService.cols), C.params[0])) {
                  case 0:
                    for (
                      N = this._activeBuffer.y,
                        this._dirtyRowTracker.markDirty(N),
                        this._eraseInBufferLine(
                          N++,
                          this._activeBuffer.x,
                          this._bufferService.cols,
                          this._activeBuffer.x === 0,
                          A
                        );
                      N < this._bufferService.rows;
                      N++
                    )
                      this._resetBufferLine(N, A)
                    this._dirtyRowTracker.markDirty(N)
                    break
                  case 1:
                    for (
                      N = this._activeBuffer.y,
                        this._dirtyRowTracker.markDirty(N),
                        this._eraseInBufferLine(N, 0, this._activeBuffer.x + 1, !0, A),
                        this._activeBuffer.x + 1 >= this._bufferService.cols &&
                          (this._activeBuffer.lines.get(N + 1).isWrapped = !1);
                      N--;

                    )
                      this._resetBufferLine(N, A)
                    this._dirtyRowTracker.markDirty(0)
                    break
                  case 2:
                    for (
                      N = this._bufferService.rows, this._dirtyRowTracker.markDirty(N - 1);
                      N--;

                    )
                      this._resetBufferLine(N, A)
                    this._dirtyRowTracker.markDirty(0)
                    break
                  case 3:
                    const B = this._activeBuffer.lines.length - this._bufferService.rows
                    B > 0 &&
                      (this._activeBuffer.lines.trimStart(B),
                      (this._activeBuffer.ybase = Math.max(this._activeBuffer.ybase - B, 0)),
                      (this._activeBuffer.ydisp = Math.max(this._activeBuffer.ydisp - B, 0)),
                      this._onScroll.fire(0))
                }
                return !0
              }
              eraseInLine(C, A = !1) {
                switch ((this._restrictCursor(this._bufferService.cols), C.params[0])) {
                  case 0:
                    this._eraseInBufferLine(
                      this._activeBuffer.y,
                      this._activeBuffer.x,
                      this._bufferService.cols,
                      this._activeBuffer.x === 0,
                      A
                    )
                    break
                  case 1:
                    this._eraseInBufferLine(
                      this._activeBuffer.y,
                      0,
                      this._activeBuffer.x + 1,
                      !1,
                      A
                    )
                    break
                  case 2:
                    this._eraseInBufferLine(
                      this._activeBuffer.y,
                      0,
                      this._bufferService.cols,
                      !0,
                      A
                    )
                }
                return (this._dirtyRowTracker.markDirty(this._activeBuffer.y), !0)
              }
              insertLines(C) {
                this._restrictCursor()
                let A = C.params[0] || 1
                if (
                  this._activeBuffer.y > this._activeBuffer.scrollBottom ||
                  this._activeBuffer.y < this._activeBuffer.scrollTop
                )
                  return !0
                const N = this._activeBuffer.ybase + this._activeBuffer.y,
                  B = this._bufferService.rows - 1 - this._activeBuffer.scrollBottom,
                  z = this._bufferService.rows - 1 + this._activeBuffer.ybase - B + 1
                for (; A--; )
                  (this._activeBuffer.lines.splice(z - 1, 1),
                    this._activeBuffer.lines.splice(
                      N,
                      0,
                      this._activeBuffer.getBlankLine(this._eraseAttrData())
                    ))
                return (
                  this._dirtyRowTracker.markRangeDirty(
                    this._activeBuffer.y,
                    this._activeBuffer.scrollBottom
                  ),
                  (this._activeBuffer.x = 0),
                  !0
                )
              }
              deleteLines(C) {
                this._restrictCursor()
                let A = C.params[0] || 1
                if (
                  this._activeBuffer.y > this._activeBuffer.scrollBottom ||
                  this._activeBuffer.y < this._activeBuffer.scrollTop
                )
                  return !0
                const N = this._activeBuffer.ybase + this._activeBuffer.y
                let B
                for (
                  B = this._bufferService.rows - 1 - this._activeBuffer.scrollBottom,
                    B = this._bufferService.rows - 1 + this._activeBuffer.ybase - B;
                  A--;

                )
                  (this._activeBuffer.lines.splice(N, 1),
                    this._activeBuffer.lines.splice(
                      B,
                      0,
                      this._activeBuffer.getBlankLine(this._eraseAttrData())
                    ))
                return (
                  this._dirtyRowTracker.markRangeDirty(
                    this._activeBuffer.y,
                    this._activeBuffer.scrollBottom
                  ),
                  (this._activeBuffer.x = 0),
                  !0
                )
              }
              insertChars(C) {
                this._restrictCursor()
                const A = this._activeBuffer.lines.get(
                  this._activeBuffer.ybase + this._activeBuffer.y
                )
                return (
                  A &&
                    (A.insertCells(
                      this._activeBuffer.x,
                      C.params[0] || 1,
                      this._activeBuffer.getNullCell(this._eraseAttrData())
                    ),
                    this._dirtyRowTracker.markDirty(this._activeBuffer.y)),
                  !0
                )
              }
              deleteChars(C) {
                this._restrictCursor()
                const A = this._activeBuffer.lines.get(
                  this._activeBuffer.ybase + this._activeBuffer.y
                )
                return (
                  A &&
                    (A.deleteCells(
                      this._activeBuffer.x,
                      C.params[0] || 1,
                      this._activeBuffer.getNullCell(this._eraseAttrData())
                    ),
                    this._dirtyRowTracker.markDirty(this._activeBuffer.y)),
                  !0
                )
              }
              scrollUp(C) {
                let A = C.params[0] || 1
                for (; A--; )
                  (this._activeBuffer.lines.splice(
                    this._activeBuffer.ybase + this._activeBuffer.scrollTop,
                    1
                  ),
                    this._activeBuffer.lines.splice(
                      this._activeBuffer.ybase + this._activeBuffer.scrollBottom,
                      0,
                      this._activeBuffer.getBlankLine(this._eraseAttrData())
                    ))
                return (
                  this._dirtyRowTracker.markRangeDirty(
                    this._activeBuffer.scrollTop,
                    this._activeBuffer.scrollBottom
                  ),
                  !0
                )
              }
              scrollDown(C) {
                let A = C.params[0] || 1
                for (; A--; )
                  (this._activeBuffer.lines.splice(
                    this._activeBuffer.ybase + this._activeBuffer.scrollBottom,
                    1
                  ),
                    this._activeBuffer.lines.splice(
                      this._activeBuffer.ybase + this._activeBuffer.scrollTop,
                      0,
                      this._activeBuffer.getBlankLine(t.DEFAULT_ATTR_DATA)
                    ))
                return (
                  this._dirtyRowTracker.markRangeDirty(
                    this._activeBuffer.scrollTop,
                    this._activeBuffer.scrollBottom
                  ),
                  !0
                )
              }
              scrollLeft(C) {
                if (
                  this._activeBuffer.y > this._activeBuffer.scrollBottom ||
                  this._activeBuffer.y < this._activeBuffer.scrollTop
                )
                  return !0
                const A = C.params[0] || 1
                for (
                  let N = this._activeBuffer.scrollTop;
                  N <= this._activeBuffer.scrollBottom;
                  ++N
                ) {
                  const B = this._activeBuffer.lines.get(this._activeBuffer.ybase + N)
                  ;(B.deleteCells(0, A, this._activeBuffer.getNullCell(this._eraseAttrData())),
                    (B.isWrapped = !1))
                }
                return (
                  this._dirtyRowTracker.markRangeDirty(
                    this._activeBuffer.scrollTop,
                    this._activeBuffer.scrollBottom
                  ),
                  !0
                )
              }
              scrollRight(C) {
                if (
                  this._activeBuffer.y > this._activeBuffer.scrollBottom ||
                  this._activeBuffer.y < this._activeBuffer.scrollTop
                )
                  return !0
                const A = C.params[0] || 1
                for (
                  let N = this._activeBuffer.scrollTop;
                  N <= this._activeBuffer.scrollBottom;
                  ++N
                ) {
                  const B = this._activeBuffer.lines.get(this._activeBuffer.ybase + N)
                  ;(B.insertCells(0, A, this._activeBuffer.getNullCell(this._eraseAttrData())),
                    (B.isWrapped = !1))
                }
                return (
                  this._dirtyRowTracker.markRangeDirty(
                    this._activeBuffer.scrollTop,
                    this._activeBuffer.scrollBottom
                  ),
                  !0
                )
              }
              insertColumns(C) {
                if (
                  this._activeBuffer.y > this._activeBuffer.scrollBottom ||
                  this._activeBuffer.y < this._activeBuffer.scrollTop
                )
                  return !0
                const A = C.params[0] || 1
                for (
                  let N = this._activeBuffer.scrollTop;
                  N <= this._activeBuffer.scrollBottom;
                  ++N
                ) {
                  const B = this._activeBuffer.lines.get(this._activeBuffer.ybase + N)
                  ;(B.insertCells(
                    this._activeBuffer.x,
                    A,
                    this._activeBuffer.getNullCell(this._eraseAttrData())
                  ),
                    (B.isWrapped = !1))
                }
                return (
                  this._dirtyRowTracker.markRangeDirty(
                    this._activeBuffer.scrollTop,
                    this._activeBuffer.scrollBottom
                  ),
                  !0
                )
              }
              deleteColumns(C) {
                if (
                  this._activeBuffer.y > this._activeBuffer.scrollBottom ||
                  this._activeBuffer.y < this._activeBuffer.scrollTop
                )
                  return !0
                const A = C.params[0] || 1
                for (
                  let N = this._activeBuffer.scrollTop;
                  N <= this._activeBuffer.scrollBottom;
                  ++N
                ) {
                  const B = this._activeBuffer.lines.get(this._activeBuffer.ybase + N)
                  ;(B.deleteCells(
                    this._activeBuffer.x,
                    A,
                    this._activeBuffer.getNullCell(this._eraseAttrData())
                  ),
                    (B.isWrapped = !1))
                }
                return (
                  this._dirtyRowTracker.markRangeDirty(
                    this._activeBuffer.scrollTop,
                    this._activeBuffer.scrollBottom
                  ),
                  !0
                )
              }
              eraseChars(C) {
                this._restrictCursor()
                const A = this._activeBuffer.lines.get(
                  this._activeBuffer.ybase + this._activeBuffer.y
                )
                return (
                  A &&
                    (A.replaceCells(
                      this._activeBuffer.x,
                      this._activeBuffer.x + (C.params[0] || 1),
                      this._activeBuffer.getNullCell(this._eraseAttrData())
                    ),
                    this._dirtyRowTracker.markDirty(this._activeBuffer.y)),
                  !0
                )
              }
              repeatPrecedingCharacter(C) {
                const A = this._parser.precedingJoinState
                if (!A) return !0
                const N = C.params[0] || 1,
                  B = _.UnicodeService.extractWidth(A),
                  z = this._activeBuffer.x - B,
                  K = this._activeBuffer.lines
                    .get(this._activeBuffer.ybase + this._activeBuffer.y)
                    .getString(z),
                  J = new Uint32Array(K.length * N)
                let Q = 0
                for (let E = 0; E < K.length; ) {
                  const G = K.codePointAt(E) || 0
                  ;((J[Q++] = G), (E += G > 65535 ? 2 : 1))
                }
                let H = Q
                for (let E = 1; E < N; ++E) (J.copyWithin(H, 0, Q), (H += Q))
                return (this.print(J, 0, H), !0)
              }
              sendDeviceAttributesPrimary(C) {
                return (
                  C.params[0] > 0 ||
                    (this._is('xterm') || this._is('rxvt-unicode') || this._is('screen')
                      ? this._coreService.triggerDataEvent(a.C0.ESC + '[?1;2c')
                      : this._is('linux') && this._coreService.triggerDataEvent(a.C0.ESC + '[?6c')),
                  !0
                )
              }
              sendDeviceAttributesSecondary(C) {
                return (
                  C.params[0] > 0 ||
                    (this._is('xterm')
                      ? this._coreService.triggerDataEvent(a.C0.ESC + '[>0;276;0c')
                      : this._is('rxvt-unicode')
                        ? this._coreService.triggerDataEvent(a.C0.ESC + '[>85;95;0c')
                        : this._is('linux')
                          ? this._coreService.triggerDataEvent(C.params[0] + 'c')
                          : this._is('screen') &&
                            this._coreService.triggerDataEvent(a.C0.ESC + '[>83;40003;0c')),
                  !0
                )
              }
              _is(C) {
                return (this._optionsService.rawOptions.termName + '').indexOf(C) === 0
              }
              setMode(C) {
                for (let A = 0; A < C.length; A++)
                  switch (C.params[A]) {
                    case 4:
                      this._coreService.modes.insertMode = !0
                      break
                    case 20:
                      this._optionsService.options.convertEol = !0
                  }
                return !0
              }
              setModePrivate(C) {
                for (let A = 0; A < C.length; A++)
                  switch (C.params[A]) {
                    case 1:
                      this._coreService.decPrivateModes.applicationCursorKeys = !0
                      break
                    case 2:
                      ;(this._charsetService.setgCharset(0, h.DEFAULT_CHARSET),
                        this._charsetService.setgCharset(1, h.DEFAULT_CHARSET),
                        this._charsetService.setgCharset(2, h.DEFAULT_CHARSET),
                        this._charsetService.setgCharset(3, h.DEFAULT_CHARSET))
                      break
                    case 3:
                      this._optionsService.rawOptions.windowOptions.setWinLines &&
                        (this._bufferService.resize(132, this._bufferService.rows),
                        this._onRequestReset.fire())
                      break
                    case 6:
                      ;((this._coreService.decPrivateModes.origin = !0), this._setCursor(0, 0))
                      break
                    case 7:
                      this._coreService.decPrivateModes.wraparound = !0
                      break
                    case 12:
                      this._optionsService.options.cursorBlink = !0
                      break
                    case 45:
                      this._coreService.decPrivateModes.reverseWraparound = !0
                      break
                    case 66:
                      ;(this._logService.debug('Serial port requested application keypad.'),
                        (this._coreService.decPrivateModes.applicationKeypad = !0),
                        this._onRequestSyncScrollBar.fire())
                      break
                    case 9:
                      this._coreMouseService.activeProtocol = 'X10'
                      break
                    case 1e3:
                      this._coreMouseService.activeProtocol = 'VT200'
                      break
                    case 1002:
                      this._coreMouseService.activeProtocol = 'DRAG'
                      break
                    case 1003:
                      this._coreMouseService.activeProtocol = 'ANY'
                      break
                    case 1004:
                      ;((this._coreService.decPrivateModes.sendFocus = !0),
                        this._onRequestSendFocus.fire())
                      break
                    case 1005:
                      this._logService.debug('DECSET 1005 not supported (see #2507)')
                      break
                    case 1006:
                      this._coreMouseService.activeEncoding = 'SGR'
                      break
                    case 1015:
                      this._logService.debug('DECSET 1015 not supported (see #2507)')
                      break
                    case 1016:
                      this._coreMouseService.activeEncoding = 'SGR_PIXELS'
                      break
                    case 25:
                      this._coreService.isCursorHidden = !1
                      break
                    case 1048:
                      this.saveCursor()
                      break
                    case 1049:
                      this.saveCursor()
                    case 47:
                    case 1047:
                      ;(this._bufferService.buffers.activateAltBuffer(this._eraseAttrData()),
                        (this._coreService.isCursorInitialized = !0),
                        this._onRequestRefreshRows.fire(0, this._bufferService.rows - 1),
                        this._onRequestSyncScrollBar.fire())
                      break
                    case 2004:
                      this._coreService.decPrivateModes.bracketedPasteMode = !0
                  }
                return !0
              }
              resetMode(C) {
                for (let A = 0; A < C.length; A++)
                  switch (C.params[A]) {
                    case 4:
                      this._coreService.modes.insertMode = !1
                      break
                    case 20:
                      this._optionsService.options.convertEol = !1
                  }
                return !0
              }
              resetModePrivate(C) {
                for (let A = 0; A < C.length; A++)
                  switch (C.params[A]) {
                    case 1:
                      this._coreService.decPrivateModes.applicationCursorKeys = !1
                      break
                    case 3:
                      this._optionsService.rawOptions.windowOptions.setWinLines &&
                        (this._bufferService.resize(80, this._bufferService.rows),
                        this._onRequestReset.fire())
                      break
                    case 6:
                      ;((this._coreService.decPrivateModes.origin = !1), this._setCursor(0, 0))
                      break
                    case 7:
                      this._coreService.decPrivateModes.wraparound = !1
                      break
                    case 12:
                      this._optionsService.options.cursorBlink = !1
                      break
                    case 45:
                      this._coreService.decPrivateModes.reverseWraparound = !1
                      break
                    case 66:
                      ;(this._logService.debug('Switching back to normal keypad.'),
                        (this._coreService.decPrivateModes.applicationKeypad = !1),
                        this._onRequestSyncScrollBar.fire())
                      break
                    case 9:
                    case 1e3:
                    case 1002:
                    case 1003:
                      this._coreMouseService.activeProtocol = 'NONE'
                      break
                    case 1004:
                      this._coreService.decPrivateModes.sendFocus = !1
                      break
                    case 1005:
                      this._logService.debug('DECRST 1005 not supported (see #2507)')
                      break
                    case 1006:
                    case 1016:
                      this._coreMouseService.activeEncoding = 'DEFAULT'
                      break
                    case 1015:
                      this._logService.debug('DECRST 1015 not supported (see #2507)')
                      break
                    case 25:
                      this._coreService.isCursorHidden = !0
                      break
                    case 1048:
                      this.restoreCursor()
                      break
                    case 1049:
                    case 47:
                    case 1047:
                      ;(this._bufferService.buffers.activateNormalBuffer(),
                        C.params[A] === 1049 && this.restoreCursor(),
                        (this._coreService.isCursorInitialized = !0),
                        this._onRequestRefreshRows.fire(0, this._bufferService.rows - 1),
                        this._onRequestSyncScrollBar.fire())
                      break
                    case 2004:
                      this._coreService.decPrivateModes.bracketedPasteMode = !1
                  }
                return !0
              }
              requestMode(C, A) {
                const N = this._coreService.decPrivateModes,
                  { activeProtocol: B, activeEncoding: z } = this._coreMouseService,
                  K = this._coreService,
                  { buffers: J, cols: Q } = this._bufferService,
                  { active: H, alt: E } = J,
                  G = this._optionsService.rawOptions,
                  q = (se) => (se ? 1 : 2),
                  Z = C.params[0]
                return (
                  (Y = Z),
                  (V = A
                    ? Z === 2
                      ? 4
                      : Z === 4
                        ? q(K.modes.insertMode)
                        : Z === 12
                          ? 3
                          : Z === 20
                            ? q(G.convertEol)
                            : 0
                    : Z === 1
                      ? q(N.applicationCursorKeys)
                      : Z === 3
                        ? G.windowOptions.setWinLines
                          ? Q === 80
                            ? 2
                            : Q === 132
                              ? 1
                              : 0
                          : 0
                        : Z === 6
                          ? q(N.origin)
                          : Z === 7
                            ? q(N.wraparound)
                            : Z === 8
                              ? 3
                              : Z === 9
                                ? q(B === 'X10')
                                : Z === 12
                                  ? q(G.cursorBlink)
                                  : Z === 25
                                    ? q(!K.isCursorHidden)
                                    : Z === 45
                                      ? q(N.reverseWraparound)
                                      : Z === 66
                                        ? q(N.applicationKeypad)
                                        : Z === 67
                                          ? 4
                                          : Z === 1e3
                                            ? q(B === 'VT200')
                                            : Z === 1002
                                              ? q(B === 'DRAG')
                                              : Z === 1003
                                                ? q(B === 'ANY')
                                                : Z === 1004
                                                  ? q(N.sendFocus)
                                                  : Z === 1005
                                                    ? 4
                                                    : Z === 1006
                                                      ? q(z === 'SGR')
                                                      : Z === 1015
                                                        ? 4
                                                        : Z === 1016
                                                          ? q(z === 'SGR_PIXELS')
                                                          : Z === 1048
                                                            ? 1
                                                            : Z === 47 || Z === 1047 || Z === 1049
                                                              ? q(H === E)
                                                              : Z === 2004
                                                                ? q(N.bracketedPasteMode)
                                                                : 0),
                  K.triggerDataEvent(`${a.C0.ESC}[${A ? '' : '?'}${Y};${V}$y`),
                  !0
                )
                var Y, V
              }
              _updateAttrColor(C, A, N, B, z) {
                return (
                  A === 2
                    ? ((C |= 50331648),
                      (C &= -16777216),
                      (C |= d.AttributeData.fromColorRGB([N, B, z])))
                    : A === 5 && ((C &= -50331904), (C |= 33554432 | (255 & N))),
                  C
                )
              }
              _extractColor(C, A, N) {
                const B = [0, 0, -1, 0, 0, 0]
                let z = 0,
                  K = 0
                do {
                  if (((B[K + z] = C.params[A + K]), C.hasSubParams(A + K))) {
                    const J = C.getSubParams(A + K)
                    let Q = 0
                    do (B[1] === 5 && (z = 1), (B[K + Q + 1 + z] = J[Q]))
                    while (++Q < J.length && Q + K + 1 + z < B.length)
                    break
                  }
                  if ((B[1] === 5 && K + z >= 2) || (B[1] === 2 && K + z >= 5)) break
                  B[1] && (z = 1)
                } while (++K + A < C.length && K + z < B.length)
                for (let J = 2; J < B.length; ++J) B[J] === -1 && (B[J] = 0)
                switch (B[0]) {
                  case 38:
                    N.fg = this._updateAttrColor(N.fg, B[1], B[3], B[4], B[5])
                    break
                  case 48:
                    N.bg = this._updateAttrColor(N.bg, B[1], B[3], B[4], B[5])
                    break
                  case 58:
                    ;((N.extended = N.extended.clone()),
                      (N.extended.underlineColor = this._updateAttrColor(
                        N.extended.underlineColor,
                        B[1],
                        B[3],
                        B[4],
                        B[5]
                      )))
                }
                return K
              }
              _processUnderline(C, A) {
                ;((A.extended = A.extended.clone()),
                  (!~C || C > 5) && (C = 1),
                  (A.extended.underlineStyle = C),
                  (A.fg |= 268435456),
                  C === 0 && (A.fg &= -268435457),
                  A.updateExtended())
              }
              _processSGR0(C) {
                ;((C.fg = t.DEFAULT_ATTR_DATA.fg),
                  (C.bg = t.DEFAULT_ATTR_DATA.bg),
                  (C.extended = C.extended.clone()),
                  (C.extended.underlineStyle = 0),
                  (C.extended.underlineColor &= -67108864),
                  C.updateExtended())
              }
              charAttributes(C) {
                if (C.length === 1 && C.params[0] === 0)
                  return (this._processSGR0(this._curAttrData), !0)
                const A = C.length
                let N
                const B = this._curAttrData
                for (let z = 0; z < A; z++)
                  ((N = C.params[z]),
                    N >= 30 && N <= 37
                      ? ((B.fg &= -50331904), (B.fg |= 16777216 | (N - 30)))
                      : N >= 40 && N <= 47
                        ? ((B.bg &= -50331904), (B.bg |= 16777216 | (N - 40)))
                        : N >= 90 && N <= 97
                          ? ((B.fg &= -50331904), (B.fg |= 16777224 | (N - 90)))
                          : N >= 100 && N <= 107
                            ? ((B.bg &= -50331904), (B.bg |= 16777224 | (N - 100)))
                            : N === 0
                              ? this._processSGR0(B)
                              : N === 1
                                ? (B.fg |= 134217728)
                                : N === 3
                                  ? (B.bg |= 67108864)
                                  : N === 4
                                    ? ((B.fg |= 268435456),
                                      this._processUnderline(
                                        C.hasSubParams(z) ? C.getSubParams(z)[0] : 1,
                                        B
                                      ))
                                    : N === 5
                                      ? (B.fg |= 536870912)
                                      : N === 7
                                        ? (B.fg |= 67108864)
                                        : N === 8
                                          ? (B.fg |= 1073741824)
                                          : N === 9
                                            ? (B.fg |= 2147483648)
                                            : N === 2
                                              ? (B.bg |= 134217728)
                                              : N === 21
                                                ? this._processUnderline(2, B)
                                                : N === 22
                                                  ? ((B.fg &= -134217729), (B.bg &= -134217729))
                                                  : N === 23
                                                    ? (B.bg &= -67108865)
                                                    : N === 24
                                                      ? ((B.fg &= -268435457),
                                                        this._processUnderline(0, B))
                                                      : N === 25
                                                        ? (B.fg &= -536870913)
                                                        : N === 27
                                                          ? (B.fg &= -67108865)
                                                          : N === 28
                                                            ? (B.fg &= -1073741825)
                                                            : N === 29
                                                              ? (B.fg &= 2147483647)
                                                              : N === 39
                                                                ? ((B.fg &= -67108864),
                                                                  (B.fg |=
                                                                    16777215 &
                                                                    t.DEFAULT_ATTR_DATA.fg))
                                                                : N === 49
                                                                  ? ((B.bg &= -67108864),
                                                                    (B.bg |=
                                                                      16777215 &
                                                                      t.DEFAULT_ATTR_DATA.bg))
                                                                  : N === 38 || N === 48 || N === 58
                                                                    ? (z += this._extractColor(
                                                                        C,
                                                                        z,
                                                                        B
                                                                      ))
                                                                    : N === 53
                                                                      ? (B.bg |= 1073741824)
                                                                      : N === 55
                                                                        ? (B.bg &= -1073741825)
                                                                        : N === 59
                                                                          ? ((B.extended =
                                                                              B.extended.clone()),
                                                                            (B.extended.underlineColor =
                                                                              -1),
                                                                            B.updateExtended())
                                                                          : N === 100
                                                                            ? ((B.fg &= -67108864),
                                                                              (B.fg |=
                                                                                16777215 &
                                                                                t.DEFAULT_ATTR_DATA
                                                                                  .fg),
                                                                              (B.bg &= -67108864),
                                                                              (B.bg |=
                                                                                16777215 &
                                                                                t.DEFAULT_ATTR_DATA
                                                                                  .bg))
                                                                            : this._logService.debug(
                                                                                'Unknown SGR attribute: %d.',
                                                                                N
                                                                              ))
                return !0
              }
              deviceStatus(C) {
                switch (C.params[0]) {
                  case 5:
                    this._coreService.triggerDataEvent(`${a.C0.ESC}[0n`)
                    break
                  case 6:
                    const A = this._activeBuffer.y + 1,
                      N = this._activeBuffer.x + 1
                    this._coreService.triggerDataEvent(`${a.C0.ESC}[${A};${N}R`)
                }
                return !0
              }
              deviceStatusPrivate(C) {
                if (C.params[0] === 6) {
                  const A = this._activeBuffer.y + 1,
                    N = this._activeBuffer.x + 1
                  this._coreService.triggerDataEvent(`${a.C0.ESC}[?${A};${N}R`)
                }
                return !0
              }
              softReset(C) {
                return (
                  (this._coreService.isCursorHidden = !1),
                  this._onRequestSyncScrollBar.fire(),
                  (this._activeBuffer.scrollTop = 0),
                  (this._activeBuffer.scrollBottom = this._bufferService.rows - 1),
                  (this._curAttrData = t.DEFAULT_ATTR_DATA.clone()),
                  this._coreService.reset(),
                  this._charsetService.reset(),
                  (this._activeBuffer.savedX = 0),
                  (this._activeBuffer.savedY = this._activeBuffer.ybase),
                  (this._activeBuffer.savedCurAttrData.fg = this._curAttrData.fg),
                  (this._activeBuffer.savedCurAttrData.bg = this._curAttrData.bg),
                  (this._activeBuffer.savedCharset = this._charsetService.charset),
                  (this._coreService.decPrivateModes.origin = !1),
                  !0
                )
              }
              setCursorStyle(C) {
                const A = C.params[0] || 1
                switch (A) {
                  case 1:
                  case 2:
                    this._optionsService.options.cursorStyle = 'block'
                    break
                  case 3:
                  case 4:
                    this._optionsService.options.cursorStyle = 'underline'
                    break
                  case 5:
                  case 6:
                    this._optionsService.options.cursorStyle = 'bar'
                }
                const N = A % 2 == 1
                return ((this._optionsService.options.cursorBlink = N), !0)
              }
              setScrollRegion(C) {
                const A = C.params[0] || 1
                let N
                return (
                  (C.length < 2 || (N = C.params[1]) > this._bufferService.rows || N === 0) &&
                    (N = this._bufferService.rows),
                  N > A &&
                    ((this._activeBuffer.scrollTop = A - 1),
                    (this._activeBuffer.scrollBottom = N - 1),
                    this._setCursor(0, 0)),
                  !0
                )
              }
              windowOptions(C) {
                if (!P(C.params[0], this._optionsService.rawOptions.windowOptions)) return !0
                const A = C.length > 1 ? C.params[1] : 0
                switch (C.params[0]) {
                  case 14:
                    A !== 2 && this._onRequestWindowsOptionsReport.fire(j.GET_WIN_SIZE_PIXELS)
                    break
                  case 16:
                    this._onRequestWindowsOptionsReport.fire(j.GET_CELL_SIZE_PIXELS)
                    break
                  case 18:
                    this._bufferService &&
                      this._coreService.triggerDataEvent(
                        `${a.C0.ESC}[8;${this._bufferService.rows};${this._bufferService.cols}t`
                      )
                    break
                  case 22:
                    ;((A !== 0 && A !== 2) ||
                      (this._windowTitleStack.push(this._windowTitle),
                      this._windowTitleStack.length > 10 && this._windowTitleStack.shift()),
                      (A !== 0 && A !== 1) ||
                        (this._iconNameStack.push(this._iconName),
                        this._iconNameStack.length > 10 && this._iconNameStack.shift()))
                    break
                  case 23:
                    ;((A !== 0 && A !== 2) ||
                      (this._windowTitleStack.length &&
                        this.setTitle(this._windowTitleStack.pop())),
                      (A !== 0 && A !== 1) ||
                        (this._iconNameStack.length && this.setIconName(this._iconNameStack.pop())))
                }
                return !0
              }
              saveCursor(C) {
                return (
                  (this._activeBuffer.savedX = this._activeBuffer.x),
                  (this._activeBuffer.savedY = this._activeBuffer.ybase + this._activeBuffer.y),
                  (this._activeBuffer.savedCurAttrData.fg = this._curAttrData.fg),
                  (this._activeBuffer.savedCurAttrData.bg = this._curAttrData.bg),
                  (this._activeBuffer.savedCharset = this._charsetService.charset),
                  !0
                )
              }
              restoreCursor(C) {
                return (
                  (this._activeBuffer.x = this._activeBuffer.savedX || 0),
                  (this._activeBuffer.y = Math.max(
                    this._activeBuffer.savedY - this._activeBuffer.ybase,
                    0
                  )),
                  (this._curAttrData.fg = this._activeBuffer.savedCurAttrData.fg),
                  (this._curAttrData.bg = this._activeBuffer.savedCurAttrData.bg),
                  (this._charsetService.charset = this._savedCharset),
                  this._activeBuffer.savedCharset &&
                    (this._charsetService.charset = this._activeBuffer.savedCharset),
                  this._restrictCursor(),
                  !0
                )
              }
              setTitle(C) {
                return ((this._windowTitle = C), this._onTitleChange.fire(C), !0)
              }
              setIconName(C) {
                return ((this._iconName = C), !0)
              }
              setOrReportIndexedColor(C) {
                const A = [],
                  N = C.split(';')
                for (; N.length > 1; ) {
                  const B = N.shift(),
                    z = N.shift()
                  if (/^\d+$/.exec(B)) {
                    const K = parseInt(B)
                    if (F(K))
                      if (z === '?') A.push({ type: 0, index: K })
                      else {
                        const J = (0, S.parseColor)(z)
                        J && A.push({ type: 1, index: K, color: J })
                      }
                  }
                }
                return (A.length && this._onColor.fire(A), !0)
              }
              setHyperlink(C) {
                const A = C.split(';')
                return (
                  !(A.length < 2) &&
                  (A[1] ? this._createHyperlink(A[0], A[1]) : !A[0] && this._finishHyperlink())
                )
              }
              _createHyperlink(C, A) {
                this._getCurrentLinkId() && this._finishHyperlink()
                const N = C.split(':')
                let B
                const z = N.findIndex((K) => K.startsWith('id='))
                return (
                  z !== -1 && (B = N[z].slice(3) || void 0),
                  (this._curAttrData.extended = this._curAttrData.extended.clone()),
                  (this._curAttrData.extended.urlId = this._oscLinkService.registerLink({
                    id: B,
                    uri: A
                  })),
                  this._curAttrData.updateExtended(),
                  !0
                )
              }
              _finishHyperlink() {
                return (
                  (this._curAttrData.extended = this._curAttrData.extended.clone()),
                  (this._curAttrData.extended.urlId = 0),
                  this._curAttrData.updateExtended(),
                  !0
                )
              }
              _setOrReportSpecialColor(C, A) {
                const N = C.split(';')
                for (let B = 0; B < N.length && !(A >= this._specialColors.length); ++B, ++A)
                  if (N[B] === '?') this._onColor.fire([{ type: 0, index: this._specialColors[A] }])
                  else {
                    const z = (0, S.parseColor)(N[B])
                    z && this._onColor.fire([{ type: 1, index: this._specialColors[A], color: z }])
                  }
                return !0
              }
              setOrReportFgColor(C) {
                return this._setOrReportSpecialColor(C, 0)
              }
              setOrReportBgColor(C) {
                return this._setOrReportSpecialColor(C, 1)
              }
              setOrReportCursorColor(C) {
                return this._setOrReportSpecialColor(C, 2)
              }
              restoreIndexedColor(C) {
                if (!C) return (this._onColor.fire([{ type: 2 }]), !0)
                const A = [],
                  N = C.split(';')
                for (let B = 0; B < N.length; ++B)
                  if (/^\d+$/.exec(N[B])) {
                    const z = parseInt(N[B])
                    F(z) && A.push({ type: 2, index: z })
                  }
                return (A.length && this._onColor.fire(A), !0)
              }
              restoreFgColor(C) {
                return (this._onColor.fire([{ type: 2, index: 256 }]), !0)
              }
              restoreBgColor(C) {
                return (this._onColor.fire([{ type: 2, index: 257 }]), !0)
              }
              restoreCursorColor(C) {
                return (this._onColor.fire([{ type: 2, index: 258 }]), !0)
              }
              nextLine() {
                return ((this._activeBuffer.x = 0), this.index(), !0)
              }
              keypadApplicationMode() {
                return (
                  this._logService.debug('Serial port requested application keypad.'),
                  (this._coreService.decPrivateModes.applicationKeypad = !0),
                  this._onRequestSyncScrollBar.fire(),
                  !0
                )
              }
              keypadNumericMode() {
                return (
                  this._logService.debug('Switching back to normal keypad.'),
                  (this._coreService.decPrivateModes.applicationKeypad = !1),
                  this._onRequestSyncScrollBar.fire(),
                  !0
                )
              }
              selectDefaultCharset() {
                return (
                  this._charsetService.setgLevel(0),
                  this._charsetService.setgCharset(0, h.DEFAULT_CHARSET),
                  !0
                )
              }
              selectCharset(C) {
                return C.length !== 2
                  ? (this.selectDefaultCharset(), !0)
                  : (C[0] === '/' ||
                      this._charsetService.setgCharset(
                        L[C[0]],
                        h.CHARSETS[C[1]] || h.DEFAULT_CHARSET
                      ),
                    !0)
              }
              index() {
                return (
                  this._restrictCursor(),
                  this._activeBuffer.y++,
                  this._activeBuffer.y === this._activeBuffer.scrollBottom + 1
                    ? (this._activeBuffer.y--, this._bufferService.scroll(this._eraseAttrData()))
                    : this._activeBuffer.y >= this._bufferService.rows &&
                      (this._activeBuffer.y = this._bufferService.rows - 1),
                  this._restrictCursor(),
                  !0
                )
              }
              tabSet() {
                return ((this._activeBuffer.tabs[this._activeBuffer.x] = !0), !0)
              }
              reverseIndex() {
                if (
                  (this._restrictCursor(), this._activeBuffer.y === this._activeBuffer.scrollTop)
                ) {
                  const C = this._activeBuffer.scrollBottom - this._activeBuffer.scrollTop
                  ;(this._activeBuffer.lines.shiftElements(
                    this._activeBuffer.ybase + this._activeBuffer.y,
                    C,
                    1
                  ),
                    this._activeBuffer.lines.set(
                      this._activeBuffer.ybase + this._activeBuffer.y,
                      this._activeBuffer.getBlankLine(this._eraseAttrData())
                    ),
                    this._dirtyRowTracker.markRangeDirty(
                      this._activeBuffer.scrollTop,
                      this._activeBuffer.scrollBottom
                    ))
                } else (this._activeBuffer.y--, this._restrictCursor())
                return !0
              }
              fullReset() {
                return (this._parser.reset(), this._onRequestReset.fire(), !0)
              }
              reset() {
                ;((this._curAttrData = t.DEFAULT_ATTR_DATA.clone()),
                  (this._eraseAttrDataInternal = t.DEFAULT_ATTR_DATA.clone()))
              }
              _eraseAttrData() {
                return (
                  (this._eraseAttrDataInternal.bg &= -67108864),
                  (this._eraseAttrDataInternal.bg |= 67108863 & this._curAttrData.bg),
                  this._eraseAttrDataInternal
                )
              }
              setgLevel(C) {
                return (this._charsetService.setgLevel(C), !0)
              }
              screenAlignmentPattern() {
                const C = new r.CellData()
                ;((C.content = 4194373),
                  (C.fg = this._curAttrData.fg),
                  (C.bg = this._curAttrData.bg),
                  this._setCursor(0, 0))
                for (let A = 0; A < this._bufferService.rows; ++A) {
                  const N = this._activeBuffer.ybase + this._activeBuffer.y + A,
                    B = this._activeBuffer.lines.get(N)
                  B && (B.fill(C), (B.isWrapped = !1))
                }
                return (this._dirtyRowTracker.markAllDirty(), this._setCursor(0, 0), !0)
              }
              requestStatusString(C, A) {
                const N = this._bufferService.buffer,
                  B = this._optionsService.rawOptions
                return ((z) => (
                  this._coreService.triggerDataEvent(`${a.C0.ESC}${z}${a.C0.ESC}\\`),
                  !0
                ))(
                  C === '"q'
                    ? `P1$r${this._curAttrData.isProtected() ? 1 : 0}"q`
                    : C === '"p'
                      ? 'P1$r61;1"p'
                      : C === 'r'
                        ? `P1$r${N.scrollTop + 1};${N.scrollBottom + 1}r`
                        : C === 'm'
                          ? 'P1$r0m'
                          : C === ' q'
                            ? `P1$r${{ block: 2, underline: 4, bar: 6 }[B.cursorStyle] - (B.cursorBlink ? 1 : 0)} q`
                            : 'P0$r'
                )
              }
              markRangeDirty(C, A) {
                this._dirtyRowTracker.markRangeDirty(C, A)
              }
            }
            i.InputHandler = O
            let $ = class {
              constructor(W) {
                ;((this._bufferService = W), this.clearRange())
              }
              clearRange() {
                ;((this.start = this._bufferService.buffer.y),
                  (this.end = this._bufferService.buffer.y))
              }
              markDirty(W) {
                W < this.start ? (this.start = W) : W > this.end && (this.end = W)
              }
              markRangeDirty(W, C) {
                ;(W > C && ((D = W), (W = C), (C = D)),
                  W < this.start && (this.start = W),
                  C > this.end && (this.end = C))
              }
              markAllDirty() {
                this.markRangeDirty(0, this._bufferService.rows - 1)
              }
            }
            function F(W) {
              return 0 <= W && W < 256
            }
            $ = l([u(0, v.IBufferService)], $)
          },
          844: (I, i) => {
            function o(l) {
              for (const u of l) u.dispose()
              l.length = 0
            }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.getDisposeArrayDisposable =
                i.disposeArray =
                i.toDisposable =
                i.MutableDisposable =
                i.Disposable =
                  void 0),
              (i.Disposable = class {
                constructor() {
                  ;((this._disposables = []), (this._isDisposed = !1))
                }
                dispose() {
                  this._isDisposed = !0
                  for (const l of this._disposables) l.dispose()
                  this._disposables.length = 0
                }
                register(l) {
                  return (this._disposables.push(l), l)
                }
                unregister(l) {
                  const u = this._disposables.indexOf(l)
                  u !== -1 && this._disposables.splice(u, 1)
                }
              }),
              (i.MutableDisposable = class {
                constructor() {
                  this._isDisposed = !1
                }
                get value() {
                  return this._isDisposed ? void 0 : this._value
                }
                set value(l) {
                  this._isDisposed ||
                    l === this._value ||
                    (this._value?.dispose(), (this._value = l))
                }
                clear() {
                  this.value = void 0
                }
                dispose() {
                  ;((this._isDisposed = !0), this._value?.dispose(), (this._value = void 0))
                }
              }),
              (i.toDisposable = function (l) {
                return { dispose: l }
              }),
              (i.disposeArray = o),
              (i.getDisposeArrayDisposable = function (l) {
                return { dispose: () => o(l) }
              }))
          },
          1505: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.FourKeyMap = i.TwoKeyMap = void 0))
            class o {
              constructor() {
                this._data = {}
              }
              set(u, a, h) {
                ;(this._data[u] || (this._data[u] = {}), (this._data[u][a] = h))
              }
              get(u, a) {
                return this._data[u] ? this._data[u][a] : void 0
              }
              clear() {
                this._data = {}
              }
            }
            ;((i.TwoKeyMap = o),
              (i.FourKeyMap = class {
                constructor() {
                  this._data = new o()
                }
                set(l, u, a, h, f) {
                  ;(this._data.get(l, u) || this._data.set(l, u, new o()),
                    this._data.get(l, u).set(a, h, f))
                }
                get(l, u, a, h) {
                  return this._data.get(l, u)?.get(a, h)
                }
                clear() {
                  this._data.clear()
                }
              }))
          },
          6114: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.isChromeOS =
                i.isLinux =
                i.isWindows =
                i.isIphone =
                i.isIpad =
                i.isMac =
                i.getSafariVersion =
                i.isSafari =
                i.isLegacyEdge =
                i.isFirefox =
                i.isNode =
                  void 0),
              (i.isNode = typeof process < 'u' && 'title' in process))
            const o = i.isNode ? 'node' : navigator.userAgent,
              l = i.isNode ? 'node' : navigator.platform
            ;((i.isFirefox = o.includes('Firefox')),
              (i.isLegacyEdge = o.includes('Edge')),
              (i.isSafari = /^((?!chrome|android).)*safari/i.test(o)),
              (i.getSafariVersion = function () {
                if (!i.isSafari) return 0
                const u = o.match(/Version\/(\d+)/)
                return u === null || u.length < 2 ? 0 : parseInt(u[1])
              }),
              (i.isMac = ['Macintosh', 'MacIntel', 'MacPPC', 'Mac68K'].includes(l)),
              (i.isIpad = l === 'iPad'),
              (i.isIphone = l === 'iPhone'),
              (i.isWindows = ['Windows', 'Win16', 'Win32', 'WinCE'].includes(l)),
              (i.isLinux = l.indexOf('Linux') >= 0),
              (i.isChromeOS = /\bCrOS\b/.test(o)))
          },
          6106: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.SortedList = void 0))
            let o = 0
            i.SortedList = class {
              constructor(l) {
                ;((this._getKey = l), (this._array = []))
              }
              clear() {
                this._array.length = 0
              }
              insert(l) {
                this._array.length !== 0
                  ? ((o = this._search(this._getKey(l))), this._array.splice(o, 0, l))
                  : this._array.push(l)
              }
              delete(l) {
                if (this._array.length === 0) return !1
                const u = this._getKey(l)
                if (
                  u === void 0 ||
                  ((o = this._search(u)), o === -1) ||
                  this._getKey(this._array[o]) !== u
                )
                  return !1
                do if (this._array[o] === l) return (this._array.splice(o, 1), !0)
                while (++o < this._array.length && this._getKey(this._array[o]) === u)
                return !1
              }
              *getKeyIterator(l) {
                if (
                  this._array.length !== 0 &&
                  ((o = this._search(l)),
                  !(o < 0 || o >= this._array.length) && this._getKey(this._array[o]) === l)
                )
                  do yield this._array[o]
                  while (++o < this._array.length && this._getKey(this._array[o]) === l)
              }
              forEachByKey(l, u) {
                if (
                  this._array.length !== 0 &&
                  ((o = this._search(l)),
                  !(o < 0 || o >= this._array.length) && this._getKey(this._array[o]) === l)
                )
                  do u(this._array[o])
                  while (++o < this._array.length && this._getKey(this._array[o]) === l)
              }
              values() {
                return [...this._array].values()
              }
              _search(l) {
                let u = 0,
                  a = this._array.length - 1
                for (; a >= u; ) {
                  let h = (u + a) >> 1
                  const f = this._getKey(this._array[h])
                  if (f > l) a = h - 1
                  else {
                    if (!(f < l)) {
                      for (; h > 0 && this._getKey(this._array[h - 1]) === l; ) h--
                      return h
                    }
                    u = h + 1
                  }
                }
                return u
              }
            }
          },
          7226: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.DebouncedIdleTask = i.IdleTaskQueue = i.PriorityTaskQueue = void 0))
            const l = o(6114)
            class u {
              constructor() {
                ;((this._tasks = []), (this._i = 0))
              }
              enqueue(f) {
                ;(this._tasks.push(f), this._start())
              }
              flush() {
                for (; this._i < this._tasks.length; ) this._tasks[this._i]() || this._i++
                this.clear()
              }
              clear() {
                ;(this._idleCallback &&
                  (this._cancelCallback(this._idleCallback), (this._idleCallback = void 0)),
                  (this._i = 0),
                  (this._tasks.length = 0))
              }
              _start() {
                this._idleCallback ||
                  (this._idleCallback = this._requestCallback(this._process.bind(this)))
              }
              _process(f) {
                this._idleCallback = void 0
                let x = 0,
                  c = 0,
                  t = f.timeRemaining(),
                  n = 0
                for (; this._i < this._tasks.length; ) {
                  if (
                    ((x = Date.now()),
                    this._tasks[this._i]() || this._i++,
                    (x = Math.max(1, Date.now() - x)),
                    (c = Math.max(x, c)),
                    (n = f.timeRemaining()),
                    1.5 * c > n)
                  )
                    return (
                      t - x < -20 &&
                        console.warn(
                          `task queue exceeded allotted deadline by ${Math.abs(Math.round(t - x))}ms`
                        ),
                      void this._start()
                    )
                  t = n
                }
                this.clear()
              }
            }
            class a extends u {
              _requestCallback(f) {
                return setTimeout(() => f(this._createDeadline(16)))
              }
              _cancelCallback(f) {
                clearTimeout(f)
              }
              _createDeadline(f) {
                const x = Date.now() + f
                return { timeRemaining: () => Math.max(0, x - Date.now()) }
              }
            }
            ;((i.PriorityTaskQueue = a),
              (i.IdleTaskQueue =
                !l.isNode && 'requestIdleCallback' in window
                  ? class extends u {
                      _requestCallback(h) {
                        return requestIdleCallback(h)
                      }
                      _cancelCallback(h) {
                        cancelIdleCallback(h)
                      }
                    }
                  : a),
              (i.DebouncedIdleTask = class {
                constructor() {
                  this._queue = new i.IdleTaskQueue()
                }
                set(h) {
                  ;(this._queue.clear(), this._queue.enqueue(h))
                }
                flush() {
                  this._queue.flush()
                }
              }))
          },
          9282: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.updateWindowsModeWrappedState = void 0))
            const l = o(643)
            i.updateWindowsModeWrappedState = function (u) {
              const a = u.buffer.lines.get(u.buffer.ybase + u.buffer.y - 1),
                h = a?.get(u.cols - 1),
                f = u.buffer.lines.get(u.buffer.ybase + u.buffer.y)
              f &&
                h &&
                (f.isWrapped =
                  h[l.CHAR_DATA_CODE_INDEX] !== l.NULL_CELL_CODE &&
                  h[l.CHAR_DATA_CODE_INDEX] !== l.WHITESPACE_CELL_CODE)
            }
          },
          3734: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.ExtendedAttrs = i.AttributeData = void 0))
            class o {
              constructor() {
                ;((this.fg = 0), (this.bg = 0), (this.extended = new l()))
              }
              static toColorRGB(a) {
                return [(a >>> 16) & 255, (a >>> 8) & 255, 255 & a]
              }
              static fromColorRGB(a) {
                return ((255 & a[0]) << 16) | ((255 & a[1]) << 8) | (255 & a[2])
              }
              clone() {
                const a = new o()
                return ((a.fg = this.fg), (a.bg = this.bg), (a.extended = this.extended.clone()), a)
              }
              isInverse() {
                return 67108864 & this.fg
              }
              isBold() {
                return 134217728 & this.fg
              }
              isUnderline() {
                return this.hasExtendedAttrs() && this.extended.underlineStyle !== 0
                  ? 1
                  : 268435456 & this.fg
              }
              isBlink() {
                return 536870912 & this.fg
              }
              isInvisible() {
                return 1073741824 & this.fg
              }
              isItalic() {
                return 67108864 & this.bg
              }
              isDim() {
                return 134217728 & this.bg
              }
              isStrikethrough() {
                return 2147483648 & this.fg
              }
              isProtected() {
                return 536870912 & this.bg
              }
              isOverline() {
                return 1073741824 & this.bg
              }
              getFgColorMode() {
                return 50331648 & this.fg
              }
              getBgColorMode() {
                return 50331648 & this.bg
              }
              isFgRGB() {
                return (50331648 & this.fg) == 50331648
              }
              isBgRGB() {
                return (50331648 & this.bg) == 50331648
              }
              isFgPalette() {
                return (50331648 & this.fg) == 16777216 || (50331648 & this.fg) == 33554432
              }
              isBgPalette() {
                return (50331648 & this.bg) == 16777216 || (50331648 & this.bg) == 33554432
              }
              isFgDefault() {
                return (50331648 & this.fg) == 0
              }
              isBgDefault() {
                return (50331648 & this.bg) == 0
              }
              isAttributeDefault() {
                return this.fg === 0 && this.bg === 0
              }
              getFgColor() {
                switch (50331648 & this.fg) {
                  case 16777216:
                  case 33554432:
                    return 255 & this.fg
                  case 50331648:
                    return 16777215 & this.fg
                  default:
                    return -1
                }
              }
              getBgColor() {
                switch (50331648 & this.bg) {
                  case 16777216:
                  case 33554432:
                    return 255 & this.bg
                  case 50331648:
                    return 16777215 & this.bg
                  default:
                    return -1
                }
              }
              hasExtendedAttrs() {
                return 268435456 & this.bg
              }
              updateExtended() {
                this.extended.isEmpty() ? (this.bg &= -268435457) : (this.bg |= 268435456)
              }
              getUnderlineColor() {
                if (268435456 & this.bg && ~this.extended.underlineColor)
                  switch (50331648 & this.extended.underlineColor) {
                    case 16777216:
                    case 33554432:
                      return 255 & this.extended.underlineColor
                    case 50331648:
                      return 16777215 & this.extended.underlineColor
                    default:
                      return this.getFgColor()
                  }
                return this.getFgColor()
              }
              getUnderlineColorMode() {
                return 268435456 & this.bg && ~this.extended.underlineColor
                  ? 50331648 & this.extended.underlineColor
                  : this.getFgColorMode()
              }
              isUnderlineColorRGB() {
                return 268435456 & this.bg && ~this.extended.underlineColor
                  ? (50331648 & this.extended.underlineColor) == 50331648
                  : this.isFgRGB()
              }
              isUnderlineColorPalette() {
                return 268435456 & this.bg && ~this.extended.underlineColor
                  ? (50331648 & this.extended.underlineColor) == 16777216 ||
                      (50331648 & this.extended.underlineColor) == 33554432
                  : this.isFgPalette()
              }
              isUnderlineColorDefault() {
                return 268435456 & this.bg && ~this.extended.underlineColor
                  ? (50331648 & this.extended.underlineColor) == 0
                  : this.isFgDefault()
              }
              getUnderlineStyle() {
                return 268435456 & this.fg
                  ? 268435456 & this.bg
                    ? this.extended.underlineStyle
                    : 1
                  : 0
              }
              getUnderlineVariantOffset() {
                return this.extended.underlineVariantOffset
              }
            }
            i.AttributeData = o
            class l {
              get ext() {
                return this._urlId
                  ? (-469762049 & this._ext) | (this.underlineStyle << 26)
                  : this._ext
              }
              set ext(a) {
                this._ext = a
              }
              get underlineStyle() {
                return this._urlId ? 5 : (469762048 & this._ext) >> 26
              }
              set underlineStyle(a) {
                ;((this._ext &= -469762049), (this._ext |= (a << 26) & 469762048))
              }
              get underlineColor() {
                return 67108863 & this._ext
              }
              set underlineColor(a) {
                ;((this._ext &= -67108864), (this._ext |= 67108863 & a))
              }
              get urlId() {
                return this._urlId
              }
              set urlId(a) {
                this._urlId = a
              }
              get underlineVariantOffset() {
                const a = (3758096384 & this._ext) >> 29
                return a < 0 ? 4294967288 ^ a : a
              }
              set underlineVariantOffset(a) {
                ;((this._ext &= 536870911), (this._ext |= (a << 29) & 3758096384))
              }
              constructor(a = 0, h = 0) {
                ;((this._ext = 0), (this._urlId = 0), (this._ext = a), (this._urlId = h))
              }
              clone() {
                return new l(this._ext, this._urlId)
              }
              isEmpty() {
                return this.underlineStyle === 0 && this._urlId === 0
              }
            }
            i.ExtendedAttrs = l
          },
          9092: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.Buffer = i.MAX_BUFFER_SIZE = void 0))
            const l = o(6349),
              u = o(7226),
              a = o(3734),
              h = o(8437),
              f = o(4634),
              x = o(511),
              c = o(643),
              t = o(4863),
              n = o(7116)
            ;((i.MAX_BUFFER_SIZE = 4294967295),
              (i.Buffer = class {
                constructor(s, r, d) {
                  ;((this._hasScrollback = s),
                    (this._optionsService = r),
                    (this._bufferService = d),
                    (this.ydisp = 0),
                    (this.ybase = 0),
                    (this.y = 0),
                    (this.x = 0),
                    (this.tabs = {}),
                    (this.savedY = 0),
                    (this.savedX = 0),
                    (this.savedCurAttrData = h.DEFAULT_ATTR_DATA.clone()),
                    (this.savedCharset = n.DEFAULT_CHARSET),
                    (this.markers = []),
                    (this._nullCell = x.CellData.fromCharData([
                      0,
                      c.NULL_CELL_CHAR,
                      c.NULL_CELL_WIDTH,
                      c.NULL_CELL_CODE
                    ])),
                    (this._whitespaceCell = x.CellData.fromCharData([
                      0,
                      c.WHITESPACE_CELL_CHAR,
                      c.WHITESPACE_CELL_WIDTH,
                      c.WHITESPACE_CELL_CODE
                    ])),
                    (this._isClearing = !1),
                    (this._memoryCleanupQueue = new u.IdleTaskQueue()),
                    (this._memoryCleanupPosition = 0),
                    (this._cols = this._bufferService.cols),
                    (this._rows = this._bufferService.rows),
                    (this.lines = new l.CircularList(this._getCorrectBufferLength(this._rows))),
                    (this.scrollTop = 0),
                    (this.scrollBottom = this._rows - 1),
                    this.setupTabStops())
                }
                getNullCell(s) {
                  return (
                    s
                      ? ((this._nullCell.fg = s.fg),
                        (this._nullCell.bg = s.bg),
                        (this._nullCell.extended = s.extended))
                      : ((this._nullCell.fg = 0),
                        (this._nullCell.bg = 0),
                        (this._nullCell.extended = new a.ExtendedAttrs())),
                    this._nullCell
                  )
                }
                getWhitespaceCell(s) {
                  return (
                    s
                      ? ((this._whitespaceCell.fg = s.fg),
                        (this._whitespaceCell.bg = s.bg),
                        (this._whitespaceCell.extended = s.extended))
                      : ((this._whitespaceCell.fg = 0),
                        (this._whitespaceCell.bg = 0),
                        (this._whitespaceCell.extended = new a.ExtendedAttrs())),
                    this._whitespaceCell
                  )
                }
                getBlankLine(s, r) {
                  return new h.BufferLine(this._bufferService.cols, this.getNullCell(s), r)
                }
                get hasScrollback() {
                  return this._hasScrollback && this.lines.maxLength > this._rows
                }
                get isCursorInViewport() {
                  const s = this.ybase + this.y - this.ydisp
                  return s >= 0 && s < this._rows
                }
                _getCorrectBufferLength(s) {
                  if (!this._hasScrollback) return s
                  const r = s + this._optionsService.rawOptions.scrollback
                  return r > i.MAX_BUFFER_SIZE ? i.MAX_BUFFER_SIZE : r
                }
                fillViewportRows(s) {
                  if (this.lines.length === 0) {
                    s === void 0 && (s = h.DEFAULT_ATTR_DATA)
                    let r = this._rows
                    for (; r--; ) this.lines.push(this.getBlankLine(s))
                  }
                }
                clear() {
                  ;((this.ydisp = 0),
                    (this.ybase = 0),
                    (this.y = 0),
                    (this.x = 0),
                    (this.lines = new l.CircularList(this._getCorrectBufferLength(this._rows))),
                    (this.scrollTop = 0),
                    (this.scrollBottom = this._rows - 1),
                    this.setupTabStops())
                }
                resize(s, r) {
                  const d = this.getNullCell(h.DEFAULT_ATTR_DATA)
                  let v = 0
                  const _ = this._getCorrectBufferLength(r)
                  if (
                    (_ > this.lines.maxLength && (this.lines.maxLength = _), this.lines.length > 0)
                  ) {
                    if (this._cols < s)
                      for (let p = 0; p < this.lines.length; p++)
                        v += +this.lines.get(p).resize(s, d)
                    let b = 0
                    if (this._rows < r)
                      for (let p = this._rows; p < r; p++)
                        this.lines.length < r + this.ybase &&
                          (this._optionsService.rawOptions.windowsMode ||
                          this._optionsService.rawOptions.windowsPty.backend !== void 0 ||
                          this._optionsService.rawOptions.windowsPty.buildNumber !== void 0
                            ? this.lines.push(new h.BufferLine(s, d))
                            : this.ybase > 0 && this.lines.length <= this.ybase + this.y + b + 1
                              ? (this.ybase--, b++, this.ydisp > 0 && this.ydisp--)
                              : this.lines.push(new h.BufferLine(s, d)))
                    else
                      for (let p = this._rows; p > r; p--)
                        this.lines.length > r + this.ybase &&
                          (this.lines.length > this.ybase + this.y + 1
                            ? this.lines.pop()
                            : (this.ybase++, this.ydisp++))
                    if (_ < this.lines.maxLength) {
                      const p = this.lines.length - _
                      ;(p > 0 &&
                        (this.lines.trimStart(p),
                        (this.ybase = Math.max(this.ybase - p, 0)),
                        (this.ydisp = Math.max(this.ydisp - p, 0)),
                        (this.savedY = Math.max(this.savedY - p, 0))),
                        (this.lines.maxLength = _))
                    }
                    ;((this.x = Math.min(this.x, s - 1)),
                      (this.y = Math.min(this.y, r - 1)),
                      b && (this.y += b),
                      (this.savedX = Math.min(this.savedX, s - 1)),
                      (this.scrollTop = 0))
                  }
                  if (
                    ((this.scrollBottom = r - 1),
                    this._isReflowEnabled && (this._reflow(s, r), this._cols > s))
                  )
                    for (let b = 0; b < this.lines.length; b++) v += +this.lines.get(b).resize(s, d)
                  ;((this._cols = s),
                    (this._rows = r),
                    this._memoryCleanupQueue.clear(),
                    v > 0.1 * this.lines.length &&
                      ((this._memoryCleanupPosition = 0),
                      this._memoryCleanupQueue.enqueue(() => this._batchedMemoryCleanup())))
                }
                _batchedMemoryCleanup() {
                  let s = !0
                  this._memoryCleanupPosition >= this.lines.length &&
                    ((this._memoryCleanupPosition = 0), (s = !1))
                  let r = 0
                  for (; this._memoryCleanupPosition < this.lines.length; )
                    if (
                      ((r += this.lines.get(this._memoryCleanupPosition++).cleanupMemory()),
                      r > 100)
                    )
                      return !0
                  return s
                }
                get _isReflowEnabled() {
                  const s = this._optionsService.rawOptions.windowsPty
                  return s && s.buildNumber
                    ? this._hasScrollback && s.backend === 'conpty' && s.buildNumber >= 21376
                    : this._hasScrollback && !this._optionsService.rawOptions.windowsMode
                }
                _reflow(s, r) {
                  this._cols !== s &&
                    (s > this._cols ? this._reflowLarger(s, r) : this._reflowSmaller(s, r))
                }
                _reflowLarger(s, r) {
                  const d = (0, f.reflowLargerGetLinesToRemove)(
                    this.lines,
                    this._cols,
                    s,
                    this.ybase + this.y,
                    this.getNullCell(h.DEFAULT_ATTR_DATA)
                  )
                  if (d.length > 0) {
                    const v = (0, f.reflowLargerCreateNewLayout)(this.lines, d)
                    ;((0, f.reflowLargerApplyNewLayout)(this.lines, v.layout),
                      this._reflowLargerAdjustViewport(s, r, v.countRemoved))
                  }
                }
                _reflowLargerAdjustViewport(s, r, d) {
                  const v = this.getNullCell(h.DEFAULT_ATTR_DATA)
                  let _ = d
                  for (; _-- > 0; )
                    this.ybase === 0
                      ? (this.y > 0 && this.y--,
                        this.lines.length < r && this.lines.push(new h.BufferLine(s, v)))
                      : (this.ydisp === this.ybase && this.ydisp--, this.ybase--)
                  this.savedY = Math.max(this.savedY - d, 0)
                }
                _reflowSmaller(s, r) {
                  const d = this.getNullCell(h.DEFAULT_ATTR_DATA),
                    v = []
                  let _ = 0
                  for (let b = this.lines.length - 1; b >= 0; b--) {
                    let p = this.lines.get(b)
                    if (!p || (!p.isWrapped && p.getTrimmedLength() <= s)) continue
                    const S = [p]
                    for (; p.isWrapped && b > 0; ) ((p = this.lines.get(--b)), S.unshift(p))
                    const L = this.ybase + this.y
                    if (L >= b && L < b + S.length) continue
                    const M = S[S.length - 1].getTrimmedLength(),
                      P = (0, f.reflowSmallerGetNewLineLengths)(S, this._cols, s),
                      j = P.length - S.length
                    let D
                    D =
                      this.ybase === 0 && this.y !== this.lines.length - 1
                        ? Math.max(0, this.y - this.lines.maxLength + j)
                        : Math.max(0, this.lines.length - this.lines.maxLength + j)
                    const O = []
                    for (let N = 0; N < j; N++) {
                      const B = this.getBlankLine(h.DEFAULT_ATTR_DATA, !0)
                      O.push(B)
                    }
                    ;(O.length > 0 &&
                      (v.push({ start: b + S.length + _, newLines: O }), (_ += O.length)),
                      S.push(...O))
                    let $ = P.length - 1,
                      F = P[$]
                    F === 0 && ($--, (F = P[$]))
                    let W = S.length - j - 1,
                      C = M
                    for (; W >= 0; ) {
                      const N = Math.min(C, F)
                      if (S[$] === void 0) break
                      if (
                        (S[$].copyCellsFrom(S[W], C - N, F - N, N, !0),
                        (F -= N),
                        F === 0 && ($--, (F = P[$])),
                        (C -= N),
                        C === 0)
                      ) {
                        W--
                        const B = Math.max(W, 0)
                        C = (0, f.getWrappedLineTrimmedLength)(S, B, this._cols)
                      }
                    }
                    for (let N = 0; N < S.length; N++) P[N] < s && S[N].setCell(P[N], d)
                    let A = j - D
                    for (; A-- > 0; )
                      this.ybase === 0
                        ? this.y < r - 1
                          ? (this.y++, this.lines.pop())
                          : (this.ybase++, this.ydisp++)
                        : this.ybase < Math.min(this.lines.maxLength, this.lines.length + _) - r &&
                          (this.ybase === this.ydisp && this.ydisp++, this.ybase++)
                    this.savedY = Math.min(this.savedY + j, this.ybase + r - 1)
                  }
                  if (v.length > 0) {
                    const b = [],
                      p = []
                    for (let $ = 0; $ < this.lines.length; $++) p.push(this.lines.get($))
                    const S = this.lines.length
                    let L = S - 1,
                      M = 0,
                      P = v[M]
                    this.lines.length = Math.min(this.lines.maxLength, this.lines.length + _)
                    let j = 0
                    for (let $ = Math.min(this.lines.maxLength - 1, S + _ - 1); $ >= 0; $--)
                      if (P && P.start > L + j) {
                        for (let F = P.newLines.length - 1; F >= 0; F--)
                          this.lines.set($--, P.newLines[F])
                        ;($++,
                          b.push({ index: L + 1, amount: P.newLines.length }),
                          (j += P.newLines.length),
                          (P = v[++M]))
                      } else this.lines.set($, p[L--])
                    let D = 0
                    for (let $ = b.length - 1; $ >= 0; $--)
                      ((b[$].index += D), this.lines.onInsertEmitter.fire(b[$]), (D += b[$].amount))
                    const O = Math.max(0, S + _ - this.lines.maxLength)
                    O > 0 && this.lines.onTrimEmitter.fire(O)
                  }
                }
                translateBufferLineToString(s, r, d = 0, v) {
                  const _ = this.lines.get(s)
                  return _ ? _.translateToString(r, d, v) : ''
                }
                getWrappedRangeForLine(s) {
                  let r = s,
                    d = s
                  for (; r > 0 && this.lines.get(r).isWrapped; ) r--
                  for (; d + 1 < this.lines.length && this.lines.get(d + 1).isWrapped; ) d++
                  return { first: r, last: d }
                }
                setupTabStops(s) {
                  for (
                    s != null
                      ? this.tabs[s] || (s = this.prevStop(s))
                      : ((this.tabs = {}), (s = 0));
                    s < this._cols;
                    s += this._optionsService.rawOptions.tabStopWidth
                  )
                    this.tabs[s] = !0
                }
                prevStop(s) {
                  for (s == null && (s = this.x); !this.tabs[--s] && s > 0; );
                  return s >= this._cols ? this._cols - 1 : s < 0 ? 0 : s
                }
                nextStop(s) {
                  for (s == null && (s = this.x); !this.tabs[++s] && s < this._cols; );
                  return s >= this._cols ? this._cols - 1 : s < 0 ? 0 : s
                }
                clearMarkers(s) {
                  this._isClearing = !0
                  for (let r = 0; r < this.markers.length; r++)
                    this.markers[r].line === s &&
                      (this.markers[r].dispose(), this.markers.splice(r--, 1))
                  this._isClearing = !1
                }
                clearAllMarkers() {
                  this._isClearing = !0
                  for (let s = 0; s < this.markers.length; s++)
                    (this.markers[s].dispose(), this.markers.splice(s--, 1))
                  this._isClearing = !1
                }
                addMarker(s) {
                  const r = new t.Marker(s)
                  return (
                    this.markers.push(r),
                    r.register(
                      this.lines.onTrim((d) => {
                        ;((r.line -= d), r.line < 0 && r.dispose())
                      })
                    ),
                    r.register(
                      this.lines.onInsert((d) => {
                        r.line >= d.index && (r.line += d.amount)
                      })
                    ),
                    r.register(
                      this.lines.onDelete((d) => {
                        ;(r.line >= d.index && r.line < d.index + d.amount && r.dispose(),
                          r.line > d.index && (r.line -= d.amount))
                      })
                    ),
                    r.register(r.onDispose(() => this._removeMarker(r))),
                    r
                  )
                }
                _removeMarker(s) {
                  this._isClearing || this.markers.splice(this.markers.indexOf(s), 1)
                }
              }))
          },
          8437: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.BufferLine = i.DEFAULT_ATTR_DATA = void 0))
            const l = o(3734),
              u = o(511),
              a = o(643),
              h = o(482)
            i.DEFAULT_ATTR_DATA = Object.freeze(new l.AttributeData())
            let f = 0
            class x {
              constructor(t, n, s = !1) {
                ;((this.isWrapped = s),
                  (this._combined = {}),
                  (this._extendedAttrs = {}),
                  (this._data = new Uint32Array(3 * t)))
                const r =
                  n ||
                  u.CellData.fromCharData([
                    0,
                    a.NULL_CELL_CHAR,
                    a.NULL_CELL_WIDTH,
                    a.NULL_CELL_CODE
                  ])
                for (let d = 0; d < t; ++d) this.setCell(d, r)
                this.length = t
              }
              get(t) {
                const n = this._data[3 * t + 0],
                  s = 2097151 & n
                return [
                  this._data[3 * t + 1],
                  2097152 & n ? this._combined[t] : s ? (0, h.stringFromCodePoint)(s) : '',
                  n >> 22,
                  2097152 & n ? this._combined[t].charCodeAt(this._combined[t].length - 1) : s
                ]
              }
              set(t, n) {
                ;((this._data[3 * t + 1] = n[a.CHAR_DATA_ATTR_INDEX]),
                  n[a.CHAR_DATA_CHAR_INDEX].length > 1
                    ? ((this._combined[t] = n[1]),
                      (this._data[3 * t + 0] = 2097152 | t | (n[a.CHAR_DATA_WIDTH_INDEX] << 22)))
                    : (this._data[3 * t + 0] =
                        n[a.CHAR_DATA_CHAR_INDEX].charCodeAt(0) |
                        (n[a.CHAR_DATA_WIDTH_INDEX] << 22)))
              }
              getWidth(t) {
                return this._data[3 * t + 0] >> 22
              }
              hasWidth(t) {
                return 12582912 & this._data[3 * t + 0]
              }
              getFg(t) {
                return this._data[3 * t + 1]
              }
              getBg(t) {
                return this._data[3 * t + 2]
              }
              hasContent(t) {
                return 4194303 & this._data[3 * t + 0]
              }
              getCodePoint(t) {
                const n = this._data[3 * t + 0]
                return 2097152 & n
                  ? this._combined[t].charCodeAt(this._combined[t].length - 1)
                  : 2097151 & n
              }
              isCombined(t) {
                return 2097152 & this._data[3 * t + 0]
              }
              getString(t) {
                const n = this._data[3 * t + 0]
                return 2097152 & n
                  ? this._combined[t]
                  : 2097151 & n
                    ? (0, h.stringFromCodePoint)(2097151 & n)
                    : ''
              }
              isProtected(t) {
                return 536870912 & this._data[3 * t + 2]
              }
              loadCell(t, n) {
                return (
                  (f = 3 * t),
                  (n.content = this._data[f + 0]),
                  (n.fg = this._data[f + 1]),
                  (n.bg = this._data[f + 2]),
                  2097152 & n.content && (n.combinedData = this._combined[t]),
                  268435456 & n.bg && (n.extended = this._extendedAttrs[t]),
                  n
                )
              }
              setCell(t, n) {
                ;(2097152 & n.content && (this._combined[t] = n.combinedData),
                  268435456 & n.bg && (this._extendedAttrs[t] = n.extended),
                  (this._data[3 * t + 0] = n.content),
                  (this._data[3 * t + 1] = n.fg),
                  (this._data[3 * t + 2] = n.bg))
              }
              setCellFromCodepoint(t, n, s, r) {
                ;(268435456 & r.bg && (this._extendedAttrs[t] = r.extended),
                  (this._data[3 * t + 0] = n | (s << 22)),
                  (this._data[3 * t + 1] = r.fg),
                  (this._data[3 * t + 2] = r.bg))
              }
              addCodepointToCell(t, n, s) {
                let r = this._data[3 * t + 0]
                ;(2097152 & r
                  ? (this._combined[t] += (0, h.stringFromCodePoint)(n))
                  : 2097151 & r
                    ? ((this._combined[t] =
                        (0, h.stringFromCodePoint)(2097151 & r) + (0, h.stringFromCodePoint)(n)),
                      (r &= -2097152),
                      (r |= 2097152))
                    : (r = n | 4194304),
                  s && ((r &= -12582913), (r |= s << 22)),
                  (this._data[3 * t + 0] = r))
              }
              insertCells(t, n, s) {
                if (
                  ((t %= this.length) &&
                    this.getWidth(t - 1) === 2 &&
                    this.setCellFromCodepoint(t - 1, 0, 1, s),
                  n < this.length - t)
                ) {
                  const r = new u.CellData()
                  for (let d = this.length - t - n - 1; d >= 0; --d)
                    this.setCell(t + n + d, this.loadCell(t + d, r))
                  for (let d = 0; d < n; ++d) this.setCell(t + d, s)
                } else for (let r = t; r < this.length; ++r) this.setCell(r, s)
                this.getWidth(this.length - 1) === 2 &&
                  this.setCellFromCodepoint(this.length - 1, 0, 1, s)
              }
              deleteCells(t, n, s) {
                if (((t %= this.length), n < this.length - t)) {
                  const r = new u.CellData()
                  for (let d = 0; d < this.length - t - n; ++d)
                    this.setCell(t + d, this.loadCell(t + n + d, r))
                  for (let d = this.length - n; d < this.length; ++d) this.setCell(d, s)
                } else for (let r = t; r < this.length; ++r) this.setCell(r, s)
                ;(t && this.getWidth(t - 1) === 2 && this.setCellFromCodepoint(t - 1, 0, 1, s),
                  this.getWidth(t) !== 0 ||
                    this.hasContent(t) ||
                    this.setCellFromCodepoint(t, 0, 1, s))
              }
              replaceCells(t, n, s, r = !1) {
                if (r)
                  for (
                    t &&
                      this.getWidth(t - 1) === 2 &&
                      !this.isProtected(t - 1) &&
                      this.setCellFromCodepoint(t - 1, 0, 1, s),
                      n < this.length &&
                        this.getWidth(n - 1) === 2 &&
                        !this.isProtected(n) &&
                        this.setCellFromCodepoint(n, 0, 1, s);
                    t < n && t < this.length;

                  )
                    (this.isProtected(t) || this.setCell(t, s), t++)
                else
                  for (
                    t && this.getWidth(t - 1) === 2 && this.setCellFromCodepoint(t - 1, 0, 1, s),
                      n < this.length &&
                        this.getWidth(n - 1) === 2 &&
                        this.setCellFromCodepoint(n, 0, 1, s);
                    t < n && t < this.length;

                  )
                    this.setCell(t++, s)
              }
              resize(t, n) {
                if (t === this.length)
                  return 4 * this._data.length * 2 < this._data.buffer.byteLength
                const s = 3 * t
                if (t > this.length) {
                  if (this._data.buffer.byteLength >= 4 * s)
                    this._data = new Uint32Array(this._data.buffer, 0, s)
                  else {
                    const r = new Uint32Array(s)
                    ;(r.set(this._data), (this._data = r))
                  }
                  for (let r = this.length; r < t; ++r) this.setCell(r, n)
                } else {
                  this._data = this._data.subarray(0, s)
                  const r = Object.keys(this._combined)
                  for (let v = 0; v < r.length; v++) {
                    const _ = parseInt(r[v], 10)
                    _ >= t && delete this._combined[_]
                  }
                  const d = Object.keys(this._extendedAttrs)
                  for (let v = 0; v < d.length; v++) {
                    const _ = parseInt(d[v], 10)
                    _ >= t && delete this._extendedAttrs[_]
                  }
                }
                return ((this.length = t), 4 * s * 2 < this._data.buffer.byteLength)
              }
              cleanupMemory() {
                if (4 * this._data.length * 2 < this._data.buffer.byteLength) {
                  const t = new Uint32Array(this._data.length)
                  return (t.set(this._data), (this._data = t), 1)
                }
                return 0
              }
              fill(t, n = !1) {
                if (n)
                  for (let s = 0; s < this.length; ++s) this.isProtected(s) || this.setCell(s, t)
                else {
                  ;((this._combined = {}), (this._extendedAttrs = {}))
                  for (let s = 0; s < this.length; ++s) this.setCell(s, t)
                }
              }
              copyFrom(t) {
                ;(this.length !== t.length
                  ? (this._data = new Uint32Array(t._data))
                  : this._data.set(t._data),
                  (this.length = t.length),
                  (this._combined = {}))
                for (const n in t._combined) this._combined[n] = t._combined[n]
                this._extendedAttrs = {}
                for (const n in t._extendedAttrs) this._extendedAttrs[n] = t._extendedAttrs[n]
                this.isWrapped = t.isWrapped
              }
              clone() {
                const t = new x(0)
                ;((t._data = new Uint32Array(this._data)), (t.length = this.length))
                for (const n in this._combined) t._combined[n] = this._combined[n]
                for (const n in this._extendedAttrs) t._extendedAttrs[n] = this._extendedAttrs[n]
                return ((t.isWrapped = this.isWrapped), t)
              }
              getTrimmedLength() {
                for (let t = this.length - 1; t >= 0; --t)
                  if (4194303 & this._data[3 * t + 0]) return t + (this._data[3 * t + 0] >> 22)
                return 0
              }
              getNoBgTrimmedLength() {
                for (let t = this.length - 1; t >= 0; --t)
                  if (4194303 & this._data[3 * t + 0] || 50331648 & this._data[3 * t + 2])
                    return t + (this._data[3 * t + 0] >> 22)
                return 0
              }
              copyCellsFrom(t, n, s, r, d) {
                const v = t._data
                if (d)
                  for (let b = r - 1; b >= 0; b--) {
                    for (let p = 0; p < 3; p++) this._data[3 * (s + b) + p] = v[3 * (n + b) + p]
                    268435456 & v[3 * (n + b) + 2] &&
                      (this._extendedAttrs[s + b] = t._extendedAttrs[n + b])
                  }
                else
                  for (let b = 0; b < r; b++) {
                    for (let p = 0; p < 3; p++) this._data[3 * (s + b) + p] = v[3 * (n + b) + p]
                    268435456 & v[3 * (n + b) + 2] &&
                      (this._extendedAttrs[s + b] = t._extendedAttrs[n + b])
                  }
                const _ = Object.keys(t._combined)
                for (let b = 0; b < _.length; b++) {
                  const p = parseInt(_[b], 10)
                  p >= n && (this._combined[p - n + s] = t._combined[p])
                }
              }
              translateToString(t, n, s, r) {
                ;((n = n ?? 0),
                  (s = s ?? this.length),
                  t && (s = Math.min(s, this.getTrimmedLength())),
                  r && (r.length = 0))
                let d = ''
                for (; n < s; ) {
                  const v = this._data[3 * n + 0],
                    _ = 2097151 & v,
                    b =
                      2097152 & v
                        ? this._combined[n]
                        : _
                          ? (0, h.stringFromCodePoint)(_)
                          : a.WHITESPACE_CELL_CHAR
                  if (((d += b), r)) for (let p = 0; p < b.length; ++p) r.push(n)
                  n += v >> 22 || 1
                }
                return (r && r.push(n), d)
              }
            }
            i.BufferLine = x
          },
          4841: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.getRangeLength = void 0),
              (i.getRangeLength = function (o, l) {
                if (o.start.y > o.end.y)
                  throw new Error(
                    `Buffer range end (${o.end.x}, ${o.end.y}) cannot be before start (${o.start.x}, ${o.start.y})`
                  )
                return l * (o.end.y - o.start.y) + (o.end.x - o.start.x + 1)
              }))
          },
          4634: (I, i) => {
            function o(l, u, a) {
              if (u === l.length - 1) return l[u].getTrimmedLength()
              const h = !l[u].hasContent(a - 1) && l[u].getWidth(a - 1) === 1,
                f = l[u + 1].getWidth(0) === 2
              return h && f ? a - 1 : a
            }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.getWrappedLineTrimmedLength =
                i.reflowSmallerGetNewLineLengths =
                i.reflowLargerApplyNewLayout =
                i.reflowLargerCreateNewLayout =
                i.reflowLargerGetLinesToRemove =
                  void 0),
              (i.reflowLargerGetLinesToRemove = function (l, u, a, h, f) {
                const x = []
                for (let c = 0; c < l.length - 1; c++) {
                  let t = c,
                    n = l.get(++t)
                  if (!n.isWrapped) continue
                  const s = [l.get(c)]
                  for (; t < l.length && n.isWrapped; ) (s.push(n), (n = l.get(++t)))
                  if (h >= c && h < t) {
                    c += s.length - 1
                    continue
                  }
                  let r = 0,
                    d = o(s, r, u),
                    v = 1,
                    _ = 0
                  for (; v < s.length; ) {
                    const p = o(s, v, u),
                      S = p - _,
                      L = a - d,
                      M = Math.min(S, L)
                    ;(s[r].copyCellsFrom(s[v], _, d, M, !1),
                      (d += M),
                      d === a && (r++, (d = 0)),
                      (_ += M),
                      _ === p && (v++, (_ = 0)),
                      d === 0 &&
                        r !== 0 &&
                        s[r - 1].getWidth(a - 1) === 2 &&
                        (s[r].copyCellsFrom(s[r - 1], a - 1, d++, 1, !1),
                        s[r - 1].setCell(a - 1, f)))
                  }
                  s[r].replaceCells(d, a, f)
                  let b = 0
                  for (let p = s.length - 1; p > 0 && (p > r || s[p].getTrimmedLength() === 0); p--)
                    b++
                  ;(b > 0 && (x.push(c + s.length - b), x.push(b)), (c += s.length - 1))
                }
                return x
              }),
              (i.reflowLargerCreateNewLayout = function (l, u) {
                const a = []
                let h = 0,
                  f = u[h],
                  x = 0
                for (let c = 0; c < l.length; c++)
                  if (f === c) {
                    const t = u[++h]
                    ;(l.onDeleteEmitter.fire({ index: c - x, amount: t }),
                      (c += t - 1),
                      (x += t),
                      (f = u[++h]))
                  } else a.push(c)
                return { layout: a, countRemoved: x }
              }),
              (i.reflowLargerApplyNewLayout = function (l, u) {
                const a = []
                for (let h = 0; h < u.length; h++) a.push(l.get(u[h]))
                for (let h = 0; h < a.length; h++) l.set(h, a[h])
                l.length = u.length
              }),
              (i.reflowSmallerGetNewLineLengths = function (l, u, a) {
                const h = [],
                  f = l.map((n, s) => o(l, s, u)).reduce((n, s) => n + s)
                let x = 0,
                  c = 0,
                  t = 0
                for (; t < f; ) {
                  if (f - t < a) {
                    h.push(f - t)
                    break
                  }
                  x += a
                  const n = o(l, c, u)
                  x > n && ((x -= n), c++)
                  const s = l[c].getWidth(x - 1) === 2
                  s && x--
                  const r = s ? a - 1 : a
                  ;(h.push(r), (t += r))
                }
                return h
              }),
              (i.getWrappedLineTrimmedLength = o))
          },
          5295: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.BufferSet = void 0))
            const l = o(8460),
              u = o(844),
              a = o(9092)
            class h extends u.Disposable {
              constructor(x, c) {
                ;(super(),
                  (this._optionsService = x),
                  (this._bufferService = c),
                  (this._onBufferActivate = this.register(new l.EventEmitter())),
                  (this.onBufferActivate = this._onBufferActivate.event),
                  this.reset(),
                  this.register(
                    this._optionsService.onSpecificOptionChange('scrollback', () =>
                      this.resize(this._bufferService.cols, this._bufferService.rows)
                    )
                  ),
                  this.register(
                    this._optionsService.onSpecificOptionChange('tabStopWidth', () =>
                      this.setupTabStops()
                    )
                  ))
              }
              reset() {
                ;((this._normal = new a.Buffer(!0, this._optionsService, this._bufferService)),
                  this._normal.fillViewportRows(),
                  (this._alt = new a.Buffer(!1, this._optionsService, this._bufferService)),
                  (this._activeBuffer = this._normal),
                  this._onBufferActivate.fire({
                    activeBuffer: this._normal,
                    inactiveBuffer: this._alt
                  }),
                  this.setupTabStops())
              }
              get alt() {
                return this._alt
              }
              get active() {
                return this._activeBuffer
              }
              get normal() {
                return this._normal
              }
              activateNormalBuffer() {
                this._activeBuffer !== this._normal &&
                  ((this._normal.x = this._alt.x),
                  (this._normal.y = this._alt.y),
                  this._alt.clearAllMarkers(),
                  this._alt.clear(),
                  (this._activeBuffer = this._normal),
                  this._onBufferActivate.fire({
                    activeBuffer: this._normal,
                    inactiveBuffer: this._alt
                  }))
              }
              activateAltBuffer(x) {
                this._activeBuffer !== this._alt &&
                  (this._alt.fillViewportRows(x),
                  (this._alt.x = this._normal.x),
                  (this._alt.y = this._normal.y),
                  (this._activeBuffer = this._alt),
                  this._onBufferActivate.fire({
                    activeBuffer: this._alt,
                    inactiveBuffer: this._normal
                  }))
              }
              resize(x, c) {
                ;(this._normal.resize(x, c), this._alt.resize(x, c), this.setupTabStops(x))
              }
              setupTabStops(x) {
                ;(this._normal.setupTabStops(x), this._alt.setupTabStops(x))
              }
            }
            i.BufferSet = h
          },
          511: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.CellData = void 0))
            const l = o(482),
              u = o(643),
              a = o(3734)
            class h extends a.AttributeData {
              constructor() {
                ;(super(...arguments),
                  (this.content = 0),
                  (this.fg = 0),
                  (this.bg = 0),
                  (this.extended = new a.ExtendedAttrs()),
                  (this.combinedData = ''))
              }
              static fromCharData(x) {
                const c = new h()
                return (c.setFromCharData(x), c)
              }
              isCombined() {
                return 2097152 & this.content
              }
              getWidth() {
                return this.content >> 22
              }
              getChars() {
                return 2097152 & this.content
                  ? this.combinedData
                  : 2097151 & this.content
                    ? (0, l.stringFromCodePoint)(2097151 & this.content)
                    : ''
              }
              getCode() {
                return this.isCombined()
                  ? this.combinedData.charCodeAt(this.combinedData.length - 1)
                  : 2097151 & this.content
              }
              setFromCharData(x) {
                ;((this.fg = x[u.CHAR_DATA_ATTR_INDEX]), (this.bg = 0))
                let c = !1
                if (x[u.CHAR_DATA_CHAR_INDEX].length > 2) c = !0
                else if (x[u.CHAR_DATA_CHAR_INDEX].length === 2) {
                  const t = x[u.CHAR_DATA_CHAR_INDEX].charCodeAt(0)
                  if (55296 <= t && t <= 56319) {
                    const n = x[u.CHAR_DATA_CHAR_INDEX].charCodeAt(1)
                    56320 <= n && n <= 57343
                      ? (this.content =
                          (1024 * (t - 55296) + n - 56320 + 65536) |
                          (x[u.CHAR_DATA_WIDTH_INDEX] << 22))
                      : (c = !0)
                  } else c = !0
                } else
                  this.content =
                    x[u.CHAR_DATA_CHAR_INDEX].charCodeAt(0) | (x[u.CHAR_DATA_WIDTH_INDEX] << 22)
                c &&
                  ((this.combinedData = x[u.CHAR_DATA_CHAR_INDEX]),
                  (this.content = 2097152 | (x[u.CHAR_DATA_WIDTH_INDEX] << 22)))
              }
              getAsCharData() {
                return [this.fg, this.getChars(), this.getWidth(), this.getCode()]
              }
            }
            i.CellData = h
          },
          643: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.WHITESPACE_CELL_CODE =
                i.WHITESPACE_CELL_WIDTH =
                i.WHITESPACE_CELL_CHAR =
                i.NULL_CELL_CODE =
                i.NULL_CELL_WIDTH =
                i.NULL_CELL_CHAR =
                i.CHAR_DATA_CODE_INDEX =
                i.CHAR_DATA_WIDTH_INDEX =
                i.CHAR_DATA_CHAR_INDEX =
                i.CHAR_DATA_ATTR_INDEX =
                i.DEFAULT_EXT =
                i.DEFAULT_ATTR =
                i.DEFAULT_COLOR =
                  void 0),
              (i.DEFAULT_COLOR = 0),
              (i.DEFAULT_ATTR = 256 | (i.DEFAULT_COLOR << 9)),
              (i.DEFAULT_EXT = 0),
              (i.CHAR_DATA_ATTR_INDEX = 0),
              (i.CHAR_DATA_CHAR_INDEX = 1),
              (i.CHAR_DATA_WIDTH_INDEX = 2),
              (i.CHAR_DATA_CODE_INDEX = 3),
              (i.NULL_CELL_CHAR = ''),
              (i.NULL_CELL_WIDTH = 1),
              (i.NULL_CELL_CODE = 0),
              (i.WHITESPACE_CELL_CHAR = ' '),
              (i.WHITESPACE_CELL_WIDTH = 1),
              (i.WHITESPACE_CELL_CODE = 32))
          },
          4863: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.Marker = void 0))
            const l = o(8460),
              u = o(844)
            class a {
              get id() {
                return this._id
              }
              constructor(f) {
                ;((this.line = f),
                  (this.isDisposed = !1),
                  (this._disposables = []),
                  (this._id = a._nextId++),
                  (this._onDispose = this.register(new l.EventEmitter())),
                  (this.onDispose = this._onDispose.event))
              }
              dispose() {
                this.isDisposed ||
                  ((this.isDisposed = !0),
                  (this.line = -1),
                  this._onDispose.fire(),
                  (0, u.disposeArray)(this._disposables),
                  (this._disposables.length = 0))
              }
              register(f) {
                return (this._disposables.push(f), f)
              }
            }
            ;((i.Marker = a), (a._nextId = 1))
          },
          7116: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.DEFAULT_CHARSET = i.CHARSETS = void 0),
              (i.CHARSETS = {}),
              (i.DEFAULT_CHARSET = i.CHARSETS.B),
              (i.CHARSETS[0] = {
                '`': '◆',
                a: '▒',
                b: '␉',
                c: '␌',
                d: '␍',
                e: '␊',
                f: '°',
                g: '±',
                h: '␤',
                i: '␋',
                j: '┘',
                k: '┐',
                l: '┌',
                m: '└',
                n: '┼',
                o: '⎺',
                p: '⎻',
                q: '─',
                r: '⎼',
                s: '⎽',
                t: '├',
                u: '┤',
                v: '┴',
                w: '┬',
                x: '│',
                y: '≤',
                z: '≥',
                '{': 'π',
                '|': '≠',
                '}': '£',
                '~': '·'
              }),
              (i.CHARSETS.A = { '#': '£' }),
              (i.CHARSETS.B = void 0),
              (i.CHARSETS[4] = {
                '#': '£',
                '@': '¾',
                '[': 'ij',
                '\\': '½',
                ']': '|',
                '{': '¨',
                '|': 'f',
                '}': '¼',
                '~': '´'
              }),
              (i.CHARSETS.C = i.CHARSETS[5] =
                {
                  '[': 'Ä',
                  '\\': 'Ö',
                  ']': 'Å',
                  '^': 'Ü',
                  '`': 'é',
                  '{': 'ä',
                  '|': 'ö',
                  '}': 'å',
                  '~': 'ü'
                }),
              (i.CHARSETS.R = {
                '#': '£',
                '@': 'à',
                '[': '°',
                '\\': 'ç',
                ']': '§',
                '{': 'é',
                '|': 'ù',
                '}': 'è',
                '~': '¨'
              }),
              (i.CHARSETS.Q = {
                '@': 'à',
                '[': 'â',
                '\\': 'ç',
                ']': 'ê',
                '^': 'î',
                '`': 'ô',
                '{': 'é',
                '|': 'ù',
                '}': 'è',
                '~': 'û'
              }),
              (i.CHARSETS.K = {
                '@': '§',
                '[': 'Ä',
                '\\': 'Ö',
                ']': 'Ü',
                '{': 'ä',
                '|': 'ö',
                '}': 'ü',
                '~': 'ß'
              }),
              (i.CHARSETS.Y = {
                '#': '£',
                '@': '§',
                '[': '°',
                '\\': 'ç',
                ']': 'é',
                '`': 'ù',
                '{': 'à',
                '|': 'ò',
                '}': 'è',
                '~': 'ì'
              }),
              (i.CHARSETS.E = i.CHARSETS[6] =
                {
                  '@': 'Ä',
                  '[': 'Æ',
                  '\\': 'Ø',
                  ']': 'Å',
                  '^': 'Ü',
                  '`': 'ä',
                  '{': 'æ',
                  '|': 'ø',
                  '}': 'å',
                  '~': 'ü'
                }),
              (i.CHARSETS.Z = {
                '#': '£',
                '@': '§',
                '[': '¡',
                '\\': 'Ñ',
                ']': '¿',
                '{': '°',
                '|': 'ñ',
                '}': 'ç'
              }),
              (i.CHARSETS.H = i.CHARSETS[7] =
                {
                  '@': 'É',
                  '[': 'Ä',
                  '\\': 'Ö',
                  ']': 'Å',
                  '^': 'Ü',
                  '`': 'é',
                  '{': 'ä',
                  '|': 'ö',
                  '}': 'å',
                  '~': 'ü'
                }),
              (i.CHARSETS['='] = {
                '#': 'ù',
                '@': 'à',
                '[': 'é',
                '\\': 'ç',
                ']': 'ê',
                '^': 'î',
                _: 'è',
                '`': 'ô',
                '{': 'ä',
                '|': 'ö',
                '}': 'ü',
                '~': 'û'
              }))
          },
          2584: (I, i) => {
            var o, l, u
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.C1_ESCAPED = i.C1 = i.C0 = void 0),
              (function (a) {
                ;((a.NUL = '\0'),
                  (a.SOH = ''),
                  (a.STX = ''),
                  (a.ETX = ''),
                  (a.EOT = ''),
                  (a.ENQ = ''),
                  (a.ACK = ''),
                  (a.BEL = '\x07'),
                  (a.BS = '\b'),
                  (a.HT = '	'),
                  (a.LF = `
`),
                  (a.VT = '\v'),
                  (a.FF = '\f'),
                  (a.CR = '\r'),
                  (a.SO = ''),
                  (a.SI = ''),
                  (a.DLE = ''),
                  (a.DC1 = ''),
                  (a.DC2 = ''),
                  (a.DC3 = ''),
                  (a.DC4 = ''),
                  (a.NAK = ''),
                  (a.SYN = ''),
                  (a.ETB = ''),
                  (a.CAN = ''),
                  (a.EM = ''),
                  (a.SUB = ''),
                  (a.ESC = '\x1B'),
                  (a.FS = ''),
                  (a.GS = ''),
                  (a.RS = ''),
                  (a.US = ''),
                  (a.SP = ' '),
                  (a.DEL = ''))
              })(o || (i.C0 = o = {})),
              (function (a) {
                ;((a.PAD = ''),
                  (a.HOP = ''),
                  (a.BPH = ''),
                  (a.NBH = ''),
                  (a.IND = ''),
                  (a.NEL = ''),
                  (a.SSA = ''),
                  (a.ESA = ''),
                  (a.HTS = ''),
                  (a.HTJ = ''),
                  (a.VTS = ''),
                  (a.PLD = ''),
                  (a.PLU = ''),
                  (a.RI = ''),
                  (a.SS2 = ''),
                  (a.SS3 = ''),
                  (a.DCS = ''),
                  (a.PU1 = ''),
                  (a.PU2 = ''),
                  (a.STS = ''),
                  (a.CCH = ''),
                  (a.MW = ''),
                  (a.SPA = ''),
                  (a.EPA = ''),
                  (a.SOS = ''),
                  (a.SGCI = ''),
                  (a.SCI = ''),
                  (a.CSI = ''),
                  (a.ST = ''),
                  (a.OSC = ''),
                  (a.PM = ''),
                  (a.APC = ''))
              })(l || (i.C1 = l = {})),
              (function (a) {
                a.ST = `${o.ESC}\\`
              })(u || (i.C1_ESCAPED = u = {})))
          },
          7399: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.evaluateKeyboardEvent = void 0))
            const l = o(2584),
              u = {
                48: ['0', ')'],
                49: ['1', '!'],
                50: ['2', '@'],
                51: ['3', '#'],
                52: ['4', '$'],
                53: ['5', '%'],
                54: ['6', '^'],
                55: ['7', '&'],
                56: ['8', '*'],
                57: ['9', '('],
                186: [';', ':'],
                187: ['=', '+'],
                188: [',', '<'],
                189: ['-', '_'],
                190: ['.', '>'],
                191: ['/', '?'],
                192: ['`', '~'],
                219: ['[', '{'],
                220: ['\\', '|'],
                221: [']', '}'],
                222: ["'", '"']
              }
            i.evaluateKeyboardEvent = function (a, h, f, x) {
              const c = { type: 0, cancel: !1, key: void 0 },
                t =
                  (a.shiftKey ? 1 : 0) |
                  (a.altKey ? 2 : 0) |
                  (a.ctrlKey ? 4 : 0) |
                  (a.metaKey ? 8 : 0)
              switch (a.keyCode) {
                case 0:
                  a.key === 'UIKeyInputUpArrow'
                    ? (c.key = h ? l.C0.ESC + 'OA' : l.C0.ESC + '[A')
                    : a.key === 'UIKeyInputLeftArrow'
                      ? (c.key = h ? l.C0.ESC + 'OD' : l.C0.ESC + '[D')
                      : a.key === 'UIKeyInputRightArrow'
                        ? (c.key = h ? l.C0.ESC + 'OC' : l.C0.ESC + '[C')
                        : a.key === 'UIKeyInputDownArrow' &&
                          (c.key = h ? l.C0.ESC + 'OB' : l.C0.ESC + '[B')
                  break
                case 8:
                  ;((c.key = a.ctrlKey ? '\b' : l.C0.DEL), a.altKey && (c.key = l.C0.ESC + c.key))
                  break
                case 9:
                  if (a.shiftKey) {
                    c.key = l.C0.ESC + '[Z'
                    break
                  }
                  ;((c.key = l.C0.HT), (c.cancel = !0))
                  break
                case 13:
                  ;((c.key = a.altKey ? l.C0.ESC + l.C0.CR : l.C0.CR), (c.cancel = !0))
                  break
                case 27:
                  ;((c.key = l.C0.ESC), a.altKey && (c.key = l.C0.ESC + l.C0.ESC), (c.cancel = !0))
                  break
                case 37:
                  if (a.metaKey) break
                  t
                    ? ((c.key = l.C0.ESC + '[1;' + (t + 1) + 'D'),
                      c.key === l.C0.ESC + '[1;3D' && (c.key = l.C0.ESC + (f ? 'b' : '[1;5D')))
                    : (c.key = h ? l.C0.ESC + 'OD' : l.C0.ESC + '[D')
                  break
                case 39:
                  if (a.metaKey) break
                  t
                    ? ((c.key = l.C0.ESC + '[1;' + (t + 1) + 'C'),
                      c.key === l.C0.ESC + '[1;3C' && (c.key = l.C0.ESC + (f ? 'f' : '[1;5C')))
                    : (c.key = h ? l.C0.ESC + 'OC' : l.C0.ESC + '[C')
                  break
                case 38:
                  if (a.metaKey) break
                  t
                    ? ((c.key = l.C0.ESC + '[1;' + (t + 1) + 'A'),
                      f || c.key !== l.C0.ESC + '[1;3A' || (c.key = l.C0.ESC + '[1;5A'))
                    : (c.key = h ? l.C0.ESC + 'OA' : l.C0.ESC + '[A')
                  break
                case 40:
                  if (a.metaKey) break
                  t
                    ? ((c.key = l.C0.ESC + '[1;' + (t + 1) + 'B'),
                      f || c.key !== l.C0.ESC + '[1;3B' || (c.key = l.C0.ESC + '[1;5B'))
                    : (c.key = h ? l.C0.ESC + 'OB' : l.C0.ESC + '[B')
                  break
                case 45:
                  a.shiftKey || a.ctrlKey || (c.key = l.C0.ESC + '[2~')
                  break
                case 46:
                  c.key = t ? l.C0.ESC + '[3;' + (t + 1) + '~' : l.C0.ESC + '[3~'
                  break
                case 36:
                  c.key = t
                    ? l.C0.ESC + '[1;' + (t + 1) + 'H'
                    : h
                      ? l.C0.ESC + 'OH'
                      : l.C0.ESC + '[H'
                  break
                case 35:
                  c.key = t
                    ? l.C0.ESC + '[1;' + (t + 1) + 'F'
                    : h
                      ? l.C0.ESC + 'OF'
                      : l.C0.ESC + '[F'
                  break
                case 33:
                  a.shiftKey
                    ? (c.type = 2)
                    : a.ctrlKey
                      ? (c.key = l.C0.ESC + '[5;' + (t + 1) + '~')
                      : (c.key = l.C0.ESC + '[5~')
                  break
                case 34:
                  a.shiftKey
                    ? (c.type = 3)
                    : a.ctrlKey
                      ? (c.key = l.C0.ESC + '[6;' + (t + 1) + '~')
                      : (c.key = l.C0.ESC + '[6~')
                  break
                case 112:
                  c.key = t ? l.C0.ESC + '[1;' + (t + 1) + 'P' : l.C0.ESC + 'OP'
                  break
                case 113:
                  c.key = t ? l.C0.ESC + '[1;' + (t + 1) + 'Q' : l.C0.ESC + 'OQ'
                  break
                case 114:
                  c.key = t ? l.C0.ESC + '[1;' + (t + 1) + 'R' : l.C0.ESC + 'OR'
                  break
                case 115:
                  c.key = t ? l.C0.ESC + '[1;' + (t + 1) + 'S' : l.C0.ESC + 'OS'
                  break
                case 116:
                  c.key = t ? l.C0.ESC + '[15;' + (t + 1) + '~' : l.C0.ESC + '[15~'
                  break
                case 117:
                  c.key = t ? l.C0.ESC + '[17;' + (t + 1) + '~' : l.C0.ESC + '[17~'
                  break
                case 118:
                  c.key = t ? l.C0.ESC + '[18;' + (t + 1) + '~' : l.C0.ESC + '[18~'
                  break
                case 119:
                  c.key = t ? l.C0.ESC + '[19;' + (t + 1) + '~' : l.C0.ESC + '[19~'
                  break
                case 120:
                  c.key = t ? l.C0.ESC + '[20;' + (t + 1) + '~' : l.C0.ESC + '[20~'
                  break
                case 121:
                  c.key = t ? l.C0.ESC + '[21;' + (t + 1) + '~' : l.C0.ESC + '[21~'
                  break
                case 122:
                  c.key = t ? l.C0.ESC + '[23;' + (t + 1) + '~' : l.C0.ESC + '[23~'
                  break
                case 123:
                  c.key = t ? l.C0.ESC + '[24;' + (t + 1) + '~' : l.C0.ESC + '[24~'
                  break
                default:
                  if (!a.ctrlKey || a.shiftKey || a.altKey || a.metaKey)
                    if ((f && !x) || !a.altKey || a.metaKey)
                      !f || a.altKey || a.ctrlKey || a.shiftKey || !a.metaKey
                        ? a.key &&
                          !a.ctrlKey &&
                          !a.altKey &&
                          !a.metaKey &&
                          a.keyCode >= 48 &&
                          a.key.length === 1
                          ? (c.key = a.key)
                          : a.key &&
                            a.ctrlKey &&
                            (a.key === '_' && (c.key = l.C0.US),
                            a.key === '@' && (c.key = l.C0.NUL))
                        : a.keyCode === 65 && (c.type = 1)
                    else {
                      const n = u[a.keyCode],
                        s = n?.[a.shiftKey ? 1 : 0]
                      if (s) c.key = l.C0.ESC + s
                      else if (a.keyCode >= 65 && a.keyCode <= 90) {
                        const r = a.ctrlKey ? a.keyCode - 64 : a.keyCode + 32
                        let d = String.fromCharCode(r)
                        ;(a.shiftKey && (d = d.toUpperCase()), (c.key = l.C0.ESC + d))
                      } else if (a.keyCode === 32) c.key = l.C0.ESC + (a.ctrlKey ? l.C0.NUL : ' ')
                      else if (a.key === 'Dead' && a.code.startsWith('Key')) {
                        let r = a.code.slice(3, 4)
                        ;(a.shiftKey || (r = r.toLowerCase()),
                          (c.key = l.C0.ESC + r),
                          (c.cancel = !0))
                      }
                    }
                  else
                    a.keyCode >= 65 && a.keyCode <= 90
                      ? (c.key = String.fromCharCode(a.keyCode - 64))
                      : a.keyCode === 32
                        ? (c.key = l.C0.NUL)
                        : a.keyCode >= 51 && a.keyCode <= 55
                          ? (c.key = String.fromCharCode(a.keyCode - 51 + 27))
                          : a.keyCode === 56
                            ? (c.key = l.C0.DEL)
                            : a.keyCode === 219
                              ? (c.key = l.C0.ESC)
                              : a.keyCode === 220
                                ? (c.key = l.C0.FS)
                                : a.keyCode === 221 && (c.key = l.C0.GS)
              }
              return c
            }
          },
          482: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.Utf8ToUtf32 = i.StringToUtf32 = i.utf32ToString = i.stringFromCodePoint = void 0),
              (i.stringFromCodePoint = function (o) {
                return o > 65535
                  ? ((o -= 65536),
                    String.fromCharCode(55296 + (o >> 10)) +
                      String.fromCharCode((o % 1024) + 56320))
                  : String.fromCharCode(o)
              }),
              (i.utf32ToString = function (o, l = 0, u = o.length) {
                let a = ''
                for (let h = l; h < u; ++h) {
                  let f = o[h]
                  f > 65535
                    ? ((f -= 65536),
                      (a +=
                        String.fromCharCode(55296 + (f >> 10)) +
                        String.fromCharCode((f % 1024) + 56320)))
                    : (a += String.fromCharCode(f))
                }
                return a
              }),
              (i.StringToUtf32 = class {
                constructor() {
                  this._interim = 0
                }
                clear() {
                  this._interim = 0
                }
                decode(o, l) {
                  const u = o.length
                  if (!u) return 0
                  let a = 0,
                    h = 0
                  if (this._interim) {
                    const f = o.charCodeAt(h++)
                    ;(56320 <= f && f <= 57343
                      ? (l[a++] = 1024 * (this._interim - 55296) + f - 56320 + 65536)
                      : ((l[a++] = this._interim), (l[a++] = f)),
                      (this._interim = 0))
                  }
                  for (let f = h; f < u; ++f) {
                    const x = o.charCodeAt(f)
                    if (55296 <= x && x <= 56319) {
                      if (++f >= u) return ((this._interim = x), a)
                      const c = o.charCodeAt(f)
                      56320 <= c && c <= 57343
                        ? (l[a++] = 1024 * (x - 55296) + c - 56320 + 65536)
                        : ((l[a++] = x), (l[a++] = c))
                    } else x !== 65279 && (l[a++] = x)
                  }
                  return a
                }
              }),
              (i.Utf8ToUtf32 = class {
                constructor() {
                  this.interim = new Uint8Array(3)
                }
                clear() {
                  this.interim.fill(0)
                }
                decode(o, l) {
                  const u = o.length
                  if (!u) return 0
                  let a,
                    h,
                    f,
                    x,
                    c = 0,
                    t = 0,
                    n = 0
                  if (this.interim[0]) {
                    let d = !1,
                      v = this.interim[0]
                    v &= (224 & v) == 192 ? 31 : (240 & v) == 224 ? 15 : 7
                    let _,
                      b = 0
                    for (; (_ = 63 & this.interim[++b]) && b < 4; ) ((v <<= 6), (v |= _))
                    const p =
                        (224 & this.interim[0]) == 192 ? 2 : (240 & this.interim[0]) == 224 ? 3 : 4,
                      S = p - b
                    for (; n < S; ) {
                      if (n >= u) return 0
                      if (((_ = o[n++]), (192 & _) != 128)) {
                        ;(n--, (d = !0))
                        break
                      }
                      ;((this.interim[b++] = _), (v <<= 6), (v |= 63 & _))
                    }
                    ;(d ||
                      (p === 2
                        ? v < 128
                          ? n--
                          : (l[c++] = v)
                        : p === 3
                          ? v < 2048 || (v >= 55296 && v <= 57343) || v === 65279 || (l[c++] = v)
                          : v < 65536 || v > 1114111 || (l[c++] = v)),
                      this.interim.fill(0))
                  }
                  const s = u - 4
                  let r = n
                  for (; r < u; ) {
                    for (
                      ;
                      !(
                        !(r < s) ||
                        128 & (a = o[r]) ||
                        128 & (h = o[r + 1]) ||
                        128 & (f = o[r + 2]) ||
                        128 & (x = o[r + 3])
                      );

                    )
                      ((l[c++] = a), (l[c++] = h), (l[c++] = f), (l[c++] = x), (r += 4))
                    if (((a = o[r++]), a < 128)) l[c++] = a
                    else if ((224 & a) == 192) {
                      if (r >= u) return ((this.interim[0] = a), c)
                      if (((h = o[r++]), (192 & h) != 128)) {
                        r--
                        continue
                      }
                      if (((t = ((31 & a) << 6) | (63 & h)), t < 128)) {
                        r--
                        continue
                      }
                      l[c++] = t
                    } else if ((240 & a) == 224) {
                      if (r >= u) return ((this.interim[0] = a), c)
                      if (((h = o[r++]), (192 & h) != 128)) {
                        r--
                        continue
                      }
                      if (r >= u) return ((this.interim[0] = a), (this.interim[1] = h), c)
                      if (((f = o[r++]), (192 & f) != 128)) {
                        r--
                        continue
                      }
                      if (
                        ((t = ((15 & a) << 12) | ((63 & h) << 6) | (63 & f)),
                        t < 2048 || (t >= 55296 && t <= 57343) || t === 65279)
                      )
                        continue
                      l[c++] = t
                    } else if ((248 & a) == 240) {
                      if (r >= u) return ((this.interim[0] = a), c)
                      if (((h = o[r++]), (192 & h) != 128)) {
                        r--
                        continue
                      }
                      if (r >= u) return ((this.interim[0] = a), (this.interim[1] = h), c)
                      if (((f = o[r++]), (192 & f) != 128)) {
                        r--
                        continue
                      }
                      if (r >= u)
                        return (
                          (this.interim[0] = a),
                          (this.interim[1] = h),
                          (this.interim[2] = f),
                          c
                        )
                      if (((x = o[r++]), (192 & x) != 128)) {
                        r--
                        continue
                      }
                      if (
                        ((t = ((7 & a) << 18) | ((63 & h) << 12) | ((63 & f) << 6) | (63 & x)),
                        t < 65536 || t > 1114111)
                      )
                        continue
                      l[c++] = t
                    }
                  }
                  return c
                }
              }))
          },
          225: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.UnicodeV6 = void 0))
            const l = o(1480),
              u = [
                [768, 879],
                [1155, 1158],
                [1160, 1161],
                [1425, 1469],
                [1471, 1471],
                [1473, 1474],
                [1476, 1477],
                [1479, 1479],
                [1536, 1539],
                [1552, 1557],
                [1611, 1630],
                [1648, 1648],
                [1750, 1764],
                [1767, 1768],
                [1770, 1773],
                [1807, 1807],
                [1809, 1809],
                [1840, 1866],
                [1958, 1968],
                [2027, 2035],
                [2305, 2306],
                [2364, 2364],
                [2369, 2376],
                [2381, 2381],
                [2385, 2388],
                [2402, 2403],
                [2433, 2433],
                [2492, 2492],
                [2497, 2500],
                [2509, 2509],
                [2530, 2531],
                [2561, 2562],
                [2620, 2620],
                [2625, 2626],
                [2631, 2632],
                [2635, 2637],
                [2672, 2673],
                [2689, 2690],
                [2748, 2748],
                [2753, 2757],
                [2759, 2760],
                [2765, 2765],
                [2786, 2787],
                [2817, 2817],
                [2876, 2876],
                [2879, 2879],
                [2881, 2883],
                [2893, 2893],
                [2902, 2902],
                [2946, 2946],
                [3008, 3008],
                [3021, 3021],
                [3134, 3136],
                [3142, 3144],
                [3146, 3149],
                [3157, 3158],
                [3260, 3260],
                [3263, 3263],
                [3270, 3270],
                [3276, 3277],
                [3298, 3299],
                [3393, 3395],
                [3405, 3405],
                [3530, 3530],
                [3538, 3540],
                [3542, 3542],
                [3633, 3633],
                [3636, 3642],
                [3655, 3662],
                [3761, 3761],
                [3764, 3769],
                [3771, 3772],
                [3784, 3789],
                [3864, 3865],
                [3893, 3893],
                [3895, 3895],
                [3897, 3897],
                [3953, 3966],
                [3968, 3972],
                [3974, 3975],
                [3984, 3991],
                [3993, 4028],
                [4038, 4038],
                [4141, 4144],
                [4146, 4146],
                [4150, 4151],
                [4153, 4153],
                [4184, 4185],
                [4448, 4607],
                [4959, 4959],
                [5906, 5908],
                [5938, 5940],
                [5970, 5971],
                [6002, 6003],
                [6068, 6069],
                [6071, 6077],
                [6086, 6086],
                [6089, 6099],
                [6109, 6109],
                [6155, 6157],
                [6313, 6313],
                [6432, 6434],
                [6439, 6440],
                [6450, 6450],
                [6457, 6459],
                [6679, 6680],
                [6912, 6915],
                [6964, 6964],
                [6966, 6970],
                [6972, 6972],
                [6978, 6978],
                [7019, 7027],
                [7616, 7626],
                [7678, 7679],
                [8203, 8207],
                [8234, 8238],
                [8288, 8291],
                [8298, 8303],
                [8400, 8431],
                [12330, 12335],
                [12441, 12442],
                [43014, 43014],
                [43019, 43019],
                [43045, 43046],
                [64286, 64286],
                [65024, 65039],
                [65056, 65059],
                [65279, 65279],
                [65529, 65531]
              ],
              a = [
                [68097, 68099],
                [68101, 68102],
                [68108, 68111],
                [68152, 68154],
                [68159, 68159],
                [119143, 119145],
                [119155, 119170],
                [119173, 119179],
                [119210, 119213],
                [119362, 119364],
                [917505, 917505],
                [917536, 917631],
                [917760, 917999]
              ]
            let h
            i.UnicodeV6 = class {
              constructor() {
                if (((this.version = '6'), !h)) {
                  ;((h = new Uint8Array(65536)),
                    h.fill(1),
                    (h[0] = 0),
                    h.fill(0, 1, 32),
                    h.fill(0, 127, 160),
                    h.fill(2, 4352, 4448),
                    (h[9001] = 2),
                    (h[9002] = 2),
                    h.fill(2, 11904, 42192),
                    (h[12351] = 1),
                    h.fill(2, 44032, 55204),
                    h.fill(2, 63744, 64256),
                    h.fill(2, 65040, 65050),
                    h.fill(2, 65072, 65136),
                    h.fill(2, 65280, 65377),
                    h.fill(2, 65504, 65511))
                  for (let f = 0; f < u.length; ++f) h.fill(0, u[f][0], u[f][1] + 1)
                }
              }
              wcwidth(f) {
                return f < 32
                  ? 0
                  : f < 127
                    ? 1
                    : f < 65536
                      ? h[f]
                      : (function (x, c) {
                            let t,
                              n = 0,
                              s = c.length - 1
                            if (x < c[0][0] || x > c[s][1]) return !1
                            for (; s >= n; )
                              if (((t = (n + s) >> 1), x > c[t][1])) n = t + 1
                              else {
                                if (!(x < c[t][0])) return !0
                                s = t - 1
                              }
                            return !1
                          })(f, a)
                        ? 0
                        : (f >= 131072 && f <= 196605) || (f >= 196608 && f <= 262141)
                          ? 2
                          : 1
              }
              charProperties(f, x) {
                let c = this.wcwidth(f),
                  t = c === 0 && x !== 0
                if (t) {
                  const n = l.UnicodeService.extractWidth(x)
                  n === 0 ? (t = !1) : n > c && (c = n)
                }
                return l.UnicodeService.createPropertyValue(0, c, t)
              }
            }
          },
          5981: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.WriteBuffer = void 0))
            const l = o(8460),
              u = o(844)
            class a extends u.Disposable {
              constructor(f) {
                ;(super(),
                  (this._action = f),
                  (this._writeBuffer = []),
                  (this._callbacks = []),
                  (this._pendingData = 0),
                  (this._bufferOffset = 0),
                  (this._isSyncWriting = !1),
                  (this._syncCalls = 0),
                  (this._didUserInput = !1),
                  (this._onWriteParsed = this.register(new l.EventEmitter())),
                  (this.onWriteParsed = this._onWriteParsed.event))
              }
              handleUserInput() {
                this._didUserInput = !0
              }
              writeSync(f, x) {
                if (x !== void 0 && this._syncCalls > x) return void (this._syncCalls = 0)
                if (
                  ((this._pendingData += f.length),
                  this._writeBuffer.push(f),
                  this._callbacks.push(void 0),
                  this._syncCalls++,
                  this._isSyncWriting)
                )
                  return
                let c
                for (this._isSyncWriting = !0; (c = this._writeBuffer.shift()); ) {
                  this._action(c)
                  const t = this._callbacks.shift()
                  t && t()
                }
                ;((this._pendingData = 0),
                  (this._bufferOffset = 2147483647),
                  (this._isSyncWriting = !1),
                  (this._syncCalls = 0))
              }
              write(f, x) {
                if (this._pendingData > 5e7)
                  throw new Error('write data discarded, use flow control to avoid losing data')
                if (!this._writeBuffer.length) {
                  if (((this._bufferOffset = 0), this._didUserInput))
                    return (
                      (this._didUserInput = !1),
                      (this._pendingData += f.length),
                      this._writeBuffer.push(f),
                      this._callbacks.push(x),
                      void this._innerWrite()
                    )
                  setTimeout(() => this._innerWrite())
                }
                ;((this._pendingData += f.length),
                  this._writeBuffer.push(f),
                  this._callbacks.push(x))
              }
              _innerWrite(f = 0, x = !0) {
                const c = f || Date.now()
                for (; this._writeBuffer.length > this._bufferOffset; ) {
                  const t = this._writeBuffer[this._bufferOffset],
                    n = this._action(t, x)
                  if (n) {
                    const r = (d) =>
                      Date.now() - c >= 12
                        ? setTimeout(() => this._innerWrite(0, d))
                        : this._innerWrite(c, d)
                    return void n
                      .catch(
                        (d) => (
                          queueMicrotask(() => {
                            throw d
                          }),
                          Promise.resolve(!1)
                        )
                      )
                      .then(r)
                  }
                  const s = this._callbacks[this._bufferOffset]
                  if (
                    (s && s(),
                    this._bufferOffset++,
                    (this._pendingData -= t.length),
                    Date.now() - c >= 12)
                  )
                    break
                }
                ;(this._writeBuffer.length > this._bufferOffset
                  ? (this._bufferOffset > 50 &&
                      ((this._writeBuffer = this._writeBuffer.slice(this._bufferOffset)),
                      (this._callbacks = this._callbacks.slice(this._bufferOffset)),
                      (this._bufferOffset = 0)),
                    setTimeout(() => this._innerWrite()))
                  : ((this._writeBuffer.length = 0),
                    (this._callbacks.length = 0),
                    (this._pendingData = 0),
                    (this._bufferOffset = 0)),
                  this._onWriteParsed.fire())
              }
            }
            i.WriteBuffer = a
          },
          5941: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.toRgbString = i.parseColor = void 0))
            const o =
                /^([\da-f])\/([\da-f])\/([\da-f])$|^([\da-f]{2})\/([\da-f]{2})\/([\da-f]{2})$|^([\da-f]{3})\/([\da-f]{3})\/([\da-f]{3})$|^([\da-f]{4})\/([\da-f]{4})\/([\da-f]{4})$/,
              l = /^[\da-f]+$/
            function u(a, h) {
              const f = a.toString(16),
                x = f.length < 2 ? '0' + f : f
              switch (h) {
                case 4:
                  return f[0]
                case 8:
                  return x
                case 12:
                  return (x + x).slice(0, 3)
                default:
                  return x + x
              }
            }
            ;((i.parseColor = function (a) {
              if (!a) return
              let h = a.toLowerCase()
              if (h.indexOf('rgb:') === 0) {
                h = h.slice(4)
                const f = o.exec(h)
                if (f) {
                  const x = f[1] ? 15 : f[4] ? 255 : f[7] ? 4095 : 65535
                  return [
                    Math.round((parseInt(f[1] || f[4] || f[7] || f[10], 16) / x) * 255),
                    Math.round((parseInt(f[2] || f[5] || f[8] || f[11], 16) / x) * 255),
                    Math.round((parseInt(f[3] || f[6] || f[9] || f[12], 16) / x) * 255)
                  ]
                }
              } else if (
                h.indexOf('#') === 0 &&
                ((h = h.slice(1)), l.exec(h) && [3, 6, 9, 12].includes(h.length))
              ) {
                const f = h.length / 3,
                  x = [0, 0, 0]
                for (let c = 0; c < 3; ++c) {
                  const t = parseInt(h.slice(f * c, f * c + f), 16)
                  x[c] = f === 1 ? t << 4 : f === 2 ? t : f === 3 ? t >> 4 : t >> 8
                }
                return x
              }
            }),
              (i.toRgbString = function (a, h = 16) {
                const [f, x, c] = a
                return `rgb:${u(f, h)}/${u(x, h)}/${u(c, h)}`
              }))
          },
          5770: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.PAYLOAD_LIMIT = void 0),
              (i.PAYLOAD_LIMIT = 1e7))
          },
          6351: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.DcsHandler = i.DcsParser = void 0))
            const l = o(482),
              u = o(8742),
              a = o(5770),
              h = []
            i.DcsParser = class {
              constructor() {
                ;((this._handlers = Object.create(null)),
                  (this._active = h),
                  (this._ident = 0),
                  (this._handlerFb = () => {}),
                  (this._stack = { paused: !1, loopPosition: 0, fallThrough: !1 }))
              }
              dispose() {
                ;((this._handlers = Object.create(null)),
                  (this._handlerFb = () => {}),
                  (this._active = h))
              }
              registerHandler(x, c) {
                this._handlers[x] === void 0 && (this._handlers[x] = [])
                const t = this._handlers[x]
                return (
                  t.push(c),
                  {
                    dispose: () => {
                      const n = t.indexOf(c)
                      n !== -1 && t.splice(n, 1)
                    }
                  }
                )
              }
              clearHandler(x) {
                this._handlers[x] && delete this._handlers[x]
              }
              setHandlerFallback(x) {
                this._handlerFb = x
              }
              reset() {
                if (this._active.length)
                  for (
                    let x = this._stack.paused
                      ? this._stack.loopPosition - 1
                      : this._active.length - 1;
                    x >= 0;
                    --x
                  )
                    this._active[x].unhook(!1)
                ;((this._stack.paused = !1), (this._active = h), (this._ident = 0))
              }
              hook(x, c) {
                if (
                  (this.reset(),
                  (this._ident = x),
                  (this._active = this._handlers[x] || h),
                  this._active.length)
                )
                  for (let t = this._active.length - 1; t >= 0; t--) this._active[t].hook(c)
                else this._handlerFb(this._ident, 'HOOK', c)
              }
              put(x, c, t) {
                if (this._active.length)
                  for (let n = this._active.length - 1; n >= 0; n--) this._active[n].put(x, c, t)
                else this._handlerFb(this._ident, 'PUT', (0, l.utf32ToString)(x, c, t))
              }
              unhook(x, c = !0) {
                if (this._active.length) {
                  let t = !1,
                    n = this._active.length - 1,
                    s = !1
                  if (
                    (this._stack.paused &&
                      ((n = this._stack.loopPosition - 1),
                      (t = c),
                      (s = this._stack.fallThrough),
                      (this._stack.paused = !1)),
                    !s && t === !1)
                  ) {
                    for (; n >= 0 && ((t = this._active[n].unhook(x)), t !== !0); n--)
                      if (t instanceof Promise)
                        return (
                          (this._stack.paused = !0),
                          (this._stack.loopPosition = n),
                          (this._stack.fallThrough = !1),
                          t
                        )
                    n--
                  }
                  for (; n >= 0; n--)
                    if (((t = this._active[n].unhook(!1)), t instanceof Promise))
                      return (
                        (this._stack.paused = !0),
                        (this._stack.loopPosition = n),
                        (this._stack.fallThrough = !0),
                        t
                      )
                } else this._handlerFb(this._ident, 'UNHOOK', x)
                ;((this._active = h), (this._ident = 0))
              }
            }
            const f = new u.Params()
            ;(f.addParam(0),
              (i.DcsHandler = class {
                constructor(x) {
                  ;((this._handler = x),
                    (this._data = ''),
                    (this._params = f),
                    (this._hitLimit = !1))
                }
                hook(x) {
                  ;((this._params = x.length > 1 || x.params[0] ? x.clone() : f),
                    (this._data = ''),
                    (this._hitLimit = !1))
                }
                put(x, c, t) {
                  this._hitLimit ||
                    ((this._data += (0, l.utf32ToString)(x, c, t)),
                    this._data.length > a.PAYLOAD_LIMIT &&
                      ((this._data = ''), (this._hitLimit = !0)))
                }
                unhook(x) {
                  let c = !1
                  if (this._hitLimit) c = !1
                  else if (
                    x &&
                    ((c = this._handler(this._data, this._params)), c instanceof Promise)
                  )
                    return c.then(
                      (t) => ((this._params = f), (this._data = ''), (this._hitLimit = !1), t)
                    )
                  return ((this._params = f), (this._data = ''), (this._hitLimit = !1), c)
                }
              }))
          },
          2015: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.EscapeSequenceParser = i.VT500_TRANSITION_TABLE = i.TransitionTable = void 0))
            const l = o(844),
              u = o(8742),
              a = o(6242),
              h = o(6351)
            class f {
              constructor(n) {
                this.table = new Uint8Array(n)
              }
              setDefault(n, s) {
                this.table.fill((n << 4) | s)
              }
              add(n, s, r, d) {
                this.table[(s << 8) | n] = (r << 4) | d
              }
              addMany(n, s, r, d) {
                for (let v = 0; v < n.length; v++) this.table[(s << 8) | n[v]] = (r << 4) | d
              }
            }
            i.TransitionTable = f
            const x = 160
            i.VT500_TRANSITION_TABLE = (function () {
              const t = new f(4095),
                n = Array.apply(null, Array(256)).map((b, p) => p),
                s = (b, p) => n.slice(b, p),
                r = s(32, 127),
                d = s(0, 24)
              ;(d.push(25), d.push.apply(d, s(28, 32)))
              const v = s(0, 14)
              let _
              for (_ in (t.setDefault(1, 0), t.addMany(r, 0, 2, 0), v))
                (t.addMany([24, 26, 153, 154], _, 3, 0),
                  t.addMany(s(128, 144), _, 3, 0),
                  t.addMany(s(144, 152), _, 3, 0),
                  t.add(156, _, 0, 0),
                  t.add(27, _, 11, 1),
                  t.add(157, _, 4, 8),
                  t.addMany([152, 158, 159], _, 0, 7),
                  t.add(155, _, 11, 3),
                  t.add(144, _, 11, 9))
              return (
                t.addMany(d, 0, 3, 0),
                t.addMany(d, 1, 3, 1),
                t.add(127, 1, 0, 1),
                t.addMany(d, 8, 0, 8),
                t.addMany(d, 3, 3, 3),
                t.add(127, 3, 0, 3),
                t.addMany(d, 4, 3, 4),
                t.add(127, 4, 0, 4),
                t.addMany(d, 6, 3, 6),
                t.addMany(d, 5, 3, 5),
                t.add(127, 5, 0, 5),
                t.addMany(d, 2, 3, 2),
                t.add(127, 2, 0, 2),
                t.add(93, 1, 4, 8),
                t.addMany(r, 8, 5, 8),
                t.add(127, 8, 5, 8),
                t.addMany([156, 27, 24, 26, 7], 8, 6, 0),
                t.addMany(s(28, 32), 8, 0, 8),
                t.addMany([88, 94, 95], 1, 0, 7),
                t.addMany(r, 7, 0, 7),
                t.addMany(d, 7, 0, 7),
                t.add(156, 7, 0, 0),
                t.add(127, 7, 0, 7),
                t.add(91, 1, 11, 3),
                t.addMany(s(64, 127), 3, 7, 0),
                t.addMany(s(48, 60), 3, 8, 4),
                t.addMany([60, 61, 62, 63], 3, 9, 4),
                t.addMany(s(48, 60), 4, 8, 4),
                t.addMany(s(64, 127), 4, 7, 0),
                t.addMany([60, 61, 62, 63], 4, 0, 6),
                t.addMany(s(32, 64), 6, 0, 6),
                t.add(127, 6, 0, 6),
                t.addMany(s(64, 127), 6, 0, 0),
                t.addMany(s(32, 48), 3, 9, 5),
                t.addMany(s(32, 48), 5, 9, 5),
                t.addMany(s(48, 64), 5, 0, 6),
                t.addMany(s(64, 127), 5, 7, 0),
                t.addMany(s(32, 48), 4, 9, 5),
                t.addMany(s(32, 48), 1, 9, 2),
                t.addMany(s(32, 48), 2, 9, 2),
                t.addMany(s(48, 127), 2, 10, 0),
                t.addMany(s(48, 80), 1, 10, 0),
                t.addMany(s(81, 88), 1, 10, 0),
                t.addMany([89, 90, 92], 1, 10, 0),
                t.addMany(s(96, 127), 1, 10, 0),
                t.add(80, 1, 11, 9),
                t.addMany(d, 9, 0, 9),
                t.add(127, 9, 0, 9),
                t.addMany(s(28, 32), 9, 0, 9),
                t.addMany(s(32, 48), 9, 9, 12),
                t.addMany(s(48, 60), 9, 8, 10),
                t.addMany([60, 61, 62, 63], 9, 9, 10),
                t.addMany(d, 11, 0, 11),
                t.addMany(s(32, 128), 11, 0, 11),
                t.addMany(s(28, 32), 11, 0, 11),
                t.addMany(d, 10, 0, 10),
                t.add(127, 10, 0, 10),
                t.addMany(s(28, 32), 10, 0, 10),
                t.addMany(s(48, 60), 10, 8, 10),
                t.addMany([60, 61, 62, 63], 10, 0, 11),
                t.addMany(s(32, 48), 10, 9, 12),
                t.addMany(d, 12, 0, 12),
                t.add(127, 12, 0, 12),
                t.addMany(s(28, 32), 12, 0, 12),
                t.addMany(s(32, 48), 12, 9, 12),
                t.addMany(s(48, 64), 12, 0, 11),
                t.addMany(s(64, 127), 12, 12, 13),
                t.addMany(s(64, 127), 10, 12, 13),
                t.addMany(s(64, 127), 9, 12, 13),
                t.addMany(d, 13, 13, 13),
                t.addMany(r, 13, 13, 13),
                t.add(127, 13, 0, 13),
                t.addMany([27, 156, 24, 26], 13, 14, 0),
                t.add(x, 0, 2, 0),
                t.add(x, 8, 5, 8),
                t.add(x, 6, 0, 6),
                t.add(x, 11, 0, 11),
                t.add(x, 13, 13, 13),
                t
              )
            })()
            class c extends l.Disposable {
              constructor(n = i.VT500_TRANSITION_TABLE) {
                ;(super(),
                  (this._transitions = n),
                  (this._parseStack = {
                    state: 0,
                    handlers: [],
                    handlerPos: 0,
                    transition: 0,
                    chunkPos: 0
                  }),
                  (this.initialState = 0),
                  (this.currentState = this.initialState),
                  (this._params = new u.Params()),
                  this._params.addParam(0),
                  (this._collect = 0),
                  (this.precedingJoinState = 0),
                  (this._printHandlerFb = (s, r, d) => {}),
                  (this._executeHandlerFb = (s) => {}),
                  (this._csiHandlerFb = (s, r) => {}),
                  (this._escHandlerFb = (s) => {}),
                  (this._errorHandlerFb = (s) => s),
                  (this._printHandler = this._printHandlerFb),
                  (this._executeHandlers = Object.create(null)),
                  (this._csiHandlers = Object.create(null)),
                  (this._escHandlers = Object.create(null)),
                  this.register(
                    (0, l.toDisposable)(() => {
                      ;((this._csiHandlers = Object.create(null)),
                        (this._executeHandlers = Object.create(null)),
                        (this._escHandlers = Object.create(null)))
                    })
                  ),
                  (this._oscParser = this.register(new a.OscParser())),
                  (this._dcsParser = this.register(new h.DcsParser())),
                  (this._errorHandler = this._errorHandlerFb),
                  this.registerEscHandler({ final: '\\' }, () => !0))
              }
              _identifier(n, s = [64, 126]) {
                let r = 0
                if (n.prefix) {
                  if (n.prefix.length > 1) throw new Error('only one byte as prefix supported')
                  if (((r = n.prefix.charCodeAt(0)), (r && 60 > r) || r > 63))
                    throw new Error('prefix must be in range 0x3c .. 0x3f')
                }
                if (n.intermediates) {
                  if (n.intermediates.length > 2)
                    throw new Error('only two bytes as intermediates are supported')
                  for (let v = 0; v < n.intermediates.length; ++v) {
                    const _ = n.intermediates.charCodeAt(v)
                    if (32 > _ || _ > 47)
                      throw new Error('intermediate must be in range 0x20 .. 0x2f')
                    ;((r <<= 8), (r |= _))
                  }
                }
                if (n.final.length !== 1) throw new Error('final must be a single byte')
                const d = n.final.charCodeAt(0)
                if (s[0] > d || d > s[1])
                  throw new Error(`final must be in range ${s[0]} .. ${s[1]}`)
                return ((r <<= 8), (r |= d), r)
              }
              identToString(n) {
                const s = []
                for (; n; ) (s.push(String.fromCharCode(255 & n)), (n >>= 8))
                return s.reverse().join('')
              }
              setPrintHandler(n) {
                this._printHandler = n
              }
              clearPrintHandler() {
                this._printHandler = this._printHandlerFb
              }
              registerEscHandler(n, s) {
                const r = this._identifier(n, [48, 126])
                this._escHandlers[r] === void 0 && (this._escHandlers[r] = [])
                const d = this._escHandlers[r]
                return (
                  d.push(s),
                  {
                    dispose: () => {
                      const v = d.indexOf(s)
                      v !== -1 && d.splice(v, 1)
                    }
                  }
                )
              }
              clearEscHandler(n) {
                this._escHandlers[this._identifier(n, [48, 126])] &&
                  delete this._escHandlers[this._identifier(n, [48, 126])]
              }
              setEscHandlerFallback(n) {
                this._escHandlerFb = n
              }
              setExecuteHandler(n, s) {
                this._executeHandlers[n.charCodeAt(0)] = s
              }
              clearExecuteHandler(n) {
                this._executeHandlers[n.charCodeAt(0)] &&
                  delete this._executeHandlers[n.charCodeAt(0)]
              }
              setExecuteHandlerFallback(n) {
                this._executeHandlerFb = n
              }
              registerCsiHandler(n, s) {
                const r = this._identifier(n)
                this._csiHandlers[r] === void 0 && (this._csiHandlers[r] = [])
                const d = this._csiHandlers[r]
                return (
                  d.push(s),
                  {
                    dispose: () => {
                      const v = d.indexOf(s)
                      v !== -1 && d.splice(v, 1)
                    }
                  }
                )
              }
              clearCsiHandler(n) {
                this._csiHandlers[this._identifier(n)] &&
                  delete this._csiHandlers[this._identifier(n)]
              }
              setCsiHandlerFallback(n) {
                this._csiHandlerFb = n
              }
              registerDcsHandler(n, s) {
                return this._dcsParser.registerHandler(this._identifier(n), s)
              }
              clearDcsHandler(n) {
                this._dcsParser.clearHandler(this._identifier(n))
              }
              setDcsHandlerFallback(n) {
                this._dcsParser.setHandlerFallback(n)
              }
              registerOscHandler(n, s) {
                return this._oscParser.registerHandler(n, s)
              }
              clearOscHandler(n) {
                this._oscParser.clearHandler(n)
              }
              setOscHandlerFallback(n) {
                this._oscParser.setHandlerFallback(n)
              }
              setErrorHandler(n) {
                this._errorHandler = n
              }
              clearErrorHandler() {
                this._errorHandler = this._errorHandlerFb
              }
              reset() {
                ;((this.currentState = this.initialState),
                  this._oscParser.reset(),
                  this._dcsParser.reset(),
                  this._params.reset(),
                  this._params.addParam(0),
                  (this._collect = 0),
                  (this.precedingJoinState = 0),
                  this._parseStack.state !== 0 &&
                    ((this._parseStack.state = 2), (this._parseStack.handlers = [])))
              }
              _preserveStack(n, s, r, d, v) {
                ;((this._parseStack.state = n),
                  (this._parseStack.handlers = s),
                  (this._parseStack.handlerPos = r),
                  (this._parseStack.transition = d),
                  (this._parseStack.chunkPos = v))
              }
              parse(n, s, r) {
                let d,
                  v = 0,
                  _ = 0,
                  b = 0
                if (this._parseStack.state)
                  if (this._parseStack.state === 2)
                    ((this._parseStack.state = 0), (b = this._parseStack.chunkPos + 1))
                  else {
                    if (r === void 0 || this._parseStack.state === 1)
                      throw (
                        (this._parseStack.state = 1),
                        new Error(
                          'improper continuation due to previous async handler, giving up parsing'
                        )
                      )
                    const p = this._parseStack.handlers
                    let S = this._parseStack.handlerPos - 1
                    switch (this._parseStack.state) {
                      case 3:
                        if (r === !1 && S > -1) {
                          for (; S >= 0 && ((d = p[S](this._params)), d !== !0); S--)
                            if (d instanceof Promise) return ((this._parseStack.handlerPos = S), d)
                        }
                        this._parseStack.handlers = []
                        break
                      case 4:
                        if (r === !1 && S > -1) {
                          for (; S >= 0 && ((d = p[S]()), d !== !0); S--)
                            if (d instanceof Promise) return ((this._parseStack.handlerPos = S), d)
                        }
                        this._parseStack.handlers = []
                        break
                      case 6:
                        if (
                          ((v = n[this._parseStack.chunkPos]),
                          (d = this._dcsParser.unhook(v !== 24 && v !== 26, r)),
                          d)
                        )
                          return d
                        ;(v === 27 && (this._parseStack.transition |= 1),
                          this._params.reset(),
                          this._params.addParam(0),
                          (this._collect = 0))
                        break
                      case 5:
                        if (
                          ((v = n[this._parseStack.chunkPos]),
                          (d = this._oscParser.end(v !== 24 && v !== 26, r)),
                          d)
                        )
                          return d
                        ;(v === 27 && (this._parseStack.transition |= 1),
                          this._params.reset(),
                          this._params.addParam(0),
                          (this._collect = 0))
                    }
                    ;((this._parseStack.state = 0),
                      (b = this._parseStack.chunkPos + 1),
                      (this.precedingJoinState = 0),
                      (this.currentState = 15 & this._parseStack.transition))
                  }
                for (let p = b; p < s; ++p) {
                  switch (
                    ((v = n[p]),
                    (_ = this._transitions.table[(this.currentState << 8) | (v < 160 ? v : x)]),
                    _ >> 4)
                  ) {
                    case 2:
                      for (let j = p + 1; ; ++j) {
                        if (j >= s || (v = n[j]) < 32 || (v > 126 && v < x)) {
                          ;(this._printHandler(n, p, j), (p = j - 1))
                          break
                        }
                        if (++j >= s || (v = n[j]) < 32 || (v > 126 && v < x)) {
                          ;(this._printHandler(n, p, j), (p = j - 1))
                          break
                        }
                        if (++j >= s || (v = n[j]) < 32 || (v > 126 && v < x)) {
                          ;(this._printHandler(n, p, j), (p = j - 1))
                          break
                        }
                        if (++j >= s || (v = n[j]) < 32 || (v > 126 && v < x)) {
                          ;(this._printHandler(n, p, j), (p = j - 1))
                          break
                        }
                      }
                      break
                    case 3:
                      ;(this._executeHandlers[v]
                        ? this._executeHandlers[v]()
                        : this._executeHandlerFb(v),
                        (this.precedingJoinState = 0))
                      break
                    case 0:
                      break
                    case 1:
                      if (
                        this._errorHandler({
                          position: p,
                          code: v,
                          currentState: this.currentState,
                          collect: this._collect,
                          params: this._params,
                          abort: !1
                        }).abort
                      )
                        return
                      break
                    case 7:
                      const S = this._csiHandlers[(this._collect << 8) | v]
                      let L = S ? S.length - 1 : -1
                      for (; L >= 0 && ((d = S[L](this._params)), d !== !0); L--)
                        if (d instanceof Promise) return (this._preserveStack(3, S, L, _, p), d)
                      ;(L < 0 && this._csiHandlerFb((this._collect << 8) | v, this._params),
                        (this.precedingJoinState = 0))
                      break
                    case 8:
                      do
                        switch (v) {
                          case 59:
                            this._params.addParam(0)
                            break
                          case 58:
                            this._params.addSubParam(-1)
                            break
                          default:
                            this._params.addDigit(v - 48)
                        }
                      while (++p < s && (v = n[p]) > 47 && v < 60)
                      p--
                      break
                    case 9:
                      ;((this._collect <<= 8), (this._collect |= v))
                      break
                    case 10:
                      const M = this._escHandlers[(this._collect << 8) | v]
                      let P = M ? M.length - 1 : -1
                      for (; P >= 0 && ((d = M[P]()), d !== !0); P--)
                        if (d instanceof Promise) return (this._preserveStack(4, M, P, _, p), d)
                      ;(P < 0 && this._escHandlerFb((this._collect << 8) | v),
                        (this.precedingJoinState = 0))
                      break
                    case 11:
                      ;(this._params.reset(), this._params.addParam(0), (this._collect = 0))
                      break
                    case 12:
                      this._dcsParser.hook((this._collect << 8) | v, this._params)
                      break
                    case 13:
                      for (let j = p + 1; ; ++j)
                        if (
                          j >= s ||
                          (v = n[j]) === 24 ||
                          v === 26 ||
                          v === 27 ||
                          (v > 127 && v < x)
                        ) {
                          ;(this._dcsParser.put(n, p, j), (p = j - 1))
                          break
                        }
                      break
                    case 14:
                      if (((d = this._dcsParser.unhook(v !== 24 && v !== 26)), d))
                        return (this._preserveStack(6, [], 0, _, p), d)
                      ;(v === 27 && (_ |= 1),
                        this._params.reset(),
                        this._params.addParam(0),
                        (this._collect = 0),
                        (this.precedingJoinState = 0))
                      break
                    case 4:
                      this._oscParser.start()
                      break
                    case 5:
                      for (let j = p + 1; ; j++)
                        if (j >= s || (v = n[j]) < 32 || (v > 127 && v < x)) {
                          ;(this._oscParser.put(n, p, j), (p = j - 1))
                          break
                        }
                      break
                    case 6:
                      if (((d = this._oscParser.end(v !== 24 && v !== 26)), d))
                        return (this._preserveStack(5, [], 0, _, p), d)
                      ;(v === 27 && (_ |= 1),
                        this._params.reset(),
                        this._params.addParam(0),
                        (this._collect = 0),
                        (this.precedingJoinState = 0))
                  }
                  this.currentState = 15 & _
                }
              }
            }
            i.EscapeSequenceParser = c
          },
          6242: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.OscHandler = i.OscParser = void 0))
            const l = o(5770),
              u = o(482),
              a = []
            ;((i.OscParser = class {
              constructor() {
                ;((this._state = 0),
                  (this._active = a),
                  (this._id = -1),
                  (this._handlers = Object.create(null)),
                  (this._handlerFb = () => {}),
                  (this._stack = { paused: !1, loopPosition: 0, fallThrough: !1 }))
              }
              registerHandler(h, f) {
                this._handlers[h] === void 0 && (this._handlers[h] = [])
                const x = this._handlers[h]
                return (
                  x.push(f),
                  {
                    dispose: () => {
                      const c = x.indexOf(f)
                      c !== -1 && x.splice(c, 1)
                    }
                  }
                )
              }
              clearHandler(h) {
                this._handlers[h] && delete this._handlers[h]
              }
              setHandlerFallback(h) {
                this._handlerFb = h
              }
              dispose() {
                ;((this._handlers = Object.create(null)),
                  (this._handlerFb = () => {}),
                  (this._active = a))
              }
              reset() {
                if (this._state === 2)
                  for (
                    let h = this._stack.paused
                      ? this._stack.loopPosition - 1
                      : this._active.length - 1;
                    h >= 0;
                    --h
                  )
                    this._active[h].end(!1)
                ;((this._stack.paused = !1), (this._active = a), (this._id = -1), (this._state = 0))
              }
              _start() {
                if (((this._active = this._handlers[this._id] || a), this._active.length))
                  for (let h = this._active.length - 1; h >= 0; h--) this._active[h].start()
                else this._handlerFb(this._id, 'START')
              }
              _put(h, f, x) {
                if (this._active.length)
                  for (let c = this._active.length - 1; c >= 0; c--) this._active[c].put(h, f, x)
                else this._handlerFb(this._id, 'PUT', (0, u.utf32ToString)(h, f, x))
              }
              start() {
                ;(this.reset(), (this._state = 1))
              }
              put(h, f, x) {
                if (this._state !== 3) {
                  if (this._state === 1)
                    for (; f < x; ) {
                      const c = h[f++]
                      if (c === 59) {
                        ;((this._state = 2), this._start())
                        break
                      }
                      if (c < 48 || 57 < c) return void (this._state = 3)
                      ;(this._id === -1 && (this._id = 0), (this._id = 10 * this._id + c - 48))
                    }
                  this._state === 2 && x - f > 0 && this._put(h, f, x)
                }
              }
              end(h, f = !0) {
                if (this._state !== 0) {
                  if (this._state !== 3)
                    if ((this._state === 1 && this._start(), this._active.length)) {
                      let x = !1,
                        c = this._active.length - 1,
                        t = !1
                      if (
                        (this._stack.paused &&
                          ((c = this._stack.loopPosition - 1),
                          (x = f),
                          (t = this._stack.fallThrough),
                          (this._stack.paused = !1)),
                        !t && x === !1)
                      ) {
                        for (; c >= 0 && ((x = this._active[c].end(h)), x !== !0); c--)
                          if (x instanceof Promise)
                            return (
                              (this._stack.paused = !0),
                              (this._stack.loopPosition = c),
                              (this._stack.fallThrough = !1),
                              x
                            )
                        c--
                      }
                      for (; c >= 0; c--)
                        if (((x = this._active[c].end(!1)), x instanceof Promise))
                          return (
                            (this._stack.paused = !0),
                            (this._stack.loopPosition = c),
                            (this._stack.fallThrough = !0),
                            x
                          )
                    } else this._handlerFb(this._id, 'END', h)
                  ;((this._active = a), (this._id = -1), (this._state = 0))
                }
              }
            }),
              (i.OscHandler = class {
                constructor(h) {
                  ;((this._handler = h), (this._data = ''), (this._hitLimit = !1))
                }
                start() {
                  ;((this._data = ''), (this._hitLimit = !1))
                }
                put(h, f, x) {
                  this._hitLimit ||
                    ((this._data += (0, u.utf32ToString)(h, f, x)),
                    this._data.length > l.PAYLOAD_LIMIT &&
                      ((this._data = ''), (this._hitLimit = !0)))
                }
                end(h) {
                  let f = !1
                  if (this._hitLimit) f = !1
                  else if (h && ((f = this._handler(this._data)), f instanceof Promise))
                    return f.then((x) => ((this._data = ''), (this._hitLimit = !1), x))
                  return ((this._data = ''), (this._hitLimit = !1), f)
                }
              }))
          },
          8742: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.Params = void 0))
            const o = 2147483647
            class l {
              static fromArray(a) {
                const h = new l()
                if (!a.length) return h
                for (let f = Array.isArray(a[0]) ? 1 : 0; f < a.length; ++f) {
                  const x = a[f]
                  if (Array.isArray(x)) for (let c = 0; c < x.length; ++c) h.addSubParam(x[c])
                  else h.addParam(x)
                }
                return h
              }
              constructor(a = 32, h = 32) {
                if (((this.maxLength = a), (this.maxSubParamsLength = h), h > 256))
                  throw new Error('maxSubParamsLength must not be greater than 256')
                ;((this.params = new Int32Array(a)),
                  (this.length = 0),
                  (this._subParams = new Int32Array(h)),
                  (this._subParamsLength = 0),
                  (this._subParamsIdx = new Uint16Array(a)),
                  (this._rejectDigits = !1),
                  (this._rejectSubDigits = !1),
                  (this._digitIsSub = !1))
              }
              clone() {
                const a = new l(this.maxLength, this.maxSubParamsLength)
                return (
                  a.params.set(this.params),
                  (a.length = this.length),
                  a._subParams.set(this._subParams),
                  (a._subParamsLength = this._subParamsLength),
                  a._subParamsIdx.set(this._subParamsIdx),
                  (a._rejectDigits = this._rejectDigits),
                  (a._rejectSubDigits = this._rejectSubDigits),
                  (a._digitIsSub = this._digitIsSub),
                  a
                )
              }
              toArray() {
                const a = []
                for (let h = 0; h < this.length; ++h) {
                  a.push(this.params[h])
                  const f = this._subParamsIdx[h] >> 8,
                    x = 255 & this._subParamsIdx[h]
                  x - f > 0 && a.push(Array.prototype.slice.call(this._subParams, f, x))
                }
                return a
              }
              reset() {
                ;((this.length = 0),
                  (this._subParamsLength = 0),
                  (this._rejectDigits = !1),
                  (this._rejectSubDigits = !1),
                  (this._digitIsSub = !1))
              }
              addParam(a) {
                if (((this._digitIsSub = !1), this.length >= this.maxLength))
                  this._rejectDigits = !0
                else {
                  if (a < -1) throw new Error('values lesser than -1 are not allowed')
                  ;((this._subParamsIdx[this.length] =
                    (this._subParamsLength << 8) | this._subParamsLength),
                    (this.params[this.length++] = a > o ? o : a))
                }
              }
              addSubParam(a) {
                if (((this._digitIsSub = !0), this.length))
                  if (this._rejectDigits || this._subParamsLength >= this.maxSubParamsLength)
                    this._rejectSubDigits = !0
                  else {
                    if (a < -1) throw new Error('values lesser than -1 are not allowed')
                    ;((this._subParams[this._subParamsLength++] = a > o ? o : a),
                      this._subParamsIdx[this.length - 1]++)
                  }
              }
              hasSubParams(a) {
                return (255 & this._subParamsIdx[a]) - (this._subParamsIdx[a] >> 8) > 0
              }
              getSubParams(a) {
                const h = this._subParamsIdx[a] >> 8,
                  f = 255 & this._subParamsIdx[a]
                return f - h > 0 ? this._subParams.subarray(h, f) : null
              }
              getSubParamsAll() {
                const a = {}
                for (let h = 0; h < this.length; ++h) {
                  const f = this._subParamsIdx[h] >> 8,
                    x = 255 & this._subParamsIdx[h]
                  x - f > 0 && (a[h] = this._subParams.slice(f, x))
                }
                return a
              }
              addDigit(a) {
                let h
                if (
                  this._rejectDigits ||
                  !(h = this._digitIsSub ? this._subParamsLength : this.length) ||
                  (this._digitIsSub && this._rejectSubDigits)
                )
                  return
                const f = this._digitIsSub ? this._subParams : this.params,
                  x = f[h - 1]
                f[h - 1] = ~x ? Math.min(10 * x + a, o) : a
              }
            }
            i.Params = l
          },
          5741: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.AddonManager = void 0),
              (i.AddonManager = class {
                constructor() {
                  this._addons = []
                }
                dispose() {
                  for (let o = this._addons.length - 1; o >= 0; o--)
                    this._addons[o].instance.dispose()
                }
                loadAddon(o, l) {
                  const u = { instance: l, dispose: l.dispose, isDisposed: !1 }
                  ;(this._addons.push(u),
                    (l.dispose = () => this._wrappedAddonDispose(u)),
                    l.activate(o))
                }
                _wrappedAddonDispose(o) {
                  if (o.isDisposed) return
                  let l = -1
                  for (let u = 0; u < this._addons.length; u++)
                    if (this._addons[u] === o) {
                      l = u
                      break
                    }
                  if (l === -1)
                    throw new Error('Could not dispose an addon that has not been loaded')
                  ;((o.isDisposed = !0), o.dispose.apply(o.instance), this._addons.splice(l, 1))
                }
              }))
          },
          8771: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.BufferApiView = void 0))
            const l = o(3785),
              u = o(511)
            i.BufferApiView = class {
              constructor(a, h) {
                ;((this._buffer = a), (this.type = h))
              }
              init(a) {
                return ((this._buffer = a), this)
              }
              get cursorY() {
                return this._buffer.y
              }
              get cursorX() {
                return this._buffer.x
              }
              get viewportY() {
                return this._buffer.ydisp
              }
              get baseY() {
                return this._buffer.ybase
              }
              get length() {
                return this._buffer.lines.length
              }
              getLine(a) {
                const h = this._buffer.lines.get(a)
                if (h) return new l.BufferLineApiView(h)
              }
              getNullCell() {
                return new u.CellData()
              }
            }
          },
          3785: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.BufferLineApiView = void 0))
            const l = o(511)
            i.BufferLineApiView = class {
              constructor(u) {
                this._line = u
              }
              get isWrapped() {
                return this._line.isWrapped
              }
              get length() {
                return this._line.length
              }
              getCell(u, a) {
                if (!(u < 0 || u >= this._line.length))
                  return a
                    ? (this._line.loadCell(u, a), a)
                    : this._line.loadCell(u, new l.CellData())
              }
              translateToString(u, a, h) {
                return this._line.translateToString(u, a, h)
              }
            }
          },
          8285: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.BufferNamespaceApi = void 0))
            const l = o(8771),
              u = o(8460),
              a = o(844)
            class h extends a.Disposable {
              constructor(x) {
                ;(super(),
                  (this._core = x),
                  (this._onBufferChange = this.register(new u.EventEmitter())),
                  (this.onBufferChange = this._onBufferChange.event),
                  (this._normal = new l.BufferApiView(this._core.buffers.normal, 'normal')),
                  (this._alternate = new l.BufferApiView(this._core.buffers.alt, 'alternate')),
                  this._core.buffers.onBufferActivate(() => this._onBufferChange.fire(this.active)))
              }
              get active() {
                if (this._core.buffers.active === this._core.buffers.normal) return this.normal
                if (this._core.buffers.active === this._core.buffers.alt) return this.alternate
                throw new Error('Active buffer is neither normal nor alternate')
              }
              get normal() {
                return this._normal.init(this._core.buffers.normal)
              }
              get alternate() {
                return this._alternate.init(this._core.buffers.alt)
              }
            }
            i.BufferNamespaceApi = h
          },
          7975: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.ParserApi = void 0),
              (i.ParserApi = class {
                constructor(o) {
                  this._core = o
                }
                registerCsiHandler(o, l) {
                  return this._core.registerCsiHandler(o, (u) => l(u.toArray()))
                }
                addCsiHandler(o, l) {
                  return this.registerCsiHandler(o, l)
                }
                registerDcsHandler(o, l) {
                  return this._core.registerDcsHandler(o, (u, a) => l(u, a.toArray()))
                }
                addDcsHandler(o, l) {
                  return this.registerDcsHandler(o, l)
                }
                registerEscHandler(o, l) {
                  return this._core.registerEscHandler(o, l)
                }
                addEscHandler(o, l) {
                  return this.registerEscHandler(o, l)
                }
                registerOscHandler(o, l) {
                  return this._core.registerOscHandler(o, l)
                }
                addOscHandler(o, l) {
                  return this.registerOscHandler(o, l)
                }
              }))
          },
          7090: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.UnicodeApi = void 0),
              (i.UnicodeApi = class {
                constructor(o) {
                  this._core = o
                }
                register(o) {
                  this._core.unicodeService.register(o)
                }
                get versions() {
                  return this._core.unicodeService.versions
                }
                get activeVersion() {
                  return this._core.unicodeService.activeVersion
                }
                set activeVersion(o) {
                  this._core.unicodeService.activeVersion = o
                }
              }))
          },
          744: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (t, n, s, r) {
                  var d,
                    v = arguments.length,
                    _ = v < 3 ? n : r === null ? (r = Object.getOwnPropertyDescriptor(n, s)) : r
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    _ = Reflect.decorate(t, n, s, r)
                  else
                    for (var b = t.length - 1; b >= 0; b--)
                      (d = t[b]) && (_ = (v < 3 ? d(_) : v > 3 ? d(n, s, _) : d(n, s)) || _)
                  return (v > 3 && _ && Object.defineProperty(n, s, _), _)
                },
              u =
                (this && this.__param) ||
                function (t, n) {
                  return function (s, r) {
                    n(s, r, t)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.BufferService = i.MINIMUM_ROWS = i.MINIMUM_COLS = void 0))
            const a = o(8460),
              h = o(844),
              f = o(5295),
              x = o(2585)
            ;((i.MINIMUM_COLS = 2), (i.MINIMUM_ROWS = 1))
            let c = (i.BufferService = class extends h.Disposable {
              get buffer() {
                return this.buffers.active
              }
              constructor(t) {
                ;(super(),
                  (this.isUserScrolling = !1),
                  (this._onResize = this.register(new a.EventEmitter())),
                  (this.onResize = this._onResize.event),
                  (this._onScroll = this.register(new a.EventEmitter())),
                  (this.onScroll = this._onScroll.event),
                  (this.cols = Math.max(t.rawOptions.cols || 0, i.MINIMUM_COLS)),
                  (this.rows = Math.max(t.rawOptions.rows || 0, i.MINIMUM_ROWS)),
                  (this.buffers = this.register(new f.BufferSet(t, this))))
              }
              resize(t, n) {
                ;((this.cols = t),
                  (this.rows = n),
                  this.buffers.resize(t, n),
                  this._onResize.fire({ cols: t, rows: n }))
              }
              reset() {
                ;(this.buffers.reset(), (this.isUserScrolling = !1))
              }
              scroll(t, n = !1) {
                const s = this.buffer
                let r
                ;((r = this._cachedBlankLine),
                  (r && r.length === this.cols && r.getFg(0) === t.fg && r.getBg(0) === t.bg) ||
                    ((r = s.getBlankLine(t, n)), (this._cachedBlankLine = r)),
                  (r.isWrapped = n))
                const d = s.ybase + s.scrollTop,
                  v = s.ybase + s.scrollBottom
                if (s.scrollTop === 0) {
                  const _ = s.lines.isFull
                  ;(v === s.lines.length - 1
                    ? _
                      ? s.lines.recycle().copyFrom(r)
                      : s.lines.push(r.clone())
                    : s.lines.splice(v + 1, 0, r.clone()),
                    _
                      ? this.isUserScrolling && (s.ydisp = Math.max(s.ydisp - 1, 0))
                      : (s.ybase++, this.isUserScrolling || s.ydisp++))
                } else {
                  const _ = v - d + 1
                  ;(s.lines.shiftElements(d + 1, _ - 1, -1), s.lines.set(v, r.clone()))
                }
                ;(this.isUserScrolling || (s.ydisp = s.ybase), this._onScroll.fire(s.ydisp))
              }
              scrollLines(t, n, s) {
                const r = this.buffer
                if (t < 0) {
                  if (r.ydisp === 0) return
                  this.isUserScrolling = !0
                } else t + r.ydisp >= r.ybase && (this.isUserScrolling = !1)
                const d = r.ydisp
                ;((r.ydisp = Math.max(Math.min(r.ydisp + t, r.ybase), 0)),
                  d !== r.ydisp && (n || this._onScroll.fire(r.ydisp)))
              }
            })
            i.BufferService = c = l([u(0, x.IOptionsService)], c)
          },
          7994: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.CharsetService = void 0),
              (i.CharsetService = class {
                constructor() {
                  ;((this.glevel = 0), (this._charsets = []))
                }
                reset() {
                  ;((this.charset = void 0), (this._charsets = []), (this.glevel = 0))
                }
                setgLevel(o) {
                  ;((this.glevel = o), (this.charset = this._charsets[o]))
                }
                setgCharset(o, l) {
                  ;((this._charsets[o] = l), this.glevel === o && (this.charset = l))
                }
              }))
          },
          1753: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (r, d, v, _) {
                  var b,
                    p = arguments.length,
                    S = p < 3 ? d : _ === null ? (_ = Object.getOwnPropertyDescriptor(d, v)) : _
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    S = Reflect.decorate(r, d, v, _)
                  else
                    for (var L = r.length - 1; L >= 0; L--)
                      (b = r[L]) && (S = (p < 3 ? b(S) : p > 3 ? b(d, v, S) : b(d, v)) || S)
                  return (p > 3 && S && Object.defineProperty(d, v, S), S)
                },
              u =
                (this && this.__param) ||
                function (r, d) {
                  return function (v, _) {
                    d(v, _, r)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.CoreMouseService = void 0))
            const a = o(2585),
              h = o(8460),
              f = o(844),
              x = {
                NONE: { events: 0, restrict: () => !1 },
                X10: {
                  events: 1,
                  restrict: (r) =>
                    r.button !== 4 &&
                    r.action === 1 &&
                    ((r.ctrl = !1), (r.alt = !1), (r.shift = !1), !0)
                },
                VT200: { events: 19, restrict: (r) => r.action !== 32 },
                DRAG: { events: 23, restrict: (r) => r.action !== 32 || r.button !== 3 },
                ANY: { events: 31, restrict: (r) => !0 }
              }
            function c(r, d) {
              let v = (r.ctrl ? 16 : 0) | (r.shift ? 4 : 0) | (r.alt ? 8 : 0)
              return (
                r.button === 4
                  ? ((v |= 64), (v |= r.action))
                  : ((v |= 3 & r.button),
                    4 & r.button && (v |= 64),
                    8 & r.button && (v |= 128),
                    r.action === 32 ? (v |= 32) : r.action !== 0 || d || (v |= 3)),
                v
              )
            }
            const t = String.fromCharCode,
              n = {
                DEFAULT: (r) => {
                  const d = [c(r, !1) + 32, r.col + 32, r.row + 32]
                  return d[0] > 255 || d[1] > 255 || d[2] > 255
                    ? ''
                    : `\x1B[M${t(d[0])}${t(d[1])}${t(d[2])}`
                },
                SGR: (r) => {
                  const d = r.action === 0 && r.button !== 4 ? 'm' : 'M'
                  return `\x1B[<${c(r, !0)};${r.col};${r.row}${d}`
                },
                SGR_PIXELS: (r) => {
                  const d = r.action === 0 && r.button !== 4 ? 'm' : 'M'
                  return `\x1B[<${c(r, !0)};${r.x};${r.y}${d}`
                }
              }
            let s = (i.CoreMouseService = class extends f.Disposable {
              constructor(r, d) {
                ;(super(),
                  (this._bufferService = r),
                  (this._coreService = d),
                  (this._protocols = {}),
                  (this._encodings = {}),
                  (this._activeProtocol = ''),
                  (this._activeEncoding = ''),
                  (this._lastEvent = null),
                  (this._onProtocolChange = this.register(new h.EventEmitter())),
                  (this.onProtocolChange = this._onProtocolChange.event))
                for (const v of Object.keys(x)) this.addProtocol(v, x[v])
                for (const v of Object.keys(n)) this.addEncoding(v, n[v])
                this.reset()
              }
              addProtocol(r, d) {
                this._protocols[r] = d
              }
              addEncoding(r, d) {
                this._encodings[r] = d
              }
              get activeProtocol() {
                return this._activeProtocol
              }
              get areMouseEventsActive() {
                return this._protocols[this._activeProtocol].events !== 0
              }
              set activeProtocol(r) {
                if (!this._protocols[r]) throw new Error(`unknown protocol "${r}"`)
                ;((this._activeProtocol = r),
                  this._onProtocolChange.fire(this._protocols[r].events))
              }
              get activeEncoding() {
                return this._activeEncoding
              }
              set activeEncoding(r) {
                if (!this._encodings[r]) throw new Error(`unknown encoding "${r}"`)
                this._activeEncoding = r
              }
              reset() {
                ;((this.activeProtocol = 'NONE'),
                  (this.activeEncoding = 'DEFAULT'),
                  (this._lastEvent = null))
              }
              triggerMouseEvent(r) {
                if (
                  r.col < 0 ||
                  r.col >= this._bufferService.cols ||
                  r.row < 0 ||
                  r.row >= this._bufferService.rows ||
                  (r.button === 4 && r.action === 32) ||
                  (r.button === 3 && r.action !== 32) ||
                  (r.button !== 4 && (r.action === 2 || r.action === 3)) ||
                  (r.col++,
                  r.row++,
                  r.action === 32 &&
                    this._lastEvent &&
                    this._equalEvents(this._lastEvent, r, this._activeEncoding === 'SGR_PIXELS')) ||
                  !this._protocols[this._activeProtocol].restrict(r)
                )
                  return !1
                const d = this._encodings[this._activeEncoding](r)
                return (
                  d &&
                    (this._activeEncoding === 'DEFAULT'
                      ? this._coreService.triggerBinaryEvent(d)
                      : this._coreService.triggerDataEvent(d, !0)),
                  (this._lastEvent = r),
                  !0
                )
              }
              explainEvents(r) {
                return {
                  down: !!(1 & r),
                  up: !!(2 & r),
                  drag: !!(4 & r),
                  move: !!(8 & r),
                  wheel: !!(16 & r)
                }
              }
              _equalEvents(r, d, v) {
                if (v) {
                  if (r.x !== d.x || r.y !== d.y) return !1
                } else if (r.col !== d.col || r.row !== d.row) return !1
                return (
                  r.button === d.button &&
                  r.action === d.action &&
                  r.ctrl === d.ctrl &&
                  r.alt === d.alt &&
                  r.shift === d.shift
                )
              }
            })
            i.CoreMouseService = s = l([u(0, a.IBufferService), u(1, a.ICoreService)], s)
          },
          6975: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (s, r, d, v) {
                  var _,
                    b = arguments.length,
                    p = b < 3 ? r : v === null ? (v = Object.getOwnPropertyDescriptor(r, d)) : v
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    p = Reflect.decorate(s, r, d, v)
                  else
                    for (var S = s.length - 1; S >= 0; S--)
                      (_ = s[S]) && (p = (b < 3 ? _(p) : b > 3 ? _(r, d, p) : _(r, d)) || p)
                  return (b > 3 && p && Object.defineProperty(r, d, p), p)
                },
              u =
                (this && this.__param) ||
                function (s, r) {
                  return function (d, v) {
                    r(d, v, s)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.CoreService = void 0))
            const a = o(1439),
              h = o(8460),
              f = o(844),
              x = o(2585),
              c = Object.freeze({ insertMode: !1 }),
              t = Object.freeze({
                applicationCursorKeys: !1,
                applicationKeypad: !1,
                bracketedPasteMode: !1,
                origin: !1,
                reverseWraparound: !1,
                sendFocus: !1,
                wraparound: !0
              })
            let n = (i.CoreService = class extends f.Disposable {
              constructor(s, r, d) {
                ;(super(),
                  (this._bufferService = s),
                  (this._logService = r),
                  (this._optionsService = d),
                  (this.isCursorInitialized = !1),
                  (this.isCursorHidden = !1),
                  (this._onData = this.register(new h.EventEmitter())),
                  (this.onData = this._onData.event),
                  (this._onUserInput = this.register(new h.EventEmitter())),
                  (this.onUserInput = this._onUserInput.event),
                  (this._onBinary = this.register(new h.EventEmitter())),
                  (this.onBinary = this._onBinary.event),
                  (this._onRequestScrollToBottom = this.register(new h.EventEmitter())),
                  (this.onRequestScrollToBottom = this._onRequestScrollToBottom.event),
                  (this.modes = (0, a.clone)(c)),
                  (this.decPrivateModes = (0, a.clone)(t)))
              }
              reset() {
                ;((this.modes = (0, a.clone)(c)), (this.decPrivateModes = (0, a.clone)(t)))
              }
              triggerDataEvent(s, r = !1) {
                if (this._optionsService.rawOptions.disableStdin) return
                const d = this._bufferService.buffer
                ;(r &&
                  this._optionsService.rawOptions.scrollOnUserInput &&
                  d.ybase !== d.ydisp &&
                  this._onRequestScrollToBottom.fire(),
                  r && this._onUserInput.fire(),
                  this._logService.debug(`sending data "${s}"`, () =>
                    s.split('').map((v) => v.charCodeAt(0))
                  ),
                  this._onData.fire(s))
              }
              triggerBinaryEvent(s) {
                this._optionsService.rawOptions.disableStdin ||
                  (this._logService.debug(`sending binary "${s}"`, () =>
                    s.split('').map((r) => r.charCodeAt(0))
                  ),
                  this._onBinary.fire(s))
              }
            })
            i.CoreService = n = l(
              [u(0, x.IBufferService), u(1, x.ILogService), u(2, x.IOptionsService)],
              n
            )
          },
          9074: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.DecorationService = void 0))
            const l = o(8055),
              u = o(8460),
              a = o(844),
              h = o(6106)
            let f = 0,
              x = 0
            class c extends a.Disposable {
              get decorations() {
                return this._decorations.values()
              }
              constructor() {
                ;(super(),
                  (this._decorations = new h.SortedList((s) => s?.marker.line)),
                  (this._onDecorationRegistered = this.register(new u.EventEmitter())),
                  (this.onDecorationRegistered = this._onDecorationRegistered.event),
                  (this._onDecorationRemoved = this.register(new u.EventEmitter())),
                  (this.onDecorationRemoved = this._onDecorationRemoved.event),
                  this.register((0, a.toDisposable)(() => this.reset())))
              }
              registerDecoration(s) {
                if (s.marker.isDisposed) return
                const r = new t(s)
                if (r) {
                  const d = r.marker.onDispose(() => r.dispose())
                  ;(r.onDispose(() => {
                    r &&
                      (this._decorations.delete(r) && this._onDecorationRemoved.fire(r),
                      d.dispose())
                  }),
                    this._decorations.insert(r),
                    this._onDecorationRegistered.fire(r))
                }
                return r
              }
              reset() {
                for (const s of this._decorations.values()) s.dispose()
                this._decorations.clear()
              }
              *getDecorationsAtCell(s, r, d) {
                let v = 0,
                  _ = 0
                for (const b of this._decorations.getKeyIterator(r))
                  ((v = b.options.x ?? 0),
                    (_ = v + (b.options.width ?? 1)),
                    s >= v && s < _ && (!d || (b.options.layer ?? 'bottom') === d) && (yield b))
              }
              forEachDecorationAtCell(s, r, d, v) {
                this._decorations.forEachByKey(r, (_) => {
                  ;((f = _.options.x ?? 0),
                    (x = f + (_.options.width ?? 1)),
                    s >= f && s < x && (!d || (_.options.layer ?? 'bottom') === d) && v(_))
                })
              }
            }
            i.DecorationService = c
            class t extends a.Disposable {
              get isDisposed() {
                return this._isDisposed
              }
              get backgroundColorRGB() {
                return (
                  this._cachedBg === null &&
                    (this.options.backgroundColor
                      ? (this._cachedBg = l.css.toColor(this.options.backgroundColor))
                      : (this._cachedBg = void 0)),
                  this._cachedBg
                )
              }
              get foregroundColorRGB() {
                return (
                  this._cachedFg === null &&
                    (this.options.foregroundColor
                      ? (this._cachedFg = l.css.toColor(this.options.foregroundColor))
                      : (this._cachedFg = void 0)),
                  this._cachedFg
                )
              }
              constructor(s) {
                ;(super(),
                  (this.options = s),
                  (this.onRenderEmitter = this.register(new u.EventEmitter())),
                  (this.onRender = this.onRenderEmitter.event),
                  (this._onDispose = this.register(new u.EventEmitter())),
                  (this.onDispose = this._onDispose.event),
                  (this._cachedBg = null),
                  (this._cachedFg = null),
                  (this.marker = s.marker),
                  this.options.overviewRulerOptions &&
                    !this.options.overviewRulerOptions.position &&
                    (this.options.overviewRulerOptions.position = 'full'))
              }
              dispose() {
                ;(this._onDispose.fire(), super.dispose())
              }
            }
          },
          4348: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.InstantiationService = i.ServiceCollection = void 0))
            const l = o(2585),
              u = o(8343)
            class a {
              constructor(...f) {
                this._entries = new Map()
                for (const [x, c] of f) this.set(x, c)
              }
              set(f, x) {
                const c = this._entries.get(f)
                return (this._entries.set(f, x), c)
              }
              forEach(f) {
                for (const [x, c] of this._entries.entries()) f(x, c)
              }
              has(f) {
                return this._entries.has(f)
              }
              get(f) {
                return this._entries.get(f)
              }
            }
            ;((i.ServiceCollection = a),
              (i.InstantiationService = class {
                constructor() {
                  ;((this._services = new a()), this._services.set(l.IInstantiationService, this))
                }
                setService(h, f) {
                  this._services.set(h, f)
                }
                getService(h) {
                  return this._services.get(h)
                }
                createInstance(h, ...f) {
                  const x = (0, u.getServiceDependencies)(h).sort((n, s) => n.index - s.index),
                    c = []
                  for (const n of x) {
                    const s = this._services.get(n.id)
                    if (!s)
                      throw new Error(
                        `[createInstance] ${h.name} depends on UNKNOWN service ${n.id}.`
                      )
                    c.push(s)
                  }
                  const t = x.length > 0 ? x[0].index : f.length
                  if (f.length !== t)
                    throw new Error(
                      `[createInstance] First service dependency of ${h.name} at position ${t + 1} conflicts with ${f.length} static arguments`
                    )
                  return new h(...f, ...c)
                }
              }))
          },
          7866: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (t, n, s, r) {
                  var d,
                    v = arguments.length,
                    _ = v < 3 ? n : r === null ? (r = Object.getOwnPropertyDescriptor(n, s)) : r
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    _ = Reflect.decorate(t, n, s, r)
                  else
                    for (var b = t.length - 1; b >= 0; b--)
                      (d = t[b]) && (_ = (v < 3 ? d(_) : v > 3 ? d(n, s, _) : d(n, s)) || _)
                  return (v > 3 && _ && Object.defineProperty(n, s, _), _)
                },
              u =
                (this && this.__param) ||
                function (t, n) {
                  return function (s, r) {
                    n(s, r, t)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.traceCall = i.setTraceLogger = i.LogService = void 0))
            const a = o(844),
              h = o(2585),
              f = {
                trace: h.LogLevelEnum.TRACE,
                debug: h.LogLevelEnum.DEBUG,
                info: h.LogLevelEnum.INFO,
                warn: h.LogLevelEnum.WARN,
                error: h.LogLevelEnum.ERROR,
                off: h.LogLevelEnum.OFF
              }
            let x,
              c = (i.LogService = class extends a.Disposable {
                get logLevel() {
                  return this._logLevel
                }
                constructor(t) {
                  ;(super(),
                    (this._optionsService = t),
                    (this._logLevel = h.LogLevelEnum.OFF),
                    this._updateLogLevel(),
                    this.register(
                      this._optionsService.onSpecificOptionChange('logLevel', () =>
                        this._updateLogLevel()
                      )
                    ),
                    (x = this))
                }
                _updateLogLevel() {
                  this._logLevel = f[this._optionsService.rawOptions.logLevel]
                }
                _evalLazyOptionalParams(t) {
                  for (let n = 0; n < t.length; n++) typeof t[n] == 'function' && (t[n] = t[n]())
                }
                _log(t, n, s) {
                  ;(this._evalLazyOptionalParams(s),
                    t.call(
                      console,
                      (this._optionsService.options.logger ? '' : 'xterm.js: ') + n,
                      ...s
                    ))
                }
                trace(t, ...n) {
                  this._logLevel <= h.LogLevelEnum.TRACE &&
                    this._log(
                      this._optionsService.options.logger?.trace.bind(
                        this._optionsService.options.logger
                      ) ?? console.log,
                      t,
                      n
                    )
                }
                debug(t, ...n) {
                  this._logLevel <= h.LogLevelEnum.DEBUG &&
                    this._log(
                      this._optionsService.options.logger?.debug.bind(
                        this._optionsService.options.logger
                      ) ?? console.log,
                      t,
                      n
                    )
                }
                info(t, ...n) {
                  this._logLevel <= h.LogLevelEnum.INFO &&
                    this._log(
                      this._optionsService.options.logger?.info.bind(
                        this._optionsService.options.logger
                      ) ?? console.info,
                      t,
                      n
                    )
                }
                warn(t, ...n) {
                  this._logLevel <= h.LogLevelEnum.WARN &&
                    this._log(
                      this._optionsService.options.logger?.warn.bind(
                        this._optionsService.options.logger
                      ) ?? console.warn,
                      t,
                      n
                    )
                }
                error(t, ...n) {
                  this._logLevel <= h.LogLevelEnum.ERROR &&
                    this._log(
                      this._optionsService.options.logger?.error.bind(
                        this._optionsService.options.logger
                      ) ?? console.error,
                      t,
                      n
                    )
                }
              })
            ;((i.LogService = c = l([u(0, h.IOptionsService)], c)),
              (i.setTraceLogger = function (t) {
                x = t
              }),
              (i.traceCall = function (t, n, s) {
                if (typeof s.value != 'function') throw new Error('not supported')
                const r = s.value
                s.value = function (...d) {
                  if (x.logLevel !== h.LogLevelEnum.TRACE) return r.apply(this, d)
                  x.trace(`GlyphRenderer#${r.name}(${d.map((_) => JSON.stringify(_)).join(', ')})`)
                  const v = r.apply(this, d)
                  return (x.trace(`GlyphRenderer#${r.name} return`, v), v)
                }
              }))
          },
          7302: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.OptionsService = i.DEFAULT_OPTIONS = void 0))
            const l = o(8460),
              u = o(844),
              a = o(6114)
            i.DEFAULT_OPTIONS = {
              cols: 80,
              rows: 24,
              cursorBlink: !1,
              cursorStyle: 'block',
              cursorWidth: 1,
              cursorInactiveStyle: 'outline',
              customGlyphs: !0,
              drawBoldTextInBrightColors: !0,
              documentOverride: null,
              fastScrollModifier: 'alt',
              fastScrollSensitivity: 5,
              fontFamily: 'courier-new, courier, monospace',
              fontSize: 15,
              fontWeight: 'normal',
              fontWeightBold: 'bold',
              ignoreBracketedPasteMode: !1,
              lineHeight: 1,
              letterSpacing: 0,
              linkHandler: null,
              logLevel: 'info',
              logger: null,
              scrollback: 1e3,
              scrollOnUserInput: !0,
              scrollSensitivity: 1,
              screenReaderMode: !1,
              smoothScrollDuration: 0,
              macOptionIsMeta: !1,
              macOptionClickForcesSelection: !1,
              minimumContrastRatio: 1,
              disableStdin: !1,
              allowProposedApi: !1,
              allowTransparency: !1,
              tabStopWidth: 8,
              theme: {},
              rescaleOverlappingGlyphs: !1,
              rightClickSelectsWord: a.isMac,
              windowOptions: {},
              windowsMode: !1,
              windowsPty: {},
              wordSeparator: ' ()[]{}\',"`',
              altClickMovesCursor: !0,
              convertEol: !1,
              termName: 'xterm',
              cancelEvents: !1,
              overviewRulerWidth: 0
            }
            const h = [
              'normal',
              'bold',
              '100',
              '200',
              '300',
              '400',
              '500',
              '600',
              '700',
              '800',
              '900'
            ]
            class f extends u.Disposable {
              constructor(c) {
                ;(super(),
                  (this._onOptionChange = this.register(new l.EventEmitter())),
                  (this.onOptionChange = this._onOptionChange.event))
                const t = { ...i.DEFAULT_OPTIONS }
                for (const n in c)
                  if (n in t)
                    try {
                      const s = c[n]
                      t[n] = this._sanitizeAndValidateOption(n, s)
                    } catch (s) {
                      console.error(s)
                    }
                ;((this.rawOptions = t),
                  (this.options = { ...t }),
                  this._setupOptions(),
                  this.register(
                    (0, u.toDisposable)(() => {
                      ;((this.rawOptions.linkHandler = null),
                        (this.rawOptions.documentOverride = null))
                    })
                  ))
              }
              onSpecificOptionChange(c, t) {
                return this.onOptionChange((n) => {
                  n === c && t(this.rawOptions[c])
                })
              }
              onMultipleOptionChange(c, t) {
                return this.onOptionChange((n) => {
                  c.indexOf(n) !== -1 && t()
                })
              }
              _setupOptions() {
                const c = (n) => {
                    if (!(n in i.DEFAULT_OPTIONS)) throw new Error(`No option with key "${n}"`)
                    return this.rawOptions[n]
                  },
                  t = (n, s) => {
                    if (!(n in i.DEFAULT_OPTIONS)) throw new Error(`No option with key "${n}"`)
                    ;((s = this._sanitizeAndValidateOption(n, s)),
                      this.rawOptions[n] !== s &&
                        ((this.rawOptions[n] = s), this._onOptionChange.fire(n)))
                  }
                for (const n in this.rawOptions) {
                  const s = { get: c.bind(this, n), set: t.bind(this, n) }
                  Object.defineProperty(this.options, n, s)
                }
              }
              _sanitizeAndValidateOption(c, t) {
                switch (c) {
                  case 'cursorStyle':
                    if (
                      (t || (t = i.DEFAULT_OPTIONS[c]),
                      !(function (n) {
                        return n === 'block' || n === 'underline' || n === 'bar'
                      })(t))
                    )
                      throw new Error(`"${t}" is not a valid value for ${c}`)
                    break
                  case 'wordSeparator':
                    t || (t = i.DEFAULT_OPTIONS[c])
                    break
                  case 'fontWeight':
                  case 'fontWeightBold':
                    if (typeof t == 'number' && 1 <= t && t <= 1e3) break
                    t = h.includes(t) ? t : i.DEFAULT_OPTIONS[c]
                    break
                  case 'cursorWidth':
                    t = Math.floor(t)
                  case 'lineHeight':
                  case 'tabStopWidth':
                    if (t < 1) throw new Error(`${c} cannot be less than 1, value: ${t}`)
                    break
                  case 'minimumContrastRatio':
                    t = Math.max(1, Math.min(21, Math.round(10 * t) / 10))
                    break
                  case 'scrollback':
                    if ((t = Math.min(t, 4294967295)) < 0)
                      throw new Error(`${c} cannot be less than 0, value: ${t}`)
                    break
                  case 'fastScrollSensitivity':
                  case 'scrollSensitivity':
                    if (t <= 0)
                      throw new Error(`${c} cannot be less than or equal to 0, value: ${t}`)
                    break
                  case 'rows':
                  case 'cols':
                    if (!t && t !== 0) throw new Error(`${c} must be numeric, value: ${t}`)
                    break
                  case 'windowsPty':
                    t = t ?? {}
                }
                return t
              }
            }
            i.OptionsService = f
          },
          2660: function (I, i, o) {
            var l =
                (this && this.__decorate) ||
                function (f, x, c, t) {
                  var n,
                    s = arguments.length,
                    r = s < 3 ? x : t === null ? (t = Object.getOwnPropertyDescriptor(x, c)) : t
                  if (typeof Reflect == 'object' && typeof Reflect.decorate == 'function')
                    r = Reflect.decorate(f, x, c, t)
                  else
                    for (var d = f.length - 1; d >= 0; d--)
                      (n = f[d]) && (r = (s < 3 ? n(r) : s > 3 ? n(x, c, r) : n(x, c)) || r)
                  return (s > 3 && r && Object.defineProperty(x, c, r), r)
                },
              u =
                (this && this.__param) ||
                function (f, x) {
                  return function (c, t) {
                    x(c, t, f)
                  }
                }
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.OscLinkService = void 0))
            const a = o(2585)
            let h = (i.OscLinkService = class {
              constructor(f) {
                ;((this._bufferService = f),
                  (this._nextId = 1),
                  (this._entriesWithId = new Map()),
                  (this._dataByLinkId = new Map()))
              }
              registerLink(f) {
                const x = this._bufferService.buffer
                if (f.id === void 0) {
                  const d = x.addMarker(x.ybase + x.y),
                    v = { data: f, id: this._nextId++, lines: [d] }
                  return (
                    d.onDispose(() => this._removeMarkerFromLink(v, d)),
                    this._dataByLinkId.set(v.id, v),
                    v.id
                  )
                }
                const c = f,
                  t = this._getEntryIdKey(c),
                  n = this._entriesWithId.get(t)
                if (n) return (this.addLineToLink(n.id, x.ybase + x.y), n.id)
                const s = x.addMarker(x.ybase + x.y),
                  r = { id: this._nextId++, key: this._getEntryIdKey(c), data: c, lines: [s] }
                return (
                  s.onDispose(() => this._removeMarkerFromLink(r, s)),
                  this._entriesWithId.set(r.key, r),
                  this._dataByLinkId.set(r.id, r),
                  r.id
                )
              }
              addLineToLink(f, x) {
                const c = this._dataByLinkId.get(f)
                if (c && c.lines.every((t) => t.line !== x)) {
                  const t = this._bufferService.buffer.addMarker(x)
                  ;(c.lines.push(t), t.onDispose(() => this._removeMarkerFromLink(c, t)))
                }
              }
              getLinkData(f) {
                return this._dataByLinkId.get(f)?.data
              }
              _getEntryIdKey(f) {
                return `${f.id};;${f.uri}`
              }
              _removeMarkerFromLink(f, x) {
                const c = f.lines.indexOf(x)
                c !== -1 &&
                  (f.lines.splice(c, 1),
                  f.lines.length === 0 &&
                    (f.data.id !== void 0 && this._entriesWithId.delete(f.key),
                    this._dataByLinkId.delete(f.id)))
              }
            })
            i.OscLinkService = h = l([u(0, a.IBufferService)], h)
          },
          8343: (I, i) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.createDecorator = i.getServiceDependencies = i.serviceRegistry = void 0))
            const o = 'di$target',
              l = 'di$dependencies'
            ;((i.serviceRegistry = new Map()),
              (i.getServiceDependencies = function (u) {
                return u[l] || []
              }),
              (i.createDecorator = function (u) {
                if (i.serviceRegistry.has(u)) return i.serviceRegistry.get(u)
                const a = function (h, f, x) {
                  if (arguments.length !== 3)
                    throw new Error(
                      '@IServiceName-decorator can only be used to decorate a parameter'
                    )
                  ;(function (c, t, n) {
                    t[o] === t
                      ? t[l].push({ id: c, index: n })
                      : ((t[l] = [{ id: c, index: n }]), (t[o] = t))
                  })(a, h, x)
                }
                return ((a.toString = () => u), i.serviceRegistry.set(u, a), a)
              }))
          },
          2585: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }),
              (i.IDecorationService =
                i.IUnicodeService =
                i.IOscLinkService =
                i.IOptionsService =
                i.ILogService =
                i.LogLevelEnum =
                i.IInstantiationService =
                i.ICharsetService =
                i.ICoreService =
                i.ICoreMouseService =
                i.IBufferService =
                  void 0))
            const l = o(8343)
            var u
            ;((i.IBufferService = (0, l.createDecorator)('BufferService')),
              (i.ICoreMouseService = (0, l.createDecorator)('CoreMouseService')),
              (i.ICoreService = (0, l.createDecorator)('CoreService')),
              (i.ICharsetService = (0, l.createDecorator)('CharsetService')),
              (i.IInstantiationService = (0, l.createDecorator)('InstantiationService')),
              (function (a) {
                ;((a[(a.TRACE = 0)] = 'TRACE'),
                  (a[(a.DEBUG = 1)] = 'DEBUG'),
                  (a[(a.INFO = 2)] = 'INFO'),
                  (a[(a.WARN = 3)] = 'WARN'),
                  (a[(a.ERROR = 4)] = 'ERROR'),
                  (a[(a.OFF = 5)] = 'OFF'))
              })(u || (i.LogLevelEnum = u = {})),
              (i.ILogService = (0, l.createDecorator)('LogService')),
              (i.IOptionsService = (0, l.createDecorator)('OptionsService')),
              (i.IOscLinkService = (0, l.createDecorator)('OscLinkService')),
              (i.IUnicodeService = (0, l.createDecorator)('UnicodeService')),
              (i.IDecorationService = (0, l.createDecorator)('DecorationService')))
          },
          1480: (I, i, o) => {
            ;(Object.defineProperty(i, '__esModule', { value: !0 }), (i.UnicodeService = void 0))
            const l = o(8460),
              u = o(225)
            class a {
              static extractShouldJoin(f) {
                return (1 & f) != 0
              }
              static extractWidth(f) {
                return (f >> 1) & 3
              }
              static extractCharKind(f) {
                return f >> 3
              }
              static createPropertyValue(f, x, c = !1) {
                return ((16777215 & f) << 3) | ((3 & x) << 1) | (c ? 1 : 0)
              }
              constructor() {
                ;((this._providers = Object.create(null)),
                  (this._active = ''),
                  (this._onChange = new l.EventEmitter()),
                  (this.onChange = this._onChange.event))
                const f = new u.UnicodeV6()
                ;(this.register(f), (this._active = f.version), (this._activeProvider = f))
              }
              dispose() {
                this._onChange.dispose()
              }
              get versions() {
                return Object.keys(this._providers)
              }
              get activeVersion() {
                return this._active
              }
              set activeVersion(f) {
                if (!this._providers[f]) throw new Error(`unknown Unicode version "${f}"`)
                ;((this._active = f),
                  (this._activeProvider = this._providers[f]),
                  this._onChange.fire(f))
              }
              register(f) {
                this._providers[f.version] = f
              }
              wcwidth(f) {
                return this._activeProvider.wcwidth(f)
              }
              getStringCellWidth(f) {
                let x = 0,
                  c = 0
                const t = f.length
                for (let n = 0; n < t; ++n) {
                  let s = f.charCodeAt(n)
                  if (55296 <= s && s <= 56319) {
                    if (++n >= t) return x + this.wcwidth(s)
                    const v = f.charCodeAt(n)
                    56320 <= v && v <= 57343
                      ? (s = 1024 * (s - 55296) + v - 56320 + 65536)
                      : (x += this.wcwidth(v))
                  }
                  const r = this.charProperties(s, c)
                  let d = a.extractWidth(r)
                  ;(a.extractShouldJoin(r) && (d -= a.extractWidth(c)), (x += d), (c = r))
                }
                return x
              }
              charProperties(f, x) {
                return this._activeProvider.charProperties(f, x)
              }
            }
            i.UnicodeService = a
          }
        },
        w = {}
      function T(I) {
        var i = w[I]
        if (i !== void 0) return i.exports
        var o = (w[I] = { exports: {} })
        return (g[I].call(o.exports, o, o.exports, T), o.exports)
      }
      var R = {}
      return (
        (() => {
          var I = R
          ;(Object.defineProperty(I, '__esModule', { value: !0 }), (I.Terminal = void 0))
          const i = T(9042),
            o = T(3236),
            l = T(844),
            u = T(5741),
            a = T(8285),
            h = T(7975),
            f = T(7090),
            x = ['cols', 'rows']
          class c extends l.Disposable {
            constructor(n) {
              ;(super(),
                (this._core = this.register(new o.Terminal(n))),
                (this._addonManager = this.register(new u.AddonManager())),
                (this._publicOptions = { ...this._core.options }))
              const s = (d) => this._core.options[d],
                r = (d, v) => {
                  ;(this._checkReadonlyOptions(d), (this._core.options[d] = v))
                }
              for (const d in this._core.options) {
                const v = { get: s.bind(this, d), set: r.bind(this, d) }
                Object.defineProperty(this._publicOptions, d, v)
              }
            }
            _checkReadonlyOptions(n) {
              if (x.includes(n)) throw new Error(`Option "${n}" can only be set in the constructor`)
            }
            _checkProposedApi() {
              if (!this._core.optionsService.rawOptions.allowProposedApi)
                throw new Error(
                  'You must set the allowProposedApi option to true to use proposed API'
                )
            }
            get onBell() {
              return this._core.onBell
            }
            get onBinary() {
              return this._core.onBinary
            }
            get onCursorMove() {
              return this._core.onCursorMove
            }
            get onData() {
              return this._core.onData
            }
            get onKey() {
              return this._core.onKey
            }
            get onLineFeed() {
              return this._core.onLineFeed
            }
            get onRender() {
              return this._core.onRender
            }
            get onResize() {
              return this._core.onResize
            }
            get onScroll() {
              return this._core.onScroll
            }
            get onSelectionChange() {
              return this._core.onSelectionChange
            }
            get onTitleChange() {
              return this._core.onTitleChange
            }
            get onWriteParsed() {
              return this._core.onWriteParsed
            }
            get element() {
              return this._core.element
            }
            get parser() {
              return (this._parser || (this._parser = new h.ParserApi(this._core)), this._parser)
            }
            get unicode() {
              return (this._checkProposedApi(), new f.UnicodeApi(this._core))
            }
            get textarea() {
              return this._core.textarea
            }
            get rows() {
              return this._core.rows
            }
            get cols() {
              return this._core.cols
            }
            get buffer() {
              return (
                this._buffer ||
                  (this._buffer = this.register(new a.BufferNamespaceApi(this._core))),
                this._buffer
              )
            }
            get markers() {
              return (this._checkProposedApi(), this._core.markers)
            }
            get modes() {
              const n = this._core.coreService.decPrivateModes
              let s = 'none'
              switch (this._core.coreMouseService.activeProtocol) {
                case 'X10':
                  s = 'x10'
                  break
                case 'VT200':
                  s = 'vt200'
                  break
                case 'DRAG':
                  s = 'drag'
                  break
                case 'ANY':
                  s = 'any'
              }
              return {
                applicationCursorKeysMode: n.applicationCursorKeys,
                applicationKeypadMode: n.applicationKeypad,
                bracketedPasteMode: n.bracketedPasteMode,
                insertMode: this._core.coreService.modes.insertMode,
                mouseTrackingMode: s,
                originMode: n.origin,
                reverseWraparoundMode: n.reverseWraparound,
                sendFocusMode: n.sendFocus,
                wraparoundMode: n.wraparound
              }
            }
            get options() {
              return this._publicOptions
            }
            set options(n) {
              for (const s in n) this._publicOptions[s] = n[s]
            }
            blur() {
              this._core.blur()
            }
            focus() {
              this._core.focus()
            }
            input(n, s = !0) {
              this._core.input(n, s)
            }
            resize(n, s) {
              ;(this._verifyIntegers(n, s), this._core.resize(n, s))
            }
            open(n) {
              this._core.open(n)
            }
            attachCustomKeyEventHandler(n) {
              this._core.attachCustomKeyEventHandler(n)
            }
            attachCustomWheelEventHandler(n) {
              this._core.attachCustomWheelEventHandler(n)
            }
            registerLinkProvider(n) {
              return this._core.registerLinkProvider(n)
            }
            registerCharacterJoiner(n) {
              return (this._checkProposedApi(), this._core.registerCharacterJoiner(n))
            }
            deregisterCharacterJoiner(n) {
              ;(this._checkProposedApi(), this._core.deregisterCharacterJoiner(n))
            }
            registerMarker(n = 0) {
              return (this._verifyIntegers(n), this._core.registerMarker(n))
            }
            registerDecoration(n) {
              return (
                this._checkProposedApi(),
                this._verifyPositiveIntegers(n.x ?? 0, n.width ?? 0, n.height ?? 0),
                this._core.registerDecoration(n)
              )
            }
            hasSelection() {
              return this._core.hasSelection()
            }
            select(n, s, r) {
              ;(this._verifyIntegers(n, s, r), this._core.select(n, s, r))
            }
            getSelection() {
              return this._core.getSelection()
            }
            getSelectionPosition() {
              return this._core.getSelectionPosition()
            }
            clearSelection() {
              this._core.clearSelection()
            }
            selectAll() {
              this._core.selectAll()
            }
            selectLines(n, s) {
              ;(this._verifyIntegers(n, s), this._core.selectLines(n, s))
            }
            dispose() {
              super.dispose()
            }
            scrollLines(n) {
              ;(this._verifyIntegers(n), this._core.scrollLines(n))
            }
            scrollPages(n) {
              ;(this._verifyIntegers(n), this._core.scrollPages(n))
            }
            scrollToTop() {
              this._core.scrollToTop()
            }
            scrollToBottom() {
              this._core.scrollToBottom()
            }
            scrollToLine(n) {
              ;(this._verifyIntegers(n), this._core.scrollToLine(n))
            }
            clear() {
              this._core.clear()
            }
            write(n, s) {
              this._core.write(n, s)
            }
            writeln(n, s) {
              ;(this._core.write(n),
                this._core.write(
                  `\r
`,
                  s
                ))
            }
            paste(n) {
              this._core.paste(n)
            }
            refresh(n, s) {
              ;(this._verifyIntegers(n, s), this._core.refresh(n, s))
            }
            reset() {
              this._core.reset()
            }
            clearTextureAtlas() {
              this._core.clearTextureAtlas()
            }
            loadAddon(n) {
              this._addonManager.loadAddon(this, n)
            }
            static get strings() {
              return i
            }
            _verifyIntegers(...n) {
              for (const s of n)
                if (s === 1 / 0 || isNaN(s) || s % 1 != 0)
                  throw new Error('This API only accepts integers')
            }
            _verifyPositiveIntegers(...n) {
              for (const s of n)
                if (s && (s === 1 / 0 || isNaN(s) || s % 1 != 0 || s < 0))
                  throw new Error('This API only accepts positive integers')
            }
          }
          I.Terminal = c
        })(),
        R
      )
    })()
  )
})(zs)
var wi = zs.exports,
  Gs = { exports: {} }
;(function (m, y) {
  ;(function (g, w) {
    m.exports = w()
  })(self, () =>
    (() => {
      var g = {}
      return (
        (() => {
          var w = g
          ;(Object.defineProperty(w, '__esModule', { value: !0 }),
            (w.FitAddon = void 0),
            (w.FitAddon = class {
              activate(T) {
                this._terminal = T
              }
              dispose() {}
              fit() {
                const T = this.proposeDimensions()
                if (!T || !this._terminal || isNaN(T.cols) || isNaN(T.rows)) return
                const R = this._terminal._core
                ;(this._terminal.rows === T.rows && this._terminal.cols === T.cols) ||
                  (R._renderService.clear(), this._terminal.resize(T.cols, T.rows))
              }
              proposeDimensions() {
                if (
                  !this._terminal ||
                  !this._terminal.element ||
                  !this._terminal.element.parentElement
                )
                  return
                const T = this._terminal._core,
                  R = T._renderService.dimensions
                if (R.css.cell.width === 0 || R.css.cell.height === 0) return
                const I = this._terminal.options.scrollback === 0 ? 0 : T.viewport.scrollBarWidth,
                  i = window.getComputedStyle(this._terminal.element.parentElement),
                  o = parseInt(i.getPropertyValue('height')),
                  l = Math.max(0, parseInt(i.getPropertyValue('width'))),
                  u = window.getComputedStyle(this._terminal.element),
                  a =
                    o -
                    (parseInt(u.getPropertyValue('padding-top')) +
                      parseInt(u.getPropertyValue('padding-bottom'))),
                  h =
                    l -
                    (parseInt(u.getPropertyValue('padding-right')) +
                      parseInt(u.getPropertyValue('padding-left'))) -
                    I
                return {
                  cols: Math.max(2, Math.floor(h / R.css.cell.width)),
                  rows: Math.max(1, Math.floor(a / R.css.cell.height))
                }
              }
            }))
        })(),
        g
      )
    })()
  )
})(Gs)
var It = Gs.exports
const Ci = Ys(It),
  ki = er({ __proto__: null, default: Ci }, [It]),
  Xt = qe()
function ji() {
  const { sessionId: m } = xe(),
    y = Ze(),
    g = k.useRef(null),
    w = k.useRef(null),
    T = k.useRef(null),
    [R, I] = k.useState(null),
    [i, o] = k.useState([]),
    [l, u] = k.useState(!0),
    [a, h] = k.useState(!1),
    [f, x] = k.useState(0),
    [c, t] = k.useState(0),
    [n, s] = k.useState(1),
    r = k.useRef(null)
  ;(k.useEffect(() => {
    ;(async () => {
      if (m)
        try {
          const j = await fetch(`${Xt}/v1/webshell/sessions/${m}/events`, {
            headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
          })
          if (j.ok) {
            const D = await j.json()
            ;(I(D.session), o(D.events))
          }
        } catch {
        } finally {
          u(!1)
        }
    })()
  }, [m]),
    k.useEffect(() => {
      if (!g.current || !R) return
      const P = new wi.Terminal({
          cursorBlink: !1,
          disableStdin: !0,
          fontSize: 14,
          fontFamily: 'Menlo, Monaco, "Courier New", monospace',
          theme: { background: '#1e1e1e', foreground: '#d4d4d4' }
        }),
        j = new It.FitAddon()
      ;(P.loadAddon(j), P.open(g.current), j.fit(), (w.current = P), (T.current = j))
      const D = () => {
        j.fit()
      }
      return (
        window.addEventListener('resize', D),
        () => {
          ;(window.removeEventListener('resize', D), P.dispose())
        }
      )
    }, [R]),
    k.useEffect(() => {
      if (!a || i.length === 0) {
        r.current && (clearTimeout(r.current), (r.current = null))
        return
      }
      const P = () => {
        if (f >= i.length) {
          h(!1)
          return
        }
        const j = i[f],
          D = w.current
        if (D) {
          switch (j.event_type) {
            case 'output':
              j.data && D.write(j.data)
              break
            case 'resize':
              j.cols && j.rows && D.resize(j.cols, j.rows)
              break
            case 'connected':
              j.data &&
                D.write(`\r
${j.data}\r
`)
              break
          }
          if ((t(j.time_offset), x(f + 1), f + 1 < i.length)) {
            const $ = (i[f + 1].time_offset - j.time_offset) / n
            r.current = setTimeout(P, $)
          } else h(!1)
        }
      }
      return (
        P(),
        () => {
          r.current && (clearTimeout(r.current), (r.current = null))
        }
      )
    }, [a, f, i, n]))
  const d = () => {
      f >= i.length ? v() : h(!a)
    },
    v = () => {
      ;(x(0), t(0), h(!1))
      const P = w.current
      P && P.clear()
    },
    _ = () => {
      const P = Math.max(0, f - 10)
      ;(x(P), h(!1))
      const j = w.current
      if (j) {
        j.clear()
        for (let D = 0; D < P; D++) {
          const O = i[D]
          O.event_type === 'output' && O.data && j.write(O.data)
        }
        P > 0 && t(i[P - 1].time_offset)
      }
    },
    b = () => {
      const P = Math.min(i.length, f + 10),
        j = w.current
      if (j)
        for (let D = f; D < P; D++) {
          const O = i[D]
          O.event_type === 'output' && O.data && j.write(O.data)
        }
      ;(x(P), P > 0 && P <= i.length && t(i[P - 1].time_offset))
    },
    p = (P) => {
      h(!1)
      let j = 0
      for (let O = 0; O < i.length && i[O].time_offset <= P; O++) j = O + 1
      const D = w.current
      if (D) {
        D.clear()
        for (let O = 0; O < j; O++) {
          const $ = i[O]
          $.event_type === 'output' && $.data && D.write($.data)
        }
      }
      ;(x(j), t(P))
    },
    S = (P) => {
      const j = Math.floor(P / 1e3),
        D = Math.floor(j / 60),
        O = j % 60
      return `${D.toString().padStart(2, '0')}:${O.toString().padStart(2, '0')}`
    },
    L = i.length > 0 ? i[i.length - 1].time_offset : 0,
    M = async () => {
      if (m)
        try {
          const P = await fetch(`${Xt}/v1/webshell/sessions/${m}/export`, {
            headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
          })
          if (P.ok) {
            const j = await P.blob(),
              D = window.URL.createObjectURL(j),
              O = document.createElement('a')
            ;((O.href = D),
              (O.download = `webshell-${m}-${R?.started_at}.cast`),
              document.body.appendChild(O),
              O.click(),
              document.body.removeChild(O),
              window.URL.revokeObjectURL(D))
          }
        } catch {}
    }
  return l
    ? e.jsx('div', {
        className: 'flex items-center justify-center h-screen',
        children: e.jsx('div', {
          className: 'text-gray-500 dark:text-gray-400',
          children: '加载中...'
        })
      })
    : R
      ? e.jsxs('div', {
          className: 'flex flex-col h-screen bg-gray-100 dark:bg-gray-900',
          children: [
            e.jsx('div', {
              className:
                'bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4',
              children: e.jsxs('div', {
                className: 'flex items-center justify-between',
                children: [
                  e.jsxs('div', {
                    className: 'flex items-center gap-4',
                    children: [
                      e.jsx('button', {
                        onClick: () => y('/webshell/sessions'),
                        className: 'p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded',
                        children: e.jsx(ni, { className: 'w-5 h-5' })
                      }),
                      e.jsxs('div', {
                        children: [
                          e.jsx('h1', {
                            className: 'text-xl font-bold text-gray-900 dark:text-white',
                            children: '会话回放'
                          }),
                          e.jsxs('p', {
                            className: 'text-sm text-gray-500 dark:text-gray-400',
                            children: [
                              R.username,
                              ' @ ',
                              R.remote_host,
                              ':',
                              R.remote_port,
                              ' ·',
                              ' ',
                              new Date(R.started_at).toLocaleString('zh-CN')
                            ]
                          })
                        ]
                      })
                    ]
                  }),
                  e.jsxs('button', {
                    onClick: M,
                    className:
                      'flex items-center gap-2 px-4 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-600',
                    children: [e.jsx(Ws, { className: 'w-4 h-4' }), '导出']
                  })
                ]
              })
            }),
            e.jsx('div', {
              className: 'flex-1 p-6',
              children: e.jsx('div', {
                className: 'h-full bg-[#1e1e1e] rounded-lg overflow-hidden shadow-lg',
                children: e.jsx('div', { ref: g, className: 'w-full h-full p-4' })
              })
            }),
            e.jsx('div', {
              className:
                'bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4',
              children: e.jsxs('div', {
                className: 'space-y-4',
                children: [
                  e.jsxs('div', {
                    className: 'flex items-center gap-4',
                    children: [
                      e.jsx('span', {
                        className: 'text-sm text-gray-600 dark:text-gray-400 font-mono w-16',
                        children: S(c)
                      }),
                      e.jsx('div', {
                        className: 'flex-1',
                        children: e.jsx('input', {
                          type: 'range',
                          min: 0,
                          max: L,
                          value: c,
                          onChange: (P) => p(parseInt(P.target.value)),
                          className:
                            'w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer',
                          style: {
                            background: `linear-gradient(to right, #3b82f6 0%, #3b82f6 ${(c / L) * 100}%, #e5e7eb ${(c / L) * 100}%, #e5e7eb 100%)`
                          }
                        })
                      }),
                      e.jsx('span', {
                        className: 'text-sm text-gray-600 dark:text-gray-400 font-mono w-16',
                        children: S(L)
                      })
                    ]
                  }),
                  e.jsxs('div', {
                    className: 'flex items-center justify-center gap-4',
                    children: [
                      e.jsx('button', {
                        onClick: v,
                        className: 'p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded',
                        title: '重新开始',
                        children: e.jsx(Gt, { className: 'w-5 h-5' })
                      }),
                      e.jsx('button', {
                        onClick: _,
                        className: 'p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded',
                        title: '后退',
                        children: e.jsx(Gt, { className: 'w-4 h-4' })
                      }),
                      e.jsx('button', {
                        onClick: d,
                        className: 'p-3 bg-blue-600 text-white rounded-full hover:bg-blue-700',
                        children: a
                          ? e.jsx(fi, { className: 'w-6 h-6' })
                          : e.jsx(Us, { className: 'w-6 h-6' })
                      }),
                      e.jsx('button', {
                        onClick: b,
                        className: 'p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded',
                        title: '前进',
                        children: e.jsx(vi, { className: 'w-4 h-4' })
                      }),
                      e.jsxs('div', {
                        className: 'flex items-center gap-2 ml-4',
                        children: [
                          e.jsx('span', {
                            className: 'text-sm text-gray-600 dark:text-gray-400',
                            children: '速度:'
                          }),
                          e.jsxs('select', {
                            value: n,
                            onChange: (P) => s(parseFloat(P.target.value)),
                            className:
                              'px-2 py-1 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded text-sm',
                            children: [
                              e.jsx('option', { value: 0.5, children: '0.5x' }),
                              e.jsx('option', { value: 1, children: '1x' }),
                              e.jsx('option', { value: 1.5, children: '1.5x' }),
                              e.jsx('option', { value: 2, children: '2x' }),
                              e.jsx('option', { value: 3, children: '3x' })
                            ]
                          })
                        ]
                      })
                    ]
                  }),
                  e.jsxs('div', {
                    className:
                      'flex items-center justify-center gap-8 text-sm text-gray-600 dark:text-gray-400',
                    children: [
                      e.jsxs('span', { children: ['事件: ', f, ' / ', i.length] }),
                      e.jsxs('span', { children: ['上传: ', Jt(R.bytes_sent)] }),
                      e.jsxs('span', { children: ['下载: ', Jt(R.bytes_received)] })
                    ]
                  })
                ]
              })
            })
          ]
        })
      : e.jsx('div', {
          className: 'flex items-center justify-center h-screen',
          children: e.jsxs('div', {
            className: 'text-center',
            children: [
              e.jsx('p', {
                className: 'text-gray-500 dark:text-gray-400 mb-4',
                children: '会话未找到'
              }),
              e.jsx('button', {
                onClick: () => y('/webshell/sessions'),
                className: 'px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700',
                children: '返回列表'
              })
            ]
          })
        })
}
function Jt(m) {
  if (m === 0) return '0 B'
  const y = 1024,
    g = ['B', 'KB', 'MB', 'GB'],
    w = Math.floor(Math.log(m) / Math.log(y))
  return Math.round((m / Math.pow(y, w)) * 100) / 100 + ' ' + g[w]
}
function Ni() {
  const { projectId: m } = xe(),
    y = Te((h) => h.setActiveProjectId)
  k.useEffect(() => {
    m && y(m)
  }, [m, y])
  const {
      vpcs: g,
      routes: w,
      lbs: T,
      sgRules: R,
      asns: I,
      clusters: i,
      snapshots: o,
      volumes: l
    } = Ce(),
    u = m,
    a = {
      vpcs: g.filter((h) => h.projectId === u).length,
      routes: w.filter((h) => h.projectId === u).length,
      lbs: T.filter((h) => h.projectId === u).length,
      sg: R.filter((h) => h.projectId === u).length,
      asns: I.filter((h) => h.projectId === u).length,
      clusters: i.filter((h) => h.projectId === u).length,
      snapshots: o.filter((h) => h.projectId === u).length,
      volumes: l.filter((h) => h.projectId === u).length
    }
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsxs('div', {
        className: 'card p-4',
        children: [
          e.jsx('h1', { className: 'text-lg font-semibold', children: 'Project Overview' }),
          e.jsxs('p', { className: 'text-gray-400', children: ['Project ID: ', m] })
        ]
      }),
      e.jsxs('div', {
        className: 'grid md:grid-cols-2 lg:grid-cols-3 gap-3',
        children: [
          e.jsxs('div', {
            className: 'card p-4',
            children: [
              e.jsx('div', { className: 'text-gray-400', children: 'VPCs' }),
              e.jsx('div', { className: 'text-2xl font-semibold', children: a.vpcs }),
              e.jsx(Oe, {
                className: 'text-oxide-300 hover:underline',
                to: `/project/${m}/network/vpc`,
                children: 'View'
              })
            ]
          }),
          e.jsxs('div', {
            className: 'card p-4',
            children: [
              e.jsx('div', { className: 'text-gray-400', children: 'ASNs' }),
              e.jsx('div', { className: 'text-2xl font-semibold', children: a.asns }),
              e.jsx(Oe, {
                className: 'text-oxide-300 hover:underline',
                to: `/project/${m}/network/asns`,
                children: 'View'
              })
            ]
          }),
          e.jsxs('div', {
            className: 'card p-4',
            children: [
              e.jsx('div', { className: 'text-gray-400', children: 'Clusters' }),
              e.jsx('div', { className: 'text-2xl font-semibold', children: a.clusters }),
              e.jsx(Oe, {
                className: 'text-oxide-300 hover:underline',
                to: `/project/${m}/compute/k8s`,
                children: 'View'
              })
            ]
          }),
          e.jsxs('div', {
            className: 'card p-4',
            children: [
              e.jsx('div', { className: 'text-gray-400', children: 'Snapshots' }),
              e.jsx('div', { className: 'text-2xl font-semibold', children: a.snapshots }),
              e.jsx(Oe, {
                className: 'text-oxide-300 hover:underline',
                to: `/project/${m}/compute/snapshots`,
                children: 'View'
              })
            ]
          }),
          e.jsxs('div', {
            className: 'card p-4',
            children: [
              e.jsx('div', { className: 'text-gray-400', children: 'Volumes' }),
              e.jsx('div', { className: 'text-2xl font-semibold', children: a.volumes }),
              e.jsx(Oe, {
                className: 'text-oxide-300 hover:underline',
                to: `/project/${m}/storage`,
                children: 'View'
              })
            ]
          })
        ]
      })
    ]
  })
}
function Yt() {
  const [m, y] = k.useState([]),
    [g, w] = k.useState(!1),
    [T, R] = k.useState(!1),
    [I, i] = k.useState(null),
    [o, l] = k.useState(''),
    [u, a] = k.useState(''),
    [h, f] = k.useState(!1)
  k.useEffect(() => {
    ;(async () => y(await ge()))()
  }, [])
  const x = k.useMemo(
      () =>
        m.filter(
          (t) =>
            (t.disk_format === 'qcow2' || t.disk_format === 'raw') && (!u || t.name.includes(u))
        ),
      [m, u]
    ),
    c = [
      { key: 'name', header: 'Name' },
      {
        key: 'status',
        header: 'State',
        render: (t) =>
          e.jsx(pe, {
            variant: t.status === 'active' || t.status === 'available' ? 'success' : 'info',
            children: t.status
          })
      },
      {
        key: 'disk_format',
        header: 'OS Type',
        render: (t) => e.jsx('span', { className: 'uppercase', children: t.disk_format })
      },
      { key: 'hypervisor', header: 'Hypervisor', render: () => e.jsx('span', { children: 'KVM' }) },
      {
        key: 'owner',
        header: 'Account',
        render: (t) => e.jsx('span', { children: t.owner ?? '-' })
      },
      {
        key: 'actions',
        header: 'Actions',
        render: (t) =>
          e.jsx('div', {
            className: 'flex gap-2',
            children: e.jsx('button', {
              className: 'btn-danger btn-xs',
              onClick: async () => {
                ;(await ht(t.id), y(await ge()))
              },
              children: 'Delete'
            })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, { title: 'Images', subtitle: 'Templates' }),
      e.jsxs(Ne, {
        placeholder: 'Search templates',
        onSearch: a,
        children: [
          e.jsx('button', {
            className: 'btn-secondary',
            onClick: async () => y(await ge()),
            children: 'Refresh'
          }),
          e.jsx('button', {
            className: 'btn-secondary',
            onClick: () => R(!0),
            children: 'Register from URL'
          }),
          e.jsx('button', { className: 'btn-primary', onClick: () => w(!0), children: 'Upload' })
        ]
      }),
      e.jsx(de, { columns: c, data: x, empty: 'No templates' }),
      e.jsx(le, {
        title: 'Upload Template',
        open: g,
        onClose: () => w(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => w(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: !I || h,
              onClick: async () => {
                if (I) {
                  f(!0)
                  try {
                    ;(await ct(I, { name: I.name }), y(await ge()), w(!1), i(null))
                  } finally {
                    f(!1)
                  }
                }
              },
              children: 'Upload'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsx('input', {
              type: 'file',
              accept: '.qcow2,.raw,.img',
              onChange: (t) => i(t.target.files?.[0] ?? null)
            }),
            e.jsx('p', {
              className: 'text-xs text-muted-foreground',
              children: '支持 qcow2/raw/img 文件，上传后将保存到后端的 VC_IMAGE_DIR 并注册为镜像。'
            })
          ]
        })
      }),
      e.jsx(le, {
        title: 'Register Template from URL',
        open: T,
        onClose: () => R(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => R(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: !o || h,
              onClick: async () => {
                f(!0)
                try {
                  const t = o.split('/').pop() || 'template',
                    { registerImage: n } = await Ke(
                      async () => {
                        const { registerImage: s } = await Promise.resolve().then(() => Hs)
                        return { registerImage: s }
                      },
                      void 0
                    )
                  ;(await n({ name: t, disk_format: 'qcow2', rgw_url: o }),
                    y(await ge()),
                    R(!1),
                    l(''))
                } finally {
                  f(!1)
                }
              },
              children: 'Register'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsx('input', {
              className: 'input w-full',
              placeholder: 'https://rgw.example.com/bucket/key.qcow2',
              value: o,
              onChange: (t) => l(t.target.value)
            }),
            e.jsx('p', {
              className: 'text-xs text-muted-foreground',
              children:
                '支持 RGW/HTTP URL；注册后可在 Images 页面点击 Import 将其落地到 RBD 或文件路径。'
            })
          ]
        })
      })
    ]
  })
}
function Zt() {
  const [m, y] = k.useState([]),
    [g, w] = k.useState(!1),
    [T, R] = k.useState(!1),
    [I, i] = k.useState(null),
    [o, l] = k.useState(!1),
    [u, a] = k.useState('')
  k.useEffect(() => {
    ;(async () => y(await ge()))()
  }, [])
  const h = k.useMemo(
      () => m.filter((x) => x.disk_format === 'iso' && (!u || x.name.includes(u))),
      [m, u]
    ),
    f = [
      { key: 'name', header: 'Name' },
      {
        key: 'status',
        header: 'State',
        render: (x) =>
          e.jsx(pe, {
            variant: x.status === 'active' || x.status === 'available' ? 'success' : 'info',
            children: x.status
          })
      },
      {
        key: 'disk_format',
        header: 'OS Type',
        render: () => e.jsx('span', { className: 'uppercase', children: 'ISO' })
      },
      { key: 'sizeGiB', header: 'Size (GiB)' },
      {
        key: 'owner',
        header: 'Account',
        render: (x) => e.jsx('span', { children: x.owner ?? '-' })
      },
      {
        key: 'actions',
        header: 'Actions',
        render: (x) =>
          e.jsx('div', {
            className: 'flex gap-2',
            children: e.jsx('button', {
              className: 'btn-danger btn-xs',
              onClick: async () => {
                ;(await ht(x.id), y(await ge()))
              },
              children: 'Delete'
            })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, { title: 'Images', subtitle: 'ISOs' }),
      e.jsxs(Ne, {
        placeholder: 'Search ISO',
        onSearch: a,
        children: [
          e.jsx('button', {
            className: 'btn-secondary',
            onClick: async () => y(await ge()),
            children: 'Refresh'
          }),
          e.jsx('button', {
            className: 'btn-secondary',
            onClick: () => R(!0),
            children: 'Register from URL'
          }),
          e.jsx('button', { className: 'btn-primary', onClick: () => w(!0), children: 'Upload' })
        ]
      }),
      e.jsx(de, { columns: f, data: h, empty: 'No ISOs' }),
      e.jsx(le, {
        title: 'Upload ISO',
        open: g,
        onClose: () => w(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => w(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: !I || o,
              onClick: async () => {
                if (I) {
                  l(!0)
                  try {
                    ;(await ct(I, { name: I.name }), y(await ge()), w(!1), i(null))
                  } finally {
                    l(!1)
                  }
                }
              },
              children: 'Upload'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsx('input', {
              type: 'file',
              accept: '.iso',
              onChange: (x) => i(x.target.files?.[0] ?? null)
            }),
            e.jsx('p', {
              className: 'text-xs text-muted-foreground',
              children: '上传 ISO 文件用于手动安装系统，保存到后端的 VC_IMAGE_DIR。'
            })
          ]
        })
      }),
      e.jsx(le, {
        title: 'Register ISO from URL',
        open: T,
        onClose: () => R(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => R(!1),
              children: 'Cancel'
            }),
            e.jsx(Ks, {
              kind: 'iso',
              onDone: async () => {
                ;(y(await ge()), R(!1))
              }
            })
          ]
        }),
        children: e.jsx(qs, { placeholder: 'https://rgw.example.com/bucket/ubuntu.iso' })
      })
    ]
  })
}
function Qt() {
  const [m, y] = k.useState([]),
    [g, w] = k.useState(!1),
    [T, R] = k.useState(!1),
    [I, i] = k.useState(null),
    [o, l] = k.useState(!1),
    [u, a] = k.useState('')
  k.useEffect(() => {
    ;(async () => y(await ge()))()
  }, [])
  const h = m.filter(
      (x) => x.disk_format === 'iso' && /k8s|kubernetes/i.test(x.name) && (!u || x.name.includes(u))
    ),
    f = [
      { key: 'name', header: 'Name' },
      {
        key: 'status',
        header: 'State',
        render: (x) =>
          e.jsx(pe, {
            variant: x.status === 'active' || x.status === 'available' ? 'success' : 'info',
            children: x.status
          })
      },
      {
        key: 'disk_format',
        header: 'OS Type',
        render: () => e.jsx('span', { className: 'uppercase', children: 'ISO' })
      },
      { key: 'sizeGiB', header: 'Size (GiB)' },
      {
        key: 'owner',
        header: 'Account',
        render: (x) => e.jsx('span', { children: x.owner ?? '-' })
      },
      {
        key: 'actions',
        header: 'Actions',
        render: (x) =>
          e.jsx('div', {
            className: 'flex gap-2',
            children: e.jsx('button', {
              className: 'btn-danger btn-xs',
              onClick: async () => {
                ;(await ht(x.id), y(await ge()))
              },
              children: 'Delete'
            })
          })
      }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, { title: 'Images', subtitle: 'Kubernetes ISO' }),
      e.jsxs(Ne, {
        placeholder: 'Search K8s ISO',
        onSearch: a,
        children: [
          e.jsx('button', {
            className: 'btn-secondary',
            onClick: async () => y(await ge()),
            children: 'Refresh'
          }),
          e.jsx('button', {
            className: 'btn-secondary',
            onClick: () => R(!0),
            children: 'Register from URL'
          }),
          e.jsx('button', { className: 'btn-primary', onClick: () => w(!0), children: 'Upload' })
        ]
      }),
      e.jsx(de, { columns: f, data: h, empty: 'No K8s ISOs' }),
      e.jsx(le, {
        title: 'Upload K8s ISO',
        open: g,
        onClose: () => w(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => w(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              disabled: !I || o,
              onClick: async () => {
                if (I) {
                  l(!0)
                  try {
                    ;(await ct(I, { name: I.name }), y(await ge()), w(!1), i(null))
                  } finally {
                    l(!1)
                  }
                }
              },
              children: 'Upload'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsx('input', {
              type: 'file',
              accept: '.iso',
              onChange: (x) => i(x.target.files?.[0] ?? null)
            }),
            e.jsx('p', {
              className: 'text-xs text-muted-foreground',
              children: '上传 Kubernetes 集群节点镜像 ISO，保存到后端的 VC_IMAGE_DIR。'
            })
          ]
        })
      }),
      e.jsx(le, {
        title: 'Register K8s ISO from URL',
        open: T,
        onClose: () => R(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => R(!1),
              children: 'Cancel'
            }),
            e.jsx(Ks, {
              kind: 'iso',
              onDone: async () => {
                ;(y(await ge()), R(!1))
              }
            })
          ]
        }),
        children: e.jsx(qs, { placeholder: 'https://rgw.example.com/bucket/k8s-node.iso' })
      })
    ]
  })
}
function Ks({ kind: m, onDone: y }) {
  const [g, w] = k.useState(!1)
  return e.jsx('button', {
    className: 'btn-primary',
    disabled: g,
    onClick: async () => {
      w(!0)
      try {
        const T = document.getElementById('register-url')?.value || ''
        if (!T) return
        const R = T.split('/').pop() || (m === 'iso' ? 'iso' : 'template'),
          { registerImage: I } = await Ke(
            async () => {
              const { registerImage: i } = await Promise.resolve().then(() => Hs)
              return { registerImage: i }
            },
            void 0
          )
        ;(await I({ name: R, disk_format: m === 'iso' ? 'iso' : 'qcow2', rgw_url: T }), y())
      } finally {
        w(!1)
      }
    },
    children: 'Register'
  })
}
function qs({ placeholder: m }) {
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx('input', { id: 'register-url', className: 'input w-full', placeholder: m }),
      e.jsx('p', {
        className: 'text-xs text-muted-foreground',
        children: '支持 RGW/HTTP URL；注册后可在 Images 页面点击 Import 将其落地到 RBD 或文件路径。'
      })
    ]
  })
}
function Ei() {
  const { roles: m, policies: y, addRole: g, updateRole: w, removeRole: T } = Ce(),
    [R, I] = k.useState(!1),
    [i, o] = k.useState(null),
    l = [
      { key: 'name', header: 'Name' },
      { key: 'roleType', header: 'Role Type' },
      {
        key: 'policyIds',
        header: 'Policies',
        render: (c) => e.jsxs('span', { children: [c.policyIds?.length || 0, ' attached'] })
      },
      {
        key: 'id',
        header: '',
        className: 'w-32 text-right',
        render: (c) =>
          e.jsxs('div', {
            className: 'flex justify-end gap-2',
            children: [
              e.jsx('button', {
                className: 'text-blue-400 hover:underline',
                onClick: () => {
                  ;(o(c.id), I(!0))
                },
                children: 'Policies'
              }),
              e.jsx('button', {
                className: 'text-red-400 hover:underline disabled:opacity-50',
                onClick: () => T(c.id),
                disabled: c.roleType === 'system',
                children: 'Delete'
              })
            ]
          })
      }
    ],
    [u, a] = k.useState(!1),
    [h, f] = k.useState(''),
    x = m.find((c) => c.id === i)
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'IAM - Roles',
        subtitle: 'Roles configuration',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => a(!0),
          children: 'Create Role'
        })
      }),
      e.jsx(de, { columns: l, data: m, empty: 'No roles' }),
      e.jsx(le, {
        title: 'Create Role',
        open: u,
        onClose: () => a(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => a(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: () => {
                h && (g({ name: h, roleType: 'custom', policyIds: [] }), f(''), a(!1))
              },
              children: 'Create'
            })
          ]
        }),
        children: e.jsx('div', {
          className: 'space-y-3',
          children: e.jsxs('div', {
            children: [
              e.jsx('label', { className: 'label', children: 'Name' }),
              e.jsx('input', {
                className: 'input w-full',
                value: h,
                onChange: (c) => f(c.target.value)
              })
            ]
          })
        })
      }),
      e.jsx(le, {
        title: `Manage Policies for ${x?.name}`,
        open: R,
        onClose: () => I(!1),
        footer: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => I(!1),
          children: 'Done'
        }),
        children: e.jsx('div', {
          className: 'space-y-2 max-h-96 overflow-y-auto',
          children: y.map((c) => {
            const t = x?.policyIds?.includes(c.id)
            return e.jsxs(
              'div',
              {
                className: 'flex items-center justify-between p-2 border rounded',
                children: [
                  e.jsxs('div', {
                    children: [
                      e.jsx('div', { className: 'font-medium', children: c.name }),
                      e.jsx('div', { className: 'text-xs text-gray-500', children: c.type })
                    ]
                  }),
                  e.jsx('button', {
                    className: `btn-sm ${t ? 'btn-danger' : 'btn-secondary'}`,
                    onClick: () => {
                      if (!x) return
                      const n = x.policyIds || [],
                        s = t ? n.filter((r) => r !== c.id) : [...n, c.id]
                      w(x.id, { policyIds: s })
                    },
                    children: t ? 'Detach' : 'Attach'
                  })
                ]
              },
              c.id
            )
          })
        })
      })
    ]
  })
}
function Li() {
  const { policies: m, addPolicy: y, removePolicy: g } = Ce(),
    w = [
      { key: 'name', header: 'Name' },
      { key: 'type', header: 'Type' },
      { key: 'description', header: 'Description' },
      {
        key: 'id',
        header: '',
        className: 'w-10 text-right',
        render: (h) =>
          e.jsx('div', {
            className: 'flex justify-end gap-2',
            children: e.jsx('button', {
              className:
                'text-red-400 hover:underline disabled:opacity-50 disabled:cursor-not-allowed',
              onClick: () => g(h.id),
              disabled: h.type === 'system',
              children: 'Delete'
            })
          })
      }
    ],
    [T, R] = k.useState(!1),
    [I, i] = k.useState(''),
    [o, l] = k.useState(''),
    [u, a] = k.useState(`{
  "Version": "2012-10-17",
  "Statement": []
}`)
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'IAM - Policies',
        subtitle: 'Manage access control policies',
        actions: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => R(!0),
          children: 'Create Policy'
        })
      }),
      e.jsx(de, { columns: w, data: m, empty: 'No policies' }),
      e.jsx(le, {
        title: 'Create Policy',
        open: T,
        onClose: () => R(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', {
              className: 'btn-secondary',
              onClick: () => R(!1),
              children: 'Cancel'
            }),
            e.jsx('button', {
              className: 'btn-primary',
              onClick: () => {
                I &&
                  u &&
                  (y({ name: I, description: o, document: u, type: 'custom' }),
                  i(''),
                  l(''),
                  a(`{
  "Version": "2012-10-17",
  "Statement": []
}`),
                  R(!1))
              },
              children: 'Create'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: I,
                  onChange: (h) => i(h.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Description' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: o,
                  onChange: (h) => l(h.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Policy Document (JSON)' }),
                e.jsx('textarea', {
                  className: 'input w-full h-40 font-mono text-sm',
                  value: u,
                  onChange: (h) => a(h.target.value)
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Ri() {
  const { accounts: m, policies: y, updateAccount: g } = Ce(),
    [w, T] = k.useState(!1),
    [R, I] = k.useState(null),
    i = [
      { key: 'name', header: 'Name' },
      { key: 'status', header: 'Status' },
      { key: 'role', header: 'Role' },
      {
        key: 'policyIds',
        header: 'Policies',
        render: (l) => e.jsxs('span', { children: [l.policyIds?.length || 0, ' attached'] })
      },
      { key: 'source', header: 'Source' },
      {
        key: 'id',
        header: '',
        className: 'w-32 text-right',
        render: (l) =>
          e.jsx('div', {
            className: 'flex justify-end gap-2',
            children: e.jsx('button', {
              className: 'text-blue-400 hover:underline',
              onClick: () => {
                ;(I(l.id), T(!0))
              },
              children: 'Policies'
            })
          })
      }
    ],
    o = m.find((l) => l.id === R)
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, { title: 'Accounts', subtitle: 'Current users' }),
      e.jsx(de, { columns: i, data: m, empty: 'No accounts' }),
      e.jsx(le, {
        title: `Manage Policies for ${o?.name}`,
        open: w,
        onClose: () => T(!1),
        footer: e.jsx('button', {
          className: 'btn-primary',
          onClick: () => T(!1),
          children: 'Done'
        }),
        children: e.jsx('div', {
          className: 'space-y-2 max-h-96 overflow-y-auto',
          children: y.map((l) => {
            const u = o?.policyIds?.includes(l.id)
            return e.jsxs(
              'div',
              {
                className: 'flex items-center justify-between p-2 border rounded',
                children: [
                  e.jsxs('div', {
                    children: [
                      e.jsx('div', { className: 'font-medium', children: l.name }),
                      e.jsx('div', { className: 'text-xs text-gray-500', children: l.type })
                    ]
                  }),
                  e.jsx('button', {
                    className: `btn-sm ${u ? 'btn-danger' : 'btn-secondary'}`,
                    onClick: () => {
                      if (!o) return
                      const a = o.policyIds || [],
                        h = u ? a.filter((f) => f !== l.id) : [...a, l.id]
                      g(o.id, { policyIds: h })
                    },
                    children: u ? 'Detach' : 'Attach'
                  })
                ]
              },
              l.id
            )
          })
        })
      })
    ]
  })
}
function es() {
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Infrastructure - Overview',
        subtitle: 'Summary of infrastructure components'
      }),
      e.jsx('div', { className: 'card p-4', children: 'Overview placeholder' })
    ]
  })
}
function Di() {
  const [m, y] = k.useState([]),
    [g, w] = k.useState(!1),
    [T, R] = k.useState(''),
    [I, i] = k.useState(!1),
    [o, l] = k.useState(''),
    [u, a] = k.useState('core'),
    [h, f] = k.useState('Advanced'),
    [x, c] = k.useState('enabled'),
    t = async () => {
      w(!0)
      try {
        const r = await it()
        y(
          r.map((d) => ({
            id: d.id,
            name: d.name,
            allocation: d.allocation,
            type: d.type,
            networkType: d.network_type
          }))
        )
      } finally {
        w(!1)
      }
    }
  k.useEffect(() => {
    t()
  }, [])
  const n = k.useMemo(() => {
      const r = T.trim().toLowerCase()
      return r
        ? m.filter((d) =>
            [d.name, d.type, d.networkType, d.allocation].some((v) =>
              String(v).toLowerCase().includes(r)
            )
          )
        : m
    }, [T, m]),
    s = [
      { key: 'name', header: 'Name' },
      {
        key: 'allocation',
        header: 'Allocation state',
        render: (r) =>
          e.jsx(pe, {
            variant: r.allocation === 'enabled' ? 'success' : 'warning',
            children: r.allocation
          })
      },
      {
        key: 'type',
        header: 'Type',
        render: (r) => e.jsx('span', { className: 'uppercase', children: r.type })
      },
      { key: 'networkType', header: 'Network Type' }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, {
        title: 'Zones',
        subtitle: 'Resource zones',
        actions: e.jsxs('div', {
          className: 'flex items-center gap-2',
          children: [
            e.jsx('button', {
              className: 'btn',
              onClick: t,
              disabled: g,
              children: g ? 'Refreshing…' : 'Refresh'
            }),
            e.jsx('button', {
              className: 'btn btn-primary',
              onClick: () => i(!0),
              children: 'Add Zone'
            }),
            e.jsx(Ne, { placeholder: 'Search zones', onSearch: R })
          ]
        })
      }),
      e.jsx(de, { columns: s, data: n, empty: g ? 'Loading…' : 'No zones' }),
      e.jsx(le, {
        title: 'Add Zone',
        open: I,
        onClose: () => i(!1),
        footer: e.jsxs(e.Fragment, {
          children: [
            e.jsx('button', { className: 'btn', onClick: () => i(!1), children: 'Cancel' }),
            e.jsx('button', {
              className: 'btn btn-primary',
              onClick: async () => {
                if (!o) return
                const r = await Ds({ name: o, type: u, network_type: h, allocation: x })
                ;(y((d) => [
                  ...d,
                  {
                    id: r.id,
                    name: r.name,
                    allocation: r.allocation,
                    type: r.type,
                    networkType: r.network_type
                  }
                ]),
                  l(''),
                  a('core'),
                  f('Advanced'),
                  c('enabled'),
                  i(!1))
              },
              children: 'Save'
            })
          ]
        }),
        children: e.jsxs('div', {
          className: 'space-y-3',
          children: [
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Name' }),
                e.jsx('input', {
                  className: 'input w-full',
                  value: o,
                  onChange: (r) => l(r.target.value)
                })
              ]
            }),
            e.jsxs('div', {
              className: 'grid grid-cols-2 gap-3',
              children: [
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Type' }),
                    e.jsxs('select', {
                      className: 'input w-full',
                      value: u,
                      onChange: (r) => a(r.target.value),
                      children: [
                        e.jsx('option', { value: 'core', children: 'core' }),
                        e.jsx('option', { value: 'edge', children: 'edge' })
                      ]
                    })
                  ]
                }),
                e.jsxs('div', {
                  children: [
                    e.jsx('label', { className: 'label', children: 'Network Type' }),
                    e.jsxs('select', {
                      className: 'input w-full',
                      value: h,
                      onChange: (r) => f(r.target.value),
                      children: [
                        e.jsx('option', { value: 'Basic', children: 'Basic' }),
                        e.jsx('option', { value: 'Advanced', children: 'Advanced' })
                      ]
                    })
                  ]
                })
              ]
            }),
            e.jsxs('div', {
              children: [
                e.jsx('label', { className: 'label', children: 'Allocation state' }),
                e.jsxs('select', {
                  className: 'input w-full',
                  value: x,
                  onChange: (r) => c(r.target.value),
                  children: [
                    e.jsx('option', { value: 'enabled', children: 'enabled' }),
                    e.jsx('option', { value: 'disabled', children: 'disabled' })
                  ]
                })
              ]
            })
          ]
        })
      })
    ]
  })
}
function Ai() {
  return e.jsx('div', {
    className: 'card p-4',
    children: e.jsx(oe, { title: 'Clusters', subtitle: 'Compute clusters' })
  })
}
function Bi() {
  const [m, y] = k.useState(!1),
    [g, w] = k.useState([]),
    [T, R] = k.useState(''),
    [I, i] = k.useState('all'),
    [o, l] = k.useState(!1),
    [u, a] = k.useState(new Set()),
    h = async () => {
      y(!0)
      try {
        const c = await vs()
        w(c)
      } finally {
        y(!1)
      }
    }
  k.useEffect(() => {
    h()
  }, [])
  const f = k.useMemo(() => {
      const c = Date.now()
      return g
        .map((t) => {
          const n = t.last_heartbeat ? new Date(t.last_heartbeat).getTime() : 0,
            s = n > 0 && c - n < 6e4,
            r = t.labels?.disabled !== 'true',
            d = !1,
            v = t.address?.replace(/^https?:\/\//, '').replace(/:.*/, '') || '',
            _ = t.labels?.arch || '',
            b = t.labels?.hypervisor || t.labels?.driver || '',
            p = t.labels?.kernel || t.labels?.version || '',
            S = s ? 'up' : 'down',
            L = r ? 'enabled' : 'disabled'
          return {
            id: t.id,
            name: t.hostname || t.id,
            state: S,
            resourceState: L,
            ip: v,
            arch: _,
            hypervisor: b,
            version: p,
            _alive: s,
            _enabled: r,
            _alarm: d
          }
        })
        .filter((t) => {
          if (T) {
            const n = T.toLowerCase()
            if (!t.name.toLowerCase().includes(n) && !t.ip.includes(T)) return !1
          }
          switch (I) {
            case 'up':
              return t._alive
            case 'down':
              return !t._alive
            case 'enabled':
              return t._enabled
            case 'disabled':
              return !t._enabled
            case 'alarm':
              return t._alarm
            default:
              return !0
          }
        })
    }, [g, T, I]),
    x = [
      {
        key: '__sel__',
        header: '',
        headerRender: e.jsx('input', {
          type: 'checkbox',
          'aria-label': 'Select all',
          checked: f.length > 0 && f.every((c) => u.has(c.id)),
          onChange: (c) => {
            c.target.checked ? a(new Set(f.map((t) => t.id))) : a(new Set())
          }
        }),
        render: (c) =>
          e.jsx('input', {
            type: 'checkbox',
            'aria-label': `Select ${c.name}`,
            checked: u.has(c.id),
            onChange: (t) => {
              ;(t.stopPropagation(),
                a((n) => {
                  const s = new Set(n)
                  return (t.target.checked ? s.add(c.id) : s.delete(c.id), s)
                }))
            },
            onClick: (t) => t.stopPropagation()
          }),
        className: 'w-8'
      },
      { key: 'name', header: 'Name' },
      {
        key: 'state',
        header: 'State',
        render: (c) =>
          e.jsx(pe, { variant: c.state === 'up' ? 'success' : 'danger', children: c.state })
      },
      {
        key: 'resourceState',
        header: 'Resource State',
        render: (c) =>
          e.jsx(pe, {
            variant: c.resourceState === 'enabled' ? 'success' : 'warning',
            children: c.resourceState
          })
      },
      { key: 'ip', header: 'IP' },
      { key: 'arch', header: 'Arch' },
      { key: 'hypervisor', header: 'Hypervisor' },
      { key: 'version', header: 'Version' }
    ]
  return e.jsxs('div', {
    className: 'space-y-3',
    children: [
      e.jsx(oe, { title: 'Hosts', subtitle: 'Hypervisor hosts' }),
      e.jsxs('div', {
        className: 'card p-3 space-y-3',
        children: [
          e.jsxs('div', {
            className: 'flex items-center justify-between gap-3',
            children: [
              e.jsxs('div', {
                className: 'flex items-center gap-2',
                children: [
                  e.jsx('button', {
                    className: 'btn',
                    onClick: h,
                    disabled: m,
                    children: m ? 'Refreshing…' : 'Refresh'
                  }),
                  e.jsxs('select', {
                    className: 'select',
                    value: I,
                    onChange: (c) => i(c.target.value),
                    children: [
                      e.jsx('option', { value: 'all', children: 'All' }),
                      e.jsx('option', { value: 'up', children: 'Up' }),
                      e.jsx('option', { value: 'down', children: 'Down' }),
                      e.jsx('option', { value: 'enabled', children: 'Enabled' }),
                      e.jsx('option', { value: 'disabled', children: 'Disabled' }),
                      e.jsx('option', { value: 'alarm', children: 'Alarm' })
                    ]
                  })
                ]
              }),
              e.jsxs('div', {
                className: 'flex items-center gap-2',
                children: [
                  u.size > 0 &&
                    e.jsx('button', {
                      className: 'btn btn-danger',
                      onClick: async () => {
                        if (confirm(`Delete ${u.size} host(s) from scheduler?`))
                          try {
                            ;(await Promise.all(Array.from(u).map((c) => xs(c))),
                              me.success(`Deleted ${u.size} host(s)`),
                              a(new Set()),
                              await h())
                          } catch {
                            me.error('Delete failed')
                          }
                      },
                      children: 'Delete Selected'
                    }),
                  e.jsx('button', {
                    className: 'btn btn-primary',
                    onClick: () => l(!0),
                    children: 'Add Host'
                  }),
                  e.jsx(Ne, { placeholder: 'Search by IP or hostname', onSearch: R })
                ]
              })
            ]
          }),
          e.jsx(de, {
            columns: x,
            data: f,
            empty: m ? 'Loading…' : 'No hosts',
            onRowClick: (c) => {
              const t = c
              a((n) => {
                const s = new Set(n)
                return (s.has(t.id) ? s.delete(t.id) : s.add(t.id), s)
              })
            },
            isRowSelected: (c) => u.has(c.id)
          })
        ]
      }),
      e.jsx(le, {
        title: 'Add Host',
        open: o,
        onClose: () => l(!1),
        footer: e.jsx('button', { className: 'btn', onClick: () => l(!1), children: 'Close' }),
        children: e.jsxs('div', {
          className: 'text-sm text-gray-300 space-y-2',
          children: [
            e.jsx('p', { children: 'On the target node (Debian 12):' }),
            e.jsxs('ol', {
              className: 'list-decimal list-inside space-y-1',
              children: [
                e.jsx('li', {
                  children:
                    'Install dependencies: qemu-kvm, libvirt-daemon-system, libvirt-clients; enable libvirtd.'
                }),
                e.jsx('li', {
                  children: 'Copy vc-lite binary to /opt/tiger/bin/ and make it executable.'
                }),
                e.jsx('li', { children: 'Create /opt/tiger/configs/env with:' })
              ]
            }),
            e.jsx('pre', {
              className: 'bg-oxide-950 border border-oxide-800 rounded p-2 text-xs overflow-auto',
              children: `VC_SCHEDULER_URL=http://<control-ip>:8092
VC_LITE_PUBLIC_URL=http://<node-ip>:8091
# Optional:
LIBVIRT_URI=qemu:///system
VC_NODE_ID=<unique-id>`
            }),
            e.jsx('p', {
              children: 'Create systemd unit /etc/systemd/system/vc-lite.service and start it:'
            }),
            e.jsx('pre', {
              className: 'bg-oxide-950 border border-oxide-800 rounded p-2 text-xs overflow-auto',
              children: `[Unit]
Description=VC Stack Lite (Node Agent)
After=network-online.target libvirtd.service
Wants=network-online.target

[Service]
User=tiger
Group=tiger
EnvironmentFile=-/opt/tiger/configs/env
ExecStart=/opt/tiger/bin/vc-lite
WorkingDirectory=/opt/tiger
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target`
            }),
            e.jsx('p', { children: 'Once started, click Refresh to see the host appear here.' })
          ]
        })
      })
    ]
  })
}
function Ii() {
  return e.jsx('div', {
    className: 'card p-4',
    children: e.jsx(oe, { title: 'Primary Storage (S3)', subtitle: 'Primary storage backends' })
  })
}
function Mi() {
  return e.jsx('div', {
    className: 'card p-4',
    children: e.jsx(oe, {
      title: 'Secondary Storage (Ceph RBD)',
      subtitle: 'Secondary storage backends'
    })
  })
}
function Pi() {
  return e.jsx('div', {
    className: 'card p-4',
    children: e.jsx(oe, { title: 'DB / Usage Server', subtitle: 'Database and usage services' })
  })
}
function Ti() {
  return e.jsx('div', {
    className: 'card p-4',
    children: e.jsx(oe, { title: 'Alarms', subtitle: 'Infrastructure alarms' })
  })
}
function Oi() {
  return e.jsx('div', {
    className: 'space-y-4',
    children: e.jsxs(Ve, {
      children: [
        e.jsx(te, { path: '', element: e.jsx(St, { to: 'overview', replace: !0 }) }),
        e.jsx(te, { path: 'overview', element: e.jsx(es, {}) }),
        e.jsx(te, { path: 'zones', element: e.jsx(Di, {}) }),
        e.jsx(te, { path: 'clusters', element: e.jsx(Ai, {}) }),
        e.jsx(te, { path: 'hosts', element: e.jsx(Bi, {}) }),
        e.jsx(te, { path: 'primary-storage', element: e.jsx(Ii, {}) }),
        e.jsx(te, { path: 'secondary-storage', element: e.jsx(Mi, {}) }),
        e.jsx(te, { path: 'db-usage', element: e.jsx(Pi, {}) }),
        e.jsx(te, { path: 'alarms', element: e.jsx(Ti, {}) }),
        e.jsx(te, { path: '*', element: e.jsx(es, {}) })
      ]
    })
  })
}
function Hi({ children: m }) {
  const y = lt((I) => I.token),
    g = at(),
    [w, T] = k.useState(!1),
    R = (I) => {
      console.log(I)
      try {
        const i = JSON.parse(localStorage.getItem('debug_logs') || '[]')
        ;(i.push({ time: new Date().toISOString(), msg: I }),
          i.length > 50 && i.shift(),
          localStorage.setItem('debug_logs', JSON.stringify(i)))
      } catch {}
    }
  return (
    k.useEffect(() => {
      ;(async () => {
        try {
          const i = localStorage.getItem('auth')
          ;(R(`[RequireAuth] Checking localStorage auth: ${i ? 'Found' : 'Not found'}`),
            i &&
              JSON.parse(i)?.state?.token &&
              (R('[RequireAuth] Token found in localStorage, waiting for Zustand hydration...'),
              await new Promise((l) => setTimeout(l, 100))))
        } catch {}
        T(!0)
      })()
    }, []),
    w
      ? (R(
          `[RequireAuth] Ready. Token in Zustand store: ${y ? 'Found' : 'Not found'} at ${g.pathname}`
        ),
        y
          ? (R(`[RequireAuth] Authenticated, rendering children for: ${g.pathname}`),
            e.jsx(e.Fragment, { children: m }))
          : (R(`[RequireAuth] No token, redirecting to /login from: ${g.pathname}`),
            e.jsx(St, { to: '/login', state: { from: g }, replace: !0 })))
      : null
  )
}
function Fi() {
  return (
    k.useEffect(() => {
      console.log('[App] VC Console loaded - Version: 2025-12-08-23:20')
    }, []),
    e.jsxs(Ve, {
      children: [
        e.jsx(te, { path: '/login', element: e.jsx(Gr, {}) }),
        e.jsx(te, { path: '/auth/oidc/callback', element: e.jsx(Kr, {}) }),
        e.jsx(te, {
          path: '/*',
          element: e.jsx(Hi, {
            children: e.jsx(ur, {
              children: e.jsxs(Ve, {
                children: [
                  e.jsx(te, { path: '/', element: e.jsx(St, { to: '/projects', replace: !0 }) }),
                  e.jsx(te, { path: '/docs', element: e.jsx(Xr, {}) }),
                  e.jsx(te, { path: '/images', element: e.jsx(Ft, {}) }),
                  e.jsx(te, { path: '/utilization', element: e.jsx($t, {}) }),
                  e.jsx(te, { path: '/webshell', element: e.jsx(Zr, {}) }),
                  e.jsx(te, { path: '/webshell/sessions', element: e.jsx(Si, {}) }),
                  e.jsx(te, { path: '/webshell/replay/:sessionId', element: e.jsx(ji, {}) }),
                  e.jsx(te, { path: '/images/templates', element: e.jsx(Yt, {}) }),
                  e.jsx(te, { path: '/images/iso', element: e.jsx(Zt, {}) }),
                  e.jsx(te, { path: '/images/k8s-iso', element: e.jsx(Qt, {}) }),
                  e.jsx(te, { path: '/iam/roles', element: e.jsx(Ei, {}) }),
                  e.jsx(te, { path: '/iam/policies', element: e.jsx(Li, {}) }),
                  e.jsx(te, { path: '/accounts', element: e.jsx(Ri, {}) }),
                  e.jsx(te, {
                    path: '/project/:projectId/infrastructure/*',
                    element: e.jsx(Oi, {})
                  }),
                  e.jsx(te, { path: '/projects/*', element: e.jsx(zr, {}) }),
                  e.jsx(te, { path: '/project/:projectId', element: e.jsx(Ni, {}) }),
                  e.jsx(te, { path: '/project/:projectId/images', element: e.jsx(Ft, {}) }),
                  e.jsx(te, { path: '/project/:projectId/utilization', element: e.jsx($t, {}) }),
                  e.jsx(te, {
                    path: '/project/:projectId/images/templates',
                    element: e.jsx(Yt, {})
                  }),
                  e.jsx(te, { path: '/project/:projectId/images/iso', element: e.jsx(Zt, {}) }),
                  e.jsx(te, { path: '/project/:projectId/images/k8s-iso', element: e.jsx(Qt, {}) }),
                  e.jsx(te, { path: '/project/:projectId/compute/*', element: e.jsx(Or, {}) }),
                  e.jsx(te, { path: '/project/:projectId/network/*', element: e.jsx(Cr, {}) }),
                  e.jsx(te, { path: '/project/:projectId/storage/*', element: e.jsx(Fr, {}) }),
                  e.jsx(te, { path: '/settings/*', element: e.jsx(gr, {}) }),
                  e.jsx(te, { path: '/notifications', element: e.jsx(Vr, {}) })
                ]
              })
            })
          })
        })
      ]
    })
  )
}
pt.createRoot(document.getElementById('root')).render(
  e.jsx(Zs.StrictMode, { children: e.jsx(Qs, { children: e.jsx(Fi, {}) }) })
)
