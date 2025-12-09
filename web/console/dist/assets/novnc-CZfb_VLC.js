import { g as Lo } from './react-vendor-B-2j4S7D.js'
var la = {},
  Sr = {}
Object.defineProperty(Sr, '__esModule', { value: !0 })
Sr.toSigned32bit = Mo
Sr.toUnsigned32bit = Ro
function Ro(e) {
  return e >>> 0
}
function Mo(e) {
  return e | 0
}
var Re = {}
Object.defineProperty(Re, '__esModule', { value: !0 })
Re.Warn = Re.Info = Re.Error = Re.Debug = void 0
Re.getLogging = Do
Re.initLogging = fa
var gn = 'warn'
Re.Debug = function () {}
Re.Info = function () {}
Re.Warn = function () {}
Re.Error = function () {}
function fa(e) {
  if (
    (typeof e > 'u' ? (e = gn) : (gn = e),
    (Re.Debug = Re.Info = Re.Warn = Re.Error = function () {}),
    typeof window.console < 'u')
  )
    switch (e) {
      case 'debug':
        Re.Debug = console.debug.bind(window.console)
      case 'info':
        Re.Info = console.info.bind(window.console)
      case 'warn':
        Re.Warn = console.warn.bind(window.console)
      case 'error':
        Re.Error = console.error.bind(window.console)
      case 'none':
        break
      default:
        throw new window.Error("invalid logging type '" + e + "'")
    }
}
function Do() {
  return gn
}
fa()
var Er = {}
Object.defineProperty(Er, '__esModule', { value: !0 })
Er.decodeUTF8 = Oo
Er.encodeUTF8 = Bo
function Oo(e) {
  var t = arguments.length > 1 && arguments[1] !== void 0 ? arguments[1] : !1
  try {
    return decodeURIComponent(escape(e))
  } catch (n) {
    if (n instanceof URIError && t) return e
    throw n
  }
}
function Bo(e) {
  return unescape(encodeURIComponent(e))
}
var Ae = {},
  Zr = {}
;(function (e) {
  function t(o) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (a) {
              return typeof a
            }
          : function (a) {
              return a &&
                typeof Symbol == 'function' &&
                a.constructor === Symbol &&
                a !== Symbol.prototype
                ? 'symbol'
                : typeof a
            }),
      t(o)
    )
  }
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var n = i(Re)
  function r(o) {
    if (typeof WeakMap != 'function') return null
    var a = new WeakMap(),
      s = new WeakMap()
    return (r = function (u) {
      return u ? s : a
    })(o)
  }
  function i(o, a) {
    if (o && o.__esModule) return o
    if (o === null || (t(o) != 'object' && typeof o != 'function')) return { default: o }
    var s = r(a)
    if (s && s.has(o)) return s.get(o)
    var x = { __proto__: null },
      u = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var f in o)
      if (f !== 'default' && {}.hasOwnProperty.call(o, f)) {
        var F = u ? Object.getOwnPropertyDescriptor(o, f) : null
        F && (F.get || F.set) ? Object.defineProperty(x, f, F) : (x[f] = o[f])
      }
    return ((x.default = o), s && s.set(o, x), x)
  }
  e.default = {
    toBase64Table: 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/='.split(''),
    base64Pad: '=',
    encode: function (a) {
      for (var s = '', x = a.length, u = x % 3, f = 0; f < x - 2; f += 3)
        ((s += this.toBase64Table[a[f] >> 2]),
          (s += this.toBase64Table[((a[f] & 3) << 4) + (a[f + 1] >> 4)]),
          (s += this.toBase64Table[((a[f + 1] & 15) << 2) + (a[f + 2] >> 6)]),
          (s += this.toBase64Table[a[f + 2] & 63]))
      var F = x - u
      return (
        u === 2
          ? ((s += this.toBase64Table[a[F] >> 2]),
            (s += this.toBase64Table[((a[F] & 3) << 4) + (a[F + 1] >> 4)]),
            (s += this.toBase64Table[(a[F + 1] & 15) << 2]),
            (s += this.toBase64Table[64]))
          : u === 1 &&
            ((s += this.toBase64Table[a[F] >> 2]),
            (s += this.toBase64Table[(a[F] & 3) << 4]),
            (s += this.toBase64Table[64]),
            (s += this.toBase64Table[64])),
        s
      )
    },
    toBinaryTable: [
      -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
      -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 62, -1, -1,
      -1, 63, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, -1, -1, -1, 0, -1, -1, -1, 0, 1, 2, 3, 4, 5,
      6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, -1, -1, -1, -1,
      -1, -1, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46,
      47, 48, 49, 50, 51, -1, -1, -1, -1, -1
    ],
    decode: function (a) {
      var s = arguments.length > 1 && arguments[1] !== void 0 ? arguments[1] : 0,
        x = a.indexOf('=') - s
      x < 0 && (x = a.length - s)
      for (
        var u = (x >> 2) * 3 + Math.floor((x % 4) / 1.5),
          f = new Array(u),
          F = 0,
          X = 0,
          g = 0,
          k = s;
        k < a.length;
        k++
      ) {
        var _ = this.toBinaryTable[a.charCodeAt(k) & 127],
          l = a.charAt(k) === this.base64Pad
        if (_ === -1) {
          n.Error('Illegal character code ' + a.charCodeAt(k) + ' at position ' + k)
          continue
        }
        ;((X = (X << 6) | _),
          (F += 6),
          F >= 8 && ((F -= 8), l || (f[g++] = (X >> F) & 255), (X &= (1 << F) - 1)))
      }
      if (F) {
        var h = new Error('Corrupted base64 string')
        throw ((h.name = 'Base64-Error'), h)
      }
      return f
    }
  }
})(Zr)
Object.defineProperty(Ae, '__esModule', { value: !0 })
Ae.hasScrollbarGutter = Ae.dragThreshold = void 0
Ae.isAndroid = Vo
Ae.isBlink = is
Ae.isChrome = $o
Ae.isChromeOS = Zo
Ae.isChromium = Jo
Ae.isEdge = ts
Ae.isFirefox = Yo
Ae.isGecko = rs
Ae.isIOS = Go
Ae.isMac = Wo
Ae.isOpera = es
Ae.isSafari = qo
Ae.isTouchDevice = void 0
Ae.isWebKit = ns
Ae.isWindows = zo
Ae.supportsWebCodecsH264Decode = Ae.supportsCursorURIs = void 0
var Ur = Uo(Re),
  Io = Qo(Zr)
function Qo(e) {
  return e && e.__esModule ? e : { default: e }
}
function ca(e) {
  if (typeof WeakMap != 'function') return null
  var t = new WeakMap(),
    n = new WeakMap()
  return (ca = function (i) {
    return i ? n : t
  })(e)
}
function Uo(e, t) {
  if (e && e.__esModule) return e
  if (e === null || (yr(e) != 'object' && typeof e != 'function')) return { default: e }
  var n = ca(t)
  if (n && n.has(e)) return n.get(e)
  var r = { __proto__: null },
    i = Object.defineProperty && Object.getOwnPropertyDescriptor
  for (var o in e)
    if (o !== 'default' && {}.hasOwnProperty.call(e, o)) {
      var a = i ? Object.getOwnPropertyDescriptor(e, o) : null
      a && (a.get || a.set) ? Object.defineProperty(r, o, a) : (r[o] = e[o])
    }
  return ((r.default = e), n && n.set(e, r), r)
}
function yr(e) {
  '@babel/helpers - typeof'
  return (
    (yr =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    yr(e)
  )
}
function bn() {
  /*! regenerator-runtime -- Copyright (c) 2014-present, Facebook, Inc. -- license (MIT): https://github.com/facebook/regenerator/blob/main/LICENSE */ bn =
    function () {
      return t
    }
  var e,
    t = {},
    n = Object.prototype,
    r = n.hasOwnProperty,
    i =
      Object.defineProperty ||
      function (S, p, b) {
        S[p] = b.value
      },
    o = typeof Symbol == 'function' ? Symbol : {},
    a = o.iterator || '@@iterator',
    s = o.asyncIterator || '@@asyncIterator',
    x = o.toStringTag || '@@toStringTag'
  function u(S, p, b) {
    return (
      Object.defineProperty(S, p, { value: b, enumerable: !0, configurable: !0, writable: !0 }),
      S[p]
    )
  }
  try {
    u({}, '')
  } catch {
    u = function (b, L, B) {
      return (b[L] = B)
    }
  }
  function f(S, p, b, L) {
    var B = p && p.prototype instanceof h ? p : h,
      M = Object.create(B.prototype),
      V = new $(L || [])
    return (i(M, '_invoke', { value: O(S, b, V) }), M)
  }
  function F(S, p, b) {
    try {
      return { type: 'normal', arg: S.call(p, b) }
    } catch (L) {
      return { type: 'throw', arg: L }
    }
  }
  t.wrap = f
  var X = 'suspendedStart',
    g = 'suspendedYield',
    k = 'executing',
    _ = 'completed',
    l = {}
  function h() {}
  function c() {}
  function E() {}
  var y = {}
  u(y, a, function () {
    return this
  })
  var K = Object.getPrototypeOf,
    w = K && K(K(ne([])))
  w && w !== n && r.call(w, a) && (y = w)
  var m = (E.prototype = h.prototype = Object.create(y))
  function C(S) {
    ;['next', 'throw', 'return'].forEach(function (p) {
      u(S, p, function (b) {
        return this._invoke(p, b)
      })
    })
  }
  function A(S, p) {
    function b(B, M, V, re) {
      var oe = F(S[B], S, M)
      if (oe.type !== 'throw') {
        var ce = oe.arg,
          pe = ce.value
        return pe && yr(pe) == 'object' && r.call(pe, '__await')
          ? p.resolve(pe.__await).then(
              function (ye) {
                b('next', ye, V, re)
              },
              function (ye) {
                b('throw', ye, V, re)
              }
            )
          : p.resolve(pe).then(
              function (ye) {
                ;((ce.value = ye), V(ce))
              },
              function (ye) {
                return b('throw', ye, V, re)
              }
            )
      }
      re(oe.arg)
    }
    var L
    i(this, '_invoke', {
      value: function (M, V) {
        function re() {
          return new p(function (oe, ce) {
            b(M, V, oe, ce)
          })
        }
        return (L = L ? L.then(re, re) : re())
      }
    })
  }
  function O(S, p, b) {
    var L = X
    return function (B, M) {
      if (L === k) throw Error('Generator is already running')
      if (L === _) {
        if (B === 'throw') throw M
        return { value: e, done: !0 }
      }
      for (b.method = B, b.arg = M; ; ) {
        var V = b.delegate
        if (V) {
          var re = I(V, b)
          if (re) {
            if (re === l) continue
            return re
          }
        }
        if (b.method === 'next') b.sent = b._sent = b.arg
        else if (b.method === 'throw') {
          if (L === X) throw ((L = _), b.arg)
          b.dispatchException(b.arg)
        } else b.method === 'return' && b.abrupt('return', b.arg)
        L = k
        var oe = F(S, p, b)
        if (oe.type === 'normal') {
          if (((L = b.done ? _ : g), oe.arg === l)) continue
          return { value: oe.arg, done: b.done }
        }
        oe.type === 'throw' && ((L = _), (b.method = 'throw'), (b.arg = oe.arg))
      }
    }
  }
  function I(S, p) {
    var b = p.method,
      L = S.iterator[b]
    if (L === e)
      return (
        (p.delegate = null),
        (b === 'throw' &&
          S.iterator.return &&
          ((p.method = 'return'), (p.arg = e), I(S, p), p.method === 'throw')) ||
          (b !== 'return' &&
            ((p.method = 'throw'),
            (p.arg = new TypeError("The iterator does not provide a '" + b + "' method")))),
        l
      )
    var B = F(L, S.iterator, p.arg)
    if (B.type === 'throw') return ((p.method = 'throw'), (p.arg = B.arg), (p.delegate = null), l)
    var M = B.arg
    return M
      ? M.done
        ? ((p[S.resultName] = M.value),
          (p.next = S.nextLoc),
          p.method !== 'return' && ((p.method = 'next'), (p.arg = e)),
          (p.delegate = null),
          l)
        : M
      : ((p.method = 'throw'),
        (p.arg = new TypeError('iterator result is not an object')),
        (p.delegate = null),
        l)
  }
  function Z(S) {
    var p = { tryLoc: S[0] }
    ;(1 in S && (p.catchLoc = S[1]),
      2 in S && ((p.finallyLoc = S[2]), (p.afterLoc = S[3])),
      this.tryEntries.push(p))
  }
  function q(S) {
    var p = S.completion || {}
    ;((p.type = 'normal'), delete p.arg, (S.completion = p))
  }
  function $(S) {
    ;((this.tryEntries = [{ tryLoc: 'root' }]), S.forEach(Z, this), this.reset(!0))
  }
  function ne(S) {
    if (S || S === '') {
      var p = S[a]
      if (p) return p.call(S)
      if (typeof S.next == 'function') return S
      if (!isNaN(S.length)) {
        var b = -1,
          L = function B() {
            for (; ++b < S.length; ) if (r.call(S, b)) return ((B.value = S[b]), (B.done = !1), B)
            return ((B.value = e), (B.done = !0), B)
          }
        return (L.next = L)
      }
    }
    throw new TypeError(yr(S) + ' is not iterable')
  }
  return (
    (c.prototype = E),
    i(m, 'constructor', { value: E, configurable: !0 }),
    i(E, 'constructor', { value: c, configurable: !0 }),
    (c.displayName = u(E, x, 'GeneratorFunction')),
    (t.isGeneratorFunction = function (S) {
      var p = typeof S == 'function' && S.constructor
      return !!p && (p === c || (p.displayName || p.name) === 'GeneratorFunction')
    }),
    (t.mark = function (S) {
      return (
        Object.setPrototypeOf
          ? Object.setPrototypeOf(S, E)
          : ((S.__proto__ = E), u(S, x, 'GeneratorFunction')),
        (S.prototype = Object.create(m)),
        S
      )
    }),
    (t.awrap = function (S) {
      return { __await: S }
    }),
    C(A.prototype),
    u(A.prototype, s, function () {
      return this
    }),
    (t.AsyncIterator = A),
    (t.async = function (S, p, b, L, B) {
      B === void 0 && (B = Promise)
      var M = new A(f(S, p, b, L), B)
      return t.isGeneratorFunction(p)
        ? M
        : M.next().then(function (V) {
            return V.done ? V.value : M.next()
          })
    }),
    C(m),
    u(m, x, 'Generator'),
    u(m, a, function () {
      return this
    }),
    u(m, 'toString', function () {
      return '[object Generator]'
    }),
    (t.keys = function (S) {
      var p = Object(S),
        b = []
      for (var L in p) b.push(L)
      return (
        b.reverse(),
        function B() {
          for (; b.length; ) {
            var M = b.pop()
            if (M in p) return ((B.value = M), (B.done = !1), B)
          }
          return ((B.done = !0), B)
        }
      )
    }),
    (t.values = ne),
    ($.prototype = {
      constructor: $,
      reset: function (p) {
        if (
          ((this.prev = 0),
          (this.next = 0),
          (this.sent = this._sent = e),
          (this.done = !1),
          (this.delegate = null),
          (this.method = 'next'),
          (this.arg = e),
          this.tryEntries.forEach(q),
          !p)
        )
          for (var b in this)
            b.charAt(0) === 't' && r.call(this, b) && !isNaN(+b.slice(1)) && (this[b] = e)
      },
      stop: function () {
        this.done = !0
        var p = this.tryEntries[0].completion
        if (p.type === 'throw') throw p.arg
        return this.rval
      },
      dispatchException: function (p) {
        if (this.done) throw p
        var b = this
        function L(ce, pe) {
          return (
            (V.type = 'throw'),
            (V.arg = p),
            (b.next = ce),
            pe && ((b.method = 'next'), (b.arg = e)),
            !!pe
          )
        }
        for (var B = this.tryEntries.length - 1; B >= 0; --B) {
          var M = this.tryEntries[B],
            V = M.completion
          if (M.tryLoc === 'root') return L('end')
          if (M.tryLoc <= this.prev) {
            var re = r.call(M, 'catchLoc'),
              oe = r.call(M, 'finallyLoc')
            if (re && oe) {
              if (this.prev < M.catchLoc) return L(M.catchLoc, !0)
              if (this.prev < M.finallyLoc) return L(M.finallyLoc)
            } else if (re) {
              if (this.prev < M.catchLoc) return L(M.catchLoc, !0)
            } else {
              if (!oe) throw Error('try statement without catch or finally')
              if (this.prev < M.finallyLoc) return L(M.finallyLoc)
            }
          }
        }
      },
      abrupt: function (p, b) {
        for (var L = this.tryEntries.length - 1; L >= 0; --L) {
          var B = this.tryEntries[L]
          if (B.tryLoc <= this.prev && r.call(B, 'finallyLoc') && this.prev < B.finallyLoc) {
            var M = B
            break
          }
        }
        M && (p === 'break' || p === 'continue') && M.tryLoc <= b && b <= M.finallyLoc && (M = null)
        var V = M ? M.completion : {}
        return (
          (V.type = p),
          (V.arg = b),
          M ? ((this.method = 'next'), (this.next = M.finallyLoc), l) : this.complete(V)
        )
      },
      complete: function (p, b) {
        if (p.type === 'throw') throw p.arg
        return (
          p.type === 'break' || p.type === 'continue'
            ? (this.next = p.arg)
            : p.type === 'return'
              ? ((this.rval = this.arg = p.arg), (this.method = 'return'), (this.next = 'end'))
              : p.type === 'normal' && b && (this.next = b),
          l
        )
      },
      finish: function (p) {
        for (var b = this.tryEntries.length - 1; b >= 0; --b) {
          var L = this.tryEntries[b]
          if (L.finallyLoc === p) return (this.complete(L.completion, L.afterLoc), q(L), l)
        }
      },
      catch: function (p) {
        for (var b = this.tryEntries.length - 1; b >= 0; --b) {
          var L = this.tryEntries[b]
          if (L.tryLoc === p) {
            var B = L.completion
            if (B.type === 'throw') {
              var M = B.arg
              q(L)
            }
            return M
          }
        }
        throw Error('illegal catch attempt')
      },
      delegateYield: function (p, b, L) {
        return (
          (this.delegate = { iterator: ne(p), resultName: b, nextLoc: L }),
          this.method === 'next' && (this.arg = e),
          l
        )
      }
    }),
    t
  )
}
function oi(e, t, n, r, i, o, a) {
  try {
    var s = e[o](a),
      x = s.value
  } catch (u) {
    return void n(u)
  }
  s.done ? t(x) : Promise.resolve(x).then(r, i)
}
function No(e) {
  return function () {
    var t = this,
      n = arguments
    return new Promise(function (r, i) {
      var o = e.apply(t, n)
      function a(x) {
        oi(o, r, i, a, s, 'next', x)
      }
      function s(x) {
        oi(o, r, i, a, s, 'throw', x)
      }
      a(void 0)
    })
  }
}
Ae.isTouchDevice =
  'ontouchstart' in document.documentElement ||
  document.ontouchstart !== void 0 ||
  navigator.maxTouchPoints > 0 ||
  navigator.msMaxTouchPoints > 0
window.addEventListener(
  'touchstart',
  function e() {
    ;((Ae.isTouchDevice = !0), window.removeEventListener('touchstart', e, !1))
  },
  !1
)
Ae.dragThreshold = 10 * (window.devicePixelRatio || 1)
var ha = !1
try {
  var si = document.createElement('canvas')
  ;((si.style.cursor =
    'url("data:image/x-icon;base64,AAACAAEACAgAAAIAAgA4AQAAFgAAACgAAAAIAAAAEAAAAAEAIAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAD/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////AAAAAAAAAAAAAAAAAAAAAA==") 2 2, default'),
    si.style.cursor.indexOf('url') === 0
      ? (Ur.Info('Data URI scheme cursor supported'), (ha = !0))
      : Ur.Warn('Data URI scheme cursor not supported'))
} catch (e) {
  Ur.Error('Data URI scheme cursor test exception: ' + e)
}
Ae.supportsCursorURIs = ha
var da = !0
try {
  var jt = document.createElement('div')
  ;((jt.style.visibility = 'hidden'), (jt.style.overflow = 'scroll'), document.body.appendChild(jt))
  var ui = document.createElement('div')
  jt.appendChild(ui)
  var jo = jt.offsetWidth - ui.offsetWidth
  ;(jt.parentNode.removeChild(jt), (da = jo != 0))
} catch (e) {
  Ur.Error('Scrollbar test exception: ' + e)
}
Ae.hasScrollbarGutter = da
Ae.supportsWebCodecsH264Decode = !1
function Ho() {
  return mn.apply(this, arguments)
}
function mn() {
  return (
    (mn = No(
      bn().mark(function e() {
        var t, n, r, i, o, a, s
        return bn().wrap(
          function (u) {
            for (;;)
              switch ((u.prev = u.next)) {
                case 0:
                  if ('VideoDecoder' in window) {
                    u.next = 2
                    break
                  }
                  return u.abrupt('return', !1)
                case 2:
                  return (
                    (t = {
                      codec: 'avc1.42401f',
                      codedWidth: 1920,
                      codedHeight: 1080,
                      optimizeForLatency: !0
                    }),
                    (u.next = 5),
                    VideoDecoder.isConfigSupported(t)
                  )
                case 5:
                  if (((n = u.sent), n.supported)) {
                    u.next = 8
                    break
                  }
                  return u.abrupt('return', !1)
                case 8:
                  return (
                    (r = new Uint8Array(
                      Io.default.decode(
                        'AAAAAWdCwBTZnpuAgICgAAADACAAAAZB4oVNAAAAAWjJYyyAAAABBgX//4HcRem95tlIt5Ys2CDZI+7veDI2NCAtIGNvcmUgMTY0IHIzMTA4IDMxZTE5ZjkgLSBILjI2NC9NUEVHLTQgQVZDIGNvZGVjIC0gQ29weWxlZnQgMjAwMy0yMDIzIC0gaHR0cDovL3d3dy52aWRlb2xhbi5vcmcveDI2NC5odG1sIC0gb3B0aW9uczogY2FiYWM9MCByZWY9NSBkZWJsb2NrPTE6MDowIGFuYWx5c2U9MHgxOjB4MTExIG1lPWhleCBzdWJtZT04IHBzeT0xIHBzeV9yZD0xLjAwOjAuMDAgbWl4ZWRfcmVmPTEgbWVfcmFuZ2U9MTYgY2hyb21hX21lPTEgdHJlbGxpcz0yIDh4OGRjdD0wIGNxbT0wIGRlYWR6b25lPTIxLDExIGZhc3RfcHNraXA9MSBjaHJvbWFfcXBfb2Zmc2V0PS0yIHRocmVhZHM9MSBsb29rYWhlYWRfdGhyZWFkcz0xIHNsaWNlZF90aHJlYWRzPTAgbnI9MCBkZWNpbWF0ZT0xIGludGVybGFjZWQ9MCBibHVyYXlfY29tcGF0PTAgY29uc3RyYWluZWRfaW50cmE9MCBiZnJhbWVzPTAgd2VpZ2h0cD0wIGtleWludD1pbmZpbml0ZSBrZXlpbnRfbWluPTI1IHNjZW5lY3V0PTQwIGludHJhX3JlZnJlc2g9MCByY19sb29rYWhlYWQ9NTAgcmM9YWJyIG1idHJlZT0xIGJpdHJhdGU9NDAwIHJhdGV0b2w9MS4wIHFjb21wPTAuNjAgcXBtaW49MCBxcG1heD02OSBxcHN0ZXA9NCBpcF9yYXRpbz0xLjQwIGFxPTE6MS4wMACAAAABZYiEBrxmKAAPVccAAS044AA5DRJMnkycJk4TPw=='
                      )
                    )),
                    (i = !1),
                    (o = null),
                    (a = new VideoDecoder({
                      output: function (F) {
                        i = !0
                      },
                      error: function (F) {
                        o = F
                      }
                    })),
                    (s = new EncodedVideoChunk({ timestamp: 0, type: 'key', data: r })),
                    a.configure(t),
                    a.decode(s),
                    (u.prev = 15),
                    (u.next = 18),
                    a.flush()
                  )
                case 18:
                  u.next = 23
                  break
                case 20:
                  ;((u.prev = 20), (u.t0 = u.catch(15)), (o = u.t0))
                case 23:
                  if (i) {
                    u.next = 25
                    break
                  }
                  return u.abrupt('return', !1)
                case 25:
                  if (o === null) {
                    u.next = 27
                    break
                  }
                  return u.abrupt('return', !1)
                case 27:
                  return u.abrupt('return', !0)
                case 28:
                case 'end':
                  return u.stop()
              }
          },
          e,
          null,
          [[15, 20]]
        )
      })
    )),
    mn.apply(this, arguments)
  )
}
Ae.supportsWebCodecsH264Decode = await Ho()
function Wo() {
  return !!/mac/i.exec(navigator.platform)
}
function zo() {
  return !!/win/i.exec(navigator.platform)
}
function Go() {
  return (
    !!/ipad/i.exec(navigator.platform) ||
    !!/iphone/i.exec(navigator.platform) ||
    !!/ipod/i.exec(navigator.platform)
  )
}
function Vo() {
  return !!navigator.userAgent.match('Android ')
}
function Zo() {
  return !!navigator.userAgent.match(' CrOS ')
}
function qo() {
  return (
    !!navigator.userAgent.match('Safari/...') &&
    !navigator.userAgent.match('Chrome/...') &&
    !navigator.userAgent.match('Chromium/...') &&
    !navigator.userAgent.match('Epiphany/...')
  )
}
function Yo() {
  return !!navigator.userAgent.match('Firefox/...') && !navigator.userAgent.match('Seamonkey/...')
}
function $o() {
  return (
    !!navigator.userAgent.match('Chrome/...') &&
    !navigator.userAgent.match('Chromium/...') &&
    !navigator.userAgent.match('Edg/...') &&
    !navigator.userAgent.match('OPR/...')
  )
}
function Jo() {
  return !!navigator.userAgent.match('Chromium/...')
}
function es() {
  return !!navigator.userAgent.match('OPR/...')
}
function ts() {
  return !!navigator.userAgent.match('Edg/...')
}
function rs() {
  return !!navigator.userAgent.match('Gecko/...')
}
function ns() {
  return !!navigator.userAgent.match('AppleWebKit/...') && !navigator.userAgent.match('Chrome/...')
}
function is() {
  return !!navigator.userAgent.match('Chrome/...')
}
var Ln = {}
Object.defineProperty(Ln, '__esModule', { value: !0 })
Ln.clientToElement = as
function as(e, t, n) {
  var r = n.getBoundingClientRect(),
    i = { x: 0, y: 0 }
  return (
    e < r.left ? (i.x = 0) : e >= r.right ? (i.x = r.width - 1) : (i.x = e - r.left),
    t < r.top ? (i.y = 0) : t >= r.bottom ? (i.y = r.height - 1) : (i.y = t - r.top),
    i
  )
}
var Jt = {}
Object.defineProperty(Jt, '__esModule', { value: !0 })
Jt.getPointerEvent = os
Jt.releaseCapture = Rn
Jt.setCapture = us
Jt.stopEvent = ss
function os(e) {
  return e.changedTouches ? e.changedTouches[0] : e.touches ? e.touches[0] : e
}
function ss(e) {
  ;(e.stopPropagation(), e.preventDefault())
}
var sn = !1,
  _a = null
document.captureElement = null
function Wt(e) {
  if (!sn) {
    var t = new e.constructor(e.type, e)
    ;((sn = !0),
      document.captureElement ? document.captureElement.dispatchEvent(t) : _a.dispatchEvent(t),
      (sn = !1),
      e.stopPropagation(),
      t.defaultPrevented && e.preventDefault(),
      e.type === 'mouseup' && Rn())
  }
}
function pa() {
  var e = document.getElementById('noVNC_mouse_capture_elem')
  e.style.cursor = window.getComputedStyle(document.captureElement).cursor
}
var va = new MutationObserver(pa)
function us(e) {
  if (e.setCapture) (e.setCapture(), (document.captureElement = e))
  else {
    Rn()
    var t = document.getElementById('noVNC_mouse_capture_elem')
    ;(t === null &&
      ((t = document.createElement('div')),
      (t.id = 'noVNC_mouse_capture_elem'),
      (t.style.position = 'fixed'),
      (t.style.top = '0px'),
      (t.style.left = '0px'),
      (t.style.width = '100%'),
      (t.style.height = '100%'),
      (t.style.zIndex = 1e4),
      (t.style.display = 'none'),
      document.body.appendChild(t),
      t.addEventListener('contextmenu', Wt),
      t.addEventListener('mousemove', Wt),
      t.addEventListener('mouseup', Wt)),
      (document.captureElement = e),
      va.observe(e, { attributes: !0 }),
      pa(),
      (t.style.display = ''),
      window.addEventListener('mousemove', Wt),
      window.addEventListener('mouseup', Wt))
  }
}
function Rn() {
  if (document.releaseCapture) (document.releaseCapture(), (document.captureElement = null))
  else {
    if (!document.captureElement) return
    ;((_a = document.captureElement), (document.captureElement = null), va.disconnect())
    var e = document.getElementById('noVNC_mouse_capture_elem')
    ;((e.style.display = 'none'),
      window.removeEventListener('mousemove', Wt),
      window.removeEventListener('mouseup', Wt))
  }
}
var Mn = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  function t(s) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (x) {
              return typeof x
            }
          : function (x) {
              return x &&
                typeof Symbol == 'function' &&
                x.constructor === Symbol &&
                x !== Symbol.prototype
                ? 'symbol'
                : typeof x
            }),
      t(s)
    )
  }
  function n(s, x) {
    if (!(s instanceof x)) throw new TypeError('Cannot call a class as a function')
  }
  function r(s, x) {
    for (var u = 0; u < x.length; u++) {
      var f = x[u]
      ;((f.enumerable = f.enumerable || !1),
        (f.configurable = !0),
        'value' in f && (f.writable = !0),
        Object.defineProperty(s, o(f.key), f))
    }
  }
  function i(s, x, u) {
    return (x && r(s.prototype, x), Object.defineProperty(s, 'prototype', { writable: !1 }), s)
  }
  function o(s) {
    var x = a(s, 'string')
    return t(x) == 'symbol' ? x : x + ''
  }
  function a(s, x) {
    if (t(s) != 'object' || !s) return s
    var u = s[Symbol.toPrimitive]
    if (u !== void 0) {
      var f = u.call(s, x)
      if (t(f) != 'object') return f
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(s)
  }
  e.default = (function () {
    function s() {
      ;(n(this, s), (this._listeners = new Map()))
    }
    return i(s, [
      {
        key: 'addEventListener',
        value: function (u, f) {
          ;(this._listeners.has(u) || this._listeners.set(u, new Set()),
            this._listeners.get(u).add(f))
        }
      },
      {
        key: 'removeEventListener',
        value: function (u, f) {
          this._listeners.has(u) && this._listeners.get(u).delete(f)
        }
      },
      {
        key: 'dispatchEvent',
        value: function (u) {
          var f = this
          return this._listeners.has(u.type)
            ? (this._listeners.get(u.type).forEach(function (F) {
                return F.call(f, u)
              }),
              !u.defaultPrevented)
            : !0
        }
      }
    ])
  })()
})(Mn)
var ya = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = a(Re),
    n = i(Zr),
    r = Sr
  function i(g) {
    return g && g.__esModule ? g : { default: g }
  }
  function o(g) {
    if (typeof WeakMap != 'function') return null
    var k = new WeakMap(),
      _ = new WeakMap()
    return (o = function (h) {
      return h ? _ : k
    })(g)
  }
  function a(g, k) {
    if (g && g.__esModule) return g
    if (g === null || (s(g) != 'object' && typeof g != 'function')) return { default: g }
    var _ = o(k)
    if (_ && _.has(g)) return _.get(g)
    var l = { __proto__: null },
      h = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var c in g)
      if (c !== 'default' && {}.hasOwnProperty.call(g, c)) {
        var E = h ? Object.getOwnPropertyDescriptor(g, c) : null
        E && (E.get || E.set) ? Object.defineProperty(l, c, E) : (l[c] = g[c])
      }
    return ((l.default = g), _ && _.set(g, l), l)
  }
  function s(g) {
    '@babel/helpers - typeof'
    return (
      (s =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (k) {
              return typeof k
            }
          : function (k) {
              return k &&
                typeof Symbol == 'function' &&
                k.constructor === Symbol &&
                k !== Symbol.prototype
                ? 'symbol'
                : typeof k
            }),
      s(g)
    )
  }
  function x(g, k) {
    if (!(g instanceof k)) throw new TypeError('Cannot call a class as a function')
  }
  function u(g, k) {
    for (var _ = 0; _ < k.length; _++) {
      var l = k[_]
      ;((l.enumerable = l.enumerable || !1),
        (l.configurable = !0),
        'value' in l && (l.writable = !0),
        Object.defineProperty(g, F(l.key), l))
    }
  }
  function f(g, k, _) {
    return (k && u(g.prototype, k), Object.defineProperty(g, 'prototype', { writable: !1 }), g)
  }
  function F(g) {
    var k = X(g, 'string')
    return s(k) == 'symbol' ? k : k + ''
  }
  function X(g, k) {
    if (s(g) != 'object' || !g) return g
    var _ = g[Symbol.toPrimitive]
    if (_ !== void 0) {
      var l = _.call(g, k)
      if (s(l) != 'object') return l
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(g)
  }
  e.default = (function () {
    function g(k) {
      if (
        (x(this, g),
        (this._drawCtx = null),
        (this._renderQ = []),
        (this._flushPromise = null),
        (this._fbWidth = 0),
        (this._fbHeight = 0),
        (this._prevDrawStyle = ''),
        t.Debug('>> Display.constructor'),
        (this._target = k),
        !this._target)
      )
        throw new Error('Target must be set')
      if (typeof this._target == 'string') throw new Error('target must be a DOM element')
      if (!this._target.getContext) throw new Error('no getContext method')
      ;((this._targetCtx = this._target.getContext('2d')),
        (this._viewportLoc = { x: 0, y: 0, w: this._target.width, h: this._target.height }),
        (this._backbuffer = document.createElement('canvas')),
        (this._drawCtx = this._backbuffer.getContext('2d')),
        (this._damageBounds = {
          left: 0,
          top: 0,
          right: this._backbuffer.width,
          bottom: this._backbuffer.height
        }),
        t.Debug('User Agent: ' + navigator.userAgent),
        t.Debug('<< Display.constructor'),
        (this._scale = 1),
        (this._clipViewport = !1))
    }
    return f(g, [
      {
        key: 'scale',
        get: function () {
          return this._scale
        },
        set: function (_) {
          this._rescale(_)
        }
      },
      {
        key: 'clipViewport',
        get: function () {
          return this._clipViewport
        },
        set: function (_) {
          this._clipViewport = _
          var l = this._viewportLoc
          ;(this.viewportChangeSize(l.w, l.h), this.viewportChangePos(0, 0))
        }
      },
      {
        key: 'width',
        get: function () {
          return this._fbWidth
        }
      },
      {
        key: 'height',
        get: function () {
          return this._fbHeight
        }
      },
      {
        key: 'viewportChangePos',
        value: function (_, l) {
          var h = this._viewportLoc
          ;((_ = Math.floor(_)),
            (l = Math.floor(l)),
            this._clipViewport || ((_ = -h.w), (l = -h.h)))
          var c = h.x + h.w - 1,
            E = h.y + h.h - 1
          ;(_ < 0 && h.x + _ < 0 && (_ = -h.x),
            c + _ >= this._fbWidth && (_ -= c + _ - this._fbWidth + 1),
            h.y + l < 0 && (l = -h.y),
            E + l >= this._fbHeight && (l -= E + l - this._fbHeight + 1),
            !(_ === 0 && l === 0) &&
              (t.Debug('viewportChange deltaX: ' + _ + ', deltaY: ' + l),
              (h.x += _),
              (h.y += l),
              this._damage(h.x, h.y, h.w, h.h),
              this.flip()))
        }
      },
      {
        key: 'viewportChangeSize',
        value: function (_, l) {
          ;((!this._clipViewport || typeof _ > 'u' || typeof l > 'u') &&
            (t.Debug('Setting viewport to full display region'),
            (_ = this._fbWidth),
            (l = this._fbHeight)),
            (_ = Math.floor(_)),
            (l = Math.floor(l)),
            _ > this._fbWidth && (_ = this._fbWidth),
            l > this._fbHeight && (l = this._fbHeight))
          var h = this._viewportLoc
          if (h.w !== _ || h.h !== l) {
            ;((h.w = _), (h.h = l))
            var c = this._target
            ;((c.width = _),
              (c.height = l),
              this.viewportChangePos(0, 0),
              this._damage(h.x, h.y, h.w, h.h),
              this.flip(),
              this._rescale(this._scale))
          }
        }
      },
      {
        key: 'absX',
        value: function (_) {
          return this._scale === 0 ? 0 : (0, r.toSigned32bit)(_ / this._scale + this._viewportLoc.x)
        }
      },
      {
        key: 'absY',
        value: function (_) {
          return this._scale === 0 ? 0 : (0, r.toSigned32bit)(_ / this._scale + this._viewportLoc.y)
        }
      },
      {
        key: 'resize',
        value: function (_, l) {
          ;((this._prevDrawStyle = ''), (this._fbWidth = _), (this._fbHeight = l))
          var h = this._backbuffer
          if (h.width !== _ || h.height !== l) {
            var c = null
            ;(h.width > 0 &&
              h.height > 0 &&
              (c = this._drawCtx.getImageData(0, 0, h.width, h.height)),
              h.width !== _ && (h.width = _),
              h.height !== l && (h.height = l),
              c && this._drawCtx.putImageData(c, 0, 0))
          }
          var E = this._viewportLoc
          ;(this.viewportChangeSize(E.w, E.h), this.viewportChangePos(0, 0))
        }
      },
      {
        key: 'getImageData',
        value: function () {
          return this._drawCtx.getImageData(0, 0, this.width, this.height)
        }
      },
      {
        key: 'toDataURL',
        value: function (_, l) {
          return this._backbuffer.toDataURL(_, l)
        }
      },
      {
        key: 'toBlob',
        value: function (_, l, h) {
          return this._backbuffer.toBlob(_, l, h)
        }
      },
      {
        key: '_damage',
        value: function (_, l, h, c) {
          ;(_ < this._damageBounds.left && (this._damageBounds.left = _),
            l < this._damageBounds.top && (this._damageBounds.top = l),
            _ + h > this._damageBounds.right && (this._damageBounds.right = _ + h),
            l + c > this._damageBounds.bottom && (this._damageBounds.bottom = l + c))
        }
      },
      {
        key: 'flip',
        value: function (_) {
          if (this._renderQ.length !== 0 && !_) this._renderQPush({ type: 'flip' })
          else {
            var l = this._damageBounds.left,
              h = this._damageBounds.top,
              c = this._damageBounds.right - l,
              E = this._damageBounds.bottom - h,
              y = l - this._viewportLoc.x,
              K = h - this._viewportLoc.y
            ;(y < 0 && ((c += y), (l -= y), (y = 0)),
              K < 0 && ((E += K), (h -= K), (K = 0)),
              y + c > this._viewportLoc.w && (c = this._viewportLoc.w - y),
              K + E > this._viewportLoc.h && (E = this._viewportLoc.h - K),
              c > 0 && E > 0 && this._targetCtx.drawImage(this._backbuffer, l, h, c, E, y, K, c, E),
              (this._damageBounds.left = this._damageBounds.top = 65535),
              (this._damageBounds.right = this._damageBounds.bottom = 0))
          }
        }
      },
      {
        key: 'pending',
        value: function () {
          return this._renderQ.length > 0
        }
      },
      {
        key: 'flush',
        value: function () {
          var _ = this
          return this._renderQ.length === 0
            ? Promise.resolve()
            : (this._flushPromise === null &&
                (this._flushPromise = new Promise(function (l) {
                  _._flushResolve = l
                })),
              this._flushPromise)
        }
      },
      {
        key: 'fillRect',
        value: function (_, l, h, c, E, y) {
          this._renderQ.length !== 0 && !y
            ? this._renderQPush({ type: 'fill', x: _, y: l, width: h, height: c, color: E })
            : (this._setFillColor(E), this._drawCtx.fillRect(_, l, h, c), this._damage(_, l, h, c))
        }
      },
      {
        key: 'copyImage',
        value: function (_, l, h, c, E, y, K) {
          this._renderQ.length !== 0 && !K
            ? this._renderQPush({ type: 'copy', oldX: _, oldY: l, x: h, y: c, width: E, height: y })
            : ((this._drawCtx.mozImageSmoothingEnabled = !1),
              (this._drawCtx.webkitImageSmoothingEnabled = !1),
              (this._drawCtx.msImageSmoothingEnabled = !1),
              (this._drawCtx.imageSmoothingEnabled = !1),
              this._drawCtx.drawImage(this._backbuffer, _, l, E, y, h, c, E, y),
              this._damage(h, c, E, y))
        }
      },
      {
        key: 'imageRect',
        value: function (_, l, h, c, E, y) {
          if (!(h === 0 || c === 0)) {
            var K = new Image()
            ;((K.src = 'data: ' + E + ';base64,' + n.default.encode(y)),
              this._renderQPush({ type: 'img', img: K, x: _, y: l, width: h, height: c }))
          }
        }
      },
      {
        key: 'videoFrame',
        value: function (_, l, h, c, E) {
          this._renderQPush({ type: 'frame', frame: E, x: _, y: l, width: h, height: c })
        }
      },
      {
        key: 'blitImage',
        value: function (_, l, h, c, E, y, K) {
          if (this._renderQ.length !== 0 && !K) {
            var w = new Uint8Array(h * c * 4)
            ;(w.set(new Uint8Array(E.buffer, 0, w.length)),
              this._renderQPush({ type: 'blit', data: w, x: _, y: l, width: h, height: c }))
          } else {
            var m = new Uint8ClampedArray(E.buffer, E.byteOffset + y, h * c * 4),
              C = new ImageData(m, h, c)
            ;(this._drawCtx.putImageData(C, _, l), this._damage(_, l, h, c))
          }
        }
      },
      {
        key: 'drawImage',
        value: function (_) {
          for (var l, h = arguments.length, c = new Array(h > 1 ? h - 1 : 0), E = 1; E < h; E++)
            c[E - 1] = arguments[E]
          if (((l = this._drawCtx).drawImage.apply(l, [_].concat(c)), c.length <= 4)) {
            var y = c[0],
              K = c[1]
            this._damage(y, K, _.width, _.height)
          } else {
            var w = c[2],
              m = c[3],
              C = c[4],
              A = c[5]
            this._damage(C, A, w, m)
          }
        }
      },
      {
        key: 'autoscale',
        value: function (_, l) {
          var h
          if (_ === 0 || l === 0) h = 0
          else {
            var c = this._viewportLoc,
              E = _ / l,
              y = c.w / c.h
            y >= E ? (h = _ / c.w) : (h = l / c.h)
          }
          this._rescale(h)
        }
      },
      {
        key: '_rescale',
        value: function (_) {
          this._scale = _
          var l = this._viewportLoc,
            h = _ * l.w + 'px',
            c = _ * l.h + 'px'
          ;(this._target.style.width !== h || this._target.style.height !== c) &&
            ((this._target.style.width = h), (this._target.style.height = c))
        }
      },
      {
        key: '_setFillColor',
        value: function (_) {
          var l = 'rgb(' + _[0] + ',' + _[1] + ',' + _[2] + ')'
          l !== this._prevDrawStyle && ((this._drawCtx.fillStyle = l), (this._prevDrawStyle = l))
        }
      },
      {
        key: '_renderQPush',
        value: function (_) {
          ;(this._renderQ.push(_), this._renderQ.length === 1 && this._scanRenderQ())
        }
      },
      {
        key: '_resumeRenderQ',
        value: function () {
          ;(this.removeEventListener('load', this._noVNCDisplay._resumeRenderQ),
            this._noVNCDisplay._scanRenderQ())
        }
      },
      {
        key: '_scanRenderQ',
        value: function () {
          for (
            var _ = this,
              l = !0,
              h = function () {
                var y = _._renderQ[0]
                switch (y.type) {
                  case 'flip':
                    _.flip(!0)
                    break
                  case 'copy':
                    _.copyImage(y.oldX, y.oldY, y.x, y.y, y.width, y.height, !0)
                    break
                  case 'fill':
                    _.fillRect(y.x, y.y, y.width, y.height, y.color, !0)
                    break
                  case 'blit':
                    _.blitImage(y.x, y.y, y.width, y.height, y.data, 0, !0)
                    break
                  case 'img':
                    if (y.img.complete) {
                      if (y.img.width !== y.width || y.img.height !== y.height)
                        return (
                          t.Error(
                            'Decoded image has incorrect dimensions. Got ' +
                              y.img.width +
                              'x' +
                              y.img.height +
                              '. Expected ' +
                              y.width +
                              'x' +
                              y.height +
                              '.'
                          ),
                          { v: void 0 }
                        )
                      _.drawImage(y.img, y.x, y.y)
                    } else
                      ((y.img._noVNCDisplay = _),
                        y.img.addEventListener('load', _._resumeRenderQ),
                        (l = !1))
                    break
                  case 'frame':
                    if (y.frame.ready) {
                      var K = y.frame.frame
                      ;(K.codedWidth < y.width || K.codedHeight < y.height) &&
                        t.Warn(
                          'Decoded video frame does not cover its full rectangle area. Expecting at least ' +
                            y.width +
                            'x' +
                            y.height +
                            ' but got ' +
                            K.codedWidth +
                            'x' +
                            K.codedHeight
                        )
                      var w = 0,
                        m = 0,
                        C = y.width,
                        A = y.height,
                        O = y.x,
                        I = y.y,
                        Z = C,
                        q = A
                      ;(_.drawImage(K, w, m, C, A, O, I, Z, q), K.close())
                    } else {
                      var $ = _
                      ;(y.frame.promise.then(function () {
                        $._scanRenderQ()
                      }),
                        (l = !1))
                    }
                    break
                }
                l && _._renderQ.shift()
              },
              c;
            l && this._renderQ.length > 0;

          )
            if (((c = h()), c)) return c.v
          this._renderQ.length === 0 &&
            this._flushPromise !== null &&
            (this._flushResolve(), (this._flushPromise = null), (this._flushResolve = null))
        }
      }
    ])
  })()
})(ya)
var Kr = {},
  Ke = {},
  ot = {}
Object.defineProperty(ot, '__esModule', { value: !0 })
ot.Buf8 = ot.Buf32 = ot.Buf16 = void 0
ot.arraySet = fs
ot.flattenChunks = cs
ot.shrinkBuf = ls
function ls(e, t) {
  return e.length === t ? e : e.subarray ? e.subarray(0, t) : ((e.length = t), e)
}
function fs(e, t, n, r, i) {
  if (t.subarray && e.subarray) {
    e.set(t.subarray(n, n + r), i)
    return
  }
  for (var o = 0; o < r; o++) e[i + o] = t[n + o]
}
function cs(e) {
  var t, n, r, i, o, a
  for (r = 0, t = 0, n = e.length; t < n; t++) r += e[t].length
  for (a = new Uint8Array(r), i = 0, t = 0, n = e.length; t < n; t++)
    ((o = e[t]), a.set(o, i), (i += o.length))
  return a
}
ot.Buf8 = Uint8Array
ot.Buf16 = Uint16Array
ot.Buf32 = Int32Array
var Dn = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = t))
  function t(n, r, i, o) {
    for (var a = (n & 65535) | 0, s = ((n >>> 16) & 65535) | 0, x = 0; i !== 0; ) {
      ;((x = i > 2e3 ? 2e3 : i), (i -= x))
      do ((a = (a + r[o++]) | 0), (s = (s + a) | 0))
      while (--x)
      ;((a %= 65521), (s %= 65521))
    }
    return a | (s << 16) | 0
  }
})(Dn)
var On = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = t))
  function t() {
    for (var n, r = [], i = 0; i < 256; i++) {
      n = i
      for (var o = 0; o < 8; o++) n = n & 1 ? 3988292384 ^ (n >>> 1) : n >>> 1
      r[i] = n
    }
    return r
  }
  t()
})(On)
var xa = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = r))
  var t = 30,
    n = 12
  function r(i, o) {
    var a, s, x, u, f, F, X, g, k, _, l, h, c, E, y, K, w, m, C, A, O, I, Z, q, $
    ;((a = i.state),
      (s = i.next_in),
      (q = i.input),
      (x = s + (i.avail_in - 5)),
      (u = i.next_out),
      ($ = i.output),
      (f = u - (o - i.avail_out)),
      (F = u + (i.avail_out - 257)),
      (X = a.dmax),
      (g = a.wsize),
      (k = a.whave),
      (_ = a.wnext),
      (l = a.window),
      (h = a.hold),
      (c = a.bits),
      (E = a.lencode),
      (y = a.distcode),
      (K = (1 << a.lenbits) - 1),
      (w = (1 << a.distbits) - 1))
    e: do {
      ;(c < 15 && ((h += q[s++] << c), (c += 8), (h += q[s++] << c), (c += 8)), (m = E[h & K]))
      t: for (;;) {
        if (((C = m >>> 24), (h >>>= C), (c -= C), (C = (m >>> 16) & 255), C === 0))
          $[u++] = m & 65535
        else if (C & 16) {
          ;((A = m & 65535),
            (C &= 15),
            C &&
              (c < C && ((h += q[s++] << c), (c += 8)),
              (A += h & ((1 << C) - 1)),
              (h >>>= C),
              (c -= C)),
            c < 15 && ((h += q[s++] << c), (c += 8), (h += q[s++] << c), (c += 8)),
            (m = y[h & w]))
          r: for (;;) {
            if (((C = m >>> 24), (h >>>= C), (c -= C), (C = (m >>> 16) & 255), C & 16)) {
              if (
                ((O = m & 65535),
                (C &= 15),
                c < C && ((h += q[s++] << c), (c += 8), c < C && ((h += q[s++] << c), (c += 8))),
                (O += h & ((1 << C) - 1)),
                O > X)
              ) {
                ;((i.msg = 'invalid distance too far back'), (a.mode = t))
                break e
              }
              if (((h >>>= C), (c -= C), (C = u - f), O > C)) {
                if (((C = O - C), C > k && a.sane)) {
                  ;((i.msg = 'invalid distance too far back'), (a.mode = t))
                  break e
                }
                if (((I = 0), (Z = l), _ === 0)) {
                  if (((I += g - C), C < A)) {
                    A -= C
                    do $[u++] = l[I++]
                    while (--C)
                    ;((I = u - O), (Z = $))
                  }
                } else if (_ < C) {
                  if (((I += g + _ - C), (C -= _), C < A)) {
                    A -= C
                    do $[u++] = l[I++]
                    while (--C)
                    if (((I = 0), _ < A)) {
                      ;((C = _), (A -= C))
                      do $[u++] = l[I++]
                      while (--C)
                      ;((I = u - O), (Z = $))
                    }
                  }
                } else if (((I += _ - C), C < A)) {
                  A -= C
                  do $[u++] = l[I++]
                  while (--C)
                  ;((I = u - O), (Z = $))
                }
                for (; A > 2; ) (($[u++] = Z[I++]), ($[u++] = Z[I++]), ($[u++] = Z[I++]), (A -= 3))
                A && (($[u++] = Z[I++]), A > 1 && ($[u++] = Z[I++]))
              } else {
                I = u - O
                do (($[u++] = $[I++]), ($[u++] = $[I++]), ($[u++] = $[I++]), (A -= 3))
                while (A > 2)
                A && (($[u++] = $[I++]), A > 1 && ($[u++] = $[I++]))
              }
            } else if (C & 64) {
              ;((i.msg = 'invalid distance code'), (a.mode = t))
              break e
            } else {
              m = y[(m & 65535) + (h & ((1 << C) - 1))]
              continue r
            }
            break
          }
        } else if (C & 64)
          if (C & 32) {
            a.mode = n
            break e
          } else {
            ;((i.msg = 'invalid literal/length code'), (a.mode = t))
            break e
          }
        else {
          m = E[(m & 65535) + (h & ((1 << C) - 1))]
          continue t
        }
        break
      }
    } while (s < x && u < F)
    ;((A = c >> 3),
      (s -= A),
      (c -= A << 3),
      (h &= (1 << c) - 1),
      (i.next_in = s),
      (i.next_out = u),
      (i.avail_in = s < x ? 5 + (x - s) : 5 - (s - x)),
      (i.avail_out = u < F ? 257 + (F - u) : 257 - (u - F)),
      (a.hold = h),
      (a.bits = c))
  }
})(xa)
var ga = {}
;(function (e) {
  function t(l) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (h) {
              return typeof h
            }
          : function (h) {
              return h &&
                typeof Symbol == 'function' &&
                h.constructor === Symbol &&
                h !== Symbol.prototype
                ? 'symbol'
                : typeof h
            }),
      t(l)
    )
  }
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = _))
  var n = i(ot)
  function r(l) {
    if (typeof WeakMap != 'function') return null
    var h = new WeakMap(),
      c = new WeakMap()
    return (r = function (y) {
      return y ? c : h
    })(l)
  }
  function i(l, h) {
    if (l && l.__esModule) return l
    if (l === null || (t(l) != 'object' && typeof l != 'function')) return { default: l }
    var c = r(h)
    if (c && c.has(l)) return c.get(l)
    var E = { __proto__: null },
      y = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var K in l)
      if (K !== 'default' && {}.hasOwnProperty.call(l, K)) {
        var w = y ? Object.getOwnPropertyDescriptor(l, K) : null
        w && (w.get || w.set) ? Object.defineProperty(E, K, w) : (E[K] = l[K])
      }
    return ((E.default = l), c && c.set(l, E), E)
  }
  var o = 15,
    a = 852,
    s = 592,
    x = 0,
    u = 1,
    f = 2,
    F = [
      3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 15, 17, 19, 23, 27, 31, 35, 43, 51, 59, 67, 83, 99, 115, 131,
      163, 195, 227, 258, 0, 0
    ],
    X = [
      16, 16, 16, 16, 16, 16, 16, 16, 17, 17, 17, 17, 18, 18, 18, 18, 19, 19, 19, 19, 20, 20, 20,
      20, 21, 21, 21, 21, 16, 72, 78
    ],
    g = [
      1, 2, 3, 4, 5, 7, 9, 13, 17, 25, 33, 49, 65, 97, 129, 193, 257, 385, 513, 769, 1025, 1537,
      2049, 3073, 4097, 6145, 8193, 12289, 16385, 24577, 0, 0
    ],
    k = [
      16, 16, 16, 16, 17, 17, 18, 18, 19, 19, 20, 20, 21, 21, 22, 22, 23, 23, 24, 24, 25, 25, 26,
      26, 27, 27, 28, 28, 29, 29, 64, 64
    ]
  function _(l, h, c, E, y, K, w, m) {
    var C = m.bits,
      A = 0,
      O = 0,
      I = 0,
      Z = 0,
      q = 0,
      $ = 0,
      ne = 0,
      S = 0,
      p = 0,
      b = 0,
      L,
      B,
      M,
      V,
      re,
      oe = null,
      ce = 0,
      pe,
      ye = new n.Buf16(o + 1),
      Ne = new n.Buf16(o + 1),
      je = null,
      st = 0,
      Ue,
      He,
      rt
    for (A = 0; A <= o; A++) ye[A] = 0
    for (O = 0; O < E; O++) ye[h[c + O]]++
    for (q = C, Z = o; Z >= 1 && ye[Z] === 0; Z--);
    if ((q > Z && (q = Z), Z === 0))
      return (
        (y[K++] = (1 << 24) | (64 << 16) | 0),
        (y[K++] = (1 << 24) | (64 << 16) | 0),
        (m.bits = 1),
        0
      )
    for (I = 1; I < Z && ye[I] === 0; I++);
    for (q < I && (q = I), S = 1, A = 1; A <= o; A++)
      if (((S <<= 1), (S -= ye[A]), S < 0)) return -1
    if (S > 0 && (l === x || Z !== 1)) return -1
    for (Ne[1] = 0, A = 1; A < o; A++) Ne[A + 1] = Ne[A] + ye[A]
    for (O = 0; O < E; O++) h[c + O] !== 0 && (w[Ne[h[c + O]]++] = O)
    if (
      (l === x
        ? ((oe = je = w), (pe = 19))
        : l === u
          ? ((oe = F), (ce -= 257), (je = X), (st -= 257), (pe = 256))
          : ((oe = g), (je = k), (pe = -1)),
      (b = 0),
      (O = 0),
      (A = I),
      (re = K),
      ($ = q),
      (ne = 0),
      (M = -1),
      (p = 1 << q),
      (V = p - 1),
      (l === u && p > a) || (l === f && p > s))
    )
      return 1
    for (;;) {
      ;((Ue = A - ne),
        w[O] < pe
          ? ((He = 0), (rt = w[O]))
          : w[O] > pe
            ? ((He = je[st + w[O]]), (rt = oe[ce + w[O]]))
            : ((He = 96), (rt = 0)),
        (L = 1 << (A - ne)),
        (B = 1 << $),
        (I = B))
      do ((B -= L), (y[re + (b >> ne) + B] = (Ue << 24) | (He << 16) | rt | 0))
      while (B !== 0)
      for (L = 1 << (A - 1); b & L; ) L >>= 1
      if ((L !== 0 ? ((b &= L - 1), (b += L)) : (b = 0), O++, --ye[A] === 0)) {
        if (A === Z) break
        A = h[c + w[O]]
      }
      if (A > q && (b & V) !== M) {
        for (
          ne === 0 && (ne = q), re += I, $ = A - ne, S = 1 << $;
          $ + ne < Z && ((S -= ye[$ + ne]), !(S <= 0));

        )
          ($++, (S <<= 1))
        if (((p += 1 << $), (l === u && p > a) || (l === f && p > s))) return 1
        ;((M = b & V), (y[M] = (q << 24) | ($ << 16) | (re - K) | 0))
      }
    }
    return (b !== 0 && (y[re + b] = ((A - ne) << 24) | (64 << 16) | 0), (m.bits = q), 0)
  }
})(ga)
function wn(e) {
  '@babel/helpers - typeof'
  return (
    (wn =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    wn(e)
  )
}
Object.defineProperty(Ke, '__esModule', { value: !0 })
Ke.Z_TREES =
  Ke.Z_STREAM_ERROR =
  Ke.Z_STREAM_END =
  Ke.Z_OK =
  Ke.Z_NEED_DICT =
  Ke.Z_MEM_ERROR =
  Ke.Z_FINISH =
  Ke.Z_DEFLATED =
  Ke.Z_DATA_ERROR =
  Ke.Z_BUF_ERROR =
  Ke.Z_BLOCK =
    void 0
Ke.inflate = Xs
Ke.inflateEnd = Fs
Ke.inflateGetHeader = Cs
Ke.inflateInfo = void 0
Ke.inflateInit = Es
Ke.inflateInit2 = Aa
Ke.inflateReset = Fa
Ke.inflateReset2 = Ca
Ke.inflateResetKeep = Xa
Ke.inflateSetDictionary = As
var lt = ds(ot),
  kn = qr(Dn),
  kt = qr(On),
  hs = qr(xa),
  _r = qr(ga)
function qr(e) {
  return e && e.__esModule ? e : { default: e }
}
function ba(e) {
  if (typeof WeakMap != 'function') return null
  var t = new WeakMap(),
    n = new WeakMap()
  return (ba = function (i) {
    return i ? n : t
  })(e)
}
function ds(e, t) {
  if (e && e.__esModule) return e
  if (e === null || (wn(e) != 'object' && typeof e != 'function')) return { default: e }
  var n = ba(t)
  if (n && n.has(e)) return n.get(e)
  var r = { __proto__: null },
    i = Object.defineProperty && Object.getOwnPropertyDescriptor
  for (var o in e)
    if (o !== 'default' && {}.hasOwnProperty.call(e, o)) {
      var a = i ? Object.getOwnPropertyDescriptor(e, o) : null
      a && (a.get || a.set) ? Object.defineProperty(r, o, a) : (r[o] = e[o])
    }
  return ((r.default = e), n && n.set(e, r), r)
}
var _s = 0,
  ma = 1,
  wa = 2,
  li = (Ke.Z_FINISH = 4),
  ps = (Ke.Z_BLOCK = 5),
  Or = (Ke.Z_TREES = 6),
  Vt = (Ke.Z_OK = 0),
  vs = (Ke.Z_STREAM_END = 1),
  ys = (Ke.Z_NEED_DICT = 2),
  _t = (Ke.Z_STREAM_ERROR = -2),
  ka = (Ke.Z_DATA_ERROR = -3),
  Sa = (Ke.Z_MEM_ERROR = -4),
  xs = (Ke.Z_BUF_ERROR = -5),
  fi = (Ke.Z_DEFLATED = 8),
  Ea = 1,
  ci = 2,
  hi = 3,
  di = 4,
  _i = 5,
  pi = 6,
  vi = 7,
  yi = 8,
  xi = 9,
  gi = 10,
  zr = 11,
  Ft = 12,
  un = 13,
  bi = 14,
  ln = 15,
  mi = 16,
  wi = 17,
  ki = 18,
  Si = 19,
  Br = 20,
  Ir = 21,
  Ei = 22,
  Ki = 23,
  Xi = 24,
  Fi = 25,
  Ci = 26,
  fn = 27,
  Ai = 28,
  Pi = 29,
  Be = 30,
  Ka = 31,
  gs = 32,
  bs = 852,
  ms = 592,
  ws = 15,
  ks = ws
function Ti(e) {
  return ((e >>> 24) & 255) + ((e >>> 8) & 65280) + ((e & 65280) << 8) + ((e & 255) << 24)
}
function Ss() {
  ;((this.mode = 0),
    (this.last = !1),
    (this.wrap = 0),
    (this.havedict = !1),
    (this.flags = 0),
    (this.dmax = 0),
    (this.check = 0),
    (this.total = 0),
    (this.head = null),
    (this.wbits = 0),
    (this.wsize = 0),
    (this.whave = 0),
    (this.wnext = 0),
    (this.window = null),
    (this.hold = 0),
    (this.bits = 0),
    (this.length = 0),
    (this.offset = 0),
    (this.extra = 0),
    (this.lencode = null),
    (this.distcode = null),
    (this.lenbits = 0),
    (this.distbits = 0),
    (this.ncode = 0),
    (this.nlen = 0),
    (this.ndist = 0),
    (this.have = 0),
    (this.next = null),
    (this.lens = new lt.Buf16(320)),
    (this.work = new lt.Buf16(288)),
    (this.lendyn = null),
    (this.distdyn = null),
    (this.sane = 0),
    (this.back = 0),
    (this.was = 0))
}
function Xa(e) {
  var t
  return !e || !e.state
    ? _t
    : ((t = e.state),
      (e.total_in = e.total_out = t.total = 0),
      (e.msg = ''),
      t.wrap && (e.adler = t.wrap & 1),
      (t.mode = Ea),
      (t.last = 0),
      (t.havedict = 0),
      (t.dmax = 32768),
      (t.head = null),
      (t.hold = 0),
      (t.bits = 0),
      (t.lencode = t.lendyn = new lt.Buf32(bs)),
      (t.distcode = t.distdyn = new lt.Buf32(ms)),
      (t.sane = 1),
      (t.back = -1),
      Vt)
}
function Fa(e) {
  var t
  return !e || !e.state ? _t : ((t = e.state), (t.wsize = 0), (t.whave = 0), (t.wnext = 0), Xa(e))
}
function Ca(e, t) {
  var n, r
  return !e ||
    !e.state ||
    ((r = e.state),
    t < 0 ? ((n = 0), (t = -t)) : ((n = (t >> 4) + 1), t < 48 && (t &= 15)),
    t && (t < 8 || t > 15))
    ? _t
    : (r.window !== null && r.wbits !== t && (r.window = null), (r.wrap = n), (r.wbits = t), Fa(e))
}
function Aa(e, t) {
  var n, r
  return e
    ? ((r = new Ss()),
      (e.state = r),
      (r.window = null),
      (n = Ca(e, t)),
      n !== Vt && (e.state = null),
      n)
    : _t
}
function Es(e) {
  return Aa(e, ks)
}
var Li = !0,
  cn,
  hn
function Ks(e) {
  if (Li) {
    var t
    for (cn = new lt.Buf32(512), hn = new lt.Buf32(32), t = 0; t < 144; ) e.lens[t++] = 8
    for (; t < 256; ) e.lens[t++] = 9
    for (; t < 280; ) e.lens[t++] = 7
    for (; t < 288; ) e.lens[t++] = 8
    for ((0, _r.default)(ma, e.lens, 0, 288, cn, 0, e.work, { bits: 9 }), t = 0; t < 32; )
      e.lens[t++] = 5
    ;((0, _r.default)(wa, e.lens, 0, 32, hn, 0, e.work, { bits: 5 }), (Li = !1))
  }
  ;((e.lencode = cn), (e.lenbits = 9), (e.distcode = hn), (e.distbits = 5))
}
function Pa(e, t, n, r) {
  var i,
    o = e.state
  return (
    o.window === null &&
      ((o.wsize = 1 << o.wbits), (o.wnext = 0), (o.whave = 0), (o.window = new lt.Buf8(o.wsize))),
    r >= o.wsize
      ? (lt.arraySet(o.window, t, n - o.wsize, o.wsize, 0), (o.wnext = 0), (o.whave = o.wsize))
      : ((i = o.wsize - o.wnext),
        i > r && (i = r),
        lt.arraySet(o.window, t, n - r, i, o.wnext),
        (r -= i),
        r
          ? (lt.arraySet(o.window, t, n - r, r, 0), (o.wnext = r), (o.whave = o.wsize))
          : ((o.wnext += i),
            o.wnext === o.wsize && (o.wnext = 0),
            o.whave < o.wsize && (o.whave += i))),
    0
  )
}
function Xs(e, t) {
  var n,
    r,
    i,
    o,
    a,
    s,
    x,
    u,
    f,
    F,
    X,
    g,
    k,
    _,
    l = 0,
    h,
    c,
    E,
    y,
    K,
    w,
    m,
    C,
    A = new lt.Buf8(4),
    O,
    I,
    Z = [16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15]
  if (!e || !e.state || !e.output || (!e.input && e.avail_in !== 0)) return _t
  ;((n = e.state),
    n.mode === Ft && (n.mode = un),
    (a = e.next_out),
    (i = e.output),
    (x = e.avail_out),
    (o = e.next_in),
    (r = e.input),
    (s = e.avail_in),
    (u = n.hold),
    (f = n.bits),
    (F = s),
    (X = x),
    (C = Vt))
  e: for (;;)
    switch (n.mode) {
      case Ea:
        if (n.wrap === 0) {
          n.mode = un
          break
        }
        for (; f < 16; ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        if (n.wrap & 2 && u === 35615) {
          ;((n.check = 0),
            (A[0] = u & 255),
            (A[1] = (u >>> 8) & 255),
            (n.check = (0, kt.default)(n.check, A, 2, 0)),
            (u = 0),
            (f = 0),
            (n.mode = ci))
          break
        }
        if (
          ((n.flags = 0),
          n.head && (n.head.done = !1),
          !(n.wrap & 1) || (((u & 255) << 8) + (u >> 8)) % 31)
        ) {
          ;((e.msg = 'incorrect header check'), (n.mode = Be))
          break
        }
        if ((u & 15) !== fi) {
          ;((e.msg = 'unknown compression method'), (n.mode = Be))
          break
        }
        if (((u >>>= 4), (f -= 4), (m = (u & 15) + 8), n.wbits === 0)) n.wbits = m
        else if (m > n.wbits) {
          ;((e.msg = 'invalid window size'), (n.mode = Be))
          break
        }
        ;((n.dmax = 1 << m),
          (e.adler = n.check = 1),
          (n.mode = u & 512 ? gi : Ft),
          (u = 0),
          (f = 0))
        break
      case ci:
        for (; f < 16; ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        if (((n.flags = u), (n.flags & 255) !== fi)) {
          ;((e.msg = 'unknown compression method'), (n.mode = Be))
          break
        }
        if (n.flags & 57344) {
          ;((e.msg = 'unknown header flags set'), (n.mode = Be))
          break
        }
        ;(n.head && (n.head.text = (u >> 8) & 1),
          n.flags & 512 &&
            ((A[0] = u & 255),
            (A[1] = (u >>> 8) & 255),
            (n.check = (0, kt.default)(n.check, A, 2, 0))),
          (u = 0),
          (f = 0),
          (n.mode = hi))
      case hi:
        for (; f < 32; ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        ;(n.head && (n.head.time = u),
          n.flags & 512 &&
            ((A[0] = u & 255),
            (A[1] = (u >>> 8) & 255),
            (A[2] = (u >>> 16) & 255),
            (A[3] = (u >>> 24) & 255),
            (n.check = (0, kt.default)(n.check, A, 4, 0))),
          (u = 0),
          (f = 0),
          (n.mode = di))
      case di:
        for (; f < 16; ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        ;(n.head && ((n.head.xflags = u & 255), (n.head.os = u >> 8)),
          n.flags & 512 &&
            ((A[0] = u & 255),
            (A[1] = (u >>> 8) & 255),
            (n.check = (0, kt.default)(n.check, A, 2, 0))),
          (u = 0),
          (f = 0),
          (n.mode = _i))
      case _i:
        if (n.flags & 1024) {
          for (; f < 16; ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          ;((n.length = u),
            n.head && (n.head.extra_len = u),
            n.flags & 512 &&
              ((A[0] = u & 255),
              (A[1] = (u >>> 8) & 255),
              (n.check = (0, kt.default)(n.check, A, 2, 0))),
            (u = 0),
            (f = 0))
        } else n.head && (n.head.extra = null)
        n.mode = pi
      case pi:
        if (
          n.flags & 1024 &&
          ((g = n.length),
          g > s && (g = s),
          g &&
            (n.head &&
              ((m = n.head.extra_len - n.length),
              n.head.extra || (n.head.extra = new Array(n.head.extra_len)),
              lt.arraySet(n.head.extra, r, o, g, m)),
            n.flags & 512 && (n.check = (0, kt.default)(n.check, r, g, o)),
            (s -= g),
            (o += g),
            (n.length -= g)),
          n.length)
        )
          break e
        ;((n.length = 0), (n.mode = vi))
      case vi:
        if (n.flags & 2048) {
          if (s === 0) break e
          g = 0
          do
            ((m = r[o + g++]),
              n.head && m && n.length < 65536 && (n.head.name += String.fromCharCode(m)))
          while (m && g < s)
          if (
            (n.flags & 512 && (n.check = (0, kt.default)(n.check, r, g, o)), (s -= g), (o += g), m)
          )
            break e
        } else n.head && (n.head.name = null)
        ;((n.length = 0), (n.mode = yi))
      case yi:
        if (n.flags & 4096) {
          if (s === 0) break e
          g = 0
          do
            ((m = r[o + g++]),
              n.head && m && n.length < 65536 && (n.head.comment += String.fromCharCode(m)))
          while (m && g < s)
          if (
            (n.flags & 512 && (n.check = (0, kt.default)(n.check, r, g, o)), (s -= g), (o += g), m)
          )
            break e
        } else n.head && (n.head.comment = null)
        n.mode = xi
      case xi:
        if (n.flags & 512) {
          for (; f < 16; ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          if (u !== (n.check & 65535)) {
            ;((e.msg = 'header crc mismatch'), (n.mode = Be))
            break
          }
          ;((u = 0), (f = 0))
        }
        ;(n.head && ((n.head.hcrc = (n.flags >> 9) & 1), (n.head.done = !0)),
          (e.adler = n.check = 0),
          (n.mode = Ft))
        break
      case gi:
        for (; f < 32; ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        ;((e.adler = n.check = Ti(u)), (u = 0), (f = 0), (n.mode = zr))
      case zr:
        if (n.havedict === 0)
          return (
            (e.next_out = a),
            (e.avail_out = x),
            (e.next_in = o),
            (e.avail_in = s),
            (n.hold = u),
            (n.bits = f),
            ys
          )
        ;((e.adler = n.check = 1), (n.mode = Ft))
      case Ft:
        if (t === ps || t === Or) break e
      case un:
        if (n.last) {
          ;((u >>>= f & 7), (f -= f & 7), (n.mode = fn))
          break
        }
        for (; f < 3; ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        switch (((n.last = u & 1), (u >>>= 1), (f -= 1), u & 3)) {
          case 0:
            n.mode = bi
            break
          case 1:
            if ((Ks(n), (n.mode = Br), t === Or)) {
              ;((u >>>= 2), (f -= 2))
              break e
            }
            break
          case 2:
            n.mode = wi
            break
          case 3:
            ;((e.msg = 'invalid block type'), (n.mode = Be))
        }
        ;((u >>>= 2), (f -= 2))
        break
      case bi:
        for (u >>>= f & 7, f -= f & 7; f < 32; ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        if ((u & 65535) !== ((u >>> 16) ^ 65535)) {
          ;((e.msg = 'invalid stored block lengths'), (n.mode = Be))
          break
        }
        if (((n.length = u & 65535), (u = 0), (f = 0), (n.mode = ln), t === Or)) break e
      case ln:
        n.mode = mi
      case mi:
        if (((g = n.length), g)) {
          if ((g > s && (g = s), g > x && (g = x), g === 0)) break e
          ;(lt.arraySet(i, r, o, g, a), (s -= g), (o += g), (x -= g), (a += g), (n.length -= g))
          break
        }
        n.mode = Ft
        break
      case wi:
        for (; f < 14; ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        if (
          ((n.nlen = (u & 31) + 257),
          (u >>>= 5),
          (f -= 5),
          (n.ndist = (u & 31) + 1),
          (u >>>= 5),
          (f -= 5),
          (n.ncode = (u & 15) + 4),
          (u >>>= 4),
          (f -= 4),
          n.nlen > 286 || n.ndist > 30)
        ) {
          ;((e.msg = 'too many length or distance symbols'), (n.mode = Be))
          break
        }
        ;((n.have = 0), (n.mode = ki))
      case ki:
        for (; n.have < n.ncode; ) {
          for (; f < 3; ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          ;((n.lens[Z[n.have++]] = u & 7), (u >>>= 3), (f -= 3))
        }
        for (; n.have < 19; ) n.lens[Z[n.have++]] = 0
        if (
          ((n.lencode = n.lendyn),
          (n.lenbits = 7),
          (O = { bits: n.lenbits }),
          (C = (0, _r.default)(_s, n.lens, 0, 19, n.lencode, 0, n.work, O)),
          (n.lenbits = O.bits),
          C)
        ) {
          ;((e.msg = 'invalid code lengths set'), (n.mode = Be))
          break
        }
        ;((n.have = 0), (n.mode = Si))
      case Si:
        for (; n.have < n.nlen + n.ndist; ) {
          for (
            ;
            (l = n.lencode[u & ((1 << n.lenbits) - 1)]),
              (h = l >>> 24),
              (c = (l >>> 16) & 255),
              (E = l & 65535),
              !(h <= f);

          ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          if (E < 16) ((u >>>= h), (f -= h), (n.lens[n.have++] = E))
          else {
            if (E === 16) {
              for (I = h + 2; f < I; ) {
                if (s === 0) break e
                ;(s--, (u += r[o++] << f), (f += 8))
              }
              if (((u >>>= h), (f -= h), n.have === 0)) {
                ;((e.msg = 'invalid bit length repeat'), (n.mode = Be))
                break
              }
              ;((m = n.lens[n.have - 1]), (g = 3 + (u & 3)), (u >>>= 2), (f -= 2))
            } else if (E === 17) {
              for (I = h + 3; f < I; ) {
                if (s === 0) break e
                ;(s--, (u += r[o++] << f), (f += 8))
              }
              ;((u >>>= h), (f -= h), (m = 0), (g = 3 + (u & 7)), (u >>>= 3), (f -= 3))
            } else {
              for (I = h + 7; f < I; ) {
                if (s === 0) break e
                ;(s--, (u += r[o++] << f), (f += 8))
              }
              ;((u >>>= h), (f -= h), (m = 0), (g = 11 + (u & 127)), (u >>>= 7), (f -= 7))
            }
            if (n.have + g > n.nlen + n.ndist) {
              ;((e.msg = 'invalid bit length repeat'), (n.mode = Be))
              break
            }
            for (; g--; ) n.lens[n.have++] = m
          }
        }
        if (n.mode === Be) break
        if (n.lens[256] === 0) {
          ;((e.msg = 'invalid code -- missing end-of-block'), (n.mode = Be))
          break
        }
        if (
          ((n.lenbits = 9),
          (O = { bits: n.lenbits }),
          (C = (0, _r.default)(ma, n.lens, 0, n.nlen, n.lencode, 0, n.work, O)),
          (n.lenbits = O.bits),
          C)
        ) {
          ;((e.msg = 'invalid literal/lengths set'), (n.mode = Be))
          break
        }
        if (
          ((n.distbits = 6),
          (n.distcode = n.distdyn),
          (O = { bits: n.distbits }),
          (C = (0, _r.default)(wa, n.lens, n.nlen, n.ndist, n.distcode, 0, n.work, O)),
          (n.distbits = O.bits),
          C)
        ) {
          ;((e.msg = 'invalid distances set'), (n.mode = Be))
          break
        }
        if (((n.mode = Br), t === Or)) break e
      case Br:
        n.mode = Ir
      case Ir:
        if (s >= 6 && x >= 258) {
          ;((e.next_out = a),
            (e.avail_out = x),
            (e.next_in = o),
            (e.avail_in = s),
            (n.hold = u),
            (n.bits = f),
            (0, hs.default)(e, X),
            (a = e.next_out),
            (i = e.output),
            (x = e.avail_out),
            (o = e.next_in),
            (r = e.input),
            (s = e.avail_in),
            (u = n.hold),
            (f = n.bits),
            n.mode === Ft && (n.back = -1))
          break
        }
        for (
          n.back = 0;
          (l = n.lencode[u & ((1 << n.lenbits) - 1)]),
            (h = l >>> 24),
            (c = (l >>> 16) & 255),
            (E = l & 65535),
            !(h <= f);

        ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        if (c && !(c & 240)) {
          for (
            y = h, K = c, w = E;
            (l = n.lencode[w + ((u & ((1 << (y + K)) - 1)) >> y)]),
              (h = l >>> 24),
              (c = (l >>> 16) & 255),
              (E = l & 65535),
              !(y + h <= f);

          ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          ;((u >>>= y), (f -= y), (n.back += y))
        }
        if (((u >>>= h), (f -= h), (n.back += h), (n.length = E), c === 0)) {
          n.mode = Ci
          break
        }
        if (c & 32) {
          ;((n.back = -1), (n.mode = Ft))
          break
        }
        if (c & 64) {
          ;((e.msg = 'invalid literal/length code'), (n.mode = Be))
          break
        }
        ;((n.extra = c & 15), (n.mode = Ei))
      case Ei:
        if (n.extra) {
          for (I = n.extra; f < I; ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          ;((n.length += u & ((1 << n.extra) - 1)),
            (u >>>= n.extra),
            (f -= n.extra),
            (n.back += n.extra))
        }
        ;((n.was = n.length), (n.mode = Ki))
      case Ki:
        for (
          ;
          (l = n.distcode[u & ((1 << n.distbits) - 1)]),
            (h = l >>> 24),
            (c = (l >>> 16) & 255),
            (E = l & 65535),
            !(h <= f);

        ) {
          if (s === 0) break e
          ;(s--, (u += r[o++] << f), (f += 8))
        }
        if (!(c & 240)) {
          for (
            y = h, K = c, w = E;
            (l = n.distcode[w + ((u & ((1 << (y + K)) - 1)) >> y)]),
              (h = l >>> 24),
              (c = (l >>> 16) & 255),
              (E = l & 65535),
              !(y + h <= f);

          ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          ;((u >>>= y), (f -= y), (n.back += y))
        }
        if (((u >>>= h), (f -= h), (n.back += h), c & 64)) {
          ;((e.msg = 'invalid distance code'), (n.mode = Be))
          break
        }
        ;((n.offset = E), (n.extra = c & 15), (n.mode = Xi))
      case Xi:
        if (n.extra) {
          for (I = n.extra; f < I; ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          ;((n.offset += u & ((1 << n.extra) - 1)),
            (u >>>= n.extra),
            (f -= n.extra),
            (n.back += n.extra))
        }
        if (n.offset > n.dmax) {
          ;((e.msg = 'invalid distance too far back'), (n.mode = Be))
          break
        }
        n.mode = Fi
      case Fi:
        if (x === 0) break e
        if (((g = X - x), n.offset > g)) {
          if (((g = n.offset - g), g > n.whave && n.sane)) {
            ;((e.msg = 'invalid distance too far back'), (n.mode = Be))
            break
          }
          ;(g > n.wnext ? ((g -= n.wnext), (k = n.wsize - g)) : (k = n.wnext - g),
            g > n.length && (g = n.length),
            (_ = n.window))
        } else ((_ = i), (k = a - n.offset), (g = n.length))
        ;(g > x && (g = x), (x -= g), (n.length -= g))
        do i[a++] = _[k++]
        while (--g)
        n.length === 0 && (n.mode = Ir)
        break
      case Ci:
        if (x === 0) break e
        ;((i[a++] = n.length), x--, (n.mode = Ir))
        break
      case fn:
        if (n.wrap) {
          for (; f < 32; ) {
            if (s === 0) break e
            ;(s--, (u |= r[o++] << f), (f += 8))
          }
          if (
            ((X -= x),
            (e.total_out += X),
            (n.total += X),
            X &&
              (e.adler = n.check =
                n.flags
                  ? (0, kt.default)(n.check, i, X, a - X)
                  : (0, kn.default)(n.check, i, X, a - X)),
            (X = x),
            (n.flags ? u : Ti(u)) !== n.check)
          ) {
            ;((e.msg = 'incorrect data check'), (n.mode = Be))
            break
          }
          ;((u = 0), (f = 0))
        }
        n.mode = Ai
      case Ai:
        if (n.wrap && n.flags) {
          for (; f < 32; ) {
            if (s === 0) break e
            ;(s--, (u += r[o++] << f), (f += 8))
          }
          if (u !== (n.total & 4294967295)) {
            ;((e.msg = 'incorrect length check'), (n.mode = Be))
            break
          }
          ;((u = 0), (f = 0))
        }
        n.mode = Pi
      case Pi:
        C = vs
        break e
      case Be:
        C = ka
        break e
      case Ka:
        return Sa
      case gs:
      default:
        return _t
    }
  return (
    (e.next_out = a),
    (e.avail_out = x),
    (e.next_in = o),
    (e.avail_in = s),
    (n.hold = u),
    (n.bits = f),
    (n.wsize || (X !== e.avail_out && n.mode < Be && (n.mode < fn || t !== li))) &&
      Pa(e, e.output, e.next_out, X - e.avail_out),
    (F -= e.avail_in),
    (X -= e.avail_out),
    (e.total_in += F),
    (e.total_out += X),
    (n.total += X),
    n.wrap &&
      X &&
      (e.adler = n.check =
        n.flags
          ? (0, kt.default)(n.check, i, X, e.next_out - X)
          : (0, kn.default)(n.check, i, X, e.next_out - X)),
    (e.data_type =
      n.bits +
      (n.last ? 64 : 0) +
      (n.mode === Ft ? 128 : 0) +
      (n.mode === Br || n.mode === ln ? 256 : 0)),
    ((F === 0 && X === 0) || t === li) && C === Vt && (C = xs),
    C
  )
}
function Fs(e) {
  if (!e || !e.state) return _t
  var t = e.state
  return (t.window && (t.window = null), (e.state = null), Vt)
}
function Cs(e, t) {
  var n
  return !e || !e.state || ((n = e.state), !(n.wrap & 2)) ? _t : ((n.head = t), (t.done = !1), Vt)
}
function As(e, t) {
  var n = t.length,
    r,
    i,
    o
  return !e || !e.state || ((r = e.state), r.wrap !== 0 && r.mode !== zr)
    ? _t
    : r.mode === zr && ((i = 1), (i = (0, kn.default)(i, t, n, 0)), i !== r.check)
      ? ka
      : ((o = Pa(e, t, n, n)), o ? ((r.mode = Ka), Sa) : ((r.havedict = 1), Vt))
}
Ke.inflateInfo = 'pako inflate (from Nodeca project)'
var Bn = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = t))
  function t() {
    ;((this.input = null),
      (this.next_in = 0),
      (this.avail_in = 0),
      (this.total_in = 0),
      (this.output = null),
      (this.next_out = 0),
      (this.avail_out = 0),
      (this.total_out = 0),
      (this.msg = ''),
      (this.state = null),
      (this.data_type = 2),
      (this.adler = 0))
  }
})(Bn)
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = Ke,
    n = r(Bn)
  function r(f) {
    return f && f.__esModule ? f : { default: f }
  }
  function i(f) {
    '@babel/helpers - typeof'
    return (
      (i =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (F) {
              return typeof F
            }
          : function (F) {
              return F &&
                typeof Symbol == 'function' &&
                F.constructor === Symbol &&
                F !== Symbol.prototype
                ? 'symbol'
                : typeof F
            }),
      i(f)
    )
  }
  function o(f, F) {
    if (!(f instanceof F)) throw new TypeError('Cannot call a class as a function')
  }
  function a(f, F) {
    for (var X = 0; X < F.length; X++) {
      var g = F[X]
      ;((g.enumerable = g.enumerable || !1),
        (g.configurable = !0),
        'value' in g && (g.writable = !0),
        Object.defineProperty(f, x(g.key), g))
    }
  }
  function s(f, F, X) {
    return (F && a(f.prototype, F), Object.defineProperty(f, 'prototype', { writable: !1 }), f)
  }
  function x(f) {
    var F = u(f, 'string')
    return i(F) == 'symbol' ? F : F + ''
  }
  function u(f, F) {
    if (i(f) != 'object' || !f) return f
    var X = f[Symbol.toPrimitive]
    if (X !== void 0) {
      var g = X.call(f, F)
      if (i(g) != 'object') return g
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(f)
  }
  e.default = (function () {
    function f() {
      ;(o(this, f),
        (this.strm = new n.default()),
        (this.chunkSize = 1024 * 10 * 10),
        (this.strm.output = new Uint8Array(this.chunkSize)),
        (0, t.inflateInit)(this.strm))
    }
    return s(f, [
      {
        key: 'setInput',
        value: function (X) {
          X
            ? ((this.strm.input = X),
              (this.strm.avail_in = this.strm.input.length),
              (this.strm.next_in = 0))
            : ((this.strm.input = null), (this.strm.avail_in = 0), (this.strm.next_in = 0))
        }
      },
      {
        key: 'inflate',
        value: function (X) {
          ;(X > this.chunkSize &&
            ((this.chunkSize = X), (this.strm.output = new Uint8Array(this.chunkSize))),
            (this.strm.next_out = 0),
            (this.strm.avail_out = X))
          var g = (0, t.inflate)(this.strm, 0)
          if (g < 0) throw new Error('zlib inflate failed')
          if (this.strm.next_out != X) throw new Error('Incomplete zlib block')
          return new Uint8Array(this.strm.output.buffer, 0, this.strm.next_out)
        }
      },
      {
        key: 'reset',
        value: function () {
          ;(0, t.inflateReset)(this.strm)
        }
      }
    ])
  })()
})(Kr)
var Ta = {},
  ge = {},
  er = {}
function Sn(e) {
  '@babel/helpers - typeof'
  return (
    (Sn =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    Sn(e)
  )
}
Object.defineProperty(er, '__esModule', { value: !0 })
er._tr_align = qs
er._tr_flush_block = Ys
er._tr_init = Zs
er._tr_stored_block = Va
er._tr_tally = $s
var Ps = Ts(ot)
function La(e) {
  if (typeof WeakMap != 'function') return null
  var t = new WeakMap(),
    n = new WeakMap()
  return (La = function (i) {
    return i ? n : t
  })(e)
}
function Ts(e, t) {
  if (e && e.__esModule) return e
  if (e === null || (Sn(e) != 'object' && typeof e != 'function')) return { default: e }
  var n = La(t)
  if (n && n.has(e)) return n.get(e)
  var r = { __proto__: null },
    i = Object.defineProperty && Object.getOwnPropertyDescriptor
  for (var o in e)
    if (o !== 'default' && {}.hasOwnProperty.call(e, o)) {
      var a = i ? Object.getOwnPropertyDescriptor(e, o) : null
      a && (a.get || a.set) ? Object.defineProperty(r, o, a) : (r[o] = e[o])
    }
  return ((r.default = e), n && n.set(e, r), r)
}
var Ls = 4,
  Ri = 0,
  Mi = 1,
  Rs = 2
function sr(e) {
  for (var t = e.length; --t >= 0; ) e[t] = 0
}
var Ms = 0,
  Ra = 1,
  Ds = 2,
  Os = 3,
  Bs = 258,
  In = 29,
  Xr = 256,
  xr = Xr + 1 + In,
  ir = 30,
  Qn = 19,
  Ma = 2 * xr + 1,
  zt = 15,
  dn = 16,
  Is = 7,
  Un = 256,
  Da = 16,
  Oa = 17,
  Ba = 18,
  En = [0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0],
  Nr = [
    0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13
  ],
  Qs = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 3, 7],
  Ia = [16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15],
  Us = 512,
  Ct = new Array((xr + 2) * 2)
sr(Ct)
var pr = new Array(ir * 2)
sr(pr)
var gr = new Array(Us)
sr(gr)
var br = new Array(Bs - Os + 1)
sr(br)
var Nn = new Array(In)
sr(Nn)
var Gr = new Array(ir)
sr(Gr)
function _n(e, t, n, r, i) {
  ;((this.static_tree = e),
    (this.extra_bits = t),
    (this.extra_base = n),
    (this.elems = r),
    (this.max_length = i),
    (this.has_stree = e && e.length))
}
var Qa, Ua, Na
function pn(e, t) {
  ;((this.dyn_tree = e), (this.max_code = 0), (this.stat_desc = t))
}
function ja(e) {
  return e < 256 ? gr[e] : gr[256 + (e >>> 7)]
}
function mr(e, t) {
  ;((e.pending_buf[e.pending++] = t & 255), (e.pending_buf[e.pending++] = (t >>> 8) & 255))
}
function at(e, t, n) {
  e.bi_valid > dn - n
    ? ((e.bi_buf |= (t << e.bi_valid) & 65535),
      mr(e, e.bi_buf),
      (e.bi_buf = t >> (dn - e.bi_valid)),
      (e.bi_valid += n - dn))
    : ((e.bi_buf |= (t << e.bi_valid) & 65535), (e.bi_valid += n))
}
function Kt(e, t, n) {
  at(e, n[t * 2], n[t * 2 + 1])
}
function Ha(e, t) {
  var n = 0
  do ((n |= e & 1), (e >>>= 1), (n <<= 1))
  while (--t > 0)
  return n >>> 1
}
function Ns(e) {
  e.bi_valid === 16
    ? (mr(e, e.bi_buf), (e.bi_buf = 0), (e.bi_valid = 0))
    : e.bi_valid >= 8 &&
      ((e.pending_buf[e.pending++] = e.bi_buf & 255), (e.bi_buf >>= 8), (e.bi_valid -= 8))
}
function js(e, t) {
  var n = t.dyn_tree,
    r = t.max_code,
    i = t.stat_desc.static_tree,
    o = t.stat_desc.has_stree,
    a = t.stat_desc.extra_bits,
    s = t.stat_desc.extra_base,
    x = t.stat_desc.max_length,
    u,
    f,
    F,
    X,
    g,
    k,
    _ = 0
  for (X = 0; X <= zt; X++) e.bl_count[X] = 0
  for (n[e.heap[e.heap_max] * 2 + 1] = 0, u = e.heap_max + 1; u < Ma; u++)
    ((f = e.heap[u]),
      (X = n[n[f * 2 + 1] * 2 + 1] + 1),
      X > x && ((X = x), _++),
      (n[f * 2 + 1] = X),
      !(f > r) &&
        (e.bl_count[X]++,
        (g = 0),
        f >= s && (g = a[f - s]),
        (k = n[f * 2]),
        (e.opt_len += k * (X + g)),
        o && (e.static_len += k * (i[f * 2 + 1] + g))))
  if (_ !== 0) {
    do {
      for (X = x - 1; e.bl_count[X] === 0; ) X--
      ;(e.bl_count[X]--, (e.bl_count[X + 1] += 2), e.bl_count[x]--, (_ -= 2))
    } while (_ > 0)
    for (X = x; X !== 0; X--)
      for (f = e.bl_count[X]; f !== 0; )
        ((F = e.heap[--u]),
          !(F > r) &&
            (n[F * 2 + 1] !== X &&
              ((e.opt_len += (X - n[F * 2 + 1]) * n[F * 2]), (n[F * 2 + 1] = X)),
            f--))
  }
}
function Wa(e, t, n) {
  var r = new Array(zt + 1),
    i = 0,
    o,
    a
  for (o = 1; o <= zt; o++) r[o] = i = (i + n[o - 1]) << 1
  for (a = 0; a <= t; a++) {
    var s = e[a * 2 + 1]
    s !== 0 && (e[a * 2] = Ha(r[s]++, s))
  }
}
function Hs() {
  var e,
    t,
    n,
    r,
    i,
    o = new Array(zt + 1)
  for (n = 0, r = 0; r < In - 1; r++) for (Nn[r] = n, e = 0; e < 1 << En[r]; e++) br[n++] = r
  for (br[n - 1] = r, i = 0, r = 0; r < 16; r++)
    for (Gr[r] = i, e = 0; e < 1 << Nr[r]; e++) gr[i++] = r
  for (i >>= 7; r < ir; r++)
    for (Gr[r] = i << 7, e = 0; e < 1 << (Nr[r] - 7); e++) gr[256 + i++] = r
  for (t = 0; t <= zt; t++) o[t] = 0
  for (e = 0; e <= 143; ) ((Ct[e * 2 + 1] = 8), e++, o[8]++)
  for (; e <= 255; ) ((Ct[e * 2 + 1] = 9), e++, o[9]++)
  for (; e <= 279; ) ((Ct[e * 2 + 1] = 7), e++, o[7]++)
  for (; e <= 287; ) ((Ct[e * 2 + 1] = 8), e++, o[8]++)
  for (Wa(Ct, xr + 1, o), e = 0; e < ir; e++) ((pr[e * 2 + 1] = 5), (pr[e * 2] = Ha(e, 5)))
  ;((Qa = new _n(Ct, En, Xr + 1, xr, zt)),
    (Ua = new _n(pr, Nr, 0, ir, zt)),
    (Na = new _n(new Array(0), Qs, 0, Qn, Is)))
}
function za(e) {
  var t
  for (t = 0; t < xr; t++) e.dyn_ltree[t * 2] = 0
  for (t = 0; t < ir; t++) e.dyn_dtree[t * 2] = 0
  for (t = 0; t < Qn; t++) e.bl_tree[t * 2] = 0
  ;((e.dyn_ltree[Un * 2] = 1), (e.opt_len = e.static_len = 0), (e.last_lit = e.matches = 0))
}
function Ga(e) {
  ;(e.bi_valid > 8 ? mr(e, e.bi_buf) : e.bi_valid > 0 && (e.pending_buf[e.pending++] = e.bi_buf),
    (e.bi_buf = 0),
    (e.bi_valid = 0))
}
function Ws(e, t, n, r) {
  ;(Ga(e),
    mr(e, n),
    mr(e, ~n),
    Ps.arraySet(e.pending_buf, e.window, t, n, e.pending),
    (e.pending += n))
}
function Di(e, t, n, r) {
  var i = t * 2,
    o = n * 2
  return e[i] < e[o] || (e[i] === e[o] && r[t] <= r[n])
}
function vn(e, t, n) {
  for (
    var r = e.heap[n], i = n << 1;
    i <= e.heap_len &&
    (i < e.heap_len && Di(t, e.heap[i + 1], e.heap[i], e.depth) && i++,
    !Di(t, r, e.heap[i], e.depth));

  )
    ((e.heap[n] = e.heap[i]), (n = i), (i <<= 1))
  e.heap[n] = r
}
function Oi(e, t, n) {
  var r,
    i,
    o = 0,
    a,
    s
  if (e.last_lit !== 0)
    do
      ((r = (e.pending_buf[e.d_buf + o * 2] << 8) | e.pending_buf[e.d_buf + o * 2 + 1]),
        (i = e.pending_buf[e.l_buf + o]),
        o++,
        r === 0
          ? Kt(e, i, t)
          : ((a = br[i]),
            Kt(e, a + Xr + 1, t),
            (s = En[a]),
            s !== 0 && ((i -= Nn[a]), at(e, i, s)),
            r--,
            (a = ja(r)),
            Kt(e, a, n),
            (s = Nr[a]),
            s !== 0 && ((r -= Gr[a]), at(e, r, s))))
    while (o < e.last_lit)
  Kt(e, Un, t)
}
function Kn(e, t) {
  var n = t.dyn_tree,
    r = t.stat_desc.static_tree,
    i = t.stat_desc.has_stree,
    o = t.stat_desc.elems,
    a,
    s,
    x = -1,
    u
  for (e.heap_len = 0, e.heap_max = Ma, a = 0; a < o; a++)
    n[a * 2] !== 0 ? ((e.heap[++e.heap_len] = x = a), (e.depth[a] = 0)) : (n[a * 2 + 1] = 0)
  for (; e.heap_len < 2; )
    ((u = e.heap[++e.heap_len] = x < 2 ? ++x : 0),
      (n[u * 2] = 1),
      (e.depth[u] = 0),
      e.opt_len--,
      i && (e.static_len -= r[u * 2 + 1]))
  for (t.max_code = x, a = e.heap_len >> 1; a >= 1; a--) vn(e, n, a)
  u = o
  do
    ((a = e.heap[1]),
      (e.heap[1] = e.heap[e.heap_len--]),
      vn(e, n, 1),
      (s = e.heap[1]),
      (e.heap[--e.heap_max] = a),
      (e.heap[--e.heap_max] = s),
      (n[u * 2] = n[a * 2] + n[s * 2]),
      (e.depth[u] = (e.depth[a] >= e.depth[s] ? e.depth[a] : e.depth[s]) + 1),
      (n[a * 2 + 1] = n[s * 2 + 1] = u),
      (e.heap[1] = u++),
      vn(e, n, 1))
  while (e.heap_len >= 2)
  ;((e.heap[--e.heap_max] = e.heap[1]), js(e, t), Wa(n, x, e.bl_count))
}
function Bi(e, t, n) {
  var r,
    i = -1,
    o,
    a = t[0 * 2 + 1],
    s = 0,
    x = 7,
    u = 4
  for (a === 0 && ((x = 138), (u = 3)), t[(n + 1) * 2 + 1] = 65535, r = 0; r <= n; r++)
    ((o = a),
      (a = t[(r + 1) * 2 + 1]),
      !(++s < x && o === a) &&
        (s < u
          ? (e.bl_tree[o * 2] += s)
          : o !== 0
            ? (o !== i && e.bl_tree[o * 2]++, e.bl_tree[Da * 2]++)
            : s <= 10
              ? e.bl_tree[Oa * 2]++
              : e.bl_tree[Ba * 2]++,
        (s = 0),
        (i = o),
        a === 0 ? ((x = 138), (u = 3)) : o === a ? ((x = 6), (u = 3)) : ((x = 7), (u = 4))))
}
function Ii(e, t, n) {
  var r,
    i = -1,
    o,
    a = t[0 * 2 + 1],
    s = 0,
    x = 7,
    u = 4
  for (a === 0 && ((x = 138), (u = 3)), r = 0; r <= n; r++)
    if (((o = a), (a = t[(r + 1) * 2 + 1]), !(++s < x && o === a))) {
      if (s < u)
        do Kt(e, o, e.bl_tree)
        while (--s !== 0)
      else
        o !== 0
          ? (o !== i && (Kt(e, o, e.bl_tree), s--), Kt(e, Da, e.bl_tree), at(e, s - 3, 2))
          : s <= 10
            ? (Kt(e, Oa, e.bl_tree), at(e, s - 3, 3))
            : (Kt(e, Ba, e.bl_tree), at(e, s - 11, 7))
      ;((s = 0),
        (i = o),
        a === 0 ? ((x = 138), (u = 3)) : o === a ? ((x = 6), (u = 3)) : ((x = 7), (u = 4)))
    }
}
function zs(e) {
  var t
  for (
    Bi(e, e.dyn_ltree, e.l_desc.max_code),
      Bi(e, e.dyn_dtree, e.d_desc.max_code),
      Kn(e, e.bl_desc),
      t = Qn - 1;
    t >= 3 && e.bl_tree[Ia[t] * 2 + 1] === 0;
    t--
  );
  return ((e.opt_len += 3 * (t + 1) + 5 + 5 + 4), t)
}
function Gs(e, t, n, r) {
  var i
  for (at(e, t - 257, 5), at(e, n - 1, 5), at(e, r - 4, 4), i = 0; i < r; i++)
    at(e, e.bl_tree[Ia[i] * 2 + 1], 3)
  ;(Ii(e, e.dyn_ltree, t - 1), Ii(e, e.dyn_dtree, n - 1))
}
function Vs(e) {
  var t = 4093624447,
    n
  for (n = 0; n <= 31; n++, t >>>= 1) if (t & 1 && e.dyn_ltree[n * 2] !== 0) return Ri
  if (e.dyn_ltree[9 * 2] !== 0 || e.dyn_ltree[10 * 2] !== 0 || e.dyn_ltree[13 * 2] !== 0) return Mi
  for (n = 32; n < Xr; n++) if (e.dyn_ltree[n * 2] !== 0) return Mi
  return Ri
}
var Qi = !1
function Zs(e) {
  ;(Qi || (Hs(), (Qi = !0)),
    (e.l_desc = new pn(e.dyn_ltree, Qa)),
    (e.d_desc = new pn(e.dyn_dtree, Ua)),
    (e.bl_desc = new pn(e.bl_tree, Na)),
    (e.bi_buf = 0),
    (e.bi_valid = 0),
    za(e))
}
function Va(e, t, n, r) {
  ;(at(e, (Ms << 1) + (r ? 1 : 0), 3), Ws(e, t, n))
}
function qs(e) {
  ;(at(e, Ra << 1, 3), Kt(e, Un, Ct), Ns(e))
}
function Ys(e, t, n, r) {
  var i,
    o,
    a = 0
  ;(e.level > 0
    ? (e.strm.data_type === Rs && (e.strm.data_type = Vs(e)),
      Kn(e, e.l_desc),
      Kn(e, e.d_desc),
      (a = zs(e)),
      (i = (e.opt_len + 3 + 7) >>> 3),
      (o = (e.static_len + 3 + 7) >>> 3),
      o <= i && (i = o))
    : (i = o = n + 5),
    n + 4 <= i && t !== -1
      ? Va(e, t, n, r)
      : e.strategy === Ls || o === i
        ? (at(e, (Ra << 1) + (r ? 1 : 0), 3), Oi(e, Ct, pr))
        : (at(e, (Ds << 1) + (r ? 1 : 0), 3),
          Gs(e, e.l_desc.max_code + 1, e.d_desc.max_code + 1, a + 1),
          Oi(e, e.dyn_ltree, e.dyn_dtree)),
    za(e),
    r && Ga(e))
}
function $s(e, t, n) {
  return (
    (e.pending_buf[e.d_buf + e.last_lit * 2] = (t >>> 8) & 255),
    (e.pending_buf[e.d_buf + e.last_lit * 2 + 1] = t & 255),
    (e.pending_buf[e.l_buf + e.last_lit] = n & 255),
    e.last_lit++,
    t === 0
      ? e.dyn_ltree[n * 2]++
      : (e.matches++, t--, e.dyn_ltree[(br[n] + Xr + 1) * 2]++, e.dyn_dtree[ja(t) * 2]++),
    e.last_lit === e.lit_bufsize - 1
  )
}
var Za = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }),
    (e.default = void 0),
    (e.default = {
      2: 'need dictionary',
      1: 'stream end',
      0: '',
      '-1': 'file error',
      '-2': 'stream error',
      '-3': 'data error',
      '-4': 'insufficient memory',
      '-5': 'buffer error',
      '-6': 'incompatible version'
    }))
})(Za)
function Xn(e) {
  '@babel/helpers - typeof'
  return (
    (Xn =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    Xn(e)
  )
}
Object.defineProperty(ge, '__esModule', { value: !0 })
ge.Z_UNKNOWN =
  ge.Z_STREAM_ERROR =
  ge.Z_STREAM_END =
  ge.Z_RLE =
  ge.Z_PARTIAL_FLUSH =
  ge.Z_OK =
  ge.Z_NO_FLUSH =
  ge.Z_HUFFMAN_ONLY =
  ge.Z_FULL_FLUSH =
  ge.Z_FIXED =
  ge.Z_FINISH =
  ge.Z_FILTERED =
  ge.Z_DEFLATED =
  ge.Z_DEFAULT_STRATEGY =
  ge.Z_DEFAULT_COMPRESSION =
  ge.Z_DATA_ERROR =
  ge.Z_BUF_ERROR =
  ge.Z_BLOCK =
    void 0
ge.deflate = Fu
ge.deflateEnd = Cu
ge.deflateInfo = void 0
ge.deflateInit = Xu
ge.deflateInit2 = ro
ge.deflateReset = to
ge.deflateResetKeep = eo
ge.deflateSetDictionary = Au
ge.deflateSetHeader = Ku
var tt = $a(ot),
  ht = $a(er),
  qa = jn(Dn),
  Lt = jn(On),
  Js = jn(Za)
function jn(e) {
  return e && e.__esModule ? e : { default: e }
}
function Ya(e) {
  if (typeof WeakMap != 'function') return null
  var t = new WeakMap(),
    n = new WeakMap()
  return (Ya = function (i) {
    return i ? n : t
  })(e)
}
function $a(e, t) {
  if (e && e.__esModule) return e
  if (e === null || (Xn(e) != 'object' && typeof e != 'function')) return { default: e }
  var n = Ya(t)
  if (n && n.has(e)) return n.get(e)
  var r = { __proto__: null },
    i = Object.defineProperty && Object.getOwnPropertyDescriptor
  for (var o in e)
    if (o !== 'default' && {}.hasOwnProperty.call(e, o)) {
      var a = i ? Object.getOwnPropertyDescriptor(e, o) : null
      a && (a.get || a.set) ? Object.defineProperty(r, o, a) : (r[o] = e[o])
    }
  return ((r.default = e), n && n.set(e, r), r)
}
var tr = (ge.Z_NO_FLUSH = 0),
  eu = (ge.Z_PARTIAL_FLUSH = 1),
  tu = (ge.Z_FULL_FLUSH = 3),
  It = (ge.Z_FINISH = 4),
  Ui = (ge.Z_BLOCK = 5),
  Xt = (ge.Z_OK = 0),
  Ni = (ge.Z_STREAM_END = 1),
  dt = (ge.Z_STREAM_ERROR = -2),
  ru = (ge.Z_DATA_ERROR = -3),
  yn = (ge.Z_BUF_ERROR = -5),
  nu = (ge.Z_DEFAULT_COMPRESSION = -1),
  iu = (ge.Z_FILTERED = 1),
  Qr = (ge.Z_HUFFMAN_ONLY = 2),
  au = (ge.Z_RLE = 3),
  ou = (ge.Z_FIXED = 4),
  su = (ge.Z_DEFAULT_STRATEGY = 0),
  uu = (ge.Z_UNKNOWN = 2),
  Yr = (ge.Z_DEFLATED = 8),
  lu = 9,
  fu = 15,
  cu = 8,
  hu = 29,
  du = 256,
  Fn = du + 1 + hu,
  _u = 30,
  pu = 19,
  vu = 2 * Fn + 1,
  yu = 15,
  Ee = 3,
  Dt = 258,
  vt = Dt + Ee + 1,
  xu = 32,
  $r = 42,
  Cn = 69,
  jr = 73,
  Hr = 91,
  Wr = 103,
  Gt = 113,
  dr = 666,
  Qe = 1,
  Fr = 2,
  Zt = 3,
  ur = 4,
  gu = 3
function Ot(e, t) {
  return ((e.msg = Js.default[t]), t)
}
function ji(e) {
  return (e << 1) - (e > 4 ? 9 : 0)
}
function Mt(e) {
  for (var t = e.length; --t >= 0; ) e[t] = 0
}
function Rt(e) {
  var t = e.state,
    n = t.pending
  ;(n > e.avail_out && (n = e.avail_out),
    n !== 0 &&
      (tt.arraySet(e.output, t.pending_buf, t.pending_out, n, e.next_out),
      (e.next_out += n),
      (t.pending_out += n),
      (e.total_out += n),
      (e.avail_out -= n),
      (t.pending -= n),
      t.pending === 0 && (t.pending_out = 0)))
}
function Ge(e, t) {
  ;(ht._tr_flush_block(e, e.block_start >= 0 ? e.block_start : -1, e.strstart - e.block_start, t),
    (e.block_start = e.strstart),
    Rt(e.strm))
}
function Fe(e, t) {
  e.pending_buf[e.pending++] = t
}
function cr(e, t) {
  ;((e.pending_buf[e.pending++] = (t >>> 8) & 255), (e.pending_buf[e.pending++] = t & 255))
}
function bu(e, t, n, r) {
  var i = e.avail_in
  return (
    i > r && (i = r),
    i === 0
      ? 0
      : ((e.avail_in -= i),
        tt.arraySet(t, e.input, e.next_in, i, n),
        e.state.wrap === 1
          ? (e.adler = (0, qa.default)(e.adler, t, i, n))
          : e.state.wrap === 2 && (e.adler = (0, Lt.default)(e.adler, t, i, n)),
        (e.next_in += i),
        (e.total_in += i),
        i)
  )
}
function Ja(e, t) {
  var n = e.max_chain_length,
    r = e.strstart,
    i,
    o,
    a = e.prev_length,
    s = e.nice_match,
    x = e.strstart > e.w_size - vt ? e.strstart - (e.w_size - vt) : 0,
    u = e.window,
    f = e.w_mask,
    F = e.prev,
    X = e.strstart + Dt,
    g = u[r + a - 1],
    k = u[r + a]
  ;(e.prev_length >= e.good_match && (n >>= 2), s > e.lookahead && (s = e.lookahead))
  do
    if (
      ((i = t), !(u[i + a] !== k || u[i + a - 1] !== g || u[i] !== u[r] || u[++i] !== u[r + 1]))
    ) {
      ;((r += 2), i++)
      do;
      while (
        u[++r] === u[++i] &&
        u[++r] === u[++i] &&
        u[++r] === u[++i] &&
        u[++r] === u[++i] &&
        u[++r] === u[++i] &&
        u[++r] === u[++i] &&
        u[++r] === u[++i] &&
        u[++r] === u[++i] &&
        r < X
      )
      if (((o = Dt - (X - r)), (r = X - Dt), o > a)) {
        if (((e.match_start = t), (a = o), o >= s)) break
        ;((g = u[r + a - 1]), (k = u[r + a]))
      }
    }
  while ((t = F[t & f]) > x && --n !== 0)
  return a <= e.lookahead ? a : e.lookahead
}
function qt(e) {
  var t = e.w_size,
    n,
    r,
    i,
    o,
    a
  do {
    if (((o = e.window_size - e.lookahead - e.strstart), e.strstart >= t + (t - vt))) {
      ;(tt.arraySet(e.window, e.window, t, t, 0),
        (e.match_start -= t),
        (e.strstart -= t),
        (e.block_start -= t),
        (r = e.hash_size),
        (n = r))
      do ((i = e.head[--n]), (e.head[n] = i >= t ? i - t : 0))
      while (--r)
      ;((r = t), (n = r))
      do ((i = e.prev[--n]), (e.prev[n] = i >= t ? i - t : 0))
      while (--r)
      o += t
    }
    if (e.strm.avail_in === 0) break
    if (
      ((r = bu(e.strm, e.window, e.strstart + e.lookahead, o)),
      (e.lookahead += r),
      e.lookahead + e.insert >= Ee)
    )
      for (
        a = e.strstart - e.insert,
          e.ins_h = e.window[a],
          e.ins_h = ((e.ins_h << e.hash_shift) ^ e.window[a + 1]) & e.hash_mask;
        e.insert &&
        ((e.ins_h = ((e.ins_h << e.hash_shift) ^ e.window[a + Ee - 1]) & e.hash_mask),
        (e.prev[a & e.w_mask] = e.head[e.ins_h]),
        (e.head[e.ins_h] = a),
        a++,
        e.insert--,
        !(e.lookahead + e.insert < Ee));

      );
  } while (e.lookahead < vt && e.strm.avail_in !== 0)
}
function mu(e, t) {
  var n = 65535
  for (n > e.pending_buf_size - 5 && (n = e.pending_buf_size - 5); ; ) {
    if (e.lookahead <= 1) {
      if ((qt(e), e.lookahead === 0 && t === tr)) return Qe
      if (e.lookahead === 0) break
    }
    ;((e.strstart += e.lookahead), (e.lookahead = 0))
    var r = e.block_start + n
    if (
      ((e.strstart === 0 || e.strstart >= r) &&
        ((e.lookahead = e.strstart - r), (e.strstart = r), Ge(e, !1), e.strm.avail_out === 0)) ||
      (e.strstart - e.block_start >= e.w_size - vt && (Ge(e, !1), e.strm.avail_out === 0))
    )
      return Qe
  }
  return (
    (e.insert = 0),
    t === It
      ? (Ge(e, !0), e.strm.avail_out === 0 ? Zt : ur)
      : (e.strstart > e.block_start && (Ge(e, !1), e.strm.avail_out === 0), Qe)
  )
}
function xn(e, t) {
  for (var n, r; ; ) {
    if (e.lookahead < vt) {
      if ((qt(e), e.lookahead < vt && t === tr)) return Qe
      if (e.lookahead === 0) break
    }
    if (
      ((n = 0),
      e.lookahead >= Ee &&
        ((e.ins_h = ((e.ins_h << e.hash_shift) ^ e.window[e.strstart + Ee - 1]) & e.hash_mask),
        (n = e.prev[e.strstart & e.w_mask] = e.head[e.ins_h]),
        (e.head[e.ins_h] = e.strstart)),
      n !== 0 && e.strstart - n <= e.w_size - vt && (e.match_length = Ja(e, n)),
      e.match_length >= Ee)
    )
      if (
        ((r = ht._tr_tally(e, e.strstart - e.match_start, e.match_length - Ee)),
        (e.lookahead -= e.match_length),
        e.match_length <= e.max_lazy_match && e.lookahead >= Ee)
      ) {
        e.match_length--
        do
          (e.strstart++,
            (e.ins_h = ((e.ins_h << e.hash_shift) ^ e.window[e.strstart + Ee - 1]) & e.hash_mask),
            (n = e.prev[e.strstart & e.w_mask] = e.head[e.ins_h]),
            (e.head[e.ins_h] = e.strstart))
        while (--e.match_length !== 0)
        e.strstart++
      } else
        ((e.strstart += e.match_length),
          (e.match_length = 0),
          (e.ins_h = e.window[e.strstart]),
          (e.ins_h = ((e.ins_h << e.hash_shift) ^ e.window[e.strstart + 1]) & e.hash_mask))
    else ((r = ht._tr_tally(e, 0, e.window[e.strstart])), e.lookahead--, e.strstart++)
    if (r && (Ge(e, !1), e.strm.avail_out === 0)) return Qe
  }
  return (
    (e.insert = e.strstart < Ee - 1 ? e.strstart : Ee - 1),
    t === It
      ? (Ge(e, !0), e.strm.avail_out === 0 ? Zt : ur)
      : e.last_lit && (Ge(e, !1), e.strm.avail_out === 0)
        ? Qe
        : Fr
  )
}
function rr(e, t) {
  for (var n, r, i; ; ) {
    if (e.lookahead < vt) {
      if ((qt(e), e.lookahead < vt && t === tr)) return Qe
      if (e.lookahead === 0) break
    }
    if (
      ((n = 0),
      e.lookahead >= Ee &&
        ((e.ins_h = ((e.ins_h << e.hash_shift) ^ e.window[e.strstart + Ee - 1]) & e.hash_mask),
        (n = e.prev[e.strstart & e.w_mask] = e.head[e.ins_h]),
        (e.head[e.ins_h] = e.strstart)),
      (e.prev_length = e.match_length),
      (e.prev_match = e.match_start),
      (e.match_length = Ee - 1),
      n !== 0 &&
        e.prev_length < e.max_lazy_match &&
        e.strstart - n <= e.w_size - vt &&
        ((e.match_length = Ja(e, n)),
        e.match_length <= 5 &&
          (e.strategy === iu || (e.match_length === Ee && e.strstart - e.match_start > 4096)) &&
          (e.match_length = Ee - 1)),
      e.prev_length >= Ee && e.match_length <= e.prev_length)
    ) {
      ;((i = e.strstart + e.lookahead - Ee),
        (r = ht._tr_tally(e, e.strstart - 1 - e.prev_match, e.prev_length - Ee)),
        (e.lookahead -= e.prev_length - 1),
        (e.prev_length -= 2))
      do
        ++e.strstart <= i &&
          ((e.ins_h = ((e.ins_h << e.hash_shift) ^ e.window[e.strstart + Ee - 1]) & e.hash_mask),
          (n = e.prev[e.strstart & e.w_mask] = e.head[e.ins_h]),
          (e.head[e.ins_h] = e.strstart))
      while (--e.prev_length !== 0)
      if (
        ((e.match_available = 0),
        (e.match_length = Ee - 1),
        e.strstart++,
        r && (Ge(e, !1), e.strm.avail_out === 0))
      )
        return Qe
    } else if (e.match_available) {
      if (
        ((r = ht._tr_tally(e, 0, e.window[e.strstart - 1])),
        r && Ge(e, !1),
        e.strstart++,
        e.lookahead--,
        e.strm.avail_out === 0)
      )
        return Qe
    } else ((e.match_available = 1), e.strstart++, e.lookahead--)
  }
  return (
    e.match_available &&
      ((r = ht._tr_tally(e, 0, e.window[e.strstart - 1])), (e.match_available = 0)),
    (e.insert = e.strstart < Ee - 1 ? e.strstart : Ee - 1),
    t === It
      ? (Ge(e, !0), e.strm.avail_out === 0 ? Zt : ur)
      : e.last_lit && (Ge(e, !1), e.strm.avail_out === 0)
        ? Qe
        : Fr
  )
}
function wu(e, t) {
  for (var n, r, i, o, a = e.window; ; ) {
    if (e.lookahead <= Dt) {
      if ((qt(e), e.lookahead <= Dt && t === tr)) return Qe
      if (e.lookahead === 0) break
    }
    if (
      ((e.match_length = 0),
      e.lookahead >= Ee &&
        e.strstart > 0 &&
        ((i = e.strstart - 1), (r = a[i]), r === a[++i] && r === a[++i] && r === a[++i]))
    ) {
      o = e.strstart + Dt
      do;
      while (
        r === a[++i] &&
        r === a[++i] &&
        r === a[++i] &&
        r === a[++i] &&
        r === a[++i] &&
        r === a[++i] &&
        r === a[++i] &&
        r === a[++i] &&
        i < o
      )
      ;((e.match_length = Dt - (o - i)),
        e.match_length > e.lookahead && (e.match_length = e.lookahead))
    }
    if (
      (e.match_length >= Ee
        ? ((n = ht._tr_tally(e, 1, e.match_length - Ee)),
          (e.lookahead -= e.match_length),
          (e.strstart += e.match_length),
          (e.match_length = 0))
        : ((n = ht._tr_tally(e, 0, e.window[e.strstart])), e.lookahead--, e.strstart++),
      n && (Ge(e, !1), e.strm.avail_out === 0))
    )
      return Qe
  }
  return (
    (e.insert = 0),
    t === It
      ? (Ge(e, !0), e.strm.avail_out === 0 ? Zt : ur)
      : e.last_lit && (Ge(e, !1), e.strm.avail_out === 0)
        ? Qe
        : Fr
  )
}
function ku(e, t) {
  for (var n; ; ) {
    if (e.lookahead === 0 && (qt(e), e.lookahead === 0)) {
      if (t === tr) return Qe
      break
    }
    if (
      ((e.match_length = 0),
      (n = ht._tr_tally(e, 0, e.window[e.strstart])),
      e.lookahead--,
      e.strstart++,
      n && (Ge(e, !1), e.strm.avail_out === 0))
    )
      return Qe
  }
  return (
    (e.insert = 0),
    t === It
      ? (Ge(e, !0), e.strm.avail_out === 0 ? Zt : ur)
      : e.last_lit && (Ge(e, !1), e.strm.avail_out === 0)
        ? Qe
        : Fr
  )
}
function St(e, t, n, r, i) {
  ;((this.good_length = e),
    (this.max_lazy = t),
    (this.nice_length = n),
    (this.max_chain = r),
    (this.func = i))
}
var nr
nr = [
  new St(0, 0, 0, 0, mu),
  new St(4, 4, 8, 4, xn),
  new St(4, 5, 16, 8, xn),
  new St(4, 6, 32, 32, xn),
  new St(4, 4, 16, 16, rr),
  new St(8, 16, 32, 32, rr),
  new St(8, 16, 128, 128, rr),
  new St(8, 32, 128, 256, rr),
  new St(32, 128, 258, 1024, rr),
  new St(32, 258, 258, 4096, rr)
]
function Su(e) {
  ;((e.window_size = 2 * e.w_size),
    Mt(e.head),
    (e.max_lazy_match = nr[e.level].max_lazy),
    (e.good_match = nr[e.level].good_length),
    (e.nice_match = nr[e.level].nice_length),
    (e.max_chain_length = nr[e.level].max_chain),
    (e.strstart = 0),
    (e.block_start = 0),
    (e.lookahead = 0),
    (e.insert = 0),
    (e.match_length = e.prev_length = Ee - 1),
    (e.match_available = 0),
    (e.ins_h = 0))
}
function Eu() {
  ;((this.strm = null),
    (this.status = 0),
    (this.pending_buf = null),
    (this.pending_buf_size = 0),
    (this.pending_out = 0),
    (this.pending = 0),
    (this.wrap = 0),
    (this.gzhead = null),
    (this.gzindex = 0),
    (this.method = Yr),
    (this.last_flush = -1),
    (this.w_size = 0),
    (this.w_bits = 0),
    (this.w_mask = 0),
    (this.window = null),
    (this.window_size = 0),
    (this.prev = null),
    (this.head = null),
    (this.ins_h = 0),
    (this.hash_size = 0),
    (this.hash_bits = 0),
    (this.hash_mask = 0),
    (this.hash_shift = 0),
    (this.block_start = 0),
    (this.match_length = 0),
    (this.prev_match = 0),
    (this.match_available = 0),
    (this.strstart = 0),
    (this.match_start = 0),
    (this.lookahead = 0),
    (this.prev_length = 0),
    (this.max_chain_length = 0),
    (this.max_lazy_match = 0),
    (this.level = 0),
    (this.strategy = 0),
    (this.good_match = 0),
    (this.nice_match = 0),
    (this.dyn_ltree = new tt.Buf16(vu * 2)),
    (this.dyn_dtree = new tt.Buf16((2 * _u + 1) * 2)),
    (this.bl_tree = new tt.Buf16((2 * pu + 1) * 2)),
    Mt(this.dyn_ltree),
    Mt(this.dyn_dtree),
    Mt(this.bl_tree),
    (this.l_desc = null),
    (this.d_desc = null),
    (this.bl_desc = null),
    (this.bl_count = new tt.Buf16(yu + 1)),
    (this.heap = new tt.Buf16(2 * Fn + 1)),
    Mt(this.heap),
    (this.heap_len = 0),
    (this.heap_max = 0),
    (this.depth = new tt.Buf16(2 * Fn + 1)),
    Mt(this.depth),
    (this.l_buf = 0),
    (this.lit_bufsize = 0),
    (this.last_lit = 0),
    (this.d_buf = 0),
    (this.opt_len = 0),
    (this.static_len = 0),
    (this.matches = 0),
    (this.insert = 0),
    (this.bi_buf = 0),
    (this.bi_valid = 0))
}
function eo(e) {
  var t
  return !e || !e.state
    ? Ot(e, dt)
    : ((e.total_in = e.total_out = 0),
      (e.data_type = uu),
      (t = e.state),
      (t.pending = 0),
      (t.pending_out = 0),
      t.wrap < 0 && (t.wrap = -t.wrap),
      (t.status = t.wrap ? $r : Gt),
      (e.adler = t.wrap === 2 ? 0 : 1),
      (t.last_flush = tr),
      ht._tr_init(t),
      Xt)
}
function to(e) {
  var t = eo(e)
  return (t === Xt && Su(e.state), t)
}
function Ku(e, t) {
  return !e || !e.state || e.state.wrap !== 2 ? dt : ((e.state.gzhead = t), Xt)
}
function ro(e, t, n, r, i, o) {
  if (!e) return dt
  var a = 1
  if (
    (t === nu && (t = 6),
    r < 0 ? ((a = 0), (r = -r)) : r > 15 && ((a = 2), (r -= 16)),
    i < 1 || i > lu || n !== Yr || r < 8 || r > 15 || t < 0 || t > 9 || o < 0 || o > ou)
  )
    return Ot(e, dt)
  r === 8 && (r = 9)
  var s = new Eu()
  return (
    (e.state = s),
    (s.strm = e),
    (s.wrap = a),
    (s.gzhead = null),
    (s.w_bits = r),
    (s.w_size = 1 << s.w_bits),
    (s.w_mask = s.w_size - 1),
    (s.hash_bits = i + 7),
    (s.hash_size = 1 << s.hash_bits),
    (s.hash_mask = s.hash_size - 1),
    (s.hash_shift = ~~((s.hash_bits + Ee - 1) / Ee)),
    (s.window = new tt.Buf8(s.w_size * 2)),
    (s.head = new tt.Buf16(s.hash_size)),
    (s.prev = new tt.Buf16(s.w_size)),
    (s.lit_bufsize = 1 << (i + 6)),
    (s.pending_buf_size = s.lit_bufsize * 4),
    (s.pending_buf = new tt.Buf8(s.pending_buf_size)),
    (s.d_buf = 1 * s.lit_bufsize),
    (s.l_buf = 3 * s.lit_bufsize),
    (s.level = t),
    (s.strategy = o),
    (s.method = n),
    to(e)
  )
}
function Xu(e, t) {
  return ro(e, t, Yr, fu, cu, su)
}
function Fu(e, t) {
  var n, r, i, o
  if (!e || !e.state || t > Ui || t < 0) return e ? Ot(e, dt) : dt
  if (((r = e.state), !e.output || (!e.input && e.avail_in !== 0) || (r.status === dr && t !== It)))
    return Ot(e, e.avail_out === 0 ? yn : dt)
  if (((r.strm = e), (n = r.last_flush), (r.last_flush = t), r.status === $r))
    if (r.wrap === 2)
      ((e.adler = 0),
        Fe(r, 31),
        Fe(r, 139),
        Fe(r, 8),
        r.gzhead
          ? (Fe(
              r,
              (r.gzhead.text ? 1 : 0) +
                (r.gzhead.hcrc ? 2 : 0) +
                (r.gzhead.extra ? 4 : 0) +
                (r.gzhead.name ? 8 : 0) +
                (r.gzhead.comment ? 16 : 0)
            ),
            Fe(r, r.gzhead.time & 255),
            Fe(r, (r.gzhead.time >> 8) & 255),
            Fe(r, (r.gzhead.time >> 16) & 255),
            Fe(r, (r.gzhead.time >> 24) & 255),
            Fe(r, r.level === 9 ? 2 : r.strategy >= Qr || r.level < 2 ? 4 : 0),
            Fe(r, r.gzhead.os & 255),
            r.gzhead.extra &&
              r.gzhead.extra.length &&
              (Fe(r, r.gzhead.extra.length & 255), Fe(r, (r.gzhead.extra.length >> 8) & 255)),
            r.gzhead.hcrc && (e.adler = (0, Lt.default)(e.adler, r.pending_buf, r.pending, 0)),
            (r.gzindex = 0),
            (r.status = Cn))
          : (Fe(r, 0),
            Fe(r, 0),
            Fe(r, 0),
            Fe(r, 0),
            Fe(r, 0),
            Fe(r, r.level === 9 ? 2 : r.strategy >= Qr || r.level < 2 ? 4 : 0),
            Fe(r, gu),
            (r.status = Gt)))
    else {
      var a = (Yr + ((r.w_bits - 8) << 4)) << 8,
        s = -1
      ;(r.strategy >= Qr || r.level < 2
        ? (s = 0)
        : r.level < 6
          ? (s = 1)
          : r.level === 6
            ? (s = 2)
            : (s = 3),
        (a |= s << 6),
        r.strstart !== 0 && (a |= xu),
        (a += 31 - (a % 31)),
        (r.status = Gt),
        cr(r, a),
        r.strstart !== 0 && (cr(r, e.adler >>> 16), cr(r, e.adler & 65535)),
        (e.adler = 1))
    }
  if (r.status === Cn)
    if (r.gzhead.extra) {
      for (
        i = r.pending;
        r.gzindex < (r.gzhead.extra.length & 65535) &&
        !(
          r.pending === r.pending_buf_size &&
          (r.gzhead.hcrc &&
            r.pending > i &&
            (e.adler = (0, Lt.default)(e.adler, r.pending_buf, r.pending - i, i)),
          Rt(e),
          (i = r.pending),
          r.pending === r.pending_buf_size)
        );

      )
        (Fe(r, r.gzhead.extra[r.gzindex] & 255), r.gzindex++)
      ;(r.gzhead.hcrc &&
        r.pending > i &&
        (e.adler = (0, Lt.default)(e.adler, r.pending_buf, r.pending - i, i)),
        r.gzindex === r.gzhead.extra.length && ((r.gzindex = 0), (r.status = jr)))
    } else r.status = jr
  if (r.status === jr)
    if (r.gzhead.name) {
      i = r.pending
      do {
        if (
          r.pending === r.pending_buf_size &&
          (r.gzhead.hcrc &&
            r.pending > i &&
            (e.adler = (0, Lt.default)(e.adler, r.pending_buf, r.pending - i, i)),
          Rt(e),
          (i = r.pending),
          r.pending === r.pending_buf_size)
        ) {
          o = 1
          break
        }
        ;(r.gzindex < r.gzhead.name.length
          ? (o = r.gzhead.name.charCodeAt(r.gzindex++) & 255)
          : (o = 0),
          Fe(r, o))
      } while (o !== 0)
      ;(r.gzhead.hcrc &&
        r.pending > i &&
        (e.adler = (0, Lt.default)(e.adler, r.pending_buf, r.pending - i, i)),
        o === 0 && ((r.gzindex = 0), (r.status = Hr)))
    } else r.status = Hr
  if (r.status === Hr)
    if (r.gzhead.comment) {
      i = r.pending
      do {
        if (
          r.pending === r.pending_buf_size &&
          (r.gzhead.hcrc &&
            r.pending > i &&
            (e.adler = (0, Lt.default)(e.adler, r.pending_buf, r.pending - i, i)),
          Rt(e),
          (i = r.pending),
          r.pending === r.pending_buf_size)
        ) {
          o = 1
          break
        }
        ;(r.gzindex < r.gzhead.comment.length
          ? (o = r.gzhead.comment.charCodeAt(r.gzindex++) & 255)
          : (o = 0),
          Fe(r, o))
      } while (o !== 0)
      ;(r.gzhead.hcrc &&
        r.pending > i &&
        (e.adler = (0, Lt.default)(e.adler, r.pending_buf, r.pending - i, i)),
        o === 0 && (r.status = Wr))
    } else r.status = Wr
  if (
    (r.status === Wr &&
      (r.gzhead.hcrc
        ? (r.pending + 2 > r.pending_buf_size && Rt(e),
          r.pending + 2 <= r.pending_buf_size &&
            (Fe(r, e.adler & 255), Fe(r, (e.adler >> 8) & 255), (e.adler = 0), (r.status = Gt)))
        : (r.status = Gt)),
    r.pending !== 0)
  ) {
    if ((Rt(e), e.avail_out === 0)) return ((r.last_flush = -1), Xt)
  } else if (e.avail_in === 0 && ji(t) <= ji(n) && t !== It) return Ot(e, yn)
  if (r.status === dr && e.avail_in !== 0) return Ot(e, yn)
  if (e.avail_in !== 0 || r.lookahead !== 0 || (t !== tr && r.status !== dr)) {
    var x = r.strategy === Qr ? ku(r, t) : r.strategy === au ? wu(r, t) : nr[r.level].func(r, t)
    if (((x === Zt || x === ur) && (r.status = dr), x === Qe || x === Zt))
      return (e.avail_out === 0 && (r.last_flush = -1), Xt)
    if (
      x === Fr &&
      (t === eu
        ? ht._tr_align(r)
        : t !== Ui &&
          (ht._tr_stored_block(r, 0, 0, !1),
          t === tu &&
            (Mt(r.head),
            r.lookahead === 0 && ((r.strstart = 0), (r.block_start = 0), (r.insert = 0)))),
      Rt(e),
      e.avail_out === 0)
    )
      return ((r.last_flush = -1), Xt)
  }
  return t !== It
    ? Xt
    : r.wrap <= 0
      ? Ni
      : (r.wrap === 2
          ? (Fe(r, e.adler & 255),
            Fe(r, (e.adler >> 8) & 255),
            Fe(r, (e.adler >> 16) & 255),
            Fe(r, (e.adler >> 24) & 255),
            Fe(r, e.total_in & 255),
            Fe(r, (e.total_in >> 8) & 255),
            Fe(r, (e.total_in >> 16) & 255),
            Fe(r, (e.total_in >> 24) & 255))
          : (cr(r, e.adler >>> 16), cr(r, e.adler & 65535)),
        Rt(e),
        r.wrap > 0 && (r.wrap = -r.wrap),
        r.pending !== 0 ? Xt : Ni)
}
function Cu(e) {
  var t
  return !e || !e.state
    ? dt
    : ((t = e.state.status),
      t !== $r && t !== Cn && t !== jr && t !== Hr && t !== Wr && t !== Gt && t !== dr
        ? Ot(e, dt)
        : ((e.state = null), t === Gt ? Ot(e, ru) : Xt))
}
function Au(e, t) {
  var n = t.length,
    r,
    i,
    o,
    a,
    s,
    x,
    u,
    f
  if (
    !e ||
    !e.state ||
    ((r = e.state), (a = r.wrap), a === 2 || (a === 1 && r.status !== $r) || r.lookahead)
  )
    return dt
  for (
    a === 1 && (e.adler = (0, qa.default)(e.adler, t, n, 0)),
      r.wrap = 0,
      n >= r.w_size &&
        (a === 0 && (Mt(r.head), (r.strstart = 0), (r.block_start = 0), (r.insert = 0)),
        (f = new tt.Buf8(r.w_size)),
        tt.arraySet(f, t, n - r.w_size, r.w_size, 0),
        (t = f),
        (n = r.w_size)),
      s = e.avail_in,
      x = e.next_in,
      u = e.input,
      e.avail_in = n,
      e.next_in = 0,
      e.input = t,
      qt(r);
    r.lookahead >= Ee;

  ) {
    ;((i = r.strstart), (o = r.lookahead - (Ee - 1)))
    do
      ((r.ins_h = ((r.ins_h << r.hash_shift) ^ r.window[i + Ee - 1]) & r.hash_mask),
        (r.prev[i & r.w_mask] = r.head[r.ins_h]),
        (r.head[r.ins_h] = i),
        i++)
    while (--o)
    ;((r.strstart = i), (r.lookahead = Ee - 1), qt(r))
  }
  return (
    (r.strstart += r.lookahead),
    (r.block_start = r.strstart),
    (r.insert = r.lookahead),
    (r.lookahead = 0),
    (r.match_length = r.prev_length = Ee - 1),
    (r.match_available = 0),
    (e.next_in = x),
    (e.input = u),
    (e.avail_in = s),
    (r.wrap = a),
    Xt
  )
}
ge.deflateInfo = 'pako deflate (from Nodeca project)'
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = ge,
    n = r(Bn)
  function r(f) {
    return f && f.__esModule ? f : { default: f }
  }
  function i(f) {
    '@babel/helpers - typeof'
    return (
      (i =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (F) {
              return typeof F
            }
          : function (F) {
              return F &&
                typeof Symbol == 'function' &&
                F.constructor === Symbol &&
                F !== Symbol.prototype
                ? 'symbol'
                : typeof F
            }),
      i(f)
    )
  }
  function o(f, F) {
    if (!(f instanceof F)) throw new TypeError('Cannot call a class as a function')
  }
  function a(f, F) {
    for (var X = 0; X < F.length; X++) {
      var g = F[X]
      ;((g.enumerable = g.enumerable || !1),
        (g.configurable = !0),
        'value' in g && (g.writable = !0),
        Object.defineProperty(f, x(g.key), g))
    }
  }
  function s(f, F, X) {
    return (F && a(f.prototype, F), Object.defineProperty(f, 'prototype', { writable: !1 }), f)
  }
  function x(f) {
    var F = u(f, 'string')
    return i(F) == 'symbol' ? F : F + ''
  }
  function u(f, F) {
    if (i(f) != 'object' || !f) return f
    var X = f[Symbol.toPrimitive]
    if (X !== void 0) {
      var g = X.call(f, F)
      if (i(g) != 'object') return g
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(f)
  }
  e.default = (function () {
    function f() {
      ;(o(this, f),
        (this.strm = new n.default()),
        (this.chunkSize = 1024 * 10 * 10),
        (this.outputBuffer = new Uint8Array(this.chunkSize)),
        (0, t.deflateInit)(this.strm, t.Z_DEFAULT_COMPRESSION))
    }
    return s(f, [
      {
        key: 'deflate',
        value: function (X) {
          ;((this.strm.input = X),
            (this.strm.avail_in = this.strm.input.length),
            (this.strm.next_in = 0),
            (this.strm.output = this.outputBuffer),
            (this.strm.avail_out = this.chunkSize),
            (this.strm.next_out = 0))
          var g = (0, t.deflate)(this.strm, t.Z_FULL_FLUSH),
            k = new Uint8Array(this.strm.output.buffer, 0, this.strm.next_out)
          if (g < 0) throw new Error('zlib deflate failed')
          if (this.strm.avail_in > 0) {
            var _ = [k],
              l = k.length
            do {
              if (
                ((this.strm.output = new Uint8Array(this.chunkSize)),
                (this.strm.next_out = 0),
                (this.strm.avail_out = this.chunkSize),
                (g = (0, t.deflate)(this.strm, t.Z_FULL_FLUSH)),
                g < 0)
              )
                throw new Error('zlib deflate failed')
              var h = new Uint8Array(this.strm.output.buffer, 0, this.strm.next_out)
              ;((l += h.length), _.push(h))
            } while (this.strm.avail_in > 0)
            for (var c = new Uint8Array(l), E = 0, y = 0; y < _.length; y++)
              (c.set(_[y], E), (E += _[y].length))
            k = c
          }
          return ((this.strm.input = null), (this.strm.avail_in = 0), (this.strm.next_in = 0), k)
        }
      }
    ])
  })()
})(Ta)
var no = {},
  Cr = {},
  Ar = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }),
    (e.default = void 0),
    (e.default = {
      XK_VoidSymbol: 16777215,
      XK_BackSpace: 65288,
      XK_Tab: 65289,
      XK_Linefeed: 65290,
      XK_Clear: 65291,
      XK_Return: 65293,
      XK_Pause: 65299,
      XK_Scroll_Lock: 65300,
      XK_Sys_Req: 65301,
      XK_Escape: 65307,
      XK_Delete: 65535,
      XK_Multi_key: 65312,
      XK_Codeinput: 65335,
      XK_SingleCandidate: 65340,
      XK_MultipleCandidate: 65341,
      XK_PreviousCandidate: 65342,
      XK_Kanji: 65313,
      XK_Muhenkan: 65314,
      XK_Henkan_Mode: 65315,
      XK_Henkan: 65315,
      XK_Romaji: 65316,
      XK_Hiragana: 65317,
      XK_Katakana: 65318,
      XK_Hiragana_Katakana: 65319,
      XK_Zenkaku: 65320,
      XK_Hankaku: 65321,
      XK_Zenkaku_Hankaku: 65322,
      XK_Touroku: 65323,
      XK_Massyo: 65324,
      XK_Kana_Lock: 65325,
      XK_Kana_Shift: 65326,
      XK_Eisu_Shift: 65327,
      XK_Eisu_toggle: 65328,
      XK_Kanji_Bangou: 65335,
      XK_Zen_Koho: 65341,
      XK_Mae_Koho: 65342,
      XK_Home: 65360,
      XK_Left: 65361,
      XK_Up: 65362,
      XK_Right: 65363,
      XK_Down: 65364,
      XK_Prior: 65365,
      XK_Page_Up: 65365,
      XK_Next: 65366,
      XK_Page_Down: 65366,
      XK_End: 65367,
      XK_Begin: 65368,
      XK_Select: 65376,
      XK_Print: 65377,
      XK_Execute: 65378,
      XK_Insert: 65379,
      XK_Undo: 65381,
      XK_Redo: 65382,
      XK_Menu: 65383,
      XK_Find: 65384,
      XK_Cancel: 65385,
      XK_Help: 65386,
      XK_Break: 65387,
      XK_Mode_switch: 65406,
      XK_script_switch: 65406,
      XK_Num_Lock: 65407,
      XK_KP_Space: 65408,
      XK_KP_Tab: 65417,
      XK_KP_Enter: 65421,
      XK_KP_F1: 65425,
      XK_KP_F2: 65426,
      XK_KP_F3: 65427,
      XK_KP_F4: 65428,
      XK_KP_Home: 65429,
      XK_KP_Left: 65430,
      XK_KP_Up: 65431,
      XK_KP_Right: 65432,
      XK_KP_Down: 65433,
      XK_KP_Prior: 65434,
      XK_KP_Page_Up: 65434,
      XK_KP_Next: 65435,
      XK_KP_Page_Down: 65435,
      XK_KP_End: 65436,
      XK_KP_Begin: 65437,
      XK_KP_Insert: 65438,
      XK_KP_Delete: 65439,
      XK_KP_Equal: 65469,
      XK_KP_Multiply: 65450,
      XK_KP_Add: 65451,
      XK_KP_Separator: 65452,
      XK_KP_Subtract: 65453,
      XK_KP_Decimal: 65454,
      XK_KP_Divide: 65455,
      XK_KP_0: 65456,
      XK_KP_1: 65457,
      XK_KP_2: 65458,
      XK_KP_3: 65459,
      XK_KP_4: 65460,
      XK_KP_5: 65461,
      XK_KP_6: 65462,
      XK_KP_7: 65463,
      XK_KP_8: 65464,
      XK_KP_9: 65465,
      XK_F1: 65470,
      XK_F2: 65471,
      XK_F3: 65472,
      XK_F4: 65473,
      XK_F5: 65474,
      XK_F6: 65475,
      XK_F7: 65476,
      XK_F8: 65477,
      XK_F9: 65478,
      XK_F10: 65479,
      XK_F11: 65480,
      XK_L1: 65480,
      XK_F12: 65481,
      XK_L2: 65481,
      XK_F13: 65482,
      XK_L3: 65482,
      XK_F14: 65483,
      XK_L4: 65483,
      XK_F15: 65484,
      XK_L5: 65484,
      XK_F16: 65485,
      XK_L6: 65485,
      XK_F17: 65486,
      XK_L7: 65486,
      XK_F18: 65487,
      XK_L8: 65487,
      XK_F19: 65488,
      XK_L9: 65488,
      XK_F20: 65489,
      XK_L10: 65489,
      XK_F21: 65490,
      XK_R1: 65490,
      XK_F22: 65491,
      XK_R2: 65491,
      XK_F23: 65492,
      XK_R3: 65492,
      XK_F24: 65493,
      XK_R4: 65493,
      XK_F25: 65494,
      XK_R5: 65494,
      XK_F26: 65495,
      XK_R6: 65495,
      XK_F27: 65496,
      XK_R7: 65496,
      XK_F28: 65497,
      XK_R8: 65497,
      XK_F29: 65498,
      XK_R9: 65498,
      XK_F30: 65499,
      XK_R10: 65499,
      XK_F31: 65500,
      XK_R11: 65500,
      XK_F32: 65501,
      XK_R12: 65501,
      XK_F33: 65502,
      XK_R13: 65502,
      XK_F34: 65503,
      XK_R14: 65503,
      XK_F35: 65504,
      XK_R15: 65504,
      XK_Shift_L: 65505,
      XK_Shift_R: 65506,
      XK_Control_L: 65507,
      XK_Control_R: 65508,
      XK_Caps_Lock: 65509,
      XK_Shift_Lock: 65510,
      XK_Meta_L: 65511,
      XK_Meta_R: 65512,
      XK_Alt_L: 65513,
      XK_Alt_R: 65514,
      XK_Super_L: 65515,
      XK_Super_R: 65516,
      XK_Hyper_L: 65517,
      XK_Hyper_R: 65518,
      XK_ISO_Level3_Shift: 65027,
      XK_ISO_Next_Group: 65032,
      XK_ISO_Prev_Group: 65034,
      XK_ISO_First_Group: 65036,
      XK_ISO_Last_Group: 65038,
      XK_space: 32,
      XK_exclam: 33,
      XK_quotedbl: 34,
      XK_numbersign: 35,
      XK_dollar: 36,
      XK_percent: 37,
      XK_ampersand: 38,
      XK_apostrophe: 39,
      XK_quoteright: 39,
      XK_parenleft: 40,
      XK_parenright: 41,
      XK_asterisk: 42,
      XK_plus: 43,
      XK_comma: 44,
      XK_minus: 45,
      XK_period: 46,
      XK_slash: 47,
      XK_0: 48,
      XK_1: 49,
      XK_2: 50,
      XK_3: 51,
      XK_4: 52,
      XK_5: 53,
      XK_6: 54,
      XK_7: 55,
      XK_8: 56,
      XK_9: 57,
      XK_colon: 58,
      XK_semicolon: 59,
      XK_less: 60,
      XK_equal: 61,
      XK_greater: 62,
      XK_question: 63,
      XK_at: 64,
      XK_A: 65,
      XK_B: 66,
      XK_C: 67,
      XK_D: 68,
      XK_E: 69,
      XK_F: 70,
      XK_G: 71,
      XK_H: 72,
      XK_I: 73,
      XK_J: 74,
      XK_K: 75,
      XK_L: 76,
      XK_M: 77,
      XK_N: 78,
      XK_O: 79,
      XK_P: 80,
      XK_Q: 81,
      XK_R: 82,
      XK_S: 83,
      XK_T: 84,
      XK_U: 85,
      XK_V: 86,
      XK_W: 87,
      XK_X: 88,
      XK_Y: 89,
      XK_Z: 90,
      XK_bracketleft: 91,
      XK_backslash: 92,
      XK_bracketright: 93,
      XK_asciicircum: 94,
      XK_underscore: 95,
      XK_grave: 96,
      XK_quoteleft: 96,
      XK_a: 97,
      XK_b: 98,
      XK_c: 99,
      XK_d: 100,
      XK_e: 101,
      XK_f: 102,
      XK_g: 103,
      XK_h: 104,
      XK_i: 105,
      XK_j: 106,
      XK_k: 107,
      XK_l: 108,
      XK_m: 109,
      XK_n: 110,
      XK_o: 111,
      XK_p: 112,
      XK_q: 113,
      XK_r: 114,
      XK_s: 115,
      XK_t: 116,
      XK_u: 117,
      XK_v: 118,
      XK_w: 119,
      XK_x: 120,
      XK_y: 121,
      XK_z: 122,
      XK_braceleft: 123,
      XK_bar: 124,
      XK_braceright: 125,
      XK_asciitilde: 126,
      XK_nobreakspace: 160,
      XK_exclamdown: 161,
      XK_cent: 162,
      XK_sterling: 163,
      XK_currency: 164,
      XK_yen: 165,
      XK_brokenbar: 166,
      XK_section: 167,
      XK_diaeresis: 168,
      XK_copyright: 169,
      XK_ordfeminine: 170,
      XK_guillemotleft: 171,
      XK_notsign: 172,
      XK_hyphen: 173,
      XK_registered: 174,
      XK_macron: 175,
      XK_degree: 176,
      XK_plusminus: 177,
      XK_twosuperior: 178,
      XK_threesuperior: 179,
      XK_acute: 180,
      XK_mu: 181,
      XK_paragraph: 182,
      XK_periodcentered: 183,
      XK_cedilla: 184,
      XK_onesuperior: 185,
      XK_masculine: 186,
      XK_guillemotright: 187,
      XK_onequarter: 188,
      XK_onehalf: 189,
      XK_threequarters: 190,
      XK_questiondown: 191,
      XK_Agrave: 192,
      XK_Aacute: 193,
      XK_Acircumflex: 194,
      XK_Atilde: 195,
      XK_Adiaeresis: 196,
      XK_Aring: 197,
      XK_AE: 198,
      XK_Ccedilla: 199,
      XK_Egrave: 200,
      XK_Eacute: 201,
      XK_Ecircumflex: 202,
      XK_Ediaeresis: 203,
      XK_Igrave: 204,
      XK_Iacute: 205,
      XK_Icircumflex: 206,
      XK_Idiaeresis: 207,
      XK_ETH: 208,
      XK_Eth: 208,
      XK_Ntilde: 209,
      XK_Ograve: 210,
      XK_Oacute: 211,
      XK_Ocircumflex: 212,
      XK_Otilde: 213,
      XK_Odiaeresis: 214,
      XK_multiply: 215,
      XK_Oslash: 216,
      XK_Ooblique: 216,
      XK_Ugrave: 217,
      XK_Uacute: 218,
      XK_Ucircumflex: 219,
      XK_Udiaeresis: 220,
      XK_Yacute: 221,
      XK_THORN: 222,
      XK_Thorn: 222,
      XK_ssharp: 223,
      XK_agrave: 224,
      XK_aacute: 225,
      XK_acircumflex: 226,
      XK_atilde: 227,
      XK_adiaeresis: 228,
      XK_aring: 229,
      XK_ae: 230,
      XK_ccedilla: 231,
      XK_egrave: 232,
      XK_eacute: 233,
      XK_ecircumflex: 234,
      XK_ediaeresis: 235,
      XK_igrave: 236,
      XK_iacute: 237,
      XK_icircumflex: 238,
      XK_idiaeresis: 239,
      XK_eth: 240,
      XK_ntilde: 241,
      XK_ograve: 242,
      XK_oacute: 243,
      XK_ocircumflex: 244,
      XK_otilde: 245,
      XK_odiaeresis: 246,
      XK_division: 247,
      XK_oslash: 248,
      XK_ooblique: 248,
      XK_ugrave: 249,
      XK_uacute: 250,
      XK_ucircumflex: 251,
      XK_udiaeresis: 252,
      XK_yacute: 253,
      XK_thorn: 254,
      XK_ydiaeresis: 255,
      XK_Hangul: 65329,
      XK_Hangul_Hanja: 65332,
      XK_Hangul_Jeonja: 65336,
      XF86XK_ModeLock: 269025025,
      XF86XK_MonBrightnessUp: 269025026,
      XF86XK_MonBrightnessDown: 269025027,
      XF86XK_KbdLightOnOff: 269025028,
      XF86XK_KbdBrightnessUp: 269025029,
      XF86XK_KbdBrightnessDown: 269025030,
      XF86XK_Standby: 269025040,
      XF86XK_AudioLowerVolume: 269025041,
      XF86XK_AudioMute: 269025042,
      XF86XK_AudioRaiseVolume: 269025043,
      XF86XK_AudioPlay: 269025044,
      XF86XK_AudioStop: 269025045,
      XF86XK_AudioPrev: 269025046,
      XF86XK_AudioNext: 269025047,
      XF86XK_HomePage: 269025048,
      XF86XK_Mail: 269025049,
      XF86XK_Start: 269025050,
      XF86XK_Search: 269025051,
      XF86XK_AudioRecord: 269025052,
      XF86XK_Calculator: 269025053,
      XF86XK_Memo: 269025054,
      XF86XK_ToDoList: 269025055,
      XF86XK_Calendar: 269025056,
      XF86XK_PowerDown: 269025057,
      XF86XK_ContrastAdjust: 269025058,
      XF86XK_RockerUp: 269025059,
      XF86XK_RockerDown: 269025060,
      XF86XK_RockerEnter: 269025061,
      XF86XK_Back: 269025062,
      XF86XK_Forward: 269025063,
      XF86XK_Stop: 269025064,
      XF86XK_Refresh: 269025065,
      XF86XK_PowerOff: 269025066,
      XF86XK_WakeUp: 269025067,
      XF86XK_Eject: 269025068,
      XF86XK_ScreenSaver: 269025069,
      XF86XK_WWW: 269025070,
      XF86XK_Sleep: 269025071,
      XF86XK_Favorites: 269025072,
      XF86XK_AudioPause: 269025073,
      XF86XK_AudioMedia: 269025074,
      XF86XK_MyComputer: 269025075,
      XF86XK_VendorHome: 269025076,
      XF86XK_LightBulb: 269025077,
      XF86XK_Shop: 269025078,
      XF86XK_History: 269025079,
      XF86XK_OpenURL: 269025080,
      XF86XK_AddFavorite: 269025081,
      XF86XK_HotLinks: 269025082,
      XF86XK_BrightnessAdjust: 269025083,
      XF86XK_Finance: 269025084,
      XF86XK_Community: 269025085,
      XF86XK_AudioRewind: 269025086,
      XF86XK_BackForward: 269025087,
      XF86XK_Launch0: 269025088,
      XF86XK_Launch1: 269025089,
      XF86XK_Launch2: 269025090,
      XF86XK_Launch3: 269025091,
      XF86XK_Launch4: 269025092,
      XF86XK_Launch5: 269025093,
      XF86XK_Launch6: 269025094,
      XF86XK_Launch7: 269025095,
      XF86XK_Launch8: 269025096,
      XF86XK_Launch9: 269025097,
      XF86XK_LaunchA: 269025098,
      XF86XK_LaunchB: 269025099,
      XF86XK_LaunchC: 269025100,
      XF86XK_LaunchD: 269025101,
      XF86XK_LaunchE: 269025102,
      XF86XK_LaunchF: 269025103,
      XF86XK_ApplicationLeft: 269025104,
      XF86XK_ApplicationRight: 269025105,
      XF86XK_Book: 269025106,
      XF86XK_CD: 269025107,
      XF86XK_Calculater: 269025108,
      XF86XK_Clear: 269025109,
      XF86XK_Close: 269025110,
      XF86XK_Copy: 269025111,
      XF86XK_Cut: 269025112,
      XF86XK_Display: 269025113,
      XF86XK_DOS: 269025114,
      XF86XK_Documents: 269025115,
      XF86XK_Excel: 269025116,
      XF86XK_Explorer: 269025117,
      XF86XK_Game: 269025118,
      XF86XK_Go: 269025119,
      XF86XK_iTouch: 269025120,
      XF86XK_LogOff: 269025121,
      XF86XK_Market: 269025122,
      XF86XK_Meeting: 269025123,
      XF86XK_MenuKB: 269025125,
      XF86XK_MenuPB: 269025126,
      XF86XK_MySites: 269025127,
      XF86XK_New: 269025128,
      XF86XK_News: 269025129,
      XF86XK_OfficeHome: 269025130,
      XF86XK_Open: 269025131,
      XF86XK_Option: 269025132,
      XF86XK_Paste: 269025133,
      XF86XK_Phone: 269025134,
      XF86XK_Q: 269025136,
      XF86XK_Reply: 269025138,
      XF86XK_Reload: 269025139,
      XF86XK_RotateWindows: 269025140,
      XF86XK_RotationPB: 269025141,
      XF86XK_RotationKB: 269025142,
      XF86XK_Save: 269025143,
      XF86XK_ScrollUp: 269025144,
      XF86XK_ScrollDown: 269025145,
      XF86XK_ScrollClick: 269025146,
      XF86XK_Send: 269025147,
      XF86XK_Spell: 269025148,
      XF86XK_SplitScreen: 269025149,
      XF86XK_Support: 269025150,
      XF86XK_TaskPane: 269025151,
      XF86XK_Terminal: 269025152,
      XF86XK_Tools: 269025153,
      XF86XK_Travel: 269025154,
      XF86XK_UserPB: 269025156,
      XF86XK_User1KB: 269025157,
      XF86XK_User2KB: 269025158,
      XF86XK_Video: 269025159,
      XF86XK_WheelButton: 269025160,
      XF86XK_Word: 269025161,
      XF86XK_Xfer: 269025162,
      XF86XK_ZoomIn: 269025163,
      XF86XK_ZoomOut: 269025164,
      XF86XK_Away: 269025165,
      XF86XK_Messenger: 269025166,
      XF86XK_WebCam: 269025167,
      XF86XK_MailForward: 269025168,
      XF86XK_Pictures: 269025169,
      XF86XK_Music: 269025170,
      XF86XK_Battery: 269025171,
      XF86XK_Bluetooth: 269025172,
      XF86XK_WLAN: 269025173,
      XF86XK_UWB: 269025174,
      XF86XK_AudioForward: 269025175,
      XF86XK_AudioRepeat: 269025176,
      XF86XK_AudioRandomPlay: 269025177,
      XF86XK_Subtitle: 269025178,
      XF86XK_AudioCycleTrack: 269025179,
      XF86XK_CycleAngle: 269025180,
      XF86XK_FrameBack: 269025181,
      XF86XK_FrameForward: 269025182,
      XF86XK_Time: 269025183,
      XF86XK_Select: 269025184,
      XF86XK_View: 269025185,
      XF86XK_TopMenu: 269025186,
      XF86XK_Red: 269025187,
      XF86XK_Green: 269025188,
      XF86XK_Yellow: 269025189,
      XF86XK_Blue: 269025190,
      XF86XK_Suspend: 269025191,
      XF86XK_Hibernate: 269025192,
      XF86XK_TouchpadToggle: 269025193,
      XF86XK_TouchpadOn: 269025200,
      XF86XK_TouchpadOff: 269025201,
      XF86XK_AudioMicMute: 269025202,
      XF86XK_Switch_VT_1: 269024769,
      XF86XK_Switch_VT_2: 269024770,
      XF86XK_Switch_VT_3: 269024771,
      XF86XK_Switch_VT_4: 269024772,
      XF86XK_Switch_VT_5: 269024773,
      XF86XK_Switch_VT_6: 269024774,
      XF86XK_Switch_VT_7: 269024775,
      XF86XK_Switch_VT_8: 269024776,
      XF86XK_Switch_VT_9: 269024777,
      XF86XK_Switch_VT_10: 269024778,
      XF86XK_Switch_VT_11: 269024779,
      XF86XK_Switch_VT_12: 269024780,
      XF86XK_Ungrab: 269024800,
      XF86XK_ClearGrab: 269024801,
      XF86XK_Next_VMode: 269024802,
      XF86XK_Prev_VMode: 269024803,
      XF86XK_LogWindowTree: 269024804,
      XF86XK_LogGrabInfo: 269024805
    }))
})(Ar)
var io = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = {
    256: 960,
    257: 992,
    258: 451,
    259: 483,
    260: 417,
    261: 433,
    262: 454,
    263: 486,
    264: 710,
    265: 742,
    266: 709,
    267: 741,
    268: 456,
    269: 488,
    270: 463,
    271: 495,
    272: 464,
    273: 496,
    274: 938,
    275: 954,
    278: 972,
    279: 1004,
    280: 458,
    281: 490,
    282: 460,
    283: 492,
    284: 728,
    285: 760,
    286: 683,
    287: 699,
    288: 725,
    289: 757,
    290: 939,
    291: 955,
    292: 678,
    293: 694,
    294: 673,
    295: 689,
    296: 933,
    297: 949,
    298: 975,
    299: 1007,
    302: 967,
    303: 999,
    304: 681,
    305: 697,
    308: 684,
    309: 700,
    310: 979,
    311: 1011,
    312: 930,
    313: 453,
    314: 485,
    315: 934,
    316: 950,
    317: 421,
    318: 437,
    321: 419,
    322: 435,
    323: 465,
    324: 497,
    325: 977,
    326: 1009,
    327: 466,
    328: 498,
    330: 957,
    331: 959,
    332: 978,
    333: 1010,
    336: 469,
    337: 501,
    338: 5052,
    339: 5053,
    340: 448,
    341: 480,
    342: 931,
    343: 947,
    344: 472,
    345: 504,
    346: 422,
    347: 438,
    348: 734,
    349: 766,
    350: 426,
    351: 442,
    352: 425,
    353: 441,
    354: 478,
    355: 510,
    356: 427,
    357: 443,
    358: 940,
    359: 956,
    360: 989,
    361: 1021,
    362: 990,
    363: 1022,
    364: 733,
    365: 765,
    366: 473,
    367: 505,
    368: 475,
    369: 507,
    370: 985,
    371: 1017,
    376: 5054,
    377: 428,
    378: 444,
    379: 431,
    380: 447,
    381: 430,
    382: 446,
    402: 2294,
    466: 16777681,
    711: 439,
    728: 418,
    729: 511,
    731: 434,
    733: 445,
    901: 1966,
    902: 1953,
    904: 1954,
    905: 1955,
    906: 1956,
    908: 1959,
    910: 1960,
    911: 1963,
    912: 1974,
    913: 1985,
    914: 1986,
    915: 1987,
    916: 1988,
    917: 1989,
    918: 1990,
    919: 1991,
    920: 1992,
    921: 1993,
    922: 1994,
    923: 1995,
    924: 1996,
    925: 1997,
    926: 1998,
    927: 1999,
    928: 2e3,
    929: 2001,
    931: 2002,
    932: 2004,
    933: 2005,
    934: 2006,
    935: 2007,
    936: 2008,
    937: 2009,
    938: 1957,
    939: 1961,
    940: 1969,
    941: 1970,
    942: 1971,
    943: 1972,
    944: 1978,
    945: 2017,
    946: 2018,
    947: 2019,
    948: 2020,
    949: 2021,
    950: 2022,
    951: 2023,
    952: 2024,
    953: 2025,
    954: 2026,
    955: 2027,
    956: 2028,
    957: 2029,
    958: 2030,
    959: 2031,
    960: 2032,
    961: 2033,
    962: 2035,
    963: 2034,
    964: 2036,
    965: 2037,
    966: 2038,
    967: 2039,
    968: 2040,
    969: 2041,
    970: 1973,
    971: 1977,
    972: 1975,
    973: 1976,
    974: 1979,
    1025: 1715,
    1026: 1713,
    1027: 1714,
    1028: 1716,
    1029: 1717,
    1030: 1718,
    1031: 1719,
    1032: 1720,
    1033: 1721,
    1034: 1722,
    1035: 1723,
    1036: 1724,
    1038: 1726,
    1039: 1727,
    1040: 1761,
    1041: 1762,
    1042: 1783,
    1043: 1767,
    1044: 1764,
    1045: 1765,
    1046: 1782,
    1047: 1786,
    1048: 1769,
    1049: 1770,
    1050: 1771,
    1051: 1772,
    1052: 1773,
    1053: 1774,
    1054: 1775,
    1055: 1776,
    1056: 1778,
    1057: 1779,
    1058: 1780,
    1059: 1781,
    1060: 1766,
    1061: 1768,
    1062: 1763,
    1063: 1790,
    1064: 1787,
    1065: 1789,
    1066: 1791,
    1067: 1785,
    1068: 1784,
    1069: 1788,
    1070: 1760,
    1071: 1777,
    1072: 1729,
    1073: 1730,
    1074: 1751,
    1075: 1735,
    1076: 1732,
    1077: 1733,
    1078: 1750,
    1079: 1754,
    1080: 1737,
    1081: 1738,
    1082: 1739,
    1083: 1740,
    1084: 1741,
    1085: 1742,
    1086: 1743,
    1087: 1744,
    1088: 1746,
    1089: 1747,
    1090: 1748,
    1091: 1749,
    1092: 1734,
    1093: 1736,
    1094: 1731,
    1095: 1758,
    1096: 1755,
    1097: 1757,
    1098: 1759,
    1099: 1753,
    1100: 1752,
    1101: 1756,
    1102: 1728,
    1103: 1745,
    1105: 1699,
    1106: 1697,
    1107: 1698,
    1108: 1700,
    1109: 1701,
    1110: 1702,
    1111: 1703,
    1112: 1704,
    1113: 1705,
    1114: 1706,
    1115: 1707,
    1116: 1708,
    1118: 1710,
    1119: 1711,
    1168: 1725,
    1169: 1709,
    1488: 3296,
    1489: 3297,
    1490: 3298,
    1491: 3299,
    1492: 3300,
    1493: 3301,
    1494: 3302,
    1495: 3303,
    1496: 3304,
    1497: 3305,
    1498: 3306,
    1499: 3307,
    1500: 3308,
    1501: 3309,
    1502: 3310,
    1503: 3311,
    1504: 3312,
    1505: 3313,
    1506: 3314,
    1507: 3315,
    1508: 3316,
    1509: 3317,
    1510: 3318,
    1511: 3319,
    1512: 3320,
    1513: 3321,
    1514: 3322,
    1548: 1452,
    1563: 1467,
    1567: 1471,
    1569: 1473,
    1570: 1474,
    1571: 1475,
    1572: 1476,
    1573: 1477,
    1574: 1478,
    1575: 1479,
    1576: 1480,
    1577: 1481,
    1578: 1482,
    1579: 1483,
    1580: 1484,
    1581: 1485,
    1582: 1486,
    1583: 1487,
    1584: 1488,
    1585: 1489,
    1586: 1490,
    1587: 1491,
    1588: 1492,
    1589: 1493,
    1590: 1494,
    1591: 1495,
    1592: 1496,
    1593: 1497,
    1594: 1498,
    1600: 1504,
    1601: 1505,
    1602: 1506,
    1603: 1507,
    1604: 1508,
    1605: 1509,
    1606: 1510,
    1607: 1511,
    1608: 1512,
    1609: 1513,
    1610: 1514,
    1611: 1515,
    1612: 1516,
    1613: 1517,
    1614: 1518,
    1615: 1519,
    1616: 1520,
    1617: 1521,
    1618: 1522,
    3585: 3489,
    3586: 3490,
    3587: 3491,
    3588: 3492,
    3589: 3493,
    3590: 3494,
    3591: 3495,
    3592: 3496,
    3593: 3497,
    3594: 3498,
    3595: 3499,
    3596: 3500,
    3597: 3501,
    3598: 3502,
    3599: 3503,
    3600: 3504,
    3601: 3505,
    3602: 3506,
    3603: 3507,
    3604: 3508,
    3605: 3509,
    3606: 3510,
    3607: 3511,
    3608: 3512,
    3609: 3513,
    3610: 3514,
    3611: 3515,
    3612: 3516,
    3613: 3517,
    3614: 3518,
    3615: 3519,
    3616: 3520,
    3617: 3521,
    3618: 3522,
    3619: 3523,
    3620: 3524,
    3621: 3525,
    3622: 3526,
    3623: 3527,
    3624: 3528,
    3625: 3529,
    3626: 3530,
    3627: 3531,
    3628: 3532,
    3629: 3533,
    3630: 3534,
    3631: 3535,
    3632: 3536,
    3633: 3537,
    3634: 3538,
    3635: 3539,
    3636: 3540,
    3637: 3541,
    3638: 3542,
    3639: 3543,
    3640: 3544,
    3641: 3545,
    3642: 3546,
    3647: 3551,
    3648: 3552,
    3649: 3553,
    3650: 3554,
    3651: 3555,
    3652: 3556,
    3653: 3557,
    3654: 3558,
    3655: 3559,
    3656: 3560,
    3657: 3561,
    3658: 3562,
    3659: 3563,
    3660: 3564,
    3661: 3565,
    3664: 3568,
    3665: 3569,
    3666: 3570,
    3667: 3571,
    3668: 3572,
    3669: 3573,
    3670: 3574,
    3671: 3575,
    3672: 3576,
    3673: 3577,
    8194: 2722,
    8195: 2721,
    8196: 2723,
    8197: 2724,
    8199: 2725,
    8200: 2726,
    8201: 2727,
    8202: 2728,
    8210: 2747,
    8211: 2730,
    8212: 2729,
    8213: 1967,
    8215: 3295,
    8216: 2768,
    8217: 2769,
    8218: 2813,
    8220: 2770,
    8221: 2771,
    8222: 2814,
    8224: 2801,
    8225: 2802,
    8226: 2790,
    8229: 2735,
    8230: 2734,
    8240: 2773,
    8242: 2774,
    8243: 2775,
    8248: 2812,
    8254: 1150,
    8361: 3839,
    8364: 8364,
    8453: 2744,
    8470: 1712,
    8471: 2811,
    8478: 2772,
    8482: 2761,
    8531: 2736,
    8532: 2737,
    8533: 2738,
    8534: 2739,
    8535: 2740,
    8536: 2741,
    8537: 2742,
    8538: 2743,
    8539: 2755,
    8540: 2756,
    8541: 2757,
    8542: 2758,
    8592: 2299,
    8593: 2300,
    8594: 2301,
    8595: 2302,
    8658: 2254,
    8660: 2253,
    8706: 2287,
    8711: 2245,
    8728: 3018,
    8730: 2262,
    8733: 2241,
    8734: 2242,
    8743: 2270,
    8744: 2271,
    8745: 2268,
    8746: 2269,
    8747: 2239,
    8756: 2240,
    8764: 2248,
    8771: 2249,
    8773: 16785992,
    8800: 2237,
    8801: 2255,
    8804: 2236,
    8805: 2238,
    8834: 2266,
    8835: 2267,
    8866: 3068,
    8867: 3036,
    8868: 3010,
    8869: 3022,
    8968: 3027,
    8970: 3012,
    8981: 2810,
    8992: 2212,
    8993: 2213,
    9109: 3020,
    9115: 2219,
    9117: 2220,
    9118: 2221,
    9120: 2222,
    9121: 2215,
    9123: 2216,
    9124: 2217,
    9126: 2218,
    9128: 2223,
    9132: 2224,
    9143: 2209,
    9146: 2543,
    9147: 2544,
    9148: 2546,
    9149: 2547,
    9225: 2530,
    9226: 2533,
    9227: 2537,
    9228: 2531,
    9229: 2532,
    9251: 2732,
    9252: 2536,
    9472: 2211,
    9474: 2214,
    9484: 2210,
    9488: 2539,
    9492: 2541,
    9496: 2538,
    9500: 2548,
    9508: 2549,
    9516: 2551,
    9524: 2550,
    9532: 2542,
    9618: 2529,
    9642: 2791,
    9643: 2785,
    9644: 2779,
    9645: 2786,
    9646: 2783,
    9647: 2767,
    9650: 2792,
    9651: 2787,
    9654: 2781,
    9655: 2765,
    9660: 2793,
    9661: 2788,
    9664: 2780,
    9665: 2764,
    9670: 2528,
    9675: 2766,
    9679: 2782,
    9702: 2784,
    9734: 2789,
    9742: 2809,
    9747: 2762,
    9756: 2794,
    9758: 2795,
    9792: 2808,
    9794: 2807,
    9827: 2796,
    9829: 2798,
    9830: 2797,
    9837: 2806,
    9839: 2805,
    10003: 2803,
    10007: 2804,
    10013: 2777,
    10016: 2800,
    10216: 2748,
    10217: 2750,
    12289: 1188,
    12290: 1185,
    12300: 1186,
    12301: 1187,
    12443: 1246,
    12444: 1247,
    12449: 1191,
    12450: 1201,
    12451: 1192,
    12452: 1202,
    12453: 1193,
    12454: 1203,
    12455: 1194,
    12456: 1204,
    12457: 1195,
    12458: 1205,
    12459: 1206,
    12461: 1207,
    12463: 1208,
    12465: 1209,
    12467: 1210,
    12469: 1211,
    12471: 1212,
    12473: 1213,
    12475: 1214,
    12477: 1215,
    12479: 1216,
    12481: 1217,
    12483: 1199,
    12484: 1218,
    12486: 1219,
    12488: 1220,
    12490: 1221,
    12491: 1222,
    12492: 1223,
    12493: 1224,
    12494: 1225,
    12495: 1226,
    12498: 1227,
    12501: 1228,
    12504: 1229,
    12507: 1230,
    12510: 1231,
    12511: 1232,
    12512: 1233,
    12513: 1234,
    12514: 1235,
    12515: 1196,
    12516: 1236,
    12517: 1197,
    12518: 1237,
    12519: 1198,
    12520: 1238,
    12521: 1239,
    12522: 1240,
    12523: 1241,
    12524: 1242,
    12525: 1243,
    12527: 1244,
    12530: 1190,
    12531: 1245,
    12539: 1189,
    12540: 1200
  }
  e.default = {
    lookup: function (r) {
      if (r >= 32 && r <= 255) return r
      var i = t[r]
      return i !== void 0 ? i : 16777216 | r
    }
  }
})(io)
var ao = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }),
    (e.default = void 0),
    (e.default = {
      8: 'Backspace',
      9: 'Tab',
      10: 'NumpadClear',
      13: 'Enter',
      16: 'ShiftLeft',
      17: 'ControlLeft',
      18: 'AltLeft',
      19: 'Pause',
      20: 'CapsLock',
      21: 'Lang1',
      25: 'Lang2',
      27: 'Escape',
      28: 'Convert',
      29: 'NonConvert',
      32: 'Space',
      33: 'PageUp',
      34: 'PageDown',
      35: 'End',
      36: 'Home',
      37: 'ArrowLeft',
      38: 'ArrowUp',
      39: 'ArrowRight',
      40: 'ArrowDown',
      41: 'Select',
      44: 'PrintScreen',
      45: 'Insert',
      46: 'Delete',
      47: 'Help',
      48: 'Digit0',
      49: 'Digit1',
      50: 'Digit2',
      51: 'Digit3',
      52: 'Digit4',
      53: 'Digit5',
      54: 'Digit6',
      55: 'Digit7',
      56: 'Digit8',
      57: 'Digit9',
      91: 'MetaLeft',
      92: 'MetaRight',
      93: 'ContextMenu',
      95: 'Sleep',
      96: 'Numpad0',
      97: 'Numpad1',
      98: 'Numpad2',
      99: 'Numpad3',
      100: 'Numpad4',
      101: 'Numpad5',
      102: 'Numpad6',
      103: 'Numpad7',
      104: 'Numpad8',
      105: 'Numpad9',
      106: 'NumpadMultiply',
      107: 'NumpadAdd',
      108: 'NumpadDecimal',
      109: 'NumpadSubtract',
      110: 'NumpadDecimal',
      111: 'NumpadDivide',
      112: 'F1',
      113: 'F2',
      114: 'F3',
      115: 'F4',
      116: 'F5',
      117: 'F6',
      118: 'F7',
      119: 'F8',
      120: 'F9',
      121: 'F10',
      122: 'F11',
      123: 'F12',
      124: 'F13',
      125: 'F14',
      126: 'F15',
      127: 'F16',
      128: 'F17',
      129: 'F18',
      130: 'F19',
      131: 'F20',
      132: 'F21',
      133: 'F22',
      134: 'F23',
      135: 'F24',
      144: 'NumLock',
      145: 'ScrollLock',
      166: 'BrowserBack',
      167: 'BrowserForward',
      168: 'BrowserRefresh',
      169: 'BrowserStop',
      170: 'BrowserSearch',
      171: 'BrowserFavorites',
      172: 'BrowserHome',
      173: 'AudioVolumeMute',
      174: 'AudioVolumeDown',
      175: 'AudioVolumeUp',
      176: 'MediaTrackNext',
      177: 'MediaTrackPrevious',
      178: 'MediaStop',
      179: 'MediaPlayPause',
      180: 'LaunchMail',
      181: 'MediaSelect',
      182: 'LaunchApp1',
      183: 'LaunchApp2',
      225: 'AltRight'
    }))
})(ao)
var oo = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }),
    (e.default = void 0),
    (e.default = {
      Backspace: 'Backspace',
      AltLeft: 'Alt',
      AltRight: 'Alt',
      CapsLock: 'CapsLock',
      ContextMenu: 'ContextMenu',
      ControlLeft: 'Control',
      ControlRight: 'Control',
      Enter: 'Enter',
      MetaLeft: 'Meta',
      MetaRight: 'Meta',
      ShiftLeft: 'Shift',
      ShiftRight: 'Shift',
      Tab: 'Tab',
      Delete: 'Delete',
      End: 'End',
      Help: 'Help',
      Home: 'Home',
      Insert: 'Insert',
      PageDown: 'PageDown',
      PageUp: 'PageUp',
      ArrowDown: 'ArrowDown',
      ArrowLeft: 'ArrowLeft',
      ArrowRight: 'ArrowRight',
      ArrowUp: 'ArrowUp',
      NumLock: 'NumLock',
      NumpadBackspace: 'Backspace',
      NumpadClear: 'Clear',
      Escape: 'Escape',
      F1: 'F1',
      F2: 'F2',
      F3: 'F3',
      F4: 'F4',
      F5: 'F5',
      F6: 'F6',
      F7: 'F7',
      F8: 'F8',
      F9: 'F9',
      F10: 'F10',
      F11: 'F11',
      F12: 'F12',
      F13: 'F13',
      F14: 'F14',
      F15: 'F15',
      F16: 'F16',
      F17: 'F17',
      F18: 'F18',
      F19: 'F19',
      F20: 'F20',
      F21: 'F21',
      F22: 'F22',
      F23: 'F23',
      F24: 'F24',
      F25: 'F25',
      F26: 'F26',
      F27: 'F27',
      F28: 'F28',
      F29: 'F29',
      F30: 'F30',
      F31: 'F31',
      F32: 'F32',
      F33: 'F33',
      F34: 'F34',
      F35: 'F35',
      PrintScreen: 'PrintScreen',
      ScrollLock: 'ScrollLock',
      Pause: 'Pause',
      BrowserBack: 'BrowserBack',
      BrowserFavorites: 'BrowserFavorites',
      BrowserForward: 'BrowserForward',
      BrowserHome: 'BrowserHome',
      BrowserRefresh: 'BrowserRefresh',
      BrowserSearch: 'BrowserSearch',
      BrowserStop: 'BrowserStop',
      Eject: 'Eject',
      LaunchApp1: 'LaunchMyComputer',
      LaunchApp2: 'LaunchCalendar',
      LaunchMail: 'LaunchMail',
      MediaPlayPause: 'MediaPlay',
      MediaStop: 'MediaStop',
      MediaTrackNext: 'MediaTrackNext',
      MediaTrackPrevious: 'MediaTrackPrevious',
      Power: 'Power',
      Sleep: 'Sleep',
      AudioVolumeDown: 'AudioVolumeDown',
      AudioVolumeMute: 'AudioVolumeMute',
      AudioVolumeUp: 'AudioVolumeUp',
      WakeUp: 'WakeUp'
    }))
})(oo)
var so = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = n(Ar)
  function n(s) {
    return s && s.__esModule ? s : { default: s }
  }
  var r = {}
  function i(s, x) {
    if (x === void 0) throw new Error('Undefined keysym for key "' + s + '"')
    if (s in r) throw new Error('Duplicate entry for key "' + s + '"')
    r[s] = [x, x, x, x]
  }
  function o(s, x, u) {
    if (x === void 0) throw new Error('Undefined keysym for key "' + s + '"')
    if (u === void 0) throw new Error('Undefined keysym for key "' + s + '"')
    if (s in r) throw new Error('Duplicate entry for key "' + s + '"')
    r[s] = [x, x, u, x]
  }
  function a(s, x, u) {
    if (x === void 0) throw new Error('Undefined keysym for key "' + s + '"')
    if (u === void 0) throw new Error('Undefined keysym for key "' + s + '"')
    if (s in r) throw new Error('Duplicate entry for key "' + s + '"')
    r[s] = [x, x, x, u]
  }
  ;(o('Alt', t.default.XK_Alt_L, t.default.XK_Alt_R),
    i('AltGraph', t.default.XK_ISO_Level3_Shift),
    i('CapsLock', t.default.XK_Caps_Lock),
    o('Control', t.default.XK_Control_L, t.default.XK_Control_R),
    o('Meta', t.default.XK_Super_L, t.default.XK_Super_R),
    i('NumLock', t.default.XK_Num_Lock),
    i('ScrollLock', t.default.XK_Scroll_Lock),
    o('Shift', t.default.XK_Shift_L, t.default.XK_Shift_R),
    a('Enter', t.default.XK_Return, t.default.XK_KP_Enter),
    i('Tab', t.default.XK_Tab),
    a(' ', t.default.XK_space, t.default.XK_KP_Space),
    a('ArrowDown', t.default.XK_Down, t.default.XK_KP_Down),
    a('ArrowLeft', t.default.XK_Left, t.default.XK_KP_Left),
    a('ArrowRight', t.default.XK_Right, t.default.XK_KP_Right),
    a('ArrowUp', t.default.XK_Up, t.default.XK_KP_Up),
    a('End', t.default.XK_End, t.default.XK_KP_End),
    a('Home', t.default.XK_Home, t.default.XK_KP_Home),
    a('PageDown', t.default.XK_Next, t.default.XK_KP_Next),
    a('PageUp', t.default.XK_Prior, t.default.XK_KP_Prior),
    i('Backspace', t.default.XK_BackSpace),
    a('Clear', t.default.XK_Clear, t.default.XK_KP_Begin),
    i('Copy', t.default.XF86XK_Copy),
    i('Cut', t.default.XF86XK_Cut),
    a('Delete', t.default.XK_Delete, t.default.XK_KP_Delete),
    a('Insert', t.default.XK_Insert, t.default.XK_KP_Insert),
    i('Paste', t.default.XF86XK_Paste),
    i('Redo', t.default.XK_Redo),
    i('Undo', t.default.XK_Undo),
    i('Cancel', t.default.XK_Cancel),
    i('ContextMenu', t.default.XK_Menu),
    i('Escape', t.default.XK_Escape),
    i('Execute', t.default.XK_Execute),
    i('Find', t.default.XK_Find),
    i('Help', t.default.XK_Help),
    i('Pause', t.default.XK_Pause),
    i('Select', t.default.XK_Select),
    i('ZoomIn', t.default.XF86XK_ZoomIn),
    i('ZoomOut', t.default.XF86XK_ZoomOut),
    i('BrightnessDown', t.default.XF86XK_MonBrightnessDown),
    i('BrightnessUp', t.default.XF86XK_MonBrightnessUp),
    i('Eject', t.default.XF86XK_Eject),
    i('LogOff', t.default.XF86XK_LogOff),
    i('Power', t.default.XF86XK_PowerOff),
    i('PowerOff', t.default.XF86XK_PowerDown),
    i('PrintScreen', t.default.XK_Print),
    i('Hibernate', t.default.XF86XK_Hibernate),
    i('Standby', t.default.XF86XK_Standby),
    i('WakeUp', t.default.XF86XK_WakeUp),
    i('AllCandidates', t.default.XK_MultipleCandidate),
    i('Alphanumeric', t.default.XK_Eisu_toggle),
    i('CodeInput', t.default.XK_Codeinput),
    i('Compose', t.default.XK_Multi_key),
    i('Convert', t.default.XK_Henkan),
    i('GroupFirst', t.default.XK_ISO_First_Group),
    i('GroupLast', t.default.XK_ISO_Last_Group),
    i('GroupNext', t.default.XK_ISO_Next_Group),
    i('GroupPrevious', t.default.XK_ISO_Prev_Group),
    i('NonConvert', t.default.XK_Muhenkan),
    i('PreviousCandidate', t.default.XK_PreviousCandidate),
    i('SingleCandidate', t.default.XK_SingleCandidate),
    i('HangulMode', t.default.XK_Hangul),
    i('HanjaMode', t.default.XK_Hangul_Hanja),
    i('JunjaMode', t.default.XK_Hangul_Jeonja),
    i('Eisu', t.default.XK_Eisu_toggle),
    i('Hankaku', t.default.XK_Hankaku),
    i('Hiragana', t.default.XK_Hiragana),
    i('HiraganaKatakana', t.default.XK_Hiragana_Katakana),
    i('KanaMode', t.default.XK_Kana_Shift),
    i('KanjiMode', t.default.XK_Kanji),
    i('Katakana', t.default.XK_Katakana),
    i('Romaji', t.default.XK_Romaji),
    i('Zenkaku', t.default.XK_Zenkaku),
    i('ZenkakuHankaku', t.default.XK_Zenkaku_Hankaku),
    i('F1', t.default.XK_F1),
    i('F2', t.default.XK_F2),
    i('F3', t.default.XK_F3),
    i('F4', t.default.XK_F4),
    i('F5', t.default.XK_F5),
    i('F6', t.default.XK_F6),
    i('F7', t.default.XK_F7),
    i('F8', t.default.XK_F8),
    i('F9', t.default.XK_F9),
    i('F10', t.default.XK_F10),
    i('F11', t.default.XK_F11),
    i('F12', t.default.XK_F12),
    i('F13', t.default.XK_F13),
    i('F14', t.default.XK_F14),
    i('F15', t.default.XK_F15),
    i('F16', t.default.XK_F16),
    i('F17', t.default.XK_F17),
    i('F18', t.default.XK_F18),
    i('F19', t.default.XK_F19),
    i('F20', t.default.XK_F20),
    i('F21', t.default.XK_F21),
    i('F22', t.default.XK_F22),
    i('F23', t.default.XK_F23),
    i('F24', t.default.XK_F24),
    i('F25', t.default.XK_F25),
    i('F26', t.default.XK_F26),
    i('F27', t.default.XK_F27),
    i('F28', t.default.XK_F28),
    i('F29', t.default.XK_F29),
    i('F30', t.default.XK_F30),
    i('F31', t.default.XK_F31),
    i('F32', t.default.XK_F32),
    i('F33', t.default.XK_F33),
    i('F34', t.default.XK_F34),
    i('F35', t.default.XK_F35),
    i('Close', t.default.XF86XK_Close),
    i('MailForward', t.default.XF86XK_MailForward),
    i('MailReply', t.default.XF86XK_Reply),
    i('MailSend', t.default.XF86XK_Send),
    i('MediaFastForward', t.default.XF86XK_AudioForward),
    i('MediaPause', t.default.XF86XK_AudioPause),
    i('MediaPlay', t.default.XF86XK_AudioPlay),
    i('MediaRecord', t.default.XF86XK_AudioRecord),
    i('MediaRewind', t.default.XF86XK_AudioRewind),
    i('MediaStop', t.default.XF86XK_AudioStop),
    i('MediaTrackNext', t.default.XF86XK_AudioNext),
    i('MediaTrackPrevious', t.default.XF86XK_AudioPrev),
    i('New', t.default.XF86XK_New),
    i('Open', t.default.XF86XK_Open),
    i('Print', t.default.XK_Print),
    i('Save', t.default.XF86XK_Save),
    i('SpellCheck', t.default.XF86XK_Spell),
    i('AudioVolumeDown', t.default.XF86XK_AudioLowerVolume),
    i('AudioVolumeUp', t.default.XF86XK_AudioRaiseVolume),
    i('AudioVolumeMute', t.default.XF86XK_AudioMute),
    i('MicrophoneVolumeMute', t.default.XF86XK_AudioMicMute),
    i('LaunchApplication1', t.default.XF86XK_MyComputer),
    i('LaunchApplication2', t.default.XF86XK_Calculator),
    i('LaunchCalendar', t.default.XF86XK_Calendar),
    i('LaunchMail', t.default.XF86XK_Mail),
    i('LaunchMediaPlayer', t.default.XF86XK_AudioMedia),
    i('LaunchMusicPlayer', t.default.XF86XK_Music),
    i('LaunchPhone', t.default.XF86XK_Phone),
    i('LaunchScreenSaver', t.default.XF86XK_ScreenSaver),
    i('LaunchSpreadsheet', t.default.XF86XK_Excel),
    i('LaunchWebBrowser', t.default.XF86XK_WWW),
    i('LaunchWebCam', t.default.XF86XK_WebCam),
    i('LaunchWordProcessor', t.default.XF86XK_Word),
    i('BrowserBack', t.default.XF86XK_Back),
    i('BrowserFavorites', t.default.XF86XK_Favorites),
    i('BrowserForward', t.default.XF86XK_Forward),
    i('BrowserHome', t.default.XF86XK_HomePage),
    i('BrowserRefresh', t.default.XF86XK_Refresh),
    i('BrowserSearch', t.default.XF86XK_Search),
    i('BrowserStop', t.default.XF86XK_Stop),
    i('Dimmer', t.default.XF86XK_BrightnessAdjust),
    i('MediaAudioTrack', t.default.XF86XK_AudioCycleTrack),
    i('RandomToggle', t.default.XF86XK_AudioRandomPlay),
    i('SplitScreenToggle', t.default.XF86XK_SplitScreen),
    i('Subtitle', t.default.XF86XK_Subtitle),
    i('VideoModeNext', t.default.XF86XK_Next_VMode),
    a('=', t.default.XK_equal, t.default.XK_KP_Equal),
    a('+', t.default.XK_plus, t.default.XK_KP_Add),
    a('-', t.default.XK_minus, t.default.XK_KP_Subtract),
    a('*', t.default.XK_asterisk, t.default.XK_KP_Multiply),
    a('/', t.default.XK_slash, t.default.XK_KP_Divide),
    a('.', t.default.XK_period, t.default.XK_KP_Decimal),
    a(',', t.default.XK_comma, t.default.XK_KP_Separator),
    a('0', t.default.XK_0, t.default.XK_KP_0),
    a('1', t.default.XK_1, t.default.XK_KP_1),
    a('2', t.default.XK_2, t.default.XK_KP_2),
    a('3', t.default.XK_3, t.default.XK_KP_3),
    a('4', t.default.XK_4, t.default.XK_KP_4),
    a('5', t.default.XK_5, t.default.XK_KP_5),
    a('6', t.default.XK_6, t.default.XK_KP_6),
    a('7', t.default.XK_7, t.default.XK_KP_7),
    a('8', t.default.XK_8, t.default.XK_KP_8),
    a('9', t.default.XK_9, t.default.XK_KP_9),
    (e.default = r))
})(so)
function An(e) {
  '@babel/helpers - typeof'
  return (
    (An =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    An(e)
  )
}
Object.defineProperty(Cr, '__esModule', { value: !0 })
Cr.getKey = fo
Cr.getKeycode = vr
Cr.getKeysym = Lu
var hr = Pr(Ar),
  Pu = Pr(io),
  Hi = Pr(ao),
  Wi = Pr(oo),
  zi = Pr(so),
  uo = Tu(Ae)
function lo(e) {
  if (typeof WeakMap != 'function') return null
  var t = new WeakMap(),
    n = new WeakMap()
  return (lo = function (i) {
    return i ? n : t
  })(e)
}
function Tu(e, t) {
  if (e && e.__esModule) return e
  if (e === null || (An(e) != 'object' && typeof e != 'function')) return { default: e }
  var n = lo(t)
  if (n && n.has(e)) return n.get(e)
  var r = { __proto__: null },
    i = Object.defineProperty && Object.getOwnPropertyDescriptor
  for (var o in e)
    if (o !== 'default' && {}.hasOwnProperty.call(e, o)) {
      var a = i ? Object.getOwnPropertyDescriptor(e, o) : null
      a && (a.get || a.set) ? Object.defineProperty(r, o, a) : (r[o] = e[o])
    }
  return ((r.default = e), n && n.set(e, r), r)
}
function Pr(e) {
  return e && e.__esModule ? e : { default: e }
}
function vr(e) {
  if (e.code) {
    switch (e.code) {
      case 'OSLeft':
        return 'MetaLeft'
      case 'OSRight':
        return 'MetaRight'
    }
    return e.code
  }
  if (e.keyCode in Hi.default) {
    var t = Hi.default[e.keyCode]
    if ((uo.isMac() && t === 'ContextMenu' && (t = 'MetaRight'), e.location === 2))
      switch (t) {
        case 'ShiftLeft':
          return 'ShiftRight'
        case 'ControlLeft':
          return 'ControlRight'
        case 'AltLeft':
          return 'AltRight'
      }
    if (e.location === 3)
      switch (t) {
        case 'Delete':
          return 'NumpadDecimal'
        case 'Insert':
          return 'Numpad0'
        case 'End':
          return 'Numpad1'
        case 'ArrowDown':
          return 'Numpad2'
        case 'PageDown':
          return 'Numpad3'
        case 'ArrowLeft':
          return 'Numpad4'
        case 'ArrowRight':
          return 'Numpad6'
        case 'Home':
          return 'Numpad7'
        case 'ArrowUp':
          return 'Numpad8'
        case 'PageUp':
          return 'Numpad9'
        case 'Enter':
          return 'NumpadEnter'
      }
    return t
  }
  return 'Unidentified'
}
function fo(e) {
  if (e.key !== void 0 && e.key !== 'Unidentified') {
    switch (e.key) {
      case 'OS':
        return 'Meta'
      case 'LaunchMyComputer':
        return 'LaunchApplication1'
      case 'LaunchCalculator':
        return 'LaunchApplication2'
    }
    switch (e.key) {
      case 'UIKeyInputUpArrow':
        return 'ArrowUp'
      case 'UIKeyInputDownArrow':
        return 'ArrowDown'
      case 'UIKeyInputLeftArrow':
        return 'ArrowLeft'
      case 'UIKeyInputRightArrow':
        return 'ArrowRight'
      case 'UIKeyInputEscape':
        return 'Escape'
    }
    return e.key === '\0' && e.code === 'NumpadDecimal' ? 'Delete' : e.key
  }
  var t = vr(e)
  return t in Wi.default
    ? Wi.default[t]
    : e.charCode
      ? String.fromCharCode(e.charCode)
      : 'Unidentified'
}
function Lu(e) {
  var t = fo(e)
  if (t === 'Unidentified') return null
  if (t in zi.default) {
    var n = e.location
    if ((t === 'Meta' && n === 0 && (n = 2), t === 'Clear' && n === 3)) {
      var r = vr(e)
      r === 'NumLock' && (n = 0)
    }
    if (((n === void 0 || n > 3) && (n = 0), t === 'Meta')) {
      var i = vr(e)
      if (i === 'AltLeft') return hr.default.XK_Meta_L
      if (i === 'AltRight') return hr.default.XK_Meta_R
    }
    if (t === 'Clear') {
      var o = vr(e)
      if (o === 'NumLock') return hr.default.XK_Num_Lock
    }
    if (uo.isWindows())
      switch (t) {
        case 'Zenkaku':
        case 'Hankaku':
          return hr.default.XK_Zenkaku_Hankaku
        case 'Romaji':
        case 'KanaMode':
          return hr.default.XK_Romaji
      }
    return zi.default[t][n]
  }
  if (t.length !== 1) return null
  var a = t.charCodeAt()
  return a ? Pu.default.lookup(a) : null
}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = x(Re),
    n = Jt,
    r = x(Cr),
    i = a(Ar),
    o = x(Ae)
  function a(_) {
    return _ && _.__esModule ? _ : { default: _ }
  }
  function s(_) {
    if (typeof WeakMap != 'function') return null
    var l = new WeakMap(),
      h = new WeakMap()
    return (s = function (E) {
      return E ? h : l
    })(_)
  }
  function x(_, l) {
    if (_ && _.__esModule) return _
    if (_ === null || (u(_) != 'object' && typeof _ != 'function')) return { default: _ }
    var h = s(l)
    if (h && h.has(_)) return h.get(_)
    var c = { __proto__: null },
      E = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var y in _)
      if (y !== 'default' && {}.hasOwnProperty.call(_, y)) {
        var K = E ? Object.getOwnPropertyDescriptor(_, y) : null
        K && (K.get || K.set) ? Object.defineProperty(c, y, K) : (c[y] = _[y])
      }
    return ((c.default = _), h && h.set(_, c), c)
  }
  function u(_) {
    '@babel/helpers - typeof'
    return (
      (u =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (l) {
              return typeof l
            }
          : function (l) {
              return l &&
                typeof Symbol == 'function' &&
                l.constructor === Symbol &&
                l !== Symbol.prototype
                ? 'symbol'
                : typeof l
            }),
      u(_)
    )
  }
  function f(_, l) {
    if (!(_ instanceof l)) throw new TypeError('Cannot call a class as a function')
  }
  function F(_, l) {
    for (var h = 0; h < l.length; h++) {
      var c = l[h]
      ;((c.enumerable = c.enumerable || !1),
        (c.configurable = !0),
        'value' in c && (c.writable = !0),
        Object.defineProperty(_, g(c.key), c))
    }
  }
  function X(_, l, h) {
    return (l && F(_.prototype, l), Object.defineProperty(_, 'prototype', { writable: !1 }), _)
  }
  function g(_) {
    var l = k(_, 'string')
    return u(l) == 'symbol' ? l : l + ''
  }
  function k(_, l) {
    if (u(_) != 'object' || !_) return _
    var h = _[Symbol.toPrimitive]
    if (h !== void 0) {
      var c = h.call(_, l)
      if (u(c) != 'object') return c
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(_)
  }
  e.default = (function () {
    function _(l) {
      ;(f(this, _),
        (this._target = l || null),
        (this._keyDownList = {}),
        (this._altGrArmed = !1),
        (this._eventHandlers = {
          keyup: this._handleKeyUp.bind(this),
          keydown: this._handleKeyDown.bind(this),
          blur: this._allKeysUp.bind(this)
        }),
        (this.onkeyevent = function () {}))
    }
    return X(_, [
      {
        key: '_sendKeyEvent',
        value: function (h, c, E) {
          var y = arguments.length > 3 && arguments[3] !== void 0 ? arguments[3] : null,
            K = arguments.length > 4 && arguments[4] !== void 0 ? arguments[4] : null
          if (E) this._keyDownList[c] = h
          else {
            if (!(c in this._keyDownList)) return
            delete this._keyDownList[c]
          }
          ;(t.Debug(
            'onkeyevent ' + (E ? 'down' : 'up') + ', keysym: ' + h,
            ', code: ' + c + ', numlock: ' + y + ', capslock: ' + K
          ),
            this.onkeyevent(h, c, E, y, K))
        }
      },
      {
        key: '_getKeyCode',
        value: function (h) {
          var c = r.getKeycode(h)
          if (c !== 'Unidentified') return c
          if (h.keyCode && h.keyCode !== 229) return 'Platform' + h.keyCode
          if (h.keyIdentifier) {
            if (h.keyIdentifier.substr(0, 2) !== 'U+') return h.keyIdentifier
            var E = parseInt(h.keyIdentifier.substr(2), 16),
              y = String.fromCharCode(E).toUpperCase()
            return 'Platform' + y.charCodeAt()
          }
          return 'Unidentified'
        }
      },
      {
        key: '_handleKeyDown',
        value: function (h) {
          var c = this._getKeyCode(h),
            E = r.getKeysym(h),
            y = h.getModifierState('NumLock'),
            K = h.getModifierState('CapsLock')
          if (
            ((o.isMac() || o.isIOS()) && (y = null),
            this._altGrArmed &&
              ((this._altGrArmed = !1),
              clearTimeout(this._altGrTimeout),
              c === 'AltRight' && h.timeStamp - this._altGrCtrlTime < 50
                ? (E = i.default.XK_ISO_Level3_Shift)
                : this._sendKeyEvent(i.default.XK_Control_L, 'ControlLeft', !0, y, K)),
            c === 'Unidentified')
          ) {
            ;(E && (this._sendKeyEvent(E, c, !0, y, K), this._sendKeyEvent(E, c, !1, y, K)),
              (0, n.stopEvent)(h))
            return
          }
          if (o.isMac() || o.isIOS())
            switch (E) {
              case i.default.XK_Super_L:
                E = i.default.XK_Alt_L
                break
              case i.default.XK_Super_R:
                E = i.default.XK_Super_L
                break
              case i.default.XK_Alt_L:
                E = i.default.XK_Mode_switch
                break
              case i.default.XK_Alt_R:
                E = i.default.XK_ISO_Level3_Shift
                break
            }
          if (
            (c in this._keyDownList && (E = this._keyDownList[c]),
            (o.isMac() || o.isIOS()) && h.metaKey && c !== 'MetaLeft' && c !== 'MetaRight')
          ) {
            ;(this._sendKeyEvent(E, c, !0, y, K),
              this._sendKeyEvent(E, c, !1, y, K),
              (0, n.stopEvent)(h))
            return
          }
          if ((o.isMac() || o.isIOS()) && c === 'CapsLock') {
            ;(this._sendKeyEvent(i.default.XK_Caps_Lock, 'CapsLock', !0, y, K),
              this._sendKeyEvent(i.default.XK_Caps_Lock, 'CapsLock', !1, y, K),
              (0, n.stopEvent)(h))
            return
          }
          var w = [
            i.default.XK_Zenkaku_Hankaku,
            i.default.XK_Eisu_toggle,
            i.default.XK_Katakana,
            i.default.XK_Hiragana,
            i.default.XK_Romaji
          ]
          if (o.isWindows() && w.includes(E)) {
            ;(this._sendKeyEvent(E, c, !0, y, K),
              this._sendKeyEvent(E, c, !1, y, K),
              (0, n.stopEvent)(h))
            return
          }
          if (
            ((0, n.stopEvent)(h),
            c === 'ControlLeft' && o.isWindows() && !('ControlLeft' in this._keyDownList))
          ) {
            ;((this._altGrArmed = !0),
              (this._altGrTimeout = setTimeout(this._interruptAltGrSequence.bind(this), 100)),
              (this._altGrCtrlTime = h.timeStamp))
            return
          }
          this._sendKeyEvent(E, c, !0, y, K)
        }
      },
      {
        key: '_handleKeyUp',
        value: function (h) {
          ;(0, n.stopEvent)(h)
          var c = this._getKeyCode(h)
          if ((this._interruptAltGrSequence(), (o.isMac() || o.isIOS()) && c === 'CapsLock')) {
            ;(this._sendKeyEvent(i.default.XK_Caps_Lock, 'CapsLock', !0),
              this._sendKeyEvent(i.default.XK_Caps_Lock, 'CapsLock', !1))
            return
          }
          ;(this._sendKeyEvent(this._keyDownList[c], c, !1),
            o.isWindows() &&
              (c === 'ShiftLeft' || c === 'ShiftRight') &&
              ('ShiftRight' in this._keyDownList &&
                this._sendKeyEvent(this._keyDownList.ShiftRight, 'ShiftRight', !1),
              'ShiftLeft' in this._keyDownList &&
                this._sendKeyEvent(this._keyDownList.ShiftLeft, 'ShiftLeft', !1)))
        }
      },
      {
        key: '_interruptAltGrSequence',
        value: function () {
          this._altGrArmed &&
            ((this._altGrArmed = !1),
            clearTimeout(this._altGrTimeout),
            this._sendKeyEvent(i.default.XK_Control_L, 'ControlLeft', !0))
        }
      },
      {
        key: '_allKeysUp',
        value: function () {
          ;(t.Debug('>> Keyboard.allKeysUp'), this._interruptAltGrSequence())
          for (var h in this._keyDownList) this._sendKeyEvent(this._keyDownList[h], h, !1)
          t.Debug('<< Keyboard.allKeysUp')
        }
      },
      {
        key: 'grab',
        value: function () {
          ;(this._target.addEventListener('keydown', this._eventHandlers.keydown),
            this._target.addEventListener('keyup', this._eventHandlers.keyup),
            window.addEventListener('blur', this._eventHandlers.blur))
        }
      },
      {
        key: 'ungrab',
        value: function () {
          ;(this._target.removeEventListener('keydown', this._eventHandlers.keydown),
            this._target.removeEventListener('keyup', this._eventHandlers.keyup),
            window.removeEventListener('blur', this._eventHandlers.blur),
            this._allKeysUp())
        }
      }
    ])
  })()
})(no)
var co = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  function t(w) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (m) {
              return typeof m
            }
          : function (m) {
              return m &&
                typeof Symbol == 'function' &&
                m.constructor === Symbol &&
                m !== Symbol.prototype
                ? 'symbol'
                : typeof m
            }),
      t(w)
    )
  }
  function n(w, m) {
    if (!(w instanceof m)) throw new TypeError('Cannot call a class as a function')
  }
  function r(w, m) {
    for (var C = 0; C < m.length; C++) {
      var A = m[C]
      ;((A.enumerable = A.enumerable || !1),
        (A.configurable = !0),
        'value' in A && (A.writable = !0),
        Object.defineProperty(w, o(A.key), A))
    }
  }
  function i(w, m, C) {
    return (m && r(w.prototype, m), Object.defineProperty(w, 'prototype', { writable: !1 }), w)
  }
  function o(w) {
    var m = a(w, 'string')
    return t(m) == 'symbol' ? m : m + ''
  }
  function a(w, m) {
    if (t(w) != 'object' || !w) return w
    var C = w[Symbol.toPrimitive]
    if (C !== void 0) {
      var A = C.call(w, m)
      if (t(A) != 'object') return A
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(w)
  }
  var s = 0,
    x = 1,
    u = 2,
    f = 4,
    F = 8,
    X = 16,
    g = 32,
    k = 64,
    _ = 127,
    l = 50,
    h = 90,
    c = 250,
    E = 1e3,
    y = 1e3,
    K = 50
  e.default = (function () {
    function w() {
      ;(n(this, w),
        (this._target = null),
        (this._state = _),
        (this._tracked = []),
        (this._ignored = []),
        (this._waitingRelease = !1),
        (this._releaseStart = 0),
        (this._longpressTimeoutId = null),
        (this._twoTouchTimeoutId = null),
        (this._boundEventHandler = this._eventHandler.bind(this)))
    }
    return i(w, [
      {
        key: 'attach',
        value: function (C) {
          ;(this.detach(),
            (this._target = C),
            this._target.addEventListener('touchstart', this._boundEventHandler),
            this._target.addEventListener('touchmove', this._boundEventHandler),
            this._target.addEventListener('touchend', this._boundEventHandler),
            this._target.addEventListener('touchcancel', this._boundEventHandler))
        }
      },
      {
        key: 'detach',
        value: function () {
          this._target &&
            (this._stopLongpressTimeout(),
            this._stopTwoTouchTimeout(),
            this._target.removeEventListener('touchstart', this._boundEventHandler),
            this._target.removeEventListener('touchmove', this._boundEventHandler),
            this._target.removeEventListener('touchend', this._boundEventHandler),
            this._target.removeEventListener('touchcancel', this._boundEventHandler),
            (this._target = null))
        }
      },
      {
        key: '_eventHandler',
        value: function (C) {
          var A
          switch ((C.stopPropagation(), C.preventDefault(), C.type)) {
            case 'touchstart':
              A = this._touchStart
              break
            case 'touchmove':
              A = this._touchMove
              break
            case 'touchend':
            case 'touchcancel':
              A = this._touchEnd
              break
          }
          for (var O = 0; O < C.changedTouches.length; O++) {
            var I = C.changedTouches[O]
            A.call(this, I.identifier, I.clientX, I.clientY)
          }
        }
      },
      {
        key: '_touchStart',
        value: function (C, A, O) {
          if (this._hasDetectedGesture() || this._state === s) {
            this._ignored.push(C)
            return
          }
          if (this._tracked.length > 0 && Date.now() - this._tracked[0].started > c) {
            ;((this._state = s), this._ignored.push(C))
            return
          }
          if (this._waitingRelease) {
            ;((this._state = s), this._ignored.push(C))
            return
          }
          switch (
            (this._tracked.push({
              id: C,
              started: Date.now(),
              active: !0,
              firstX: A,
              firstY: O,
              lastX: A,
              lastY: O,
              angle: 0
            }),
            this._tracked.length)
          ) {
            case 1:
              this._startLongpressTimeout()
              break
            case 2:
              ;((this._state &= -26), this._stopLongpressTimeout())
              break
            case 3:
              this._state &= -99
              break
            default:
              this._state = s
          }
        }
      },
      {
        key: '_touchMove',
        value: function (C, A, O) {
          var I = this._tracked.find(function (p) {
            return p.id === C
          })
          if (I !== void 0) {
            ;((I.lastX = A), (I.lastY = O))
            var Z = A - I.firstX,
              q = O - I.firstY
            if (
              ((I.firstX !== I.lastX || I.firstY !== I.lastY) &&
                (I.angle = (Math.atan2(q, Z) * 180) / Math.PI),
              !this._hasDetectedGesture())
            ) {
              if (Math.hypot(Z, q) < l) return
              if (
                ((this._state &= -24),
                this._stopLongpressTimeout(),
                this._tracked.length !== 1 && (this._state &= ~F),
                this._tracked.length !== 2 && (this._state &= -97),
                this._tracked.length === 2)
              ) {
                var $ = this._tracked.find(function (p) {
                    return p.id !== C
                  }),
                  ne = Math.hypot($.firstX - $.lastX, $.firstY - $.lastY)
                if (ne > l) {
                  var S = Math.abs(I.angle - $.angle)
                  ;((S = Math.abs(((S + 180) % 360) - 180)),
                    S > h ? (this._state &= ~g) : (this._state &= ~k),
                    this._isTwoTouchTimeoutRunning() && this._stopTwoTouchTimeout())
                } else this._isTwoTouchTimeoutRunning() || this._startTwoTouchTimeout()
              }
              if (!this._hasDetectedGesture()) return
              this._pushEvent('gesturestart')
            }
            this._pushEvent('gesturemove')
          }
        }
      },
      {
        key: '_touchEnd',
        value: function (C, A, O) {
          if (this._ignored.indexOf(C) !== -1) {
            ;(this._ignored.splice(this._ignored.indexOf(C), 1),
              this._ignored.length === 0 &&
                this._tracked.length === 0 &&
                ((this._state = _), (this._waitingRelease = !1)))
            return
          }
          if (
            (!this._hasDetectedGesture() &&
              this._isTwoTouchTimeoutRunning() &&
              (this._stopTwoTouchTimeout(), (this._state = s)),
            !this._hasDetectedGesture() &&
              ((this._state &= -105),
              (this._state &= ~X),
              this._stopLongpressTimeout(),
              !this._waitingRelease))
          )
            switch (
              ((this._releaseStart = Date.now()), (this._waitingRelease = !0), this._tracked.length)
            ) {
              case 1:
                this._state &= -7
                break
              case 2:
                this._state &= -6
                break
            }
          if (this._waitingRelease) {
            ;(Date.now() - this._releaseStart > c && (this._state = s),
              this._tracked.some(function (q) {
                return Date.now() - q.started > E
              }) && (this._state = s))
            var I = this._tracked.find(function (q) {
              return q.id === C
            })
            if (((I.active = !1), this._hasDetectedGesture())) this._pushEvent('gesturestart')
            else if (this._state !== s) return
          }
          this._hasDetectedGesture() && this._pushEvent('gestureend')
          for (var Z = 0; Z < this._tracked.length; Z++)
            this._tracked[Z].active && this._ignored.push(this._tracked[Z].id)
          ;((this._tracked = []),
            (this._state = s),
            this._ignored.indexOf(C) !== -1 && this._ignored.splice(this._ignored.indexOf(C), 1),
            this._ignored.length === 0 && ((this._state = _), (this._waitingRelease = !1)))
        }
      },
      {
        key: '_hasDetectedGesture',
        value: function () {
          return !(
            this._state === s ||
            this._state & (this._state - 1) ||
            (this._state & (x | u | f) &&
              this._tracked.some(function (C) {
                return C.active
              }))
          )
        }
      },
      {
        key: '_startLongpressTimeout',
        value: function () {
          var C = this
          ;(this._stopLongpressTimeout(),
            (this._longpressTimeoutId = setTimeout(function () {
              return C._longpressTimeout()
            }, y)))
        }
      },
      {
        key: '_stopLongpressTimeout',
        value: function () {
          ;(clearTimeout(this._longpressTimeoutId), (this._longpressTimeoutId = null))
        }
      },
      {
        key: '_longpressTimeout',
        value: function () {
          if (this._hasDetectedGesture())
            throw new Error('A longpress gesture failed, conflict with a different gesture')
          ;((this._state = X), this._pushEvent('gesturestart'))
        }
      },
      {
        key: '_startTwoTouchTimeout',
        value: function () {
          var C = this
          ;(this._stopTwoTouchTimeout(),
            (this._twoTouchTimeoutId = setTimeout(function () {
              return C._twoTouchTimeout()
            }, K)))
        }
      },
      {
        key: '_stopTwoTouchTimeout',
        value: function () {
          ;(clearTimeout(this._twoTouchTimeoutId), (this._twoTouchTimeoutId = null))
        }
      },
      {
        key: '_isTwoTouchTimeoutRunning',
        value: function () {
          return this._twoTouchTimeoutId !== null
        }
      },
      {
        key: '_twoTouchTimeout',
        value: function () {
          if (this._tracked.length === 0)
            throw new Error('A pinch or two drag gesture failed, no tracked touches')
          var C = this._getAverageMovement(),
            A = Math.abs(C.x),
            O = Math.abs(C.y),
            I = this._getAverageDistance(),
            Z = Math.abs(Math.hypot(I.first.x, I.first.y) - Math.hypot(I.last.x, I.last.y))
          ;(O < Z && A < Z ? (this._state = k) : (this._state = g),
            this._pushEvent('gesturestart'),
            this._pushEvent('gesturemove'))
        }
      },
      {
        key: '_pushEvent',
        value: function (C) {
          var A = { type: this._stateToGesture(this._state) },
            O = this._getPosition(),
            I = O.last
          switch ((C === 'gesturestart' && (I = O.first), this._state)) {
            case g:
            case k:
              I = O.first
              break
          }
          if (((A.clientX = I.x), (A.clientY = I.y), this._state === k)) {
            var Z = this._getAverageDistance()
            C === 'gesturestart'
              ? ((A.magnitudeX = Z.first.x), (A.magnitudeY = Z.first.y))
              : ((A.magnitudeX = Z.last.x), (A.magnitudeY = Z.last.y))
          } else if (this._state === g)
            if (C === 'gesturestart') ((A.magnitudeX = 0), (A.magnitudeY = 0))
            else {
              var q = this._getAverageMovement()
              ;((A.magnitudeX = q.x), (A.magnitudeY = q.y))
            }
          var $ = new CustomEvent(C, { detail: A })
          this._target.dispatchEvent($)
        }
      },
      {
        key: '_stateToGesture',
        value: function (C) {
          switch (C) {
            case x:
              return 'onetap'
            case u:
              return 'twotap'
            case f:
              return 'threetap'
            case F:
              return 'drag'
            case X:
              return 'longpress'
            case g:
              return 'twodrag'
            case k:
              return 'pinch'
          }
          throw new Error('Unknown gesture state: ' + C)
        }
      },
      {
        key: '_getPosition',
        value: function () {
          if (this._tracked.length === 0)
            throw new Error('Failed to get gesture position, no tracked touches')
          for (
            var C = this._tracked.length, A = 0, O = 0, I = 0, Z = 0, q = 0;
            q < this._tracked.length;
            q++
          )
            ((A += this._tracked[q].firstX),
              (O += this._tracked[q].firstY),
              (I += this._tracked[q].lastX),
              (Z += this._tracked[q].lastY))
          return { first: { x: A / C, y: O / C }, last: { x: I / C, y: Z / C } }
        }
      },
      {
        key: '_getAverageMovement',
        value: function () {
          if (this._tracked.length === 0)
            throw new Error('Failed to get gesture movement, no tracked touches')
          var C, A
          C = A = 0
          for (var O = this._tracked.length, I = 0; I < this._tracked.length; I++)
            ((C += this._tracked[I].lastX - this._tracked[I].firstX),
              (A += this._tracked[I].lastY - this._tracked[I].firstY))
          return { x: C / O, y: A / O }
        }
      },
      {
        key: '_getAverageDistance',
        value: function () {
          if (this._tracked.length === 0)
            throw new Error('Failed to get gesture distance, no tracked touches')
          var C = this._tracked[0],
            A = this._tracked[this._tracked.length - 1],
            O = Math.abs(A.firstX - C.firstX),
            I = Math.abs(A.firstY - C.firstY),
            Z = Math.abs(A.lastX - C.lastX),
            q = Math.abs(A.lastY - C.lastY)
          return { first: { x: O, y: I }, last: { x: Z, y: q } }
        }
      }
    ])
  })()
})(co)
var ho = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = Ae
  function n(u) {
    '@babel/helpers - typeof'
    return (
      (n =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (f) {
              return typeof f
            }
          : function (f) {
              return f &&
                typeof Symbol == 'function' &&
                f.constructor === Symbol &&
                f !== Symbol.prototype
                ? 'symbol'
                : typeof f
            }),
      n(u)
    )
  }
  function r(u, f) {
    if (!(u instanceof f)) throw new TypeError('Cannot call a class as a function')
  }
  function i(u, f) {
    for (var F = 0; F < f.length; F++) {
      var X = f[F]
      ;((X.enumerable = X.enumerable || !1),
        (X.configurable = !0),
        'value' in X && (X.writable = !0),
        Object.defineProperty(u, a(X.key), X))
    }
  }
  function o(u, f, F) {
    return (f && i(u.prototype, f), Object.defineProperty(u, 'prototype', { writable: !1 }), u)
  }
  function a(u) {
    var f = s(u, 'string')
    return n(f) == 'symbol' ? f : f + ''
  }
  function s(u, f) {
    if (n(u) != 'object' || !u) return u
    var F = u[Symbol.toPrimitive]
    if (F !== void 0) {
      var X = F.call(u, f)
      if (n(X) != 'object') return X
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(u)
  }
  var x = !t.supportsCursorURIs || t.isTouchDevice
  e.default = (function () {
    function u() {
      ;(r(this, u),
        (this._target = null),
        (this._canvas = document.createElement('canvas')),
        x &&
          ((this._canvas.style.position = 'fixed'),
          (this._canvas.style.zIndex = '65535'),
          (this._canvas.style.pointerEvents = 'none'),
          (this._canvas.style.userSelect = 'none'),
          (this._canvas.style.WebkitUserSelect = 'none'),
          (this._canvas.style.visibility = 'hidden')),
        (this._position = { x: 0, y: 0 }),
        (this._hotSpot = { x: 0, y: 0 }),
        (this._eventHandlers = {
          mouseover: this._handleMouseOver.bind(this),
          mouseleave: this._handleMouseLeave.bind(this),
          mousemove: this._handleMouseMove.bind(this),
          mouseup: this._handleMouseUp.bind(this)
        }))
    }
    return o(u, [
      {
        key: 'attach',
        value: function (F) {
          if ((this._target && this.detach(), (this._target = F), x)) {
            document.body.appendChild(this._canvas)
            var X = { capture: !0, passive: !0 }
            ;(this._target.addEventListener('mouseover', this._eventHandlers.mouseover, X),
              this._target.addEventListener('mouseleave', this._eventHandlers.mouseleave, X),
              this._target.addEventListener('mousemove', this._eventHandlers.mousemove, X),
              this._target.addEventListener('mouseup', this._eventHandlers.mouseup, X))
          }
          this.clear()
        }
      },
      {
        key: 'detach',
        value: function () {
          if (this._target) {
            if (x) {
              var F = { capture: !0, passive: !0 }
              ;(this._target.removeEventListener('mouseover', this._eventHandlers.mouseover, F),
                this._target.removeEventListener('mouseleave', this._eventHandlers.mouseleave, F),
                this._target.removeEventListener('mousemove', this._eventHandlers.mousemove, F),
                this._target.removeEventListener('mouseup', this._eventHandlers.mouseup, F),
                document.contains(this._canvas) && document.body.removeChild(this._canvas))
            }
            this._target = null
          }
        }
      },
      {
        key: 'change',
        value: function (F, X, g, k, _) {
          if (k === 0 || _ === 0) {
            this.clear()
            return
          }
          ;((this._position.x = this._position.x + this._hotSpot.x - X),
            (this._position.y = this._position.y + this._hotSpot.y - g),
            (this._hotSpot.x = X),
            (this._hotSpot.y = g))
          var l = this._canvas.getContext('2d')
          ;((this._canvas.width = k), (this._canvas.height = _))
          var h = new ImageData(new Uint8ClampedArray(F), k, _)
          if ((l.clearRect(0, 0, k, _), l.putImageData(h, 0, 0), x)) this._updatePosition()
          else {
            var c = this._canvas.toDataURL()
            this._target.style.cursor = 'url(' + c + ')' + X + ' ' + g + ', default'
          }
        }
      },
      {
        key: 'clear',
        value: function () {
          ;((this._target.style.cursor = 'none'),
            (this._canvas.width = 0),
            (this._canvas.height = 0),
            (this._position.x = this._position.x + this._hotSpot.x),
            (this._position.y = this._position.y + this._hotSpot.y),
            (this._hotSpot.x = 0),
            (this._hotSpot.y = 0))
        }
      },
      {
        key: 'move',
        value: function (F, X) {
          if (x) {
            ;(window.visualViewport
              ? ((this._position.x = F + window.visualViewport.offsetLeft),
                (this._position.y = X + window.visualViewport.offsetTop))
              : ((this._position.x = F), (this._position.y = X)),
              this._updatePosition())
            var g = document.elementFromPoint(F, X)
            this._updateVisibility(g)
          }
        }
      },
      {
        key: '_handleMouseOver',
        value: function (F) {
          this._handleMouseMove(F)
        }
      },
      {
        key: '_handleMouseLeave',
        value: function (F) {
          this._updateVisibility(F.relatedTarget)
        }
      },
      {
        key: '_handleMouseMove',
        value: function (F) {
          ;(this._updateVisibility(F.target),
            (this._position.x = F.clientX - this._hotSpot.x),
            (this._position.y = F.clientY - this._hotSpot.y),
            this._updatePosition())
        }
      },
      {
        key: '_handleMouseUp',
        value: function (F) {
          var X = this,
            g = document.elementFromPoint(F.clientX, F.clientY)
          ;(this._updateVisibility(g),
            this._captureIsActive() &&
              window.setTimeout(function () {
                X._target &&
                  ((g = document.elementFromPoint(F.clientX, F.clientY)), X._updateVisibility(g))
              }, 0))
        }
      },
      {
        key: '_showCursor',
        value: function () {
          this._canvas.style.visibility === 'hidden' && (this._canvas.style.visibility = '')
        }
      },
      {
        key: '_hideCursor',
        value: function () {
          this._canvas.style.visibility !== 'hidden' && (this._canvas.style.visibility = 'hidden')
        }
      },
      {
        key: '_shouldShowCursor',
        value: function (F) {
          return F
            ? F === this._target
              ? !0
              : !(!this._target.contains(F) || window.getComputedStyle(F).cursor !== 'none')
            : !1
        }
      },
      {
        key: '_updateVisibility',
        value: function (F) {
          ;(this._captureIsActive() && (F = document.captureElement),
            this._shouldShowCursor(F) ? this._showCursor() : this._hideCursor())
        }
      },
      {
        key: '_updatePosition',
        value: function () {
          ;((this._canvas.style.left = this._position.x + 'px'),
            (this._canvas.style.top = this._position.y + 'px'))
        }
      },
      {
        key: '_captureIsActive',
        value: function () {
          return (
            document.captureElement && document.documentElement.contains(document.captureElement)
          )
        }
      }
    ])
  })()
})(ho)
var _o = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = r(Re)
  function n(y) {
    if (typeof WeakMap != 'function') return null
    var K = new WeakMap(),
      w = new WeakMap()
    return (n = function (C) {
      return C ? w : K
    })(y)
  }
  function r(y, K) {
    if (y && y.__esModule) return y
    if (y === null || (i(y) != 'object' && typeof y != 'function')) return { default: y }
    var w = n(K)
    if (w && w.has(y)) return w.get(y)
    var m = { __proto__: null },
      C = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var A in y)
      if (A !== 'default' && {}.hasOwnProperty.call(y, A)) {
        var O = C ? Object.getOwnPropertyDescriptor(y, A) : null
        O && (O.get || O.set) ? Object.defineProperty(m, A, O) : (m[A] = y[A])
      }
    return ((m.default = y), w && w.set(y, m), m)
  }
  function i(y) {
    '@babel/helpers - typeof'
    return (
      (i =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (K) {
              return typeof K
            }
          : function (K) {
              return K &&
                typeof Symbol == 'function' &&
                K.constructor === Symbol &&
                K !== Symbol.prototype
                ? 'symbol'
                : typeof K
            }),
      i(y)
    )
  }
  function o(y) {
    return u(y) || x(y) || s(y) || a()
  }
  function a() {
    throw new TypeError(`Invalid attempt to spread non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)
  }
  function s(y, K) {
    if (y) {
      if (typeof y == 'string') return f(y, K)
      var w = {}.toString.call(y).slice(8, -1)
      return (
        w === 'Object' && y.constructor && (w = y.constructor.name),
        w === 'Map' || w === 'Set'
          ? Array.from(y)
          : w === 'Arguments' || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(w)
            ? f(y, K)
            : void 0
      )
    }
  }
  function x(y) {
    if ((typeof Symbol < 'u' && y[Symbol.iterator] != null) || y['@@iterator'] != null)
      return Array.from(y)
  }
  function u(y) {
    if (Array.isArray(y)) return f(y)
  }
  function f(y, K) {
    ;(K == null || K > y.length) && (K = y.length)
    for (var w = 0, m = Array(K); w < K; w++) m[w] = y[w]
    return m
  }
  function F(y, K) {
    if (!(y instanceof K)) throw new TypeError('Cannot call a class as a function')
  }
  function X(y, K) {
    for (var w = 0; w < K.length; w++) {
      var m = K[w]
      ;((m.enumerable = m.enumerable || !1),
        (m.configurable = !0),
        'value' in m && (m.writable = !0),
        Object.defineProperty(y, k(m.key), m))
    }
  }
  function g(y, K, w) {
    return (K && X(y.prototype, K), Object.defineProperty(y, 'prototype', { writable: !1 }), y)
  }
  function k(y) {
    var K = _(y, 'string')
    return i(K) == 'symbol' ? K : K + ''
  }
  function _(y, K) {
    if (i(y) != 'object' || !y) return y
    var w = y[Symbol.toPrimitive]
    if (w !== void 0) {
      var m = w.call(y, K)
      if (i(m) != 'object') return m
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(y)
  }
  var l = 40 * 1024 * 1024,
    h = { CONNECTING: 'connecting', OPEN: 'open', CLOSING: 'closing', CLOSED: 'closed' },
    c = {
      CONNECTING: [WebSocket.CONNECTING, h.CONNECTING],
      OPEN: [WebSocket.OPEN, h.OPEN],
      CLOSING: [WebSocket.CLOSING, h.CLOSING],
      CLOSED: [WebSocket.CLOSED, h.CLOSED]
    },
    E = ['send', 'close', 'binaryType', 'onerror', 'onmessage', 'onopen', 'protocol', 'readyState']
  e.default = (function () {
    function y() {
      ;(F(this, y),
        (this._websocket = null),
        (this._rQi = 0),
        (this._rQlen = 0),
        (this._rQbufferSize = 1024 * 1024 * 4),
        (this._rQ = null),
        (this._sQbufferSize = 1024 * 10),
        (this._sQlen = 0),
        (this._sQ = null),
        (this._eventHandlers = {
          message: function () {},
          open: function () {},
          close: function () {},
          error: function () {}
        }))
    }
    return g(y, [
      {
        key: 'readyState',
        get: function () {
          var w
          return this._websocket === null
            ? 'unused'
            : ((w = this._websocket.readyState),
              c.CONNECTING.includes(w)
                ? 'connecting'
                : c.OPEN.includes(w)
                  ? 'open'
                  : c.CLOSING.includes(w)
                    ? 'closing'
                    : c.CLOSED.includes(w)
                      ? 'closed'
                      : 'unknown')
        }
      },
      {
        key: 'rQpeek8',
        value: function () {
          return this._rQ[this._rQi]
        }
      },
      {
        key: 'rQskipBytes',
        value: function (w) {
          this._rQi += w
        }
      },
      {
        key: 'rQshift8',
        value: function () {
          return this._rQshift(1)
        }
      },
      {
        key: 'rQshift16',
        value: function () {
          return this._rQshift(2)
        }
      },
      {
        key: 'rQshift32',
        value: function () {
          return this._rQshift(4)
        }
      },
      {
        key: '_rQshift',
        value: function (w) {
          for (var m = 0, C = w - 1; C >= 0; C--) m += this._rQ[this._rQi++] << (C * 8)
          return m >>> 0
        }
      },
      {
        key: 'rQshiftStr',
        value: function (w) {
          for (var m = '', C = 0; C < w; C += 4096) {
            var A = this.rQshiftBytes(Math.min(4096, w - C), !1)
            m += String.fromCharCode.apply(null, A)
          }
          return m
        }
      },
      {
        key: 'rQshiftBytes',
        value: function (w) {
          var m = arguments.length > 1 && arguments[1] !== void 0 ? arguments[1] : !0
          return (
            (this._rQi += w),
            m
              ? this._rQ.slice(this._rQi - w, this._rQi)
              : this._rQ.subarray(this._rQi - w, this._rQi)
          )
        }
      },
      {
        key: 'rQshiftTo',
        value: function (w, m) {
          ;(w.set(new Uint8Array(this._rQ.buffer, this._rQi, m)), (this._rQi += m))
        }
      },
      {
        key: 'rQpeekBytes',
        value: function (w) {
          var m = arguments.length > 1 && arguments[1] !== void 0 ? arguments[1] : !0
          return m
            ? this._rQ.slice(this._rQi, this._rQi + w)
            : this._rQ.subarray(this._rQi, this._rQi + w)
        }
      },
      {
        key: 'rQwait',
        value: function (w, m, C) {
          if (this._rQlen - this._rQi < m) {
            if (C) {
              if (this._rQi < C) throw new Error('rQwait cannot backup ' + C + ' bytes')
              this._rQi -= C
            }
            return !0
          }
          return !1
        }
      },
      {
        key: 'sQpush8',
        value: function (w) {
          ;(this._sQensureSpace(1), (this._sQ[this._sQlen++] = w))
        }
      },
      {
        key: 'sQpush16',
        value: function (w) {
          ;(this._sQensureSpace(2),
            (this._sQ[this._sQlen++] = (w >> 8) & 255),
            (this._sQ[this._sQlen++] = (w >> 0) & 255))
        }
      },
      {
        key: 'sQpush32',
        value: function (w) {
          ;(this._sQensureSpace(4),
            (this._sQ[this._sQlen++] = (w >> 24) & 255),
            (this._sQ[this._sQlen++] = (w >> 16) & 255),
            (this._sQ[this._sQlen++] = (w >> 8) & 255),
            (this._sQ[this._sQlen++] = (w >> 0) & 255))
        }
      },
      {
        key: 'sQpushString',
        value: function (w) {
          var m = w.split('').map(function (C) {
            return C.charCodeAt(0)
          })
          this.sQpushBytes(new Uint8Array(m))
        }
      },
      {
        key: 'sQpushBytes',
        value: function (w) {
          for (var m = 0; m < w.length; ) {
            this._sQensureSpace(1)
            var C = this._sQbufferSize - this._sQlen
            ;(C > w.length - m && (C = w.length - m),
              this._sQ.set(w.subarray(m, m + C), this._sQlen),
              (this._sQlen += C),
              (m += C))
          }
        }
      },
      {
        key: 'flush',
        value: function () {
          this._sQlen > 0 &&
            this.readyState === 'open' &&
            (this._websocket.send(new Uint8Array(this._sQ.buffer, 0, this._sQlen)),
            (this._sQlen = 0))
        }
      },
      {
        key: '_sQensureSpace',
        value: function (w) {
          this._sQbufferSize - this._sQlen < w && this.flush()
        }
      },
      {
        key: 'off',
        value: function (w) {
          this._eventHandlers[w] = function () {}
        }
      },
      {
        key: 'on',
        value: function (w, m) {
          this._eventHandlers[w] = m
        }
      },
      {
        key: '_allocateBuffers',
        value: function () {
          ;((this._rQ = new Uint8Array(this._rQbufferSize)),
            (this._sQ = new Uint8Array(this._sQbufferSize)))
        }
      },
      {
        key: 'init',
        value: function () {
          ;(this._allocateBuffers(), (this._rQi = 0), (this._websocket = null))
        }
      },
      {
        key: 'open',
        value: function (w, m) {
          this.attach(new WebSocket(w, m))
        }
      },
      {
        key: 'attach',
        value: function (w) {
          var m = this
          this.init()
          for (
            var C = [].concat(
                o(Object.keys(w)),
                o(Object.getOwnPropertyNames(Object.getPrototypeOf(w)))
              ),
              A = 0;
            A < E.length;
            A++
          ) {
            var O = E[A]
            if (C.indexOf(O) < 0) throw new Error('Raw channel missing property: ' + O)
          }
          ;((this._websocket = w),
            (this._websocket.binaryType = 'arraybuffer'),
            (this._websocket.onmessage = this._recvMessage.bind(this)),
            (this._websocket.onopen = function () {
              ;(t.Debug('>> WebSock.onopen'),
                m._websocket.protocol &&
                  t.Info('Server choose sub-protocol: ' + m._websocket.protocol),
                m._eventHandlers.open(),
                t.Debug('<< WebSock.onopen'))
            }),
            (this._websocket.onclose = function (I) {
              ;(t.Debug('>> WebSock.onclose'),
                m._eventHandlers.close(I),
                t.Debug('<< WebSock.onclose'))
            }),
            (this._websocket.onerror = function (I) {
              ;(t.Debug('>> WebSock.onerror: ' + I),
                m._eventHandlers.error(I),
                t.Debug('<< WebSock.onerror: ' + I))
            }))
        }
      },
      {
        key: 'close',
        value: function () {
          this._websocket &&
            ((this.readyState === 'connecting' || this.readyState === 'open') &&
              (t.Info('Closing WebSocket connection'), this._websocket.close()),
            (this._websocket.onmessage = function () {}))
        }
      },
      {
        key: '_expandCompactRQ',
        value: function (w) {
          var m = (this._rQlen - this._rQi + w) * 8,
            C = this._rQbufferSize < m
          if (
            (C && (this._rQbufferSize = Math.max(this._rQbufferSize * 2, m)),
            this._rQbufferSize > l &&
              ((this._rQbufferSize = l), this._rQbufferSize - (this._rQlen - this._rQi) < w))
          )
            throw new Error(
              'Receive queue buffer exceeded ' + l + ' bytes, and the new message could not fit'
            )
          if (C) {
            var A = this._rQ.buffer
            ;((this._rQ = new Uint8Array(this._rQbufferSize)),
              this._rQ.set(new Uint8Array(A, this._rQi, this._rQlen - this._rQi)))
          } else this._rQ.copyWithin(0, this._rQi, this._rQlen)
          ;((this._rQlen = this._rQlen - this._rQi), (this._rQi = 0))
        }
      },
      {
        key: '_recvMessage',
        value: function (w) {
          this._rQlen == this._rQi && ((this._rQlen = 0), (this._rQi = 0))
          var m = new Uint8Array(w.data)
          ;(m.length > this._rQbufferSize - this._rQlen && this._expandCompactRQ(m.length),
            this._rQ.set(m, this._rQlen),
            (this._rQlen += m.length),
            this._rQlen - this._rQi > 0
              ? this._eventHandlers.message()
              : t.Debug('Ignoring empty message'))
        }
      }
    ])
  })()
})(_o)
var po = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }),
    (e.default = void 0),
    (e.default = {
      Again: 57349,
      AltLeft: 56,
      AltRight: 57400,
      ArrowDown: 57424,
      ArrowLeft: 57419,
      ArrowRight: 57421,
      ArrowUp: 57416,
      AudioVolumeDown: 57390,
      AudioVolumeMute: 57376,
      AudioVolumeUp: 57392,
      Backquote: 41,
      Backslash: 43,
      Backspace: 14,
      BracketLeft: 26,
      BracketRight: 27,
      BrowserBack: 57450,
      BrowserFavorites: 57446,
      BrowserForward: 57449,
      BrowserHome: 57394,
      BrowserRefresh: 57447,
      BrowserSearch: 57445,
      BrowserStop: 57448,
      CapsLock: 58,
      Comma: 51,
      ContextMenu: 57437,
      ControlLeft: 29,
      ControlRight: 57373,
      Convert: 121,
      Copy: 57464,
      Cut: 57404,
      Delete: 57427,
      Digit0: 11,
      Digit1: 2,
      Digit2: 3,
      Digit3: 4,
      Digit4: 5,
      Digit5: 6,
      Digit6: 7,
      Digit7: 8,
      Digit8: 9,
      Digit9: 10,
      Eject: 57469,
      End: 57423,
      Enter: 28,
      Equal: 13,
      Escape: 1,
      F1: 59,
      F10: 68,
      F11: 87,
      F12: 88,
      F13: 93,
      F14: 94,
      F15: 95,
      F16: 85,
      F17: 57347,
      F18: 57463,
      F19: 57348,
      F2: 60,
      F20: 90,
      F21: 116,
      F22: 57465,
      F23: 109,
      F24: 111,
      F3: 61,
      F4: 62,
      F5: 63,
      F6: 64,
      F7: 65,
      F8: 66,
      F9: 67,
      Find: 57409,
      Help: 57461,
      Hiragana: 119,
      Home: 57415,
      Insert: 57426,
      IntlBackslash: 86,
      IntlRo: 115,
      IntlYen: 125,
      KanaMode: 112,
      Katakana: 120,
      KeyA: 30,
      KeyB: 48,
      KeyC: 46,
      KeyD: 32,
      KeyE: 18,
      KeyF: 33,
      KeyG: 34,
      KeyH: 35,
      KeyI: 23,
      KeyJ: 36,
      KeyK: 37,
      KeyL: 38,
      KeyM: 50,
      KeyN: 49,
      KeyO: 24,
      KeyP: 25,
      KeyQ: 16,
      KeyR: 19,
      KeyS: 31,
      KeyT: 20,
      KeyU: 22,
      KeyV: 47,
      KeyW: 17,
      KeyX: 45,
      KeyY: 21,
      KeyZ: 44,
      Lang1: 114,
      Lang2: 113,
      Lang3: 120,
      Lang4: 119,
      Lang5: 118,
      LaunchApp1: 57451,
      LaunchApp2: 57377,
      LaunchMail: 57452,
      MediaPlayPause: 57378,
      MediaSelect: 57453,
      MediaStop: 57380,
      MediaTrackNext: 57369,
      MediaTrackPrevious: 57360,
      MetaLeft: 57435,
      MetaRight: 57436,
      Minus: 12,
      NonConvert: 123,
      NumLock: 69,
      Numpad0: 82,
      Numpad1: 79,
      Numpad2: 80,
      Numpad3: 81,
      Numpad4: 75,
      Numpad5: 76,
      Numpad6: 77,
      Numpad7: 71,
      Numpad8: 72,
      Numpad9: 73,
      NumpadAdd: 78,
      NumpadComma: 126,
      NumpadDecimal: 83,
      NumpadDivide: 57397,
      NumpadEnter: 57372,
      NumpadEqual: 89,
      NumpadMultiply: 55,
      NumpadParenLeft: 57462,
      NumpadParenRight: 57467,
      NumpadSubtract: 74,
      Open: 100,
      PageDown: 57425,
      PageUp: 57417,
      Paste: 101,
      Pause: 57414,
      Period: 52,
      Power: 57438,
      PrintScreen: 84,
      Props: 57350,
      Quote: 40,
      ScrollLock: 70,
      Semicolon: 39,
      ShiftLeft: 42,
      ShiftRight: 54,
      Slash: 53,
      Sleep: 57439,
      Space: 57,
      Suspend: 57381,
      Tab: 15,
      Undo: 57351,
      WakeUp: 57443
    }))
})(po)
var Tr = {}
Object.defineProperty(Tr, '__esModule', { value: !0 })
Tr.encodingName = Ru
Tr.encodings = void 0
var Et = (Tr.encodings = {
  encodingRaw: 0,
  encodingCopyRect: 1,
  encodingRRE: 2,
  encodingHextile: 5,
  encodingZlib: 6,
  encodingTight: 7,
  encodingZRLE: 16,
  encodingTightPNG: -260,
  encodingJPEG: 21,
  encodingH264: 50,
  pseudoEncodingQualityLevel9: -23,
  pseudoEncodingQualityLevel0: -32,
  pseudoEncodingDesktopSize: -223,
  pseudoEncodingLastRect: -224,
  pseudoEncodingCursor: -239,
  pseudoEncodingQEMUExtendedKeyEvent: -258,
  pseudoEncodingQEMULedEvent: -261,
  pseudoEncodingDesktopName: -307,
  pseudoEncodingExtendedDesktopSize: -308,
  pseudoEncodingXvp: -309,
  pseudoEncodingFence: -312,
  pseudoEncodingContinuousUpdates: -313,
  pseudoEncodingExtendedMouseButtons: -316,
  pseudoEncodingCompressLevel9: -247,
  pseudoEncodingCompressLevel0: -256,
  pseudoEncodingVMwareCursor: 1464686180,
  pseudoEncodingExtendedClipboard: 3231835598
})
function Ru(e) {
  switch (e) {
    case Et.encodingRaw:
      return 'Raw'
    case Et.encodingCopyRect:
      return 'CopyRect'
    case Et.encodingRRE:
      return 'RRE'
    case Et.encodingHextile:
      return 'Hextile'
    case Et.encodingZlib:
      return 'Zlib'
    case Et.encodingTight:
      return 'Tight'
    case Et.encodingZRLE:
      return 'ZRLE'
    case Et.encodingTightPNG:
      return 'TightPNG'
    case Et.encodingJPEG:
      return 'JPEG'
    case Et.encodingH264:
      return 'H.264'
    default:
      return '[unknown encoding ' + e + ']'
  }
}
var vo = {},
  Hn = {},
  ar = {}
Object.defineProperty(ar, '__esModule', { value: !0 })
ar.AESECBCipher = ar.AESEAXCipher = void 0
function Yt(e) {
  '@babel/helpers - typeof'
  return (
    (Yt =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    Yt(e)
  )
}
function De() {
  /*! regenerator-runtime -- Copyright (c) 2014-present, Facebook, Inc. -- license (MIT): https://github.com/facebook/regenerator/blob/main/LICENSE */ De =
    function () {
      return t
    }
  var e,
    t = {},
    n = Object.prototype,
    r = n.hasOwnProperty,
    i =
      Object.defineProperty ||
      function (S, p, b) {
        S[p] = b.value
      },
    o = typeof Symbol == 'function' ? Symbol : {},
    a = o.iterator || '@@iterator',
    s = o.asyncIterator || '@@asyncIterator',
    x = o.toStringTag || '@@toStringTag'
  function u(S, p, b) {
    return (
      Object.defineProperty(S, p, { value: b, enumerable: !0, configurable: !0, writable: !0 }),
      S[p]
    )
  }
  try {
    u({}, '')
  } catch {
    u = function (b, L, B) {
      return (b[L] = B)
    }
  }
  function f(S, p, b, L) {
    var B = p && p.prototype instanceof h ? p : h,
      M = Object.create(B.prototype),
      V = new $(L || [])
    return (i(M, '_invoke', { value: O(S, b, V) }), M)
  }
  function F(S, p, b) {
    try {
      return { type: 'normal', arg: S.call(p, b) }
    } catch (L) {
      return { type: 'throw', arg: L }
    }
  }
  t.wrap = f
  var X = 'suspendedStart',
    g = 'suspendedYield',
    k = 'executing',
    _ = 'completed',
    l = {}
  function h() {}
  function c() {}
  function E() {}
  var y = {}
  u(y, a, function () {
    return this
  })
  var K = Object.getPrototypeOf,
    w = K && K(K(ne([])))
  w && w !== n && r.call(w, a) && (y = w)
  var m = (E.prototype = h.prototype = Object.create(y))
  function C(S) {
    ;['next', 'throw', 'return'].forEach(function (p) {
      u(S, p, function (b) {
        return this._invoke(p, b)
      })
    })
  }
  function A(S, p) {
    function b(B, M, V, re) {
      var oe = F(S[B], S, M)
      if (oe.type !== 'throw') {
        var ce = oe.arg,
          pe = ce.value
        return pe && Yt(pe) == 'object' && r.call(pe, '__await')
          ? p.resolve(pe.__await).then(
              function (ye) {
                b('next', ye, V, re)
              },
              function (ye) {
                b('throw', ye, V, re)
              }
            )
          : p.resolve(pe).then(
              function (ye) {
                ;((ce.value = ye), V(ce))
              },
              function (ye) {
                return b('throw', ye, V, re)
              }
            )
      }
      re(oe.arg)
    }
    var L
    i(this, '_invoke', {
      value: function (M, V) {
        function re() {
          return new p(function (oe, ce) {
            b(M, V, oe, ce)
          })
        }
        return (L = L ? L.then(re, re) : re())
      }
    })
  }
  function O(S, p, b) {
    var L = X
    return function (B, M) {
      if (L === k) throw Error('Generator is already running')
      if (L === _) {
        if (B === 'throw') throw M
        return { value: e, done: !0 }
      }
      for (b.method = B, b.arg = M; ; ) {
        var V = b.delegate
        if (V) {
          var re = I(V, b)
          if (re) {
            if (re === l) continue
            return re
          }
        }
        if (b.method === 'next') b.sent = b._sent = b.arg
        else if (b.method === 'throw') {
          if (L === X) throw ((L = _), b.arg)
          b.dispatchException(b.arg)
        } else b.method === 'return' && b.abrupt('return', b.arg)
        L = k
        var oe = F(S, p, b)
        if (oe.type === 'normal') {
          if (((L = b.done ? _ : g), oe.arg === l)) continue
          return { value: oe.arg, done: b.done }
        }
        oe.type === 'throw' && ((L = _), (b.method = 'throw'), (b.arg = oe.arg))
      }
    }
  }
  function I(S, p) {
    var b = p.method,
      L = S.iterator[b]
    if (L === e)
      return (
        (p.delegate = null),
        (b === 'throw' &&
          S.iterator.return &&
          ((p.method = 'return'), (p.arg = e), I(S, p), p.method === 'throw')) ||
          (b !== 'return' &&
            ((p.method = 'throw'),
            (p.arg = new TypeError("The iterator does not provide a '" + b + "' method")))),
        l
      )
    var B = F(L, S.iterator, p.arg)
    if (B.type === 'throw') return ((p.method = 'throw'), (p.arg = B.arg), (p.delegate = null), l)
    var M = B.arg
    return M
      ? M.done
        ? ((p[S.resultName] = M.value),
          (p.next = S.nextLoc),
          p.method !== 'return' && ((p.method = 'next'), (p.arg = e)),
          (p.delegate = null),
          l)
        : M
      : ((p.method = 'throw'),
        (p.arg = new TypeError('iterator result is not an object')),
        (p.delegate = null),
        l)
  }
  function Z(S) {
    var p = { tryLoc: S[0] }
    ;(1 in S && (p.catchLoc = S[1]),
      2 in S && ((p.finallyLoc = S[2]), (p.afterLoc = S[3])),
      this.tryEntries.push(p))
  }
  function q(S) {
    var p = S.completion || {}
    ;((p.type = 'normal'), delete p.arg, (S.completion = p))
  }
  function $(S) {
    ;((this.tryEntries = [{ tryLoc: 'root' }]), S.forEach(Z, this), this.reset(!0))
  }
  function ne(S) {
    if (S || S === '') {
      var p = S[a]
      if (p) return p.call(S)
      if (typeof S.next == 'function') return S
      if (!isNaN(S.length)) {
        var b = -1,
          L = function B() {
            for (; ++b < S.length; ) if (r.call(S, b)) return ((B.value = S[b]), (B.done = !1), B)
            return ((B.value = e), (B.done = !0), B)
          }
        return (L.next = L)
      }
    }
    throw new TypeError(Yt(S) + ' is not iterable')
  }
  return (
    (c.prototype = E),
    i(m, 'constructor', { value: E, configurable: !0 }),
    i(E, 'constructor', { value: c, configurable: !0 }),
    (c.displayName = u(E, x, 'GeneratorFunction')),
    (t.isGeneratorFunction = function (S) {
      var p = typeof S == 'function' && S.constructor
      return !!p && (p === c || (p.displayName || p.name) === 'GeneratorFunction')
    }),
    (t.mark = function (S) {
      return (
        Object.setPrototypeOf
          ? Object.setPrototypeOf(S, E)
          : ((S.__proto__ = E), u(S, x, 'GeneratorFunction')),
        (S.prototype = Object.create(m)),
        S
      )
    }),
    (t.awrap = function (S) {
      return { __await: S }
    }),
    C(A.prototype),
    u(A.prototype, s, function () {
      return this
    }),
    (t.AsyncIterator = A),
    (t.async = function (S, p, b, L, B) {
      B === void 0 && (B = Promise)
      var M = new A(f(S, p, b, L), B)
      return t.isGeneratorFunction(p)
        ? M
        : M.next().then(function (V) {
            return V.done ? V.value : M.next()
          })
    }),
    C(m),
    u(m, x, 'Generator'),
    u(m, a, function () {
      return this
    }),
    u(m, 'toString', function () {
      return '[object Generator]'
    }),
    (t.keys = function (S) {
      var p = Object(S),
        b = []
      for (var L in p) b.push(L)
      return (
        b.reverse(),
        function B() {
          for (; b.length; ) {
            var M = b.pop()
            if (M in p) return ((B.value = M), (B.done = !1), B)
          }
          return ((B.done = !0), B)
        }
      )
    }),
    (t.values = ne),
    ($.prototype = {
      constructor: $,
      reset: function (p) {
        if (
          ((this.prev = 0),
          (this.next = 0),
          (this.sent = this._sent = e),
          (this.done = !1),
          (this.delegate = null),
          (this.method = 'next'),
          (this.arg = e),
          this.tryEntries.forEach(q),
          !p)
        )
          for (var b in this)
            b.charAt(0) === 't' && r.call(this, b) && !isNaN(+b.slice(1)) && (this[b] = e)
      },
      stop: function () {
        this.done = !0
        var p = this.tryEntries[0].completion
        if (p.type === 'throw') throw p.arg
        return this.rval
      },
      dispatchException: function (p) {
        if (this.done) throw p
        var b = this
        function L(ce, pe) {
          return (
            (V.type = 'throw'),
            (V.arg = p),
            (b.next = ce),
            pe && ((b.method = 'next'), (b.arg = e)),
            !!pe
          )
        }
        for (var B = this.tryEntries.length - 1; B >= 0; --B) {
          var M = this.tryEntries[B],
            V = M.completion
          if (M.tryLoc === 'root') return L('end')
          if (M.tryLoc <= this.prev) {
            var re = r.call(M, 'catchLoc'),
              oe = r.call(M, 'finallyLoc')
            if (re && oe) {
              if (this.prev < M.catchLoc) return L(M.catchLoc, !0)
              if (this.prev < M.finallyLoc) return L(M.finallyLoc)
            } else if (re) {
              if (this.prev < M.catchLoc) return L(M.catchLoc, !0)
            } else {
              if (!oe) throw Error('try statement without catch or finally')
              if (this.prev < M.finallyLoc) return L(M.finallyLoc)
            }
          }
        }
      },
      abrupt: function (p, b) {
        for (var L = this.tryEntries.length - 1; L >= 0; --L) {
          var B = this.tryEntries[L]
          if (B.tryLoc <= this.prev && r.call(B, 'finallyLoc') && this.prev < B.finallyLoc) {
            var M = B
            break
          }
        }
        M && (p === 'break' || p === 'continue') && M.tryLoc <= b && b <= M.finallyLoc && (M = null)
        var V = M ? M.completion : {}
        return (
          (V.type = p),
          (V.arg = b),
          M ? ((this.method = 'next'), (this.next = M.finallyLoc), l) : this.complete(V)
        )
      },
      complete: function (p, b) {
        if (p.type === 'throw') throw p.arg
        return (
          p.type === 'break' || p.type === 'continue'
            ? (this.next = p.arg)
            : p.type === 'return'
              ? ((this.rval = this.arg = p.arg), (this.method = 'return'), (this.next = 'end'))
              : p.type === 'normal' && b && (this.next = b),
          l
        )
      },
      finish: function (p) {
        for (var b = this.tryEntries.length - 1; b >= 0; --b) {
          var L = this.tryEntries[b]
          if (L.finallyLoc === p) return (this.complete(L.completion, L.afterLoc), q(L), l)
        }
      },
      catch: function (p) {
        for (var b = this.tryEntries.length - 1; b >= 0; --b) {
          var L = this.tryEntries[b]
          if (L.tryLoc === p) {
            var B = L.completion
            if (B.type === 'throw') {
              var M = B.arg
              q(L)
            }
            return M
          }
        }
        throw Error('illegal catch attempt')
      },
      delegateYield: function (p, b, L) {
        return (
          (this.delegate = { iterator: ne(p), resultName: b, nextLoc: L }),
          this.method === 'next' && (this.arg = e),
          l
        )
      }
    }),
    t
  )
}
function Gi(e, t, n, r, i, o, a) {
  try {
    var s = e[o](a),
      x = s.value
  } catch (u) {
    return void n(u)
  }
  s.done ? t(x) : Promise.resolve(x).then(r, i)
}
function ct(e) {
  return function () {
    var t = this,
      n = arguments
    return new Promise(function (r, i) {
      var o = e.apply(t, n)
      function a(x) {
        Gi(o, r, i, a, s, 'next', x)
      }
      function s(x) {
        Gi(o, r, i, a, s, 'throw', x)
      }
      a(void 0)
    })
  }
}
function yo(e, t) {
  if (!(e instanceof t)) throw new TypeError('Cannot call a class as a function')
}
function Vi(e, t) {
  for (var n = 0; n < t.length; n++) {
    var r = t[n]
    ;((r.enumerable = r.enumerable || !1),
      (r.configurable = !0),
      'value' in r && (r.writable = !0),
      Object.defineProperty(e, Mu(r.key), r))
  }
}
function xo(e, t, n) {
  return (
    t && Vi(e.prototype, t),
    n && Vi(e, n),
    Object.defineProperty(e, 'prototype', { writable: !1 }),
    e
  )
}
function Mu(e) {
  var t = Du(e, 'string')
  return Yt(t) == 'symbol' ? t : t + ''
}
function Du(e, t) {
  if (Yt(e) != 'object' || !e) return e
  var n = e[Symbol.toPrimitive]
  if (n !== void 0) {
    var r = n.call(e, t)
    if (Yt(r) != 'object') return r
    throw new TypeError('@@toPrimitive must return a primitive value.')
  }
  return String(e)
}
ar.AESECBCipher = (function () {
  function e() {
    ;(yo(this, e), (this._key = null))
  }
  return xo(
    e,
    [
      {
        key: 'algorithm',
        get: function () {
          return { name: 'AES-ECB' }
        }
      },
      {
        key: '_importKey',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o, a) {
              return De().wrap(
                function (x) {
                  for (;;)
                    switch ((x.prev = x.next)) {
                      case 0:
                        return (
                          (x.next = 2),
                          window.crypto.subtle.importKey('raw', i, { name: 'AES-CBC' }, o, a)
                        )
                      case 2:
                        this._key = x.sent
                      case 3:
                      case 'end':
                        return x.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i, o) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: 'encrypt',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o) {
              var a, s, x, u
              return De().wrap(
                function (F) {
                  for (;;)
                    switch ((F.prev = F.next)) {
                      case 0:
                        if (
                          ((a = new Uint8Array(o)), !(a.length % 16 !== 0 || this._key === null))
                        ) {
                          F.next = 3
                          break
                        }
                        return F.abrupt('return', null)
                      case 3:
                        ;((s = a.length / 16), (x = 0))
                      case 5:
                        if (!(x < s)) {
                          F.next = 15
                          break
                        }
                        return (
                          (F.t0 = Uint8Array),
                          (F.next = 9),
                          window.crypto.subtle.encrypt(
                            { name: 'AES-CBC', iv: new Uint8Array(16) },
                            this._key,
                            a.slice(x * 16, x * 16 + 16)
                          )
                        )
                      case 9:
                        ;((F.t1 = F.sent), (u = new F.t0(F.t1).slice(0, 16)), a.set(u, x * 16))
                      case 12:
                        ;(x++, (F.next = 5))
                        break
                      case 15:
                        return F.abrupt('return', a)
                      case 16:
                      case 'end':
                        return F.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      }
    ],
    [
      {
        key: 'importKey',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o, a, s) {
              var x
              return De().wrap(function (f) {
                for (;;)
                  switch ((f.prev = f.next)) {
                    case 0:
                      return ((x = new e()), (f.next = 3), x._importKey(i, a, s))
                    case 3:
                      return f.abrupt('return', x)
                    case 4:
                    case 'end':
                      return f.stop()
                  }
              }, r)
            })
          )
          function n(r, i, o, a) {
            return t.apply(this, arguments)
          }
          return n
        })()
      }
    ]
  )
})()
ar.AESEAXCipher = (function () {
  function e() {
    ;(yo(this, e),
      (this._rawKey = null),
      (this._ctrKey = null),
      (this._cbcKey = null),
      (this._zeroBlock = new Uint8Array(16)),
      (this._prefixBlock0 = this._zeroBlock),
      (this._prefixBlock1 = new Uint8Array([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1])),
      (this._prefixBlock2 = new Uint8Array([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2])))
  }
  return xo(
    e,
    [
      {
        key: 'algorithm',
        get: function () {
          return { name: 'AES-EAX' }
        }
      },
      {
        key: '_encryptBlock',
        value: (function () {
          var t = ct(
            De().mark(function r(i) {
              var o
              return De().wrap(
                function (s) {
                  for (;;)
                    switch ((s.prev = s.next)) {
                      case 0:
                        return (
                          (s.next = 2),
                          window.crypto.subtle.encrypt(
                            { name: 'AES-CBC', iv: this._zeroBlock },
                            this._cbcKey,
                            i
                          )
                        )
                      case 2:
                        return ((o = s.sent), s.abrupt('return', new Uint8Array(o).slice(0, 16)))
                      case 4:
                      case 'end':
                        return s.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: '_initCMAC',
        value: (function () {
          var t = ct(
            De().mark(function r() {
              var i, o, a, s, x
              return De().wrap(
                function (f) {
                  for (;;)
                    switch ((f.prev = f.next)) {
                      case 0:
                        return ((f.next = 2), this._encryptBlock(this._zeroBlock))
                      case 2:
                        for (i = f.sent, o = new Uint8Array(16), a = i[0] >>> 6, s = 0; s < 15; s++)
                          ((o[s] = (i[s + 1] >> 6) | (i[s] << 2)),
                            (i[s] = (i[s + 1] >> 7) | (i[s] << 1)))
                        ;((x = [0, 135, 14, 137]),
                          (o[14] ^= a >>> 1),
                          (o[15] = (i[15] << 2) ^ x[a]),
                          (i[15] = (i[15] << 1) ^ x[a >> 1]),
                          (this._k1 = i),
                          (this._k2 = o))
                      case 12:
                      case 'end':
                        return f.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n() {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: '_encryptCTR',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o) {
              var a
              return De().wrap(
                function (x) {
                  for (;;)
                    switch ((x.prev = x.next)) {
                      case 0:
                        return (
                          (x.next = 2),
                          window.crypto.subtle.encrypt(
                            { name: 'AES-CTR', counter: o, length: 128 },
                            this._ctrKey,
                            i
                          )
                        )
                      case 2:
                        return ((a = x.sent), x.abrupt('return', new Uint8Array(a)))
                      case 4:
                      case 'end':
                        return x.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: '_decryptCTR',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o) {
              var a
              return De().wrap(
                function (x) {
                  for (;;)
                    switch ((x.prev = x.next)) {
                      case 0:
                        return (
                          (x.next = 2),
                          window.crypto.subtle.decrypt(
                            { name: 'AES-CTR', counter: o, length: 128 },
                            this._ctrKey,
                            i
                          )
                        )
                      case 2:
                        return ((a = x.sent), x.abrupt('return', new Uint8Array(a)))
                      case 4:
                      case 'end':
                        return x.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: '_computeCMAC',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o) {
              var a, s, x, u, f, F, X, g
              return De().wrap(
                function (_) {
                  for (;;)
                    switch ((_.prev = _.next)) {
                      case 0:
                        if (o.length === 16) {
                          _.next = 2
                          break
                        }
                        return _.abrupt('return', null)
                      case 2:
                        if (
                          ((a = Math.floor(i.length / 16)),
                          (s = Math.ceil(i.length / 16)),
                          (x = i.length - a * 16),
                          (u = new Uint8Array((s + 1) * 16)),
                          u.set(o),
                          u.set(i, 16),
                          x === 0)
                        )
                          for (f = 0; f < 16; f++) u[a * 16 + f] ^= this._k1[f]
                        else
                          for (u[(a + 1) * 16 + x] = 128, F = 0; F < 16; F++)
                            u[(a + 1) * 16 + F] ^= this._k2[F]
                        return (
                          (_.next = 11),
                          window.crypto.subtle.encrypt(
                            { name: 'AES-CBC', iv: this._zeroBlock },
                            this._cbcKey,
                            u
                          )
                        )
                      case 11:
                        return (
                          (X = _.sent),
                          (X = new Uint8Array(X)),
                          (g = X.slice(X.length - 32, X.length - 16)),
                          _.abrupt('return', g)
                        )
                      case 15:
                      case 'end':
                        return _.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: '_importKey',
        value: (function () {
          var t = ct(
            De().mark(function r(i) {
              return De().wrap(
                function (a) {
                  for (;;)
                    switch ((a.prev = a.next)) {
                      case 0:
                        return (
                          (this._rawKey = i),
                          (a.next = 3),
                          window.crypto.subtle.importKey('raw', i, { name: 'AES-CTR' }, !1, [
                            'encrypt',
                            'decrypt'
                          ])
                        )
                      case 3:
                        return (
                          (this._ctrKey = a.sent),
                          (a.next = 6),
                          window.crypto.subtle.importKey('raw', i, { name: 'AES-CBC' }, !1, [
                            'encrypt'
                          ])
                        )
                      case 6:
                        return ((this._cbcKey = a.sent), (a.next = 9), this._initCMAC())
                      case 9:
                      case 'end':
                        return a.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: 'encrypt',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o) {
              var a, s, x, u, f, F, X, g
              return De().wrap(
                function (_) {
                  for (;;)
                    switch ((_.prev = _.next)) {
                      case 0:
                        return (
                          (a = i.additionalData),
                          (s = i.iv),
                          (_.next = 4),
                          this._computeCMAC(s, this._prefixBlock0)
                        )
                      case 4:
                        return ((x = _.sent), (_.next = 7), this._encryptCTR(o, x))
                      case 7:
                        return (
                          (u = _.sent),
                          (_.next = 10),
                          this._computeCMAC(a, this._prefixBlock1)
                        )
                      case 10:
                        return (
                          (f = _.sent),
                          (_.next = 13),
                          this._computeCMAC(u, this._prefixBlock2)
                        )
                      case 13:
                        for (F = _.sent, X = 0; X < 16; X++) F[X] ^= x[X] ^ f[X]
                        return (
                          (g = new Uint8Array(16 + u.length)),
                          g.set(u),
                          g.set(F, u.length),
                          _.abrupt('return', g)
                        )
                      case 19:
                      case 'end':
                        return _.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: 'decrypt',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o) {
              var a, s, x, u, f, F, X, g, k, _
              return De().wrap(
                function (h) {
                  for (;;)
                    switch ((h.prev = h.next)) {
                      case 0:
                        return (
                          (a = o.slice(0, o.length - 16)),
                          (s = i.additionalData),
                          (x = i.iv),
                          (u = o.slice(o.length - 16)),
                          (h.next = 6),
                          this._computeCMAC(x, this._prefixBlock0)
                        )
                      case 6:
                        return (
                          (f = h.sent),
                          (h.next = 9),
                          this._computeCMAC(s, this._prefixBlock1)
                        )
                      case 9:
                        return (
                          (F = h.sent),
                          (h.next = 12),
                          this._computeCMAC(a, this._prefixBlock2)
                        )
                      case 12:
                        for (X = h.sent, g = 0; g < 16; g++) X[g] ^= f[g] ^ F[g]
                        if (X.length === u.length) {
                          h.next = 16
                          break
                        }
                        return h.abrupt('return', null)
                      case 16:
                        k = 0
                      case 17:
                        if (!(k < u.length)) {
                          h.next = 23
                          break
                        }
                        if (X[k] === u[k]) {
                          h.next = 20
                          break
                        }
                        return h.abrupt('return', null)
                      case 20:
                        ;(k++, (h.next = 17))
                        break
                      case 23:
                        return ((h.next = 25), this._decryptCTR(a, f))
                      case 25:
                        return ((_ = h.sent), h.abrupt('return', _))
                      case 27:
                      case 'end':
                        return h.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      }
    ],
    [
      {
        key: 'importKey',
        value: (function () {
          var t = ct(
            De().mark(function r(i, o, a, s) {
              var x
              return De().wrap(function (f) {
                for (;;)
                  switch ((f.prev = f.next)) {
                    case 0:
                      return ((x = new e()), (f.next = 3), x._importKey(i))
                    case 3:
                      return f.abrupt('return', x)
                    case 4:
                    case 'end':
                      return f.stop()
                  }
              }, r)
            })
          )
          function n(r, i, o, a) {
            return t.apply(this, arguments)
          }
          return n
        })()
      }
    ]
  )
})()
var or = {}
Object.defineProperty(or, '__esModule', { value: !0 })
or.DESECBCipher = or.DESCBCCipher = void 0
function wr(e) {
  '@babel/helpers - typeof'
  return (
    (wr =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    wr(e)
  )
}
function Wn(e, t) {
  if (!(e instanceof t)) throw new TypeError('Cannot call a class as a function')
}
function Zi(e, t) {
  for (var n = 0; n < t.length; n++) {
    var r = t[n]
    ;((r.enumerable = r.enumerable || !1),
      (r.configurable = !0),
      'value' in r && (r.writable = !0),
      Object.defineProperty(e, Ou(r.key), r))
  }
}
function zn(e, t, n) {
  return (
    t && Zi(e.prototype, t),
    n && Zi(e, n),
    Object.defineProperty(e, 'prototype', { writable: !1 }),
    e
  )
}
function Ou(e) {
  var t = Bu(e, 'string')
  return wr(t) == 'symbol' ? t : t + ''
}
function Bu(e, t) {
  if (wr(e) != 'object' || !e) return e
  var n = e[Symbol.toPrimitive]
  if (n !== void 0) {
    var r = n.call(e, t)
    if (wr(r) != 'object') return r
    throw new TypeError('@@toPrimitive must return a primitive value.')
  }
  return String(e)
}
var qi = [
    13, 16, 10, 23, 0, 4, 2, 27, 14, 5, 20, 9, 22, 18, 11, 3, 25, 7, 15, 6, 26, 19, 12, 1, 40, 51,
    30, 36, 46, 54, 29, 39, 50, 44, 32, 47, 43, 48, 38, 55, 33, 52, 45, 41, 49, 35, 28, 31
  ],
  Iu = [1, 2, 4, 6, 8, 10, 12, 14, 15, 17, 19, 21, 23, 25, 27, 28],
  P = 0,
  Q,
  U,
  W,
  N,
  j,
  z
Q = 65536
U = 1 << 24
W = Q | U
N = 4
j = 1024
z = N | j
var Yi = [
  W | j,
  P | P,
  Q | P,
  W | z,
  W | N,
  Q | z,
  P | N,
  Q | P,
  P | j,
  W | j,
  W | z,
  P | j,
  U | z,
  W | N,
  U | P,
  P | N,
  P | z,
  U | j,
  U | j,
  Q | j,
  Q | j,
  W | P,
  W | P,
  U | z,
  Q | N,
  U | N,
  U | N,
  Q | N,
  P | P,
  P | z,
  Q | z,
  U | P,
  Q | P,
  W | z,
  P | N,
  W | P,
  W | j,
  U | P,
  U | P,
  P | j,
  W | N,
  Q | P,
  Q | j,
  U | N,
  P | j,
  P | N,
  U | z,
  Q | z,
  W | z,
  Q | N,
  W | P,
  U | z,
  U | N,
  P | z,
  Q | z,
  W | j,
  P | z,
  U | j,
  U | j,
  P | P,
  Q | N,
  Q | j,
  P | P,
  W | N
]
Q = 1 << 20
U = 1 << 31
W = Q | U
N = 32
j = 32768
z = N | j
var $i = [
  W | z,
  U | j,
  P | j,
  Q | z,
  Q | P,
  P | N,
  W | N,
  U | z,
  U | N,
  W | z,
  W | j,
  U | P,
  U | j,
  Q | P,
  P | N,
  W | N,
  Q | j,
  Q | N,
  U | z,
  P | P,
  U | P,
  P | j,
  Q | z,
  W | P,
  Q | N,
  U | N,
  P | P,
  Q | j,
  P | z,
  W | j,
  W | P,
  P | z,
  P | P,
  Q | z,
  W | N,
  Q | P,
  U | z,
  W | P,
  W | j,
  P | j,
  W | P,
  U | j,
  P | N,
  W | z,
  Q | z,
  P | N,
  P | j,
  U | P,
  P | z,
  W | j,
  Q | P,
  U | N,
  Q | N,
  U | z,
  U | N,
  Q | N,
  Q | j,
  P | P,
  U | j,
  P | z,
  U | P,
  W | N,
  W | z,
  Q | j
]
Q = 1 << 17
U = 1 << 27
W = Q | U
N = 8
j = 512
z = N | j
var Ji = [
  P | z,
  W | j,
  P | P,
  W | N,
  U | j,
  P | P,
  Q | z,
  U | j,
  Q | N,
  U | N,
  U | N,
  Q | P,
  W | z,
  Q | N,
  W | P,
  P | z,
  U | P,
  P | N,
  W | j,
  P | j,
  Q | j,
  W | P,
  W | N,
  Q | z,
  U | z,
  Q | j,
  Q | P,
  U | z,
  P | N,
  W | z,
  P | j,
  U | P,
  W | j,
  U | P,
  Q | N,
  P | z,
  Q | P,
  W | j,
  U | j,
  P | P,
  P | j,
  Q | N,
  W | z,
  U | j,
  U | N,
  P | j,
  P | P,
  W | N,
  U | z,
  Q | P,
  U | P,
  W | z,
  P | N,
  Q | z,
  Q | j,
  U | N,
  W | P,
  U | z,
  P | z,
  W | P,
  Q | z,
  P | N,
  W | N,
  Q | j
]
Q = 8192
U = 1 << 23
W = Q | U
N = 1
j = 128
z = N | j
var ea = [
  W | N,
  Q | z,
  Q | z,
  P | j,
  W | j,
  U | z,
  U | N,
  Q | N,
  P | P,
  W | P,
  W | P,
  W | z,
  P | z,
  P | P,
  U | j,
  U | N,
  P | N,
  Q | P,
  U | P,
  W | N,
  P | j,
  U | P,
  Q | N,
  Q | j,
  U | z,
  P | N,
  Q | j,
  U | j,
  Q | P,
  W | j,
  W | z,
  P | z,
  U | j,
  U | N,
  W | P,
  W | z,
  P | z,
  P | P,
  P | P,
  W | P,
  Q | j,
  U | j,
  U | z,
  P | N,
  W | N,
  Q | z,
  Q | z,
  P | j,
  W | z,
  P | z,
  P | N,
  Q | P,
  U | N,
  Q | N,
  W | j,
  U | z,
  Q | N,
  Q | j,
  U | P,
  W | N,
  P | j,
  U | P,
  Q | P,
  W | j
]
Q = 1 << 25
U = 1 << 30
W = Q | U
N = 256
j = 1 << 19
z = N | j
var ta = [
  P | N,
  Q | z,
  Q | j,
  W | N,
  P | j,
  P | N,
  U | P,
  Q | j,
  U | z,
  P | j,
  Q | N,
  U | z,
  W | N,
  W | j,
  P | z,
  U | P,
  Q | P,
  U | j,
  U | j,
  P | P,
  U | N,
  W | z,
  W | z,
  Q | N,
  W | j,
  U | N,
  P | P,
  W | P,
  Q | z,
  Q | P,
  W | P,
  P | z,
  P | j,
  W | N,
  P | N,
  Q | P,
  U | P,
  Q | j,
  W | N,
  U | z,
  Q | N,
  U | P,
  W | j,
  Q | z,
  U | z,
  P | N,
  Q | P,
  W | j,
  W | z,
  P | z,
  W | P,
  W | z,
  Q | j,
  P | P,
  U | j,
  W | P,
  P | z,
  Q | N,
  U | N,
  P | j,
  P | P,
  U | j,
  Q | z,
  U | N
]
Q = 1 << 22
U = 1 << 29
W = Q | U
N = 16
j = 16384
z = N | j
var ra = [
  U | N,
  W | P,
  P | j,
  W | z,
  W | P,
  P | N,
  W | z,
  Q | P,
  U | j,
  Q | z,
  Q | P,
  U | N,
  Q | N,
  U | j,
  U | P,
  P | z,
  P | P,
  Q | N,
  U | z,
  P | j,
  Q | j,
  U | z,
  P | N,
  W | N,
  W | N,
  P | P,
  Q | z,
  W | j,
  P | z,
  Q | j,
  W | j,
  U | P,
  U | j,
  P | N,
  W | N,
  Q | j,
  W | z,
  Q | P,
  P | z,
  U | N,
  Q | P,
  U | j,
  U | P,
  P | z,
  U | N,
  W | z,
  Q | j,
  W | P,
  Q | z,
  W | j,
  P | P,
  W | N,
  P | N,
  P | j,
  W | P,
  Q | z,
  P | j,
  Q | N,
  U | z,
  P | P,
  W | j,
  U | P,
  Q | N,
  U | z
]
Q = 1 << 21
U = 1 << 26
W = Q | U
N = 2
j = 2048
z = N | j
var na = [
  Q | P,
  W | N,
  U | z,
  P | P,
  P | j,
  U | z,
  Q | z,
  W | j,
  W | z,
  Q | P,
  P | P,
  U | N,
  P | N,
  U | P,
  W | N,
  P | z,
  U | j,
  Q | z,
  Q | N,
  U | j,
  U | N,
  W | P,
  W | j,
  Q | N,
  W | P,
  P | j,
  P | z,
  W | z,
  Q | j,
  P | N,
  U | P,
  Q | j,
  U | P,
  Q | j,
  Q | P,
  U | z,
  U | z,
  W | N,
  W | N,
  P | N,
  Q | N,
  U | P,
  U | j,
  Q | P,
  W | j,
  P | z,
  Q | z,
  W | j,
  P | z,
  U | N,
  W | z,
  W | P,
  Q | j,
  P | P,
  P | N,
  W | z,
  P | P,
  Q | z,
  W | P,
  P | j,
  U | N,
  U | j,
  P | j,
  Q | N
]
Q = 1 << 18
U = 1 << 28
W = Q | U
N = 64
j = 4096
z = N | j
var ia = [
    U | z,
    P | j,
    Q | P,
    W | z,
    U | P,
    U | z,
    P | N,
    U | P,
    Q | N,
    W | P,
    W | z,
    Q | j,
    W | j,
    Q | z,
    P | j,
    P | N,
    W | P,
    U | N,
    U | j,
    P | z,
    Q | j,
    Q | N,
    W | N,
    W | j,
    P | z,
    P | P,
    P | P,
    W | N,
    U | N,
    U | j,
    Q | z,
    Q | P,
    Q | z,
    Q | P,
    W | j,
    P | j,
    P | N,
    W | N,
    P | j,
    Q | z,
    U | j,
    P | N,
    U | N,
    W | P,
    W | N,
    U | P,
    Q | P,
    U | z,
    P | P,
    W | z,
    Q | N,
    U | N,
    W | P,
    U | j,
    U | z,
    P | P,
    W | z,
    Q | j,
    Q | j,
    P | z,
    P | z,
    Q | N,
    U | P,
    W | j
  ],
  go = (function () {
    function e(t) {
      ;(Wn(this, e), (this.keys = []))
      for (var n = [], r = [], i = [], o = 0, a = 56; o < 56; ++o, a -= 8) {
        a += a < -5 ? 65 : a < -3 ? 31 : a < -1 ? 63 : a === 27 ? 35 : 0
        var s = a & 7
        n[o] = t[a >>> 3] & (1 << s) ? 1 : 0
      }
      for (var x = 0; x < 16; ++x) {
        var u = x << 1,
          f = u + 1
        i[u] = i[f] = 0
        for (var F = 28; F < 59; F += 28)
          for (var X = F - 28; X < F; ++X) {
            var g = X + Iu[x]
            r[X] = g < F ? n[g] : n[g - 28]
          }
        for (var k = 0; k < 24; ++k)
          (r[qi[k]] !== 0 && (i[u] |= 1 << (23 - k)),
            r[qi[k + 24]] !== 0 && (i[f] |= 1 << (23 - k)))
      }
      for (var _ = 0, l = 0, h = 0; _ < 16; ++_) {
        var c = i[l++],
          E = i[l++]
        ;((this.keys[h] = (c & 16515072) << 6),
          (this.keys[h] |= (c & 4032) << 10),
          (this.keys[h] |= (E & 16515072) >>> 10),
          (this.keys[h] |= (E & 4032) >>> 6),
          ++h,
          (this.keys[h] = (c & 258048) << 12),
          (this.keys[h] |= (c & 63) << 16),
          (this.keys[h] |= (E & 258048) >>> 4),
          (this.keys[h] |= E & 63),
          ++h)
      }
    }
    return zn(e, [
      {
        key: 'enc8',
        value: function (n) {
          var r = n.slice(),
            i = 0,
            o,
            a,
            s
          ;((o = (r[i++] << 24) | (r[i++] << 16) | (r[i++] << 8) | r[i++]),
            (a = (r[i++] << 24) | (r[i++] << 16) | (r[i++] << 8) | r[i++]),
            (s = ((o >>> 4) ^ a) & 252645135),
            (a ^= s),
            (o ^= s << 4),
            (s = ((o >>> 16) ^ a) & 65535),
            (a ^= s),
            (o ^= s << 16),
            (s = ((a >>> 2) ^ o) & 858993459),
            (o ^= s),
            (a ^= s << 2),
            (s = ((a >>> 8) ^ o) & 16711935),
            (o ^= s),
            (a ^= s << 8),
            (a = (a << 1) | ((a >>> 31) & 1)),
            (s = (o ^ a) & 2863311530),
            (o ^= s),
            (a ^= s),
            (o = (o << 1) | ((o >>> 31) & 1)))
          for (var x = 0, u = 0; x < 8; ++x) {
            ;((s = (a << 28) | (a >>> 4)), (s ^= this.keys[u++]))
            var f = na[s & 63]
            ;((f |= ta[(s >>> 8) & 63]),
              (f |= Ji[(s >>> 16) & 63]),
              (f |= Yi[(s >>> 24) & 63]),
              (s = a ^ this.keys[u++]),
              (f |= ia[s & 63]),
              (f |= ra[(s >>> 8) & 63]),
              (f |= ea[(s >>> 16) & 63]),
              (f |= $i[(s >>> 24) & 63]),
              (o ^= f),
              (s = (o << 28) | (o >>> 4)),
              (s ^= this.keys[u++]),
              (f = na[s & 63]),
              (f |= ta[(s >>> 8) & 63]),
              (f |= Ji[(s >>> 16) & 63]),
              (f |= Yi[(s >>> 24) & 63]),
              (s = o ^ this.keys[u++]),
              (f |= ia[s & 63]),
              (f |= ra[(s >>> 8) & 63]),
              (f |= ea[(s >>> 16) & 63]),
              (f |= $i[(s >>> 24) & 63]),
              (a ^= f))
          }
          for (
            a = (a << 31) | (a >>> 1),
              s = (o ^ a) & 2863311530,
              o ^= s,
              a ^= s,
              o = (o << 31) | (o >>> 1),
              s = ((o >>> 8) ^ a) & 16711935,
              a ^= s,
              o ^= s << 8,
              s = ((o >>> 2) ^ a) & 858993459,
              a ^= s,
              o ^= s << 2,
              s = ((a >>> 16) ^ o) & 65535,
              o ^= s,
              a ^= s << 16,
              s = ((a >>> 4) ^ o) & 252645135,
              o ^= s,
              a ^= s << 4,
              s = [a, o],
              i = 0;
            i < 8;
            i++
          )
            ((r[i] = (s[i >>> 2] >>> (8 * (3 - (i % 4)))) % 256), r[i] < 0 && (r[i] += 256))
          return r
        }
      }
    ])
  })()
or.DESECBCipher = (function () {
  function e() {
    ;(Wn(this, e), (this._cipher = null))
  }
  return zn(
    e,
    [
      {
        key: 'algorithm',
        get: function () {
          return { name: 'DES-ECB' }
        }
      },
      {
        key: '_importKey',
        value: function (n, r, i) {
          this._cipher = new go(n)
        }
      },
      {
        key: 'encrypt',
        value: function (n, r) {
          var i = new Uint8Array(r)
          if (i.length % 8 !== 0 || this._cipher === null) return null
          for (var o = i.length / 8, a = 0; a < o; a++)
            i.set(this._cipher.enc8(i.slice(a * 8, a * 8 + 8)), a * 8)
          return i
        }
      }
    ],
    [
      {
        key: 'importKey',
        value: function (n, r, i, o) {
          var a = new e()
          return (a._importKey(n), a)
        }
      }
    ]
  )
})()
or.DESCBCCipher = (function () {
  function e() {
    ;(Wn(this, e), (this._cipher = null))
  }
  return zn(
    e,
    [
      {
        key: 'algorithm',
        get: function () {
          return { name: 'DES-CBC' }
        }
      },
      {
        key: '_importKey',
        value: function (n) {
          this._cipher = new go(n)
        }
      },
      {
        key: 'encrypt',
        value: function (n, r) {
          var i = new Uint8Array(r),
            o = new Uint8Array(n.iv)
          if (i.length % 8 !== 0 || this._cipher === null) return null
          for (var a = i.length / 8, s = 0; s < a; s++) {
            for (var x = 0; x < 8; x++) o[x] ^= r[s * 8 + x]
            ;((o = this._cipher.enc8(o)), i.set(o, s * 8))
          }
          return i
        }
      }
    ],
    [
      {
        key: 'importKey',
        value: function (n, r, i, o) {
          var a = new e()
          return (a._importKey(n), a)
        }
      }
    ]
  )
})()
var Jr = {},
  lr = {}
Object.defineProperty(lr, '__esModule', { value: !0 })
lr.bigIntToU8Array = Uu
lr.modPow = Qu
lr.u8ArrayToBigInt = Nu
function Qu(e, t, n) {
  var r = 1n
  for (e = e % n; t > 0n; ) ((t & 1n) === 1n && (r = (r * e) % n), (t = t >> 1n), (e = (e * e) % n))
  return r
}
function Uu(e) {
  var t = arguments.length > 1 && arguments[1] !== void 0 ? arguments[1] : 0,
    n = e.toString(16)
  ;(t === 0 && (t = Math.ceil(n.length / 2)), (n = n.padStart(t * 2, '0')))
  for (var r = n.length / 2, i = new Uint8Array(r), o = 0; o < r; o++)
    i[o] = parseInt(n.slice(o * 2, o * 2 + 2), 16)
  return i
}
function Nu(e) {
  for (var t = '0x', n = 0; n < e.length; n++) t += e[n].toString(16).padStart(2, '0')
  return BigInt(t)
}
Object.defineProperty(Jr, '__esModule', { value: !0 })
Jr.RSACipher = void 0
var ju = Hu(Zr),
  pt = lr
function Hu(e) {
  return e && e.__esModule ? e : { default: e }
}
function $t(e) {
  '@babel/helpers - typeof'
  return (
    ($t =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    $t(e)
  )
}
function et() {
  /*! regenerator-runtime -- Copyright (c) 2014-present, Facebook, Inc. -- license (MIT): https://github.com/facebook/regenerator/blob/main/LICENSE */ et =
    function () {
      return t
    }
  var e,
    t = {},
    n = Object.prototype,
    r = n.hasOwnProperty,
    i =
      Object.defineProperty ||
      function (S, p, b) {
        S[p] = b.value
      },
    o = typeof Symbol == 'function' ? Symbol : {},
    a = o.iterator || '@@iterator',
    s = o.asyncIterator || '@@asyncIterator',
    x = o.toStringTag || '@@toStringTag'
  function u(S, p, b) {
    return (
      Object.defineProperty(S, p, { value: b, enumerable: !0, configurable: !0, writable: !0 }),
      S[p]
    )
  }
  try {
    u({}, '')
  } catch {
    u = function (b, L, B) {
      return (b[L] = B)
    }
  }
  function f(S, p, b, L) {
    var B = p && p.prototype instanceof h ? p : h,
      M = Object.create(B.prototype),
      V = new $(L || [])
    return (i(M, '_invoke', { value: O(S, b, V) }), M)
  }
  function F(S, p, b) {
    try {
      return { type: 'normal', arg: S.call(p, b) }
    } catch (L) {
      return { type: 'throw', arg: L }
    }
  }
  t.wrap = f
  var X = 'suspendedStart',
    g = 'suspendedYield',
    k = 'executing',
    _ = 'completed',
    l = {}
  function h() {}
  function c() {}
  function E() {}
  var y = {}
  u(y, a, function () {
    return this
  })
  var K = Object.getPrototypeOf,
    w = K && K(K(ne([])))
  w && w !== n && r.call(w, a) && (y = w)
  var m = (E.prototype = h.prototype = Object.create(y))
  function C(S) {
    ;['next', 'throw', 'return'].forEach(function (p) {
      u(S, p, function (b) {
        return this._invoke(p, b)
      })
    })
  }
  function A(S, p) {
    function b(B, M, V, re) {
      var oe = F(S[B], S, M)
      if (oe.type !== 'throw') {
        var ce = oe.arg,
          pe = ce.value
        return pe && $t(pe) == 'object' && r.call(pe, '__await')
          ? p.resolve(pe.__await).then(
              function (ye) {
                b('next', ye, V, re)
              },
              function (ye) {
                b('throw', ye, V, re)
              }
            )
          : p.resolve(pe).then(
              function (ye) {
                ;((ce.value = ye), V(ce))
              },
              function (ye) {
                return b('throw', ye, V, re)
              }
            )
      }
      re(oe.arg)
    }
    var L
    i(this, '_invoke', {
      value: function (M, V) {
        function re() {
          return new p(function (oe, ce) {
            b(M, V, oe, ce)
          })
        }
        return (L = L ? L.then(re, re) : re())
      }
    })
  }
  function O(S, p, b) {
    var L = X
    return function (B, M) {
      if (L === k) throw Error('Generator is already running')
      if (L === _) {
        if (B === 'throw') throw M
        return { value: e, done: !0 }
      }
      for (b.method = B, b.arg = M; ; ) {
        var V = b.delegate
        if (V) {
          var re = I(V, b)
          if (re) {
            if (re === l) continue
            return re
          }
        }
        if (b.method === 'next') b.sent = b._sent = b.arg
        else if (b.method === 'throw') {
          if (L === X) throw ((L = _), b.arg)
          b.dispatchException(b.arg)
        } else b.method === 'return' && b.abrupt('return', b.arg)
        L = k
        var oe = F(S, p, b)
        if (oe.type === 'normal') {
          if (((L = b.done ? _ : g), oe.arg === l)) continue
          return { value: oe.arg, done: b.done }
        }
        oe.type === 'throw' && ((L = _), (b.method = 'throw'), (b.arg = oe.arg))
      }
    }
  }
  function I(S, p) {
    var b = p.method,
      L = S.iterator[b]
    if (L === e)
      return (
        (p.delegate = null),
        (b === 'throw' &&
          S.iterator.return &&
          ((p.method = 'return'), (p.arg = e), I(S, p), p.method === 'throw')) ||
          (b !== 'return' &&
            ((p.method = 'throw'),
            (p.arg = new TypeError("The iterator does not provide a '" + b + "' method")))),
        l
      )
    var B = F(L, S.iterator, p.arg)
    if (B.type === 'throw') return ((p.method = 'throw'), (p.arg = B.arg), (p.delegate = null), l)
    var M = B.arg
    return M
      ? M.done
        ? ((p[S.resultName] = M.value),
          (p.next = S.nextLoc),
          p.method !== 'return' && ((p.method = 'next'), (p.arg = e)),
          (p.delegate = null),
          l)
        : M
      : ((p.method = 'throw'),
        (p.arg = new TypeError('iterator result is not an object')),
        (p.delegate = null),
        l)
  }
  function Z(S) {
    var p = { tryLoc: S[0] }
    ;(1 in S && (p.catchLoc = S[1]),
      2 in S && ((p.finallyLoc = S[2]), (p.afterLoc = S[3])),
      this.tryEntries.push(p))
  }
  function q(S) {
    var p = S.completion || {}
    ;((p.type = 'normal'), delete p.arg, (S.completion = p))
  }
  function $(S) {
    ;((this.tryEntries = [{ tryLoc: 'root' }]), S.forEach(Z, this), this.reset(!0))
  }
  function ne(S) {
    if (S || S === '') {
      var p = S[a]
      if (p) return p.call(S)
      if (typeof S.next == 'function') return S
      if (!isNaN(S.length)) {
        var b = -1,
          L = function B() {
            for (; ++b < S.length; ) if (r.call(S, b)) return ((B.value = S[b]), (B.done = !1), B)
            return ((B.value = e), (B.done = !0), B)
          }
        return (L.next = L)
      }
    }
    throw new TypeError($t(S) + ' is not iterable')
  }
  return (
    (c.prototype = E),
    i(m, 'constructor', { value: E, configurable: !0 }),
    i(E, 'constructor', { value: c, configurable: !0 }),
    (c.displayName = u(E, x, 'GeneratorFunction')),
    (t.isGeneratorFunction = function (S) {
      var p = typeof S == 'function' && S.constructor
      return !!p && (p === c || (p.displayName || p.name) === 'GeneratorFunction')
    }),
    (t.mark = function (S) {
      return (
        Object.setPrototypeOf
          ? Object.setPrototypeOf(S, E)
          : ((S.__proto__ = E), u(S, x, 'GeneratorFunction')),
        (S.prototype = Object.create(m)),
        S
      )
    }),
    (t.awrap = function (S) {
      return { __await: S }
    }),
    C(A.prototype),
    u(A.prototype, s, function () {
      return this
    }),
    (t.AsyncIterator = A),
    (t.async = function (S, p, b, L, B) {
      B === void 0 && (B = Promise)
      var M = new A(f(S, p, b, L), B)
      return t.isGeneratorFunction(p)
        ? M
        : M.next().then(function (V) {
            return V.done ? V.value : M.next()
          })
    }),
    C(m),
    u(m, x, 'Generator'),
    u(m, a, function () {
      return this
    }),
    u(m, 'toString', function () {
      return '[object Generator]'
    }),
    (t.keys = function (S) {
      var p = Object(S),
        b = []
      for (var L in p) b.push(L)
      return (
        b.reverse(),
        function B() {
          for (; b.length; ) {
            var M = b.pop()
            if (M in p) return ((B.value = M), (B.done = !1), B)
          }
          return ((B.done = !0), B)
        }
      )
    }),
    (t.values = ne),
    ($.prototype = {
      constructor: $,
      reset: function (p) {
        if (
          ((this.prev = 0),
          (this.next = 0),
          (this.sent = this._sent = e),
          (this.done = !1),
          (this.delegate = null),
          (this.method = 'next'),
          (this.arg = e),
          this.tryEntries.forEach(q),
          !p)
        )
          for (var b in this)
            b.charAt(0) === 't' && r.call(this, b) && !isNaN(+b.slice(1)) && (this[b] = e)
      },
      stop: function () {
        this.done = !0
        var p = this.tryEntries[0].completion
        if (p.type === 'throw') throw p.arg
        return this.rval
      },
      dispatchException: function (p) {
        if (this.done) throw p
        var b = this
        function L(ce, pe) {
          return (
            (V.type = 'throw'),
            (V.arg = p),
            (b.next = ce),
            pe && ((b.method = 'next'), (b.arg = e)),
            !!pe
          )
        }
        for (var B = this.tryEntries.length - 1; B >= 0; --B) {
          var M = this.tryEntries[B],
            V = M.completion
          if (M.tryLoc === 'root') return L('end')
          if (M.tryLoc <= this.prev) {
            var re = r.call(M, 'catchLoc'),
              oe = r.call(M, 'finallyLoc')
            if (re && oe) {
              if (this.prev < M.catchLoc) return L(M.catchLoc, !0)
              if (this.prev < M.finallyLoc) return L(M.finallyLoc)
            } else if (re) {
              if (this.prev < M.catchLoc) return L(M.catchLoc, !0)
            } else {
              if (!oe) throw Error('try statement without catch or finally')
              if (this.prev < M.finallyLoc) return L(M.finallyLoc)
            }
          }
        }
      },
      abrupt: function (p, b) {
        for (var L = this.tryEntries.length - 1; L >= 0; --L) {
          var B = this.tryEntries[L]
          if (B.tryLoc <= this.prev && r.call(B, 'finallyLoc') && this.prev < B.finallyLoc) {
            var M = B
            break
          }
        }
        M && (p === 'break' || p === 'continue') && M.tryLoc <= b && b <= M.finallyLoc && (M = null)
        var V = M ? M.completion : {}
        return (
          (V.type = p),
          (V.arg = b),
          M ? ((this.method = 'next'), (this.next = M.finallyLoc), l) : this.complete(V)
        )
      },
      complete: function (p, b) {
        if (p.type === 'throw') throw p.arg
        return (
          p.type === 'break' || p.type === 'continue'
            ? (this.next = p.arg)
            : p.type === 'return'
              ? ((this.rval = this.arg = p.arg), (this.method = 'return'), (this.next = 'end'))
              : p.type === 'normal' && b && (this.next = b),
          l
        )
      },
      finish: function (p) {
        for (var b = this.tryEntries.length - 1; b >= 0; --b) {
          var L = this.tryEntries[b]
          if (L.finallyLoc === p) return (this.complete(L.completion, L.afterLoc), q(L), l)
        }
      },
      catch: function (p) {
        for (var b = this.tryEntries.length - 1; b >= 0; --b) {
          var L = this.tryEntries[b]
          if (L.tryLoc === p) {
            var B = L.completion
            if (B.type === 'throw') {
              var M = B.arg
              q(L)
            }
            return M
          }
        }
        throw Error('illegal catch attempt')
      },
      delegateYield: function (p, b, L) {
        return (
          (this.delegate = { iterator: ne(p), resultName: b, nextLoc: L }),
          this.method === 'next' && (this.arg = e),
          l
        )
      }
    }),
    t
  )
}
function aa(e, t, n, r, i, o, a) {
  try {
    var s = e[o](a),
      x = s.value
  } catch (u) {
    return void n(u)
  }
  s.done ? t(x) : Promise.resolve(x).then(r, i)
}
function Ht(e) {
  return function () {
    var t = this,
      n = arguments
    return new Promise(function (r, i) {
      var o = e.apply(t, n)
      function a(x) {
        aa(o, r, i, a, s, 'next', x)
      }
      function s(x) {
        aa(o, r, i, a, s, 'throw', x)
      }
      a(void 0)
    })
  }
}
function Wu(e, t) {
  if (!(e instanceof t)) throw new TypeError('Cannot call a class as a function')
}
function oa(e, t) {
  for (var n = 0; n < t.length; n++) {
    var r = t[n]
    ;((r.enumerable = r.enumerable || !1),
      (r.configurable = !0),
      'value' in r && (r.writable = !0),
      Object.defineProperty(e, Gu(r.key), r))
  }
}
function zu(e, t, n) {
  return (
    t && oa(e.prototype, t),
    n && oa(e, n),
    Object.defineProperty(e, 'prototype', { writable: !1 }),
    e
  )
}
function Gu(e) {
  var t = Vu(e, 'string')
  return $t(t) == 'symbol' ? t : t + ''
}
function Vu(e, t) {
  if ($t(e) != 'object' || !e) return e
  var n = e[Symbol.toPrimitive]
  if (n !== void 0) {
    var r = n.call(e, t)
    if ($t(r) != 'object') return r
    throw new TypeError('@@toPrimitive must return a primitive value.')
  }
  return String(e)
}
Jr.RSACipher = (function () {
  function e() {
    ;(Wu(this, e),
      (this._keyLength = 0),
      (this._keyBytes = 0),
      (this._n = null),
      (this._e = null),
      (this._d = null),
      (this._nBigInt = null),
      (this._eBigInt = null),
      (this._dBigInt = null),
      (this._extractable = !1))
  }
  return zu(
    e,
    [
      {
        key: 'algorithm',
        get: function () {
          return { name: 'RSA-PKCS1-v1_5' }
        }
      },
      {
        key: '_base64urlDecode',
        value: function (n) {
          return (
            (n = n.replace(/-/g, '+').replace(/_/g, '/')),
            (n = n.padEnd(Math.ceil(n.length / 4) * 4, '=')),
            ju.default.decode(n)
          )
        }
      },
      {
        key: '_padArray',
        value: function (n, r) {
          var i = new Uint8Array(r)
          return (i.set(n, r - n.length), i)
        }
      },
      {
        key: '_generateKey',
        value: (function () {
          var t = Ht(
            et().mark(function r(i, o) {
              var a, s
              return et().wrap(
                function (u) {
                  for (;;)
                    switch ((u.prev = u.next)) {
                      case 0:
                        return (
                          (this._keyLength = i.modulusLength),
                          (this._keyBytes = Math.ceil(this._keyLength / 8)),
                          (u.next = 4),
                          window.crypto.subtle.generateKey(
                            {
                              name: 'RSA-OAEP',
                              modulusLength: i.modulusLength,
                              publicExponent: i.publicExponent,
                              hash: { name: 'SHA-256' }
                            },
                            !0,
                            ['encrypt', 'decrypt']
                          )
                        )
                      case 4:
                        return (
                          (a = u.sent),
                          (u.next = 7),
                          window.crypto.subtle.exportKey('jwk', a.privateKey)
                        )
                      case 7:
                        ;((s = u.sent),
                          (this._n = this._padArray(this._base64urlDecode(s.n), this._keyBytes)),
                          (this._nBigInt = (0, pt.u8ArrayToBigInt)(this._n)),
                          (this._e = this._padArray(this._base64urlDecode(s.e), this._keyBytes)),
                          (this._eBigInt = (0, pt.u8ArrayToBigInt)(this._e)),
                          (this._d = this._padArray(this._base64urlDecode(s.d), this._keyBytes)),
                          (this._dBigInt = (0, pt.u8ArrayToBigInt)(this._d)),
                          (this._extractable = o))
                      case 15:
                      case 'end':
                        return u.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: '_importKey',
        value: (function () {
          var t = Ht(
            et().mark(function r(i, o) {
              var a, s
              return et().wrap(
                function (u) {
                  for (;;)
                    switch ((u.prev = u.next)) {
                      case 0:
                        if (((a = i.n), (s = i.e), a.length === s.length)) {
                          u.next = 4
                          break
                        }
                        throw new Error('the sizes of modulus and public exponent do not match')
                      case 4:
                        ;((this._keyBytes = a.length),
                          (this._keyLength = this._keyBytes * 8),
                          (this._n = new Uint8Array(this._keyBytes)),
                          (this._e = new Uint8Array(this._keyBytes)),
                          this._n.set(a),
                          this._e.set(s),
                          (this._nBigInt = (0, pt.u8ArrayToBigInt)(this._n)),
                          (this._eBigInt = (0, pt.u8ArrayToBigInt)(this._e)),
                          (this._extractable = o))
                      case 13:
                      case 'end':
                        return u.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: 'encrypt',
        value: (function () {
          var t = Ht(
            et().mark(function r(i, o) {
              var a, s, x, u, f
              return et().wrap(
                function (X) {
                  for (;;)
                    switch ((X.prev = X.next)) {
                      case 0:
                        if (!(o.length > this._keyBytes - 11)) {
                          X.next = 2
                          break
                        }
                        return X.abrupt('return', null)
                      case 2:
                        for (
                          a = new Uint8Array(this._keyBytes - o.length - 3),
                            window.crypto.getRandomValues(a),
                            s = 0;
                          s < a.length;
                          s++
                        )
                          a[s] = Math.floor((a[s] * 254) / 255 + 1)
                        return (
                          (x = new Uint8Array(this._keyBytes)),
                          (x[1] = 2),
                          x.set(a, 2),
                          x.set(o, a.length + 3),
                          (u = (0, pt.u8ArrayToBigInt)(x)),
                          (f = (0, pt.modPow)(u, this._eBigInt, this._nBigInt)),
                          X.abrupt('return', (0, pt.bigIntToU8Array)(f, this._keyBytes))
                        )
                      case 12:
                      case 'end':
                        return X.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: 'decrypt',
        value: (function () {
          var t = Ht(
            et().mark(function r(i, o) {
              var a, s, x, u
              return et().wrap(
                function (F) {
                  for (;;)
                    switch ((F.prev = F.next)) {
                      case 0:
                        if (o.length === this._keyBytes) {
                          F.next = 2
                          break
                        }
                        return F.abrupt('return', null)
                      case 2:
                        if (
                          ((a = (0, pt.u8ArrayToBigInt)(o)),
                          (s = (0, pt.modPow)(a, this._dBigInt, this._nBigInt)),
                          (x = (0, pt.bigIntToU8Array)(s, this._keyBytes)),
                          !(x[0] !== 0 || x[1] !== 2))
                        ) {
                          F.next = 7
                          break
                        }
                        return F.abrupt('return', null)
                      case 7:
                        u = 2
                      case 8:
                        if (!(u < x.length)) {
                          F.next = 14
                          break
                        }
                        if (x[u] !== 0) {
                          F.next = 11
                          break
                        }
                        return F.abrupt('break', 14)
                      case 11:
                        ;(u++, (F.next = 8))
                        break
                      case 14:
                        if (u !== x.length) {
                          F.next = 16
                          break
                        }
                        return F.abrupt('return', null)
                      case 16:
                        return F.abrupt('return', x.slice(u + 1, x.length))
                      case 17:
                      case 'end':
                        return F.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n(r, i) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: 'exportKey',
        value: (function () {
          var t = Ht(
            et().mark(function r() {
              return et().wrap(
                function (o) {
                  for (;;)
                    switch ((o.prev = o.next)) {
                      case 0:
                        if (this._extractable) {
                          o.next = 2
                          break
                        }
                        throw new Error('key is not extractable')
                      case 2:
                        return o.abrupt('return', { n: this._n, e: this._e, d: this._d })
                      case 3:
                      case 'end':
                        return o.stop()
                    }
                },
                r,
                this
              )
            })
          )
          function n() {
            return t.apply(this, arguments)
          }
          return n
        })()
      }
    ],
    [
      {
        key: 'generateKey',
        value: (function () {
          var t = Ht(
            et().mark(function r(i, o, a) {
              var s
              return et().wrap(function (u) {
                for (;;)
                  switch ((u.prev = u.next)) {
                    case 0:
                      return ((s = new e()), (u.next = 3), s._generateKey(i, o))
                    case 3:
                      return u.abrupt('return', { privateKey: s })
                    case 4:
                    case 'end':
                      return u.stop()
                  }
              }, r)
            })
          )
          function n(r, i, o) {
            return t.apply(this, arguments)
          }
          return n
        })()
      },
      {
        key: 'importKey',
        value: (function () {
          var t = Ht(
            et().mark(function r(i, o, a, s) {
              var x
              return et().wrap(function (f) {
                for (;;)
                  switch ((f.prev = f.next)) {
                    case 0:
                      if (!(s.length !== 1 || s[0] !== 'encrypt')) {
                        f.next = 2
                        break
                      }
                      throw new Error('only support importing RSA public key')
                    case 2:
                      return ((x = new e()), (f.next = 5), x._importKey(i, a))
                    case 5:
                      return f.abrupt('return', x)
                    case 6:
                    case 'end':
                      return f.stop()
                  }
              }, r)
            })
          )
          function n(r, i, o, a) {
            return t.apply(this, arguments)
          }
          return n
        })()
      }
    ]
  )
})()
var en = {}
Object.defineProperty(en, '__esModule', { value: !0 })
en.DHCipher = void 0
var Tt = lr
function kr(e) {
  '@babel/helpers - typeof'
  return (
    (kr =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    kr(e)
  )
}
function bo(e, t) {
  if (!(e instanceof t)) throw new TypeError('Cannot call a class as a function')
}
function sa(e, t) {
  for (var n = 0; n < t.length; n++) {
    var r = t[n]
    ;((r.enumerable = r.enumerable || !1),
      (r.configurable = !0),
      'value' in r && (r.writable = !0),
      Object.defineProperty(e, Zu(r.key), r))
  }
}
function mo(e, t, n) {
  return (
    t && sa(e.prototype, t),
    n && sa(e, n),
    Object.defineProperty(e, 'prototype', { writable: !1 }),
    e
  )
}
function Zu(e) {
  var t = qu(e, 'string')
  return kr(t) == 'symbol' ? t : t + ''
}
function qu(e, t) {
  if (kr(e) != 'object' || !e) return e
  var n = e[Symbol.toPrimitive]
  if (n !== void 0) {
    var r = n.call(e, t)
    if (kr(r) != 'object') return r
    throw new TypeError('@@toPrimitive must return a primitive value.')
  }
  return String(e)
}
var Yu = (function () {
  function e(t) {
    ;(bo(this, e), (this._key = t))
  }
  return mo(e, [
    {
      key: 'algorithm',
      get: function () {
        return { name: 'DH' }
      }
    },
    {
      key: 'exportKey',
      value: function () {
        return this._key
      }
    }
  ])
})()
en.DHCipher = (function () {
  function e() {
    ;(bo(this, e),
      (this._g = null),
      (this._p = null),
      (this._gBigInt = null),
      (this._pBigInt = null),
      (this._privateKey = null))
  }
  return mo(
    e,
    [
      {
        key: 'algorithm',
        get: function () {
          return { name: 'DH' }
        }
      },
      {
        key: '_generateKey',
        value: function (n) {
          var r = n.g,
            i = n.p
          ;((this._keyBytes = i.length),
            (this._gBigInt = (0, Tt.u8ArrayToBigInt)(r)),
            (this._pBigInt = (0, Tt.u8ArrayToBigInt)(i)),
            (this._privateKey = window.crypto.getRandomValues(new Uint8Array(this._keyBytes))),
            (this._privateKeyBigInt = (0, Tt.u8ArrayToBigInt)(this._privateKey)),
            (this._publicKey = (0, Tt.bigIntToU8Array)(
              (0, Tt.modPow)(this._gBigInt, this._privateKeyBigInt, this._pBigInt),
              this._keyBytes
            )))
        }
      },
      {
        key: 'deriveBits',
        value: function (n, r) {
          var i = Math.ceil(r / 8),
            o = new Uint8Array(n.public),
            a = i > this._keyBytes ? i : this._keyBytes,
            s = (0, Tt.modPow)((0, Tt.u8ArrayToBigInt)(o), this._privateKeyBigInt, this._pBigInt)
          return (0, Tt.bigIntToU8Array)(s, a).slice(0, a)
        }
      }
    ],
    [
      {
        key: 'generateKey',
        value: function (n, r) {
          var i = new e()
          return (i._generateKey(n), { privateKey: i, publicKey: new Yu(i._publicKey) })
        }
      }
    ]
  )
})()
var Gn = {}
Object.defineProperty(Gn, '__esModule', { value: !0 })
Gn.MD5 = Ju
function Vr(e) {
  '@babel/helpers - typeof'
  return (
    (Vr =
      typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
        ? function (t) {
            return typeof t
          }
        : function (t) {
            return t &&
              typeof Symbol == 'function' &&
              t.constructor === Symbol &&
              t !== Symbol.prototype
              ? 'symbol'
              : typeof t
          }),
    Vr(e)
  )
}
function Pn() {
  /*! regenerator-runtime -- Copyright (c) 2014-present, Facebook, Inc. -- license (MIT): https://github.com/facebook/regenerator/blob/main/LICENSE */ Pn =
    function () {
      return t
    }
  var e,
    t = {},
    n = Object.prototype,
    r = n.hasOwnProperty,
    i =
      Object.defineProperty ||
      function (S, p, b) {
        S[p] = b.value
      },
    o = typeof Symbol == 'function' ? Symbol : {},
    a = o.iterator || '@@iterator',
    s = o.asyncIterator || '@@asyncIterator',
    x = o.toStringTag || '@@toStringTag'
  function u(S, p, b) {
    return (
      Object.defineProperty(S, p, { value: b, enumerable: !0, configurable: !0, writable: !0 }),
      S[p]
    )
  }
  try {
    u({}, '')
  } catch {
    u = function (b, L, B) {
      return (b[L] = B)
    }
  }
  function f(S, p, b, L) {
    var B = p && p.prototype instanceof h ? p : h,
      M = Object.create(B.prototype),
      V = new $(L || [])
    return (i(M, '_invoke', { value: O(S, b, V) }), M)
  }
  function F(S, p, b) {
    try {
      return { type: 'normal', arg: S.call(p, b) }
    } catch (L) {
      return { type: 'throw', arg: L }
    }
  }
  t.wrap = f
  var X = 'suspendedStart',
    g = 'suspendedYield',
    k = 'executing',
    _ = 'completed',
    l = {}
  function h() {}
  function c() {}
  function E() {}
  var y = {}
  u(y, a, function () {
    return this
  })
  var K = Object.getPrototypeOf,
    w = K && K(K(ne([])))
  w && w !== n && r.call(w, a) && (y = w)
  var m = (E.prototype = h.prototype = Object.create(y))
  function C(S) {
    ;['next', 'throw', 'return'].forEach(function (p) {
      u(S, p, function (b) {
        return this._invoke(p, b)
      })
    })
  }
  function A(S, p) {
    function b(B, M, V, re) {
      var oe = F(S[B], S, M)
      if (oe.type !== 'throw') {
        var ce = oe.arg,
          pe = ce.value
        return pe && Vr(pe) == 'object' && r.call(pe, '__await')
          ? p.resolve(pe.__await).then(
              function (ye) {
                b('next', ye, V, re)
              },
              function (ye) {
                b('throw', ye, V, re)
              }
            )
          : p.resolve(pe).then(
              function (ye) {
                ;((ce.value = ye), V(ce))
              },
              function (ye) {
                return b('throw', ye, V, re)
              }
            )
      }
      re(oe.arg)
    }
    var L
    i(this, '_invoke', {
      value: function (M, V) {
        function re() {
          return new p(function (oe, ce) {
            b(M, V, oe, ce)
          })
        }
        return (L = L ? L.then(re, re) : re())
      }
    })
  }
  function O(S, p, b) {
    var L = X
    return function (B, M) {
      if (L === k) throw Error('Generator is already running')
      if (L === _) {
        if (B === 'throw') throw M
        return { value: e, done: !0 }
      }
      for (b.method = B, b.arg = M; ; ) {
        var V = b.delegate
        if (V) {
          var re = I(V, b)
          if (re) {
            if (re === l) continue
            return re
          }
        }
        if (b.method === 'next') b.sent = b._sent = b.arg
        else if (b.method === 'throw') {
          if (L === X) throw ((L = _), b.arg)
          b.dispatchException(b.arg)
        } else b.method === 'return' && b.abrupt('return', b.arg)
        L = k
        var oe = F(S, p, b)
        if (oe.type === 'normal') {
          if (((L = b.done ? _ : g), oe.arg === l)) continue
          return { value: oe.arg, done: b.done }
        }
        oe.type === 'throw' && ((L = _), (b.method = 'throw'), (b.arg = oe.arg))
      }
    }
  }
  function I(S, p) {
    var b = p.method,
      L = S.iterator[b]
    if (L === e)
      return (
        (p.delegate = null),
        (b === 'throw' &&
          S.iterator.return &&
          ((p.method = 'return'), (p.arg = e), I(S, p), p.method === 'throw')) ||
          (b !== 'return' &&
            ((p.method = 'throw'),
            (p.arg = new TypeError("The iterator does not provide a '" + b + "' method")))),
        l
      )
    var B = F(L, S.iterator, p.arg)
    if (B.type === 'throw') return ((p.method = 'throw'), (p.arg = B.arg), (p.delegate = null), l)
    var M = B.arg
    return M
      ? M.done
        ? ((p[S.resultName] = M.value),
          (p.next = S.nextLoc),
          p.method !== 'return' && ((p.method = 'next'), (p.arg = e)),
          (p.delegate = null),
          l)
        : M
      : ((p.method = 'throw'),
        (p.arg = new TypeError('iterator result is not an object')),
        (p.delegate = null),
        l)
  }
  function Z(S) {
    var p = { tryLoc: S[0] }
    ;(1 in S && (p.catchLoc = S[1]),
      2 in S && ((p.finallyLoc = S[2]), (p.afterLoc = S[3])),
      this.tryEntries.push(p))
  }
  function q(S) {
    var p = S.completion || {}
    ;((p.type = 'normal'), delete p.arg, (S.completion = p))
  }
  function $(S) {
    ;((this.tryEntries = [{ tryLoc: 'root' }]), S.forEach(Z, this), this.reset(!0))
  }
  function ne(S) {
    if (S || S === '') {
      var p = S[a]
      if (p) return p.call(S)
      if (typeof S.next == 'function') return S
      if (!isNaN(S.length)) {
        var b = -1,
          L = function B() {
            for (; ++b < S.length; ) if (r.call(S, b)) return ((B.value = S[b]), (B.done = !1), B)
            return ((B.value = e), (B.done = !0), B)
          }
        return (L.next = L)
      }
    }
    throw new TypeError(Vr(S) + ' is not iterable')
  }
  return (
    (c.prototype = E),
    i(m, 'constructor', { value: E, configurable: !0 }),
    i(E, 'constructor', { value: c, configurable: !0 }),
    (c.displayName = u(E, x, 'GeneratorFunction')),
    (t.isGeneratorFunction = function (S) {
      var p = typeof S == 'function' && S.constructor
      return !!p && (p === c || (p.displayName || p.name) === 'GeneratorFunction')
    }),
    (t.mark = function (S) {
      return (
        Object.setPrototypeOf
          ? Object.setPrototypeOf(S, E)
          : ((S.__proto__ = E), u(S, x, 'GeneratorFunction')),
        (S.prototype = Object.create(m)),
        S
      )
    }),
    (t.awrap = function (S) {
      return { __await: S }
    }),
    C(A.prototype),
    u(A.prototype, s, function () {
      return this
    }),
    (t.AsyncIterator = A),
    (t.async = function (S, p, b, L, B) {
      B === void 0 && (B = Promise)
      var M = new A(f(S, p, b, L), B)
      return t.isGeneratorFunction(p)
        ? M
        : M.next().then(function (V) {
            return V.done ? V.value : M.next()
          })
    }),
    C(m),
    u(m, x, 'Generator'),
    u(m, a, function () {
      return this
    }),
    u(m, 'toString', function () {
      return '[object Generator]'
    }),
    (t.keys = function (S) {
      var p = Object(S),
        b = []
      for (var L in p) b.push(L)
      return (
        b.reverse(),
        function B() {
          for (; b.length; ) {
            var M = b.pop()
            if (M in p) return ((B.value = M), (B.done = !1), B)
          }
          return ((B.done = !0), B)
        }
      )
    }),
    (t.values = ne),
    ($.prototype = {
      constructor: $,
      reset: function (p) {
        if (
          ((this.prev = 0),
          (this.next = 0),
          (this.sent = this._sent = e),
          (this.done = !1),
          (this.delegate = null),
          (this.method = 'next'),
          (this.arg = e),
          this.tryEntries.forEach(q),
          !p)
        )
          for (var b in this)
            b.charAt(0) === 't' && r.call(this, b) && !isNaN(+b.slice(1)) && (this[b] = e)
      },
      stop: function () {
        this.done = !0
        var p = this.tryEntries[0].completion
        if (p.type === 'throw') throw p.arg
        return this.rval
      },
      dispatchException: function (p) {
        if (this.done) throw p
        var b = this
        function L(ce, pe) {
          return (
            (V.type = 'throw'),
            (V.arg = p),
            (b.next = ce),
            pe && ((b.method = 'next'), (b.arg = e)),
            !!pe
          )
        }
        for (var B = this.tryEntries.length - 1; B >= 0; --B) {
          var M = this.tryEntries[B],
            V = M.completion
          if (M.tryLoc === 'root') return L('end')
          if (M.tryLoc <= this.prev) {
            var re = r.call(M, 'catchLoc'),
              oe = r.call(M, 'finallyLoc')
            if (re && oe) {
              if (this.prev < M.catchLoc) return L(M.catchLoc, !0)
              if (this.prev < M.finallyLoc) return L(M.finallyLoc)
            } else if (re) {
              if (this.prev < M.catchLoc) return L(M.catchLoc, !0)
            } else {
              if (!oe) throw Error('try statement without catch or finally')
              if (this.prev < M.finallyLoc) return L(M.finallyLoc)
            }
          }
        }
      },
      abrupt: function (p, b) {
        for (var L = this.tryEntries.length - 1; L >= 0; --L) {
          var B = this.tryEntries[L]
          if (B.tryLoc <= this.prev && r.call(B, 'finallyLoc') && this.prev < B.finallyLoc) {
            var M = B
            break
          }
        }
        M && (p === 'break' || p === 'continue') && M.tryLoc <= b && b <= M.finallyLoc && (M = null)
        var V = M ? M.completion : {}
        return (
          (V.type = p),
          (V.arg = b),
          M ? ((this.method = 'next'), (this.next = M.finallyLoc), l) : this.complete(V)
        )
      },
      complete: function (p, b) {
        if (p.type === 'throw') throw p.arg
        return (
          p.type === 'break' || p.type === 'continue'
            ? (this.next = p.arg)
            : p.type === 'return'
              ? ((this.rval = this.arg = p.arg), (this.method = 'return'), (this.next = 'end'))
              : p.type === 'normal' && b && (this.next = b),
          l
        )
      },
      finish: function (p) {
        for (var b = this.tryEntries.length - 1; b >= 0; --b) {
          var L = this.tryEntries[b]
          if (L.finallyLoc === p) return (this.complete(L.completion, L.afterLoc), q(L), l)
        }
      },
      catch: function (p) {
        for (var b = this.tryEntries.length - 1; b >= 0; --b) {
          var L = this.tryEntries[b]
          if (L.tryLoc === p) {
            var B = L.completion
            if (B.type === 'throw') {
              var M = B.arg
              q(L)
            }
            return M
          }
        }
        throw Error('illegal catch attempt')
      },
      delegateYield: function (p, b, L) {
        return (
          (this.delegate = { iterator: ne(p), resultName: b, nextLoc: L }),
          this.method === 'next' && (this.arg = e),
          l
        )
      }
    }),
    t
  )
}
function ua(e, t, n, r, i, o, a) {
  try {
    var s = e[o](a),
      x = s.value
  } catch (u) {
    return void n(u)
  }
  s.done ? t(x) : Promise.resolve(x).then(r, i)
}
function $u(e) {
  return function () {
    var t = this,
      n = arguments
    return new Promise(function (r, i) {
      var o = e.apply(t, n)
      function a(x) {
        ua(o, r, i, a, s, 'next', x)
      }
      function s(x) {
        ua(o, r, i, a, s, 'throw', x)
      }
      a(void 0)
    })
  }
}
function Ju(e) {
  return Tn.apply(this, arguments)
}
function Tn() {
  return (
    (Tn = $u(
      Pn().mark(function e(t) {
        var n, r
        return Pn().wrap(function (o) {
          for (;;)
            switch ((o.prev = o.next)) {
              case 0:
                for (n = '', r = 0; r < t.length; r++) n += String.fromCharCode(t[r])
                return o.abrupt('return', el(rl(nl(tl(n), 8 * n.length))))
              case 3:
              case 'end':
                return o.stop()
            }
        }, e)
      })
    )),
    Tn.apply(this, arguments)
  )
}
function el(e) {
  for (var t = new Uint8Array(e.length), n = 0; n < e.length; n++) t[n] = e.charCodeAt(n)
  return t
}
function tl(e) {
  for (var t = Array(e.length >> 2), n = 0; n < t.length; n++) t[n] = 0
  for (var r = 0; r < 8 * e.length; r += 8) t[r >> 5] |= (255 & e.charCodeAt(r / 8)) << r % 32
  return t
}
function rl(e) {
  for (var t = '', n = 0; n < 32 * e.length; n += 8)
    t += String.fromCharCode((e[n >> 5] >>> n % 32) & 255)
  return t
}
function nl(e, t) {
  ;((e[t >> 5] |= 128 << t % 32), (e[14 + (((t + 64) >>> 9) << 4)] = t))
  for (
    var n = 1732584193, r = -271733879, i = -1732584194, o = 271733878, a = 0;
    a < e.length;
    a += 16
  ) {
    var s = n,
      x = r,
      u = i,
      f = o
    ;((r = Ye(
      (r = Ye(
        (r = Ye(
          (r = Ye(
            (r = qe(
              (r = qe(
                (r = qe(
                  (r = qe(
                    (r = Ze(
                      (r = Ze(
                        (r = Ze(
                          (r = Ze(
                            (r = Ve(
                              (r = Ve(
                                (r = Ve(
                                  (r = Ve(
                                    r,
                                    (i = Ve(
                                      i,
                                      (o = Ve(
                                        o,
                                        (n = Ve(n, r, i, o, e[a + 0], 7, -680876936)),
                                        r,
                                        i,
                                        e[a + 1],
                                        12,
                                        -389564586
                                      )),
                                      n,
                                      r,
                                      e[a + 2],
                                      17,
                                      606105819
                                    )),
                                    o,
                                    n,
                                    e[a + 3],
                                    22,
                                    -1044525330
                                  )),
                                  (i = Ve(
                                    i,
                                    (o = Ve(
                                      o,
                                      (n = Ve(n, r, i, o, e[a + 4], 7, -176418897)),
                                      r,
                                      i,
                                      e[a + 5],
                                      12,
                                      1200080426
                                    )),
                                    n,
                                    r,
                                    e[a + 6],
                                    17,
                                    -1473231341
                                  )),
                                  o,
                                  n,
                                  e[a + 7],
                                  22,
                                  -45705983
                                )),
                                (i = Ve(
                                  i,
                                  (o = Ve(
                                    o,
                                    (n = Ve(n, r, i, o, e[a + 8], 7, 1770035416)),
                                    r,
                                    i,
                                    e[a + 9],
                                    12,
                                    -1958414417
                                  )),
                                  n,
                                  r,
                                  e[a + 10],
                                  17,
                                  -42063
                                )),
                                o,
                                n,
                                e[a + 11],
                                22,
                                -1990404162
                              )),
                              (i = Ve(
                                i,
                                (o = Ve(
                                  o,
                                  (n = Ve(n, r, i, o, e[a + 12], 7, 1804603682)),
                                  r,
                                  i,
                                  e[a + 13],
                                  12,
                                  -40341101
                                )),
                                n,
                                r,
                                e[a + 14],
                                17,
                                -1502002290
                              )),
                              o,
                              n,
                              e[a + 15],
                              22,
                              1236535329
                            )),
                            (i = Ze(
                              i,
                              (o = Ze(
                                o,
                                (n = Ze(n, r, i, o, e[a + 1], 5, -165796510)),
                                r,
                                i,
                                e[a + 6],
                                9,
                                -1069501632
                              )),
                              n,
                              r,
                              e[a + 11],
                              14,
                              643717713
                            )),
                            o,
                            n,
                            e[a + 0],
                            20,
                            -373897302
                          )),
                          (i = Ze(
                            i,
                            (o = Ze(
                              o,
                              (n = Ze(n, r, i, o, e[a + 5], 5, -701558691)),
                              r,
                              i,
                              e[a + 10],
                              9,
                              38016083
                            )),
                            n,
                            r,
                            e[a + 15],
                            14,
                            -660478335
                          )),
                          o,
                          n,
                          e[a + 4],
                          20,
                          -405537848
                        )),
                        (i = Ze(
                          i,
                          (o = Ze(
                            o,
                            (n = Ze(n, r, i, o, e[a + 9], 5, 568446438)),
                            r,
                            i,
                            e[a + 14],
                            9,
                            -1019803690
                          )),
                          n,
                          r,
                          e[a + 3],
                          14,
                          -187363961
                        )),
                        o,
                        n,
                        e[a + 8],
                        20,
                        1163531501
                      )),
                      (i = Ze(
                        i,
                        (o = Ze(
                          o,
                          (n = Ze(n, r, i, o, e[a + 13], 5, -1444681467)),
                          r,
                          i,
                          e[a + 2],
                          9,
                          -51403784
                        )),
                        n,
                        r,
                        e[a + 7],
                        14,
                        1735328473
                      )),
                      o,
                      n,
                      e[a + 12],
                      20,
                      -1926607734
                    )),
                    (i = qe(
                      i,
                      (o = qe(
                        o,
                        (n = qe(n, r, i, o, e[a + 5], 4, -378558)),
                        r,
                        i,
                        e[a + 8],
                        11,
                        -2022574463
                      )),
                      n,
                      r,
                      e[a + 11],
                      16,
                      1839030562
                    )),
                    o,
                    n,
                    e[a + 14],
                    23,
                    -35309556
                  )),
                  (i = qe(
                    i,
                    (o = qe(
                      o,
                      (n = qe(n, r, i, o, e[a + 1], 4, -1530992060)),
                      r,
                      i,
                      e[a + 4],
                      11,
                      1272893353
                    )),
                    n,
                    r,
                    e[a + 7],
                    16,
                    -155497632
                  )),
                  o,
                  n,
                  e[a + 10],
                  23,
                  -1094730640
                )),
                (i = qe(
                  i,
                  (o = qe(
                    o,
                    (n = qe(n, r, i, o, e[a + 13], 4, 681279174)),
                    r,
                    i,
                    e[a + 0],
                    11,
                    -358537222
                  )),
                  n,
                  r,
                  e[a + 3],
                  16,
                  -722521979
                )),
                o,
                n,
                e[a + 6],
                23,
                76029189
              )),
              (i = qe(
                i,
                (o = qe(
                  o,
                  (n = qe(n, r, i, o, e[a + 9], 4, -640364487)),
                  r,
                  i,
                  e[a + 12],
                  11,
                  -421815835
                )),
                n,
                r,
                e[a + 15],
                16,
                530742520
              )),
              o,
              n,
              e[a + 2],
              23,
              -995338651
            )),
            (i = Ye(
              i,
              (o = Ye(
                o,
                (n = Ye(n, r, i, o, e[a + 0], 6, -198630844)),
                r,
                i,
                e[a + 7],
                10,
                1126891415
              )),
              n,
              r,
              e[a + 14],
              15,
              -1416354905
            )),
            o,
            n,
            e[a + 5],
            21,
            -57434055
          )),
          (i = Ye(
            i,
            (o = Ye(
              o,
              (n = Ye(n, r, i, o, e[a + 12], 6, 1700485571)),
              r,
              i,
              e[a + 3],
              10,
              -1894986606
            )),
            n,
            r,
            e[a + 10],
            15,
            -1051523
          )),
          o,
          n,
          e[a + 1],
          21,
          -2054922799
        )),
        (i = Ye(
          i,
          (o = Ye(
            o,
            (n = Ye(n, r, i, o, e[a + 8], 6, 1873313359)),
            r,
            i,
            e[a + 15],
            10,
            -30611744
          )),
          n,
          r,
          e[a + 6],
          15,
          -1560198380
        )),
        o,
        n,
        e[a + 13],
        21,
        1309151649
      )),
      (i = Ye(
        i,
        (o = Ye(
          o,
          (n = Ye(n, r, i, o, e[a + 4], 6, -145523070)),
          r,
          i,
          e[a + 11],
          10,
          -1120210379
        )),
        n,
        r,
        e[a + 2],
        15,
        718787259
      )),
      o,
      n,
      e[a + 9],
      21,
      -343485551
    )),
      (n = Bt(n, s)),
      (r = Bt(r, x)),
      (i = Bt(i, u)),
      (o = Bt(o, f)))
  }
  return Array(n, r, i, o)
}
function tn(e, t, n, r, i, o) {
  return Bt(il(Bt(Bt(t, e), Bt(r, o)), i), n)
}
function Ve(e, t, n, r, i, o, a) {
  return tn((t & n) | (~t & r), e, t, i, o, a)
}
function Ze(e, t, n, r, i, o, a) {
  return tn((t & r) | (n & ~r), e, t, i, o, a)
}
function qe(e, t, n, r, i, o, a) {
  return tn(t ^ n ^ r, e, t, i, o, a)
}
function Ye(e, t, n, r, i, o, a) {
  return tn(n ^ (t | ~r), e, t, i, o, a)
}
function Bt(e, t) {
  var n = (65535 & e) + (65535 & t)
  return (((e >> 16) + (t >> 16) + (n >> 16)) << 16) | (65535 & n)
}
function il(e, t) {
  return (e << t) | (e >>> (32 - t))
}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = ar,
    n = or,
    r = Jr,
    i = en,
    o = Gn
  function a(g) {
    '@babel/helpers - typeof'
    return (
      (a =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (k) {
              return typeof k
            }
          : function (k) {
              return k &&
                typeof Symbol == 'function' &&
                k.constructor === Symbol &&
                k !== Symbol.prototype
                ? 'symbol'
                : typeof k
            }),
      a(g)
    )
  }
  function s(g, k) {
    if (!(g instanceof k)) throw new TypeError('Cannot call a class as a function')
  }
  function x(g, k) {
    for (var _ = 0; _ < k.length; _++) {
      var l = k[_]
      ;((l.enumerable = l.enumerable || !1),
        (l.configurable = !0),
        'value' in l && (l.writable = !0),
        Object.defineProperty(g, f(l.key), l))
    }
  }
  function u(g, k, _) {
    return (k && x(g.prototype, k), Object.defineProperty(g, 'prototype', { writable: !1 }), g)
  }
  function f(g) {
    var k = F(g, 'string')
    return a(k) == 'symbol' ? k : k + ''
  }
  function F(g, k) {
    if (a(g) != 'object' || !g) return g
    var _ = g[Symbol.toPrimitive]
    if (_ !== void 0) {
      var l = _.call(g, k)
      if (a(l) != 'object') return l
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(g)
  }
  var X = (function () {
    function g() {
      ;(s(this, g),
        (this._algorithms = {
          'AES-ECB': t.AESECBCipher,
          'AES-EAX': t.AESEAXCipher,
          'DES-ECB': n.DESECBCipher,
          'DES-CBC': n.DESCBCCipher,
          'RSA-PKCS1-v1_5': r.RSACipher,
          DH: i.DHCipher,
          MD5: o.MD5
        }))
    }
    return u(g, [
      {
        key: 'encrypt',
        value: function (_, l, h) {
          if (l.algorithm.name !== _.name) throw new Error('algorithm does not match')
          if (typeof l.encrypt != 'function') throw new Error('key does not support encryption')
          return l.encrypt(_, h)
        }
      },
      {
        key: 'decrypt',
        value: function (_, l, h) {
          if (l.algorithm.name !== _.name) throw new Error('algorithm does not match')
          if (typeof l.decrypt != 'function') throw new Error('key does not support encryption')
          return l.decrypt(_, h)
        }
      },
      {
        key: 'importKey',
        value: function (_, l, h, c, E) {
          if (_ !== 'raw') throw new Error('key format is not supported')
          var y = this._algorithms[h.name]
          if (typeof y > 'u' || typeof y.importKey != 'function')
            throw new Error('algorithm is not supported')
          return y.importKey(l, h, c, E)
        }
      },
      {
        key: 'generateKey',
        value: function (_, l, h) {
          var c = this._algorithms[_.name]
          if (typeof c > 'u' || typeof c.generateKey != 'function')
            throw new Error('algorithm is not supported')
          return c.generateKey(_, l, h)
        }
      },
      {
        key: 'exportKey',
        value: function (_, l) {
          if (_ !== 'raw') throw new Error('key format is not supported')
          if (typeof l.exportKey != 'function') throw new Error('key does not support exportKey')
          return l.exportKey()
        }
      },
      {
        key: 'digest',
        value: function (_, l) {
          var h = this._algorithms[_]
          if (typeof h != 'function') throw new Error('algorithm is not supported')
          return h(l)
        }
      },
      {
        key: 'deriveBits',
        value: function (_, l, h) {
          if (l.algorithm.name !== _.name) throw new Error('algorithm does not match')
          if (typeof l.deriveBits != 'function') throw new Error('key does not support deriveBits')
          return l.deriveBits(_, h)
        }
      }
    ])
  })()
  e.default = new X()
})(Hn)
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = Er,
    n = i(Mn),
    r = i(Hn)
  function i(w) {
    return w && w.__esModule ? w : { default: w }
  }
  function o(w, m, C) {
    return ((m = u(m)), a(w, x() ? Reflect.construct(m, [], u(w).constructor) : m.apply(w, C)))
  }
  function a(w, m) {
    if (m && (X(m) == 'object' || typeof m == 'function')) return m
    if (m !== void 0)
      throw new TypeError('Derived constructors may only return object or undefined')
    return s(w)
  }
  function s(w) {
    if (w === void 0)
      throw new ReferenceError("this hasn't been initialised - super() hasn't been called")
    return w
  }
  function x() {
    try {
      var w = !Boolean.prototype.valueOf.call(Reflect.construct(Boolean, [], function () {}))
    } catch {}
    return (x = function () {
      return !!w
    })()
  }
  function u(w) {
    return (
      (u = Object.setPrototypeOf
        ? Object.getPrototypeOf.bind()
        : function (m) {
            return m.__proto__ || Object.getPrototypeOf(m)
          }),
      u(w)
    )
  }
  function f(w, m) {
    if (typeof m != 'function' && m !== null)
      throw new TypeError('Super expression must either be null or a function')
    ;((w.prototype = Object.create(m && m.prototype, {
      constructor: { value: w, writable: !0, configurable: !0 }
    })),
      Object.defineProperty(w, 'prototype', { writable: !1 }),
      m && F(w, m))
  }
  function F(w, m) {
    return (
      (F = Object.setPrototypeOf
        ? Object.setPrototypeOf.bind()
        : function (C, A) {
            return ((C.__proto__ = A), C)
          }),
      F(w, m)
    )
  }
  function X(w) {
    '@babel/helpers - typeof'
    return (
      (X =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (m) {
              return typeof m
            }
          : function (m) {
              return m &&
                typeof Symbol == 'function' &&
                m.constructor === Symbol &&
                m !== Symbol.prototype
                ? 'symbol'
                : typeof m
            }),
      X(w)
    )
  }
  function g() {
    /*! regenerator-runtime -- Copyright (c) 2014-present, Facebook, Inc. -- license (MIT): https://github.com/facebook/regenerator/blob/main/LICENSE */ g =
      function () {
        return m
      }
    var w,
      m = {},
      C = Object.prototype,
      A = C.hasOwnProperty,
      O =
        Object.defineProperty ||
        function (se, Y, ie) {
          se[Y] = ie.value
        },
      I = typeof Symbol == 'function' ? Symbol : {},
      Z = I.iterator || '@@iterator',
      q = I.asyncIterator || '@@asyncIterator',
      $ = I.toStringTag || '@@toStringTag'
    function ne(se, Y, ie) {
      return (
        Object.defineProperty(se, Y, { value: ie, enumerable: !0, configurable: !0, writable: !0 }),
        se[Y]
      )
    }
    try {
      ne({}, '')
    } catch {
      ne = function (ie, he, ve) {
        return (ie[he] = ve)
      }
    }
    function S(se, Y, ie, he) {
      var ve = Y && Y.prototype instanceof re ? Y : re,
        de = Object.create(ve.prototype),
        Pe = new $e(he || [])
      return (O(de, '_invoke', { value: He(se, ie, Pe) }), de)
    }
    function p(se, Y, ie) {
      try {
        return { type: 'normal', arg: se.call(Y, ie) }
      } catch (he) {
        return { type: 'throw', arg: he }
      }
    }
    m.wrap = S
    var b = 'suspendedStart',
      L = 'suspendedYield',
      B = 'executing',
      M = 'completed',
      V = {}
    function re() {}
    function oe() {}
    function ce() {}
    var pe = {}
    ne(pe, Z, function () {
      return this
    })
    var ye = Object.getPrototypeOf,
      Ne = ye && ye(ye(ft([])))
    Ne && Ne !== C && A.call(Ne, Z) && (pe = Ne)
    var je = (ce.prototype = re.prototype = Object.create(pe))
    function st(se) {
      ;['next', 'throw', 'return'].forEach(function (Y) {
        ne(se, Y, function (ie) {
          return this._invoke(Y, ie)
        })
      })
    }
    function Ue(se, Y) {
      function ie(ve, de, Pe, Me) {
        var Oe = p(se[ve], se, de)
        if (Oe.type !== 'throw') {
          var le = Oe.arg,
            yt = le.value
          return yt && X(yt) == 'object' && A.call(yt, '__await')
            ? Y.resolve(yt.__await).then(
                function (xt) {
                  ie('next', xt, Pe, Me)
                },
                function (xt) {
                  ie('throw', xt, Pe, Me)
                }
              )
            : Y.resolve(yt).then(
                function (xt) {
                  ;((le.value = xt), Pe(le))
                },
                function (xt) {
                  return ie('throw', xt, Pe, Me)
                }
              )
        }
        Me(Oe.arg)
      }
      var he
      O(this, '_invoke', {
        value: function (de, Pe) {
          function Me() {
            return new Y(function (Oe, le) {
              ie(de, Pe, Oe, le)
            })
          }
          return (he = he ? he.then(Me, Me) : Me())
        }
      })
    }
    function He(se, Y, ie) {
      var he = b
      return function (ve, de) {
        if (he === B) throw Error('Generator is already running')
        if (he === M) {
          if (ve === 'throw') throw de
          return { value: w, done: !0 }
        }
        for (ie.method = ve, ie.arg = de; ; ) {
          var Pe = ie.delegate
          if (Pe) {
            var Me = rt(Pe, ie)
            if (Me) {
              if (Me === V) continue
              return Me
            }
          }
          if (ie.method === 'next') ie.sent = ie._sent = ie.arg
          else if (ie.method === 'throw') {
            if (he === b) throw ((he = M), ie.arg)
            ie.dispatchException(ie.arg)
          } else ie.method === 'return' && ie.abrupt('return', ie.arg)
          he = B
          var Oe = p(se, Y, ie)
          if (Oe.type === 'normal') {
            if (((he = ie.done ? M : L), Oe.arg === V)) continue
            return { value: Oe.arg, done: ie.done }
          }
          Oe.type === 'throw' && ((he = M), (ie.method = 'throw'), (ie.arg = Oe.arg))
        }
      }
    }
    function rt(se, Y) {
      var ie = Y.method,
        he = se.iterator[ie]
      if (he === w)
        return (
          (Y.delegate = null),
          (ie === 'throw' &&
            se.iterator.return &&
            ((Y.method = 'return'), (Y.arg = w), rt(se, Y), Y.method === 'throw')) ||
            (ie !== 'return' &&
              ((Y.method = 'throw'),
              (Y.arg = new TypeError("The iterator does not provide a '" + ie + "' method")))),
          V
        )
      var ve = p(he, se.iterator, Y.arg)
      if (ve.type === 'throw')
        return ((Y.method = 'throw'), (Y.arg = ve.arg), (Y.delegate = null), V)
      var de = ve.arg
      return de
        ? de.done
          ? ((Y[se.resultName] = de.value),
            (Y.next = se.nextLoc),
            Y.method !== 'return' && ((Y.method = 'next'), (Y.arg = w)),
            (Y.delegate = null),
            V)
          : de
        : ((Y.method = 'throw'),
          (Y.arg = new TypeError('iterator result is not an object')),
          (Y.delegate = null),
          V)
    }
    function At(se) {
      var Y = { tryLoc: se[0] }
      ;(1 in se && (Y.catchLoc = se[1]),
        2 in se && ((Y.finallyLoc = se[2]), (Y.afterLoc = se[3])),
        this.tryEntries.push(Y))
    }
    function nt(se) {
      var Y = se.completion || {}
      ;((Y.type = 'normal'), delete Y.arg, (se.completion = Y))
    }
    function $e(se) {
      ;((this.tryEntries = [{ tryLoc: 'root' }]), se.forEach(At, this), this.reset(!0))
    }
    function ft(se) {
      if (se || se === '') {
        var Y = se[Z]
        if (Y) return Y.call(se)
        if (typeof se.next == 'function') return se
        if (!isNaN(se.length)) {
          var ie = -1,
            he = function ve() {
              for (; ++ie < se.length; )
                if (A.call(se, ie)) return ((ve.value = se[ie]), (ve.done = !1), ve)
              return ((ve.value = w), (ve.done = !0), ve)
            }
          return (he.next = he)
        }
      }
      throw new TypeError(X(se) + ' is not iterable')
    }
    return (
      (oe.prototype = ce),
      O(je, 'constructor', { value: ce, configurable: !0 }),
      O(ce, 'constructor', { value: oe, configurable: !0 }),
      (oe.displayName = ne(ce, $, 'GeneratorFunction')),
      (m.isGeneratorFunction = function (se) {
        var Y = typeof se == 'function' && se.constructor
        return !!Y && (Y === oe || (Y.displayName || Y.name) === 'GeneratorFunction')
      }),
      (m.mark = function (se) {
        return (
          Object.setPrototypeOf
            ? Object.setPrototypeOf(se, ce)
            : ((se.__proto__ = ce), ne(se, $, 'GeneratorFunction')),
          (se.prototype = Object.create(je)),
          se
        )
      }),
      (m.awrap = function (se) {
        return { __await: se }
      }),
      st(Ue.prototype),
      ne(Ue.prototype, q, function () {
        return this
      }),
      (m.AsyncIterator = Ue),
      (m.async = function (se, Y, ie, he, ve) {
        ve === void 0 && (ve = Promise)
        var de = new Ue(S(se, Y, ie, he), ve)
        return m.isGeneratorFunction(Y)
          ? de
          : de.next().then(function (Pe) {
              return Pe.done ? Pe.value : de.next()
            })
      }),
      st(je),
      ne(je, $, 'Generator'),
      ne(je, Z, function () {
        return this
      }),
      ne(je, 'toString', function () {
        return '[object Generator]'
      }),
      (m.keys = function (se) {
        var Y = Object(se),
          ie = []
        for (var he in Y) ie.push(he)
        return (
          ie.reverse(),
          function ve() {
            for (; ie.length; ) {
              var de = ie.pop()
              if (de in Y) return ((ve.value = de), (ve.done = !1), ve)
            }
            return ((ve.done = !0), ve)
          }
        )
      }),
      (m.values = ft),
      ($e.prototype = {
        constructor: $e,
        reset: function (Y) {
          if (
            ((this.prev = 0),
            (this.next = 0),
            (this.sent = this._sent = w),
            (this.done = !1),
            (this.delegate = null),
            (this.method = 'next'),
            (this.arg = w),
            this.tryEntries.forEach(nt),
            !Y)
          )
            for (var ie in this)
              ie.charAt(0) === 't' && A.call(this, ie) && !isNaN(+ie.slice(1)) && (this[ie] = w)
        },
        stop: function () {
          this.done = !0
          var Y = this.tryEntries[0].completion
          if (Y.type === 'throw') throw Y.arg
          return this.rval
        },
        dispatchException: function (Y) {
          if (this.done) throw Y
          var ie = this
          function he(le, yt) {
            return (
              (Pe.type = 'throw'),
              (Pe.arg = Y),
              (ie.next = le),
              yt && ((ie.method = 'next'), (ie.arg = w)),
              !!yt
            )
          }
          for (var ve = this.tryEntries.length - 1; ve >= 0; --ve) {
            var de = this.tryEntries[ve],
              Pe = de.completion
            if (de.tryLoc === 'root') return he('end')
            if (de.tryLoc <= this.prev) {
              var Me = A.call(de, 'catchLoc'),
                Oe = A.call(de, 'finallyLoc')
              if (Me && Oe) {
                if (this.prev < de.catchLoc) return he(de.catchLoc, !0)
                if (this.prev < de.finallyLoc) return he(de.finallyLoc)
              } else if (Me) {
                if (this.prev < de.catchLoc) return he(de.catchLoc, !0)
              } else {
                if (!Oe) throw Error('try statement without catch or finally')
                if (this.prev < de.finallyLoc) return he(de.finallyLoc)
              }
            }
          }
        },
        abrupt: function (Y, ie) {
          for (var he = this.tryEntries.length - 1; he >= 0; --he) {
            var ve = this.tryEntries[he]
            if (ve.tryLoc <= this.prev && A.call(ve, 'finallyLoc') && this.prev < ve.finallyLoc) {
              var de = ve
              break
            }
          }
          de &&
            (Y === 'break' || Y === 'continue') &&
            de.tryLoc <= ie &&
            ie <= de.finallyLoc &&
            (de = null)
          var Pe = de ? de.completion : {}
          return (
            (Pe.type = Y),
            (Pe.arg = ie),
            de ? ((this.method = 'next'), (this.next = de.finallyLoc), V) : this.complete(Pe)
          )
        },
        complete: function (Y, ie) {
          if (Y.type === 'throw') throw Y.arg
          return (
            Y.type === 'break' || Y.type === 'continue'
              ? (this.next = Y.arg)
              : Y.type === 'return'
                ? ((this.rval = this.arg = Y.arg), (this.method = 'return'), (this.next = 'end'))
                : Y.type === 'normal' && ie && (this.next = ie),
            V
          )
        },
        finish: function (Y) {
          for (var ie = this.tryEntries.length - 1; ie >= 0; --ie) {
            var he = this.tryEntries[ie]
            if (he.finallyLoc === Y) return (this.complete(he.completion, he.afterLoc), nt(he), V)
          }
        },
        catch: function (Y) {
          for (var ie = this.tryEntries.length - 1; ie >= 0; --ie) {
            var he = this.tryEntries[ie]
            if (he.tryLoc === Y) {
              var ve = he.completion
              if (ve.type === 'throw') {
                var de = ve.arg
                nt(he)
              }
              return de
            }
          }
          throw Error('illegal catch attempt')
        },
        delegateYield: function (Y, ie, he) {
          return (
            (this.delegate = { iterator: ft(Y), resultName: ie, nextLoc: he }),
            this.method === 'next' && (this.arg = w),
            V
          )
        }
      }),
      m
    )
  }
  function k(w, m, C, A, O, I, Z) {
    try {
      var q = w[I](Z),
        $ = q.value
    } catch (ne) {
      return void C(ne)
    }
    q.done ? m($) : Promise.resolve($).then(A, O)
  }
  function _(w) {
    return function () {
      var m = this,
        C = arguments
      return new Promise(function (A, O) {
        var I = w.apply(m, C)
        function Z($) {
          k(I, A, O, Z, q, 'next', $)
        }
        function q($) {
          k(I, A, O, Z, q, 'throw', $)
        }
        Z(void 0)
      })
    }
  }
  function l(w, m) {
    if (!(w instanceof m)) throw new TypeError('Cannot call a class as a function')
  }
  function h(w, m) {
    for (var C = 0; C < m.length; C++) {
      var A = m[C]
      ;((A.enumerable = A.enumerable || !1),
        (A.configurable = !0),
        'value' in A && (A.writable = !0),
        Object.defineProperty(w, E(A.key), A))
    }
  }
  function c(w, m, C) {
    return (m && h(w.prototype, m), Object.defineProperty(w, 'prototype', { writable: !1 }), w)
  }
  function E(w) {
    var m = y(w, 'string')
    return X(m) == 'symbol' ? m : m + ''
  }
  function y(w, m) {
    if (X(w) != 'object' || !w) return w
    var C = w[Symbol.toPrimitive]
    if (C !== void 0) {
      var A = C.call(w, m)
      if (X(A) != 'object') return A
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(w)
  }
  var K = (function () {
    function w() {
      ;(l(this, w), (this._cipher = null), (this._counter = new Uint8Array(16)))
    }
    return c(w, [
      {
        key: 'setKey',
        value: (function () {
          var m = _(
            g().mark(function A(O) {
              return g().wrap(
                function (Z) {
                  for (;;)
                    switch ((Z.prev = Z.next)) {
                      case 0:
                        return (
                          (Z.next = 2),
                          r.default.importKey('raw', O, { name: 'AES-EAX' }, !1, [
                            'encrypt, decrypt'
                          ])
                        )
                      case 2:
                        this._cipher = Z.sent
                      case 3:
                      case 'end':
                        return Z.stop()
                    }
                },
                A,
                this
              )
            })
          )
          function C(A) {
            return m.apply(this, arguments)
          }
          return C
        })()
      },
      {
        key: 'makeMessage',
        value: (function () {
          var m = _(
            g().mark(function A(O) {
              var I, Z, q, $
              return g().wrap(
                function (S) {
                  for (;;)
                    switch ((S.prev = S.next)) {
                      case 0:
                        return (
                          (I = new Uint8Array([(O.length & 65280) >>> 8, O.length & 255])),
                          (S.next = 3),
                          r.default.encrypt(
                            { name: 'AES-EAX', iv: this._counter, additionalData: I },
                            this._cipher,
                            O
                          )
                        )
                      case 3:
                        for (Z = S.sent, q = 0; q < 16 && this._counter[q]++ === 255; q++);
                        return (
                          ($ = new Uint8Array(O.length + 2 + 16)),
                          $.set(I),
                          $.set(Z, 2),
                          S.abrupt('return', $)
                        )
                      case 9:
                      case 'end':
                        return S.stop()
                    }
                },
                A,
                this
              )
            })
          )
          function C(A) {
            return m.apply(this, arguments)
          }
          return C
        })()
      },
      {
        key: 'receiveMessage',
        value: (function () {
          var m = _(
            g().mark(function A(O, I) {
              var Z, q, $
              return g().wrap(
                function (S) {
                  for (;;)
                    switch ((S.prev = S.next)) {
                      case 0:
                        return (
                          (Z = new Uint8Array([(O & 65280) >>> 8, O & 255])),
                          (S.next = 3),
                          r.default.decrypt(
                            { name: 'AES-EAX', iv: this._counter, additionalData: Z },
                            this._cipher,
                            I
                          )
                        )
                      case 3:
                        for (q = S.sent, $ = 0; $ < 16 && this._counter[$]++ === 255; $++);
                        return S.abrupt('return', q)
                      case 6:
                      case 'end':
                        return S.stop()
                    }
                },
                A,
                this
              )
            })
          )
          function C(A, O) {
            return m.apply(this, arguments)
          }
          return C
        })()
      }
    ])
  })()
  e.default = (function (w) {
    function m(C, A) {
      var O
      return (
        l(this, m),
        (O = o(this, m)),
        (O._hasStarted = !1),
        (O._checkSock = null),
        (O._checkCredentials = null),
        (O._approveServerResolve = null),
        (O._sockReject = null),
        (O._credentialsReject = null),
        (O._approveServerReject = null),
        (O._sock = C),
        (O._getCredentials = A),
        O
      )
    }
    return (
      f(m, w),
      c(m, [
        {
          key: '_waitSockAsync',
          value: function (A) {
            var O = this
            return new Promise(function (I, Z) {
              var q = function () {
                return !O._sock.rQwait('RA2', A)
              }
              q()
                ? I()
                : ((O._checkSock = function () {
                    q() && (I(), (O._checkSock = null), (O._sockReject = null))
                  }),
                  (O._sockReject = Z))
            })
          }
        },
        {
          key: '_waitApproveKeyAsync',
          value: function () {
            var A = this
            return new Promise(function (O, I) {
              ;((A._approveServerResolve = O), (A._approveServerReject = I))
            })
          }
        },
        {
          key: '_waitCredentialsAsync',
          value: function (A) {
            var O = this,
              I = function () {
                return A === 1 &&
                  O._getCredentials().username !== void 0 &&
                  O._getCredentials().password !== void 0
                  ? !0
                  : A === 2 && O._getCredentials().password !== void 0
              }
            return new Promise(function (Z, q) {
              I()
                ? Z()
                : ((O._checkCredentials = function () {
                    I() && (Z(), (O._checkCredentials = null), (O._credentialsReject = null))
                  }),
                  (O._credentialsReject = q))
            })
          }
        },
        {
          key: 'checkInternalEvents',
          value: function () {
            ;(this._checkSock !== null && this._checkSock(),
              this._checkCredentials !== null && this._checkCredentials())
          }
        },
        {
          key: 'approveServer',
          value: function () {
            this._approveServerResolve !== null &&
              (this._approveServerResolve(), (this._approveServerResolve = null))
          }
        },
        {
          key: 'disconnect',
          value: function () {
            ;(this._sockReject !== null &&
              (this._sockReject(new Error('disconnect normally')), (this._sockReject = null)),
              this._credentialsReject !== null &&
                (this._credentialsReject(new Error('disconnect normally')),
                (this._credentialsReject = null)),
              this._approveServerReject !== null &&
                (this._approveServerReject(new Error('disconnect normally')),
                (this._approveServerReject = null)))
          }
        },
        {
          key: 'negotiateRA2neAuthAsync',
          value: (function () {
            var C = _(
              g().mark(function O() {
                var I,
                  Z,
                  q,
                  $,
                  ne,
                  S,
                  p,
                  b,
                  L,
                  B,
                  M,
                  V,
                  re,
                  oe,
                  ce,
                  pe,
                  ye,
                  Ne,
                  je,
                  st,
                  Ue,
                  He,
                  rt,
                  At,
                  nt,
                  $e,
                  ft,
                  se,
                  Y,
                  ie,
                  he,
                  ve,
                  de,
                  Pe,
                  Me
                return g().wrap(
                  function (le) {
                    for (;;)
                      switch ((le.prev = le.next)) {
                        case 0:
                          return ((this._hasStarted = !0), (le.next = 3), this._waitSockAsync(4))
                        case 3:
                          if (
                            ((I = this._sock.rQpeekBytes(4)),
                            (Z = this._sock.rQshift32()),
                            !(Z < 1024))
                          ) {
                            le.next = 9
                            break
                          }
                          throw new Error('RA2: server public key is too short: ' + Z)
                        case 9:
                          if (!(Z > 8192)) {
                            le.next = 11
                            break
                          }
                          throw new Error('RA2: server public key is too long: ' + Z)
                        case 11:
                          return (
                            (q = Math.ceil(Z / 8)),
                            (le.next = 14),
                            this._waitSockAsync(q * 2)
                          )
                        case 14:
                          return (
                            ($ = this._sock.rQshiftBytes(q)),
                            (ne = this._sock.rQshiftBytes(q)),
                            (le.next = 18),
                            r.default.importKey(
                              'raw',
                              { n: $, e: ne },
                              { name: 'RSA-PKCS1-v1_5' },
                              !1,
                              ['encrypt']
                            )
                          )
                        case 18:
                          return (
                            (S = le.sent),
                            (p = new Uint8Array(4 + q * 2)),
                            p.set(I),
                            p.set($, 4),
                            p.set(ne, 4 + q),
                            (b = this._waitApproveKeyAsync()),
                            this.dispatchEvent(
                              new CustomEvent('serververification', {
                                detail: { type: 'RSA', publickey: p }
                              })
                            ),
                            (le.next = 27),
                            b
                          )
                        case 27:
                          return (
                            (L = 2048),
                            (B = Math.ceil(L / 8)),
                            (le.next = 31),
                            r.default.generateKey(
                              {
                                name: 'RSA-PKCS1-v1_5',
                                modulusLength: L,
                                publicExponent: new Uint8Array([1, 0, 1])
                              },
                              !0,
                              ['encrypt']
                            )
                          )
                        case 31:
                          return (
                            (M = le.sent.privateKey),
                            (le.next = 34),
                            r.default.exportKey('raw', M)
                          )
                        case 34:
                          return (
                            (V = le.sent),
                            (re = V.n),
                            (oe = V.e),
                            (ce = new Uint8Array(4 + B * 2)),
                            (ce[0] = (L & 4278190080) >>> 24),
                            (ce[1] = (L & 16711680) >>> 16),
                            (ce[2] = (L & 65280) >>> 8),
                            (ce[3] = L & 255),
                            ce.set(re, 4),
                            ce.set(oe, 4 + B),
                            this._sock.sQpushBytes(ce),
                            this._sock.flush(),
                            (pe = new Uint8Array(16)),
                            window.crypto.getRandomValues(pe),
                            (le.next = 50),
                            r.default.encrypt({ name: 'RSA-PKCS1-v1_5' }, S, pe)
                          )
                        case 50:
                          return (
                            (ye = le.sent),
                            (Ne = new Uint8Array(2 + q)),
                            (Ne[0] = (q & 65280) >>> 8),
                            (Ne[1] = q & 255),
                            Ne.set(ye, 2),
                            this._sock.sQpushBytes(Ne),
                            this._sock.flush(),
                            (le.next = 59),
                            this._waitSockAsync(2)
                          )
                        case 59:
                          if (this._sock.rQshift16() === B) {
                            le.next = 61
                            break
                          }
                          throw new Error('RA2: wrong encrypted message length')
                        case 61:
                          return (
                            (je = this._sock.rQshiftBytes(B)),
                            (le.next = 64),
                            r.default.decrypt({ name: 'RSA-PKCS1-v1_5' }, M, je)
                          )
                        case 64:
                          if (((st = le.sent), !(st === null || st.length !== 16))) {
                            le.next = 67
                            break
                          }
                          throw new Error('RA2: corrupted server encrypted random')
                        case 67:
                          return (
                            (Ue = new Uint8Array(32)),
                            (He = new Uint8Array(32)),
                            Ue.set(st),
                            Ue.set(pe, 16),
                            He.set(pe),
                            He.set(st, 16),
                            (le.next = 75),
                            window.crypto.subtle.digest('SHA-1', Ue)
                          )
                        case 75:
                          return (
                            (Ue = le.sent),
                            (Ue = new Uint8Array(Ue).slice(0, 16)),
                            (le.next = 79),
                            window.crypto.subtle.digest('SHA-1', He)
                          )
                        case 79:
                          return (
                            (He = le.sent),
                            (He = new Uint8Array(He).slice(0, 16)),
                            (rt = new K()),
                            (le.next = 84),
                            rt.setKey(Ue)
                          )
                        case 84:
                          return ((At = new K()), (le.next = 87), At.setKey(He))
                        case 87:
                          return (
                            (nt = new Uint8Array(8 + q * 2 + B * 2)),
                            ($e = new Uint8Array(8 + q * 2 + B * 2)),
                            nt.set(p),
                            nt.set(ce, 4 + q * 2),
                            $e.set(ce),
                            $e.set(p, 4 + B * 2),
                            (le.next = 95),
                            window.crypto.subtle.digest('SHA-1', nt)
                          )
                        case 95:
                          return (
                            (nt = le.sent),
                            (le.next = 98),
                            window.crypto.subtle.digest('SHA-1', $e)
                          )
                        case 98:
                          return (
                            ($e = le.sent),
                            (nt = new Uint8Array(nt)),
                            ($e = new Uint8Array($e)),
                            (le.t0 = this._sock),
                            (le.next = 104),
                            rt.makeMessage($e)
                          )
                        case 104:
                          return (
                            (le.t1 = le.sent),
                            le.t0.sQpushBytes.call(le.t0, le.t1),
                            this._sock.flush(),
                            (le.next = 109),
                            this._waitSockAsync(38)
                          )
                        case 109:
                          if (this._sock.rQshift16() === 20) {
                            le.next = 111
                            break
                          }
                          throw new Error('RA2: wrong server hash')
                        case 111:
                          return (
                            (le.next = 113),
                            At.receiveMessage(20, this._sock.rQshiftBytes(36))
                          )
                        case 113:
                          if (((ft = le.sent), ft !== null)) {
                            le.next = 116
                            break
                          }
                          throw new Error('RA2: failed to authenticate the message')
                        case 116:
                          se = 0
                        case 117:
                          if (!(se < 20)) {
                            le.next = 123
                            break
                          }
                          if (ft[se] === nt[se]) {
                            le.next = 120
                            break
                          }
                          throw new Error('RA2: wrong server hash')
                        case 120:
                          ;(se++, (le.next = 117))
                          break
                        case 123:
                          return ((le.next = 125), this._waitSockAsync(19))
                        case 125:
                          if (this._sock.rQshift16() === 1) {
                            le.next = 127
                            break
                          }
                          throw new Error('RA2: wrong subtype')
                        case 127:
                          return (
                            (le.next = 129),
                            At.receiveMessage(1, this._sock.rQshiftBytes(17))
                          )
                        case 129:
                          if (((Y = le.sent), Y !== null)) {
                            le.next = 132
                            break
                          }
                          throw new Error('RA2: failed to authenticate the message')
                        case 132:
                          if (((Y = Y[0]), (ie = this._waitCredentialsAsync(Y)), Y !== 1)) {
                            le.next = 138
                            break
                          }
                          ;((this._getCredentials().username === void 0 ||
                            this._getCredentials().password === void 0) &&
                            this.dispatchEvent(
                              new CustomEvent('credentialsrequired', {
                                detail: { types: ['username', 'password'] }
                              })
                            ),
                            (le.next = 143))
                          break
                        case 138:
                          if (Y !== 2) {
                            le.next = 142
                            break
                          }
                          ;(this._getCredentials().password === void 0 &&
                            this.dispatchEvent(
                              new CustomEvent('credentialsrequired', {
                                detail: { types: ['password'] }
                              })
                            ),
                            (le.next = 143))
                          break
                        case 142:
                          throw new Error('RA2: wrong subtype')
                        case 143:
                          return ((le.next = 145), ie)
                        case 145:
                          for (
                            Y === 1
                              ? (he = (0, t.encodeUTF8)(this._getCredentials().username).slice(
                                  0,
                                  255
                                ))
                              : (he = ''),
                              ve = (0, t.encodeUTF8)(this._getCredentials().password).slice(0, 255),
                              de = new Uint8Array(he.length + ve.length + 2),
                              de[0] = he.length,
                              de[he.length + 1] = ve.length,
                              Pe = 0;
                            Pe < he.length;
                            Pe++
                          )
                            de[Pe + 1] = he.charCodeAt(Pe)
                          for (Me = 0; Me < ve.length; Me++)
                            de[he.length + 2 + Me] = ve.charCodeAt(Me)
                          return ((le.t2 = this._sock), (le.next = 155), rt.makeMessage(de))
                        case 155:
                          ;((le.t3 = le.sent),
                            le.t2.sQpushBytes.call(le.t2, le.t3),
                            this._sock.flush())
                        case 158:
                        case 'end':
                          return le.stop()
                      }
                  },
                  O,
                  this
                )
              })
            )
            function A() {
              return C.apply(this, arguments)
            }
            return A
          })()
        },
        {
          key: 'hasStarted',
          get: function () {
            return this._hasStarted
          },
          set: function (A) {
            this._hasStarted = A
          }
        }
      ])
    )
  })(n.default)
})(vo)
var wo = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  function t(s) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (x) {
              return typeof x
            }
          : function (x) {
              return x &&
                typeof Symbol == 'function' &&
                x.constructor === Symbol &&
                x !== Symbol.prototype
                ? 'symbol'
                : typeof x
            }),
      t(s)
    )
  }
  function n(s, x) {
    if (!(s instanceof x)) throw new TypeError('Cannot call a class as a function')
  }
  function r(s, x) {
    for (var u = 0; u < x.length; u++) {
      var f = x[u]
      ;((f.enumerable = f.enumerable || !1),
        (f.configurable = !0),
        'value' in f && (f.writable = !0),
        Object.defineProperty(s, o(f.key), f))
    }
  }
  function i(s, x, u) {
    return (x && r(s.prototype, x), Object.defineProperty(s, 'prototype', { writable: !1 }), s)
  }
  function o(s) {
    var x = a(s, 'string')
    return t(x) == 'symbol' ? x : x + ''
  }
  function a(s, x) {
    if (t(s) != 'object' || !s) return s
    var u = s[Symbol.toPrimitive]
    if (u !== void 0) {
      var f = u.call(s, x)
      if (t(f) != 'object') return f
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(s)
  }
  e.default = (function () {
    function s() {
      ;(n(this, s), (this._lines = 0))
    }
    return i(s, [
      {
        key: 'decodeRect',
        value: function (u, f, F, X, g, k, _) {
          if (F === 0 || X === 0) return !0
          this._lines === 0 && (this._lines = X)
          for (var l = _ == 8 ? 1 : 4, h = F * l; this._lines > 0; ) {
            if (g.rQwait('RAW', h)) return !1
            var c = f + (X - this._lines),
              E = g.rQshiftBytes(h, !1)
            if (_ == 8) {
              for (var y = new Uint8Array(F * 4), K = 0; K < F; K++)
                ((y[K * 4 + 0] = (((E[K] >> 0) & 3) * 255) / 3),
                  (y[K * 4 + 1] = (((E[K] >> 2) & 3) * 255) / 3),
                  (y[K * 4 + 2] = (((E[K] >> 4) & 3) * 255) / 3),
                  (y[K * 4 + 3] = 255))
              E = y
            }
            for (var w = 0; w < F; w++) E[w * 4 + 3] = 255
            ;(k.blitImage(u, c, F, 1, E, 0), this._lines--)
          }
          return !0
        }
      }
    ])
  })()
})(wo)
var ko = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  function t(s) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (x) {
              return typeof x
            }
          : function (x) {
              return x &&
                typeof Symbol == 'function' &&
                x.constructor === Symbol &&
                x !== Symbol.prototype
                ? 'symbol'
                : typeof x
            }),
      t(s)
    )
  }
  function n(s, x) {
    if (!(s instanceof x)) throw new TypeError('Cannot call a class as a function')
  }
  function r(s, x) {
    for (var u = 0; u < x.length; u++) {
      var f = x[u]
      ;((f.enumerable = f.enumerable || !1),
        (f.configurable = !0),
        'value' in f && (f.writable = !0),
        Object.defineProperty(s, o(f.key), f))
    }
  }
  function i(s, x, u) {
    return (x && r(s.prototype, x), Object.defineProperty(s, 'prototype', { writable: !1 }), s)
  }
  function o(s) {
    var x = a(s, 'string')
    return t(x) == 'symbol' ? x : x + ''
  }
  function a(s, x) {
    if (t(s) != 'object' || !s) return s
    var u = s[Symbol.toPrimitive]
    if (u !== void 0) {
      var f = u.call(s, x)
      if (t(f) != 'object') return f
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(s)
  }
  e.default = (function () {
    function s() {
      n(this, s)
    }
    return i(s, [
      {
        key: 'decodeRect',
        value: function (u, f, F, X, g, k, _) {
          if (g.rQwait('COPYRECT', 4)) return !1
          var l = g.rQshift16(),
            h = g.rQshift16()
          return (F === 0 || X === 0 || k.copyImage(l, h, u, f, F, X), !0)
        }
      }
    ])
  })()
})(ko)
var So = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  function t(s) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (x) {
              return typeof x
            }
          : function (x) {
              return x &&
                typeof Symbol == 'function' &&
                x.constructor === Symbol &&
                x !== Symbol.prototype
                ? 'symbol'
                : typeof x
            }),
      t(s)
    )
  }
  function n(s, x) {
    if (!(s instanceof x)) throw new TypeError('Cannot call a class as a function')
  }
  function r(s, x) {
    for (var u = 0; u < x.length; u++) {
      var f = x[u]
      ;((f.enumerable = f.enumerable || !1),
        (f.configurable = !0),
        'value' in f && (f.writable = !0),
        Object.defineProperty(s, o(f.key), f))
    }
  }
  function i(s, x, u) {
    return (x && r(s.prototype, x), Object.defineProperty(s, 'prototype', { writable: !1 }), s)
  }
  function o(s) {
    var x = a(s, 'string')
    return t(x) == 'symbol' ? x : x + ''
  }
  function a(s, x) {
    if (t(s) != 'object' || !s) return s
    var u = s[Symbol.toPrimitive]
    if (u !== void 0) {
      var f = u.call(s, x)
      if (t(f) != 'object') return f
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(s)
  }
  e.default = (function () {
    function s() {
      ;(n(this, s), (this._subrects = 0))
    }
    return i(s, [
      {
        key: 'decodeRect',
        value: function (u, f, F, X, g, k, _) {
          if (this._subrects === 0) {
            if (g.rQwait('RRE', 8)) return !1
            this._subrects = g.rQshift32()
            var l = g.rQshiftBytes(4)
            k.fillRect(u, f, F, X, l)
          }
          for (; this._subrects > 0; ) {
            if (g.rQwait('RRE', 12)) return !1
            var h = g.rQshiftBytes(4),
              c = g.rQshift16(),
              E = g.rQshift16(),
              y = g.rQshift16(),
              K = g.rQshift16()
            ;(k.fillRect(u + c, f + E, y, K, h), this._subrects--)
          }
          return !0
        }
      }
    ])
  })()
})(So)
var Eo = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = r(Re)
  function n(f) {
    if (typeof WeakMap != 'function') return null
    var F = new WeakMap(),
      X = new WeakMap()
    return (n = function (k) {
      return k ? X : F
    })(f)
  }
  function r(f, F) {
    if (f && f.__esModule) return f
    if (f === null || (i(f) != 'object' && typeof f != 'function')) return { default: f }
    var X = n(F)
    if (X && X.has(f)) return X.get(f)
    var g = { __proto__: null },
      k = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var _ in f)
      if (_ !== 'default' && {}.hasOwnProperty.call(f, _)) {
        var l = k ? Object.getOwnPropertyDescriptor(f, _) : null
        l && (l.get || l.set) ? Object.defineProperty(g, _, l) : (g[_] = f[_])
      }
    return ((g.default = f), X && X.set(f, g), g)
  }
  function i(f) {
    '@babel/helpers - typeof'
    return (
      (i =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (F) {
              return typeof F
            }
          : function (F) {
              return F &&
                typeof Symbol == 'function' &&
                F.constructor === Symbol &&
                F !== Symbol.prototype
                ? 'symbol'
                : typeof F
            }),
      i(f)
    )
  }
  function o(f, F) {
    if (!(f instanceof F)) throw new TypeError('Cannot call a class as a function')
  }
  function a(f, F) {
    for (var X = 0; X < F.length; X++) {
      var g = F[X]
      ;((g.enumerable = g.enumerable || !1),
        (g.configurable = !0),
        'value' in g && (g.writable = !0),
        Object.defineProperty(f, x(g.key), g))
    }
  }
  function s(f, F, X) {
    return (F && a(f.prototype, F), Object.defineProperty(f, 'prototype', { writable: !1 }), f)
  }
  function x(f) {
    var F = u(f, 'string')
    return i(F) == 'symbol' ? F : F + ''
  }
  function u(f, F) {
    if (i(f) != 'object' || !f) return f
    var X = f[Symbol.toPrimitive]
    if (X !== void 0) {
      var g = X.call(f, F)
      if (i(g) != 'object') return g
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(f)
  }
  e.default = (function () {
    function f() {
      ;(o(this, f),
        (this._tiles = 0),
        (this._lastsubencoding = 0),
        (this._tileBuffer = new Uint8Array(16 * 16 * 4)))
    }
    return s(f, [
      {
        key: 'decodeRect',
        value: function (X, g, k, _, l, h, c) {
          for (
            this._tiles === 0 &&
            ((this._tilesX = Math.ceil(k / 16)),
            (this._tilesY = Math.ceil(_ / 16)),
            (this._totalTiles = this._tilesX * this._tilesY),
            (this._tiles = this._totalTiles));
            this._tiles > 0;

          ) {
            var E = 1
            if (l.rQwait('HEXTILE', E)) return !1
            var y = l.rQpeek8()
            if (y > 30) throw new Error('Illegal hextile subencoding (subencoding: ' + y + ')')
            var K = this._totalTiles - this._tiles,
              w = K % this._tilesX,
              m = Math.floor(K / this._tilesX),
              C = X + w * 16,
              A = g + m * 16,
              O = Math.min(16, X + k - C),
              I = Math.min(16, g + _ - A)
            if (y & 1) E += O * I * 4
            else if ((y & 2 && (E += 4), y & 4 && (E += 4), y & 8)) {
              if ((E++, l.rQwait('HEXTILE', E))) return !1
              var Z = l.rQpeekBytes(E).at(-1)
              y & 16 ? (E += Z * 6) : (E += Z * 2)
            }
            if (l.rQwait('HEXTILE', E)) return !1
            if ((l.rQshift8(), y === 0))
              this._lastsubencoding & 1
                ? t.Debug('     Ignoring blank after RAW')
                : h.fillRect(C, A, O, I, this._background)
            else if (y & 1) {
              for (var q = O * I, $ = l.rQshiftBytes(q * 4, !1), ne = 0; ne < q; ne++)
                $[ne * 4 + 3] = 255
              h.blitImage(C, A, O, I, $, 0)
            } else {
              if (
                (y & 2 && (this._background = new Uint8Array(l.rQshiftBytes(4))),
                y & 4 && (this._foreground = new Uint8Array(l.rQshiftBytes(4))),
                this._startTile(C, A, O, I, this._background),
                y & 8)
              )
                for (var S = l.rQshift8(), p = 0; p < S; p++) {
                  var b = void 0
                  y & 16 ? (b = l.rQshiftBytes(4)) : (b = this._foreground)
                  var L = l.rQshift8(),
                    B = L >> 4,
                    M = L & 15,
                    V = l.rQshift8(),
                    re = (V >> 4) + 1,
                    oe = (V & 15) + 1
                  this._subTile(B, M, re, oe, b)
                }
              this._finishTile(h)
            }
            ;((this._lastsubencoding = y), this._tiles--)
          }
          return !0
        }
      },
      {
        key: '_startTile',
        value: function (X, g, k, _, l) {
          ;((this._tileX = X), (this._tileY = g), (this._tileW = k), (this._tileH = _))
          for (var h = l[0], c = l[1], E = l[2], y = this._tileBuffer, K = 0; K < k * _ * 4; K += 4)
            ((y[K] = h), (y[K + 1] = c), (y[K + 2] = E), (y[K + 3] = 255))
        }
      },
      {
        key: '_subTile',
        value: function (X, g, k, _, l) {
          for (
            var h = l[0],
              c = l[1],
              E = l[2],
              y = X + k,
              K = g + _,
              w = this._tileBuffer,
              m = this._tileW,
              C = g;
            C < K;
            C++
          )
            for (var A = X; A < y; A++) {
              var O = (A + C * m) * 4
              ;((w[O] = h), (w[O + 1] = c), (w[O + 2] = E), (w[O + 3] = 255))
            }
        }
      },
      {
        key: '_finishTile',
        value: function (X) {
          X.blitImage(this._tileX, this._tileY, this._tileW, this._tileH, this._tileBuffer, 0)
        }
      }
    ])
  })()
})(Eo)
var Ko = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = n(Kr)
  function n(u) {
    return u && u.__esModule ? u : { default: u }
  }
  function r(u) {
    '@babel/helpers - typeof'
    return (
      (r =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (f) {
              return typeof f
            }
          : function (f) {
              return f &&
                typeof Symbol == 'function' &&
                f.constructor === Symbol &&
                f !== Symbol.prototype
                ? 'symbol'
                : typeof f
            }),
      r(u)
    )
  }
  function i(u, f) {
    if (!(u instanceof f)) throw new TypeError('Cannot call a class as a function')
  }
  function o(u, f) {
    for (var F = 0; F < f.length; F++) {
      var X = f[F]
      ;((X.enumerable = X.enumerable || !1),
        (X.configurable = !0),
        'value' in X && (X.writable = !0),
        Object.defineProperty(u, s(X.key), X))
    }
  }
  function a(u, f, F) {
    return (f && o(u.prototype, f), Object.defineProperty(u, 'prototype', { writable: !1 }), u)
  }
  function s(u) {
    var f = x(u, 'string')
    return r(f) == 'symbol' ? f : f + ''
  }
  function x(u, f) {
    if (r(u) != 'object' || !u) return u
    var F = u[Symbol.toPrimitive]
    if (F !== void 0) {
      var X = F.call(u, f)
      if (r(X) != 'object') return X
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(u)
  }
  e.default = (function () {
    function u() {
      ;(i(this, u), (this._zlib = new t.default()), (this._length = 0))
    }
    return a(u, [
      {
        key: 'decodeRect',
        value: function (F, X, g, k, _, l, h) {
          if (g === 0 || k === 0) return !0
          if (this._length === 0) {
            if (_.rQwait('ZLIB', 4)) return !1
            this._length = _.rQshift32()
          }
          if (_.rQwait('ZLIB', this._length)) return !1
          var c = new Uint8Array(_.rQshiftBytes(this._length, !1))
          ;((this._length = 0),
            this._zlib.setInput(c),
            (c = this._zlib.inflate(g * k * 4)),
            this._zlib.setInput(null))
          for (var E = 0; E < g * k; E++) c[E * 4 + 3] = 255
          return (l.blitImage(F, X, g, k, c, 0), !0)
        }
      }
    ])
  })()
})(Ko)
var Vn = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = o(Re),
    n = r(Kr)
  function r(X) {
    return X && X.__esModule ? X : { default: X }
  }
  function i(X) {
    if (typeof WeakMap != 'function') return null
    var g = new WeakMap(),
      k = new WeakMap()
    return (i = function (l) {
      return l ? k : g
    })(X)
  }
  function o(X, g) {
    if (X && X.__esModule) return X
    if (X === null || (a(X) != 'object' && typeof X != 'function')) return { default: X }
    var k = i(g)
    if (k && k.has(X)) return k.get(X)
    var _ = { __proto__: null },
      l = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var h in X)
      if (h !== 'default' && {}.hasOwnProperty.call(X, h)) {
        var c = l ? Object.getOwnPropertyDescriptor(X, h) : null
        c && (c.get || c.set) ? Object.defineProperty(_, h, c) : (_[h] = X[h])
      }
    return ((_.default = X), k && k.set(X, _), _)
  }
  function a(X) {
    '@babel/helpers - typeof'
    return (
      (a =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (g) {
              return typeof g
            }
          : function (g) {
              return g &&
                typeof Symbol == 'function' &&
                g.constructor === Symbol &&
                g !== Symbol.prototype
                ? 'symbol'
                : typeof g
            }),
      a(X)
    )
  }
  function s(X, g) {
    if (!(X instanceof g)) throw new TypeError('Cannot call a class as a function')
  }
  function x(X, g) {
    for (var k = 0; k < g.length; k++) {
      var _ = g[k]
      ;((_.enumerable = _.enumerable || !1),
        (_.configurable = !0),
        'value' in _ && (_.writable = !0),
        Object.defineProperty(X, f(_.key), _))
    }
  }
  function u(X, g, k) {
    return (g && x(X.prototype, g), Object.defineProperty(X, 'prototype', { writable: !1 }), X)
  }
  function f(X) {
    var g = F(X, 'string')
    return a(g) == 'symbol' ? g : g + ''
  }
  function F(X, g) {
    if (a(X) != 'object' || !X) return X
    var k = X[Symbol.toPrimitive]
    if (k !== void 0) {
      var _ = k.call(X, g)
      if (a(_) != 'object') return _
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(X)
  }
  e.default = (function () {
    function X() {
      ;(s(this, X),
        (this._ctl = null),
        (this._filter = null),
        (this._numColors = 0),
        (this._palette = new Uint8Array(1024)),
        (this._len = 0),
        (this._zlibs = []))
      for (var g = 0; g < 4; g++) this._zlibs[g] = new n.default()
    }
    return u(X, [
      {
        key: 'decodeRect',
        value: function (k, _, l, h, c, E, y) {
          if (this._ctl === null) {
            if (c.rQwait('TIGHT compression-control', 1)) return !1
            this._ctl = c.rQshift8()
            for (var K = 0; K < 4; K++)
              (this._ctl >> K) & 1 && (this._zlibs[K].reset(), t.Info('Reset zlib stream ' + K))
            this._ctl = this._ctl >> 4
          }
          var w
          if (this._ctl === 8) w = this._fillRect(k, _, l, h, c, E, y)
          else if (this._ctl === 9) w = this._jpegRect(k, _, l, h, c, E, y)
          else if (this._ctl === 10) w = this._pngRect(k, _, l, h, c, E, y)
          else if (!(this._ctl & 8)) w = this._basicRect(this._ctl, k, _, l, h, c, E, y)
          else throw new Error('Illegal tight compression received (ctl: ' + this._ctl + ')')
          return (w && (this._ctl = null), w)
        }
      },
      {
        key: '_fillRect',
        value: function (k, _, l, h, c, E, y) {
          if (c.rQwait('TIGHT', 3)) return !1
          var K = c.rQshiftBytes(3)
          return (E.fillRect(k, _, l, h, K, !1), !0)
        }
      },
      {
        key: '_jpegRect',
        value: function (k, _, l, h, c, E, y) {
          var K = this._readData(c)
          return K === null ? !1 : (E.imageRect(k, _, l, h, 'image/jpeg', K), !0)
        }
      },
      {
        key: '_pngRect',
        value: function (k, _, l, h, c, E, y) {
          throw new Error('PNG received in standard Tight rect')
        }
      },
      {
        key: '_basicRect',
        value: function (k, _, l, h, c, E, y, K) {
          if (this._filter === null)
            if (k & 4) {
              if (E.rQwait('TIGHT', 1)) return !1
              this._filter = E.rQshift8()
            } else this._filter = 0
          var w = k & 3,
            m
          switch (this._filter) {
            case 0:
              m = this._copyFilter(w, _, l, h, c, E, y, K)
              break
            case 1:
              m = this._paletteFilter(w, _, l, h, c, E, y, K)
              break
            case 2:
              m = this._gradientFilter(w, _, l, h, c, E, y, K)
              break
            default:
              throw new Error('Illegal tight filter received (ctl: ' + this._filter + ')')
          }
          return (m && (this._filter = null), m)
        }
      },
      {
        key: '_copyFilter',
        value: function (k, _, l, h, c, E, y, K) {
          var w = h * c * 3,
            m
          if (w === 0) return !0
          if (w < 12) {
            if (E.rQwait('TIGHT', w)) return !1
            m = E.rQshiftBytes(w)
          } else {
            if (((m = this._readData(E)), m === null)) return !1
            ;(this._zlibs[k].setInput(m),
              (m = this._zlibs[k].inflate(w)),
              this._zlibs[k].setInput(null))
          }
          for (var C = new Uint8Array(h * c * 4), A = 0, O = 0; A < h * c * 4; A += 4, O += 3)
            ((C[A] = m[O]), (C[A + 1] = m[O + 1]), (C[A + 2] = m[O + 2]), (C[A + 3] = 255))
          return (y.blitImage(_, l, h, c, C, 0, !1), !0)
        }
      },
      {
        key: '_paletteFilter',
        value: function (k, _, l, h, c, E, y, K) {
          if (this._numColors === 0) {
            if (E.rQwait('TIGHT palette', 1)) return !1
            var w = E.rQpeek8() + 1,
              m = w * 3
            if (E.rQwait('TIGHT palette', 1 + m)) return !1
            ;((this._numColors = w), E.rQskipBytes(1), E.rQshiftTo(this._palette, m))
          }
          var C = this._numColors <= 2 ? 1 : 8,
            A = Math.floor((h * C + 7) / 8),
            O = A * c,
            I
          if (O === 0) return !0
          if (O < 12) {
            if (E.rQwait('TIGHT', O)) return !1
            I = E.rQshiftBytes(O)
          } else {
            if (((I = this._readData(E)), I === null)) return !1
            ;(this._zlibs[k].setInput(I),
              (I = this._zlibs[k].inflate(O)),
              this._zlibs[k].setInput(null))
          }
          return (
            this._numColors == 2
              ? this._monoRect(_, l, h, c, I, this._palette, y)
              : this._paletteRect(_, l, h, c, I, this._palette, y),
            (this._numColors = 0),
            !0
          )
        }
      },
      {
        key: '_monoRect',
        value: function (k, _, l, h, c, E, y) {
          for (
            var K = this._getScratchBuffer(l * h * 4),
              w = Math.floor((l + 7) / 8),
              m = Math.floor(l / 8),
              C = 0;
            C < h;
            C++
          ) {
            var A = void 0,
              O = void 0,
              I = void 0
            for (I = 0; I < m; I++)
              for (var Z = 7; Z >= 0; Z--)
                ((A = (C * l + I * 8 + 7 - Z) * 4),
                  (O = ((c[C * w + I] >> Z) & 1) * 3),
                  (K[A] = E[O]),
                  (K[A + 1] = E[O + 1]),
                  (K[A + 2] = E[O + 2]),
                  (K[A + 3] = 255))
            for (var q = 7; q >= 8 - (l % 8); q--)
              ((A = (C * l + I * 8 + 7 - q) * 4),
                (O = ((c[C * w + I] >> q) & 1) * 3),
                (K[A] = E[O]),
                (K[A + 1] = E[O + 1]),
                (K[A + 2] = E[O + 2]),
                (K[A + 3] = 255))
          }
          y.blitImage(k, _, l, h, K, 0, !1)
        }
      },
      {
        key: '_paletteRect',
        value: function (k, _, l, h, c, E, y) {
          for (
            var K = this._getScratchBuffer(l * h * 4), w = l * h * 4, m = 0, C = 0;
            m < w;
            m += 4, C++
          ) {
            var A = c[C] * 3
            ;((K[m] = E[A]), (K[m + 1] = E[A + 1]), (K[m + 2] = E[A + 2]), (K[m + 3] = 255))
          }
          y.blitImage(k, _, l, h, K, 0, !1)
        }
      },
      {
        key: '_gradientFilter',
        value: function (k, _, l, h, c, E, y, K) {
          var w = h * c * 3,
            m
          if (w === 0) return !0
          if (w < 12) {
            if (E.rQwait('TIGHT', w)) return !1
            m = E.rQshiftBytes(w)
          } else {
            if (((m = this._readData(E)), m === null)) return !1
            ;(this._zlibs[k].setInput(m),
              (m = this._zlibs[k].inflate(w)),
              this._zlibs[k].setInput(null))
          }
          for (
            var C = new Uint8Array(4 * h * c), A = 0, O = 0, I = new Uint8Array(3), Z = 0;
            Z < h;
            Z++
          ) {
            for (var q = 0; q < 3; q++) {
              var $ = I[q],
                ne = m[O++] + $
              ;((C[A++] = ne), (I[q] = ne))
            }
            C[A++] = 255
          }
          for (var S = 0, p = new Uint8Array(3), b = new Uint8Array(3), L = 1; L < c; L++) {
            ;(I.fill(0), b.fill(0))
            for (var B = 0; B < h; B++) {
              for (var M = 0; M < 3; M++) {
                p[M] = C[S++]
                var V = I[M] + p[M] - b[M]
                V < 0 ? (V = 0) : V > 255 && (V = 255)
                var re = m[O++] + V
                ;((C[A++] = re), (b[M] = p[M]), (I[M] = re))
              }
              ;((C[A++] = 255), S++)
            }
          }
          return (y.blitImage(_, l, h, c, C, 0, !1), !0)
        }
      },
      {
        key: '_readData',
        value: function (k) {
          if (this._len === 0) {
            if (k.rQwait('TIGHT', 3)) return null
            var _
            ;((_ = k.rQshift8()),
              (this._len = _ & 127),
              _ & 128 &&
                ((_ = k.rQshift8()),
                (this._len |= (_ & 127) << 7),
                _ & 128 && ((_ = k.rQshift8()), (this._len |= _ << 14))))
          }
          if (k.rQwait('TIGHT', this._len)) return null
          var l = k.rQshiftBytes(this._len, !1)
          return ((this._len = 0), l)
        }
      },
      {
        key: '_getScratchBuffer',
        value: function (k) {
          return (
            (!this._scratchBuffer || this._scratchBuffer.length < k) &&
              (this._scratchBuffer = new Uint8Array(k)),
            this._scratchBuffer
          )
        }
      }
    ])
  })()
})(Vn)
var Xo = {}
;(function (e) {
  function t(l) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (h) {
              return typeof h
            }
          : function (h) {
              return h &&
                typeof Symbol == 'function' &&
                h.constructor === Symbol &&
                h !== Symbol.prototype
                ? 'symbol'
                : typeof h
            }),
      t(l)
    )
  }
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var n = r(Vn)
  function r(l) {
    return l && l.__esModule ? l : { default: l }
  }
  function i(l, h) {
    if (!(l instanceof h)) throw new TypeError('Cannot call a class as a function')
  }
  function o(l, h) {
    for (var c = 0; c < h.length; c++) {
      var E = h[c]
      ;((E.enumerable = E.enumerable || !1),
        (E.configurable = !0),
        'value' in E && (E.writable = !0),
        Object.defineProperty(l, s(E.key), E))
    }
  }
  function a(l, h, c) {
    return (h && o(l.prototype, h), Object.defineProperty(l, 'prototype', { writable: !1 }), l)
  }
  function s(l) {
    var h = x(l, 'string')
    return t(h) == 'symbol' ? h : h + ''
  }
  function x(l, h) {
    if (t(l) != 'object' || !l) return l
    var c = l[Symbol.toPrimitive]
    if (c !== void 0) {
      var E = c.call(l, h)
      if (t(E) != 'object') return E
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(l)
  }
  function u(l, h, c) {
    return ((h = g(h)), f(l, X() ? Reflect.construct(h, c || [], g(l).constructor) : h.apply(l, c)))
  }
  function f(l, h) {
    if (h && (t(h) == 'object' || typeof h == 'function')) return h
    if (h !== void 0)
      throw new TypeError('Derived constructors may only return object or undefined')
    return F(l)
  }
  function F(l) {
    if (l === void 0)
      throw new ReferenceError("this hasn't been initialised - super() hasn't been called")
    return l
  }
  function X() {
    try {
      var l = !Boolean.prototype.valueOf.call(Reflect.construct(Boolean, [], function () {}))
    } catch {}
    return (X = function () {
      return !!l
    })()
  }
  function g(l) {
    return (
      (g = Object.setPrototypeOf
        ? Object.getPrototypeOf.bind()
        : function (h) {
            return h.__proto__ || Object.getPrototypeOf(h)
          }),
      g(l)
    )
  }
  function k(l, h) {
    if (typeof h != 'function' && h !== null)
      throw new TypeError('Super expression must either be null or a function')
    ;((l.prototype = Object.create(h && h.prototype, {
      constructor: { value: l, writable: !0, configurable: !0 }
    })),
      Object.defineProperty(l, 'prototype', { writable: !1 }),
      h && _(l, h))
  }
  function _(l, h) {
    return (
      (_ = Object.setPrototypeOf
        ? Object.setPrototypeOf.bind()
        : function (c, E) {
            return ((c.__proto__ = E), c)
          }),
      _(l, h)
    )
  }
  e.default = (function (l) {
    function h() {
      return (i(this, h), u(this, h, arguments))
    }
    return (
      k(h, l),
      a(h, [
        {
          key: '_pngRect',
          value: function (E, y, K, w, m, C, A) {
            var O = this._readData(m)
            return O === null ? !1 : (C.imageRect(E, y, K, w, 'image/png', O), !0)
          }
        },
        {
          key: '_basicRect',
          value: function (E, y, K, w, m, C, A, O) {
            throw new Error('BasicCompression received in TightPNG rect')
          }
        }
      ])
    )
  })(n.default)
})(Xo)
var Fo = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var t = n(Kr)
  function n(F) {
    return F && F.__esModule ? F : { default: F }
  }
  function r(F) {
    '@babel/helpers - typeof'
    return (
      (r =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (X) {
              return typeof X
            }
          : function (X) {
              return X &&
                typeof Symbol == 'function' &&
                X.constructor === Symbol &&
                X !== Symbol.prototype
                ? 'symbol'
                : typeof X
            }),
      r(F)
    )
  }
  function i(F, X) {
    if (!(F instanceof X)) throw new TypeError('Cannot call a class as a function')
  }
  function o(F, X) {
    for (var g = 0; g < X.length; g++) {
      var k = X[g]
      ;((k.enumerable = k.enumerable || !1),
        (k.configurable = !0),
        'value' in k && (k.writable = !0),
        Object.defineProperty(F, s(k.key), k))
    }
  }
  function a(F, X, g) {
    return (X && o(F.prototype, X), Object.defineProperty(F, 'prototype', { writable: !1 }), F)
  }
  function s(F) {
    var X = x(F, 'string')
    return r(X) == 'symbol' ? X : X + ''
  }
  function x(F, X) {
    if (r(F) != 'object' || !F) return F
    var g = F[Symbol.toPrimitive]
    if (g !== void 0) {
      var k = g.call(F, X)
      if (r(k) != 'object') return k
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(F)
  }
  var u = 64,
    f = 64
  e.default = (function () {
    function F() {
      ;(i(this, F),
        (this._length = 0),
        (this._inflator = new t.default()),
        (this._pixelBuffer = new Uint8Array(u * f * 4)),
        (this._tileBuffer = new Uint8Array(u * f * 4)))
    }
    return a(F, [
      {
        key: 'decodeRect',
        value: function (g, k, _, l, h, c, E) {
          if (this._length === 0) {
            if (h.rQwait('ZLib data length', 4)) return !1
            this._length = h.rQshift32()
          }
          if (h.rQwait('Zlib data', this._length)) return !1
          var y = h.rQshiftBytes(this._length, !1)
          this._inflator.setInput(y)
          for (var K = k; K < k + l; K += f)
            for (var w = Math.min(f, k + l - K), m = g; m < g + _; m += u) {
              var C = Math.min(u, g + _ - m),
                A = C * w,
                O = this._inflator.inflate(1)[0]
              if (O === 0) {
                var I = this._readPixels(A)
                c.blitImage(m, K, C, w, I, 0, !1)
              } else if (O === 1) {
                var Z = this._readPixels(1)
                c.fillRect(m, K, C, w, [Z[0], Z[1], Z[2]])
              } else if (O >= 2 && O <= 16) {
                var q = this._decodePaletteTile(O, A, C, w)
                c.blitImage(m, K, C, w, q, 0, !1)
              } else if (O === 128) {
                var $ = this._decodeRLETile(A)
                c.blitImage(m, K, C, w, $, 0, !1)
              } else if (O >= 130 && O <= 255) {
                var ne = this._decodeRLEPaletteTile(O - 128, A)
                c.blitImage(m, K, C, w, ne, 0, !1)
              } else throw new Error('Unknown subencoding: ' + O)
            }
          return ((this._length = 0), !0)
        }
      },
      {
        key: '_getBitsPerPixelInPalette',
        value: function (g) {
          if (g <= 2) return 1
          if (g <= 4) return 2
          if (g <= 16) return 4
        }
      },
      {
        key: '_readPixels',
        value: function (g) {
          for (
            var k = this._pixelBuffer, _ = this._inflator.inflate(3 * g), l = 0, h = 0;
            l < g * 4;
            l += 4, h += 3
          )
            ((k[l] = _[h]), (k[l + 1] = _[h + 1]), (k[l + 2] = _[h + 2]), (k[l + 3] = 255))
          return k
        }
      },
      {
        key: '_decodePaletteTile',
        value: function (g, k, _, l) {
          for (
            var h = this._tileBuffer,
              c = this._readPixels(g),
              E = this._getBitsPerPixelInPalette(g),
              y = (1 << E) - 1,
              K = 0,
              w = this._inflator.inflate(1)[0],
              m = 0;
            m < l;
            m++
          ) {
            for (var C = 8 - E, A = 0; A < _; A++) {
              C < 0 && ((C = 8 - E), (w = this._inflator.inflate(1)[0]))
              var O = (w >> C) & y
              ;((h[K] = c[O * 4]),
                (h[K + 1] = c[O * 4 + 1]),
                (h[K + 2] = c[O * 4 + 2]),
                (h[K + 3] = c[O * 4 + 3]),
                (K += 4),
                (C -= E))
            }
            C < 8 - E && m < l - 1 && (w = this._inflator.inflate(1)[0])
          }
          return h
        }
      },
      {
        key: '_decodeRLETile',
        value: function (g) {
          for (var k = this._tileBuffer, _ = 0; _ < g; )
            for (var l = this._readPixels(1), h = this._readRLELength(), c = 0; c < h; c++)
              ((k[_ * 4] = l[0]),
                (k[_ * 4 + 1] = l[1]),
                (k[_ * 4 + 2] = l[2]),
                (k[_ * 4 + 3] = l[3]),
                _++)
          return k
        }
      },
      {
        key: '_decodeRLEPaletteTile',
        value: function (g, k) {
          for (var _ = this._tileBuffer, l = this._readPixels(g), h = 0; h < k; ) {
            var c = this._inflator.inflate(1)[0],
              E = 1
            if ((c >= 128 && ((c -= 128), (E = this._readRLELength())), c > g))
              throw new Error('Too big index in palette: ' + c + ', palette size: ' + g)
            if (h + E > k)
              throw new Error(
                'Too big rle length in palette mode: ' + E + ', allowed length is: ' + (k - h)
              )
            for (var y = 0; y < E; y++)
              ((_[h * 4] = l[c * 4]),
                (_[h * 4 + 1] = l[c * 4 + 1]),
                (_[h * 4 + 2] = l[c * 4 + 2]),
                (_[h * 4 + 3] = l[c * 4 + 3]),
                h++)
          }
          return _
        }
      },
      {
        key: '_readRLELength',
        value: function () {
          var g = 0,
            k = 0
          do ((k = this._inflator.inflate(1)[0]), (g += k))
          while (k === 255)
          return g + 1
        }
      }
    ])
  })()
})(Fo)
var Co = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  function t(k) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (_) {
              return typeof _
            }
          : function (_) {
              return _ &&
                typeof Symbol == 'function' &&
                _.constructor === Symbol &&
                _ !== Symbol.prototype
                ? 'symbol'
                : typeof _
            }),
      t(k)
    )
  }
  function n(k) {
    return o(k) || i(k) || s(k) || r()
  }
  function r() {
    throw new TypeError(`Invalid attempt to spread non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)
  }
  function i(k) {
    if ((typeof Symbol < 'u' && k[Symbol.iterator] != null) || k['@@iterator'] != null)
      return Array.from(k)
  }
  function o(k) {
    if (Array.isArray(k)) return x(k)
  }
  function a(k, _) {
    var l = (typeof Symbol < 'u' && k[Symbol.iterator]) || k['@@iterator']
    if (!l) {
      if (Array.isArray(k) || (l = s(k)) || _) {
        l && (k = l)
        var h = 0,
          c = function () {}
        return {
          s: c,
          n: function () {
            return h >= k.length ? { done: !0 } : { done: !1, value: k[h++] }
          },
          e: function (m) {
            throw m
          },
          f: c
        }
      }
      throw new TypeError(`Invalid attempt to iterate non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)
    }
    var E,
      y = !0,
      K = !1
    return {
      s: function () {
        l = l.call(k)
      },
      n: function () {
        var m = l.next()
        return ((y = m.done), m)
      },
      e: function (m) {
        ;((K = !0), (E = m))
      },
      f: function () {
        try {
          y || l.return == null || l.return()
        } finally {
          if (K) throw E
        }
      }
    }
  }
  function s(k, _) {
    if (k) {
      if (typeof k == 'string') return x(k, _)
      var l = {}.toString.call(k).slice(8, -1)
      return (
        l === 'Object' && k.constructor && (l = k.constructor.name),
        l === 'Map' || l === 'Set'
          ? Array.from(k)
          : l === 'Arguments' || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(l)
            ? x(k, _)
            : void 0
      )
    }
  }
  function x(k, _) {
    ;(_ == null || _ > k.length) && (_ = k.length)
    for (var l = 0, h = Array(_); l < _; l++) h[l] = k[l]
    return h
  }
  function u(k, _) {
    if (!(k instanceof _)) throw new TypeError('Cannot call a class as a function')
  }
  function f(k, _) {
    for (var l = 0; l < _.length; l++) {
      var h = _[l]
      ;((h.enumerable = h.enumerable || !1),
        (h.configurable = !0),
        'value' in h && (h.writable = !0),
        Object.defineProperty(k, X(h.key), h))
    }
  }
  function F(k, _, l) {
    return (_ && f(k.prototype, _), Object.defineProperty(k, 'prototype', { writable: !1 }), k)
  }
  function X(k) {
    var _ = g(k, 'string')
    return t(_) == 'symbol' ? _ : _ + ''
  }
  function g(k, _) {
    if (t(k) != 'object' || !k) return k
    var l = k[Symbol.toPrimitive]
    if (l !== void 0) {
      var h = l.call(k, _)
      if (t(h) != 'object') return h
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(k)
  }
  e.default = (function () {
    function k() {
      ;(u(this, k),
        (this._cachedQuantTables = []),
        (this._cachedHuffmanTables = []),
        (this._segments = []))
    }
    return F(k, [
      {
        key: 'decodeRect',
        value: function (l, h, c, E, y, K, w) {
          for (;;) {
            var m = this._readSegment(y)
            if (m === null) return !1
            if ((this._segments.push(m), m[1] === 217)) break
          }
          var C = [],
            A = [],
            O = a(this._segments),
            I
          try {
            for (O.s(); !(I = O.n()).done; ) {
              var Z = I.value,
                q = Z[1]
              q === 196 ? C.push(Z) : q === 219 && A.push(Z)
            }
          } catch (ce) {
            O.e(ce)
          } finally {
            O.f()
          }
          var $ = this._segments.findIndex(function (ce) {
            return ce[1] == 192 || ce[1] == 194
          })
          if ($ == -1) throw new Error('Illegal JPEG image without SOF')
          if (A.length === 0) {
            var ne
            ;(ne = this._segments).splice.apply(ne, [$ + 1, 0].concat(n(this._cachedQuantTables)))
          }
          if (C.length === 0) {
            var S
            ;(S = this._segments).splice.apply(S, [$ + 1, 0].concat(n(this._cachedHuffmanTables)))
          }
          var p = 0,
            b = a(this._segments),
            L
          try {
            for (b.s(); !(L = b.n()).done; ) {
              var B = L.value
              p += B.length
            }
          } catch (ce) {
            b.e(ce)
          } finally {
            b.f()
          }
          var M = new Uint8Array(p)
          p = 0
          var V = a(this._segments),
            re
          try {
            for (V.s(); !(re = V.n()).done; ) {
              var oe = re.value
              ;(M.set(oe, p), (p += oe.length))
            }
          } catch (ce) {
            V.e(ce)
          } finally {
            V.f()
          }
          return (
            K.imageRect(l, h, c, E, 'image/jpeg', M),
            C.length !== 0 && (this._cachedHuffmanTables = C),
            A.length !== 0 && (this._cachedQuantTables = A),
            (this._segments = []),
            !0
          )
        }
      },
      {
        key: '_readSegment',
        value: function (l) {
          if (l.rQwait('JPEG', 2)) return null
          var h = l.rQshift8()
          if (h != 255) throw new Error('Illegal JPEG marker received (byte: ' + h + ')')
          var c = l.rQshift8()
          if ((c >= 208 && c <= 217) || c == 1) return new Uint8Array([h, c])
          if (l.rQwait('JPEG', 2, 2)) return null
          var E = l.rQshift16()
          if (E < 2) throw new Error('Illegal JPEG length received (length: ' + E + ')')
          if (l.rQwait('JPEG', E - 2, 4)) return null
          var y = 0
          if (c === 218)
            for (y += 2; ; ) {
              if (l.rQwait('JPEG', E - 2 + y, 4)) return null
              var K = l.rQpeekBytes(E - 2 + y, !1)
              if (K.at(-2) === 255 && K.at(-1) !== 0 && !(K.at(-1) >= 208 && K.at(-1) <= 215)) {
                y -= 2
                break
              }
              y++
            }
          var w = new Uint8Array(2 + E + y)
          return (
            (w[0] = h),
            (w[1] = c),
            (w[2] = E >> 8),
            (w[3] = E),
            w.set(l.rQshiftBytes(E - 2 + y, !1), 4),
            w
          )
        }
      }
    ])
  })()
})(Co)
var Ao = {}
;(function (e) {
  ;(Object.defineProperty(e, '__esModule', { value: !0 }),
    (e.default = e.H264Parser = e.H264Context = void 0))
  var t = r(Re)
  function n(c) {
    if (typeof WeakMap != 'function') return null
    var E = new WeakMap(),
      y = new WeakMap()
    return (n = function (w) {
      return w ? y : E
    })(c)
  }
  function r(c, E) {
    if (c && c.__esModule) return c
    if (c === null || (i(c) != 'object' && typeof c != 'function')) return { default: c }
    var y = n(E)
    if (y && y.has(c)) return y.get(c)
    var K = { __proto__: null },
      w = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var m in c)
      if (m !== 'default' && {}.hasOwnProperty.call(c, m)) {
        var C = w ? Object.getOwnPropertyDescriptor(c, m) : null
        C && (C.get || C.set) ? Object.defineProperty(K, m, C) : (K[m] = c[m])
      }
    return ((K.default = c), y && y.set(c, K), K)
  }
  function i(c) {
    '@babel/helpers - typeof'
    return (
      (i =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (E) {
              return typeof E
            }
          : function (E) {
              return E &&
                typeof Symbol == 'function' &&
                E.constructor === Symbol &&
                E !== Symbol.prototype
                ? 'symbol'
                : typeof E
            }),
      i(c)
    )
  }
  function o(c, E) {
    return f(c) || u(c, E) || s(c, E) || a()
  }
  function a() {
    throw new TypeError(`Invalid attempt to destructure non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)
  }
  function s(c, E) {
    if (c) {
      if (typeof c == 'string') return x(c, E)
      var y = {}.toString.call(c).slice(8, -1)
      return (
        y === 'Object' && c.constructor && (y = c.constructor.name),
        y === 'Map' || y === 'Set'
          ? Array.from(c)
          : y === 'Arguments' || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(y)
            ? x(c, E)
            : void 0
      )
    }
  }
  function x(c, E) {
    ;(E == null || E > c.length) && (E = c.length)
    for (var y = 0, K = Array(E); y < E; y++) K[y] = c[y]
    return K
  }
  function u(c, E) {
    var y = c == null ? null : (typeof Symbol < 'u' && c[Symbol.iterator]) || c['@@iterator']
    if (y != null) {
      var K,
        w,
        m,
        C,
        A = [],
        O = !0,
        I = !1
      try {
        if (((m = (y = y.call(c)).next), E !== 0))
          for (; !(O = (K = m.call(y)).done) && (A.push(K.value), A.length !== E); O = !0);
      } catch (Z) {
        ;((I = !0), (w = Z))
      } finally {
        try {
          if (!O && y.return != null && ((C = y.return()), Object(C) !== C)) return
        } finally {
          if (I) throw w
        }
      }
      return A
    }
  }
  function f(c) {
    if (Array.isArray(c)) return c
  }
  function F(c, E) {
    if (!(c instanceof E)) throw new TypeError('Cannot call a class as a function')
  }
  function X(c, E) {
    for (var y = 0; y < E.length; y++) {
      var K = E[y]
      ;((K.enumerable = K.enumerable || !1),
        (K.configurable = !0),
        'value' in K && (K.writable = !0),
        Object.defineProperty(c, k(K.key), K))
    }
  }
  function g(c, E, y) {
    return (E && X(c.prototype, E), Object.defineProperty(c, 'prototype', { writable: !1 }), c)
  }
  function k(c) {
    var E = _(c, 'string')
    return i(E) == 'symbol' ? E : E + ''
  }
  function _(c, E) {
    if (i(c) != 'object' || !c) return c
    var y = c[Symbol.toPrimitive]
    if (y !== void 0) {
      var K = y.call(c, E)
      if (i(K) != 'object') return K
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(c)
  }
  var l = (e.H264Parser = (function () {
      function c(E) {
        ;(F(this, c),
          (this._data = E),
          (this._index = 0),
          (this.profileIdc = null),
          (this.constraintSet = null),
          (this.levelIdc = null))
      }
      return g(c, [
        {
          key: '_getStartSequenceLen',
          value: function (y) {
            var K = this._data
            return K[y + 0] == 0 && K[y + 1] == 0 && K[y + 2] == 0 && K[y + 3] == 1
              ? 4
              : K[y + 0] == 0 && K[y + 1] == 0 && K[y + 2] == 1
                ? 3
                : 0
          }
        },
        {
          key: '_indexOfNextNalUnit',
          value: function (y) {
            for (var K = this._data, w = y; w < K.length; ++w)
              if (this._getStartSequenceLen(w) != 0) return w
            return -1
          }
        },
        {
          key: '_parseSps',
          value: function (y) {
            ;((this.profileIdc = this._data[y]),
              (this.constraintSet = this._data[y + 1]),
              (this.levelIdc = this._data[y + 2]))
          }
        },
        {
          key: '_parseNalUnit',
          value: function (y) {
            var K = this._data[y]
            if (K & 128)
              throw new Error('H264 parsing sanity check failed, forbidden zero bit is set')
            var w = K & 31
            switch (w) {
              case 1:
                return { slice: !0 }
              case 5:
                return { slice: !0, key: !0 }
              case 6:
                return {}
              case 7:
                return (this._parseSps(y + 1), {})
              case 8:
                return {}
              default:
                t.Warn('Unhandled unit type: ', w)
                break
            }
            return {}
          }
        },
        {
          key: 'parse',
          value: function () {
            for (var y = this._index, K = !1; this._index < this._data.length; ) {
              var w = this._getStartSequenceLen(this._index)
              if (w == 0) throw new Error('Invalid start sequence in bit stream')
              var m = this._parseNalUnit(this._index + w),
                C = m.slice,
                A = m.key,
                O = this._indexOfNextNalUnit(this._index + w)
              if (
                (O == -1 ? (this._index = this._data.length) : (this._index = O), A && (K = !0), C)
              )
                break
            }
            return y === this._index ? null : { frame: this._data.subarray(y, this._index), key: K }
          }
        }
      ])
    })()),
    h = (e.H264Context = (function () {
      function c(E, y) {
        ;(F(this, c),
          (this.lastUsed = 0),
          (this._width = E),
          (this._height = y),
          (this._profileIdc = null),
          (this._constraintSet = null),
          (this._levelIdc = null),
          (this._decoder = null),
          (this._pendingFrames = []))
      }
      return g(c, [
        {
          key: '_handleFrame',
          value: function (y) {
            var K = this._pendingFrames.shift()
            if (K === void 0)
              throw new Error('Pending frame queue empty when receiving frame from decoder')
            if (K.timestamp != y.timestamp)
              throw new Error(
                'Video frame timestamp mismatch. Expected ' +
                  y.timestamp +
                  ' but but got ' +
                  K.timestamp
              )
            ;((K.frame = y), (K.ready = !0), K.resolve(), K.keep || y.close())
          }
        },
        {
          key: '_handleError',
          value: function (y) {
            throw new Error('Failed to decode frame: ' + y.message)
          }
        },
        {
          key: '_configureDecoder',
          value: function (y, K, w) {
            var m = this
            ;(this._decoder === null || this._decoder.state === 'closed') &&
              (this._decoder = new VideoDecoder({
                output: function (O) {
                  return m._handleFrame(O)
                },
                error: function (O) {
                  return m._handleError(O)
                }
              }))
            var C =
              'avc1.' +
              y.toString(16).padStart(2, '0') +
              K.toString(16).padStart(2, '0') +
              w.toString(16).padStart(2, '0')
            this._decoder.configure({
              codec: C,
              codedWidth: this._width,
              codedHeight: this._height,
              optimizeForLatency: !0
            })
          }
        },
        {
          key: '_preparePendingFrame',
          value: function (y) {
            var K = { timestamp: y, promise: null, resolve: null, frame: null, ready: !1, keep: !1 }
            return (
              (K.promise = new Promise(function (w) {
                K.resolve = w
              })),
              this._pendingFrames.push(K),
              K
            )
          }
        },
        {
          key: 'decode',
          value: function (y) {
            for (var K = new l(y), w = null, m = Math.round(window.performance.now() * 1e3); ; ) {
              var C = K.parse()
              if (C === null) break
              if (
                (K.profileIdc !== null &&
                  ((self._profileIdc = K.profileIdc),
                  (self._constraintSet = K.constraintSet),
                  (self._levelIdc = K.levelIdc)),
                this._decoder === null || this._decoder.state !== 'configured')
              ) {
                if (!C.key) {
                  t.Warn("Missing key frame. Can't decode until one arrives")
                  continue
                }
                if (self._profileIdc === null) {
                  t.Warn('Cannot config decoder. Have not received SPS and PPS yet.')
                  continue
                }
                this._configureDecoder(self._profileIdc, self._constraintSet, self._levelIdc)
              }
              w = this._preparePendingFrame(m)
              var A = new EncodedVideoChunk({
                timestamp: m,
                type: C.key ? 'key' : 'delta',
                data: C.frame
              })
              try {
                this._decoder.decode(A)
              } catch (O) {
                t.Warn('Failed to decode:', O)
              }
            }
            return (w !== null && (w.keep = !0), w)
          }
        }
      ])
    })())
  e.default = (function () {
    function c() {
      ;(F(this, c), (this._tick = 0), (this._contexts = {}))
    }
    return g(c, [
      {
        key: '_contextId',
        value: function (y, K, w, m) {
          return [y, K, w, m].join(',')
        }
      },
      {
        key: '_findOldestContextId',
        value: function () {
          for (
            var y = Number.MAX_VALUE, K = void 0, w = 0, m = Object.entries(this._contexts);
            w < m.length;
            w++
          ) {
            var C = o(m[w], 2),
              A = C[0],
              O = C[1]
            O.lastUsed < y && ((y = O.lastUsed), (K = A))
          }
          return K
        }
      },
      {
        key: '_createContext',
        value: function (y, K, w, m) {
          var C = 64
          if (Object.keys(this._contexts).length >= C) {
            var A = this._findOldestContextId()
            delete this._contexts[A]
          }
          var O = new h(w, m)
          return ((this._contexts[this._contextId(y, K, w, m)] = O), O)
        }
      },
      {
        key: '_getContext',
        value: function (y, K, w, m) {
          var C = this._contexts[this._contextId(y, K, w, m)]
          return C !== void 0 ? C : this._createContext(y, K, w, m)
        }
      },
      {
        key: '_resetContext',
        value: function (y, K, w, m) {
          delete this._contexts[this._contextId(y, K, w, m)]
        }
      },
      {
        key: '_resetAllContexts',
        value: function () {
          this._contexts = {}
        }
      },
      {
        key: 'decodeRect',
        value: function (y, K, w, m, C, A, O) {
          var I = 1,
            Z = 2
          if (C.rQwait('h264 header', 8)) return !1
          var q = C.rQshift32(),
            $ = C.rQshift32()
          if (C.rQwait('h264 payload', q, 8)) return !1
          $ & Z ? this._resetAllContexts() : $ & I && this._resetContext(y, K, w, m)
          var ne = this._getContext(y, K, w, m)
          if (((ne.lastUsed = this._tick++), q !== 0)) {
            var S = C.rQshiftBytes(q, !1),
              p = ne.decode(S)
            p !== null && A.videoFrame(y, K, w, m, p)
          }
          return !0
        }
      }
    ])
  })()
})(Ao)
;(function (e) {
  function t(G) {
    '@babel/helpers - typeof'
    return (
      (t =
        typeof Symbol == 'function' && typeof Symbol.iterator == 'symbol'
          ? function (R) {
              return typeof R
            }
          : function (R) {
              return R &&
                typeof Symbol == 'function' &&
                R.constructor === Symbol &&
                R !== Symbol.prototype
                ? 'symbol'
                : typeof R
            }),
      t(G)
    )
  }
  ;(Object.defineProperty(e, '__esModule', { value: !0 }), (e.default = void 0))
  var n = Sr,
    r = p(Re),
    i = Er,
    o = Ae,
    a = Ln,
    s = Jt,
    x = ne(Mn),
    u = ne(ya),
    f = ne(Kr),
    F = ne(Ta),
    X = ne(no),
    g = ne(co),
    k = ne(ho),
    _ = ne(_o),
    l = ne(Ar),
    h = ne(po),
    c = Tr,
    E = ne(vo),
    y = ne(Hn),
    K = ne(wo),
    w = ne(ko),
    m = ne(So),
    C = ne(Eo),
    A = ne(Ko),
    O = ne(Vn),
    I = ne(Xo),
    Z = ne(Fo),
    q = ne(Co),
    $ = ne(Ao)
  function ne(G) {
    return G && G.__esModule ? G : { default: G }
  }
  function S(G) {
    if (typeof WeakMap != 'function') return null
    var R = new WeakMap(),
      D = new WeakMap()
    return (S = function (T) {
      return T ? D : R
    })(G)
  }
  function p(G, R) {
    if (G && G.__esModule) return G
    if (G === null || (t(G) != 'object' && typeof G != 'function')) return { default: G }
    var D = S(R)
    if (D && D.has(G)) return D.get(G)
    var d = { __proto__: null },
      T = Object.defineProperty && Object.getOwnPropertyDescriptor
    for (var v in G)
      if (v !== 'default' && {}.hasOwnProperty.call(G, v)) {
        var H = T ? Object.getOwnPropertyDescriptor(G, v) : null
        H && (H.get || H.set) ? Object.defineProperty(d, v, H) : (d[v] = G[v])
      }
    return ((d.default = G), D && D.set(G, d), d)
  }
  function b() {
    /*! regenerator-runtime -- Copyright (c) 2014-present, Facebook, Inc. -- license (MIT): https://github.com/facebook/regenerator/blob/main/LICENSE */ b =
      function () {
        return R
      }
    var G,
      R = {},
      D = Object.prototype,
      d = D.hasOwnProperty,
      T =
        Object.defineProperty ||
        function (fe, ee, ue) {
          fe[ee] = ue.value
        },
      v = typeof Symbol == 'function' ? Symbol : {},
      H = v.iterator || '@@iterator',
      J = v.asyncIterator || '@@asyncIterator',
      te = v.toStringTag || '@@toStringTag'
    function ae(fe, ee, ue) {
      return (
        Object.defineProperty(fe, ee, {
          value: ue,
          enumerable: !0,
          configurable: !0,
          writable: !0
        }),
        fe[ee]
      )
    }
    try {
      ae({}, '')
    } catch {
      ae = function (ue, xe, me) {
        return (ue[xe] = me)
      }
    }
    function we(fe, ee, ue, xe) {
      var me = ee && ee.prototype instanceof We ? ee : We,
        be = Object.create(me.prototype),
        Ie = new an(xe || [])
      return (T(be, '_invoke', { value: Po(fe, ue, Ie) }), be)
    }
    function Ce(fe, ee, ue) {
      try {
        return { type: 'normal', arg: fe.call(ee, ue) }
      } catch (xe) {
        return { type: 'throw', arg: xe }
      }
    }
    R.wrap = we
    var Xe = 'suspendedStart',
      Se = 'suspendedYield',
      Te = 'executing',
      _e = 'completed',
      ke = {}
    function We() {}
    function ut() {}
    function Le() {}
    var ze = {}
    ae(ze, H, function () {
      return this
    })
    var mt = Object.getPrototypeOf,
      Qt = mt && mt(mt(on([])))
    Qt && Qt !== D && d.call(Qt, H) && (ze = Qt)
    var wt = (Le.prototype = We.prototype = Object.create(ze))
    function ii(fe) {
      ;['next', 'throw', 'return'].forEach(function (ee) {
        ae(fe, ee, function (ue) {
          return this._invoke(ee, ue)
        })
      })
    }
    function Dr(fe, ee) {
      function ue(me, be, Ie, Je) {
        var it = Ce(fe[me], fe, be)
        if (it.type !== 'throw') {
          var Ut = it.arg,
            Pt = Ut.value
          return Pt && t(Pt) == 'object' && d.call(Pt, '__await')
            ? ee.resolve(Pt.__await).then(
                function (Nt) {
                  ue('next', Nt, Ie, Je)
                },
                function (Nt) {
                  ue('throw', Nt, Ie, Je)
                }
              )
            : ee.resolve(Pt).then(
                function (Nt) {
                  ;((Ut.value = Nt), Ie(Ut))
                },
                function (Nt) {
                  return ue('throw', Nt, Ie, Je)
                }
              )
        }
        Je(it.arg)
      }
      var xe
      T(this, '_invoke', {
        value: function (be, Ie) {
          function Je() {
            return new ee(function (it, Ut) {
              ue(be, Ie, it, Ut)
            })
          }
          return (xe = xe ? xe.then(Je, Je) : Je())
        }
      })
    }
    function Po(fe, ee, ue) {
      var xe = Xe
      return function (me, be) {
        if (xe === Te) throw Error('Generator is already running')
        if (xe === _e) {
          if (me === 'throw') throw be
          return { value: G, done: !0 }
        }
        for (ue.method = me, ue.arg = be; ; ) {
          var Ie = ue.delegate
          if (Ie) {
            var Je = ai(Ie, ue)
            if (Je) {
              if (Je === ke) continue
              return Je
            }
          }
          if (ue.method === 'next') ue.sent = ue._sent = ue.arg
          else if (ue.method === 'throw') {
            if (xe === Xe) throw ((xe = _e), ue.arg)
            ue.dispatchException(ue.arg)
          } else ue.method === 'return' && ue.abrupt('return', ue.arg)
          xe = Te
          var it = Ce(fe, ee, ue)
          if (it.type === 'normal') {
            if (((xe = ue.done ? _e : Se), it.arg === ke)) continue
            return { value: it.arg, done: ue.done }
          }
          it.type === 'throw' && ((xe = _e), (ue.method = 'throw'), (ue.arg = it.arg))
        }
      }
    }
    function ai(fe, ee) {
      var ue = ee.method,
        xe = fe.iterator[ue]
      if (xe === G)
        return (
          (ee.delegate = null),
          (ue === 'throw' &&
            fe.iterator.return &&
            ((ee.method = 'return'), (ee.arg = G), ai(fe, ee), ee.method === 'throw')) ||
            (ue !== 'return' &&
              ((ee.method = 'throw'),
              (ee.arg = new TypeError("The iterator does not provide a '" + ue + "' method")))),
          ke
        )
      var me = Ce(xe, fe.iterator, ee.arg)
      if (me.type === 'throw')
        return ((ee.method = 'throw'), (ee.arg = me.arg), (ee.delegate = null), ke)
      var be = me.arg
      return be
        ? be.done
          ? ((ee[fe.resultName] = be.value),
            (ee.next = fe.nextLoc),
            ee.method !== 'return' && ((ee.method = 'next'), (ee.arg = G)),
            (ee.delegate = null),
            ke)
          : be
        : ((ee.method = 'throw'),
          (ee.arg = new TypeError('iterator result is not an object')),
          (ee.delegate = null),
          ke)
    }
    function To(fe) {
      var ee = { tryLoc: fe[0] }
      ;(1 in fe && (ee.catchLoc = fe[1]),
        2 in fe && ((ee.finallyLoc = fe[2]), (ee.afterLoc = fe[3])),
        this.tryEntries.push(ee))
    }
    function nn(fe) {
      var ee = fe.completion || {}
      ;((ee.type = 'normal'), delete ee.arg, (fe.completion = ee))
    }
    function an(fe) {
      ;((this.tryEntries = [{ tryLoc: 'root' }]), fe.forEach(To, this), this.reset(!0))
    }
    function on(fe) {
      if (fe || fe === '') {
        var ee = fe[H]
        if (ee) return ee.call(fe)
        if (typeof fe.next == 'function') return fe
        if (!isNaN(fe.length)) {
          var ue = -1,
            xe = function me() {
              for (; ++ue < fe.length; )
                if (d.call(fe, ue)) return ((me.value = fe[ue]), (me.done = !1), me)
              return ((me.value = G), (me.done = !0), me)
            }
          return (xe.next = xe)
        }
      }
      throw new TypeError(t(fe) + ' is not iterable')
    }
    return (
      (ut.prototype = Le),
      T(wt, 'constructor', { value: Le, configurable: !0 }),
      T(Le, 'constructor', { value: ut, configurable: !0 }),
      (ut.displayName = ae(Le, te, 'GeneratorFunction')),
      (R.isGeneratorFunction = function (fe) {
        var ee = typeof fe == 'function' && fe.constructor
        return !!ee && (ee === ut || (ee.displayName || ee.name) === 'GeneratorFunction')
      }),
      (R.mark = function (fe) {
        return (
          Object.setPrototypeOf
            ? Object.setPrototypeOf(fe, Le)
            : ((fe.__proto__ = Le), ae(fe, te, 'GeneratorFunction')),
          (fe.prototype = Object.create(wt)),
          fe
        )
      }),
      (R.awrap = function (fe) {
        return { __await: fe }
      }),
      ii(Dr.prototype),
      ae(Dr.prototype, J, function () {
        return this
      }),
      (R.AsyncIterator = Dr),
      (R.async = function (fe, ee, ue, xe, me) {
        me === void 0 && (me = Promise)
        var be = new Dr(we(fe, ee, ue, xe), me)
        return R.isGeneratorFunction(ee)
          ? be
          : be.next().then(function (Ie) {
              return Ie.done ? Ie.value : be.next()
            })
      }),
      ii(wt),
      ae(wt, te, 'Generator'),
      ae(wt, H, function () {
        return this
      }),
      ae(wt, 'toString', function () {
        return '[object Generator]'
      }),
      (R.keys = function (fe) {
        var ee = Object(fe),
          ue = []
        for (var xe in ee) ue.push(xe)
        return (
          ue.reverse(),
          function me() {
            for (; ue.length; ) {
              var be = ue.pop()
              if (be in ee) return ((me.value = be), (me.done = !1), me)
            }
            return ((me.done = !0), me)
          }
        )
      }),
      (R.values = on),
      (an.prototype = {
        constructor: an,
        reset: function (ee) {
          if (
            ((this.prev = 0),
            (this.next = 0),
            (this.sent = this._sent = G),
            (this.done = !1),
            (this.delegate = null),
            (this.method = 'next'),
            (this.arg = G),
            this.tryEntries.forEach(nn),
            !ee)
          )
            for (var ue in this)
              ue.charAt(0) === 't' && d.call(this, ue) && !isNaN(+ue.slice(1)) && (this[ue] = G)
        },
        stop: function () {
          this.done = !0
          var ee = this.tryEntries[0].completion
          if (ee.type === 'throw') throw ee.arg
          return this.rval
        },
        dispatchException: function (ee) {
          if (this.done) throw ee
          var ue = this
          function xe(Ut, Pt) {
            return (
              (Ie.type = 'throw'),
              (Ie.arg = ee),
              (ue.next = Ut),
              Pt && ((ue.method = 'next'), (ue.arg = G)),
              !!Pt
            )
          }
          for (var me = this.tryEntries.length - 1; me >= 0; --me) {
            var be = this.tryEntries[me],
              Ie = be.completion
            if (be.tryLoc === 'root') return xe('end')
            if (be.tryLoc <= this.prev) {
              var Je = d.call(be, 'catchLoc'),
                it = d.call(be, 'finallyLoc')
              if (Je && it) {
                if (this.prev < be.catchLoc) return xe(be.catchLoc, !0)
                if (this.prev < be.finallyLoc) return xe(be.finallyLoc)
              } else if (Je) {
                if (this.prev < be.catchLoc) return xe(be.catchLoc, !0)
              } else {
                if (!it) throw Error('try statement without catch or finally')
                if (this.prev < be.finallyLoc) return xe(be.finallyLoc)
              }
            }
          }
        },
        abrupt: function (ee, ue) {
          for (var xe = this.tryEntries.length - 1; xe >= 0; --xe) {
            var me = this.tryEntries[xe]
            if (me.tryLoc <= this.prev && d.call(me, 'finallyLoc') && this.prev < me.finallyLoc) {
              var be = me
              break
            }
          }
          be &&
            (ee === 'break' || ee === 'continue') &&
            be.tryLoc <= ue &&
            ue <= be.finallyLoc &&
            (be = null)
          var Ie = be ? be.completion : {}
          return (
            (Ie.type = ee),
            (Ie.arg = ue),
            be ? ((this.method = 'next'), (this.next = be.finallyLoc), ke) : this.complete(Ie)
          )
        },
        complete: function (ee, ue) {
          if (ee.type === 'throw') throw ee.arg
          return (
            ee.type === 'break' || ee.type === 'continue'
              ? (this.next = ee.arg)
              : ee.type === 'return'
                ? ((this.rval = this.arg = ee.arg), (this.method = 'return'), (this.next = 'end'))
                : ee.type === 'normal' && ue && (this.next = ue),
            ke
          )
        },
        finish: function (ee) {
          for (var ue = this.tryEntries.length - 1; ue >= 0; --ue) {
            var xe = this.tryEntries[ue]
            if (xe.finallyLoc === ee) return (this.complete(xe.completion, xe.afterLoc), nn(xe), ke)
          }
        },
        catch: function (ee) {
          for (var ue = this.tryEntries.length - 1; ue >= 0; --ue) {
            var xe = this.tryEntries[ue]
            if (xe.tryLoc === ee) {
              var me = xe.completion
              if (me.type === 'throw') {
                var be = me.arg
                nn(xe)
              }
              return be
            }
          }
          throw Error('illegal catch attempt')
        },
        delegateYield: function (ee, ue, xe) {
          return (
            (this.delegate = { iterator: on(ee), resultName: ue, nextLoc: xe }),
            this.method === 'next' && (this.arg = G),
            ke
          )
        }
      }),
      R
    )
  }
  function L(G, R, D, d, T, v, H) {
    try {
      var J = G[v](H),
        te = J.value
    } catch (ae) {
      return void D(ae)
    }
    J.done ? R(te) : Promise.resolve(te).then(d, T)
  }
  function B(G) {
    return function () {
      var R = this,
        D = arguments
      return new Promise(function (d, T) {
        var v = G.apply(R, D)
        function H(te) {
          L(v, d, T, H, J, 'next', te)
        }
        function J(te) {
          L(v, d, T, H, J, 'throw', te)
        }
        H(void 0)
      })
    }
  }
  function M(G, R) {
    return oe(G) || re(G, R) || pe(G, R) || V()
  }
  function V() {
    throw new TypeError(`Invalid attempt to destructure non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)
  }
  function re(G, R) {
    var D = G == null ? null : (typeof Symbol < 'u' && G[Symbol.iterator]) || G['@@iterator']
    if (D != null) {
      var d,
        T,
        v,
        H,
        J = [],
        te = !0,
        ae = !1
      try {
        if (((v = (D = D.call(G)).next), R !== 0))
          for (; !(te = (d = v.call(D)).done) && (J.push(d.value), J.length !== R); te = !0);
      } catch (we) {
        ;((ae = !0), (T = we))
      } finally {
        try {
          if (!te && D.return != null && ((H = D.return()), Object(H) !== H)) return
        } finally {
          if (ae) throw T
        }
      }
      return J
    }
  }
  function oe(G) {
    if (Array.isArray(G)) return G
  }
  function ce(G, R) {
    var D = (typeof Symbol < 'u' && G[Symbol.iterator]) || G['@@iterator']
    if (!D) {
      if (Array.isArray(G) || (D = pe(G)) || R) {
        D && (G = D)
        var d = 0,
          T = function () {}
        return {
          s: T,
          n: function () {
            return d >= G.length ? { done: !0 } : { done: !1, value: G[d++] }
          },
          e: function (ae) {
            throw ae
          },
          f: T
        }
      }
      throw new TypeError(`Invalid attempt to iterate non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)
    }
    var v,
      H = !0,
      J = !1
    return {
      s: function () {
        D = D.call(G)
      },
      n: function () {
        var ae = D.next()
        return ((H = ae.done), ae)
      },
      e: function (ae) {
        ;((J = !0), (v = ae))
      },
      f: function () {
        try {
          H || D.return == null || D.return()
        } finally {
          if (J) throw v
        }
      }
    }
  }
  function pe(G, R) {
    if (G) {
      if (typeof G == 'string') return ye(G, R)
      var D = {}.toString.call(G).slice(8, -1)
      return (
        D === 'Object' && G.constructor && (D = G.constructor.name),
        D === 'Map' || D === 'Set'
          ? Array.from(G)
          : D === 'Arguments' || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(D)
            ? ye(G, R)
            : void 0
      )
    }
  }
  function ye(G, R) {
    ;(R == null || R > G.length) && (R = G.length)
    for (var D = 0, d = Array(R); D < R; D++) d[D] = G[D]
    return d
  }
  function Ne(G, R) {
    if (!(G instanceof R)) throw new TypeError('Cannot call a class as a function')
  }
  function je(G, R) {
    for (var D = 0; D < R.length; D++) {
      var d = R[D]
      ;((d.enumerable = d.enumerable || !1),
        (d.configurable = !0),
        'value' in d && (d.writable = !0),
        Object.defineProperty(G, Ue(d.key), d))
    }
  }
  function st(G, R, D) {
    return (
      R && je(G.prototype, R),
      D && je(G, D),
      Object.defineProperty(G, 'prototype', { writable: !1 }),
      G
    )
  }
  function Ue(G) {
    var R = He(G, 'string')
    return t(R) == 'symbol' ? R : R + ''
  }
  function He(G, R) {
    if (t(G) != 'object' || !G) return G
    var D = G[Symbol.toPrimitive]
    if (D !== void 0) {
      var d = D.call(G, R)
      if (t(d) != 'object') return d
      throw new TypeError('@@toPrimitive must return a primitive value.')
    }
    return String(G)
  }
  function rt(G, R, D) {
    return ((R = ft(R)), At(G, $e() ? Reflect.construct(R, [], ft(G).constructor) : R.apply(G, D)))
  }
  function At(G, R) {
    if (R && (t(R) == 'object' || typeof R == 'function')) return R
    if (R !== void 0)
      throw new TypeError('Derived constructors may only return object or undefined')
    return nt(G)
  }
  function nt(G) {
    if (G === void 0)
      throw new ReferenceError("this hasn't been initialised - super() hasn't been called")
    return G
  }
  function $e() {
    try {
      var G = !Boolean.prototype.valueOf.call(Reflect.construct(Boolean, [], function () {}))
    } catch {}
    return ($e = function () {
      return !!G
    })()
  }
  function ft(G) {
    return (
      (ft = Object.setPrototypeOf
        ? Object.getPrototypeOf.bind()
        : function (R) {
            return R.__proto__ || Object.getPrototypeOf(R)
          }),
      ft(G)
    )
  }
  function se(G, R) {
    if (typeof R != 'function' && R !== null)
      throw new TypeError('Super expression must either be null or a function')
    ;((G.prototype = Object.create(R && R.prototype, {
      constructor: { value: G, writable: !0, configurable: !0 }
    })),
      Object.defineProperty(G, 'prototype', { writable: !1 }),
      R && Y(G, R))
  }
  function Y(G, R) {
    return (
      (Y = Object.setPrototypeOf
        ? Object.setPrototypeOf.bind()
        : function (D, d) {
            return ((D.__proto__ = d), D)
          }),
      Y(G, R)
    )
  }
  var ie = 3,
    he = 'rgb(40, 40, 40)',
    ve = 17,
    de = 50,
    Pe = 19,
    Me = 75,
    Oe = 50,
    le = 1e3,
    yt = 50,
    xt = 1,
    Lr = 2,
    Zn = 6,
    qn = 16,
    rn = 19,
    Yn = 22,
    $n = 30,
    Jn = 113,
    ei = 129,
    ti = 256,
    gt = 1,
    ri = 1 << 24,
    Rr = 1 << 25,
    ni = 1 << 26,
    fr = 1 << 27,
    Mr = 1 << 28,
    bt = (e.default = (function (G) {
      function R(D, d, T) {
        var v
        if ((Ne(this, R), !D)) throw new Error('Must specify target')
        if (!d) throw new Error('Must specify URL, WebSocket or RTCDataChannel')
        ;(window.isSecureContext ||
          r.Error('noVNC requires a secure context (TLS). Expect crashes!'),
          (v = rt(this, R)),
          (v._target = D),
          typeof d == 'string' ? (v._url = d) : ((v._url = null), (v._rawChannel = d)),
          (T = T || {}),
          (v._rfbCredentials = T.credentials || {}),
          (v._shared = 'shared' in T ? !!T.shared : !0),
          (v._repeaterID = T.repeaterID || ''),
          (v._wsProtocols = T.wsProtocols || []),
          (v._rfbConnectionState = ''),
          (v._rfbInitState = ''),
          (v._rfbAuthScheme = -1),
          (v._rfbCleanDisconnect = !0),
          (v._rfbRSAAESAuthenticationState = null),
          (v._rfbVersion = 0),
          (v._rfbMaxVersion = 3.8),
          (v._rfbTightVNC = !1),
          (v._rfbVeNCryptState = 0),
          (v._rfbXvpVer = 0),
          (v._fbWidth = 0),
          (v._fbHeight = 0),
          (v._fbName = ''),
          (v._capabilities = { power: !1 }),
          (v._supportsFence = !1),
          (v._supportsContinuousUpdates = !1),
          (v._enabledContinuousUpdates = !1),
          (v._supportsSetDesktopSize = !1),
          (v._screenID = 0),
          (v._screenFlags = 0),
          (v._pendingRemoteResize = !1),
          (v._lastResize = 0),
          (v._qemuExtKeyEventSupported = !1),
          (v._extendedPointerEventSupported = !1),
          (v._clipboardText = null),
          (v._clipboardServerCapabilitiesActions = {}),
          (v._clipboardServerCapabilitiesFormats = {}),
          (v._sock = null),
          (v._display = null),
          (v._flushing = !1),
          (v._keyboard = null),
          (v._gestures = null),
          (v._resizeObserver = null),
          (v._disconnTimer = null),
          (v._resizeTimeout = null),
          (v._mouseMoveTimer = null),
          (v._decoders = {}),
          (v._FBU = { rects: 0, x: 0, y: 0, width: 0, height: 0, encoding: null }),
          (v._mousePos = {}),
          (v._mouseButtonMask = 0),
          (v._mouseLastMoveTime = 0),
          (v._viewportDragging = !1),
          (v._viewportDragPos = {}),
          (v._viewportHasMoved = !1),
          (v._accumulatedWheelDeltaX = 0),
          (v._accumulatedWheelDeltaY = 0),
          (v._gestureLastTapTime = null),
          (v._gestureFirstDoubleTapEv = null),
          (v._gestureLastMagnitudeX = 0),
          (v._gestureLastMagnitudeY = 0),
          (v._eventHandlers = {
            focusCanvas: v._focusCanvas.bind(v),
            handleResize: v._handleResize.bind(v),
            handleMouse: v._handleMouse.bind(v),
            handleWheel: v._handleWheel.bind(v),
            handleGesture: v._handleGesture.bind(v),
            handleRSAAESCredentialsRequired: v._handleRSAAESCredentialsRequired.bind(v),
            handleRSAAESServerVerification: v._handleRSAAESServerVerification.bind(v)
          }),
          r.Debug('>> RFB.constructor'),
          (v._screen = document.createElement('div')),
          (v._screen.style.display = 'flex'),
          (v._screen.style.width = '100%'),
          (v._screen.style.height = '100%'),
          (v._screen.style.overflow = 'auto'),
          (v._screen.style.background = he),
          (v._canvas = document.createElement('canvas')),
          (v._canvas.style.margin = 'auto'),
          (v._canvas.style.outline = 'none'),
          (v._canvas.width = 0),
          (v._canvas.height = 0),
          (v._canvas.tabIndex = -1),
          v._screen.appendChild(v._canvas),
          (v._cursor = new k.default()),
          (v._cursorImage = R.cursors.none),
          (v._decoders[c.encodings.encodingRaw] = new K.default()),
          (v._decoders[c.encodings.encodingCopyRect] = new w.default()),
          (v._decoders[c.encodings.encodingRRE] = new m.default()),
          (v._decoders[c.encodings.encodingHextile] = new C.default()),
          (v._decoders[c.encodings.encodingZlib] = new A.default()),
          (v._decoders[c.encodings.encodingTight] = new O.default()),
          (v._decoders[c.encodings.encodingTightPNG] = new I.default()),
          (v._decoders[c.encodings.encodingZRLE] = new Z.default()),
          (v._decoders[c.encodings.encodingJPEG] = new q.default()),
          (v._decoders[c.encodings.encodingH264] = new $.default()))
        try {
          v._display = new u.default(v._canvas)
        } catch (H) {
          throw (r.Error('Display exception: ' + H), H)
        }
        return (
          (v._keyboard = new X.default(v._canvas)),
          (v._keyboard.onkeyevent = v._handleKeyEvent.bind(v)),
          (v._remoteCapsLock = null),
          (v._remoteNumLock = null),
          (v._gestures = new g.default()),
          (v._sock = new _.default()),
          v._sock.on('open', v._socketOpen.bind(v)),
          v._sock.on('close', v._socketClose.bind(v)),
          v._sock.on('message', v._handleMessage.bind(v)),
          v._sock.on('error', v._socketError.bind(v)),
          (v._expectedClientWidth = null),
          (v._expectedClientHeight = null),
          (v._resizeObserver = new ResizeObserver(v._eventHandlers.handleResize)),
          v._updateConnectionState('connecting'),
          r.Debug('<< RFB.constructor'),
          (v.dragViewport = !1),
          (v.focusOnClick = !0),
          (v._viewOnly = !1),
          (v._clipViewport = !1),
          (v._clippingViewport = !1),
          (v._scaleViewport = !1),
          (v._resizeSession = !1),
          (v._showDotCursor = !1),
          T.showDotCursor !== void 0 &&
            (r.Warn('Specifying showDotCursor as a RFB constructor argument is deprecated'),
            (v._showDotCursor = T.showDotCursor)),
          (v._qualityLevel = 6),
          (v._compressionLevel = 2),
          v
        )
      }
      return (
        se(R, G),
        st(
          R,
          [
            {
              key: 'viewOnly',
              get: function () {
                return this._viewOnly
              },
              set: function (d) {
                ;((this._viewOnly = d),
                  (this._rfbConnectionState === 'connecting' ||
                    this._rfbConnectionState === 'connected') &&
                    (d ? this._keyboard.ungrab() : this._keyboard.grab()))
              }
            },
            {
              key: 'capabilities',
              get: function () {
                return this._capabilities
              }
            },
            {
              key: 'clippingViewport',
              get: function () {
                return this._clippingViewport
              }
            },
            {
              key: '_setClippingViewport',
              value: function (d) {
                d !== this._clippingViewport &&
                  ((this._clippingViewport = d),
                  this.dispatchEvent(
                    new CustomEvent('clippingviewport', { detail: this._clippingViewport })
                  ))
              }
            },
            {
              key: 'touchButton',
              get: function () {
                return 0
              },
              set: function (d) {
                r.Warn('Using old API!')
              }
            },
            {
              key: 'clipViewport',
              get: function () {
                return this._clipViewport
              },
              set: function (d) {
                ;((this._clipViewport = d), this._updateClip())
              }
            },
            {
              key: 'scaleViewport',
              get: function () {
                return this._scaleViewport
              },
              set: function (d) {
                ;((this._scaleViewport = d),
                  d && this._clipViewport && this._updateClip(),
                  this._updateScale(),
                  !d && this._clipViewport && this._updateClip())
              }
            },
            {
              key: 'resizeSession',
              get: function () {
                return this._resizeSession
              },
              set: function (d) {
                ;((this._resizeSession = d), d && this._requestRemoteResize())
              }
            },
            {
              key: 'showDotCursor',
              get: function () {
                return this._showDotCursor
              },
              set: function (d) {
                ;((this._showDotCursor = d), this._refreshCursor())
              }
            },
            {
              key: 'background',
              get: function () {
                return this._screen.style.background
              },
              set: function (d) {
                this._screen.style.background = d
              }
            },
            {
              key: 'qualityLevel',
              get: function () {
                return this._qualityLevel
              },
              set: function (d) {
                if (!Number.isInteger(d) || d < 0 || d > 9) {
                  r.Error('qualityLevel must be an integer between 0 and 9')
                  return
                }
                this._qualityLevel !== d &&
                  ((this._qualityLevel = d),
                  this._rfbConnectionState === 'connected' && this._sendEncodings())
              }
            },
            {
              key: 'compressionLevel',
              get: function () {
                return this._compressionLevel
              },
              set: function (d) {
                if (!Number.isInteger(d) || d < 0 || d > 9) {
                  r.Error('compressionLevel must be an integer between 0 and 9')
                  return
                }
                this._compressionLevel !== d &&
                  ((this._compressionLevel = d),
                  this._rfbConnectionState === 'connected' && this._sendEncodings())
              }
            },
            {
              key: 'disconnect',
              value: function () {
                ;(this._updateConnectionState('disconnecting'),
                  this._sock.off('error'),
                  this._sock.off('message'),
                  this._sock.off('open'),
                  this._rfbRSAAESAuthenticationState !== null &&
                    this._rfbRSAAESAuthenticationState.disconnect())
              }
            },
            {
              key: 'approveServer',
              value: function () {
                this._rfbRSAAESAuthenticationState !== null &&
                  this._rfbRSAAESAuthenticationState.approveServer()
              }
            },
            {
              key: 'sendCredentials',
              value: function (d) {
                ;((this._rfbCredentials = d), this._resumeAuthentication())
              }
            },
            {
              key: 'sendCtrlAltDel',
              value: function () {
                this._rfbConnectionState !== 'connected' ||
                  this._viewOnly ||
                  (r.Info('Sending Ctrl-Alt-Del'),
                  this.sendKey(l.default.XK_Control_L, 'ControlLeft', !0),
                  this.sendKey(l.default.XK_Alt_L, 'AltLeft', !0),
                  this.sendKey(l.default.XK_Delete, 'Delete', !0),
                  this.sendKey(l.default.XK_Delete, 'Delete', !1),
                  this.sendKey(l.default.XK_Alt_L, 'AltLeft', !1),
                  this.sendKey(l.default.XK_Control_L, 'ControlLeft', !1))
              }
            },
            {
              key: 'machineShutdown',
              value: function () {
                this._xvpOp(1, 2)
              }
            },
            {
              key: 'machineReboot',
              value: function () {
                this._xvpOp(1, 3)
              }
            },
            {
              key: 'machineReset',
              value: function () {
                this._xvpOp(1, 4)
              }
            },
            {
              key: 'sendKey',
              value: function (d, T, v) {
                if (!(this._rfbConnectionState !== 'connected' || this._viewOnly)) {
                  if (v === void 0) {
                    ;(this.sendKey(d, T, !0), this.sendKey(d, T, !1))
                    return
                  }
                  var H = h.default[T]
                  if (this._qemuExtKeyEventSupported && H)
                    ((d = d || 0),
                      r.Info(
                        'Sending key (' + (v ? 'down' : 'up') + '): keysym ' + d + ', scancode ' + H
                      ),
                      R.messages.QEMUExtendedKeyEvent(this._sock, d, v, H))
                  else {
                    if (!d) return
                    ;(r.Info('Sending keysym (' + (v ? 'down' : 'up') + '): ' + d),
                      R.messages.keyEvent(this._sock, d, v ? 1 : 0))
                  }
                }
              }
            },
            {
              key: 'focus',
              value: function (d) {
                this._canvas.focus(d)
              }
            },
            {
              key: 'blur',
              value: function () {
                this._canvas.blur()
              }
            },
            {
              key: 'clipboardPasteFrom',
              value: function (d) {
                if (!(this._rfbConnectionState !== 'connected' || this._viewOnly))
                  if (
                    this._clipboardServerCapabilitiesFormats[gt] &&
                    this._clipboardServerCapabilitiesActions[fr]
                  )
                    ((this._clipboardText = d),
                      R.messages.extendedClipboardNotify(this._sock, [gt]))
                  else {
                    var T, v, H
                    T = 0
                    var J = ce(d),
                      te
                    try {
                      for (J.s(); !(te = J.n()).done; ) {
                        var ae = te.value
                        T++
                      }
                    } catch (Te) {
                      J.e(Te)
                    } finally {
                      J.f()
                    }
                    ;((H = new Uint8Array(T)), (v = 0))
                    var we = ce(d),
                      Ce
                    try {
                      for (we.s(); !(Ce = we.n()).done; ) {
                        var Xe = Ce.value,
                          Se = Xe.codePointAt(0)
                        ;(Se > 255 && (Se = 63), (H[v++] = Se))
                      }
                    } catch (Te) {
                      we.e(Te)
                    } finally {
                      we.f()
                    }
                    R.messages.clientCutText(this._sock, H)
                  }
              }
            },
            {
              key: 'getImageData',
              value: function () {
                return this._display.getImageData()
              }
            },
            {
              key: 'toDataURL',
              value: function (d, T) {
                return this._display.toDataURL(d, T)
              }
            },
            {
              key: 'toBlob',
              value: function (d, T, v) {
                return this._display.toBlob(d, T, v)
              }
            },
            {
              key: '_connect',
              value: function () {
                if ((r.Debug('>> RFB.connect'), this._url))
                  (r.Info('connecting to '.concat(this._url)),
                    this._sock.open(this._url, this._wsProtocols))
                else {
                  if (
                    (r.Info('attaching '.concat(this._rawChannel, ' to Websock')),
                    this._sock.attach(this._rawChannel),
                    this._sock.readyState === 'closed')
                  )
                    throw Error('Cannot use already closed WebSocket/RTCDataChannel')
                  this._sock.readyState === 'open' && this._socketOpen()
                }
                ;(this._target.appendChild(this._screen),
                  this._gestures.attach(this._canvas),
                  this._cursor.attach(this._canvas),
                  this._refreshCursor(),
                  this._resizeObserver.observe(this._screen),
                  this._canvas.addEventListener('mousedown', this._eventHandlers.focusCanvas),
                  this._canvas.addEventListener('touchstart', this._eventHandlers.focusCanvas),
                  this._canvas.addEventListener('mousedown', this._eventHandlers.handleMouse),
                  this._canvas.addEventListener('mouseup', this._eventHandlers.handleMouse),
                  this._canvas.addEventListener('mousemove', this._eventHandlers.handleMouse),
                  this._canvas.addEventListener('click', this._eventHandlers.handleMouse),
                  this._canvas.addEventListener('contextmenu', this._eventHandlers.handleMouse),
                  this._canvas.addEventListener('wheel', this._eventHandlers.handleWheel),
                  this._canvas.addEventListener('gesturestart', this._eventHandlers.handleGesture),
                  this._canvas.addEventListener('gesturemove', this._eventHandlers.handleGesture),
                  this._canvas.addEventListener('gestureend', this._eventHandlers.handleGesture),
                  r.Debug('<< RFB.connect'))
              }
            },
            {
              key: '_disconnect',
              value: function () {
                ;(r.Debug('>> RFB.disconnect'),
                  this._cursor.detach(),
                  this._canvas.removeEventListener(
                    'gesturestart',
                    this._eventHandlers.handleGesture
                  ),
                  this._canvas.removeEventListener(
                    'gesturemove',
                    this._eventHandlers.handleGesture
                  ),
                  this._canvas.removeEventListener('gestureend', this._eventHandlers.handleGesture),
                  this._canvas.removeEventListener('wheel', this._eventHandlers.handleWheel),
                  this._canvas.removeEventListener('mousedown', this._eventHandlers.handleMouse),
                  this._canvas.removeEventListener('mouseup', this._eventHandlers.handleMouse),
                  this._canvas.removeEventListener('mousemove', this._eventHandlers.handleMouse),
                  this._canvas.removeEventListener('click', this._eventHandlers.handleMouse),
                  this._canvas.removeEventListener('contextmenu', this._eventHandlers.handleMouse),
                  this._canvas.removeEventListener('mousedown', this._eventHandlers.focusCanvas),
                  this._canvas.removeEventListener('touchstart', this._eventHandlers.focusCanvas),
                  this._resizeObserver.disconnect(),
                  this._keyboard.ungrab(),
                  this._gestures.detach(),
                  this._sock.close())
                try {
                  this._target.removeChild(this._screen)
                } catch (d) {
                  if (d.name !== 'NotFoundError') throw d
                }
                ;(clearTimeout(this._resizeTimeout),
                  clearTimeout(this._mouseMoveTimer),
                  r.Debug('<< RFB.disconnect'))
              }
            },
            {
              key: '_socketOpen',
              value: function () {
                this._rfbConnectionState === 'connecting' && this._rfbInitState === ''
                  ? ((this._rfbInitState = 'ProtocolVersion'), r.Debug('Starting VNC handshake'))
                  : this._fail('Unexpected server connection while ' + this._rfbConnectionState)
              }
            },
            {
              key: '_socketClose',
              value: function (d) {
                r.Debug('WebSocket on-close event')
                var T = ''
                switch (
                  (d.code &&
                    ((T = '(code: ' + d.code),
                    d.reason && (T += ', reason: ' + d.reason),
                    (T += ')')),
                  this._rfbConnectionState)
                ) {
                  case 'connecting':
                    this._fail('Connection closed ' + T)
                    break
                  case 'connected':
                    ;(this._updateConnectionState('disconnecting'),
                      this._updateConnectionState('disconnected'))
                    break
                  case 'disconnecting':
                    this._updateConnectionState('disconnected')
                    break
                  case 'disconnected':
                    this._fail('Unexpected server disconnect when already disconnected ' + T)
                    break
                  default:
                    this._fail('Unexpected server disconnect before connecting ' + T)
                    break
                }
                ;(this._sock.off('close'), (this._rawChannel = null))
              }
            },
            {
              key: '_socketError',
              value: function (d) {
                r.Warn('WebSocket on-error event')
              }
            },
            {
              key: '_focusCanvas',
              value: function (d) {
                this.focusOnClick && this.focus({ preventScroll: !0 })
              }
            },
            {
              key: '_setDesktopName',
              value: function (d) {
                ;((this._fbName = d),
                  this.dispatchEvent(
                    new CustomEvent('desktopname', { detail: { name: this._fbName } })
                  ))
              }
            },
            {
              key: '_saveExpectedClientSize',
              value: function () {
                ;((this._expectedClientWidth = this._screen.clientWidth),
                  (this._expectedClientHeight = this._screen.clientHeight))
              }
            },
            {
              key: '_currentClientSize',
              value: function () {
                return [this._screen.clientWidth, this._screen.clientHeight]
              }
            },
            {
              key: '_clientHasExpectedSize',
              value: function () {
                var d = this._currentClientSize(),
                  T = M(d, 2),
                  v = T[0],
                  H = T[1]
                return v == this._expectedClientWidth && H == this._expectedClientHeight
              }
            },
            {
              key: '_handleResize',
              value: function () {
                var d = this
                this._clientHasExpectedSize() ||
                  (window.requestAnimationFrame(function () {
                    ;(d._updateClip(), d._updateScale(), d._saveExpectedClientSize())
                  }),
                  this._requestRemoteResize())
              }
            },
            {
              key: '_updateClip',
              value: function () {
                var d = this._display.clipViewport,
                  T = this._clipViewport
                if (
                  (this._scaleViewport && (T = !1), d !== T && (this._display.clipViewport = T), T)
                ) {
                  var v = this._screenSize()
                  ;(this._display.viewportChangeSize(v.w, v.h),
                    this._fixScrollbars(),
                    this._setClippingViewport(
                      v.w < this._display.width || v.h < this._display.height
                    ))
                } else this._setClippingViewport(!1)
                d !== T && this._saveExpectedClientSize()
              }
            },
            {
              key: '_updateScale',
              value: function () {
                if (!this._scaleViewport) this._display.scale = 1
                else {
                  var d = this._screenSize()
                  this._display.autoscale(d.w, d.h)
                }
                this._fixScrollbars()
              }
            },
            {
              key: '_requestRemoteResize',
              value: function () {
                if (
                  this._resizeSession &&
                  !this._viewOnly &&
                  this._supportsSetDesktopSize &&
                  !this._pendingRemoteResize
                ) {
                  if (Date.now() - this._lastResize < 100) {
                    ;(clearTimeout(this._resizeTimeout),
                      (this._resizeTimeout = setTimeout(
                        this._requestRemoteResize.bind(this),
                        100 - (Date.now() - this._lastResize)
                      )))
                    return
                  }
                  this._resizeTimeout = null
                  var d = this._screenSize()
                  ;(d.w === this._fbWidth && d.h === this._fbHeight) ||
                    ((this._pendingRemoteResize = !0),
                    (this._lastResize = Date.now()),
                    R.messages.setDesktopSize(
                      this._sock,
                      Math.floor(d.w),
                      Math.floor(d.h),
                      this._screenID,
                      this._screenFlags
                    ),
                    r.Debug('Requested new desktop size: ' + d.w + 'x' + d.h))
                }
              }
            },
            {
              key: '_screenSize',
              value: function () {
                var d = this._screen.getBoundingClientRect()
                return { w: d.width, h: d.height }
              }
            },
            {
              key: '_fixScrollbars',
              value: function () {
                var d = this._screen.style.overflow
                ;((this._screen.style.overflow = 'hidden'),
                  this._screen.getBoundingClientRect(),
                  (this._screen.style.overflow = d))
              }
            },
            {
              key: '_updateConnectionState',
              value: function (d) {
                var T = this,
                  v = this._rfbConnectionState
                if (d === v) {
                  r.Debug("Already in state '" + d + "', ignoring")
                  return
                }
                if (v === 'disconnected') {
                  r.Error('Tried changing state of a disconnected RFB object')
                  return
                }
                switch (d) {
                  case 'connected':
                    if (v !== 'connecting') {
                      r.Error('Bad transition to connected state, previous connection state: ' + v)
                      return
                    }
                    break
                  case 'disconnected':
                    if (v !== 'disconnecting') {
                      r.Error(
                        'Bad transition to disconnected state, previous connection state: ' + v
                      )
                      return
                    }
                    break
                  case 'connecting':
                    if (v !== '') {
                      r.Error('Bad transition to connecting state, previous connection state: ' + v)
                      return
                    }
                    break
                  case 'disconnecting':
                    if (v !== 'connected' && v !== 'connecting') {
                      r.Error(
                        'Bad transition to disconnecting state, previous connection state: ' + v
                      )
                      return
                    }
                    break
                  default:
                    r.Error('Unknown connection state: ' + d)
                    return
                }
                switch (
                  ((this._rfbConnectionState = d),
                  r.Debug("New state '" + d + "', was '" + v + "'."),
                  this._disconnTimer &&
                    d !== 'disconnecting' &&
                    (r.Debug('Clearing disconnect timer'),
                    clearTimeout(this._disconnTimer),
                    (this._disconnTimer = null),
                    this._sock.off('close')),
                  d)
                ) {
                  case 'connecting':
                    this._connect()
                    break
                  case 'connected':
                    this.dispatchEvent(new CustomEvent('connect', { detail: {} }))
                    break
                  case 'disconnecting':
                    ;(this._disconnect(),
                      (this._disconnTimer = setTimeout(function () {
                        ;(r.Error('Disconnection timed out.'),
                          T._updateConnectionState('disconnected'))
                      }, ie * 1e3)))
                    break
                  case 'disconnected':
                    this.dispatchEvent(
                      new CustomEvent('disconnect', { detail: { clean: this._rfbCleanDisconnect } })
                    )
                    break
                }
              }
            },
            {
              key: '_fail',
              value: function (d) {
                switch (this._rfbConnectionState) {
                  case 'disconnecting':
                    r.Error('Failed when disconnecting: ' + d)
                    break
                  case 'connected':
                    r.Error('Failed while connected: ' + d)
                    break
                  case 'connecting':
                    r.Error('Failed when connecting: ' + d)
                    break
                  default:
                    r.Error('RFB failure: ' + d)
                    break
                }
                return (
                  (this._rfbCleanDisconnect = !1),
                  this._updateConnectionState('disconnecting'),
                  this._updateConnectionState('disconnected'),
                  !1
                )
              }
            },
            {
              key: '_setCapability',
              value: function (d, T) {
                ;((this._capabilities[d] = T),
                  this.dispatchEvent(
                    new CustomEvent('capabilities', {
                      detail: { capabilities: this._capabilities }
                    })
                  ))
              }
            },
            {
              key: '_handleMessage',
              value: function () {
                if (this._sock.rQwait('message', 1)) {
                  r.Warn('handleMessage called on an empty receive queue')
                  return
                }
                switch (this._rfbConnectionState) {
                  case 'disconnected':
                    r.Error('Got data while disconnected')
                    break
                  case 'connected':
                    for (
                      ;
                      !(this._flushing || !this._normalMsg() || this._sock.rQwait('message', 1));

                    );
                    break
                  case 'connecting':
                    for (; this._rfbConnectionState === 'connecting' && this._initMsg(); );
                    break
                  default:
                    r.Error('Got data while in an invalid state')
                    break
                }
              }
            },
            {
              key: '_handleKeyEvent',
              value: function (d, T, v, H, J) {
                ;(T == 'CapsLock' && v && (this._remoteCapsLock = null),
                  this._remoteCapsLock !== null &&
                    J !== null &&
                    this._remoteCapsLock !== J &&
                    v &&
                    (r.Debug('Fixing remote caps lock'),
                    this.sendKey(l.default.XK_Caps_Lock, 'CapsLock', !0),
                    this.sendKey(l.default.XK_Caps_Lock, 'CapsLock', !1),
                    (this._remoteCapsLock = null)),
                  T == 'NumLock' && v && (this._remoteNumLock = null),
                  this._remoteNumLock !== null &&
                    H !== null &&
                    this._remoteNumLock !== H &&
                    v &&
                    (r.Debug('Fixing remote num lock'),
                    this.sendKey(l.default.XK_Num_Lock, 'NumLock', !0),
                    this.sendKey(l.default.XK_Num_Lock, 'NumLock', !1),
                    (this._remoteNumLock = null)),
                  this.sendKey(d, T, v))
              }
            },
            {
              key: '_handleMouse',
              value: function (d) {
                if (
                  !(d.type === 'click' && d.target !== this._canvas) &&
                  (d.stopPropagation(),
                  d.preventDefault(),
                  !(d.type === 'click' || d.type === 'contextmenu'))
                ) {
                  var T = (0, a.clientToElement)(d.clientX, d.clientY, this._canvas),
                    v = R._convertButtonMask(d.buttons),
                    H = d.type == 'mousedown'
                  switch (d.type) {
                    case 'mousedown':
                    case 'mouseup':
                      if (this.dragViewport)
                        if (H && !this._viewportDragging) {
                          ;((this._viewportDragging = !0),
                            (this._viewportDragPos = { x: T.x, y: T.y }),
                            (this._viewportHasMoved = !1),
                            this._flushMouseMoveTimer(T.x, T.y),
                            (this._mouseButtonMask = v))
                          break
                        } else {
                          if (((this._viewportDragging = !1), this._viewportHasMoved)) {
                            this._mouseButtonMask = v
                            break
                          }
                          this._sendMouse(T.x, T.y, this._mouseButtonMask)
                        }
                      ;(H && (0, s.setCapture)(this._canvas), this._handleMouseButton(T.x, T.y, v))
                      break
                    case 'mousemove':
                      if (this._viewportDragging) {
                        var J = this._viewportDragPos.x - T.x,
                          te = this._viewportDragPos.y - T.y
                        ;(this._viewportHasMoved ||
                          Math.abs(J) > o.dragThreshold ||
                          Math.abs(te) > o.dragThreshold) &&
                          ((this._viewportHasMoved = !0),
                          (this._viewportDragPos = { x: T.x, y: T.y }),
                          this._display.viewportChangePos(J, te))
                        break
                      }
                      this._handleMouseMove(T.x, T.y)
                      break
                  }
                }
              }
            },
            {
              key: '_handleMouseButton',
              value: function (d, T, v) {
                ;(this._flushMouseMoveTimer(d, T),
                  (this._mouseButtonMask = v),
                  this._sendMouse(d, T, this._mouseButtonMask))
              }
            },
            {
              key: '_handleMouseMove',
              value: function (d, T) {
                var v = this
                if (((this._mousePos = { x: d, y: T }), this._mouseMoveTimer == null)) {
                  var H = Date.now() - this._mouseLastMoveTime
                  H > ve
                    ? (this._sendMouse(d, T, this._mouseButtonMask),
                      (this._mouseLastMoveTime = Date.now()))
                    : (this._mouseMoveTimer = setTimeout(function () {
                        v._handleDelayedMouseMove()
                      }, ve - H))
                }
              }
            },
            {
              key: '_handleDelayedMouseMove',
              value: function () {
                ;((this._mouseMoveTimer = null),
                  this._sendMouse(this._mousePos.x, this._mousePos.y, this._mouseButtonMask),
                  (this._mouseLastMoveTime = Date.now()))
              }
            },
            {
              key: '_sendMouse',
              value: function (d, T, v) {
                if (this._rfbConnectionState === 'connected' && !this._viewOnly) {
                  if (v & 32768) throw new Error('Illegal mouse button mask (mask: ' + v + ')')
                  var H = v & 32640
                  this._extendedPointerEventSupported && H
                    ? R.messages.extendedPointerEvent(
                        this._sock,
                        this._display.absX(d),
                        this._display.absY(T),
                        v
                      )
                    : R.messages.pointerEvent(
                        this._sock,
                        this._display.absX(d),
                        this._display.absY(T),
                        v
                      )
                }
              }
            },
            {
              key: '_handleWheel',
              value: function (d) {
                if (this._rfbConnectionState === 'connected' && !this._viewOnly) {
                  ;(d.stopPropagation(), d.preventDefault())
                  var T = (0, a.clientToElement)(d.clientX, d.clientY, this._canvas),
                    v = R._convertButtonMask(d.buttons),
                    H = d.deltaX,
                    J = d.deltaY
                  ;(d.deltaMode !== 0 && ((H *= Pe), (J *= Pe)),
                    (this._accumulatedWheelDeltaX += H),
                    (this._accumulatedWheelDeltaY += J),
                    Math.abs(this._accumulatedWheelDeltaX) >= de &&
                      (this._accumulatedWheelDeltaX < 0
                        ? (this._handleMouseButton(T.x, T.y, v | 32),
                          this._handleMouseButton(T.x, T.y, v))
                        : this._accumulatedWheelDeltaX > 0 &&
                          (this._handleMouseButton(T.x, T.y, v | 64),
                          this._handleMouseButton(T.x, T.y, v)),
                      (this._accumulatedWheelDeltaX = 0)),
                    Math.abs(this._accumulatedWheelDeltaY) >= de &&
                      (this._accumulatedWheelDeltaY < 0
                        ? (this._handleMouseButton(T.x, T.y, v | 8),
                          this._handleMouseButton(T.x, T.y, v))
                        : this._accumulatedWheelDeltaY > 0 &&
                          (this._handleMouseButton(T.x, T.y, v | 16),
                          this._handleMouseButton(T.x, T.y, v)),
                      (this._accumulatedWheelDeltaY = 0)))
                }
              }
            },
            {
              key: '_fakeMouseMove',
              value: function (d, T, v) {
                ;(this._handleMouseMove(T, v),
                  this._cursor.move(d.detail.clientX, d.detail.clientY))
              }
            },
            {
              key: '_handleTapEvent',
              value: function (d, T) {
                var v = (0, a.clientToElement)(d.detail.clientX, d.detail.clientY, this._canvas)
                if (
                  this._gestureLastTapTime !== null &&
                  Date.now() - this._gestureLastTapTime < le &&
                  this._gestureFirstDoubleTapEv.detail.type === d.detail.type
                ) {
                  var H = this._gestureFirstDoubleTapEv.detail.clientX - d.detail.clientX,
                    J = this._gestureFirstDoubleTapEv.detail.clientY - d.detail.clientY,
                    te = Math.hypot(H, J)
                  te < yt
                    ? (v = (0, a.clientToElement)(
                        this._gestureFirstDoubleTapEv.detail.clientX,
                        this._gestureFirstDoubleTapEv.detail.clientY,
                        this._canvas
                      ))
                    : (this._gestureFirstDoubleTapEv = d)
                } else this._gestureFirstDoubleTapEv = d
                ;((this._gestureLastTapTime = Date.now()),
                  this._fakeMouseMove(this._gestureFirstDoubleTapEv, v.x, v.y),
                  this._handleMouseButton(v.x, v.y, T),
                  this._handleMouseButton(v.x, v.y, 0))
              }
            },
            {
              key: '_handleGesture',
              value: function (d) {
                var T,
                  v = (0, a.clientToElement)(d.detail.clientX, d.detail.clientY, this._canvas)
                switch (d.type) {
                  case 'gesturestart':
                    switch (d.detail.type) {
                      case 'onetap':
                        this._handleTapEvent(d, 1)
                        break
                      case 'twotap':
                        this._handleTapEvent(d, 4)
                        break
                      case 'threetap':
                        this._handleTapEvent(d, 2)
                        break
                      case 'drag':
                        this.dragViewport
                          ? ((this._viewportHasMoved = !1),
                            (this._viewportDragging = !0),
                            (this._viewportDragPos = { x: v.x, y: v.y }))
                          : (this._fakeMouseMove(d, v.x, v.y), this._handleMouseButton(v.x, v.y, 1))
                        break
                      case 'longpress':
                        this.dragViewport
                          ? ((this._viewportHasMoved = !1),
                            (this._viewportDragPos = { x: v.x, y: v.y }))
                          : (this._fakeMouseMove(d, v.x, v.y), this._handleMouseButton(v.x, v.y, 4))
                        break
                      case 'twodrag':
                        ;((this._gestureLastMagnitudeX = d.detail.magnitudeX),
                          (this._gestureLastMagnitudeY = d.detail.magnitudeY),
                          this._fakeMouseMove(d, v.x, v.y))
                        break
                      case 'pinch':
                        ;((this._gestureLastMagnitudeX = Math.hypot(
                          d.detail.magnitudeX,
                          d.detail.magnitudeY
                        )),
                          this._fakeMouseMove(d, v.x, v.y))
                        break
                    }
                    break
                  case 'gesturemove':
                    switch (d.detail.type) {
                      case 'onetap':
                      case 'twotap':
                      case 'threetap':
                        break
                      case 'drag':
                      case 'longpress':
                        if (this.dragViewport) {
                          this._viewportDragging = !0
                          var H = this._viewportDragPos.x - v.x,
                            J = this._viewportDragPos.y - v.y
                          ;(this._viewportHasMoved ||
                            Math.abs(H) > o.dragThreshold ||
                            Math.abs(J) > o.dragThreshold) &&
                            ((this._viewportHasMoved = !0),
                            (this._viewportDragPos = { x: v.x, y: v.y }),
                            this._display.viewportChangePos(H, J))
                        } else this._fakeMouseMove(d, v.x, v.y)
                        break
                      case 'twodrag':
                        for (
                          this._fakeMouseMove(d, v.x, v.y);
                          d.detail.magnitudeY - this._gestureLastMagnitudeY > Oe;

                        )
                          (this._handleMouseButton(v.x, v.y, 8),
                            this._handleMouseButton(v.x, v.y, 0),
                            (this._gestureLastMagnitudeY += Oe))
                        for (; d.detail.magnitudeY - this._gestureLastMagnitudeY < -Oe; )
                          (this._handleMouseButton(v.x, v.y, 16),
                            this._handleMouseButton(v.x, v.y, 0),
                            (this._gestureLastMagnitudeY -= Oe))
                        for (; d.detail.magnitudeX - this._gestureLastMagnitudeX > Oe; )
                          (this._handleMouseButton(v.x, v.y, 32),
                            this._handleMouseButton(v.x, v.y, 0),
                            (this._gestureLastMagnitudeX += Oe))
                        for (; d.detail.magnitudeX - this._gestureLastMagnitudeX < -Oe; )
                          (this._handleMouseButton(v.x, v.y, 64),
                            this._handleMouseButton(v.x, v.y, 0),
                            (this._gestureLastMagnitudeX -= Oe))
                        break
                      case 'pinch':
                        if (
                          (this._fakeMouseMove(d, v.x, v.y),
                          (T = Math.hypot(d.detail.magnitudeX, d.detail.magnitudeY)),
                          Math.abs(T - this._gestureLastMagnitudeX) > Me)
                        ) {
                          for (
                            this._handleKeyEvent(l.default.XK_Control_L, 'ControlLeft', !0);
                            T - this._gestureLastMagnitudeX > Me;

                          )
                            (this._handleMouseButton(v.x, v.y, 8),
                              this._handleMouseButton(v.x, v.y, 0),
                              (this._gestureLastMagnitudeX += Me))
                          for (; T - this._gestureLastMagnitudeX < -Me; )
                            (this._handleMouseButton(v.x, v.y, 16),
                              this._handleMouseButton(v.x, v.y, 0),
                              (this._gestureLastMagnitudeX -= Me))
                        }
                        this._handleKeyEvent(l.default.XK_Control_L, 'ControlLeft', !1)
                        break
                    }
                    break
                  case 'gestureend':
                    switch (d.detail.type) {
                      case 'onetap':
                      case 'twotap':
                      case 'threetap':
                      case 'pinch':
                      case 'twodrag':
                        break
                      case 'drag':
                        this.dragViewport
                          ? (this._viewportDragging = !1)
                          : (this._fakeMouseMove(d, v.x, v.y), this._handleMouseButton(v.x, v.y, 0))
                        break
                      case 'longpress':
                        if (this._viewportHasMoved) break
                        this.dragViewport && !this._viewportHasMoved
                          ? (this._fakeMouseMove(d, v.x, v.y),
                            this._handleMouseButton(v.x, v.y, 4),
                            this._handleMouseButton(v.x, v.y, 0),
                            (this._viewportDragging = !1))
                          : (this._fakeMouseMove(d, v.x, v.y), this._handleMouseButton(v.x, v.y, 0))
                        break
                    }
                    break
                }
              }
            },
            {
              key: '_flushMouseMoveTimer',
              value: function (d, T) {
                this._mouseMoveTimer !== null &&
                  (clearTimeout(this._mouseMoveTimer),
                  (this._mouseMoveTimer = null),
                  this._sendMouse(d, T, this._mouseButtonMask))
              }
            },
            {
              key: '_negotiateProtocolVersion',
              value: function () {
                if (this._sock.rQwait('version', 12)) return !1
                var d = this._sock.rQshiftStr(12).substr(4, 7)
                r.Info('Server ProtocolVersion: ' + d)
                var T = 0
                switch (d) {
                  case '000.000':
                    T = 1
                    break
                  case '003.003':
                  case '003.006':
                    this._rfbVersion = 3.3
                    break
                  case '003.007':
                    this._rfbVersion = 3.7
                    break
                  case '003.008':
                  case '003.889':
                  case '004.000':
                  case '004.001':
                  case '005.000':
                    this._rfbVersion = 3.8
                    break
                  default:
                    return this._fail('Invalid server version ' + d)
                }
                if (T) {
                  for (var v = 'ID:' + this._repeaterID; v.length < 250; ) v += '\0'
                  return (this._sock.sQpushString(v), this._sock.flush(), !0)
                }
                this._rfbVersion > this._rfbMaxVersion && (this._rfbVersion = this._rfbMaxVersion)
                var H =
                  '00' + parseInt(this._rfbVersion, 10) + '.00' + ((this._rfbVersion * 10) % 10)
                ;(this._sock.sQpushString(
                  'RFB ' +
                    H +
                    `
`
                ),
                  this._sock.flush(),
                  r.Debug('Sent ProtocolVersion: ' + H),
                  (this._rfbInitState = 'Security'))
              }
            },
            {
              key: '_isSupportedSecurityType',
              value: function (d) {
                var T = [xt, Lr, Zn, qn, rn, Yn, $n, Jn, ti]
                return T.includes(d)
              }
            },
            {
              key: '_negotiateSecurity',
              value: function () {
                if (this._rfbVersion >= 3.7) {
                  var d = this._sock.rQshift8()
                  if (this._sock.rQwait('security type', d, 1)) return !1
                  if (d === 0)
                    return (
                      (this._rfbInitState = 'SecurityReason'),
                      (this._securityContext = 'no security types'),
                      (this._securityStatus = 1),
                      !0
                    )
                  var T = this._sock.rQshiftBytes(d)
                  ;(r.Debug('Server security types: ' + T), (this._rfbAuthScheme = -1))
                  var v = ce(T),
                    H
                  try {
                    for (v.s(); !(H = v.n()).done; ) {
                      var J = H.value
                      if (this._isSupportedSecurityType(J)) {
                        this._rfbAuthScheme = J
                        break
                      }
                    }
                  } catch (te) {
                    v.e(te)
                  } finally {
                    v.f()
                  }
                  if (this._rfbAuthScheme === -1)
                    return this._fail('Unsupported security types (types: ' + T + ')')
                  ;(this._sock.sQpush8(this._rfbAuthScheme), this._sock.flush())
                } else {
                  if (this._sock.rQwait('security scheme', 4)) return !1
                  if (((this._rfbAuthScheme = this._sock.rQshift32()), this._rfbAuthScheme == 0))
                    return (
                      (this._rfbInitState = 'SecurityReason'),
                      (this._securityContext = 'authentication scheme'),
                      (this._securityStatus = 1),
                      !0
                    )
                }
                return (
                  (this._rfbInitState = 'Authentication'),
                  r.Debug('Authenticating using scheme: ' + this._rfbAuthScheme),
                  !0
                )
              }
            },
            {
              key: '_handleSecurityReason',
              value: function () {
                if (this._sock.rQwait('reason length', 4)) return !1
                var d = this._sock.rQshift32(),
                  T = ''
                if (d > 0) {
                  if (this._sock.rQwait('reason', d, 4)) return !1
                  T = this._sock.rQshiftStr(d)
                }
                return T !== ''
                  ? (this.dispatchEvent(
                      new CustomEvent('securityfailure', {
                        detail: { status: this._securityStatus, reason: T }
                      })
                    ),
                    this._fail(
                      'Security negotiation failed on ' +
                        this._securityContext +
                        ' (reason: ' +
                        T +
                        ')'
                    ))
                  : (this.dispatchEvent(
                      new CustomEvent('securityfailure', {
                        detail: { status: this._securityStatus }
                      })
                    ),
                    this._fail('Security negotiation failed on ' + this._securityContext))
              }
            },
            {
              key: '_negotiateXvpAuth',
              value: function () {
                return this._rfbCredentials.username === void 0 ||
                  this._rfbCredentials.password === void 0 ||
                  this._rfbCredentials.target === void 0
                  ? (this.dispatchEvent(
                      new CustomEvent('credentialsrequired', {
                        detail: { types: ['username', 'password', 'target'] }
                      })
                    ),
                    !1)
                  : (this._sock.sQpush8(this._rfbCredentials.username.length),
                    this._sock.sQpush8(this._rfbCredentials.target.length),
                    this._sock.sQpushString(this._rfbCredentials.username),
                    this._sock.sQpushString(this._rfbCredentials.target),
                    this._sock.flush(),
                    (this._rfbAuthScheme = Lr),
                    this._negotiateAuthentication())
              }
            },
            {
              key: '_negotiateVeNCryptAuth',
              value: function () {
                if (this._rfbVeNCryptState == 0) {
                  if (this._sock.rQwait('vencrypt version', 2)) return !1
                  var d = this._sock.rQshift8(),
                    T = this._sock.rQshift8()
                  if (!(d == 0 && T == 2))
                    return this._fail('Unsupported VeNCrypt version ' + d + '.' + T)
                  ;(this._sock.sQpush8(0),
                    this._sock.sQpush8(2),
                    this._sock.flush(),
                    (this._rfbVeNCryptState = 1))
                }
                if (this._rfbVeNCryptState == 1) {
                  if (this._sock.rQwait('vencrypt ack', 1)) return !1
                  var v = this._sock.rQshift8()
                  if (v != 0) return this._fail('VeNCrypt failure ' + v)
                  this._rfbVeNCryptState = 2
                }
                if (this._rfbVeNCryptState == 2) {
                  if (this._sock.rQwait('vencrypt subtypes length', 1)) return !1
                  var H = this._sock.rQshift8()
                  if (H < 1) return this._fail('VeNCrypt subtypes empty')
                  ;((this._rfbVeNCryptSubtypesLength = H), (this._rfbVeNCryptState = 3))
                }
                if (this._rfbVeNCryptState == 3) {
                  if (this._sock.rQwait('vencrypt subtypes', 4 * this._rfbVeNCryptSubtypesLength))
                    return !1
                  for (var J = [], te = 0; te < this._rfbVeNCryptSubtypesLength; te++)
                    J.push(this._sock.rQshift32())
                  this._rfbAuthScheme = -1
                  for (var ae = 0, we = J; ae < we.length; ae++) {
                    var Ce = we[ae]
                    if (Ce !== rn && this._isSupportedSecurityType(Ce)) {
                      this._rfbAuthScheme = Ce
                      break
                    }
                  }
                  return this._rfbAuthScheme === -1
                    ? this._fail('Unsupported security types (types: ' + J + ')')
                    : (this._sock.sQpush32(this._rfbAuthScheme),
                      this._sock.flush(),
                      (this._rfbVeNCryptState = 4),
                      !0)
                }
              }
            },
            {
              key: '_negotiatePlainAuth',
              value: function () {
                if (
                  this._rfbCredentials.username === void 0 ||
                  this._rfbCredentials.password === void 0
                )
                  return (
                    this.dispatchEvent(
                      new CustomEvent('credentialsrequired', {
                        detail: { types: ['username', 'password'] }
                      })
                    ),
                    !1
                  )
                var d = (0, i.encodeUTF8)(this._rfbCredentials.username),
                  T = (0, i.encodeUTF8)(this._rfbCredentials.password)
                return (
                  this._sock.sQpush32(d.length),
                  this._sock.sQpush32(T.length),
                  this._sock.sQpushString(d),
                  this._sock.sQpushString(T),
                  this._sock.flush(),
                  (this._rfbInitState = 'SecurityResult'),
                  !0
                )
              }
            },
            {
              key: '_negotiateStdVNCAuth',
              value: function () {
                if (this._sock.rQwait('auth challenge', 16)) return !1
                if (this._rfbCredentials.password === void 0)
                  return (
                    this.dispatchEvent(
                      new CustomEvent('credentialsrequired', { detail: { types: ['password'] } })
                    ),
                    !1
                  )
                var d = Array.prototype.slice.call(this._sock.rQshiftBytes(16)),
                  T = R.genDES(this._rfbCredentials.password, d)
                return (
                  this._sock.sQpushBytes(T),
                  this._sock.flush(),
                  (this._rfbInitState = 'SecurityResult'),
                  !0
                )
              }
            },
            {
              key: '_negotiateARDAuth',
              value: function () {
                if (
                  this._rfbCredentials.username === void 0 ||
                  this._rfbCredentials.password === void 0
                )
                  return (
                    this.dispatchEvent(
                      new CustomEvent('credentialsrequired', {
                        detail: { types: ['username', 'password'] }
                      })
                    ),
                    !1
                  )
                if (
                  this._rfbCredentials.ardPublicKey != null &&
                  this._rfbCredentials.ardCredentials != null
                )
                  return (
                    this._sock.sQpushBytes(this._rfbCredentials.ardCredentials),
                    this._sock.sQpushBytes(this._rfbCredentials.ardPublicKey),
                    this._sock.flush(),
                    (this._rfbCredentials.ardCredentials = null),
                    (this._rfbCredentials.ardPublicKey = null),
                    (this._rfbInitState = 'SecurityResult'),
                    !0
                  )
                if (this._sock.rQwait('read ard', 4)) return !1
                var d = this._sock.rQshiftBytes(2),
                  T = this._sock.rQshift16()
                if (this._sock.rQwait('read ard keylength', T * 2, 4)) return !1
                var v = this._sock.rQshiftBytes(T),
                  H = this._sock.rQshiftBytes(T),
                  J = y.default.generateKey({ name: 'DH', g: d, p: v }, !1, ['deriveBits'])
                return (this._negotiateARDAuthAsync(T, H, J), !1)
              }
            },
            {
              key: '_negotiateARDAuthAsync',
              value: (function () {
                var D = B(
                  b().mark(function T(v, H, J) {
                    var te, ae, we, Ce, Xe, Se, Te, _e, ke, We
                    return b().wrap(
                      function (Le) {
                        for (;;)
                          switch ((Le.prev = Le.next)) {
                            case 0:
                              for (
                                te = y.default.exportKey('raw', J.publicKey),
                                  ae = y.default.deriveBits(
                                    { name: 'DH', public: H },
                                    J.privateKey,
                                    v * 8
                                  ),
                                  we = (0, i.encodeUTF8)(this._rfbCredentials.username).substring(
                                    0,
                                    63
                                  ),
                                  Ce = (0, i.encodeUTF8)(this._rfbCredentials.password).substring(
                                    0,
                                    63
                                  ),
                                  Xe = window.crypto.getRandomValues(new Uint8Array(128)),
                                  Se = 0;
                                Se < we.length;
                                Se++
                              )
                                Xe[Se] = we.charCodeAt(Se)
                              for (Xe[we.length] = 0, Te = 0; Te < Ce.length; Te++)
                                Xe[64 + Te] = Ce.charCodeAt(Te)
                              return (
                                (Xe[64 + Ce.length] = 0),
                                (Le.next = 11),
                                y.default.digest('MD5', ae)
                              )
                            case 11:
                              return (
                                (_e = Le.sent),
                                (Le.next = 14),
                                y.default.importKey('raw', _e, { name: 'AES-ECB' }, !1, ['encrypt'])
                              )
                            case 14:
                              return (
                                (ke = Le.sent),
                                (Le.next = 17),
                                y.default.encrypt({ name: 'AES-ECB' }, ke, Xe)
                              )
                            case 17:
                              ;((We = Le.sent),
                                (this._rfbCredentials.ardCredentials = We),
                                (this._rfbCredentials.ardPublicKey = te),
                                this._resumeAuthentication())
                            case 21:
                            case 'end':
                              return Le.stop()
                          }
                      },
                      T,
                      this
                    )
                  })
                )
                function d(T, v, H) {
                  return D.apply(this, arguments)
                }
                return d
              })()
            },
            {
              key: '_negotiateTightUnixAuth',
              value: function () {
                return this._rfbCredentials.username === void 0 ||
                  this._rfbCredentials.password === void 0
                  ? (this.dispatchEvent(
                      new CustomEvent('credentialsrequired', {
                        detail: { types: ['username', 'password'] }
                      })
                    ),
                    !1)
                  : (this._sock.sQpush32(this._rfbCredentials.username.length),
                    this._sock.sQpush32(this._rfbCredentials.password.length),
                    this._sock.sQpushString(this._rfbCredentials.username),
                    this._sock.sQpushString(this._rfbCredentials.password),
                    this._sock.flush(),
                    (this._rfbInitState = 'SecurityResult'),
                    !0)
              }
            },
            {
              key: '_negotiateTightTunnels',
              value: function (d) {
                for (
                  var T = { 0: { vendor: 'TGHT', signature: 'NOTUNNEL' } }, v = {}, H = 0;
                  H < d;
                  H++
                ) {
                  var J = this._sock.rQshift32(),
                    te = this._sock.rQshiftStr(4),
                    ae = this._sock.rQshiftStr(8)
                  v[J] = { vendor: te, signature: ae }
                }
                return (
                  r.Debug('Server Tight tunnel types: ' + v),
                  v[1] &&
                    v[1].vendor === 'SICR' &&
                    v[1].signature === 'SCHANNEL' &&
                    (r.Debug('Detected Siemens server. Assuming NOTUNNEL support.'),
                    (v[0] = { vendor: 'TGHT', signature: 'NOTUNNEL' })),
                  v[0]
                    ? v[0].vendor != T[0].vendor || v[0].signature != T[0].signature
                      ? this._fail("Client's tunnel type had the incorrect vendor or signature")
                      : (r.Debug('Selected tunnel type: ' + T[0]),
                        this._sock.sQpush32(0),
                        this._sock.flush(),
                        !1)
                    : this._fail("Server wanted tunnels, but doesn't support the notunnel type")
                )
              }
            },
            {
              key: '_negotiateTightAuth',
              value: function () {
                if (!this._rfbTightVNC) {
                  if (this._sock.rQwait('num tunnels', 4)) return !1
                  var d = this._sock.rQshift32()
                  if (d > 0 && this._sock.rQwait('tunnel capabilities', 16 * d, 4)) return !1
                  if (((this._rfbTightVNC = !0), d > 0)) return (this._negotiateTightTunnels(d), !1)
                }
                if (this._sock.rQwait('sub auth count', 4)) return !1
                var T = this._sock.rQshift32()
                if (T === 0) return ((this._rfbInitState = 'SecurityResult'), !0)
                if (this._sock.rQwait('sub auth capabilities', 16 * T, 4)) return !1
                for (
                  var v = { STDVNOAUTH__: 1, STDVVNCAUTH_: 2, TGHTULGNAUTH: 129 }, H = [], J = 0;
                  J < T;
                  J++
                ) {
                  this._sock.rQshift32()
                  var te = this._sock.rQshiftStr(12)
                  H.push(te)
                }
                r.Debug('Server Tight authentication types: ' + H)
                for (var ae in v)
                  if (H.indexOf(ae) != -1)
                    switch (
                      (this._sock.sQpush32(v[ae]),
                      this._sock.flush(),
                      r.Debug('Selected authentication type: ' + ae),
                      ae)
                    ) {
                      case 'STDVNOAUTH__':
                        return ((this._rfbInitState = 'SecurityResult'), !0)
                      case 'STDVVNCAUTH_':
                        return ((this._rfbAuthScheme = Lr), !0)
                      case 'TGHTULGNAUTH':
                        return ((this._rfbAuthScheme = ei), !0)
                      default:
                        return this._fail('Unsupported tiny auth scheme (scheme: ' + ae + ')')
                    }
                return this._fail('No supported sub-auth types!')
              }
            },
            {
              key: '_handleRSAAESCredentialsRequired',
              value: function (d) {
                this.dispatchEvent(d)
              }
            },
            {
              key: '_handleRSAAESServerVerification',
              value: function (d) {
                this.dispatchEvent(d)
              }
            },
            {
              key: '_negotiateRA2neAuth',
              value: function () {
                var d = this
                return (
                  this._rfbRSAAESAuthenticationState === null &&
                    ((this._rfbRSAAESAuthenticationState = new E.default(this._sock, function () {
                      return d._rfbCredentials
                    })),
                    this._rfbRSAAESAuthenticationState.addEventListener(
                      'serververification',
                      this._eventHandlers.handleRSAAESServerVerification
                    ),
                    this._rfbRSAAESAuthenticationState.addEventListener(
                      'credentialsrequired',
                      this._eventHandlers.handleRSAAESCredentialsRequired
                    )),
                  this._rfbRSAAESAuthenticationState.checkInternalEvents(),
                  this._rfbRSAAESAuthenticationState.hasStarted ||
                    this._rfbRSAAESAuthenticationState
                      .negotiateRA2neAuthAsync()
                      .catch(function (T) {
                        T.message !== 'disconnect normally' && d._fail(T.message)
                      })
                      .then(function () {
                        return ((d._rfbInitState = 'SecurityResult'), !0)
                      })
                      .finally(function () {
                        ;(d._rfbRSAAESAuthenticationState.removeEventListener(
                          'serververification',
                          d._eventHandlers.handleRSAAESServerVerification
                        ),
                          d._rfbRSAAESAuthenticationState.removeEventListener(
                            'credentialsrequired',
                            d._eventHandlers.handleRSAAESCredentialsRequired
                          ),
                          (d._rfbRSAAESAuthenticationState = null))
                      }),
                  !1
                )
              }
            },
            {
              key: '_negotiateMSLogonIIAuth',
              value: function () {
                if (this._sock.rQwait('mslogonii dh param', 24)) return !1
                if (
                  this._rfbCredentials.username === void 0 ||
                  this._rfbCredentials.password === void 0
                )
                  return (
                    this.dispatchEvent(
                      new CustomEvent('credentialsrequired', {
                        detail: { types: ['username', 'password'] }
                      })
                    ),
                    !1
                  )
                var d = this._sock.rQshiftBytes(8),
                  T = this._sock.rQshiftBytes(8),
                  v = this._sock.rQshiftBytes(8),
                  H = y.default.generateKey({ name: 'DH', g: d, p: T }, !0, ['deriveBits']),
                  J = y.default.exportKey('raw', H.publicKey),
                  te = y.default.deriveBits({ name: 'DH', public: v }, H.privateKey, 64),
                  ae = y.default.importKey('raw', te, { name: 'DES-CBC' }, !1, ['encrypt']),
                  we = (0, i.encodeUTF8)(this._rfbCredentials.username).substring(0, 255),
                  Ce = (0, i.encodeUTF8)(this._rfbCredentials.password).substring(0, 63),
                  Xe = new Uint8Array(256),
                  Se = new Uint8Array(64)
                ;(window.crypto.getRandomValues(Xe), window.crypto.getRandomValues(Se))
                for (var Te = 0; Te < we.length; Te++) Xe[Te] = we.charCodeAt(Te)
                Xe[we.length] = 0
                for (var _e = 0; _e < Ce.length; _e++) Se[_e] = Ce.charCodeAt(_e)
                return (
                  (Se[Ce.length] = 0),
                  (Xe = y.default.encrypt({ name: 'DES-CBC', iv: te }, ae, Xe)),
                  (Se = y.default.encrypt({ name: 'DES-CBC', iv: te }, ae, Se)),
                  this._sock.sQpushBytes(J),
                  this._sock.sQpushBytes(Xe),
                  this._sock.sQpushBytes(Se),
                  this._sock.flush(),
                  (this._rfbInitState = 'SecurityResult'),
                  !0
                )
              }
            },
            {
              key: '_negotiateAuthentication',
              value: function () {
                switch (this._rfbAuthScheme) {
                  case xt:
                    return (
                      this._rfbVersion >= 3.8
                        ? (this._rfbInitState = 'SecurityResult')
                        : (this._rfbInitState = 'ClientInitialisation'),
                      !0
                    )
                  case Yn:
                    return this._negotiateXvpAuth()
                  case $n:
                    return this._negotiateARDAuth()
                  case Lr:
                    return this._negotiateStdVNCAuth()
                  case qn:
                    return this._negotiateTightAuth()
                  case rn:
                    return this._negotiateVeNCryptAuth()
                  case ti:
                    return this._negotiatePlainAuth()
                  case ei:
                    return this._negotiateTightUnixAuth()
                  case Zn:
                    return this._negotiateRA2neAuth()
                  case Jn:
                    return this._negotiateMSLogonIIAuth()
                  default:
                    return this._fail(
                      'Unsupported auth scheme (scheme: ' + this._rfbAuthScheme + ')'
                    )
                }
              }
            },
            {
              key: '_handleSecurityResult',
              value: function () {
                if (this._sock.rQwait('VNC auth response ', 4)) return !1
                var d = this._sock.rQshift32()
                return d === 0
                  ? ((this._rfbInitState = 'ClientInitialisation'),
                    r.Debug('Authentication OK'),
                    !0)
                  : this._rfbVersion >= 3.8
                    ? ((this._rfbInitState = 'SecurityReason'),
                      (this._securityContext = 'security result'),
                      (this._securityStatus = d),
                      !0)
                    : (this.dispatchEvent(
                        new CustomEvent('securityfailure', { detail: { status: d } })
                      ),
                      this._fail('Security handshake failed'))
              }
            },
            {
              key: '_negotiateServerInit',
              value: function () {
                if (this._sock.rQwait('server initialization', 24)) return !1
                var d = this._sock.rQshift16(),
                  T = this._sock.rQshift16(),
                  v = this._sock.rQshift8(),
                  H = this._sock.rQshift8(),
                  J = this._sock.rQshift8(),
                  te = this._sock.rQshift8(),
                  ae = this._sock.rQshift16(),
                  we = this._sock.rQshift16(),
                  Ce = this._sock.rQshift16(),
                  Xe = this._sock.rQshift8(),
                  Se = this._sock.rQshift8(),
                  Te = this._sock.rQshift8()
                this._sock.rQskipBytes(3)
                var _e = this._sock.rQshift32()
                if (this._sock.rQwait('server init name', _e, 24)) return !1
                var ke = this._sock.rQshiftStr(_e)
                if (((ke = (0, i.decodeUTF8)(ke, !0)), this._rfbTightVNC)) {
                  if (this._sock.rQwait('TightVNC extended server init header', 8, 24 + _e))
                    return !1
                  var We = this._sock.rQshift16(),
                    ut = this._sock.rQshift16(),
                    Le = this._sock.rQshift16()
                  this._sock.rQskipBytes(2)
                  var ze = (We + ut + Le) * 16
                  if (this._sock.rQwait('TightVNC extended server init header', ze, 32 + _e))
                    return !1
                  ;(this._sock.rQskipBytes(16 * We),
                    this._sock.rQskipBytes(16 * ut),
                    this._sock.rQskipBytes(16 * Le))
                }
                return (
                  r.Info(
                    'Screen: ' +
                      d +
                      'x' +
                      T +
                      ', bpp: ' +
                      v +
                      ', depth: ' +
                      H +
                      ', bigEndian: ' +
                      J +
                      ', trueColor: ' +
                      te +
                      ', redMax: ' +
                      ae +
                      ', greenMax: ' +
                      we +
                      ', blueMax: ' +
                      Ce +
                      ', redShift: ' +
                      Xe +
                      ', greenShift: ' +
                      Se +
                      ', blueShift: ' +
                      Te
                  ),
                  this._setDesktopName(ke),
                  this._resize(d, T),
                  this._viewOnly || this._keyboard.grab(),
                  (this._fbDepth = 24),
                  this._fbName === 'Intel(r) AMT KVM' &&
                    (r.Warn('Intel AMT KVM only supports 8/16 bit depths. Using low color mode.'),
                    (this._fbDepth = 8)),
                  R.messages.pixelFormat(this._sock, this._fbDepth, !0),
                  this._sendEncodings(),
                  R.messages.fbUpdateRequest(this._sock, !1, 0, 0, this._fbWidth, this._fbHeight),
                  this._updateConnectionState('connected'),
                  !0
                )
              }
            },
            {
              key: '_sendEncodings',
              value: function () {
                var d = []
                ;(d.push(c.encodings.encodingCopyRect),
                  this._fbDepth == 24 &&
                    (o.supportsWebCodecsH264Decode && d.push(c.encodings.encodingH264),
                    d.push(c.encodings.encodingTight),
                    d.push(c.encodings.encodingTightPNG),
                    d.push(c.encodings.encodingZRLE),
                    d.push(c.encodings.encodingJPEG),
                    d.push(c.encodings.encodingHextile),
                    d.push(c.encodings.encodingRRE),
                    d.push(c.encodings.encodingZlib)),
                  d.push(c.encodings.encodingRaw),
                  d.push(c.encodings.pseudoEncodingQualityLevel0 + this._qualityLevel),
                  d.push(c.encodings.pseudoEncodingCompressLevel0 + this._compressionLevel),
                  d.push(c.encodings.pseudoEncodingDesktopSize),
                  d.push(c.encodings.pseudoEncodingLastRect),
                  d.push(c.encodings.pseudoEncodingQEMUExtendedKeyEvent),
                  d.push(c.encodings.pseudoEncodingQEMULedEvent),
                  d.push(c.encodings.pseudoEncodingExtendedDesktopSize),
                  d.push(c.encodings.pseudoEncodingXvp),
                  d.push(c.encodings.pseudoEncodingFence),
                  d.push(c.encodings.pseudoEncodingContinuousUpdates),
                  d.push(c.encodings.pseudoEncodingDesktopName),
                  d.push(c.encodings.pseudoEncodingExtendedClipboard),
                  d.push(c.encodings.pseudoEncodingExtendedMouseButtons),
                  this._fbDepth == 24 &&
                    (d.push(c.encodings.pseudoEncodingVMwareCursor),
                    d.push(c.encodings.pseudoEncodingCursor)),
                  R.messages.clientEncodings(this._sock, d))
              }
            },
            {
              key: '_initMsg',
              value: function () {
                switch (this._rfbInitState) {
                  case 'ProtocolVersion':
                    return this._negotiateProtocolVersion()
                  case 'Security':
                    return this._negotiateSecurity()
                  case 'Authentication':
                    return this._negotiateAuthentication()
                  case 'SecurityResult':
                    return this._handleSecurityResult()
                  case 'SecurityReason':
                    return this._handleSecurityReason()
                  case 'ClientInitialisation':
                    return (
                      this._sock.sQpush8(this._shared ? 1 : 0),
                      this._sock.flush(),
                      (this._rfbInitState = 'ServerInitialisation'),
                      !0
                    )
                  case 'ServerInitialisation':
                    return this._negotiateServerInit()
                  default:
                    return this._fail('Unknown init state (state: ' + this._rfbInitState + ')')
                }
              }
            },
            {
              key: '_resumeAuthentication',
              value: function () {
                setTimeout(this._initMsg.bind(this), 0)
              }
            },
            {
              key: '_handleSetColourMapMsg',
              value: function () {
                return (
                  r.Debug('SetColorMapEntries'),
                  this._fail('Unexpected SetColorMapEntries message')
                )
              }
            },
            {
              key: '_handleServerCutText',
              value: function () {
                if ((r.Debug('ServerCutText'), this._sock.rQwait('ServerCutText header', 7, 1)))
                  return !1
                this._sock.rQskipBytes(3)
                var d = this._sock.rQshift32()
                if (
                  ((d = (0, n.toSigned32bit)(d)),
                  this._sock.rQwait('ServerCutText content', Math.abs(d), 8))
                )
                  return !1
                if (d >= 0) {
                  var T = this._sock.rQshiftStr(d)
                  if (this._viewOnly) return !0
                  this.dispatchEvent(new CustomEvent('clipboard', { detail: { text: T } }))
                } else {
                  d = Math.abs(d)
                  var v = this._sock.rQshift32(),
                    H = v & 65535,
                    J = v & 4278190080,
                    te = !!(J & ri)
                  if (te) {
                    ;((this._clipboardServerCapabilitiesFormats = {}),
                      (this._clipboardServerCapabilitiesActions = {}))
                    for (var ae = 0; ae <= 15; ae++) {
                      var we = 1 << ae
                      H & we &&
                        ((this._clipboardServerCapabilitiesFormats[we] = !0),
                        this._sock.rQshift32())
                    }
                    for (var Ce = 24; Ce <= 31; Ce++) {
                      var Xe = 1 << Ce
                      this._clipboardServerCapabilitiesActions[Xe] = !!(J & Xe)
                    }
                    var Se = [ri, Rr, ni, fr, Mr]
                    R.messages.extendedClipboardCaps(this._sock, Se, {
                      extendedClipboardFormatText: 0
                    })
                  } else if (J === Rr) {
                    if (this._viewOnly) return !0
                    this._clipboardText != null &&
                      this._clipboardServerCapabilitiesActions[Mr] &&
                      H & gt &&
                      R.messages.extendedClipboardProvide(this._sock, [gt], [this._clipboardText])
                  } else if (J === ni) {
                    if (this._viewOnly) return !0
                    this._clipboardServerCapabilitiesActions[fr] &&
                      (this._clipboardText != null
                        ? R.messages.extendedClipboardNotify(this._sock, [gt])
                        : R.messages.extendedClipboardNotify(this._sock, []))
                  } else if (J === fr) {
                    if (this._viewOnly) return !0
                    this._clipboardServerCapabilitiesActions[Rr] &&
                      H & gt &&
                      R.messages.extendedClipboardRequest(this._sock, [gt])
                  } else if (J === Mr) {
                    if (this._viewOnly || !(H & gt)) return !0
                    this._clipboardText = null
                    var Te = this._sock.rQshiftBytes(d - 4),
                      _e = new f.default(),
                      ke = null
                    _e.setInput(Te)
                    for (var We = 0; We <= 15; We++) {
                      var ut = 1 << We
                      if (H & ut) {
                        var Le = 0,
                          ze = _e.inflate(4)
                        ;((Le |= ze[0] << 24),
                          (Le |= ze[1] << 16),
                          (Le |= ze[2] << 8),
                          (Le |= ze[3]))
                        var mt = _e.inflate(Le)
                        ut === gt && (ke = mt)
                      }
                    }
                    if ((_e.setInput(null), ke !== null)) {
                      for (var Qt = '', wt = 0; wt < ke.length; wt++)
                        Qt += String.fromCharCode(ke[wt])
                      ;((ke = Qt),
                        (ke = (0, i.decodeUTF8)(ke)),
                        ke.length > 0 &&
                          ke.charAt(ke.length - 1) === '\0' &&
                          (ke = ke.slice(0, -1)),
                        (ke = ke.replaceAll(
                          `\r
`,
                          `
`
                        )),
                        this.dispatchEvent(new CustomEvent('clipboard', { detail: { text: ke } })))
                    }
                  } else return this._fail('Unexpected action in extended clipboard message: ' + J)
                }
                return !0
              }
            },
            {
              key: '_handleServerFenceMsg',
              value: function () {
                if (this._sock.rQwait('ServerFence header', 8, 1)) return !1
                this._sock.rQskipBytes(3)
                var d = this._sock.rQshift32(),
                  T = this._sock.rQshift8()
                if (this._sock.rQwait('ServerFence payload', T, 9)) return !1
                T > 64 && (r.Warn('Bad payload length (' + T + ') in fence response'), (T = 64))
                var v = this._sock.rQshiftStr(T)
                return (
                  (this._supportsFence = !0),
                  d & (1 << 31)
                    ? ((d &= 3), R.messages.clientFence(this._sock, d, v), !0)
                    : this._fail('Unexpected fence response')
                )
              }
            },
            {
              key: '_handleXvpMsg',
              value: function () {
                if (this._sock.rQwait('XVP version and message', 3, 1)) return !1
                this._sock.rQskipBytes(1)
                var d = this._sock.rQshift8(),
                  T = this._sock.rQshift8()
                switch (T) {
                  case 0:
                    r.Error('XVP operation failed')
                    break
                  case 1:
                    ;((this._rfbXvpVer = d),
                      r.Info('XVP extensions enabled (version ' + this._rfbXvpVer + ')'),
                      this._setCapability('power', !0))
                    break
                  default:
                    this._fail('Illegal server XVP message (msg: ' + T + ')')
                    break
                }
                return !0
              }
            },
            {
              key: '_normalMsg',
              value: function () {
                var d
                this._FBU.rects > 0 ? (d = 0) : (d = this._sock.rQshift8())
                var T, v
                switch (d) {
                  case 0:
                    return (
                      (v = this._framebufferUpdate()),
                      v &&
                        !this._enabledContinuousUpdates &&
                        R.messages.fbUpdateRequest(
                          this._sock,
                          !0,
                          0,
                          0,
                          this._fbWidth,
                          this._fbHeight
                        ),
                      v
                    )
                  case 1:
                    return this._handleSetColourMapMsg()
                  case 2:
                    return (
                      r.Debug('Bell'),
                      this.dispatchEvent(new CustomEvent('bell', { detail: {} })),
                      !0
                    )
                  case 3:
                    return this._handleServerCutText()
                  case 150:
                    return (
                      (T = !this._supportsContinuousUpdates),
                      (this._supportsContinuousUpdates = !0),
                      (this._enabledContinuousUpdates = !1),
                      T &&
                        ((this._enabledContinuousUpdates = !0),
                        this._updateContinuousUpdates(),
                        r.Info('Enabling continuous updates.')),
                      !0
                    )
                  case 248:
                    return this._handleServerFenceMsg()
                  case 250:
                    return this._handleXvpMsg()
                  default:
                    return (
                      this._fail('Unexpected server message (type ' + d + ')'),
                      r.Debug('sock.rQpeekBytes(30): ' + this._sock.rQpeekBytes(30)),
                      !0
                    )
                }
              }
            },
            {
              key: '_framebufferUpdate',
              value: function () {
                var d = this
                if (this._FBU.rects === 0) {
                  if (this._sock.rQwait('FBU header', 3, 1)) return !1
                  if (
                    (this._sock.rQskipBytes(1),
                    (this._FBU.rects = this._sock.rQshift16()),
                    this._display.pending())
                  )
                    return (
                      (this._flushing = !0),
                      this._display.flush().then(function () {
                        ;((d._flushing = !1), d._sock.rQwait('message', 1) || d._handleMessage())
                      }),
                      !1
                    )
                }
                for (; this._FBU.rects > 0; ) {
                  if (this._FBU.encoding === null) {
                    if (this._sock.rQwait('rect header', 12)) return !1
                    ;((this._FBU.x = this._sock.rQshift16()),
                      (this._FBU.y = this._sock.rQshift16()),
                      (this._FBU.width = this._sock.rQshift16()),
                      (this._FBU.height = this._sock.rQshift16()),
                      (this._FBU.encoding = this._sock.rQshift32()),
                      (this._FBU.encoding >>= 0))
                  }
                  if (!this._handleRect()) return !1
                  ;(this._FBU.rects--, (this._FBU.encoding = null))
                }
                return (this._display.flip(), !0)
              }
            },
            {
              key: '_handleRect',
              value: function () {
                switch (this._FBU.encoding) {
                  case c.encodings.pseudoEncodingLastRect:
                    return ((this._FBU.rects = 1), !0)
                  case c.encodings.pseudoEncodingVMwareCursor:
                    return this._handleVMwareCursor()
                  case c.encodings.pseudoEncodingCursor:
                    return this._handleCursor()
                  case c.encodings.pseudoEncodingQEMUExtendedKeyEvent:
                    return ((this._qemuExtKeyEventSupported = !0), !0)
                  case c.encodings.pseudoEncodingDesktopName:
                    return this._handleDesktopName()
                  case c.encodings.pseudoEncodingDesktopSize:
                    return (this._resize(this._FBU.width, this._FBU.height), !0)
                  case c.encodings.pseudoEncodingExtendedDesktopSize:
                    return this._handleExtendedDesktopSize()
                  case c.encodings.pseudoEncodingExtendedMouseButtons:
                    return ((this._extendedPointerEventSupported = !0), !0)
                  case c.encodings.pseudoEncodingQEMULedEvent:
                    return this._handleLedEvent()
                  default:
                    return this._handleDataRect()
                }
              }
            },
            {
              key: '_handleVMwareCursor',
              value: function () {
                var d = this._FBU.x,
                  T = this._FBU.y,
                  v = this._FBU.width,
                  H = this._FBU.height
                if (this._sock.rQwait('VMware cursor encoding', 1)) return !1
                var J = this._sock.rQshift8()
                this._sock.rQshift8()
                var te,
                  ae = 4
                if (J == 0) {
                  var we = -256
                  if (
                    ((te = new Array(v * H * ae)),
                    this._sock.rQwait('VMware cursor classic encoding', v * H * ae * 2, 2))
                  )
                    return !1
                  for (var Ce = new Array(v * H), Xe = 0; Xe < v * H; Xe++)
                    Ce[Xe] = this._sock.rQshift32()
                  for (var Se = new Array(v * H), Te = 0; Te < v * H; Te++)
                    Se[Te] = this._sock.rQshift32()
                  for (var _e = 0; _e < v * H; _e++)
                    if (Ce[_e] == 0) {
                      var ke = Se[_e],
                        We = (ke >> 8) & 255,
                        ut = (ke >> 16) & 255,
                        Le = (ke >> 24) & 255
                      ;((te[_e * ae] = We),
                        (te[_e * ae + 1] = ut),
                        (te[_e * ae + 2] = Le),
                        (te[_e * ae + 3] = 255))
                    } else
                      (Ce[_e] & we) == we
                        ? Se[_e] == 0
                          ? ((te[_e * ae] = 0),
                            (te[_e * ae + 1] = 0),
                            (te[_e * ae + 2] = 0),
                            (te[_e * ae + 3] = 0))
                          : ((Se[_e] & we) == we,
                            (te[_e * ae] = 0),
                            (te[_e * ae + 1] = 0),
                            (te[_e * ae + 2] = 0),
                            (te[_e * ae + 3] = 255))
                        : ((te[_e * ae] = 0),
                          (te[_e * ae + 1] = 0),
                          (te[_e * ae + 2] = 0),
                          (te[_e * ae + 3] = 255))
                } else if (J == 1) {
                  if (this._sock.rQwait('VMware cursor alpha encoding', v * H * 4, 2)) return !1
                  te = new Array(v * H * ae)
                  for (var ze = 0; ze < v * H; ze++) {
                    var mt = this._sock.rQshift32()
                    ;((te[ze * 4] = (mt >> 24) & 255),
                      (te[ze * 4 + 1] = (mt >> 16) & 255),
                      (te[ze * 4 + 2] = (mt >> 8) & 255),
                      (te[ze * 4 + 3] = mt & 255))
                  }
                } else
                  return (r.Warn('The given cursor type is not supported: ' + J + ' given.'), !1)
                return (this._updateCursor(te, d, T, v, H), !0)
              }
            },
            {
              key: '_handleCursor',
              value: function () {
                var d = this._FBU.x,
                  T = this._FBU.y,
                  v = this._FBU.width,
                  H = this._FBU.height,
                  J = v * H * 4,
                  te = Math.ceil(v / 8) * H,
                  ae = J + te
                if (this._sock.rQwait('cursor encoding', ae)) return !1
                for (
                  var we = this._sock.rQshiftBytes(J),
                    Ce = this._sock.rQshiftBytes(te),
                    Xe = new Uint8Array(v * H * 4),
                    Se = 0,
                    Te = 0;
                  Te < H;
                  Te++
                )
                  for (var _e = 0; _e < v; _e++) {
                    var ke = Te * Math.ceil(v / 8) + Math.floor(_e / 8),
                      We = (Ce[ke] << _e % 8) & 128 ? 255 : 0
                    ;((Xe[Se] = we[Se + 2]),
                      (Xe[Se + 1] = we[Se + 1]),
                      (Xe[Se + 2] = we[Se]),
                      (Xe[Se + 3] = We),
                      (Se += 4))
                  }
                return (this._updateCursor(Xe, d, T, v, H), !0)
              }
            },
            {
              key: '_handleDesktopName',
              value: function () {
                if (this._sock.rQwait('DesktopName', 4)) return !1
                var d = this._sock.rQshift32()
                if (this._sock.rQwait('DesktopName', d, 4)) return !1
                var T = this._sock.rQshiftStr(d)
                return ((T = (0, i.decodeUTF8)(T, !0)), this._setDesktopName(T), !0)
              }
            },
            {
              key: '_handleLedEvent',
              value: function () {
                if (this._sock.rQwait('LED status', 1)) return !1
                var d = this._sock.rQshift8(),
                  T = !!(d & 2),
                  v = !!(d & 4)
                return ((this._remoteCapsLock = v), (this._remoteNumLock = T), !0)
              }
            },
            {
              key: '_handleExtendedDesktopSize',
              value: function () {
                if (this._sock.rQwait('ExtendedDesktopSize', 4)) return !1
                var d = this._sock.rQpeek8(),
                  T = 4 + d * 16
                if (this._sock.rQwait('ExtendedDesktopSize', T)) return !1
                var v = !this._supportsSetDesktopSize
                ;((this._supportsSetDesktopSize = !0),
                  this._sock.rQskipBytes(1),
                  this._sock.rQskipBytes(3))
                for (var H = 0; H < d; H += 1)
                  H === 0
                    ? ((this._screenID = this._sock.rQshift32()),
                      this._sock.rQskipBytes(2),
                      this._sock.rQskipBytes(2),
                      this._sock.rQskipBytes(2),
                      this._sock.rQskipBytes(2),
                      (this._screenFlags = this._sock.rQshift32()))
                    : this._sock.rQskipBytes(16)
                if (
                  (this._FBU.x === 1 && (this._pendingRemoteResize = !1),
                  this._FBU.x === 1 && this._FBU.y !== 0)
                ) {
                  var J = ''
                  switch (this._FBU.y) {
                    case 1:
                      J = 'Resize is administratively prohibited'
                      break
                    case 2:
                      J = 'Out of resources'
                      break
                    case 3:
                      J = 'Invalid screen layout'
                      break
                    default:
                      J = 'Unknown reason'
                      break
                  }
                  r.Warn('Server did not accept the resize request: ' + J)
                } else this._resize(this._FBU.width, this._FBU.height)
                return (
                  v && this._requestRemoteResize(),
                  this._FBU.x === 1 && this._FBU.y === 0 && this._requestRemoteResize(),
                  !0
                )
              }
            },
            {
              key: '_handleDataRect',
              value: function () {
                var d = this._decoders[this._FBU.encoding]
                if (!d)
                  return (
                    this._fail('Unsupported encoding (encoding: ' + this._FBU.encoding + ')'),
                    !1
                  )
                try {
                  return d.decodeRect(
                    this._FBU.x,
                    this._FBU.y,
                    this._FBU.width,
                    this._FBU.height,
                    this._sock,
                    this._display,
                    this._fbDepth
                  )
                } catch (T) {
                  return (this._fail('Error decoding rect: ' + T), !1)
                }
              }
            },
            {
              key: '_updateContinuousUpdates',
              value: function () {
                this._enabledContinuousUpdates &&
                  R.messages.enableContinuousUpdates(
                    this._sock,
                    !0,
                    0,
                    0,
                    this._fbWidth,
                    this._fbHeight
                  )
              }
            },
            {
              key: '_resize',
              value: function (d, T) {
                ;((this._fbWidth = d),
                  (this._fbHeight = T),
                  this._display.resize(this._fbWidth, this._fbHeight),
                  this._updateClip(),
                  this._updateScale(),
                  this._updateContinuousUpdates(),
                  this._saveExpectedClientSize())
              }
            },
            {
              key: '_xvpOp',
              value: function (d, T) {
                this._rfbXvpVer < d ||
                  (r.Info('Sending XVP operation ' + T + ' (version ' + d + ')'),
                  R.messages.xvpOp(this._sock, d, T))
              }
            },
            {
              key: '_updateCursor',
              value: function (d, T, v, H, J) {
                ;((this._cursorImage = { rgbaPixels: d, hotx: T, hoty: v, w: H, h: J }),
                  this._refreshCursor())
              }
            },
            {
              key: '_shouldShowDotCursor',
              value: function () {
                if (!this._showDotCursor) return !1
                for (var d = 3; d < this._cursorImage.rgbaPixels.length; d += 4)
                  if (this._cursorImage.rgbaPixels[d]) return !1
                return !0
              }
            },
            {
              key: '_refreshCursor',
              value: function () {
                if (
                  !(
                    this._rfbConnectionState !== 'connecting' &&
                    this._rfbConnectionState !== 'connected'
                  )
                ) {
                  var d = this._shouldShowDotCursor() ? R.cursors.dot : this._cursorImage
                  this._cursor.change(d.rgbaPixels, d.hotx, d.hoty, d.w, d.h)
                }
              }
            }
          ],
          [
            {
              key: '_convertButtonMask',
              value: function (d) {
                for (var T = { 0: 1, 1: 4, 2: 2, 3: 128, 4: 256 }, v = 0, H = 0; H < 5; H++)
                  d & (1 << H) && (v |= T[H])
                return v
              }
            },
            {
              key: 'genDES',
              value: function (d, T) {
                var v = d.split('').map(function (J) {
                    return J.charCodeAt(0)
                  }),
                  H = y.default.importKey('raw', v, { name: 'DES-ECB' }, !1, ['encrypt'])
                return y.default.encrypt({ name: 'DES-ECB' }, H, T)
              }
            }
          ]
        )
      )
    })(x.default))
  ;((bt.messages = {
    keyEvent: function (R, D, d) {
      ;(R.sQpush8(4), R.sQpush8(d), R.sQpush16(0), R.sQpush32(D), R.flush())
    },
    QEMUExtendedKeyEvent: function (R, D, d, T) {
      function v(J) {
        var te = T >> 8,
          ae = T & 255
        return te === 224 && ae < 127 ? ae | 128 : J
      }
      ;(R.sQpush8(255), R.sQpush8(0), R.sQpush16(d), R.sQpush32(D))
      var H = v(T)
      ;(R.sQpush32(H), R.flush())
    },
    pointerEvent: function (R, D, d, T) {
      ;(R.sQpush8(5), (T = T & 127), R.sQpush8(T), R.sQpush16(D), R.sQpush16(d), R.flush())
    },
    extendedPointerEvent: function (R, D, d, T) {
      R.sQpush8(5)
      var v = (T >> 7) & 255
      if (v & 252) throw new Error('Invalid mouse button mask: ' + T)
      var H = T & 127
      ;((H |= 128), R.sQpush8(H), R.sQpush16(D), R.sQpush16(d), R.sQpush8(v), R.flush())
    },
    _buildExtendedClipboardFlags: function (R, D) {
      for (var d = new Uint8Array(4), T = 0, v = 0, H = 0; H < R.length; H++) v |= R[H]
      for (var J = 0; J < D.length; J++) T |= D[J]
      return ((d[0] = v >> 24), (d[1] = 0), (d[2] = 0), (d[3] = T), d)
    },
    extendedClipboardProvide: function (R, D, d) {
      for (var T = new F.default(), v = [], H = 0; H < D.length; H++) {
        if (D[H] != gt)
          throw new Error('Unsupported extended clipboard format for Provide message.')
        d[H] = d[H].replace(
          /\r\n|\r|\n/gm,
          `\r
`
        )
        var J = (0, i.encodeUTF8)(d[H] + '\0')
        v.push(
          (J.length >> 24) & 255,
          (J.length >> 16) & 255,
          (J.length >> 8) & 255,
          J.length & 255
        )
        for (var te = 0; te < J.length; te++) v.push(J.charCodeAt(te))
      }
      var ae = T.deflate(new Uint8Array(v)),
        we = new Uint8Array(4 + ae.length)
      ;(we.set(bt.messages._buildExtendedClipboardFlags([Mr], D)),
        we.set(ae, 4),
        bt.messages.clientCutText(R, we, !0))
    },
    extendedClipboardNotify: function (R, D) {
      var d = bt.messages._buildExtendedClipboardFlags([fr], D)
      bt.messages.clientCutText(R, d, !0)
    },
    extendedClipboardRequest: function (R, D) {
      var d = bt.messages._buildExtendedClipboardFlags([Rr], D)
      bt.messages.clientCutText(R, d, !0)
    },
    extendedClipboardCaps: function (R, D, d) {
      var T = Object.keys(d),
        v = new Uint8Array(4 + 4 * T.length)
      ;(T.map(function (te) {
        return parseInt(te)
      }),
        T.sort(function (te, ae) {
          return te - ae
        }),
        v.set(bt.messages._buildExtendedClipboardFlags(D, [])))
      for (var H = 4, J = 0; J < T.length; J++)
        ((v[H] = d[T[J]] >> 24),
          (v[H + 1] = d[T[J]] >> 16),
          (v[H + 2] = d[T[J]] >> 8),
          (v[H + 3] = d[T[J]] >> 0),
          (H += 4),
          (v[3] |= 1 << T[J]))
      bt.messages.clientCutText(R, v, !0)
    },
    clientCutText: function (R, D) {
      var d = arguments.length > 2 && arguments[2] !== void 0 ? arguments[2] : !1
      ;(R.sQpush8(6), R.sQpush8(0), R.sQpush8(0), R.sQpush8(0))
      var T
      ;(d ? (T = (0, n.toUnsigned32bit)(-D.length)) : (T = D.length),
        R.sQpush32(T),
        R.sQpushBytes(D),
        R.flush())
    },
    setDesktopSize: function (R, D, d, T, v) {
      ;(R.sQpush8(251),
        R.sQpush8(0),
        R.sQpush16(D),
        R.sQpush16(d),
        R.sQpush8(1),
        R.sQpush8(0),
        R.sQpush32(T),
        R.sQpush16(0),
        R.sQpush16(0),
        R.sQpush16(D),
        R.sQpush16(d),
        R.sQpush32(v),
        R.flush())
    },
    clientFence: function (R, D, d) {
      ;(R.sQpush8(248),
        R.sQpush8(0),
        R.sQpush8(0),
        R.sQpush8(0),
        R.sQpush32(D),
        R.sQpush8(d.length),
        R.sQpushString(d),
        R.flush())
    },
    enableContinuousUpdates: function (R, D, d, T, v, H) {
      ;(R.sQpush8(150),
        R.sQpush8(D),
        R.sQpush16(d),
        R.sQpush16(T),
        R.sQpush16(v),
        R.sQpush16(H),
        R.flush())
    },
    pixelFormat: function (R, D, d) {
      var T
      D > 16 ? (T = 32) : D > 8 ? (T = 16) : (T = 8)
      var v = Math.floor(D / 3)
      ;(R.sQpush8(0),
        R.sQpush8(0),
        R.sQpush8(0),
        R.sQpush8(0),
        R.sQpush8(T),
        R.sQpush8(D),
        R.sQpush8(0),
        R.sQpush8(d ? 1 : 0),
        R.sQpush16((1 << v) - 1),
        R.sQpush16((1 << v) - 1),
        R.sQpush16((1 << v) - 1),
        R.sQpush8(v * 0),
        R.sQpush8(v * 1),
        R.sQpush8(v * 2),
        R.sQpush8(0),
        R.sQpush8(0),
        R.sQpush8(0),
        R.flush())
    },
    clientEncodings: function (R, D) {
      ;(R.sQpush8(2), R.sQpush8(0), R.sQpush16(D.length))
      for (var d = 0; d < D.length; d++) R.sQpush32(D[d])
      R.flush()
    },
    fbUpdateRequest: function (R, D, d, T, v, H) {
      ;(typeof d > 'u' && (d = 0),
        typeof T > 'u' && (T = 0),
        R.sQpush8(3),
        R.sQpush8(D ? 1 : 0),
        R.sQpush16(d),
        R.sQpush16(T),
        R.sQpush16(v),
        R.sQpush16(H),
        R.flush())
    },
    xvpOp: function (R, D, d) {
      ;(R.sQpush8(250), R.sQpush8(0), R.sQpush8(D), R.sQpush8(d), R.flush())
    }
  }),
    (bt.cursors = {
      none: { rgbaPixels: new Uint8Array(), w: 0, h: 0, hotx: 0, hoty: 0 },
      dot: {
        rgbaPixels: new Uint8Array([
          255, 255, 255, 255, 0, 0, 0, 255, 255, 255, 255, 255, 0, 0, 0, 255, 0, 0, 0, 0, 0, 0, 0,
          255, 255, 255, 255, 255, 0, 0, 0, 255, 255, 255, 255, 255
        ]),
        w: 3,
        h: 3,
        hotx: 1,
        hoty: 1
      }
    }))
})(la)
const al = Lo(la)
window.RFB = al
