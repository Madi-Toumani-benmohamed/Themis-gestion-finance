# Règles et références fiscales V1

Vérification : 30 octobre 2026. Les règles officielles ne rendent pas le simulateur officiel. Chaque réponse porte l'année des revenus, l'année d'imposition, la version et les sources. Une année absente est refusée, sans extrapolation silencieuse.

| Revenus / imposition | Bornes par part (€) | Frais salariaux 10 % min / max (€) | Plafond demi-part / parent isolé (€) | Décote seul / couple (€) |
|---|---|---|---|---|
| 2024 / 2025 | 11497, 29315, 83823, 180294 | 504 / 14426 | 1791 / 4224 | 889 / 1470 |
| 2025 / 2026 | 11600, 29579, 84577, 181917 | 509 / 14555 | 1807 / 4262 | 897 / 1483 |

Taux : 0, 11, 30, 41 et 45 %. Décote : forfait moins 45,25 % de l'impôt brut, bornée à l'impôt ; montants déclaratifs/base, impôt et décote arrondis à l'euro le plus proche, fractions de 0,50 € vers l'euro supérieur. Seuil de recouvrement : impôt inférieur à 61 € non recouvré dans le périmètre sans crédits/réductions pris en charge ici. La tranche marginale exposée correspond à la branche active après plafonnement familial.

Salaires : utiliser le net fiscal de la fiche de paie avant frais professionnels, jamais brut, net social ou net payé. Les frais sont déduits par déclarant, plafonnés aux revenus pour le forfait. Les frais réels qualifiés sont annuels et exclusifs du forfait ; les déficits sont refusés. Une base finale déjà après abattements ne reçoit aucune seconde déduction.

Sources officielles :

- [Brochure DGFiP 2026, calcul pp369–382](https://www.impots.gouv.fr/www2/fichiers/documentation/brochure/ir_2026/pdf_som/21-calcul_impot_369a382.pdf), en particulier exemple p371, arrondis p372 et fiche facultative p379.
- [Brochure DGFiP 2025](https://www.impots.gouv.fr/www2/fichiers/documentation/brochure/ir_2025/pdf_integral/Brochure-IR-2025.pdf).
- [Barème 2026 et exemple RNI30000, Service-Public, vérifié 15 avril 2026](https://www.service-public.gouv.fr/particuliers/vosdroits/F1419).
- [BOI-IR-LIQ-20-10 du 7 avril 2026](https://bofip.impots.gouv.fr/bofip/2491-PGP.html/identifiant=BOI-IR-LIQ-20-10-20260407) ; [version du 14 avril 2025](https://bofip.impots.gouv.fr/bofip/2491-PGP.html/identifiant=BOI-IR-LIQ-20-10-20250414).
- [Frais professionnels revenus 2025](https://simulateur-ir-ifi.impots.gouv.fr/calcul_impot/2026/aides/frais.htm) ; [revenus 2024](https://simulateur-ir-ifi.impots.gouv.fr/calcul_impot/2025/aides/frais.htm).
- [Parent isolé](https://simulateur-ir-ifi.impots.gouv.fr/calcul_impot/2026/aides/parentisole.htm).

Références de tests indépendantes : Service-Public, célibataire RNI30000 en 2025, brut2103,99 € arrondi2104 € ; brochure2026 p371, couple avec2 enfants RNI54000, brut2112 €. Les frontières des tranches et du recouvrement, frais et autres foyers sont testés séparément. Une FAQ DGFiP présente un titre2026 avec des chiffres2025 et des soustractions prématurément arrondies : ses exemples incohérents ne servent pas d'oracle ; les brochures millésimées et le CGI priment.

Périmètre : résidents métropolitains de moins de65ans, situation stable, salaires ordinaires ou base fiscale finale déjà déterminée, mineurs à charge exclusive, cas général/parent isolé. Mariés/Pacs = imposition commune. Veuf sans enfant et sans particularité seulement. Une confirmation du périmètre est nécessaire ; situations particulières, garde alternée, invalidité, décès/changement familial dans l'année, DOM/non-résidence, réductions/crédits, hauts revenus et contributions complémentaires sont refusés, pas estimés à zéro.

PAS : cumul explicite imputable à la même année, jamais un débit bancaire ou un montant mensuel supposé annuel. Cumul inconnu = pas de solde. Partiel = écart avant prélèvements futurs, pas de solde final ; projection future distincte uniquement sur hypothèse fournie. Les revenus mensuels ou cumuls incomplets exigent l'autorisation explicite d'une annualisation constante. Aucun calcul n'écrit dans Crédits & taxes ou les données bancaires.
