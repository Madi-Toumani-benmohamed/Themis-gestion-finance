import { Component, Suspense, lazy } from 'react'

const modules = {
  dashboard: () => import('./pages/Dashboard'),
  transactions: () => import('./pages/Transactions'),
  statements: () => import('./pages/Statements'),
  advice: () => import('./pages/Advice'),
  budgets: () => import('./pages/Budgets'),
  admin: () => import('./pages/Admin'),
  accounts: () => import('./pages/Accounts'),
  credits: () => import('./pages/Credits'),
  analyse: () => import('./pages/Analyse'),
  fiscal: () => import('./components/FiscalSimulation'),
  laboratory: () => import('./pages/Laboratory'),
  simulate: () => import('./pages/Simulate'),
  assistant: () => import('./pages/Assistant'),
}
const pending = new Map()

export function preloadPage(name) {
  if (!modules[name]) return Promise.resolve()
  if (!pending.has(name)) pending.set(name, modules[name]())
  return pending.get(name)
}

export const Pages = Object.fromEntries(
  Object.keys(modules).map(name => [name, lazy(() => preloadPage(name))]),
)

class PageLoadBoundary extends Component {
  state = { failed: false }
  static getDerivedStateFromError() { return { failed: true } }
  render() {
    if (this.state.failed) return (
      <div className="notice err" role="alert">
        <p>Cette page n’a pas pu être chargée.</p>
        <button type="button" className="btn-secondary" onClick={() => window.location.reload()}>Recharger la page</button>
      </div>
    )
    return this.props.children
  }
}

export function PageLoader({ children }) {
  return (
    <PageLoadBoundary>
      <Suspense fallback={<div role="status" aria-live="polite"><span className="spinner" aria-hidden="true" /> Chargement de la page…</div>}>
        {children}
      </Suspense>
    </PageLoadBoundary>
  )
}
