import { useState, useMemo, useRef, useCallback } from 'react'

// Supprime les accents pour la comparaison insensible
function stripAccents(str) {
  return str.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase()
}

// Trier les lettres par ordre logique : A-Z puis symboles/chiffres à la fin
const SORT_ORDER = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789#'.split('')
function letterSortKey(c) {
  const idx = SORT_ORDER.indexOf(c.toUpperCase())
  return idx === -1 ? 99 : idx
}

/**
 * Navigation alphabétique pour la sélection de tags.
 *
 * Features :
 *  - Recherche instantanée (insensible à la casse,无视 accents)
 *  - Index alphabétique cliquable (lettres actives uniquement)
 *  - Groupement par initiale avec ancres de défilement
 *  - La sélection (pickedTags) est préservée pendant la navigation
 */
export default function TagPickerNavigation({ tags, pickedTags, onToggle }) {
  const [search, setSearch] = useState('')
  const groupRefs = useRef({})

  // Tags filtrés par recherche, triés alphabétiquement
  const filtered = useMemo(() => {
    const q = stripAccents(search)
    const list = q
      ? tags.filter((t) => stripAccents(t).includes(q))
      : [...tags]
    return list.sort((a, b) => stripAccents(a).localeCompare(stripAccents(b), 'fr'))
  }, [tags, search])

  // Groupement par première lettre (utilisée pour l'index ET le rendu)
  const grouped = useMemo(() => {
    const map = new Map()
    for (const t of filtered) {
      const key = stripAccents(t)[0]?.toUpperCase() || '#'
      const realKey = /[A-Z]/.test(key) ? key : '#'
      if (!map.has(realKey)) map.set(realKey, [])
      map.get(realKey).push(t)
    }
    // Trier les clés par ordre logique
    const sorted = [...map.keys()].sort((a, b) => letterSortKey(a) - letterSortKey(b))
    return sorted.map((k) => ({ letter: k, items: map.get(k) }))
  }, [filtered])

  // Lettres disponibles (pour l'index)
  const availableLetters = useMemo(
    () => grouped.map((g) => g.letter),
    [grouped]
  )

  // Détecter la lettre active pendant le scroll
  const [activeLetter, setActiveLetter] = useState(null)
  const handleScroll = useCallback(() => {
    const keys = Object.keys(groupRefs.current)
    if (!keys.length) return
    let current = keys[0]
    for (const k of keys) {
      const el = groupRefs.current[k]
      if (el && el.getBoundingClientRect().top <= 80) current = k
    }
    setActiveLetter((prev) => (prev === current ? prev : current))
  }, [])

  // Scroll vers un groupe
  const scrollTo = useCallback((letter) => {
    const el = groupRefs.current[letter]
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
      setActiveLetter(letter)
    }
  }, [])

  // Nettoyage du search (bouton X)
  const clearSearch = () => setSearch('')

  return (
    <div className="tag-picker-nav">
      {/* Barre de recherche */}
      <div className="tag-picker-search-wrap">
        <span className="tag-picker-search-icon" aria-hidden="true">⌕</span>
        <input
          className="tag-picker-search"
          type="text"
          placeholder="Rechercher un tag…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Rechercher un tag"
        />
        {search && (
          <button
            className="tag-picker-search-clear"
            onClick={clearSearch}
            aria-label="Effacer la recherche"
            title="Effacer"
          >
            ✕
          </button>
        )}
      </div>

      <div className="tag-picker-body">
        {/* Index alphabétique */}
        <nav className="tag-picker-index" aria-label="Index alphabétique">
          {availableLetters.map((letter) => (
            <button
              key={letter}
              className={[
                'tag-picker-index-btn',
                activeLetter === letter ? 'active' : '',
              ]
                .filter(Boolean)
                .join(' ')}
              onClick={() => scrollTo(letter)}
              aria-label={`Aller aux tags commençant par ${letter}`}
            >
              {letter}
            </button>
          ))}
        </nav>

        {/* Liste groupée des tags */}
        <div className="tag-picker-scroll" onScroll={handleScroll}>
          {grouped.length === 0 ? (
            <p className="tag-picker-empty">Aucun tag trouvé.</p>
          ) : (
            grouped.map((group) => (
              <div
                key={group.letter}
                ref={(el) => { groupRefs.current[group.letter] = el }}
                className="tag-picker-group"
                id={`tag-group-${group.letter}`}
              >
                <div className="tag-picker-group-title">{group.letter}</div>
                <div className="tag-picker-list">
                  {group.items.map((t) => (
                    <button
                      key={t}
                      className={[
                        'tag-pick',
                        pickedTags.includes(t) ? 'on' : '',
                      ]
                        .filter(Boolean)
                        .join(' ')}
                      aria-pressed={pickedTags.includes(t)} onClick={() => onToggle(t)}
                    >
                      {t}
                    </button>
                  ))}
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
