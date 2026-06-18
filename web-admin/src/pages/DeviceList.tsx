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
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-primary-900">ログイン中の端末</h1>
        <button
          onClick={handleLogoutOthers}
          className="text-sm font-medium px-4 py-2 rounded bg-red-600 text-white hover:bg-red-800"
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
      ) : (
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-primary-200 text-sm text-primary-600">
              <th className="py-2">端末</th>
              <th className="py-2">ブラウザ</th>
              <th className="py-2">IP</th>
              <th className="py-2">最終利用</th>
              <th className="py-2"></th>
            </tr>
          </thead>
          <tbody>
            {devices.map((d) => (
              <tr key={`${d.kind}:${d.id}`} className="border-b border-primary-100">
                <td className="py-2">
                  {d.device}
                  {d.is_ephemeral && (
                    <span className="ml-2 text-xs text-primary-500">
                      （一時セッション・再起動で消えます）
                    </span>
                  )}
                </td>
                <td className="py-2">{d.browser || '—'}</td>
                <td className="py-2">{d.ip || '—'}</td>
                <td className="py-2">{formatDate(d.last_used_at)}</td>
                <td className="py-2 text-right">
                  {d.is_current ? (
                    <span className="inline-block text-xs bg-primary-100 text-primary-700 rounded px-2 py-0.5">
                      この端末
                    </span>
                  ) : (
                    <button
                      onClick={() => handleRevoke(d)}
                      className="text-sm text-red-600 hover:text-red-800"
                    >
                      ログアウト
                    </button>
                  )}
                </td>
              </tr>
            ))}
            {devices.length === 0 && (
              <tr>
                <td colSpan={5} className="py-4 text-center text-primary-500">
                  端末がありません
                </td>
              </tr>
            )}
          </tbody>
        </table>
      )}
    </div>
  );
}
