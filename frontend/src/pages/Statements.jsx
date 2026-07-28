import { useEffect, useState } from 'react'
import { api, eur } from '../api'
import { useToast } from '../Toast'

export default function Statements({ onChanged }) {
  const toast = useToast()
  const [stmts, setStmts] = useState(null)
  const [err, setErr] = useState(null)

  const load = () => api.statements().then(setStmts).catch((e) => setErr(e.message))
  useEffect(() => { load() }, [])

  const remove = async (s) => {
    if (!confirm(`Supprimer le relevé « ${s.filename} » et ses ${s.tx_count} transactions ?`)) return
    try {
      await api.deleteStatement(s.id)
      toast.ok('Relevé supprimé.')
      load()
      onChanged?.()
    } catch (e) {
      toast.err(e.message)
    }
  }

  if (err) return <div className="notice err">{err}</div>
  if (!stmts) return <p style={{ color: 'var(--muted)' }}><span className="spinner" />Chargement…</p>

  return (
    <div className="section">
      {stmts.length === 0 ? (
        <div className="empty">Aucun relevé importé. Dépose un PDF via la fente en haut de page.</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Fichier</th>
              <th>Période</th>
              <th className="num">Solde début</th>
              <th className="num">Solde fin</th>
              <th className="num">Opérations</th>
              <th>Import</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {stmts.map((s) => (
              <tr key={s.id}>
                <td>{s.filename}</td>
                <td className="amt" style={{ fontSize: 13, color: 'var(--muted)' }}>
                  {s.period_start ?? '?'} → {s.period_end ?? '?'}
                </td>
                <td className="num amt">{s.opening_balance != null ? eur(s.opening_balance) : '—'}</td>
                <td className="num amt">{s.closing_balance != null ? eur(s.closing_balance) : '—'}</td>
                <td className="num amt">{s.tx_count}</td>
                <td>
                  <span className="badge">{s.parser === 'local' ? 'parseur local' : s.parser === 'csv' ? 'import CSV' : 'IA'}</span>{' '}
                  {s.validated
                    ? <span className="stamp ok" title="Solde initial + Σ transactions = solde final, au centime">Validé</span>
                    : <span className="stamp warn" title="La somme des transactions ne correspond pas au delta de solde : vérifie les montants">À vérifier</span>}
                </td>
                <td className="num">
                  <button className="btn-danger" onClick={() => remove(s)}>
                    Supprimer
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
