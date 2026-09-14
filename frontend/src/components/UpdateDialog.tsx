import React, { useEffect, useState } from 'react';
import { useAppStore, setModal } from '../store/useAppStore';
import * as api from '../../wailsjs/go/main/App';
import { EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { UpdateCheckResult } from '../types';

export const UpdateDialog: React.FC = () => {
  const { activeModal } = useAppStore();
  const [updateInfo, setUpdateInfo] = useState<UpdateCheckResult | null>(null);
  const [isChecking, setIsChecking] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);
  const [downloadProgress, setDownloadProgress] = useState(0);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [statusText, setStatusText] = useState<string>('');

  useEffect(() => {
    if (activeModal === 'update') {
      setIsChecking(true);
      setErrorMessage(null);
      setIsDownloading(false);
      setDownloadProgress(0);
      setStatusText('');

      api.CheckForUpdates()
        .then((res) => {
          setUpdateInfo(res);
          if (res.error) {
            setErrorMessage(res.error);
          }
        })
        .catch((err: any) => {
          setErrorMessage(err?.message || 'Failed to check for updates');
        })
        .finally(() => setIsChecking(false));
    }
  }, [activeModal]);

  useEffect(() => {
    const unsub = EventsOn('update_progress', (percent: number) => {
      setDownloadProgress(percent);
      if (percent >= 100) {
        setStatusText('Download complete. Launching installer...');
      } else {
        setStatusText(`Downloading update... ${percent}%`);
      }
    });

    return () => {
      unsub();
    };
  }, []);

  const handleStartUpdate = async () => {
    if (!updateInfo?.download_url) return;
    setIsDownloading(true);
    setErrorMessage(null);
    setDownloadProgress(0);
    setStatusText('Connecting to update server...');

    try {
      await api.InstallUpdate(updateInfo.download_url);
      setStatusText('Installer launched! Closing app to apply update...');
      setTimeout(async () => {
        try {
          await api.QuitApplication();
        } catch {
          // ignore
        }
      }, 1500);
    } catch (err: any) {
      setIsDownloading(false);
      setErrorMessage(err?.message || 'Failed to download and start installer.');
    }
  };

  const handleOpenBrowser = () => {
    const target = updateInfo?.release_url || updateInfo?.download_url || 'https://github.com/its-Sohan/DTES/releases';
    BrowserOpenURL(target);
  };

  if (activeModal !== 'update') return null;

  return (
    <div
      className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4 select-none"
      onClick={() => {
        if (!isDownloading) setModal('none');
      }}
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
          {!isDownloading && (
            <button onClick={() => setModal('none')} className="text-ink-secondaryLight dark:text-ink-secondaryDark hover:opacity-75">
              ✕
            </button>
          )}
        </div>

        <div className="p-5 text-xs select-text space-y-3">
          {isChecking ? (
            <div className="py-8 flex flex-col items-center justify-center space-y-2 text-ink-secondaryLight dark:text-ink-secondaryDark">
              <svg className="animate-spin w-5 h-5" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
              <span>Checking for latest release from GitHub...</span>
            </div>
          ) : updateInfo?.has_update ? (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-emerald-600 dark:text-emerald-400 text-sm">
                  New version available: v{updateInfo.latest_version}
                </span>
                <span className="font-mono text-[11px] px-2 py-0.5 rounded bg-inset-light dark:bg-inset-dark text-ink-secondaryLight dark:text-ink-secondaryDark">
                  Current: v{updateInfo.current_version}
                </span>
              </div>

              {updateInfo.release_name && (
                <div className="font-medium text-ink-primaryLight dark:text-ink-primaryDark">
                  {updateInfo.release_name}
                </div>
              )}

              <div className="p-3 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark font-sans text-xs leading-relaxed max-h-40 overflow-y-auto whitespace-pre-wrap">
                {updateInfo.release_notes || 'Bug fixes, performance improvements, and new updates.'}
              </div>

              {isDownloading ? (
                <div className="space-y-2 pt-2">
                  <div className="flex justify-between text-xs text-ink-secondaryLight dark:text-ink-secondaryDark font-medium">
                    <span>{statusText || 'Downloading update...'}</span>
                    <span>{downloadProgress}%</span>
                  </div>
                  <div className="w-full bg-inset-light dark:bg-inset-dark rounded-full h-2 overflow-hidden border border-hairline-light dark:border-hairline-dark">
                    <div
                      className="bg-brand-light dark:bg-brand-dark h-2 rounded-full transition-all duration-150"
                      style={{ width: `${downloadProgress}%` }}
                    />
                  </div>
                </div>
              ) : null}

              {errorMessage && (
                <div className="p-2.5 rounded bg-rose-500/10 border border-rose-500/20 text-rose-500 text-xs">
                  {errorMessage}
                </div>
              )}
            </div>
          ) : (
            <div className="py-6 text-center space-y-2">
              <div className="text-emerald-600 dark:text-emerald-400 font-semibold text-sm">
                You are on the latest version!
              </div>
              <div className="font-mono text-xs text-ink-secondaryLight dark:text-ink-secondaryDark">
                Installed Version: v{updateInfo?.current_version || '0.1.3'}
              </div>
              {errorMessage && (
                <div className="mt-2 text-rose-400 text-[11px]">
                  {errorMessage}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="p-3 border-t border-hairline-light dark:border-hairline-dark flex items-center justify-between bg-inset-light/30 dark:bg-inset-dark/30">
          <div>
            {updateInfo?.has_update && (
              <button
                type="button"
                onClick={handleOpenBrowser}
                className="text-xs text-brand-light dark:text-brand-dark hover:underline flex items-center space-x-1"
                title="Download directly from GitHub release page in your browser"
              >
                <span>Manual Download</span>
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
              </button>
            )}
          </div>

          <div className="flex items-center space-x-2">
            {!isDownloading && (
              <button
                onClick={() => setModal('none')}
                className="px-3 py-1.5 rounded-panel text-xs text-ink-secondaryLight dark:text-ink-secondaryDark hover:bg-inset-light dark:hover:bg-inset-dark"
              >
                Close
              </button>
            )}

            {updateInfo?.has_update && (
              <button
                disabled={isDownloading}
                onClick={handleStartUpdate}
                className="px-4 py-1.5 rounded-panel bg-brand-light dark:bg-brand-dark text-white text-xs font-semibold shadow hover:opacity-95 disabled:opacity-50 flex items-center space-x-1.5"
              >
                {isDownloading ? (
                  <>
                    <svg className="animate-spin w-3.5 h-3.5" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                    </svg>
                    <span>Updating...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                    <span>Update & Restart</span>
                  </>
                )}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
