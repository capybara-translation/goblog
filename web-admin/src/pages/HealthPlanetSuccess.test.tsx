import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { HealthPlanetSuccess } from './HealthPlanetSuccess';
import { apiClient } from '../api/client';

vi.mock('../api/client', () => ({
  apiClient: {
    exchangeHealthPlanetCode: vi.fn(),
  },
}));

function renderWithCode(query: string) {
  return render(
    <MemoryRouter initialEntries={[`/healthplanet/success${query}`]}>
      <Routes>
        <Route path="/healthplanet/success" element={<HealthPlanetSuccess />} />
        <Route path="/healthplanet" element={<div>連携ページ</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('HealthPlanetSuccess', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  it('shows an error when code is missing', () => {
    renderWithCode('');
    expect(screen.getByText('認可コードが見つかりません')).toBeInTheDocument();
  });

  it('does NOT auto-exchange; requires explicit confirmation', () => {
    sessionStorage.setItem('hp_oauth_pending', '1');
    renderWithCode('?code=abc123');
    expect(apiClient.exchangeHealthPlanetCode).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: '連携を完了する' })).toBeInTheDocument();
  });

  it('exchanges the code and navigates on confirm', async () => {
    sessionStorage.setItem('hp_oauth_pending', '1');
    vi.mocked(apiClient.exchangeHealthPlanetCode).mockResolvedValue(undefined);
    renderWithCode('?code=abc123');
    fireEvent.click(screen.getByRole('button', { name: '連携を完了する' }));
    await waitFor(() => {
      expect(apiClient.exchangeHealthPlanetCode).toHaveBeenCalledWith('abc123');
      expect(screen.getByText('連携ページ')).toBeInTheDocument();
    });
  });

  it('shows the error when exchange fails', async () => {
    sessionStorage.setItem('hp_oauth_pending', '1');
    vi.mocked(apiClient.exchangeHealthPlanetCode).mockRejectedValue(new Error('token exchange failed'));
    renderWithCode('?code=expired');
    fireEvent.click(screen.getByRole('button', { name: '連携を完了する' }));
    await waitFor(() => {
      expect(screen.getByText(/token exchange failed/)).toBeInTheDocument();
    });
  });

  it('shows a warning and hides the confirm button when code is present but nonce is missing', () => {
    // No sessionStorage.setItem('hp_oauth_pending') — simulates a crafted link
    // arriving in a browser that never started the OAuth flow here.
    renderWithCode('?code=crafted-code');
    expect(
      screen.getByText(
        'この連携フローはこのブラウザで開始されたものではありません。連携ページからやり直してください。'
      )
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '連携を完了する' })).not.toBeInTheDocument();
    expect(apiClient.exchangeHealthPlanetCode).not.toHaveBeenCalled();
  });
});
