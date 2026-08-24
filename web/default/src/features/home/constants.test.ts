import assert from 'node:assert/strict'
import test from 'node:test'

import { getPublicModelNames } from './lib/home-content.ts'

test('projects the public model list from server data without inventing names', () => {
  assert.deepEqual(
    getPublicModelNames([
      { model_name: ' deepseek-v4-flash-0731 ' },
      { model_name: 'glm-5.2' },
      { model_name: 'DEEPSEEK-V4-FLASH-0731' },
      { model_name: '' },
      { model_name: 'qwen3.5' },
    ]),
    ['deepseek-v4-flash-0731', 'glm-5.2', 'qwen3.5']
  )
  assert.deepEqual(getPublicModelNames([]), [])
})
