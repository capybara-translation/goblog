import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ModalProvider, useModal } from './useModal';

// テスト用のコンポーネント
function TestComponent() {
  const { showAlert, confirm } = useModal();

  return (
    <div>
      <button onClick={() => showAlert('Alert message')}>Show Alert</button>
      <button onClick={() => showAlert('Alert with title', { title: 'Custom Title' })}>
        Show Alert with Title
      </button>
      <button onClick={() => showAlert('Custom button', { buttonText: 'Got it' })}>
        Show Alert with Custom Button
      </button>
      <button
        onClick={async () => {
          const result = await confirm('Confirm message');
          document.body.setAttribute('data-confirm-result', String(result));
        }}
      >
        Show Confirm
      </button>
      <button
        onClick={async () => {
          const result = await confirm('Delete?', {
            title: 'Delete Item',
            confirmText: 'Delete',
            cancelText: 'Cancel',
            danger: true,
          });
          document.body.setAttribute('data-confirm-result', String(result));
        }}
      >
        Show Danger Confirm
      </button>
    </div>
  );
}

describe('useModal', () => {
  describe('useModal hook', () => {
    it('should throw error when used outside ModalProvider', () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

      expect(() => {
        render(<TestComponent />);
      }).toThrow('useModal must be used within a ModalProvider');

      consoleError.mockRestore();
    });
  });

  describe('showAlert', () => {
    it('should display alert dialog with message', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Alert'));

      expect(screen.getByText('Alert message')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'OK' })).toBeInTheDocument();
    });

    it('should display alert dialog with custom title', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Alert with Title'));

      expect(screen.getByText('Custom Title')).toBeInTheDocument();
      expect(screen.getByText('Alert with title')).toBeInTheDocument();
    });

    it('should display alert dialog with custom button text', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Alert with Custom Button'));

      expect(screen.getByRole('button', { name: 'Got it' })).toBeInTheDocument();
    });

    it('should close alert when OK button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Alert'));
      expect(screen.getByText('Alert message')).toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: 'OK' }));

      await waitFor(() => {
        expect(screen.queryByText('Alert message')).not.toBeInTheDocument();
      });
    });

    it('should close alert when Escape is pressed', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Alert'));
      expect(screen.getByText('Alert message')).toBeInTheDocument();

      fireEvent.keyDown(document, { key: 'Escape' });

      await waitFor(() => {
        expect(screen.queryByText('Alert message')).not.toBeInTheDocument();
      });
    });

    it('should close alert when overlay is clicked', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Alert'));
      expect(screen.getByText('Alert message')).toBeInTheDocument();

      const overlay = screen.getByRole('dialog');
      await user.click(overlay);

      await waitFor(() => {
        expect(screen.queryByText('Alert message')).not.toBeInTheDocument();
      });
    });
  });

  describe('confirm', () => {
    it('should display confirm dialog with message', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Confirm'));

      expect(screen.getByText('Confirm message')).toBeInTheDocument();
      expect(screen.getByText('Confirm')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'OK' })).toBeInTheDocument();
    });

    it('should display confirm dialog with custom options', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Danger Confirm'));

      expect(screen.getByText('Delete Item')).toBeInTheDocument();
      expect(screen.getByText('Delete?')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument();
    });

    it('should return true when confirm button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Confirm'));
      await user.click(screen.getByRole('button', { name: 'OK' }));

      await waitFor(() => {
        expect(document.body.getAttribute('data-confirm-result')).toBe('true');
      });
    });

    it('should return false when cancel button is clicked', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Confirm'));
      await user.click(screen.getByRole('button', { name: 'Cancel' }));

      await waitFor(() => {
        expect(document.body.getAttribute('data-confirm-result')).toBe('false');
      });
    });

    it('should return false when Escape is pressed', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Confirm'));
      fireEvent.keyDown(document, { key: 'Escape' });

      await waitFor(() => {
        expect(document.body.getAttribute('data-confirm-result')).toBe('false');
      });
    });

    it('should not close confirm when overlay is clicked', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Confirm'));
      expect(screen.getByText('Confirm message')).toBeInTheDocument();

      const overlay = screen.getByRole('dialog');
      await user.click(overlay);

      // Confirm dialog should still be visible (closeOnOverlayClick is false)
      expect(screen.getByText('Confirm message')).toBeInTheDocument();
    });

    it('should apply danger styling when danger option is true', async () => {
      const user = userEvent.setup();
      render(
        <ModalProvider>
          <TestComponent />
        </ModalProvider>
      );

      await user.click(screen.getByText('Show Danger Confirm'));

      const deleteButton = screen.getByRole('button', { name: 'Delete' });
      expect(deleteButton).toHaveClass('bg-red-600');
    });
  });
});
