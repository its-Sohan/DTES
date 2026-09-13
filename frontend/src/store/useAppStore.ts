import { useSyncExternalStore } from 'react';
import { Config, QueueItem, UsageStats } from '../types';
import * as api from '../../wailsjs/go/main/App';

export interface AppState {
  queue: QueueItem[];
  selectedItemId: string | null;
  activeOutputMode: 'document' | 'spreadsheet' | 'key_value' | 'raw_text';
  activeQuality: 'standard' | 'high';
  themeMode: 'dark' | 'light';
  auditMode: boolean;
  activeBlockIndex: number;
  config: Config;
  isProcessingAll: boolean;
  isExtracting: boolean;
  sidebarOpen: boolean;
  statusMessage: string;
  activeModal: 'none' | 'settings' | 'dashboard' | 'command_palette' | 'update' | 'help';
}

const initialConfig: Config = {
  session_account: 'default_user',
  api_key: '',
  base_url: 'https://api.openai.com/v1',
  model_name: 'gpt-4o-mini',
  auto_extract: true,
  default_output_mode: 'document',
  quality: 'standard',
  releases_repo: 'its-Sohan/itt-ocr-release',
  check_updates_on_startup: true,
  usage_stats: {
    total_scanned_or_uploaded: 0,
    total_processed: 0,
    total_characters_extracted: 0,
    successful_runs: 0,
    failed_runs: 0,
  },
};

let state: AppState = {
  queue: [],
  selectedItemId: null,
  activeOutputMode: 'document',
  activeQuality: 'standard',
  themeMode: 'dark',
  auditMode: false,
  activeBlockIndex: 0,
  config: initialConfig,
  isProcessingAll: false,
  isExtracting: false,
  sidebarOpen: true,
  statusMessage: 'Ready',
  activeModal: 'none',
};

const listeners = new Set<() => void>();

function notify() {
  listeners.forEach((l) => l());
}

export function getState(): AppState {
  return state;
}

export function setState(partial: Partial<AppState>) {
  state = { ...state, ...partial };
  notify();
}

export function useAppStore(): AppState {
  return useSyncExternalStore(
    (onStoreChange) => {
      listeners.add(onStoreChange);
      return () => listeners.delete(onStoreChange);
    },
    () => state,
    () => state
  );
}

// ---------------- Actions ---------------- //

export async function initApp() {
  try {
    const loadedConfig = await api.GetConfig();
    const history = await api.LoadHistory();
    const initialSelected = history.length > 0 ? history[0].id : null;

    setState({
      config: loadedConfig,
      activeOutputMode: loadedConfig.default_output_mode || 'document',
      activeQuality: loadedConfig.quality || 'standard',
      queue: history,
      selectedItemId: initialSelected,
    });

    if (loadedConfig.check_updates_on_startup) {
      checkUpdatesSilent();
    }
  } catch (err) {
    console.error('Failed to initialize app state:', err);
  }
}

export function toggleTheme() {
  const newTheme = state.themeMode === 'dark' ? 'light' : 'dark';
  if (newTheme === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
  setState({ themeMode: newTheme });
}

export function setModal(modal: AppState['activeModal']) {
  setState({ activeModal: modal });
}

export function toggleSidebar() {
  setState({ sidebarOpen: !state.sidebarOpen });
}

export function selectItem(id: string) {
  setState({ selectedItemId: id, activeBlockIndex: 0 });
}

export function setOutputMode(mode: AppState['activeOutputMode']) {
  setState({ activeOutputMode: mode });
  if (state.selectedItemId) {
    updateItem(state.selectedItemId, { output_mode: mode });
  }
}

export function setQuality(quality: AppState['activeQuality']) {
  setState({ activeQuality: quality });
}

export function setAuditMode(enabled: boolean) {
  setState({ auditMode: enabled });
}

export function setActiveBlockIndex(index: number) {
  setState({ activeBlockIndex: Math.max(0, index) });
}

export function addItems(newItems: QueueItem[]) {
  if (newItems.length === 0) return;
  const updatedQueue = [...state.queue, ...newItems];
  const selected = state.selectedItemId || newItems[0].id;
  setState({ queue: updatedQueue, selectedItemId: selected });
  api.SaveHistory(updatedQueue);

  if (state.config.auto_extract) {
    for (const item of newItems) {
      extractItem(item);
    }
  }
}

export function removeItem(id: string) {
  const updatedQueue = state.queue.filter((i) => i.id !== id);
  let newSelected = state.selectedItemId;
  if (state.selectedItemId === id) {
    newSelected = updatedQueue.length > 0 ? updatedQueue[0].id : null;
  }
  setState({ queue: updatedQueue, selectedItemId: newSelected });
  api.SaveHistory(updatedQueue);
}

export function clearCompleted() {
  const updatedQueue = state.queue.filter((i) => i.status !== 'Done');
  const newSelected = updatedQueue.length > 0 ? updatedQueue[0].id : null;
  setState({ queue: updatedQueue, selectedItemId: newSelected });
  api.SaveHistory(updatedQueue);
}

export function updateItem(id: string, partial: Partial<QueueItem>) {
  const updatedQueue = state.queue.map((item) => {
    if (item.id === id) {
      return { ...item, ...partial };
    }
    return item;
  });
  setState({ queue: updatedQueue });
  api.SaveHistory(updatedQueue);
}

export async function extractItem(item: QueueItem) {
  updateItem(item.id, { status: 'Processing', error_message: '' });
  setState({ isExtracting: true, statusMessage: `Extracting ${item.file_name}...` });

  try {
    const text = await api.ExtractText(item.file_path, state.activeOutputMode, state.activeQuality);
    updateItem(item.id, { status: 'Done', extracted_text: text, output_mode: state.activeOutputMode });
    setState({ isExtracting: false, statusMessage: 'Ready' });

    // Background block bounding box alignment
    alignItemBlocks(item.id, item.file_path, text);
  } catch (err: any) {
    const errMsg = err?.message || String(err);
    updateItem(item.id, { status: 'Failed', error_message: errMsg });
    setState({ isExtracting: false, statusMessage: `Error: ${errMsg}` });
  }
}

async function alignItemBlocks(id: string, filePath: string, text: string) {
  const paragraphs = text
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter((p) => p.length > 0);
  if (paragraphs.length === 0) return;

  try {
    const boxes = await api.AlignBlocks(filePath, paragraphs);
    if (boxes && boxes.length > 0) {
      updateItem(id, { block_boxes: boxes });
    }
  } catch {
    // Non-fatal, alignment is auxiliary
  }
}

export async function runAllPending() {
  const pending = state.queue.filter((i) => i.status === 'Ready' || i.status === 'Failed');
  if (pending.length === 0) return;

  setState({ isProcessingAll: true });
  for (const item of pending) {
    selectItem(item.id);
    await extractItem(item);
  }
  setState({ isProcessingAll: false });
}

async function checkUpdatesSilent() {
  try {
    const res = await api.CheckForUpdates();
    if (res.has_update) {
      setModal('update');
    }
  } catch {
    // Silent
  }
}
