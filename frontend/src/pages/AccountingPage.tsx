import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Tabs, Card, Table, Tag, Button, Form, Input, Select, Switch,
  InputNumber, Modal, Spin, Badge, Checkbox, Radio, Collapse, Row, Col,
  Popconfirm, message,
} from 'antd'
import { PlusOutlined, FolderOutlined, FileOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getAccounts, createAccount, updateAccount, deleteAccount, createJournalEntry, getJournalEntries, getJournalNetWorth } from '../api/endpoints'
import { ReportsTab } from './ReportsTab'
import type { Account, CreateAccountRequest, UpdateAccountRequest, CreateJournalEntryRequest, JournalEntry } from '../api/types'

function normalSide(type: Account['type']): 'debit' | 'credit' {
  return type === 'asset' || type === 'expense' ? 'debit' : 'credit'
}

const TYPE_COLORS: Record<string, string> = {
  asset: 'green', liability: 'red', equity: 'blue', income: 'cyan', expense: 'orange',
}

const fmtVND = (s: string) => `₫${Math.round(Math.abs(parseFloat(s))).toLocaleString('vi-VN')}`

const DEFAULT_GROUPS: CreateAccountRequest[] = [
  { name: 'Assets',      type: 'asset',     currency: 'VND', is_group: true, sort_order: 0, parent_id: null },
  { name: 'Liabilities', type: 'liability', currency: 'VND', is_group: true, sort_order: 1, parent_id: null },
  { name: 'Equity',      type: 'equity',    currency: 'VND', is_group: true, sort_order: 2, parent_id: null },
  { name: 'Income',      type: 'income',    currency: 'VND', is_group: true, sort_order: 3, parent_id: null },
  { name: 'Expenses',    type: 'expense',   currency: 'VND', is_group: true, sort_order: 4, parent_id: null },
]

type LeafDef = { name: string; type: CreateAccountRequest['type']; parentGroup: string; sortOrder: number }

const DEFAULT_LEAVES: LeafDef[] = [
  { name: 'Cash',            type: 'asset',     parentGroup: 'Assets',      sortOrder: 0 },
  { name: 'Bank Account',    type: 'asset',     parentGroup: 'Assets',      sortOrder: 1 },
  { name: 'Credit Card',     type: 'liability', parentGroup: 'Liabilities', sortOrder: 0 },
  { name: 'Opening Balance', type: 'equity',    parentGroup: 'Equity',      sortOrder: 0 },
  { name: 'Salary',          type: 'income',    parentGroup: 'Income',      sortOrder: 0 },
  { name: 'Living Expenses', type: 'expense',   parentGroup: 'Expenses',    sortOrder: 0 },
]

function SetupWizard({ open, onDone }: { open: boolean; onDone: () => void }) {
  const [selected, setSelected] = useState<string[]>(DEFAULT_LEAVES.map(l => l.name))
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const qc = useQueryClient()

  const toggle = (name: string, checked: boolean) => {
    setSelected(prev => checked ? [...prev, name] : prev.filter(n => n !== name))
  }

  const handleSetUp = async () => {
    setLoading(true)
    setError(null)
    try {
      const groups = await Promise.all(DEFAULT_GROUPS.map(g => createAccount(g)))
      const groupMap: Record<string, string> = {}
      DEFAULT_GROUPS.forEach((g, i) => { groupMap[g.name] = groups[i].id })

      const chosenLeaves = DEFAULT_LEAVES.filter(l => selected.includes(l.name))
      await Promise.all(chosenLeaves.map(l => createAccount({
        name: l.name,
        type: l.type,
        currency: 'VND',
        is_group: false,
        sort_order: l.sortOrder,
        parent_id: groupMap[l.parentGroup],
      })))

      await qc.invalidateQueries({ queryKey: ['accounts'] })
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create accounts. Please retry.')
      setLoading(false)
    }
  }

  if (!open) return null

  return (
    <Modal
      open={open}
      title="Set up your accounts"
      footer={null}
      closable={false}
      maskClosable={false}
    >
      <p style={{ color: '#666', marginBottom: 16 }}>
        We'll create a starter chart of accounts. Uncheck any you don't need.
      </p>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 20 }}>
        {DEFAULT_LEAVES.map(l => (
          <Checkbox
            key={l.name}
            checked={selected.includes(l.name)}
            onChange={e => toggle(l.name, e.target.checked)}
          >
            {l.name} <Tag color={TYPE_COLORS[l.type]}>{l.type}</Tag>
          </Checkbox>
        ))}
      </div>
      {error && <div style={{ color: '#ff4d4f', marginBottom: 12 }}>{error}</div>}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Button type="link" onClick={onDone} disabled={loading}>Skip</Button>
        <Button type="primary" loading={loading} onClick={handleSetUp}>Set Up</Button>
      </div>
    </Modal>
  )
}

type AccountTreeNode = Account & { children?: AccountTreeNode[] }

function buildTree(accounts: Account[]): AccountTreeNode[] {
  const byId = new Map(accounts.map(a => [a.id, { ...a, children: [] as AccountTreeNode[] }]))
  const roots: AccountTreeNode[] = []
  for (const node of byId.values()) {
    if (node.parent_id && byId.has(node.parent_id)) {
      byId.get(node.parent_id)!.children!.push(node)
    } else {
      roots.push(node)
    }
  }
  // strip empty children arrays so Ant Design doesn't show expand icon
  const clean = (n: AccountTreeNode): AccountTreeNode => ({
    ...n,
    children: n.children && n.children.length > 0 ? n.children.map(clean) : undefined,
  })
  return roots.map(clean).sort((a, b) => a.sort_order - b.sort_order)
}

function AccountsTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [wizardDone, setWizardDone] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: accounts = [], isLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: getAccounts,
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateAccountRequest) => createAccount(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['accounts'] })
      setAddOpen(false)
      form.resetFields()
    },
  })

  const [editOpen, setEditOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Account | null>(null)
  const [editForm] = Form.useForm()

  const editMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateAccountRequest }) => updateAccount(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['accounts'] })
      setEditOpen(false)
      setEditTarget(null)
      editForm.resetFields()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteAccount(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['accounts'] })
      message.success('Account deleted')
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Failed to delete account'
      message.error(msg)
    },
  })

  const openEdit = (account: Account) => {
    setEditTarget(account)
    editForm.setFieldsValue({
      name: account.name,
      type: account.type,
      parent_id: account.parent_id,
      sort_order: account.sort_order,
      is_group: account.is_group,
      asset_meta_purchase_value: account.asset_meta?.purchase_value ? parseFloat(account.asset_meta.purchase_value) : undefined,
      asset_meta_purchased_at: account.asset_meta?.purchased_at ?? undefined,
      asset_meta_depreciation_rate: account.asset_meta?.depreciation_rate ? parseFloat(account.asset_meta.depreciation_rate) : undefined,
      asset_meta_notes: account.asset_meta?.notes ?? '',
    })
    setEditOpen(true)
  }

  const groupAccounts = accounts.filter(a => a.is_group)
  const treeData = buildTree(accounts)
  const defaultExpandedKeys = accounts.filter(a => a.is_group).map(a => a.id)

  const columns: ColumnsType<AccountTreeNode> = [
    {
      title: 'Name', dataIndex: 'name',
      render: (name, row) => (
        <span>
          {row.is_group
            ? <FolderOutlined style={{ marginRight: 6, color: '#faad14' }} />
            : <FileOutlined style={{ marginRight: 6, color: '#8c8c8c' }} />}
          {name}
          {row.archived && <Badge count="archived" style={{ marginLeft: 8, backgroundColor: '#d9d9d9', color: '#595959', fontSize: 10 }} />}
        </span>
      ),
    },
    {
      title: 'Type', dataIndex: 'type', width: 110,
      render: t => <Tag color={TYPE_COLORS[t]}>{t}</Tag>,
    },
    { title: 'Currency', dataIndex: 'currency', width: 90 },
    {
      title: 'Balance', dataIndex: 'balance', width: 160, align: 'right',
      render: (bal: number, row) => {
        const neg = bal < 0
        return (
          <span style={{ fontWeight: row.is_group ? 600 : 400, color: neg ? '#ff4d4f' : undefined }}>
            {neg ? '-' : ''}{fmtVND(String(bal))}
          </span>
        )
      },
    },
    {
      title: '',
      width: 48,
      render: (_: unknown, row: AccountTreeNode) => (
        <Button
          type="text"
          size="small"
          icon={<EditOutlined />}
          onClick={() => openEdit(row)}
        />
      ),
    },
    {
      title: '',
      width: 48,
      render: (_: unknown, row: AccountTreeNode) => (
        <Popconfirm
          title="Delete account?"
          description="This cannot be undone."
          onConfirm={() => deleteMutation.mutate(row.id)}
          okText="Delete"
          okButtonProps={{ danger: true }}
        >
          <Button
            type="text"
            size="small"
            danger
            icon={<DeleteOutlined />}
          />
        </Popconfirm>
      ),
    },
  ]

  const showWizard = !isLoading && accounts.length === 0 && !wizardDone

  return (
    <>
      <SetupWizard open={showWizard} onDone={() => setWizardDone(true)} />
      <Card
        size="small"
        title="Chart of Accounts"
        extra={
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>
            Add Account
          </Button>
        }
      >
        {isLoading ? <Spin /> : (
          <Table<AccountTreeNode>
            dataSource={treeData}
            columns={columns}
            size="small"
            rowKey="id"
            pagination={false}
            scroll={{ x: true }}
            expandable={{ defaultExpandedRowKeys: defaultExpandedKeys }}
          />
        )}
      </Card>

      <Modal
        title="New Account"
        open={addOpen}
        onCancel={() => { setAddOpen(false); form.resetFields() }}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ type: 'asset', currency: 'VND', is_group: false, sort_order: 0 }}
          onFinish={(values) => {
            const req: CreateAccountRequest = {
              name: values.name,
              type: values.type,
              currency: values.currency ?? 'VND',
              is_group: values.is_group ?? false,
              sort_order: values.sort_order ?? 0,
              parent_id: values.parent_id ?? null,
              opening_balance: values.opening_balance ?? undefined,
            }
            createMutation.mutate(req)
          }}
        >
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Required' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="Type" rules={[{ required: true }]}>
            <Select options={['asset','liability','equity','income','expense'].map(t => ({ value: t, label: t }))} />
          </Form.Item>
          <Form.Item name="currency" label="Currency">
            <Input disabled />
          </Form.Item>
          <Form.Item name="parent_id" label="Parent Group">
            <Select
              allowClear
              placeholder="None (root)"
              options={groupAccounts.map(a => ({ value: a.id, label: a.name }))}
            />
          </Form.Item>
          <Form.Item name="is_group" label="Is Group?" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="sort_order" label="Sort Order">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.is_group !== cur.is_group}>
            {({ getFieldValue }) =>
              !getFieldValue('is_group') && (
                <Form.Item name="opening_balance" label="Opening Balance (VND)">
                  <InputNumber min={0} style={{ width: '100%' }} placeholder="0 — leave empty to skip" />
                </Form.Item>
              )
            }
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type || prev.is_group !== cur.is_group}>
            {({ getFieldValue }) =>
              getFieldValue('type') === 'asset' && !getFieldValue('is_group') && (
                <Collapse ghost size="small" style={{ marginBottom: 8 }}>
                  <Collapse.Panel header="Asset Details (optional)" key="asset">
                    <Form.Item name="asset_meta_purchase_value" label="Purchase Value (VND)">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="asset_meta_purchased_at" label="Purchase Date">
                      <Input type="date" />
                    </Form.Item>
                    <Form.Item name="asset_meta_depreciation_rate" label="Annual Depreciation Rate (0–1)">
                      <InputNumber min={0} max={1} step={0.01} style={{ width: '100%' }} placeholder="e.g. 0.15 for 15%" />
                    </Form.Item>
                    <Form.Item name="asset_meta_notes" label="Notes">
                      <Input />
                    </Form.Item>
                  </Collapse.Panel>
                </Collapse>
              )
            }
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={createMutation.isPending} block>
              Save
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Edit Account"
        open={editOpen}
        onCancel={() => { setEditOpen(false); setEditTarget(null); editForm.resetFields() }}
        footer={null}
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={(values) => {
            if (!editTarget) return
            const assetMeta = (values.type === 'asset' && !editTarget.is_group && (
              values.asset_meta_purchase_value || values.asset_meta_purchased_at || values.asset_meta_depreciation_rate
            )) ? {
              purchase_value: values.asset_meta_purchase_value ?? null,
              purchased_at: values.asset_meta_purchased_at ?? null,
              depreciation_rate: values.asset_meta_depreciation_rate ?? null,
              notes: values.asset_meta_notes ?? '',
            } : null
            editMutation.mutate({
              id: editTarget.id,
              data: {
                name: values.name,
                type: values.type,
                parent_id: values.parent_id ?? null,
                sort_order: values.sort_order ?? 0,
                asset_meta: assetMeta,
              },
            })
          }}
        >
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Required' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="Type" rules={[{ required: true }]}>
            <Select options={['asset','liability','equity','income','expense'].map(t => ({ value: t, label: t }))} />
          </Form.Item>
          <Form.Item name="parent_id" label="Parent Group">
            <Select
              allowClear
              placeholder="None (root)"
              options={groupAccounts.filter(a => a.id !== editTarget?.id).map(a => ({ value: a.id, label: a.name }))}
            />
          </Form.Item>
          <Form.Item name="sort_order" label="Sort Order">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type}>
            {({ getFieldValue }) =>
              getFieldValue('type') === 'asset' && editTarget && !editTarget.is_group && (
                <Collapse ghost size="small" style={{ marginBottom: 8 }}>
                  <Collapse.Panel header="Asset Details (optional)" key="asset">
                    <Form.Item name="asset_meta_purchase_value" label="Purchase Value (VND)">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="asset_meta_purchased_at" label="Purchase Date">
                      <Input type="date" />
                    </Form.Item>
                    <Form.Item name="asset_meta_depreciation_rate" label="Annual Depreciation Rate (0–1)">
                      <InputNumber min={0} max={1} step={0.01} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="asset_meta_notes" label="Notes">
                      <Input />
                    </Form.Item>
                  </Collapse.Panel>
                </Collapse>
              )
            }
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={editMutation.isPending} block>
              Save Changes
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

function JournalTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: accounts = [] } = useQuery({ queryKey: ['accounts'], queryFn: getAccounts })
  const { data: entries = [], isLoading: entriesLoading } = useQuery({ queryKey: ['journal-entries'], queryFn: getJournalEntries })
  const { data: nw } = useQuery({ queryKey: ['journal-networth'], queryFn: getJournalNetWorth })

  const leafAccounts = accounts.filter(a => !a.is_group)

  const recordMutation = useMutation({
    mutationFn: (values: { date: string; description: string; memo: string; lines: { account_id: string; amount: number; side: 'debit' | 'credit' }[] }) => {
      const req: CreateJournalEntryRequest = {
        date: values.date,
        description: values.description,
        memo: values.memo ?? '',
        lines: values.lines.map(l => ({
          account_id: l.account_id,
          amount: String(l.amount),
          currency: 'VND',
          side: l.side,
        })),
      }
      return createJournalEntry(req)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['journal-networth'] })
      qc.invalidateQueries({ queryKey: ['journal-entries'] })
      setAddOpen(false)
      form.resetFields()
    },
  })

  return (
    <>
      {nw && (
        <Row gutter={12} style={{ marginBottom: 12 }}>
          <Col xs={24} sm={12}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>Live Net Worth</div>
              <div style={{ fontSize: 28, fontWeight: 700, color: '#1677ff' }}>
                {fmtVND(nw.net_worth)}
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>Net Income (YTD)</div>
              <div style={{
                fontSize: 28, fontWeight: 700,
                color: parseFloat(nw.net_income_ytd) >= 0 ? '#52c41a' : '#ff4d4f',
              }}>
                {parseFloat(nw.net_income_ytd) < 0 ? '-' : ''}{fmtVND(nw.net_income_ytd)}
              </div>
            </Card>
          </Col>
        </Row>
      )}
      <Card
        size="small"
        title="Journal"
        extra={
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>
            Record Entry
          </Button>
        }
      >
        {entriesLoading ? <Spin /> : (
          <Table<JournalEntry>
            dataSource={[...entries].reverse()}
            rowKey="id"
            size="small"
            pagination={{ pageSize: 20, size: 'small' }}
            scroll={{ x: true }}
            columns={[
              { title: 'Date', dataIndex: 'date', width: 110 },
              { title: 'Description', dataIndex: 'description' },
              {
                title: 'Lines', dataIndex: 'lines',
                render: (lines: JournalEntry['lines']) => (
                  <div style={{ fontSize: 12 }}>
                    {lines.map(l => {
                      const acct = accounts.find(a => a.id === l.account_id)
                      return (
                        <div key={l.id}>
                          <Tag color={l.side === 'debit' ? 'blue' : 'green'} style={{ fontSize: 11 }}>
                            {l.side === 'debit' ? 'DR' : 'CR'}
                          </Tag>
                          {acct?.name ?? l.account_id} — {fmtVND(l.amount)}
                        </div>
                      )
                    })}
                  </div>
                ),
              },
            ]}
            locale={{ emptyText: 'No entries yet. Record your first entry above.' }}
          />
        )}
      </Card>

      <Modal
        title="Record Journal Entry"
        open={addOpen}
        onCancel={() => { setAddOpen(false); form.resetFields() }}
        footer={null}
        width={560}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ lines: [{ side: 'debit' }, { side: 'credit' }] }}
          onFinish={values => recordMutation.mutate(values)}
        >
          <Form.Item name="date" label="Date" rules={[{ required: true }]}>
            <Input type="date" />
          </Form.Item>
          <Form.Item name="description" label="Description" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="memo" label="Memo">
            <Input />
          </Form.Item>

          <Form.List name="lines">
            {(fields, { add, remove }) => {
              const lines: { account_id?: string; amount?: number; side?: 'debit' | 'credit' }[] =
                form.getFieldValue('lines') ?? []
              const drTotal = lines.reduce((s, l) => l.side === 'debit' ? s + (l.amount ?? 0) : s, 0)
              const crTotal = lines.reduce((s, l) => l.side === 'credit' ? s + (l.amount ?? 0) : s, 0)
              const balanced = drTotal > 0 && drTotal === crTotal

              return (
                <>
                  {/* Header row */}
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 160px 80px 32px', gap: 8, marginBottom: 4, padding: '0 4px' }}>
                    <span style={{ fontSize: 12, color: '#8c8c8c' }}>Account</span>
                    <span style={{ fontSize: 12, color: '#8c8c8c' }}>Amount (VND)</span>
                    <span style={{ fontSize: 12, color: '#8c8c8c' }}>DR / CR</span>
                    <span />
                  </div>

                  {fields.map((field) => (
                    <div key={field.key} style={{ display: 'grid', gridTemplateColumns: '1fr 160px 80px 32px', gap: 8, marginBottom: 8, alignItems: 'flex-start' }}>
                      <Form.Item name={[field.name, 'account_id']} style={{ margin: 0 }} rules={[{ required: true, message: 'Required' }]}>
                        <Select
                          showSearch
                          optionFilterProp="label"
                          placeholder="Account"
                          options={leafAccounts.map(a => ({
                            value: a.id,
                            label: `${a.name} (${a.type} · ${normalSide(a.type) === 'debit' ? 'DR+' : 'CR+'})`,
                          }))}
                          onChange={(accountId: string) => {
                            const acct = leafAccounts.find(a => a.id === accountId)
                            if (!acct) return
                            const side = normalSide(acct.type)
                            const currentLines: { account_id?: string; amount?: number; side?: 'debit' | 'credit' }[] =
                              form.getFieldValue('lines')
                            // set this line's side to the account's normal side
                            currentLines[field.name] = { ...currentLines[field.name], side }
                            form.setFieldsValue({ lines: currentLines })
                            // if this is the first line and there's only one line, add a second with opposite side
                            if (field.name === 0 && fields.length === 1) {
                              add({ side: side === 'debit' ? 'credit' : 'debit' })
                            }
                          }}
                        />
                      </Form.Item>

                      <Form.Item name={[field.name, 'amount']} style={{ margin: 0 }} rules={[{ required: true, message: 'Required' }]}>
                        <InputNumber min={1} style={{ width: '100%' }} placeholder="0" />
                      </Form.Item>

                      <Form.Item name={[field.name, 'side']} style={{ margin: 0 }} rules={[{ required: true, message: 'Required' }]}>
                        <Radio.Group size="small">
                          <Radio.Button value="debit">DR</Radio.Button>
                          <Radio.Button value="credit">CR</Radio.Button>
                        </Radio.Group>
                      </Form.Item>

                      <Button
                        type="text"
                        size="small"
                        danger
                        disabled={fields.length <= 2}
                        onClick={() => remove(field.name)}
                        style={{ marginTop: 4 }}
                      >
                        ✕
                      </Button>
                    </div>
                  ))}

                  <Button type="dashed" block icon={<PlusOutlined />} onClick={() => {
                    const currentLines: { side?: 'debit' | 'credit' }[] = form.getFieldValue('lines') ?? []
                    const drCount = currentLines.filter(l => l.side === 'debit').length
                    const crCount = currentLines.filter(l => l.side === 'credit').length
                    add({ side: drCount <= crCount ? 'debit' : 'credit' })
                  }} style={{ marginBottom: 12 }}>
                    Add Line
                  </Button>

                  {/* Balance indicator */}
                  <div style={{ display: 'flex', gap: 16, alignItems: 'center', padding: '8px 4px', background: '#fafafa', borderRadius: 6, marginBottom: 8 }}>
                    <span style={{ fontSize: 12 }}>DR <b style={{ color: '#1677ff' }}>₫{Math.round(drTotal).toLocaleString('vi-VN')}</b></span>
                    <span style={{ fontSize: 12 }}>CR <b style={{ color: '#1677ff' }}>₫{Math.round(crTotal).toLocaleString('vi-VN')}</b></span>
                    {balanced
                      ? <span style={{ fontSize: 12, color: '#52c41a' }}>✓ Balanced</span>
                      : drTotal + crTotal > 0
                        ? <span style={{ fontSize: 12, color: '#ff4d4f' }}>₫{Math.round(Math.abs(drTotal - crTotal)).toLocaleString('vi-VN')} unbalanced</span>
                        : null}
                  </div>
                </>
              )
            }}
          </Form.List>

          <Form.Item style={{ marginTop: 8 }}>
            <Form.Item noStyle shouldUpdate>
              {() => {
                const lines: { amount?: number; side?: 'debit' | 'credit' }[] = form.getFieldValue('lines') ?? []
                const dr = lines.reduce((s, l) => l.side === 'debit' ? s + (l.amount ?? 0) : s, 0)
                const cr = lines.reduce((s, l) => l.side === 'credit' ? s + (l.amount ?? 0) : s, 0)
                const balanced = dr > 0 && dr === cr
                return (
                  <Button type="primary" htmlType="submit" loading={recordMutation.isPending} block disabled={!balanced}>
                    Post Entry
                  </Button>
                )
              }}
            </Form.Item>
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

function AssetsTab() {
  const { data: accounts = [], isLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: getAccounts,
  })

  const assetAccounts = accounts.filter(a => a.asset_meta !== null && !a.is_group)

  const columns: ColumnsType<Account> = [
    {
      title: 'Name', dataIndex: 'name',
      render: (name) => <span><FileOutlined style={{ marginRight: 6, color: '#8c8c8c' }} />{name}</span>,
    },
    { title: 'Purchase Value', dataIndex: ['asset_meta', 'purchase_value'], width: 160, align: 'right',
      render: (v: string | null) => v ? fmtVND(v) : '—' },
    { title: 'Purchased', dataIndex: ['asset_meta', 'purchased_at'], width: 110,
      render: (v: string | null) => v ?? '—' },
    { title: 'Depr. Rate', dataIndex: ['asset_meta', 'depreciation_rate'], width: 100,
      render: (v: string | null) => v ? `${(parseFloat(v) * 100).toFixed(0)}%/yr` : '—' },
    { title: 'Current Balance', dataIndex: 'balance', width: 160, align: 'right',
      render: (bal: number) => {
        const neg = bal < 0
        return <span style={{ color: neg ? '#ff4d4f' : undefined }}>{neg ? '-' : ''}{fmtVND(String(bal))}</span>
      } },
    { title: 'Notes', dataIndex: ['asset_meta', 'notes'],
      render: (v: string) => v || '—' },
  ]

  return (
    <Card size="small" title="Physical Assets">
      {isLoading ? <Spin /> : (
        <Table<Account>
          dataSource={assetAccounts}
          columns={columns}
          size="small"
          rowKey="id"
          pagination={false}
          scroll={{ x: true }}
          locale={{ emptyText: 'No physical assets tracked. Add an account with type "asset" and fill in Asset Details.' }}
        />
      )}
    </Card>
  )
}

export function AccountingPage() {
  const { data: accounts = [] } = useQuery({ queryKey: ['accounts'], queryFn: getAccounts })
  const { data: entries = [] } = useQuery({ queryKey: ['journal-entries'], queryFn: getJournalEntries })

  return (
    <Tabs
      defaultActiveKey="journal"
      items={[
        { key: 'journal', label: 'Journal', children: <JournalTab /> },
        { key: 'accounts', label: 'Accounts', children: <AccountsTab /> },
        { key: 'assets', label: 'Assets', children: <AssetsTab /> },
        { key: 'reports', label: 'Reports', children: <ReportsTab accounts={accounts} entries={entries} /> },
      ]}
    />
  )
}
