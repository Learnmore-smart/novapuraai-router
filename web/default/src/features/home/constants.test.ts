import assert from 'node:assert/strict'
import test from 'node:test'

import { LANDING_MODEL_ROWS } from './constants.ts'

test('advertises current flagship models instead of unavailable provider families', () => {
  assert.deepEqual(LANDING_MODEL_ROWS, [
    { name: 'GLM 5.2', note: 'Pay per token' },
    { name: 'DeepSeek V4 Pro', note: 'Pay per token' },
    { name: 'Kimi K2.6', note: 'Pay per token' },
    { name: 'Nemotron 3 Ultra', note: 'Pay per token' },
  ])

  const advertisedNames = LANDING_MODEL_ROWS.map((model) => model.name).join(
    ' '
  )
  assert.doesNotMatch(advertisedNames, /GPT|Claude|Gemini/i)
})
