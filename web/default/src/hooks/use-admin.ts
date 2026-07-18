import { ROLE } from '@/lib/roles'
/**
 * Hook for checking admin privileges
 */
import { useAuthStore } from '@/stores/auth-store'

/**
 * Check if current user has admin privileges
 */
export function useIsAdmin(): boolean {
  const { user } = useAuthStore((state) => state.auth)
  return (user?.role ?? 0) >= ROLE.ADMIN
}
