import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Input, Button, Modal, Form, Tag, Switch, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined, PushpinOutlined } from '@ant-design/icons'
import { getNotes, createNote, deleteNote, updateNote } from '../api/endpoints'

interface Note { id: string; title: string; content: string; pinned: boolean; tags: string[] }

export function NotesPage() {
  const [search, setSearch] = useState('')
  const [addOpen, setAddOpen] = useState(false)
  const [editNote, setEditNote] = useState<Note | null>(null)
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()
  const qc = useQueryClient()

  const { data: notes = [], isLoading } = useQuery({
    queryKey: ['notes', search],
    queryFn: () => getNotes({ search }),
  })

  const addMutation = useMutation({
    mutationFn: (values: { title: string; content: string; pinned?: boolean; tags?: string }) =>
      createNote({ title: values.title, content: values.content, pinned: values.pinned ?? false, tags: values.tags ? values.tags.split(',').map((t: string) => t.trim()) : [] }),
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

  const editMutation = useMutation({
    mutationFn: ({ id, values }: { id: string; values: { title: string; content: string; pinned?: boolean; tags?: string } }) =>
      updateNote(id, { title: values.title, content: values.content, pinned: values.pinned ?? false, tags: values.tags ? values.tags.split(',').map((t: string) => t.trim()) : [] }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['notes'] }); setEditNote(null); editForm.resetFields() },
  })

  const openEdit = (n: Note) => {
    setEditNote(n)
    editForm.setFieldsValue({ title: n.title, content: n.content, pinned: n.pinned, tags: n.tags.join(', ') })
  }

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
              <Card size="small" hoverable style={{ cursor: 'pointer' }}
                onClick={() => openEdit(n)}
                title={<span style={{ fontSize: 13 }}>{n.pinned && <PushpinOutlined style={{ color: '#faad14', marginRight: 6 }} />}{n.title}</span>}
                extra={
                  <div style={{ display: 'flex', gap: 4 }} onClick={e => e.stopPropagation()}>
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

      <Modal title="Edit Note" open={editNote !== null} onCancel={() => { setEditNote(null); editForm.resetFields() }} footer={null}>
        <Form form={editForm} layout="vertical" onFinish={values => editMutation.mutate({ id: editNote!.id, values })}>
          <Form.Item name="title" label="Title" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="content" label="Content"><Input.TextArea rows={6} /></Form.Item>
          <Form.Item name="tags" label="Tags (comma-separated)"><Input /></Form.Item>
          <Form.Item name="pinned" label="Pinned" valuePropName="checked"><Switch /></Form.Item>
          <Button type="primary" htmlType="submit" loading={editMutation.isPending} block>Save</Button>
        </Form>
      </Modal>

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
