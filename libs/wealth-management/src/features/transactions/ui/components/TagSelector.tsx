"use client";

import { X } from "lucide-react";

interface TagSelectorProps {
  selectedTags: string[];
  availableTags: string[];
  addTag: (tag: string) => void;
  removeTag: (tag: string) => void;
}

export function TagSelector({ 
  selectedTags, 
  availableTags, 
  addTag, 
  removeTag 
}: TagSelectorProps) {
  return (
    <div className="space-y-1">
      <label className="text-sm font-medium">Tags</label>

      {/* Selected tag chips */}
      {selectedTags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-1.5">
          {selectedTags.map(tag => (
            <span
              key={tag}
              className="inline-flex items-center gap-1 bg-primary/10 text-primary text-xs font-medium px-2 py-0.5 rounded-full"
            >
              {tag}
              <button
                type="button"
                onClick={() => removeTag(tag)}
                className="hover:text-destructive transition-colors cursor-pointer"
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
      )}

      {/* Dynamic select from existing tags in the sheet */}
      <select
        value=""
        onChange={e => { addTag(e.target.value); (e.target as HTMLSelectElement).value = ""; }}
        className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        disabled={availableTags.length === 0}
      >
        <option value="">
          {availableTags.length === 0 ? "Thinking…" : "Add a tag…"}
        </option>
        {availableTags.filter(t => !selectedTags.includes(t)).map(tag => (
          <option key={tag} value={tag}>{tag}</option>
        ))}
      </select>
    </div>
  );
}
