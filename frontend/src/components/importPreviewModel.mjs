export function previewPage(transactions, filter, page, pageSize = 50) {
  const filtered = transactions.filter(t => filter === 'new' ? !t.duplicate : filter === 'duplicates' ? t.duplicate : true)
  const pages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const currentPage = Math.max(0, Math.min(page, pages - 1))
  return { rows: filtered.slice(currentPage * pageSize, (currentPage + 1) * pageSize), pages, currentPage, total: filtered.length }
}

export const previewExpired = (expiresAt, now = Date.now()) => new Date(expiresAt).getTime() <= now
