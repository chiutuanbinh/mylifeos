import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Table, Tag, Button, Form, Input, Select, InputNumber, Modal, Progress, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getTransactions, createTransaction, deleteTransaction, getBudgets } from '../api/endpoints'
import type { Transaction } from '../api/types'

const CATEGORIES = ['Food', 'Income', 'Entertainment', 'Health', 'Tech', 'Auto', 'Utilities', 'Shopping']
const CAT_COLORS: Record<string, string> = { Food: 'green', Income: 'blue', Entertainment: 'purple', Health: 'volcano', Tech: 'cyan', Auto: 'orange', Utilities: 'gold', Shopping: 'magenta' }

export function FinancePage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: txs = [], isLoading } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })
  const { data: budgets = [] } = useQuery({ queryKey: ['budgets'], queryFn: getBudgets })

  const addMutation = useMutation({
    mutationFn: createTransaction,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['transactions'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteTransaction,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['transactions'] }),
  })

  const columns: ColumnsType<Transaction> = [
    { title: 'Date', dataIndex: 'date', width: 90 },
    { title: 'Description', dataIndex: 'description', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 130, render: c => <Tag color={CAT_COLORS[c]}>{c}</Tag> },
    { title: 'Amount', dataIndex: 'amount', align: 'right', width: 100,
      render: a => <span style={{ color: a > 0 ? '#52c41a' : '#ff4d4f', fontWeight: 600 }}>{a > 0 ? '+' : '-'}${Math.abs(a).toFixed(2)}</span> },
    { title: '', width: 40, render: (_, row) => <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} /> },
  ]

  const totalIncome = txs.filter(t => t.amount > 0).reduce((s, t) => s + t.amount, 0)
  const totalExpenses = txs.filter(t => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        {[
          { label: 'Income', val: `$${totalIncome.toFixed(2)}`, color: '#52c41a' },
          { label: 'Expenses', val: `$${totalExpenses.toFixed(2)}`, color: '#ff4d4f' },
          { label: 'Net', val: `$${(totalIncome - totalExpenses).toFixed(2)}`, color: '#1677ff' },
        ].map((s, i) => (
          <Col span={8} key={i}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>{s.label}</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: s.color }}>{s.val}</div>
            </Card>
          </Col>
        ))}
      </Row>

      {budgets.length > 0 && (
        <Card size="small" title="Budgets" style={{ marginBottom: 12 }}>
          <Row gutter={[12, 8]}>
            {budgets.map(b => {
              const spent = txs.filter(t => t.category === b.category && t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)
              const pct = Math.min(Math.round(spent / b.monthly_limit * 100), 100)
              return (
                <Col span={8} key={b.id}>
                  <div style={{ fontSize: 12, marginBottom: 2 }}>{b.category} <span style={{ color: '#999' }}>${spent.toFixed(0)} / ${b.monthly_limit.toFixed(0)}</span></div>
                  <Progress percent={pct} size="small" strokeColor={pct > 90 ? '#ff4d4f' : '#1677ff'} />
                </Col>
              )
            })}
          </Row>
        </Card>
      )}

      <Card size="small" title="Transactions" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={txs} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>

      <Modal title="Add Transaction" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="description" label="Description" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="category" label="Category" rules={[{ required: true }]}>
            <Select options={CATEGORIES.map(c => ({ value: c, label: c }))} />
          </Form.Item>
          <Form.Item name="amount" label="Amount (negative = expense)" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} step={0.01} /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
