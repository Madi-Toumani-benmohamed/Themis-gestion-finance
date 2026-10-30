# Chargement des pages et activité du dashboard

## Mesure du build

Comparaison sur le même environnement et avec les mêmes dépendances, via
`npm run build -- --manifest`. Les valeurs gzip utilisent le même niveau de
compression Python des deux côtés et additionnent les dépendances statiques.

| JavaScript requis au démarrage | Avant | Après | Réduction |
| --- | ---: | ---: | ---: |
| Fichiers minifiés | 1 080 220 octets | 355 990 octets | 67,0 % |
| Gzip | 299 226 octets | 115 216 octets | 61,5 % |

Après connexion et chargement du dashboard, les dépendances nécessaires totalisent
845 925 octets au lieu de 1 080 220 (−21,7 %), ou 243 760 octets gzip au lieu de
299 226 (−18,5 %).

Il s’agit du code chargé avant connexion, pas d’une mesure du temps de réponse
ni d’une promesse de FPS. Après connexion, le dashboard et sa bibliothèque de
graphiques sont téléchargés. Les autres pages restent différées jusqu’à leur
utilisation ou au survol/focus de leur entrée de navigation.

## Changements

- `PageLoader.jsx` charge les pages avec React.lazy et Suspense, partage les
  promesses de préchargement et présente un état de chargement accessible.
  Un échec de téléchargement affiche une action de rechargement au lieu de
  laisser l’application vide. La navigation et le shell restent disponibles.
- `App.jsx` conserve l’authentification immédiatement disponible et précharge
  uniquement la destination approchée au survol ou au focus. Aucun téléchargement
  de toutes les pages en tâche de fond n’est ajouté.
- `IntelligenceHighlights.jsx` suspend le suivi périodique des jobs lorsque
  `document.hidden` est vrai, le reprend dès le retour dans l’onglet et empêche
  le chevauchement des requêtes de suivi. Les calculs serveur continuent.

Le chargement différé ajoute une requête lors de la première visite d’une page.
Le module est ensuite réutilisé. Il n’ajoute aucun cache de données financières,
aucune modification d’API ou de calcul, ni aucune dépendance.

## Vérification

- `npm run test:loading` construit puis sert le véritable build de production.
  Les fixtures synthétiques vérifient les modules effectivement téléchargés
  avant connexion, le préchargement au focus, la réutilisation du dashboard,
  un téléchargement interrompu et la récupération après rechargement.
- `tests/intelligence.browser.cjs` vérifie aussi l’absence de requêtes de suivi
  pendant quinze secondes simulées dans un onglet masqué et leur reprise.
- Ces vérifications complètent les parcours métier, responsive, Axe et Motion
  existants dans `npm run test:browser`.
