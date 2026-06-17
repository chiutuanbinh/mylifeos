import { useQuery } from '@tanstack/react-query'
import { Card, Spin } from 'antd'
import { getJournalNetWorth } from '../api/endpoints'

const fmtVND = (s: string) => {
  const n = parseFloat(s)
  return `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`
}

export function LiveNetWorthCard() {
  const { data, isLoading } = useQuery({
    queryKey: ['journal-networth'],
    queryFn: getJournalNetWorth,
    refetchInterval: 30_000,
  })

  return (
    <Card size="small">
      <div style={{ fontSize: 12, color: '#999' }}>Live Net Worth</div>
      {isLoading || !data
        ? <Spin />
        : <div style={{ fontSize: 28, fontWeight: 700, color: '#1677ff' }}>
            {fmtVND(data.net_worth)}
          </div>
      }
    </Card>
  )
}
