export const statementStatus = (status) =>
  ({
    validated: 'Validé',
    to_review: 'À vérifier',
    failed: 'Contrôles en échec',
    analyzed: 'Analysé · non validé',
  })[status] || 'Contrôle non exécuté'
export const statementStatusIcon = (status) =>
  ({ validated: '✓', to_review: '!', failed: '×', analyzed: '○' })[status] ||
  '?'
export const controlStatus = (status) =>
  ({
    passed: 'Vérifié',
    failed: 'Échec confirmé',
    unknown: 'Non établi',
    suspected: 'À examiner',
  })[status] || 'Non exécuté'
export function statementListModel(rows) {
  if (!Array.isArray(rows)) throw new Error('Liste des relevés indisponible.')
  const ids = new Set()
  rows.forEach((st) => {
    if (
      !Number.isSafeInteger(st.id) ||
      ids.has(st.id) ||
      !Array.isArray(st.controls) ||
      !Array.isArray(st.history)
    )
      throw new Error('Contrôles des relevés incomplets.')
    ids.add(st.id)
    for (const key of [
      'opening_balance_cents',
      'closing_balance_cents',
      'theoretical_balance_cents',
      'difference_cents',
    ])
      if (st[key] != null && !Number.isSafeInteger(st[key]))
        throw new Error('Montant de contrôle incohérent.')
    if (!Number.isSafeInteger(st.sum_cents))
      throw new Error('Somme des opérations indisponible.')
    if (
      ![st.credits_cents, st.debits_cents].every(
        (n) => Number.isSafeInteger(n) && n >= 0,
      ) ||
      st.credits_cents - st.debits_cents !== st.sum_cents
    )
      throw new Error(
        'Les crédits et débits ne se réconcilient pas avec la somme des opérations.',
      )
    if (
      st.opening_balance_cents != null &&
      st.theoretical_balance_cents !== st.opening_balance_cents + st.sum_cents
    )
      throw new Error('Le solde théorique transmis est incohérent.')
    if (
      (st.opening_balance_cents == null &&
        st.theoretical_balance_cents != null) ||
      ((st.opening_balance_cents == null || st.closing_balance_cents == null) &&
        st.difference_cents != null)
    )
      throw new Error(
        'Un solde absent ne permet pas de conclure au rapprochement.',
      )
    if (
      st.opening_balance_cents != null &&
      st.closing_balance_cents != null &&
      st.difference_cents !==
        st.closing_balance_cents - st.opening_balance_cents - st.sum_cents
    )
      throw new Error('Le rapprochement ne se réconcilie pas au centime.')
  })
  return rows
}
export function statementSummary(rows) {
  const ends = rows
    .filter(
      (s) => s.effective_validated && s.status === 'validated' && s.period_end,
    )
    .map((s) => s.period_end)
    .sort()
  return {
    total: rows.length,
    review: rows.filter((s) => s.status !== 'validated').length,
    lastValidated: ends.at(-1) || null,
  }
}
export function filterStatements(
  rows,
  {
    account = '',
    status = '',
    search = '',
    month = '',
    ids = null,
    sort = 'period',
  },
) {
  const q = search.trim().toLocaleLowerCase('fr')
  return rows
    .filter(
      (s) =>
        (!account || String(s.account_id) === account) &&
        (!status || s.status === status) &&
        (!q ||
          `${s.filename} ${s.account_name}`
            .toLocaleLowerCase('fr')
            .includes(q)) &&
        (!ids || ids.has(s.id)) &&
        (!month ||
          !s.period_start ||
          !s.period_end ||
          (s.period_start.slice(0, 7) <= month &&
            s.period_end.slice(0, 7) >= month)),
    )
    .sort(
      (a, b) =>
        String(
          sort === 'import' ? b.uploaded_at : b.period_start || '',
        ).localeCompare(
          String(sort === 'import' ? a.uploaded_at : a.period_start || ''),
        ) || b.id - a.id,
    )
}
