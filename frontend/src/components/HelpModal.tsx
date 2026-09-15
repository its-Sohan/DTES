import React from 'react';
import { useAppStore, setModal } from '../store/useAppStore';

export const HelpModal: React.FC = () => {
  const { activeModal } = useAppStore();

  if (activeModal !== 'help') return null;

  return (
    <div
      className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4 select-none"
      onClick={() => setModal('none')}
    >
      <div
        className="w-full max-w-xl max-h-[85vh] rounded-panel border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark shadow-2xl flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-100"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-hairline-light dark:border-hairline-dark flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <svg className="w-5 h-5 text-brand-light dark:text-brand-dark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <h2 className="text-sm font-semibold text-ink-primaryLight dark:text-ink-primaryDark">Help & Documentation</h2>
          </div>
          <button onClick={() => setModal('none')} className="text-ink-secondaryLight dark:text-ink-secondaryDark hover:opacity-75">
            ✕
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-5 text-xs select-text space-y-4">
          {/* Keyboard Shortcuts */}
          <div>
            <h3 className="font-semibold text-ink-primaryLight dark:text-ink-primaryDark mb-2">
              Keyboard Shortcuts
            </h3>
            <div className="grid grid-cols-2 gap-2 font-mono text-[11px]">
              <div className="flex items-center justify-between p-2 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark">
                <span>Command Palette</span>
                <kbd className="px-1.5 py-0.5 rounded bg-surface-light dark:bg-surface-dark border">Ctrl + K</kbd>
              </div>
              <div className="flex items-center justify-between p-2 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark">
                <span>Extract Document</span>
                <kbd className="px-1.5 py-0.5 rounded bg-surface-light dark:bg-surface-dark border">Ctrl + Enter</kbd>
              </div>
              <div className="flex items-center justify-between p-2 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark">
                <span>Browse Files</span>
                <kbd className="px-1.5 py-0.5 rounded bg-surface-light dark:bg-surface-dark border">Ctrl + O</kbd>
              </div>
              <div className="flex items-center justify-between p-2 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark">
                <span>Paste Image</span>
                <kbd className="px-1.5 py-0.5 rounded bg-surface-light dark:bg-surface-dark border">Ctrl + V</kbd>
              </div>
              <div className="flex items-center justify-between p-2 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark">
                <span>Toggle Theme</span>
                <kbd className="px-1.5 py-0.5 rounded bg-surface-light dark:bg-surface-dark border">Ctrl + T</kbd>
              </div>
              <div className="flex items-center justify-between p-2 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark">
                <span>Toggle Sidebar</span>
                <kbd className="px-1.5 py-0.5 rounded bg-surface-light dark:bg-surface-dark border">Ctrl + B</kbd>
              </div>
            </div>
          </div>

          {/* Extraction Modes */}
          <div>
            <h3 className="font-semibold text-ink-primaryLight dark:text-ink-primaryDark mb-1">
              Extraction Modes
            </h3>
            <ul className="list-disc pl-4 space-y-1 text-ink-secondaryLight dark:text-ink-secondaryDark">
              <li><strong>Standard</strong>: Fast vision LLM extraction using the configured base vision model.</li>
              <li><strong>High Precision</strong>: Upgrades the model to its higher-accuracy reasoning tier for complex or degraded pages.</li>
              <li><strong>Document</strong>: Routes requests directly to a dedicated OCR model (<code className="font-mono text-[10px]">mistral-ocr-latest</code> or custom configured model) rather than a general vision LLM.</li>
            </ul>
          </div>

          {/* Bengali Typography & OCR */}
          <div>
            <h3 className="font-semibold text-ink-primaryLight dark:text-ink-primaryDark mb-1">
              Bengali & Multilingual Precision
            </h3>
            <p className="text-ink-secondaryLight dark:text-ink-secondaryDark leading-relaxed">
              ITT OCR includes built-in Unicode NFC normalization to preserve complex Bengali conjuncts (যুক্তাক্ষর) and vowel signs (কার). The Kalpurush and Noto Sans Bengali typefaces are embedded directly into the native bundle.
            </p>
          </div>

          {/* Post-Processing Tools */}
          <div>
            <h3 className="font-semibold text-ink-primaryLight dark:text-ink-primaryDark mb-1">
              Deterministic Post-Processing
            </h3>
            <ul className="list-disc pl-4 space-y-1 text-ink-secondaryLight dark:text-ink-secondaryDark">
              <li><strong>০-৯ ↔ 0-9</strong>: Instant 100% deterministic numeral translation.</li>
              <li><strong>Unwrap Lines</strong>: Heuristically joins newspaper and scanner margin wraps while strictly preserving bullet points, tables, and markdown headers.</li>
              <li><strong>Align Tables</strong>: Uniformly formats markdown ASCII table columns.</li>
            </ul>
          </div>
        </div>

        <div className="p-3 border-t border-hairline-light dark:border-hairline-dark flex justify-end bg-inset-light/30 dark:bg-inset-dark/30">
          <button
            onClick={() => setModal('none')}
            className="px-4 py-1.5 rounded-panel bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-xs font-medium text-ink-primaryLight dark:text-ink-primaryDark hover:border-brand-light"
          >
            Got it
          </button>
        </div>
      </div>
    </div>
  );
};
