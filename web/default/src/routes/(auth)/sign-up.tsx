import { createFileRoute, redirect } from '@tanstack/react-router'

import { SignUp } from '@/features/auth/sign-up'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/(auth)/sign-up')({
  component: SignUp,
  beforeLoad: async () => {
    const { auth } = useAuthStore.getState()

    // 如果已经有用户信息，说明已登录，注册页对其无意义，跳转到 dashboard
    if (auth.user) {
      throw redirect({ to: '/dashboard' })
    }
  },
})
