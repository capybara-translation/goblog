import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusBadge } from './StatusBadge'

describe('StatusBadge', () => {
  describe('Draft status', () => {
    it('should display "下書き" when status is draft', () => {
      render(<StatusBadge status="draft" />)
      expect(screen.getByText('下書き')).toBeInTheDocument()
    })

    it('should not display "公開済み" when status is draft', () => {
      render(<StatusBadge status="draft" />)
      expect(screen.queryByText('公開済み')).not.toBeInTheDocument()
    })
  })

  describe('Published status', () => {
    it('should display "公開済み" when status is published', () => {
      render(<StatusBadge status="published" />)
      expect(screen.getByText('公開済み')).toBeInTheDocument()
    })

    it('should not display "下書き" when status is published', () => {
      render(<StatusBadge status="published" />)
      expect(screen.queryByText('下書き')).not.toBeInTheDocument()
    })
  })

  describe('Rendering', () => {
    it('should render as a span element', () => {
      const { container } = render(<StatusBadge status="draft" />)
      const badge = container.querySelector('span')
      expect(badge).toBeInTheDocument()
    })
  })
})
