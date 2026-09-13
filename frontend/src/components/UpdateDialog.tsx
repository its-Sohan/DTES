import React, { useEffect, useState } from 'react';
import { useAppStore, setModal } from '../store/useAppStore';
import * as api from '../../wailsjs/go/main/App';
import { UpdateCheckResult } from '../types';

export const UpdateDialog: React.FC = () => {
  const { activeModal } = useAppStore();
  const [updateInfo, setUpdateInfo] = useState<UpdateCheckResult | null>(null);
  const [isChecking, setIsChecking] = useState(false);

  useEffect(() => {
    if (activeModal === 'update') {
      setIsChecking(true);
      api.CheckForUpdates()
        .then(setUpdateInfo)
        .finally(() => setIsChecking(false));
    }
  }, [activeModal]);

  if (activeModal !== 'update') return null;

  return (
    <div
      className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4 select-none"
      onClick={() => setModal('none')}
    >
      <div
        className="w-full max-w-md rounded-panel border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark shadow-2xl flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-100"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-hairline-light dark:border-hairline-dark flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <svg className="w-5 h-5 text-brand-light dark:text-brand-dark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <h2 className="text-sm font-semibold text-ink-primaryLight dark:text-ink-primaryDark">Software Updates</h2>
          </div>
          <button onClick={() => setModal('none')} className="text-ink-secondaryLight dark:text-ink-secondaryDark hover:opacity-75">
            ✕
          </button>
        </div>

        <div className="p-5 text-xs select-text space-y-3">
          {isChecking ? (
            <div className="py-8 flex flex-col items-center justify-center space-y-2 text-ink-secondaryLight dark:text-ink-secondaryDark">
              <svg className="animate-spin w-5 h-5" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
              <span>Checking for latest release...</span>
            </div>
          ) : updateInfo?.has_update ? (
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="font-semibold text-emerald-600 dark:text-emerald-400">
                  New version available: v{updateInfo.latest_version}
                </span>
                <span className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-inset-light dark:bg-inset-dark">
                  Current: v{updateInfo.current_version}
                </span>
              </div>
              <div className="p-3 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark font-sans text-xs leading-relaxed max-h-48 overflow-y-auto whitespace-pre-wrap">
                {updateInfo.release_notes || 'Bug fixes and performance improvements.'}
              </div>
            </div>
          ) : (
            <div className="py-6 text-center space-y-1">
              <div className="text-emerald-600 dark:text-emerald-400 font-medium">You are on the latest version!</div>
              <div className="font-mono text-[11px] text-ink-secondaryLight dark:text-ink-secondaryDark">
                Version {updateInfo?.current_version || '1.0.0'}
              </div>
            </div>
          )}
        </div>

        <div className="p-3 border-t border-hairline-light dark:border-hairline-dark flex items-center justify-end space-x-2 bg-inset-light/30 dark:bg-inset-dark/30">
          <button
            onClick={() => setModal('none')}
            className="px-3 py-1.5 rounded-panel text-xs text-ink-secondaryLight dark:text-ink-secondaryDark hover:bg-inset-light"
          >
            Close
          </button>
          {updateInfo?.has_update && (
            <a
              href={updateInfo.download_url || updateInfo.release_url}
              target="_blank"
              rel="noreferrer"
              className="px-4 py-1.5 rounded-panel bg-brand-light dark:bg-brand-dark text-white text-xs font-semibold shadow hover:opacity-95 text-center"
            >
              Download Update
            </a>
          )}
        </div>
      </div>
    </div>
  );
};
