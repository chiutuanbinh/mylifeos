import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Row, Col, Card, Progress, Table, Tag, Spin } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getDashboardSummary } from '../api/endpoints'
import { Sparkline } from '../components/Sparkline'
import type { Transaction } from '../api/types'

const CAT_COLORS: Record<string, string> = {
  Food: 'green', Income: 'blue', Entertainment: 'purple', Health: 'volcano',
  Tech: 'cyan', Auto: 'orange', Utilities: 'gold', Shopping: 'magenta',
}

const txColumns: ColumnsType<Transaction> = [
  { title: 'Date',        dataIndex: 'date',        width: 72,  render: v => <span style={{ color: '#bbb', fontSize: 12 }}>{v}</span> },
  { title: 'Description', dataIndex: 'description', ellipsis: true, render: v => <span style={{ fontSize: 12 }}>{v}</span> },
  { title: 'Category',    dataIndex: 'category',    width: 120, render: c => <Tag color={CAT_COLORS[c]} style={{ fontSize: 11, margin: 0 }}>{c}</Tag> },
  { title: 'Amount',      dataIndex: 'amount',      align: 'right', width: 92,
    render: a => <span style={{ color: a > 0 ? '#52c41a' : '#ff4d4f', fontFamily: 'monospace', fontSize: 12, fontWeight: 600 }}>{a > 0 ? '+' : '-'}${Math.abs(a).toFixed(2)}</span> },
]

export function DashboardPage() {
  const navigate = useNavigate()
  const { data, isLoading } = useQuery({ queryKey: ['dashboard'], queryFn: getDashboardSummary })

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />
  if (!data) return null

  const habitPct = data.habits_total ? Math.round(data.habits_done_today / data.habits_total * 100) : 0
  const budgetPct = data.budget_total ? Math.round(data.budget_spent / data.budget_total * 100) : 0

  const trend = data.net_worth_trend
  const netWorth = data.net_worth
  const prevNetWorth = trend.length >= 2 ? trend[trend.length - 2] : null
  const netWorthChange = prevNetWorth && prevNetWorth !== 0
    ? ((netWorth - prevNetWorth) / Math.abs(prevNetWorth) * 100).toFixed(1)
    : null

  const stats = [
    {
      label: 'Net Worth',
      val: `$${netWorth.toLocaleString(undefined, { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`,
      sub: netWorthChange !== null
        ? `${Number(netWorthChange) >= 0 ? '↑' : '↓'} ${Math.abs(Number(netWorthChange))}% vs last snapshot`
        : 'No history yet',
      subC: netWorthChange !== null && Number(netWorthChange) >= 0 ? '#52c41a' : '#ff4d4f',
      spark: trend,
      sparkC: '#52c41a',
      nav: '/wealth',
    },
    { label: "Today's Habits", val: `${data.habits_done_today} / ${data.habits_total}`, sub: `${habitPct}% complete`, subC: '#1677ff', pct: habitPct, nav: '/health' },
    { label: 'Goals (avg)',    val: `${data.goals_avg_progress}%`, sub: 'active OKRs', subC: '#722ed1', pct: data.goals_avg_progress, pctC: '#722ed1', nav: '/goals' },
    { label: 'Monthly Budget', val: `$${data.budget_total.toLocaleString()}`, sub: `$${data.budget_spent.toLocaleString()} spent · ${budgetPct}%`, subC: '#fa8c16', pct: budgetPct, pctC: '#fa8c16', nav: '/wealth' },
  ]

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        {stats.map((s, i) => (
          <Col span={6} key={i}>
            <Card size="small" hoverable style={{ cursor: 'pointer' }} onClick={() => navigate(s.nav)}>
              <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>{s.label}</div>
              <div style={{ fontSize: 22, fontWeight: 700, marginBottom: 4 }}>{s.val}</div>
              {s.spark && <Sparkline data={s.spark} color={s.sparkC} width={100} height={28} />}
              {s.pct !== undefined && <Progress percent={s.pct} size="small" showInfo={false} strokeColor={s.pctC ?? '#1677ff'} style={{ margin: '4px 0 2px' }} />}
              <div style={{ fontSize: 12, color: s.subC }}>{s.sub}</div>
            </Card>
          </Col>
        ))}
      </Row>
      <Row gutter={[12, 12]}>
        <Col span={24}>
          <Card size="small" title={<span style={{ fontSize: 13 }}>Recent Transactions</span>} extra={<a onClick={() => navigate('/wealth')} style={{ fontSize: 12 }}>View all →</a>}>
            <Table dataSource={data.recent_transactions} columns={txColumns} size="small" pagination={false} rowKey="id" />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
