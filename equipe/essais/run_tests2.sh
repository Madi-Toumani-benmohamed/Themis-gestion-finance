#!/usr/bin/env bash
set -e

COOKIE="themis_session=QRrUSn1yOKbPcsvr2MovweWGZ5Pthvnff2-u8g7EHmc"
BASE="http://localhost:8080"

echo "=== Test 1: Summary par mois ==="
for month in 2026-10 2026-09 2026-08 2026-01 2025-12 2025-01; do
  echo -n "  $month: "
  curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$month" | jq -r '.monthly[] | select(.month == "'$month'") | "\(.income) € / \(.expenses) € / \(.status) / is_current=\(.is_current)"'
done

echo ""
echo "=== Test 2: Transactions par mois ==="
for month in 2026-10 2026-09 2026-08 2026-01 2025-12; do
  count=$(curl -s -b "$COOKIE" "$BASE/api/transactions?month=$month" | jq 'length')
  negatives=$(curl -s -b "$COOKIE" "$BASE/api/transactions?month=$month" | jq '[.[] | select(.amount < 0)] | length')
  echo "  $month: $count transactions, $negatives dépenses"
done

echo ""
echo "=== Test 3: Top widget simulation ==="
# Récupérer les IDs de transfert
transfer_ids=$(curl -s -b "$COOKIE" "$BASE/api/categories" | jq -r '[.[] | select(.is_transfer) | .id] | join(",")')
echo "  IDs transferts: $transfer_ids"

for month in 2026-10 2026-09 2026-08 2026-01 2025-12; do
  echo -n "  $month top5: "
  curl -s -b "$COOKIE" "$BASE/api/transactions?month=$month" | jq --arg tids "$transfer_ids" '
    map(select(.amount < 0)) |
    map(select(.domain_id as $d | $tids | split(",") | map tonumber | index($d) | not)) |
    sort_by(.amount) | .[0:5] | map(.display_label // .label) | join(" | ")
  '
  # Total dépenses (hors transferts)
  total=$(curl -s -b "$COOKIE" "$BASE/api/transactions?month=$month" | jq --arg tids "$transfer_ids" '
    map(select(.amount < 0)) |
    map(select(.domain_id as $d | $tids | split(",") | map tonumber | index($d) | not)) |
    map(.amount) | add | . * -1
  ')
  # Comparer avec summary
  summary_expenses=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$month" | jq -r '.monthly[] | select(.month == "'$month'") | .expenses')
  diff=$(echo "$total $summary_expenses" | awk '{printf "%.2f", $1 - $2}')
  echo "    Total dépenses (hors transferts): $total € | Summary: $summary_expenses € | Écart: $diff"
done

echo ""
echo "=== Test 4: Cohérence KPI / Catégories ==="
month="2026-09"
summary=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$month")
monthly_expenses=$(echo "$summary" | jq -r '.monthly[] | select(.month == "'$month'") | .expenses')
by_cat_total=$(echo "$summary" | jq '[.by_category[] | select(.month == "'$month'") | .total] | add')
echo "  $month: dépenses summary=$monthly_expenses € | total catégories=$by_cat_total €"
if (( $(echo "$monthly_expenses $by_cat_total" | awk '{print ($1 == $2)}') )); then
  echo "    ✓ Cohérent"
else
  echo "    ✗ Incohérent"
fi

echo ""
echo "=== Test 5: Badges période ==="
for month in 2026-10 2026-08 2025-10 2025-04; do
  info=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$month" | jq -r '.monthly[] | select(.month == "'$month'") | "is_current=\(.is_current) status=\(.status) days_observed=\(.days_observed)/\(.days_in_month)"')
  echo "  $month: $info"
done

echo ""
echo "=== Test 6: Navigation mois suivant (bloqué) ==="
current_month=$(date +%Y-%m)
echo "  Mois courant: $current_month"
next_month=$(date -d "next month" +%Y-%m 2>/dev/null || date -v+1m +%Y-%m 2>/dev/null)
summary=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$current_month")
months_list=$(echo "$summary" | jq -r '.monthly[].month')
if echo "$months_list" | grep -q "^$next_month$"; then
  echo "  ✗ Le mois suivant $next_month existe encore dans les données (bouton activé)"
else
  echo "  ✓ Le mois suivant $next_month n'existe pas encore (bouton désactivé)"
fi

echo ""
echo "=== Test 7: Rechargement des données après changement de mois ==="
prev_month="2026-09"
curr_month="2026-10"
exp_prev=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$prev_month" | jq -r '.monthly[] | select(.month == "'$prev_month'") | .expenses')
exp_curr=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$curr_month" | jq -r '.monthly[] | select(.month == "'$curr_month'") | .expenses')
echo "  Dépenses $prev_month: $exp_prev €"
echo "  Dépenses $curr_month: $exp_curr €"
if [ "$exp_prev" != "$exp_curr" ]; then
  echo "  ✓ Les données changent bien entre mois"
else
  echo "  ✗ Les données sont identiques (bug potentiel)"
fi

echo ""
echo "=== Test 8: Changement rapide entre mois ==="
for i in {1..5}; do
  month1="2026-08"
  month2="2026-10"
  exp1=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$month1" | jq -r '.monthly[] | select(.month == "'$month1'") | .expenses')
  exp2=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$month2" | jq -r '.monthly[] | select(.month == "'$month2'") | .expenses')
  echo "  Itération $i: $month1=$exp1 € | $month2=$exp2 €"
done

echo ""
echo "=== Test 9: Messages vides ==="
month="2027-01"
summary=$(curl -s -b "$COOKIE" "$BASE/api/summary?months=24&month=$month")
empty=$(echo "$summary" | jq -r '.monthly | length')
echo "  $month: monthly array length = $empty"
if [ "$empty" -eq 0 ]; then
  echo "  ✓ Pas de données, le widget Top devrait afficher vide (condition rows.length === 0)"
else
  echo "  ✗ Données présentes pour un mois futur"
fi

echo ""
echo "✅ Tests terminés."
