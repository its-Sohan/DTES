import React, { useState } from 'react';
import {
  useAppStore,
  setOutputMode,
  setQuality,
  updateItem,
  extractItem,
  setAuditMode,
  setActiveBlockIndex,
} from '../store/useAppStore';
import * as api from '../../wailsjs/go/main/App';
import { InvoiceValidationResult } from '../types';

export const TextPanel: React.FC = () => {
  const {
    queue,
    selectedItemId,
    activeOutputMode,
    activeQuality,
    isExtracting,
    auditMode,
    activeBlockIndex,
  } = useAppStore();

  const [copied, setCopied] = useState<boolean>(false);
  const [exportOpen, setExportOpen] = useState<boolean>(false);
  const [mathResult, setMathResult] = useState<InvoiceValidationResult | null>(null);

  const selectedItem = queue.find((i) => i.id === selectedItemId);
  const extractedText = selectedItem?.extracted_text || '';

  const handleCopy = async () => {
    if (!extractedText) return;
    try {
      await navigator.clipboard.writeText(extractedText);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback
    }
  };

  const handleTransform = async (transformType: string) => {
    if (!extractedText || !selectedItem) return;
    try {
      const transformed = await api.TransformText(extractedText, transformType);
      updateItem(selectedItem.id, { extracted_text: transformed });
    } catch (err) {
      console.error('Transform failed:', err);
    }
  };

  const handleCheckMath = async () => {
    if (!extractedText) return;
    try {
      const res = await api.CheckInvoiceMath(extractedText);
      setMathResult(res);
    } catch (err) {
      console.error('Invoice math check failed:', err);
    }
  };

  const handleExport = async (format: 'txt' | 'md' | 'csv' | 'tsv' | 'json') => {
    if (!extractedText || !selectedItem) return;
    setExportOpen(false);

    const baseName = selectedItem.file_name.replace(/\.[^/.]+$/, '');
    let defaultName = `${baseName}.${format}`;
    let contentToSave = extractedText;

    if (format === 'json') {
      contentToSave = JSON.stringify(
        {
          file_name: selectedItem.file_name,
          mode: selectedItem.output_mode,
          extracted_text: extractedText,
        },
        null,
        2
      );
    } else if (format === 'csv') {
      // Convert markdown table to CSV
      const lines = extractedText.split('\n');
      const csvLines = lines
        .filter((l) => l.trim().startsWith('|') && l.trim().endsWith('|'))
        .map((l) => {
          const cells = l.trim().replace(/^\||\|$/g, '').split('|').map((c) => `"${c.trim().replace(/"/g, '""')}"`);
          return cells.join(',');
        });
      if (csvLines.length > 0) {
        contentToSave = csvLines.join('\n');
      }
    }

    try {
      await api.SaveExportFile(defaultName, contentToSave);
    } catch (err) {
      console.error('Export failed:', err);
    }
  };

  // Split text into paragraph blocks for audit mode
  const paragraphs = extractedText
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter((p) => p.length > 0);

  const wordCount = extractedText.trim() ? extractedText.trim().split(/\s+/).length : 0;
  const charCount = extractedText.length;

  return (
    <div className="w-[520px] flex flex-col h-full bg-surface-light dark:bg-surface-dark border-l border-hairline-light dark:border-hairline-dark select-none">
      {/* Top Controls: Modes & Primary Extract */}
      <div className="p-3 border-b border-hairline-light dark:border-hairline-dark space-y-2.5">
        {/* Mode Selector Tabs */}
        <div className="flex items-center space-x-1 p-0.5 rounded-panel bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-xs">
          {[
            { id: 'document', label: 'Document' },
            { id: 'spreadsheet', label: 'Spreadsheet' },
            { id: 'key_value', label: 'Key-Value' },
            { id: 'raw_text', label: 'Raw' },
          ].map((m) => {
            const isActive = activeOutputMode === m.id;
            return (
              <button
                key={m.id}
                onClick={() => setOutputMode(m.id as any)}
                className={`flex-1 py-1 rounded-[4px] font-medium transition-all ${
                  isActive
                    ? 'bg-surface-light dark:bg-surface-dark text-brand-light dark:text-brand-dark shadow-sm'
                    : 'text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark'
                }`}
              >
                {m.label}
              </button>
            );
          })}
        </div>

        {/* Quality & Primary Action */}
        <div className="flex items-center justify-between space-x-2">
          {/* Quality Select */}
          <div className="flex items-center space-x-1.5 text-xs text-ink-secondaryLight dark:text-ink-secondaryDark">
            <span className="font-mono text-[10px] uppercase">Quality:</span>
            <select
              value={activeQuality}
              onChange={(e) => setQuality(e.target.value as any)}
              className="text-xs bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark rounded px-2 py-1 text-ink-primaryLight dark:text-ink-primaryDark outline-none cursor-pointer"
            >
              <option value="standard">Standard (Fast)</option>
              <option value="high">High Precision (Gemini 3.7)</option>
            </select>
          </div>

          {/* Extract Button */}
          <button
            onClick={() => selectedItem && extractItem(selectedItem)}
            disabled={!selectedItem || isExtracting}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-panel bg-brand-light dark:bg-brand-dark text-white text-xs font-semibold shadow hover:opacity-95 transition-opacity disabled:opacity-40"
          >
            {isExtracting ? (
              <>
                <svg className="animate-spin w-3.5 h-3.5" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                </svg>
                <span>Extracting...</span>
              </>
            ) : (
              <>
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                <span>Extract Text</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Secondary Action Toolbar: View Mode, Transforms, Export, Copy */}
      <div className="px-3 py-2 border-b border-hairline-light dark:border-hairline-dark flex items-center justify-between bg-inset-light/50 dark:bg-inset-dark/50 text-xs">
        {/* View Switch: Editor vs Audit Blocks */}
        <div className="flex items-center space-x-1">
          <button
            onClick={() => setAuditMode(false)}
            className={`px-2 py-1 rounded text-[11px] font-medium transition-colors ${
              !auditMode
                ? 'bg-surface-light dark:bg-surface-dark text-ink-primaryLight dark:text-ink-primaryDark border border-hairline-light dark:border-hairline-dark'
                : 'text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark'
            }`}
          >
            Editor
          </button>
          <button
            onClick={() => setAuditMode(true)}
            className={`px-2 py-1 rounded text-[11px] font-medium transition-colors ${
              auditMode
                ? 'bg-surface-light dark:bg-surface-dark text-brand-light dark:text-brand-dark border border-brand-light/30 dark:border-brand-dark/30 font-semibold'
                : 'text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark'
            }`}
          >
            Audit Mode
          </button>
        </div>

        {/* Right Tools: Transform Menus, Export, Copy */}
        <div className="flex items-center space-x-1">
          {/* Quick Copy */}
          <button
            onClick={handleCopy}
            disabled={!extractedText}
            title="Copy to clipboard"
            className="flex items-center space-x-1 px-2 py-1 rounded border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark disabled:opacity-40 transition-colors"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
            </svg>
            <span className="text-[11px]">{copied ? 'Copied!' : 'Copy'}</span>
          </button>

          {/* Export Dropdown */}
          <div className="relative">
            <button
              onClick={() => setExportOpen(!exportOpen)}
              disabled={!extractedText}
              className="flex items-center space-x-1 px-2 py-1 rounded border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark disabled:opacity-40 transition-colors"
            >
              <span className="text-[11px]">Export</span>
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>

            {exportOpen && (
              <div className="absolute right-0 top-full mt-1 w-32 rounded-panel border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark shadow-xl py-1 z-30">
                {(['txt', 'md', 'csv', 'tsv', 'json'] as const).map((fmt) => (
                  <button
                    key={fmt}
                    onClick={() => handleExport(fmt)}
                    className="w-full text-left px-3 py-1 text-xs uppercase font-mono text-ink-primaryLight dark:text-ink-primaryDark hover:bg-inset-light dark:hover:bg-inset-dark transition-colors"
                  >
                    .{fmt}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Post-Processing Deterministic Tools Bar */}
      <div className="px-3 py-1.5 border-b border-hairline-light dark:border-hairline-dark flex items-center justify-between overflow-x-auto text-[11px] font-mono text-ink-secondaryLight dark:text-ink-secondaryDark bg-inset-light/30 dark:bg-inset-dark/30">
        <span className="uppercase text-[9px] mr-1">Tools:</span>
        <div className="flex items-center space-x-1.5">
          <button
            onClick={() => handleTransform('digits_to_english')}
            title="Convert Bengali numerals (০-৯) to standard 0-9"
            className="hover:text-brand-light dark:hover:text-brand-dark px-1.5 py-0.5 rounded border border-hairline-light/50 dark:border-hairline-dark/50"
          >
            ০-৯ → 0-9
          </button>
          <button
            onClick={() => handleTransform('digits_to_bengali')}
            title="Convert Arabic numerals (0-9) to Bengali ০-৯"
            className="hover:text-brand-light dark:hover:text-brand-dark px-1.5 py-0.5 rounded border border-hairline-light/50 dark:border-hairline-dark/50"
          >
            0-9 → ০-৯
          </button>
          <button
            onClick={() => handleTransform('unwrap_lines')}
            title="Unwrap artificial line breaks from scanner margins"
            className="hover:text-brand-light dark:hover:text-brand-dark px-1.5 py-0.5 rounded border border-hairline-light/50 dark:border-hairline-dark/50"
          >
            Unwrap
          </button>
          <button
            onClick={() => handleTransform('clean_tables')}
            title="Format and align markdown table pipes"
            className="hover:text-brand-light dark:hover:text-brand-dark px-1.5 py-0.5 rounded border border-hairline-light/50 dark:border-hairline-dark/50"
          >
            Align Table
          </button>
          <button
            onClick={handleCheckMath}
            title="Deterministically audit invoice arithmetic"
            className="hover:text-brand-light dark:hover:text-brand-dark px-1.5 py-0.5 rounded border border-brand-light/40 dark:border-brand-dark/40 text-brand-light dark:text-brand-dark font-semibold"
          >
            Audit Math
          </button>
        </div>
      </div>

      {/* Math Verification Alert Banner */}
      {mathResult && (
        <div
          className={`px-3 py-2 border-b text-xs flex items-center justify-between ${
            mathResult.matched
              ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-700 dark:text-emerald-400'
              : 'bg-rose-500/10 border-rose-500/30 text-rose-700 dark:text-rose-400'
          }`}
        >
          <div className="flex items-center space-x-1.5">
            <span className="font-bold">{mathResult.matched ? '✓ Math Verified:' : '⚠ Discrepancy Found:'}</span>
            <span>
              Total {mathResult.total.toFixed(2)} vs Calculated {mathResult.calculated.toFixed(2)}
            </span>
          </div>
          <button onClick={() => setMathResult(null)} className="p-0.5 hover:opacity-75">
            ✕
          </button>
        </div>
      )}

      {/* Text Area / Paragraph Blocks View */}
      <div className="flex-1 overflow-y-auto p-4 select-text">
        {!selectedItem ? (
          <div className="h-full flex items-center justify-center text-xs text-ink-secondaryLight dark:text-ink-secondaryDark">
            No document selected
          </div>
        ) : auditMode ? (
          /* Synchronized Audit Blocks Mode */
          <div className="space-y-3">
            {paragraphs.length === 0 ? (
              <p className="text-xs text-ink-secondaryLight dark:text-ink-secondaryDark">No text extracted yet.</p>
            ) : (
              paragraphs.map((para, idx) => {
                const isActive = activeBlockIndex === idx;
                return (
                  <div
                    key={idx}
                    onClick={() => setActiveBlockIndex(idx)}
                    className={`p-3 rounded-panel border transition-all cursor-pointer ${
                      isActive
                        ? 'border-brand-light dark:border-brand-dark bg-inset-light dark:bg-inset-dark ring-1 ring-brand-light/30'
                        : 'border-hairline-light dark:border-hairline-dark hover:border-brand-light/40 bg-surface-light dark:bg-surface-dark'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1.5">
                      <span
                        className={`font-mono text-[10px] font-bold px-1.5 py-0.5 rounded ${
                          isActive
                            ? 'bg-brand-light dark:bg-brand-dark text-white'
                            : 'bg-inset-light dark:bg-inset-dark text-ink-secondaryLight dark:text-ink-secondaryDark'
                        }`}
                      >
                        BLOCK {String(idx + 1).padStart(2, '0')}
                      </span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          navigator.clipboard.writeText(para);
                        }}
                        className="text-[10px] text-ink-secondaryLight hover:text-ink-primaryLight dark:text-ink-secondaryDark dark:hover:text-ink-primaryDark"
                      >
                        Copy block
                      </button>
                    </div>
                    <p className="text-xs font-sans whitespace-pre-wrap leading-relaxed text-ink-primaryLight dark:text-ink-primaryDark">
                      {para}
                    </p>
                  </div>
                );
              })
            )}
          </div>
        ) : (
          /* Editor Mode */
          <textarea
            value={extractedText}
            onChange={(e) => updateItem(selectedItem.id, { extracted_text: e.target.value })}
            placeholder="Extracted text will appear here. You can freely edit or format it."
            className="w-full h-full resize-none bg-transparent font-mono text-xs leading-relaxed text-ink-primaryLight dark:text-ink-primaryDark outline-none"
          />
        )}
      </div>

      {/* Footer Metrics */}
      <div className="p-2.5 border-t border-hairline-light dark:border-hairline-dark flex items-center justify-between text-[11px] font-mono text-ink-secondaryLight dark:text-ink-secondaryDark">
        <div className="flex items-center space-x-3">
          <span>{charCount} chars</span>
          <span>·</span>
          <span>{wordCount} words</span>
          <span>·</span>
          <span>{paragraphs.length} blocks</span>
        </div>
        <div className="text-[10px] uppercase">{activeOutputMode}</div>
      </div>
    </div>
  );
};
