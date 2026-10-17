// @vitest-environment jsdom
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { RequestError } from '../api'
import { LoginPage } from './LoginPage'
import { SetupPage } from './SetupPage'

vi.mock('../api', () => ({
  RequestError: class RequestError extends Error {
    constructor(public status: number, public code: string, message: string) { super(message) }
  },
  api: {
    setupGUIPassword: vi.fn(),
    loginGUI: vi.fn(),
  },
}))

import { api } from '../api'

describe('GUI password pages', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('rejects a short or mismatched setup password without calling the API', async () => {
    const onSuccess = vi.fn()
    render(<SetupPage onSuccess={onSuccess} />)
    await userEvent.type(screen.getByLabelText('面板密码'), 'short')
    const confirm = screen.getByLabelText('确认密码')
    await userEvent.type(confirm, 'short')
    await userEvent.click(screen.getByRole('button', { name: '进入控制台' }))
    expect(screen.getByRole('alert').textContent).toContain('密码至少 8 个字符')
    expect(api.setupGUIPassword).not.toHaveBeenCalled()
  })

  it('submits setup when passwords match', async () => {
    vi.mocked(api.setupGUIPassword).mockResolvedValue(undefined)
    const onSuccess = vi.fn()
    render(<SetupPage onSuccess={onSuccess} />)
    await userEvent.type(screen.getByLabelText('面板密码'), 'long-enough-password')
    await userEvent.type(screen.getByLabelText('确认密码'), 'long-enough-password')
    await userEvent.click(screen.getByRole('button', { name: '进入控制台' }))
    await waitFor(() => expect(api.setupGUIPassword).toHaveBeenCalledWith('long-enough-password'))
    expect(onSuccess).toHaveBeenCalled()
  })

  it('shows an inline error for a wrong login without treating it as session expiry', async () => {
    vi.mocked(api.loginGUI).mockRejectedValue(new RequestError(401, 'invalid_password', 'password is incorrect'))
    render(<LoginPage onSuccess={vi.fn()} />)
    await userEvent.type(screen.getByLabelText('面板密码'), 'wrong-password-xx')
    await userEvent.click(screen.getByRole('button', { name: '进入控制台' }))
    expect((await screen.findByRole('alert')).textContent).toContain('密码不正确')
  })
})
