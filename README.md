# thémis.

**Thémis, Titanide de la justice, tient la balance** — ici, celle de tes
comptes. Gestion de finances personnelles sans saisie : un relevé bancaire
PDF déposé par mois, et tout le reste est automatique — extraction avec
validation comptable au centime, catégorisation, tableaux de bord, conseils IA.

> Renommage : l'appli s'appelait « solde ». Le module Go, le binaire et les
> identifiants par défaut de la base sont passés à `themis`. Si tu migres un
> déploiement existant, garde simplement ton `DATABASE_URL` actuel — le
> schéma est inchangé.

- **Palette de commande (Cmd/Ctrl+K)** — une palette clavier ouvrable partout :
  navigation entre les pages, actions (importer un relevé), et recherche de
  transactions en direct. Flèches pour naviguer, Entrée pour ouvrir, Échap pour
  fermer. L'app devient pilotable entièrement au clavier.
- **Détection d'anomalies** — un widget « Ça mérite ton attention » remonte
  spontanément quatre types de signaux : hausse d'abonnement (ex. Netflix
  13 € → 16 €), dépense inhabituelle pour un marchand, nouveau prélèvement
  régulier (abonnement récemment souscrit), et abonnement disparu (résiliation
  ou incident). Règles déterministes, calibrées pour éviter les faux positifs.
- **Tri en pile** — plutôt que de scanner le tableau, un bouton « Trier en pile »
  présente les transactions sans domaine une par une, en plein écran, avec les
  domaines en gros boutons (clic ou touches 1-6 ; → pour passer, Échap pour
  fermer). On traite quelques cartes au lieu de parcourir des dizaines de lignes.
- **Pouls du mois** — en tête du tableau de bord, une phrase de synthèse en
  français répond d'un coup d'œil à « est-ce que tout va bien ? » : ton calme /
  attention / alerte selon l'écart à tes habitudes, poste en hausse, solde du
  mois, transactions à ranger. Calcul déterministe (sans IA), et l'app s'abstient
  de tout jugement tant qu'elle n'a pas au moins deux mois d'historique fiable.

- **Import CSV** — en plus des relevés PDF, thémis avale les exports CSV de la
  plupart des banques, avec détection automatique du format : séparateur (`,`
  `;` ou tabulation), colonnes (date / libellé / montant, ou débit et crédit
  séparés), format de date (JJ/MM/AAAA, AAAA-MM-JJ, MM/JJ/AAAA) et de montant
  (virgule ou point décimal, symbole €). Aucun réglage à faire côté utilisateur.
  Un CSV n'ayant pas de solde de contrôle, il est marqué « à vérifier » plutôt
  que « validé au centime » comme le PDF.
- **Icônes de navigation** — chaque onglet du menu porte un pictogramme en trait.

- **Thème clair / sombre** — bascule via l'icône dans l'en-tête, choix mémorisé
  d'une session à l'autre. Tout le design system suit automatiquement.

- **Analyse croisée catégorie × tag** — dans l'onglet Analyse, deux vues :
  la répartition des dépenses par tag à l'intérieur d'une catégorie (ex.
  Alimentation → courses / resto / livraison), et l'évolution de tags choisis
  au fil des mois toutes catégories confondues. La catégorie donne la structure
  auto-remplie ; le tag donne la finesse optionnelle. En taguant
  perso/amoureux/famille, on retrouve une lecture « domaines de vie » sans
  imposer ce classement partout.

- **Navigation confortable** — la barre d'outils des transactions (mois, filtres,
  « Appliquer les règles ») reste collée en haut pendant le défilement, et un
  bouton « remonter en haut » apparaît sur les longues pages.

- **Isolation des données par utilisateur** — chaque compte dispose de son
  espace totalement cloisonné : transactions, relevés, comptes, catégories,
  règles, budgets, crédits, tags et analyses lui sont propres. Aucun utilisateur
  ne voit ni ne peut modifier les données d'un autre, même en ciblant un
  identifiant directement. Un nouvel utilisateur démarre avec ses propres
  catégories et un compte courant.

- **Transferts internes neutres** — une catégorie « Transfert interne » pour tes
  virements entre comptes personnels (courant ↔ Livret A). Ces mouvements ne sont
  ni dépense ni revenu : ils sont exclus de tous les calculs de flux (dépenses,
  revenus, pouls, journal, budgets), mais restent comptés dans le reste à vivre
  (l'argent sort réellement du compte courant). Ton épargne se lit au solde du
  Livret A, pas au flux des virements.
- **Couleurs de catégories modifiables** — en mode « gérer » dans l'onglet
  Transactions, clique sur la pastille de couleur d'une catégorie pour choisir
  la tienne, au lieu de subir l'aléatoire.
- **Reste à vivre projeté** — un chiffre en tête du tableau de bord : ce qu'il
  reste sur le compte une fois retirés les prélèvements récurrents encore
  attendus d'ici la fin du mois. Répond à « combien puis-je dépenser sans
  mauvaise surprise ? ». Chaque euro retiré correspond à un prélèvement réel
  identifié (jamais une estimation), et le détail est affiché.

- **Journal d'événements** — l'écran d'accueil de thémis. Plutôt qu'un tableau
  de bord à interpréter, une chronologie de faits marquants : revenu reçu,
  abonnement en hausse, budget tenu, prélèvement à venir. Chaque événement peut
  porter une action (créer une règle, voir les transactions). L'app raconte tes
  finances au lieu de te laisser les déchiffrer.

## Interface

Design « outil premium » : fond clair épuré, accent indigo unique (#5B5BD6), typographie Inter, surfaces en cartes à ombres douces (aucune bordure dure), rayons harmonisés, micro-interactions rapides. Aucune police externe (Inter et IBM Plex Mono sont empaquetées).

## Fonctionnement

```
PDF du relevé ──► extraction native ── pas de couche texte ? ──► OCR (pdftoppm + tesseract)
                       │                                              │
                       └────────────► parseur local ◄─────────────────┘
                          (analyse positionnelle des colonnes Débit/Crédit)
                       │
                       ├─ validation comptable : solde initial + Σ transactions = solde final
                       ├─ réconciliation : arbitrage des lectures OCR ambiguës par l'équation comptable
                       │
                       └─ si échec ou validation KO ──► fallback API Claude (texte extrait → JSON)
                                                              │
                    PostgreSQL ◄── déduplication (hash) ◄─────┘
                       │
                       ├──► catégorisation automatique par règles (enrichissables depuis l'UI)
                       ├──► dashboard React (KPIs, ruban des catégories, évolution 12 mois)
                       └──► analyse IA mensuelle (agrégats anonymes uniquement)
```

## Architecture (une seule image)

thémis se déploie en **une seule image** : le backend Go sert à la fois l'API
(`/api/*`) et l'interface web compilée (tout le reste, avec fallback SPA), sur
le port 8080. Il n'y a **pas de reverse proxy** ni de conteneur nginx séparé.

> Historique : les toutes premières versions utilisaient deux conteneurs
> (nginx pour le front + reverse proxy `/api`, et le backend). C'était une
> source de bugs — il fallait reconstruire les deux images à chaque
> changement, et une désynchronisation donnait « pas de page de connexion ».
> L'image unique supprime ce problème : un seul artefact, jamais désynchronisé.
> Le proxy n'était donc pas nécessaire, c'était un reliquat de l'archi initiale.

Si ton cluster impose un ingress/reverse proxy en amont (TLS, auth), il route
simplement **tout** vers le service thémis sur :8080 — pas besoin de séparer
`/api` du reste.

## Stack

- **Backend** : Go 1.22+ (stdlib http, `lib/pq`, `ledongthuc/pdf`), migrations SQL embarquées, arrêt propre sur SIGTERM.
- **Frontend** : React 18 + Vite, Recharts. Servi par nginx, ou par le backend lui-même (copier `frontend/dist` dans `backend/static` pour une image unique).
- **Base** : PostgreSQL 14+.
- **IA (optionnelle)** : **Ollama en local** (recommandé pour des données bancaires — rien ne quitte la machine) ou API Anthropic. Fallback de parsing + conseils mensuels. Sans fournisseur, l'appli fonctionne, seules ces deux fonctions sont désactivées.

## Tu ne vois pas les nouveautés / pas de page de connexion ?

C'est presque toujours un **frontend pas reconstruit**. Le front est compilé
en fichiers statiques : redéployer le backend seul ne le met pas à jour.

Depuis cette version, le **backend sert lui-même l'interface** (image unique,
UI + API sur le port 8080) — plus aucun risque de désynchronisation :

```bash
docker compose build --no-cache app   # reconstruit frontend PUIS backend
docker compose up -d
# → http://localhost:8080  (page de connexion au premier lancement)
```

Sur ton cluster : reconstruis et pousse l'image `backend/Dockerfile` (son
contexte de build est la racine du repo), puis force le rollout
(`kubectl rollout restart deploy/themis`). Vérifie la version servie avec
`curl https://…/api/auth/status` → doit répondre `{"setup_required":true}`
tant qu'aucun compte n'est créé.

## Démarrage rapide (dev)

```bash
# 1. Base
docker compose up -d db

# 2. Backend
cd backend
DATABASE_URL="postgres://themis:themis@localhost:5432/themis?sslmode=disable" \
ANTHROPIC_API_KEY="sk-ant-..." \
go run ./cmd/server

# 3. Frontend (proxy /api → :8080 déjà configuré)
cd frontend && npm install && npm run dev
```

Ou tout-en-un : `ANTHROPIC_API_KEY=sk-ant-... docker compose up --build` → http://localhost:3000

## Configuration (env)

| Variable            | Défaut                          | Rôle                                        |
|---------------------|---------------------------------|---------------------------------------------|
| `PORT`              | `8080`                          | Port HTTP du backend                        |
| `DATABASE_URL`      | `postgres://themis:themis@localhost:5432/themis?sslmode=disable` | Connexion PostgreSQL |
| `AI_PROVIDER`       | *(auto)*                        | `ollama`, `anthropic` ou `none`. Auto : ollama si `OLLAMA_URL` est défini, sinon anthropic si clé, sinon none |
| `OLLAMA_URL`        | *(vide)*                        | URL d'une instance Ollama (ex. `http://ollama:11434`) → IA 100 % locale |
| `OLLAMA_MODEL`      | `qwen2.5:14b`                   | Modèle Ollama (`ollama pull …` au préalable) |
| `ANTHROPIC_API_KEY` | *(vide)*                        | Active l'API Claude (cloud)                 |
| `ANTHROPIC_MODEL`   | `claude-sonnet-4-6`             | Modèle Claude utilisé                       |
| `AI_AUTO_CATEGORIZE`| `true`                          | Catégorisation IA auto à chaque import (`false` pour désactiver) |

### IA locale avec Ollama (recommandé)

Ce sont des données bancaires : autant que l'IA tourne chez toi. Le backend
parle à n'importe quelle instance [Ollama](https://ollama.com) :

```bash
# en dev
ollama pull qwen2.5:14b
OLLAMA_URL=http://localhost:11434 go run ./cmd/server

# avec docker compose
docker compose --profile local-ai up -d
docker compose exec ollama ollama pull qwen2.5:14b
OLLAMA_URL=http://ollama:11434 docker compose up backend
```

Sur ton cluster : un Deployment Ollama (GPU si dispo, sinon CPU — plus lent
mais fonctionnel) + `OLLAMA_URL=http://ollama.<ns>.svc:11434` sur le backend.
Le démarrage vérifie que l'instance répond et que le modèle est présent, et
`/api/health` expose `ai_provider`.

Choix du modèle : `qwen2.5:14b` ou `mistral-small` donnent une extraction
JSON fiable pour le parsing de secours ; `llama3.1:8b` suffit largement pour
les conseils mensuels si ta machine est modeste. Le parsing force
`format:"json"` côté Ollama : la réponse est du JSON valide garanti.

À noter : avec le parseur positionnel + OCR intégrés, le fallback IA ne sert
que pour les formats de relevés vraiment exotiques — la plupart des imports
n'utilisent aucune IA du tout.

#### Ollama distant sur le Mac

Pour éviter de faire tourner Ollama dans un conteneur Docker (plus lent sur
macOS sans GPU), tu peux utiliser une instance Ollama installée nativement
sur ton Mac hôte. Le backend et le Knowledge Graph pointeront vers elle via
`host.docker.internal`.

1. **Trouver l'IP privée du Mac** (optionnel, `host.docker.internal` suffit) :
   ```bash
   ipconfig getifaddr en0   # Wi-Fi
   ipconfig getifaddr en1   # Ethernet
   ```

2. **Configurer Ollama pour écouter sur toutes les interfaces** :
   ```bash
   OLLAMA_ORIGINS="*" ollama serve
   ```
   Ou définis la variable d'environnement `OLLAMA_HOST=0.0.0.0:11434` avant
   de lancer `ollama serve`.

3. **Définir dans `.env`** (à la racine du repo) :
   ```bash
   REMOTE_OLLAMA_BASE_URL=http://host.docker.internal:11434
   ```
   Le suffixe `/v1` est ajouté automatiquement par le compose pour le service
   Knowledge Graph ; le backend Go utilise l'URL brute. Une seule variable suffit — elle est propagée aux deux services.

4. **Lancer sans le profile `local-ai`** (le conteneur ollama n'est pas démarré) :
   ```bash
   docker compose up -d
   ```

5. **Vérifier** :
   ```bash
   curl http://localhost:8080/api/health
   # → doit répondre avec "ai_provider": "ollama"
   ```

> **Note réseau Docker** : `host.docker.internal` n'est pas disponible sous
> Docker Engine classique (Linux).
> - **Docker Desktop (macOS / Windows)** — utilise `host.docker.internal` comme indiqué ci-dessus.
> - **Docker Engine (Linux)** — ajoute dans `docker-compose.yml` :
>   ```yaml
>   services:
>     backend:
>       extra_hosts:
>         - "host.docker.internal:host-gateway"
>   ```
>   ou, si tu préfères une IP fixe, trouve l'IP de la machine hôte avec
>   `hostname -I` et remplace `host.docker.internal` par cette adresse.

## API

| Méthode | Route                     | Rôle |
|---------|---------------------------|------|
| GET     | `/api/health`             | Liveness/readiness (+ état IA) |
| POST    | `/api/statements`         | Upload PDF (multipart `file`, 20 Mo max) |
| GET     | `/api/statements`         | Historique des relevés |
| DELETE  | `/api/statements/{id}`    | Supprime relevé + transactions |
| GET     | `/api/transactions`       | Liste (`?month=YYYY-MM`, `?category=id\|none`, `?q=recherche`) |
| PATCH   | `/api/transactions/{id}`  | Change la catégorie |
| GET/POST| `/api/categories`         | Catégories |
| GET/POST| `/api/rules`              | Règles (`apply:true` = rétroactif) ; DELETE `/api/rules/{id}` |
| GET     | `/api/summary?months=N`   | Séries mensuelles + répartition par catégorie |
| GET/POST| `/api/budgets`            | Budgets par catégorie (`?month=`) ; DELETE `/api/budgets/{id}` |
| GET     | `/api/recurring`          | Abonnements et dépenses récurrentes détectés (`?min_months=`) |
| POST    | `/api/categorize/ai`      | Classe par IA les transactions sans catégorie (`{"month": "YYYY-MM"}` optionnel) |
| GET     | `/api/transactions/export`| Export CSV (`;`, décimales FR) (`?month=`) |
| GET/PUT | `/api/settings/{key}`     | Réglages persistés (layout du dashboard…) |
| GET/PUT | `/api/admin/config`       | Config IA effective (fournisseur, modèles) — appliquée à chaud |
| POST    | `/api/admin/ai/test`      | Teste la connexion au fournisseur (celui du corps, ou l'actif) |
| GET     | `/api/admin/ollama/models`| Modèles disponibles sur l'instance Ollama (`?url=`) |
| GET     | `/api/admin/stats`        | Compteurs, taille de la base, bornes de l'historique |
| POST    | `/api/admin/reapply-rules`| Réapplique les règles aux transactions non classées |
| POST    | `/api/admin/purge`        | Efface relevés/transactions/analyses (`{"confirm":"TOUT EFFACER"}`) |
| GET/POST| `/api/analyses/{YYYY-MM}` | Conseils IA du mois |

## Relevés sans couche texte (OCR)

Certaines banques — le Crédit Agricole notamment — génèrent des PDF où chaque
caractère est dessiné en **image bitmap** : aucune extraction de texte n'est
possible, même avec poppler. Le backend le détecte automatiquement et bascule
sur un pipeline OCR entièrement local :

1. rasterisation à 480 dpi (`pdftoppm`) — multiple entier des 240 ppi des
   glyphes bitmap, donc sans interpolation floue ;
2. OCR positionnel (`tesseract -l fra`, sortie TSV) puis reconstruction des
   lignes par regroupement vertical ;
3. **seconde passe** ciblée sur chaque montant (crop + psm 7 + whitelist
   chiffres), les deux lectures étant conservées ;
4. **réconciliation** : si la validation comptable échoue, le parseur essaie
   les combinaisons de lectures ambiguës et applique celle qui équilibre
   exactement le relevé.

Dépendances runtime : `poppler-utils`, `tesseract-ocr`, `tesseract-ocr-fra`
(déjà dans l'image Docker du backend). Sans elles, l'appli fonctionne mais ces
relevés partent directement au fallback IA. Compter ~25-30 s d'import pour un
relevé de 3 pages en OCR — sans importance pour une opération mensuelle.

## Adapter le parseur à ta banque

Le parseur local est générique pour les relevés français : colonnes Débit/Crédit repérées par
l'en-tête, dates `jj/mm[/aaaa]`, montants `1 234,56`, soldes « ancien/nouveau solde ».
Pour tester rapidement sur un de tes relevés :

```bash
cd backend && go run ./cmd/parsetest /chemin/vers/releve.pdf
```

Si le format de ta banque diffère, ajuste les regex en tête de
`backend/internal/parser/local.go` (`reDate`, `reAmount`, `reSoldeOld/New`, `rePeriod`).
En attendant, le fallback IA prend le relais dès qu'une clé API est fournie —
la validation comptable te dit dans l'UI si un import mérite vérification.

## Revue de code & durcissement

Passage de revue complet réalisé :
- **Sécurité** : cookie de session marqué `Secure` en production via
  `COOKIE_SECURE=true` ; nettoyage horaire des sessions expirées ; validation
  stricte des IDs (entiers) dans le graphe par catégories ; changement de mot
  de passe personnel qui invalide les autres sessions ouvertes.
- **Portabilité** : `GET /api/export` produit une archive JSON complète
  (comptes, transactions avec catégories/domaines/tags, règles, budgets,
  crédits, taxes) — sauvegarde et RGPD. Accessible depuis l'onglet Admin.
- **Chargement** — l'interface est livrée en un seul bundle (~140 kB
  compressé) : aucun fragment chargé à la volée, donc aucun risque d'« écran
  blanc » au changement d'onglet.
- **Tests** : couverture ajoutée sur le hachage PBKDF2, la validation des mots
  de passe et le calcul d'échéancier de crédit, en plus des tests parseur.
- **UX** : système de toasts empilables et animés (succès/erreur/info),
  transitions de page, micro-interactions au survol.

## Authentification & rôles

Au premier lancement, thémis demande la **création d'un compte administrateur**
(mot de passe : 10 caractères minimum, lettres + chiffres, haché en
**PBKDF2-HMAC-SHA256**, 210 000 itérations, bibliothèque standard uniquement —
aucune dépendance crypto externe). Ensuite, toute l'API exige une session
(cookie `HttpOnly`, `SameSite=Lax`, 30 jours) hormis `/api/health` (probes) et
`/api/auth/*`. Limitation anti-force-brute : 5 essais → pause de 30 s.

Deux rôles : **admin** (accès à l'onglet Admin, gestion des utilisateurs) et
**utilisateur** (tout le reste). L'onglet Admin permet de créer/supprimer des
comptes ; le dernier administrateur ne peut pas être supprimé.

## Comptes, crédits, domaines

- **Multi-comptes** : compte courant, Livret A, LDDS, PEL, compte-titres…
  chacun avec son type, son taux et son plafond. Les relevés PDF alimentent le
  solde des comptes courants (champ `account_id` à l'upload) ; pour les livrets
  et la bourse, un **pointage manuel** de solde suffit. Le patrimoine total est
  agrégé.
- **Crédits & taxes** : suivi des mensualités restantes de chaque crédit
  (barre de progression, date de fin calculée), taxes avec **compte à rebours
  avant échéance** et alerte sous 30 jours, et une bannière de synthèse de la
  dette totale.
- **Graphe par catégories** : sélection multiple de catégories, période
  (mois / trimestre / année), rendu en barres empilées ou en courbes.
- **Simulation budgétaire** : à partir de tes moyennes réelles sur 3 mois,
  teste des réductions par catégorie (curseurs), de nouvelles dépenses et un
  emprunt (mensualité et coût total calculés), avec le reste à vivre projeté.
- **Domaine (taxonomie unique) & tags** — la catégorie a été supprimée : le
  **domaine** de vie est désormais le seul axe de classement structurant. Petit
  ensemble fixe (Logement, Vie pro, Vie perso, Vie amoureuse, Famille, Loisirs),
  un seul par transaction, pastille colorée. Les **budgets**, le **graphe**, les
  **règles automatiques** et l'**auto-tri IA** opèrent tous sur le domaine. Les
  **tags** restent libres et multiples, avec autocomplétion anti-doublon.
- **Gestion des règles** — un panneau « règles automatiques » (onglet Transactions) liste toutes les règles motif → domaine, avec le nombre de transactions couvertes et le domaine cible coloré. Chaque règle peut être modifiée (motif et/ou domaine, réappliquée aussitôt) ou supprimée. Un même motif ne peut exister qu'une fois : recréer une règle existante met à jour sa catégorie au lieu de créer un doublon. Une règle peut aussi poser des tags automatiquement (ex. « Bouygues Telecom » → catégorie Télécommunications + tags factures, abonnement, mobile). Les règles s'éditent en cliquant directement sur le motif ou la catégorie (sauvegarde immédiate).


## Page d'administration

L'onglet **Admin** permet de :

- **Changer de fournisseur/modèle d'IA à chaud** : bascule none ↔ Ollama ↔ API
  Claude sans redémarrer le backend. Pour Ollama, un bouton liste les modèles
  réellement installés sur l'instance ; « Tester la connexion » valide la
  config *avant* de l'appliquer. La configuration est persistée en base et
  **prend le pas sur les variables d'environnement** aux démarrages suivants.
  La clé API n'est jamais renvoyée par l'API (seul un indicateur « définie »).
- **Consulter les statistiques** : relevés, transactions (et % classé),
  règles, taille de la base, bornes de l'historique.
- **Maintenance** : réappliquer les règles aux transactions non classées ;
  zone dangereuse pour purger les données (confirmation textuelle exigée,
  catégories/règles/budgets/réglages conservés).

⚠️ L'appli n'a pas d'authentification (conçue pour un usage personnel
auto-hébergé) : si tu l'exposes au-delà de ton réseau, protège-la au niveau
de l'ingress (auth basique, OAuth2-proxy, VPN…) — la page admin comme le reste.

## Tableau de bord personnalisable

Le dashboard est une grille de **widgets** : bouton « Personnaliser » pour
ajouter, retirer, réordonner (↑/↓) ou redimensionner (demi / pleine largeur)
chaque bloc. La disposition est **persistée côté serveur** (`settings`),
donc identique sur tous tes appareils.

Widgets disponibles : indicateurs du mois (revenus, dépenses, net, taux
d'épargne, avec variation vs mois précédent), courbes revenus/dépenses/net
interactives (légende cliquable pour masquer une série, zoom par brosse
au-delà de 14 mois), cumul net, donut des catégories (survol interactif),
tendance d'une catégorie au choix, top dépenses, mini-budgets, abonnements.

## Budgets & abonnements

- **Budgets** : un plafond mensuel par catégorie ; l'UI affiche la jauge de
  consommation, le dépassement, et — pour le mois en cours — une projection
  fin de mois au prorata des jours écoulés.
- **Abonnements** : détection automatique des dépenses récurrentes. Les
  libellés sont réduits à l'identité du marchand (préfixes bancaires, dates
  et références supprimés) puis regroupés ; un montant stable (< 15 %
  d'écart) sur ≥ 2 mois est marqué abonnement probable. Le total mensuel des
  abonnements est affiché — souvent la découverte la plus utile de l'appli.

## Catégorisation IA

Les règles classent les marchands connus ; l'IA s'occupe du reste :

- **À l'import** : après les règles, les transactions restées sans catégorie
  sont soumises au fournisseur d'IA configuré (désactivable via
  `AI_AUTO_CATEGORIZE=false`). Le nombre de transactions classées est indiqué
  dans la notification d'import.
- **À la demande** : bouton « Classer avec l'IA » dans l'onglet Transactions
  (respecte le filtre de mois), utile pour rattraper l'historique.

Garde-fous : l'IA ne remplace **jamais** une catégorie déjà posée (règle ou
choix manuel), n'utilise que des noms de catégories existants (les noms
inventés sont ignorés), et le prompt lui demande d'omettre plutôt que de
classer au hasard. Tu peux toujours corriger dans l'UI — et en profiter pour
créer une règle, qui prendra le pas sur l'IA aux imports suivants.

**Confidentialité** : contrairement aux conseils mensuels (agrégats anonymes),
cette fonction envoie les **libellés** des transactions au fournisseur. Avec
Ollama, tout reste local ; avec l'API Anthropic, à toi de juger — ou de
désactiver avec `AI_AUTO_CATEGORIZE=false`.

## Règles de catégorisation

Les catégories se créent **à la volée** depuis le sélecteur d'une transaction
(« ＋ Nouvelle catégorie… ») ou depuis le formulaire de budget — pas besoin de
passer par une page d'administration. La création est idempotente : recréer un
nom existant renvoie la catégorie existante.

Motif = sous-chaîne insensible à la casse (`CARREFOUR`), ou regex si préfixé `re:`
(`re:CB\s+AMAZON(?!\s+PRIME)`). Les plus longs sont testés d'abord. L'UI propose de créer
une règle à chaque re-catégorisation manuelle, avec application rétroactive aux
transactions non classées.

## Design & animations

Direction « guichet réédité » : un objet imprimé de banque française —
papier vergé, encres de guichet, bandeau **guilloché** en tête (le motif
gravé des chèques), filets fins sans arrondi, montants en mono tabulaire,
et un **tampon Validé** sur les relevés dont l'équation comptable est exacte.
Typographies Fraunces / Archivo / IBM Plex Mono **embarquées dans le build**
(Fontsource) : l'interface ne charge rien depuis un CDN, cohérent avec une
appli de données bancaires. Sous le guilloché court **la grecque** — le méandre, clin d'œil à Thémis.

Animations orchestrées : séquence d'impression au chargement (le guilloché se
grave de gauche à droite, la grecque suit, la marque se tamponne, la fente et
les onglets se lèvent en cascade), **claquement plein écran du tampon
« Validé »** quand un relevé est équilibré au centime, compteurs déroulants
sur les indicateurs, révélation en cascade des widgets et des lignes,
balayage des jauges, voile de dépôt plein écran au glisser-déposer.
`prefers-reduced-motion` désactive tout, responsive mobile.

## Confidentialité

- Parsing local par défaut : rien ne sort de ton cluster tant que le parseur valide.
- Avec **Ollama**, même les fonctions IA sont 100 % locales : rien ne sort, point.
- Avec l'API Anthropic : le fallback de parsing envoie le **texte extrait** du relevé (à toi de juger) ; les **conseils mensuels** n'envoient que des agrégats anonymes (totaux par catégorie), jamais les libellés.

## Notes pour ton déploiement (tu gères l'infra, mais au cas où)

- `/api/health` fait un ping DB → utilisable en readinessProbe.
- SIGTERM → drain 15 s : compatible rolling updates.
- Image backend debian-slim non-root (l'OCR exige poppler + tesseract, exit distroless), binaire statique.
- Backups Postgres fortement recommandés : ce sont tes finances.
