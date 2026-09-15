import React, { useState, useEffect, useRef } from 'react';
import {
  useAppStore,
  setModal,
  addItems,
  runAllPending,
  extractItem,
  toggleTheme,
  setQuality,
} from '../store/useAppStore';
import * as api from '../../wailsjs/go/main/App';

export const CommandPaletteModal: React.FC = () => {
  const { activeModal, queue, selectedItemId } = useAppStore();
  const [query, setQuery] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (activeModal === 'command_palette') {
      setTimeout(() => inputRef.current?.focus(), 50);
    } else {
      setQuery('');
    }
  }, [activeModal]);

  if (activeModal !== 'command_palette') return null;

  const selectedItem = queue.find((i) => i.id === selectedItemId);

  const actions = [
    {
      id: 'browse',
      title: 'Browse local files',
      category: 'Ingestion',
      shortcut: 'Ctrl+O',
      perform: async () => {
        setModal('none');
        const items = await api.PickFiles();
        if (items) addItems(items);
      },
    },
    {
      id: 'scan',
      title: 'Scan document hardware',
      category: 'Ingestion',
      shortcut: 'Ctrl+Shift+S',
      perform: async () => {
        setModal('none');
        try {
          const item = await api.ScanDocument();
          if (item) addItems([item]);
        } catch (e: any) {
          alert(e.message || 'Scanning failed');
        }
      },
    },
    {
      id: 'paste',
      title: 'Paste image from clipboard',
      category: 'Ingestion',
      shortcut: 'Ctrl+V',
      perform: async () => {
        setModal('none');
        try {
          const item = await api.GetClipboardImage();
          if (item) addItems([item]);
        } catch (e: any) {
          alert(e.message || 'No image found on clipboard');
        }
      },
    },
    {
      id: 'extract',
      title: 'Extract current document',
      category: 'OCR',
      shortcut: 'Ctrl+Enter',
      perform: () => {
        setModal('none');
        if (selectedItem) extractItem(selectedItem);
      },
    },
    {
      id: 'run_all',
      title: 'Run all pending items',
      category: 'Queue',
      shortcut: 'Ctrl+Shift+Enter',
      perform: () => {
        setModal('none');
        runAllPending();
      },
    },
    {
      id: 'theme',
      title: 'Toggle light / dark theme',
      category: 'View',
      shortcut: 'Ctrl+T',
      perform: () => {
        toggleTheme();
        setModal('none');
      },
    },
    {
      id: 'mode_standard',
      title: 'Mode: Standard (Vision LLM)',
      category: 'Mode',
      perform: () => {
        setQuality('standard');
        setModal('none');
      },
    },
    {
      id: 'mode_high',
      title: 'Mode: High Precision (Advanced Vision LLM)',
      category: 'Mode',
      perform: () => {
        setQuality('high');
        setModal('none');
      },
    },
    {
      id: 'mode_document',
      title: 'Mode: Document (Dedicated OCR Model)',
      category: 'Mode',
      perform: () => {
        setQuality('document');
        setModal('none');
      },
    },
    {
      id: 'settings',
      title: 'Open Settings & API credentials',
      category: 'App',
      shortcut: 'Ctrl+,',
      perform: () => setModal('settings'),
    },
    {
      id: 'dashboard',
      title: 'Open Usage & Analytics Dashboard',
      category: 'App',
      shortcut: 'Ctrl+D',
      perform: () => setModal('dashboard'),
    },
    {
      id: 'update',
      title: 'Check for software updates',
      category: 'App',
      shortcut: 'Ctrl+U',
      perform: () => setModal('update'),
    },
    {
      id: 'help',
      title: 'Help Center & OCR guide',
      category: 'Help',
      shortcut: 'F1',
      perform: () => setModal('help'),
    },
  ];

  const filtered = actions.filter((a) =>
    a.title.toLowerCase().includes(query.toLowerCase()) ||
    a.category.toLowerCase().includes(query.toLowerCase())
  );

  return (
    <div
      className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-start justify-center pt-24 select-none"
      onClick={() => setModal('none')}
    >
      <div
        className="w-full max-w-lg rounded-panel border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-100"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-3 border-b border-hairline-light dark:border-hairline-dark flex items-center space-x-2">
          <svg className="w-4 h-4 text-ink-secondaryLight dark:text-ink-secondaryDark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Type a command or action..."
            className="flex-1 bg-transparent text-sm text-ink-primaryLight dark:text-ink-primaryDark outline-none"
          />
          <kbd className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-ink-secondaryLight dark:text-ink-secondaryDark">
            ESC
          </kbd>
        </div>

        <div className="max-h-80 overflow-y-auto p-1.5 space-y-0.5">
          {filtered.length === 0 ? (
            <div className="p-6 text-center text-xs text-ink-secondaryLight dark:text-ink-secondaryDark">
              No matching commands found.
            </div>
          ) : (
            filtered.map((action) => (
              <button
                key={action.id}
                onClick={action.perform}
                className="w-full flex items-center justify-between px-3 py-2 rounded text-left hover:bg-inset-light dark:hover:bg-inset-dark text-ink-primaryLight dark:text-ink-primaryDark transition-colors group"
              >
                <div className="flex items-center space-x-2.5">
                  <span className="font-mono text-[9px] uppercase px-1.5 py-0.5 rounded bg-inset-light dark:bg-inset-dark text-ink-secondaryLight dark:text-ink-secondaryDark border border-hairline-light/60 dark:border-hairline-dark/60">
                    {action.category}
                  </span>
                  <span className="text-xs font-medium">{action.title}</span>
                </div>
                {action.shortcut && (
                  <span className="font-mono text-[10px] text-ink-secondaryLight dark:text-ink-secondaryDark opacity-75">
                    {action.shortcut}
                  </span>
                )}
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
};
