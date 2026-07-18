import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

const topupSearchSchema = z.record(z.string(), z.unknown()).catch({})

export const Route = createFileRoute('/console/topup')({
  validateSearch: topupSearchSchema,
  beforeLoad: ({ search }) => {
    throw redirect({
      to: '/wallet',
      search: { show_history: true, ...search },
    })
  },
})
