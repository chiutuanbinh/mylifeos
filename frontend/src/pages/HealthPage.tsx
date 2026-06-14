import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, Row, Col, Button, Modal, Form, Input, Spin, Tooltip, Space, Tag } from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined, FireOutlined } from '@ant-design/icons'
import type { Habit, HabitLog } from '../api/types'
import {
  getHabits, createHabit, updateHabit, deleteHabit,
  getHabitLogs, toggleHabitLog, getHabitLogRange,
} from '../api/endpoints'

const today = new Date().toISOString().split('T')[0]

function getMonthDays(year: number, month: number): string[] {
  const days: string[] = []
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  for (let d = 1; d <= daysInMonth; d++) {
    days.push(`${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`)
  }
  return days
}

function computeStreak(logs: HabitLog[], habitId: string): number {
  const doneSet = new Set(logs.filter(l => l.done && l.habit_id === habitId).map(l => l.logged_date))
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

function HeatmapCard({ habit, logs }: { habit: Habit; logs: HabitLog[] }) {
  const now = new Date()
  const year = now.getFullYear()
  const month = now.getMonth()
  const days = getMonthDays(year, month)
  const doneSet = new Set(logs.filter(l => l.done && l.habit_id === habit.id).map(l => l.logged_date))
  const streak = computeStreak(logs, habit.id)
  const monthName = now.toLocaleString('default', { month: 'long', year: 'numeric' })

  return (
    <div>
      <div style={{ fontSize: 11, color: '#999', marginBottom: 6 }}>{monthName}</div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 3, marginBottom: 8 }}>
        {['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'].map(d => (
          <div key={d} style={{ fontSize: 10, color: '#bbb', textAlign: 'center' }}>{d}</div>
        ))}
        {Array.from({ length: new Date(year, month, 1).getDay() }).map((_, i) => (
          <div key={`e${i}`} />
        ))}
        {days.map(date => {
          const done = doneSet.has(date)
          const isToday = date === today
          return (
            <Tooltip key={date} title={date}>
              <div style={{
                height: 14, borderRadius: 3,
                background: done ? '#52c41a' : '#f0f0f0',
                border: isToday ? '1.5px solid #1677ff' : 'none',
              }} />
            </Tooltip>
          )
        })}
      </div>
      {streak > 0 && (
        <Tag color="orange" icon={<FireOutlined />} style={{ fontSize: 11 }}>{streak} day streak</Tag>
      )}
    </div>
  )
}

export function HealthPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [editHabit, setEditHabit] = useState<Habit | null>(null)
  const [addForm] = Form.useForm()
  const [editForm] = Form.useForm()
  const qc = useQueryClient()

  const now = new Date()
  const year = now.getFullYear()
  const month = now.getMonth()
  const monthFrom = `${year}-${String(month + 1).padStart(2, '0')}-01`
  const monthTo = `${year}-${String(month + 1).padStart(2, '0')}-${new Date(year, month + 1, 0).getDate()}`

  const { data: habits = [], isLoading } = useQuery({ queryKey: ['habits'], queryFn: getHabits })
  const { data: todayLogs = [] } = useQuery({ queryKey: ['habit-logs', today], queryFn: () => getHabitLogs(today) })

  const { data: monthLogs = [] } = useQuery({
    queryKey: ['habit-logs-month', monthFrom, monthTo, habits.map(h => h.id).join(',')],
    queryFn: async () => {
      const results = await Promise.all(habits.map(h => getHabitLogRange(h.id, monthFrom, monthTo)))
      return results.flat()
    },
    enabled: habits.length > 0,
  })

  const addMutation = useMutation({
    mutationFn: createHabit,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['habits'] }); setAddOpen(false); addForm.resetFields() },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name: string; icon: string } }) => updateHabit(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['habits'] }); setEditHabit(null) },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteHabit,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['habits'] }),
  })

  const toggleMutation = useMutation({
    mutationFn: ({ habitId }: { habitId: string }) => toggleHabitLog(habitId, today),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['habit-logs', today] })
      qc.invalidateQueries({ queryKey: ['habit-logs-month', monthFrom, monthTo, habits.map(h => h.id).join(',')] })
    },
  })

  const doneSet = new Set(todayLogs.filter(l => l.done).map(l => l.habit_id))
  const donePct = habits.length ? Math.round(doneSet.size / habits.length * 100) : 0

  const openEdit = (h: Habit) => {
    setEditHabit(h)
    editForm.setFieldsValue({ name: h.name, icon: h.icon })
  }

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      <Row gutter={[12, 12]}>
        <Col span={10}>
          <Card
            size="small"
            title={`Today's Habits — ${donePct}% done`}
            extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}
          >
            {habits.map(h => {
              const done = doneSet.has(h.id)
              return (
                <div key={h.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
                  <div
                    onClick={() => toggleMutation.mutate({ habitId: h.id })}
                    style={{
                      width: 22, height: 22, borderRadius: '50%', cursor: 'pointer', flexShrink: 0,
                      background: done ? '#52c41a' : '#f0f0f0',
                      border: done ? 'none' : '1.5px solid #d9d9d9',
                      display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, color: '#fff',
                    }}
                  >{done ? '✓' : ''}</div>
                  <span style={{ fontSize: 13, flex: 1, textDecoration: done ? 'line-through' : 'none', color: done ? '#bbb' : '#222' }}>
                    {h.icon} {h.name}
                  </span>
                  <Space size={2}>
                    <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(h)} />
                    <Tooltip title="Delete">
                      <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(h.id)} />
                    </Tooltip>
                  </Space>
                </div>
              )
            })}
            {habits.length === 0 && <div style={{ color: '#bbb', textAlign: 'center', padding: 20 }}>No habits yet.</div>}
          </Card>
        </Col>

        <Col span={14}>
          <Row gutter={[12, 12]}>
            {habits.map(h => (
              <Col span={12} key={h.id}>
                <Card size="small" title={<span style={{ fontSize: 12 }}>{h.icon} {h.name}</span>}>
                  <HeatmapCard habit={h} logs={monthLogs} />
                </Card>
              </Col>
            ))}
            {habits.length === 0 && (
              <Col span={24}><div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>Add habits to see heatmaps.</div></Col>
            )}
          </Row>
        </Col>
      </Row>

      <Modal title="Add Habit" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={addForm} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="name" label="Habit name" rules={[{ required: true, max: 80 }]}><Input /></Form.Item>
          <Form.Item name="icon" label="Icon (emoji)" initialValue="✓"><Input /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>

      <Modal title="Edit Habit" open={!!editHabit} onCancel={() => setEditHabit(null)} footer={null}>
        <Form
          form={editForm}
          layout="vertical"
          onFinish={values => editHabit && updateMutation.mutate({ id: editHabit.id, data: values })}
        >
          <Form.Item name="name" label="Habit name" rules={[{ required: true, max: 80 }]}><Input /></Form.Item>
          <Form.Item name="icon" label="Icon (emoji)"><Input /></Form.Item>
          <Button type="primary" htmlType="submit" loading={updateMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
