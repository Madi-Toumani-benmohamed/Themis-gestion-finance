import { useEffect, useRef, useState } from 'react'
import { api } from '../api'
import {
  creditPayload,
  decimalInput,
  inputAmount,
  formatDate,
  formatEuro,
  moneyText,
} from './creditsPageModel.mjs'

function Field({ label, name, form, setForm, ...props }) {
  return (
    <label className="form-field">
      <span>{label}</span>
      <input
        className="form-input"
        name={name}
        value={form[name]}
        onChange={(e) => setForm((f) => ({ ...f, [name]: e.target.value }))}
        {...props}
      />
    </label>
  )
}
export function CreditDialog({ action, onClose, onSaved }) {
  const { kind, mode, row } = action
  const credit = kind === 'credits'
  const [form, setForm] = useState({
    name: row?.name || '',
    monthly: inputAmount(row, 'monthly_amount'),
    months: row?.months_total || '',
    start: row?.start_month || new Date().toISOString().slice(0, 7),
    note: row?.note || '',
    opening: inputAmount(row, 'opening_paid'),
    amount: mode === 'payment' ? '' : inputAmount(row, 'amount'),
    due: row?.due_date || '',
    date: new Date().toISOString().slice(0, 10),
  })
  const [error, setError] = useState(''),
    [busy, setBusy] = useState(false)
  const lock = useRef(false),
    ref = useRef(null),
    submission = useRef(null),
    trigger = useRef(document.activeElement)
  const title = `${mode === 'delete' ? 'Retirer' : mode === 'payment' ? 'Déclarer un paiement' : mode === 'close' ? 'Clôturer' : mode === 'reopen' ? 'Rouvrir' : mode === 'late' ? (row?.overdue_confirmed ? 'Retirer la confirmation de retard' : 'Confirmer le retard') : mode === 'reset' ? 'Corriger le montant payé' : mode === 'edit' ? 'Modifier' : 'Ajouter'}${mode === 'payment' ? '' : credit ? ' un crédit' : ' une échéance'}`
  useEffect(() => {
    ref.current.showModal()
    return () => {
      ref.current?.close()
      ;(trigger.current?.isConnected
        ? trigger.current
        : document.querySelector('.credits-context h2')
      )?.focus()
    }
  }, [])
  // A changed payment is a different submission; a retry of the same payload keeps its key.
  const changeForm = (update) => {
    submission.current = null
    setForm(update)
  }
  const close = () => {
    if (!lock.current) onClose()
  }
  async function save(e) {
    e.preventDefault()
    if (lock.current) return
    lock.current = true
    setBusy(true)
    setError('')
    try {
      if (mode === 'delete')
        await (credit ? api.deleteCredit(row.id) : api.deleteTax(row.id))
      else if (mode === 'close' || mode === 'reopen')
        await api.updateCredit(row.id, { closed: mode === 'close' })
      else if (mode === 'late')
        await api.patchTax(row.id, {
          overdue_confirmed: !row.overdue_confirmed,
        })
      else if (mode === 'reset')
        await api.patchTax(row.id, {
          paid_amount: decimalInput(form.amount, false),
          note: form.note,
        })
      else if (mode === 'payment') {
        const amount = decimalInput(form.amount)
        submission.current ||= crypto.randomUUID()
        const payment = {
          payment_date: form.date,
          note: form.note,
          idempotency_key: submission.current,
        }
        if (credit) await api.creditPayment(row.id, { ...payment, amount })
        else await api.patchTax(row.id, { ...payment, payment_amount: amount })
      } else if (credit) {
        const payload = creditPayload(form)
        await (mode === 'edit'
          ? api.updateCredit(row.id, payload)
          : api.createCredit(payload))
      } else {
        const payload = {
          name: form.name.trim(),
          amount: decimalInput(form.amount),
          due_date: form.due,
        }
        await (mode === 'edit'
          ? api.patchTax(row.id, payload)
          : api.createTax(payload))
      }
      onSaved()
    } catch (e) {
      setError(e.message)
    } finally {
      lock.current = false
      setBusy(false)
    }
  }
  const plain = ['delete', 'close', 'reopen', 'late'].includes(mode)
  return (
    <dialog
      ref={ref}
      className="credit-dialog credits-page"
      aria-labelledby="credit-dialog-title"
      onCancel={(e) => {
        e.preventDefault()
        close()
      }}
    >
      <form onSubmit={save} aria-busy={busy}>
        <h2 id="credit-dialog-title">{title}</h2>
        {row && <p>{row.name}</p>}
        {mode === 'delete' && (
          <p>
            L’engagement sera retiré de la synthèse. Ses paiements et son
            historique seront conservés.
          </p>
        )}
        {mode === 'close' && (
          <p>
            La clôture confirme un solde déclaré nul. Elle ne crée aucun
            paiement.
          </p>
        )}
        {mode === 'reopen' && (
          <p>L’historique et les paiements seront conservés.</p>
        )}
        {mode === 'late' && (
          <p>
            {row.overdue_confirmed
              ? 'Retirer cette confirmation si le retard n’est plus établi. L’historique sera conservé.'
              : 'Confirmer uniquement si vous avez vérifié que ce paiement est réellement en retard.'}
          </p>
        )}
        {mode === 'reset' && (
          <p>
            Correction du total payé déclaré, conservée dans l’historique. Elle
            ne modifie aucune transaction bancaire.
          </p>
        )}
        {!plain && (
          <div className="form-row">
            {!['payment', 'reset'].includes(mode) && (
              <Field
                label="Nom"
                name="name"
                form={form}
                setForm={changeForm}
                required
                maxLength={200}
              />
            )}
            {credit && !['payment', 'reset'].includes(mode) ? (
              <>
                <Field
                  label="Mensualité (€)"
                  name="monthly"
                  inputMode="decimal"
                  form={form}
                  setForm={changeForm}
                  required
                />
                <Field
                  label="Nombre de mois"
                  name="months"
                  type="number"
                  min="1"
                  max="1200"
                  step="1"
                  form={form}
                  setForm={changeForm}
                  required
                />
                <Field
                  label="Première échéance"
                  name="start"
                  type="month"
                  form={form}
                  setForm={changeForm}
                  required
                />
                <Field
                  label="Déjà remboursé avant les paiements enregistrés ici (€)"
                  name="opening"
                  inputMode="decimal"
                  placeholder="Inconnu"
                  form={form}
                  setForm={changeForm}
                />
                <p className="muted form-help">
                  Laisser vide si l’historique est inconnu. Saisir 0 uniquement
                  si aucun paiement antérieur n’est à prendre en compte. Le
                  total nominal est la mensualité multipliée par la durée, sans
                  calcul d’intérêts bancaires.
                </p>
              </>
            ) : (
              <Field
                label={
                  mode === 'reset'
                    ? 'Total payé corrigé (€)'
                    : mode === 'payment'
                      ? 'Montant du paiement (€)'
                      : 'Montant (€)'
                }
                name="amount"
                inputMode="decimal"
                form={form}
                setForm={changeForm}
                required
              />
            )}
            {mode === 'payment' ? (
              <Field
                label="Date du paiement"
                name="date"
                type="date"
                max={new Date().toISOString().slice(0, 10)}
                form={form}
                setForm={changeForm}
                required
              />
            ) : (
              !credit &&
              mode !== 'reset' && (
                <Field
                  label="Date d’échéance"
                  name="due"
                  type="date"
                  form={form}
                  setForm={changeForm}
                  required
                />
              )
            )}
            {(credit || mode === 'payment' || mode === 'reset') && (
              <Field
                label={
                  mode === 'reset' ? 'Motif de correction' : 'Note (facultatif)'
                }
                name="note"
                form={form}
                setForm={changeForm}
                maxLength={2000}
                required={mode === 'reset'}
              />
            )}
          </div>
        )}
        {mode === 'payment' && (
          <p className="muted">
            Déclaration manuelle, sans création de transaction bancaire.
            Vérifiez les paiements déjà enregistrés avant de valider.
          </p>
        )}
        {error && (
          <p className="notice err" role="alert">
            {error}
          </p>
        )}
        <div className="form-actions">
          <button className="btn" type="submit" disabled={busy}>
            {busy ? 'Enregistrement…' : plain ? 'Confirmer' : 'Enregistrer'}
          </button>
          <button
            type="button"
            className="btn-secondary"
            disabled={busy}
            onClick={close}
          >
            Annuler
          </button>
        </div>
      </form>
    </dialog>
  )
}
const events = {
  created: 'Création',
  updated: 'Modification',
  archived: 'Retrait de la synthèse',
  payment: 'Paiement déclaré',
  legacy_status: 'Ancienne confirmation de statut',
  payment_voided: 'Paiement annulé',
}
export function CreditHistory({ kind, row, onClose, onSaved }) {
  const ref = useRef(null),
    trigger = useRef(document.activeElement),
    [data, setData] = useState(null),
    [error, setError] = useState(''),
    [offset, setOffset] = useState(0),
    [reload, setReload] = useState(0),
    [busy, setBusy] = useState(false),
    lock = useRef(false)
  const [voiding, setVoiding] = useState(null),
    [note, setNote] = useState('')
  useEffect(() => {
    ref.current.showModal()
    return () => {
      ref.current?.close()
      ;(trigger.current?.isConnected
        ? trigger.current
        : document.querySelector('.credits-context h2')
      )?.focus()
    }
  }, [])
  useEffect(() => {
    const abort = new AbortController()
    setData(null)
    setError('')
    api
      .engagementHistory(kind, row.id, offset, { signal: abort.signal })
      .then(setData)
      .catch((e) => {
        if (e.name !== 'AbortError') setError(e.message)
      })
    return () => abort.abort()
  }, [kind, row.id, offset, reload])
  async function voidPayment(e) {
    e.preventDefault()
    if (lock.current) return
    lock.current = true
    setBusy(true)
    setError('')
    try {
      await api.voidCreditPayment(row.id, voiding.id, note)
      setVoiding(null)
      setNote('')
      setReload((n) => n + 1)
      onSaved()
    } catch (e) {
      setError(e.message)
    } finally {
      lock.current = false
      setBusy(false)
    }
  }
  return (
    <dialog
      ref={ref}
      className="credit-dialog credits-page"
      aria-labelledby="credit-history-title"
      onCancel={(e) => {
        e.preventDefault()
        if (!lock.current) onClose()
      }}
    >
      <h2 id="credit-history-title">Historique — {row.name}</h2>
      <p className="muted">
        Déclarations utilisateur. Aucun rapprochement bancaire automatique. Les
        anciennes modifications ne disposent pas toujours d’un détail
        avant/après.
      </p>
      {error && (
        <p role="alert" className="notice err">
          {error}
        </p>
      )}
      {!data && !error && <p role="status">Chargement de l’historique…</p>}
      {data && (
        <>
          {kind === 'credits' && (
            <section>
              <h3>Paiements enregistrés</h3>
              {!data.payments.length && <p>Aucun paiement sur cette page.</p>}
              <ul className="engagement-history">
                {data.payments.map((p) => (
                  <li key={p.id}>
                    <strong>{formatEuro(p.amount_cents)}</strong> ·{' '}
                    {formatDate(p.payment_date)} ·{' '}
                    {p.voided_at ? `Annulé : ${p.void_note}` : 'Déclaré'}
                    {p.note && <p>{p.note}</p>}
                    {!p.voided_at && !row.closed_at && !row.archived_at && (
                      <button
                        type="button"
                        className="link"
                        disabled={busy}
                        onClick={() => {
                          setVoiding(p)
                          setNote('')
                        }}
                      >
                        Annuler ce paiement
                      </button>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          )}
          {voiding && (
            <form onSubmit={voidPayment}>
              <p>
                Annuler le paiement de {formatEuro(voiding.amount_cents)} ? Son
                écriture sera conservée dans l’historique.
              </p>
              <label className="form-field">
                Motif d’annulation
                <input
                  className="form-input"
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  required
                  maxLength={2000}
                />
              </label>
              <div className="form-actions">
                <button className="btn" disabled={busy}>
                  Confirmer l’annulation
                </button>
                <button
                  className="btn-secondary"
                  type="button"
                  disabled={busy}
                  onClick={() => setVoiding(null)}
                >
                  Conserver le paiement
                </button>
              </div>
            </form>
          )}
          <section>
            <h3>Modifications de l’engagement</h3>
            {!data.changes.length && <p>Aucune modification sur cette page.</p>}
            <ul className="engagement-history">
              {data.changes.map((c) => (
                <li key={c.id}>
                  <strong>{events[c.event_type] || 'Modification'}</strong> ·{' '}
                  {formatDate(c.changed_at)}
                  {c.payment_amount_cents != null && (
                    <p>
                      {formatEuro(c.payment_amount_cents)} · paiement du{' '}
                      {formatDate(c.payment_date)}
                    </p>
                  )}
                  {c.note && <p>{c.note}</p>}
                  {c.after_state && (
                    <details>
                      <summary>Valeurs avant et après</summary>
                      <dl>
                        {(kind === 'credits'
                          ? [
                              ['Mensualité', 'monthly_amount'],
                              ['Total nominal', 'total_scheduled'],
                              ['Déjà remboursé', 'opening_paid'],
                              ['Total payé', 'paid_total'],
                              ['Solde', 'remaining'],
                            ]
                          : [
                              ['Montant', 'amount'],
                              ['Montant payé', 'paid_amount'],
                              ['Solde', 'remaining'],
                            ]
                        ).map(([label, field]) => (
                          <div key={field}>
                            <dt>{label}</dt>
                            <dd>
                              {c.before_state
                                ? moneyText(c.before_state, field)
                                : '—'}{' '}
                              → {moneyText(c.after_state, field)}
                            </dd>
                          </div>
                        ))}
                        <div>
                          <dt>Nom</dt>
                          <dd>
                            {c.before_state?.name || '—'} → {c.after_state.name}
                          </dd>
                        </div>
                        <div>
                          <dt>
                            {kind === 'credits' ? 'Échéancier' : 'Échéance'}
                          </dt>
                          <dd>
                            {kind === 'credits'
                              ? `${c.before_state?.months_total ?? '—'} → ${c.after_state.months_total} mois · début ${c.after_state.start_month}`
                              : `${c.before_state?.due_date || '—'} → ${c.after_state.due_date}`}
                          </dd>
                        </div>
                      </dl>
                    </details>
                  )}
                </li>
              ))}
            </ul>
          </section>
          <div className="form-actions">
            <button
              type="button"
              className="btn-secondary"
              disabled={!offset || busy}
              onClick={() => setOffset((n) => n - 50)}
            >
              Page précédente
            </button>
            <span>Page {offset / 50 + 1}</span>
            <button
              type="button"
              className="btn-secondary"
              disabled={busy || !(data.has_more || data.payments_has_more)}
              onClick={() => setOffset((n) => n + 50)}
            >
              Page suivante
            </button>
          </div>
        </>
      )}
      <div className="form-actions">
        <button
          type="button"
          className="btn-secondary"
          disabled={busy}
          onClick={onClose}
        >
          Fermer
        </button>
        {error && (
          <button
            type="button"
            className="btn-secondary"
            onClick={() => setReload((n) => n + 1)}
          >
            Réessayer
          </button>
        )}
      </div>
    </dialog>
  )
}

// Retired records remain discoverable without loading them with the summary.
export function ArchivedEngagements({ onHistory }) {
  const [open, setOpen] = useState(false),
    [rows, setRows] = useState(null),
    [error, setError] = useState(''),
    [retry, setRetry] = useState(0)
  useEffect(() => {
    if (!open) return
    const abort = new AbortController()
    setRows(null)
    setError('')
    Promise.all([
      api.credits({ include_archived: 1 }, { signal: abort.signal }),
      api.taxes({ include_archived: 1 }, { signal: abort.signal }),
    ])
      .then(([credits, taxes]) => {
        if (!Array.isArray(credits) || !Array.isArray(taxes))
          throw new Error('Archives indisponibles.')
        if (!abort.signal.aborted)
          setRows([
            ...credits
              .filter((c) => c.archived_at)
              .map((row) => ({ kind: 'credits', row })),
            ...taxes
              .filter((t) => t.archived_at)
              .map((row) => ({ kind: 'taxes', row })),
          ])
      })
      .catch((e) => {
        if (e.name !== 'AbortError') setError(e.message)
      })
    return () => abort.abort()
  }, [open, retry])
  return (
    <details
      className="engagement-archives"
      onToggle={(e) => setOpen(e.currentTarget.open)}
    >
      <summary>Engagements retirés et historiques</summary>
      {open && (
        <>
          <p className="muted">
            Exclus de la synthèse. Les paiements et modifications restent
            consultables.
          </p>
          {error ? (
            <p role="alert">
              {error}{' '}
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setRetry((n) => n + 1)}
              >
                Réessayer
              </button>
            </p>
          ) : !rows ? (
            <p role="status">Chargement des archives…</p>
          ) : !rows.length ? (
            <p>Aucun engagement retiré.</p>
          ) : (
            <ul className="engagement-history">
              {rows.map(({ kind, row }) => (
                <li key={`${kind}-${row.id}`}>
                  <strong>{row.name}</strong> ·{' '}
                  {kind === 'credits' ? 'Crédit' : 'Échéance'} · retiré le{' '}
                  {formatDate(row.archived_at)}{' '}
                  <button
                    type="button"
                    className="btn-secondary"
                    aria-label={`Historique archivé de ${row.name}`}
                    onClick={() => onHistory({ kind, row })}
                  >
                    Historique
                  </button>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </details>
  )
}
