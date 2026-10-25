import { useEffect, useId, useRef, useState } from 'react'
import { eur } from '../api'
import { previewExpired, previewPage } from './importPreviewModel.mjs'
import './ImportPreview.css'
import ImportReport from './ImportReport'

const dateLabel = value => value ? new Date(`${value.slice(0, 10)}T12:00:00`).toLocaleDateString('fr-FR') : 'Non déterminée'
const amountLabel = value => value == null ? 'Non disponible' : eur(value)

export default function ImportPreview({ preview, onDecision }) {
  const dialog = useRef(null)
  const titleId = useId()
  const helpId = useId()
  const filterId = useId()
  const [filter, setFilter] = useState('all')
  const [page, setPage] = useState(0)
  const [expired, setExpired] = useState(() => previewExpired(preview.expires_at))
  useEffect(() => {
    dialog.current?.showModal()
    const timeout = setTimeout(() => setExpired(true), Math.max(0, new Date(preview.expires_at).getTime() - Date.now()))
    return () => clearTimeout(timeout)
  }, [preview.expires_at])
  const decide = accepted => { dialog.current?.close(); onDecision(accepted) }
  const view = previewPage(preview.transactions, filter, page)

  return (
    <dialog ref={dialog} className="import-preview-dialog" aria-labelledby={titleId} aria-describedby={helpId}
      onCancel={event => { event.preventDefault(); decide(false) }}>
      <header className="import-preview-header">
        <div>
          <h2 id={titleId}>Vérifier avant d’importer</h2>
          <p className="import-preview-filename">{preview.filename} → <strong>{preview.account_name}</strong></p>
        </div>
        <button className="btn-secondary" onClick={() => decide(false)} autoFocus>Annuler ce fichier</button>
      </header>
      <div className="import-preview-body">
        <p id={helpId}>Aucun relevé ni transaction n’a encore été enregistré. Vérifie les éléments extraits avant de confirmer.</p>
        <dl className="import-preview-metrics">
          <div><dt>Opérations détectées</dt><dd>{preview.transactions.length}</dd></div>
          <div><dt>Nouvelles à cet instant</dt><dd>{preview.new_count}</dd></div>
          <div><dt>Doublons confirmés</dt><dd>{preview.duplicates}</dd></div>
        </dl>
        <div className="import-preview-checks">
          <div>
            <h3>{preview.parser === 'csv' ? 'Dates des opérations détectées' : 'Période extraite du relevé'}</h3>
            <p>{dateLabel(preview.period_start)} → {dateLabel(preview.period_end)}</p>
            <p>Lecture : {preview.parser === 'csv' ? 'CSV' : preview.parser === 'local' ? 'parseur local' : 'IA'}</p>
          </div>
          <div>
            <h3>{preview.validated ? 'Équilibre comptable validé' : 'Équilibre comptable à vérifier'}</h3>
            <dl className="import-preview-balances">
              <div><dt>Solde initial</dt><dd>{amountLabel(preview.opening_balance)}</dd></div>
              <div><dt>Solde final</dt><dd>{amountLabel(preview.closing_balance)}</dd></div>
              <div><dt>Écart : final − (initial + opérations)</dt><dd>{amountLabel(preview.balance_difference)}</dd></div>
            </dl>
            {preview.balance_difference == null && <p>Les soldes de contrôle ne sont pas tous disponibles. Un CSV ne permet généralement pas de valider cet équilibre.</p>}
          </div>
        </div>
        {preview.blocked_reason && <p className="notice err" role="alert">{preview.blocked_reason}</p>}
        <ImportReport report={preview.import_report} />
        <div className="import-preview-table-heading">
          <h3>Opérations extraites</h3>
          <label htmlFor={filterId}>Afficher</label>
          <select id={filterId} value={filter} onChange={event => { setFilter(event.target.value); setPage(0) }}>
            <option value="all">Toutes</option><option value="new">Nouvelles</option><option value="duplicates">Doublons</option>
          </select>
        </div>
        <div className="import-preview-table-scroll" tabIndex={0} role="region" aria-label="Opérations extraites">
          <table>
            <thead><tr><th>Date</th><th>Libellé</th><th className="num">Montant</th><th>Traitement prévu</th></tr></thead>
            <tbody>
              {view.rows.map((t, index) => (
                <tr key={`${view.currentPage}-${index}`}>
                  <td>{dateLabel(t.date)}</td><td>{t.label}</td><td className="num amt">{eur(t.amount)}</td>
                  <td>{t.duplicate ? 'Déjà enregistré · même document' : 'À importer'}</td>
                </tr>
              ))}
              {view.total === 0 && <tr><td colSpan={4}>Aucune opération dans ce filtre.</td></tr>}
            </tbody>
          </table>
        </div>
        <nav className="import-preview-pagination" aria-label="Pages des opérations extraites">
          <button className="btn-secondary" onClick={() => setPage(view.currentPage - 1)} disabled={view.currentPage === 0}>Précédente</button>
          <span aria-live="polite">Page {view.currentPage + 1} / {view.pages} · {view.total} opérations</span>
          <button className="btn-secondary" onClick={() => setPage(view.currentPage + 1)} disabled={view.currentPage + 1 >= view.pages}>Suivante</button>
        </nav>
        <p className="import-preview-note">Compare cet aperçu au document original et consulte les contrôles non exécutés. Deux lignes identiques d’un même document sont conservées comme occurrences distinctes. Seule la réimportation du même document avec le même résultat est reconnue comme déjà enregistrée.</p>
        {preview.potential_duplicates > 0 && <p className="import-preview-note">{preview.potential_duplicates} opération(s) ressemblent à des opérations d’un autre import. Cette similarité est informative et ne supprime ni ne bloque aucune occurrence.</p>}
        <p className="import-preview-note">Les règles et la catégorisation automatique s’appliqueront après confirmation. L’analyse du fichier a pu utiliser le fournisseur IA configuré. L’aperçu expire à {new Date(preview.expires_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })} et disparaît au redémarrage du serveur.</p>
      </div>
      <footer className="import-preview-footer">
        {expired ? <p role="alert">Cet aperçu a expiré. Annule et sélectionne à nouveau le fichier.</p> : <p>Confirmer enregistre les opérations nouvelles dans <strong>{preview.account_name}</strong>.</p>}
        <button className="btn" onClick={() => decide(true)} disabled={expired || !!preview.blocked_reason}>
          {preview.blocked_reason ? 'Import suspendu — vérification nécessaire' : preview.new_count > 0 ? `${preview.validated ? 'Confirmer l’import' : 'Enregistrer comme provisoire'} (${preview.new_count} nouvelles)` : 'Terminer — aucune nouvelle opération'}
        </button>
      </footer>
    </dialog>
  )
}
