import { useState, useEffect } from 'react'
import { api } from '../api'
import { Narrative, Discoveries, NatureBreakdown, CategoryComparison, MerchantsView } from './Intelligence'
import TagFolders from './TagFolders'
import CategoryChart from './CategoryChart'
import TagAnalysis from './TagAnalysis'
import TagManager from './TagManager'

// Onglet Analyse : beaucoup de sections, donc une sous-navigation à gauche
// plutôt qu'une longue page à faire défiler. Chaque entrée affiche une section.
const SECTIONS = [
  { id: 'synthese', label: 'Synthèse' },
  { id: 'reperage', label: 'Ce que thémis a repéré' },
  { id: 'nature', label: 'Où va ton argent' },
  { id: 'comparaison', label: "Ce mois-ci vs d'habitude" },
  { id: 'marchands', label: 'Chez qui tu dépenses' },
  { id: 'projets', label: 'Budgets projet' },
  { id: 'categories', label: 'Dépenses par catégorie' },
  { id: 'tags', label: 'Analyse par tag' },
  { id: 'gerer-tags', label: 'Gérer les tags' },
]

export default function Analyse({ onNavigate }) {
  const [section, setSection] = useState('synthese')

  return (
    <div className="analyse-layout">
      <nav className="analyse-nav" aria-label="Sections d'analyse">
        {SECTIONS.map((s) => (
          <button
            key={s.id}
            className={`analyse-nav-item ${section === s.id ? 'active' : ''}`}
            onClick={() => setSection(s.id)}
          >
            {s.label}
          </button>
        ))}
      </nav>

      <div className="analyse-content">
        {section === 'synthese' && <Narrative />}
        {section === 'reperage' && <Discoveries onNavigate={onNavigate} />}
        {section === 'nature' && <NatureBreakdown />}
        {section === 'comparaison' && <CategoryComparison />}
        {section === 'marchands' && <MerchantsView />}
        {section === 'projets' && <TagFolders onNavigate={onNavigate} />}
        {section === 'categories' && <CategoryChart />}
        {section === 'tags' && <TagAnalysisStandalone />}
        {section === 'gerer-tags' && <TagManager />}
      </div>
    </div>
  )
}

// TagAnalysis a besoin de la liste des catégories ; on la charge ici pour la
// rendre autonome dans sa propre section.
function TagAnalysisStandalone() {
  const [cats, setCats] = useState([])
  useEffect(() => { api.categories().then(setCats).catch(() => setCats([])) }, [])
  return <TagAnalysis categories={cats} />
}
