// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, authenticationRequiredEvent, randomUUID, request } from './api'

describe('Control API requests', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('announces authentication expiry for any API request that receives 401', async () => {
    const listener = vi.fn()
    window.addEventListener(authenticationRequiredEvent, listener, { once: true })
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: false,
      status: 401,
      statusText: 'Unauthorized',
      json: async () => ({ error: { code: 'authentication_required', message: 'expired' } }),
    })))

    await expect(request('/api/v1/overview')).rejects.toMatchObject({
      status: 401,
      code: 'authentication_required',
    })
    expect(listener).toHaveBeenCalledOnce()
  })

  it('generates a valid UUID v4 in insecure contexts without crypto.randomUUID', async () => {
    const getRandomValues = (arr: Uint8Array) => {
      for (let i = 0; i < arr.length; i++) arr[i] = Math.floor(Math.random() * 256)
      return arr
    }
    vi.stubGlobal('crypto', { getRandomValues })

    const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
    const uuid = randomUUID()
    expect(uuidPattern.test(uuid)).toBe(true)

    let capturedId = ''
    vi.stubGlobal('fetch', vi.fn((path: string, init?: RequestInit): Promise<Response> => {
      if (path.endsWith('/gateway/start')) {
        capturedId = (init?.headers as Record<string, string>)['Idempotency-Key']
        return Promise.resolve({ ok: true, json: async () => ({ id: capturedId, kind: 'start', state: 'succeeded' }) } as Response)
      }
      return Promise.resolve({ ok: true, json: async () => ({}) } as Response)
    }))

    await api.gateway('start')
    expect(uuidPattern.test(capturedId)).toBe(true)
  })

  it('falls back to pseudo-random UUID when crypto is completely absent', () => {
    vi.stubGlobal('crypto', undefined)
    const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
    const uuid = randomUUID()
    expect(uuidPattern.test(uuid)).toBe(true)
  })
})
