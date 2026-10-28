import { ImageIcon, Loader2 } from 'lucide-react'
import { useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Button } from '@/components/ui/button'
import { api, type MediaFolder, type StoredImage } from '@/lib/api'
import { cn } from '@/lib/utils'

const MAX_SIZE = 5 * 1024 * 1024
const TYPES = ['image/jpeg', 'image/png', 'image/webp']

type Props = {
  label: string
  folder: MediaFolder
  value: StoredImage | null // the form keeps the ref; the url is only for the preview
  onChange: (value: StoredImage | null) => void
  shape?: 'square' | 'wide' | 'large' | 'avatar' // large: full width, as tall as a map; avatar: a 96px circle
  hideLabel?: boolean // when the card's heading already names it
}

/**
 * Picks an image, uploads it to the storage (POST /media) and hands back
 * { ref, url }. Forms save the ref; the url is only ever used to show it.
 */
export function ImagePicker({ label, folder, value, onChange, shape = 'square', hideLabel = false }: Props) {
  const { t } = useTranslation()
  const auth = useAuth()
  const id = useId()
  const input = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [dragging, setDragging] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const upload = async (file: File | undefined) => {
    if (!file) return
    setError(null)
    if (!TYPES.includes(file.type)) return setError(t('imagePicker.wrongType'))
    if (file.size > MAX_SIZE) return setError(t('imagePicker.tooLarge'))
    const token = auth.user?.access_token
    if (!token) return
    setUploading(true)
    try {
      onChange(await api.uploadImage(token, folder, file))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setUploading(false)
      if (input.current) input.current.value = '' // the same file can be picked again
    }
  }

  return (
    <div className="grid gap-2">
      <span id={`${id}-label`} className={cn('text-sm leading-none font-medium', hideLabel && 'sr-only')}>
        {label}
      </span>
      <div className={cn('flex gap-4', shape === 'large' ? 'flex-col' : 'items-center')}>
        <button
          type="button"
          aria-labelledby={`${id}-label`}
          aria-describedby={`${id}-hint`}
          disabled={uploading}
          onClick={() => input.current?.click()}
          onDragOver={(e) => {
            e.preventDefault()
            setDragging(true)
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={(e) => {
            e.preventDefault()
            setDragging(false)
            void upload(e.dataTransfer.files[0])
          }}
          className={cn(
            'relative flex shrink-0 items-center justify-center overflow-hidden border bg-muted text-muted-foreground transition-colors',
            shape === 'avatar' ? 'size-24 rounded-full' : 'rounded-md',
            shape === 'square' ? 'size-20' : shape === 'wide' ? 'h-20 w-32' : shape === 'large' ? 'h-64 w-full' : '',
            value ? 'border-solid' : 'border-dashed',
            dragging && 'border-primary bg-accent',
          )}
        >
          {value ? (
            <img src={value.url} alt="" className={cn('size-full', shape === 'square' ? 'object-contain p-1' : 'object-cover')} />
          ) : (
            <ImageIcon className={shape === 'large' || shape === 'avatar' ? 'size-10' : 'size-6'} aria-hidden="true" />
          )}
          {uploading && (
            <span className="absolute inset-0 flex items-center justify-center bg-card/80">
              <Loader2 className="size-5 animate-spin" aria-hidden="true" />
            </span>
          )}
          <span className="sr-only">{value ? t('imagePicker.change') : t('imagePicker.upload')}</span>
        </button>

        <div className="grid gap-1.5">
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" size="sm" disabled={uploading} onClick={() => input.current?.click()}>
              {uploading ? t('imagePicker.uploading') : value ? t('imagePicker.change') : t('imagePicker.upload')}
            </Button>
            {value && !uploading && (
              <Button type="button" variant="ghost" size="sm" onClick={() => onChange(null)}>
                {t('imagePicker.remove')}
              </Button>
            )}
          </div>
          <p id={`${id}-hint`} className="text-xs text-muted-foreground">
            {t('imagePicker.hint')}
          </p>
        </div>
      </div>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      <input
        ref={input}
        type="file"
        accept={TYPES.join(',')}
        className="hidden"
        tabIndex={-1}
        onChange={(e) => void upload(e.target.files?.[0])}
      />
    </div>
  )
}
