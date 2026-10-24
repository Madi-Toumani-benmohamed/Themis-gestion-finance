# Transactions — audit et plan de fiabilisation

Audit du code effectué le 24 octobre 2026, sans consultation des données bancaires réelles. Les exemples de test sont isolés. Référence de départ : `a38d6e1` (main).

## Diagnostic vérifié avant modifications

| Priorité | Anomalie constatée et cause | Correction prévue |
|---|---|---|
| P0 | `normalize_backfill.go` charge les surnoms sans propriétaire et les applique à toutes les transactions : un surnom privé peut passer d'un utilisateur à l'autre. | Isoler les clés par propriétaire, tester deux utilisateurs et corriger la clé de version du normaliseur. |
| P1 | `PATCH /transactions/{id}` ignore les erreurs JSON, ne vérifie pas le propriétaire de la catégorie et sépare catégorie/tags en deux transactions SQL. | Validation stricte, catégorie autorisée et écriture atomique avec rollback. |
| P1 | La liste est limitée à 1 000 lignes, sans total complet ni synthèse du périmètre. Les tags sont joints sans filtre de propriétaire. | Rapport optionnel rétrocompatible, filtres communs, agrégats exacts et lignes paginées lus dans un snapshot, tags isolés. |
| P1 | Aucun filtre compte/type, recherche incomplète (marchand et montant), dates contrôlées seulement par regex et mois filtré par `to_char`. | Filtres serveur et dates calendaires strictes, bornes SQL indexables. |
| P1 | `ai/analyze.go` et les séries par catégorie n'excluent pas les enfants des catégories de transfert ni `label_kind=savings`, contrairement aux budgets/simulations/synthèses. | Prédicat partagé entre les services concernés et tests de réconciliation. |
| P1 | La provenance de catégorisation n'est pas conservée. L'enrichissement IA peut poser des catégories et tags dès l'import. | Métadonnée additive avec historique « inconnu » ; suggestions distinctes et validation explicite avant application. |
| P1 | Le résultat d'orchestration du chat annonce systématiquement une qualité « complète » même lorsque la couverture n'est pas établie. | Ne restituer que la qualité vérifiée par les outils. |
| P2 | L'application rétroactive des règles utilise `LIKE`, tandis que l'import utilise des sous-chaînes littérales ou des regex `re:`. | Réutiliser la même sémantique, vérifier le domaine cible. |
| P2 | Le bouton export ne respecte que le mois et peut exporter un périmètre différent des filtres. | Exporter le périmètre réellement filtré ou annoncer explicitement sa portée. |
| P2 | Les outils de gestion occupent la première zone de la page ; absence de titre, synthèse, reset, pagination générale et détail. Date répétée, édition tags non accessible au clavier. | Outils secondaires repliables, synthèse compacte, filtres denses, groupes journaliers, drawer natif. |

## Éléments existants à préserver

- Navigation depuis budgets et récurrences avec mois, plage de dates, budget ou liste d'identifiants sources.
- Libellé bancaire brut séparé du libellé d'affichage et du marchand normalisé.
- Édition catégorie/tags, tri en pile, règles, gestion des catégories et surnoms marchands.
- Imports atomiques, verrou par propriétaire, empreintes isolées par utilisateur et refus des collisions ambiguës dans un import ou entre comptes.
- Classification des transferts via domaine (y compris parent) et `label_kind=savings` ; couverture documentaire existante, indépendante de la dernière transaction.

## Plan par lots

1. Contrats et sécurité : rapport optionnel sur le GET existant, métadonnée de classification, filtres communs, édition atomique, isolation des surnoms et tests PostgreSQL.
2. UX essentielle : synthèse du périmètre, filtres et pagination, tableau dense et panneau de détail avec édition persistée et gestion des erreurs.
3. Cohérence : règles déterministes partagées, provenance, suggestions IA contrôlées, alignement des exclusions analytiques.
4. Gestion avancée : sélection limitée à la page, confirmation des modifications multiples, résultats partiels explicites. Aucun rapprochement ni doublon certifié sur une ressemblance.
5. Validation : tests Go avec PostgreSQL isolé, race/vet, tests de modèles frontend, parcours navigateur, accessibilité, tailles d'écran et builds. Les résultats réels sont consignés en fin de mission.

## Invariants et limites établis

- Les sommes sont calculées en SQL `NUMERIC` puis représentées en centimes entiers ou décimaux exacts. Aucune somme monétaire en virgule flottante côté interface.
- Dépenses = somme positive des débits ordinaires observés ; revenus = crédits ordinaires observés, **dont la nature revenu/remboursement/annulation n'est pas toujours connue** ; flux net = crédits moins débits. Les transferts sont isolés. Ce flux n'est pas un solde bancaire.
- Les remboursements restent des crédits distincts : sans lien de rapprochement validé, aucune compensation automatique du budget ni suppression d'écriture.
- Une période sans opérations n'établit ni un revenu nul réel ni une couverture complète. Le mois courant est partiel ; les autres périodes ne sont complètes qu'avec preuve documentaire.
- Une date de transaction n'est pas une date de comptabilisation : le modèle ne conserve actuellement pas deux dates distinctes. Ne pas inventer la seconde.
- L'empreinte historique `date + libellé + montant` ne distingue pas deux opérations réellement distinctes strictement identiques sur le même compte dans deux imports séparés. Aucun nettoyage rétroactif n'est autorisé par cet audit.
- Une paire −22 €/+22 €, PayPal ou le tag `taxes` ne prouve ni annulation, ni nature économique, ni catégorie correcte. Aucune recatégorisation de ces exemples n'est automatisée.
- Les données issues du Knowledge Graph ne remplacent pas les transactions PostgreSQL. Les suggestions IA doivent rester des propositions jusqu'à validation explicite.
