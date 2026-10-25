# Roadmap — intelligence financière de Thémis

Les lots sont ordonnés selon le risque sur les données et l’impact produit. Les estimations sont relatives à l’architecture actuelle : XS (≤1 jour), S (1–3 jours), M (3–7 jours), L (>1 semaine), hors revue produit et livraison.

## P0 — Corriger les incohérences existantes

| Proposition | Impact utilisateur et justification | Composants / dépendances | Risques | Tests et critères d’acceptation | Complexité |
|---|---|---|---|---|---|
| Normaliser retraits DAB et libellés variables | Les heures ne fragmentent plus un même type d’opération ; brut et occurrences restent traçables. Défaut confirmé dans `normalize.Label`. | `internal/normalize`, backfill versionné, import, `recurring_review`. Aucun schéma nouveau. | Un motif trop large pourrait classifier un marchand contenant « ATM » ou « retrait ». Le motif doit exiger un préfixe opérationnel explicite. Backfill touche seulement colonnes de présentation/type. | Tests de variantes `Ret Distrib 14h23`, `Ret DAB X… 03/10 10H21`; nom affiché commun, kind opérationnel, merchant vide, données brutes distinctes. Tests UTF‑8 et heures sur marchands. | S |
| Rendre la propagation d’alias atomique | Le renommage affiché ne peut plus réussir à moitié ni ignorer une erreur SQL. | `merchant_alias.go`, table existante, tests PostgreSQL isolés. | Verrouiller ensemble upsert et propagation ; ne jamais étendre les patterns sans consentement explicite. | Transaction unique alias+update ; rollback intégral sur erreur ; sélection cohérente avec `aliasFor`; owner isolation; exact/pattern, valeurs brutes inchangées. | S |
| Résoudre les motifs trop larges | Les deux premiers mots ne suffisent pas pour fusionner des identités. | `MerchantRenamer.jsx`, API alias et UX de confirmation. | Changement UX, alias existants conservés ; aucune migration destructive. | Prévisualiser le nombre/exemples concernés avant confirmation; alias exact par défaut; exceptions possibles ou annulation; jamais de matching approximatif silencieux. | M |
| Valider strictement les suggestions marchand IA | Éviter qu’une sortie malformée ou une clé étrangère à la demande devienne une suggestion crédible. | `merchant_alias.go`, prompt, tests parser, interface existante. | Un filtrage prudent peut réduire le nombre de suggestions. | Limiter les clés à celles demandées et au propriétaire, noms bornés/non vides, erreur de fournisseur sans contenu sensible; aucune écriture sans confirmation. | S |

Lot livré dans cette mission : normalisation déterministe des retraits, heure et Unicode ; application atomique des alias, erreurs propagées et conservation du mode exact/motif pendant la correction manuelle. La prévisualisation des alias motif et la validation stricte des suggestions IA restent planifiées car elles nécessitent une évolution UX/contrat au-delà du correctif local.

## P1 — Exploiter les tags et fiabiliser l’analyse existante

| Proposition | Impact utilisateur et justification | Composants / dépendances | Risques | Tests et critères d’acceptation | Complexité |
|---|---|---|---|---|---|
| Clarifier les règles de tag et leurs exceptions | Une règle de catégorie ne doit pas réintroduire automatiquement un tag qu’un utilisateur a retiré sur une opération particulière. Aujourd’hui, la réapplication ajoute les tags des règles à toutes les correspondances. | Moteur de règles, `transaction_tags`, PATCH transactions ; provenance ou exclusion à concevoir. | Modification d’un comportement établi ; migration additive probablement nécessaire pour provenance ou exceptions. Préparer une proposition avant schéma. | Cas manuel retiré / règle réappliquée, tag manuel conservé, import neuf, portée propriétaire. Aucun effacement d’historique. | M |
| Aligner l’audit de cohérence sur `CompilePattern` | Les diagnostics doivent utiliser le même sens des jokers et regex que l’application réelle des règles. | `coherence.go`, package `categorize`, extraction des candidats owner-scoped. | Matcher regex en mémoire augmente le coût ; borner et paginer l’ensemble. | `%`, `_`, regex valides/invalides, même résultat entre audit et réapplication. | S |
| Propager couverture et précision dans tous les widgets | Le Dashboard doit clairement séparer période courante, données incomplètes et historique confirmé ; remplacer les deltas `Number` par le modèle exact partagé. | Widgets, modèles frontend, `/api/summary`, métadonnées reliability. | Contrat API additif ; états visuels existants. | Mois en cours avant salaire/relevé, absence ≠ zéro, sommes exactes, valeurs non observées explicites; non-régression layouts. | M |
| Ajouter provenance des opérations aux récits, KG compris | Une correction d’opération ne doit pas laisser un fait de mémoire apparaître comme vérité actuelle. | API import/correction, `memory`, chat, éventuellement révocation d’épisodes. | Service externe, coût et rétention ; KG doit rester optionnel. | Faits synthétiques, mises à jour et échecs KG, isolation propriétaire; analyses utilisables sans KG. | L |

## P2 — Détection d’habitudes et moteur de découvertes

| Proposition | Impact utilisateur et justification | Composants / dépendances | Risques | Tests et critères d’acceptation | Complexité |
|---|---|---|---|---|---|
| Ajouter des faits d’habitude par jour de semaine et contexte | Répondre à « où/quand va mon argent » avec explication navigable. | `analysis_discoveries.go`, rapports par date, tags/catégories; modèle de découverte sourcé. | Biais de périodes incomplètes et échantillons insuffisants ; aucun score de confiance non calibré. | Minimum d’observations documenté, comparaison de jours réellement couverts, montants/signaux exacts, transaction IDs, tests avec mois partiel et exceptions. | M |
| Séparer abonnement, charge récurrente, habitude et transaction unique | Éviter de présenter une cantine fréquente comme abonnement. | `recurring_review.go`, décisions utilisateur déjà en place, catégories/tags contextuels. | Classification commerciale erronée ; nature proposée, jamais affirmée sans preuve contractuelle. | Séries hebdomadaires restent habitudes; cadence mensuelle répétée peut être récurrente sans garantie de contrat; remboursements et virements séparés. | M |
| Persister l’historique des découvertes présentées | Réduire les répétitions et recalculer après correction/import nouveau. | Nouveau stockage owner-scoped à proposer, empreinte de règle + preuves, invalidation. | Données analytiques dérivées et concurrence ; aucune suppression destructive. | Idempotence, preuves sources, correction source invalide signal, propriétaires isolés, aucun remplissage artificiel. | L |

## P3 — Simulations et personnalisation

| Proposition | Impact utilisateur et justification | Composants / dépendances | Risques | Tests et critères d’acceptation | Complexité |
|---|---|---|---|---|---|
| Scénarios de réduction par tag/marchand | Simuler l’effet d’un changement de comportement à partir d’une base historique explicite. | `simulation.go`, UI simulateur, filtres par tags/marchands, rapport de couverture. | Tags chevauchants et données partiellement documentées ; ne pas additionner des segments recouvrants. | Hypothèses utilisateur distinctes de l’historique; exclusions documentées; exactitude centime et limites visibles. | M |
| Scénarios de remboursement et dette | Comparer remboursement et intérêts sous hypothèses. | moteur dette/crédits existant, historique des taux/échéances si disponible. | Taux/frais/assurance incomplets, pas de prévision garantie. | Entrées explicites, sensibilité et limites, tests taux nul/positif/arrondis. | M |
| Personnaliser les découvertes | Favoriser types de constats souhaités sans juger les choix volontaires. | Préférences utilisateur, exposition des preuves, historique P2. | Sur-personnalisation ou dissimulation d’alertes critiques. | Préférence n’écrase pas les contrôles financiers; raison visible; possibilité de désactiver les constats exploratoires. | M |

## Ordre recommandé

1. Stabiliser la normalisation et l’application d’alias; ajouter les cas de retrait sans fusion d’opérations.
2. Préparer séparément une proposition de modèle provenance/exceptions des tags avant migration.
3. Faire converger les contrôles de règles et l’état qualité du Dashboard vers les composants d’analyse déjà corrigés.
4. Étendre les découvertes uniquement avec couverture documentée et preuves navigables.
5. Ajouter les nouveaux scénarios après validation du modèle de tags, sans traiter des regroupements multitag comme des parts disjointes.

Chaque lot conserve les transactions brutes, les montants/date/source, la validation propriétaire et les options sans IA. Aucun travail de roadmap ne suppose qu’une fréquence établit une intention, qu’un relevé équilibré est exhaustif, ni qu’un scénario est une prévision.
