import {
  createContext,
  lazy,
  Suspense,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'
import {
  emptyContent,
  validateContent,
  migrateDocument,
  convertCalculation,
} from '../../lib/laboratory/workbookModel.mjs'
const QuickScratchpad = lazy(() => import('./QuickScratchpad'))
const Context = createContext(null)
const prefix = 'themis-laboratory:'
export function clearLaboratoryRecovery() {
  try {
    for (let i = sessionStorage.length - 1; i >= 0; i--) {
      const key = sessionStorage.key(i)
      if (key?.startsWith(prefix)) sessionStorage.removeItem(key)
    }
  } catch {}
}
export const useLaboratory = () => useContext(Context)
const initial = () => ({
  drafts: {},
  active: null,
  quick: emptyContent(),
  quickDirty: false,
  listRevision: 0,
  shortcut: 'space',
  storageError: '',
  locked: false,
})
function restore(owner) {
  try {
    const raw = sessionStorage.getItem(prefix + owner)
    if (!raw) return initial()
    if (raw.length > 1200000)
      throw new Error('Récupération locale trop grande.')
    const s = JSON.parse(raw)
    if (
      s.version !== 1 ||
      s.owner !== owner ||
      !s.drafts ||
      Object.keys(s.drafts).length > 4
    )
      throw new Error('Version de récupération non prise en charge.')
    const drafts = Object.fromEntries(
      Object.entries(s.drafts).map(([key, doc]) => {
        if (
          !/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i.test(key) ||
          doc.client_key !== key ||
          typeof doc.title !== 'string' ||
          doc.title.length > 120 ||
          !['scratchpad', 'workbook'].includes(doc.type) ||
          !Number.isSafeInteger(doc.revision) ||
          doc.revision < 0 ||
          (doc.id !== undefined &&
            (!Number.isSafeInteger(doc.id) || doc.id < 1))
        )
          throw new Error('Métadonnées de récupération invalides.')
        return [key, migrateDocument(doc)]
      }),
    )
    if (s.active !== null && !drafts[s.active])
      throw new Error('Brouillon actif absent.')
    validateContent(s.quick)
    return {
      ...initial(),
      ...s,
      drafts,
      shortcut: ['space', 'l', 'off'].includes(s.shortcut)
        ? s.shortcut
        : 'space',
      locked: false,
      storageError: '',
    }
  } catch (e) {
    return { ...initial(), storageError: e.message, locked: true }
  }
}
export function LaboratoryProvider({ owner, onNavigate, children }) {
  const [state, setState] = useState(() => restore(owner)),
    [open, setOpen] = useState(false)
  useEffect(() => {
    if (state.locked) return
    const persist = () => {
      try {
        sessionStorage.setItem(
          prefix + owner,
          JSON.stringify({
            version: 1,
            owner,
            drafts: state.drafts,
            active: state.active,
            quick: state.quick,
            quickDirty: state.quickDirty,
            shortcut: state.shortcut,
          }),
        )
      } catch {
        setState((s) => ({
          ...s,
          locked: true,
          storageError:
            'Récupération locale indisponible. Le travail reste dans cet onglet ; enregistrez-le sur le serveur.',
        }))
      }
    }
    const timer = setTimeout(persist, 300)
    window.addEventListener('pagehide', persist)
    window.addEventListener('beforeunload', persist)
    return () => {
      clearTimeout(timer)
      window.removeEventListener('pagehide', persist)
      window.removeEventListener('beforeunload', persist)
    }
  }, [
    owner,
    state.drafts,
    state.active,
    state.quick,
    state.quickDirty,
    state.shortcut,
    state.locked,
  ])
  useEffect(() => {
    const key = (e) => {
      if (
        e.defaultPrevented ||
        e.repeat ||
        e.isComposing ||
        state.shortcut === 'off' ||
        !(e.ctrlKey || e.metaKey) ||
        !e.shiftKey ||
        e.altKey
      )
        return
      if (
        e.target instanceof Element &&
        e.target.closest(
          'input,textarea,select,[contenteditable="true"],dialog',
        )
      )
        return
      const matches =
        state.shortcut === 'space' ? e.code === 'Space' : e.code === 'KeyL'
      if (matches) {
        e.preventDefault()
        setOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', key)
    return () => window.removeEventListener('keydown', key)
  }, [state.shortcut])
  useEffect(() => {
    const warning = (e) => {
      if (
        Object.values(state.drafts).some((d) => d.dirty) ||
        (state.quickDirty && state.quick.expression)
      ) {
        e.preventDefault()
        e.returnValue = ''
      }
    }
    window.addEventListener('beforeunload', warning)
    return () => window.removeEventListener('beforeunload', warning)
  }, [state.drafts, state.quick.expression, state.quickDirty])
  const context = useMemo(
    () => ({
      state,
      setState,
      updateQuick: (update) =>
        setState((s) => {
          const quick = update(s.quick)
          const active = s.drafts[s.active]
          const drafts =
            active?.type === 'scratchpad'
              ? {
                  ...s.drafts,
                  [s.active]: { ...active, content: quick, dirty: true },
                }
              : s.drafts
          return { ...s, quick, quickDirty: true, drafts }
        }),
      open: () => setOpen(true),
      close: () => setOpen(false),
      add: (doc) => {
        if (Object.keys(state.drafts).length >= 4)
          throw new Error(
            'Quatre brouillons sont déjà ouverts. Enregistrez puis fermez une feuille avant de continuer.',
          )
        setState((s) => ({
          ...s,
          drafts: { ...s.drafts, [doc.client_key]: doc },
          active: doc.client_key,
        }))
        return doc.client_key
      },
      convert: (expression, result) => {
        const doc = convertCalculation(expression, result, state.quick.history)
        if (Object.keys(state.drafts).length >= 4)
          throw new Error(
            'Quatre brouillons ouverts : fermez une feuille avant conversion.',
          )
        setState((s) => ({
          ...s,
          drafts: { ...s.drafts, [doc.client_key]: doc },
          active: doc.client_key,
        }))
        setOpen(false)
        onNavigate('laboratory')
      },
      forgetRecovery: () => {
        sessionStorage.removeItem(prefix + owner)
        setState((s) => ({ ...s, locked: false, storageError: '' }))
      },
    }),
    [state, owner, onNavigate],
  )
  return (
    <Context.Provider value={context}>
      {children}
      {open && (
        <Suspense fallback={<p role="status">Ouverture du calcul rapide…</p>}>
          <QuickScratchpad modal onClose={() => setOpen(false)} />
        </Suspense>
      )}
    </Context.Provider>
  )
}
export function LaboratoryTrigger() {
  const lab = useLaboratory()
  return (
    <button
      className="btn-secondary laboratory-trigger"
      aria-label="Calcul rapide"
      onClick={lab.open}
      title="Ouvrir le calcul rapide"
    >
      <svg
        aria-hidden="true"
        width="17"
        height="17"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      >
        <rect x="4" y="3" width="16" height="18" rx="2" />
        <path d="M7 7h10M7 12h3m4 0h3M7 17h3m4 0h3" />
      </svg>
    </button>
  )
}
