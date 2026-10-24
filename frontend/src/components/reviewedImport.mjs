// Le seul chemin vers la confirmation passe par la décision de l'utilisateur.
// Aucun repli vers l'ancien import direct en cas d'erreur ou d'expiration.
export async function reviewedImport({ api, file, accountId, token, review, onPreviewReady, onConfirm }) {
  const preview = await api.previewStatement(file, accountId, token)
  onPreviewReady?.()
  if (!await review(preview)) {
    // Une annulation réseau manquée laisse seulement un aperçu éphémère qui
    // expirera côté serveur ; elle ne doit jamais déclencher d'enregistrement.
    await api.cancelStatementPreview(preview.token).catch(() => {})
    return { cancelled: true, preview }
  }
  onConfirm?.()
  try {
    const result = await api.confirmStatementPreview(preview.token)
    return { cancelled: false, preview, result }
  } catch (error) {
    throw new Error(`La confirmation n’a pas pu être vérifiée : ${error.message}. Consulte les relevés avant de sélectionner à nouveau le fichier.`)
  }
}
