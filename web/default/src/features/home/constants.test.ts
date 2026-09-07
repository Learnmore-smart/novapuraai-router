import assert from 'node:assert/strict'
import test from 'node:test'

import { getPublicModelNames } from './lib/home-content.ts'

test('prefers the featured home models when they exist in the catalogue', () => {
  assert.deepEqual(
    getPublicModelNames([
      { model_name: 'nvidia/neva-22b' },
      { model_name: ' kimi-k3 ' },
      { model_name: 'deepseek-v4-pro-0813' },
      { model_name: 'glm-5.2' },
      { model_name: 'deepseek-v4-flash-0731' },
    ]),
    ['kimi-k3', 'deepseek-v4-flash-0731', 'deepseek-v4-pro-0813']
  )
})

test('does not invent featured names that are not hosted', () => {
  assert.deepEqual(getPublicModelNames([{ model_name: 'kimi-k3' }]), [
    'kimi-k3',
  ])
})

test('falls back to catalogue order when none of the featured models are hosted', () => {
  assert.deepEqual(
    getPublicModelNames([
      { model_name: ' nvidia/neva-22b ' },
      { model_name: 'NVIDIA/NEVA-22B' },
      { model_name: '' },
      { model_name: 'glm-5.2' },
    ]),
    ['nvidia/neva-22b', 'glm-5.2']
  )
})

test('projects an empty list when the public catalogue has no names', () => {
  assert.deepEqual(getPublicModelNames([]), [])
})
