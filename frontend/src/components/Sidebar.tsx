import React from 'react';
import {
  useAppStore,
  selectItem,
  removeItem,
  clearCompleted,
  addItems,
  runAllPending,
} from '../store/useAppStore';
import * as api from '../../wailsjs/go/main/App';

export const Sidebar: React.FC = () => {
  const { queue, selectedItemId, isProcessingAll, sidebarOpen } = useAppStore();

  if (!sidebarOpen) return null;

  const handleBrowse = async () => {
    try {
      const items = await api.PickFiles();
      if (items && items.length > 0) {
        addItems(items);
      }
    } catch (err) {
      console.error('Browse error:', err);
    }
  };

  const handleScan = async () => {
    try {
      const item = await api.ScanDocument();
      if (item) {
        addItems([item]);
      }
    } catch (err: any) {
      alert(err?.message || 'Scanning failed');
    }
  };

  const handlePaste = async () => {
    try {
      const item = await api.GetClipboardImage();
      if (item) {
        addItems([item]);
      }
    } catch (err: any) {
      alert(err?.message || 'No image found on clipboard');
    }
  };

  return (
    <aside className="w-72 border-r border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark flex flex-col h-[calc(100vh-3rem)] select-none">
      {/* Ingestion Actions Header */}
      <div className="p-3 border-b border-hairline-light dark:border-hairline-dark space-y-2">
        <div className="text-[11px] font-mono uppercase tracking-wider text-ink-secondaryLight dark:text-ink-secondaryDark">
          Ingestion
        </div>

        <div className="grid grid-cols-3 gap-1.5">
          <button
            onClick={handleBrowse}
            className="flex flex-col items-center justify-center p-2 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light dark:bg-inset-dark hover:border-brand-light dark:hover:border-brand-dark transition-colors"
          >
            <svg className="w-4 h-4 text-brand-light dark:text-brand-dark mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
            </svg>
            <span className="text-[11px] font-medium text-ink-primaryLight dark:text-ink-primaryDark">Browse</span>
          </button>

          <button
            onClick={handleScan}
            className="flex flex-col items-center justify-center p-2 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light dark:bg-inset-dark hover:border-brand-light dark:hover:border-brand-dark transition-colors"
          >
            <svg className="w-4 h-4 text-brand-light dark:text-brand-dark mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
            </svg>
            <span className="text-[11px] font-medium text-ink-primaryLight dark:text-ink-primaryDark">Scan</span>
          </button>

          <button
            onClick={handlePaste}
            className="flex flex-col items-center justify-center p-2 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light dark:bg-inset-dark hover:border-brand-light dark:hover:border-brand-dark transition-colors"
          >
            <svg className="w-4 h-4 text-brand-light dark:text-brand-dark mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <span className="text-[11px] font-medium text-ink-primaryLight dark:text-ink-primaryDark">Paste</span>
          </button>
        </div>

        {queue.some((i) => i.status === 'Ready' || i.status === 'Failed') && (
          <button
            onClick={runAllPending}
            disabled={isProcessingAll}
            className="w-full flex items-center justify-center space-x-2 py-1.5 px-3 rounded-panel bg-brand-light dark:bg-brand-dark text-white font-medium text-xs shadow-sm hover:opacity-95 transition-opacity disabled:opacity-50"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{isProcessingAll ? 'Running Queue...' : 'Extract All Pending'}</span>
          </button>
        )}
      </div>

      {/* Queue List */}
      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {queue.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center p-6 text-center text-ink-secondaryLight dark:text-ink-secondaryDark">
            <svg className="w-8 h-8 stroke-1 mb-2 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <p className="text-xs font-medium">Queue is empty</p>
            <p className="text-[11px] opacity-70 mt-0.5">Drop files or paste from clipboard</p>
          </div>
        ) : (
          queue.map((item) => {
            const isSelected = item.id === selectedItemId;
            return (
              <div
                key={item.id}
                onClick={() => selectItem(item.id)}
                className={`group relative p-2.5 rounded-panel border transition-all cursor-pointer ${
                  isSelected
                    ? 'border-brand-light dark:border-brand-dark bg-inset-light dark:bg-inset-dark ring-1 ring-brand-light/30 dark:ring-brand-dark/30'
                    : 'border-hairline-light/60 dark:border-hairline-dark/60 hover:border-hairline-light dark:hover:border-hairline-dark bg-surface-light dark:bg-surface-dark'
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0 pr-2">
                    <p className="text-xs font-medium truncate text-ink-primaryLight dark:text-ink-primaryDark">
                      {item.file_name}
                    </p>
                    <div className="flex items-center space-x-2 mt-1 font-mono text-[10px] text-ink-secondaryLight dark:text-ink-secondaryDark">
                      <span>{item.file_size_str}</span>
                      <span>·</span>
                      <span className="uppercase">{item.source}</span>
                    </div>
                  </div>

                  {/* Status indicator */}
                  <div className="flex items-center space-x-1.5">
                    {item.status === 'Processing' && (
                      <span className="w-2 h-2 rounded-full bg-amber-500 animate-ping" />
                    )}
                    {item.status === 'Done' && (
                      <span className="text-[10px] font-mono px-1 py-0.5 rounded bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                        DONE
                      </span>
                    )}
                    {item.status === 'Failed' && (
                      <span className="text-[10px] font-mono px-1 py-0.5 rounded bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20">
                        FAIL
                      </span>
                    )}
                    {item.status === 'Ready' && (
                      <span className="text-[10px] font-mono px-1 py-0.5 rounded bg-zinc-500/10 text-zinc-500 dark:text-zinc-400">
                        READY
                      </span>
                    )}

                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        removeItem(item.id);
                      }}
                      className="opacity-0 group-hover:opacity-100 p-1 text-ink-secondaryLight hover:text-rose-500 dark:text-ink-secondaryDark dark:hover:text-rose-400 transition-opacity"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Footer metadata */}
      <div className="p-3 border-t border-hairline-light dark:border-hairline-dark flex items-center justify-between text-xs text-ink-secondaryLight dark:text-ink-secondaryDark">
        <span className="font-mono text-[11px]">{queue.length} item{queue.length === 1 ? '' : 's'}</span>
        {queue.some((i) => i.status === 'Done') && (
          <button
            onClick={clearCompleted}
            className="text-[11px] font-medium hover:text-brand-light dark:hover:text-brand-dark transition-colors"
          >
            Clear completed
          </button>
        )}
      </div>
    </aside>
  );
};
