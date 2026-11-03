import { useAsync } from '@/hooks/useAsync'
import { api, type TagCategory } from '@/lib/api'

/**
 * The platform's tag catalogue, and what screens need from it: a tag's name by its
 * slug, and tags put in the catalogue's order (by category, then name). Slugs no
 * longer in the catalogue are left out.
 */
export function useTags() {
  const catalogue = useAsync(() => api.tags(), 'tags')
  const categories = catalogue.data?.categories ?? []
  const names = new Map(categories.flatMap((c) => c.tags).map((tag) => [tag.slug, tag.name]))

  const label = (slug: string) => names.get(slug) ?? slug
  /** The categories with only the given tags, empty ones left out. */
  const grouped = (slugs: Iterable<string>): TagCategory[] => {
    const present = new Set(slugs)
    return categories.map((c) => ({ ...c, tags: c.tags.filter((tag) => present.has(tag.slug)) })).filter((c) => c.tags.length > 0)
  }

  return { categories, label, grouped, loading: catalogue.loading, error: catalogue.error, reload: catalogue.reload }
}
