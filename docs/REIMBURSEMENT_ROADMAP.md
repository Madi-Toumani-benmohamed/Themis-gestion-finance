# Roadmap — gestion des remboursements

Plan proposé après l’audit et la conception. Cette mission ne réalise aucun lot. Complexité relative : XS (≤1 jour), S (1–3 jours), M (3–7 jours), L (>1 semaine), hors revue et livraison.

## Lot 0 — Décisions produit et contrats (prérequis, S)

- **Périmètre :** décider des indicateurs de crédits/revenus non qualifiés, de l’effet sur le mois budgétaire, du périmètre personnel/commercial, des excédents et des avances hors MVP. Écrire formules et contrat JSON exacts.
- **Composants :** modèle financier, parcours Transactions, labels Dashboard/Analyse/Budget, API de rapports.
- **Dépendances :** aucun changement technique; s’appuie sur `docs/REIMBURSEMENT_AUDIT.md` et `docs/REIMBURSEMENT_DESIGN.md`.
- **Risques :** changer « revenus » peut déplacer des indicateurs historiques; doit être présenté comme un nouveau périmètre, pas comme une correction invisible des chiffres passés.
- **Tests/acceptation :** exemples synthétiques validés pour chacun des cas chiffrés; décision écrite sur report d’un remboursement en septembre depuis octobre; aucun chiffre bancaire altéré.

## Lot 0 bis — Parité des exclusions de transferts (S, avant nouveaux agrégats)

- **Périmètre :** les filtres de dépense des signaux Hero/Insights et du top du Dashboard suivent le prédicat partagé, y compris les domaines enfants d’un transfert. Ne pas changer `is_transfer`, les lignes brutes ou leur présence dans les soldes.
- **Composants :** `hero.go`, `insights.go`, `widgets/index.jsx`, contrat `listCategories`/transactions et tests d’intégration synthétiques.
- **Dépendances :** aucune migration; s’appuyer sur `financialsql.ExpenseT` comme référence.
- **Risques :** les signaux et tops existants peuvent perdre les opérations d’un enfant de transfert qui apparaissaient jusque-là par erreur; tous les transferts restent accessibles dans Transactions.
- **Tests :** transfert parent + transaction enfant, transfert direct, dépense non-transfert; comparer API de flux, signaux Hero/Insights, top widget, réconciliation de relevé et solde post-relevé.
- **Critères d’acceptation :** aucun descendant de transfert n’alimente un signal ou une dépense économique; toutes les lignes restent incluses dans le brut bancaire et le solde. Ce lot n’introduit aucune sémantique de remboursement.

## Lot 1 — Modèle relationnel et service manuel (M, implémenté)

- **Périmètre :** migration additive des dossiers et allocations centimes; endpoints propriétaires pour lister crédit/dépenses éligibles, créer/confirmer, modifier par révocation+nouvelle allocation, annuler et lire l’historique. Aucun rapprochement IA. Aucun auto-classificateur.
- **Composants :** nouvelle migration/rollback manuel, `transactions.go` et service transactionnel dédié, `financialsql`/`money`, contrats API. Le routeur expose uniquement les endpoints authentifiés existants.
- **Dépendances :** Lot 0; clés uniques composées `(id, owner_id)` ou mécanisme DB équivalent; convention d’audit inspirée de `credit_payments`/`credit_history`.
- **Dépendances complémentaires :** Lot 0 bis terminé afin que les nouveaux calculs partagent un périmètre de dépense fiable.
- **Risques :** allocations en concurrence, liens inter-propriétaires, suppression de relevé en cascade, intégrité des historiques. La migration ne modifie aucune opération historique et ne fait aucun appariement initial.
- **Tests :** crédit non positif rejeté; dépense cible non négative/inéligible rejetée; un lien N:N; somme reçue <= crédit; somme allouée <= dépense disponible; arrondi centime; owner isolation par clés DB; rollback du dossier et historique après erreur; deux requêtes concurrentes au dernier centime : une seule réussit; annulation idempotente; suppression de relevé liée bloquée; modification conservée dans l’audit.
- **Critères d’acceptation :** l’utilisateur relie manuellement une dépense et un crédit, peut répartir un crédit sur plusieurs dépenses ou plusieurs crédits sur une dépense, voit reste alloué/non alloué; montant, date, brut, hash, relevé et solde ne changent pas; pas de dépendance IA.

## Lot 2 — Interface de rapprochement manuel (M, implémenté)

- **Périmètre :** action dans le détail d’une transaction pour qualifier un crédit comme remboursement; sélectionner une ou plusieurs dépenses, répartir les montants disponibles, examiner les liens actifs/historique, retirer ou modifier une allocation avec confirmation.
- **Composants :** `TransactionDetails.jsx`, `Transactions.jsx`, `api.js`, modèles frontend et composants financiers réutilisables.
- **Risques :** sélection d’une mauvaise transaction similaire; confusion entre brut, montant compensé et reste; interaction clavier/mobile; réponses périmées après modification.
- **Tests :** états sans source, candidate sélectionnée, allocation partielle/totale, crédit trop grand avec reliquat à qualifier, refus, retrait, erreur API puis reprise; propriétés d’isolation simulées; clavier et affichage responsive; montant final vérifié en centimes.
- **Critères d’acceptation :** rien n’est lié par défaut; toute confirmation est explicite; l’écran montre les deux libellés bruts, dates, montants originaux et montant affecté; la somme des allocations et résidus est réconciliable.

## Lot 3 — Indicateurs de trésorerie et économiques (L, dépend du Lot 1 et Lot 0)

- **Périmètre :** ajouter des postes analytiques distincts : flux bruts par mois réel, crédits non qualifiés, remboursements reçus, dépenses brutes, remboursements affectés et coût net par date de dépense d’origine. Revenus confirmés seulement selon la politique décidée. Couverture documentaire reste indépendante des rapprochements.
- **Composants :** `analysis_report.go`, `intelligence.go`, `transactions.go`, `pulse.go`, `hero.go`, `journal.go`, `handlers.go`/summary, frontend `Dashboard.jsx`, widgets, `Analyse.jsx`, modèles d’analyse.
- **Dépendances :** schéma validé, vues SQL partagées, définition « à date », contrats additifs; ne pas implémenter une somme différente dans chaque endpoint.
- **Risques :** changement de net historique, sommes brutes/nettes confondues, mois d’origine révisé, transfert interne compté à tort. La réconciliation bancaire et les totaux bruts doivent être strictement invariants.
- **Tests :** cas 1/2/3/4; septembre/ octobre (vue cash intacte, septembre net révisé à as-of); crédit non qualifié, dossier non rapproché; crédits et dépenses qui se compensent; transferts parents/enfants exclus du net mais présents dans cash/solde; tags qui ne changent pas le filtre; pas de double comptage par jointure N:M; propriété propriétaire; couverture partielle/inconnue.
- **Critères d’acceptation :** chaque poste se réconcilie au centime; les montants bruts sont inchangés; aucun crédit marqué remboursement n’est simultanément revenu économique; une seule dépense nette par achat; sources exactes navigables; reports historiques indiquent la date de calcul.

## Lot 4 — Budgets, simulations et découvertes (M/L, dépend du Lot 3)

- **Périmètre :** appliquer au budget le choix du Lot 0; mettre à jour la base du simulateur afin que les retours ponctuels ne deviennent pas des revenus récurrents; ajuster les découvertes à la dépense nette confirmée et rendre visibles les preuves brutes.
- **Composants :** `budgets.go`, `Budgets.jsx`, `simulation.go`, `Simulate.jsx`, `analysis_discoveries.go`, rapports et modèles frontend.
- **Dépendances :** calcul net partagé et règles de période; décision écrite sur révision du mois budgétaire d’origine.
- **Risques :** crédit attribué à une mauvaise enveloppe, mélange avec montant du mois du crédit, hypothèse de revenu récurrent tirée d’un remboursement; historique de simulation modifié après nouveaux liens.
- **Tests :** remboursement même mois et mois suivant; allocation entre catégories/budgets; budget d’origine mis à jour sans crédit dépensé en double; simulateur sans crédit exceptionnel comme salaire; historique incomplet/aucune couverture; découverte source brute et net; contrat de crédits séparé des paiements d’engagement `credit_payments`.
- **Critères d’acceptation :** les paiements hypothétiques et récurrences n’utilisent pas les remboursements; l’UI sépare mois d’encaissement du mois d’achat; les simulations disent explicitement quand leur base a été recalculée.

## Lot 5 — Suggestions déterministes et IA optionnelle (M, après la revue du MVP)

- **Périmètre :** proposer, sans confirmer, des candidats de rapprochement selon montant restant, signe, proximité de date, libellé/marchand/contrepartie et propriétaire. Enregistrer le refus d’une suggestion si la répétition gêne; recalculer après correction.
- **Composants :** service candidat dédié, éventuellement table de propositions à durée/révision, UI du Lot 2; IA seulement après filtrage déterministe.
- **Dépendances :** données relationnelles de Lot 1 et retours manuels synthétiques; critères de suggestion observables. KG facultatif.
- **Risques :** faux rapprochement de paiements identiques, exposition inutile de libellés au LLM, réutilisation de suggestion refusée, score de confiance non calibré.
- **Tests :** même montant/marchand à plusieurs dates, deux paiements identiques, source absente, remboursements partiels, rejet persistant, données multi-propriétaires isolées, réponse IA malformée/ambiguë/hors candidats et LLM indisponible. Aucun de ces cas ne crée une allocation confirmée.
- **Critères d’acceptation :** les raisons et montants proposés sont auditables; l’utilisateur accepte ou refuse; aucune IA n’est nécessaire au flux manuel; zéro mutation bancaire; aucun score numérique sans calibration indépendante.

## Lot 6 — Avances, prêts personnels et dettes (L, séparé)

- **Périmètre :** décider si Thémis doit suivre des créances vers des tiers; distinguer avance, principal remboursé, intérêts et dépenses payées pour le compte d’autrui. Étudier ensuite un lien avec l’échéancier/crédit `engagements`, sans importer deux fois un paiement.
- **Composants :** nouveau modèle métier éventuel, interaction avec crédits et engagement history, rapports de trésorerie et actifs/dettes.
- **Dépendances :** décision produit distincte; définition comptable du capital et des intérêts; jamais le domaine `Transfert interne` par défaut.
- **Risques :** modéliser une solvabilité ou un actif certain à partir d’une simple transaction, confusion entre mensualité et intérêt, double comptage paiement déclaré + débit importé.
- **Tests :** prêt 200 € / retour 200 € / intérêt séparé; absence d’opération source; paiement de dette rapproché à une seule écriture; annulation, owner isolation, historique et totaux de trésorerie conservés.
- **Critères d’acceptation :** principal retourné n’est ni revenu ni remboursement de consommation; l’intérêt est séparé seulement s’il est documenté; un paiement ne figure qu’une fois dans le flux bancaire.

## Contrôles de sortie obligatoires pour chaque lot

- Tester le rapprochement brut d’un relevé validé avant/après; toutes les transactions source et montants restent identiques.
- Exécuter les tests Go pertinents, les intégrations PostgreSQL isolées, les tests Node frontend et les builds des pages touchées.
- Utiliser uniquement des données synthétiques; auditer propriétaire, concurrence, annulation et fuite de labels sensibles.
- Mettre à jour les rapports financiers/API et `CODEBASE_GRAPH.md`; expliciter les chiffres bruts, nets, non qualifiés, inconnus et les dates de calcul.
- Ne pas lancer les lots 3–6 avant acceptation de la portée du MVP et des décisions de politique économique/budgétaire.

## État du MVP (lots 1 et 2)

Les lots 1 et 2 ont été implémentés sur la branche `feat/reimbursement-mvp`, sans commit. La migration est `027_reimbursements.sql`; le service et les handlers sont dans `backend/internal/api/reimbursement_service.go` et `reimbursements.go`; le parcours est intégré au tiroir de `Transactions.jsx` via `TransactionReimbursements.jsx`.

Les scénarios et validations exécutés sont répertoriés dans le rapport de mission transmis avec les changements. La couverture navigateur/DOM demeure absente du dépôt : les tests frontend ajoutés vérifient le modèle exact en centimes et les règles de saisie avec Node, tandis que le build valide le rendu JSX. Les lots 3, 4 et 5 restent non commencés. Cette roadmap ne vaut pas décision de lancement de ces lots.
