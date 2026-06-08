import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { ReactionTypeList } from './ReactionTypeList';
import { apiClient } from '../api/client';
import type { ReactionType } from '../api/client';

// Mirror the PostList pattern: mock the apiClient module directly
vi.mock('../api/client', () => ({
  apiClient: {
    getReactionTypes: vi.fn(),
    createReactionType: vi.fn(),
    updateReactionType: vi.fn(),
    deleteReactionType: vi.fn(),
  },
}));

const seedRow: ReactionType = {
  id: 1,
  emoji: '👍',
  label: 'いいね',
  sort_order: 10,
  is_active: true,
  is_seed: true,
};

const customRow: ReactionType = {
  id: 7,
  emoji: '🎉',
  label: 'やったー',
  sort_order: 20,
  is_active: true,
  is_seed: false,
};

describe('ReactionTypeList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Loading and data rendering', () => {
    it('shows loading indicator initially', () => {
      vi.mocked(apiClient.getReactionTypes).mockReturnValue(new Promise(() => {}));
      render(<ReactionTypeList />);
      expect(screen.getByText('読み込み中...')).toBeInTheDocument();
    });

    it('renders the seeded rows from the API', async () => {
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([seedRow]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('いいね')).toBeInTheDocument());
      expect(screen.getByText('👍')).toBeInTheDocument();
    });

    it('shows error when load fails', async () => {
      vi.mocked(apiClient.getReactionTypes).mockRejectedValue(new Error('fetch failed'));
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('fetch failed')).toBeInTheDocument());
    });

    it('shows default error for non-Error rejection', async () => {
      vi.mocked(apiClient.getReactionTypes).mockRejectedValue('oops');
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('failed to load')).toBeInTheDocument());
    });

    it('dims inactive rows', async () => {
      const inactive: ReactionType = { ...customRow, is_active: false };
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([inactive]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('やったー')).toBeInTheDocument());
      const row = screen.getByText('やったー').closest('tr');
      expect(row).toHaveClass('opacity-50');
    });

    it('shows 無効 for inactive rows', async () => {
      const inactive: ReactionType = { ...customRow, is_active: false };
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([inactive]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('無効')).toBeInTheDocument());
    });
  });

  describe('Create modal', () => {
    it('opens the create modal on 新規追加 click', async () => {
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([seedRow]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('いいね')).toBeInTheDocument());
      fireEvent.click(screen.getByRole('button', { name: /新規追加/ }));
      expect(screen.getByLabelText(/絵文字/)).toBeInTheDocument();
    });

    it('closes the modal on キャンセル click', async () => {
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([seedRow]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('いいね')).toBeInTheDocument());
      fireEvent.click(screen.getByRole('button', { name: /新規追加/ }));
      expect(screen.getByLabelText(/絵文字/)).toBeInTheDocument();
      fireEvent.click(screen.getByRole('button', { name: /キャンセル/ }));
      expect(screen.queryByLabelText(/絵文字/)).not.toBeInTheDocument();
    });

    it('calls createReactionType and reloads on save', async () => {
      vi.mocked(apiClient.getReactionTypes)
        .mockResolvedValueOnce([seedRow])
        .mockResolvedValueOnce([seedRow, customRow]);
      vi.mocked(apiClient.createReactionType).mockResolvedValue(customRow);

      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('いいね')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /新規追加/ }));

      fireEvent.change(screen.getByLabelText(/絵文字/), { target: { value: '🎉' } });
      fireEvent.change(screen.getByLabelText(/ラベル/), { target: { value: 'やったー' } });
      fireEvent.change(screen.getByLabelText(/並び順/), { target: { value: '20' } });

      fireEvent.click(screen.getByRole('button', { name: /保存/ }));

      await waitFor(() =>
        expect(apiClient.createReactionType).toHaveBeenCalledWith({
          emoji: '🎉',
          label: 'やったー',
          sort_order: 20,
        }),
      );
      // After reload the new row should appear
      await waitFor(() => expect(screen.getByText('やったー')).toBeInTheDocument());
    });
  });

  describe('Edit modal', () => {
    it('opens the edit modal with pre-filled values on 編集 click', async () => {
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([customRow]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('やったー')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /編集/ }));

      expect((screen.getByLabelText(/絵文字/) as HTMLInputElement).value).toBe('🎉');
      expect((screen.getByLabelText(/ラベル/) as HTMLInputElement).value).toBe('やったー');
    });

    it('emoji field is disabled for seed rows in the edit modal', async () => {
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([seedRow]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('いいね')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /編集/ }));

      expect(screen.getByLabelText(/絵文字/)).toBeDisabled();
    });

    it('calls updateReactionType and reloads on save', async () => {
      const updated: ReactionType = { ...customRow, label: '最高！' };
      vi.mocked(apiClient.getReactionTypes)
        .mockResolvedValueOnce([customRow])
        .mockResolvedValueOnce([updated]);
      vi.mocked(apiClient.updateReactionType).mockResolvedValue(updated);

      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('やったー')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /編集/ }));
      fireEvent.change(screen.getByLabelText(/ラベル/), { target: { value: '最高！' } });
      fireEvent.click(screen.getByRole('button', { name: /保存/ }));

      await waitFor(() =>
        expect(apiClient.updateReactionType).toHaveBeenCalledWith(7, {
          emoji: '🎉',
          label: '最高！',
          sort_order: 20,
          is_active: true,
        }),
      );
      await waitFor(() => expect(screen.getByText('最高！')).toBeInTheDocument());
    });
  });

  describe('Validation', () => {
    it('shows error and does not call createReactionType when emoji and label are empty', async () => {
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([seedRow]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('いいね')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /新規追加/ }));

      // Leave emoji and label empty, just click 保存
      fireEvent.click(screen.getByRole('button', { name: /保存/ }));

      expect(apiClient.createReactionType).not.toHaveBeenCalled();
      expect(screen.getByText('絵文字とラベルは必須です')).toBeInTheDocument();
    });

    it('shows the API error message when createReactionType rejects', async () => {
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([seedRow]);
      vi.mocked(apiClient.createReactionType).mockRejectedValue(
        new Error('この絵文字は既に登録されています'),
      );

      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('いいね')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /新規追加/ }));
      fireEvent.change(screen.getByLabelText(/絵文字/), { target: { value: '🎉' } });
      fireEvent.change(screen.getByLabelText(/ラベル/), { target: { value: 'やったー' } });
      fireEvent.click(screen.getByRole('button', { name: /保存/ }));

      await waitFor(() =>
        expect(screen.getByText('この絵文字は既に登録されています')).toBeInTheDocument(),
      );
    });
  });

  describe('is_active toggle', () => {
    it('calls updateReactionType with is_active: false when the checkbox is toggled off', async () => {
      const updated: ReactionType = { ...customRow, is_active: false };
      vi.mocked(apiClient.getReactionTypes)
        .mockResolvedValueOnce([customRow])
        .mockResolvedValueOnce([updated]);
      vi.mocked(apiClient.updateReactionType).mockResolvedValue(updated);

      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('やったー')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /編集/ }));

      // The 有効 checkbox should be checked initially
      const checkbox = screen.getByLabelText(/有効/) as HTMLInputElement;
      expect(checkbox.checked).toBe(true);

      // Toggle it off
      fireEvent.click(checkbox);
      expect(checkbox.checked).toBe(false);

      fireEvent.click(screen.getByRole('button', { name: /保存/ }));

      await waitFor(() =>
        expect(apiClient.updateReactionType).toHaveBeenCalledWith(7, {
          emoji: '🎉',
          label: 'やったー',
          sort_order: 20,
          is_active: false,
        }),
      );
    });
  });

  describe('Delete', () => {
    it('delete button is disabled for seed rows', async () => {
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([seedRow]);
      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('いいね')).toBeInTheDocument());
      expect(screen.getByRole('button', { name: /削除/ })).toBeDisabled();
    });

    it('calls deleteReactionType and reloads after confirmation', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(true);
      vi.mocked(apiClient.getReactionTypes)
        .mockResolvedValueOnce([customRow])
        .mockResolvedValueOnce([]);
      vi.mocked(apiClient.deleteReactionType).mockResolvedValue(undefined);

      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('やったー')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /削除/ }));

      await waitFor(() =>
        expect(apiClient.deleteReactionType).toHaveBeenCalledWith(7),
      );
      await waitFor(() => expect(screen.queryByText('やったー')).not.toBeInTheDocument());
    });

    it('does NOT call deleteReactionType when confirm is cancelled', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(false);
      vi.mocked(apiClient.getReactionTypes).mockResolvedValue([customRow]);

      render(<ReactionTypeList />);
      await waitFor(() => expect(screen.getByText('やったー')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: /削除/ }));

      expect(apiClient.deleteReactionType).not.toHaveBeenCalled();
    });
  });
});
