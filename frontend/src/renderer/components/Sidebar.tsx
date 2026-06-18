import React from 'react';
import { useAppStore } from '../stores/appStore';
import type { ViewTab } from '../types';
import { HistoryIcon, HeartIcon, TextIcon, ImageIcon, SettingsIcon } from './Icons';

const tabs: { id: ViewTab; label: string; icon: React.FC<{ size?: number; className?: string }> }[] = [
  { id: 'all', label: '全部', icon: HistoryIcon },
  { id: 'favorites', label: '收藏', icon: HeartIcon },
  { id: 'text', label: '文本', icon: TextIcon },
  { id: 'images', label: '图片', icon: ImageIcon },
  { id: 'settings', label: '设置', icon: SettingsIcon },
];

export const Sidebar: React.FC = () => {
  const { activeTab, setActiveTab, stats } = useAppStore();

  const getCount = (tab: ViewTab): number | undefined => {
    if (!stats) return undefined;
    switch (tab) {
      case 'all':
        return stats.totalItems;
      case 'favorites':
        return stats.favoriteItems;
      case 'text':
        return stats.textItems;
      case 'images':
        return stats.imageItems;
      default:
        return undefined;
    }
  };

  return (
    <div className="w-40 h-full glass-sidebar flex flex-col py-4 px-2">
      <div className="px-3 pb-4 mb-2 border-b border-gray-200/50 dark:border-gray-700/50">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 5v14" />
              <path d="m19 12-7 7-7-7" />
            </svg>
          </div>
          <div>
            <div className="text-sm font-semibold text-gray-800 dark:text-gray-100">Paste</div>
            <div className="text-[10px] text-gray-500 dark:text-gray-400">剪贴板管理</div>
          </div>
        </div>
      </div>

      <nav className="flex-1 space-y-0.5">
        {tabs.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            className={`tab-btn w-full flex items-center gap-2.5 justify-start ${activeTab === id ? 'active' : 'text-gray-600 dark:text-gray-400'}`}
          >
            <Icon size={15} />
            <span className="flex-1 text-left">{label}</span>
            {getCount(id) !== undefined && (
              <span className={`text-xs ${activeTab === id ? 'text-blue-500' : 'text-gray-400 dark:text-gray-500'}`}>
                {getCount(id)}
              </span>
            )}
          </button>
        ))}
      </nav>

      <div className="pt-3 mt-2 border-t border-gray-200/50 dark:border-gray-700/50 px-3">
        <div className="text-[11px] text-gray-500 dark:text-gray-400">
          快捷键
        </div>
        <div className="mt-1 flex items-center gap-1">
          <kbd className="px-1.5 py-0.5 text-[10px] rounded bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300 font-mono">
            ⌘
          </kbd>
          <kbd className="px-1.5 py-0.5 text-[10px] rounded bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300 font-mono">
            ⇧
          </kbd>
          <kbd className="px-1.5 py-0.5 text-[10px] rounded bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300 font-mono">
            V
          </kbd>
        </div>
      </div>
    </div>
  );
};
