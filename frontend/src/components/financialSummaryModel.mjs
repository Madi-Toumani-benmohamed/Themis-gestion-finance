// Presentation only: financial calculations and the diagnosis belong to the API.
export function financialSummaryModel(h) {
  const current = h.period?.is_current === true
  const complete = !current && h.period_complete === true
  const tone = ['positive', 'neutral', 'attention', 'alert'].includes(h.tone) ? h.tone : 'neutral'
  const labels = { positive: 'Situation favorable', neutral: 'Flux enregistrés', attention: 'À surveiller', alert: 'Situation sous tension' }
  const points = (h.points ?? []).filter(p => !['savings', 'deficit'].includes(p.kind))
  const priority = { alert: 0, attention: 1, neutral: 2, positive: 3 }
  return {
    current, complete, tone,
    status: !current && !complete ? 'Données incomplètes' : labels[tone],
    context: current ? 'Mois en cours · Données provisoires' : complete ? 'Mois historique · Couverture documentée' : 'Mois historique · Couverture insuffisante',
    resultLabel: current ? 'Solde des flux à date' : complete ? 'Résultat des flux du mois' : 'Résultat des flux disponibles',
    attention: points.filter(p => ['alert', 'attention'].includes(p.tone)).sort((a, b) => priority[a.tone] - priority[b.tone]),
    observations: points.filter(p => !['alert', 'attention'].includes(p.tone)),
  }
}
