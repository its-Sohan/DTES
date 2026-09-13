import React, { useEffect, useState } from 'react';
import { useAppStore, setModal } from '../store/useAppStore';
import * as api from '../../wailsjs/go/main/App';
import { UsageStats } from '../types';

export const DashboardModal: React.FC = () => {
  const { activeModal } = useAppStore();
  const [stats, setStats] = useState<UsageStats | null>(null);

  useEffect(() => {
    if (activeModal === 'dashboard') {
      api.GetUsageStats().then(setStats);
    }
  }, [activeModal]);

  if (activeModal !== 'dashboard') return null;

  const total = stats?.total_processed || 0;
  const successful = stats?.successful_runs || 0;
  const successRate = total > 0 ? Math.round((successful / total) * 100) : 100;

  return (
    <div
      className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4 select-none"
      onClick={() => setModal('none')}
    >
      <div
        className="w-full max-w-lg rounded-panel border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark shadow-2xl flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-100"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-hairline-light dark:border-hairline-dark flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <svg className="w-5 h-5 text-brand-light dark:text-brand-dark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            <h2 className="text-sm font-semibold text-ink-primaryLight dark:text-ink-primaryDark">Usage & Metrics</h2>
          </div>
          <button onClick={() => setModal('none')} className="text-ink-secondaryLight dark:text-ink-secondaryDark hover:opacity-75">
            ✕
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div className="p-3.5 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light dark:bg-inset-dark">
              <div className="text-[11px] uppercase font-mono text-ink-secondaryLight dark:text-ink-secondaryDark">
                Documents Processed
              </div>
              <div className="text-2xl font-mono font-bold text-ink-primaryLight dark:text-ink-primaryDark mt-1">
                {total}
              </div>
            </div>

            <div className="p-3.5 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light dark:bg-inset-dark">
              <div className="text-[11px] uppercase font-mono text-ink-secondaryLight dark:text-ink-secondaryDark">
                Success Rate
              </div>
              <div className="text-2xl font-mono font-bold text-emerald-600 dark:text-emerald-400 mt-1">
                {successRate}%
              </div>
            </div>

            <div className="p-3.5 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light dark:bg-inset-dark">
              <div className="text-[11px] uppercase font-mono text-ink-secondaryLight dark:text-ink-secondaryDark">
                Characters Extracted
              </div>
              <div className="text-2xl font-mono font-bold text-brand-light dark:text-brand-dark mt-1">
                {stats?.total_characters_extracted.toLocaleString() || 0}
              </div>
            </div>

            <div className="p-3.5 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light dark:bg-inset-dark">
              <div className="text-[11px] uppercase font-mono text-ink-secondaryLight dark:text-ink-secondaryDark">
                Files Ingested
              </div>
              <div className="text-2xl font-mono font-bold text-ink-primaryLight dark:text-ink-primaryDark mt-1">
                {stats?.total_scanned_or_uploaded || 0}
              </div>
            </div>
          </div>

          <div className="p-3 rounded-panel border border-hairline-light dark:border-hairline-dark bg-inset-light/50 dark:bg-inset-dark/50 flex items-center justify-between text-xs">
            <span className="text-ink-secondaryLight dark:text-ink-secondaryDark">Completed / Failed Runs:</span>
            <span className="font-mono">
              <span className="text-emerald-600 dark:text-emerald-400 font-semibold">{successful} OK</span>
              {' / '}
              <span className="text-rose-600 dark:text-rose-400 font-semibold">{stats?.failed_runs || 0} Failed</span>
            </span>
          </div>
        </div>

        <div className="p-3 border-t border-hairline-light dark:border-hairline-dark flex justify-end bg-inset-light/30 dark:bg-inset-dark/30">
          <button
            onClick={() => setModal('none')}
            className="px-4 py-1.5 rounded-panel bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-xs font-medium text-ink-primaryLight dark:text-ink-primaryDark hover:border-brand-light"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
