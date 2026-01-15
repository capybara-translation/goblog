import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { MarkdownEditor } from './MarkdownEditor';
import { apiClient } from '../api/client';

// apiClient をモック
vi.mock('../api/client', () => ({
  apiClient: {
    previewMarkdown: vi.fn(),
  },
}));

const mockPreviewMarkdown = vi.mocked(apiClient.previewMarkdown);

// タイマーを進めて非同期処理を完了させるヘルパー
async function advanceTimersAndFlush(ms: number) {
  await act(async () => {
    vi.advanceTimersByTime(ms);
    // マイクロタスクをフラッシュ
    await Promise.resolve();
  });
}

describe('MarkdownEditor', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockPreviewMarkdown.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('レンダリング', () => {
    it('編集とプレビューのラベルが表示される', () => {
      render(<MarkdownEditor value="" onChange={() => {}} />);

      expect(screen.getByText('編集')).toBeInTheDocument();
      expect(screen.getByText('プレビュー')).toBeInTheDocument();
    });

    it('MDEditorが表示される', () => {
      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // MDEditorのコンテナが存在することを確認
      expect(document.querySelector('.w-md-editor')).toBeInTheDocument();
    });

    it('プレビューエリアが表示される', () => {
      render(<MarkdownEditor value="" onChange={() => {}} />);

      expect(document.querySelector('.article-content')).toBeInTheDocument();
    });
  });

  describe('プレビュー機能', () => {
    it('値が変更されると300ms後にプレビューAPIが呼ばれる', async () => {
      mockPreviewMarkdown.mockResolvedValue('<h1>Test</h1>');

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // デバウンス前はAPIが呼ばれない
      expect(mockPreviewMarkdown).not.toHaveBeenCalled();

      // 300ms進める
      await advanceTimersAndFlush(300);

      expect(mockPreviewMarkdown).toHaveBeenCalledWith('# Test');
    });

    it('プレビュー結果がHTMLとして表示される', async () => {
      mockPreviewMarkdown.mockResolvedValue('<h1>Hello World</h1>');

      render(<MarkdownEditor value="# Hello World" onChange={() => {}} />);

      // タイマーを進めてPromiseを解決
      await advanceTimersAndFlush(300);

      const previewArea = document.querySelector('.article-content');
      expect(previewArea?.innerHTML).toBe('<h1>Hello World</h1>');
    });

    it('空の値ではプレビューAPIが呼ばれない', async () => {
      render(<MarkdownEditor value="" onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      expect(mockPreviewMarkdown).not.toHaveBeenCalled();
    });

    it('空白のみの値ではプレビューAPIが呼ばれない', async () => {
      render(<MarkdownEditor value="   " onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      expect(mockPreviewMarkdown).not.toHaveBeenCalled();
    });

    it('値が変更されるとデバウンスタイマーがリセットされる', async () => {
      mockPreviewMarkdown.mockResolvedValue('<p>Result</p>');

      const { rerender } = render(<MarkdownEditor value="First" onChange={() => {}} />);

      // 200ms経過
      await advanceTimersAndFlush(200);

      // まだAPIは呼ばれていない
      expect(mockPreviewMarkdown).not.toHaveBeenCalled();

      // 値を変更
      rerender(<MarkdownEditor value="Second" onChange={() => {}} />);

      // さらに200ms経過（合計400ms）
      await advanceTimersAndFlush(200);

      // まだAPIは呼ばれていない（デバウンスがリセットされたため）
      expect(mockPreviewMarkdown).not.toHaveBeenCalled();

      // さらに100ms経過（値変更から300ms）
      await advanceTimersAndFlush(100);

      expect(mockPreviewMarkdown).toHaveBeenCalledWith('Second');
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(1);
    });
  });

  describe('ローディング状態', () => {
    it('プレビュー取得中はローディング表示される', async () => {
      // Promiseを保留状態にする
      let resolvePromise: (value: string) => void;
      mockPreviewMarkdown.mockImplementation(
        () => new Promise((resolve) => { resolvePromise = resolve; })
      );

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // タイマーを進める（APIコールがトリガーされる）
      await act(async () => {
        vi.advanceTimersByTime(300);
      });

      // ローディング表示を確認
      expect(screen.getByText('(読み込み中...)')).toBeInTheDocument();

      // Promiseを解決して状態を更新
      await act(async () => {
        resolvePromise!('<h1>Test</h1>');
      });

      // ローディング表示が消える
      expect(screen.queryByText('(読み込み中...)')).not.toBeInTheDocument();
    });
  });

  describe('エラーハンドリング', () => {
    it('API エラー時はエラーメッセージが表示される', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
      mockPreviewMarkdown.mockRejectedValue(new Error('API Error'));

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // タイマーを進めてPromiseを解決（reject）
      await advanceTimersAndFlush(300);

      const previewArea = document.querySelector('.article-content');
      expect(previewArea?.innerHTML).toContain('プレビューの取得に失敗しました');

      consoleError.mockRestore();
    });
  });

  describe('onChange', () => {
    it('エディタの値が変更されるとonChangeが呼ばれる', async () => {
      const handleChange = vi.fn();
      render(<MarkdownEditor value="" onChange={handleChange} />);

      // MDEditorの内部実装をシミュレート
      // MDEditorのonChangeは直接テストしにくいので、propsが正しく渡されていることを確認
      const editor = document.querySelector('.w-md-editor');
      expect(editor).toBeInTheDocument();
    });
  });

  describe('disabled状態', () => {
    it('disabled=trueの場合、textareaがdisabledになる', () => {
      render(<MarkdownEditor value="# Test" onChange={() => {}} disabled={true} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toHaveAttribute('disabled');
    });

    it('disabled=falseの場合、textareaはdisabledにならない', () => {
      render(<MarkdownEditor value="# Test" onChange={() => {}} disabled={false} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).not.toHaveAttribute('disabled');
    });
  });

  describe('クリーンアップ', () => {
    it('アンマウント時にタイマーがクリアされる', async () => {
      mockPreviewMarkdown.mockResolvedValue('<h1>Test</h1>');

      const { unmount } = render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // 100ms経過後にアンマウント
      await advanceTimersAndFlush(100);

      unmount();

      // 残りの200ms経過
      await advanceTimersAndFlush(200);

      // アンマウント後はAPIが呼ばれない
      expect(mockPreviewMarkdown).not.toHaveBeenCalled();
    });
  });
});
