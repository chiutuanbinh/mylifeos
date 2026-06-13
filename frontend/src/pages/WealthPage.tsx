import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Tabs, Row, Col, Card, Table, Tag, Button, Form, Input, Select,
  InputNumber, Modal, Progress, Spin, Tooltip, Drawer,
} from 'antd'
import type { FormInstance } from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import {
  getTransactions, createTransaction, deleteTransaction,
  getBudgets, upsertBudget,
  getAssets, createAsset, updateAsset, deleteAsset,
} from '../api/endpoints'
import type { Transaction, Asset } from '../api/types'

const CATEGORIES = ['Food', 'Income', 'Entertainment', 'Health', 'Tech', 'Auto', 'Utilities', 'Shopping']
const CAT_COLORS: Record<string, string> = {
  Food: 'green', Income: 'blue', Entertainment: 'purple', Health: 'volcano',
  Tech: 'cyan', Auto: 'orange', Utilities: 'gold', Shopping: 'magenta',
}

function TransactionsTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: txs = [], isLoading } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })

  const addMutation = useMutation({
    mutationFn: createTransaction,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['transactions'] }); setAddOpen(false); form.resetFields() },
  })
  const deleteMutation = useMutation({
    mutationFn: deleteTransaction,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['transactions'] }),
  })

  const totalIncome = txs.filter(t => t.amount > 0).reduce((s, t) => s + t.amount, 0)
  const totalExpenses = txs.filter(t => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)

  const columns: ColumnsType<Transaction> = [
    { title: 'Date', dataIndex: 'date', width: 90 },
    { title: 'Description', dataIndex: 'description', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 130, render: c => <Tag color={CAT_COLORS[c]}>{c}</Tag> },
    { title: 'Amount', dataIndex: 'amount', align: 'right', width: 100,
      render: a => <span style={{ color: a > 0 ? '#52c41a' : '#ff4d4f', fontWeight: 600 }}>{a > 0 ? '+' : '-'}${Math.abs(a).toFixed(2)}</span> },
    { title: '', width: 40, render: (_, row) => <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} /> },
  ]

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        {[
          { label: 'Income', val: `$${totalIncome.toFixed(2)}`, color: '#52c41a' },
          { label: 'Expenses', val: `$${totalExpenses.toFixed(2)}`, color: '#ff4d4f' },
          { label: 'Net Cash', val: `$${(totalIncome - totalExpenses).toFixed(2)}`, color: '#1677ff' },
        ].map((s, i) => (
          <Col span={8} key={i}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>{s.label}</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: s.color }}>{s.val}</div>
            </Card>
          </Col>
        ))}
      </Row>
      <Card size="small" title="Transactions" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={txs} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>
      <Modal title="Add Transaction" open={addOpen} onCancel={() => { setAddOpen(false); form.resetFields() }} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="description" label="Description" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="category" label="Category" rules={[{ required: true }]}>
            <Select options={CATEGORIES.map(c => ({ value: c, label: c }))} />
          </Form.Item>
          <Form.Item name="amount" label="Amount (negative = expense)" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} step={0.01} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </>
  )
}

function BudgetsTab() {
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: txs = [] } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })
  const { data: budgets = [] } = useQuery({ queryKey: ['budgets'], queryFn: getBudgets })

  const upsertMutation = useMutation({
    mutationFn: ({ category, monthly_limit }: { category: string; monthly_limit: number }) =>
      upsertBudget(category, monthly_limit),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['budgets'] }); form.resetFields() },
  })

  return (
    <>
      {budgets.length > 0 && (
        <Card size="small" title="Budget Progress" style={{ marginBottom: 12 }}>
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
      <Card size="small" title="Set Budget Limit">
        <Form form={form} layout="inline" onFinish={values => upsertMutation.mutate(values)}>
          <Form.Item name="category" rules={[{ required: true }]}>
            <Select placeholder="Category" style={{ width: 160 }} options={CATEGORIES.map(c => ({ value: c, label: c }))} />
          </Form.Item>
          <Form.Item name="monthly_limit" rules={[{ required: true }]}>
            <InputNumber placeholder="Monthly limit $" min={0} step={1} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={upsertMutation.isPending}>Save</Button>
        </Form>
      </Card>
    </>
  )
}

function assetFormFields(form: FormInstance<AssetFormValues>, onFinish: (v: AssetFormValues) => void, loading: boolean) {
  return (
    <Form form={form} layout="vertical" onFinish={onFinish}>
      <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}><Input /></Form.Item>
      <Form.Item name="category" label="Category" rules={[{ required: true, message: 'Category is required' }]}><Input /></Form.Item>
      <Form.Item name="purchase_value" label="Purchase Value ($)" rules={[{ required: true, message: 'Purchase value is required' }, { type: 'number', min: 0, message: 'Must be >= 0' }]}>
        <InputNumber min={0} step={0.01} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="depreciation_rate_pct" label="Depreciation Rate (% per year)" initialValue={0} rules={[{ type: 'number', min: 0, max: 100, message: 'Must be 0-100' }]}>
        <InputNumber min={0} max={100} step={1} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="purchased_at" label="Purchase Date"><Input type="date" /></Form.Item>
      <Form.Item name="notes" label="Notes"><Input.TextArea rows={2} /></Form.Item>
      <Button type="primary" htmlType="submit" loading={loading} block>Save</Button>
    </Form>
  )
}

interface AssetFormValues {
  name: string
  category: string
  purchase_value: number | null
  depreciation_rate_pct: number
  purchased_at: string | null
  notes: string
}

function buildAssetPayload(values: AssetFormValues) {
  return {
    name: values.name,
    category: values.category,
    value: values.purchase_value ?? 0,
    notes: values.notes || '',
    depreciation_rate: (values.depreciation_rate_pct ?? 0) / 100,
    purchase_value: values.purchase_value ?? null,
    purchased_at: values.purchased_at || null,
  }
}

function AssetsTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [editAsset, setEditAsset] = useState<Asset | null>(null)
  const [addForm] = Form.useForm<AssetFormValues>()
  const [editForm] = Form.useForm<AssetFormValues>()
  const qc = useQueryClient()

  const { data: assets = [], isLoading } = useQuery({ queryKey: ['assets'], queryFn: getAssets })

  const addMutation = useMutation({
    mutationFn: (values: AssetFormValues) => createAsset(buildAssetPayload(values)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['assets'] }); setAddOpen(false); addForm.resetFields() },
  })

  const editMutation = useMutation({
    mutationFn: ({ id, values }: { id: string; values: AssetFormValues }) => updateAsset(id, buildAssetPayload(values)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['assets'] }); setEditAsset(null) },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteAsset,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['assets'] }),
  })

  const grandTotal = assets.reduce((s, a) => s + a.current_value, 0)
  const categories = [...new Set(assets.map(a => a.category))]

  const columns: ColumnsType<Asset> = [
    { title: 'Name', dataIndex: 'name', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 120 },
    {
      title: 'Current Value', dataIndex: 'current_value', width: 140, align: 'right',
      render: (cv, row) => (
        <Tooltip title={row.purchase_value ? `Purchased $${row.purchase_value.toLocaleString()}, ${(row.depreciation_rate * 100).toFixed(0)}%/yr depreciation` : undefined}>
          <span>${cv.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
        </Tooltip>
      ),
    },
    { title: 'Bought', dataIndex: 'purchased_at', width: 110, render: v => v ?? '—' },
    {
      title: '', width: 72,
      render: (_, row) => (
        <>
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => {
            setEditAsset(row)
            editForm.setFieldsValue({
              name: row.name,
              category: row.category,
              purchase_value: row.purchase_value,
              depreciation_rate_pct: Math.round(row.depreciation_rate * 100),
              purchased_at: row.purchased_at,
              notes: row.notes,
            })
          }} />
          <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} />
        </>
      ),
    },
  ]

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Total Assets</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#52c41a' }}>${grandTotal.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</div>
          </Card>
        </Col>
        {categories.slice(0, 3).map(cat => {
          const total = assets.filter(a => a.category === cat).reduce((s, a) => s + a.current_value, 0)
          return (
            <Col span={6} key={cat}>
              <Card size="small">
                <div style={{ fontSize: 12, color: '#999' }}>{cat}</div>
                <div style={{ fontSize: 18, fontWeight: 600 }}>${total.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</div>
              </Card>
            </Col>
          )
        })}
      </Row>

      <Card size="small" title="Assets" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={assets} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>

      <Modal title="Add Asset" open={addOpen} onCancel={() => { setAddOpen(false); addForm.resetFields() }} footer={null}>
        {assetFormFields(addForm, values => addMutation.mutate(values), addMutation.isPending)}
      </Modal>

      <Drawer title="Edit Asset" open={editAsset !== null} onClose={() => setEditAsset(null)} width={400} footer={null}>
        {editAsset && assetFormFields(editForm, values => editMutation.mutate({ id: editAsset.id, values }), editMutation.isPending)}
      </Drawer>
    </>
  )
}

export function WealthPage() {
  return (
    <Tabs
      defaultActiveKey="transactions"
      items={[
        { key: 'transactions', label: 'Transactions', children: <TransactionsTab /> },
        { key: 'budgets',      label: 'Budgets',      children: <BudgetsTab /> },
        { key: 'assets',       label: 'Assets',       children: <AssetsTab /> },
      ]}
    />
  )
}
