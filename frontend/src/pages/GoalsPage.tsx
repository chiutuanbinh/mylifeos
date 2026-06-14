import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Progress, Button, Modal, Form, Input, Select, Tag, Spin, Tooltip, Space } from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons'
import type { Goal } from '../api/types'
import {
  getGoals, createGoal, updateGoal, deleteGoal,
  addKeyResult, updateKeyResult, deleteKeyResult,
} from '../api/endpoints'

const STATUS_COLORS: Record<string, string> = {
  active: 'blue',
  completed: 'green',
  archived: 'default',
}

export function GoalsPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [editGoal, setEditGoal] = useState<Goal | null>(null)
  const [newKr, setNewKr] = useState<Record<string, string>>({})
  const [addForm] = Form.useForm()
  const [editForm] = Form.useForm()
  const qc = useQueryClient()

  const { data: goals = [], isLoading } = useQuery({ queryKey: ['goals'], queryFn: getGoals })

  const createMutation = useMutation({
    mutationFn: createGoal,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); setAddOpen(false); addForm.resetFields() },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Goal> }) => updateGoal(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); setEditGoal(null) },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteGoal,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })

  const toggleKrMutation = useMutation({
    mutationFn: ({ goalId, krId, done }: { goalId: string; krId: string; done: boolean }) =>
      updateKeyResult(goalId, krId, { done }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })

  const addKrMutation = useMutation({
    mutationFn: ({ goalId, description }: { goalId: string; description: string }) =>
      addKeyResult(goalId, description),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ['goals'] })
      setNewKr(prev => ({ ...prev, [vars.goalId]: '' }))
    },
  })

  const deleteKrMutation = useMutation({
    mutationFn: ({ goalId, krId }: { goalId: string; krId: string }) => deleteKeyResult(goalId, krId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })

  const openEdit = (g: Goal) => {
    setEditGoal(g)
    editForm.setFieldsValue({ name: g.name, description: g.description, color: g.color, status: g.status, target_date: g.target_date ?? '' })
  }

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add Goal</Button>
      </div>

      <Row gutter={[12, 12]}>
        {goals.map(g => (
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
                  <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(g)} />
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(g.id)} />
                </Space>
              }
            >
              <Progress percent={g.progress} strokeColor={g.color} size="small" style={{ marginBottom: 8 }} />
              <div style={{ fontSize: 11, color: '#999', marginBottom: g.description ? 6 : 0 }}>
                {g.key_results.filter(kr => kr.done).length}/{g.key_results.length} key results · {g.progress}%
              </div>
              {g.description && <div style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>{g.description}</div>}

              {(g.key_results ?? []).map(kr => (
                <div key={kr.id} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                  <div
                    onClick={() => toggleKrMutation.mutate({ goalId: g.id, krId: kr.id, done: !kr.done })}
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
                  <Tooltip title="Remove">
                    <Button
                      type="text" size="small" danger icon={<DeleteOutlined />}
                      style={{ padding: 0, height: 16, width: 16, fontSize: 11 }}
                      onClick={() => deleteKrMutation.mutate({ goalId: g.id, krId: kr.id })}
                    />
                  </Tooltip>
                </div>
              ))}

              <div style={{ display: 'flex', gap: 6, marginTop: 8 }}>
                <Input
                  size="small"
                  placeholder="Add key result…"
                  value={newKr[g.id] ?? ''}
                  onChange={e => setNewKr(prev => ({ ...prev, [g.id]: e.target.value }))}
                  onPressEnter={() => {
                    const desc = (newKr[g.id] ?? '').trim()
                    if (desc) addKrMutation.mutate({ goalId: g.id, description: desc })
                  }}
                  style={{ fontSize: 12 }}
                />
                <Button
                  size="small" icon={<PlusOutlined />}
                  onClick={() => {
                    const desc = (newKr[g.id] ?? '').trim()
                    if (desc) addKrMutation.mutate({ goalId: g.id, description: desc })
                  }}
                />
              </div>
            </Card>
          </Col>
        ))}
        {goals.length === 0 && (
          <Col span={24}><div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>No goals yet. Add your first!</div></Col>
        )}
      </Row>

      {/* Add Goal Modal */}
      <Modal title="Add Goal" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={addForm} layout="vertical" onFinish={values => createMutation.mutate({ ...values, key_results: [], status: 'active', progress: 0 })}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true, max: 100 }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="target_date" label="Target date"><Input type="date" /></Form.Item>
          <Form.Item name="color" label="Color" initialValue="#1677ff"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={createMutation.isPending} block>Save</Button>
        </Form>
      </Modal>

      {/* Edit Goal Modal */}
      <Modal
        title="Edit Goal"
        open={!!editGoal}
        onCancel={() => setEditGoal(null)}
        footer={null}
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={values => editGoal && updateMutation.mutate({ id: editGoal.id, data: values })}
        >
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
          <Button type="primary" htmlType="submit" loading={updateMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
