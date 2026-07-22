import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { HealthPlanet } from './HealthPlanet';
import { apiClient } from '../api/client';

vi.mock('../api/client', () => ({
  apiClient: {
    getHealthPlanetStatus: vi.fn(),
    getHealthPlanetAuthURL: vi.fn(),
  },
}));

function renderPage() {
  return render(
    <MemoryRouter>
      <HealthPlanet />
    </MemoryRouter>
  );
}

describe('HealthPlanet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows unauthorized state with connect button', async () => {
    vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({
      enabled: true,
      authorized: false,
      token_expires_at: null,
      last_refreshed_at: null,
    });
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('未連携')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: '連携する' })).toBeInTheDocument();
  });

  it('shows authorized state with expiry and re-auth button', async () => {
    vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({
      enabled: true,
      authorized: true,
      token_expires_at: '2026-08-21T00:00:00+09:00',
      last_refreshed_at: '2026-07-22T00:00:00+09:00',
    });
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('連携済み')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: '再認可する' })).toBeInTheDocument();
  });

  it('redirects to the auth URL when connect is clicked and sets the oauth pending nonce', async () => {
    vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({
      enabled: true,
      authorized: false,
    });
    vi.mocked(apiClient.getHealthPlanetAuthURL).mockResolvedValue({
      url: 'https://www.healthplanet.jp/oauth/auth?x=1',
    });
    // window.location.href への代入を捕捉
    const original = window.location;
    Object.defineProperty(window, 'location', {
      writable: true,
      value: { ...original, href: original.href },
    });

    renderPage();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: '連携する' })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: '連携する' }));
    await waitFor(() => {
      expect(window.location.href).toBe('https://www.healthplanet.jp/oauth/auth?x=1');
    });
    // The nonce must be set so HealthPlanetSuccess accepts the returning code.
    expect(sessionStorage.getItem('hp_oauth_pending')).toBe('1');

    Object.defineProperty(window, 'location', { writable: true, value: original });
    sessionStorage.clear();
  });
});
