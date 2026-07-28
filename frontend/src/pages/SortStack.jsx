import { useEffect, useState, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { api, eur, kindIcon } from '../api'
import { useToast } from '../Toast'

// SortStack — le « tri en pile » : les transactions sans catégorie sont présentées
// une par une, en grand, avec les domaines en gros boutons (clic ou touches 1-9).
// Objectif : transformer « scanner 42 lignes » en « traiter quelques cartes ».
export default function SortStack({ transactions, domains, onClose, onSorted }) {
  const toast = useToast()
  const [queue, setQueue] = useState(transactions)
  const [idx, setIdx] = useState(0)
  const [done, setDone] = useState(0)
  const [leaving, setLeaving] = useState(null) // 'sorted' | 'skipped' pour l'animation

  const topDomains = domains.filter((d) => !d.parent_id)
  const tx = queue[idx]

  const advance = useCallback(() => {
    setLeaving(null)
    if (idx + 1 >= queue.length) {
      setIdx(queue.length) // écran de fin
    } else {
      setIdx((i) => i + 1)
    }
  }, [idx, queue.length])

  const assign = useCallback(async (domainId) => {
    if (!tx || leaving) return
    setLeaving('sorted')
    try {
      await api.setTransactionMeta(tx.id, { domain_id: domainId })
      setDone((d) => d + 1)
      onSorted?.(tx.id, domainId)
    } catch (e) {
      toast.err(e.message)
    }
    setTimeout(advance, 220)
  }, [tx, leaving, advance, onSorted, toast])

  const skip = useCallback(() => {
    if (!tx || leaving) return
    setLeaving('skipped')
    setTimeout(advance, 220)
  }, [tx, leaving, advance])

  // Raccourcis clavier : 1-9 → domaine, Échap → fermer, → / espace → passer
  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape') { onClose(); return }
      if (!tx) return
      if (e.key === 'ArrowRight' || e.key === ' ') { e.preventDefault(); skip(); return }
      const n = parseInt(e.key, 10)
      if (!isNaN(n) && n >= 1 && n <= topDomains.length) {
        assign(topDomains[n - 1].id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [tx, topDomains, assign, skip, onClose])

  const total = queue.length
  const finished = idx >= total

  return createPortal(
    <div className="stack-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div className="stack-modal">
        <div className="stack-head">
          <span className="stack-progress">
            {finished ? `${done} rangée${done > 1 ? 's' : ''}` : `${idx + 1} / ${total}`}
          </span>
          <button className="link" onClick={onClose}>fermer (Échap)</button>
        </div>

        {finished ? (
          <div className="stack-done">
            <div className="stack-done-icon">✓</div>
            <h2>{done > 0 ? 'Pile traitée !' : 'Rien à ranger'}</h2>
            <p className="muted">
              {done > 0
                ? `${done} transaction${done > 1 ? 's' : ''} rangée${done > 1 ? 's' : ''}.`
                : 'Toutes tes transactions ont déjà une catégorie.'}
            </p>
            <button className="btn" onClick={onClose}>Terminer</button>
          </div>
        ) : (
          <>
            <div className="stack-progress-bar">
              <span style={{ width: `${(idx / total) * 100}%` }} />
            </div>

            <div className={'stack-card' + (leaving ? ` leaving-${leaving}` : '')}>
              <div className="stack-date">{tx.date}</div>
              <div className="stack-label">{kindIcon(tx.kind) && <span className="kicon" aria-hidden="true">{kindIcon(tx.kind)}</span>}{tx.display_label || tx.label}</div>
              {tx.display_label && tx.display_label !== tx.label && (
                <div className="stack-raw" title="Libellé bancaire d'origine">{tx.label}</div>
              )}
              <div className={'stack-amount ' + (tx.amount >= 0 ? 'credit' : 'debit')}>
                {tx.amount >= 0 ? '+' : ''}{eur(tx.amount)}
              </div>
            </div>

            <p className="stack-question">Dans quelle catégorie ranger cette transaction ?</p>

            <div className="stack-domains">
              {topDomains.map((d, i) => (
                <button
                  key={d.id}
                  className="stack-domain"
                  style={{ '--dc': d.color }}
                  onClick={() => assign(d.id)}
                >
                  <span className="stack-domain-key">{i + 1}</span>
                  <span className="stack-dot" />
                  {d.name}
                </button>
              ))}
            </div>

            <div className="stack-foot">
              <button className="link" onClick={skip}>passer cette transaction (→)</button>
            </div>
          </>
        )}
      </div>
    </div>
  , document.body)
}
