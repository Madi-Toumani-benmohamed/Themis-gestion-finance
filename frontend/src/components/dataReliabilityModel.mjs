export function reliabilitySummary(report) {
  const accounts = (report.accounts ?? []).filter(a => a.tracking_mode !== 'manual')
  return {
    total: accounts.length,
    documented: accounts.filter(a => a.status === 'documented').length,
    toReview: accounts.reduce((sum, a) => sum + a.to_review, 0),
    unknownPeriods: accounts.reduce((sum, a) => sum + a.unknown_periods, 0),
  }
}

export const reliabilityStatusLabel = (account, current) => ({
  documented: current ? 'Couverture documentée à date' : 'Couverture documentée',
  to_review: 'Couverture à vérifier',
  no_statement: 'Aucun relevé pour cette période',
  insufficient: 'Couverture insuffisante',
}[account.status] || 'Couverture indéterminée')

export const reliabilityDate = value => value
  ? new Date(`${value.slice(0, 10)}T12:00:00`).toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
  : 'Indisponible'
