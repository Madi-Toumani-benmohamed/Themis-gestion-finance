# Instructions des agents — Thémis

## 1. Contexte du projet

Thémis est une application de gestion financière personnelle.

Architecture principale :
- Backend : Go, API REST, PostgreSQL.
- Frontend : React, Vite, Recharts.
- Infrastructure : Docker, GitHub Actions.
- Services optionnels : assistant IA, Ollama, Knowledge Graph.

Avant toute modification importante :
- Consulter `CODEBASE_GRAPH.md` s'il existe.
- Examiner le code concerné et ses dépendances.
- Identifier les conventions et tests existants.
- Ne jamais supposer qu'une fonctionnalité est isolée sans vérifier ses dépendances.

**Priorité du projet : fiabilité, cohérence financière, sécurité, performance, puis nouvelles fonctionnalités.**

## 2. Principes fondamentaux

- Préserver la confidentialité des données personnelles et bancaires.
- Respecter l'architecture existante.
- Privilégier les modifications ciblées, lisibles et maintenables.
- Éviter les refactorisations non sollicitées.
- Ne jamais inventer de données financières.
- Ne pas masquer les erreurs par des valeurs fictives.
- Distinguer les données absentes, incomplètes, invalides et réellement nulles.
- Vérifier les impacts sur les autres composants.
- Ne pas supprimer une fonctionnalité existante pour simplifier une implémentation.

Ne jamais privilégier une amélioration esthétique au détriment de l'exactitude des informations.

## 3. Organisation du travail — Codex et OpenCode

Le projet peut être développé alternativement avec Codex et OpenCode.

Les deux agents peuvent intervenir sur des fonctionnalités différentes ou reprendre un travail existant.

### Règles Git

- Une branche Git par fonctionnalité ou correction significative.
- Privilégier un `git worktree` distinct par fonctionnalité lorsque plusieurs chantiers sont ouverts.
- Vérifier la branche courante et l'état du dépôt avant toute modification.
- Ne jamais écraser les changements non commités.
- Ne jamais supprimer ou modifier les travaux d'un autre agent sans justification.
- Ne jamais changer de branche automatiquement si cela risque d'affecter le travail en cours.
- Ne jamais effectuer de `git reset --hard`, `git clean -fd`, `git push --force` ou opération destructive sans autorisation explicite.
- Ne jamais merger directement dans `main`.
- Ne pas créer de commit ou pousser des modifications sans demande explicite de l'utilisateur.
- Ne pas modifier les fichiers hors périmètre sans nécessité technique démontrée.

### Reprise d'un travail existant

Lorsqu'un agent reprend une fonctionnalité :

1. Vérifier `git status` et la branche active.
2. Examiner les différences existantes avec `git diff`.
3. Identifier les fichiers non suivis pertinents.
4. Comprendre les modifications déjà réalisées.
5. Préserver les décisions techniques et le travail existant.
6. Continuer uniquement après avoir identifié le périmètre réel.

Ne jamais considérer qu'un répertoire de travail est propre sans vérification.

### Fichiers partagés

Porter une attention particulière aux modifications concernant :
- Les endpoints et contrats API.
- Les modèles financiers.
- Les composants React réutilisables.
- Les styles globaux.
- La configuration Docker.
- Les migrations PostgreSQL.
- Les workflows GitHub Actions.

Signaler explicitement toute modification d'un fichier partagé et ses impacts potentiels.

## 4. Fiabilité des données financières

La fiabilité des données est une exigence non négociable.

### Exactitude

- Les montants affichés doivent correspondre aux données réellement disponibles.
- Aucun montant ne doit être inventé ou estimé sans indication explicite.
- Éviter les doubles comptages.
- Vérifier la cohérence entre transactions, budgets, soldes et agrégations.
- Respecter les règles d'arrondi monétaire.
- Préserver les conventions de signe des revenus et dépenses.
- Ne jamais remplacer silencieusement une donnée manquante par zéro.

### Gestion temporelle

- Distinguer les périodes complètes et incomplètes.
- Ne pas présenter un mois en cours comme un bilan définitif.
- Respecter le mois sélectionné dans tous les composants concernés.
- Éviter de comparer directement un mois partiel à un mois complet sans contextualisation.
- Prendre en compte la date de disponibilité des données bancaires.

Contexte métier :
- Le salaire est généralement versé en fin de mois.
- Les relevés bancaires sont disponibles mensuellement, autour du 21.
- Un déséquilibre temporaire entre revenus et dépenses ne constitue pas automatiquement une anomalie.

Les indicateurs financiers doivent refléter ces contraintes sans masquer les véritables anomalies.

### Traçabilité

Toute transformation financière significative doit être :
- Compréhensible.
- Déterministe lorsque possible.
- Testable.
- Cohérente entre backend et frontend.

## 5. Sécurité et confidentialité

- Ne jamais exposer de secrets, tokens ou identifiants dans le code ou les logs.
- Ne jamais utiliser de données bancaires réelles dans les tests.
- Utiliser des données synthétiques pour les tests et démonstrations.
- Vérifier l'authentification et l'autorisation des endpoints concernés.
- Respecter l'isolation des données entre utilisateurs.
- Valider les entrées côté backend.
- Utiliser des requêtes SQL paramétrées.
- Éviter de journaliser les données sensibles.
- Ne pas désactiver les contrôles de sécurité pour contourner un problème.

Toute modification touchant l'authentification, les permissions ou les données financières doit faire l'objet d'une attention renforcée.

## 6. Développement

Avant modification :
1. Comprendre le comportement actuel.
2. Identifier le problème ou l'objectif.
3. Localiser les fichiers concernés.
4. Analyser les dépendances.
5. Vérifier les tests existants.

Pendant modification :
- Respecter les conventions du dépôt.
- Préférer les solutions simples et maintenables.
- Réutiliser les composants et fonctions existants lorsque pertinent.
- Éviter l'ajout de dépendances inutiles.
- Ne pas modifier les interfaces publiques sans vérifier leurs consommateurs.
- Conserver la compatibilité avec les fonctionnalités existantes.

Après modification :
- Examiner le diff.
- Supprimer le code devenu inutile uniquement s'il relève du périmètre.
- Vérifier l'absence de régression évidente.
- Ajouter ou adapter les tests nécessaires.

## 7. Tests et prévention des régressions

Une modification n'est pas considérée comme terminée uniquement parce qu'elle compile.

### Backend

Selon les fichiers modifiés :
- Exécuter les tests Go concernés.
- Vérifier les calculs et règles métier.
- Tester les cas limites.
- Vérifier les autorisations et traitements d'erreurs.

Commandes de référence :

```bash
cd backend
go test ./...
go vet ./...
```

### Frontend

Selon les fichiers modifiés :
- Exécuter les tests disponibles.
- Vérifier les états chargement, erreur et données absentes.
- Vérifier le mois sélectionné.
- Contrôler les interactions et la cohérence des informations.
- Vérifier les régressions sur les composants partagés.

Commande de référence :

```bash
cd frontend
npm run build
```

### CI/CD

- Ne jamais désactiver un test pour rendre la CI verte.
- Ne jamais ignorer silencieusement un échec.
- Distinguer les erreurs introduites des erreurs préexistantes.
- Ne jamais affirmer que les tests passent sans les avoir exécutés.
- Vérifier les workflows concernés lorsque leur configuration est modifiée.

En cas d'échec, signaler précisément la commande, le résultat et la cause identifiée ou supposée.

## 8. Interface et expérience utilisateur

L'interface de Thémis doit être claire, cohérente et sobre.

Principes :
- Privilégier la lisibilité des données financières.
- Réduire la charge cognitive.
- Éviter les indicateurs redondants.
- Préserver la cohérence visuelle entre les pages.
- Maintenir une hiérarchie claire de l'information.
- Éviter les longs défilements lorsqu'une meilleure organisation est possible.
- Préserver l'accessibilité et le responsive design.
- Ne pas ajouter de graphiques dépourvus de valeur informative.

L'interface doit faciliter la compréhension, pas simplement présenter davantage de données.

## 9. Gestion des changements importants

Avant une modification importante, présenter brièvement :
- Le problème identifié.
- La solution envisagée.
- Les fichiers ou composants potentiellement concernés.
- Les risques de régression.

Privilégier les modifications progressives et vérifiables.

Demander une validation explicite avant :
- Une migration destructive de base de données.
- La suppression d'une fonctionnalité.
- Une refonte architecturale majeure.
- Une modification significative de l'authentification.
- L'ajout d'un service externe ou d'une dépendance structurante.
- Une opération Git destructive.

## 10. Compte rendu obligatoire

À la fin de chaque intervention, présenter :

### Modifications
- Fonctionnalités corrigées ou améliorées.
- Fichiers principaux modifiés.

### Validation
- Tests exécutés et résultats.
- Build exécuté et résultat.
- Vérifications non effectuées, le cas échéant.

### Risques
- Régressions potentielles.
- Limitations connues.
- Points restant à vérifier.

### Git
- Branche de travail.
- État des modifications.
- Éventuels conflits ou impacts sur d'autres fonctionnalités.

Ne jamais déclarer une tâche terminée si des validations essentielles restent en échec.

## 11. Règle de priorité

En cas de conflit entre plusieurs objectifs, appliquer l'ordre suivant :

1. Sécurité et confidentialité.
2. Intégrité et exactitude des données financières.
3. Préservation des fonctionnalités existantes.
4. Tests et prévention des régressions.
5. Maintenabilité et performance.
6. Ergonomie et présentation.
7. Nouvelles fonctionnalités.

**Objectif final : améliorer Thémis sans dégrader sa fiabilité ni compromettre les travaux existants.**