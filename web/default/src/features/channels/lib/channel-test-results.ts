export type ChannelTestResultStatus = 'idle' | 'testing' | 'success' | 'error'

export type ChannelTestResultLike = {
  status: ChannelTestResultStatus
  responseTime?: number
}

export type ChannelTestTarget = {
  model: string
  keyIndex?: number
}

export type ChannelTestSummary = {
  tested: number
  failed: number
  failureRate: number
}

export type ModelTestSummary = ChannelTestSummary & {
  status: ChannelTestResultStatus
  responseTime?: number
}

export function getChannelTestResultKey(model: string, keyIndex?: number) {
  return keyIndex === undefined ? `single:${model}` : `key:${keyIndex}:${model}`
}

export function buildChannelTestTargets(
  models: string[],
  keyCount: number,
  selectedKeyIndex: number | null
): ChannelTestTarget[] {
  const uniqueModels = [
    ...new Set(models.map((model) => model.trim()).filter(Boolean)),
  ]
  if (keyCount <= 1) {
    return uniqueModels.map((model) => ({ model }))
  }
  const keyIndexes =
    selectedKeyIndex === null
      ? Array.from({ length: keyCount }, (_, index) => index)
      : [selectedKeyIndex]

  return uniqueModels.flatMap((model) =>
    keyIndexes.map((keyIndex) => ({ model, keyIndex }))
  )
}

function failureRate(failed: number, tested: number) {
  return tested === 0 ? 0 : Math.round((failed / tested) * 1000) / 10
}

export function summarizeModelTestResults(
  results: Record<string, ChannelTestResultLike>,
  model: string,
  keyIndexes: Array<number | undefined>
): ModelTestSummary {
  const scoped = keyIndexes
    .map((keyIndex) => results[getChannelTestResultKey(model, keyIndex)])
    .filter((result): result is ChannelTestResultLike => Boolean(result))
  const completed = scoped.filter(
    (result) => result.status === 'success' || result.status === 'error'
  )
  const failed = completed.filter((result) => result.status === 'error').length
  const responseTimes = completed
    .map((result) => result.responseTime)
    .filter((value): value is number => typeof value === 'number')

  let status: ChannelTestResultStatus = 'idle'
  if (scoped.some((result) => result.status === 'testing')) {
    status = 'testing'
  } else if (failed > 0) {
    status = 'error'
  } else if (completed.length > 0) {
    status = 'success'
  }

  return {
    status,
    tested: completed.length,
    failed,
    failureRate: failureRate(failed, completed.length),
    ...(responseTimes.length > 0
      ? {
          responseTime:
            responseTimes.reduce((sum, value) => sum + value, 0) /
            responseTimes.length,
        }
      : {}),
  }
}

export function summarizeChannelTestScope(
  results: Record<string, ChannelTestResultLike>,
  models: string[],
  keyIndexes: Array<number | undefined>
): ChannelTestSummary {
  let tested = 0
  let failed = 0
  for (const model of models) {
    const summary = summarizeModelTestResults(results, model, keyIndexes)
    tested += summary.tested
    failed += summary.failed
  }
  return { tested, failed, failureRate: failureRate(failed, tested) }
}
