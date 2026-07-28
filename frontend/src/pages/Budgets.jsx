import { useEffect, useMemo, useState } from 'react'
import { api, eur, monthLabel } from '../api'
import { useToast } from '../Toast'

export default function Budgets() {
  const toast = useToast()
  const [budgets, setBudgets] = useState([])
  const [recurring, setRecurring] = useState([])
  const [domains, setDomains] = useState([])
  const [month, setMonth] = useState(new Date().toISOString().slice(0, 7))
  const [months, setMonths] = useState([])
  const [newDomain, setNewDomain] = useState('')
  const [newLimit, setNewLimit] = useState('')
  const [err, setErr] = useState(null)

  const load = (m) => {
    api.budgets(m).then(setBudgets).catch((e) => setErr(e.message))
    api.recurring().then(setRecurring).catch((e) => setErr(e.message))
  }

  useEffect(() => {
    api.domains().then((ds) => setDomains(ds.filter((d) => !d.parent_id))).catch((e) => setErr(e.message))
    api.summary(24).then((d) => {
      const ms = (d.monthly ?? []).map((x) => x.month).reverse()
      setMonths(ms)
      const initial = ms[0] || new Date().toISOString().slice(0, 7)
      setMonth(initial)
      load(initial)
    }).catch((e) => setErr(e.message))
  }, [])

  const changeMonth = (m) => { setMonth(m); load(m) }

  const addBudget = async (e) => {
    e.preventDefault()
    if (!newLimit || !newDomain) return
    try {
      await api.setBudget(+newDomain, +newLimit)
      setNewDomain(''); setNewLimit('')
      toast.ok('Budget défini.')
      load(month)
    } catch (e2) { toast.err(e2.message) }
  }

  const remove = async (b) => {
    try { await api.deleteBudget(b.id); toast.ok('Budget retiré.'); load(month) } catch (e2) { toast.err(e2.message) }
  }

  const withoutBudget = useMemo(
    () => domains.filter((d) => !budgets.some((b) => b.domain_id === d.id)),
    [domains, budgets],
  )

  const subs = recurring.filter((r) => r.regular)
  const others = recurring.filter((r) => !r.regular)
  const subsTotal = subs.reduce((s, r) => s + r.avg_amount, 0)
  const isCurrentMonth = month === new Date().toISOString().slice(0, 7)

  return (
    <div className="section">
      {err && <p className="error">{err}</p>}

      <div className="section-head">
        <h2>Budgets</h2>
        <select value={month} onChange={(e) => changeMonth(e.target.value)}>
          {!months.includes(month) && <option value={month}>{monthLabel(month)}</option>}
          {months.map((m) => <option key={m} value={m}>{monthLabel(m)}</option>)}
        </select>
      </div>

      {budgets.length === 0 && (
        <div className="empty"><p>Aucun budget défini. Fixe un plafond mensuel par catégorie ci-dessous&nbsp;:
        tu verras la consommation se remplir à chaque relevé importé.</p></div>
      )}

      <div className="budget-list">
        {budgets.map((b) => {
          const pct = Math.min(b.ratio * 100, 100)
          const over = b.ratio >= 1
          const warn = !over && (isCurrentMonth ? b.projected > b.monthly_limit : b.ratio >= 0.85)
          return (
            <div className="budget" key={b.id}>
              <div className="budget-head">
                <b>{b.domain_name}</b>
                <span className={'amount ' + (over ? 'neg' : '')}>
                  {eur(b.spent)} / {eur(b.monthly_limit)}
                </span>
              </div>
              <div className="gauge"><div
                className={'gauge-fill' + (over ? ' over' : warn ? ' warn' : '')}
                style={{ width: pct + '%' }}
              /></div>
              <div className="budget-foot">
                {over && <span className="neg">dépassé de {eur(b.spent - b.monthly_limit)}</span>}
                {!over && warn && isCurrentMonth && <span className="warn-txt">projection fin de mois&nbsp;: {eur(b.projected)}</span>}
                {!over && !warn && <span className="muted">reste {eur(b.monthly_limit - b.spent)}</span>}
                <button className="link" onClick={() => remove(b)}>retirer</button>
              </div>
            </div>
          )
        })}
      </div>

      {withoutBudget.length > 0 ? (
        <form className="budget-add" onSubmit={addBudget}>
          <select value={newDomain} onChange={(e) => setNewDomain(e.target.value)}>
            <option value="">Catégorie…</option>
            {withoutBudget.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
          </select>
          <input type="number" min="1" step="1" placeholder="Plafond €/mois"
                 value={newLimit} onChange={(e) => setNewLimit(e.target.value)} />
          <button type="submit" disabled={!newDomain || !newLimit}>Définir le budget</button>
        </form>
      ) : (
        <p className="muted" style={{ fontSize: 13 }}>Toutes les catégories ont un budget. Retire-en un pour le redéfinir.</p>
      )}

      <div className="section-head" style={{ marginTop: 36 }}>
        <h2>Abonnements &amp; récurrents</h2>
        {subs.length > 0 && <span className="muted">≈ {eur(subsTotal)} / mois d'abonnements</span>}
      </div>

      {recurring.length === 0 && (
        <div className="empty"><p>Rien de récurrent détecté pour l'instant — il faut au moins deux mois
        de relevés pour repérer les paiements qui reviennent.</p></div>
      )}

      {subs.length > 0 && (
        <table className="table">
          <thead><tr><th>Abonnement probable</th><th>Montant</th><th>Vu sur</th><th>Dernier</th></tr></thead>
          <tbody>
            {subs.map((r, i) => (
              <tr key={i}>
                <td>{r.label}</td>
                <td className="amount neg">{eur(-r.amount)}</td>
                <td className="muted">{r.months.length} mois</td>
                <td className="muted">{r.last_date}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {others.length > 0 && (
        <>
          <h3 className="muted" style={{ marginTop: 20 }}>Dépenses qui reviennent (montant variable)</h3>
          <table className="table">
            <thead><tr><th>Libellé</th><th>Moyenne</th><th>Dernier</th><th>Vu sur</th></tr></thead>
            <tbody>
              {others.map((r, i) => (
                <tr key={i}>
                  <td>{r.label}</td>
                  <td className="amount neg">{eur(-r.avg_amount)}</td>
                  <td className="amount neg">{eur(-r.amount)}</td>
                  <td className="muted">{r.months.length} mois</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </div>
  )
}
