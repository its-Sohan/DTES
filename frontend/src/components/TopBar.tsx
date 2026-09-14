import React from 'react';
import { useAppStore, toggleTheme, toggleSidebar, setModal } from '../store/useAppStore';
import appIconLight from '../assets/app_icon.svg';
import appIconDark from '../assets/app_icon_dark.svg';

export const TopBar: React.FC = () => {
  const { themeMode, statusMessage, isExtracting, queue, appVersion } = useAppStore();

  const isDark = themeMode === 'dark';
  const appIcon = isDark ? appIconDark : appIconLight;

  return (
    <header className="h-12 border-b border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark px-4 flex items-center justify-between z-10 select-none">
      {/* Brand & History Toggle */}
      <div className="flex items-center space-x-3">
        <button
          onClick={toggleSidebar}
          title="Toggle history rail"
          className="p-1.5 rounded-panel text-ink-secondaryLight dark:text-ink-secondaryDark hover:bg-inset-light dark:hover:bg-inset-dark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h7" />
          </svg>
        </button>

        <div className="flex items-baseline space-x-1.5">
          <img src={appIcon} alt="ITT OCR" className="w-4 h-4 object-contain self-center" />
          <span className="font-semibold text-sm tracking-tight text-ink-primaryLight dark:text-ink-primaryDark">
            ITT OCR
          </span>
          <button
            onClick={() => setModal('update')}
            title="Check for software updates"
            className="text-[10px] font-mono px-1 py-0.5 rounded bg-inset-light dark:bg-inset-dark text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-brand-light dark:hover:text-brand-dark border border-hairline-light/50 dark:border-hairline-dark/50 transition-colors"
          >
            v{appVersion || '0.1.3'}
          </button>
        </div>
      </div>

      {/* Center: Command Palette Pill */}
      <div className="flex-1 max-w-md mx-6">
        <button
          onClick={() => setModal('command_palette')}
          className="w-full flex items-center justify-between px-3 py-1.5 text-xs rounded-panel bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-ink-secondaryLight dark:text-ink-secondaryDark hover:border-brand-light dark:hover:border-brand-dark transition-colors"
        >
          <div className="flex items-center space-x-2">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <span>Search actions & commands...</span>
          </div>
        </button>
      </div>

      {/* Right Controls */}
      <div className="flex items-center space-x-2">

        {/* Dashboard Metrics */}
        <button
          onClick={() => setModal('dashboard')}
          title="Usage metrics"
          className="p-1.5 rounded-panel text-ink-secondaryLight dark:text-ink-secondaryDark hover:bg-inset-light dark:hover:bg-inset-dark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        </button>

        {/* Help */}
        <button
          onClick={() => setModal('help')}
          title="Help & Shortcuts"
          className="p-1.5 rounded-panel text-ink-secondaryLight dark:text-ink-secondaryDark hover:bg-inset-light dark:hover:bg-inset-dark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </button>

        {/* Theme Toggle */}
        <button
          onClick={toggleTheme}
          title={`Switch to ${isDark ? 'Light' : 'Dark'} mode`}
          className="p-1.5 rounded-panel text-ink-secondaryLight dark:text-ink-secondaryDark hover:bg-inset-light dark:hover:bg-inset-dark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark transition-colors"
        >
          {isDark ? (
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
          ) : (
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
          )}
        </button>

        {/* Settings */}
        <button
          onClick={() => setModal('settings')}
          title="Settings"
          className="p-1.5 rounded-panel text-ink-secondaryLight dark:text-ink-secondaryDark hover:bg-inset-light dark:hover:bg-inset-dark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </button>
      </div>
    </header>
  );
};
