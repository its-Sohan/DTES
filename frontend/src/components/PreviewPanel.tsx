import React, { useState } from 'react';
import { useAppStore, extractItem } from '../store/useAppStore';

export const PreviewPanel: React.FC = () => {
  const { queue, selectedItemId, isExtracting, auditMode, activeBlockIndex } = useAppStore();
  const [rotation, setRotation] = useState<number>(0);
  const [fitMode, setFitMode] = useState<'contain' | 'cover'>('contain');

  const selectedItem = queue.find((i) => i.id === selectedItemId);

  const rotateClockwise = () => {
    setRotation((r) => (r + 90) % 360);
  };

  const resetRotation = () => {
    setRotation(0);
  };

  const toggleFit = () => {
    setFitMode((m) => (m === 'contain' ? 'cover' : 'contain'));
  };

  // Active bounding box for synchronized audit
  const activeBox =
    auditMode && selectedItem?.block_boxes && selectedItem.block_boxes.length > activeBlockIndex
      ? selectedItem.block_boxes[activeBlockIndex]
      : null;

  return (
    <div className="flex-1 flex flex-col h-full bg-desk-light dark:bg-desk-dark p-3 select-none relative overflow-hidden">
      <div className="flex-1 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light dark:bg-inset-dark relative flex items-center justify-center overflow-hidden shadow-inner">
        {selectedItem ? (
          <div className="relative w-full h-full flex items-center justify-center p-4">
            {/* Document Image Viewport */}
            <div className="relative max-w-full max-h-full flex items-center justify-center">
              <img
                src={selectedItem.file_path ? `/${selectedItem.file_path}` : ''}
                alt={selectedItem.file_name}
                style={{
                  transform: `rotate(${rotation}deg)`,
                  transition: 'transform 250ms ease-out',
                }}
                className={`max-h-[calc(100vh-10rem)] max-w-full rounded shadow-md object-${fitMode}`}
                onError={(e) => {
                  // Fallback for local files: render placeholder card if image path is not servable via webview direct URL
                  const target = e.currentTarget;
                  target.style.display = 'none';
                  const parent = target.parentElement;
                  if (parent && !parent.querySelector('.img-fallback')) {
                    const fallback = document.createElement('div');
                    fallback.className = 'img-fallback flex flex-col items-center justify-center p-12 text-center text-ink-secondaryLight dark:text-ink-secondaryDark font-mono text-xs';
                    fallback.innerHTML = `<span class="text-sm font-semibold text-ink-primaryLight dark:text-ink-primaryDark mb-1">${selectedItem.file_name}</span><span>${selectedItem.file_path}</span>`;
                    parent.appendChild(fallback);
                  }
                }}
              />

              {/* Synchronized Audit Focus Guide Box */}
              {activeBox && (
                <div
                  className="absolute pointer-events-none border-2 border-brand-light dark:border-brand-dark bg-brand-light/15 dark:bg-brand-dark/20 rounded transition-all duration-200 z-10"
                  style={{
                    top: `${activeBox.ymin / 10}%`,
                    left: `${activeBox.xmin / 10}%`,
                    width: `${(activeBox.xmax - activeBox.xmin) / 10}%`,
                    height: `${(activeBox.ymax - activeBox.ymin) / 10}%`,
                  }}
                >
                  <span className="absolute -top-5 left-0 font-mono text-[9px] font-bold px-1.5 py-0.5 rounded bg-brand-light dark:bg-brand-dark text-white uppercase tracking-wider">
                    LINE {String(activeBlockIndex + 1).padStart(2, '0')}
                  </span>
                </div>
              )}
            </div>

            {/* Signature Motion: Precision Scan-Line Sweep */}
            {isExtracting && (
              <div className="absolute inset-x-4 pointer-events-none z-20">
                <div className="absolute inset-x-0 h-[2px] bg-brand-light dark:bg-brand-dark shadow-[0_0_12px_2px_rgba(37,99,235,0.75)] animate-scan-sweep" />
              </div>
            )}

            {/* Floating Glass Toolbar */}
            <div className="absolute bottom-4 right-4 flex items-center space-x-1 px-2.5 py-1.5 rounded-glass bg-surface-light/80 dark:bg-surface-dark/80 backdrop-blur-md border border-hairline-light dark:border-hairline-dark shadow-lg z-30">
              <button
                onClick={rotateClockwise}
                title="Rotate 90° clockwise"
                className="p-1.5 rounded text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark transition-colors"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              </button>

              <button
                onClick={resetRotation}
                title="Reset rotation"
                className="p-1.5 rounded text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark transition-colors"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12.066 11.2a1 1 0 000 1.6l5.334 4A1 1 0 0019 16V8a1 1 0 00-1.6-.8l-5.333 4zM4.066 11.2a1 1 0 000 1.6l5.334 4A1 1 0 0011 16V8a1 1 0 00-1.6-.8l-5.334 4z" />
                </svg>
              </button>

              <button
                onClick={toggleFit}
                title={`Toggle fit mode (${fitMode})`}
                className="p-1.5 rounded text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark transition-colors"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                </svg>
              </button>

              <div className="w-[1px] h-4 bg-hairline-light dark:bg-hairline-dark mx-1" />

              <button
                onClick={() => extractItem(selectedItem)}
                title="Re-extract document"
                className="p-1.5 rounded text-brand-light dark:text-brand-dark hover:opacity-80 transition-opacity"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </button>
            </div>
          </div>
        ) : (
          /* Empty State Invitation */
          <div className="flex flex-col items-center justify-center p-8 text-center max-w-sm">
            <div className="w-14 h-14 rounded-full bg-brand-light/10 dark:bg-brand-dark/15 border border-brand-light/20 dark:border-brand-dark/30 flex items-center justify-center mb-3">
              <svg className="w-6 h-6 text-brand-light dark:text-brand-dark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <h3 className="text-sm font-semibold text-ink-primaryLight dark:text-ink-primaryDark mb-1">
              No document selected
            </h3>
            <p className="text-xs text-ink-secondaryLight dark:text-ink-secondaryDark mb-4">
              Select an item from the queue or upload a document to inspect preview and run OCR.
            </p>
            <span className="font-mono text-[10px] uppercase px-3 py-1 rounded-full border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark text-ink-secondaryLight dark:text-ink-secondaryDark">
              PNG · JPG · WebP · PDF
            </span>
          </div>
        )}
      </div>
    </div>
  );
};
