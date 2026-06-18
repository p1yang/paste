import React, { useEffect, useRef, useState } from 'react';
import { useAppStore } from './stores/appStore';
import { Sidebar } from './components/Sidebar';
import { ClipboardCard } from './components/ClipboardCard';
import { SettingsPage } from './components/SettingsPage';
import { SearchIcon, CloseIcon } from './components/Icons';

const App: React.FC = () => {
  const {
    items,
    totalItems,
    loading,
    searchQuery,
    selectedIndex,
    activeTab,
    setSearchQuery,
    setSelectedIndex,
    loadItems,
    loadConfig,
    loadStats,
    pasteItem,
    copyItem,
    setIsDark,
    config,
  } = useAppStore();

  const searchRef = useRef<HTMLInputElement>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    loadConfig();
    loadItems();
    loadStats();

    const interval = setInterval(() => {
      setIsRefreshing(true);
      loadItems().finally(() => setIsRefreshing(false));
      loadStats();
    }, 3000);

    return () => clearInterval(interval);
  }, [loadItems, loadConfig, loadStats]);

  useEffect(() => {
    const initTheme = async () => {
      try {
        if (window.pasteAPI) {
          const dark = await window.pasteAPI.shouldUseDarkColors();
          setIsDark(dark);
          if (dark) {
            document.documentElement.classList.add('dark');
          }

          window.pasteAPI.onThemeChanged((dark: boolean) => {
            setIsDark(dark);
            if (dark) {
              document.documentElement.classList.add('dark');
            } else {
              document.documentElement.classList.remove('dark');
            }
          });
        }
      } catch {
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        setIsDark(prefersDark);
        if (prefersDark) {
          document.documentElement.classList.add('dark');
        }
      }
    };
    initTheme();
  }, [setIsDark]);

  useEffect(() => {
    if (config) {
      if (config.theme === 'dark') {
        document.documentElement.classList.add('dark');
        setIsDark(true);
      } else if (config.theme === 'light') {
        document.documentElement.classList.remove('dark');
        setIsDark(false);
      }
    }
  }, [config, setIsDark]);

  useEffect(() => {
    const timer = setTimeout(() => {
      searchRef.current?.focus();
    }, 100);
    return () => clearTimeout(timer);
  }, [activeTab]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (window.pasteAPI) {
          window.pasteAPI.hideWindow();
        }
      }

      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex(selectedIndex + 1);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex(selectedIndex - 1);
      }

      if ((e.key === 'Enter') && items.length > 0) {
        e.preventDefault();
        const item = items[selectedIndex];
        if (item) {
          if (config?.autoPaste) {
            pasteItem(item.id);
          } else {
            copyItem(item.id);
          }
        }
      }

      if (e.key === 'Delete' || e.key === 'Backspace') {
        if (!searchQuery && items.length > 0 && document.activeElement?.tagName !== 'INPUT') {
          e.preventDefault();
          const item = items[selectedIndex];
          if (item) {
            useAppStore.getState().deleteItem(item.id);
          }
        }
      }

      if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
        e.preventDefault();
        searchRef.current?.focus();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedIndex, items, searchQuery, pasteItem, copyItem, setSelectedIndex, config?.autoPaste]);

  const handleHide = () => {
    if (window.pasteAPI) {
      window.pasteAPI.hideWindow();
    }
  };

  const renderContent = () => {
    if (activeTab === 'settings') {
      return <SettingsPage />;
    }

    return (
      <div className="flex flex-col h-full">
        <div className="p-4 pb-3">
          <div className="relative">
            <SearchIcon size={16} className="absolute left-3.5 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              ref={searchRef}
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={
                searchQuery
                  ? `搜索 ${totalItems} 条记录...`
                  : '搜索剪贴板历史...'
              }
              className="search-input pl-9 pr-9"
              autoFocus
              spellCheck={false}
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 p-1 rounded-md hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
              >
                <CloseIcon size={14} className="text-gray-400" />
              </button>
            )}
          </div>

          {searchQuery && (
            <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
              找到 {totalItems} 条相关记录
            </div>
          )}
        </div>

        <div className="flex-1 overflow-y-auto px-4 pb-4">
          {loading ? (
            <div className="flex items-center justify-center h-full">
              <div className="flex flex-col items-center gap-2">
                <div className="w-8 h-8 border-2 border-gray-200 dark:border-gray-700 border-t-blue-500 rounded-full animate-spin" />
                <span className="text-sm text-gray-400">加载中...</span>
              </div>
            </div>
          ) : items.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full gap-3">
              <div className="w-16 h-16 rounded-2xl bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
                <SearchIcon size={28} className="text-gray-300 dark:text-gray-600" />
              </div>
              <div className="text-center">
                <div className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {searchQuery ? '未找到相关记录' : '暂无剪贴板记录'}
                </div>
                <div className="text-xs text-gray-400 mt-1">
                  {searchQuery ? '尝试其他关键词' : '复制一些内容后会自动记录在此'}
                </div>
              </div>
            </div>
          ) : (
            <div>
              {items.map((item, index) => (
                <ClipboardCard
                  key={item.id}
                  item={item}
                  index={index}
                  isSelected={index === selectedIndex}
                />
              ))}

              {items.length > 0 && items.length < totalItems && (
                <div className="text-center py-4 text-xs text-gray-400">
                  显示 {items.length} / {totalItems} 条记录
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="w-full h-full p-3">
      <div className="glass-window w-full h-full flex">
        <Sidebar />
        <div className="flex-1 flex flex-col overflow-hidden">
          <div className="flex items-center justify-between px-4 py-2.5 border-b border-gray-200/50 dark:border-gray-700/30">
            <div className="flex items-center gap-2">
              <span
                className={`inline-block w-2.5 h-2.5 rounded-full ${isRefreshing ? 'bg-green-400 animate-pulse' : 'bg-green-500'}`}
              />
              <span className="text-xs text-gray-500 dark:text-gray-400">
                {isRefreshing ? '同步中...' : '正在监听剪贴板'}
              </span>
            </div>

            <button
              onClick={handleHide}
              className="action-btn hover:opacity-100"
              title="关闭 (Esc)"
            >
              <CloseIcon size={14} />
            </button>
          </div>

          <div className="flex-1 overflow-hidden">{renderContent()}</div>
        </div>
      </div>
    </div>
  );
};

export default App;
