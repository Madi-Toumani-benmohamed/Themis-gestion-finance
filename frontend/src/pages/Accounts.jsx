import { useEffect, useState } from 'react'
import { api, eur } from '../api'
import { useToast } from '../Toast'

const KINDS = [
  ['courant', 'Compte courant'],
  ['livret_a', 'Livret A'],
  ['ldds', 'LDDS'],
  ['pel', 'PEL'],
  ['bourse', 'Compte-titres / PEA'],
  ['autre', 'Autre'],
]
const kindLabel = (k) => (KINDS.find(([v]) => v === k)?.[1] ?? k)

export default function Accounts() {
  const toast = useToast()
  const [accounts, setAccounts] = useState(null)
  const [err, setErr] = useState(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({ name: '', kind: 'livret_a', interest_rate: '', deposit_limit: '' })
  const [snapFor, setSnapFor] = useState(null)
  const [snap, setSnap] = useState({ date: new Date().toISOString().slice(0, 10), balance: '' })

  const load = () => api.accounts().then((x) => setAccounts(Array.isArray(x) ? x : [])).catch((e) => setErr(e.message))
  useEffect(() => { load() }, [])

  const create = async (e) => {
    e.preventDefault()
    try {
      await api.createAccount({
        name: form.name.trim(),
        kind: form.kind,
        interest_rate: form.interest_rate ? +form.interest_rate : null,
        deposit_limit: form.deposit_limit ? +form.deposit_limit : null,
      })
      setForm({ name: '', kind: 'livret_a', interest_rate: '', deposit_limit: '' })
      setAdding(false)
      toast.ok('Compte créé.')
      load()
    } catch (e2) { toast.err(e2.message) }
  }

  const saveSnap = async (e) => {
    e.preventDefault()
    try {
      await api.addSnapshot(snapFor, snap.date, +snap.balance)
      setSnapFor(null); setSnap({ date: new Date().toISOString().slice(0, 10), balance: '' })
      toast.ok('Solde pointé.')
      load()
    } catch (e2) { toast.err(e2.message) }
  }

  const remove = async (a) => {
    if (!confirm(`Supprimer le compte « ${a.name} » ?`)) return
    try { await api.deleteAccount(a.id); toast.ok('Compte supprimé.'); load() } catch (e2) { toast.err(e2.message) }
  }

  if (!accounts) return <p className="muted"><span className="spinner" />Chargement…</p>

  const total = accounts.reduce((s, a) => s + (a.balance || 0), 0)

  return (
    <div className="section">
      {err && <div className="notice err">{err}</div>}

      <div className="section-head">
        <h2>Mes comptes</h2>
        <div className="patrimoine">Patrimoine connu <b>{eur(total)}</b></div>
      </div>

      <div className="account-grid">
        {accounts.map((a, i) => {
          const fill = a.deposit_limit && a.balance != null
            ? Math.min((a.balance / a.deposit_limit) * 100, 100) : null
          return (
            <div className="account-card" key={a.id} style={{ '--i': i }}>
              <div className="account-top">
                <span className="account-kind">{kindLabel(a.kind)}</span>
                {a.id !== 1 && <button className="link" onClick={() => remove(a)}>retirer</button>}
              </div>
              <div className="account-name">{a.name}</div>
              <div className="account-balance">{a.balance != null ? eur(a.balance) : '—'}</div>
              <label className="account-tracking">Mode de suivi
                <select aria-label={`Mode de suivi de ${a.name}`} value={a.tracking_mode || (a.kind==='courant'?'statements':'manual')} onChange={async e=>{try{await api.updateAccountTracking(a.id,e.target.value);toast.ok('Mode de suivi enregistré.');load()}catch(error){toast.err(error.message)}}}>
                  <option value="statements">Relevés et contrôle documentaire</option><option value="manual">Pointage / valorisation manuelle</option>
                </select>
              </label>
              <div className="account-meta">
                {a.balance_date && <span>au {a.balance_date}</span>}
                {a.interest_rate != null && <span>· {a.interest_rate} %/an</span>}
                {a.statements > 0 && <span>· {a.statements} relevé{a.statements > 1 ? 's' : ''}</span>}
              </div>
              {fill != null && (
                <div className="account-limit">
                  <div className="gauge"><div className="gauge-fill" style={{ width: fill + '%' }} /></div>
                  <span className="muted">plafond {eur(a.deposit_limit)}</span>
                </div>
              )}
              {snapFor === a.id ? (
                <form className="snap-form" onSubmit={saveSnap}>
                  <input type="date" value={snap.date} onChange={(e) => setSnap({ ...snap, date: e.target.value })} required />
                  <input type="number" step="0.01" placeholder="Solde €" value={snap.balance}
                         onChange={(e) => setSnap({ ...snap, balance: e.target.value })} required />
                  <button type="submit" className="btn-secondary">OK</button>
                  <button type="button" className="link" onClick={() => setSnapFor(null)}>×</button>
                </form>
              ) : (
                <button className="btn-secondary account-snap-btn" onClick={() => setSnapFor(a.id)}>
                  Pointer le solde
                </button>
              )}
            </div>
          )
        })}
      </div>

      {adding ? (
        <form className="admin-block account-add" onSubmit={create}>
          <div className="admin-row">
            <input placeholder="Nom du compte" value={form.name}
                   onChange={(e) => setForm({ ...form, name: e.target.value })} required style={{ width: 200 }} />
            <select value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}>
              {KINDS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
            </select>
          </div>
          <div className="admin-row">
            <input type="number" step="0.01" placeholder="Taux %/an (option.)" value={form.interest_rate}
                   onChange={(e) => setForm({ ...form, interest_rate: e.target.value })} style={{ width: 160 }} />
            <input type="number" step="0.01" placeholder="Plafond € (option.)" value={form.deposit_limit}
                   onChange={(e) => setForm({ ...form, deposit_limit: e.target.value })} style={{ width: 160 }} />
          </div>
          <div className="admin-actions">
            <button type="submit" className="btn">Créer le compte</button>
            <button type="button" className="link" onClick={() => setAdding(false)}>annuler</button>
          </div>
        </form>
      ) : (
        <button className="btn-secondary" onClick={() => setAdding(true)}>＋ Ajouter un compte</button>
      )}

      <p className="muted" style={{ fontSize: 13, marginTop: 14 }}>
        Les relevés PDF alimentent le solde des comptes courants ; pour un livret ou un compte-titres,
        pointe le solde à la main de temps en temps.
      </p>
    </div>
  )
}
