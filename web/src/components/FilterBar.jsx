import React from 'react';

export default function FilterBar({ filters, setFilter }) {
  return (
    <div className="filter-bar">
      <input
        className="filter-input"
        type="text"
        placeholder="Filter by image..."
        value={filters.image}
        onChange={(e) => setFilter('image', e.target.value)}
      />
      <input
        className="filter-input"
        type="text"
        placeholder="Filter by node..."
        value={filters.node}
        onChange={(e) => setFilter('node', e.target.value)}
      />
      <input
        className="filter-input"
        type="text"
        placeholder="Filter by namespace..."
        value={filters.namespace}
        onChange={(e) => setFilter('namespace', e.target.value)}
      />
      <input
        className="filter-input"
        type="text"
        placeholder="Filter by pod name..."
        value={filters.pod}
        onChange={(e) => setFilter('pod', e.target.value)}
      />
    </div>
  );
}
