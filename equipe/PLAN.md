# Plan d'intervention : bugs navigation temporelle et Top dépenses

## 1. Besoin reformulé

La navigation par mois sur le dashboard ne permet pas de reculer dans le temps : le bouton « mois précédent » ne change pas le mois sélectionné, contrairement au bouton « mois suivant ». Par ailleurs, le widget « Top dépenses du mois » affiche « Pas encore de données » pour les mois historiques alors que les transactions existent et que les autres widgets affichent correctement les données passées.

## 2. Décisions techniques

| Décision | Choix | Raison |
|----------|-------|--------|
| Correction navigation | Corriger le calcul de date dans `MonthSelector.goToPrev` | Le constructeur `Date` attend un mois 0-indexé ; le code soustrait `m-1` au lieu de `m-2`, provoquant un décalage d'un mois |
| Correction Top dépenses | Supprimer la dépendance à `api.summary(1, month)` et utiliser directement `selectedMonth` pour requêter les transactions | Le endpoint `summary` avec `months=1` ne retourne que le mois le plus récent, puis le filtre `month <= maxMonth` vide la série pour les mois historiques |
| Architecture widgets | Garder le pattern existant où les widgets reçoivent `selectedMonth` et utilisent les données du dashboard principal | Évite les requêtes redondantes et garantit la cohérence des données |
| Pas de modification backend | Corriger uniquement côté frontend pour ces deux bugs | La logique de navigation est purement client ; le problème Top dépenses vient de l'usage incorrect de l'API, pas de l'API elle-même |

## 3. Étapes d'intervention

### Étape 1 — Corriger le calcul du mois précédent dans MonthSelector
**Fichiers concernés :** `frontend/src/pages/Dashboard.jsx`

Corriger la fonction `goToPrev` pour utiliser le bon décalage de mois :
- Remplacer `new Date(y, m - 1, 1)` par `new Date(y, m - 2, 1)`
- Le mois provenant de la chaîne `YYYY-MM` est 1-indexé (1-12) alors que `Date` attend 0-indexé (0-11)

**Critère de finition :**
- Cliquer sur « ‹ » depuis Oct 2026 affiche Sep 2026
- Cliquer sur « ‹ » depuis Jan 2026 affiche Dec 2025
- Cliquer sur « › » continue de fonctionner (régression testée)

### Étape 2 — Simplifier TopWidget pour éviter le summary redondant
**Fichiers concernés :** `frontend/src/widgets/index.jsx`

Modifier le widget `TopWidget` :
- Remplacer la requête `api.summary(1, month)` par utilisation directe de `selectedMonth`
- Conserver le chargement des catégories pour filtrer les transferts
- Requêter directement `api.transactions({ month: selectedMonth || latestMonth })`
- Pour déterminer le mois à afficher quand `selectedMonth` est vide, utiliser `data.monthly?.at(-1)?.month` depuis les props si disponibles, sinon se baser sur le mois courant

Alternative plus simple : changer `api.summary(1, month)` en `api.summary(24, month)` pour garantir que le mois historique est inclus dans les séries.

**Critère de finition :**
- Sélectionner un mois historique (ex: Nov 2025) affiche les top dépenses réelles
- Le badge « Mois en cours »/« Incomplet » s'affiche correctement
- Pas de message « Pas encore de données » sur mois avec transactions

### Étape 3 — Vérifier la cohérence avec KpiRow
**Fichiers concernés :** `frontend/src/pages/Dashboard.jsx`

Analyser si `KpiRow` présente le même problème avec `api.summary(2, selectedMonth)` :
- Si le mois sélectionné est plus ancien que 2 mois, la série sera vide
- Évaluer si nécessaire de corriger en passant à `months=24` ou en utilisant les données du dashboard principal

**Critère de finition :**
- Les KPI affichent bien les données pour tout mois historique
- Pas de régression sur le calcul des deltas vs mois précédent

### Étape 4 — Tests de non-régression navigation
**Vérifications manuelles :**

Tester la navigation complète :
- Depuis mois courant, reculer de 12 mois avec la flèche gauche
- Reculer au-delà de janvier → changement d'année correct
- Avancer avec la flèche droite s'arrête bien au mois courant
- Le sélecteur déroulant et le bouton « retour mois en cours » fonctionnent
- Tous les widgets se mettent à jour en synchronie

**Critère de finition :**
- Navigation fluide dans les deux sens sans saut
- Aucun widget ne se désynchronise du mois sélectionné

## 4. Règles à respecter

1. **Ne pas modifier la logique backend** : les bugs sont identifiés comme frontend uniquement
2. **Ne pas casser la navigation future** : le bouton « mois suivant » doit rester bloqué sur le mois courant
3. **Préserver les badges de période** : l'indicateur « Mois en cours »/« Incomplet » doit rester fonctionnel
4. **Ne pas introduire de requêtes supplémentaires** : éviter de doubler les appels API déjà effectués par le dashboard principal
5. **Maintenir la cohérence des données** : tous les widgets doivent afficher les mêmes données pour le même mois sélectionné
6. **Ne jamais supposer la disponibilité des données** : gérer le cas où `selectedMonth` est vide ou où aucune transaction n'existe

## 5. Critères de validation finale

- **Bug 1 résolu** : la flèche « mois précédent » déplace correctement le mois (Oct→Sep, Jan→Dec N-1, etc.)
- **Bug 2 résolu** : le widget Top dépenses affiche les données pour tout mois historique avec transactions
- **Non-régression** : les autres widgets (KPI, catégories, flux) continuent d'afficher correctement les données historiques
- **Performance** : pas d'augmentation notable du nombre de requêtes API
