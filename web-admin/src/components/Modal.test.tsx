import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from './Modal';

describe('Modal', () => {
  describe('Rendering', () => {
    it('should render nothing when isOpen is false', () => {
      const onClose = vi.fn();
      const { container } = render(
        <Modal isOpen={false} onClose={onClose}>
          <p>Content</p>
        </Modal>
      );

      expect(container.firstChild).toBeNull();
    });

    it('should render modal when isOpen is true', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose}>
          <p>Content</p>
        </Modal>
      );

      expect(screen.getByText('Content')).toBeInTheDocument();
    });

    it('should render title when provided', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose} title="Test Title">
          <p>Content</p>
        </Modal>
      );

      expect(screen.getByText('Test Title')).toBeInTheDocument();
    });

    it('should not render title section when title is not provided', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose}>
          <p>Content</p>
        </Modal>
      );

      expect(screen.queryByRole('heading')).not.toBeInTheDocument();
    });

    it('should have correct aria attributes', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose} title="Test Title">
          <p>Content</p>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-modal', 'true');
      expect(dialog).toHaveAttribute('aria-labelledby', 'modal-title');
    });
  });

  describe('Overlay click', () => {
    it('should call onClose when overlay is clicked and closeOnOverlayClick is true', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose} closeOnOverlayClick={true}>
          <p>Content</p>
        </Modal>
      );

      const overlay = screen.getByRole('dialog');
      await user.click(overlay);

      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('should not call onClose when overlay is clicked and closeOnOverlayClick is false', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose} closeOnOverlayClick={false}>
          <p>Content</p>
        </Modal>
      );

      const overlay = screen.getByRole('dialog');
      await user.click(overlay);

      expect(onClose).not.toHaveBeenCalled();
    });

    it('should not call onClose when dialog content is clicked', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose} closeOnOverlayClick={true}>
          <p>Content</p>
        </Modal>
      );

      await user.click(screen.getByText('Content'));

      expect(onClose).not.toHaveBeenCalled();
    });
  });

  describe('Escape key', () => {
    it('should call onClose when Escape is pressed and closeOnEscape is true', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose} closeOnEscape={true}>
          <p>Content</p>
        </Modal>
      );

      fireEvent.keyDown(document, { key: 'Escape' });

      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('should not call onClose when Escape is pressed and closeOnEscape is false', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose} closeOnEscape={false}>
          <p>Content</p>
        </Modal>
      );

      fireEvent.keyDown(document, { key: 'Escape' });

      expect(onClose).not.toHaveBeenCalled();
    });

    it('should not respond to Escape when modal is closed', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={false} onClose={onClose} closeOnEscape={true}>
          <p>Content</p>
        </Modal>
      );

      fireEvent.keyDown(document, { key: 'Escape' });

      expect(onClose).not.toHaveBeenCalled();
    });
  });

  describe('Focus management', () => {
    it('should focus first focusable element when modal opens', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose}>
          <button>First Button</button>
          <button>Second Button</button>
        </Modal>
      );

      expect(screen.getByText('First Button')).toHaveFocus();
    });
  });

  describe('Body scroll lock', () => {
    it('should add overflow hidden to body when modal opens', () => {
      const onClose = vi.fn();
      render(
        <Modal isOpen={true} onClose={onClose}>
          <p>Content</p>
        </Modal>
      );

      expect(document.body.style.overflow).toBe('hidden');
    });

    it('should restore body overflow when modal closes', () => {
      const onClose = vi.fn();
      const { rerender } = render(
        <Modal isOpen={true} onClose={onClose}>
          <p>Content</p>
        </Modal>
      );

      expect(document.body.style.overflow).toBe('hidden');

      rerender(
        <Modal isOpen={false} onClose={onClose}>
          <p>Content</p>
        </Modal>
      );

      expect(document.body.style.overflow).toBe('');
    });
  });
});
