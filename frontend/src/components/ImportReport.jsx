import { eur } from '../api'
import { importControlLabel, importIssueLabel } from './importReportModel.mjs'
import './ImportReport.css'

const money = cents => cents == null ? 'Non disponible' : eur(cents / 100)
export default function ImportReport({ report }) {
  if (!report) return <p>Rapport détaillé non disponible pour cet ancien import. L’équilibre des soldes seul ne prouve pas la complétude.</p>
  const problems = (report.candidates || []).filter(c => ['uninterpretable', 'rejected'].includes(c.status))
  return <section className="import-report" aria-label="Rapport de fidélité de l’import">
    <h3>Contrôles de l’import</h3>
    {report.extraction && <p>Lecture : {report.extraction === "poppler_layout" ? "texte PDF et colonnes (Poppler)" : report.extraction === "ocr" ? "reconnaissance OCR, à comparer à l’original" : "texte PDF natif"}. Les références de ligne correspondent au texte extrait.</p>}
    <dl>
      <div><dt>Opérations candidates</dt><dd>{report.detected}</dd></div>
      <div><dt>Opérations interprétées</dt><dd>{report.parsed}</dd></div>
      <div><dt>Enregistrées lors de cet import</dt><dd>{report.inserted}</dd></div>
      <div><dt>Non interprétées / rejetées</dt><dd>{report.rejected}</dd></div>
      <div><dt>Débits calculés / déclarés</dt><dd>{money(report.debits_cents)} / {money(report.declared_debits_cents)}</dd></div>
      <div><dt>Crédits calculés / déclarés</dt><dd>{money(report.credits_cents)} / {money(report.declared_credits_cents)}</dd></div>
      <div><dt>Écart de rapprochement</dt><dd>{money(report.balance_difference_cents)}</dd></div>
      <div><dt>Nombre attendu déclaré</dt><dd>{report.expected_count ?? 'Non fourni par le document'}</dd></div>
    </dl>
    <ul>
      <li>Nombre d’opérations : {importControlLabel(report.count_control)}</li>
      <li>Totaux débits et crédits indépendants : {importControlLabel(report.totals_control)}</li>
      <li>Équilibre comptable : {importControlLabel(report.balance_control)}</li>
      {report.traceability_control && <li>Traçabilité des opérations : {importControlLabel(report.traceability_control)}</li>}
    </ul>
    {report.completeness === 'unknown' && <p>Complétude non établie : les contrôles disponibles ne prouvent pas à eux seuls l’absence d’opérations manquantes.</p>}
    {report.completeness === 'failed' && <p role="alert">Import non validable : au moins un contrôle a échoué ou une lecture reste ambiguë.</p>}
    {report.requires_review && <p>Une comparaison avec l’original est requise avant validation.</p>}
    {!!report.warnings?.length && <ul>{report.warnings.map(warning => <li key={warning}>{importIssueLabel(warning)}</li>)}</ul>}
    {!!report.issues?.length && <ul>{report.issues.map(issue => <li key={issue}>{importIssueLabel(issue)}</li>)}</ul>}
    {!!problems.length && <details open><summary>{problems.length} opération(s) à examiner dans l’original</summary><ul>
      {problems.map(c => <li key={c.index}>Page {c.page}, ligne {c.line} : {importIssueLabel(c.reason)}</li>)}
    </ul></details>}
    {!problems.length && !!report.candidates?.length && <details><summary>Références des {report.candidates.length} opérations candidates</summary><ul tabIndex={0} aria-label="Références des opérations extraites">
      {report.candidates.map(c => <li key={c.index}>Opération {c.index + 1} · page {c.page}, ligne {c.line} · {c.status === 'imported' ? 'Enregistrée' : c.status === 'duplicate' ? 'Document identique déjà enregistré' : 'Interprétée'}</li>)}
    </ul></details>}
  </section>
}
