# Fiscalité et Laboratoire — utilisation et contrat V1

## Fiscalité

Ouvrir **Impôts** dans la navigation pour accéder directement au simulateur fiscal. Il reste également accessible dans **Simulation**, via **Fiscalité** dans l’en-tête. Le mode Budget conserve son scénario pendant le changement de mode. Les données fiscales sont saisies explicitement : aucune ligne bancaire n’est assimilée automatiquement à un salaire ou à un PAS.

Choisir l’année des revenus (2024 ou2025), le foyer, les enfants à charge exclusive et la base de revenu. **Salaire net fiscal** signifie avant déduction des frais professionnels ; **revenu net imposable final** signifie déjà après déductions. Ne pas saisir un salaire brut, un net social ou un net versé dans ces champs. En mode salaire, pour un couple, renseigner aussi le conjoint, y compris un zéro explicite. En mode revenu net imposable final, saisir la base du **foyer entier**, conjoint déjà compris : aucun second revenu du conjoint n’est ajouté. Les frais réels doivent être annuels, qualifiés et justifiés ; la V1 applique le même régime de déduction aux deux déclarants. Le forfait est calculé séparément sur chacun de leurs salaires.

La confirmation de périmètre est obligatoire. Les cas particuliers et dispositifs non implémentés sont exclus, sans prétendre qu’ils valent zéro. Les champs avancés concernent les frais réels et la comparaison d’un autre revenu, pas un conseil automatique de défiscalisation. La comparaison d’un couple exige une saisie annuelle ; le revenu alternatif remplace celui du premier déclarant en mode salaire ou la base finale du foyer en mode net imposable final, à situation identique.

Les revenus mensuels ou cumuls partiels exigent une annualisation explicitement autorisée. Le PAS est un **cumul observé pour la même année des revenus**. L’absence de cumul n’est pas zéro. Un cumul partiel permet seulement un écart avant les futurs PAS ; il ne devient un solde projeté que si une mensualité future est renseignée explicitement. Un remboursement/complément reste une estimation, pas un avis d’impôt. Les hypothèses, étapes, arrondis, limites et liens officiels sont consultables sous le résultat.

Toute modification de saisie invalide le résultat courant ; les réponses d’une saisie précédente sont ignorées. Le moteur fonctionne sans IA et sans modifier les obligations Crédits & taxes.

## Calcul rapide

Le bouton calculatrice dans le pied de navigation ouvre un dialogue depuis toutes les pages authentifiées. Raccourci par défaut : **Ctrl/Cmd+Maj+Espace**, remplaçable par **Ctrl/Cmd+Maj+L** ou désactivable. Il n’intercepte pas une saisie, un élément éditable, un autre dialogue, une composition clavier ou un événement déjà consommé. Certains systèmes réservent ces raccourcis ; le bouton reste disponible.

Utiliser le point décimal, les parenthèses, `+ - * / %`. Les arguments utilisent `;`. Le pourcentage suit mathjs : `10% = 0.1`, `100+10% = 110`, `100*10% = 10`. Les références de cellules sont réservées au mode feuille. Un calcul n’est jamais exécuté comme du JavaScript. Le résultat peut être copié, réutilisé depuis les 50 derniers calculs ou enregistré explicitement comme brouillon serveur.

**Ouvrir en feuille** crée un nouveau document. L’expression originale, son résultat et l’historique restent dans les métadonnées. Les opérandes numériques simples sont extraits quand l’expression arithmétique comporte au plus96 opérandes ; sinon l’expression complète reste conservée. La cellule B2 recalcule l’expression. Une feuille déjà ouverte n’est pas réduite ni remplacée.

## Feuille

**Laboratoire → Nouvelle feuille**. Entrée/F2/double-clic pour éditer, Entrée/Tab pour valider, Échap pour abandonner. Les flèches déplacent la cellule active ; Maj+flèches/clic étend la sélection. La barre de formule affiche toujours le contenu brut. Un texte reste du texte, y compris une balise HTML.

Références : `=A1+B2`, `$A$1`, `A$1`, `$A1`, plages `A1:B5`. Fonctions : `SUM`, `AVERAGE`, `MIN`, `MAX`, `COUNT`, `IF`, `ROUND`. Exemples : `=SUM(A1:A5)`, `=IF(A1>0;ROUND(A1/3;2);0)`. IF n’évalue que la branche retenue. COUNT compte les nombres ; les agrégats ignorent les textes et cellules vides. Une cellule vide vaut0 dans une opération arithmétique directe. Les noms, opérateurs et fonctions non autorisés sont refusés. Ce langage n’est pas Excel ; aucune importation XLSX ou compatibilité générale n’est annoncée.

Erreurs visibles par cellule : syntaxe, nom, valeur incompatible, `#DIV/0!`, `#REF!`, `#CYCLE!`, `#NUM!`, `#LIMIT!`. Une erreur affecte ses dépendants mais ne remplace pas toute la feuille par des zéros. Une nouvelle version n’affiche pas d’anciennes valeurs pendant son recalcul.

Les calculs décimaux utilisent40 chiffres significatifs. Le format monétaire arrondit seulement l’affichage au centime, moitié à l’opposé de zéro (`1.005 → 1.01`, `-1.005 → -1.01`), puis emploie les centimes exacts de Thémis. Les cellules dépendantes utilisent la valeur décimale non arrondie. Les formats nombre/pourcentage préservent aussi la valeur brute ; `0.1` formaté en pourcentage donne `10 %`.

Copier/coller une plage interne conserve les formules et ajuste les références relatives ; les références absolues restent fixes. Un collage externe accepte du TSV et les seules formules autorisées. La copie textuelle préfixe les chaînes pouvant déclencher une formule externe ; elle ne transforme pas un nombre négatif en texte. Aucun presse-papiers ne donne accès aux fichiers ou au réseau. Les collages hors taille sont refusés atomiquement.

Annuler/rétablir : boutons ou Ctrl/Cmd+Z, Maj+Z/Y, 20 états en mémoire. Ajout/suppression de **la dernière** ligne/colonne uniquement en V1. La suppression demande confirmation, invalide les références supprimées et reste annulable. Pas d’insertion arbitraire au milieu de la grille.

## Sauvegarde et confidentialité

Jusqu’à4 documents ouverts, 100lignes×26colonnes, 1000 cellules stockées (formats seuls compris), 512caractères par entrée, 200ko par contenu, 50 calculs dans l’historique. Valeur absolue numérique ≤10^18 ; valeurs non nulles ≥10^-40. Format monétaire limité à9 999 999 999,99€. AST : 200nœuds/profondeur32 par expression, budget100000 par recalcul ; Worker arrêté après3secondes.

La frappe ne déclenche **aucune écriture PostgreSQL**. La récupération temporaire est dans sessionStorage, liée à l’utilisateur et à l’onglet. Elle n’est ni une sauvegarde durable ni une synchronisation multi-onglets. La déconnexion/session expirée la supprime ; fermer un onglet peut la perdre selon le navigateur. Un stockage local bloqué est signalé et le travail reste en mémoire. Une version locale inconnue ou corrompue n’est pas écrasée automatiquement.

**Enregistrer le document**, **Enregistrer une copie**, renommer via le titre, ouvrir/recharger/supprimer via le gestionnaire. La sauvegarde explicite utilise une révision : conflit409 = brouillon conservé, avec choix de copie ou rechargement confirmé. Une réponse réseau perdue à la création peut être rejouée avec le même UUID sans création en double. Si la réponse retrouve une première copie, les modifications courantes restent signalées comme à enregistrer. Une réponse reçue pendant une nouvelle saisie ne marque pas cette saisie comme sauvegardée.

Supprimer un document serveur ne détruit pas son éventuel brouillon ouvert : il devient un nouveau document non enregistré. Fermer un brouillon sale ou recharger une version serveur demande confirmation. Les routes filtrent systématiquement par propriétaire ; un identifiant étranger répond404. Les documents sont inclus dans l’export de leur propriétaire uniquement. Le contenu est du texte/données, pas du HTML exécutable. Aucune formule n’interroge les données bancaires ou un fournisseur IA.

## API et migration

- `GET /api/fiscal/rules` ; `POST /api/fiscal/simulate` : centimes entiers, années explicites, résultat millésimé,400 pour entrée invalide,422 pour périmètre non pris en charge. Aucune sauvegarde fiscale.
- `GET /api/laboratory?offset=0` :50 métadonnées et `has_more` ; `GET /api/laboratory/{id}` : contenu.
- `POST /api/laboratory` : `{client_key,title,type,schema_version:1,revision:0,content}`,201 ou200 avec `replayed:true`.
- `PUT /api/laboratory/{id}` : même contrat, révision serveur requise ; `DELETE` : `{revision}`.404 étranger/absent,409 révision périmée,400 validation,500 erreur générique sans détails SQL.

Toutes les routes exigent une session, même en première installation. `content` contient `rows`, `columns`, `cells:{A1:{input,format}}`, `expression`, `history:[{expression,result,created_at}]`, `source_expression`, `source_result`, `source_operands`. Seuls les contenus bruts sont persistés ; le Worker recalcule à la lecture. Le schéma local0 (`grid`) est migré en1 ; les autres versions sont refusées sans écraser l’original.

Migration032 additive : table `laboratory_documents`, clé étrangère users, contrainte UUID/propriétaire, JSONB, schéma1, révision/timestamps et index de liste. Aucun montant bancaire existant n’est transformé. Application par le mécanisme de migration existant ; rejeu sans effet. Le rollback fourni refuse de retirer la table si elle contient un document. Aucun nouveau service Docker ni variable secrète requis.
