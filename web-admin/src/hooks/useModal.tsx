import { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { Modal } from '../components/Modal';

// オプション型定義
interface AlertOptions {
  title?: string;
  buttonText?: string;
}

interface ConfirmOptions {
  title?: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
}

// Context型定義
interface ModalContextType {
  showAlert: (message: string, options?: AlertOptions) => Promise<void>;
  confirm: (message: string, options?: ConfirmOptions) => Promise<boolean>;
}

const ModalContext = createContext<ModalContextType | undefined>(undefined);

interface ModalProviderProps {
  children: ReactNode;
}

// 内部状態の型
type ModalState =
  | { type: 'closed' }
  | { type: 'alert'; message: string; options: AlertOptions; resolve: () => void }
  | { type: 'confirm'; message: string; options: ConfirmOptions; resolve: (value: boolean) => void };

export function ModalProvider({ children }: ModalProviderProps) {
  const [modalState, setModalState] = useState<ModalState>({ type: 'closed' });

  const showAlert = useCallback((message: string, options: AlertOptions = {}): Promise<void> => {
    return new Promise((resolve) => {
      setModalState({ type: 'alert', message, options, resolve });
    });
  }, []);

  const confirm = useCallback((message: string, options: ConfirmOptions = {}): Promise<boolean> => {
    return new Promise((resolve) => {
      setModalState({ type: 'confirm', message, options, resolve });
    });
  }, []);

  const handleAlertClose = useCallback(() => {
    if (modalState.type === 'alert') {
      modalState.resolve();
    }
    setModalState({ type: 'closed' });
  }, [modalState]);

  const handleConfirmOk = useCallback(() => {
    if (modalState.type === 'confirm') {
      modalState.resolve(true);
    }
    setModalState({ type: 'closed' });
  }, [modalState]);

  const handleConfirmCancel = useCallback(() => {
    if (modalState.type === 'confirm') {
      modalState.resolve(false);
    }
    setModalState({ type: 'closed' });
  }, [modalState]);

  const value: ModalContextType = {
    showAlert,
    confirm,
  };

  return (
    <ModalContext.Provider value={value}>
      {children}

      {/* Alert ダイアログ */}
      {modalState.type === 'alert' && (
        <Modal
          isOpen={true}
          onClose={handleAlertClose}
          title={modalState.options.title}
          closeOnOverlayClick={true}
          closeOnEscape={true}
        >
          <div className="space-y-4">
            <p className="text-primary-700">{modalState.message}</p>
            <div className="flex justify-end">
              <button
                type="button"
                onClick={handleAlertClose}
                className="px-4 py-2 bg-primary-900 text-white rounded-md hover:bg-primary-800 transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                {modalState.options.buttonText || 'OK'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Confirm ダイアログ */}
      {modalState.type === 'confirm' && (
        <Modal
          isOpen={true}
          onClose={handleConfirmCancel}
          title={modalState.options.title || '確認'}
          closeOnOverlayClick={false}
          closeOnEscape={true}
        >
          <div className="space-y-4">
            <p className="text-primary-700">{modalState.message}</p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={handleConfirmCancel}
                className="px-4 py-2 bg-white border border-primary-300 text-primary-700 rounded-md hover:bg-primary-50 transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                {modalState.options.cancelText || 'キャンセル'}
              </button>
              <button
                type="button"
                onClick={handleConfirmOk}
                className={`px-4 py-2 rounded-md transition-colors focus:outline-none focus:ring-2 ${
                  modalState.options.danger
                    ? 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500'
                    : 'bg-primary-900 text-white hover:bg-primary-800 focus:ring-primary-500'
                }`}
              >
                {modalState.options.confirmText || 'OK'}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </ModalContext.Provider>
  );
}

export function useModal(): ModalContextType {
  const context = useContext(ModalContext);
  if (context === undefined) {
    throw new Error('useModal must be used within a ModalProvider');
  }
  return context;
}
