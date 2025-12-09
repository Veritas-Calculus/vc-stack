import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import App from '@/App'

describe('app smoke', () => {
  it('renders login when unauthenticated', () => {
    const { getByRole } = render(
      <BrowserRouter>
        <App />
      </BrowserRouter>
    )
    expect(getByRole('heading', { name: 'Sign in to VC Console' })).toBeInTheDocument()
  })
})
