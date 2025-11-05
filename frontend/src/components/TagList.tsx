import { Badge } from './ui/Badge';
import { Tag } from '../types/api';

interface TagListProps {
  tags: Tag[];
  onTagClick?: (tagName: string) => void;
  selectedTag?: string;
}

export function TagList({ tags, onTagClick, selectedTag }: TagListProps) {
  if (!tags.length) {
    return <span className="text-gray-500 text-sm">No tags</span>;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {tags.map((tag) => (
        <Badge
          key={tag.id}
          variant={selectedTag === tag.name ? 'default' : 'secondary'}
          className={`cursor-pointer hover:opacity-80 ${
            tag.color ? `bg-${tag.color}-100 text-${tag.color}-800 border-${tag.color}-300` : ''
          }`}
          onClick={() => onTagClick?.(tag.name)}
        >
          {tag.name}
        </Badge>
      ))}
    </div>
  );
}
