// Synthetic configuration and fully intercepted APIs. No external provider is contacted.
const knowledgeFixture = require('./fixtures/knowledge.cjs');
const { chromium } = require("playwright");
const { AxeBuilder } = require("@axe-core/playwright");
const assert = require("node:assert/strict");
const fs = require("node:fs");
let failLoad = true,
  failSave = false,
  failTest = false,
  holdTest = false,
  memoryFail = false;
const calls = [],
  saved = [],
  tests = [];
let lastTest = null;
let cfg = {
  provider: "ollama",
  enabled: true,
  active: "ollama (unit)",
  source: "database",
  revision: "r1",
  ollama_url: "https://models.example",
  ollama_model: "unit",
  openai_model: "unit-openai",
  anthropic_model: "unit-claude",
  mistral_model: "unit-mistral",
  freellm_model: "unit-free",
  freellm_url: "https://gateway.example/v1",
  auto_categorize: true,
  context: "Note déclarative TEST",
  openai_key_set: true,
  anthropic_key_set: true,
  mistral_key_set: true,
  freellm_key_set: true,
  key_sources: {
    openai: "stored",
    anthropic: "environment",
    mistral: "stored",
    freellm: "stored",
  },
  diagnostic: { status: "not_tested", checked_at: null, latency_ms: null },
  memory: {
    enabled: true,
    configured: true,
    required: false,
    diagnostic: { status: "not_tested", checked_at: null, latency_ms: null },
  },
};
const stamp = "2026-10-25T12:00:00Z";
async function main() {
  const browser = await chromium.launch({
    headless: true,
    channel: "chromium",
  });
  try {
    const context = await browser.newContext({
      viewport: { width: 1920, height: 1080 },
    });
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.addInitScript(() =>
      localStorage.setItem("themis-theme", "dark"),
    );
    await page.route("**/api/**", async (route) => {
      const request = route.request(),
        url = new URL(request.url()),
        path = url.pathname,
        method = request.method();
      calls.push({ path, method });
      let data = {},
        status = 200;
      if (path === "/api/auth/status") data = { setup_required: false };
      else if (path === "/api/auth/me")
        data = { id: 1, username: "IA TEST", role: "admin" };
      else if (path === "/api/health") data = { ai_enabled: true };
      else if (path === "/api/summary")
        data = { monthly: [], by_category: [], uncategorized: 0 };
      else if (path === "/api/hero" || path === "/api/remaining")
        data = { available: false };
      else if (path === "/api/anomalies") data = { anomalies: [] };
      else if (path === "/api/journal") data = { events: [] };
      else if (path === "/api/admin/users" || path === "/api/accounts")
        data = [];
      else if (path === "/api/knowledge/overview") data = knowledgeFixture.overview();
      else if (path === "/api/admin/config") {
        if (method === "GET") {
          if (failLoad) {
            status = 500;
            data = { error: "ERREUR LECTURE TEST" };
          } else data = cfg;
        } else if (failSave) {
          status = 500;
          data = { error: "ERREUR SAUVEGARDE TEST" };
        } else if (request.postDataJSON().expected_revision !== cfg.revision) {
          status = 409;
          data = { error: "CONFIGURATION MODIFIEE AILLEURS TEST" };
        } else {
          const body = request.postDataJSON();
          saved.push(body);
          const next = { ...cfg, key_sources: { ...cfg.key_sources } };
          for (const [key, value] of Object.entries(body)) {
            if (key.endsWith("_key")) {
              next[key + "_set"] = true;
              next.key_sources[key.replace("_key", "")] = "stored";
            } else if (!["clear_keys", "expected_revision"].includes(key))
              next[key] = value;
          }
          for (const key of body.clear_keys || []) {
            next[key + "_set"] = false;
            next.key_sources[key.replace("_key", "")] = "none";
          }
          next.enabled = next.provider !== "none";
          next.revision = "r" + (saved.length + 1);
          next.diagnostic =
            lastTest && lastTest.provider === next.provider
              ? { ...lastTest, scope: "saved" }
              : { status: next.enabled ? "not_tested" : "not_configured" };
          cfg = next;
          data = cfg;
        }
      } else if (path === "/api/admin/ai/test") {
        const body = request.postDataJSON();
        tests.push(body);
        if (holdTest) await new Promise((r) => setTimeout(r, 450));
        const provider = body.provider || cfg.provider;
        lastTest = {
          ok: !failTest,
          provider,
          model: body[provider + "_model"],
          status: failTest ? "unavailable" : "available",
          detail: failTest
            ? "FOURNISSEUR INDISPONIBLE TEST"
            : "CONNEXION SYNTHETIQUE TEST",
          latency_ms: 12,
          checked_at: stamp,
          scope:
            body.provider === cfg.provider &&
            body[provider + "_model"] === cfg[provider + "_model"]
              ? "saved"
              : "draft",
        };
        data = lastTest;
      } else if (path === "/api/admin/ollama/models")
        data = { models: ["unit", "other"] };
      else if (path === "/api/admin/ai/memory/test")
        data = {
          enabled: true,
          configured: true,
          required: false,
          diagnostic: {
            ok: !memoryFail,
            status: memoryFail ? "unavailable" : "available",
            detail: memoryFail
              ? "MEMOIRE INDISPONIBLE TEST"
              : "MEMOIRE DISPONIBLE TEST",
            checked_at: stamp,
            latency_ms: 3,
          },
        };
      await route.fulfill({
        status,
        contentType: "application/json",
        body: JSON.stringify(data),
      });
    });
    const root = page.locator(".ai-settings"),
      panel = root.getByRole("tabpanel");
    const tab = (name) => root.getByRole("tab", { name, exact: true });
    const settle = () =>
      page.evaluate(async () => {
        await document.fonts.ready;
        await Promise.all(
          document
            .getAnimations()
            .filter((a) =>
              Number.isFinite(a.effect.getComputedTiming().endTime),
            )
            .map((a) => a.finished.catch(() => {})),
        );
      });
    const axe = async () => {
      await settle();
      const report = await new AxeBuilder({ page })
        .include(".ai-settings")
        .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
        .analyze();
      assert.deepEqual(
        report.violations.map((v) => ({
          id: v.id,
          nodes: v.nodes.map((n) => ({
            target: n.target,
            failure: n.failureSummary,
          })),
        })),
        [],
      );
    };
    await page.goto(process.env.THEMIS_UI_URL || "http://127.0.0.1:5176");
    await page.getByRole("button", { name: "Admin", exact: true }).click();
    await root.getByRole("alert").waitFor();
    failLoad = false;
    await root
      .getByRole("button", { name: "Recharger la configuration IA" })
      .click();
    await root.getByLabel("Fournisseur", { exact: true }).waitFor();
    assert.equal(
      calls.filter(
        (c) =>
          c.path === "/api/admin/ai/test" ||
          c.path === "/api/admin/ai/memory/test" ||
          c.path === "/api/admin/ollama/models",
      ).length,
      0,
    );
    assert.match(await panel.innerText(), /Non testé/);
    assert.equal(
      await root
        .getByRole("button", { name: "Enregistrer et appliquer" })
        .isDisabled(),
      true,
    );
    await root
      .getByRole("button", { name: "Tester la connexion", exact: true })
      .click();
    await root
      .getByText("CONNEXION SYNTHETIQUE TEST", { exact: true })
      .waitFor();
    assert.ok(!JSON.stringify(tests[0]).includes("Note déclarative"));
    await root.getByLabel("Modèle", { exact: true }).fill("custom-local");
    await root.getByRole("button", { name: "Rechercher les modèles" }).click();
    await root
      .getByRole("status")
      .filter({ hasText: "modèle(s) disponible(s)" })
      .waitFor();
    assert.equal(
      await root.getByLabel("Modèle", { exact: true }).inputValue(),
      "custom-local",
    );
    assert.match(await root.innerText(), /Modifications non enregistrées/);
    assert.match(await panel.innerText(), /Non testé/);
    for (const provider of [
      "openai",
      "anthropic",
      "mistral",
      "freellm",
      "ollama",
      "none",
    ]) {
      await root
        .getByLabel("Fournisseur", { exact: true })
        .selectOption(provider);
      assert.equal(
        await panel.getByLabel("URL du serveur", { exact: true }).count(),
        ["ollama", "freellm"].includes(provider) ? 1 : 0,
      );
      assert.equal(
        await panel.getByLabel("Nouvelle clé API", { exact: true }).count(),
        ["openai", "anthropic", "mistral", "freellm"].includes(provider)
          ? 1
          : 0,
      );
    }
    await root
      .getByLabel("Fournisseur", { exact: true })
      .selectOption("openai");
    await root
      .getByLabel("Nouvelle clé API", { exact: true })
      .fill("synthetic-new-key");
    failTest = true;
    await root
      .getByRole("button", { name: "Tester la connexion", exact: true })
      .click();
    await root
      .getByText("FOURNISSEUR INDISPONIBLE TEST", { exact: true })
      .waitFor();
    assert.equal(cfg.provider, "ollama");
    failTest = false;
    await root
      .getByRole("button", { name: "Tester la connexion", exact: true })
      .click();
    await root
      .getByText("CONNEXION SYNTHETIQUE TEST", { exact: true })
      .waitFor();
    failSave = true;
    await root
      .getByRole("button", { name: "Enregistrer et appliquer" })
      .click();
    await root.getByRole("alert").waitFor();
    assert.equal(
      await root.getByLabel("Nouvelle clé API", { exact: true }).inputValue(),
      "synthetic-new-key",
    );
    assert.equal(cfg.provider, "ollama");
    failSave = false;
    await root
      .getByRole("button", { name: "Enregistrer et appliquer" })
      .click();
    await root
      .getByText("Configuration enregistrée", { exact: true })
      .waitFor();
    assert.equal(cfg.provider, "openai");
    assert.equal(
      await root.getByLabel("Nouvelle clé API", { exact: true }).inputValue(),
      "",
    );
    assert.equal(saved[0].expected_revision, "r1");
    await tab("Comportement").click();
    await root
      .getByLabel("Contexte personnel", { exact: true })
      .fill("Brouillon conservé TEST");
    await page
      .getByRole("button", {
        name: "Réappliquer les règles aux transactions non classées",
      })
      .click();
    assert.equal(
      await root.getByLabel("Contexte personnel", { exact: true }).inputValue(),
      "Brouillon conservé TEST",
    );
    await root
      .getByRole("button", { name: "Annuler les modifications" })
      .click();
    assert.equal(
      await root.getByLabel("Contexte personnel", { exact: true }).inputValue(),
      "Note déclarative TEST",
    );
    await tab("Fournisseur et connexion").click();
    holdTest = true;
    await root
      .getByRole("button", { name: "Tester la connexion", exact: true })
      .click();
    await root.getByLabel("Modèle", { exact: true }).fill("next-model");
    holdTest = false;
    await page.waitForTimeout(550);
    assert.match(await panel.innerText(), /Non testé/);
    await root
      .getByRole("button", { name: "Annuler les modifications" })
      .click();
    await root
      .getByLabel("Nouvelle clé API", { exact: true })
      .fill("key-to-discard");
    await root
      .getByRole("checkbox", {
        name: "Retirer cette clé enregistrée à l’enregistrement",
      })
      .check();
    assert.equal(
      await root.getByLabel("Nouvelle clé API", { exact: true }).isDisabled(),
      true,
    );
    assert.equal(
      await root.getByLabel("Nouvelle clé API", { exact: true }).inputValue(),
      "",
    );
    await root
      .getByRole("button", { name: "Annuler les modifications" })
      .click();
    await tab("Diagnostic IA").click();
    assert.match(await panel.innerText(), /Configuration enregistrée/);
    assert.match(await panel.innerText(), /12 ms/);
    await tab("Mémoire").click();
    assert.match(await panel.innerText(), /Non testé/);
    await root.getByRole("button", { name: "Vérifier la mémoire" }).click();
    await panel.getByText("MEMOIRE DISPONIBLE TEST", { exact: true }).waitFor();
    memoryFail = true;
    await root.getByRole("button", { name: "Vérifier la mémoire" }).click();
    await panel
      .getByText("MEMOIRE INDISPONIBLE TEST", { exact: true })
      .waitFor();
    assert.equal(cfg.enabled, true);
    const artifacts = process.env.THEMIS_AI_ARTIFACTS || "/tmp/themis-ai-ui";
    fs.mkdirSync(artifacts, { recursive: true });
    for (const width of [1920, 960, 768, 390, 320]) {
      await page.setViewportSize({ width, height: 1080 });
      for (const [, name] of [
        ["connection", "Fournisseur et connexion"],
        ["behavior", "Comportement"],
        ["diagnostic", "Diagnostic IA"],
        ["memory", "Mémoire"],
      ]) {
        await tab(name).click();
        await settle();
        assert.ok(
          await root.evaluate((el) => el.scrollWidth <= el.clientWidth + 1),
          `AI overflow ${width}/${name}`,
        );
        await axe();
      }
    }
    await tab("Fournisseur et connexion").click();
    await tab("Fournisseur et connexion").focus();
    await page.keyboard.press("ArrowRight");
    assert.equal(
      await tab("Comportement").getAttribute("aria-selected"),
      "true",
    );
    await tab("Fournisseur et connexion").click();
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({
      path: `${artifacts}/ai-mobile.png`,
      fullPage: true,
    });
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.locator(".sidebar-footer .theme-toggle").click();
    await axe();
    await page.screenshot({
      path: `${artifacts}/ai-light.png`,
      fullPage: true,
    });
    // A conflicting save preserves the draft until an explicit reload succeeds.
    await root.getByLabel("Modèle", { exact: true }).fill("unsaved-conflict");
    cfg.revision = "external-r3";
    cfg.memory = {
      enabled: false,
      configured: true,
      diagnostic: { status: "disabled" },
    };
    await root
      .getByRole("button", { name: "Enregistrer et appliquer" })
      .click();
    await root
      .getByRole("alert")
      .filter({ hasText: "CONFIGURATION MODIFIEE AILLEURS TEST" })
      .waitFor();
    assert.equal(
      await root.getByLabel("Modèle", { exact: true }).inputValue(),
      "unsaved-conflict",
    );
    await root
      .getByRole("button", { name: "Recharger les réglages enregistrés" })
      .click();
    await root
      .getByText("Configuration enregistrée", { exact: true })
      .waitFor();
    assert.equal(
      await root.getByLabel("Modèle", { exact: true }).inputValue(),
      cfg.openai_model,
    );
    await tab("Mémoire").click();
    assert.match(await panel.innerText(), /Désactivée côté serveur/);
    assert.equal(
      await root
        .getByRole("button", { name: "Vérifier la mémoire" })
        .isDisabled(),
      true,
    );
    await tab("Fournisseur et connexion").click();
    await root.getByLabel("Fournisseur", { exact: true }).selectOption("none");
    await root
      .getByRole("button", { name: "Enregistrer et appliquer" })
      .click();
    await root
      .getByText("Configuration enregistrée", { exact: true })
      .waitFor();
    assert.equal(cfg.enabled, false);
    assert.equal(
      await page
        .locator(".sidebar-nav")
        .getByRole("tab", { name: "Assistant", exact: true })
        .count(),
      0,
    );
    assert.equal(cfg.context, "Note déclarative TEST");
    assert.deepEqual(errors, []);
    console.log(
      "Configuration IA navigateur PASS : fournisseurs, brouillons, clés, erreurs, tests, sauvegarde, mémoire, réponses obsolètes, clavier, responsive et Axe.",
    );
  } finally {
    await browser.close();
  }
}
main().catch((e) => {
  console.error(e);
  process.exit(1);
});
