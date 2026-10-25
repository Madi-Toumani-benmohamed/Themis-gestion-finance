export const AI_PROVIDERS = [
  { id: 'none', label: 'Désactivé' },
  { id: 'ollama', label: 'Ollama', model: 'ollama_model', url: 'ollama_url' },
  { id: 'openai', label: 'OpenAI', model: 'openai_model', key: 'openai_key' },
  {
    id: 'anthropic',
    label: 'Anthropic / Claude',
    model: 'anthropic_model',
    key: 'anthropic_key',
  },
  {
    id: 'mistral',
    label: 'Mistral AI',
    model: 'mistral_model',
    key: 'mistral_key',
  },
  {
    id: 'freellm',
    label: 'FreeLLM / OpenRouter',
    model: 'freellm_model',
    url: 'freellm_url',
    key: 'freellm_key',
  },
]
export const AI_CONTEXT_LIMIT = 1000
export const aiProvider = (id) =>
  AI_PROVIDERS.find((p) => p.id === id) || {
    id: 'unknown',
    label: 'Fournisseur inconnu',
  }
const textFields = [
  'ollama_url',
  'ollama_model',
  'openai_model',
  'anthropic_model',
  'mistral_model',
  'freellm_model',
  'freellm_url',
  'context',
]
const keyFields = AI_PROVIDERS.flatMap((p) => (p.key ? [p.key] : []))
export function aiForm(config) {
  return {
    provider: config.provider || 'none',
    ...Object.fromEntries(textFields.map((k) => [k, config[k] || ''])),
    ...Object.fromEntries(keyFields.map((k) => [k, ''])),
    auto_categorize: Boolean(config.auto_categorize),
    clear_keys: [],
  }
}
export function aiFormDirty(form, config) {
  const initial = aiForm(config)
  return Object.keys(initial).some(
    (k) => JSON.stringify(form[k]) !== JSON.stringify(initial[k]),
  )
}
export function aiConnectionDirty(form, config) {
  if (form.provider !== config.provider) return true
  const provider = aiProvider(form.provider)
  return (
    [provider.model, provider.url]
      .filter(Boolean)
      .some((k) => form[k].trim() !== String(config[k] || '').trim()) ||
    Boolean(
      provider.key &&
        (form[provider.key].trim() || form.clear_keys.includes(provider.key)),
    )
  )
}
export function aiPayload(form, config, connectionOnly = false) {
  if (!AI_PROVIDERS.some((p) => p.id === form.provider))
    throw new Error('Fournisseur inconnu.')
  if (!connectionOnly && Array.from(form.context).length > AI_CONTEXT_LIMIT)
    throw new Error('Le contexte est limité à 1 000 caractères.')
  const provider = aiProvider(form.provider)
  if (form.provider !== 'none') {
    if (!form[provider.model]?.trim())
      throw new Error('Renseigner le modèle à utiliser.')
    if (provider.url) {
      let url
      try {
        url = new URL(form[provider.url].trim())
      } catch {
        throw new Error('Renseigner une URL de serveur valide.')
      }
      if (
        !['http:', 'https:'].includes(url.protocol) ||
        url.username ||
        url.password ||
        url.search ||
        url.hash
      )
        throw new Error(
          'L’URL doit être HTTP(S), sans clé, identifiants ni paramètres.',
        )
    }
  }
  if (connectionOnly) {
    return {
      provider: form.provider,
      ...Object.fromEntries(
        [provider.model, provider.url, provider.key]
          .filter((k) => k && form[k].trim())
          .map((k) => [k, form[k].trim()]),
      ),
      ...(provider.key && form.clear_keys.includes(provider.key)
        ? { clear_keys: [provider.key] }
        : {}),
    }
  }
  const payload = {
    provider: form.provider,
    auto_categorize: form.auto_categorize,
    context: form.context.trim(),
    ...Object.fromEntries(
      textFields.filter((k) => k !== 'context').map((k) => [k, form[k].trim()]),
    ),
    ...Object.fromEntries(
      keyFields.filter((k) => form[k].trim()).map((k) => [k, form[k].trim()]),
    ),
  }
  if (form.clear_keys.length) payload.clear_keys = [...form.clear_keys]
  if (config.revision) payload.expected_revision = config.revision
  return payload
}
export const aiStatusLabel = (status) =>
  ({
    available: 'Disponible',
    unavailable: 'Indisponible',
    not_tested: 'Non testé',
    not_configured: 'Non configuré',
    disabled: 'Désactivée',
  })[status] || 'État inconnu'
export const aiCheckedAt = (value) => {
  const date = value ? new Date(value) : null
  return date && Number.isFinite(date.getTime())
    ? date.toLocaleString('fr-FR')
    : 'Aucun contrôle effectué'
}
