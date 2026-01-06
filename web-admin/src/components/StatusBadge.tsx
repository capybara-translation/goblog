interface StatusBadgeProps {
  status: 'draft' | 'published';
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const isDraft = status === 'draft';

  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
        isDraft
          ? 'bg-primary-100 text-primary-800'
          : 'bg-green-100 text-green-800'
      }`}
    >
      {isDraft ? '下書き' : '公開済み'}
    </span>
  );
}
