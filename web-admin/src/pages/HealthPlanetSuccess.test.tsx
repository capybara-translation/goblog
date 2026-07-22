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
  });

  it('shows an error when code is missing', () => {
    renderWithCode('');
    expect(screen.getByText('認可コードが見つかりません')).toBeInTheDocument();
  });

  it('does NOT auto-exchange; requires explicit confirmation', () => {
    renderWithCode('?code=abc123');
    expect(apiClient.exchangeHealthPlanetCode).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: '連携を完了する' })).toBeInTheDocument();
  });

  it('exchanges the code and navigates on confirm', async () => {
    vi.mocked(apiClient.exchangeHealthPlanetCode).mockResolvedValue(undefined);
    renderWithCode('?code=abc123');
    fireEvent.click(screen.getByRole('button', { name: '連携を完了する' }));
    await waitFor(() => {
      expect(apiClient.exchangeHealthPlanetCode).toHaveBeenCalledWith('abc123');
      expect(screen.getByText('連携ページ')).toBeInTheDocument();
    });
  });

  it('shows the error when exchange fails', async () => {
    vi.mocked(apiClient.exchangeHealthPlanetCode).mockRejectedValue(new Error('token exchange failed'));
    renderWithCode('?code=expired');
    fireEvent.click(screen.getByRole('button', { name: '連携を完了する' }));
    await waitFor(() => {
      expect(screen.getByText(/token exchange failed/)).toBeInTheDocument();
    });
  });
});
