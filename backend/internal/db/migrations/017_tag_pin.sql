-- Choix manuel de l'utilisateur sur le statut « dossier » d'un tag, par-dessus
-- la détection automatique. state : 'pinned' (toujours dossier) ou 'hidden'
-- (jamais dossier). Un tag sans entrée suit la détection auto.
CREATE TABLE IF NOT EXISTS tag_folder_prefs (
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_name TEXT   NOT NULL,
    state    TEXT   NOT NULL CHECK (state IN ('pinned','hidden')),
    PRIMARY KEY (owner_id, tag_name)
);
