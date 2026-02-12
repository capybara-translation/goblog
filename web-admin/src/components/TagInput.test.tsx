import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TagInput } from './TagInput';
import { apiClient } from '../api/client';

// Mock apiClient
vi.mock('../api/client', () => ({
  apiClient: {
    getTags: vi.fn(),
  },
}));

describe('TagInput', () => {
  const mockTags = [
    { name: 'Go', count: 5 },
    { name: 'React', count: 3 },
    { name: 'TypeScript', count: 4 },
    { name: 'Testing', count: 2 },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(apiClient.getTags).mockResolvedValue(mockTags);
  });

  describe('Rendering', () => {
    it('should render empty state correctly', async () => {
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      expect(screen.getByRole('textbox')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('Enter tags...')).toBeInTheDocument();
    });

    it('should render existing tags as chips', async () => {
      const onChange = vi.fn();
      render(<TagInput value="Go, React" onChange={onChange} />);

      await waitFor(() => {
        expect(screen.getByText('Go')).toBeInTheDocument();
        expect(screen.getByText('React')).toBeInTheDocument();
      });
    });

    it('should not show placeholder when tags exist', async () => {
      const onChange = vi.fn();
      render(<TagInput value="Go" onChange={onChange} />);

      await waitFor(() => {
        expect(screen.getByRole('textbox')).not.toHaveAttribute('placeholder', 'Enter tags...');
      });
    });

    it('should render custom placeholder', async () => {
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} placeholder="Custom placeholder" />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      expect(screen.getByPlaceholderText('Custom placeholder')).toBeInTheDocument();
    });

    it('should render disabled state', async () => {
      const onChange = vi.fn();
      render(<TagInput value="Go" onChange={onChange} disabled />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      expect(screen.getByRole('textbox')).toBeDisabled();
      // Verify remove button is not displayed
      expect(screen.queryByLabelText('Remove Go')).not.toBeInTheDocument();
    });
  });

  describe('Tag Chip Operations', () => {
    it('should remove tag when clicking X button', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="Go, React" onChange={onChange} />);

      await waitFor(() => {
        expect(screen.getByText('Go')).toBeInTheDocument();
      });

      const removeButton = screen.getByLabelText('Remove Go');
      await user.click(removeButton);

      expect(onChange).toHaveBeenCalledWith('React');
    });

    it('should remove last tag when pressing Backspace on empty input', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="Go, React" onChange={onChange} />);

      await waitFor(() => {
        expect(screen.getByText('React')).toBeInTheDocument();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);
      await user.keyboard('{Backspace}');

      expect(onChange).toHaveBeenCalledWith('Go');
    });
  });

  describe('Autocomplete', () => {
    it('should fetch tags on mount', async () => {
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalledTimes(1);
      });
    });

    it('should show suggestions when input is focused', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
        expect(screen.getByText('(5)')).toBeInTheDocument(); // Go tag count
      });
    });

    it('should filter suggestions based on input', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.type(input, 'Te');

      await waitFor(() => {
        expect(screen.getByText('Testing')).toBeInTheDocument();
        expect(screen.queryByText('Go')).not.toBeInTheDocument();
      });
    });

    it('should exclude already selected tags from suggestions', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="Go" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);

      await waitFor(() => {
        expect(screen.queryByRole('option', { name: /^Go/ })).not.toBeInTheDocument();
        expect(screen.getByText('React')).toBeInTheDocument();
      });
    });

    it('should add tag when clicking suggestion', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);

      await waitFor(() => {
        expect(screen.getByText('Go')).toBeInTheDocument();
      });

      // Click Go list item (select parent element since it includes count display)
      const goOption = screen.getByRole('option', { name: /Go/ });
      await user.click(goOption);

      expect(onChange).toHaveBeenCalledWith('Go');
    });
  });

  describe('Keyboard Navigation', () => {
    it('should move selection down with ArrowDown', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
      });

      await user.keyboard('{ArrowDown}');

      // First item is selected
      const firstOption = screen.getByRole('option', { name: /Go/ });
      expect(firstOption).toHaveAttribute('aria-selected', 'true');
    });

    it('should move selection up with ArrowUp', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
      });

      // Move down twice then move up once
      await user.keyboard('{ArrowDown}{ArrowDown}{ArrowUp}');

      const firstOption = screen.getByRole('option', { name: /Go/ });
      expect(firstOption).toHaveAttribute('aria-selected', 'true');
    });

    it('should select highlighted suggestion with Enter', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
      });

      await user.keyboard('{ArrowDown}{Enter}');

      expect(onChange).toHaveBeenCalledWith('Go');
    });

    it('should close dropdown with Escape', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
      });

      await user.keyboard('{Escape}');

      await waitFor(() => {
        expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
      });
    });

    it('should add custom tag with Enter when no suggestion is selected', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.type(input, 'NewTag');
      await user.keyboard('{Enter}');

      expect(onChange).toHaveBeenCalledWith('NewTag');
    });
  });

  describe('Comma Input', () => {
    it('should add tag when comma is typed', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.type(input, 'NewTag,');

      expect(onChange).toHaveBeenCalledWith('NewTag');
    });
  });

  describe('Duplicate Prevention', () => {
    it('should not add duplicate tags', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<TagInput value="Go" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.type(input, 'Go{Enter}');

      // onChange should not be called with duplicate
      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe('Click Outside', () => {
    it('should close dropdown when clicking outside', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(
        <div>
          <TagInput value="" onChange={onChange} />
          <button>Outside</button>
        </div>
      );

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');
      await user.click(input);

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
      });

      // Click outside
      await user.click(screen.getByText('Outside'));

      await waitFor(() => {
        expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
      });
    });
  });

  describe('IME Handling', () => {
    it('should not trigger keyboard handlers during IME composition', async () => {
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      const input = screen.getByRole('textbox');

      // Start IME composition
      fireEvent.compositionStart(input);
      fireEvent.change(input, { target: { value: 'にほんご' } });

      // Pressing Enter during IME composition does nothing (becomes composition commit)
      fireEvent.keyDown(input, { key: 'Enter' });

      expect(onChange).not.toHaveBeenCalled();

      // End IME composition
      fireEvent.compositionEnd(input);
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty value', async () => {
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(apiClient.getTags).toHaveBeenCalled();
      });

      // Verify no tag chips exist
      expect(screen.queryByLabelText(/Remove/)).not.toBeInTheDocument();
    });

    it('should handle whitespace-only tags', async () => {
      const onChange = vi.fn();
      render(<TagInput value="Go,   , React" onChange={onChange} />);

      await waitFor(() => {
        // Whitespace-only tags are filtered out
        expect(screen.getByText('Go')).toBeInTheDocument();
        expect(screen.getByText('React')).toBeInTheDocument();
      });
    });

    it('should trim tag names', async () => {
      const onChange = vi.fn();
      render(<TagInput value="  Go  ,  React  " onChange={onChange} />);

      await waitFor(() => {
        expect(screen.getByText('Go')).toBeInTheDocument();
        expect(screen.getByText('React')).toBeInTheDocument();
      });
    });

    it('should handle API error gracefully', async () => {
      vi.mocked(apiClient.getTags).mockRejectedValue(new Error('API Error'));
      const onChange = vi.fn();
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      render(<TagInput value="" onChange={onChange} />);

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith('Failed to load tags:', expect.any(Error));
      });

      consoleSpy.mockRestore();
    });
  });
});
