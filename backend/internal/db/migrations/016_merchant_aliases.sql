-- Surnoms de marchand définis par l'utilisateur. Quand le pipeline automatique
-- ne peut pas deviner le nom (loyer, virements perso, commerçant local),
-- l'utilisateur renomme un marchand une fois ; le surnom s'applique à toutes
-- ses transactions, passées et futures. C'est la source de vérité : il prime
-- sur le dictionnaire et sur toute suggestion IA.
--
-- match_key = clé normalisée du marchand détecté (merchant), OU un fragment du
-- libellé brut si aucun marchand n'a été reconnu. display = le nom choisi.
CREATE TABLE IF NOT EXISTS merchant_aliases (
    id         BIGSERIAL PRIMARY KEY,
    owner_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    match_key  TEXT NOT NULL,
    display    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_merchant_aliases_owner_key ON merchant_aliases(owner_id, lower(match_key));
