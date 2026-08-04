import { useEffect, useState } from 'react'
import { api, eur } from '../api'

export default function Simulate() {
  const [result, setResult] = useState(null)
  const [err, setErr] = useState(null)
  const [busy, setBusy] = useState(false)

  const [income, setIncome] = useState('')
  const [cuts, setCuts] = useState({}) // category -> pct
  const [expenses, setExpenses] = useState([{ label: '', amount: '' }])
  const [loan, setLoan] = useState({ amount: '', rate: '', months: '' })

  // amorce : récupère les catégories via une première simulation à vide
  useEffect(() => { run(true) }, [])

  const run = async (silent) => {
    if (!silent) setBusy(true)
    setErr(null)
    try {
      const scenario = {
        base_income: income ? +income : null,
        cuts: Object.entries(cuts).filter(([, p]) => +p > 0).map(([category, p]) => ({ category, reduce_pct: +p })),
        new_expenses: expenses.filter((e) => e.label && +e.amount > 0).map((e) => ({ label: e.label, amount: +e.amount })),
        loan: loan.amount && loan.months ? { amount: +loan.amount, rate: +loan.rate || 0, months: +loan.months } : null,
      }
      setResult(await api.simulate(scenario))
    } catch (e) { setErr(e.message) } finally { if (!silent) setBusy(false) }
  }

  const setCut = (cat, pct) => setCuts((c) => ({ ...c, [cat]: pct }))
  const setExp = (i, k, v) => setExpenses((es) => es.map((e, j) => (j === i ? { ...e, [k]: v } : e)))
  const addExp = () => setExpenses((es) => [...es, { label: '', amount: '' }])
  const rmExp = (i) => setExpenses((es) => es.filter((_, j) => j !== i))

  return (
    <div className="section">
      {err && <div className="notice err">{err}</div>}

      <div className="section-head"><h2>Simulateur budgétaire</h2></div>
      <p className="muted" style={{ marginTop: -8 }}>
        Point de départ : tes moyennes réelles des 3 derniers mois. Ajuste, et vois l'effet sur ton reste à vivre.
      </p>

      <div className="sim-layout">
        <div className="sim-controls">
          <div className="admin-block">
            <h3 className="sim-h">Revenu mensuel</h3>
            <input type="number" step="0.01" placeholder="auto (moyenne observée)" value={income}
                   onChange={(e) => setIncome(e.target.value)} style={{ width: 220 }} />
          </div>

          <div className="admin-block">
            <h3 className="sim-h">Réduire des dépenses</h3>
            {(result?.categories ?? []).map((c) => (
              <div className="sim-cut" key={c.category}>
                <span className="sim-cut-name">{c.category}</span>
                <input type="range" min="0" max="100" step="5" value={cuts[c.category] ?? 0}
                       onChange={(e) => setCut(c.category, e.target.value)} />
                <span className="sim-cut-pct">−{cuts[c.category] ?? 0} %</span>
                <span className="sim-cut-amt muted">{eur(c.before)}</span>
              </div>
            ))}
            {(!result || (result.categories ?? []).length === 0) && <p className="muted">Importe des relevés pour voir tes postes de dépense.</p>}
          </div>

          <div className="admin-block">
            <h3 className="sim-h">Nouvelles dépenses</h3>
            {expenses.map((e, i) => (
              <div className="admin-row" key={i}>
                <input placeholder="Intitulé" value={e.label} onChange={(ev) => setExp(i, 'label', ev.target.value)} style={{ width: 180 }} />
                <input type="number" step="0.01" placeholder="€/mois" value={e.amount} onChange={(ev) => setExp(i, 'amount', ev.target.value)} style={{ width: 110 }} />
                {expenses.length > 1 && <button className="link" onClick={() => rmExp(i)}>retirer</button>}
              </div>
            ))}
            <button className="link" onClick={addExp}>＋ ajouter une ligne</button>
          </div>

          <div className="admin-block">
            <h3 className="sim-h">Simuler un emprunt</h3>
            <div className="admin-row">
              <input type="number" step="0.01" placeholder="Montant €" value={loan.amount} onChange={(e) => setLoan({ ...loan, amount: e.target.value })} style={{ width: 130 }} />
              <input type="number" step="0.01" placeholder="Taux %/an" value={loan.rate} onChange={(e) => setLoan({ ...loan, rate: e.target.value })} style={{ width: 110 }} />
              <input type="number" placeholder="Durée (mois)" value={loan.months} onChange={(e) => setLoan({ ...loan, months: e.target.value })} style={{ width: 120 }} />
            </div>
          </div>

          <button className="btn" onClick={() => run(false)} disabled={busy}>
            {busy && <span className="spinner" />} Calculer
          </button>
        </div>

        {result && (
          <div className="sim-result">
            <div className="sim-verdict">
              <span className="label">Reste à vivre simulé</span>
              <span className={'sim-net ' + (result.simulated_net >= 0 ? 'credit' : 'debit')}>
                {eur(result.simulated_net)}<span className="sim-unit">/mois</span>
              </span>
              <span className={'sim-delta ' + (result.delta >= 0 ? 'credit' : 'debit')}>
                {result.delta >= 0 ? '▲ +' : '▼ '}{eur(result.delta)} vs situation actuelle
              </span>
            </div>

            <table className="sim-table">
              <tbody>
                <tr><td>Revenu</td><td className="num amount">{eur(result.income)}</td></tr>
                <tr><td>Dépenses actuelles</td><td className="num amount">{eur(result.base_spend)}</td></tr>
                <tr><td>Reste à vivre actuel</td><td className="num amount">{eur(result.current_net)}</td></tr>
                <tr className="sim-sep"><td>Économies (coupes)</td><td className="num amount credit">{eur(result.monthly_saved)}</td></tr>
                <tr><td>Nouvelles dépenses</td><td className="num amount debit">{eur(result.new_expenses)}</td></tr>
                {result.loan_payment > 0 && (
                  <>
                    <tr><td>Mensualité d'emprunt</td><td className="num amount debit">{eur(result.loan_payment)}</td></tr>
                    <tr><td className="muted">Coût total du crédit</td><td className="num amount muted">{eur(result.loan_total_cost)}</td></tr>
                    <tr><td className="muted">dont intérêts</td><td className="num amount muted">{eur(result.loan_interest)}</td></tr>
                  </>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
