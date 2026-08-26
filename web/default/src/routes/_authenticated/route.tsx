import { createFileRoute, redirect } from '@tanstack/react-router'

import { AuthenticatedLayout } from '@/components/layout'
import { getSelf } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ location }) => {
    const { auth } = useAuthStore.getState()

    // 如果本地没有用户信息，直接跳转登录页
    if (!auth.user) {
      throw redirect({
        to: '/sign-in',
        search: { redirect: location.href },
      })
    }

    // Always refresh from /api/user/self so role and permission changes
    // (for example, promote to root) apply without a full re-login.
    // 仅 401 视为 session 失效；网络错误/超时/5xx 返回 null 放行，下次导航重验
    const res = await getSelf().catch((err: unknown) =>
      (err as { response?: { status?: number } })?.response?.status === 401
        ? { success: false }
        : null
    )
    if (res?.success && res.data) {
      auth.setUser(res.data)
    } else if (res) {
      auth.reset()
      throw redirect({
        to: '/sign-in',
        search: { redirect: location.href },
      })
    }
  },
  component: AuthenticatedLayout,
})
