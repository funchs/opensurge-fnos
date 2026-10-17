import { useState, type FormEvent } from 'react'
import { api, RequestError } from '../api'
import { t } from '../i18n'
import { productEdition, productIcon, productTitle } from '../product'

export function SetupPage({ onSuccess }: { onSuccess: () => void }) {
  const edition = productEdition(null)
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (password.length < 8) {
      setError(t('密码至少 8 个字符'))
      return
    }
    if (password !== confirm) {
      setError(t('两次输入的密码不一致'))
      return
    }
    setBusy(true)
    setError('')
    try {
      await api.setupGUIPassword(password)
      onSuccess()
    } catch (cause) {
      if (cause instanceof RequestError && cause.code === 'password_already_configured') {
        onSuccess()
        return
      }
      setError(cause instanceof Error ? cause.message : String(cause))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-shell">
      <form className="auth-card" onSubmit={event => void submit(event)}>
        <img className="auth-mark" src={productIcon(edition)} alt="" />
        <p className="auth-kicker">{productTitle(edition)}</p>
        <h1>{t('设置面板密码')}</h1>
        <p className="auth-copy">{t('首次打开需要设置密码，之后局域网和 FN Connect 访问都要登录。')}</p>
        <label>{t('面板密码')}<input type="password" autoComplete="new-password" value={password} onChange={event => setPassword(event.target.value)} /></label>
        <label>{t('确认密码')}<input type="password" autoComplete="new-password" value={confirm} onChange={event => setConfirm(event.target.value)} /></label>
        {error && <p className="auth-error" role="alert">{error}</p>}
        <button className="primary" type="submit" disabled={busy}>{t('进入控制台')}</button>
      </form>
    </div>
  )
}
