export function ResourceState({ resource, label }) {
  if (resource.loading) return <p role="status">Chargement {label}…</p>
  if (resource.error)
    return (
      <p role="alert">
        {resource.error}{' '}
        <button className="btn-secondary" onClick={resource.retry}>
          Réessayer
        </button>
      </p>
    )
  return null
}
export function Pagination({ offset, step, more, setOffset }) {
  return (
    <div className="project-actions">
      {offset > 0 && (
        <button
          className="btn-secondary"
          onClick={() => setOffset(offset - step)}
        >
          Précédentes
        </button>
      )}
      {more && (
        <button
          className="btn-secondary"
          onClick={() => setOffset(offset + step)}
        >
          Suivantes
        </button>
      )}
    </div>
  )
}
