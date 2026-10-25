export const importControlLabel = status => ({ passed: 'Conforme', failed: 'Échec', unverifiable: 'Non vérifiable automatiquement', not_applicable: 'Non applicable (non déclaré)' }[status] || 'Non exécuté')
export const importIssueLabel = code => ({
  ocr_extraction_requires_review: 'Extraction OCR : comparer les dates, libellés et montants au document original', operation_date_outside_period: 'Date d’opération hors période bancaire : vérifier la date de valeur sans correction automatique',
  invalid_amount: 'Montant invalide', ambiguous_ocr_amount: 'Lectures OCR divergentes : vérifier le montant original',
  invalid_declared_count: 'Nombre d’opérations déclaré illisible',
  conflicting_declared_count: 'Plusieurs nombres d’opérations déclarés se contredisent : comparer au document original',
  candidate_count_mismatch: 'Le nombre de références sources est inférieur aux opérations interprétées',
  conflicting_declared_totals: 'Plusieurs totaux déclarés se contredisent : comparer au document original',
  multiple_amounts_on_operation: 'Plusieurs montants possibles pour une opération', unknown_debit_credit_columns: 'Colonnes débit/crédit non identifiées',
  invalid_value_date: 'Date de valeur invalide', invalid_or_unknown_date: 'Date d’opération invalide ou année inconnue',
  missing_amount_or_label: 'Montant ou libellé absent', amount_without_operation_date: 'Montant sans date d’opération identifiable',
  ai_extraction_requires_review: 'Extraction IA non confirmée par une lecture déterministe',
  similar_operations_in_other_import: 'Des opérations similaires existent dans un autre import ; toutes les occurrences sont conservées',
}[code] || 'Lecture non interprétable : comparer au document original')
