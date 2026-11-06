import * as React from "react"
import { useTranslation } from "react-i18next"
import { cn } from "@/lib/utils"

/**
 * A text box of several lines. With a maxLength (and not read-only) it shows how
 * many characters are used of how many, under it, and is described by that count.
 */
function Textarea({ className, maxLength, readOnly, value, defaultValue, onChange, ...props }: React.ComponentProps<"textarea">) {
  const { t } = useTranslation()
  const countId = React.useId()
  const [typed, setTyped] = React.useState(String(defaultValue ?? ""))
  const field = (
    <textarea
      data-slot="textarea"
      className={cn(
        "flex field-sizing-content min-h-16 w-full rounded-md border border-input bg-transparent px-3 py-2 text-base transition-[color,box-shadow] outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 md:text-sm dark:bg-input/30 dark:aria-invalid:ring-destructive/40",
        className
      )}
      maxLength={maxLength}
      readOnly={readOnly}
      value={value}
      defaultValue={defaultValue}
      onChange={(e) => {
        setTyped(e.target.value)
        onChange?.(e)
      }}
      {...props}
      aria-describedby={maxLength && !readOnly ? cn(props["aria-describedby"], countId) : props["aria-describedby"]}
    />
  )
  if (!maxLength || readOnly) return field

  const used = (value === undefined ? typed : String(value)).length
  return (
    <div className="grid min-w-0 gap-1">
      {field}
      <p id={countId} className={cn("num justify-self-end text-xs", used >= maxLength ? "text-foreground" : "text-muted-foreground")}>
        {t("textarea.count", { used, max: maxLength })}
      </p>
    </div>
  )
}

export { Textarea }
