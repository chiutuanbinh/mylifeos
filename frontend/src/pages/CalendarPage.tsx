import { useState, useEffect, useCallback, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Button, Modal, Form, Input, Switch, Spin, Tooltip, message } from 'antd'
import { PlusOutlined, DeleteOutlined, LeftOutlined, RightOutlined, SyncOutlined } from '@ant-design/icons'
import { getEvents, createEvent, deleteEvent, syncGoogleCalendar } from '../api/endpoints'
import { supabase, getStoredProviderToken } from '../store/auth'

export function CalendarPage() {
  const todayDate = new Date()
  const [year, setYear] = useState(todayDate.getFullYear())
  const [month, setMonth] = useState(todayDate.getMonth())
  const [selectedDay, setSelectedDay] = useState(todayDate.getDate())
  const [addOpen, setAddOpen] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const fromDate = new Date(year, month, 1).toISOString()
  const toDate = new Date(year, month + 1, 0, 23, 59, 59).toISOString()

  const { data: events = [], isLoading } = useQuery({
    queryKey: ['events', year, month],
    queryFn: () => getEvents({ from: fromDate, to: toDate }),
  })

  // Track synced months so we don't re-sync on every render
  const syncedRef = useRef(new Set<string>())

  const syncGoogle = useCallback(async (from: string, to: string, manual = false) => {
    if (!supabase) return
    const key = `${from}/${to}`
    if (!manual && syncedRef.current.has(key)) return
    setSyncing(true)
    try {
      // Prefer stored token; fall back to live session (covers the first load
      // race where onAuthStateChange hasn't fired yet to persist the token).
      let providerToken = getStoredProviderToken()
      if (!providerToken) {
        const { data } = await supabase.auth.getSession()
        providerToken = data.session?.provider_token ?? null
        if (providerToken) {
          localStorage.setItem('gcal_provider_token', JSON.stringify({
            token: providerToken,
            expiresAt: Date.now() + 55 * 60 * 1000,
          }))
        }
      }
      if (!providerToken) {
        if (manual) message.warning('Google Calendar access expired — sign out and sign in again to re-grant access.')
        return
      }
      const result = await syncGoogleCalendar(providerToken, from, to)
      if (result.error) {
        if (manual) message.error(`Sync failed: ${result.error}`)
      } else {
        qc.invalidateQueries({ queryKey: ['events'] })
        if (manual) message.success(`Synced ${result.synced} events from Google Calendar`)
      }
      syncedRef.current.add(key)
    } catch (e: unknown) {
      if (manual) message.error(`Sync failed: ${e instanceof Error ? e.message : 'unknown error'}`)
    } finally {
      setSyncing(false)
    }
  }, [qc])

  // Auto-sync whenever month changes
  useEffect(() => {
    syncGoogle(fromDate, toDate)
  }, [fromDate, toDate, syncGoogle])

  const addMutation = useMutation({
    mutationFn: (values: Record<string, string | boolean>) => createEvent({
      title: values.title as string,
      start_at: new Date(`${values.date}T${values.start_time || '09:00'}`).toISOString(),
      end_at: new Date(`${values.date}T${values.end_time || '10:00'}`).toISOString(),
      color: (values.color as string) || '#1677ff',
      all_day: Boolean(values.all_day),
    }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['events'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteEvent,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['events'] }),
  })

  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const firstDayOfWeek = new Date(year, month, 1).getDay()
  const monthName = new Date(year, month).toLocaleString('default', { month: 'long' })

  const dayEvents = events.filter(e => {
    const d = new Date(e.start_at)
    return d.getDate() === selectedDay && d.getMonth() === month
  })

  const eventsByDay = events.reduce<Record<number, typeof events>>((acc, e) => {
    const d = new Date(e.start_at).getDate()
    acc[d] = acc[d] || []
    acc[d].push(e)
    return acc
  }, {})

  const prevMonth = () => { if (month === 0) { setMonth(11); setYear(y => y - 1) } else setMonth(m => m - 1) }
  const nextMonth = () => { if (month === 11) { setMonth(0); setYear(y => y + 1) } else setMonth(m => m + 1) }

  const onManualSync = () => {
    syncedRef.current.delete(`${fromDate}/${toDate}`)
    syncGoogle(fromDate, toDate, true)
  }

  return (
    <div>
      <Row gutter={[12, 12]}>
        <Col span={16}>
          <Card size="small" title={
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Button type="text" size="small" icon={<LeftOutlined />} onClick={prevMonth} />
              <span style={{ fontSize: 14, fontWeight: 600 }}>{monthName} {year}</span>
              <Button type="text" size="small" icon={<RightOutlined />} onClick={nextMonth} />
              {syncing && <SyncOutlined spin style={{ fontSize: 12, color: '#1677ff' }} />}
            </div>
          } extra={
            <div style={{ display: 'flex', gap: 6 }}>
              <Tooltip title="Re-sync from Google Calendar">
                <Button size="small" icon={<SyncOutlined />} onClick={onManualSync} loading={syncing} />
              </Tooltip>
              <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>
            </div>
          }>
            {isLoading ? <Spin /> : (
              <div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 1, marginBottom: 4 }}>
                  {['Sun','Mon','Tue','Wed','Thu','Fri','Sat'].map(d => (
                    <div key={d} style={{ textAlign: 'center', fontSize: 11, color: '#999', padding: '4px 0' }}>{d}</div>
                  ))}
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 1 }}>
                  {Array.from({ length: firstDayOfWeek }).map((_, i) => <div key={`e${i}`} />)}
                  {Array.from({ length: daysInMonth }, (_, i) => i + 1).map(day => {
                    const isToday = day === todayDate.getDate() && month === todayDate.getMonth() && year === todayDate.getFullYear()
                    const isSelected = day === selectedDay
                    const hasEvents = !!eventsByDay[day]?.length
                    return (
                      <div key={day} onClick={() => setSelectedDay(day)} style={{ textAlign: 'center', padding: '6px 2px', cursor: 'pointer', borderRadius: 4, background: isSelected ? '#1677ff' : isToday ? '#e6f4ff' : 'transparent', color: isSelected ? '#fff' : isToday ? '#1677ff' : '#222', fontSize: 13, position: 'relative' }}>
                        {day}
                        {hasEvents && <div style={{ width: 4, height: 4, borderRadius: '50%', background: isSelected ? '#fff' : '#1677ff', margin: '2px auto 0' }} />}
                      </div>
                    )
                  })}
                </div>
              </div>
            )}
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title={<span style={{ fontSize: 13 }}>Events — {monthName} {selectedDay}</span>}>
            {dayEvents.length === 0 && <div style={{ color: '#bbb', textAlign: 'center', padding: 20, fontSize: 12 }}>No events this day.</div>}
            {dayEvents.map(e => (
              <div key={e.id} style={{ display: 'flex', gap: 8, padding: '6px 0', borderBottom: '1px solid #f5f5f5', alignItems: 'flex-start' }}>
                <div style={{ width: 3, height: 36, background: e.color, borderRadius: 2, flexShrink: 0, marginTop: 2 }} />
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 12, fontWeight: 500 }}>{e.title}</div>
                  <div style={{ fontSize: 11, color: '#bbb' }}>{e.all_day ? 'All day' : new Date(e.start_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</div>
                </div>
                {!e.google_event_id && (
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(e.id)} />
                )}
              </div>
            ))}
          </Card>
        </Col>
      </Row>

      <Modal title="Add Event" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="title" label="Title" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="all_day" label="All day" valuePropName="checked" initialValue={false}><Switch /></Form.Item>
          <Form.Item name="start_time" label="Start time" initialValue="09:00"><Input type="time" /></Form.Item>
          <Form.Item name="end_time" label="End time" initialValue="10:00"><Input type="time" /></Form.Item>
          <Form.Item name="color" label="Color" initialValue="#1677ff"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
