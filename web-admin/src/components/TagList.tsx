interface TagListProps {
  tags: string;
}

export function TagList({ tags }: TagListProps) {
  if (!tags || !tags.trim()) {
    return <span className="text-sm text-primary-400">タグなし</span>;
  }

  const tagArray = tags.split(',').map((tag) => tag.trim()).filter(Boolean);

  return (
    <div className="flex flex-wrap gap-1.5">
      {tagArray.map((tag, index) => (
        <span
          key={index}
          className="inline-flex items-center px-2 py-1 rounded text-xs font-medium bg-blue-50 text-blue-700"
        >
          {tag}
        </span>
      ))}
    </div>
  );
}
