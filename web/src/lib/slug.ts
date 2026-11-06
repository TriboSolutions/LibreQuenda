/** "Concello da Estrada" -> "estrada" ; "A Pobra do Caramiñal" -> "a-pobra-do-caraminal" */
export function slugify(name: string) {
  return name
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .toLowerCase()
    .replace(/^\s*concello\s+(de|da|do|das|dos)\s+/, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 64)
    .replace(/-+$/, '') // cut at 64, it must not end in a dash
}
