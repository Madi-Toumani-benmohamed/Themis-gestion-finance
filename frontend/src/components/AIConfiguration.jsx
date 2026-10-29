import { AnimatedTabs, TabIndicator } from '../motion'
import { useEffect, useId, useRef, useState } from "react";
import { api } from "../api";
import { useToast } from "../Toast";
import {
  AI_PROVIDERS,
  AI_CONTEXT_LIMIT,
  aiProvider,
  aiForm,
  aiFormDirty,
  aiConnectionDirty,
  aiPayload,
  aiStatusLabel,
  aiCheckedAt,
} from "./aiConfigurationModel.mjs";
import "./AIConfiguration.css";

const panels = [
  ["connection", "Fournisseur et connexion"],
  ["behavior", "Comportement"],
  ["diagnostic", "Diagnostic IA"],
  ["memory", "Mémoire"],
];
function Diagnostic({ result }) {
  return (
    <div className="ai-diagnostic" aria-live="polite">
      <strong className={`ai-state ${result?.status || "not_tested"}`}>
        {aiStatusLabel(result?.status || "not_tested")}
      </strong>
      <p>
        {result?.detail || "Déclencher un test pour vérifier la connexion."}
      </p>
      <dl>
        <div>
          <dt>Dernier contrôle</dt>
          <dd>{aiCheckedAt(result?.checked_at)}</dd>
        </div>
        <div>
          <dt>Durée du contrôle</dt>
          <dd>
            {Number.isFinite(result?.latency_ms)
              ? `${result.latency_ms} ms`
              : "Non mesurée"}
          </dd>
        </div>
      </dl>
    </div>
  );
}
export default function AIConfiguration({ onApplied }) {
  const id = useId(),
    toast = useToast();
  const [config, setConfig] = useState(null),
    [form, setForm] = useState(null),
    [loadError, setLoadError] = useState(""),
    [retry, setRetry] = useState(0);
  const [panel, setPanel] = useState("connection"),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(""),
    [test, setTest] = useState(null),
    [models, setModels] = useState(null);
  const controller = useRef(null),
    operation = useRef(0),
    version = useRef(0),
    busyRef = useRef("");
  useEffect(() => {
    const abort = new AbortController();
    setLoadError("");
    api
      .adminConfig({ signal: abort.signal })
      .then((c) => {
        if (abort.signal.aborted) return;
        if (!c || typeof c.provider !== "string")
          throw new Error("Configuration indisponible.");
        setConfig(c);
        setForm(aiForm(c));
      })
      .catch((e) => {
        if (!abort.signal.aborted && e.name !== "AbortError")
          setLoadError(e.message);
      });
    return () => abort.abort();
  }, [retry]);
  useEffect(
    () => () => {
      controller.current?.abort();
    },
    [],
  );
  const update = (key, value) => {
    setForm((f) => ({
      ...f,
      [key]: value,
      ...(key === "clear_keys"
        ? Object.fromEntries(value.map((field) => [field, ""]))
        : {}),
    }));
    setError("");
    if (key !== "context" && key !== "auto_categorize") {
      version.current++;
      setTest(null);
      if (busyRef.current === "test" || busyRef.current === "models") {
        controller.current?.abort();
        operation.current++;
        busyRef.current = "";
        setBusy("");
      }
      if (key === "provider" || key.endsWith("_url")) setModels(null);
    }
  };
  const begin = (mode) => {
    if (busyRef.current) return null;
    controller.current = new AbortController();
    busyRef.current = mode;
    setBusy(mode);
    setError("");
    return ++operation.current;
  };
  const finish = (token) => {
    if (token === operation.current) {
      busyRef.current = "";
      setBusy("");
    }
  };
  async function save() {
    const token = begin("save");
    if (!token) return;
    try {
      const saved = await api.adminPutConfig(aiPayload(form, config));
      setConfig(saved);
      setForm(aiForm(saved));
      setTest(null);
      setModels(null);
      version.current++;
      onApplied?.(saved);
      toast.ok("Configuration enregistrée et appliquée.");
    } catch (e) {
      setError(e.message);
    } finally {
      finish(token);
    }
  }
  async function reloadSaved() {
    const token = begin("reload");
    if (!token) return;
    try {
      const saved = await api.adminConfig({
        signal: controller.current.signal,
      });
      setConfig(saved);
      setForm(aiForm(saved));
      setTest(null);
      setModels(null);
      version.current++;
      onApplied?.(saved);
    } catch (e) {
      if (e.name !== "AbortError") setError(e.message);
    } finally {
      finish(token);
    }
  }
  async function testConnection() {
    const token = begin("test");
    if (!token) return;
    const atVersion = version.current;
    try {
      const result = await api.adminTestAI(aiPayload(form, config, true), {
        signal: controller.current.signal,
      });
      if (token === operation.current && atVersion === version.current)
        setTest({ version: atVersion, result });
    } catch (e) {
      if (e.name !== "AbortError" && token === operation.current)
        setError(e.message);
    } finally {
      finish(token);
    }
  }
  async function discover() {
    const token = begin("models");
    if (!token) return;
    try {
      const result = await api.adminOllamaModels(form.ollama_url.trim(), {
        signal: controller.current.signal,
      });
      if (token === operation.current) {
        if (!Array.isArray(result.models))
          throw new Error("Liste de modèles indisponible.");
        setModels(result.models);
      }
    } catch (e) {
      if (e.name !== "AbortError" && token === operation.current)
        setError(e.message);
    } finally {
      finish(token);
    }
  }
  async function testMemory() {
    const token = begin("memory");
    if (!token) return;
    try {
      const memory = await api.adminTestMemory({
        signal: controller.current.signal,
      });
      if (token === operation.current) setConfig((c) => ({ ...c, memory }));
    } catch (e) {
      if (e.name !== "AbortError") setError(e.message);
    } finally {
      finish(token);
    }
  }
  function reset() {
    update("provider", config.provider);
    setForm(aiForm(config));
    setModels(null);
    setTest(null);
  }
  function tabKey(event, index) {
    let next;
    if (event.key === "ArrowRight") next = (index + 1) % panels.length;
    if (event.key === "ArrowLeft")
      next = (index + panels.length - 1) % panels.length;
    if (event.key === "Home") next = 0;
    if (event.key === "End") next = panels.length - 1;
    if (next === undefined) return;
    event.preventDefault();
    setPanel(panels[next][0]);
    document.getElementById(`${id}-tab-${panels[next][0]}`)?.focus();
  }
  if (loadError)
    return (
      <section className="ai-settings">
        <h2>Configuration IA</h2>
        <p role="alert" className="notice err">
          {loadError}
        </p>
        <button
          className="btn-secondary"
          onClick={() => setRetry((n) => n + 1)}
        >
          Recharger la configuration IA
        </button>
      </section>
    );
  if (!config || !form)
    return (
      <section className="ai-settings">
        <h2>Configuration IA</h2>
        <p role="status">Chargement de la configuration IA…</p>
      </section>
    );
  const provider = aiProvider(form.provider),
    configured = aiProvider(config.provider),
    dirty = aiFormDirty(form, config),
    connectionDirty = aiConnectionDirty(form, config);
  const result =
    test?.version === version.current
      ? test.result
      : !connectionDirty
        ? config.diagnostic
        : null;
  const memory = config.memory,
    saving = busy === "save" || busy === "reload",
    source = provider.key && config.key_sources?.[form.provider];
  const stale =
    result?.revision && config.revision && result.revision !== config.revision;
  const draftResult = connectionDirty || result?.scope === "draft";
  const state =
    result?.status ||
    (form.provider === "none" ? "not_configured" : "not_tested");
  return (
    <section className="ai-settings" aria-labelledby={`${id}-title`}>
      <header className="ai-settings-header">
        <div>
          <span className="eyebrow">Administration</span>
          <h2 id={`${id}-title`}>Configuration IA</h2>
          <p>
            Enregistré : <strong>{configured.label}</strong>
            {configured.model && config[configured.model]
              ? ` · ${config[configured.model]}`
              : ""}{" "}
            ·{" "}
            {config.enabled
              ? "fonctions IA activées"
              : "fonctions IA désactivées"}
          </p>
        </div>
        <span className="ai-state">
          {dirty
            ? "Modifications non enregistrées"
            : "Configuration enregistrée"}
        </span>
      </header>
      {config.configuration_error && (
        <p role="alert" className="notice err">
          {config.configuration_error}
        </p>
      )}
      <AnimatedTabs
        className="ai-settings-tabs"
        role="tablist"
        aria-label="Configuration de l’intelligence artificielle"
      >
        {panels.map(([key, label], index) => (
          <button
            key={key}
            id={`${id}-tab-${key}`}
            role="tab"
            aria-selected={panel === key}
            aria-controls={`${id}-panel`}
            tabIndex={panel === key ? 0 : -1}
            onKeyDown={(e) => tabKey(e, index)}
            onClick={() => setPanel(key)}
          >
            {panel === key && <TabIndicator />}{label}
          </button>
        ))}
      </AnimatedTabs>
      <div
        className="ai-settings-panel"
        id={`${id}-panel`}
        role="tabpanel"
        aria-labelledby={`${id}-tab-${panel}`}
        tabIndex={0}
      >
        {panel === "connection" && (
          <>
            <div className="ai-settings-grid">
              <label>
                Fournisseur
                <select
                  aria-label="Fournisseur"
                  value={form.provider}
                  onChange={(e) => update("provider", e.target.value)}
                  disabled={saving}
                >
                  {AI_PROVIDERS.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.label}
                    </option>
                  ))}
                </select>
              </label>
              {provider.model && (
                <label>
                  Modèle
                  <input
                    value={form[provider.model]}
                    onChange={(e) => update(provider.model, e.target.value)}
                    maxLength={200}
                    spellCheck={false}
                    list={
                      form.provider === "ollama" ? `${id}-models` : undefined
                    }
                    disabled={saving}
                    placeholder="Identifiant du modèle"
                  />
                </label>
              )}
              {provider.url && (
                <label className="ai-url-field">
                  URL du serveur
                  <input
                    type="url"
                    value={form[provider.url]}
                    onChange={(e) => update(provider.url, e.target.value)}
                    disabled={saving}
                    placeholder={
                      form.provider === "ollama"
                        ? "http://ollama:11434"
                        : "https://serveur.example/v1"
                    }
                    autoComplete="off"
                    spellCheck={false}
                  />
                </label>
              )}
              {provider.key && (
                <label>
                  Nouvelle clé API
                  <input
                    type="password"
                    value={form[provider.key]}
                    onChange={(e) => update(provider.key, e.target.value)}
                    disabled={saving || form.clear_keys.includes(provider.key)}
                    autoComplete="new-password"
                    spellCheck={false}
                    placeholder={
                      config[`${form.provider}_key_set`]
                        ? "Clé définie · laisser vide pour la conserver"
                        : "Clé API requise"
                    }
                  />
                </label>
              )}
            </div>
            {provider.key && (
              <div className="ai-key-info">
                <p>
                  Clé :{" "}
                  {source === "stored"
                    ? "enregistrée sur le serveur"
                    : source === "environment"
                      ? "fournie par l’environnement du serveur"
                      : config[`${form.provider}_key_set`]
                        ? "définie sur le serveur"
                        : "non définie"}
                  . Les clés existantes ne sont jamais renvoyées au navigateur.
                </p>
                {source === "stored" && (
                  <label className="ai-check">
                    <input
                      type="checkbox"
                      checked={form.clear_keys.includes(provider.key)}
                      onChange={(e) =>
                        update(
                          "clear_keys",
                          e.target.checked
                            ? [...form.clear_keys, provider.key]
                            : form.clear_keys.filter((k) => k !== provider.key),
                        )
                      }
                      disabled={saving}
                    />
                    Retirer cette clé enregistrée à l’enregistrement
                  </label>
                )}
                {form.clear_keys.includes(provider.key) && (
                  <p>
                    Le retrait concerne la clé enregistrée. Une clé fournie par
                    le serveur peut rester utilisée ; une configuration active
                    sans clé sera refusée.
                  </p>
                )}
              </div>
            )}
            {form.provider === "ollama" && (
              <>
                <button
                  className="btn-secondary"
                  onClick={discover}
                  disabled={Boolean(busy) || !form.ollama_url.trim()}
                >
                  {busy === "models"
                    ? "Recherche des modèles…"
                    : "Rechercher les modèles"}
                </button>
                <datalist id={`${id}-models`}>
                  {models?.map((m) => (
                    <option key={m} value={m} />
                  ))}
                </datalist>
                {models && (
                  <p role="status">
                    {models.length
                      ? `${models.length} modèle(s) disponible(s). Choisir dans les suggestions ou conserver une saisie libre.`
                      : "Aucun modèle disponible sur ce serveur."}
                  </p>
                )}
                <p className="ai-help">
                  Serveur local ou distant. Les destinations réseau privées
                  doivent être autorisées côté serveur ; une URL contenant une
                  clé ou des identifiants est refusée.
                </p>
              </>
            )}
            {provider.key && (
              <p className="ai-help">
                Les fonctions IA transmettent au fournisseur les données
                nécessaires à leur traitement. Le test utilise uniquement un
                message synthétique et peut consommer des jetons API.
              </p>
            )}
            {form.provider === "none" && (
              <p>
                Les fonctions IA sont désactivées à l’enregistrement. Les
                réglages des fournisseurs et le contexte personnel sont
                conservés.
              </p>
            )}
            <div className="ai-connection-result">
              <span className={`ai-state ${state}`}>
                {aiStatusLabel(state)}
              </span>
              <p>
                {result?.detail ||
                  "La sélection d’un fournisseur ne vérifie pas sa connexion."}
              </p>
              {result?.checked_at && (
                <small>
                  {aiCheckedAt(result.checked_at)}
                  {Number.isFinite(result.latency_ms)
                    ? ` · ${result.latency_ms} ms`
                    : ""}{" "}
                  ·{" "}
                  {result.scope === "draft"
                    ? "brouillon testé"
                    : "configuration enregistrée testée"}
                </small>
              )}
            </div>
          </>
        )}
        {panel === "behavior" && (
          <>
            <h3>Fonctions IA</h3>
            <p>
              Choisir « Désactivé » dans Fournisseur et connexion pour
              désactiver les fonctions IA. L’enregistrement applique ce choix
              sans redémarrage.
            </p>
            <label className="ai-check">
              <input
                type="checkbox"
                checked={form.auto_categorize}
                onChange={(e) => update("auto_categorize", e.target.checked)}
                disabled={saving || form.provider === "none"}
              />
              Proposer des catégories et tags à l’import, à valider dans
              Transactions
            </label>
            <label className="ai-context-label">
              Contexte personnel
              <textarea
                aria-label="Contexte personnel"
                value={form.context}
                onChange={(e) => update("context", e.target.value)}
                rows={7}
                disabled={saving}
                aria-describedby={`${id}-context-help`}
                placeholder="Ex. : mes revenus sont irréguliers ; certaines charges sont trimestrielles."
              />
            </label>
            <div className="ai-context-count">
              {Array.from(form.context).length} / {AI_CONTEXT_LIMIT} caractères
            </div>
            <p className="ai-help" id={`${id}-context-help`}>
              Note globale utilisée pour éclairer les analyses et synthèses,
              hors extraction des relevés. Elle reste déclarative : les
              transactions et calculs structurés font référence. Elle ne modifie
              aucune opération bancaire et ne confère aucun droit
              supplémentaire. Éviter les secrets et informations sensibles
              inutiles.
            </p>
          </>
        )}
        {panel === "diagnostic" && (
          <>
            <div className="ai-settings-grid">
              <div>
                <h3>Configuration enregistrée</h3>
                <dl>
                  <div>
                    <dt>Fournisseur</dt>
                    <dd>{configured.label}</dd>
                  </div>
                  <div>
                    <dt>Modèle</dt>
                    <dd>
                      {(configured.model && config[configured.model]) ||
                        "Non configuré"}
                    </dd>
                  </div>
                  <div>
                    <dt>Activation</dt>
                    <dd>
                      {config.enabled
                        ? "Fonctions IA activées"
                        : "Fonctions IA désactivées"}
                    </dd>
                  </div>
                  <div>
                    <dt>Origine des réglages</dt>
                    <dd>
                      {config.source === "database"
                        ? "Base de données"
                        : config.source === "environment"
                          ? "Environnement du serveur"
                          : "Serveur"}
                    </dd>
                  </div>
                </dl>
              </div>
              <div>
                <h3>
                  {draftResult
                    ? "Connexion du brouillon"
                    : "Connexion enregistrée"}
                </h3>
                <Diagnostic result={result} />
                {connectionDirty && (
                  <p className="ai-help">
                    Ce contrôle concerne les champs modifiés. La configuration
                    enregistrée reste en place jusqu’à l’enregistrement.
                  </p>
                )}
              </div>
            </div>
            <p className="ai-help">
              Un résultat décrit le dernier contrôle effectué, sans garantir la
              disponibilité future. Aucun contrôle de fournisseur n’est lancé
              automatiquement à l’ouverture de cette page.
            </p>
          </>
        )}
        {panel === "memory" && (
          <>
            <h3>Mémoire et Knowledge Graph</h3>
            <p>
              Graphiti et FalkorDB constituent une mémoire facultative.
              L’assistant fonctionne sans cette mémoire lorsque le fournisseur
              IA est disponible.
            </p>
            <dl>
              <div>
                <dt>Activation effective</dt>
                <dd>
                  {memory?.enabled
                    ? "Activée côté serveur"
                    : "Désactivée côté serveur"}
                </dd>
              </div>
              <div>
                <dt>Configuration du service</dt>
                <dd>
                  {memory?.configured
                    ? "Adresse de service configurée"
                    : "Non configurée"}
                </dd>
              </div>
            </dl>
            <Diagnostic result={memory?.diagnostic} />
            <button
              className="btn-secondary"
              onClick={testMemory}
              disabled={
                Boolean(busy) || !memory?.enabled || !memory?.configured
              }
            >
              {busy === "memory" ? "Contrôle en cours…" : "Vérifier la mémoire"}
            </button>
            <p className="ai-help">
              Le contrôle consulte seulement la santé du service ; il ne lit pas
              les faits du graphe. La configuration et l’activation de la
              mémoire restent gérées côté serveur.
            </p>
          </>
        )}
      </div>
      {stale && (
        <p className="notice" role="status">
          La configuration enregistrée a changé depuis son chargement. Recharger
          les réglages avant d’enregistrer.
        </p>
      )}
      {error && (
        <p className="notice err" role="alert">
          {error}
        </p>
      )}
      {(stale || error) && (
        <div>
          <button
            className="btn-secondary"
            onClick={reloadSaved}
            disabled={Boolean(busy)}
          >
            Recharger les réglages enregistrés
          </button>
          <p className="ai-help">
            Un rechargement réussi remplace le brouillon par les réglages du
            serveur.
          </p>
        </div>
      )}
      {busy && (
        <p className="ai-operation" role="status">
          {busy === "save"
            ? "Enregistrement de la configuration…"
            : busy === "reload"
              ? "Rechargement des réglages…"
              : "Contrôle en cours…"}
        </p>
      )}
      <footer className="ai-settings-actions">
        <button
          className="btn-secondary"
          onClick={testConnection}
          disabled={Boolean(busy) || form.provider === "none"}
        >
          {busy === "test" ? "Test en cours…" : "Tester la connexion"}
        </button>
        <button
          className="btn"
          onClick={save}
          disabled={Boolean(busy) || !dirty}
        >
          Enregistrer et appliquer
        </button>
        <button
          className="link"
          onClick={reset}
          disabled={Boolean(busy) || !dirty}
        >
          Annuler les modifications
        </button>
        <span className="ai-help">
          Le test ne sauvegarde pas. L’enregistrement applique les réglages sans
          prétendre vérifier la connexion.
        </span>
      </footer>
    </section>
  );
}
