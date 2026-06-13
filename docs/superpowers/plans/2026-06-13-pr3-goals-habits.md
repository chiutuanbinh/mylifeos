# PR3: Goals + Habits Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Goals: edit modal, auto-progress display, inline KR add/delete, status-based visual treatment. Habits: edit modal, month heatmap, streak counter.

**Architecture:** Frontend only. `GoalsPage.tsx` and `HealthPage.tsx` are rewritten with new mutations and UI. New API endpoints for deleteKeyResult, updateHabit, and habit log range are called from these pages.

**Tech Stack:** React, TypeScript, Ant Design, React Query

**Prerequisite:** PR1 must be merged (backend has DeleteKeyResult, habit Update, GetLogRange endpoints). PR2 optional — no dependency.

---

## File Map

| Action | File |
|--------|------|
| Modify | `frontend/src/api/types.ts` |
| Modify | `frontend/src/api/endpoints.ts` |
| Modify | `frontend/src/pages/GoalsPage.tsx` |
| Modify | `frontend/src/pages/HealthPage.tsx` |

---

## Task 1: Branch Setup

- [ ] **Create branch**

```bash
git checkout main && git pull
git checkout -b feat/goals-habits
```

---

## Task 2: Update Types

**Files:**
- Modify: `frontend/src/api/types.ts`

- [ ] **Add `status` to `Goal`**

Replace the `Goal` interface:

```typescript
export interface Goal {
  id: string
  user_id: string
  name: string
  description: string
  target_date: string | null
  progress: number
  color: string
  status: 'active' | 'completed' | 'archived'
  created_at: string
  key_results: KeyResult[]
}
```

- [ ] **Commit**

```bash
git add frontend/src/api/types.ts
git commit -m "feat: add status field to Goal type"
```

---

## Task 3: Update Endpoints

**Files:**
- Modify: `frontend/src/api/endpoints.ts`

- [ ] **Add `deleteKeyResult`, `updateHabit`, `getHabitLogRange`**

Add these after the existing habit/goal functions:

```typescript
export const deleteKeyResult = (goalId: string, krId: string) =>
  apiClient.delete(`/goals/${goalId}/key-results/${krId}`)

export const updateHabit = (id: string, data: { name: string; icon: string }) =>
  apiClient.put<Habit>(`/habits/${id}`, data).then(r => r.data)

export const getHabitLogRange = (habitId: string, from: string, to: string) =>
  apiClient.get<HabitLog[]>(`/habits/${habitId}/logs`, { params: { from, to } }).then(r => r.data)
```

- [ ] **Commit**

```bash
git add frontend/src/api/endpoints.ts
git commit -m "feat: add deleteKeyResult, updateHabit, getHabitLogRange endpoints"
```

---

## Task 4: Rewrite GoalsPage

**Files:**
- Modify: `frontend/src/pages/GoalsPage.tsx`

- [ ] **Rewrite with edit modal, inline KR management, status treatment, auto-progress**

Replace the entire file:

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Row, Col, Card, Progress, Button, Modal, Form, Input, Checkbox,
  Select, Tag, Spin, Tooltip,
} from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined, CheckCircleFilled } from '@ant-design/icons'
import {
  getGoals, createGoal, updateGoal, deleteGoal,
  addKeyResult, updateKeyResult, deleteKeyResult,
} from '../api/endpoints'
import type { Goal } from '../api/types'

const STATUS_COLORS = { active: '#1677ff', completed: '#52c41a', archived: '#bbb' }

function GoalCard({ g }: { g: Goal }) {
  const [newKR, setNewKR] = useState('')
  const [editOpen, setEditOpen] = useState(false)
  const [editForm] = Form.useForm()
  const qc = useQueryClient()

  const invalidate = () => qc.invalidateQueries({ queryKey: ['goals'] })

  const updateMutation = useMutation({
    mutationFn: (values: any) => updateGoal(g.id, values),
    onSuccess: () => { invalidate(); setEditOpen(false) },
  })
  const deleteMutation = useMutation({ mutationFn: () => deleteGoal(g.id), onSuccess: invalidate })
  const addKRMutation = useMutation({
    mutationFn: (desc: string) => addKeyResult(g.id, desc),
    onSuccess: () => { invalidate(); setNewKR('') },
  })
  const toggleKRMutation = useMutation({
    mutationFn: ({ krId, done }: { krId: string; done: boolean }) => updateKeyResult(g.id, krId, { done }),
    onSuccess: invalidate,
  })
  const deleteKRMutation = useMutation({
    mutationFn: (krId: string) => deleteKeyResult(g.id, krId),
    onSuccess: invalidate,
  })

  const isArchived = g.status === 'archived'
  const isCompleted = g.status === 'completed'

  return (
    <>
      <Card
        size="small"
        style={{
          borderTop: `3px solid ${g.color}`,
          opacity: isArchived ? 0.5 : 1,
        }}
        title={
          <span style={{ fontSize: 13, fontWeight: 600 }}>
            {isCompleted && <CheckCircleFilled style={{ color: '#52c41a', marginRight: 6 }} />}
            {g.name}
            {g.target_date && <span style={{ fontSize: 11, color: '#aaa', marginLeft: 8 }}>due {g.target_date}</span>}
          </span>
        }
        extra={
          <>
            <Button type="text" size="small" icon={<EditOutlined />} onClick={() => {
              setEditOpen(true)
              editForm.setFieldsValue({ name: g.name, description: g.description, target_date: g.target_date, color: g.color, status: g.status })
            }} />
            <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate()} />
          </>
        }
      >
        <Progress
          percent={g.progress}
          strokeColor={g.color}
          size="small"
          style={{ marginBottom: 8 }}
          format={p => `${p}%`}
        />
        {g.description && <div style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>{g.description}</div>}

        {(g.key_results ?? []).map(kr => (
          <div key={kr.id} style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
            <Checkbox
              checked={kr.done}
              onChange={e => toggleKRMutation.mutate({ krId: kr.id, done: e.target.checked })}
            />
            <span style={{ fontSize: 12, flex: 1, textDecoration: kr.done ? 'line-through' : 'none', color: kr.done ? '#bbb' : '#222' }}>
              {kr.description}
            </span>
            <Tooltip title="Remove">
              <Button
                type="text" size="small" danger
                icon={<DeleteOutlined />}
                style={{ fontSize: 10 }}
                onClick={() => deleteKRMutation.mutate(kr.id)}
              />
            </Tooltip>
          </div>
        ))}

        <div style={{ display: 'flex', gap: 6, marginTop: 8 }}>
          <Input
            size="small"
            placeholder="Add key result..."
            value={newKR}
            onChange={e => setNewKR(e.target.value)}
            onPressEnter={() => { if (newKR.trim()) addKRMutation.mutate(newKR.trim()) }}
            style={{ flex: 1 }}
          />
          <Button
            size="small"
            type="dashed"
            icon={<PlusOutlined />}
            loading={addKRMutation.isPending}
            disabled={!newKR.trim()}
            onClick={() => addKRMutation.mutate(newKR.trim())}
          />
        </div>
      </Card>

      <Modal title="Edit Goal" open={editOpen} onCancel={() => setEditOpen(false)} footer={null}>
        <Form form={editForm} layout="vertical" onFinish={values => updateMutation.mutate(values)}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true, message: 'Name is required' }, { max: 100 }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="target_date" label="Target date">
            <Input type="date" />
          </Form.Item>
          <Form.Item name="color" label="Color">
            <Input type="color" style={{ width: 80, padding: 2 }} />
          </Form.Item>
          <Form.Item name="status" label="Status" rules={[{ required: true }]}>
            <Select options={[
              { value: 'active',    label: <Tag color="blue">Active</Tag> },
              { value: 'completed', label: <Tag color="green">Completed</Tag> },
              { value: 'archived',  label: <Tag color="default">Archived</Tag> },
            ]} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={updateMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </>
  )
}

export function GoalsPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: goals = [], isLoading } = useQuery({ queryKey: ['goals'], queryFn: getGoals })

  const addMutation = useMutation({
    mutationFn: (values: any) => createGoal({ ...values, status: 'active', key_results: [] }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); setAddOpen(false); form.resetFields() },
  })

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  const active   = goals.filter(g => g.status === 'active')
  const completed = goals.filter(g => g.status === 'completed')
  const archived  = goals.filter(g => g.status === 'archived')
  const ordered  = [...active, ...completed, ...archived]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add Goal</Button>
      </div>
      <Row gutter={[12, 12]}>
        {ordered.map(g => (
          <Col span={8} key={g.id}>
            <GoalCard g={g} />
          </Col>
        ))}
        {goals.length === 0 && (
          <Col span={24}>
            <div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>No goals yet. Add your first!</div>
          </Col>
        )}
      </Row>

      <Modal title="Add Goal" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true, message: 'Name is required' }, { max: 100 }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="target_date" label="Target date">
            <Input type="date" />
          </Form.Item>
          <Form.Item name="color" label="Color" initialValue="#1677ff">
            <Input type="color" style={{ width: 80, padding: 2 }} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Commit**

```bash
git add frontend/src/pages/GoalsPage.tsx
git commit -m "feat: goals — edit modal, inline KR add/delete, status, auto-progress display"
```

---

## Task 5: Rewrite HealthPage

**Files:**
- Modify: `frontend/src/pages/HealthPage.tsx`

- [ ] **Rewrite with edit modal, month heatmap, and streak counter**

Replace the entire file:

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, Row, Col, Button, Modal, Form, Input, Spin, Tooltip } from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons'
import { getHabits, createHabit, updateHabit, deleteHabit, getHabitLogs, toggleHabitLog, getHabitLogRange } from '../api/endpoints'
import type { Habit, HabitLog } from '../api/types'

const today = new Date().toISOString().split('T')[0]

function getMonthRange(): { from: string; to: string; days: string[] } {
  const now = new Date()
  const year = now.getFullYear()
  const month = now.getMonth()
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  const from = firstDay.toISOString().split('T')[0]
  const to = lastDay.toISOString().split('T')[0]
  const days: string[] = []
  for (let d = new Date(firstDay); d <= lastDay; d.setDate(d.getDate() + 1)) {
    days.push(d.toISOString().split('T')[0])
  }
  return { from, to, days }
}

function computeStreak(logs: HabitLog[], days: string[]): number {
  const doneSet = new Set(logs.filter(l => l.done).map(l => l.logged_date))
  let streak = 0
  for (let i = days.indexOf(today); i >= 0; i--) {
    if (doneSet.has(days[i])) streak++
    else break
  }
  return streak
}

function HeatmapStrip({ logs, days }: { logs: HabitLog[]; days: string[] }) {
  const doneSet = new Set(logs.filter(l => l.done).map(l => l.logged_date))
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 2, marginTop: 6 }}>
      {days.map(day => (
        <Tooltip key={day} title={day}>
          <div style={{
            width: 10, height: 10, borderRadius: 2,
            background: doneSet.has(day) ? '#52c41a' : '#e8e8e8',
          }} />
        </Tooltip>
      ))}
    </div>
  )
}

function HabitRow({ habit, doneToday, monthLogs, days, onToggle }: {
  habit: Habit
  doneToday: boolean
  monthLogs: HabitLog[]
  days: string[]
  onToggle: () => void
}) {
  const [editOpen, setEditOpen] = useState(false)
  const [editForm] = Form.useForm()
  const qc = useQueryClient()

  const streak = computeStreak(monthLogs, days)

  const editMutation = useMutation({
    mutationFn: (values: { name: string; icon: string }) => updateHabit(habit.id, values),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['habits'] }); setEditOpen(false) },
  })
  const deleteMutation = useMutation({
    mutationFn: () => { const { deleteHabit: dh } = require('../api/endpoints'); return dh(habit.id) },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['habits'] }),
  })

  return (
    <>
      <div style={{ padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div
            onClick={onToggle}
            style={{
              width: 22, height: 22, borderRadius: '50%', cursor: 'pointer', flexShrink: 0,
              background: doneToday ? '#52c41a' : '#f0f0f0',
              border: doneToday ? 'none' : '1.5px solid #d9d9d9',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 11, color: '#fff',
            }}
          >{doneToday ? '✓' : ''}</div>
          <span style={{ fontSize: 13, flex: 1, textDecoration: doneToday ? 'line-through' : 'none', color: doneToday ? '#bbb' : '#222' }}>
            {habit.icon} {habit.name}
          </span>
          {streak > 0 && <span style={{ fontSize: 11, color: '#fa8c16' }}>🔥 {streak}d</span>}
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => {
            setEditOpen(true)
            editForm.setFieldsValue({ name: habit.name, icon: habit.icon })
          }} />
          <Tooltip title="Delete habit">
            <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate()} />
          </Tooltip>
        </div>
        <HeatmapStrip logs={monthLogs} days={days} />
      </div>

      <Modal title="Edit Habit" open={editOpen} onCancel={() => setEditOpen(false)} footer={null}>
        <Form form={editForm} layout="vertical" onFinish={values => editMutation.mutate(values)}>
          <Form.Item name="name" label="Habit name" rules={[{ required: true, message: 'Name is required' }, { max: 80 }]}>
            <Input />
          </Form.Item>
          <Form.Item name="icon" label="Icon (emoji)">
            <Input />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={editMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </>
  )
}

export function HealthPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { from, to, days } = getMonthRange()

  const { data: habits = [], isLoading } = useQuery({ queryKey: ['habits'], queryFn: getHabits })
  const { data: logs = [] } = useQuery({ queryKey: ['habit-logs', today], queryFn: () => getHabitLogs(today) })

  const monthLogsQueries = useQuery({
    queryKey: ['habit-month-logs', from, to, habits.map(h => h.id).join(',')],
    queryFn: async () => {
      const results = await Promise.all(habits.map(h => getHabitLogRange(h.id, from, to)))
      const map: Record<string, HabitLog[]> = {}
      habits.forEach((h, i) => { map[h.id] = results[i] })
      return map
    },
    enabled: habits.length > 0,
  })

  const addMutation = useMutation({
    mutationFn: createHabit,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['habits'] }); setAddOpen(false); form.resetFields() },
  })

  const toggleMutation = useMutation({
    mutationFn: (habitId: string) => toggleHabitLog(habitId, today),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['habit-logs', today] }),
  })

  const doneSet = new Set(logs.filter(l => l.done).map(l => l.habit_id))
  const donePct = habits.length ? Math.round(doneSet.size / habits.length * 100) : 0

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      <Row gutter={[12, 12]}>
        <Col span={16}>
          <Card
            size="small"
            title={`Today's Habits — ${donePct}% done`}
            extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}
          >
            {habits.map(h => (
              <HabitRow
                key={h.id}
                habit={h}
                doneToday={doneSet.has(h.id)}
                monthLogs={monthLogsQueries.data?.[h.id] ?? []}
                days={days}
                onToggle={() => toggleMutation.mutate(h.id)}
              />
            ))}
            {habits.length === 0 && (
              <div style={{ color: '#bbb', textAlign: 'center', padding: 20 }}>No habits yet. Add your first!</div>
            )}
          </Card>
        </Col>
      </Row>

      <Modal title="Add Habit" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="name" label="Habit name" rules={[{ required: true, message: 'Name is required' }, { max: 80 }]}>
            <Input />
          </Form.Item>
          <Form.Item name="icon" label="Icon (emoji)" initialValue="✓">
            <Input />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Fix the dynamic require in HabitRow** — replace the deleteMutation with a prop-based approach

The `require('../api/endpoints')` pattern won't work in ES modules. Refactor `HabitRow` to accept an `onDelete` prop:

Change the `HabitRow` component signature to:

```tsx
function HabitRow({ habit, doneToday, monthLogs, days, onToggle, onDelete }: {
  habit: Habit
  doneToday: boolean
  monthLogs: HabitLog[]
  days: string[]
  onToggle: () => void
  onDelete: () => void
}) {
```

Remove the `deleteMutation` and replace its usage with `onDelete`:

```tsx
          <Tooltip title="Delete habit">
            <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={onDelete} />
          </Tooltip>
```

In `HealthPage`, add a `deleteMutation` and pass it:

```tsx
  const deleteMutation = useMutation({
    mutationFn: deleteHabit,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['habits'] }),
  })
```

Update `HabitRow` usage in the map:

```tsx
            habits.map(h => (
              <HabitRow
                key={h.id}
                habit={h}
                doneToday={doneSet.has(h.id)}
                monthLogs={monthLogsQueries.data?.[h.id] ?? []}
                days={days}
                onToggle={() => toggleMutation.mutate(h.id)}
                onDelete={() => deleteMutation.mutate(h.id)}
              />
            ))
```

- [ ] **Commit**

```bash
git add frontend/src/pages/HealthPage.tsx
git commit -m "feat: habits — edit modal, month heatmap, streak counter"
```

---

## Task 6: Build Check + PR

- [ ] **Run lint and build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: no errors, build succeeds.

- [ ] **Create PR**

```bash
git push -u origin feat/goals-habits
gh pr create --title "feat: goals edit + KR management, habits edit + month heatmap + streak" --body "$(cat <<'EOF'
## Summary
- Goals: edit modal (name, description, target date, color, status), inline key result add/delete, auto-progress from KR completion, status visual treatment (completed badge, archived opacity)
- Habits: edit modal (name, icon), 30-day heatmap per habit, 🔥 streak counter
- New API calls: deleteKeyResult, updateHabit, getHabitLogRange

## Test plan
- [ ] `npm run lint && npm run build` passes
- [ ] Add a goal, add key results, check them off — progress bar updates
- [ ] Edit a goal, change status to completed — green checkmark appears
- [ ] Archive a goal — moves to bottom at 50% opacity
- [ ] Delete a key result
- [ ] Edit a habit name/icon
- [ ] Heatmap shows colored squares for completed days
- [ ] Streak counter increments on consecutive days

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
gh pr merge --auto --squash
```
