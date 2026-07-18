import { createFileRoute } from '@tanstack/react-router'
import z from 'zod'

import { ApiKeys } from '@/features/keys'
import { API_KEY_STATUS_OPTIONS } from '@/features/keys/constants'

const apiKeySearchSchema = z.object({
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(undefined),
  status: z
    .array(z.enum(API_KEY_STATUS_OPTIONS.map((s) => s.value as `${number}`)))
    .optional()
    .catch([]),
  filter: z.string().optional().catch(''),
  token: z.string().optional().catch(''),
})

export const Route = createFileRoute('/_authenticated/keys/')({
  validateSearch: apiKeySearchSchema,
  component: ApiKeys,
})
