import { useState, useEffect, useRef } from 'react';
import MDEditor from '@uiw/react-md-editor';
import { apiClient } from '../api/client';

interface MarkdownEditorProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

/**
 * Markdownエディタコンポーネント
 * サーバーサイドでMarkdownをHTMLに変換してプレビュー表示
 * 公開ページと同じ見た目でプレビューできる
 */
export function MarkdownEditor({ value, onChange, disabled = false }: MarkdownEditorProps) {
  const [previewHtml, setPreviewHtml] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const debounceTimerRef = useRef<number | null>(null);

  // デバウンス付きでプレビューを取得
  useEffect(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    if (!value.trim()) {
      setPreviewHtml('');
      return;
    }

    debounceTimerRef.current = window.setTimeout(async () => {
      setIsLoading(true);
      try {
        const html = await apiClient.previewMarkdown(value);
        setPreviewHtml(html);
      } catch (error) {
        console.error('Failed to preview markdown:', error);
        setPreviewHtml('<p class="text-red-500">プレビューの取得に失敗しました</p>');
      } finally {
        setIsLoading(false);
      }
    }, 300); // 300ms のデバウンス

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [value]);

  return (
    <div className="flex gap-4" data-color-mode="light">
      {/* エディタ部分 */}
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-primary-600 mb-2">編集</div>
        <MDEditor
          value={value}
          onChange={(val) => onChange(val || '')}
          height={500}
          preview="edit"
          textareaProps={{
            disabled,
            placeholder: `# 見出し

本文をここに書きます...

## Markdownの書き方

- **太字**: **テキスト**
- *斜体*: *テキスト*
- [リンク](URL)
- コードブロック: \`\`\`言語名
- 引用: > 引用テキスト
- タスクリスト: - [x] 完了`,
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
          className="article-content markdown-content h-[500px] overflow-y-auto border border-primary-200"
          dangerouslySetInnerHTML={{ __html: previewHtml }}
        />
      </div>
    </div>
  );
}
