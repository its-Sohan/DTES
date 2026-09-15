import React, { useState, useRef, useEffect } from 'react';
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
import {
  WordToken,
  tokenizeParagraph,
  reconstructParagraph,
  isSuspiciousWord,
  findWordBoundaries,
} from '../utils/ocrUtils';

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
  const [transformOpen, setTransformOpen] = useState<boolean>(false);

  // Word-level editing state in Audit Mode
  const [editingTokenId, setEditingTokenId] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState<string>('');

  // Quick-Correct mode in Textarea Editor (1 click = select word, 2 clicks = drop caret)
  const [quickCorrect, setQuickCorrect] = useState<boolean>(true);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const blockRefs = useRef<{ [key: number]: HTMLDivElement | null }>({});
  const lastClickRef = useRef<{ start: number; end: number; time: number }>({
    start: -1,
    end: -1,
    time: 0,
  });

  const selectedItem = queue.find((i) => i.id === selectedItemId);
  const extractedText = selectedItem?.extracted_text || '';

  // Auto-scroll the active block into view when activeBlockIndex changes
  useEffect(() => {
    if (auditMode && blockRefs.current[activeBlockIndex]) {
      blockRefs.current[activeBlockIndex]?.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest',
      });
    }
  }, [activeBlockIndex, auditMode]);

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

  const commitWordEdit = (
    blockIdx: number,
    tokenIdx: number,
    newWord: string,
    currentTokens: WordToken[]
  ) => {
    if (!selectedItem) return;
    setEditingTokenId(null);

    const trimmedWord = newWord.trim();
    if (!trimmedWord && !currentTokens[tokenIdx].text) return;
    if (trimmedWord === currentTokens[tokenIdx].text) return;

    const updatedTokens = [...currentTokens];
    updatedTokens[tokenIdx] = {
      ...updatedTokens[tokenIdx],
      text: trimmedWord || currentTokens[tokenIdx].text,
      isSuspicious: isSuspiciousWord(trimmedWord),
    };

    const newParaText = reconstructParagraph(updatedTokens);
    const updatedParagraphs = [...paragraphs];
    updatedParagraphs[blockIdx] = newParaText;

    const fullText = updatedParagraphs.join('\n\n');
    updateItem(selectedItem.id, { extracted_text: fullText });
  };

  const jumpToToken = (
    blockIdx: number,
    targetTokenIdx: number,
    allParagraphs: string[]
  ) => {
    if (blockIdx < 0 || blockIdx >= allParagraphs.length) return;
    const tokens = tokenizeParagraph(allParagraphs[blockIdx], blockIdx);

    if (targetTokenIdx >= 0 && targetTokenIdx < tokens.length) {
      setEditingTokenId(tokens[targetTokenIdx].id);
      setEditingValue(tokens[targetTokenIdx].text);
    } else if (targetTokenIdx >= tokens.length && blockIdx + 1 < allParagraphs.length) {
      setActiveBlockIndex(blockIdx + 1);
      const nextTokens = tokenizeParagraph(allParagraphs[blockIdx + 1], blockIdx + 1);
      if (nextTokens.length > 0) {
        setEditingTokenId(nextTokens[0].id);
        setEditingValue(nextTokens[0].text);
      }
    } else if (targetTokenIdx < 0 && blockIdx - 1 >= 0) {
      setActiveBlockIndex(blockIdx - 1);
      const prevTokens = tokenizeParagraph(allParagraphs[blockIdx - 1], blockIdx - 1);
      if (prevTokens.length > 0) {
        const lastIdx = prevTokens.length - 1;
        setEditingTokenId(prevTokens[lastIdx].id);
        setEditingValue(prevTokens[lastIdx].text);
      }
    }
  };

  const handleTextareaClick = () => {
    if (!quickCorrect || !textareaRef.current) return;
    const textarea = textareaRef.current;
    const clickPos = textarea.selectionStart;

    const { start, end } = findWordBoundaries(extractedText, clickPos);
    if (start >= end) return;

    const now = Date.now();
    const last = lastClickRef.current;

    // If clicked on the already selected word range within 1.5s,
    // let standard caret drop happen (2nd click drops cursor)
    if (last.start === start && last.end === end && now - last.time < 1500) {
      lastClickRef.current = { start: -1, end: -1, time: 0 };
      return;
    }

    // 1st click: select the full word
    lastClickRef.current = { start, end, time: now };
    setTimeout(() => {
      if (textareaRef.current) {
        textareaRef.current.setSelectionRange(start, end);
      }
    }, 10);
  };

  return (
    <div className="w-[520px] flex flex-col h-full bg-surface-light dark:bg-surface-dark border-l border-hairline-light dark:border-hairline-dark select-none">
      {/* Top Controls: Modes & Primary Extract */}
      <div className="p-3 border-b border-hairline-light dark:border-hairline-dark space-y-2.5">
        {/* Mode Selector Buttons */}
        <div className="flex items-center p-1 rounded-md bg-inset-light dark:bg-inset-dark border border-hairline-light dark:border-hairline-dark text-xs gap-1">
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
                className={`flex-1 py-1 px-2.5 rounded text-xs font-medium transition-all duration-150 ${isActive
                  ? 'bg-brand-light dark:bg-brand-dark text-white shadow-sm font-semibold'
                  : 'text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark hover:bg-surface-light/60 dark:hover:bg-surface-dark/60'
                  }`}
              >
                {m.label}
              </button>
            );
          })}
        </div>

        {/* Quality & Primary Action */}
        <div className="flex items-center justify-between space-x-2">
          {/* Quality Pure Typography & Color Switch */}
          <div className="flex items-center space-x-3 text-xs">
            <span className="font-mono text-[10px] text-ink-secondaryLight/50 dark:text-ink-secondaryDark/50 uppercase">
              Mode
            </span>
            <div className="flex items-center space-x-2.5">
              <button
                type="button"
                onClick={() => setQuality('standard')}
                title="Standard: fast extraction with vision LLM"
                className={`flex items-center space-x-1.5 transition-all ${
                  activeQuality === 'standard'
                    ? 'text-ink-primaryLight dark:text-ink-primaryDark font-semibold'
                    : 'text-ink-secondaryLight/50 dark:text-ink-secondaryDark/50 hover:text-ink-secondaryLight dark:hover:text-ink-secondaryDark'
                }`}
              >
                <span
                  className={`w-1.5 h-1.5 rounded-full transition-all ${
                    activeQuality === 'standard' ? 'bg-brand-light dark:bg-brand-dark scale-100' : 'bg-transparent scale-0'
                  }`}
                />
                <span>Standard</span>
              </button>

              <span className="text-ink-secondaryLight/20 dark:text-ink-secondaryDark/20 font-mono text-[10px]">/</span>

              <button
                type="button"
                onClick={() => setQuality('high')}
                title="High Precision: upgraded vision model for difficult scripts"
                className={`flex items-center space-x-1.5 transition-all ${
                  activeQuality === 'high'
                    ? 'text-brand-light dark:text-brand-dark font-semibold'
                    : 'text-ink-secondaryLight/50 dark:text-ink-secondaryDark/50 hover:text-ink-secondaryLight dark:hover:text-ink-secondaryDark'
                }`}
              >
                <span
                  className={`w-1.5 h-1.5 rounded-full transition-all ${
                    activeQuality === 'high' ? 'bg-brand-light dark:bg-brand-dark scale-100' : 'bg-transparent scale-0'
                  }`}
                />
                <span>High Precision</span>
              </button>

              <span className="text-ink-secondaryLight/20 dark:text-ink-secondaryDark/20 font-mono text-[10px]">/</span>

              <button
                type="button"
                onClick={() => setQuality('document')}
                title="Document Mode: routes directly to dedicated OCR model"
                className={`flex items-center space-x-1.5 transition-all ${
                  activeQuality === 'document'
                    ? 'text-brand-light dark:text-brand-dark font-semibold'
                    : 'text-ink-secondaryLight/50 dark:text-ink-secondaryDark/50 hover:text-ink-secondaryLight dark:hover:text-ink-secondaryDark'
                }`}
              >
                <span
                  className={`w-1.5 h-1.5 rounded-full transition-all ${
                    activeQuality === 'document' ? 'bg-brand-light dark:bg-brand-dark scale-100' : 'bg-transparent scale-0'
                  }`}
                />
                <span>Document</span>
              </button>
            </div>
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
            className={`px-2 py-1 rounded text-[11px] font-medium transition-colors ${!auditMode
              ? 'bg-surface-light dark:bg-surface-dark text-ink-primaryLight dark:text-ink-primaryDark border border-hairline-light dark:border-hairline-dark'
              : 'text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark'
              }`}
          >
            Editor
          </button>
          <button
            onClick={() => setAuditMode(true)}
            className={`px-2 py-1 rounded text-[11px] font-medium transition-colors ${auditMode
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

          {/* Transform Dropdown */}
          <div className="relative">
            <button
              onClick={() => {
                setTransformOpen(!transformOpen);
                setExportOpen(false);
              }}
              disabled={!extractedText}
              className="flex items-center space-x-1 px-2 py-1 rounded border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark disabled:opacity-40 transition-colors"
            >
              <span className="text-[11px]">Transform</span>
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>

            {transformOpen && (
              <div className="absolute right-0 top-full mt-1 w-44 rounded-panel border border-hairline-light dark:border-hairline-dark bg-surface-light dark:bg-surface-dark shadow-xl py-1 z-30 text-xs">
                <button
                  onClick={() => {
                    handleTransform('digits_to_english');
                    setTransformOpen(false);
                  }}
                  className="w-full text-left px-3 py-1.5 text-ink-primaryLight dark:text-ink-primaryDark hover:bg-inset-light dark:hover:bg-inset-dark transition-colors flex items-center justify-between"
                >
                  <span>Numerals to English</span>
                  <span className="font-mono text-[10px] text-ink-secondaryLight/60 dark:text-ink-secondaryDark/60">০→0</span>
                </button>
                <button
                  onClick={() => {
                    handleTransform('digits_to_bengali');
                    setTransformOpen(false);
                  }}
                  className="w-full text-left px-3 py-1.5 text-ink-primaryLight dark:text-ink-primaryDark hover:bg-inset-light dark:hover:bg-inset-dark transition-colors flex items-center justify-between"
                >
                  <span>Numerals to Bengali</span>
                  <span className="font-mono text-[10px] text-ink-secondaryLight/60 dark:text-ink-secondaryDark/60">0→০</span>
                </button>
                <div className="my-1 border-t border-hairline-light dark:border-hairline-dark" />
                <button
                  onClick={() => {
                    handleTransform('unwrap_lines');
                    setTransformOpen(false);
                  }}
                  className="w-full text-left px-3 py-1.5 text-ink-primaryLight dark:text-ink-primaryDark hover:bg-inset-light dark:hover:bg-inset-dark transition-colors"
                >
                  Unwrap Margins
                </button>
                <button
                  onClick={() => {
                    handleTransform('clean_tables');
                    setTransformOpen(false);
                  }}
                  className="w-full text-left px-3 py-1.5 text-ink-primaryLight dark:text-ink-primaryDark hover:bg-inset-light dark:hover:bg-inset-dark transition-colors"
                >
                  Align Markdown Tables
                </button>
              </div>
            )}
          </div>

          {/* Export Dropdown */}
          <div className="relative">
            <button
              onClick={() => {
                setExportOpen(!exportOpen);
                setTransformOpen(false);
              }}
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
            onClick={() => setQuickCorrect(!quickCorrect)}
            title={
              quickCorrect
                ? 'Quick-Correct ON: Single click selects full word for fast replacement. 2nd click drops caret.'
                : 'Quick-Correct OFF: Standard cursor placement.'
            }
            className={`px-1.5 py-0.5 rounded border transition-colors flex items-center space-x-1 ${
              quickCorrect
                ? 'border-brand-light/50 dark:border-brand-dark/50 bg-brand-light/10 dark:bg-brand-dark/15 text-brand-light dark:text-brand-dark font-semibold'
                : 'border-hairline-light/50 dark:border-hairline-dark/50 text-ink-secondaryLight dark:text-ink-secondaryDark hover:text-ink-primaryLight dark:hover:text-ink-primaryDark'
            }`}
          >
            <span>⚡ Quick-Correct</span>
          </button>
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
        </div>
      </div>

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
                const tokens = tokenizeParagraph(para, idx);
                const suspiciousCount = tokens.filter((t) => t.isSuspicious).length;

                return (
                  <div
                    key={idx}
                    ref={(el) => {
                      blockRefs.current[idx] = el;
                    }}
                    onClick={() => setActiveBlockIndex(idx)}
                    className={`p-3 rounded-panel border transition-all cursor-pointer ${
                      isActive
                        ? 'border-brand-light dark:border-brand-dark bg-inset-light dark:bg-inset-dark ring-1 ring-brand-light/30 shadow-sm'
                        : 'border-hairline-light dark:border-hairline-dark hover:border-brand-light/40 bg-surface-light dark:bg-surface-dark'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center space-x-2">
                        <span
                          className={`font-mono text-[10px] font-bold px-1.5 py-0.5 rounded ${
                            isActive
                              ? 'bg-brand-light dark:bg-brand-dark text-white'
                              : 'bg-inset-light dark:bg-inset-dark text-ink-secondaryLight dark:text-ink-secondaryDark'
                          }`}
                        >
                          BLOCK {String(idx + 1).padStart(2, '0')}
                        </span>
                        {suspiciousCount > 0 && (
                          <span
                            title="Words that may contain OCR recognition errors"
                            className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-amber-500/15 dark:bg-amber-400/20 text-amber-700 dark:text-amber-300 font-semibold"
                          >
                            ⚠ {suspiciousCount} suspicious
                          </span>
                        )}
                      </div>
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

                    {/* Interactive Word Tokens */}
                    <p className="text-xs font-sans whitespace-pre-wrap leading-relaxed text-ink-primaryLight dark:text-ink-primaryDark">
                      {tokens.map((token, tokenIdx) => {
                        const isEditing = editingTokenId === token.id;

                        if (isEditing) {
                          return (
                            <React.Fragment key={token.id}>
                              <input
                                type="text"
                                autoFocus
                                value={editingValue}
                                ref={(input) => {
                                  if (input) {
                                    // 1st press selects whole word; 2nd press allows cursor caret to drop
                                    input.select();
                                  }
                                }}
                                onChange={(e) => setEditingValue(e.target.value)}
                                onBlur={() => {
                                  commitWordEdit(idx, tokenIdx, editingValue, tokens);
                                }}
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter') {
                                    e.preventDefault();
                                    commitWordEdit(idx, tokenIdx, editingValue, tokens);
                                  } else if (e.key === 'Tab') {
                                    e.preventDefault();
                                    commitWordEdit(idx, tokenIdx, editingValue, tokens);
                                    if (e.shiftKey) {
                                      jumpToToken(idx, tokenIdx - 1, paragraphs);
                                    } else {
                                      jumpToToken(idx, tokenIdx + 1, paragraphs);
                                    }
                                  } else if (e.key === 'Escape') {
                                    e.preventDefault();
                                    setEditingTokenId(null);
                                  }
                                }}
                                className="font-mono text-xs px-1 py-0.5 rounded border border-brand-light dark:border-brand-dark bg-surface-light dark:bg-surface-dark text-ink-primaryLight dark:text-ink-primaryDark outline-none ring-1 ring-brand-light/50 shadow-sm"
                                style={{
                                  width: `${Math.max(3, editingValue.length + 1)}ch`,
                                }}
                              />
                              <span className="whitespace-pre">{token.trailingSpace}</span>
                            </React.Fragment>
                          );
                        }

                        return (
                          <React.Fragment key={token.id}>
                            <span
                              onClick={(e) => {
                                e.stopPropagation();
                                setActiveBlockIndex(idx);
                                setEditingTokenId(token.id);
                                setEditingValue(token.text);
                              }}
                              title={
                                token.isSuspicious
                                  ? `Suspicious OCR token "${token.text}" - Click to replace`
                                  : `Click to replace "${token.text}"`
                              }
                              className={`inline-block px-0.5 rounded cursor-pointer transition-colors ${
                                token.isSuspicious
                                  ? 'border-b-2 border-dashed border-amber-500/80 bg-amber-500/15 dark:bg-amber-400/20 text-amber-900 dark:text-amber-200 font-semibold'
                                  : 'hover:bg-brand-light/10 dark:hover:bg-brand-dark/20 hover:text-brand-light dark:hover:text-brand-dark'
                              }`}
                            >
                              {token.text}
                            </span>
                            <span className="whitespace-pre">{token.trailingSpace}</span>
                          </React.Fragment>
                        );
                      })}
                    </p>
                  </div>
                );
              })
            )}
          </div>
        ) : (
          /* Editor Mode */
          <textarea
            ref={textareaRef}
            value={extractedText}
            onClick={handleTextareaClick}
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
