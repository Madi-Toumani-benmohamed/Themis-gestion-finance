import { useEffect, useState } from 'react'
import { api } from '../api'
import { useToast } from '../Toast'
import AIConfiguration from '../components/AIConfiguration'

export default function Admin({ isAdmin = true, onAIConfigSaved }) {
  const toast = useToast()
  const [stats, setStats] = useState(null)
  const [err, setErr] = useState(null)

  const [busy, setBusy] = useState(false)

  // maintenance
  const [purgeText, setPurgeText] = useState('')

  // mot de passe personnel
  const [pw, setPw] = useState({ current: '', next: '', confirm: '' })

  // utilisateurs
  const [users, setUsers] = useState([])
  const [uForm, setUForm] = useState({
    username: '',
    password: '',
    role: 'user',
  })

  const loadAll = () => {
    if (!isAdmin) return
    api
      .adminStats()
      .then(setStats)
      .catch(() => {})
    api
      .users()
      .then((x) => setUsers(Array.isArray(x) ? x : []))
      .catch(() => {})
  }
  useEffect(() => {
    loadAll()
  }, [])

  const reapplyRules = async () => {
    setBusy(true)
    setErr(null)
    try {
      const r = await api.adminReapplyRules()
      toast.ok(
        `${r.categorized} transaction${r.categorized > 1 ? 's' : ''} classée${r.categorized > 1 ? 's' : ''} par les règles`,
      )
      loadAll()
    } catch (e) {
      setErr(e.message)
    } finally {
      setBusy(false)
    }
  }

  const purge = async () => {
    setBusy(true)
    setErr(null)
    try {
      await api.adminPurge(purgeText)
      setPurgeText('')
      toast.ok('Toutes les données ont été effacées.')
      loadAll()
    } catch (e) {
      setErr(e.message)
    } finally {
      setBusy(false)
    }
  }

  const addUser = async (e) => {
    e.preventDefault()
    setErr(null)
    try {
      await api.createUser(uForm.username.trim(), uForm.password, uForm.role)
      setUForm({ username: '', password: '', role: 'user' })
      toast.ok('Utilisateur créé.')
      loadAll()
    } catch (e2) {
      setErr(e2.message)
    }
  }
  const removeUser = async (u) => {
    if (!confirm(`Supprimer l'utilisateur « ${u.username} » ?`)) return
    try {
      await api.deleteUser(u.id)
      loadAll()
    } catch (e2) {
      setErr(e2.message)
    }
  }

  const changePassword = async (e) => {
    e.preventDefault()
    if (pw.next !== pw.confirm) {
      toast.err('Les deux mots de passe ne correspondent pas.')
      return
    }
    try {
      await api.changePassword(pw.current, pw.next)
      setPw({ current: '', next: '', confirm: '' })
      toast.ok(
        'Mot de passe modifié. Les autres sessions ont été déconnectées.',
      )
    } catch (e2) {
      toast.err(e2.message)
    }
  }

  return (
    <div className="section admin-page">
      {err && <div className="notice err">{err}</div>}
      {isAdmin && <AIConfiguration onApplied={onAIConfigSaved} />}

      {isAdmin && (
        <>
          <h2 style={{ marginTop: 36 }}>Statistiques</h2>
          {stats && (
            <div className="kpi-grid">
              {[
                ['Relevés', stats.statements],
                ['Transactions', stats.transactions],
                [
                  'Classées',
                  stats.transactions
                    ? `${stats.categorized} (${Math.round((stats.categorized / stats.transactions) * 100)} %)`
                    : 0,
                ],
                ['Catégories', stats.categories],
                ['Règles', stats.rules],
                ['Analyses IA', stats.analyses],
                ['Base', stats.db_size ?? '—'],
                [
                  'Historique',
                  stats.first_tx ? `${stats.first_tx} → ${stats.last_tx}` : '—',
                ],
              ].map(([label, value]) => (
                <div className="kpi-card" key={label}>
                  <div className="label">{label}</div>
                  <div className="value" style={{ fontSize: 16 }}>
                    {value}
                  </div>
                </div>
              ))}
            </div>
          )}

          <h2 style={{ marginTop: 36 }}>Utilisateurs</h2>
          <div className="admin-block">
            <table>
              <thead>
                <tr>
                  <th>Identifiant</th>
                  <th>Rôle</th>
                  <th>Créé le</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id}>
                    <td>{u.username}</td>
                    <td>
                      {u.role === 'admin' ? (
                        <span className="badge">admin</span>
                      ) : (
                        'utilisateur'
                      )}
                    </td>
                    <td className="muted">{u.created_at?.slice(0, 10)}</td>
                    <td>
                      <button className="link" onClick={() => removeUser(u)}>
                        retirer
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <form
              className="admin-row"
              style={{ marginTop: 14 }}
              onSubmit={addUser}
            >
              <input
                placeholder="Identifiant"
                value={uForm.username}
                onChange={(e) =>
                  setUForm({ ...uForm, username: e.target.value })
                }
                required
                style={{ width: 160 }}
              />
              <input
                type="password"
                placeholder="Mot de passe"
                value={uForm.password}
                onChange={(e) =>
                  setUForm({ ...uForm, password: e.target.value })
                }
                required
                style={{ width: 180 }}
                autoComplete="new-password"
              />
              <select
                value={uForm.role}
                onChange={(e) => setUForm({ ...uForm, role: e.target.value })}
              >
                <option value="user">Utilisateur</option>
                <option value="admin">Administrateur</option>
              </select>
              <button type="submit" className="btn">
                Créer
              </button>
            </form>
            <p className="muted" style={{ fontSize: 12 }}>
              10 caractères minimum, lettres et chiffres.
            </p>
          </div>

          <h2 style={{ marginTop: 36 }}>Maintenance</h2>
          <div className="admin-block">
            <div className="admin-row">
              <button
                className="btn-secondary"
                onClick={reapplyRules}
                disabled={busy}
              >
                Réappliquer les règles aux transactions non classées
              </button>
              <span className="muted" style={{ fontSize: 13 }}>
                utile après avoir ajouté ou modifié des règles
              </span>
            </div>
          </div>
        </>
      )}

      <h2 style={{ marginTop: 36 }}>Mon compte</h2>
      <div className="admin-block">
        <form
          className="admin-row"
          onSubmit={changePassword}
          style={{ flexWrap: 'wrap' }}
        >
          <input
            type="password"
            placeholder="Mot de passe actuel"
            value={pw.current}
            onChange={(e) => setPw({ ...pw, current: e.target.value })}
            required
            autoComplete="current-password"
            style={{ width: 180 }}
          />
          <input
            type="password"
            placeholder="Nouveau"
            value={pw.next}
            onChange={(e) => setPw({ ...pw, next: e.target.value })}
            required
            autoComplete="new-password"
            style={{ width: 160 }}
          />
          <input
            type="password"
            placeholder="Confirmer"
            value={pw.confirm}
            onChange={(e) => setPw({ ...pw, confirm: e.target.value })}
            required
            autoComplete="new-password"
            style={{ width: 160 }}
          />
          <button type="submit" className="btn">
            Changer le mot de passe
          </button>
        </form>
        <div className="admin-row" style={{ marginTop: 6 }}>
          <a className="btn-secondary" href="/api/export" download>
            Exporter toutes mes données (JSON)
          </a>
          <span className="muted" style={{ fontSize: 13 }}>
            sauvegarde et portabilité — comptes, transactions, règles, budgets…
          </span>
        </div>
      </div>

      {isAdmin && (
        <div className="admin-block danger-zone">
          <b>Zone dangereuse.</b>
          <p className="muted" style={{ fontSize: 13 }}>
            Efface tous les relevés, transactions et analyses. Catégories,
            règles, budgets et réglages sont conservés. Irréversible — tape{' '}
            <code>TOUT EFFACER</code> pour confirmer.
          </p>
          <div className="admin-row">
            <input
              value={purgeText}
              onChange={(e) => setPurgeText(e.target.value)}
              placeholder="TOUT EFFACER"
              style={{ width: 180 }}
            />
            <button
              className="btn-danger"
              onClick={purge}
              disabled={busy || purgeText !== 'TOUT EFFACER'}
            >
              Purger les données
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
