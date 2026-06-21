import { useSearchParams } from 'react-router-dom'
import { useCallback } from 'react'

export function useTabParam(defaultKey: string, validKeys?: string[]): [string, (key: string) => void] {
  const [params, setParams] = useSearchParams()
  const raw = params.get('tab')
  const activeKey = (raw && (!validKeys || validKeys.includes(raw))) ? raw : defaultKey

  const setActiveKey = useCallback((key: string) => {
    setParams(prev => {
      const next = new URLSearchParams(prev)
      next.set('tab', key)
      return next
    }, { replace: true })
  }, [setParams])

  return [activeKey, setActiveKey]
}
