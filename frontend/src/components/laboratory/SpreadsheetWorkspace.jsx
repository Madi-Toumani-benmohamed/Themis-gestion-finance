import { memo, useEffect, useRef, useState } from 'react'
import {
  address,
  coordinates,
  validateContent,
  shiftedFormula,
  safeClipboardText,
  resizeContent,
} from '../../lib/laboratory/workbookModel.mjs'
import { euroCents } from '../money.mjs'
import { useCalculationWorker } from './useCalculationWorker'
const display = (result) =>
  !result
    ? ''
    : result.type === 'error'
      ? result.code
      : result.money_cents !== undefined
        ? euroCents(result.money_cents)
        : result.formatted || result.value
const Cell = memo(function Cell({
  refName,
  raw,
  shown,
  selected,
  active,
  onSelect,
  onCommit,
  onMove,
}) {
  const [editing, setEditing] = useState(false),
    [draft, setDraft] = useState(raw)
  useEffect(() => {
    if (!editing) setDraft(raw)
  }, [raw, editing])
  const commit = () => {
    if (editing && draft !== raw) onCommit(refName, draft)
    setEditing(false)
  }
  const start = (value) => {
    setDraft(value ?? raw)
    setEditing(true)
  }
  const key = (e) => {
    if (editing) {
      if (e.key === 'Escape') {
        e.preventDefault()
        setEditing(false)
        setDraft(raw)
      } else if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault()
        commit()
        onMove(
          e.key === 'Enter'
            ? 'ArrowDown'
            : e.shiftKey
              ? 'ArrowLeft'
              : 'ArrowRight',
          false,
        )
      }
      return
    }
    if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(e.key)) {
      e.preventDefault()
      onMove(e.key, e.shiftKey)
    } else if (e.key === 'Enter' || e.key === 'F2') {
      e.preventDefault()
      start()
    } else if (!e.ctrlKey && !e.metaKey && !e.altKey && e.key.length === 1) {
      e.preventDefault()
      start(e.key)
    }
  }
  return (
    <td data-selected={selected} data-error={shown?.type === 'error'}>
      <input
        aria-label={refName}
        data-cell={refName}
        data-editing={editing}
        tabIndex={active ? 0 : -1}
        maxLength={512}
        readOnly={!editing}
        value={editing ? draft : display(shown)}
        title={shown?.type === 'error' ? shown.message : raw}
        onFocus={() => onSelect(refName, null)}
        onClick={(e) => onSelect(refName, e.shiftKey)}
        onDoubleClick={() => start()}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={key}
        onBlur={commit}
      />
    </td>
  )
})
export default function SpreadsheetWorkspace({ content, onChange }) {
  const engine = useCalculationWorker(),
    [calculated, setCalculated] = useState({ source: null, values: {} }),
    [error, setError] = useState(''),
    [active, setActive] = useState('A1'),
    [anchor, setAnchor] = useState('A1'),
    [end, setEnd] = useState('A1')
  const [formula, setFormula] = useState(content.cells.A1?.input || ''),
    [past, setPast] = useState([]),
    [future, setFuture] = useState([])
  const calculation = calculated.source === content ? calculated.values : {}
  const latest = useRef(content)
  latest.current = content
  useEffect(() => {
    let valid = true
    const timer = setTimeout(() => {
      engine
        .run({ mode: 'workbook', content })
        .then((r) => {
          if (valid) setCalculated({ source: content, values: r })
        })
        .catch((e) => {
          if (valid && e.name !== 'AbortError') {
            setError(e.message)
            setCalculated({ source: content, values: {} })
          }
        })
    }, 180)
    return () => {
      valid = false
      clearTimeout(timer)
    }
  }, [content])
  useEffect(() => {
    setFormula(content.cells[active]?.input || '')
  }, [active, content.cells[active]?.input])
  const apply = (next) => {
    try {
      validateContent(next)
      setPast((p) => [...p, structuredClone(latest.current)].slice(-20))
      setFuture([])
      onChange(next)
      setError('')
    } catch (e) {
      setError(e.message)
    }
  }
  const commit = (ref, input) => {
    const next = structuredClone(latest.current)
    if (input === '') delete next.cells[ref]
    else
      next.cells[ref] = { input, format: next.cells[ref]?.format || 'number' }
    apply(next)
  }
  const select = (ref, extend) => {
    setActive(ref)
    if (extend === null) return
    setEnd(ref)
    if (!extend) setAnchor(ref)
  }
  const move = (direction, extend) => {
    const [r, c] = coordinates(active)
    const target = address(
      Math.max(
        0,
        Math.min(
          content.rows - 1,
          r +
            (direction === 'ArrowDown' ? 1 : direction === 'ArrowUp' ? -1 : 0),
        ),
      ),
      Math.max(
        0,
        Math.min(
          content.columns - 1,
          c +
            (direction === 'ArrowRight'
              ? 1
              : direction === 'ArrowLeft'
                ? -1
                : 0),
        ),
      ),
    )
    select(target, extend)
    requestAnimationFrame(() =>
      document.querySelector(`[data-cell="${target}"]`)?.focus(),
    )
  }
  const [ar, ac] = coordinates(anchor),
    [br, bc] = coordinates(end)
  const range = {
    top: Math.min(ar, br),
    bottom: Math.max(ar, br),
    left: Math.min(ac, bc),
    right: Math.max(ac, bc),
  }
  const selected = (ref) => {
    const [r, c] = coordinates(ref)
    return (
      r >= range.top && r <= range.bottom && c >= range.left && c <= range.right
    )
  }
  const undo = () => {
    if (!past.length) return
    setFuture((f) => [structuredClone(content), ...f].slice(0, 20))
    onChange(past.at(-1))
    setPast((p) => p.slice(0, -1))
  }
  const redo = () => {
    if (!future.length) return
    setPast((p) => [...p, structuredClone(content)].slice(-20))
    onChange(future[0])
    setFuture((f) => f.slice(1))
  }
  const copy = (e) => {
    if (e.target.dataset.editing === 'true') return
    const matrix = [],
      plain = []
    for (let r = range.top; r <= range.bottom; r++) {
      const row = [],
        text = []
      for (let c = range.left; c <= range.right; c++) {
        const ref = address(r, c)
        row.push(content.cells[ref] || { input: '', format: 'number' })
        text.push(safeClipboardText(display(calculation[ref])))
      }
      matrix.push(row)
      plain.push(text.join('\t'))
    }
    e.preventDefault()
    e.clipboardData.setData('text/plain', plain.join('\n'))
    e.clipboardData.setData(
      'application/x-themis-laboratory',
      JSON.stringify({ row: range.top, column: range.left, cells: matrix }),
    )
  }
  const paste = (e) => {
    if (e.target.dataset.editing === 'true') return
    e.preventDefault()
    try {
      const raw = e.clipboardData.getData('application/x-themis-laboratory'),
        text = e.clipboardData.getData('text/plain')
      if (raw.length + text.length > 200000)
        throw new Error('Presse-papiers trop volumineux.')
      const internal = raw ? JSON.parse(raw) : null
      const matrix = internal
        ? internal.cells
        : text
            .replace(/\r/g, '')
            .replace(/\n$/, '')
            .split('\n')
            .map((row) =>
              row.split('\t').map((input) => ({ input, format: 'number' })),
            )
      if (
        !Array.isArray(matrix) ||
        matrix.length > 100 ||
        matrix.some((row) => !Array.isArray(row) || row.length > 26)
      )
        throw new Error('Plage copiée invalide.')
      const [r, c] = coordinates(active)
      if (
        r + matrix.length > content.rows ||
        matrix.some((row) => c + row.length > content.columns)
      )
        throw new Error(
          'Collage hors feuille : ajoute des lignes ou colonnes avant de réessayer.',
        )
      if (
        internal &&
        (!Number.isInteger(internal.row) || !Number.isInteger(internal.column))
      )
        throw new Error('Origine du collage invalide.')
      const next = structuredClone(content)
      matrix.forEach((row, dr) =>
        row.forEach((cell, dc) => {
          if (typeof cell?.input !== 'string')
            throw new Error('Cellule collée invalide.')
          next.cells[address(r + dr, c + dc)] = {
            input: internal
              ? shiftedFormula(
                  cell.input,
                  r - internal.row,
                  c - internal.column,
                )
              : cell.input,
            format: cell.format,
          }
        }),
      )
      apply(next)
    } catch (e) {
      setError(e.message)
    }
  }
  const keyboard = (e) => {
    if (e.target.dataset.editing === 'true') return
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z') {
      e.preventDefault()
      e.shiftKey ? redo() : undo()
    } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'y') {
      e.preventDefault()
      redo()
    } else if (e.key === 'Delete' || e.key === 'Backspace') {
      e.preventDefault()
      const next = structuredClone(content)
      for (const ref of Object.keys(next.cells))
        if (selected(ref)) delete next.cells[ref]
      apply(next)
    }
  }
  const resize = (rows, cols) => {
    if (rows < content.rows || cols < content.columns) {
      if (
        !confirm(
          'Supprimer la dernière ligne/colonne et son contenu ? Les références concernées seront marquées invalides. Annuler restera possible.',
        )
      )
        return
    }
    apply(resizeContent(content, rows, cols))
    if (coordinates(active)[0] >= rows || coordinates(active)[1] >= cols)
      select('A1', false)
  }
  return (
    <section aria-label="Mini-tableur">
      <h2>Feuille de calcul</h2>
      <p>
        Entrée ou F2 pour éditer ; Échap pour abandonner. Flèches pour naviguer,
        Maj + flèches/clic pour sélectionner une plage. Ctrl/Cmd+C/V pour
        copier/coller ; Z ou Maj+Z pour annuler/rétablir. Formules avec point
        décimal et ; entre arguments.
      </p>
      <div className="lab-actions">
        <button
          className="btn-secondary"
          disabled={!past.length}
          onClick={undo}
        >
          Annuler
        </button>
        <button
          className="btn-secondary"
          disabled={!future.length}
          onClick={redo}
        >
          Rétablir
        </button>
        <button
          className="btn-secondary"
          disabled={content.rows >= 100}
          onClick={() => resize(content.rows + 1, content.columns)}
        >
          Ajouter une ligne
        </button>
        <button
          className="btn-secondary"
          disabled={content.rows <= 2}
          onClick={() => resize(content.rows - 1, content.columns)}
        >
          Supprimer la dernière ligne
        </button>
        <button
          className="btn-secondary"
          disabled={content.columns >= 26}
          onClick={() => resize(content.rows, content.columns + 1)}
        >
          Ajouter une colonne
        </button>
        <button
          className="btn-secondary"
          disabled={content.columns <= 2}
          onClick={() => resize(content.rows, content.columns - 1)}
        >
          Supprimer la dernière colonne
        </button>
      </div>
      <div className="lab-fields">
        <label>
          Formule ou valeur · {active}
          <input
            value={formula}
            maxLength={512}
            onChange={(e) => setFormula(e.target.value)}
            onBlur={() => {
              if (formula !== (content.cells[active]?.input || ''))
                commit(active, formula)
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                commit(active, formula)
              }
            }}
          />
        </label>
        <label>
          Format de la sélection
          <select
            value={content.cells[active]?.format || 'number'}
            onChange={(e) => {
              const next = structuredClone(content)
              for (let r = range.top; r <= range.bottom; r++)
                for (let c = range.left; c <= range.right; c++) {
                  const ref = address(r, c)
                  next.cells[ref] = {
                    input: next.cells[ref]?.input || '',
                    format: e.target.value,
                  }
                }
              apply(next)
            }}
          >
            <option value="number">Nombre décimal</option>
            <option value="currency">Monétaire · arrondi au centime</option>
            <option value="percent">Pourcentage</option>
          </select>
        </label>
      </div>
      <p>
        Sélection : {anchor}:{end}.{' '}
        {engine.running || calculated.source !== content
          ? 'Recalcul en cours…'
          : 'Valeurs calculées à partir de cette feuille uniquement.'}
      </p>
      {error && <p role="alert">{error}</p>}
      <div
        className="lab-grid"
        tabIndex={0}
        role="region"
        aria-label="Grille défilante"
        onCopy={copy}
        onPaste={paste}
        onKeyDown={keyboard}
      >
        <table>
          <caption>
            Feuille de calcul — {content.rows} lignes et {content.columns}{' '}
            colonnes
          </caption>
          <thead>
            <tr>
              <th scope="col">Ligne</th>
              {Array.from({ length: content.columns }, (_, c) => (
                <th scope="col" key={c}>
                  {String.fromCharCode(65 + c)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: content.rows }, (_, r) => (
              <tr key={r}>
                <th scope="row">{r + 1}</th>
                {Array.from({ length: content.columns }, (_, c) => {
                  const ref = address(r, c)
                  return (
                    <Cell
                      key={ref}
                      refName={ref}
                      raw={content.cells[ref]?.input || ''}
                      shown={calculation[ref]}
                      selected={selected(ref)}
                      active={ref === active}
                      onSelect={select}
                      onCommit={commit}
                      onMove={move}
                    />
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {content.source_expression && (
        <details>
          <summary>Origine du calcul et historique conservés</summary>
          <p>Expression originale : {content.source_expression}</p>
          <p>Résultat à la conversion : {content.source_result}</p>
          <p>
            Valeurs numériques extraites :{' '}
            {content.source_operands.join(' ; ') ||
              'Non extraites pour cette expression complexe ; original intégral conservé.'}
          </p>
          <ol>
            {content.history.map((h, i) => (
              <li key={i}>
                {h.expression} = {h.result}
              </li>
            ))}
          </ol>
        </details>
      )}
      <details>
        <summary>Règles numériques et limites</summary>
        <p>
          SUM, AVERAGE, MIN, MAX, COUNT, IF et ROUND. Références A1 et $A$1,
          plages A1:B12 ; cellules vides = 0 dans l’arithmétique et ignorées
          dans les agrégats. IF n’évalue que la branche retenue. Calcul décimal
          à 40 chiffres significatifs ; arrondi monétaire explicite au centime,
          demi-centime à l’écart de zéro. Pourcentages mathjs : 100+10%=110. Pas
          de JavaScript, fonctions réseau ou fichiers. Maximum 100×26, 1000
          cellules non vides, 512 caractères par cellule ; trois secondes par
          calcul. Vingt niveaux d’annulation en mémoire.
        </p>
      </details>
    </section>
  )
}
