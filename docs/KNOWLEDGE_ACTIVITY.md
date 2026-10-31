# Vérifier la mémoire de Thémis

## Accès et vérification

Ouvrir **Mémoire**, rubrique Sources, ou **Admin → Configuration IA → Mémoire**. L’administrateur voit son propre graphe, pas celui des autres comptes.

1. **Actualiser l’activité** lit réellement FalkorDB et affiche entités, relations et épisodes enregistrés. Un zéro est une absence confirmée ; « Non mesuré » indique un service désactivé, une erreur ou un contrôle indisponible. Ce contrôle ne sollicite aucun modèle et ne crée pas de connaissances.
2. **Explorer les relations réelles** montre un échantillon sélectionnable au clavier et à la souris. Les listes donnent accès aux entités et relations de l’échantillon ; les faits détaillent dates et sources disponibles. Le dessin reste volontairement borné, les compteurs portent sur le graphe complet du compte. Sur mobile, le dessin défile horizontalement.
3. **Tester la recherche** avec un sujet déjà présent vérifie le parcours de recherche, y compris les modèles nécessaires. Cela ne crée aucun fait ; une trace est conservée dans le journal. Aucun résultat et une erreur sont présentés séparément. Une lecture FalkorDB réussie ne garantit pas que les modèles fonctionnent.
4. Après un nouvel import confirmé et réconcilié, consulter le **journal** puis actualiser. L’apprentissage est asynchrone. Les nombres d’entités/relations traitées ne sont pas des créations nettes : Graphiti peut retrouver des éléments existants. Un import non réconcilié est indiqué comme non envoyé.
5. Dans les assistants complet et flottant, déplier la trace des **faits joints à la requête IA**. Elle décrit le contexte transmis, pas une preuve que le modèle l’a utilisé dans sa réponse. Les montants se vérifient toujours sur les transactions bancaires.

Le journal commence avec cette version. Les faits anciens restent visibles dans FalkorDB mais aucun ancien apprentissage n’est inventé. Aucun réimport ou réapprentissage automatique des anciens relevés n’est déclenché. Une expiration lors d’une écriture donne un résultat non confirmé : le traitement distant peut encore se terminer. Une entrée sans résultat depuis plus de trois minutes indique une interruption possible.

## Déploiement sur la VM

Depuis la copie du dépôt mise à jour, reconstruire **l’app et le service Python**. Pour le Compose principal :

```sh
docker compose build app knowledge-graph
docker compose up -d app knowledge-graph
```

Pour la variante Supabase, conserver son fichier Compose et son environnement habituels. Pour un déploiement par images/Portainer, reconstruire et recréer les deux services correspondants. Ne pas supprimer les volumes PostgreSQL/FalkorDB.

La migration additive 033 s’applique au démarrage : journal propriétaire, index et références composées empêchant de rattacher un relevé ou une conversation d’un autre compte. Le rollback refuse toute perte d’historique. Les exports propriétaires incluent le journal.

Les deux Compose limitent le port Python 8100 à `127.0.0.1`. L’accès à distance passe par l’API Go authentifiée ; le réseau Docker conserve l’accès interne au service. Ne pas publier directement le service Python, qui expose des faits privés. Les modèles LLM et embedding doivent être disponibles auprès de l’Ollama configuré ; les erreurs du fournisseur sont expurgées et distinguées.

## Contrats et limites

- `GET /api/knowledge/overview?before_id=…` : état courant, dernier apprentissage indépendant de la pagination et 50 événements par page.
- `POST /api/knowledge/search` avec `{"query":"…"}` : jusqu’à cinq faits du compte authentifié. Aucun propriétaire n’est accepté dans le corps.
- `POST /api/knowledge/learn` conserve sa fonction, avec validation et journalisation explicites. Le panneau ne l’appelle pas à son ouverture.
- Inspection Python : groupe numérique canonique ; requêtes Cypher statiques et paramétrées, lecture seule ; maximum 110 entités, 40 relations et 200 sources. Le dessin affiche au plus 16 entités.
- Les scores de pertinence absents sont `null`, jamais inventés. Les textes bornés sont signalés comme extraits. Les relations historiques invalidées/expirées restent identifiées.
- Apprentissage : lots de 100 faits, durée totale bornée à deux minutes ; traitement partiel et résultat incertain distincts d’un succès complet.
- L’historique peut contenir des faits financiers privés : il suit les autorisations et suppressions des références propriétaires. Aucune politique automatique de rétention n’est ajoutée.

## Validation reproductible

Tests Go avec PostgreSQL dédié : `THEMIS_TEST_DATABASE_URL=… go test -race ./...` puis `go vet ./...` depuis backend. Tests Python : installer `memory/requirements.txt`, puis `PYTHONDONTWRITEBYTECODE=1 python -m unittest discover -s tests -v`. Le test FalkorDB réel est optionnel via `THEMIS_KG_FALKOR_TEST_PORT`, sur une instance **jetable réservée aux tests** : les groupes synthétiques 90001/90002 sont créés puis supprimés. Ne jamais pointer ce test vers la production.

Frontend : tests Node, `npm run build`, `npm run test:browser` et `npm run test:e2e` avec une base exclusivement nommée `themis_e2e`. Les tests couvrent erreurs, zéro/inconnu, groupes propriétaires, provenance, recherche, journal, réponses obsolètes, assistant et Axe clair/sombre sur mobile/tablette/desktop.

Les essais locaux utilisent des données synthétiques. Ils ne valident pas la configuration Ollama ni le contenu de la VM de production, qui restent à contrôler après déploiement.

Les références visuelles Budgets utilisent un mois historique clôturé et vérifient les bornes des réponses API avant capture : figer seulement l’horloge du navigateur ne fige pas la date du serveur Go. Le seuil de comparaison reste inchangé.
