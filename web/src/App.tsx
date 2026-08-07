import { useCallback, useEffect, useRef, useState } from 'react'
import { api, authenticationRequiredEvent, RequestError } from './api'
import { PageErrorBoundary } from './components/PageErrorBoundary'
import { RecoveryBanner, StatusDot } from './components/Common'
import { DashboardPage } from './pages/DashboardPage'
import { ConnectivityPage } from './pages/ConnectivityPage'
import { DevicesPage } from './pages/DevicesPage'
import { DiagnosticsPage } from './pages/DiagnosticsPage'
import { NetworkPage } from './pages/NetworkPage'
import { PoliciesPage } from './pages/PoliciesPage'
import { SourcesPage } from './pages/SourcesPage'
import { needsNetworkRecoveryWarning, statusLabel } from './status'
import type { Overview } from './types'

type Page = 'dashboard' | 'network' | 'sources' | 'devices' | 'policies' | 'connectivity' | 'diagnostics'
type Theme = 'dark' | 'light'
type NetworkNavigationTarget = 'none' | 'control' | 'bottom'

/** Fixed 16×16 stroke icons so sidebar marks share optical size (Unicode glyphs did not). */
function NavIcon({ id }: { id: Page }) {
  const common = {
    viewBox: '0 0 16 16',
    width: 16,
    height: 16,
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 1.5,
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    'aria-hidden': true as const,
    focusable: false as const,
  }
  switch (id) {
    case 'dashboard':
      return <svg {...common}><rect x="2.5" y="2.5" width="4.5" height="4.5" rx="1" /><rect x="9" y="2.5" width="4.5" height="4.5" rx="1" /><rect x="2.5" y="9" width="4.5" height="4.5" rx="1" /><rect x="9" y="9" width="4.5" height="4.5" rx="1" /></svg>
    case 'network':
      return <svg {...common}><path d="M2.75 7.25c2.9-2.9 7.6-2.9 10.5 0M4.6 9.4c1.9-1.9 5-1.9 6.9 0" /><circle cx="8" cy="12.1" r="0.85" fill="currentColor" stroke="none" /></svg>
    case 'sources':
      return <svg {...common}><circle cx="8" cy="8" r="5.25" /><circle cx="8" cy="8" r="1.75" /></svg>
    case 'devices':
      return <svg {...common}><rect x="2.5" y="3.5" width="11" height="7.5" rx="1.25" /><path d="M6 13.5h4M8 11v2.5" /></svg>
    case 'policies':
      return <svg {...common}><path d="M3.5 5.5h9M10 3l2.5 2.5L10 8M12.5 10.5h-9M6 8l-2.5 2.5L6 13" /></svg>
    case 'connectivity':
      return <svg {...common}><path d="M2.5 6.5c1.6-1.7 3.4-1.7 5 0s3.4 1.7 5 0M2.5 10c1.6-1.7 3.4-1.7 5 0s3.4 1.7 5 0" /></svg>
    case 'diagnostics':
      return <svg {...common}><circle cx="7" cy="7" r="4.25" /><path d="m10.2 10.2 3.3 3.3" /></svg>
  }
}

const nav = [
  { id: 'dashboard', label: '总览' },
  { id: 'network', label: '网络设置' },
  { id: 'sources', label: '代理与规则源' },
  { id: 'devices', label: '设备' },
  { id: 'policies', label: '策略' },
  { id: 'connectivity', label: '连通性' },
  { id: 'diagnostics', label: '诊断' },
] as const satisfies ReadonlyArray<{ id: Page; label: string }>

function currentPage(): Page {
  const candidate = window.location.pathname.split('/').filter(Boolean)[0] as Page | undefined
  return nav.some(item => item.id === candidate) ? candidate! : 'dashboard'
}

function initialTheme(): Theme {
  const stored = window.localStorage.getItem('opensurge-theme')
  if (stored === 'dark' || stored === 'light') return stored
  return typeof window.matchMedia === 'function' && window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

function focusGatewayControl(target: Exclude<NetworkNavigationTarget, 'none'>) {
  const control = document.getElementById('gateway-control')
  if (!(control instanceof HTMLButtonElement)) return
  const reducedMotion = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (target === 'bottom') {
    window.scrollTo?.({ top: document.documentElement.scrollHeight, behavior: reducedMotion ? 'auto' : 'smooth' })
  } else {
    control.scrollIntoView?.({ behavior: reducedMotion ? 'auto' : 'smooth', block: 'center' })
  }
  if (!control.disabled) control.focus({ preventScroll: true })
}

function networkNavigationHash(target: NetworkNavigationTarget) {
  if (target === 'control') return '#gateway-control'
  if (target === 'bottom') return '#gateway-control-bottom'
  return ''
}

export function App() {
  const [page, setPage] = useState<Page>(currentPage)
  const [overview, setOverview] = useState<Overview | null>(null)
  const [error, setError] = useState('')
  const [authenticationRequired, setAuthenticationRequired] = useState(false)
  const [theme, setTheme] = useState<Theme>(initialTheme)
  const [devicesDirty, setDevicesDirty] = useState(false)
  const pageRef = useRef(page)
  const devicesDirtyRef = useRef(devicesDirty)
  pageRef.current = page
  devicesDirtyRef.current = devicesDirty

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    window.localStorage.setItem('opensurge-theme', theme)
  }, [theme])

  const refresh = useCallback(async () => {
    try {
      setOverview(await api.overview())
      setError('')
      // 成功拿到数据后清掉「已尝试 /enter」标记，下次 401 还可再进一次
      try { window.sessionStorage.removeItem('opensurge-enter-attempted') } catch { /* ignore */ }
    } catch (cause) {
      if (cause instanceof RequestError && cause.status === 401) {
        // fnOS / Docker 局域网模式提供 /enter 自动签发 session；只尝试一次避免死循环
        try {
          if (window.sessionStorage.getItem('opensurge-enter-attempted') !== '1') {
            window.sessionStorage.setItem('opensurge-enter-attempted', '1')
            window.location.replace('/enter')
            return
          }
        } catch { /* sessionStorage 不可用时退回文案提示 */ }
        setAuthenticationRequired(true)
        setError('')
        return
      }
      setError(cause instanceof Error ? cause.message : String(cause))
    }
  }, [])

  useEffect(() => {
    const requireAuthentication = () => {
      try {
        if (window.sessionStorage.getItem('opensurge-enter-attempted') !== '1') {
          window.sessionStorage.setItem('opensurge-enter-attempted', '1')
          window.location.replace('/enter')
          return
        }
      } catch { /* ignore */ }
      setAuthenticationRequired(true)
      setError('')
    }
    window.addEventListener(authenticationRequiredEvent, requireAuthentication)
    return () => window.removeEventListener(authenticationRequiredEvent, requireAuthentication)
  }, [])

  useEffect(() => {
    if (authenticationRequired) return
    void refresh()
    const timer = window.setInterval(() => void refresh(), 8000)
    const events = typeof EventSource === 'undefined' ? null : new EventSource('/api/v1/events')
    events?.addEventListener('state', () => void refresh())
    const onPop = () => {
      const next = currentPage()
      if (pageRef.current === 'devices' && next !== 'devices' && devicesDirtyRef.current && !window.confirm('设备页还有尚未保存的修改，确定离开并放弃这些修改吗？')) {
        history.pushState({}, '', '/devices')
        return
      }
      if (pageRef.current === 'devices' && next !== 'devices') setDevicesDirty(false)
      setPage(next)
    }
    window.addEventListener('popstate', onPop)
    return () => {
      window.clearInterval(timer)
      events?.close()
      window.removeEventListener('popstate', onPop)
    }
  }, [authenticationRequired, refresh])

  const go = (next: Page, networkTarget: NetworkNavigationTarget = 'none') => {
    if (next === page) {
      if (networkTarget !== 'none') {
        history.replaceState({}, '', `/${next}${networkNavigationHash(networkTarget)}`)
        focusGatewayControl(networkTarget)
      }
      return
    }
    if (page === 'devices' && next !== 'devices' && devicesDirty && !window.confirm('设备页还有尚未保存的修改，确定离开并放弃这些修改吗？')) return
    if (page === 'devices' && next !== 'devices') setDevicesDirty(false)
    history.pushState({}, '', `/${next}${networkNavigationHash(networkTarget)}`)
    setPage(next)
  }

  return <div className="app-shell">
    <aside className="sidebar">
      <div className="brand"><img className="brand-mark" src="/opensurge-mark.png" alt="" aria-hidden="true" /><div><strong>OpenSurge</strong><small>fnOS Edition</small></div></div>
      <nav aria-label="OpenSurge sections">
        {nav.map(item => <button key={item.id} className={page === item.id ? 'active' : ''} onClick={() => go(item.id)}><span className="nav-icon" aria-hidden="true"><NavIcon id={item.id} /></span>{item.label}</button>)}
      </nav>
      <button type="button" className="theme-toggle" aria-pressed={theme === 'light'} aria-label={theme === 'dark' ? '切换为浅色模式' : '切换为深色模式'} onClick={() => setTheme(current => current === 'dark' ? 'light' : 'dark')}><span aria-hidden="true">{theme === 'dark' ? '☀' : '◐'}</span>{theme === 'dark' ? '浅色模式' : '深色模式'}</button>
      <div className="sidebar-status"><StatusDot status={overview?.status.gateway ?? 'unreachable'} /><div><strong>{statusLabel(overview?.status.gateway)}</strong><small>{overview?.status.lan_ip || 'Control API'}</small></div></div>
    </aside>
    <main className="workspace">
      {authenticationRequired ? <section className="session-expired" role="alert"><span aria-hidden="true">!</span><div><h1>Web GUI 与 OpenSurge 的安全连接已过期</h1><p>请打开 <a href="/enter">/enter</a> 重新建立会话，或在 NAS 上检查 OpenSurge 服务状态与日志。若直连 IP:端口仍失败，确认配置里的 <code>gateway.lan_ip</code> 与浏览器地址一致。</p></div></section> : <>
        {overview?.recovery.required && needsNetworkRecoveryWarning(overview.recovery.stage) && <RecoveryBanner recovery={overview.recovery.stage} onOpen={() => go('network', 'control')} />}
        {error && <div className="error-banner" role="alert"><span>!</span><p>{error}</p><button onClick={() => void refresh()}>重试</button></div>}
        <PageErrorBoundary key={page}>
          {page === 'dashboard' && <DashboardPage overview={overview} onOpenNetwork={action => go('network', action === 'stop' ? 'bottom' : 'none')} />}
          {page === 'network' && <NetworkPage overview={overview} onChanged={refresh} onNavigate={() => go('devices')} />}
          {page === 'sources' && <SourcesPage overview={overview} onChanged={refresh} />}
          {page === 'devices' && <DevicesPage overview={overview} onChanged={refresh} onNavigate={go} onDirtyChange={setDevicesDirty} />}
          {page === 'policies' && <PoliciesPage overview={overview} onChanged={refresh} />}
          {page === 'connectivity' && <ConnectivityPage overview={overview} />}
          {page === 'diagnostics' && <DiagnosticsPage overview={overview} />}
        </PageErrorBoundary>
      </>}
    </main>
  </div>
}
