import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Progress, Button, Modal, Form, Input, InputNumber, Checkbox, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { getGoals, createGoal, deleteGoal, updateKeyResult } from '../api/endpoints'

export function GoalsPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: goals = [], isLoading } = useQuery({ queryKey: ['goals'], queryFn: getGoals })

  const addMutation = useMutation({
    mutationFn: createGoal,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); setAddOpen(false); form.resetFields() },
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

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add Goal</Button>
      </div>
      <Row gutter={[12, 12]}>
        {goals.map(g => (
          <Col span={8} key={g.id}>
            <Card size="small"
              title={<span style={{ fontSize: 13, fontWeight: 600 }}>{g.name}</span>}
              extra={<Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(g.id)} />}
              style={{ borderTop: `3px solid ${g.color}` }}
            >
              <Progress percent={g.progress} strokeColor={g.color} size="small" style={{ marginBottom: 10 }} />
              {g.description && <div style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>{g.description}</div>}
              {g.key_results.map(kr => (
                <div key={kr.id} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                  <Checkbox
                    checked={kr.done}
                    onChange={e => toggleKrMutation.mutate({ goalId: g.id, krId: kr.id, done: e.target.checked })}
                  />
                  <span style={{ fontSize: 12, textDecoration: kr.done ? 'line-through' : 'none', color: kr.done ? '#bbb' : '#222' }}>{kr.description}</span>
                </div>
              ))}
            </Card>
          </Col>
        ))}
        {goals.length === 0 && <Col span={24}><div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>No goals yet. Add your first!</div></Col>}
      </Row>

      <Modal title="Add Goal" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate({ ...values, key_results: [] })}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="progress" label="Initial progress %" initialValue={0}><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="color" label="Color" initialValue="#1677ff"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
