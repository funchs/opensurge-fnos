import { useState, type FormEvent } from 'react'
import { api, RequestError } from '../api'
import { t } from '../i18n'
import { productEdition, productIcon, productTitle } from '../product'

export function LoginPage({ onSuccess }: { onSuccess: () => void }) {
  const edition = productEdition(null)
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      await api.loginGUI(password)
      onSuccess()
    } catch (cause) {
      if (cause instanceof RequestError && cause.code === 'password_not_configured') {
        onSuccess()
        return
      }
      if (cause instanceof RequestError && cause.code === 'too_many_attempts') {
        setError(t('尝试次数过多，请稍后再试'))
      } else if (cause instanceof RequestError && cause.code === 'invalid_password') {
        setError(t('密码不正确'))
      } else {
        setError(cause instanceof Error ? cause.message : String(cause))
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-shell">
      <form className="auth-card" onSubmit={event => void submit(event)}>
        <img className="auth-mark" src={productIcon(edition)} alt="" />
        <p className="auth-kicker">{productTitle(edition)}</p>
        <h1>{t('登录')}</h1>
        <p className="auth-copy">{t('输入面板密码以继续。忘记密码时，删除 store 中的 gui-password 后重新设置。')}</p>
        <label>{t('面板密码')}<input type="password" autoComplete="current-password" value={password} onChange={event => setPassword(event.target.value)} /></label>
        {error && <p className="auth-error" role="alert">{error}</p>}
        <button className="primary" type="submit" disabled={busy}>{t('进入控制台')}</button>
      </form>
    </div>
  )
}
