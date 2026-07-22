import { useState, useEffect, useCallback } from 'react';
import { apiClient, HealthPlanetStatus } from '../api/client';

function formatDate(iso: string | null | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString();
}

export function HealthPlanet() {
  const [status, setStatus] = useState<HealthPlanetStatus | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setIsLoading(true);
    setError('');
    try {
      setStatus(await apiClient.getHealthPlanetStatus());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to load');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleConnect = async () => {
    setError('');
    try {
      const { url } = await apiClient.getHealthPlanetAuthURL();
      // Mark that this browser session initiated the OAuth flow so
      // HealthPlanetSuccess can reject codes that arrive without this flag
      // (e.g. a crafted link sent by a third party). This is a substitute for
      // the OAuth state parameter, which the Health Planet API does not support.
      sessionStorage.setItem('hp_oauth_pending', '1');
      window.location.href = url;
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to start authorization');
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-primary-900">Health Planet 連携</h1>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {isLoading ? (
        <p className="text-primary-600">読み込み中...</p>
      ) : !status?.enabled ? (
        <div className="bg-white rounded-lg shadow-sm p-8 text-center text-primary-500">
          Health Planet 連携は無効です（HEALTHPLANET_ENABLED を設定してください）
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm p-6 space-y-4">
          <div className="flex items-center gap-3">
            <span className="text-sm text-primary-600">状態</span>
            {status.authorized ? (
              <span className="inline-block text-xs bg-green-100 text-green-700 rounded px-2 py-0.5">
                連携済み
              </span>
            ) : (
              <span className="inline-block text-xs bg-primary-100 text-primary-700 rounded px-2 py-0.5">
                未連携
              </span>
            )}
          </div>

          {status.authorized && (
            <dl className="text-sm text-primary-600 space-y-1">
              <div className="flex gap-2">
                <dt className="font-medium">トークン失効日時:</dt>
                <dd>{formatDate(status.token_expires_at)}</dd>
              </div>
              <div className="flex gap-2">
                <dt className="font-medium">最終リフレッシュ（≒最終同期成功）:</dt>
                <dd>{formatDate(status.last_refreshed_at)}</dd>
              </div>
            </dl>
          )}

          <p className="text-sm text-primary-600">
            {status.authorized
              ? '毎日 0:00 に体重・血圧が自動で同期されます。トークンが失効した場合は再認可してください。'
              : '連携すると、Health Planet の体重・血圧データが毎日 0:00 に自動で取り込まれます。'}
          </p>

          <button
            onClick={handleConnect}
            className="text-sm font-medium px-4 py-2 rounded bg-primary-900 text-white hover:bg-primary-700"
          >
            {status.authorized ? '再認可する' : '連携する'}
          </button>
        </div>
      )}
    </div>
  );
}
