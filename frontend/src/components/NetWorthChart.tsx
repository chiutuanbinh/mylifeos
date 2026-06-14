import { useMemo, useState } from 'react'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, Legend, ResponsiveContainer,
} from 'recharts'
import { Button, Space, Checkbox } from 'antd'
import type { NetWorthSnapshot, BenchmarkData } from '../api/types'

interface Props {
  snapshots: NetWorthSnapshot[]
  benchmarks: BenchmarkData[]
}

type Range = '1M' | '3M' | '6M' | '1Y' | 'ALL'

const BENCHMARK_META: Record<string, { label: string; color: string }> = {
  vn_index: { label: 'VN-Index', color: '#f5a623' },
  sjc_gold: { label: 'SJC Gold', color: '#e8c14a' },
  gso_cpi:  { label: 'CPI',      color: '#7ed321' },
}

function getCutoff(range: Range): string {
  const now = new Date()
  const cutoffs: Record<Range, Date> = {
    '1M':  new Date(now.getFullYear(), now.getMonth() - 1, now.getDate()),
    '3M':  new Date(now.getFullYear(), now.getMonth() - 3, now.getDate()),
    '6M':  new Date(now.getFullYear(), now.getMonth() - 6, now.getDate()),
    '1Y':  new Date(now.getFullYear() - 1, now.getMonth(), now.getDate()),
    'ALL': new Date(0),
  }
  return cutoffs[range].toISOString().split('T')[0]
}

export function NetWorthChart({ snapshots, benchmarks }: Props) {
  const [range, setRange] = useState<Range>('1Y')
  const [activeOverlays, setActiveOverlays] = useState<string[]>(['vn_index', 'sjc_gold'])

  const cutoff = getCutoff(range)
  const filteredSnaps = snapshots.filter(s => s.snapshot_date >= cutoff)

  const chartData = useMemo(() => {
    if (filteredSnaps.length === 0) return []

    const baseNetWorth = filteredSnaps[0].net_worth
    const baseBySource: Record<string, number> = {}
    const byDate: Record<string, Record<string, number>> = {}

    filteredSnaps.forEach(s => {
      byDate[s.snapshot_date] = { net_worth_pct: baseNetWorth !== 0 ? ((s.net_worth - baseNetWorth) / baseNetWorth) * 100 : 0 }
    })

    benchmarks
      .filter(b => activeOverlays.includes(b.source) && b.date >= cutoff)
      .forEach(b => {
        if (baseBySource[b.source] === undefined) {
          baseBySource[b.source] = b.value
        }
        if (!byDate[b.date]) byDate[b.date] = {}
        const base = baseBySource[b.source]
        byDate[b.date][b.source + '_pct'] = base > 0 ? ((b.value - base) / base) * 100 : 0
      })

    return Object.entries(byDate)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([date, vals]) => ({ date, ...vals }))
  }, [filteredSnaps, benchmarks, activeOverlays, cutoff])

  const toggleOverlay = (source: string) => {
    setActiveOverlays(prev =>
      prev.includes(source) ? prev.filter(s => s !== source) : [...prev, source]
    )
  }

  const formatPct = (v: number) => `${v >= 0 ? '+' : ''}${v.toFixed(1)}%`

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <Space>
          {(['1M', '3M', '6M', '1Y', 'ALL'] as Range[]).map(r => (
            <Button
              key={r} size="small"
              type={range === r ? 'primary' : 'default'}
              onClick={() => setRange(r)}
            >{r}</Button>
          ))}
        </Space>
        <Space>
          {Object.entries(BENCHMARK_META).map(([source, meta]) => (
            <Checkbox
              key={source}
              checked={activeOverlays.includes(source)}
              onChange={() => toggleOverlay(source)}
              style={{ fontSize: 12 }}
            >
              <span style={{ color: meta.color }}>{meta.label}</span>
            </Checkbox>
          ))}
        </Space>
      </div>

      {chartData.length === 0 ? (
        <div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>
          No data yet. Snapshots accumulate daily.
        </div>
      ) : (
        <ResponsiveContainer width="100%" height={280}>
          <LineChart data={chartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis dataKey="date" tick={{ fontSize: 11 }} tickFormatter={d => d.slice(5)} />
            <YAxis tick={{ fontSize: 11 }} tickFormatter={v => `${(v as number).toFixed(0)}%`} />
            <Tooltip
              formatter={(v: unknown) => [formatPct(v as number)]}
              labelFormatter={l => `Date: ${l}`}
            />
            <Legend wrapperStyle={{ fontSize: 12 }} />
            <Line
              type="monotone" dataKey="net_worth_pct" name="Net Worth"
              stroke="#1677ff" strokeWidth={2} dot={false} connectNulls
            />
            {activeOverlays.map(source => (
              <Line
                key={source}
                type="monotone"
                dataKey={source + '_pct'}
                name={BENCHMARK_META[source]?.label ?? source}
                stroke={BENCHMARK_META[source]?.color ?? '#aaa'}
                strokeWidth={1.5} dot={false} connectNulls strokeDasharray="4 2"
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      )}
    </div>
  )
}
