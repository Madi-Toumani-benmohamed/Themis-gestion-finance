# Agent Thémis : contexte, missions et actions

L’Assistant et son panneau flottant utilisent désormais le même espace de conversation. Aucun écran supplémentaire ni fournisseur externe n’est ajouté.

## Utilisation

- « Donne-moi ma top dépense du mois » consulte la plus grosse opération négative enregistrée du 1er du mois courant à aujourd’hui. Les virements internes, leurs sous-catégories et l’épargne sont exclus. La période et l’opération source accompagnent le résultat. Sans données, l’agent indique l’absence d’observation et la dernière date connue ; il ne change pas discrètement de mois.
- « Le mois dernier », « hier », « cette semaine », « cette année », les mois français et les dates ISO sont résolus côté serveur. « Et le mois précédent ? » reprend la période de conversation. Le fuseau IANA du navigateur détermine le jour local ; Europe/Paris est le repli. Une date client n’est jamais acceptée comme horloge.
- Le panneau flottant connaît la page et le mois affichés. Une période explicite comme « ce mois » prévaut. Dans Transactions, sélectionner jusqu’à 100 opérations puis « Discuter de la sélection » permet « Analyse ces opérations ». Les autres filtres nécessitent cette sélection explicite : l’agent ne prétend pas les reproduire.
- « Missions et préférences » crée un objectif suivi, lié à une conversation persistante. On peut le mettre en pause, le réactiver ou le terminer. Les conversations et leurs sources se reprennent via la liste.
- Après un nouvel import, une mission active propose de faire le point. Les découvertes déjà vérifiées par Thémis peuvent également ouvrir une analyse. L’ouverture de l’agent ne déclenche pas de modèle ni d’action bancaire en arrière-plan.
- « Travail réalisé » expose les consultations effectivement exécutées et leurs dates, et les sources restent consultables. Une affirmation de preuve provenant seulement du modèle n’est pas retenue dans ces sources.

## Actions et préférences

Le modèle peut uniquement **préparer** une proposition. Les boutons « Valider et appliquer » / « Ignorer » portent la décision explicite. Le formulaire intégré permet aussi de préparer sans dépendre de la capacité du modèle à appeler un outil.

| Proposition | Effet après validation |
|---|---|
| Catégorisation | Catégorie manuelle des seules opérations listées ; suggestions antérieures retirées |
| Tag | Ajout d’un tag normalisé, en conservant les autres tags |
| Règle | Nouvelle règle pour les prochains imports ; aucun reclassement historique, aucun remplacement implicite d’une règle existante |
| Simulation | Résultat hypothétique fondé sur un mois observé, enregistré avec la proposition ; aucune écriture de montant bancaire |
| Préférence | Texte déclaré par l’utilisateur, visible et supprimable avec « Oublier » ; pour le remplacer, oublier l’ancien puis valider le nouveau |

Les aperçus de transactions sont empreintés avec leurs sources et leur catégorie de destination. Une source modifiée depuis l’aperçu bloque l’application et demande un nouvel aperçu. La décision et les mutations sont atomiques ; rejouer une action appliquée n’ajoute pas une seconde écriture. Les opérations ne changent jamais de montant, date ou libellé bancaire. Les catégories et tags restent éditables dans Transactions. Les préférences déclarées sont distinctes du Knowledge Graph et ne constituent aucune preuve financière.

## Architecture et API partagée

- `agent_period.go` : résolution déterministe des périodes, intention simple, contexte précédent.
- `agent_chat.go` / `chat.go` : historique serveur propriétaire, horloge fiable, sérialisation des tours, outils bornés, sources et étapes exécutées.
- `agent_tools.go` : lectures PostgreSQL avec centimes exacts, exclusions métier, sélection éventuelle et couverture explicite. Une comparaison ne calcule un écart que pour deux mois complets documentés et comparables.
- `agent_workspace.go` : missions, préférences, briefing, propositions, aperçus et décisions atomiques.
- `components/agent/AgentConsole.jsx` : interface partagée. Le panneau flottant charge ce composant à l’ouverture pour conserver le chargement différé initial.
- `App.jsx`, Dashboard, Analyse, Budgets et Transactions transmettent le contexte ; une action validée rafraîchit les données sans effacer la conversation. Les styles de l’agent sont limités à son composant.
- Routes authentifiées : `GET/POST /api/agent/missions`, `PATCH /api/agent/missions/{id}`, `GET/DELETE /api/agent/memories[/{id}]`, `GET /api/agent/briefing`, `GET /api/agent/conversations[/{id}]`, `GET/POST /api/agent/actions`, `POST /api/agent/actions/{id}/decision`. `POST /api/chat` conserve son contrat existant et ajoute `context` et `mission_id`.
- Migration additive **034_agent_workspace.sql** : missions, préférences, propositions et métadonnées de conversation. Les relations et toutes les lectures/écritures restent propriétaires. L’export personnel inclut ces données. Le rollback explicite refuse d’effacer des données d’agent déjà créées ; le test du rollback 033 déroule d’abord 034, sa dépendance.

## Bornes et limites

50 missions ouvertes et 50 préférences par utilisateur ; liste des 50 missions prioritaires (terminées en dernier), 30 conversations récentes, 100 messages affichés, 8 messages de contexte modèle, 100 opérations par action, 200 groupes par synthèse avec indicateur de troncature, 6 tours modèle et 12 outils maximum par échange. Pas de notifications hors application, planificateur autonome, paiement ni modification de montant. La qualité de planification générale dépend toujours du fournisseur configuré ; la demande simple de top dépense utilise directement PostgreSQL après vérification de sa formulation. La couverture inconnue n’est jamais convertie en couverture complète.

## Vérifications

- Tests Go de périodes, dates de frontière, isolation multi-utilisateur, historique non usurpable, exactitude, verrouillage/rejeu, sources périmées, scénario sans mutation et reprise de mission, sur PostgreSQL synthétique.
- `tests/agent.browser.cjs`, intégré à `test:browser` : interface réelle avec API synthétique, missions, historique, outils, validation et refus périmé, oubli, six tailles/thèmes, Axe plein écran et panneau flottant.
- `tests/e2e/agent.spec.ts` : sessions réelles, mission et proposition persistées, absence d’écriture implicite, validation/rejeu, isolation de deux propriétaires et maintien du montant.

Les tests n’utilisent pas de données bancaires personnelles. Le fournisseur réel de la VM et le déploiement de la VM ne sont pas validés par ces tests ; reconstruire/redémarrer l’application déployée est nécessaire pour recevoir le code et appliquer la migration additive.

Le contrôle de sécurité a identifié GO-2026-6859 dans la bibliothèque HTTP du Go du scan : les builds CI, le scan et le builder Docker sont alignés sur Go 1.26.11, version corrigée. Aucun contrôle de sécurité n’est ignoré.
