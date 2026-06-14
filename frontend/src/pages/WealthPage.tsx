import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Tabs, Row, Col, Card, Table, Tag, Button, Form, Input, Select,
  InputNumber, Modal, Progress, Spin, Tooltip, Drawer,
} from 'antd'
import type { FormInstance } from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined, LineChartOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import {
  getTransactions, createTransaction, deleteTransaction,
  getBudgets, upsertBudget,
  getAssets, createAsset, updateAsset, deleteAsset,
  getNetWorthSnapshots, addNetWorthSnapshot,
  getBenchmarks, getBankRates, getNews,
} from '../api/endpoints'
import type { Transaction, Asset, BankRate, NewsItem } from '../api/types'
import { NetWorthChart } from '../components/NetWorthChart'

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

const BANK_DISPLAY: Record<string, string> = {
  vcb: 'Vietcombank', bidv: 'BIDV', agribank: 'Agribank', tcb: 'Techcombank',
}

function TrendsTab() {
  const [backfillOpen, setBackfillOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const now = new Date()
  const yearAgo = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate()).toISOString().split('T')[0]
  const todayStr = now.toISOString().split('T')[0]

  const { data: snapshots = [] } = useQuery({
    queryKey: ['net-worth-snapshots'],
    queryFn: getNetWorthSnapshots,
  })

  const { data: benchmarks = [] } = useQuery({
    queryKey: ['benchmarks', yearAgo, todayStr],
    queryFn: () => getBenchmarks(['vn_index', 'sjc_gold', 'gso_cpi'], yearAgo, todayStr),
  })

  const { data: bankRates = [] } = useQuery({
    queryKey: ['bank-rates'],
    queryFn: getBankRates,
  })

  const { data: news = [] } = useQuery({
    queryKey: ['news'],
    queryFn: getNews,
  })

  const addMutation = useMutation({
    mutationFn: addNetWorthSnapshot,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['net-worth-snapshots'] })
      setBackfillOpen(false)
      form.resetFields()
    },
  })

  const latest = snapshots[snapshots.length - 1]
  const thirtyDaysAgo = new Date(now)
  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)
  const cutoff30 = thirtyDaysAgo.toISOString().split('T')[0]
  const snap30 = snapshots.filter(s => s.snapshot_date <= cutoff30).slice(-1)[0]

  const pctChange = (curr: number, prev?: number) =>
    prev && prev !== 0 ? ((curr - prev) / prev * 100).toFixed(1) : null

  const latestBenchmark = (source: string) => {
    const pts = benchmarks.filter(b => b.source === source).sort((a, b) => a.date.localeCompare(b.date))
    return { latest: pts[pts.length - 1], oldest: pts[0] }
  }

  const vnidx = latestBenchmark('vn_index')
  const gold = latestBenchmark('sjc_gold')

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>Net Worth (30d)</div>
            <div style={{ fontSize: 18, fontWeight: 700, color: '#1677ff' }}>
              {latest ? `₫${(latest.net_worth / 1e6).toFixed(1)}M` : '—'}
            </div>
            {snap30 && latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(latest.net_worth, snap30.net_worth)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(latest.net_worth, snap30.net_worth)}% vs 30d ago
              </div>
            )}
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>VN-Index (1Y)</div>
            <div style={{ fontSize: 18, fontWeight: 700 }}>
              {vnidx.latest ? vnidx.latest.value.toFixed(0) : '—'}
            </div>
            {vnidx.oldest && vnidx.latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(vnidx.latest.value, vnidx.oldest.value)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(vnidx.latest.value, vnidx.oldest.value)}% vs 1Y ago
              </div>
            )}
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>SJC Gold (1Y)</div>
            <div style={{ fontSize: 18, fontWeight: 700 }}>
              {gold.latest ? `${(gold.latest.value / 1e6).toFixed(1)}M/lượng` : '—'}
            </div>
            {gold.oldest && gold.latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(gold.latest.value, gold.oldest.value)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(gold.latest.value, gold.oldest.value)}% vs 1Y ago
              </div>
            )}
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Button size="small" icon={<PlusOutlined />} onClick={() => setBackfillOpen(true)}>
              Add past data point
            </Button>
          </Card>
        </Col>
      </Row>

      <Card size="small" title="Net Worth Trend vs Benchmarks (% change from start)" style={{ marginBottom: 16 }}>
        <NetWorthChart snapshots={snapshots} benchmarks={benchmarks} />
      </Card>

      <Card size="small" title="Bank Interest Rates" style={{ marginBottom: 16 }}>
        {bankRates.length === 0 ? (
          <div style={{ color: '#bbb', fontSize: 12 }}>Rates fetched daily. Check back tomorrow.</div>
        ) : (
          <>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <th style={{ padding: '6px 8px', textAlign: 'left', color: '#999', fontWeight: 500 }}>Bank</th>
                  <th style={{ padding: '6px 8px', textAlign: 'right', color: '#999', fontWeight: 500 }}>Saving 12m</th>
                  <th style={{ padding: '6px 8px', textAlign: 'right', color: '#999', fontWeight: 500 }}>Lending</th>
                </tr>
              </thead>
              <tbody>
                {bankRates.map((r: BankRate) => (
                  <tr key={r.bank} style={{ borderBottom: '1px solid #f5f5f5' }}>
                    <td style={{ padding: '6px 8px' }}>{BANK_DISPLAY[r.bank] ?? r.bank}</td>
                    <td style={{ padding: '6px 8px', textAlign: 'right', color: '#52c41a' }}>{r.saving_12m}%</td>
                    <td style={{ padding: '6px 8px', textAlign: 'right', color: '#ff4d4f' }}>{r.lending}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {bankRates[0] && (
              <div style={{ fontSize: 11, color: '#bbb', marginTop: 6 }}>Updated: {bankRates[0].fetched_date}</div>
            )}
          </>
        )}
      </Card>

      <Card size="small" title="Finance News (cafef.vn)">
        {news.length === 0 ? (
          <div style={{ color: '#bbb', fontSize: 12 }}>News fetched daily.</div>
        ) : (
          news.slice(0, 10).map((n: NewsItem) => (
            <div key={n.id} style={{ padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
              <a href={n.url} target="_blank" rel="noopener noreferrer"
                style={{ fontSize: 13, color: '#1677ff', textDecoration: 'none' }}>
                {n.title}
              </a>
              <div style={{ fontSize: 11, color: '#bbb', marginTop: 2 }}>
                {new Date(n.published_at).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}
              </div>
            </div>
          ))
        )}
      </Card>

      <Modal title="Add Past Net Worth" open={backfillOpen} onCancel={() => setBackfillOpen(false)} footer={null}>
        <Form form={form} layout="vertical"
          onFinish={values => addMutation.mutate({ date: values.date, net_worth: values.net_worth, note: values.note })}>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="net_worth" label="Net Worth (₫)" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} min={0} step={1000000} />
          </Form.Item>
          <Form.Item name="note" label="Note (optional)"><Input /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
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
        { key: 'trends', label: <><LineChartOutlined /> Trends</>, children: <TrendsTab /> },
      ]}
    />
  )
}
