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

50 missions ouvertes et 50 préférences par utilisateur ; liste des 50 missions prioritaires (terminées en dernier), 30 conversations récentes, 100 messages affichés, 8 messages de contexte modèle, 100 opérations par action, 200 groupes par synthèse avec indicateur de troncature, 6 tours modèle et 12 outils maximum par échange. Pas de notifications hors application, planificateur autonome, paiement ni modification de montant. La qualité de planification générale dépend toujours du fournisseur configuré ; les demandes simples de top dépense et de total des dépenses utilisent directement PostgreSQL après vérification de leur formulation. La couverture inconnue n’est jamais convertie en couverture complète.

## Vérifications

- Tests Go de périodes, dates de frontière, isolation multi-utilisateur, historique non usurpable, exactitude, verrouillage/rejeu, sources périmées, scénario sans mutation et reprise de mission, sur PostgreSQL synthétique.
- `tests/agent.browser.cjs`, intégré à `test:browser` : interface réelle avec API synthétique, missions, historique, outils, validation et refus périmé, oubli, six tailles/thèmes, Axe plein écran et panneau flottant.
- `tests/e2e/agent.spec.ts` : sessions réelles, mission et proposition persistées, absence d’écriture implicite, validation/rejeu, isolation de deux propriétaires et maintien du montant.

Les tests n’utilisent pas de données bancaires personnelles. Le fournisseur réel de la VM et le déploiement de la VM ne sont pas validés par ces tests ; reconstruire/redémarrer l’application déployée est nécessaire pour recevoir le code et appliquer la migration additive.

Le contrôle de sécurité a identifié GO-2026-6859 dans la bibliothèque HTTP du Go du scan : les builds CI, le scan et le builder Docker sont alignés sur Go 1.26.11, version corrigée. Aucun contrôle de sécurité n’est ignoré.


## Dialogue naturel et périmètres de recherche

Les dates françaises sont reconnues : « du 1er au 15 novembre 2026 », « entre le 3 et le 9 novembre », « depuis le 3 novembre » et « du 03/11/2026 au 09/11/2026 ». Les jours invalides sont refusés ; les années bissextiles et les passages décembre/janvier sont contrôlés. « La même période l’an dernier » conserve l’intervalle de conversation, avec le 29 février ramené au 28 février si nécessaire. « Et en novembre ? » reprend l’année de l’intervalle précédent. « Compare ce mois au mois dernier » revient au mois réellement courant, même après une discussion d’une ancienne année. Les quantités usuelles écrites en lettres (« deux dernières semaines », « dix derniers jours ») sont acceptées. « Depuis 2024 » et « depuis 2025-12 » s’étendent du début de l’année ou du mois demandé jusqu’à aujourd’hui. Les années glissantes et les quantités usuelles jusqu’à « soixante » sont reconnues ; les fenêtres trop grandes sont refusées. Un montant comme « plus de 2000 euros en novembre » n’est pas interprété comme l’année 2000.

Les outils de recherche, synthèse, regroupement, ventilation et comparaison partagent les filtres : marchand littéral (les caractères `%` et `_` restent littéraux), catégorie exacte, tag normalisé, direction revenus/dépenses et bornes inclusives de montant absolu au centime. Les identifiants de catégorie doivent appartenir à l’utilisateur. Les catégories parentes n’élargissent pas implicitement le filtre à leurs enfants. Les filtres réellement utilisés accompagnent les preuves et sont affichés dans les données sources.

Une recherche affiche jusqu’à 50 opérations et indique s’il reste des résultats. La synthèse calcule sur toutes les opérations du même périmètre, sans sommer une liste tronquée. Une comparaison respecte également la sélection explicite d’opérations. Un filtre inconnu, une borne inversée ou un argument mal formé provoque une erreur d’outil que le modèle peut corriger ; aucun filtre invalide n’est remplacé par une lecture générale. Les arguments numériques du modèle sont décodés sans conversion intermédiaire en nombres flottants.

Après la première ouverture, fermer le panneau flottant conserve la conversation, les propositions et le brouillon pendant la session. Le panneau fermé est masqué et n’est pas accessible au clavier. Sa réouverture ne relance pas le modèle. La première ouverture reste chargée à la demande ; ces données ne sont pas copiées dans le stockage du navigateur. La déconnexion démonte l’interface.

Validation complémentaire : périodes françaises figées, sources PostgreSQL synthétiques, filtres combinés et limites, total exact d’une liste partielle, comparaison sur sélection, refus d’arguments mal formés sans requête, conservation du panneau/brouillon, affichage des filtres et Axe.


## Totaux de dépenses sans solliciter le modèle

Les questions simples « Combien ai-je dépensé ce mois ? », « Quel est le montant de mes dépenses du 1er au 10 décembre 2025 ? » et « Total des dépenses 2025 » reçoivent une réponse calculée directement par Thémis. Le même calcul exact, les exclusions des virements/épargne, la sélection explicite et les dates résolues sont utilisés. Le résultat est conservé dans la conversation avec l’étape réellement exécutée et la source de synthèse. « Et le mois précédent ? » reprend ce calcul dans la période précédente.

Ces observations ne sollicitent pas le fournisseur, même s’il est momentanément indisponible ; l’Assistant doit toujours être activé/configuré. Une absence d’opérations de dépense produit un état de données insuffisantes, avec la dernière date connue, et non une dépense bancaire réelle nulle. La couverture complète n’est jamais affirmée.

Une catégorie, un marchand, une condition de montant, une comparaison ou une demande de projection conservent le parcours de planification et ses outils. Une demande qualifiée ne devient jamais un total général pour contourner un fournisseur indisponible. Dans une mission, après une sélection d’opérations, après une demande qualifiée ou dans une conversation ancienne sans périmètre sûr enregistré, les suivis conservent le modèle afin de ne pas oublier les contraintes précédentes ; ouvrir un nouvel échange permet de revenir au parcours direct. Les outils et les tests d’orchestration restent vérifiés séparément de ce parcours déterministe.

Les références « cette opération » et « cette transaction » utilisent l’unique opération sélectionnée (ou la dernière opération explicitement citée). Aucune sélection ou plusieurs opérations pour une référence singulière entraînent une clarification, sans interrogation financière générale. « Ces transactions » désigne la sélection entière. Les suivis temporels comme « et en novembre ? » conservent aussi l’année de la conversation et le calcul direct lorsque son périmètre est sûr.

## Point de départ contextualisé

À l’ouverture d’un nouvel échange, le briefing affiche les opérations enregistrées du mois courant (jusqu’à aujourd’hui) ou du mois affiché dans le panneau flottant. Le fuseau du navigateur accompagne la requête. La période et le fuseau sont visibles. Cette vue d’ensemble est indépendante des filtres et de la sélection de la page.

Le point distingue le nombre total d’opérations, les dépenses et les dépenses sans catégorie. Les virements internes et l’épargne ne font pas partie des dépenses à classer. « Examiner les dépenses sans catégorie » ouvre les identifiants exacts dans Transactions ; à partir de 101 résultats, les 100 plus récents sont proposés avec un avertissement de liste partielle. Le total reste complet. Les compteurs et identifiants proviennent d’un seul snapshot PostgreSQL, isolé par propriétaire.

« Préparer une question sur cette période » remplit le brouillon avec les dates exactes ; il ne lance pas le modèle. Une période sans opérations conduit à vérifier les relevés, sans conclure à une dépense nulle. La couverture bancaire reste inconnue. Une erreur du briefing ne se transforme pas en observation vide et les résultats d’une ancienne période ne sont pas présentés comme nouveaux.

Contrat additif : `GET /api/agent/briefing?month=YYYY-MM&timezone=Europe%2FParis` renvoie `period_summary`. Un mois absent signifie le mois courant. Un mois invalide/futur ou un fuseau invalide est refusé. Les clients existants conservent les découvertes et les dates de fraîcheur. Ce point ne dépend pas d’un fournisseur IA et ne crée ni message, ni mission, ni action, ni catégorie.

## Revenus et suivis de direction

« Combien ai-je reçu en janvier 2026 ? », « Mes revenus ce mois ? » et « Et mes revenus ? » utilisent le calcul direct comme les dépenses. Le suivi temporel conserve la direction précédente ; « et mes dépenses ? » la change explicitement. Le périmètre sûr et la sélection sont soumis aux mêmes garde-fous que les dépenses. Aucun appel au fournisseur n’est nécessaire pour ces observations simples lorsque l’Assistant est configuré.

La réponse porte sur les entrées d’argent enregistrées, hors virements internes et épargne. Elle précise que ces crédits peuvent inclure des remboursements et ne constituent pas un revenu fiscal calculé. La preuve n’affiche ni faux montant de dépenses ni solde global lorsque seule la direction revenus a été interrogée. L’absence de crédit est un manque d’observation, pas la preuve d’un revenu nul.

Les questions sur le salaire, le revenu fiscal, un marchand, un seuil, un classement, un mélange de directions ou une projection conservent le planificateur ; leurs contraintes ne sont jamais supprimées pour répondre hors ligne.

## Suivis par catégorie

« Et par catégorie ? » reprend la période et la direction d’un échange direct : dépenses après un total de dépenses, entrées après un total de revenus. « Répartis mes dépenses par catégorie » change explicitement de direction ; un suivi temporel reprend ensuite cette ventilation. « Et au total ? » revient au total de la même direction et de la même période. « Où va mon argent ? » démarre une ventilation de dépenses. Les montants et le nombre d’opérations de chaque catégorie proviennent de l’outil PostgreSQL existant, sans appel au modèle.

La réponse résume six catégories, avec les autres dans les données sources. L’outil expose au maximum 200 catégories et annonce toute troncature ; aucune somme de cette liste n’est présentée comme un total complet. Les entrées restent distinguées d’un revenu fiscal et peuvent inclure des remboursements. Virements internes et épargne restent exclus, la couverture demeure inconnue et l’absence d’observations reste distincte d’un zéro bancaire.

Un marchand, une catégorie nommée, un seuil, un classement, une comparaison ou une question explicative conserve le planificateur. Les missions, anciens contextes sans périmètre sûr, suites de requêtes qualifiées et suivis après sélection d’opérations n’empruntent pas ce raccourci, pour ne pas oublier leurs contraintes. Les identifiants sélectionnés sont respectés et vérifiés pour la première demande explicite.

## Lecture des preuves financières

Lorsqu’une synthèse ou une comparaison porte uniquement sur les dépenses, ses zéros techniques de revenus et son solde directionnel ne sont pas affichés comme des observations de l’ensemble des flux. La même règle s’applique aux sources limitées aux revenus. Les regroupements suivent leur direction réelle ; une lecture de l’ensemble des flux conserve les deux directions et le solde observé. Le contenu des preuves et les calculs restent inchangés.

Les catégories affichent le nombre d’opérations lorsque la source le fournit. Aucun compteur n’est inventé pour les anciens messages ou les scénarios qui n’en contiennent pas. Dans Transactions, un filtre issu d’identifiants est présenté comme une sélection de sources, et non systématiquement comme une récurrence. Les provenances relevé et enveloppe conservent leurs libellés.

Les réponses textuelles conservent leurs sauts de ligne dans l’Assistant et le panneau flottant : les listes de catégories restent lisibles. Les textes restent échappés par React ; aucun interpréteur HTML ou Markdown n’est ajouté.
