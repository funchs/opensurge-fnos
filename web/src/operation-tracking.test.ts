// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, waitForOperation, watchOperations } from './api'
import { clearOperations, getOperation, getOperations, recordOperation } from './operations'
import type { Operation } from './types'

function response(payload: unknown, status = 200) {
  return { ok: status < 400, status, statusText: String(status), json: async () => payload } as Response
}

let sequence = 0

describe('operation tracking and shared polling', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-04T00:00:00Z'))
    clearOperations()
    vi.stubGlobal('crypto', { randomUUID: () => `request-${++sequence}` })
  })
  afterEach(async () => {
    for (const operation of getOperations()) if (operation.state === 'running') recordOperation({ ...operation, state: 'failed' })
    await vi.advanceTimersByTimeAsync(1000)
    clearOperations()
    vi.useRealTimers()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('publishes startup before acknowledgement and shares one poller with the caller', async () => {
    let acknowledge!: (value: Response) => void
    let stored: Operation | undefined
    const fetcher = vi.fn((path: string, init?: RequestInit): Promise<Response> => {
      if (path.endsWith('/gateway/start')) {
        const id = (init?.headers as Record<string, string>)['Idempotency-Key']
        stored = { id, kind: 'start', state: 'running', phase: 'starting_mihomo', created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
        return new Promise(resolve => { acknowledge = resolve })
      }
      return Promise.resolve(response(stored))
    })
    vi.stubGlobal('fetch', fetcher)
    const submitted = api.gateway('start')
    const id = getOperations()[0].id
    expect(getOperation(id)?.phase).toBe('submitting')
    await vi.advanceTimersByTimeAsync(0)
    expect(getOperation(id)?.phase).toBe('starting_mihomo')
    acknowledge(response(stored, 202))
    await submitted
    const first = waitForOperation(id)
    expect(waitForOperation(id)).toBe(first)
    stored = { ...stored!, state: 'succeeded', notices: ['tailscale_warmup_started'] }
    await vi.advanceTimersByTimeAsync(500)
    await expect(first).resolves.toMatchObject({ state: 'succeeded' })
    expect(fetcher.mock.calls.filter(([path]) => path.includes('/operations/'))).toHaveLength(2)
    expect(fetcher.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
  })

  it('tracks a synchronous device save using a correlation ID without another mutation', async () => {
    let finish!: (value: Response) => void
    let id = ''
    let completed = false
    const fetcher = vi.fn((path: string, init?: RequestInit): Promise<Response> => {
      if (path.endsWith('/device-policy')) {
        id = (init?.headers as Record<string, string>)['X-OpenSurge-Operation-ID']
        expect((init?.headers as Record<string, string>)['If-Match']).toBe('"policy-revision"')
        return new Promise(resolve => { finish = resolve })
      }
      return Promise.resolve(response({ id, kind: 'save-device-policy', state: completed ? 'succeeded' : 'running', phase: 'validating_config' }))
    })
    vi.stubGlobal('fetch', fetcher)
    const save = api.saveDevicePolicy({ devices: [], profiles: [], templates: [], rule_sets: [] }, 'policy-revision')
    expect(id).not.toBe('')
    await vi.advanceTimersByTimeAsync(0)
    expect(getOperation(id)?.phase).toBe('validating_config')
    completed = true
    finish(response({ revision: 'new-revision', policy: {} }))
    await expect(save).resolves.toMatchObject({ revision: 'new-revision' })
    expect(getOperation(id)?.state).toBe('succeeded')
    expect(fetcher.mock.calls.filter(([, init]) => init?.method === 'PUT')).toHaveLength(1)
  })

  it('retries status reads after a connection loss, never the original reload', async () => {
    let polls = 0
    let id = ''
    const fetcher = vi.fn(async (path: string, init?: RequestInit) => {
      if (path.endsWith('/gateway/reload')) {
        id = (init?.headers as Record<string, string>)['Idempotency-Key']
        return response({ id, kind: 'reload', state: 'running' }, 202)
      }
      if (++polls === 1) throw new TypeError('connection lost')
      return response({ id, kind: 'reload', state: 'succeeded' })
    })
    vi.stubGlobal('fetch', fetcher)
    await api.gateway('reload')
    const completed = waitForOperation(id)
    await vi.advanceTimersByTimeAsync(500)
    await expect(completed).resolves.toMatchObject({ state: 'succeeded' })
    expect(fetcher.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
  })

  it.each([
    { kind: 'dhcp-probe', path: '/api/v1/network/dhcp-probe', submit: api.probeDHCP, state: 'succeeded', status: 200 },
    { kind: 'router-dhcp-restored', path: '/api/v1/recovery/router-restored', submit: api.confirmRouterRestored, state: 'failed', status: 409 },
  ])('tracks $kind while the OFFER probe is pending and records its response', async ({ kind, path, submit, state, status }) => {
    let finish!: (value: Response) => void
    let id = ''
    let completed = false
    const error = 'no DHCP server answered after the router was marked restored'
    const fetcher = vi.fn((url: string, init?: RequestInit): Promise<Response> => {
      if (url === path) {
        id = (init?.headers as Record<string, string>)['X-OpenSurge-Operation-ID']
        return new Promise(resolve => { finish = resolve })
      }
      return Promise.resolve(response({ id, kind, state: completed ? state : 'running', phase: 'probing_dhcp', ...(completed && state === 'failed' ? { error } : {}) }))
    })
    vi.stubGlobal('fetch', fetcher)
    const result = submit().catch(cause => cause)
    expect(getOperation(id)?.phase).toBe('submitting')
    await vi.advanceTimersByTimeAsync(0)
    expect(getOperation(id)).toMatchObject({ kind, state: 'running', phase: 'probing_dhcp', connection: 'connected' })
    completed = true
    finish(response(state === 'succeeded' ? { recovery: { stage: 'router_dhcp_disabled_confirmed' } } : { error: { code: 'router_dhcp_missing', message: error } }, status))
    await result
    expect(getOperation(id)).toMatchObject({ state, ...(state === 'failed' ? { error } : {}) })
    expect(fetcher.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
  })

  it('labels a read timeout as unknown, not a failed gateway action', async () => {
    const id = `timeout-${++sequence}`
    recordOperation({ id, kind: 'reload', state: 'running', phase: 'stopping_mihomo', created_at: new Date().toISOString() })
    vi.stubGlobal('fetch', vi.fn(async () => { throw new TypeError('offline') }))
    const result = waitForOperation(id, 1000).catch(error => error)
    await vi.advanceTimersByTimeAsync(1000)
    expect(await result).toMatchObject({ name: 'OperationStatusUnknownError' })
    expect(getOperation(id)).toMatchObject({ state: 'running', connection: 'unknown' })
  })

  it('bounds each status request even when the server never sends a response', async () => {
    let signal: AbortSignal | undefined
    vi.stubGlobal('fetch', vi.fn((_path: string, init: RequestInit) => new Promise((_resolve, reject) => {
      signal = init.signal as AbortSignal
      signal.addEventListener('abort', () => reject(new DOMException('Timed out', 'AbortError')), { once: true })
    })))
    const result = api.operation('hung-status-read').catch(error => error)
    await vi.advanceTimersByTimeAsync(5000)
    expect(signal?.aborted).toBe(true)
    expect(await result).toMatchObject({ name: 'AbortError' })
  })

  it('tracks apply-profile when crypto.randomUUID is missing', async () => {
    vi.stubGlobal('crypto', {
      getRandomValues(target: Uint8Array) {
        target.fill(0xab)
        return target
      },
    })
    let id = ''
    const fetcher = vi.fn(async (path: string, init?: RequestInit) => {
      if (path.endsWith('/sources/src-1/apply')) {
        id = (init?.headers as Record<string, string>)['X-OpenSurge-Operation-ID']
        expect((init?.headers as Record<string, string>)['If-Match']).toBe('"rev-1"')
        return response({ id: 'src-1', applied: true })
      }
      return response({ id, kind: 'apply-profile', state: 'succeeded' })
    })
    vi.stubGlobal('fetch', fetcher)
    await expect(api.applySource('src-1', 'rev-1')).resolves.toMatchObject({ applied: true })
    expect(id).toBe('abababab-abab-4bab-abab-abababababab')
    expect(getOperation(id)?.state).toBe('succeeded')
    expect(fetcher.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1)
  })

  it('does not resurrect completion with an older in-flight poll', () => {
    const operation = { id: 'completed', kind: 'start', state: 'succeeded' }
    recordOperation(operation)
    recordOperation({ ...operation, state: 'running', phase: 'starting_mihomo' })
    expect(getOperation(operation.id)?.state).toBe('succeeded')
  })

  it('resumes unfinished operations after refresh without replaying old results', async () => {
    const running: Operation = { id: `resumed-${++sequence}`, kind: 'reload', state: 'running', phase: 'checking_reservations', created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
    let finished = false
    const fetcher = vi.fn(async (path: string) => response(path.endsWith('/operations')
      ? { operations: [running, { id: 'old-success', kind: 'start', state: 'succeeded' }] }
      : { ...running, state: finished ? 'succeeded' : 'running' }))
    vi.stubGlobal('fetch', fetcher)
    const stop = watchOperations()
    await vi.advanceTimersByTimeAsync(0)
    expect(getOperation(running.id)?.phase).toBe('checking_reservations')
    expect(getOperation('old-success')).toBeUndefined()
    finished = true
    stop()
    await vi.advanceTimersByTimeAsync(8000)
    expect(fetcher.mock.calls.filter(([path]) => path.endsWith('/operations'))).toHaveLength(1)
    expect(getOperation(running.id)?.state).toBe('succeeded')
  })
})
