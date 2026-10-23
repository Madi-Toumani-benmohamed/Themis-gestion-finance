import { useEffect, useState } from 'react'
import { api } from '../api'
import { useToast } from '../Toast'

export default function Admin() {
  const toast = useToast()
  const [cfg, setCfg] = useState(null)
  const [stats, setStats] = useState(null)
  const [err, setErr] = useState(null)
  const [notice, setNotice] = useState(null)

  // formulaire IA
  const [provider, setProvider] = useState('none')
  const [ollamaUrl, setOllamaUrl] = useState('')
  const [ollamaModel, setOllamaModel] = useState('')
  const [ollamaModels, setOllamaModels] = useState(null)
  const [anthropicModel, setAnthropicModel] = useState('')
  const [anthropicKey, setAnthropicKey] = useState('')
  const [openaiModel, setOpenaiModel] = useState('')
  const [openaiKey, setOpenaiKey] = useState('')
  // Mistral AI
  const [mistralModel, setMistralModel] = useState('')
  const [mistralKey, setMistralKey] = useState('')
  // FreeLLM
  const [freellmModel, setFreellmModel] = useState('')
  const [freellmUrl, setFreellmUrl] = useState('')
  const [freellmKey, setFreellmKey] = useState('')
  const [autoCat, setAutoCat] = useState(true)
  const [aiContext, setAiContext] = useState('')
  const [testResult, setTestResult] = useState(null)
  const [busy, setBusy] = useState(false)

  // maintenance
  const [purgeText, setPurgeText] = useState('')

  // mot de passe personnel
  const [pw, setPw] = useState({ current: '', next: '', confirm: '' })

  // utilisateurs
  const [users, setUsers] = useState([])
  const [uForm, setUForm] = useState({ username: '', password: '', role: 'user' })

  const loadAll = () => {
    api.adminConfig().then((c) => {
      setCfg(c)
      setProvider(c.provider)
      setOllamaUrl(c.ollama_url || '')
      setOllamaModel(c.ollama_model || '')
      setAnthropicModel(c.anthropic_model || '')
      setOpenaiModel(c.openai_model || 'gpt-4.1-mini')
      setMistralModel(c.mistral_model || '')
      setFreellmModel(c.freellm_model || '')
      setFreellmUrl(c.freellm_url || '')
      setAutoCat(c.auto_categorize)
      setAiContext(c.context || '')
    }).catch((e) => setErr(e.message))
    api.adminStats().then(setStats).catch(() => {})
    api.users().then((x) => setUsers(Array.isArray(x) ? x : [])).catch(() => {})
  }
  useEffect(() => { loadAll() }, [])

  const currentForm = () => ({
    provider,
    openai_model: openaiModel.trim(),
    ...(openaiKey.trim() ? { openai_key: openaiKey.trim() } : {}),
    ollama_url: ollamaUrl.trim(),
    ollama_model: ollamaModel.trim(),
    anthropic_model: anthropicModel.trim(),
    ...(anthropicKey.trim() ? { anthropic_key: anthropicKey.trim() } : {}),
    mistral_model: mistralModel.trim(),
    ...(mistralKey.trim() ? { mistral_key: mistralKey.trim() } : {}),
    freellm_model: freellmModel.trim(),
    freellm_url: freellmUrl.trim(),
    ...(freellmKey.trim() ? { freellm_key: freellmKey.trim() } : {}),
    auto_categorize: autoCat,
    context: aiContext.trim(),
  })

  const save = async () => {
    setBusy(true); setErr(null); setNotice(null)
    try {
      const c = await api.adminPutConfig(currentForm())
      setCfg(c)
      setAnthropicKey('')
      setOpenaiKey('')
      toast.ok(`Configuration appliquée — fournisseur actif : ${c.active}`)
    } catch (e) { setErr(e.message) } finally { setBusy(false) }
  }

  const test = async () => {
    setBusy(true); setTestResult(null); setErr(null)
    try {
      setTestResult(await api.adminTestAI(currentForm()))
    } catch (e) { setErr(e.message) } finally { setBusy(false) }
  }

  const fetchModels = async () => {
    setOllamaModels(null); setErr(null)
    try {
      const r = await api.adminOllamaModels(ollamaUrl.trim())
      setOllamaModels(r.models)
      if (r.models.length && !r.models.includes(ollamaModel)) setOllamaModel(r.models[0])
    } catch (e) { setErr(e.message); setOllamaModels([]) }
  }

  const reapplyRules = async () => {
    setBusy(true); setErr(null)
    try {
      const r = await api.adminReapplyRules()
      toast.ok(`${r.categorized} transaction${r.categorized > 1 ? 's' : ''} classée${r.categorized > 1 ? 's' : ''} par les règles`)
      loadAll()
    } catch (e) { setErr(e.message) } finally { setBusy(false) }
  }

  const purge = async () => {
    setBusy(true); setErr(null)
    try {
      await api.adminPurge(purgeText)
      setPurgeText('')
      toast.ok('Toutes les données ont été effacées.')
      loadAll()
    } catch (e) { setErr(e.message) } finally { setBusy(false) }
  }

  const addUser = async (e) => {
    e.preventDefault()
    setErr(null)
    try {
      await api.createUser(uForm.username.trim(), uForm.password, uForm.role)
      setUForm({ username: '', password: '', role: 'user' })
      toast.ok('Utilisateur créé.')
      loadAll()
    } catch (e2) { setErr(e2.message) }
  }
  const removeUser = async (u) => {
    if (!confirm(`Supprimer l'utilisateur « ${u.username} » ?`)) return
    try { await api.deleteUser(u.id); loadAll() } catch (e2) { setErr(e2.message) }
  }

  const changePassword = async (e) => {
    e.preventDefault()
    if (pw.next !== pw.confirm) { toast.err('Les deux mots de passe ne correspondent pas.'); return }
    try {
      await api.changePassword(pw.current, pw.next)
      setPw({ current: '', next: '', confirm: '' })
      toast.ok('Mot de passe modifié. Les autres sessions ont été déconnectées.')
    } catch (e2) { toast.err(e2.message) }
  }

  if (!cfg) return <p style={{ color: 'var(--muted)' }}><span className="spinner" />Chargement…</p>

  return (
    <div className="section">
      {err && <div className="notice err">{err}</div>}
      {notice && <div className="notice ok">{notice}</div>}

      <h2>Fournisseur d'IA</h2>
      <p className="muted" style={{ marginTop: -6 }}>
        Actif : <b>{cfg.active}</b> · le changement s'applique immédiatement, sans redémarrage,
        et survit aux redémarrages (persisté en base).
      </p>

      <div className="admin-block">
        <div className="admin-row">
          <label>Fournisseur</label>
          <div className="radio-row">
            {[['none', 'Désactivé'], ['ollama', 'Ollama (local)'], ['openai', 'OpenAI (cloud)'], ['anthropic', 'API Claude (cloud)'], ['mistral', 'Mistral AI'], ['freellm', 'FreeLLM / OpenRouter']].map(([v, l]) => (
              <label key={v} className="radio">
                <input type="radio" name="provider" checked={provider === v} onChange={() => setProvider(v)} /> {l}
              </label>
            ))}
          </div>
        </div>

        {provider === 'ollama' && (
          <>
            <div className="admin-row">
              <label>URL de l'instance</label>
              <input value={ollamaUrl} onChange={(e) => setOllamaUrl(e.target.value)}
                     placeholder="http://ollama:11434" style={{ width: 280 }} />
              <button className="btn-secondary" onClick={fetchModels} disabled={!ollamaUrl.trim()}>
                Lister les modèles
              </button>
            </div>
            <div className="admin-row">
              <label>Modèle</label>
              {ollamaModels && ollamaModels.length > 0 ? (
                <select value={ollamaModel} onChange={(e) => setOllamaModel(e.target.value)}>
                  {ollamaModels.map((m) => <option key={m} value={m}>{m}</option>)}
                </select>
              ) : (
                <input value={ollamaModel} onChange={(e) => setOllamaModel(e.target.value)}
                       placeholder="qwen2.5:14b" style={{ width: 220 }} />
              )}
              {ollamaModels && ollamaModels.length === 0 && <span className="muted">aucun modèle listé</span>}
            </div>
          </>
        )}

        {provider === 'openai' && (
          <>
            <div className="admin-row">
              <label htmlFor="openai-model">Modèle</label>
              <input id="openai-model" value={openaiModel} onChange={(e) => setOpenaiModel(e.target.value)}
                     placeholder="gpt-4.1-mini" style={{ width: 220 }} />
            </div>
            <div className="admin-row">
              <label htmlFor="openai-key">Clé API</label>
              <input id="openai-key" type="password" value={openaiKey} onChange={(e) => setOpenaiKey(e.target.value)}
                     placeholder={cfg.openai_key_set ? '••••••••  (définie — laisser vide pour conserver)' : 'sk-…'}
                     style={{ width: 320, maxWidth: '100%' }} autoComplete="off" />
            </div>
            <p className="muted" style={{ fontSize: 12 }}>
              API OpenAI — les données nécessaires aux fonctions IA sont transmises à OpenAI.
              Le test de connexion envoie uniquement une requête de test et consomme des jetons API.
            </p>
          </>
        )}

        {provider === 'anthropic' && (
          <>
            <div className="admin-row">
              <label>Modèle</label>
              <input value={anthropicModel} onChange={(e) => setAnthropicModel(e.target.value)}
                     placeholder="claude-sonnet-4-6" style={{ width: 220 }} />
            </div>
            <div className="admin-row">
              <label>Clé API</label>
              <input type="password" value={anthropicKey} onChange={(e) => setAnthropicKey(e.target.value)}
                     placeholder={cfg.anthropic_key_set ? '••••••••  (définie — laisser vide pour conserver)' : 'sk-ant-…'}
                     style={{ width: 320 }} autoComplete="off" />
            </div>
            <p className="muted" style={{ fontSize: 12 }}>
              Rappel : avec l'API cloud, le fallback de parsing et la catégorisation envoient des libellés hors de ta machine.
            </p>
          </>
        )}

        {provider === 'mistral' && (
          <>
            <div className="admin-row">
              <label>Modèle</label>
              <input value={mistralModel} onChange={(e) => setMistralModel(e.target.value)}
                     placeholder="mistral-small-latest" style={{ width: 220 }} />
            </div>
            <div className="admin-row">
              <label>Clé API</label>
              <input type="password" value={mistralKey} onChange={(e) => setMistralKey(e.target.value)}
                     placeholder={cfg.mistral_key_set ? '••••••••  (définie — laisser vide pour conserver)' : 'mistral-…'}
                     style={{ width: 320 }} autoComplete="off" />
            </div>
            <p className="muted" style={{ fontSize: 12 }}>
              API Mistral AI — les données quittent ta machine. Voir https://mistral.ai
            </p>
          </>
        )}

        {provider === 'freellm' && (
          <>
            <div className="admin-row">
              <label>URL de l'API</label>
              <input value={freellmUrl} onChange={(e) => setFreellmUrl(e.target.value)}
                     placeholder="http://freellm.local:3001/v1" style={{ width: 320 }} />
            </div>
            <div className="admin-row">
              <label>Modèle</label>
              <input value={freellmModel} onChange={(e) => setFreellmModel(e.target.value)}
                     placeholder="auto:fast" style={{ width: 220 }} />
              <span className="muted" style={{ fontSize: 12, marginLeft: 8 }}>
                "auto" = chaîne active, "auto:fast" = prioritise la vitesse
              </span>
            </div>
            <div className="admin-row">
              <label>Clé API</label>
              <input type="password" value={freellmKey} onChange={(e) => setFreellmKey(e.target.value)}
                     placeholder={cfg.freellm_key_set ? '••••••••  (définie — laisser vide pour conserver)' : 'clé FreeLLM'}
                     style={{ width: 320 }} autoComplete="off" />
            </div>
            <p className="muted" style={{ fontSize: 12 }}>
              FreeLLM est un service local qui agrège plusieurs modèles via une API compatible OpenAI.
            </p>
          </>
        )}

        {provider !== 'none' && (
          <div className="admin-row">
            <label>Catégorisation auto</label>
            <label className="radio">
              <input type="checkbox" checked={autoCat} onChange={(e) => setAutoCat(e.target.checked)} />
              {' '}classer par IA les transactions sans catégorie à chaque import
            </label>
          </div>
        )}

        {provider !== 'none' && (
          <div className="admin-row admin-row-col">
            <label>Contexte personnel pour l'IA</label>
            <p className="admin-hint">
              Des faits que l'IA ne peut pas deviner depuis les chiffres, et qui rendent ses
              commentaires plus justes. Par exemple : «&nbsp;mon loyer est prélevé tous les 3&nbsp;mois&nbsp;»,
              «&nbsp;le virement à Paul chaque mois est un remboursement, pas une dépense&nbsp;»,
              «&nbsp;je suis étudiant, mes revenus sont irréguliers&nbsp;». Ce texte est ajouté aux
              analyses et synthèses, jamais à l'extraction des relevés. Ce texte est
              filtré : les tentatives de détourner l'IA (fausses «&nbsp;instructions&nbsp;»)
              sont neutralisées automatiquement.
            </p>
            <textarea
              className="ai-context-input"
              rows={5}
              placeholder="Ex. : Mon loyer (950 €) est trimestriel, prélevé en janvier, avril, juillet, octobre.&#10;Le virement mensuel de 200 € à « Sakina » est un remboursement de prêt familial.&#10;Je suis indépendant : mes revenus varient beaucoup d'un mois à l'autre."
              value={aiContext}
              onChange={(e) => setAiContext(e.target.value)}
            />
          </div>
        )}

        <div className="admin-actions">
          <button className="btn-secondary" onClick={test} disabled={busy || provider === 'none'}>Tester la connexion</button>
          <button className="btn" onClick={save} disabled={busy}>Appliquer</button>
          {testResult && (
            <span className={testResult.ok ? 'credit' : 'debit'} style={{ fontSize: 13 }}>
              {testResult.ok ? '✓' : '✗'} {testResult.detail}
            </span>
          )}
        </div>
      </div>

      <h2 style={{ marginTop: 36 }}>Statistiques</h2>
      {stats && (
        <div className="kpi-grid">
          {[
            ['Relevés', stats.statements],
            ['Transactions', stats.transactions],
            ['Classées', stats.transactions ? `${stats.categorized} (${Math.round((stats.categorized / stats.transactions) * 100)} %)` : 0],
            ['Catégories', stats.categories],
            ['Règles', stats.rules],
            ['Analyses IA', stats.analyses],
            ['Base', stats.db_size ?? '—'],
            ['Historique', stats.first_tx ? `${stats.first_tx} → ${stats.last_tx}` : '—'],
          ].map(([label, value]) => (
            <div className="kpi-card" key={label}>
              <div className="label">{label}</div>
              <div className="value" style={{ fontSize: 16 }}>{value}</div>
            </div>
          ))}
        </div>
      )}

      <h2 style={{ marginTop: 36 }}>Utilisateurs</h2>
      <div className="admin-block">
        <table>
          <thead><tr><th>Identifiant</th><th>Rôle</th><th>Créé le</th><th></th></tr></thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id}>
                <td>{u.username}</td>
                <td>{u.role === 'admin' ? <span className="badge">admin</span> : 'utilisateur'}</td>
                <td className="muted">{u.created_at?.slice(0, 10)}</td>
                <td><button className="link" onClick={() => removeUser(u)}>retirer</button></td>
              </tr>
            ))}
          </tbody>
        </table>
        <form className="admin-row" style={{ marginTop: 14 }} onSubmit={addUser}>
          <input placeholder="Identifiant" value={uForm.username}
                 onChange={(e) => setUForm({ ...uForm, username: e.target.value })} required style={{ width: 160 }} />
          <input type="password" placeholder="Mot de passe" value={uForm.password}
                 onChange={(e) => setUForm({ ...uForm, password: e.target.value })} required style={{ width: 180 }} autoComplete="new-password" />
          <select value={uForm.role} onChange={(e) => setUForm({ ...uForm, role: e.target.value })}>
            <option value="user">Utilisateur</option>
            <option value="admin">Administrateur</option>
          </select>
          <button type="submit" className="btn">Créer</button>
        </form>
        <p className="muted" style={{ fontSize: 12 }}>10 caractères minimum, lettres et chiffres.</p>
      </div>

      <h2 style={{ marginTop: 36 }}>Maintenance</h2>
      <div className="admin-block">
        <div className="admin-row">
          <button className="btn-secondary" onClick={reapplyRules} disabled={busy}>
            Réappliquer les règles aux transactions non classées
          </button>
          <span className="muted" style={{ fontSize: 13 }}>utile après avoir ajouté ou modifié des règles</span>
        </div>
      </div>

      <h2 style={{ marginTop: 36 }}>Mon compte</h2>
      <div className="admin-block">
        <form className="admin-row" onSubmit={changePassword} style={{ flexWrap: 'wrap' }}>
          <input type="password" placeholder="Mot de passe actuel" value={pw.current}
                 onChange={(e) => setPw({ ...pw, current: e.target.value })} required autoComplete="current-password" style={{ width: 180 }} />
          <input type="password" placeholder="Nouveau" value={pw.next}
                 onChange={(e) => setPw({ ...pw, next: e.target.value })} required autoComplete="new-password" style={{ width: 160 }} />
          <input type="password" placeholder="Confirmer" value={pw.confirm}
                 onChange={(e) => setPw({ ...pw, confirm: e.target.value })} required autoComplete="new-password" style={{ width: 160 }} />
          <button type="submit" className="btn">Changer le mot de passe</button>
        </form>
        <div className="admin-row" style={{ marginTop: 6 }}>
          <a className="btn-secondary" href="/api/export" download>Exporter toutes mes données (JSON)</a>
          <span className="muted" style={{ fontSize: 13 }}>sauvegarde et portabilité — comptes, transactions, règles, budgets…</span>
        </div>
      </div>

      <div className="admin-block danger-zone">
        <b>Zone dangereuse.</b>
        <p className="muted" style={{ fontSize: 13 }}>
          Efface tous les relevés, transactions et analyses. Catégories, règles, budgets
          et réglages sont conservés. Irréversible — tape <code>TOUT EFFACER</code> pour confirmer.
        </p>
        <div className="admin-row">
          <input value={purgeText} onChange={(e) => setPurgeText(e.target.value)} placeholder="TOUT EFFACER" style={{ width: 180 }} />
          <button className="btn-danger" onClick={purge} disabled={busy || purgeText !== 'TOUT EFFACER'}>
            Purger les données
          </button>
        </div>
      </div>
    </div>
  )
}
