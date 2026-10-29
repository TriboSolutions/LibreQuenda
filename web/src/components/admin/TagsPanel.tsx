import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { ArrowDown, ArrowUp, Pencil, Plus, Trash2 } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type Tag, type TagCatalogue, type TagCategory } from '@/lib/api'
import { slugify } from '@/lib/slug'

// What the open dialog edits: a category's name, a tag, or a delete to confirm
type Editing =
  | { kind: 'category'; category: TagCategory }
  | { kind: 'deleteCategory'; category: TagCategory }
  | { kind: 'tag'; tag: Tag; categoryId: string }
  | null

/**
 * The superadministrator's tag catalogue: categories ("Deportes", "Servizos"…) and their tags.
 * Councils tag their facilities from it, and residents filter by it on every page.
 */
export function TagsPanel() {
  const { t } = useTranslation(['admin', 'common'])
  const token = useAuth().user?.access_token ?? ''
  const loaded = useAsync(() => api.tags(), 'admin:tags')
  // every change answers with the whole catalogue: show that until the next load
  const [saved, setSaved] = useState<TagCatalogue | null>(null)
  const catalogue = saved ?? loaded.data
  const [newCategory, setNewCategory] = useState('')
  const [editing, setEditing] = useState<Editing>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  /** Runs a change; true when it worked. Errors show above the catalogue, or in the open dialog. */
  const run = async (change: () => Promise<TagCatalogue>) => {
    setBusy(true)
    setError(null)
    try {
      setSaved(await change())
      return true
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return false
    } finally {
      setBusy(false)
    }
  }

  const addCategory = async (event: React.FormEvent) => {
    event.preventDefault()
    if (await run(() => api.admin.createTagCategory(token, newCategory.trim()))) setNewCategory('')
  }

  // swap places with the neighbour, then number them all 1, 2, 3… (saving only those that changed)
  const move = (categories: TagCategory[], index: number, by: -1 | 1) => {
    const order = [...categories]
    ;[order[index], order[index + by]] = [order[index + by], order[index]]
    void run(async () => {
      let latest: TagCatalogue = { categories }
      for (const [i, c] of order.entries()) {
        if (c.position !== i + 1) latest = await api.admin.updateTagCategory(token, c.id, { name: c.name, position: i + 1 })
      }
      return latest
    })
  }

  const categories = catalogue?.categories ?? []

  return (
    <section aria-labelledby="tags-title" className="grid gap-4">
      <div>
        <h2 id="tags-title" className="text-lg font-semibold">
          {t('admin:tabs.tags')}
        </h2>
        <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('admin:tags.help')}</p>
      </div>

      <form onSubmit={(e) => void addCategory(e)} className="grid max-w-md gap-2">
        <Label htmlFor="tag-category-new">{t('admin:tags.newCategory')}</Label>
        <div className="flex gap-2">
          <Input id="tag-category-new" required maxLength={60} placeholder={t('admin:tags.newCategoryPlaceholder')} value={newCategory} onChange={(e) => setNewCategory(e.target.value)} />
          <Button type="submit" variant="outline" disabled={busy}>
            <Plus />
            {t('admin:tags.addCategory')}
          </Button>
        </div>
      </form>

      <div aria-live="polite" className="empty:hidden">
        {error && !editing && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>

      {loaded.error && !catalogue ? (
        <p className="text-destructive">{loaded.error.message}</p>
      ) : !catalogue ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <ul className="grid gap-4 lg:grid-cols-2">
          {categories.map((category, index) => (
            <li key={category.id}>
              <CategoryCard
                category={category}
                busy={busy}
                first={index === 0}
                last={index === categories.length - 1}
                onMove={(by) => move(categories, index, by)}
                onRename={() => setEditing({ kind: 'category', category })}
                onDelete={() => setEditing({ kind: 'deleteCategory', category })}
                onEditTag={(tag) => setEditing({ kind: 'tag', tag, categoryId: category.id })}
                onAddTag={(name) => run(() => api.admin.createTag(token, { category_id: category.id, name, slug: slugify(name) }))}
              />
            </li>
          ))}
        </ul>
      )}


      {editing?.kind === 'category' && (
        <RenameCategoryDialog
          category={editing.category}
          busy={busy}
          error={error}
          onClose={() => {
            setEditing(null)
            setError(null)
          }}
          onSave={async (name) => {
            if (await run(() => api.admin.updateTagCategory(token, editing.category.id, { name }))) setEditing(null)
          }}
        />
      )}
      {editing?.kind === 'deleteCategory' && (
        <DeleteCategoryDialog
          category={editing.category}
          busy={busy}
          error={error}
          onClose={() => {
            setEditing(null)
            setError(null)
          }}
          onConfirm={async () => {
            if (await run(() => api.admin.deleteTagCategory(token, editing.category.id))) setEditing(null)
          }}
        />
      )}
      {editing?.kind === 'tag' && (
        <TagDialog
          tag={editing.tag}
          categoryId={editing.categoryId}
          categories={categories}
          busy={busy}
          error={error}
          onClose={() => {
            setEditing(null)
            setError(null)
          }}
          onSave={async (name, categoryId) => {
            if (await run(() => api.admin.updateTag(token, editing.tag.id, { name, category_id: categoryId }))) setEditing(null)
          }}
          onDelete={async () => {
            if (await run(() => api.admin.deleteTag(token, editing.tag.id))) setEditing(null)
          }}
        />
      )}
    </section>
  )
}

type CardProps = {
  category: TagCategory
  busy: boolean
  first: boolean
  last: boolean
  onMove: (by: -1 | 1) => void
  onRename: () => void
  onDelete: () => void
  onEditTag: (tag: Tag) => void
  onAddTag: (name: string) => Promise<boolean>
}

function CategoryCard({ category, busy, first, last, onMove, onRename, onDelete, onEditTag, onAddTag }: CardProps) {
  const { t } = useTranslation('admin')
  const [name, setName] = useState('')
  const inputId = `tag-new-${category.id}`

  const add = async (event: React.FormEvent) => {
    event.preventDefault()
    if (await onAddTag(name.trim())) setName('')
  }

  return (
    <article aria-labelledby={`tag-category-${category.id}`} className="flex h-full flex-col gap-3 rounded-lg border bg-card p-4">
      <header className="flex items-center gap-2">
        <h3 id={`tag-category-${category.id}`} className="min-w-0 flex-1 truncate font-semibold">
          {category.name} <span className="num font-normal text-muted-foreground">· {category.tags.length}</span>
        </h3>
        <Button variant="ghost" size="icon" disabled={busy || first} onClick={() => onMove(-1)} aria-label={t('tags.moveUp', { name: category.name })}>
          <ArrowUp />
        </Button>
        <Button variant="ghost" size="icon" disabled={busy || last} onClick={() => onMove(1)} aria-label={t('tags.moveDown', { name: category.name })}>
          <ArrowDown />
        </Button>
        <Button variant="ghost" size="icon" onClick={onRename} aria-label={t('tags.renameCategory', { name: category.name })}>
          <Pencil />
        </Button>
        <Button variant="ghost" size="icon" className="text-destructive hover:text-destructive" onClick={onDelete} aria-label={t('tags.deleteCategory', { name: category.name })}>
          <Trash2 />
        </Button>
      </header>

      {category.tags.length === 0 ? (
        <p className="text-muted-foreground">{t('tags.noTags')}</p>
      ) : (
        <ul className="flex flex-wrap gap-2">
          {category.tags.map((tag) => (
            <li key={tag.id}>
              <button
                type="button"
                onClick={() => onEditTag(tag)}
                className="inline-flex min-h-8 items-center gap-1.5 rounded-full border px-3 text-sm hover:border-primary hover:bg-muted"
              >
                {tag.name}
                <span className="num text-xs text-muted-foreground">
                  {tag.facility_count}
                  <span className="sr-only"> {t('tags.facilitiesUsing', { count: tag.facility_count })}</span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}

      <form onSubmit={(e) => void add(e)} className="mt-auto flex gap-2 border-t pt-3">
        <Label htmlFor={inputId} className="sr-only">
          {t('tags.newTagIn', { category: category.name })}
        </Label>
        <Input id={inputId} required maxLength={40} placeholder={t('tags.newTagPlaceholder')} value={name} onChange={(e) => setName(e.target.value)} className="h-8" />
        <Button type="submit" size="sm" variant="outline" disabled={busy}>
          <Plus />
          {t('tags.addTag')}
        </Button>
      </form>
    </article>
  )
}

function DialogError({ error }: { error: string | null }) {
  return (
    <div aria-live="polite" className="empty:hidden">
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
    </div>
  )
}

type RenameProps = { category: TagCategory; busy: boolean; error: string | null; onClose: () => void; onSave: (name: string) => Promise<void> }

function RenameCategoryDialog({ category, busy, error, onClose, onSave }: RenameProps) {
  const { t } = useTranslation('admin')
  const [name, setName] = useState(category.name)
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('tags.renameTitle')}</DialogTitle>
          <DialogDescription>{category.name}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            void onSave(name.trim())
          }}
          className="grid gap-4"
        >
          <div className="grid gap-1.5">
            <Label htmlFor="tag-category-name">{t('tags.categoryName')}</Label>
            <Input id="tag-category-name" required maxLength={60} value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <DialogError error={error} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('staff.close')}
            </Button>
            <Button type="submit" disabled={busy}>
              {t('tags.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

type DeleteCategoryProps = { category: TagCategory; busy: boolean; error: string | null; onClose: () => void; onConfirm: () => Promise<void> }

function DeleteCategoryDialog({ category, busy, error, onClose, onConfirm }: DeleteCategoryProps) {
  const { t } = useTranslation('admin')
  const facilities = category.tags.reduce((total, tag) => total + tag.facility_count, 0)
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('tags.deleteCategoryTitle', { name: category.name })}</DialogTitle>
          <DialogDescription>{t('tags.deleteCategoryBody', { count: category.tags.length, facilities })}</DialogDescription>
        </DialogHeader>
        <DialogError error={error} />
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('staff.close')}
          </Button>
          <Button variant="destructive" disabled={busy} onClick={() => void onConfirm()}>
            {t('tags.delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type TagDialogProps = {
  tag: Tag
  categoryId: string
  categories: TagCategory[]
  busy: boolean
  error: string | null
  onClose: () => void
  onSave: (name: string, categoryId: string) => Promise<void>
  onDelete: () => Promise<void>
}

/** One tag: rename it, move it to another category, or delete it (from every facility too). */
function TagDialog({ tag, categoryId, categories, busy, error, onClose, onSave, onDelete }: TagDialogProps) {
  const { t } = useTranslation('admin')
  const [name, setName] = useState(tag.name)
  const [category, setCategory] = useState(categoryId)
  const [confirmDelete, setConfirmDelete] = useState(false)
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('tags.editTitle')}</DialogTitle>
          <DialogDescription>{t('tags.usedBy', { count: tag.facility_count })}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            void onSave(name.trim(), category)
          }}
          className="grid gap-4"
        >
          <div className="grid gap-1.5">
            <Label htmlFor="tag-name">{t('tags.tagName')}</Label>
            <Input id="tag-name" required maxLength={40} value={name} onChange={(e) => setName(e.target.value)} aria-describedby="tag-slug-help" />
            <p id="tag-slug-help" className="text-xs text-muted-foreground">
              {t('tags.slugHelp', { slug: tag.slug })}
            </p>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="tag-category">{t('tags.category')}</Label>
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger id="tag-category" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {categories.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {confirmDelete && (
            <Alert variant="destructive">
              <AlertDescription>{t('tags.deleteTagBody', { count: tag.facility_count })}</AlertDescription>
            </Alert>
          )}
          <DialogError error={error} />
          <DialogFooter className="sm:justify-between">
            {confirmDelete ? (
              <Button type="button" variant="destructive" disabled={busy} onClick={() => void onDelete()}>
                <Trash2 />
                {t('tags.confirmDeleteTag')}
              </Button>
            ) : (
              <Button type="button" variant="ghost" className="text-destructive hover:text-destructive" onClick={() => setConfirmDelete(true)}>
                <Trash2 />
                {t('tags.delete')}
              </Button>
            )}
            <div className="flex gap-2">
              <Button type="button" variant="outline" onClick={onClose}>
                {t('staff.close')}
              </Button>
              <Button type="submit" disabled={busy}>
                {t('tags.save')}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
