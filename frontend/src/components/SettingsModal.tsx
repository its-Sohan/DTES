import React, { useState } from 'react';
import { useAppStore, setModal, setState } from '../store/useAppStore';
import * as api from '../../wailsjs/go/main/App';

export const SettingsModal: React.FC = () => {
  const { activeModal, config } = useAppStore();
  const [formData, setFormData] = useState({ ...config });
  const [showKey, setShowKey] = useState(false);
  const [bugDesc, setBugDesc] = useState('');
  const [bugSteps, setBugSteps] = useState('');
  const [bugReportPath, setBugReportPath] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  if (activeModal !== 'settings') return null;

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await api.SaveConfig(formData as any);
      setState({ config: formData });
      setModal('none');
    } catch (err: any) {
      alert(err?.message || 'Failed to save config');
    } finally {
      setIsSaving(false);
    }
  };

  const handleGenerateBugReport = async () => {
    if (!bugDesc.trim()) {
      alert('Please enter a brief description of the issue');
      return;
    }
    try {
      const path = await api.GenerateBugReport(bugDesc, bugSteps);
      setBugReportPath(path);
    } catch (err: any) {
      alert(err?.message || 'Failed to generate diagnostic report');
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4 select-none"
      onClick={() => setModal('none')}
    >
      <div
        className="w-full max-w-xl max-h-[85vh] rounded-panel border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark shadow-2xl flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-100"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="p-4 border-b border-hairline-light dark:border-hairline-dark flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <svg className="w-5 h-5 text-brand-light dark:text-brand-dark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <h2 className="text-sm font-semibold text-ink-primaryLight dark:text-ink-primaryDark">Settings & Credentials</h2>
          </div>
          <button onClick={() => setModal('none')} className="text-ink-secondaryLight dark:text-ink-secondaryDark hover:opacity-75">
            ✕
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 text-xs select-text">
          {/* API Key */}
          <div>
            <label className="block font-medium text-ink-primaryLight dark:text-ink-primaryDark mb-1">
              API Key (OpenAI / Gemini / OpenRouter)
            </label>
            <div className="relative">
              <input
                type={showKey ? 'text' : 'password'}
                value={formData.api_key}
                onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
                placeholder="sk-..."
                className="w-full font-mono text-xs px-3 py-1.5 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-ink-primaryLight dark:text-ink-primaryDark outline-none"
              />
              <button
                type="button"
                onClick={() => setShowKey(!showKey)}
                className="absolute right-2 top-1.5 text-ink-secondaryLight dark:text-ink-secondaryDark hover:opacity-80"
              >
                {showKey ? 'Hide' : 'Show'}
              </button>
            </div>
            <p className="text-[10px] text-ink-secondaryLight dark:text-ink-secondaryDark mt-0.5">
              Leave blank only if using a local vision server (Ollama, LM Studio).
            </p>
          </div>

          {/* Options Grid */}
          <div className="grid grid-cols-2 gap-3 pt-2 border-t border-hairline-light dark:border-hairline-dark">
            <div>
              <label className="block font-medium text-ink-primaryLight dark:text-ink-primaryDark mb-1">
                Default Output Mode
              </label>
              <select
                value={formData.default_output_mode}
                onChange={(e) => setFormData({ ...formData, default_output_mode: e.target.value as any })}
                className="w-full text-xs bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark rounded px-2.5 py-1.5 text-ink-primaryLight dark:text-ink-primaryDark outline-none"
              >
                <option value="document">Document (Prose)</option>
                <option value="spreadsheet">Spreadsheet (Tables)</option>
                <option value="key_value">Key-Value Pairs</option>
                <option value="raw_text">Raw Text</option>
              </select>
            </div>

            <div>
              <label className="block font-medium text-ink-primaryLight dark:text-ink-primaryDark mb-1">
                Extraction Quality
              </label>
              <select
                value={formData.quality}
                onChange={(e) => setFormData({ ...formData, quality: e.target.value as any })}
                className="w-full text-xs bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark rounded px-2.5 py-1.5 text-ink-primaryLight dark:text-ink-primaryDark outline-none"
              >
                <option value="standard">Standard</option>
                <option value="high">High Precision</option>
              </select>
            </div>
          </div>

          {/* Toggles */}
          <div className="space-y-2 pt-2 border-t border-hairline-light dark:border-hairline-dark">
            <label className="flex items-center space-x-2 cursor-pointer">
              <input
                type="checkbox"
                checked={formData.auto_extract}
                onChange={(e) => setFormData({ ...formData, auto_extract: e.target.checked })}
                className="rounded border-hairline-light dark:border-hairline-dark text-brand-light"
              />
              <span className="text-xs text-ink-primaryLight dark:text-ink-primaryDark">
                Auto-extract immediately when files are added
              </span>
            </label>

            <label className="flex items-center space-x-2 cursor-pointer">
              <input
                type="checkbox"
                checked={formData.check_updates_on_startup}
                onChange={(e) => setFormData({ ...formData, check_updates_on_startup: e.target.checked })}
                className="rounded border-hairline-light dark:border-hairline-dark text-brand-light"
              />
              <span className="text-xs text-ink-primaryLight dark:text-ink-primaryDark">
                Check for software updates on startup
              </span>
            </label>
          </div>

          {/* Diagnostic Bug Report Section */}
          <div className="pt-3 border-t border-hairline-light dark:border-hairline-dark space-y-2">
            <h3 className="font-semibold text-xs text-ink-primaryLight dark:text-ink-primaryDark">
              Diagnostic & Bug Reporting
            </h3>
            <p className="text-[10px] text-ink-secondaryLight dark:text-ink-secondaryDark">
              Generates a redacted, offline diagnostic report on your disk. API keys and document texts are completely excluded.
            </p>
            <input
              type="text"
              value={bugDesc}
              onChange={(e) => setBugDesc(e.target.value)}
              placeholder="Brief issue description..."
              className="w-full text-xs px-2.5 py-1 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-ink-primaryLight dark:text-ink-primaryDark outline-none"
            />
            <button
              type="button"
              onClick={handleGenerateBugReport}
              className="text-xs px-2.5 py-1 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark hover:border-brand-light text-ink-primaryLight dark:text-ink-primaryDark"
            >
              Generate Diagnostic Bundle
            </button>
            {bugReportPath && (
              <p className="text-[10px] font-mono text-emerald-600 dark:text-emerald-400 truncate">
                Saved: {bugReportPath}
              </p>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="p-3 border-t border-hairline-light dark:border-hairline-dark flex items-center justify-end space-x-2 bg-inset-light/30 dark:bg-inset-dark/30">
          <button
            onClick={() => setModal('none')}
            className="px-3 py-1.5 rounded-panel text-xs text-ink-secondaryLight dark:text-ink-secondaryDark hover:bg-inset-light dark:hover:bg-inset-dark"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="px-4 py-1.5 rounded-panel bg-brand-light dark:bg-brand-dark text-white text-xs font-semibold shadow hover:opacity-95"
          >
            {isSaving ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </div>
    </div>
  );
};
