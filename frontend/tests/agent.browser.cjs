// Real application UI with explicitly synthetic APIs. PostgreSQL ownership,
// money, action locking and replay contracts are tested by agent_workspace_test.
const { chromium, expect } = require('@playwright/test')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const base = process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176'
async function settle(page, theme) {
  await page.evaluate(async (t) => {
    if (t) document.documentElement.dataset.theme = t
    await document.fonts.ready
    await new Promise(requestAnimationFrame)
    await Promise.all(
      document
        .getAnimations()
        .filter((a) => a.effect?.getComputedTiming().iterations !== Infinity)
        .map((a) => a.finished.catch(() => {})),
    )
    await new Promise(requestAnimationFrame)
  }, theme)
}
async function main() {
  const browser = await chromium.launch({ headless: true, channel: 'chromium' })
  try {
    const context = await browser.newContext({
        reducedMotion: 'reduce',
        viewport: { width: 1280, height: 900 },
      }),
      page = await context.newPage(),
      errors = [],
      calls = [],
      chatBodies = []
    page.on('pageerror', (e) => errors.push(e.message))
    let missions = [],
      memories = [],
      conversations = [],
      stored = {},
      actions = [],
      nextMission = 1,
      nextConversation = 7,
      nextAction = 1,
      stale = false,
      applied = 0,
      briefingState = 'available'
    const evidence = [
      {
        type: 'transaction',
        id: 101,
        date: '2026-11-05',
        label: 'ACHAT SYNTHETIQUE TEST',
        amount: '-123.45',
      },
    ]
    evidence.push({
      type: 'tool',
      tool: 'search_transactions',
      data: {
        transactions: [evidence[0]],
        query_from: '2026-11-01',
        query_to: '2026-11-17',
        filters: {
          direction: 'expense',
          merchant: 'TEST',
          category: 'Courses TEST',
          tag: 'scope test',
          min_amount: '60.00',
          max_amount: '200.00',
        },
        has_more: true,
        returned_count: 1,
        coverage: 'unknown',
      },
    })
    const answer = {
      answer:
        'Du 2026-11-01 au 2026-11-17, ta plus grosse dépense enregistrée est ACHAT SYNTHETIQUE TEST : 123,45 €.',
      status: 'ok',
      evidence,
      steps: [
        {
          name: 'search_transactions',
          status: 'success',
          from: '2026-11-01',
          to: '2026-11-17',
        },
      ],
      context: {
        timezone: 'Europe/Paris',
        today: '2026-11-17',
        period: { from: '2026-11-01', to: '2026-11-17' },
      },
    }
    await page.route('**/api/**', async (route) => {
      const req = route.request(),
        url = new URL(req.url()),
        path = url.pathname,
        method = req.method()
      calls.push({ path, method })
      let body = {},
        status = 200
      if (path === '/api/auth/status') body = { setup_required: false }
      else if (path === '/api/auth/me')
        body = { id: 1, username: 'AGENT SYNTHETIQUE', role: 'admin' }
      else if (path === '/api/health') body = { ai_enabled: true }
      else if (path === '/api/summary')
        body = { monthly: [], by_category: [], uncategorized: 0 }
      else if (path === '/api/domains')
        body = [{ id: 11, name: 'Courses TEST', color: '#5755cc' }]
      else if (['/api/accounts', '/api/tags'].includes(path)) body = []
      else if (path === '/api/agent/missions') {
        if (method === 'POST') {
          const input = req.postDataJSON(),
            conv = input.conversation_id || nextConversation++
          const m = {
            id: nextMission++,
            conversation_id: conv,
            title: input.objective,
            objective: input.objective,
            status: 'active',
            last_review_at: null,
            new_statements: 0,
          }
          missions.unshift(m)
          if (!conversations.some((c) => c.id === conv)) {
            conversations.unshift({ id: conv, title: input.objective })
            stored[conv] = []
          }
          body = { id: m.id, conversation_id: conv }
          status = 201
        } else body = { missions }
      } else if (/^\/api\/agent\/missions\/\d+$/.test(path)) {
        const id = Number(path.split('/').pop())
        missions = missions.map((m) =>
          m.id === id ? { ...m, status: req.postDataJSON().status } : m,
        )
        status = 204
      } else if (path === '/api/agent/conversations') body = { conversations }
      else if (/^\/api\/agent\/conversations\/\d+$/.test(path)) {
        const id = Number(path.split('/').pop())
        body = {
          conversation_id: id,
          messages: stored[id] || [],
          context: answer.context,
        }
      } else if (path === '/api/agent/memories') body = { memories }
      else if (/^\/api\/agent\/memories\/\d+$/.test(path)) {
        memories = memories.filter(
          (m) => m.id !== Number(path.split('/').pop()),
        )
        status = 204
      } else if (path === '/api/agent/briefing') {
        assert.ok(
          url.searchParams.get('timezone'),
          'Briefing uses browser timezone',
        )
        body = {
          discoveries: [],
          last_transaction_date: '2026-11-05',
          last_import_at: '2026-11-06T10:00:00Z',
          period_summary: {
            period: { from: '2026-11-01', to: '2026-11-17' },
            timezone: url.searchParams.get('timezone') || 'Europe/Paris',
            available: briefingState === 'available',
            transaction_count: 125,
            expense_count: 120,
            unclassified_expense_count: 101,
            unclassified_transaction_ids: Array.from(
              { length: 100 },
              (_, i) => i + 101,
            ),
            has_more: true,
            coverage: 'unknown',
          },
        }
        if (briefingState === 'failed') {
          status = 503
          body = { error: 'SYNTHETIC unavailable' }
        }
      } else if (path === '/api/chat') {
        const input = req.postDataJSON()
        chatBodies.push(input)
        assert.equal(input.messages.length, 1)
        assert.deepEqual(Object.keys(input.messages[0]).sort(), [
          'content',
          'role',
        ])
        assert.equal(input.messages[0].role, 'user')
        assert.ok(input.context.timezone)
        let conv = input.conversation_id
        if (!conv) {
          conv = nextConversation++
          conversations.unshift({ id: conv, title: input.messages[0].content })
          stored[conv] = []
        }
        stored[conv].push(input.messages[0], {
          role: 'assistant',
          content: answer.answer,
          ...answer,
        })
        body = { ...answer, conversation_id: conv }
      } else if (path === '/api/agent/actions') {
        if (method === 'POST') {
          const { action, conversation_id } = req.postDataJSON()
          assert.ok(conversation_id)
          body = {
            id: nextAction++,
            kind: action.kind,
            status: 'prepared',
            input: action,
            preview:
              action.kind === 'memory'
                ? {
                    content: action.content,
                    effect: 'Préférence déclarée, consultable et supprimable.',
                  }
                : {
                    count: 1,
                    tag: action.tag,
                    effect: 'Seules les opérations listées seront modifiées.',
                    transactions: [
                      {
                        id: 101,
                        date: '2026-11-05',
                        label: 'ACHAT SYNTHETIQUE TEST',
                        amount: '-123.45',
                      },
                    ],
                  },
          }
          actions.unshift(body)
          status = 201
        } else body = { actions }
      } else if (/^\/api\/agent\/actions\/\d+\/decision$/.test(path)) {
        const id = Number(path.split('/')[4]),
          decision = req.postDataJSON().decision
        if (stale && decision === 'apply') {
          actions = actions.map((a) =>
            a.id === id ? { ...a, status: 'stale' } : a,
          )
          status = 409
          body = {
            error: 'Les opérations ont changé ; préparer un nouvel aperçu.',
          }
        } else {
          actions = actions.map((a) => {
            if (a.id !== id) return a
            if (decision === 'apply') {
              applied++
              if (a.kind === 'memory')
                memories.push({ id: a.id, content: a.input.content })
            }
            return {
              ...a,
              status: decision === 'apply' ? 'applied' : 'dismissed',
            }
          })
          body = { id, status: decision === 'apply' ? 'applied' : 'dismissed' }
        }
      }
      if (status === 204) await route.fulfill({ status, body: '' })
      else await route.fulfill({ status, json: structuredClone(body) })
    })
    await page.goto(base)
    await page.getByRole('tab', { name: 'Assistant', exact: true }).click()
    const consoleUI = page.locator('main .agent-console')
    await expect(
      consoleUI.getByText('Dernière opération connue :', { exact: false }),
    ).toBeVisible()
    assert.equal(
      calls.filter((c) => c.path === '/api/chat').length,
      0,
      'Opening an agent never calls a model or writes facts',
    )
    const briefingUI = consoleUI.getByRole('region', {
      name: 'Point sur les données disponibles',
    })
    await expect(briefingUI).toContainText('101 dépense(s) sans catégorie')
    await expect(briefingUI).toContainText('la liste est partielle')
    await briefingUI
      .getByRole('button', { name: 'Préparer une question sur cette période' })
      .click()
    await expect(
      consoleUI.getByRole('textbox', { name: 'Votre question' }),
    ).toHaveValue('Combien ai-je dépensé du 2026-11-01 au 2026-11-17 ?')
    assert.equal(
      chatBodies.length,
      0,
      'Preparing a question never invokes the model',
    )
    await consoleUI
      .getByText('Missions et préférences', { exact: true })
      .click()
    await consoleUI
      .getByRole('textbox', { name: 'Nouvelle mission : quel objectif ?' })
      .fill('Réduire mes dépenses fixes TEST')
    await consoleUI
      .getByRole('button', { name: 'Créer la mission', exact: true })
      .click()
    await expect(consoleUI.getByLabel('Mission suivie')).toHaveValue('1')
    await expect(
      consoleUI.getByLabel('Conversation', { exact: true }),
    ).toHaveValue('7')
    const question = consoleUI.getByRole('textbox', { name: 'Votre question' })
    await question.fill('Donne-moi ma top dépense du mois')
    await question.press('Enter')
    await expect(consoleUI.locator('.assistant-msg-assistant')).toContainText(
      '123,45 €',
    )
    assert.equal(chatBodies[0].mission_id, 1)
    await consoleUI
      .getByText('Voir les données sources', { exact: true })
      .click()
    await expect(consoleUI.locator('.agent-read-filters')).toContainText(
      'catégorie : Courses TEST',
    )
    await expect(consoleUI.locator('.agent-read-filters')).toContainText(
      'à partir de 60,00 €',
    )
    await expect(consoleUI.locator('.assistant-evidence')).toContainText(
      'Liste partielle',
    )
    await consoleUI
      .getByText('Travail réalisé · 1 étape(s)', { exact: true })
      .click()
    await expect(consoleUI.locator('.agent-steps')).toContainText(
      'Recherche des opérations',
    )
    await consoleUI
      .getByText('Préparer une action ou mémoriser une décision', {
        exact: true,
      })
      .click()
    await consoleUI.getByLabel('Action à préparer').selectOption('tag')
    await consoleUI
      .getByRole('checkbox', {
        name: '2026-11-05 · ACHAT SYNTHETIQUE TEST · -123.45 €',
      })
      .check()
    await consoleUI.getByLabel('Tag à ajouter').fill('budget test')
    await consoleUI
      .getByRole('button', { name: 'Préparer un aperçu', exact: true })
      .click()
    await expect(consoleUI.locator('.agent-actions')).toContainText(
      'budget test',
    )
    assert.equal(applied, 0, 'A preview must not apply any mutation')
    await consoleUI
      .getByRole('button', { name: 'Valider et appliquer', exact: true })
      .click()
    await expect(consoleUI.locator('.agent-actions')).toContainText('Appliquée')
    await expect(consoleUI.locator('.assistant-msg-assistant')).toContainText(
      '123,45 €',
    )
    assert.equal(applied, 1)
    assert.equal(chatBodies.length, 1)
    stale = true
    await consoleUI
      .getByRole('button', { name: 'Préparer un aperçu', exact: true })
      .click()
    await consoleUI
      .getByRole('button', { name: 'Valider et appliquer', exact: true })
      .click()
    await expect(consoleUI.getByRole('alert')).toContainText(
      'Les opérations ont changé',
    )
    await expect(consoleUI.locator('.agent-actions')).toContainText(
      'À préparer de nouveau',
    )
    assert.equal(applied, 1)
    stale = false
    await consoleUI.getByLabel('Action à préparer').selectOption('memory')
    await consoleUI
      .getByLabel('Préférence ou décision à retenir')
      .fill('Je préfère examiner mes abonnements TEST.')
    await consoleUI
      .getByRole('button', { name: 'Préparer un aperçu', exact: true })
      .click()
    await consoleUI
      .getByRole('button', { name: 'Valider et appliquer', exact: true })
      .click()
    await expect(
      consoleUI
        .getByText('Je préfère examiner mes abonnements TEST.', { exact: true })
        .first(),
    ).toBeVisible()
    await consoleUI
      .getByRole('button', { name: 'Oublier', exact: true })
      .click()
    await expect(
      consoleUI.getByRole('button', { name: 'Oublier', exact: true }),
    ).toHaveCount(0)
    await consoleUI
      .getByRole('button', { name: 'Mettre en pause', exact: true })
      .click()
    await expect(
      consoleUI.getByRole('button', { name: 'Réactiver', exact: true }),
    ).toBeVisible()
    await consoleUI
      .getByRole('button', { name: 'Réactiver', exact: true })
      .click()
    await page.reload()
    await page.getByRole('tab', { name: 'Assistant', exact: true }).click()
    await page
      .locator('main .agent-console')
      .getByLabel('Conversation', { exact: true })
      .selectOption('7')
    await expect(page.locator('main .assistant-msg-assistant')).toContainText(
      '123,45 €',
    )
    await consoleUI
      .getByText('Missions et préférences', { exact: true })
      .click()
    await consoleUI
      .getByText('Préparer une action ou mémoriser une décision', {
        exact: true,
      })
      .click()
    const captures =
      process.env.THEMIS_AGENT_ARTIFACTS || '/tmp/themis-agent-ui'
    fs.mkdirSync(captures, { recursive: true })
    for (const width of [360, 768, 1280])
      for (const theme of ['light', 'dark']) {
        await page.setViewportSize({ width, height: 900 })
        await settle(page, theme)
        assert.equal(
          await page.evaluate(
            () => document.documentElement.scrollWidth > innerWidth,
          ),
          false,
          `Overflow ${width} ${theme}`,
        )
        const axe = await new AxeBuilder({ page })
          .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
          .analyze()
        assert.deepEqual(
          axe.violations.map((v) => ({
            id: v.id,
            nodes: v.nodes.map((n) => n.target),
          })),
          [],
          `Agent Axe ${width} ${theme}`,
        )
        await page.screenshot({
          path: `${captures}/agent-${width}-${theme}.png`,
          fullPage: true,
        })
      }
    await page
      .getByRole('button', { name: "Ouvrir l'assistant", exact: true })
      .click()
    const panel = page.locator('.assistant-panel')
    await panel
      .getByRole('textbox', { name: 'Votre question' })
      .fill('QUESTION FLOTTANTE TEST')
    await panel.getByRole('textbox', { name: 'Votre question' }).press('Enter')
    await expect(panel.locator('.assistant-panel-msg-assistant')).toContainText(
      '123,45 €',
    )
    await panel
      .getByRole('textbox', { name: 'Votre question' })
      .fill('BROUILLON CONSERVE TEST')
    const floatingCalls = chatBodies.length
    await panel.getByRole('button', { name: 'Fermer', exact: true }).click()
    await expect(panel).not.toBeVisible()
    await page
      .getByRole('button', { name: "Ouvrir l'assistant", exact: true })
      .click()
    await expect(panel.locator('.assistant-panel-msg-assistant')).toContainText(
      '123,45 €',
    )
    await expect(
      panel.getByRole('textbox', { name: 'Votre question' }),
    ).toHaveValue('BROUILLON CONSERVE TEST')
    assert.equal(
      chatBodies.length,
      floatingCalls,
      'Reopening preserves the exchange without a model call',
    )
    await settle(page)
    assert.deepEqual(
      (
        await new AxeBuilder({ page })
          .include('.assistant-panel')
          .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
          .analyze()
      ).violations.map((v) => ({
        id: v.id,
        nodes: v.nodes.map((n) => n.target),
      })),
      [],
      'Floating agent Axe',
    )
    // A new exchange shows observations without generating a model reply.
    await page.getByRole('button', { name: 'Fermer', exact: true }).click()
    await consoleUI
      .getByRole('button', { name: 'Nouvel échange', exact: true })
      .click()
    assert.equal(
      await consoleUI
        .locator('.assistant-messages')
        .evaluate((el) => el.scrollTop),
      0,
      'A new exchange starts at the briefing, not at the bottom',
    )
    for (const width of [360, 768, 1280])
      for (const theme of ['light', 'dark']) {
        await page.setViewportSize({ width, height: 1100 })
        await page.evaluate(() => window.scrollTo(0, 0))
        await settle(page, theme)
        await expect(
          consoleUI.getByRole('region', {
            name: 'Point sur les données disponibles',
          }),
        ).toContainText('101 dépense(s) sans catégorie')
        assert.equal(
          await page.evaluate(
            () => document.documentElement.scrollWidth > innerWidth,
          ),
          false,
        )
        assert.deepEqual(
          (
            await new AxeBuilder({ page })
              .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
              .analyze()
          ).violations.map((v) => v.id),
          [],
        )
        await page.screenshot({
          path: `${captures}/briefing-${width}-${theme}.png`,
          fullPage: true,
        })
      }
    briefingState = 'empty'
    await page.reload()
    await page.getByRole('tab', { name: 'Assistant', exact: true }).click()
    await expect(
      consoleUI.getByRole('region', {
        name: 'Point sur les données disponibles',
      }),
    ).toContainText('Cela ne signifie pas que tu n’as rien dépensé')
    briefingState = 'failed'
    await page.reload()
    await page.getByRole('tab', { name: 'Assistant', exact: true }).click()
    await expect(consoleUI).toContainText(
      'Une partie du suivi est indisponible',
    )
    await expect(
      consoleUI.getByRole('region', {
        name: 'Point sur les données disponibles',
      }),
    ).toHaveCount(0)
    assert.deepEqual(errors, [])
    console.log(
      'Agent browser PASS: missions, persisted conversations, page context request, real-step display, source selection, preview without write, validation, stale action, preferences/forget, contextual briefing, partial lists, prepared question without model, empty/error states, six layouts/themes and full/floating Axe.',
    )
  } finally {
    await browser.close()
  }
}
main().catch((e) => {
  console.error(e)
  process.exitCode = 1
})
