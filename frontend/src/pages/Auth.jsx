import { useState } from 'react'
import { Logo } from '../Logo'
import { api } from '../api'

export default function Auth({ setupRequired, onAuthed }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [err, setErr] = useState(null)
  const [busy, setBusy] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setErr(null)
    if (setupRequired && password !== confirm) {
      setErr('Les deux mots de passe ne correspondent pas.')
      return
    }
    setBusy(true)
    try {
      const u = setupRequired
        ? await api.authSetup(username.trim(), password)
        : await api.authLogin(username.trim(), password)
      onAuthed(u)
    } catch (e2) {
      setErr(e2.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-screen">
      <div className="auth-card">
        <div className="guilloche auth-guilloche" aria-hidden="true" />
        <div className="auth-logo"><Logo size={44} /></div>
        <h1 className="auth-brand">thémis<span className="tick">.</span></h1>
        <p className="auth-tagline">
          {setupRequired ? 'Première ouverture — crée ton compte administrateur.' : 'La balance de tes comptes.'}
        </p>

        <form onSubmit={submit} className="auth-form">
          <label>
            Identifiant
            <input value={username} onChange={(e) => setUsername(e.target.value)}
                   autoComplete="username" autoFocus required />
          </label>
          <label>
            Mot de passe
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)}
                   autoComplete={setupRequired ? 'new-password' : 'current-password'} required />
          </label>
          {setupRequired && (
            <>
              <label>
                Confirme le mot de passe
                <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)}
                       autoComplete="new-password" required />
              </label>
              <p className="auth-hint">10 caractères minimum, mêlant lettres et chiffres.</p>
            </>
          )}
          {err && <div className="notice err">{err}</div>}
          <button type="submit" className="btn" disabled={busy}>
            {busy && <span className="spinner" />}
            {setupRequired ? 'Créer le compte' : 'Se connecter'}
          </button>
        </form>
      </div>
      <p className="auth-foot">Tes finances, à ton rythme.</p>
    </div>
  )
}
