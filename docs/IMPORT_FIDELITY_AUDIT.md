# P0 — Fidélité des imports de relevés

Date : 25 octobre 2026. Travail local sur `feat/statements-documentary-hardening`, sans commit, push, merge, déploiement ni réparation des données réelles. Les changements documentaires préexistants sont conservés.

## Diagnostic démontré

Le PDF de référence 009 est un document de trois pages, trouvé sans extension. Son empreinte est contrôlée avant/après ; l’original reste inchangé. Ni le document, ni son texte, ni les libellés bancaires personnels ne sont intégrés au dépôt. Les preuves nominatives et l’inventaire historique détaillé restent dans un répertoire local privé, communiqué à l’utilisateur.

La perte n’a pas lieu dans PostgreSQL ou le frontend :

1. Le décodeur Go historique interprète mal les polices Type 3 et leurs mappings Unicode personnalisés. Il produit 273 lignes de caractères corrompus, considérées à tort comme du texte utile, sans opération exploitable.
2. Le secours OCR de l’image Debian utilisée par l’application contient les 81 lignes d’opérations. Deux lignes du 15 septembre, page 3, portent un code numérique de marge placé avant les deux dates du tableau.
3. `ParseLocal` exigeait une date en premier mot. Ces deux lignes étaient ignorées sans diagnostic, alors que leurs dates, libellés et montants figuraient dans le texte OCR. Il s’agit d’un achat carte de 2,69 € et d’un prélèvement fiscal de 243,00 €, soit 245,69 € manquants. Références : première opération de la page 3 et ligne du prélèvement fiscal de cette page ; les références précises et libellés figurent dans le rapport privé.
4. Deux autres dates sont erronées dans l’OCR historique : un débit de 40 € du 27 août lu au 21 août, et un crédit de 940,58 € du 1er septembre lu au 1er août. Aucun montant n’a été inventé pour les corriger : l’extraction textuelle indépendante conserve les dates du PDF.
5. Les 79 empreintes du parsing OCR sont distinctes. La base réelle contient exactement ces 79 opérations et les mêmes totaux. La déduplication n’est donc pas la cause de cet incident précis.

| Étape | Avant correction | Débits | Crédits | Après correction vérifiée |
|---|---:|---:|---:|---|
| Référence manuelle indépendante | 81 | 4 293,85 € | 2 965,85 € | inchangée |
| Décodeur Go natif | 0 opération exploitable ; 273 lignes corrompues | non vérifiable | non vérifiable | extraction Poppler positionnelle |
| Texte OCR, inventaire des lignes candidates | 81 | 4 293,85 € | 2 965,85 € | recours secondaire, revue explicite |
| Parsing positionnel OCR historique | 79 | 4 048,16 € | 2 965,85 € | 81 avec extraction textuelle Poppler |
| Normalisation | 79 | 4 048,16 € | 2 965,85 € | 81 ; montants et dates conservés |
| Déduplication | 79 ; aucune collision dans ce cas | 4 048,16 € | 2 965,85 € | 81, occurrences sources distinctes |
| Persistance | 79 en base réelle, laissées intactes | 4 048,16 € | 2 965,85 € | 81 dans un schéma PostgreSQL isolé |
| API des transactions | 79 à partir des données historiques | mêmes montants sources | mêmes montants sources | 81 et montants bruts réconciliés, test d’intégration |
| Frontend | consomme le résultat API | aucune restitution des opérations absentes | — | contrat 81 + rapport testé dans un navigateur avec données synthétiques |

Après correction, variation brute **−1 328,00 €** et `2 492,88 + 2 965,85 − 4 293,85 = 1 164,88 €`. Dates d’opération : 25 août–21 septembre ; période bancaire : 21 août–21 septembre.

Les indicateurs financiers de l’API excluent les transferts internes. Leur net ordinaire doit donc être rapproché du net brut en réintégrant séparément les crédits et débits de transfert. Les tests vérifient cette identité ; ils ne forcent pas un indicateur analytique à devenir un total bancaire brut.

## Criticité et corrections

### P0 — Perte silencieuse et lecture incertaine

- Extraction `pdftotext -layout` avant le décodeur historique, déjà disponible dans l’image runtime. Positions des colonnes établies pour chaque page : la largeur d’un caractère peut changer entre pages. Fichier temporaire privé, suppression systématique, contexte limité à 30 secondes et sortie plafonnée à 32 Mio. Stderr bancaire jamais journalisé.
- Codes de marge supprimés uniquement lorsqu’un token numérique distinct précède explicitement les deux dates. Aucun montant ni date déduit du solde.
- Opérations multiligne et continuité aux changements de page conservées. Date impossible, montant orphelin, montant multiple, montant/libellé absent ou colonnes de sens inconnues exposés dans le registre des candidats. Un registre entièrement rejeté reste consultable, même sans transaction exploitable.
- Suppression de l’arbitrage qui choisissait une lecture OCR alternative pour équilibrer les comptes. Une ambiguïté reste une ambiguïté. L’OCR requiert une revue ; une IA ne remplace plus un résultat déterministe incomplet par des opérations devinées. Une extraction uniquement IA est non validable et non persistable.
- Calculs en centimes, y compris conversion initiale des nombres français. Les floats des anciens modèles JSON restent une frontière de compatibilité, sans calcul financier approximatif.

### P0 — Réconciliation indépendante

Rapport disponible dans l’aperçu et après enregistrement : candidats, opérations interprétées/enregistrées/rejetées, références page/ligne, totaux calculés et déclarés, écart, contrôles exécutés/non exécutés et motifs.

Les totaux de débits et crédits sont lus dans « Total des opérations », indépendamment des transactions parsées. Le nombre attendu ne vient que d’une mention explicite « Nombre d’opérations ». Pour le relevé 009, aucun compteur explicite n’a été identifié : **nombre non vérifiable automatiquement**, même si le test privé impose la référence manuelle de 81.

L’égalité des soldes ne prouve pas la complétude. `validated` signifie rapprochement des soldes et des totaux indépendants conforme, sans lecture ambiguë, et non preuve absolue de complétude. `import_report.completeness` reste `unknown` sans compteur indépendant. Un échec bloque l’enregistrement et la validation ; l’absence de contrôles peut produire uniquement un enregistrement provisoire après confirmation explicite.

Revue de reprise : plusieurs compteurs ou totaux déclarés contradictoires ne doivent pas s’écraser successivement. Les diagnostics `conflicting_declared_count` et `conflicting_declared_totals` conservent cette ambiguïté et bloquent l’import, même après `RefreshImportReport`. Des déclarations répétées identiques restent acceptées. Le rapport partagé de l’aperçu et de Relevés explique ces deux motifs. Aucun changement de contrat API ou de schéma supplémentaire.

Les tests couvrent la contradiction suivie d’une déclaration conforme (qui masquait auparavant le problème), les totaux identiques répétés et la conservation du blocage au recalcul. Le test de persistance en échec utilise désormais une période et des libellés distincts et vérifie précisément le motif du rapport, pour éviter une réussite accidentelle due au contrôle de chevauchement.

### P0 — Déduplication et atomicité

Pour les nouveaux imports PDF/CSV : identité du fichier exact, compte et indice d’occurrence source. Deux opérations identiques légitimes sont conservées. La contrainte historique `(owner_id, hash)` est conservée, avec des empreintes sources distinctes pour les nouvelles écritures ; aucune empreinte ni donnée historique n’est réécrite.

Un fichier identique avec le même résultat de parsing est un doublon justifié et ne réapplique pas les règles, tags ou alias. Un résultat différent pour le même document exige une réparation contrôlée. Des similitudes avec une autre source ou un recouvrement de périodes bancaires suspendent l’import : aucune suppression fondée sur une ressemblance. Le CSV expose ses dates d’opérations sans les interpréter comme une période documentaire indépendante.

Toutes les insertions et le rapport sont atomiques, sous verrou du propriétaire. Aucun `ON CONFLICT DO NOTHING` dans ce nouveau chemin. Une erreur SQL annule relevé, opérations et rapport ; aucun candidat n’est annoncé enregistré avant le commit. Compte et propriétaire vérifiés dans la transaction. L’ancien chemin interne sans identité source est maintenu pour compatibilité ; les uploads HTTP PDF/CSV passent par le chemin source.

## Schéma et compatibilité

Migration additive `026_import_fidelity.sql`, après les contrôles documentaires 025 déjà présents localement : empreinte et rapport de source sur le relevé ; empreinte de similarité non unique, indice, page, ligne et date de valeur sur les transactions. Pas de backfill des transactions historiques. Les historiques, catégories et tags sont conservés.

Rollback gardé : autorisé avant la première utilisation ; refusé après création de provenance source, pour éviter sa destruction. Les migrations ne sont appliquées qu’aux schémas de test isolés. Le déploiement futur devra appliquer 025 et 026 via le mécanisme habituel.

## Historique et réparation : limites explicites

La commande `cmd/importaudit`, propriétaire explicite obligatoire, effectue une transaction PostgreSQL **READ ONLY / REPEATABLE READ**. Elle ne migre pas la base. Elle recalcule les montants enregistrés, compare les soldes et les preuves indépendantes disponibles, signale les dates hors période et distingue contrôle inconnu et réussi. Aucun libellé brut ou secret dans sa sortie.

L’audit local des 19 relevés existants trouve **12 écarts arithmétiques**, dont celui du relevé 009, et **2 autres relevés à examiner pour des dates hors période**. Sept des quatorze relevés à examiner portent un ancien flag de validation. Ces résultats ne démontrent pas une cause identique pour les autres documents. Les sept relevés équilibrés ne sont pas pour autant prouvés complets. L’inventaire nominatif reste hors Git.

Réparation proposée, **non exécutée** : sauvegarde restaurable ; extraction de chaque original disponible ; comparaison multiensemble par occurrence source ; revue explicite des ajouts et corrections de dates ; correspondance univoque des écritures avant transfert des catégories, tags, suggestions et modifications manuelles ; transaction atomique et audit avant/après. Les correspondances ambiguës doivent rester en attente. Aucun remplacement en masse ou bouton de contournement n’est ajouté.

Risques résiduels : documents image/OCR, mises en page non prises en charge et totaux indépendants absents restent provisoires ou bloqués. La ressemblance interdocuments impose parfois une revue même pour des opérations légitimes. La commande d’audit ne réextrait pas les anciens PDF, qui ne sont pas conservés par le stockage actuel. Elle ne peut donc pas établir un nombre attendu manquant ni découvrir toute paire compensée absente d’un ancien import.
