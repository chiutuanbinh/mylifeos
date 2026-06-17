import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Tabs, Card, Table, Tag, Button, Form, Input, Select, Switch,
  InputNumber, Modal, Spin, Badge,
} from 'antd'
import { PlusOutlined, FolderOutlined, FileOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getAccounts, createAccount, createJournalEntry, getJournalNetWorth } from '../api/endpoints'
import type { Account, CreateAccountRequest, CreateJournalEntryRequest } from '../api/types'

const TYPE_COLORS: Record<string, string> = {
  asset: 'green', liability: 'red', equity: 'blue', income: 'cyan', expense: 'orange',
}

const fmtVND = (s: string) => `₫${Math.round(Math.abs(parseFloat(s))).toLocaleString('vi-VN')}`

function AccountsTab() {
  const [addOpen, setAddOpen] = useState(false)
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

  const groupAccounts = accounts.filter(a => a.is_group)

  const columns: ColumnsType<Account> = [
    {
      title: 'Name', dataIndex: 'name',
      render: (name, row) => (
        <span>
          {row.is_group ? <FolderOutlined style={{ marginRight: 6, color: '#faad14' }} /> : <FileOutlined style={{ marginRight: 6, color: '#8c8c8c' }} />}
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
      title: 'Parent', dataIndex: 'parent_id', width: 160,
      render: pid => accounts.find(a => a.id === pid)?.name ?? '—',
    },
  ]

  return (
    <>
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
          <Table
            dataSource={accounts}
            columns={columns}
            size="small"
            rowKey="id"
            pagination={false}
            scroll={{ x: true }}
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
          onFinish={(values: CreateAccountRequest) => createMutation.mutate(values)}
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
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={createMutation.isPending} block>
              Save
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
      setAddOpen(false)
      form.resetFields()
    },
  })

  return (
    <>
      {nw && (
        <Card size="small" style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 12, color: '#999' }}>Live Net Worth</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#1677ff' }}>
            {fmtVND(nw.net_worth)}
          </div>
        </Card>
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
        <div style={{ color: '#999', padding: '24px 0', textAlign: 'center' }}>
          Journal history coming soon
        </div>
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
            {(fields, { add, remove }) => (
              <>
                {fields.map((field, idx) => (
                  <Card key={field.key} size="small" style={{ marginBottom: 8 }}
                    title={`Line ${idx + 1}`}
                    extra={fields.length > 2 && (
                      <Button type="text" size="small" danger onClick={() => remove(field.name)}>Remove</Button>
                    )}
                  >
                    <Form.Item name={[field.name, 'account_id']} label="Account" rules={[{ required: true }]}>
                      <Select
                        showSearch
                        optionFilterProp="label"
                        options={leafAccounts.map(a => ({ value: a.id, label: `${a.name} (${a.type})` }))}
                      />
                    </Form.Item>
                    <Form.Item name={[field.name, 'amount']} label="Amount (VND)" rules={[{ required: true }]}>
                      <InputNumber min={1} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'side']} label="Side" rules={[{ required: true }]}>
                      <Select options={[{ value: 'debit', label: 'Debit' }, { value: 'credit', label: 'Credit' }]} />
                    </Form.Item>
                  </Card>
                ))}
                <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ side: 'debit' })}>
                  Add Line
                </Button>
              </>
            )}
          </Form.List>

          <Form.Item style={{ marginTop: 16 }}>
            <Button type="primary" htmlType="submit" loading={recordMutation.isPending} block>
              Post Entry
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

export function AccountingPage() {
  return (
    <Tabs
      defaultActiveKey="accounts"
      items={[
        { key: 'accounts', label: 'Accounts', children: <AccountsTab /> },
        { key: 'journal', label: 'Journal', children: <JournalTab /> },
      ]}
    />
  )
}
