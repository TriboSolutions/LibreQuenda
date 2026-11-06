import { useTranslation } from 'react-i18next'
import { Bar, BarChart, CartesianGrid, Cell, Line, LineChart, Pie, PieChart, XAxis, YAxis } from 'recharts'
import { type ChartConfig, ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart'
import type { AnalyticsChart } from '@/lib/api'

const color = (i: number) => `var(--chart-s${i % 6})`
const formatNumber = (value: unknown) => (typeof value === 'number' ? value.toLocaleString(undefined, { maximumFractionDigits: 2 }) : '—')

/**
 * A chart the data assistant asked for, drawn like the indicators' ones; its data is
 * what its SQL returned. Every chart has its table below, for screen readers and to
 * read the exact figures.
 */
export function AnswerChart({ chart }: { chart: AnalyticsChart }) {
  const { t } = useTranslation('admin')
  const config = Object.fromEntries(chart.series.map((s, i) => [s.key, { label: s.label, color: color(i) }])) satisfies ChartConfig
  const titleId = `chart-${chart.title.replace(/\W+/g, '-')}`

  let drawing
  switch (chart.kind) {
    case 'line':
      drawing = (
        <ChartContainer config={config} className="aspect-auto h-64 w-full">
          <LineChart data={chart.rows} accessibilityLayer>
            <CartesianGrid vertical={false} />
            <XAxis dataKey="x" tickLine={false} axisLine={false} tickMargin={8} minTickGap={16} />
            <YAxis tickLine={false} axisLine={false} width={40} />
            <ChartTooltip content={<ChartTooltipContent />} />
            {chart.series.length > 1 && <ChartLegend content={<ChartLegendContent />} />}
            {chart.series.map((s) => (
              <Line key={s.key} dataKey={s.key} type="monotone" stroke={`var(--color-${s.key})`} strokeWidth={2} dot={chart.rows.length <= 31} />
            ))}
          </LineChart>
        </ChartContainer>
      )
      break
    case 'pie':
      drawing = <PieDrawing chart={chart} />
      break
    case 'heatmap':
      drawing = <Heatmap chart={chart} />
      break
    default: {
      const horizontal = chart.kind === 'horizontal_bar'
      const stacked = chart.kind === 'stacked_bar'
      drawing = (
        <ChartContainer config={config} className="aspect-auto w-full" style={{ height: horizontal ? Math.max(160, chart.rows.length * 28 + 40) : 256 }}>
          <BarChart data={chart.rows} layout={horizontal ? 'vertical' : 'horizontal'} accessibilityLayer>
            <CartesianGrid vertical={horizontal} horizontal={!horizontal} />
            {horizontal ? (
              <>
                <XAxis type="number" tickLine={false} axisLine={false} />
                <YAxis type="category" dataKey="x" tickLine={false} axisLine={false} width={140} />
              </>
            ) : (
              <>
                <XAxis dataKey="x" tickLine={false} axisLine={false} tickMargin={8} minTickGap={8} />
                <YAxis tickLine={false} axisLine={false} width={40} />
              </>
            )}
            <ChartTooltip content={<ChartTooltipContent />} />
            {chart.series.length > 1 && <ChartLegend content={<ChartLegendContent />} />}
            {chart.series.map((s, i) => (
              <Bar
                key={s.key}
                dataKey={s.key}
                stackId={stacked ? 'stack' : undefined}
                fill={`var(--color-${s.key})`}
                radius={stacked ? (i === chart.series.length - 1 ? [4, 4, 0, 0] : 0) : horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0]}
              />
            ))}
          </BarChart>
        </ChartContainer>
      )
    }
  }

  return (
    <figure aria-labelledby={titleId} className="grid gap-3 rounded-lg border p-4">
      <figcaption id={titleId} className="font-medium">
        {chart.title}
      </figcaption>
      {drawing}
      <details>
        <summary className="cursor-pointer text-sm text-muted-foreground">{t('indicators.showTable')}</summary>
        <div className="mt-2 max-h-72 overflow-auto">
          <table className="num w-full text-sm">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <td />
                {chart.y && <td />}
                {chart.series.map((s) => (
                  <th key={s.key} scope="col" className="py-1.5 text-right font-normal">
                    {s.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {chart.rows.map((row, r) => (
                <tr key={r} className="border-t">
                  {chart.y && <td className="py-1.5 pr-4 text-left">{row.y}</td>}
                  <th scope="row" className="py-1.5 pr-4 text-left font-normal">
                    {row.x}
                  </th>
                  {chart.series.map((s) => (
                    <td key={s.key} className="py-1.5 text-right">
                      {formatNumber(row[s.key])}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </figure>
  )
}

function PieDrawing({ chart }: { chart: AnalyticsChart }) {
  const key = chart.series[0].key
  const config = Object.fromEntries(chart.rows.map((row, i) => [String(row.x), { label: String(row.x), color: color(i) }])) satisfies ChartConfig
  const total = chart.rows.reduce((sum, row) => sum + (typeof row[key] === 'number' ? (row[key] as number) : 0), 0)
  return (
    <div className="grid items-center gap-4 sm:grid-cols-[minmax(0,16rem)_1fr]">
      <ChartContainer config={config} className="aspect-square w-full max-w-64">
        <PieChart accessibilityLayer>
          <ChartTooltip content={<ChartTooltipContent nameKey="x" hideLabel />} />
          <Pie data={chart.rows} dataKey={key} nameKey="x" innerRadius="55%" strokeWidth={2}>
            {chart.rows.map((row, i) => (
              <Cell key={String(row.x)} fill={color(i)} />
            ))}
          </Pie>
        </PieChart>
      </ChartContainer>
      <ul className="grid gap-1.5 text-sm">
        {chart.rows.map((row, i) => (
          <li key={String(row.x)} className="flex items-center gap-2">
            <span className="size-3 shrink-0 rounded-sm" style={{ backgroundColor: color(i) }} aria-hidden />
            <span className="min-w-0 flex-1 truncate">{row.x}</span>
            <span className="num text-muted-foreground">
              {formatNumber(row[key])}
              {total > 0 && typeof row[key] === 'number' && ` · ${Math.round(((row[key] as number) * 100) / total)}%`}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}

/** A value over two dimensions (e.g. weekday × hour), as the indicators' heatmap: darker is more. */
function Heatmap({ chart }: { chart: AnalyticsChart }) {
  const key = chart.series[0].key
  const columns = [...new Set(chart.rows.map((r) => String(r.x)))]
  const rows = [...new Set(chart.rows.map((r) => String(r.y)))]
  const value = new Map(chart.rows.map((r) => [`${r.y}|${r.x}`, typeof r[key] === 'number' ? (r[key] as number) : null]))
  const max = Math.max(1, ...[...value.values()].map((v) => v ?? 0))
  return (
    <div className="overflow-x-auto" aria-hidden>
      <table className="num border-separate border-spacing-0.5 text-xs">
        <thead>
          <tr>
            <th />
            {columns.map((c) => (
              <th key={c} className="px-1 font-normal text-muted-foreground">
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r}>
              <th className="pr-2 text-left font-normal whitespace-nowrap text-muted-foreground">{r}</th>
              {columns.map((c) => {
                const v = value.get(`${r}|${c}`) ?? null
                return (
                  <td
                    key={c}
                    title={`${r} · ${c}: ${formatNumber(v)}`}
                    className="size-7 min-w-7 rounded-sm text-center"
                    style={v ? { backgroundColor: `color-mix(in oklab, var(--chart-s0) ${Math.round(15 + (85 * v) / max)}%, transparent)` } : undefined}
                  />
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
