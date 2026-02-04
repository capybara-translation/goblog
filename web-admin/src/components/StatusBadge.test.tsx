import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusBadge } from './StatusBadge'

describe('StatusBadge', () => {
  describe('Draft status', () => {
    it('should display "Draft" when status is draft', () => {
      render(<StatusBadge status="draft" />)
      expect(screen.getByText('Draft')).toBeInTheDocument()
    })

    it('should not display "Published" when status is draft', () => {
      render(<StatusBadge status="draft" />)
      expect(screen.queryByText('Published')).not.toBeInTheDocument()
    })
  })

  describe('Published status', () => {
    it('should display "Published" when status is published', () => {
      render(<StatusBadge status="published" />)
      expect(screen.getByText('Published')).toBeInTheDocument()
    })

    it('should not display "Draft" when status is published', () => {
      render(<StatusBadge status="published" />)
      expect(screen.queryByText('Draft')).not.toBeInTheDocument()
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
