# Taxonomie existante et rationalisation proposée

21 novembre 2026. Synthèse de sprint 0, complément de l’[audit](../CLASSIFICATION_AUDIT.md) et du [référentiel](FINANCIAL_CONCEPTS.md). Aucun renommage, suppression, fusion ou reclassement n’a été effectué.

## Inventaire statique vérifié

`backend/internal/api/seed.go::seedUserData` définit neuf racines ordinaires et une racine technique :

| Domaine initial | Rôle actuel/ambiguïté | Proposition à instruire |
|---|---|---|
| Logement | Fonction ; loyer, énergie, eau via règles | Conserver la fonction, éventuels enfants bornés. |
| Transport | Fonction ; UBER inclut des libellés pouvant désigner livraison ou déplacement | Conserver, tester conflits de motifs spécifiques avant généralisation. |
| Alimentation | Supermarchés ; marchand ne prouve pas tout le panier | Conserver, admettre cas mixtes et incertitude. |
| Santé et Bien-être | Soins et confort regroupés | Examiner sous-domaines sur usages validés, pas sur un nom seul. |
| Divertissement | Loisirs et certaines dépenses contractuelles | Conserver la fonction, récurrence/abonnement séparés. |
| Épargne et Investissements | Mélange réserve, actif, frais et produits | Sortir à terme le pilotage économique du nom ; ne pas marquer toute la catégorie comme transfert. |
| Vie Professionnelle | Contexte transversal ; peut inclure salaire et frais | Domaine ou contexte à décider ; ne pas déplacer l’historique sans revue. |
| Famille | Contexte transversal, recoupe alimentation/logement/santé | Même décision ; conserver une correspondance explicite des anciens usages. |
| Autres | Divers ou ancien reclassement ; différent de NULL | Garder l’option « autre justifié », distinguer « à qualifier ». |
| Transfert interne | Drapeau `is_transfer=true`, exclusion des flux ordinaires | Conserver pour compatibilité ; future qualification ne doit pas dépendre du libellé. |

Ce sont **10 définitions initiales**, pas le nombre de domaines actuellement présents chez l’utilisateur. Le seed ne définit pas de catalogue de tags. **Nombre réel de domaines : ND ; nombre réel de tags : ND.** Méthode facultative : [métriques locales](CLASSIFICATION_METRICS.md).

Les migrations 006/008 ont déjà remappé catégories et domaines de vie : Vie perso/Vie amoureuse vers Autres, Loisirs vers Divertissement, Vie pro vers Vie Professionnelle. Certaines informations initiales ne peuvent pas être reconstituées à partir du seul état courant. Ne pas rejouer ces migrations ni utiliser suppression/recréation pour rationaliser : FK des transactions SET NULL, règles/budgets et enfants en cascade selon relations. Le schéma effectif doit être vérifié localement avant toute future intervention.

## Ordre réellement appliqué

`categorize.Load` trie longueur du motif en **caractères PostgreSQL**, décroissante, puis ID croissant ; `Engine.Match` retient le premier match. `CompilePattern` reconnaît `re:` sensible au préfixe et rend la regex insensible à la casse ; sinon recherche littérale insensible à la casse. `%`/`_` sont littéraux. Il n’existe pas de bouton actif/inactif dans la table : distinguer exécutable, regex invalide/domaine indisponible, sans correspondance observée, et jamais gagnante. Aucune correspondance historique n’implique pas inutilité future.

`applyAllRulesForStatement` applique ensuite toutes les règles ; le domaine reste limité aux lignes sans domaine, mais les tags sont ajoutés sur toutes les correspondances. Une règle moins prioritaire peut ainsi ajouter un contexte contradictoire à une catégorie correcte. Une regex invalide est sautée au chargement initial mais fait échouer sa réapplication ; les règles précédentes peuvent déjà être validées dans leurs transactions individuelles. L’outil de métriques identifie des **effets potentiels sur des correspondances observées**, pas le journal de ce qui s’est effectivement produit.

`coherence.go` utilise `ILIKE '%'||pattern||'%'` pour les motifs simples et ignore les regex. Les caractères `%` et `_` changent alors de sens. L’outil du sprint utilise directement `CompilePattern`, sans traduction SQL des regex et sans appeler cet endpoint, qui peut aussi envoyer un échantillon à l’IA si activé.

## Corrections, tags et propositions

Un domaine manuel non nul est protégé par le test `domain_id IS NULL`. Une correction manuelle vers NULL ne l’est pas. Les tags modifiés manuellement peuvent recevoir ensuite de nouveaux tags de règles. La provenance courante ne décrit ni les règles gagnantes successives ni l’origine de chaque tag. `AutoEnrich` propose uniquement pour les domaines manquants, vers des racines ; l’acceptation par formulaire devient une modification manuelle. Le rejet d’une suggestion n’a pas de registre durable général empêchant sa régénération.

Les tags sont libres, normalisés en minuscules/trim et dédoublonnés au niveau nom/lien. Aucune observation de données réelles ne permet d’affirmer ici qu’un tag particulier est redondant. Les signaux suivants nécessitent une revue : nom égal à un domaine, mot de signe/type (débit/transfert), fréquence/abonnement utilisé comme preuve, nom de compte/marchand, mois/année, projet déjà identifié, remboursement déjà structuré. Le détecteur de l’outil est volontairement lexical et limité ; il ne conclut pas à l’équivalence sémantique.

Deux tags toujours associés peuvent représenter deux axes légitimes. Un tag lié à un seul marchand peut désigner un contrat utile. Un parent sans transaction directement affectée peut couvrir des enfants/budgets ; « inutilisé directement » n’est donc pas « à supprimer ». Un marchand multi-domaines peut être correctement classé selon les achats.

## Trajectoire taxonomique proposée, sans application

Conserver Domaine + Tags comme interface simple. Natures explicites futures indépendantes ; fonctions candidates : logement, alimentation, transport, santé, loisirs/culture, éducation, communications, services/frais, impôts/taxes, autres. Ce n’est ni un nouveau seed ni une liste imposée. Contextes famille/professionnel à revoir avec l’utilisateur ; sous-domaines à un niveau pour compatibilité.

Préparer des correspondances versionnées, leurs effets sur règles/budgets/scénarios, des alias et un historique ; aperçu par propriétaire et validation avant toute écriture. Ne pas convertir automatiquement des tags en nature ou en allocations. Une information qui pilote un calcul (remboursement, dette, transfert confirmé, portion économique) doit avoir une preuve structurée, pas seulement une étiquette.

Les métriques de fréquences/confusions, la précision d’étiquetage et la cohérence du KG restent ND. Les chiffres du seed ne prouvent ni installation ni usage réel. Le prochain sprint doit d’abord sécuriser et tracer les décisions, pas rationaliser automatiquement les données.
