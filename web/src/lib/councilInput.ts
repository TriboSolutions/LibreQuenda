import type { CouncilDetail, CouncilInput } from '@/lib/api'

/** The form's starting values: a council's saved ones, or a new council's. */
export function councilInput(council: CouncilDetail | null): CouncilInput {
  return {
    name: council?.name ?? '',
    slug: council?.slug ?? '',
    timezone: council?.timezone ?? 'Europe/Madrid',
    contact_email: council?.contact_email ?? '',
    contact_phone: council?.contact_phone ?? '',
    municipality_code: council?.municipality_code ?? '',
    logo_ref: council?.logo_ref ?? '',
    is_active: council?.is_active ?? true,
    admin_email: '',
  }
}
