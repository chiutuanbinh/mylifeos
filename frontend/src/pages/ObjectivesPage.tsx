import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Row, Col, Card, Progress, Button, Modal, Form, Input, Select,
  Tag, Spin, Tooltip, Space, Switch,
} from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined, FireOutlined } from '@ant-design/icons'
import type { Goal, KRLog } from '../api/types'
import {
  getGoals, createGoal, updateGoal, deleteGoal,
  addKeyResult, updateKeyResult, deleteKeyResult,
  getKRLogs, toggleKRLog, getKRLogRange,
} from '../api/endpoints'

const today = new Date().toISOString().split('T')[0]

const STATUS_COLORS: Record<string, string> = {
  active: 'blue', completed: 'green', archived: 'default',
}

function computeStreak(logs: KRLog[], krId: string): number {
  const doneSet = new Set(logs.filter(l => l.done && l.kr_id === krId).map(l => l.logged_date))
  let streak = 0
  const cursor = new Date(today)
  while (true) {
    const d = cursor.toISOString().split('T')[0]
    if (!doneSet.has(d)) break
    streak++
    cursor.setDate(cursor.getDate() - 1)
  }
  return streak
}

function getMonthDays(year: number, month: number): string[] {
  const days: string[] = []
  const count = new Date(year, month + 1, 0).getDate()
  for (let d = 1; d <= count; d++) {
    days.push(`${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`)
  }
  return days
}

function HeatmapMini({ krId, logs }: { krId: string; logs: KRLog[] }) {
  const now = new Date()
  const days = getMonthDays(now.getFullYear(), now.getMonth())
  const doneSet = new Set(logs.filter(l => l.done && l.kr_id === krId).map(l => l.logged_date))
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 2, marginTop: 6 }}>
      {Array.from({ length: new Date(now.getFullYear(), now.getMonth(), 1).getDay() }).map((_, i) => (
        <div key={`e${i}`} />
      ))}
      {days.map(date => (
        <Tooltip key={date} title={date}>
          <div style={{
            height: 10, borderRadius: 2,
            background: doneSet.has(date) ? '#52c41a' : '#f0f0f0',
            border: date === today ? '1px solid #1677ff' : 'none',
          }} />
        </Tooltip>
      ))}
    </div>
  )
}

export function ObjectivesPage() {
  const [addGoalOpen, setAddGoalOpen] = useState(false)
  const [editGoal, setEditGoal] = useState<Goal | null>(null)
  const [expandedGoal, setExpandedGoal] = useState<string | null>(null)
  const [newKr, setNewKr] = useState<Record<string, string>>({})
  const [newKrRecurring, setNewKrRecurring] = useState<Record<string, boolean>>({})
  const [addGoalForm] = Form.useForm()
  const [editGoalForm] = Form.useForm()
  const qc = useQueryClient()

  const { data: goals = [], isLoading } = useQuery({ queryKey: ['goals'], queryFn: getGoals })
  const { data: todayLogs = [] } = useQuery({
    queryKey: ['kr-logs', today],
    queryFn: () => getKRLogs(today),
  })

  const allRecurring = goals.flatMap(g =>
    (g.key_results ?? []).filter(kr => kr.recurring).map(kr => ({ ...kr, goalName: g.name, goalColor: g.color }))
  )
  const todayDoneSet = new Set(todayLogs.filter(l => l.done).map(l => l.kr_id))
  const totalToday = allRecurring.length
  const doneToday = allRecurring.filter(kr => todayDoneSet.has(kr.id)).length
  const todayPct = totalToday ? Math.round(doneToday / totalToday * 100) : 0

  const now = new Date()
  const monthFrom = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`
  const monthTo = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate()}`
  const allKrIds = goals.flatMap(g => (g.key_results ?? []).filter(kr => kr.recurring).map(kr => kr.id))

  const { data: monthLogs = [] } = useQuery({
    queryKey: ['kr-logs-month', monthFrom, monthTo, allKrIds.join(',')],
    queryFn: async () => {
      const results = await Promise.all(allKrIds.map(id => getKRLogRange(id, monthFrom, monthTo)))
      return results.flat()
    },
    enabled: allKrIds.length > 0,
  })

  const toggleMutation = useMutation({
    mutationFn: (krId: string) => toggleKRLog(krId, today),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['kr-logs', today] })
      qc.invalidateQueries({ queryKey: ['kr-logs-month', monthFrom, monthTo, allKrIds.join(',')] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })

  const createGoalMutation = useMutation({
    mutationFn: createGoal,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); qc.invalidateQueries({ queryKey: ['dashboard'] }); setAddGoalOpen(false); addGoalForm.resetFields() },
  })

  const updateGoalMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Goal> }) => updateGoal(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); qc.invalidateQueries({ queryKey: ['dashboard'] }); setEditGoal(null) },
  })

  const deleteGoalMutation = useMutation({
    mutationFn: deleteGoal,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); qc.invalidateQueries({ queryKey: ['dashboard'] }) },
  })

  const toggleKRMutation = useMutation({
    mutationFn: ({ goalId, krId, done }: { goalId: string; krId: string; done: boolean }) =>
      updateKeyResult(goalId, krId, { done }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); qc.invalidateQueries({ queryKey: ['dashboard'] }) },
  })

  const addKrMutation = useMutation({
    mutationFn: ({ goalId, description, recurring }: { goalId: string; description: string; recurring: boolean }) =>
      addKeyResult(goalId, description, recurring),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ['goals'] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
      setNewKr(prev => ({ ...prev, [vars.goalId]: '' }))
      setNewKrRecurring(prev => ({ ...prev, [vars.goalId]: false }))
    },
  })

  const deleteKrMutation = useMutation({
    mutationFn: ({ goalId, krId }: { goalId: string; krId: string }) => deleteKeyResult(goalId, krId),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); qc.invalidateQueries({ queryKey: ['dashboard'] }) },
  })

  const openEditGoal = (g: Goal) => {
    setEditGoal(g)
    editGoalForm.setFieldsValue({
      name: g.name, description: g.description, color: g.color,
      status: g.status, target_date: g.target_date ?? '',
    })
  }

  const gateGroups = goals
    .map(g => ({ goal: g, krs: (g.key_results ?? []).filter(kr => kr.recurring) }))
    .filter(grp => grp.krs.length > 0)

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      {totalToday > 0 && (
        <Card
          size="small"
          style={{ marginBottom: 16, borderLeft: '3px solid #1677ff' }}
          title={
            <Space>
              <span style={{ fontWeight: 600 }}>Today</span>
              <span style={{ color: '#999', fontSize: 12 }}>{doneToday}/{totalToday} done</span>
            </Space>
          }
        >
          <Progress percent={todayPct} size="small" strokeColor="#1677ff" style={{ marginBottom: 12 }} />
          {gateGroups.map(({ goal, krs }) => (
            <div key={goal.id} style={{ marginBottom: 12 }}>
              <div style={{ fontSize: 11, color: goal.color, fontWeight: 600, marginBottom: 6, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                {goal.name}
              </div>
              {krs.map(kr => {
                const done = todayDoneSet.has(kr.id)
                const streak = computeStreak([...todayLogs, ...monthLogs], kr.id)
                return (
                  <div key={kr.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 0', borderBottom: '1px solid #f5f5f5' }}>
                    <div
                      onClick={() => toggleMutation.mutate(kr.id)}
                      style={{
                        width: 20, height: 20, borderRadius: '50%', cursor: 'pointer', flexShrink: 0,
                        background: done ? '#52c41a' : '#f0f0f0',
                        border: done ? 'none' : '1.5px solid #d9d9d9',
                        display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: '#fff',
                      }}
                    >{done ? '✓' : ''}</div>
                    <span style={{ fontSize: 13, flex: 1, textDecoration: done ? 'line-through' : 'none', color: done ? '#bbb' : '#222' }}>
                      {kr.description}
                    </span>
                    {streak > 0 && (
                      <Tag color="orange" icon={<FireOutlined />} style={{ fontSize: 11, margin: 0 }}>{streak}d</Tag>
                    )}
                  </div>
                )
              })}
            </div>
          ))}
        </Card>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddGoalOpen(true)}>Add Goal</Button>
      </div>

      <Row gutter={[12, 12]}>
        {goals.map(g => {
          const oneTimeKRs = (g.key_results ?? []).filter(kr => !kr.recurring)
          const recurringKRs = (g.key_results ?? []).filter(kr => kr.recurring)
          const recurringDone = recurringKRs.filter(kr => todayDoneSet.has(kr.id)).length
          const expanded = expandedGoal === g.id
          return (
            <Col span={8} key={g.id}>
              <Card
                size="small"
                style={{ borderTop: `3px solid ${g.color}` }}
                title={
                  <Space size={6}>
                    <span style={{ fontSize: 13, fontWeight: 600 }}>{g.name}</span>
                    <Tag color={STATUS_COLORS[g.status]} style={{ fontSize: 11, margin: 0 }}>{g.status}</Tag>
                  </Space>
                }
                extra={
                  <Space size={2}>
                    <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEditGoal(g)} />
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteGoalMutation.mutate(g.id)} />
                  </Space>
                }
              >
                <Progress percent={g.progress} strokeColor={g.color} size="small" style={{ marginBottom: 8 }} />
                <div style={{ fontSize: 11, color: '#999', marginBottom: 6 }}>
                  {oneTimeKRs.filter(kr => kr.done).length}/{oneTimeKRs.length} key results · {g.progress}%
                </div>

                {oneTimeKRs.map(kr => (
                  <div key={kr.id} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                    <div
                      onClick={() => toggleKRMutation.mutate({ goalId: g.id, krId: kr.id, done: !kr.done })}
                      style={{
                        width: 16, height: 16, borderRadius: 3, cursor: 'pointer', flexShrink: 0,
                        background: kr.done ? g.color : '#f0f0f0',
                        border: kr.done ? 'none' : '1.5px solid #d9d9d9',
                        display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: '#fff',
                      }}
                    >{kr.done ? '✓' : ''}</div>
                    <span style={{ fontSize: 12, flex: 1, textDecoration: kr.done ? 'line-through' : 'none', color: kr.done ? '#bbb' : '#222' }}>
                      {kr.description}
                    </span>
                    <Button type="text" size="small" danger icon={<DeleteOutlined />}
                      style={{ padding: 0, height: 16, width: 16 }}
                      onClick={() => deleteKrMutation.mutate({ goalId: g.id, krId: kr.id })} />
                  </div>
                ))}

                {recurringKRs.length > 0 && (
                  <div
                    style={{ marginTop: 8, cursor: 'pointer', padding: '4px 0', borderTop: '1px solid #f5f5f5' }}
                    onClick={() => setExpandedGoal(expanded ? null : g.id)}
                  >
                    <span style={{ fontSize: 11, color: '#1677ff' }}>
                      Daily {recurringDone}/{recurringKRs.length} done today {expanded ? '▲' : '▼'}
                    </span>
                    {expanded && (
                      <div style={{ marginTop: 8 }}>
                        {recurringKRs.map(kr => (
                          <div key={kr.id} style={{ marginBottom: 8 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                              <span style={{ fontSize: 12, flex: 1 }}>{kr.description}</span>
                              <Button type="text" size="small" danger icon={<DeleteOutlined />}
                                style={{ padding: 0, height: 16, width: 16 }}
                                onClick={e => { e.stopPropagation(); deleteKrMutation.mutate({ goalId: g.id, krId: kr.id }) }} />
                            </div>
                            <HeatmapMini krId={kr.id} logs={monthLogs} />
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                <div style={{ display: 'flex', gap: 4, marginTop: 8, alignItems: 'center' }}>
                  <Switch
                    size="small"
                    checked={newKrRecurring[g.id] ?? false}
                    onChange={v => setNewKrRecurring(prev => ({ ...prev, [g.id]: v }))}
                    checkedChildren="🔁" unCheckedChildren="1×"
                  />
                  <Input
                    size="small"
                    placeholder={newKrRecurring[g.id] ? 'Daily habit…' : 'Add key result…'}
                    value={newKr[g.id] ?? ''}
                    onChange={e => setNewKr(prev => ({ ...prev, [g.id]: e.target.value }))}
                    onPressEnter={() => {
                      const desc = (newKr[g.id] ?? '').trim()
                      if (desc) addKrMutation.mutate({ goalId: g.id, description: desc, recurring: newKrRecurring[g.id] ?? false })
                    }}
                    style={{ fontSize: 12 }}
                  />
                  <Button size="small" icon={<PlusOutlined />} onClick={() => {
                    const desc = (newKr[g.id] ?? '').trim()
                    if (desc) addKrMutation.mutate({ goalId: g.id, description: desc, recurring: newKrRecurring[g.id] ?? false })
                  }} />
                </div>
              </Card>
            </Col>
          )
        })}
        {goals.length === 0 && (
          <Col span={24}><div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>No goals yet. Add your first!</div></Col>
        )}
      </Row>

      <Modal title="Add Goal" open={addGoalOpen} onCancel={() => setAddGoalOpen(false)} footer={null}>
        <Form form={addGoalForm} layout="vertical"
          onFinish={values => createGoalMutation.mutate({ ...values, key_results: [], status: 'active', progress: 0 })}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true, max: 100 }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="target_date" label="Target date"><Input type="date" /></Form.Item>
          <Form.Item name="color" label="Color" initialValue="#1677ff"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={createGoalMutation.isPending} block>Save</Button>
        </Form>
      </Modal>

      <Modal title="Edit Goal" open={!!editGoal} onCancel={() => setEditGoal(null)} footer={null}>
        <Form form={editGoalForm} layout="vertical"
          onFinish={values => editGoal && updateGoalMutation.mutate({ id: editGoal.id, data: values })}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true, max: 100 }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="target_date" label="Target date"><Input type="date" /></Form.Item>
          <Form.Item name="status" label="Status">
            <Select options={[
              { value: 'active', label: 'Active' },
              { value: 'completed', label: 'Completed' },
              { value: 'archived', label: 'Archived' },
            ]} />
          </Form.Item>
          <Form.Item name="color" label="Color"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={updateGoalMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
