import { ChevronDown, Search } from 'lucide-react'
import { useRef, useState, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import { useTags } from '@/hooks/useTags'
import type { TagCategory } from '@/lib/api'

type Props = { value: string[]; onChange: (tags: string[]) => void }

// "Bádminton" is found typing "badminton", and "PÁDEL" typing "padel"
const fold = (text: string) => text.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase()

/**
 * The facility's tags, from the catalogue every council shares: one line per category,
 * each opening a searchable list of its tags, with what is chosen written on the button.
 */
export function TagPicker({ value, onChange }: Props) {
  const { t } = useTranslation('admin')
  const tags = useTags()

  return (
    <fieldset className="grid gap-2" aria-describedby="f-tags-help">
      <legend className="text-sm leading-none font-medium">{t('editor.tags')}</legend>
      <p id="f-tags-help" className="mt-1 text-xs text-muted-foreground">
        {t('editor.tagsHelp')}
      </p>
      {tags.loading && tags.categories.length === 0 ? (
        <Skeleton className="h-28 w-full" />
      ) : tags.categories.length === 0 ? (
        <p className="text-muted-foreground">{t('editor.tagsEmpty')}</p>
      ) : (
        <div className="grid gap-2 sm:grid-cols-[10rem_minmax(0,1fr)] sm:items-center">
          {tags.categories.map((category) => (
            <CategoryPicker key={category.id} category={category} value={value} onChange={onChange} />
          ))}
        </div>
      )}
    </fieldset>
  )
}

function CategoryPicker({ category, value, onChange }: Props & { category: TagCategory }) {
  const { t } = useTranslation('admin')
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const list = useRef<HTMLUListElement>(null)
  const id = `f-tags-${category.id}`
  const picked = category.tags.filter((tag) => value.includes(tag.slug))
  const shown = query.trim() ? category.tags.filter((tag) => fold(tag.name).includes(fold(query.trim()))) : category.tags
  const toggle = (slug: string, on: boolean) => onChange(on ? [...value, slug] : value.filter((x) => x !== slug))

  const onSearchKey = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      list.current?.querySelector<HTMLButtonElement>('button[role="checkbox"]')?.focus()
    } else if (event.key === 'Enter') {
      event.preventDefault() // never submits the facility's form
      // with a search typed, Enter picks (or drops) the first match
      const first = query.trim() ? shown[0] : undefined
      if (first) toggle(first.slug, !value.includes(first.slug))
    }
  }

  return (
    <div className="contents">
      <span id={id} className="text-sm text-muted-foreground">
        {category.name}
      </span>
      <Popover
        open={open}
        onOpenChange={(next) => {
          setOpen(next)
          if (!next) setQuery('')
        }}
      >
        <PopoverTrigger asChild>
          <Button variant="outline" className="w-full justify-between font-normal" aria-labelledby={`${id} ${id}-value`}>
            <span id={`${id}-value`} className={picked.length === 0 ? 'truncate text-muted-foreground' : 'truncate'}>
              {picked.length === 0 ? t('editor.tagsNone') : picked.map((tag) => tag.name).join(', ')}
            </span>
            <ChevronDown className="shrink-0 opacity-60" />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" aria-label={category.name} className="grid w-(--radix-popover-trigger-width) min-w-56 gap-2 p-2">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
            <Input
              type="search"
              autoFocus
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={onSearchKey}
              aria-label={t('editor.tagsSearch', { category: category.name })}
              aria-controls={shown.length > 0 ? `${id}-list` : undefined}
              placeholder={t('editor.tagsSearch', { category: category.name })}
              className="pl-8"
            />
          </div>
          {shown.length === 0 ? (
            <p role="status" className="px-2 py-3 text-sm text-muted-foreground">
              {t('editor.tagsNoMatch')}
            </p>
          ) : (
            <ul ref={list} id={`${id}-list`} aria-label={category.name} className="grid max-h-64 overflow-y-auto">
              {shown.map((tag) => {
                const box = `${id}-${tag.slug}`
                return (
                  <li key={tag.slug}>
                    <label htmlFor={box} className="flex cursor-pointer items-center gap-3 rounded-sm px-2 py-2 text-sm hover:bg-accent hover:text-accent-foreground">
                      <Checkbox id={box} checked={value.includes(tag.slug)} onCheckedChange={(on) => toggle(tag.slug, on === true)} />
                      {tag.name}
                    </label>
                  </li>
                )
              })}
            </ul>
          )}
          {picked.length > 0 && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="justify-start"
              onClick={() => onChange(value.filter((slug) => !picked.some((tag) => tag.slug === slug)))}
            >
              {t('editor.tagsClear')}
            </Button>
          )}
        </PopoverContent>
      </Popover>
    </div>
  )
}
