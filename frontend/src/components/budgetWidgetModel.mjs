import { cents } from './money.mjs'

export function budgetWidgetModel(budget) {
  const spent = cents(budget.spent)
  const limit = cents(budget.monthly_limit)
  const ratio = limit > 0 ? spent / limit : 0
  return {
    name: budget.domain_name,
    spent: spent / 100,
    limit: limit / 100,
    percent: Math.round(ratio * 100),
    width: Math.max(0, Math.min(ratio * 100, 100)),
    over: spent > limit,
    difference: Math.abs(limit - spent) / 100,
  }
}
