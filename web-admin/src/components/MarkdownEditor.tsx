import { useState, useEffect, useRef, useCallback } from 'react';
import MDEditor, { commands, ICommand, TextState, TextAreaTextApi } from '@uiw/react-md-editor';
import { apiClient } from '../api/client';

// 許可されるMIMEタイプ
const ALLOWED_MIME_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
// 最大ファイルサイズ（5MB）
const MAX_FILE_SIZE = 5 * 1024 * 1024;

// カーソル位置から行番号を取得（0-based）
function getLineFromCursorPos(content: string, cursorPos: number): number {
  return content.slice(0, cursorPos).split('\n').length - 1;
}

interface MarkdownEditorProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  onSave?: () => void;
}

/**
 * Markdownエディタコンポーネント
 * サーバーサイドでMarkdownをHTMLに変換してプレビュー表示
 * 公開ページと同じ見た目でプレビューできる
 */
export function MarkdownEditor({ value, onChange, disabled = false, onSave }: MarkdownEditorProps) {
  const [previewHtml, setPreviewHtml] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const debounceTimerRef = useRef<number | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const textApiRef = useRef<TextAreaTextApi | null>(null);
  const previewRef = useRef<HTMLDivElement>(null);
  // 現在の行番号を保持（スクロール用）
  const currentLineRef = useRef<number>(0);

  // data-line属性を使ってプレビューをスクロール
  const scrollToLine = useCallback((line: number) => {
    const container = previewRef.current;
    if (!container) return;

    const elements = container.querySelectorAll('[data-line]');
    let targetElement: Element | null = null;
    let minDiff = Infinity;

    // カーソル行以下で最も近い要素を探す
    elements.forEach((el) => {
      const elLine = parseInt(el.getAttribute('data-line') || '0', 10);
      if (elLine <= line) {
        const diff = line - elLine;
        if (diff < minDiff) {
          minDiff = diff;
          targetElement = el;
        }
      }
    });

    if (targetElement) {
      // 要素を画面の1/3の位置にスクロール
      const containerRect = container.getBoundingClientRect();
      const elementRect = targetElement.getBoundingClientRect();
      const scrollTop = elementRect.top - containerRect.top + container.scrollTop - container.clientHeight / 3;
      container.scrollTo({ top: Math.max(0, scrollTop), behavior: 'smooth' });
    }
  }, []);

  // プレビューを取得（マーカー挿入なし）
  const fetchPreview = useCallback(async (content: string) => {
    if (!content.trim()) {
      setPreviewHtml('');
      return;
    }

    setIsLoading(true);
    try {
      const html = await apiClient.previewMarkdown(content);
      setPreviewHtml(html);
    } catch (error) {
      console.error('Failed to preview markdown:', error);
      setPreviewHtml('<p class="text-red-500">Failed to load preview</p>');
    } finally {
      setIsLoading(false);
    }
  }, []);

  // デバウンス付きでプレビューを取得
  useEffect(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    if (!value.trim()) {
      setPreviewHtml('');
      return;
    }

    debounceTimerRef.current = window.setTimeout(() => {
      fetchPreview(value);
    }, 300); // 300ms のデバウンス

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [value, fetchPreview]);

  // プレビュー更新後に現在行にスクロール
  useEffect(() => {
    if (!previewHtml) return;
    scrollToLine(currentLineRef.current);
  }, [previewHtml, scrollToLine]);

  // カーソル移動でスクロール（プレビュー再取得なし）
  const handleCursorChange = useCallback((textarea: HTMLTextAreaElement) => {
    const line = getLineFromCursorPos(value, textarea.selectionStart);
    if (currentLineRef.current !== line) {
      currentLineRef.current = line;
      scrollToLine(line);
    }
  }, [value, scrollToLine]);

  // ファイル選択時の処理
  const handleFileSelect = useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    // ファイル入力をリセット（同じファイルを再選択できるように）
    event.target.value = '';

    // クライアント側バリデーション
    if (!ALLOWED_MIME_TYPES.includes(file.type)) {
      setUploadError('Supported formats: JPEG, PNG, GIF, WebP only');
      return;
    }

    if (file.size > MAX_FILE_SIZE) {
      setUploadError(`File size must be less than ${MAX_FILE_SIZE / 1024 / 1024}MB`);
      return;
    }

    setUploadError(null);
    setIsUploading(true);

    try {
      const response = await apiClient.uploadImage(file);

      // Markdown画像構文を挿入
      const imageMarkdown = `![${file.name}](${response.url})`;

      if (textApiRef.current) {
        textApiRef.current.replaceSelection(imageMarkdown);
      } else {
        // フォールバック: 末尾に追加
        onChange(value + '\n' + imageMarkdown);
      }
    } catch (error) {
      console.error('Failed to upload image:', error);
      setUploadError(error instanceof Error ? error.message : 'Failed to upload image');
    } finally {
      setIsUploading(false);
    }
  }, [value, onChange]);

  // 画像アップロードコマンド
  const imageUploadCommand: ICommand = {
    name: 'image-upload',
    keyCommand: 'image-upload',
    buttonProps: { 'aria-label': 'Upload image', title: 'Upload image' },
    icon: (
      <svg width="12" height="12" viewBox="0 0 20 20" fill="currentColor">
        <path fillRule="evenodd" d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z" clipRule="evenodd" />
      </svg>
    ),
    execute: (_state: TextState, api: TextAreaTextApi) => {
      textApiRef.current = api;
      fileInputRef.current?.click();
    },
  };

  return (
    <div className="flex gap-4" data-color-mode="light">
      {/* 非表示のファイル入力 */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp"
        onChange={handleFileSelect}
        className="hidden"
        data-testid="image-file-input"
      />

      {/* エディタ部分 */}
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-primary-600 mb-2">
          Edit
          {isUploading && <span className="ml-2 text-primary-400">(Uploading...)</span>}
        </div>
        {uploadError && (
          <div className="mb-2 p-2 bg-red-50 border border-red-200 rounded text-red-600 text-sm">
            {uploadError}
          </div>
        )}
        <MDEditor
          value={value}
          onChange={(val) => onChange(val || '')}
          height={500}
          preview="edit"
          commands={[
            commands.bold,
            commands.italic,
            commands.strikethrough,
            commands.divider,
            commands.heading,
            commands.link,
            imageUploadCommand,
            commands.quote,
            commands.code,
            commands.codeBlock,
            commands.divider,
            commands.unorderedListCommand,
            commands.orderedListCommand,
            commands.checkedListCommand,
          ]}
          textareaProps={{
            disabled: disabled || isUploading,
            placeholder: `# Heading

Write your content here...

## Markdown syntax

- **Bold**: **text**
- *Italic*: *text*
- [Link](URL)
- Code block: \`\`\`language
- Blockquote: > quoted text
- Task list: - [x] completed`,
            onSelect: (e: React.SyntheticEvent<HTMLTextAreaElement>) => {
              handleCursorChange(e.currentTarget);
            },
            onKeyUp: (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
              handleCursorChange(e.currentTarget);
            },
            onClick: (e: React.MouseEvent<HTMLTextAreaElement>) => {
              handleCursorChange(e.currentTarget);
            },
            onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
              if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                e.preventDefault();
                onSave?.();
              }
            },
          }}
        />
      </div>

      {/* プレビュー部分 */}
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-primary-600 mb-2">
          Preview
          {isLoading && <span className="ml-2 text-primary-400">(Loading...)</span>}
        </div>
        <div
          ref={previewRef}
          className="article-content markdown-content h-[500px] overflow-y-auto border border-primary-200"
          dangerouslySetInnerHTML={{ __html: previewHtml }}
        />
      </div>
    </div>
  );
}
