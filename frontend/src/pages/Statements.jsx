import { useEffect, useRef, useState } from 'react'
import { api, eur } from '../api'
import { useToast } from '../Toast'
import { DataReliability } from '../components/DataReliability'

export default function Statements({ onChanged, initialMonth, onMonthChange, onGoUpload }) {
  const toast = useToast()
  const now = new Date()
  const [month, setMonth] = useState(initialMonth || `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`)
  const [inspectedIds, setInspectedIds] = useState(null)
  const tableTitle = useRef(null)
  const chooseMonth = value => { setMonth(value); setInspectedIds(null); onMonthChange?.(value) }
  const inspect = ids => { setInspectedIds(new Set(ids)); tableTitle.current?.focus() }
  const [stmts, setStmts] = useState(null)
  const [err, setErr] = useState(null)

  const load = () => api.statements().then((x) => setStmts(Array.isArray(x) ? x : [])).catch((e) => setErr(e.message))
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

  const visibleStatements = inspectedIds ? stmts.filter(s => inspectedIds.has(s.id)) : stmts

  return (
    <div className="section">
      <DataReliability month={month} onMonthChange={chooseMonth} onInspect={inspect} onGoUpload={onGoUpload} />
      <div className="statements-table-header">
        <h2 ref={tableTitle} tabIndex={-1}>{inspectedIds ? 'Relevés concernés' : 'Tous les relevés'}</h2>
        {inspectedIds && <button className="btn-secondary" onClick={() => setInspectedIds(null)}>Afficher tous les relevés</button>}
      </div>
      {stmts.length === 0 ? (
        <div className="empty">Aucun relevé importé. Dépose un PDF via la fente en haut de page.</div>
      ) : (
        <div className="statements-table-scroll"><table>
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
            {visibleStatements.map((s) => (
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
                    : <span className="stamp warn" title="Équilibre comptable non confirmé : comparer avec le relevé original. Un CSV peut ne pas fournir de soldes de contrôle.">À vérifier</span>}
                </td>
                <td className="num">
                  <button className="btn-danger" onClick={() => remove(s)}>
                    Supprimer
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table></div>
      )}
    </div>
  )
}
