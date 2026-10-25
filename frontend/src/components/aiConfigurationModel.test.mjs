import test from 'node:test'
import assert from 'node:assert/strict'
import {
  AI_PROVIDERS,
  aiForm,
  aiFormDirty,
  aiConnectionDirty,
  aiPayload,
  aiStatusLabel,
  aiCheckedAt,
} from './aiConfigurationModel.mjs'
const config = {
  provider: 'ollama',
  ollama_url: 'https://models.example',
  ollama_model: 'unit',
  openai_model: 'unit',
  anthropic_model: 'unit',
  mistral_model: 'unit',
  freellm_model: 'unit',
  freellm_url: 'https://gateway.example/v1',
  auto_categorize: true,
  context: 'Note déclarative',
  revision: 'version-1',
  openai_key_set: true,
}
test('tous les fournisseurs existants sont proposés, sans connecteur supplémentaire', () => {
  assert.deepEqual(
    AI_PROVIDERS.map((p) => p.id),
    ['none', 'ollama', 'openai', 'anthropic', 'mistral', 'freellm'],
  )
})
test('brouillon initial sans clé récupérée, absence de changement et annulation', () => {
  const form = aiForm(config)
  assert.equal(form.openai_key, '')
  assert.equal(aiFormDirty(form, config), false)
  assert.equal(aiFormDirty({ ...form, context: 'Autre note' }, config), true)
  assert.equal(
    aiConnectionDirty({ ...form, context: 'Autre note' }, config),
    false,
  )
  assert.equal(
    aiConnectionDirty({ ...form, ollama_model: 'other' }, config),
    true,
  )
  assert.equal(aiFormDirty(aiForm(config), config), false)
})
test('un test envoie les seuls paramètres de connexion, sans contexte personnel', () => {
  const form = { ...aiForm(config), context: 'PRIVATE CONTEXT' }
  const payload = aiPayload(form, config, true)
  assert.deepEqual(payload, {
    provider: 'ollama',
    ollama_model: 'unit',
    ollama_url: 'https://models.example',
  })
  assert.ok(!JSON.stringify(payload).includes('PRIVATE'))
})
test('sauvegarde partagée : révision et retrait explicite, sans valeurs masquées', () => {
  const form = {
    ...aiForm(config),
    provider: 'openai',
    clear_keys: ['openai_key'],
  }
  const payload = aiPayload(form, config)
  assert.equal(payload.expected_revision, 'version-1')
  assert.deepEqual(payload.clear_keys, ['openai_key'])
  assert.ok(!Object.hasOwn(payload, 'openai_key'))
  assert.ok(!Object.hasOwn(payload, 'openai_key_set'))
  assert.equal(
    aiConnectionDirty({ ...form, provider: 'ollama' }, config),
    false,
  )
  assert.equal(
    aiConnectionDirty(
      { ...aiForm(config), ollama_url: 'https://other.example' },
      config,
    ),
    true,
  )
})
test('les nouvelles clés sont envoyées uniquement lorsqu’elles sont saisies', () => {
  const payload = aiPayload(
    {
      ...aiForm(config),
      provider: 'openai',
      openai_key: ' new-synthetic-key ',
    },
    config,
  )
  assert.equal(payload.openai_key, 'new-synthetic-key')
  assert.ok(!Object.hasOwn(payload, 'anthropic_key'))
})
test('validation de forme et statuts sans métrique de remplacement', () => {
  for (const url of [
    'file:///tmp/file',
    'https://user:secret@models.example',
    'https://models.example?key=secret',
    'bad',
  ])
    assert.throws(() =>
      aiPayload({ ...aiForm(config), ollama_url: url }, config),
    )
  assert.throws(() =>
    aiPayload({ ...aiForm(config), ollama_model: '' }, config),
  )
  assert.throws(() =>
    aiPayload({ ...aiForm(config), provider: 'invented' }, config),
  )
  assert.throws(() =>
    aiPayload({ ...aiForm(config), context: 'é'.repeat(1001) }, config),
  )
  assert.doesNotThrow(() =>
    aiPayload({ ...aiForm(config), context: 'é'.repeat(1000) }, config),
  )
  assert.doesNotThrow(() =>
    aiPayload({ ...aiForm(config), context: 'é'.repeat(1001) }, config, true),
  )
  assert.equal(aiStatusLabel(undefined), 'État inconnu')
  assert.equal(aiStatusLabel('not_tested'), 'Non testé')
  assert.equal(aiCheckedAt(null), 'Aucun contrôle effectué')
})
