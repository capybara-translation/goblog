import { useState } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router-dom';
import { apiClient } from '../api/client';

/**
 * Landing page for the Health Planet OAuth redirect
 * (/admin/healthplanet/success?code=...).
 *
 * The exchange is deliberately NOT automatic: requiring an explicit click
 * prevents a crafted link (with an attacker's code) from silently binding a
 * foreign Health Planet account to the blog.
 */
export function HealthPlanetSuccess() {
  const [params] = useSearchParams();
  const code = params.get('code') ?? '';
  const navigate = useNavigate();
  const [error, setError] = useState('');
  const [isBusy, setIsBusy] = useState(false);

  const handleComplete = async () => {
    setIsBusy(true);
    setError('');
    try {
      await apiClient.exchangeHealthPlanetCode(code);
      navigate('/healthplanet');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to complete authorization');
      setIsBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-primary-900">Health Planet 連携の完了</h1>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {!code ? (
        <div className="bg-white rounded-lg shadow-sm p-6 space-y-4">
          <p className="text-primary-700">認可コードが見つかりません</p>
          <Link to="/healthplanet" className="text-sm font-medium text-primary-700 underline">
            連携ページに戻る
          </Link>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm p-6 space-y-4">
          <p className="text-sm text-primary-600">
            Health Planet から認可コードを受け取りました。下のボタンを押すと連携が完了します
            （コードの有効期限は10分です）。
          </p>
          <button
            onClick={handleComplete}
            disabled={isBusy}
            className="text-sm font-medium px-4 py-2 rounded bg-primary-900 text-white hover:bg-primary-700 disabled:opacity-50"
          >
            連携を完了する
          </button>
        </div>
      )}
    </div>
  );
}
