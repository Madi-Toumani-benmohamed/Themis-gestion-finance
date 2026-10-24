# thémis.

Thémis est une application auto-hébergée de gestion de finances personnelles : import de relevés bancaires PDF et CSV, classement des transactions, budgets, tableaux de bord et assistant IA optionnel.

## Fonctionnalités

- **Import PDF et CSV** : extraction du texte, OCR local si nécessaire, déduplication et validation comptable des relevés lorsque les soldes de contrôle sont disponibles. Les CSV restent à vérifier faute de ces soldes.
- **Classement des transactions** : domaines/catégories, tags multiples, règles automatiques, renommage des marchands et tri en pile. Les propositions IA restent séparées et nécessitent une validation explicite avant affectation.
- **Dashboard personnalisable** : navigation mensuelle, synthèse financière, revenus/dépenses, graphiques Recharts, principales dépenses, budgets, récurrences et journal d’événements.
- **Points d’attention** : dépassements budgétaires, dépenses potentiellement inhabituelles et signaux sur les paiements récurrents, calculés par des règles déterministes.
- **Analyse et simulation** : évolution des dépenses, croisements catégories/tags, suivi des comptes, crédits et taxes, simulation budgétaire.
- **Assistant IA** : conversation sur les données financières et service de mémoire Graphiti/FalkorDB pour enrichir le contexte.
- **Interface** : thèmes clair et sombre, design sobre à accent indigo `#5B5BD6`, polices embarquées, disposition responsive et palette de commande `Ctrl/Cmd+K`.
- **Multi-utilisateurs** : authentification par session, rôles administrateur/utilisateur et données rattachées à leur propriétaire. Création du premier administrateur au premier lancement.

### Vérifier un import avant son enregistrement

Le dépôt d’un PDF ou CSV ouvre désormais un aperçu avant toute écriture de relevé ou de transaction. Il présente le compte cible, la période extraite (ou les dates des opérations pour un CSV), les opérations détectées, les doublons et le contrôle comptable lorsque les soldes sont disponibles. Les opérations sont filtrables et paginées ; comparer toujours l’aperçu au document original, car le parseur peut ignorer des lignes non exploitables.

**Confirmer l’import** utilise les données analysées côté serveur, sans analyser une seconde fois le document. **Annuler ce fichier** ne crée ni relevé ni transaction. En import multiple, chaque fichier est examiné séparément. Les doublons suivent les règles existantes et sont revérifiés à l’insertion : les nombres définitifs peuvent varier si un autre import a eu lieu entre-temps.

Les aperçus sont conservés uniquement en mémoire serveur pendant 15 minutes, liés à l’utilisateur et au compte cible. Ils disparaissent après annulation, confirmation, expiration ou redémarrage. Leur nombre et leur mémoire totale sont limités ; les aperçus de plus de 20 000 opérations sont refusés. L’analyse initiale peut déjà utiliser le fournisseur IA configuré ; les règles et la génération de suggestions IA s’exécutent après confirmation. Les propositions IA restent séparées des catégories et tags appliqués : leur validation se fait explicitement dans le détail d’une transaction.

L’API historique `POST /api/statements` reste un import direct pour les clients existants. L’interface utilise `POST /api/statements/previews`, puis `POST /api/statements/previews/{token}/confirm` ou `DELETE /api/statements/previews/{token}`. Les jetons sont éphémères et à usage unique. Pour un déploiement multi-instance, ce stockage mémoire nécessite que l’aperçu et sa confirmation atteignent la même instance.

### Fiabilité des données

Dans **Relevés**, le centre de fiabilité permet de choisir un mois et d’examiner la couverture compte par compte. Il distingue les périodes documentées par des relevés validés, les intervalles non documentés et les imports à vérifier. Les relevés dont la période est inconnue restent visibles sans être attribués arbitrairement au mois choisi.

Pour le mois courant, la vérification s’arrête à aujourd’hui. Un intervalle non documenté ne prouve pas l’absence de transactions. Le dernier import, toutes périodes confondues pour le compte, reste distinct de la dernière transaction du mois examiné. Un lien dans la synthèse du dashboard ouvre directement le centre sur le mois consulté.

### Lire la synthèse financière

La synthèse distingue trois contextes :

| Période | Présentation |
| --- | --- |
| Mois en cours | Solde des flux à date, explicitement provisoire. Un résultat négatif seul ne déclenche pas de diagnostic alarmiste. |
| Mois historique documenté | Résultat des flux du mois, avec diagnostic et comparaisons lorsque les périodes de référence sont documentées. |
| Mois historique incomplet | Résultat des flux disponibles et statut « Données incomplètes », sans bilan définitif. |

Les revenus de fin de mois ne sont ni estimés ni supposés reçus. Les alertes indépendantes du calendrier de salaire restent visibles.

Le résultat correspond aux **revenus moins les dépenses**, selon les conventions de flux de l’application, avec exclusion des transferts internes. Il ne représente pas le solde bancaire disponible.

Pour cette synthèse, la couverture historique exige des intervalles continus de relevés validés pour chaque compte connu. Des relevés successifs clôturés le 21 peuvent ainsi documenter un mois civil ; un trou ou des métadonnées absentes conduisent à une qualification prudente. La dernière transaction et le dernier import sont affichés séparément et ne prouvent pas, à eux seuls, la complétude. Les autres widgets conservent leurs propres règles de période.

## Architecture

```text
Navigateur → app :8080
              ├── API Go + frontend React compilé
              ├── PostgreSQL
              ├── fournisseur IA configuré (optionnel)
              └── knowledge-graph :8100 → FalkorDB + Ollama
```

L’application web et l’API sont livrées dans **la même image**, construite avec `backend/Dockerfile` depuis la racine du dépôt. Le backend sert les fichiers statiques et les routes `/api/*` ; aucun nginx séparé n’est nécessaire.

La stack Compose principale comprend `app`, `db`, `knowledge-graph` et `falkordb`. Le service `ollama` est activé par le profil `local-ai`. Le graphe de connaissances a besoin d’un serveur Ollama accessible pour ses opérations IA.

| Partie | Technologies |
| --- | --- |
| Backend | Go 1.22 minimum ; build Docker et CI en Go 1.23 ; HTTP standard, `lib/pq`, migrations SQL embarquées |
| Frontend | React 18, Vite 8, Recharts ; build Docker et CI avec Node.js 22 |
| Base | PostgreSQL |
| Mémoire | Python 3.12, FastAPI, Graphiti, FalkorDB |
| OCR | Poppler et Tesseract français, inclus dans l’image runtime Debian |

Le conteneur applicatif s’exécute avec un utilisateur non-root. Les volumes Compose conservent PostgreSQL, FalkorDB et les modèles Ollama.

## Démarrage avec Docker Compose

Prérequis : Docker avec le plugin Compose et accès aux images configurées.

```bash
cp example.env .env
```

Adapter `.env` avant le premier lancement :

- Définir `POSTGRES_USER`, `POSTGRES_PASSWORD` et `POSTGRES_DB`.
- Choisir une image PostgreSQL disponible : l’exemple référence `themis-db:2.3`, une image personnalisée. Pour une nouvelle installation standard, utiliser par exemple `POSTGRES_IMAGE=postgres:16-alpine`.
- Garder `POSTGRES_PORT=5432` pour le port interne de PostgreSQL dans la stack principale.
- Définir `APP_PORT=8080`, ou le port hôte souhaité.

```bash
docker compose up -d --build
docker compose ps
curl http://localhost:8080/api/health
```

Ouvrir **http://localhost:8080** (ou le port `APP_PORT` choisi), puis créer le premier administrateur. La base principale n’est pas publiée sur le réseau hôte par défaut.

### Mettre à jour l’application

```bash
docker compose up -d --build
```

Pour reconstruire uniquement l’application :

```bash
docker compose up -d --build app
```

Le build reconstruit le frontend et le backend ensemble. Les migrations embarquées sont appliquées au démarrage. Sauvegarder la base avant les mises à jour ; ne pas remplacer une image PostgreSQL existante par une autre version majeure sans procédure de migration.

### Ollama dans Docker

Dans `.env`, laisser `REMOTE_OLLAMA_BASE_URL` vide et définir :

```dotenv
OLLAMA_URL=http://ollama:11434
OLLAMA_MODEL=qwen2.5:14b
```

Puis lancer le profil et télécharger les modèles :

```bash
docker compose --profile local-ai up -d --build
docker compose exec ollama ollama pull qwen2.5:14b
docker compose exec ollama ollama pull nomic-embed-text
```

Le profil démarre Ollama ; `OLLAMA_URL` configure explicitement l’application Go. Le service mémoire utilise par défaut `http://ollama:11434/v1` et le modèle d’embedding `nomic-embed-text`.

### Ollama sur une autre machine ou sur l’hôte

Définir dans `.env` une URL accessible depuis les conteneurs, **sans suffixe `/v1`** :

```dotenv
REMOTE_OLLAMA_BASE_URL=http://host.docker.internal:11434
```

Cette variable a priorité sur `OLLAMA_URL` pour l’application ; Compose ajoute `/v1` pour le service mémoire. Installer les modèles de génération et d’embedding sur cette instance, puis lancer `docker compose up -d --build` sans profil `local-ai`.

Avec Docker Desktop, `host.docker.internal` désigne l’hôte. Sous Docker Engine Linux, utiliser une adresse accessible ou ajouter un mapping `host.docker.internal:host-gateway` aux services **app et knowledge-graph** via un fichier Compose de surcharge.

Ollama doit écouter sur une interface accessible aux conteneurs. Limiter cet accès au réseau prévu ; modifier les origines autorisées ne remplace pas la configuration de l’adresse d’écoute.

### Variante Supabase

Une configuration utilisant une base externe est disponible dans [compose-supabase/docker-compose.yml](compose-supabase/docker-compose.yml). Adapter les variables `SUPABASE_*` de l’exemple à l’instance réelle avant utilisation :

```bash
docker compose -f compose-supabase/docker-compose.yml --env-file .env up -d --build
```

La variante actuelle utilise `SUPABASE_HOST` comme identifiant de connexion dans l’URL PostgreSQL, et `SUPABASE_POOLER` comme hôte réseau ; `SUPABASE_USER` n’y est pas utilisé. Vérifier ces valeurs auprès de l’instance cible. Cette variante charge aussi le fichier racine `.env` via `env_file` : il doit exister, y compris lors de la validation Compose.

## Configuration

Le fichier `.env` sert à l’interpolation Docker Compose. Le backend lancé directement lit les variables de son environnement ; il ne charge pas automatiquement ce fichier.

| Variable | Rôle |
| --- | --- |
| `APP_PORT` | Port hôte de l’application dans Compose ; `PORT` contrôle le port du backend lancé directement (défaut `8080`). |
| `POSTGRES_IMAGE`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_PORT` | Configuration PostgreSQL et construction de `DATABASE_URL` dans la stack principale. |
| `DATABASE_URL` | Connexion utilisée par le backend. |
| `REMOTE_OLLAMA_BASE_URL` | URL Ollama utilisée par Compose pour l’application et la mémoire. |
| `OLLAMA_URL`, `OLLAMA_MODEL` | Instance et modèle Ollama du backend ; modèle par défaut `qwen2.5:14b`. |
| `AI_PROVIDER` | Fournisseur du backend : `none`, `ollama`, `mistral`, `freellm`, `anthropic` ou `openai`. |
| `OPENAI_API_KEY`, `OPENAI_MODEL` | Clé et modèle OpenAI (défaut `gpt-4.1-mini`), relayés par les deux configurations Compose. |
| `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL` | Configuration Anthropic. |
| `MISTRAL_API_KEY`, `MISTRAL_MODEL` | Configuration Mistral. |
| `FREELLM_API_KEY`, `FREELLM_URL`, `FREELLM_MODEL` | Fournisseur compatible OpenAI FreeLLM. |
| `AI_AUTO_CATEGORIZE` | Génération de suggestions IA à l’import (`true` par défaut), sans affectation automatique. Validation dans le détail de la transaction. |
| `MEMORY_ENABLED`, `MEMORY_SERVICE_URL` | Activation et adresse du service mémoire pour le backend. |
| `COOKIE_SECURE` | Activer avec `true` lorsque l’application est servie en HTTPS. |

Sans `AI_PROVIDER` explicite, le backend choisit Ollama si une URL est définie, puis Mistral, FreeLLM, Anthropic ou OpenAI selon les clés présentes, sinon `none`.

**Toutes les variables du backend ne sont pas relayées par le Compose principal.** Pour ajouter `COOKIE_SECURE`, `AI_PROVIDER` ou d’autres options, les déclarer dans `services.app.environment`, par exemple dans une surcharge Compose. Le Compose principal fixe actuellement `MEMORY_ENABLED=true` et l’adresse interne du service mémoire ; les placer seulement dans `.env` ne change pas ces valeurs.

Dans **Admin → Fournisseur d’IA**, l’option **OpenAI (cloud)** permet de saisir une clé API et un identifiant de modèle compatible avec l’API Responses. Le bouton de test envoie uniquement une requête synthétique (avec consommation de jetons). Laisser le champ de clé vide conserve la clé enregistrée ; l’API ne renvoie que son indicateur de présence. Aucune clé n’est nécessaire pour les tests automatisés, qui utilisent un serveur HTTP simulé.

Le client utilise `store: false` pour ne pas demander la conservation des réponses dans l’API Responses. Cela ne remplace pas les politiques de traitement des données du fournisseur ; les données nécessaires au parsing, à la catégorisation ou à l’assistant sont transmises à OpenAI lorsque ce fournisseur est choisi. Référence : [documentation officielle OpenAI](https://developers.openai.com/api/docs/guides/text).

La page **Admin** permet de configurer et tester l’IA à chaud. La configuration persistée en base prime sur les variables d’environnement aux démarrages suivants.

## Développement et validation

Pour développer hors Docker, prévoir une base PostgreSQL accessible sur l’hôte, Go, Node.js et, pour l’OCR, Poppler/Tesseract. Le service `db` du Compose principal n’expose pas de port hôte : ajouter une surcharge si cette base doit être utilisée depuis Go local.

Dans un premier terminal, depuis la racine :

```bash
cd backend
# Définir DATABASE_URL avec les identifiants de la base de développement.
AI_PROVIDER=none MEMORY_ENABLED=false go run ./cmd/server
```

Dans un second terminal :

```bash
cd frontend
npm ci
npm run dev
```

Vite affiche son URL de développement et redirige `/api` vers `http://localhost:8080`.

Commandes de validation depuis la racine :

```bash
(cd backend && go test -race ./... && go vet ./... && go build ./...)
node --test frontend/src/components/financialSummaryModel.test.mjs
npm --prefix frontend ci
npm --prefix frontend run build
docker compose --env-file example.env --profile local-ai config --quiet
docker compose -f compose-supabase/docker-compose.yml --env-file example.env config --quiet
```

Les tests de diagnostic vérifient notamment qu’un résultat provisoire négatif reste neutre, que les alertes sont conservées et que la couverture des relevés ne comporte pas de trou. La CI vérifie les tests Go, le build frontend, la syntaxe Python, l’image applicative et les deux configurations Compose.

Consulter [CODEBASE_GRAPH.md](CODEBASE_GRAPH.md) pour les fichiers clés, flux de données et conventions d’architecture.

## API : points d’entrée principaux

Les routes métier nécessitent une session authentifiée. Liste non exhaustive :

| Route | Usage |
| --- | --- |
| `GET /api/health` | État de l’application et connexion à la base. |
| `/api/auth/*` | Initialisation, connexion, déconnexion et gestion de la session. |
| `GET/POST /api/statements` | Historique et import des relevés. |
| `GET/PATCH /api/transactions[/{id}]` | Recherche et modification des transactions. |
| `GET /api/summary?months=24&month=YYYY-MM` | Agrégats mensuels et catégories. |
| `GET /api/hero?month=YYYY-MM` | Synthèse, flux, diagnostic et couverture documentée. |
| `GET /api/period?month=YYYY-MM` | Métadonnées de période ; son statut ne constitue pas la preuve de couverture de la synthèse. |
| `GET /api/anomalies?month=YYYY-MM` | Signaux détectés. |
| `GET /api/journal?month=YYYY-MM` | Événements de la période. |
| `GET /api/remaining?month=YYYY-MM` | Reste à vivre ou résultat historique selon le contexte. |
| `GET/POST /api/budgets` | Consultation et définition des budgets. |
| `POST /api/chat` | Assistant conversationnel. |
| `GET /api/export` | Export des données. |
| `GET/PUT /api/admin/config` | Configuration IA, accès administrateur. |

Les filtres et contrats exacts sont définis dans `backend/internal/api/` et utilisés par `frontend/src/api.js`.

## Confidentialité et exploitation

L’extraction native, l’OCR et les règles déterministes s’exécutent localement. Si le parsing local échoue, le fournisseur IA configuré peut recevoir le texte du relevé. La catégorisation peut envoyer les libellés et l’assistant les données nécessaires à ses outils : ne pas assimiler toutes les fonctions IA à des agrégats anonymes.

Avec une instance Ollama que vous contrôlez, ces traitements restent dans l’infrastructure choisie. Avec un fournisseur distant, des données sont transmises à ce fournisseur. La mémoire peut également conserver du contexte financier dans FalkorDB.

Ne pas committer `.env`, les relevés ou les exports bancaires. Sauvegarder PostgreSQL et, si nécessaire, les données du graphe. Pour une exposition publique, utiliser HTTPS et configurer `COOKIE_SECURE=true` dans l’environnement effectif de l’application. Le port du service mémoire est publié sur `8100` par le Compose principal : adapter son exposition réseau au déploiement.

Pour diagnostiquer un démarrage :

```bash
docker compose ps
docker compose logs --tail=100 app knowledge-graph
```

Les journaux peuvent contenir des informations sensibles : les examiner et les expurger avant partage.
