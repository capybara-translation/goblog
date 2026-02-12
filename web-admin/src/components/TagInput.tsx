import { useState, useEffect, useRef, useCallback } from 'react';
import { apiClient, Tag } from '../api/client';

interface TagInputProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

/**
 * Tag input component with autocomplete
 * - Display selected tags as chips (badges)
 * - Show existing tags as dropdown suggestions while typing
 * - Keyboard navigation support (↑↓Enter/Tab/Esc/Backspace)
 * - IME support (no autocomplete during Japanese input)
 */
export function TagInput({ value, onChange, disabled = false, placeholder = 'Enter tags...' }: TagInputProps) {
  // Internal state
  const [inputValue, setInputValue] = useState('');
  const [allTags, setAllTags] = useState<Tag[]>([]);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [isOpen, setIsOpen] = useState(false);
  const [isComposing, setIsComposing] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  // Refs
  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const listboxRef = useRef<HTMLUListElement>(null);

  // Convert string to tag array
  const selectedTags = value
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean);

  // Convert tag array to string
  const tagsToString = (tags: string[]): string => {
    return tags.join(', ');
  };

  // Fetch all tags (on mount only)
  useEffect(() => {
    const loadTags = async () => {
      try {
        setIsLoading(true);
        const tags = await apiClient.getTags();
        setAllTags(tags);
      } catch (error) {
        console.error('Failed to load tags:', error);
      } finally {
        setIsLoading(false);
      }
    };
    loadTags();
  }, []);

  // Filter suggestions
  const filteredSuggestions = allTags.filter((tag) => {
    // Exclude already selected tags
    if (selectedTags.includes(tag.name)) return false;
    // Show all if input is empty
    if (!inputValue.trim()) return true;
    // Partial match (case-insensitive)
    return tag.name.toLowerCase().includes(inputValue.toLowerCase());
  });

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setSelectedIndex(-1);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Scroll selected suggestion into view
  useEffect(() => {
    if (selectedIndex >= 0 && listboxRef.current) {
      const selectedItem = listboxRef.current.children[selectedIndex] as HTMLElement | undefined;
      if (selectedItem && typeof selectedItem.scrollIntoView === 'function') {
        selectedItem.scrollIntoView({ block: 'nearest' });
      }
    }
  }, [selectedIndex]);

  // Add tag
  const addTag = useCallback((tagName: string) => {
    const trimmed = tagName.trim();
    if (!trimmed) return;
    // Check for duplicates
    if (selectedTags.includes(trimmed)) {
      setInputValue('');
      return;
    }
    const newTags = [...selectedTags, trimmed];
    onChange(tagsToString(newTags));
    setInputValue('');
    setSelectedIndex(-1);
    setIsOpen(false);
  }, [selectedTags, onChange]);

  // Remove tag
  const removeTag = useCallback((tagToRemove: string) => {
    const newTags = selectedTags.filter((tag) => tag !== tagToRemove);
    onChange(tagsToString(newTags));
  }, [selectedTags, onChange]);

  // Input change handler
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    // Add tag when comma is entered
    if (newValue.includes(',')) {
      const parts = newValue.split(',');
      const tagToAdd = (parts[0] ?? '').trim();
      if (tagToAdd) {
        addTag(tagToAdd);
      }
      // Keep the part after comma in input
      setInputValue(parts.slice(1).join(',').trimStart());
    } else {
      setInputValue(newValue);
      setIsOpen(true);
      setSelectedIndex(-1);
    }
  };

  // Keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Skip during IME composition
    if (isComposing) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        if (filteredSuggestions.length > 0) {
          setIsOpen(true);
          setSelectedIndex((prev) =>
            prev < filteredSuggestions.length - 1 ? prev + 1 : prev
          );
        }
        break;

      case 'ArrowUp':
        e.preventDefault();
        if (filteredSuggestions.length > 0) {
          setSelectedIndex((prev) => (prev > 0 ? prev - 1 : -1));
        }
        break;

      case 'Enter': {
        e.preventDefault();
        const enterSelection = filteredSuggestions[selectedIndex];
        if (selectedIndex >= 0 && enterSelection) {
          addTag(enterSelection.name);
        } else if (inputValue.trim()) {
          addTag(inputValue);
        }
        break;
      }

      case 'Tab': {
        const tabSelection = filteredSuggestions[selectedIndex];
        if (isOpen && selectedIndex >= 0 && tabSelection) {
          e.preventDefault();
          addTag(tabSelection.name);
        }
        break;
      }

      case 'Escape':
        setIsOpen(false);
        setSelectedIndex(-1);
        break;

      case 'Backspace': {
        const lastTag = selectedTags[selectedTags.length - 1];
        if (!inputValue && lastTag) {
          removeTag(lastTag);
        }
        break;
      }
    }
  };

  // IME composition handlers
  const handleCompositionStart = () => {
    setIsComposing(true);
  };

  const handleCompositionEnd = () => {
    setIsComposing(false);
  };

  // Suggestion click handler
  const handleSuggestionClick = (tagName: string) => {
    addTag(tagName);
    inputRef.current?.focus();
  };

  // Open dropdown on focus
  const handleFocus = () => {
    setIsOpen(true);
  };

  // Close dropdown on blur
  const handleBlur = () => {
    setIsOpen(false);
    setSelectedIndex(-1);
  };

  return (
    <div ref={containerRef} className="relative">
      {/* Tag chips + input field */}
      <div
        className={`flex flex-wrap gap-1.5 items-center min-h-[42px] px-3 py-2 border rounded-md ${
          disabled
            ? 'bg-gray-100 border-primary-200 cursor-not-allowed'
            : 'border-primary-300 focus-within:ring-2 focus-within:ring-primary-500 focus-within:border-primary-500'
        }`}
        onClick={() => !disabled && inputRef.current?.focus()}
      >
        {/* Selected tags */}
        {selectedTags.map((tag) => (
          <span
            key={tag}
            className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium bg-blue-50 text-blue-700"
          >
            {tag}
            {!disabled && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  removeTag(tag);
                }}
                className="hover:text-blue-900 focus:outline-none"
                aria-label={`Remove ${tag}`}
              >
                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </span>
        ))}

        {/* Input field */}
        <input
          ref={inputRef}
          type="text"
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onCompositionStart={handleCompositionStart}
          onCompositionEnd={handleCompositionEnd}
          onFocus={handleFocus}
          onBlur={handleBlur}
          disabled={disabled}
          placeholder={selectedTags.length === 0 ? placeholder : ''}
          className="flex-1 min-w-[120px] outline-none bg-transparent text-sm disabled:cursor-not-allowed"
          aria-label="Tag input"
          aria-expanded={isOpen}
          aria-haspopup="listbox"
        />
      </div>

      {/* Dropdown suggestions */}
      {isOpen && !disabled && filteredSuggestions.length > 0 && (
        <ul
          ref={listboxRef}
          className="absolute z-10 w-full mt-1 bg-white border border-primary-200 rounded-md shadow-lg max-h-60 overflow-auto"
          role="listbox"
        >
          {filteredSuggestions.map((tag, index) => (
            <li
              key={tag.name}
              role="option"
              aria-selected={index === selectedIndex}
              className={`px-3 py-2 cursor-pointer flex justify-between items-center text-sm ${
                index === selectedIndex
                  ? 'bg-primary-100 text-primary-900'
                  : 'hover:bg-primary-50 text-primary-700'
              }`}
              onMouseDown={() => handleSuggestionClick(tag.name)}
              onMouseEnter={() => setSelectedIndex(index)}
            >
              <span>{tag.name}</span>
              <span className="text-xs text-primary-400 font-sans">({tag.count})</span>
            </li>
          ))}
        </ul>
      )}

      {/* Loading indicator */}
      {isLoading && (
        <div className="absolute right-3 top-1/2 -translate-y-1/2">
          <svg className="animate-spin h-4 w-4 text-primary-400" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        </div>
      )}
    </div>
  );
}
