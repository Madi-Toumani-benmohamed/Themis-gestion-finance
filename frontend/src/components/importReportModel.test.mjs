import test from 'node:test'
import assert from 'node:assert/strict'
import { importControlLabel, importIssueLabel } from './importReportModel.mjs'

test('un contrôle absent ne devient jamais une réussite', () => {
  assert.equal(importControlLabel('passed'), 'Conforme')
  assert.equal(importControlLabel('failed'), 'Échec')
  assert.equal(importControlLabel('unverifiable'), 'Non vérifiable automatiquement')
  assert.equal(importControlLabel(undefined), 'Non exécuté')
})
test('les lectures ambiguës et opérations perdues ont une justification visible', () => {
  for (const code of ['missing_amount_or_label', 'invalid_or_unknown_date', 'amount_without_operation_date', 'ambiguous_ocr_amount']) {
    assert.notEqual(importIssueLabel(code), 'Lecture non interprétable : comparer au document original')
  }
  assert.match(importIssueLabel('unknown'), /original/)
})
