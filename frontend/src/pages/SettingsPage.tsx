import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, Switch, Row, Col, Spin, message, Tag, Space } from 'antd'
import { BellOutlined } from '@ant-design/icons'
import { getSettings, updateSettings, getGoals } from '../api/endpoints'

const MODULES = ['finance', 'health', 'goals', 'notes', 'calendar', 'inventory']
const NOTIF_KEYS = ['email', 'push']

export function SettingsPage() {
  const qc = useQueryClient()
  const { data: settings, isLoading } = useQuery({ queryKey: ['settings'], queryFn: getSettings })
  const { data: goals = [] } = useQuery({ queryKey: ['goals'], queryFn: getGoals })

  const updateMutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['settings'] }); message.success('Settings saved') },
  })

  const remindersKRs = goals.flatMap(g =>
    (g.key_results ?? [])
      .filter(kr => kr.recurring && kr.reminder_time)
      .map(kr => ({ ...kr, goalName: g.name }))
  )

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />
  if (!settings) return null

  return (
    <Row gutter={[12, 12]}>
      <Col span={12}>
        <Card size="small" title="Notifications">
          {NOTIF_KEYS.map(key => (
            <div key={key} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
              <span style={{ fontSize: 13, textTransform: 'capitalize' }}>{key} notifications</span>
              <Switch
                checked={!!settings.notifications[key]}
                onChange={checked => updateMutation.mutate({ notifications: { ...settings.notifications, [key]: checked } })}
              />
            </div>
          ))}
        </Card>
      </Col>
      <Col span={12}>
        <Card size="small" title="Modules">
          {MODULES.map(mod => (
            <div key={mod} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
              <span style={{ fontSize: 13, textTransform: 'capitalize' }}>{mod}</span>
              <Switch
                checked={!!settings.modules_enabled[mod]}
                onChange={checked => updateMutation.mutate({ modules_enabled: { ...settings.modules_enabled, [mod]: checked } })}
              />
            </div>
          ))}
        </Card>
      </Col>
      <Col span={24}>
        <Card size="small" title={
          <Space>
            <BellOutlined />
            <span>Reminders</span>
            <Tag color="default" style={{ fontSize: 11 }}>Coming soon</Tag>
          </Space>
        }>
          {remindersKRs.length === 0 && (
            <div style={{ color: '#bbb', fontSize: 12, padding: '8px 0' }}>
              No reminders set. Add a reminder time when creating a recurring key result.
            </div>
          )}
          <Row gutter={[12, 12]} style={{ marginTop: 8 }}>
            {remindersKRs.map(kr => (
              <Col span={6} key={kr.id}>
                <div style={{
                  border: '1px solid #e8e8e8', borderRadius: 12, padding: '10px 14px',
                  background: '#fafafa', fontFamily: 'system-ui',
                }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                    <Space size={4}>
                      <BellOutlined style={{ fontSize: 11, color: '#1677ff' }} />
                      <span style={{ fontSize: 11, fontWeight: 600, color: '#333' }}>MyLifeOS</span>
                    </Space>
                    <span style={{ fontSize: 10, color: '#999' }}>{kr.reminder_time}</span>
                  </div>
                  <div style={{ fontSize: 11, fontWeight: 600, color: '#111', marginBottom: 2 }}>{kr.goalName}</div>
                  <div style={{ fontSize: 11, color: '#555' }}>Time to: {kr.description}</div>
                </div>
              </Col>
            ))}
          </Row>
          <div style={{ fontSize: 11, color: '#bbb', marginTop: 8 }}>
            Push notifications will activate when the mobile app is available.
          </div>
        </Card>
      </Col>
    </Row>
  )
}
