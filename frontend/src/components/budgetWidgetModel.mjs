export function budgetWidgetModel(budget) {
  return {
    name: budget.domain_name,
    spent: budget.spent,
    limit: budget.monthly_limit,
    percent: Math.round(budget.ratio * 100),
    width: Math.min(budget.ratio * 100, 100),
    over: budget.ratio > 1,
    difference: Math.abs(budget.monthly_limit - budget.spent),
  }
}
