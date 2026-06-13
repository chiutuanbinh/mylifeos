import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Input, Button, Modal, Form, Tag, Switch, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined, PushpinOutlined } from '@ant-design/icons'
import { getNotes, createNote, deleteNote, updateNote } from '../api/endpoints'

export function NotesPage() {
  const [search, setSearch] = useState('')
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: notes = [], isLoading } = useQuery({
    queryKey: ['notes', search],
    queryFn: () => getNotes({ search }),
  })

  const addMutation = useMutation({
    mutationFn: (values: Record<string, string>) => createNote({ ...values, tags: values.tags ? values.tags.split(',').map((t: string) => t.trim()) : [] }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['notes'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteNote,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notes'] }),
  })

  const pinMutation = useMutation({
    mutationFn: ({ id, pinned }: { id: string; pinned: boolean }) => updateNote(id, { pinned }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notes'] }),
  })

  return (
    <div>
      <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
        <Input.Search placeholder="Search notes..." value={search} onChange={e => setSearch(e.target.value)} style={{ maxWidth: 320 }} allowClear />
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add Note</Button>
      </div>

      {isLoading ? <Spin size="large" style={{ display: 'block', margin: '80px auto' }} /> : (
        <Row gutter={[12, 12]}>
          {notes.map(n => (
            <Col span={8} key={n.id}>
              <Card size="small"
                title={<span style={{ fontSize: 13 }}>{n.pinned && <PushpinOutlined style={{ color: '#faad14', marginRight: 6 }} />}{n.title}</span>}
                extra={
                  <div style={{ display: 'flex', gap: 4 }}>
                    <Button type="text" size="small" icon={<PushpinOutlined />} onClick={() => pinMutation.mutate({ id: n.id, pinned: !n.pinned })} />
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(n.id)} />
                  </div>
                }
              >
                <div style={{ fontSize: 12, color: '#555', marginBottom: 8, maxHeight: 60, overflow: 'hidden' }}>{n.content}</div>
                <div>{n.tags.map(t => <Tag key={t} style={{ fontSize: 11 }}>{t}</Tag>)}</div>
              </Card>
            </Col>
          ))}
          {notes.length === 0 && <Col span={24}><div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>No notes yet.</div></Col>}
        </Row>
      )}

      <Modal title="Add Note" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="title" label="Title" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="content" label="Content"><Input.TextArea rows={4} /></Form.Item>
          <Form.Item name="tags" label="Tags (comma-separated)"><Input /></Form.Item>
          <Form.Item name="pinned" label="Pinned" valuePropName="checked" initialValue={false}><Switch /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
