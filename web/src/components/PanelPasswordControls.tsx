import { useState, type FormEvent } from 'react'
import { api, RequestError } from '../api'
import { t } from '../i18n'

export function PanelPasswordControls({ onLogout }: { onLogout: () => void }) {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (next.length < 8) {
      setError(t('密码至少 8 个字符'))
      setMessage('')
      return
    }
    if (next !== confirm) {
      setError(t('两次输入的密码不一致'))
      setMessage('')
      return
    }
    setBusy(true)
    setError('')
    setMessage('')
    try {
      await api.changeGUIPassword(current, next)
      setCurrent('')
      setNext('')
      setConfirm('')
      setMessage(t('密码已更新'))
    } catch (cause) {
      if (cause instanceof RequestError && cause.code === 'invalid_password') {
        setError(t('密码不正确'))
      } else {
        setError(cause instanceof Error ? cause.message : String(cause))
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="panel-password-controls">
      <form onSubmit={event => void submit(event)}>
        <strong>{t('修改面板密码')}</strong>
        <label>{t('当前密码')}<input type="password" autoComplete="current-password" value={current} onChange={event => setCurrent(event.target.value)} /></label>
        <label>{t('新密码')}<input type="password" autoComplete="new-password" value={next} onChange={event => setNext(event.target.value)} /></label>
        <label>{t('确认密码')}<input type="password" autoComplete="new-password" value={confirm} onChange={event => setConfirm(event.target.value)} /></label>
        {error && <small className="sidebar-control-error" role="alert">{error}</small>}
        {message && <small className="sidebar-control-hint" role="status">{message}</small>}
        <button type="submit" className="sidebar-control-row" disabled={busy}><span className="sidebar-control-copy"><strong>{t('保存密码')}</strong></span></button>
      </form>
      <button type="button" className="sidebar-control-row" onClick={() => void onLogout()}><span className="sidebar-control-copy"><strong>{t('退出登录')}</strong></span></button>
    </div>
  )
}
