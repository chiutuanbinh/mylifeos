import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Row, Col, Card, Progress, Spin } from 'antd'
import { getDashboardSummary } from '../api/endpoints'
import { Sparkline } from '../components/Sparkline'
import { LiveNetWorthCard } from './LiveNetWorthCard'

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
      val: `₫${Math.round(netWorth).toLocaleString('vi-VN')}`,
      sub: netWorthChange !== null
        ? `${Number(netWorthChange) >= 0 ? '↑' : '↓'} ${Math.abs(Number(netWorthChange))}% vs last snapshot`
        : 'No history yet',
      subC: netWorthChange !== null && Number(netWorthChange) >= 0 ? '#52c41a' : '#ff4d4f',
      spark: trend,
      sparkC: '#52c41a',
      nav: '/wealth',
    },
    { label: "Today's Habits", val: `${data.habits_done_today} / ${data.habits_total}`, sub: `${habitPct}% complete`, subC: '#1677ff', pct: habitPct, nav: '/objectives' },
    { label: 'Goals (avg)',    val: `${data.goals_avg_progress}%`, sub: 'active OKRs', subC: '#722ed1', pct: data.goals_avg_progress, pctC: '#722ed1', nav: '/objectives' },
    { label: 'Monthly Budget', val: `₫${Math.round(data.budget_total).toLocaleString('vi-VN')}`, sub: `₫${Math.round(data.budget_spent).toLocaleString('vi-VN')} spent · ${budgetPct}%`, subC: '#fa8c16', pct: budgetPct, pctC: '#fa8c16', nav: '/finance' },
  ]

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col xs={24} sm={8}>
          <LiveNetWorthCard />
        </Col>
      </Row>
      <Row gutter={[12, 12]}>
        {stats.map((s, i) => (
          <Col xs={12} sm={6} key={i}>
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
    </div>
  )
}
