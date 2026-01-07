import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TagList } from './TagList'

describe('TagList', () => {
  describe('Empty tags', () => {
    it('should display "タグなし" when tags is empty string', () => {
      render(<TagList tags="" />)
      expect(screen.getByText('タグなし')).toBeInTheDocument()
    })

    it('should display "タグなし" when tags is only whitespace', () => {
      render(<TagList tags="   " />)
      expect(screen.getByText('タグなし')).toBeInTheDocument()
    })
  })

  describe('Single tag', () => {
    it('should display a single tag', () => {
      render(<TagList tags="React" />)
      expect(screen.getByText('React')).toBeInTheDocument()
    })

    it('should trim whitespace from single tag', () => {
      render(<TagList tags="  React  " />)
      expect(screen.getByText('React')).toBeInTheDocument()
      expect(screen.queryByText('  React  ')).not.toBeInTheDocument()
    })

    it('should not display "タグなし" when single tag exists', () => {
      render(<TagList tags="React" />)
      expect(screen.queryByText('タグなし')).not.toBeInTheDocument()
    })
  })

  describe('Multiple tags', () => {
    it('should display multiple tags separated by comma', () => {
      render(<TagList tags="React,TypeScript,Testing" />)
      expect(screen.getByText('React')).toBeInTheDocument()
      expect(screen.getByText('TypeScript')).toBeInTheDocument()
      expect(screen.getByText('Testing')).toBeInTheDocument()
    })

    it('should trim whitespace from each tag', () => {
      render(<TagList tags="React , TypeScript , Testing" />)
      expect(screen.getByText('React')).toBeInTheDocument()
      expect(screen.getByText('TypeScript')).toBeInTheDocument()
      expect(screen.getByText('Testing')).toBeInTheDocument()
    })

    it('should filter out empty tags', () => {
      render(<TagList tags="React,,TypeScript,,,Testing" />)
      expect(screen.getByText('React')).toBeInTheDocument()
      expect(screen.getByText('TypeScript')).toBeInTheDocument()
      expect(screen.getByText('Testing')).toBeInTheDocument()
    })

    it('should filter out whitespace-only tags', () => {
      render(<TagList tags="React,  ,TypeScript,   ,Testing" />)
      expect(screen.getByText('React')).toBeInTheDocument()
      expect(screen.getByText('TypeScript')).toBeInTheDocument()
      expect(screen.getByText('Testing')).toBeInTheDocument()
    })
  })

  describe('Tag with special characters', () => {
    it('should handle tags with spaces in the name', () => {
      render(<TagList tags="Web Development,Machine Learning" />)
      expect(screen.getByText('Web Development')).toBeInTheDocument()
      expect(screen.getByText('Machine Learning')).toBeInTheDocument()
    })

    it('should handle tags with numbers', () => {
      render(<TagList tags="React 18,ES2023" />)
      expect(screen.getByText('React 18')).toBeInTheDocument()
      expect(screen.getByText('ES2023')).toBeInTheDocument()
    })

    it('should handle tags with hyphens and underscores', () => {
      render(<TagList tags="web-dev,user_experience" />)
      expect(screen.getByText('web-dev')).toBeInTheDocument()
      expect(screen.getByText('user_experience')).toBeInTheDocument()
    })
  })

  describe('Edge cases', () => {
    it('should handle trailing comma', () => {
      render(<TagList tags="React,TypeScript," />)
      expect(screen.getByText('React')).toBeInTheDocument()
      expect(screen.getByText('TypeScript')).toBeInTheDocument()
    })

    it('should handle leading comma', () => {
      render(<TagList tags=",React,TypeScript" />)
      expect(screen.getByText('React')).toBeInTheDocument()
      expect(screen.getByText('TypeScript')).toBeInTheDocument()
    })

    it('should handle only commas', () => {
      render(<TagList tags=",,," />)
      // 空のタグはフィルタリングされるため、何も表示されない
      expect(screen.queryByText('タグなし')).not.toBeInTheDocument()
    })

    it('should handle mix of empty and whitespace tags', () => {
      render(<TagList tags="React,  ,,   ,TypeScript" />)
      expect(screen.getByText('React')).toBeInTheDocument()
      expect(screen.getByText('TypeScript')).toBeInTheDocument()
    })
  })
})
