import React, { useState } from 'react';
import { useAppStore, setModal, setState } from '../store/useAppStore';
import * as api from '../../wailsjs/go/main/App';

export const SettingsModal: React.FC = () => {
  const { activeModal, config } = useAppStore();
  const [formData, setFormData] = useState({ ...config });
  const [showKey, setShowKey] = useState(false);
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
        <div className="relative p-4 border-b border-hairline-light dark:border-hairline-dark flex items-center justify-center">
          <h2 className="text-sm font-semibold text-ink-primaryLight dark:text-ink-primaryDark text-center">
            Settings
          </h2>
          <button
            onClick={() => setModal('none')}
            className="absolute right-4 top-1/2 -translate-y-1/2 text-ink-secondaryLight dark:text-ink-secondaryDark hover:opacity-75"
          >
            ✕
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 text-xs select-text">
          {/* API Key */}
          <div>
            <label className="block font-medium text-ink-primaryLight dark:text-ink-primaryDark mb-1">
              Application Key
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
                Default Extraction Mode
              </label>
              <select
                value={formData.quality}
                onChange={(e) => setFormData({ ...formData, quality: e.target.value as any })}
                className="w-full text-xs bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark rounded px-2.5 py-1.5 text-ink-primaryLight dark:text-ink-primaryDark outline-none"
              >
                <option value="standard">Standard</option>
                <option value="high">High Precision</option>
                <option value="document">Document Mode</option>
              </select>
            </div>
          </div>

          {/* Document OCR Model Identifier */}
          <div className="pt-2 border-t border-hairline-light dark:border-hairline-dark">
            <label className="block font-medium text-ink-primaryLight dark:text-ink-primaryDark mb-1">
              Document OCR Model Identifier
            </label>
            <input
              type="text"
              value={formData.document_model_name || ''}
              onChange={(e) => setFormData({ ...formData, document_model_name: e.target.value })}
              placeholder="mistral-ocr-latest"
              className="w-full font-mono text-xs px-3 py-1.5 rounded bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-ink-primaryLight dark:text-ink-primaryDark outline-none"
            />
            <p className="text-[11px] text-ink-secondaryLight/60 dark:text-ink-secondaryDark/60 mt-1">
              Model name sent when Document mode is selected. Defaults to <span className="font-mono text-ink-primaryLight dark:text-ink-primaryDark">mistral-ocr-latest</span>.
            </p>
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
