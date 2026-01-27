import { useState, useEffect, useRef, useCallback } from 'react';
import MDEditor, { commands, ICommand, TextState, TextAreaTextApi } from '@uiw/react-md-editor';
import { apiClient } from '../api/client';

// 許可されるMIMEタイプ
const ALLOWED_MIME_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
// 最大ファイルサイズ（5MB）
const MAX_FILE_SIZE = 5 * 1024 * 1024;

// カーソル位置マーカーのID（バックエンドと同じ値）
const CURSOR_LINE_MARKER_ID = 'cursor-line-marker';

// カーソル位置マーカー（ゼロ幅スペース）
const CURSOR_MARKER = '\u200B';

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
  // カーソル位置を保持（プレビュー用）
  const cursorPosRef = useRef<number>(0);

  // カーソル位置にマーカーを挿入したコンテンツを生成
  const getContentWithMarker = useCallback((content: string, cursorPos: number): string => {
    // 既存のマーカーを除去してから新しい位置に挿入
    const cleanContent = content.replace(CURSOR_MARKER, '');
    const pos = Math.min(cursorPos, cleanContent.length);
    return cleanContent.slice(0, pos) + CURSOR_MARKER + cleanContent.slice(pos);
  }, []);

  // プレビューを取得
  const fetchPreview = useCallback(async (content: string, cursorPos: number) => {
    if (!content.trim()) {
      setPreviewHtml('');
      return;
    }

    setIsLoading(true);
    try {
      const contentWithMarker = getContentWithMarker(content, cursorPos);
      const html = await apiClient.previewMarkdown(contentWithMarker);
      setPreviewHtml(html);
    } catch (error) {
      console.error('Failed to preview markdown:', error);
      setPreviewHtml('<p class="text-red-500">プレビューの取得に失敗しました</p>');
    } finally {
      setIsLoading(false);
    }
  }, [getContentWithMarker]);

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
      fetchPreview(value, cursorPosRef.current);
    }, 300); // 300ms のデバウンス

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [value, fetchPreview]);

  // プレビュー更新後にマーカー位置にスクロール
  useEffect(() => {
    if (!previewHtml) return;

    const container = previewRef.current;
    const marker = container?.querySelector(`#${CURSOR_LINE_MARKER_ID}`) as HTMLElement | null;
    if (container && marker) {
      // getBoundingClientRectで正確な相対位置を計算
      const containerRect = container.getBoundingClientRect();
      const markerRect = marker.getBoundingClientRect();
      // マーカーのコンテナ内での相対位置（現在のスクロール位置を考慮）
      const markerRelativeTop = markerRect.top - containerRect.top + container.scrollTop;
      const containerHeight = container.clientHeight;
      // マーカーを中央に配置するスクロール位置
      const scrollTop = markerRelativeTop - containerHeight / 2;
      container.scrollTo({
        top: Math.max(0, scrollTop),
        behavior: 'smooth',
      });
    }
  }, [previewHtml]);

  // カーソル位置変更時にプレビューを再取得
  const handleCursorChange = useCallback((textarea: HTMLTextAreaElement) => {
    const newPos = textarea.selectionStart;
    if (cursorPosRef.current !== newPos) {
      cursorPosRef.current = newPos;
      // カーソル移動でもプレビューを更新（デバウンス）
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
      debounceTimerRef.current = window.setTimeout(() => {
        fetchPreview(value, newPos);
      }, 300);
    }
  }, [value, fetchPreview]);

  // ファイル選択時の処理
  const handleFileSelect = useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    // ファイル入力をリセット（同じファイルを再選択できるように）
    event.target.value = '';

    // クライアント側バリデーション
    if (!ALLOWED_MIME_TYPES.includes(file.type)) {
      setUploadError('対応している形式は JPEG, PNG, GIF, WebP のみです');
      return;
    }

    if (file.size > MAX_FILE_SIZE) {
      setUploadError(`ファイルサイズは ${MAX_FILE_SIZE / 1024 / 1024}MB 以下にしてください`);
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
      setUploadError(error instanceof Error ? error.message : '画像のアップロードに失敗しました');
    } finally {
      setIsUploading(false);
    }
  }, [value, onChange]);

  // 画像アップロードコマンド
  const imageUploadCommand: ICommand = {
    name: 'image-upload',
    keyCommand: 'image-upload',
    buttonProps: { 'aria-label': '画像をアップロード', title: '画像をアップロード' },
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
          編集
          {isUploading && <span className="ml-2 text-primary-400">(アップロード中...)</span>}
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
            placeholder: `# 見出し

本文をここに書きます...

## Markdownの書き方

- **太字**: **テキスト**
- *斜体*: *テキスト*
- [リンク](URL)
- コードブロック: \`\`\`言語名
- 引用: > 引用テキスト
- タスクリスト: - [x] 完了`,
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
          プレビュー
          {isLoading && <span className="ml-2 text-primary-400">(読み込み中...)</span>}
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
