import { useState, useEffect, useCallback } from 'react';
import {
  apiClient,
  ReactionType,
  ReactionTypeCreateRequest,
} from '../api/client';
import { Modal } from '../components/Modal';

interface FormState {
  id: number | null; // null = create
  emoji: string;
  label: string;
  sort_order: number;
  is_active: boolean;
  is_seed: boolean;
}

const emptyForm: FormState = {
  id: null,
  emoji: '',
  label: '',
  sort_order: 0,
  is_active: true,
  is_seed: false,
};

export function ReactionTypeList() {
  const [types, setTypes] = useState<ReactionType[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [form, setForm] = useState<FormState | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError('');
    try {
      setTypes(await apiClient.getReactionTypes());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to load');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const openCreate = () => setForm({ ...emptyForm });

  const openEdit = (t: ReactionType) =>
    setForm({
      id: t.id,
      emoji: t.emoji,
      label: t.label,
      sort_order: t.sort_order,
      is_active: t.is_active,
      is_seed: t.is_seed,
    });

  const save = async () => {
    if (!form) return;
    setError('');
    if (!form.emoji.trim() || !form.label.trim()) {
      setError('絵文字とラベルは必須です');
      return;
    }
    setIsSaving(true);
    try {
      if (form.id == null) {
        const req: ReactionTypeCreateRequest = {
          emoji: form.emoji,
          label: form.label,
          sort_order: form.sort_order,
        };
        await apiClient.createReactionType(req);
      } else {
        await apiClient.updateReactionType(form.id, {
          emoji: form.emoji,
          label: form.label,
          sort_order: form.sort_order,
          is_active: form.is_active,
        });
      }
      setForm(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'save failed');
    } finally {
      setIsSaving(false);
    }
  };

  const remove = async (t: ReactionType) => {
    if (!window.confirm(`「${t.emoji} ${t.label}」を削除しますか？`)) return;
    setError('');
    try {
      await apiClient.deleteReactionType(t.id);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'delete failed');
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-primary-900">リアクション</h1>
        <button
          onClick={openCreate}
          className="text-sm font-medium px-4 py-2 rounded bg-primary-900 text-white hover:bg-primary-700"
        >
          新規追加
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
              <th className="py-2">絵文字</th>
              <th className="py-2">ラベル</th>
              <th className="py-2">並び順</th>
              <th className="py-2">状態</th>
              <th className="py-2"></th>
            </tr>
          </thead>
          <tbody>
            {types.map((t) => (
              <tr
                key={t.id}
                className={`border-b border-primary-100 ${t.is_active ? '' : 'opacity-50'}`}
              >
                <td className="py-2 text-xl">{t.emoji}</td>
                <td className="py-2">{t.label}</td>
                <td className="py-2">{t.sort_order}</td>
                <td className="py-2">{t.is_active ? '有効' : '無効'}</td>
                <td className="py-2 text-right space-x-2">
                  <button
                    onClick={() => openEdit(t)}
                    className="text-sm text-primary-700 hover:text-primary-900"
                  >
                    編集
                  </button>
                  <button
                    onClick={() => remove(t)}
                    disabled={t.is_seed}
                    title={
                      t.is_seed
                        ? '組み込みのリアクションは削除できません（無効化のみ）'
                        : ''
                    }
                    className="text-sm text-red-600 hover:text-red-800 disabled:opacity-30 disabled:cursor-not-allowed"
                  >
                    削除
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {form && (
        <Modal
          isOpen={true}
          onClose={() => setForm(null)}
          title={form.id == null ? 'リアクションを追加' : 'リアクションを編集'}
        >
          <div className="space-y-4">
            <div className="block">
              <label htmlFor="rt-emoji" className="text-sm text-primary-700">絵文字</label>
              <input
                id="rt-emoji"
                value={form.emoji}
                disabled={form.is_seed}
                onChange={(e) => setForm({ ...form, emoji: e.target.value })}
                title={
                  form.is_seed
                    ? '組み込みのリアクションは絵文字を変更できません'
                    : ''
                }
                className="mt-1 block w-full rounded border border-primary-300 px-3 py-2 disabled:bg-primary-50"
              />
            </div>
            <div className="block">
              <label htmlFor="rt-label" className="text-sm text-primary-700">ラベル</label>
              <input
                id="rt-label"
                value={form.label}
                onChange={(e) => setForm({ ...form, label: e.target.value })}
                className="mt-1 block w-full rounded border border-primary-300 px-3 py-2"
              />
            </div>
            <div className="block">
              <label htmlFor="rt-sort" className="text-sm text-primary-700">並び順</label>
              <input
                id="rt-sort"
                type="number"
                value={form.sort_order}
                onChange={(e) => {
                  const n = Number(e.target.value);
                  setForm({ ...form, sort_order: Number.isNaN(n) ? 0 : n });
                }}
                className="mt-1 block w-full rounded border border-primary-300 px-3 py-2"
              />
            </div>
            {form.id != null && (
              <div className="flex items-center gap-2">
                <input
                  id="rt-active"
                  type="checkbox"
                  checked={form.is_active}
                  onChange={(e) =>
                    setForm({ ...form, is_active: e.target.checked })
                  }
                />
                <label htmlFor="rt-active" className="text-sm text-primary-700">有効</label>
              </div>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setForm(null)}
                className="px-4 py-2 text-sm text-primary-700"
              >
                キャンセル
              </button>
              <button
                onClick={save}
                disabled={isSaving}
                className="px-4 py-2 text-sm rounded bg-primary-900 text-white hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isSaving ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
