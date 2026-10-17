// Icônes de navigation — SVG en trait, minimalistes, sans dépendance externe.
// Chaque icône hérite de la couleur courante (stroke="currentColor").

const P = { width: 17, height: 17, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 1.8, strokeLinecap: 'round', strokeLinejoin: 'round' }

export const ICONS = {
  // Tableau de bord — grille de widgets
  dashboard: (
    <svg {...P}><rect x="3" y="3" width="7" height="9" rx="1" /><rect x="14" y="3" width="7" height="5" rx="1" /><rect x="14" y="12" width="7" height="9" rx="1" /><rect x="3" y="16" width="7" height="5" rx="1" /></svg>
  ),
  // Transactions — lignes d'un relevé
  transactions: (
    <svg {...P}><line x1="4" y1="7" x2="20" y2="7" /><line x1="4" y1="12" x2="20" y2="12" /><line x1="4" y1="17" x2="14" y2="17" /></svg>
  ),
  // Analyse — graphe
  analyse: (
    <svg {...P}><path d="M4 19V5" /><path d="M4 15l4-4 4 3 8-8" /><polyline points="16 6 20 6 20 10" /></svg>
  ),
  // Budgets — jauge
  budgets: (
    <svg {...P}><path d="M4 14a8 8 0 0 1 16 0" /><line x1="12" y1="14" x2="16" y2="10" /><line x1="4" y1="18" x2="20" y2="18" /></svg>
  ),
  // Crédits & taxes — billet
  credits: (
    <svg {...P}><rect x="2" y="6" width="20" height="12" rx="2" /><circle cx="12" cy="12" r="2.5" /><line x1="6" y1="12" x2="6" y2="12" /><line x1="18" y1="12" x2="18" y2="12" /></svg>
  ),
  // Simulation — curseurs
  simulate: (
    <svg {...P}><line x1="4" y1="8" x2="20" y2="8" /><line x1="4" y1="16" x2="20" y2="16" /><circle cx="9" cy="8" r="2.2" fill="var(--paper)" /><circle cx="15" cy="16" r="2.2" fill="var(--paper)" /></svg>
  ),
  // Comptes — portefeuille
  accounts: (
    <svg {...P}><rect x="3" y="6" width="18" height="13" rx="2" /><path d="M3 10h18" /><circle cx="16.5" cy="14.5" r="1.2" fill="currentColor" stroke="none" /></svg>
  ),
  // Relevés — document
  statements: (
    <svg {...P}><path d="M6 2h8l4 4v16H6z" /><path d="M14 2v4h4" /><line x1="9" y1="13" x2="15" y2="13" /><line x1="9" y1="17" x2="15" y2="17" /></svg>
  ),
  // Conseils IA — étincelle
  advice: (
    <svg {...P}><path d="M12 3v4M12 17v4M3 12h4M17 12h4" /><path d="M12 8l1.5 2.5L16 12l-2.5 1.5L12 16l-1.5-2.5L8 12l2.5-1.5z" /></svg>
  ),
  // Admin — réglages
  admin: (
    <svg {...P}><circle cx="12" cy="12" r="3" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M19 5l-2 2M7 17l-2 2" /></svg>
  ),
  // Assistant — bulle de dialogue
  assistant: (
    <svg {...P}><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" /></svg>
  ),
}
