import { z } from 'zod'

import {
  type PermissionCatalog,
  type AdminPermissionMatrix,
  normalizeAdminPermissions,
} from '@/lib/admin-permissions'
import { quotaUnitsToDollars } from '@/lib/format'
import { ROLE } from '@/lib/roles'

import { DEFAULT_GROUP } from '../constants'
import { type UserFormData, type User } from '../types'

// ============================================================================
// Form Schema
// ============================================================================

export const userFormSchema = z.object({
  username: z.string().min(1, 'Username is required'),
  display_name: z.string().optional(),
  password: z.string().optional(),
  role: z.number().optional(),
  quota_dollars: z.number().min(0).optional(),
  group: z.string().optional(),
  remark: z.string().optional(),
  commission_approved: z.boolean().optional(),
  admin_permissions: z
    .record(z.string(), z.record(z.string(), z.boolean()))
    .optional(),
})

export type UserFormValues = z.infer<typeof userFormSchema>

// ============================================================================
// Form Defaults
// ============================================================================

export const USER_FORM_DEFAULT_VALUES: UserFormValues = {
  username: '',
  display_name: '',
  password: '',
  role: 1, // Default to common user
  quota_dollars: 0,
  group: DEFAULT_GROUP,
  remark: '',
  commission_approved: false,
  // Filled against the backend catalog at render time; see UsersMutateDrawer.
  admin_permissions: {},
}

// ============================================================================
// Form Data Transformation
// ============================================================================

/**
 * Transform form data to API payload
 */
export function transformFormDataToPayload(
  data: UserFormValues,
  userId?: number,
  catalog?: PermissionCatalog
): UserFormData & { id?: number } {
  const payload: UserFormData & { id?: number } = {
    username: data.username,
    display_name: data.display_name || data.username,
    password: data.password || undefined,
  }

  const role = userId === undefined ? data.role || 1 : (data.role ?? 0)

  // Only send the permission matrix when the target is an admin and the catalog
  // is available; without the catalog we cannot build a full matrix, so we omit
  // the field (the backend then leaves existing permissions untouched).
  if (role >= ROLE.ADMIN && catalog) {
    payload.admin_permissions = normalizeAdminPermissions(
      data.admin_permissions as AdminPermissionMatrix | undefined,
      catalog
    )
  }

  // For create: only send required fields
  if (userId === undefined) {
    payload.role = role
  } else {
    // For update: quota is adjusted atomically via /api/user/manage, not sent here
    payload.group = data.group
    payload.remark = data.remark || undefined
    payload.commission_approved = data.commission_approved ?? false
    payload.id = userId
  }

  return payload
}

/**
 * Transform user data to form defaults. The admin permission matrix is passed
 * through as-is (the backend already returns a full matrix); it is filled against
 * the catalog at render time in UsersMutateDrawer.
 */
export function transformUserToFormDefaults(user: User): UserFormValues {
  return {
    username: user.username,
    display_name: user.display_name,
    password: '',
    role: user.role,
    quota_dollars: quotaUnitsToDollars(user.quota),
    group: user.group || DEFAULT_GROUP,
    remark: user.remark || '',
    commission_approved: user.commission_approved ?? false,
    admin_permissions: user.admin_permissions ?? {},
  }
}
