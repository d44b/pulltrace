import React from 'react';

const SearchIcon = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
  </svg>
);

const FIELDS = [
  { key: 'image',     placeholder: 'Image…'     },
  { key: 'node',      placeholder: 'Node…'      },
  { key: 'namespace', placeholder: 'Namespace…' },
  { key: 'pod',       placeholder: 'Pod…'       },
];

export default function FilterBar({ filters, setFilter }) {
  return (
    <div className="filter-bar">
      {FIELDS.map(({ key, placeholder }) => (
        <div className="filter-input-wrap" key={key}>
          <span className="filter-icon"><SearchIcon /></span>
          <input
            className="filter-input"
            type="text"
            placeholder={placeholder}
            value={filters[key]}
            onChange={(e) => setFilter(key, e.target.value)}
          />
        </div>
      ))}
    </div>
  );
}
