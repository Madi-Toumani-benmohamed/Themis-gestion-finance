import { useEffect, useRef, useState } from 'react'
import { useModal } from '../useModal'
import { useLaboratory } from './LaboratoryProvider'
import { useCalculationWorker } from './useCalculationWorker'
import './laboratory.css'
import { api } from '../../api'
import { documentKey } from '../../lib/laboratory/documentKey.mjs'
import { safeClipboardText } from '../../lib/laboratory/workbookModel.mjs'
function Content({ onClose }) {
  const lab = useLaboratory(),
    q = lab.state.quick
  const [expression, setExpression] = useState(q.expression),
    [result, setResult] = useState(null),
    [error, setError] = useState(''),
    [copied, setCopied] = useState(false)
  const saveKey = useRef(null),
    [saving, setSaving] = useState(false),
    [saved, setSaved] = useState(false)
  const input = useRef(null),
    version = useRef(0),
    engine = useCalculationWorker()
  useEffect(() => {
    input.current?.focus()
    input.current?.select()
  }, [])
  useEffect(() => {
    if (expression !== lab.state.quick.expression)
      lab.updateQuick((q) => ({ ...q, expression }))
  }, [expression])
  const calculate = async (e) => {
    e?.preventDefault()
    const revision = ++version.current
    setError('')
    setCopied(false)
    try {
      const value = await engine.run({ mode: 'quick', expression })
      if (revision !== version.current) return
      setResult(value.value)
      lab.updateQuick((q) => ({
        ...q,
        expression,
        history: [
          {
            expression,
            result: value.value,
            created_at: new Date().toISOString(),
          },
          ...q.history,
        ].slice(0, 50),
      }))
    } catch (e) {
      if (e.name !== 'AbortError' && revision === version.current) {
        setResult(null)
        setError(e.message)
      }
    }
  }
  return (
    <div className="laboratory quick-scratchpad">
      <div className="lab-actions">
        <h2>Calcul rapide</h2>
        {onClose && (
          <button className="btn-secondary" onClick={onClose}>
            Fermer
          </button>
        )}
      </div>
      <form onSubmit={calculate}>
        <label>
          Expression
          <input
            ref={input}
            autoComplete="off"
            maxLength={512}
            value={expression}
            placeholder="(1500 - 275) / 3"
            onChange={(e) => {
              version.current++
              engine.clear()
              setResult(null)
              setError('')
              setExpression(e.target.value)
            }}
          />
        </label>
        <p>
          Point décimal ; arguments séparés par ;. Le pourcentage suit mathjs :
          10 % = 0.1 et 100 + 10 % = 110.
        </p>
        <button className="btn" disabled={engine.running || !expression.trim()}>
          Calculer
        </button>
      </form>
      {engine.running && <p role="status">Calcul en cours…</p>}
      {error && <p role="alert">{error}</p>}
      {result !== null && (
        <>
          <output className="lab-result" aria-label="Résultat du calcul">
            {result}
          </output>
          <div className="lab-actions">
            <button
              className="btn-secondary"
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(safeClipboardText(result))
                  setCopied(true)
                } catch {
                  setError(
                    'Copie indisponible : sélectionne le résultat affiché.',
                  )
                }
              }}
            >
              Copier le résultat
            </button>
            <button
              className="btn"
              onClick={() => {
                try {
                  lab.convert(expression, result)
                } catch (e) {
                  setError(e.message)
                }
              }}
            >
              Ouvrir en feuille
            </button>
          </div>
        </>
      )}
      {copied && <p role="status">Résultat copié.</p>}
      <button
        className="btn-secondary"
        disabled={saving || !expression.trim()}
        onClick={async () => {
          setSaving(true)
          setError('')
          const content = { ...lab.state.quick, expression }
          try {
            if (!saveKey.current) saveKey.current = documentKey()
            const savedDocument = await api.laboratorySave(null, {
              client_key: saveKey.current,
              title: 'Calcul rapide',
              type: 'scratchpad',
              schema_version: 1,
              revision: 0,
              content,
            })
            setSaved(!savedDocument.replayed)
            if (savedDocument.replayed)
              setError(
                'Une première copie existe déjà. Le brouillon courant reste à enregistrer.',
              )
            saveKey.current = null
            lab.setState((s) => ({
              ...s,
              listRevision: s.listRevision + 1,
              quickDirty:
                !!savedDocument.replayed ||
                JSON.stringify(s.quick) !== JSON.stringify(content),
            }))
          } catch (e) {
            setError(e.message)
          } finally {
            setSaving(false)
          }
        }}
      >
        Enregistrer une copie du brouillon
      </button>
      {saved && <p role="status">Brouillon enregistré sur le serveur.</p>}
      <details>
        <summary>Historique des 50 derniers calculs</summary>
        <ol>
          {q.history.map((h, i) => (
            <li key={`${h.created_at}:${i}`}>
              <button
                className="link"
                onClick={() => {
                  version.current++
                  engine.clear()
                  setError('')
                  setExpression(h.expression)
                  setResult(h.result)
                }}
              >
                {h.expression} = {h.result}
              </button>
            </li>
          ))}
        </ol>
      </details>
      <button
        className="link"
        onClick={() => {
          if (
            confirm(
              'Effacer le calcul rapide et son historique ? Les feuilles restent conservées.',
            )
          ) {
            version.current++
            engine.clear()
            setExpression('')
            setResult(null)
            setError('')
            lab.updateQuick((q) => ({ ...q, expression: '', history: [] }))
          }
        }}
      >
        Réinitialiser le calcul rapide
      </button>
      <label>
        Raccourci global
        <select
          value={lab.state.shortcut}
          onChange={(e) =>
            lab.setState((s) => ({ ...s, shortcut: e.target.value }))
          }
        >
          <option value="space">Ctrl/Cmd + Maj + Espace</option>
          <option value="l">Ctrl/Cmd + Maj + L</option>
          <option value="off">Désactivé</option>
        </select>
      </label>
      <p>
        Raccourci ignoré pendant la saisie ou dans un dialogue. Brouillon
        temporaire lié à cet utilisateur et cet onglet ; aucune sauvegarde
        serveur par frappe.
      </p>
      {lab.state.storageError && <p role="alert">{lab.state.storageError}</p>}
    </div>
  )
}
function QuickDialog({ onClose }) {
  const ref = useRef(null)
  const dismiss = useModal(ref)
  const close = () => dismiss(onClose)
  return (
    <dialog
      ref={ref}
      className="lab-dialog"
      aria-label="Calcul rapide"
      onCancel={(e) => {
        e.preventDefault()
        close()
      }}
    >
      <Content onClose={close} />
    </dialog>
  )
}
export default function QuickScratchpad({ modal = false, onClose }) {
  return modal ? <QuickDialog onClose={onClose} /> : <Content />
}
