# Activité du Knowledge Graph — progression

Branche `feat/knowledge-graph-activity`, base `ad2cf290738d12e3393f6e66ab18dce1864c7bbf`. Chantier demandé après le correctif HTTP/Impôts PR48 (fusionné, CI verte). État initial propre ; aucune donnée réelle utilisée en test.

## Audit et choix

Le contrôle Admin existant consulte uniquement /health. Learn/Query absorbaient des échecs comme des résultats vides ; le compteur Python utilisait un graphe `graphiti` alors que Graphiti/Falkor sélectionne le groupe utilisateur. EntityEdge expose `name`, pas `relationship`, et search ne donne pas de score : correction sans inventer de pertinence. Le reranker reçoit désormais un transport OpenAI compatible, pas un OpenAIGenericClient qui n’expose pas .chat.

Le panneau lit un journal PostgreSQL propriétaire et un aperçu borné de FalkorDB. Il ne crée aucun fait de démonstration. Un graphe inexistant est confirmé par GRAPH.LIST sans création implicite ; les lectures passent par GRAPH.RO_QUERY. Ne pas cloner FalkorDriver pour inspecter : son constructeur programme une création d’index. La version Graphiti est fixée à0.31.2 après lecture des contrats du package.

## Implémenté

- Migration033 additive `knowledge_activity`, historique propriétaire des apprentissages, recherches et contextes assistant : états, sources, compteurs mesurés/nulls, faits bornés, durée, erreurs expurgées, références aux relevés/conversations avec suppression en cascade. Rollback gardé.
- Apprentissage après confirmation d’import réconcilié, asynchrone, par lots et borné2minutes ; imports non réconciliés marqués non envoyés. Une expiration ne prétend pas que l’écriture distante a échoué : résultat non confirmé. Pas de reprise automatique ni de reconstruction d’un historique fictif.
- API authentifiée `/api/knowledge/overview`, `/api/knowledge/search`, journal paginé par identifiant, dernier apprentissage indépendant de la page, inspection du compte connecté et refus des résultats d’un autre groupe. Export propriétaire complété. Aucune requête Cypher fournie par le navigateur.
- UI `KnowledgeActivity`, `KnowledgeFacts`, modèles/validations : entrée Mémoire, intégration Admin/Mémoire, compteurs réels, graphe SVG borné avec sélection clavier, faits et sources, test de recherche, journal, états vides/erreurs/indisponibles distincts.
- Trace des faits joints à la requête dans l’assistant complet et flottant. Leur présence ne prouve pas l’utilisation par le modèle ; les montants restent à vérifier via les données bancaires.
- Port8100 lié à127.0.0.1 dans les deux Compose : les faits ne sont pas exposés directement sur l’IP publique ; l’app passe par l’API Go authentifiée. Le service reste accessible sur le réseau Docker.
- CI Python installe les dépendances déjà utilisées avant les tests de contrats FastAPI/Graphiti. Aucun nouveau service ou bibliothèque frontend ajouté.

## Validation finale locale

- Go `test -race ./...` avec `THEMIS_TEST_DATABASE_URL` sur PostgreSQL synthétique : réussi, y compris migration033, contraintes propriétaires, export et contexte assistant ; `go vet ./...` réussi.
-16 tests Python, dont contrats FastAPI/Graphiti réels et inspection d’un FalkorDB jetable à deux groupes : réussis. Aucun modèle de production sollicité.
-83 tests Node : réussis. Build frontend : réussi.
- Suite navigateur complète réussie : nouveau panneau, assistants, configuration IA, parcours financiers existants, Axe clair/sombre et chargement production HTTP sans randomUUID.
- E2E :11 parcours fonctionnels réussis, dont import/API/PostgreSQL/journal mémoire et isolation.4 références visuelles existantes échouent localement ; reproduites sur main inchangé, les4 captures obtenues sont identiques octet pour octet entre main et cette branche. Aucun seuil ni référence modifié. La CI vérifiera son navigateur de référence.
- Image Python construite ; conteneur non-root démarré, `/health` et `/kg/inspect/90003` testés via HTTP contre FalkorDB réel : succès, absence confirmée sans création de groupe.

## Limites et livraison

Voir `KNOWLEDGE_ACTIVITY.md` pour la procédure et les limites. Pas d’accès à la VM de production, ni de test d’inférence avec les modèles Ollama de l’utilisateur. Reconstruire app et knowledge-graph. Les fichiers partagés touchés sont router/import, chat complet/flottant, export, navigation/chargement/API frontend, configuration IA, CI et Compose. La variante Supabase reçoit l’URL interne de mémoire par défaut ; le port Python est lié à la boucle locale dans les deux variantes.

Branche `feat/knowledge-graph-activity`. Publication et vérification CI avant fusion selon le workflow demandé. Aucun changement direct dans main, aucun déploiement sur la VM.
