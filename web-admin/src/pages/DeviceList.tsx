import { useState, useEffect, useCallback } from 'react';
import { apiClient, Device } from '../api/client';

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString();
}

export function DeviceList() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setIsLoading(true);
    setError('');
    try {
      setDevices(await apiClient.getDevices());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to load');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleRevoke = async (d: Device) => {
    if (!window.confirm('この端末をログアウトさせますか？')) return;
    setError('');
    try {
      await apiClient.revokeDevice(d.kind, d.id);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to revoke device');
    }
  };

  const handleLogoutOthers = async () => {
    if (!window.confirm('現在の端末以外をすべてログアウトさせますか？')) return;
    setError('');
    try {
      await apiClient.logoutOtherDevices();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to log out other devices');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-bold text-primary-900">ログイン中の端末</h1>
        <button
          onClick={handleLogoutOthers}
          className="self-start sm:self-auto text-sm font-medium px-4 py-2 rounded bg-red-600 text-white hover:bg-red-800"
        >
          他の端末をすべてログアウト
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {isLoading ? (
        <p className="text-primary-600">読み込み中...</p>
      ) : devices.length === 0 ? (
        <div className="bg-white rounded-lg shadow-sm p-8 text-center text-primary-500">
          端末がありません
        </div>
      ) : (
        // Table on >=sm; on mobile each row collapses into a card (mirrors PostList).
        <div className="sm:bg-white sm:rounded-lg sm:shadow-sm sm:overflow-x-auto max-sm:overflow-x-hidden">
          <table className="min-w-full text-left max-sm:block sm:divide-y sm:divide-primary-200">
            <thead className="bg-primary-50 max-sm:hidden">
              <tr className="text-sm text-primary-600">
                <th className="px-4 py-2 font-medium">端末</th>
                <th className="px-4 py-2 font-medium">ブラウザ</th>
                <th className="px-4 py-2 font-medium">IP</th>
                <th className="px-4 py-2 font-medium">最終利用</th>
                <th className="px-4 py-2"></th>
              </tr>
            </thead>
            <tbody className="bg-white sm:divide-y sm:divide-primary-200 max-sm:block max-sm:bg-transparent max-sm:space-y-3">
              {devices.map((d) => (
                <tr
                  key={`${d.kind}:${d.id}`}
                  className="sm:hover:bg-primary-50 transition-colors max-sm:block max-sm:bg-white max-sm:rounded-lg max-sm:shadow-sm max-sm:p-4 max-sm:space-y-1.5"
                >
                  <td className="px-4 py-2 max-sm:p-0 max-sm:block">
                    <span className="max-sm:text-base max-sm:font-semibold max-sm:break-words">
                      {d.device}
                    </span>
                    {d.is_ephemeral && (
                      <span className="ml-2 text-xs text-primary-500">
                        （一時セッション・再起動で消えます）
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-2 text-sm text-primary-600 max-sm:p-0 max-sm:block max-sm:before:content-['ブラウザ_'] max-sm:before:text-xs max-sm:before:text-primary-500 max-sm:before:font-medium max-sm:before:mr-2">
                    {d.browser || '—'}
                  </td>
                  <td className="px-4 py-2 text-sm text-primary-600 max-sm:p-0 max-sm:block max-sm:break-all max-sm:before:content-['IP_'] max-sm:before:text-xs max-sm:before:text-primary-500 max-sm:before:font-medium max-sm:before:mr-2">
                    {d.ip || '—'}
                  </td>
                  <td className="px-4 py-2 text-sm text-primary-600 max-sm:p-0 max-sm:block max-sm:before:content-['最終利用_'] max-sm:before:text-xs max-sm:before:text-primary-500 max-sm:before:font-medium max-sm:before:mr-2">
                    {formatDate(d.last_used_at)}
                  </td>
                  <td className="px-4 py-2 text-right max-sm:p-0 max-sm:block max-sm:text-left max-sm:pt-1">
                    {d.is_current ? (
                      <span className="inline-block text-xs bg-primary-100 text-primary-700 rounded px-2 py-0.5">
                        この端末
                      </span>
                    ) : (
                      <button
                        onClick={() => handleRevoke(d)}
                        className="text-sm font-medium text-red-600 hover:text-red-800"
                      >
                        ログアウト
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
