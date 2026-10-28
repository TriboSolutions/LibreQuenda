import { ChevronDown, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { TagCategory } from '@/lib/api'

type Props = {
  groups: TagCategory[] // the categories, with only the tags worth offering
  chosen: string[] // tag slugs
  onChange: (slugs: string[]) => void
  clearAll?: boolean // a "clear filters" button once something is chosen
}

/** One dropdown per tag category, each with several choices; a facility has to carry them all. */
export function TagFilters({ groups, chosen, onChange, clearAll = true }: Props) {
  const { t } = useTranslation('home')
  return (
    <>
      {groups.map((group) => {
        const slugs = group.tags.map((tag) => tag.slug)
        const picked = group.tags.filter((tag) => chosen.includes(tag.slug))
        const toggle = (slug: string, on: boolean) => onChange(on ? [...chosen, slug] : chosen.filter((x) => x !== slug))
        return (
          <DropdownMenu key={group.id}>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                className={picked.length > 0 ? 'border-primary text-primary' : ''}
                aria-label={picked.length > 0 ? t('tags.filterBy', { category: group.name, count: picked.length }) : undefined}
              >
                {picked.length === 1 ? picked[0].name : group.name}
                {picked.length > 1 && <span className="num rounded-full bg-primary px-1.5 text-xs text-primary-foreground">{picked.length}</span>}
                <ChevronDown className="opacity-60" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="max-h-80 min-w-52 overflow-y-auto">
              <DropdownMenuLabel>{group.name}</DropdownMenuLabel>
              {group.tags.map((tag) => (
                <DropdownMenuCheckboxItem
                  key={tag.slug}
                  checked={chosen.includes(tag.slug)}
                  onCheckedChange={(on) => toggle(tag.slug, on)}
                  onSelect={(event) => event.preventDefault()} // stays open to pick several
                >
                  {tag.name}
                </DropdownMenuCheckboxItem>
              ))}
              {picked.length > 0 && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={() => onChange(chosen.filter((slug) => !slugs.includes(slug)))}>{t('tags.clearCategory')}</DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )
      })}
      {clearAll && chosen.length > 0 && (
        <Button variant="ghost" onClick={() => onChange([])}>
          <X />
          {t('tags.clear')}
        </Button>
      )}
    </>
  )
}
