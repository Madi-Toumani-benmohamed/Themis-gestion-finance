export async function agentRequest(path, method = 'GET', body, signal) {
  const response = await fetch('/api/agent/' + path, {
    method,
    credentials: 'same-origin',
    cache: 'no-store',
    signal,
    headers: body === undefined ? {} : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const data =
    response.status === 204 ? null : await response.json().catch(() => ({}))
  if (!response.ok)
    throw new Error(data?.error || `Requête impossible (${response.status})`)
  return data
}
export const agentList = (value) => (Array.isArray(value) ? value : [])
export function agentTimezone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'Europe/Paris'
  } catch {
    return 'Europe/Paris'
  }
}
export function agentSources(messages) {
  const byID = new Map()
  for (const message of messages)
    for (const source of agentList(message?.evidence)) {
      if (!source || typeof source !== 'object') continue
      const candidates =
        source.type === 'transaction'
          ? [source]
          : agentList(source.data?.transactions)
      for (const item of candidates)
        if (Number.isSafeInteger(item.id) && item.id > 0)
          byID.set(item.id, item)
    }
  return [...byID.values()].slice(-100)
}
