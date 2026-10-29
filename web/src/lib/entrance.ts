/** The address the printed QR opens: the facility's entrance page, with the token that makes it work. */
export function entranceUrl(slug: string, facilitySlug: string, qrToken: string) {
  return `${window.location.origin}/${slug}/${facilitySlug}/entrada?c=${qrToken}`
}
