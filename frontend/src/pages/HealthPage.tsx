import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, Row, Col, Button, Modal, Form, Input, Spin, Tooltip } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { getHabits, createHabit, deleteHabit, getHabitLogs, toggleHabitLog } from '../api/endpoints'

const today = new Date().toISOString().split('T')[0]

export function HealthPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: habits = [], isLoading } = useQuery({ queryKey: ['habits'], queryFn: getHabits })
  const { data: logs = [] } = useQuery({ queryKey: ['habit-logs', today], queryFn: () => getHabitLogs(today) })

  const addMutation = useMutation({
    mutationFn: createHabit,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['habits'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteHabit,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['habits'] }),
  })

  const toggleMutation = useMutation({
    mutationFn: ({ habitId }: { habitId: string }) => toggleHabitLog(habitId, today),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['habit-logs', today] }),
  })

  const doneSet = new Set(logs.filter(l => l.done).map(l => l.habit_id))
  const donePct = habits.length ? Math.round(doneSet.size / habits.length * 100) : 0

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      <Row gutter={[12, 12]}>
        <Col span={12}>
          <Card size="small" title={`Today's Habits — ${donePct}% done`}
            extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}
          >
            {habits.map(h => {
              const done = doneSet.has(h.id)
              return (
                <div key={h.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
                  <div
                    onClick={() => toggleMutation.mutate({ habitId: h.id })}
                    style={{ width: 22, height: 22, borderRadius: '50%', cursor: 'pointer', flexShrink: 0, background: done ? '#52c41a' : '#f0f0f0', border: done ? 'none' : '1.5px solid #d9d9d9', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, color: '#fff' }}
                  >{done ? '✓' : ''}</div>
                  <span style={{ fontSize: 13, flex: 1, textDecoration: done ? 'line-through' : 'none', color: done ? '#bbb' : '#222' }}>{h.icon} {h.name}</span>
                  <Tooltip title="Delete habit">
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(h.id)} />
                  </Tooltip>
                </div>
              )
            })}
            {habits.length === 0 && <div style={{ color: '#bbb', textAlign: 'center', padding: 20 }}>No habits yet. Add your first!</div>}
          </Card>
        </Col>
      </Row>

      <Modal title="Add Habit" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="name" label="Habit name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="icon" label="Icon (emoji)" initialValue="✓"><Input /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
