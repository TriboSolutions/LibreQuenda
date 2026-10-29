import { ChevronDown } from 'lucide-react'
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
import { Skeleton } from '@/components/ui/skeleton'
import { useTags } from '@/hooks/useTags'

type Props = { value: string[]; onChange: (tags: string[]) => void }

/**
 * The facility's tags, from the catalogue every council shares: one line per category,
 * each a dropdown of its tags with what is chosen written on the button.
 */
export function TagPicker({ value, onChange }: Props) {
  const { t } = useTranslation('admin')
  const tags = useTags()
  const toggle = (slug: string, on: boolean) => onChange(on ? [...value, slug] : value.filter((x) => x !== slug))

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
          {tags.categories.map((category) => {
            const picked = category.tags.filter((tag) => value.includes(tag.slug))
            const id = `f-tags-${category.id}`
            return (
              <div key={category.id} className="contents">
                <span id={id} className="text-sm text-muted-foreground">
                  {category.name}
                </span>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" className="w-full justify-between font-normal" aria-labelledby={`${id} ${id}-value`}>
                      <span id={`${id}-value`} className={picked.length === 0 ? 'truncate text-muted-foreground' : 'truncate'}>
                        {picked.length === 0 ? t('editor.tagsNone') : picked.map((tag) => tag.name).join(', ')}
                      </span>
                      <ChevronDown className="shrink-0 opacity-60" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start" className="max-h-80 w-(--radix-dropdown-menu-trigger-width) min-w-56 overflow-y-auto">
                    <DropdownMenuLabel>{category.name}</DropdownMenuLabel>
                    {category.tags.map((tag) => (
                      <DropdownMenuCheckboxItem
                        key={tag.slug}
                        checked={value.includes(tag.slug)}
                        onCheckedChange={(on) => toggle(tag.slug, on)}
                        onSelect={(event) => event.preventDefault()} // stays open to pick several
                      >
                        {tag.name}
                      </DropdownMenuCheckboxItem>
                    ))}
                    {picked.length > 0 && (
                      <>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onSelect={() => onChange(value.filter((slug) => !picked.some((tag) => tag.slug === slug)))}>{t('editor.tagsClear')}</DropdownMenuItem>
                      </>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            )
          })}
        </div>
      )}
    </fieldset>
  )
}
