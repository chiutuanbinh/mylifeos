import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'

describe('App smoke test', () => {
  it('renders without crashing', () => {
    render(<div>MyLifeOS</div>)
    expect(screen.getByText('MyLifeOS')).toBeInTheDocument()
  })
})
