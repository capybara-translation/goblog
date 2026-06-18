import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { DeviceList } from './DeviceList';
import { apiClient } from '../api/client';
import type { Device } from '../api/client';

// Mirror the ReactionTypeList pattern: mock the apiClient module directly
vi.mock('../api/client', () => ({
  apiClient: {
    getDevices: vi.fn(),
    revokeDevice: vi.fn(),
    logoutOtherDevices: vi.fn(),
  },
}));

const otherDevice: Device = {
  kind: 'remember',
  id: 'sel-1',
  device: 'iPhone',
  browser: 'Safari',
  os: 'iOS',
  ip: '203.0.113.5',
  last_used_at: '2026-05-23T00:00:00Z',
  created_at: '2026-05-01T00:00:00Z',
  is_current: false,
  is_ephemeral: false,
};

const currentDevice: Device = {
  kind: 'session',
  id: 'sess-2',
  device: 'Mac',
  browser: 'Chrome',
  os: 'macOS',
  ip: '203.0.113.6',
  last_used_at: '2026-05-22T00:00:00Z',
  created_at: '2026-05-01T00:00:00Z',
  is_current: true,
  is_ephemeral: false,
};

describe('DeviceList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows loading indicator initially', () => {
    vi.mocked(apiClient.getDevices).mockReturnValue(new Promise(() => {}));
    render(<DeviceList />);
    expect(screen.getByText('読み込み中...')).toBeInTheDocument();
  });

  it('renders devices and marks the current one', async () => {
    vi.mocked(apiClient.getDevices).mockResolvedValue([otherDevice, currentDevice]);
    render(<DeviceList />);

    await waitFor(() => expect(screen.getByText('iPhone')).toBeInTheDocument());
    expect(screen.getByText('Mac')).toBeInTheDocument();
    expect(screen.getByText('この端末')).toBeInTheDocument();
    expect(screen.getByText('ログアウト')).toBeInTheDocument();
  });

  it('shows error when load fails', async () => {
    vi.mocked(apiClient.getDevices).mockRejectedValue(new Error('fetch failed'));
    render(<DeviceList />);
    await waitFor(() => expect(screen.getByText('fetch failed')).toBeInTheDocument());
  });

  it('shows default error for non-Error rejection', async () => {
    vi.mocked(apiClient.getDevices).mockRejectedValue('oops');
    render(<DeviceList />);
    await waitFor(() => expect(screen.getByText('failed to load')).toBeInTheDocument());
  });

  it('shows empty state when there are no devices', async () => {
    vi.mocked(apiClient.getDevices).mockResolvedValue([]);
    render(<DeviceList />);
    await waitFor(() => expect(screen.getByText('端末がありません')).toBeInTheDocument());
  });

  it('shows the ephemeral note for ephemeral devices', async () => {
    const ephemeral: Device = { ...otherDevice, is_ephemeral: true };
    vi.mocked(apiClient.getDevices).mockResolvedValue([ephemeral]);
    render(<DeviceList />);
    await waitFor(() => expect(screen.getByText('iPhone')).toBeInTheDocument());
    expect(screen.getByText(/一時セッション・再起動で消えます/)).toBeInTheDocument();
  });

  it('renders — for empty browser and ip', async () => {
    const blank: Device = { ...otherDevice, browser: '', ip: '' };
    vi.mocked(apiClient.getDevices).mockResolvedValue([blank]);
    render(<DeviceList />);
    await waitFor(() => expect(screen.getByText('iPhone')).toBeInTheDocument());
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(2);
  });

  describe('revoke', () => {
    it('calls revokeDevice and reloads after confirmation', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(true);
      vi.mocked(apiClient.getDevices)
        .mockResolvedValueOnce([otherDevice, currentDevice])
        .mockResolvedValueOnce([currentDevice]);
      vi.mocked(apiClient.revokeDevice).mockResolvedValue(undefined);

      render(<DeviceList />);
      await waitFor(() => expect(screen.getByText('iPhone')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: 'ログアウト' }));

      await waitFor(() =>
        expect(apiClient.revokeDevice).toHaveBeenCalledWith('remember', 'sel-1'),
      );
      await waitFor(() => expect(screen.queryByText('iPhone')).not.toBeInTheDocument());
    });

    it('does NOT call revokeDevice when confirm is cancelled', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(false);
      vi.mocked(apiClient.getDevices).mockResolvedValue([otherDevice, currentDevice]);

      render(<DeviceList />);
      await waitFor(() => expect(screen.getByText('iPhone')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: 'ログアウト' }));

      expect(apiClient.revokeDevice).not.toHaveBeenCalled();
    });
  });

  describe('logout others', () => {
    it('calls logoutOtherDevices and reloads after confirmation', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(true);
      vi.mocked(apiClient.getDevices)
        .mockResolvedValueOnce([otherDevice, currentDevice])
        .mockResolvedValueOnce([currentDevice]);
      vi.mocked(apiClient.logoutOtherDevices).mockResolvedValue(undefined);

      render(<DeviceList />);
      await waitFor(() => expect(screen.getByText('iPhone')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: '他の端末をすべてログアウト' }));

      await waitFor(() => expect(apiClient.logoutOtherDevices).toHaveBeenCalled());
      await waitFor(() => expect(screen.queryByText('iPhone')).not.toBeInTheDocument());
    });

    it('does NOT call logoutOtherDevices when confirm is cancelled', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(false);
      vi.mocked(apiClient.getDevices).mockResolvedValue([otherDevice, currentDevice]);

      render(<DeviceList />);
      await waitFor(() => expect(screen.getByText('iPhone')).toBeInTheDocument());

      fireEvent.click(screen.getByRole('button', { name: '他の端末をすべてログアウト' }));

      expect(apiClient.logoutOtherDevices).not.toHaveBeenCalled();
    });
  });
});
