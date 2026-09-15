import React from 'react';

export function Logo({ className = "h-7 w-7" }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" className={className} aria-hidden="true">
      <g stroke="#1f2d3b" strokeWidth="2" strokeLinejoin="round">
        {/* Top Face - Cyan */}
        <path d="M16 2 L3 9.5 L16 17 L29 9.5 Z" fill="#5bc8dc" />
        {/* Left Face - Amber */}
        <path d="M3 9.5 L3 24.5 L16 32 L16 17 Z" fill="#f2a33c" />
        {/* Right Face - Green */}
        <path d="M29 9.5 L29 24.5 L16 32 L16 17 Z" fill="#4cc573" />
      </g>
    </svg>
  );
}
