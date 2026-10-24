#!/usr/bin/env python3
"""Tests de navigation temporelle et Top dépenses pour Thémis."""

import json
import sys
import subprocess
import time
import urllib.request
import urllib.error
import urllib.parse
import http.cookiejar
import os
import ssl
import http.client

BASE_URL = "http://localhost:8080"
COOKIE_JAR_FILE = "/tmp/test_themis_cookies.txt"
__ctx = ssl.create_default_context()

def _request(url, cookie_jar, data=None):
    base = urllib.parse.urlsplit(BASE_URL)
    target = urllib.parse.urlsplit(url)
    if (target.scheme not in ("http", "https") or target.username or target.password
            or (target.scheme, target.hostname, target.port) != (base.scheme, base.hostname, base.port)):
        raise ValueError("Only the configured HTTP(S) test origin is allowed")
    request = urllib.request.Request(url, data=data)
    if data is not None:
        request.add_header("Content-Type", "application/json")
    cookie_jar.add_cookie_header(request)
    if target.scheme == "https":
        # Explicit verified SSLContext (CERT_REQUIRED + hostname checks), covered by isolated tests.
        # nosemgrep: python.lang.security.audit.httpsconnection-detected.httpsconnection-detected
        connection = http.client.HTTPSConnection(target.hostname, target.port, timeout=20, context=__ctx)
    else:
        connection = http.client.HTTPConnection(target.hostname, target.port, timeout=20)
    path = urllib.parse.urlunsplit(("", "", target.path or "/", target.query, ""))
    try:
        connection.request("POST" if data is not None else "GET", path, body=data, headers=dict(request.header_items()))
        response = connection.getresponse()
        cookie_jar.extract_cookies(response, request)
        if response.status < 200 or response.status >= 300:
            raise urllib.error.HTTPError(url, response.status, response.reason, response.headers, None)
        return json.loads(response.read().decode())
    finally:
        connection.close()


def login(username, password):
    cj = http.cookiejar.MozillaCookieJar(COOKIE_JAR_FILE)
    _request(f"{BASE_URL}/api/auth/login", cj,
             json.dumps({"username": username, "password": password}).encode())
    cj.save(ignore_discard=True, ignore_expires=True)
    return cj


def fetch(url, cookie_jar):
    return _request(url, cookie_jar)


def test_month_selector_logic():
    print("\n=== Test 1: Logique MonthSelector ===")
    now = time.localtime()
    current_month = f"{now.tm_year}-{now.tm_mon:02d}"
    print(f"Mois courant détecté: {current_month}")
    assert current_month == "2026-10", f"Mois courant attendu 2026-10, got {current_month}"
    
def test_summary_for_months():
    print("\n=== Test 2: Summary par mois ===")
    months = [
        ("2026-10", "2026-10"),
        ("2026-09", "2026-09"),
        ("2026-08", "2026-08"),
        ("2026-01", "2026-01"),
        ("2025-12", "2025-12"),
        ("2025-01", "2025-01"),
    ]
    for month, label in months:
        url = f"{BASE_URL}/api/summary?months=24&month={month}"
        data = fetch(url, cookie_jar)
        monthly = data.get("monthly", [])
        found = next((m for m in monthly if m["month"] == month), None)
        assert found is not None, f"Pas de données pour {month}"
        print(f"  {month}: income={found['income']}, expenses={found['expenses']}, status={found['status']}")
        if month == "2026-10":
            assert found["is_current"] == True, f"Mois {month} devrait être is_current"
        else:
            assert found.get("is_current") == False or found.get("status") != "current"

def test_transactions_for_month():
    print("\n=== Test 3: Transactions par mois ===")
    months = ["2026-10", "2026-09", "2026-08", "2026-01", "2025-12"]
    for month in months:
        url = f"{BASE_URL}/api/transactions?month={month}"
        txs = fetch(url, cookie_jar)
        from datetime import datetime
        year, mon = month.split("-")
        for t in txs:
            dt = datetime.fromisoformat(t["date"])
            assert dt.year == int(year) and dt.month == int(mon), f"Transaction {t['id']} hors mois {month}"
        print(f"  {month}: {len(txs)} transactions, toutes dans le mois.")
        negatives = [t for t in txs if t["amount"] < 0]
        print(f"    dont {len(negatives)} dépenses.")

def test_top_widget_data():
    print("\n=== Test 4: Top dépenses (simulation) ===")
    cats = fetch(f"{BASE_URL}/api/categories", cookie_jar)
    transfer_ids = set(c["id"] for c in cats if c.get("is_transfer"))
    print(f"  Transferts (domain_ids): {transfer_ids}")
    
    months = ["2026-10", "2026-09", "2026-08", "2026-01", "2025-12"]
    for month in months:
        txs = fetch(f"{BASE_URL}/api/transactions?month={month}", cookie_jar)
        expenses = [t for t in txs if t["amount"] < 0 and t.get("domain_id") not in transfer_ids]
        expenses.sort(key=lambda x: x["amount"])
        top5 = expenses[:5]
        total_spent = sum(-t["amount"] for t in expenses)
        print(f"  {month}: top 5 dépenses = {[t['display_label'][:30] for t in top5]}")
        print(f"    Total dépenses (hors transferts): {total_spent:.2f} €")
        summary = fetch(f"{BASE_URL}/api/summary?months=24&month={month}", cookie_jar)
        monthly_item = next(m for m in summary["monthly"] if m["month"] == month)
        assert abs(monthly_item["expenses"] - total_spent) < 0.01, f"Écart dépenses {month}: summary={monthly_item['expenses']}, top_calc={total_spent}"
        print(f"    Cohérent avec summary: {monthly_item['expenses']} €")

def test_kpi_consistency():
    print("\n=== Test 5: Cohérence KPI / Top / Catégories ===")
    month = "2026-09"
    summary = fetch(f"{BASE_URL}/api/summary?months=24&month={month}", cookie_jar)
    monthly = next(m for m in summary["monthly"] if m["month"] == month)
    by_cat = summary["by_category"]
    month_cats = [c for c in by_cat if c["month"] == month]
    total_from_cats = sum(c["total"] for c in month_cats)
    print(f"  {month}: dépenses par catégorie total = {total_from_cats:.2f}, dépenses summary = {monthly['expenses']:.2f}")
    assert abs(total_from_cats - monthly["expenses"]) < 0.01, f"Incohérence catégories vs summary pour {month}"
    print(f"    ✓ Les catégories couvrent exactement les dépenses du mois.")

def test_period_badges():
    print("\n=== Test 6: Badges période ===")
    months = ["2026-10", "2026-08", "2025-10", "2025-04"]
    for month in months:
        summary = fetch(f"{BASE_URL}/api/summary?months=24&month={month}", cookie_jar)
        monthly = next(m for m in summary["monthly"] if m["month"] == month)
        is_current = monthly.get("is_current") or monthly.get("status") == "current"
        incomplete = monthly.get("status") == "incomplete"
        print(f"  {month}: is_current={is_current}, incomplete={incomplete}, days_observed={monthly['days_observed']}/{monthly['days_in_month']}")
        if month == "2026-10":
            assert is_current, f"2026-10 devrait être mois courant"
        elif month == "2025-10":
            assert monthly["status"] == "incomplete", f"2025-10 devrait être incomplet"
        elif month == "2025-04":
            assert monthly["status"] == "incomplete", f"2025-04 devrait être incomplet"
        else:
            assert not is_current and not incomplete, f"{month} devrait être complet et non courant"

def test_next_month_disabled():
    print("\n=== Test 7: Blocage navigation mois suivant ===")
    current_month = "2026-10"
    summary = fetch(f"{BASE_URL}/api/summary?months=24&month={current_month}", cookie_jar)
    months_list = [m["month"] for m in summary["monthly"]]
    next_month = "2026-11"
    assert next_month not in months_list, f"Le mois {next_month} ne devrait pas encore exister dans les données"
    print(f"  ✓ Le mois suivant ({next_month}) n'a pas encore de données, le bouton sera désactivé.")

def main():
    global cookie_jar
    cookie_jar = login("Ben", "test12345")
    
    test_month_selector_logic()
    test_summary_for_months()
    test_transactions_for_month()
    test_top_widget_data()
    test_kpi_consistency()
    test_period_badges()
    test_next_month_disabled()
    
    print("\n✅ Tous les tests backend passent.")

if __name__ == "__main__":
    main()
