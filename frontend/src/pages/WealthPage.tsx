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
  getLiabilities, createLiability, updateLiability, deleteLiability,
  getNetWorthSnapshots, addNetWorthSnapshot,
  getBenchmarks, getBankRates, getNews, triggerScrape,
} from '../api/endpoints'
import type { Transaction, Asset, Liability, BankRate, NewsItem } from '../api/types'
import { NetWorthChart } from '../components/NetWorthChart'

const fmtVND = (n: number) => `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`

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
    { title: 'Date', dataIndex: 'date', width: 105 },
    { title: 'Description', dataIndex: 'description', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 130, render: c => <Tag color={CAT_COLORS[c]}>{c}</Tag> },
    { title: 'Amount', dataIndex: 'amount', align: 'right', width: 150,
      render: a => <span style={{ color: a > 0 ? '#52c41a' : '#ff4d4f', fontWeight: 600, whiteSpace: 'nowrap' }}>{a > 0 ? '+' : '-'}{fmtVND(a)}</span> },
    { title: '', width: 40, render: (_, row) => <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} /> },
  ]

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        {[
          { label: 'Income', val: fmtVND(totalIncome), color: '#52c41a' },
          { label: 'Expenses', val: fmtVND(totalExpenses), color: '#ff4d4f' },
          { label: 'Net Cash', val: (totalIncome - totalExpenses >= 0 ? '' : '-') + fmtVND(totalIncome - totalExpenses), color: '#1677ff' },
        ].map((s, i) => (
          <Col xs={24} sm={8} key={i}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>{s.label}</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: s.color }}>{s.val}</div>
            </Card>
          </Col>
        ))}
      </Row>
      <Card size="small" title="Transactions" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={txs} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} scroll={{ x: true }} />}
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
                <Col xs={24} sm={8} key={b.id}>
                  <div style={{ fontSize: 12, marginBottom: 2 }}>{b.category} <span style={{ color: '#999' }}>{fmtVND(spent)} / {fmtVND(b.monthly_limit)}</span></div>
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
            <InputNumber placeholder="Monthly limit ₫" min={0} step={1} />
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
      <Form.Item name="purchase_value" label="Purchase Value (₫)" rules={[{ required: true, message: 'Purchase value is required' }, { type: 'number', min: 0, message: 'Must be >= 0' }]}>
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
  const { data: liabilities = [] } = useQuery({ queryKey: ['liabilities'], queryFn: getLiabilities })
  const totalLiabilities = liabilities.reduce((s, l) => s + l.balance, 0)
  const netWorth = grandTotal - totalLiabilities

  const columns: ColumnsType<Asset> = [
    { title: 'Name', dataIndex: 'name', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 120 },
    {
      title: 'Current Value', dataIndex: 'current_value', width: 140, align: 'right',
      render: (cv, row) => (
        <Tooltip title={row.purchase_value ? `Purchased ${fmtVND(row.purchase_value)}, ${(row.depreciation_rate * 100).toFixed(0)}%/yr depreciation` : undefined}>
          <span>{fmtVND(cv)}</span>
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
        <Col xs={12} sm={6}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Total Assets</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#52c41a' }}>{fmtVND(grandTotal)}</div>
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Total Liabilities</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#ff4d4f' }}>{fmtVND(totalLiabilities)}</div>
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Net Worth</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: netWorth >= 0 ? '#1677ff' : '#ff4d4f' }}>{fmtVND(netWorth)}</div>
          </Card>
        </Col>
      </Row>

      <Card size="small" title="Assets" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={assets} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} scroll={{ x: true }} />}
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

const LIABILITY_CATEGORIES = ['Mortgage', 'Car Loan', 'Credit Card', 'Personal Loan', 'Student Loan', 'Other']

interface LiabilityFormValues {
  name: string
  category: string
  balance: number
  original_principal: number | null
  interest_rate_pct: number | null
  started_at: string | null
  due_at: string | null
  notes: string
}

function buildLiabilityPayload(values: LiabilityFormValues) {
  return {
    name: values.name,
    category: values.category,
    balance: values.balance,
    original_principal: values.original_principal ?? null,
    interest_rate: values.interest_rate_pct != null ? values.interest_rate_pct / 100 : null,
    started_at: values.started_at || null,
    due_at: values.due_at || null,
    notes: values.notes || '',
  }
}

function liabilityFormFields(form: FormInstance<LiabilityFormValues>, onFinish: (v: LiabilityFormValues) => void, loading: boolean) {
  return (
    <Form form={form} layout="vertical" onFinish={onFinish}>
      <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}><Input /></Form.Item>
      <Form.Item name="category" label="Category" rules={[{ required: true, message: 'Category is required' }]}>
        <Select options={LIABILITY_CATEGORIES.map(c => ({ value: c, label: c }))} />
      </Form.Item>
      <Form.Item name="balance" label="Current Balance (₫)" rules={[{ required: true, message: 'Balance is required' }, { type: 'number', min: 0, message: 'Must be >= 0' }]}>
        <InputNumber min={0} step={1000000} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="original_principal" label="Original Principal (₫)">
        <InputNumber min={0} step={1000000} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="interest_rate_pct" label="Interest Rate (% per year)">
        <InputNumber min={0} max={100} step={0.1} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="started_at" label="Start Date"><Input type="date" /></Form.Item>
      <Form.Item name="due_at" label="Due Date"><Input type="date" /></Form.Item>
      <Form.Item name="notes" label="Notes"><Input.TextArea rows={2} /></Form.Item>
      <Button type="primary" htmlType="submit" loading={loading} block>Save</Button>
    </Form>
  )
}

function LiabilitiesTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [editItem, setEditItem] = useState<Liability | null>(null)
  const [addForm] = Form.useForm<LiabilityFormValues>()
  const [editForm] = Form.useForm<LiabilityFormValues>()
  const qc = useQueryClient()

  const { data: liabilities = [], isLoading } = useQuery({ queryKey: ['liabilities'], queryFn: getLiabilities })

  const addMutation = useMutation({
    mutationFn: (values: LiabilityFormValues) => createLiability(buildLiabilityPayload(values)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['liabilities'] }); setAddOpen(false); addForm.resetFields() },
  })
  const editMutation = useMutation({
    mutationFn: ({ id, values }: { id: string; values: LiabilityFormValues }) => updateLiability(id, buildLiabilityPayload(values)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['liabilities'] }); setEditItem(null); editForm.resetFields() },
  })
  const deleteMutation = useMutation({
    mutationFn: deleteLiability,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['liabilities'] }),
  })

  const totalBalance = liabilities.reduce((s, l) => s + l.balance, 0)

  const columns: ColumnsType<Liability> = [
    { title: 'Name', dataIndex: 'name', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 130 },
    { title: 'Balance', dataIndex: 'balance', width: 150, align: 'right',
      render: v => <span style={{ color: '#ff4d4f', fontWeight: 600 }}>{fmtVND(v)}</span> },
    { title: 'Interest', dataIndex: 'interest_rate', width: 90, align: 'right',
      render: v => v != null ? `${(v * 100).toFixed(1)}%` : '—' },
    { title: 'Due', dataIndex: 'due_at', width: 110, render: v => v ?? '—' },
    {
      title: '', width: 72,
      render: (_, row) => (
        <>
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => {
            setEditItem(row)
            editForm.setFieldsValue({
              name: row.name,
              category: row.category,
              balance: row.balance,
              original_principal: row.original_principal,
              interest_rate_pct: row.interest_rate != null ? Math.round(row.interest_rate * 1000) / 10 : null,
              started_at: row.started_at,
              due_at: row.due_at,
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
        <Col xs={24} sm={8}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Total Liabilities</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#ff4d4f' }}>{fmtVND(totalBalance)}</div>
          </Card>
        </Col>
      </Row>
      <Card size="small" title="Liabilities" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={liabilities} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} scroll={{ x: true }} />}
      </Card>
      <Modal title="Add Liability" open={addOpen} onCancel={() => { setAddOpen(false); addForm.resetFields() }} footer={null}>
        {liabilityFormFields(addForm, values => addMutation.mutate(values), addMutation.isPending)}
      </Modal>
      <Drawer title="Edit Liability" open={editItem !== null} onClose={() => { setEditItem(null); editForm.resetFields() }} width={400} footer={null}>
        {editItem && liabilityFormFields(editForm, values => editMutation.mutate({ id: editItem.id, values }), editMutation.isPending)}
      </Drawer>
    </>
  )
}

const BANK_DISPLAY: Record<string, string> = {
  vcb: 'Vietcombank', bidv: 'BIDV', agribank: 'Agribank', tcb: 'Techcombank',
}

function TrendsTab() {
  const [backfillOpen, setBackfillOpen] = useState(false)
  const [scraping, setScraping] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const handleScrape = async () => {
    setScraping(true)
    try {
      await triggerScrape()
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ['benchmarks'] })
        qc.invalidateQueries({ queryKey: ['bank-rates'] })
        qc.invalidateQueries({ queryKey: ['news'] })
        setScraping(false)
      }, 5000)
    } catch {
      setScraping(false)
    }
  }

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

  const { data: bankRatesRaw } = useQuery({
    queryKey: ['bank-rates'],
    queryFn: getBankRates,
  })
  const bankRates: BankRate[] = bankRatesRaw ?? []

  const { data: newsRaw } = useQuery({
    queryKey: ['news'],
    queryFn: getNews,
  })
  const news: NewsItem[] = newsRaw ?? []

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
        <Col xs={12} sm={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>Net Worth (30d)</div>
            <div style={{ fontSize: 18, fontWeight: 700, color: '#1677ff' }}>
              {latest ? fmtVND(latest.net_worth) : '—'}
            </div>
            {snap30 && latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(latest.net_worth, snap30.net_worth)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(latest.net_worth, snap30.net_worth)}% vs 30d ago
              </div>
            )}
          </Card>
        </Col>
        <Col xs={12} sm={6}>
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
        <Col xs={12} sm={6}>
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
        <Col xs={12} sm={6}>
          <Card size="small" style={{ display: 'flex', flexDirection: 'column', gap: 8, alignItems: 'flex-start', justifyContent: 'center' }}>
            <Button size="small" icon={<PlusOutlined />} onClick={() => setBackfillOpen(true)}>
              Add past data point
            </Button>
            <Button size="small" loading={scraping} onClick={handleScrape}>
              Refresh market data
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

      <Card size="small" title="Finance News (vneconomy.vn)">
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
        { key: 'liabilities',  label: 'Liabilities',  children: <LiabilitiesTab /> },
        { key: 'trends', label: <><LineChartOutlined /> Trends</>, children: <TrendsTab /> },
      ]}
    />
  )
}
