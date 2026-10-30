import { useState } from 'react'
import { api } from '../api'
import { useAnalysisResource } from '../components/analysisResources'
import { useLaboratory } from '../components/laboratory/LaboratoryProvider'
import QuickScratchpad from '../components/laboratory/QuickScratchpad'
import SpreadsheetWorkspace from '../components/laboratory/SpreadsheetWorkspace'
import {
  newWorkbook,
  migrateDocument,
} from '../lib/laboratory/workbookModel.mjs'
import '../components/laboratory/laboratory.css'
function readDocument(d) {
  const doc = migrateDocument(d)
  if (
    !Number.isSafeInteger(doc.id) ||
    doc.id < 1 ||
    !Number.isSafeInteger(doc.revision) ||
    doc.revision < 1 ||
    typeof doc.client_key !== 'string' ||
    typeof doc.title !== 'string' ||
    !['workbook', 'scratchpad'].includes(doc.type)
  )
    throw new Error('Document serveur invalide ; brouillon conservé.')
  return doc
}
export default function Laboratory() {
  const lab = useLaboratory(),
    { state, setState } = lab
  const [offset, setOffset] = useState(0),
    [mode, setMode] = useState('sheet'),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(''),
    [message, setMessage] = useState('')
  const docs = useAnalysisResource(
    (signal) =>
      api.laboratoryList(offset, { signal }).then((data) => {
        if (
          !Array.isArray(data?.documents) ||
          typeof data.has_more !== 'boolean' ||
          data.documents.some(
            (d) =>
              !Number.isSafeInteger(d.id) ||
              !Number.isSafeInteger(d.revision) ||
              typeof d.title !== 'string' ||
              typeof d.client_key !== 'string',
          )
        )
          throw new Error('Liste de documents invalide.')
        return data
      }),
    [offset, state.listRevision],
  )
  const current = state.drafts[state.active]
  const update = (change) =>
    setState((s) => ({
      ...s,
      drafts: {
        ...s.drafts,
        [s.active]: { ...s.drafts[s.active], ...change, dirty: true },
      },
    }))
  const activate = (doc) => {
    setState((s) => ({
      ...s,
      active: doc.client_key,
      drafts: { ...s.drafts, [doc.client_key]: doc },
      ...(doc.type === 'scratchpad'
        ? { quick: doc.content, quickDirty: doc.dirty }
        : {}),
    }))
    setMode(doc.type === 'scratchpad' ? 'quick' : 'sheet')
  }
  const create = () => {
    try {
      lab.add(newWorkbook())
      setMode('sheet')
      setError('')
    } catch (e) {
      setError(e.message)
    }
  }
  const open = async (meta, reload = false) => {
    setBusy(true)
    setError('')
    setMessage('')
    try {
      const local = state.drafts[meta.client_key]
      if (local && !reload) {
        activate(local)
        return
      }
      if (
        reload &&
        local?.dirty &&
        !confirm(
          'Remplacer les modifications locales par la version serveur ? Tu peux enregistrer une copie pour les conserver.',
        )
      )
        return
      if (Object.keys(state.drafts).length >= 4 && !local)
        throw new Error(
          'Quatre brouillons ouverts. Ferme une feuille locale avant d’en ouvrir une autre.',
        )
      const doc = readDocument(await api.laboratoryGet(meta.id))
      activate({ ...doc, dirty: false })
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }
  const save = async (copy = false) => {
    if (!current) return
    setBusy(true)
    setError('')
    setMessage('')
    const original = current,
      key = original.client_key,
      snapshot = JSON.stringify(original.content)
    const data = {
      client_key: copy ? crypto.randomUUID() : key,
      title: copy ? `${original.title.slice(0, 100)} (copie)` : original.title,
      type: original.type,
      schema_version: 1,
      revision: copy ? 0 : original.id ? original.revision : 0,
      content: original.content,
    }
    try {
      const saved = readDocument(
        await api.laboratorySave(copy ? null : original.id, data),
      )
      setState((s) => {
        if (copy) {
          return { ...s, listRevision: s.listRevision + 1 }
        }
        const now = s.drafts[key]
        if (!now) return { ...s, listRevision: s.listRevision + 1 }
        const dirty =
          saved.replayed ||
          JSON.stringify(now.content) !== snapshot ||
          now.title !== data.title
        return {
          ...s,
          drafts: {
            ...s.drafts,
            [key]: {
              ...now,
              id: saved.id,
              revision: saved.revision,
              created_at: saved.created_at,
              updated_at: saved.updated_at,
              dirty,
            },
          },
          listRevision: s.listRevision + 1,
          ...(now.type === 'scratchpad' ? { quickDirty: dirty } : {}),
        }
      })
      setMessage(
        saved.replayed
          ? 'Document déjà créé retrouvé. Le brouillon est conservé ; enregistre ses dernières modifications.'
          : copy
            ? 'Copie enregistrée, original local conservé.'
            : 'Document enregistré sur le serveur.',
      )
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }
  const close = (key) => {
    const d = state.drafts[key]
    if (
      d.dirty &&
      !confirm(
        'Fermer ce brouillon local sans enregistrer ses modifications ? Le document serveur, s’il existe, est conservé.',
      )
    )
      return
    setState((s) => {
      const drafts = { ...s.drafts }
      delete drafts[key]
      return {
        ...s,
        drafts,
        active: s.active === key ? Object.keys(drafts)[0] || null : s.active,
      }
    })
  }
  const remove = async (meta) => {
    if (!confirm(`Supprimer définitivement « ${meta.title} » du serveur ?`))
      return
    setBusy(true)
    setError('')
    try {
      await api.laboratoryDelete(meta.id, meta.revision)
      setState((s) => {
        const drafts = { ...s.drafts }
        if (drafts[meta.client_key]) {
          const d = drafts[meta.client_key]
          drafts[meta.client_key] = {
            ...d,
            id: undefined,
            revision: 0,
            client_key: crypto.randomUUID(),
            dirty: true,
          }
          const next = drafts[meta.client_key]
          delete drafts[meta.client_key]
          drafts[next.client_key] = next
          return {
            ...s,
            drafts,
            active: s.active === meta.client_key ? next.client_key : s.active,
            listRevision: s.listRevision + 1,
          }
        }
        return { ...s, listRevision: s.listRevision + 1 }
      })
      setMessage(
        'Document serveur supprimé. Un éventuel brouillon ouvert reste conservé localement.',
      )
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="laboratory">
      <div className="lab-actions">
        <h1>Laboratoire</h1>
        <button className="btn" onClick={create}>
          Nouvelle feuille
        </button>
        <button className="btn-secondary" onClick={lab.open}>
          Calcul rapide
        </button>
      </div>
      <p>
        Calculs et tableaux libres, indépendants de vos transactions et budgets.
        Aucun calcul ne modifie vos données bancaires ni ne contacte un
        fournisseur IA.
      </p>
      {state.storageError && (
        <div role="alert">
          <p>
            {state.storageError} La récupération existante n’est pas écrasée.
          </p>
          <button
            className="btn-secondary"
            onClick={() => {
              if (
                confirm(
                  'Effacer la récupération locale illisible de cet utilisateur et repartir avec les brouillons visibles ?',
                )
              )
                lab.forgetRecovery()
            }}
          >
            Réinitialiser la récupération locale
          </button>
        </div>
      )}
      <div className="lab-manager">
        <h2>Documents et brouillons</h2>
        <p>
          Jusqu’à quatre documents ouverts dans cet onglet ; sauvegarde serveur
          explicite. La récupération locale est effacée à la déconnexion. Un
          autre onglet ne peut pas écraser une révision plus récente.
        </p>
        <ul>
          {Object.entries(state.drafts).map(([key, d]) => (
            <li key={key}>
              <button className="link" onClick={() => activate(d)}>
                {d.title} · {d.dirty ? 'Modifications locales' : 'Enregistré'}
              </button>{' '}
              <button
                className="btn-secondary"
                disabled={busy}
                onClick={() => close(key)}
              >
                Fermer {d.title}
              </button>
            </li>
          ))}
        </ul>
        <details>
          <summary>Ouvrir ou supprimer un document serveur</summary>
          {docs.loading && <p role="status">Chargement des documents…</p>}
          {docs.error && (
            <p role="alert">
              {docs.error} <button onClick={docs.retry}>Réessayer</button>
            </p>
          )}
          {docs.data?.documents.length === 0 && (
            <p>Aucun document enregistré.</p>
          )}
          <ul>
            {docs.data?.documents.map((d) => (
              <li key={d.id}>
                <button
                  className="link"
                  disabled={busy}
                  onClick={() => open(d)}
                >
                  {d.title} · révision {d.revision}
                </button>{' '}
                <button
                  className="btn-secondary"
                  disabled={busy}
                  onClick={() => remove(d)}
                >
                  Supprimer {d.title}
                </button>
              </li>
            ))}
          </ul>
          <div className="lab-actions">
            {offset > 0 && (
              <button onClick={() => setOffset((n) => n - 50)}>
                Précédents
              </button>
            )}
            {docs.data?.has_more && (
              <button onClick={() => setOffset((n) => n + 50)}>Suivants</button>
            )}
          </div>
        </details>
      </div>
      {error && <p role="alert">{error} Le brouillon courant est conservé.</p>}
      {message && <p role="status">{message}</p>}
      <div className="lab-tabs">
        <button
          className="btn-secondary"
          aria-pressed={mode === 'quick'}
          onClick={() => setMode('quick')}
        >
          Mode rapide
        </button>
        <button
          className="btn-secondary"
          aria-pressed={mode === 'sheet'}
          onClick={() => setMode('sheet')}
        >
          Mode feuille
        </button>
      </div>
      {current && (
        <>
          <div className="lab-fields">
            <label>
              Titre du document
              <input
                maxLength={120}
                value={current.title}
                onChange={(e) => update({ title: e.target.value })}
              />
            </label>
          </div>
          <div className="lab-actions">
            <button
              className="btn"
              disabled={busy || !current.title.trim()}
              onClick={() => save(false)}
            >
              {busy ? 'Enregistrement…' : 'Enregistrer le document'}
            </button>
            <button
              className="btn-secondary"
              disabled={busy || !current.title.trim()}
              onClick={() => save(true)}
            >
              Enregistrer une copie
            </button>
            {current.id && (
              <button
                className="btn-secondary"
                disabled={busy}
                onClick={() => open(current, true)}
              >
                Recharger la version serveur
              </button>
            )}
          </div>
          <p>
            {current.id
              ? `Révision serveur ${current.revision}. `
              : 'Brouillon non encore enregistré sur le serveur. '}
            {current.dirty ? 'Modifications à enregistrer.' : 'Enregistré.'}
          </p>
        </>
      )}
      {mode === 'quick' ? (
        <QuickScratchpad key={state.active || 'quick'} />
      ) : current ? (
        <SpreadsheetWorkspace
          key={current.client_key}
          content={current.content}
          onChange={(content) => update({ content })}
        />
      ) : (
        <p>
          Aucune feuille ouverte. Crée une feuille ou transforme un calcul
          rapide.
        </p>
      )}
    </div>
  )
}
