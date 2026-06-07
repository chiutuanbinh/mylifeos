import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Table, Button, Modal, Form, Input, InputNumber, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getAssets, createAsset, deleteAsset } from '../api/endpoints'
import type { Asset } from '../api/types'

export function InventoryPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: assets = [], isLoading } = useQuery({ queryKey: ['assets'], queryFn: getAssets })

  const addMutation = useMutation({
    mutationFn: createAsset,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['assets'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteAsset,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['assets'] }),
  })

  const categories = [...new Set(assets.map(a => a.category))]
  const categoryTotals = categories.map(cat => ({
    category: cat,
    total: assets.filter(a => a.category === cat).reduce((s, a) => s + a.value, 0),
    count: assets.filter(a => a.category === cat).length,
  }))
  const grandTotal = assets.reduce((s, a) => s + a.value, 0)

  const columns: ColumnsType<Asset> = [
    { title: 'Name',     dataIndex: 'name',        ellipsis: true },
    { title: 'Category', dataIndex: 'category',    width: 120 },
    { title: 'Value',    dataIndex: 'value',        width: 120, align: 'right', render: v => `$${v.toLocaleString()}` },
    { title: 'Bought',   dataIndex: 'purchased_at', width: 110, render: v => v ?? '—' },
    { title: '',         width: 40, render: (_, row) => <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} /> },
  ]

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Total Assets</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#52c41a' }}>${grandTotal.toLocaleString()}</div>
          </Card>
        </Col>
        {categoryTotals.map(ct => (
          <Col span={6} key={ct.category}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>{ct.category} ({ct.count})</div>
              <div style={{ fontSize: 18, fontWeight: 600 }}>${ct.total.toLocaleString()}</div>
            </Card>
          </Col>
        ))}
      </Row>

      <Card size="small" title="Assets" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={assets} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>

      <Modal title="Add Asset" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate({ ...values, notes: values.notes || '' })}>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="category" label="Category" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="value" label="Value ($)" rules={[{ required: true }]}><InputNumber min={0} step={0.01} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="purchased_at" label="Purchase date"><Input type="date" /></Form.Item>
          <Form.Item name="notes" label="Notes"><Input.TextArea rows={2} /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
