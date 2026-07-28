import { useEffect, useState } from 'react'
import { api, eur, monthLabel } from '../api'
import { useToast } from '../Toast'

export default function Credits() {
  const toast = useToast()
  const [credits, setCredits] = useState(null)
  const [taxes, setTaxes] = useState(null)
  const [obl, setObl] = useState(null)
  const [err, setErr] = useState(null)
  const [cForm, setCForm] = useState(null)
  const [tForm, setTForm] = useState(null)

  const load = () => {
    api.credits().then(setCredits).catch((e) => setErr(e.message))
    api.taxes().then(setTaxes).catch((e) => setErr(e.message))
    api.obligations().then(setObl).catch(() => {})
  }
  useEffect(() => { load() }, [])

  const addCredit = async (e) => {
    e.preventDefault()
    try {
      await api.createCredit({
        name: cForm.name.trim(),
        monthly_amount: +cForm.monthly,
        months_total: +cForm.months,
        start_month: cForm.start,
        note: cForm.note || '',
      })
      setCForm(null); toast.ok('Crédit ajouté.'); load()
    } catch (e2) { toast.err(e2.message) }
  }
  const addTax = async (e) => {
    e.preventDefault()
    try {
      await api.createTax({ name: tForm.name.trim(), amount: +tForm.amount, due_date: tForm.due })
      setTForm(null); toast.ok('Taxe ajoutée.'); load()
    } catch (e2) { toast.err(e2.message) }
  }

  if (!credits || !taxes) return <p className="muted"><span className="spinner" />Chargement…</p>

  return (
    <div className="section">
      {err && <div className="notice err">{err}</div>}

      {obl && (
        <div className="debt-banner">
          <div className="debt-cell">
            <span className="label">Dette totale</span>
            <span className="value debit">{eur(obl.total_debt)}</span>
          </div>
          <div className="debt-cell">
            <span className="label">Crédits restants</span>
            <span className="value">{eur(obl.credits_remaining)}</span>
            <span className="sub">{eur(obl.monthly_load)}/mois</span>
          </div>
          <div className="debt-cell">
            <span className="label">Taxes à payer</span>
            <span className="value">{eur(obl.taxes_due)}</span>
            {obl.taxes_due_soon > 0 && <span className="sub warn-txt">{obl.taxes_due_soon} sous 30 j</span>}
          </div>
        </div>
      )}

      <div className="section-head">
        <h2>Crédits en cours</h2>
        {!cForm && <button className="btn-secondary" onClick={() => setCForm({ name: '', monthly: '', months: '', start: new Date().toISOString().slice(0, 7), note: '' })}>＋ Crédit</button>}
      </div>

      {cForm && (
        <form className="admin-block" onSubmit={addCredit}>
          <div className="admin-row">
            <input placeholder="Nom (ex. Prêt auto)" value={cForm.name} onChange={(e) => setCForm({ ...cForm, name: e.target.value })} required style={{ width: 200 }} />
            <input type="number" step="0.01" placeholder="Mensualité €" value={cForm.monthly} onChange={(e) => setCForm({ ...cForm, monthly: e.target.value })} required style={{ width: 130 }} />
            <input type="number" placeholder="Nb mois" value={cForm.months} onChange={(e) => setCForm({ ...cForm, months: e.target.value })} required style={{ width: 100 }} />
            <label style={{ fontSize: 13, color: 'var(--muted)' }}>1re éch. <input type="month" value={cForm.start} onChange={(e) => setCForm({ ...cForm, start: e.target.value })} required /></label>
          </div>
          <div className="admin-actions">
            <button type="submit" className="btn">Ajouter</button>
            <button type="button" className="link" onClick={() => setCForm(null)}>annuler</button>
          </div>
        </form>
      )}

      {credits.length === 0 ? (
        <div className="empty">Aucun crédit enregistré.</div>
      ) : (
        <div className="credit-list">
          {credits.map((c, i) => {
            const pct = (c.months_paid / c.months_total) * 100
            const done = c.months_remaining === 0
            return (
              <div className="credit-card" key={c.id} style={{ '--i': i }}>
                <div className="credit-head">
                  <div>
                    <b>{c.name}</b>
                    {c.note && <span className="muted"> — {c.note}</span>}
                  </div>
                  <button className="link" onClick={() => api.deleteCredit(c.id).then(() => { toast.ok('Crédit retiré.'); load() }).catch((e) => toast.err(e.message))}>retirer</button>
                </div>
                <div className="credit-progress">
                  <div className="gauge"><div className={'gauge-fill' + (done ? ' done' : '')} style={{ width: pct + '%' }} /></div>
                  <div className="credit-nums">
                    <span>{done ? 'Soldé' : `${c.months_remaining} mensualité${c.months_remaining > 1 ? 's' : ''} restante${c.months_remaining > 1 ? 's' : ''}`}</span>
                    <span className="muted">{c.months_paid}/{c.months_total} · fin {monthLabel(c.end_month)}</span>
                  </div>
                </div>
                <div className="credit-foot">
                  <span className="amount">{eur(c.monthly_amount)}<span className="muted">/mois</span></span>
                  {!done && <span className="amount debit">reste {eur(c.remaining)}</span>}
                </div>
              </div>
            )
          })}
        </div>
      )}

      <div className="section-head" style={{ marginTop: 34 }}>
        <h2>Taxes &amp; échéances</h2>
        {!tForm && <button className="btn-secondary" onClick={() => setTForm({ name: '', amount: '', due: '' })}>＋ Taxe</button>}
      </div>

      {tForm && (
        <form className="admin-block" onSubmit={addTax}>
          <div className="admin-row">
            <input placeholder="Nom (ex. Taxe foncière)" value={tForm.name} onChange={(e) => setTForm({ ...tForm, name: e.target.value })} required style={{ width: 220 }} />
            <input type="number" step="0.01" placeholder="Montant €" value={tForm.amount} onChange={(e) => setTForm({ ...tForm, amount: e.target.value })} required style={{ width: 130 }} />
            <label style={{ fontSize: 13, color: 'var(--muted)' }}>échéance <input type="date" value={tForm.due} onChange={(e) => setTForm({ ...tForm, due: e.target.value })} required /></label>
          </div>
          <div className="admin-actions">
            <button type="submit" className="btn">Ajouter</button>
            <button type="button" className="link" onClick={() => setTForm(null)}>annuler</button>
          </div>
        </form>
      )}

      {taxes.length === 0 ? (
        <div className="empty">Aucune taxe enregistrée.</div>
      ) : (
        <table>
          <thead><tr><th></th><th>Taxe</th><th className="num">Montant</th><th>Échéance</th><th></th></tr></thead>
          <tbody>
            {taxes.map((t) => {
              const urgent = !t.paid && t.days_left <= 30
              const late = !t.paid && t.days_left < 0
              return (
                <tr key={t.id}>
                  <td style={{ width: 34 }}>
                    <input type="checkbox" checked={t.paid} onChange={(e) => api.setTaxPaid(t.id, e.target.checked).then(load).catch((er) => toast.err(er.message))}
                           title="Marquer payée" style={{ accentColor: 'var(--green)' }} />
                  </td>
                  <td className={t.paid ? 'muted paid' : ''}>{t.name}</td>
                  <td className="num amount">{eur(t.amount)}</td>
                  <td>
                    {t.due_date}
                    {!t.paid && (
                      <span className={'due-chip' + (late ? ' late' : urgent ? ' soon' : '')}>
                        {late ? `en retard de ${-t.days_left} j` : t.days_left === 0 ? "aujourd'hui" : `dans ${t.days_left} j`}
                      </span>
                    )}
                  </td>
                  <td><button className="link" onClick={() => api.deleteTax(t.id).then(() => { toast.ok('Taxe retirée.'); load() }).catch((e) => toast.err(e.message))}>retirer</button></td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </div>
  )
}
