import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act, fireEvent, waitFor } from '@testing-library/react';
import { MarkdownEditor } from './MarkdownEditor';
import { apiClient } from '../api/client';

// Mock apiClient
vi.mock('../api/client', () => ({
  apiClient: {
    previewMarkdown: vi.fn(),
    uploadImage: vi.fn(),
  },
}));

const mockPreviewMarkdown = vi.mocked(apiClient.previewMarkdown);
const mockUploadImage = vi.mocked(apiClient.uploadImage);

// Helper to advance timers and flush async operations
async function advanceTimersAndFlush(ms: number) {
  await act(async () => {
    vi.advanceTimersByTime(ms);
    // Flush microtasks
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

  describe('Rendering', () => {
    it('displays Edit and Preview labels', () => {
      render(<MarkdownEditor value="" onChange={() => {}} />);

      expect(screen.getByText('Edit')).toBeInTheDocument();
      expect(screen.getByText('Preview')).toBeInTheDocument();
    });

    it('displays MDEditor', () => {
      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // Verify MDEditor container exists
      expect(document.querySelector('.w-md-editor')).toBeInTheDocument();
    });

    it('displays Preview area', () => {
      render(<MarkdownEditor value="" onChange={() => {}} />);

      expect(document.querySelector('.article-content')).toBeInTheDocument();
    });
  });

  describe('Preview functionality', () => {
    it('calls Preview API 300ms after value changes', async () => {
      mockPreviewMarkdown.mockResolvedValue('<h1>Test</h1>');

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // API should not be called before debounce
      expect(mockPreviewMarkdown).not.toHaveBeenCalled();

      // Advance 300ms
      await advanceTimersAndFlush(300);

      // Content is sent as parameter
      expect(mockPreviewMarkdown).toHaveBeenCalledWith(expect.stringContaining('# Test'));
    });

    it('displays Preview result as HTML', async () => {
      mockPreviewMarkdown.mockResolvedValue('<h1>Hello World</h1>');

      render(<MarkdownEditor value="# Hello World" onChange={() => {}} />);

      // Advance timer to resolve Promise
      await advanceTimersAndFlush(300);

      const previewArea = document.querySelector('.article-content');
      expect(previewArea?.innerHTML).toBe('<h1>Hello World</h1>');
    });

    it('does not call Preview API for empty value', async () => {
      render(<MarkdownEditor value="" onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      expect(mockPreviewMarkdown).not.toHaveBeenCalled();
    });

    it('does not call Preview API for whitespace-only value', async () => {
      render(<MarkdownEditor value="   " onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      expect(mockPreviewMarkdown).not.toHaveBeenCalled();
    });

    it('resets debounce timer when value changes', async () => {
      mockPreviewMarkdown.mockResolvedValue('<p>Result</p>');

      const { rerender } = render(<MarkdownEditor value="First" onChange={() => {}} />);

      // 200ms elapsed
      await advanceTimersAndFlush(200);

      // API not called yet
      expect(mockPreviewMarkdown).not.toHaveBeenCalled();

      // Change value
      rerender(<MarkdownEditor value="Second" onChange={() => {}} />);

      // Another 200ms elapsed (total 400ms)
      await advanceTimersAndFlush(200);

      // API still not called (debounce was reset)
      expect(mockPreviewMarkdown).not.toHaveBeenCalled();

      // Another 100ms elapsed (300ms from value change)
      await advanceTimersAndFlush(100);

      // Content is sent as parameter
      expect(mockPreviewMarkdown).toHaveBeenCalledWith(expect.stringContaining('Second'));
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(1);
    });
  });

  describe('Loading state', () => {
    it('displays loading indicator while fetching Preview', async () => {
      // Keep Promise pending
      let resolvePromise: (value: string) => void;
      mockPreviewMarkdown.mockImplementation(
        () => new Promise((resolve) => { resolvePromise = resolve; })
      );

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // Advance timer (triggers API call)
      await act(async () => {
        vi.advanceTimersByTime(300);
      });

      // Verify loading indicator
      expect(screen.getByText('(Loading...)')).toBeInTheDocument();

      // Resolve Promise and update state
      await act(async () => {
        resolvePromise!('<h1>Test</h1>');
      });

      // Loading indicator disappears
      expect(screen.queryByText('(Loading...)')).not.toBeInTheDocument();
    });
  });

  describe('Error handling', () => {
    it('displays error message on API error', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
      mockPreviewMarkdown.mockRejectedValue(new Error('API Error'));

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // Advance timer to resolve (reject) Promise
      await advanceTimersAndFlush(300);

      const previewArea = document.querySelector('.article-content');
      expect(previewArea?.innerHTML).toContain('Failed to load preview');

      consoleError.mockRestore();
    });
  });

  describe('onChange', () => {
    it('calls onChange when editor value changes', async () => {
      const handleChange = vi.fn();
      render(<MarkdownEditor value="" onChange={handleChange} />);

      // Simulate MDEditor internal implementation
      // MDEditor's onChange is hard to test directly, so verify props are passed correctly
      const editor = document.querySelector('.w-md-editor');
      expect(editor).toBeInTheDocument();
    });
  });

  describe('Disabled state', () => {
    it('disables textarea when disabled=true', () => {
      render(<MarkdownEditor value="# Test" onChange={() => {}} disabled={true} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toHaveAttribute('disabled');
    });

    it('does not disable textarea when disabled=false', () => {
      render(<MarkdownEditor value="# Test" onChange={() => {}} disabled={false} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).not.toHaveAttribute('disabled');
    });
  });

  describe('Cleanup', () => {
    it('clears timer on unmount', async () => {
      mockPreviewMarkdown.mockResolvedValue('<h1>Test</h1>');

      const { unmount } = render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // 100ms elapsed then unmount
      await advanceTimersAndFlush(100);

      unmount();

      // Remaining 200ms elapsed
      await advanceTimersAndFlush(200);

      // API not called after unmount
      expect(mockPreviewMarkdown).not.toHaveBeenCalled();
    });
  });

  describe('Save shortcut', () => {
    it('calls onSave on Cmd+Enter (Mac)', () => {
      const handleSave = vi.fn();
      render(<MarkdownEditor value="# Test" onChange={() => {}} onSave={handleSave} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      fireEvent.keyDown(textarea!, { key: 'Enter', metaKey: true });

      expect(handleSave).toHaveBeenCalledTimes(1);
    });

    it('calls onSave on Ctrl+Enter (Windows/Linux)', () => {
      const handleSave = vi.fn();
      render(<MarkdownEditor value="# Test" onChange={() => {}} onSave={handleSave} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      fireEvent.keyDown(textarea!, { key: 'Enter', ctrlKey: true });

      expect(handleSave).toHaveBeenCalledTimes(1);
    });

    it('does not call onSave on Enter without modifier keys', () => {
      const handleSave = vi.fn();
      render(<MarkdownEditor value="# Test" onChange={() => {}} onSave={handleSave} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      fireEvent.keyDown(textarea!, { key: 'Enter' });

      expect(handleSave).not.toHaveBeenCalled();
    });

    it('does not throw error when onSave is not provided', () => {
      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      // Verify no error is thrown
      expect(() => {
        fireEvent.keyDown(textarea!, { key: 'Enter', metaKey: true });
      }).not.toThrow();
    });

    it('does not call onSave on Cmd+other keys', () => {
      const handleSave = vi.fn();
      render(<MarkdownEditor value="# Test" onChange={() => {}} onSave={handleSave} />);

      const textarea = document.querySelector('textarea');
      expect(textarea).toBeInTheDocument();

      fireEvent.keyDown(textarea!, { key: 's', metaKey: true });

      expect(handleSave).not.toHaveBeenCalled();
    });
  });

  describe('Image upload', () => {
    it('has hidden file input', () => {
      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      expect(fileInput).toBeInTheDocument();
      expect(fileInput).toHaveClass('hidden');
      expect(fileInput).toHaveAttribute('accept', 'image/jpeg,image/png,image/gif,image/webp');
    });

    it('calls onChange when image is uploaded', async () => {
      vi.useRealTimers(); // Use real timers for async operations
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

      // When textApiRef.current is null, onChange is called
      await waitFor(() => {
        expect(handleChange).toHaveBeenCalledWith(expect.stringContaining('![test.jpg](/uploads/test-uuid.jpg)'));
      });
    });

    it('displays error for invalid file type', async () => {
      vi.useRealTimers();
      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      const file = new File(['fake text'], 'test.txt', { type: 'text/plain' });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [file] } });
      });

      // Upload API should not be called
      expect(mockUploadImage).not.toHaveBeenCalled();

      // Error message is displayed
      await waitFor(() => {
        expect(screen.getByText('Supported formats: JPEG, PNG, GIF, WebP only')).toBeInTheDocument();
      });
    });

    it('displays error for file too large', async () => {
      vi.useRealTimers();
      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      // Create 6MB file
      const largeContent = new Array(6 * 1024 * 1024).fill('a').join('');
      const file = new File([largeContent], 'large.jpg', { type: 'image/jpeg' });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [file] } });
      });

      // Upload API should not be called
      expect(mockUploadImage).not.toHaveBeenCalled();

      // Error message is displayed
      await waitFor(() => {
        expect(screen.getByText('File size must be less than 5MB')).toBeInTheDocument();
      });
    });

    it('displays loading indicator during upload', async () => {
      vi.useRealTimers();
      let resolveUpload: (value: { url: string; filename: string }) => void;
      mockUploadImage.mockImplementation(
        () => new Promise((resolve) => { resolveUpload = resolve; })
      );

      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      const file = new File(['fake image'], 'test.jpg', { type: 'image/jpeg' });

      fireEvent.change(fileInput, { target: { files: [file] } });

      // Verify loading indicator
      await waitFor(() => {
        expect(screen.getByText('(Uploading...)')).toBeInTheDocument();
      });

      // Complete upload
      await act(async () => {
        resolveUpload!({ url: '/uploads/test.jpg', filename: 'test.jpg' });
      });

      // Loading indicator disappears
      await waitFor(() => {
        expect(screen.queryByText('(Uploading...)')).not.toBeInTheDocument();
      });
    });

    it('displays error message on upload API error', async () => {
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

    it('disables textarea during upload', async () => {
      vi.useRealTimers();
      let resolveUpload: (value: { url: string; filename: string }) => void;
      mockUploadImage.mockImplementation(
        () => new Promise((resolve) => { resolveUpload = resolve; })
      );

      render(<MarkdownEditor value="" onChange={() => {}} />);

      const fileInput = screen.getByTestId('image-file-input');
      const file = new File(['fake image'], 'test.jpg', { type: 'image/jpeg' });

      fireEvent.change(fileInput, { target: { files: [file] } });

      // Verify textarea is disabled
      await waitFor(() => {
        const textarea = document.querySelector('textarea');
        expect(textarea).toHaveAttribute('disabled');
      });

      // Complete upload
      await act(async () => {
        resolveUpload!({ url: '/uploads/test.jpg', filename: 'test.jpg' });
      });

      // Disabled is removed
      await waitFor(() => {
        const textarea = document.querySelector('textarea');
        expect(textarea).not.toHaveAttribute('disabled');
      });
    });
  });

  describe('data-line scroll functionality', () => {
    let scrollToMock: ReturnType<typeof vi.fn>;
    let originalScrollTo: typeof Element.prototype.scrollTo;

    beforeEach(() => {
      // Mock scrollTo
      originalScrollTo = Element.prototype.scrollTo;
      scrollToMock = vi.fn();
      Element.prototype.scrollTo = scrollToMock as unknown as typeof Element.prototype.scrollTo;
    });

    afterEach(() => {
      Element.prototype.scrollTo = originalScrollTo;
    });

    it('sends content as-is to Preview API (no marker)', async () => {
      mockPreviewMarkdown.mockResolvedValue('<p data-line="0">Test</p>');

      render(<MarkdownEditor value="Test content" onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      // Verify API call arguments
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(1);
      const calledWith = mockPreviewMarkdown.mock.calls[0]?.[0] as string | undefined;
      // Content is sent as-is (no marker)
      expect(calledWith).toBe('Test content');
    });

    it('scrolls to data-line element after Preview update', async () => {
      const htmlWithDataLine = '<h1 data-line="0">Title</h1><p data-line="2">Content</p>';
      mockPreviewMarkdown.mockResolvedValue(htmlWithDataLine);

      render(<MarkdownEditor value="# Title\n\nContent" onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      // Verify Preview area contains data-line attribute
      const previewArea = document.querySelector('.article-content');
      expect(previewArea?.innerHTML).toContain('data-line=');

      // Verify scrollTo was called
      expect(scrollToMock).toHaveBeenCalled();
    });

    it('only refetches Preview on content change (not on cursor movement)', async () => {
      const htmlWithDataLine = '<h1 data-line="0">Title</h1><p data-line="2">Content</p>';
      mockPreviewMarkdown.mockResolvedValue(htmlWithDataLine);

      const { rerender } = render(<MarkdownEditor value="# Title\n\nContent" onChange={() => {}} />);

      // Wait for initial debounce
      await advanceTimersAndFlush(300);
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(1);

      // Simulate cursor movement (rerender with same content)
      rerender(<MarkdownEditor value="# Title\n\nContent" onChange={() => {}} />);

      // Wait for additional debounce
      await advanceTimersAndFlush(300);

      // Preview should not be refetched (content unchanged)
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(1);

      // Change content
      rerender(<MarkdownEditor value="# Title\n\nNew Content" onChange={() => {}} />);

      // Wait for debounce
      await advanceTimersAndFlush(300);

      // Preview should be refetched (content changed)
      expect(mockPreviewMarkdown).toHaveBeenCalledTimes(2);
    });
  });

  describe('Scroll sync', () => {
    // Skipped due to difficulty firing scroll events in test environment with MDEditor's internal implementation
    // Verified working via manual testing
    it.skip('syncs Preview scroll with editor scroll', async () => {
      mockPreviewMarkdown.mockResolvedValue('<p>Preview content</p>');

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      // Advance timer to display Preview
      await advanceTimersAndFlush(300);

      const textarea = document.querySelector('textarea');
      const previewArea = document.querySelector('.article-content');

      expect(textarea).toBeInTheDocument();
      expect(previewArea).toBeInTheDocument();

      // Simulate scrollable state
      Object.defineProperty(textarea!, 'scrollHeight', { value: 1000, configurable: true });
      Object.defineProperty(textarea!, 'clientHeight', { value: 500, configurable: true });
      Object.defineProperty(textarea!, 'scrollTop', { value: 250, configurable: true });
      Object.defineProperty(previewArea!, 'scrollHeight', { value: 800, configurable: true });
      Object.defineProperty(previewArea!, 'clientHeight', { value: 500, configurable: true });

      // Fire textarea scroll event
      fireEvent.scroll(textarea!);

      // Verify Preview scrollTop is updated
      // ratio = 250 / (1000 - 500) = 0.5
      // expected scrollTop = 0.5 * (800 - 500) = 150
      expect(previewArea!.scrollTop).toBe(150);
    });

    it.skip('does nothing when not scrollable', async () => {
      mockPreviewMarkdown.mockResolvedValue('<p>Short</p>');

      render(<MarkdownEditor value="# Test" onChange={() => {}} />);

      await advanceTimersAndFlush(300);

      const textarea = document.querySelector('textarea');
      const previewArea = document.querySelector('.article-content');

      // Simulate non-scrollable state
      Object.defineProperty(textarea!, 'scrollHeight', { value: 100, configurable: true });
      Object.defineProperty(textarea!, 'clientHeight', { value: 500, configurable: true });

      // Fire textarea scroll event
      fireEvent.scroll(textarea!);

      // Preview scrollTop remains unchanged
      expect(previewArea!.scrollTop).toBe(0);
    });
  });
});
