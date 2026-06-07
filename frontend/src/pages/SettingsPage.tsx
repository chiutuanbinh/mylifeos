import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, Switch, Row, Col, Spin, message } from 'antd'
import { getSettings, updateSettings } from '../api/endpoints'

const MODULES = ['finance', 'health', 'goals', 'notes', 'calendar', 'inventory']
const NOTIF_KEYS = ['email', 'push']

export function SettingsPage() {
  const qc = useQueryClient()
  const { data: settings, isLoading } = useQuery({ queryKey: ['settings'], queryFn: getSettings })

  const updateMutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['settings'] }); message.success('Settings saved') },
  })

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
    </Row>
  )
}
