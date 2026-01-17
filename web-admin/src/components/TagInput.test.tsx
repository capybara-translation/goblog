import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TagInput } from './TagInput';
import { apiClient } from '../api/client';

// apiClient をモック
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
      expect(screen.getByPlaceholderText('タグを入力...')).toBeInTheDocument();
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
        expect(screen.getByRole('textbox')).not.toHaveAttribute('placeholder', 'タグを入力...');
      });
    });

    it('should render custom placeholder', async () => {
      const onChange = vi.fn();
      render(<TagInput value="" onChange={onChange} placeholder="カスタムプレースホルダー" />);

      expect(screen.getByPlaceholderText('カスタムプレースホルダー')).toBeInTheDocument();
    });

    it('should render disabled state', async () => {
      const onChange = vi.fn();
      render(<TagInput value="Go" onChange={onChange} disabled />);

      expect(screen.getByRole('textbox')).toBeDisabled();
      // 削除ボタンが表示されないことを確認
      expect(screen.queryByLabelText('Goを削除')).not.toBeInTheDocument();
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

      const removeButton = screen.getByLabelText('Goを削除');
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
        expect(screen.getByText('5件')).toBeInTheDocument(); // Go の件数
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

      // Goのリストアイテムをクリック（件数表示を含むので親要素を選択）
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

      // 最初の項目が選択される
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

      // 2回下に移動してから1回上に移動
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

      // 外側をクリック
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

      // IME変換開始
      fireEvent.compositionStart(input);
      fireEvent.change(input, { target: { value: 'にほんご' } });

      // IME変換中にEnterを押しても何も起きない（変換確定になる）
      fireEvent.keyDown(input, { key: 'Enter' });

      expect(onChange).not.toHaveBeenCalled();

      // IME変換終了
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

      // タグチップがないことを確認
      expect(screen.queryByLabelText(/を削除/)).not.toBeInTheDocument();
    });

    it('should handle whitespace-only tags', async () => {
      const onChange = vi.fn();
      render(<TagInput value="Go,   , React" onChange={onChange} />);

      await waitFor(() => {
        // 空白のみのタグは除外される
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
