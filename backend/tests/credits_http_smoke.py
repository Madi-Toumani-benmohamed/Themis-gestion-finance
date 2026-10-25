"""Smoke HTTP réel, exclusivement sur une instance de test vide et isolée.

THEMIS_CREDITS_SMOKE_URL doit viser le conteneur de validation. Le script refuse
une instance déjà configurée, avant toute mutation. Données synthétiques seules.
"""
import http.cookiejar
import json
import os
import urllib.error
import urllib.request

base = os.environ["THEMIS_CREDITS_SMOKE_URL"].rstrip("/")
if os.environ.get("THEMIS_SYNTHETIC_TEST_INSTANCE") != "1":
    raise SystemExit("Instance synthétique explicite requise")


def client():
    return urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))


def call(opener, method, path, data=None, status=200):
    request = urllib.request.Request(base + path, method=method, data=None if data is None else json.dumps(data).encode(), headers={"Content-Type": "application/json"})
    try:
        response = opener.open(request, timeout=15)
    except urllib.error.HTTPError as error:
        response = error
    assert response.status == status, f"{method} {path}: {response.status}, attendu {status}"
    body = response.read()
    return json.loads(body) if body else None


admin, member, anonymous = client(), client(), client()
assert call(anonymous, "GET", "/api/auth/status")["setup_required"], "Refus : instance déjà configurée"
for path in ("/api/credits", "/api/taxes", "/api/obligations", "/api/credits/1/history"):
    call(anonymous, "GET", path, status=401)
password = "SyntheticTestPassword2026"
call(admin, "POST", "/api/auth/setup", {"username": "CREDITS ADMIN TEST", "password": password}, 201)
call(admin, "POST", "/api/admin/users", {"username": "CREDITS MEMBER TEST", "password": password, "role": "user"}, 201)
call(member, "POST", "/api/auth/login", {"username": "CREDITS MEMBER TEST", "password": password})
credit = call(admin, "POST", "/api/credits", {"name": "CREDIT SYNTHETIQUE", "monthly_amount": "33.33", "months_total": 3, "start_month": "2020-01", "opening_paid": "0.00"}, 201)
credit_path = f'/api/credits/{credit["id"]}'
payment = {"amount": "22.22", "payment_date": "2020-01-01", "idempotency_key": "synthetic-http-payment-0001"}
for method, path, body in (("PATCH", credit_path, {"monthly_amount": "1.00"}), ("DELETE", credit_path, None), ("GET", credit_path + "/history", None), ("POST", credit_path + "/payments", payment)):
    call(member, method, path, body, 404)
assert call(member, "GET", "/api/credits") == []
foreign = call(member, "POST", "/api/credits", {"name": "MEMBRE SYNTHETIQUE", "monthly_amount": "1.00", "months_total": 1, "start_month": "2020-01", "opening_paid": "0.00"}, 201)
call(admin, "PATCH", f'/api/credits/{foreign["id"]}', {"name": "INTERDIT"}, 404)
first = call(admin, "POST", credit_path + "/payments", payment, 201)
assert call(admin, "POST", credit_path + "/payments", payment)["replayed"]
assert len(call(admin, "GET", credit_path + "/history")["payments"]) == 1
report = call(admin, "GET", "/api/obligations")
assert report["amounts_cents"]["credits_remaining"] == 7777
assert report["credits"][0]["amounts_cents"]["paid_total"] == 2222
call(member, "DELETE", credit_path + f'/payments/{first["payment"]["id"]}', {"note": "TEST"}, 404)
tax = call(admin, "POST", "/api/taxes", {"name": "TAXE SYNTHETIQUE", "amount": "100.01", "due_date": "2020-01-01"}, 201)
tax_path = f'/api/taxes/{tax["id"]}'
for method, path, body in (("PATCH", tax_path, {"paid": True}), ("DELETE", tax_path, None), ("GET", tax_path + "/history", None)):
    call(member, method, path, body, 404)
assert tax["status"] == "to_verify"
call(admin, "PATCH", tax_path, {"payment_amount": "33.33", "payment_date": "2020-01-01", "idempotency_key": "synthetic-tax-payment-0001"})
report = call(admin, "GET", "/api/obligations")
assert report["amounts_cents"]["taxes_due"] == 6668
assert report["amounts_cents"]["total_debt"] == 14445
assert report["taxes"][0]["status"] == "partial"
export = call(admin, "GET", "/api/export")
assert len(export["credit_payments"]) == 1 and export["tax_history"]
assert call(member, "GET", "/api/export")["credit_payments"] == []
call(admin, "DELETE", credit_path)
assert call(admin, "GET", "/api/credits") == []
assert len(call(admin, "GET", credit_path + "/history")["payments"]) == 1
print("Crédits HTTP réel : authentification, isolation, paiements, historique, agrégats et export OK")
