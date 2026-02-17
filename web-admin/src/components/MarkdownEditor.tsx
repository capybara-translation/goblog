import { useState, useEffect, useRef, useCallback } from 'react';
import MDEditor, { commands, ICommand, TextState, TextAreaTextApi } from '@uiw/react-md-editor';
import { apiClient } from '../api/client';
import { ImageUploadIcon } from './icons/ImageUploadIcon';
import { AmazonIcon } from './icons/AmazonIcon';

// Allowed MIME types
const ALLOWED_MIME_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
// Maximum file size (5MB)
const MAX_FILE_SIZE = 5 * 1024 * 1024;

// Get line number from cursor position (0-based)
function getLineFromCursorPos(content: string, cursorPos: number): number {
  return content.slice(0, cursorPos).split('\n').length - 1;
}

// Shorten Amazon product URL to https://www.amazon.{tld}/dp/{ASIN}
// Supports /dp/, /gp/product/, /gp/aw/d/, /o/ASIN/, /exec/obidos/ASIN/ patterns
const AMAZON_ASIN_RE =
  /https?:\/\/(?:www\.)?(amazon\.[^/]+)\/(?:[^/?#]+\/)?(?:dp|gp\/product|gp\/aw\/d|o\/ASIN|exec\/obidos\/ASIN)\/([A-Z0-9]{10})(?:[/?#]|$)/i;

function shortenAmazonURL(text: string): string | null {
  const match = text.match(AMAZON_ASIN_RE);
  if (!match || !match[1] || !match[2]) return null;
  return `https://www.${match[1]}/dp/${match[2].toUpperCase()}`;
}

interface MarkdownEditorProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  onSave?: () => void;
}

/**
 * Markdown editor component
 * Converts Markdown to HTML on the server side for preview display
 * Preview matches the same appearance as the public page
 */
export function MarkdownEditor({ value, onChange, disabled = false, onSave }: MarkdownEditorProps) {
  const [previewHtml, setPreviewHtml] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [shortenAmazon, setShortenAmazon] = useState(true);
  const debounceTimerRef = useRef<number | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const textApiRef = useRef<TextAreaTextApi | null>(null);
  const previewRef = useRef<HTMLDivElement>(null);
  // Keep track of current line number (for scrolling)
  const currentLineRef = useRef<number>(0);

  // Scroll preview using data-line attribute
  const scrollToLine = useCallback((line: number) => {
    const container = previewRef.current;
    if (!container) return;

    const elements = container.querySelectorAll('[data-line]');
    let targetElement: Element | null = null;
    let minDiff = Infinity;

    // Find the closest element at or before the cursor line
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

    if (targetElement !== null) {
      // Scroll the element to 1/3 position of the screen
      const containerRect = container.getBoundingClientRect();
      const elementRect = (targetElement as Element).getBoundingClientRect();
      const scrollTop = elementRect.top - containerRect.top + container.scrollTop - container.clientHeight / 3;
      container.scrollTo({ top: Math.max(0, scrollTop), behavior: 'smooth' });
    }
  }, []);

  // Fetch preview (without marker insertion)
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

  // Fetch preview with debounce
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
    }, 300); // 300ms debounce

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [value, fetchPreview]);

  // Scroll to current line after preview update
  useEffect(() => {
    if (!previewHtml) return;
    scrollToLine(currentLineRef.current);
  }, [previewHtml, scrollToLine]);

  // Scroll on cursor move (without re-fetching preview)
  const handleCursorChange = useCallback((textarea: HTMLTextAreaElement) => {
    const line = getLineFromCursorPos(value, textarea.selectionStart);
    if (currentLineRef.current !== line) {
      currentLineRef.current = line;
      scrollToLine(line);
    }
  }, [value, scrollToLine]);

  // File selection handler
  const handleFileSelect = useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    // Reset file input (to allow re-selecting the same file)
    event.target.value = '';

    // Client-side validation
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

      // Insert Markdown image syntax
      const imageMarkdown = `![${file.name}](${response.url})`;

      if (textApiRef.current) {
        textApiRef.current.replaceSelection(imageMarkdown);
      } else {
        // Fallback: append to end
        onChange(value + '\n' + imageMarkdown);
      }
    } catch (error) {
      console.error('Failed to upload image:', error);
      setUploadError(error instanceof Error ? error.message : 'Failed to upload image');
    } finally {
      setIsUploading(false);
    }
  }, [value, onChange]);

  // Image upload command
  const imageUploadCommand: ICommand = {
    name: 'image-upload',
    keyCommand: 'image-upload',
    buttonProps: { 'aria-label': 'Upload image', title: 'Upload image' },
    icon: <ImageUploadIcon />,
    execute: (_state: TextState, api: TextAreaTextApi) => {
      textApiRef.current = api;
      fileInputRef.current?.click();
    },
  };

  // Amazon URL shortening toggle command (toolbar checkbox)
  const amazonShortenCommand: ICommand = {
    name: 'amazon-shorten',
    keyCommand: 'amazon-shorten',
    render: () => (
      <label
        className="flex items-center gap-1 px-2 cursor-pointer"
        title="Shorten Amazon URLs on paste"
      >
        <input
          type="checkbox"
          checked={shortenAmazon}
          onChange={(e) => {
            e.stopPropagation();
            setShortenAmazon(e.target.checked);
          }}
          className="h-3 w-3"
        />
        <AmazonIcon opacity={shortenAmazon ? 1 : 0.4} />
      </label>
    ),
    execute: () => {},
  };

  return (
    <div className="flex gap-4" data-color-mode="light">
      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp"
        onChange={handleFileSelect}
        className="hidden"
        data-testid="image-file-input"
      />

      {/* Editor section */}
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
            commands.divider,
            amazonShortenCommand,
          ]}
          extraCommands={[]}
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
            onPaste: (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
              if (!shortenAmazon) return;
              const text = e.clipboardData.getData('text/plain').trim();
              const shortened = shortenAmazonURL(text);
              if (shortened) {
                e.preventDefault();
                document.execCommand('insertText', false, shortened);
              }
            },
          }}
        />
      </div>

      {/* Preview section */}
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
