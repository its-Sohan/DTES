import React, { useEffect } from 'react';
import {
  initApp,
  useAppStore,
  setModal,
  toggleTheme,
  toggleSidebar,
  addItems,
  extractItem,
  runAllPending,
} from './store/useAppStore';
import * as api from '../wailsjs/go/main/App';

import { TopBar } from './components/TopBar';
import { Sidebar } from './components/Sidebar';
import { PreviewPanel } from './components/PreviewPanel';
import { TextPanel } from './components/TextPanel';
import { CommandPaletteModal } from './components/CommandPaletteModal';
import { SettingsModal } from './components/SettingsModal';
import { DashboardModal } from './components/DashboardModal';
import { UpdateDialog } from './components/UpdateDialog';
import { HelpModal } from './components/HelpModal';

export const App: React.FC = () => {
  const { themeMode, activeModal, queue, selectedItemId } = useAppStore();

  useEffect(() => {
    initApp();
  }, []);

  // Global Keyboard Shortcuts
  useEffect(() => {
    const handleKeyDown = async (e: KeyboardEvent) => {
      const isCtrlOrCmd = e.ctrlKey || e.metaKey;
      const targetTag = (e.target as HTMLElement)?.tagName?.toLowerCase();
      const isInput = targetTag === 'input' || targetTag === 'textarea';

      // Escape closes modals
      if (e.key === 'Escape' && activeModal !== 'none') {
        e.preventDefault();
        setModal('none');
        return;
      }

      // Ctrl + K: Command Palette
      if (isCtrlOrCmd && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setModal(activeModal === 'command_palette' ? 'none' : 'command_palette');
        return;
      }

      // Ctrl + T: Theme Toggle
      if (isCtrlOrCmd && e.key.toLowerCase() === 't') {
        e.preventDefault();
        toggleTheme();
        return;
      }

      // Ctrl + B: Toggle Sidebar
      if (isCtrlOrCmd && e.key.toLowerCase() === 'b') {
        e.preventDefault();
        toggleSidebar();
        return;
      }

      // Ctrl + O: Browse Files
      if (isCtrlOrCmd && e.key.toLowerCase() === 'o') {
        e.preventDefault();
        const items = await api.PickFiles();
        if (items) addItems(items);
        return;
      }

      // Ctrl + Shift + Enter: Run All Pending
      if (isCtrlOrCmd && e.shiftKey && e.key === 'Enter') {
        e.preventDefault();
        runAllPending();
        return;
      }

      // Ctrl + Enter: Extract selected
      if (isCtrlOrCmd && !e.shiftKey && e.key === 'Enter') {
        const item = queue.find((i) => i.id === selectedItemId);
        if (item) {
          e.preventDefault();
          extractItem(item);
        }
        return;
      }

      // Paste shortcut when not in input
      if (isCtrlOrCmd && e.key.toLowerCase() === 'v' && !isInput) {
        try {
          const item = await api.GetClipboardImage();
          if (item) addItems([item]);
        } catch {
          // Ignore
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [activeModal, queue, selectedItemId]);

  return (
    <div className={`h-screen w-screen flex flex-col ${themeMode === 'dark' ? 'dark' : ''} bg-desk-light dark:bg-desk-dark text-ink-primaryLight dark:text-ink-primaryDark overflow-hidden font-sans`}>
      {/* Top Application Bar */}
      <TopBar />

      {/* Main Workspace Layout: Ingestion Sidebar + Document Canvas + Extracted Text Panel */}
      <div className="flex-1 flex overflow-hidden">
        <Sidebar />
        <PreviewPanel />
        <TextPanel />
      </div>

      {/* Application Modals */}
      <CommandPaletteModal />
      <SettingsModal />
      <DashboardModal />
      <UpdateDialog />
      <HelpModal />
    </div>
  );
};

export default App;
