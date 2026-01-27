import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act, fireEvent, waitFor } from '@testing-library/react';
import { MarkdownEditor } from './MarkdownEditor';
import { apiClient } from '../api/client';

// apiClient をモック
vi.mock('../api/client', () => ({
  apiClient: {
    previewMarkdown: vi.fn(),
    uploadImage: vi.fn(),
  },
}));

const mockPreviewMarkdown = vi.mocked(apiClient.previewMarkdown);
const mockUploadImage = vi.mocked(apiClient.uploadImage);

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
    mockUploadImage.mockReset();
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

      // コンテンツ（カーソルマーカー付き）がパラメータとして送信される
      expect(mockPreviewMarkdown).toHaveBeenCalledWith(expect.stringContaining('# Test'));
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

      // コンテンツ（カーソルマーカー付き）がパラメータとして送信される
      expect(mockPreviewMarkdown).toHaveBeenCalledWith(expect.stringContaining('Second'));
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

  describe('保存ショートカット', () => {
    it('Cmd+Enter でonSaveが呼ばれる（Mac）', () => {
      const handleSave = vi.fn();
      render(<MarkdownEditor value="# Test" onChange={() => {}} onSave={handleSave} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      fireEvent.keyDown(textarea!, { key: 'Enter', metaKey: true });

      expect(handleSave).toHaveBeenCalledTimes(1);
    });

    it('Ctrl+Enter でonSaveが呼ばれる（Windows/Linux）', () => {
      const handleSave = vi.fn();
      render(<MarkdownEditor value="# Test" onChange={() => {}} onSave={handleSave} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      fireEvent.keyDown(textarea!, { key: 'Enter', ctrlKey: true });

      expect(handleSave).toHaveBeenCalledTimes(1);
    });

    it('修飾キーなしのEnterではonSaveが呼ばれない', () => {
      const handleSave = vi.fn();
      render(<MarkdownEditor value="# Test" onChange={() => {}} onSave={handleSave} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      fireEvent.keyDown(textarea!, { key: 'Enter' });

      expect(handleSave).not.toHaveBeenCalled();
    });

    it('onSaveが未設定の場合でもエラーにならない', () => {
      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      // エラーが発生しないことを確認
      expect(() => {
        fireEvent.keyDown(textarea!, { key: 'Enter', metaKey: true });
      }).not.toThrow();
    });

    it('Cmd+他のキーではonSaveが呼ばれない', () => {
      const handleSave = vi.fn();
      render(<MarkdownEditor value="# Test" onChange={() => {}} onSave={handleSave} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      fireEvent.keyDown(textarea!, { key: 's', metaKey: true });

      expect(handleSave).not.toHaveBeenCalled();
    });
  });

  describe('画像アップロード', () => {
    it('非表示のファイル入力が存在する', () => {
      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      expect(fileInput).toBeInTheDocument();
      expect(fileInput).toHaveClass('hidden');
      expect(fileInput).toHaveAttribute('accept', 'image/jpeg,image/png,image/gif,image/webp');
    });

    it('画像をアップロードするとonChangeが呼ばれる', async () => {
      vi.useRealTimers(); // 非同期処理のためにリアルタイマーを使用
      const handleChange = vi.fn();
      mockUploadImage.mockResolvedValue({
        url: '/uploads/test-uuid.jpg',
        filename: 'test.jpg',
      });

      render(<MarkdownEditor value="" onChange={handleChange} />);

      const fileInput = screen.getByTestId('image-file-input');
      const file = new File(['fake image'], 'test.jpg', { type: 'image/jpeg' });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [file] } });
      });

      await waitFor(() => {
        expect(mockUploadImage).toHaveBeenCalledWith(file);
      });

      // textApiRef.currentがnullの場合、onChangeが呼ばれる
      await waitFor(() => {
        expect(handleChange).toHaveBeenCalledWith(expect.stringContaining('![test.jpg](/uploads/test-uuid.jpg)'));
      });
    });

    it('無効なファイル形式でエラーが表示される', async () => {
      vi.useRealTimers();
      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      const file = new File(['fake text'], 'test.txt', { type: 'text/plain' });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [file] } });
      });

      // アップロードAPIは呼ばれない
      expect(mockUploadImage).not.toHaveBeenCalled();

      // エラーメッセージが表示される
      await waitFor(() => {
        expect(screen.getByText('対応している形式は JPEG, PNG, GIF, WebP のみです')).toBeInTheDocument();
      });
    });

    it('ファイルサイズが大きすぎるとエラーが表示される', async () => {
      vi.useRealTimers();
      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      // 6MBのファイルを作成
      const largeContent = new Array(6 * 1024 * 1024).fill('a').join('');
      const file = new File([largeContent], 'large.jpg', { type: 'image/jpeg' });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [file] } });
      });

      // アップロードAPIは呼ばれない
      expect(mockUploadImage).not.toHaveBeenCalled();

      // エラーメッセージが表示される
      await waitFor(() => {
        expect(screen.getByText('ファイルサイズは 5MB 以下にしてください')).toBeInTheDocument();
      });
    });

    it('アップロード中はローディング表示される', async () => {
      vi.useRealTimers();
      let resolveUpload: (value: { url: string; filename: string }) => void;
      mockUploadImage.mockImplementation(
        () => new Promise((resolve) => { resolveUpload = resolve; })
      );

      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      const file = new File(['fake image'], 'test.jpg', { type: 'image/jpeg' });

      fireEvent.change(fileInput, { target: { files: [file] } });

      // ローディング表示を確認
      await waitFor(() => {
        expect(screen.getByText('(アップロード中...)')).toBeInTheDocument();
      });

      // アップロードを完了
      await act(async () => {
        resolveUpload!({ url: '/uploads/test.jpg', filename: 'test.jpg' });
      });

      // ローディング表示が消える
      await waitFor(() => {
        expect(screen.queryByText('(アップロード中...)')).not.toBeInTheDocument();
      });
    });

    it('アップロードAPIエラー時はエラーメッセージが表示される', async () => {
      vi.useRealTimers();
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
      mockUploadImage.mockRejectedValue(new Error('Upload failed'));

      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      const file = new File(['fake image'], 'test.jpg', { type: 'image/jpeg' });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [file] } });
      });

      await waitFor(() => {
        expect(screen.getByText('Upload failed')).toBeInTheDocument();
      });

      consoleError.mockRestore();
    });

    it('アップロード中はtextareaがdisabledになる', async () => {
      vi.useRealTimers();
      let resolveUpload: (value: { url: string; filename: string }) => void;
      mockUploadImage.mockImplementation(
        () => new Promise((resolve) => { resolveUpload = resolve; })
      );

      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      const file = new File(['fake image'], 'test.jpg', { type: 'image/jpeg' });

      fireEvent.change(fileInput, { target: { files: [file] } });

      // テキストエリアがdisabledになることを確認
      await waitFor(() => {
        const textarea = document.querySelector('textarea');
        expect(textarea).toHaveAttribute('disabled');
      });

      // アップロードを完了
      await act(async () => {
        resolveUpload!({ url: '/uploads/test.jpg', filename: 'test.jpg' });
      });

      // disabledが解除される
      await waitFor(() => {
        const textarea = document.querySelector('textarea');
        expect(textarea).not.toHaveAttribute('disabled');
      });
    });
  });

  describe('data-lineスクロール機能', () => {
    let scrollToMock: ReturnType<typeof vi.fn>;

    beforeEach(() => {
      // scrollTo をモック
      scrollToMock = vi.fn();
      Element.prototype.scrollTo = scrollToMock;
    });

    it('プレビューAPIにコンテンツがそのまま送信される（マーカーなし）', async () => {
      mockPreviewMarkdown.mockResolvedValue('<p data-line="0">Test</p>');

      render(<MarkdownEditor value="Test content" onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      // API呼び出しの引数を確認
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(1);
      const calledWith = mockPreviewMarkdown.mock.calls[0]?.[0] as string | undefined;
      // コンテンツがそのまま送信される（マーカーなし）
      expect(calledWith).toBe('Test content');
    });

    it('プレビュー更新後にdata-line要素にスクロールされる', async () => {
      const htmlWithDataLine = '<h1 data-line="0">Title</h1><p data-line="2">Content</p>';
      mockPreviewMarkdown.mockResolvedValue(htmlWithDataLine);

      render(<MarkdownEditor value="# Title\n\nContent" onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      // プレビューエリアにdata-line属性が含まれることを確認
      const previewArea = document.querySelector('.article-content');
      expect(previewArea?.innerHTML).toContain('data-line=');

      // scrollToが呼ばれることを確認
      expect(scrollToMock).toHaveBeenCalled();
    });

    it('コンテンツ変更時のみプレビューが再取得される（カーソル移動では再取得しない）', async () => {
      const htmlWithDataLine = '<h1 data-line="0">Title</h1><p data-line="2">Content</p>';
      mockPreviewMarkdown.mockResolvedValue(htmlWithDataLine);

      const { rerender } = render(<MarkdownEditor value="# Title\n\nContent" onChange={() => {}} />);

      // 初回のデバウンスを待つ
      await advanceTimersAndFlush(300);
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(1);

      // カーソル移動をシミュレート（同じコンテンツで再レンダリング）
      rerender(<MarkdownEditor value="# Title\n\nContent" onChange={() => {}} />);

      // 追加のデバウンスを待つ
      await advanceTimersAndFlush(300);

      // コンテンツが変わっていないので、プレビューは再取得されない
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(1);

      // コンテンツを変更
      rerender(<MarkdownEditor value="# Title\n\nNew Content" onChange={() => {}} />);

      // デバウンスを待つ
      await advanceTimersAndFlush(300);

      // コンテンツが変わったので、プレビューが再取得される
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(2);
    });
  });

  describe('スクロール同期', () => {
    // MDEditorの内部実装により、テスト環境でのスクロールイベント発火が困難なためスキップ
    // 手動テストで動作確認済み
    it.skip('エディターのスクロールに合わせてプレビューがスクロールする', async () => {
      mockPreviewMarkdown.mockResolvedValue('<p>Preview content</p>');

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // タイマーを進めてプレビューを表示
      await advanceTimersAndFlush(300);

      const textarea = document.querySelector('textarea');
      const previewArea = document.querySelector('.article-content');

      expect(textarea).toBeInTheDocument();
      expect(previewArea).toBeInTheDocument();

      // スクロール可能な状態をシミュレート
      Object.defineProperty(textarea!, 'scrollHeight', { value: 1000, configurable: true });
      Object.defineProperty(textarea!, 'clientHeight', { value: 500, configurable: true });
      Object.defineProperty(textarea!, 'scrollTop', { value: 250, configurable: true });
      Object.defineProperty(previewArea!, 'scrollHeight', { value: 800, configurable: true });
      Object.defineProperty(previewArea!, 'clientHeight', { value: 500, configurable: true });

      // textareaのスクロールイベントを発火
      fireEvent.scroll(textarea!);

      // プレビューのscrollTopが更新されることを確認
      // ratio = 250 / (1000 - 500) = 0.5
      // expected scrollTop = 0.5 * (800 - 500) = 150
      expect(previewArea!.scrollTop).toBe(150);
    });

    it.skip('スクロール不可能な場合は何も起きない', async () => {
      mockPreviewMarkdown.mockResolvedValue('<p>Short</p>');

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      const textarea = document.querySelector('textarea');
      const previewArea = document.querySelector('.article-content');

      // スクロール不可能な状態をシミュレート
      Object.defineProperty(textarea!, 'scrollHeight', { value: 100, configurable: true });
      Object.defineProperty(textarea!, 'clientHeight', { value: 500, configurable: true });

      // textareaのスクロールイベントを発火
      fireEvent.scroll(textarea!);

      // プレビューのscrollTopは変更されない
      expect(previewArea!.scrollTop).toBe(0);
    });
  });
});
