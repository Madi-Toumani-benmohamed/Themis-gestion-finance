// Exact cents at the API/presentation boundary; never round financial inputs.
export function cents(value) {
  const text = String(value)
  if (!/^-?\d+(?:\.\d{1,2})?$/.test(text))
    throw new Error('Montant invalide ou de précision incohérente.')
  const negative = text.startsWith('-')
  const [units, fraction = ''] = text.replace('-', '').split('.')
  const amount =
    Number(BigInt(units) * 100n + BigInt((fraction + '00').slice(0, 2))) *
    (negative ? -1 : 1)
  if (!Number.isSafeInteger(amount)) throw new Error('Montant hors limites.')
  return amount
}
export const euroCents = (value) =>
  ((value === 0 ? 0 : value) / 100).toLocaleString('fr-FR', {
    style: 'currency',
    currency: 'EUR',
  })
