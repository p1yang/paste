import React, { useRef, useEffect } from 'react';
import { motion } from 'framer-motion';
import type { ClipboardItem } from '../types';
import { useAppStore } from '../stores/appStore';
import { StarIcon, TrashIcon, CopyIcon, TextIcon, ImageIcon } from './Icons';
import { formatRelativeTime, formatFirstLine, truncateText, formatBytes, countLines } from '../utils/format';

interface ClipboardCardProps {
  item: ClipboardItem;
  index: number;
  isSelected: boolean;
}

export const ClipboardCard: React.FC<ClipboardCardProps> = ({ item, index, isSelected }) => {
  const { toggleFavorite, deleteItem, copyItem, pasteItem, setSelectedIndex, config } = useAppStore();
  const cardRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isSelected && cardRef.current) {
      cardRef.current.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
  }, [isSelected]);

  const handleClick = () => {
    setSelectedIndex(index);
  };

  const handleDoubleClick = async () => {
    if (config?.autoPaste) {
      await pasteItem(item.id);
    } else {
      await copyItem(item.id);
    }
  };

  const handleToggleFavorite = (e: React.MouseEvent) => {
    e.stopPropagation();
    toggleFavorite(item.id, !item.isFavorite);
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    deleteItem(item.id);
  };

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    await copyItem(item.id);
  };

  const handlePaste = async (e: React.MouseEvent) => {
    e.stopPropagation();
    await pasteItem(item.id);
  };

  return (
    <motion.div
      ref={cardRef}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.15, delay: index * 0.02 }}
      onClick={handleClick}
      onDoubleClick={handleDoubleClick}
      className={`clipboard-item p-3 mb-2 cursor-pointer ${isSelected ? 'selected' : ''}`}
    >
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 mt-0.5">
          {item.type === 'text' ? (
            <TextIcon size={18} className="text-gray-400 dark:text-gray-500" />
          ) : (
            <ImageIcon size={18} className="text-gray-400 dark:text-gray-500" />
          )}
        </div>

        <div className="flex-1 min-w-0">
          {item.type === 'text' ? (
            <div>
              <div className="text-sm text-gray-800 dark:text-gray-200 font-medium leading-tight line-clamp-2">
                {truncateText(formatFirstLine(item.content || ''), 120)}
              </div>
              {(item.content || '').length > 120 || countLines(item.content || '') > 2 ? (
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {formatBytes(item.sizeBytes)} · {countLines(item.content || '')} 行
                </div>
              ) : null}
            </div>
          ) : (
            <div className="flex items-center gap-3">
              <div className="w-16 h-16 rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-800 flex-shrink-0">
                <img
                  src={item.imageUrl}
                  alt="clipboard"
                  className="w-full h-full object-cover"
                  draggable={false}
                />
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">
                图片 · {formatBytes(item.sizeBytes)}
              </div>
            </div>
          )}

          <div className="flex items-center gap-2 mt-2">
            <span className="text-xs text-gray-400 dark:text-gray-500">
              {formatRelativeTime(item.updatedAt)}
            </span>
            {item.appName && (
              <>
                <span className="text-gray-300 dark:text-gray-600">·</span>
                <span className="text-xs text-gray-400 dark:text-gray-500 truncate max-w-[120px]">
                  {item.appName}
                </span>
              </>
            )}
            {item.pasteCount > 0 && (
              <>
                <span className="text-gray-300 dark:text-gray-600">·</span>
                <span className="text-xs text-gray-400 dark:text-gray-500">
                  {item.pasteCount} 次使用
                </span>
              </>
            )}
          </div>
        </div>

        <div className="flex items-center gap-1 flex-shrink-0">
          <button
            className="action-btn"
            onClick={handleToggleFavorite}
            title={item.isFavorite ? '取消收藏' : '收藏'}
          >
            <StarIcon size={16} filled={item.isFavorite} />
          </button>
          <button
            className="action-btn"
            onClick={handleCopy}
            title="复制"
          >
            <CopyIcon size={16} />
          </button>
          <button
            className="action-btn"
            onClick={handlePaste}
            title="粘贴"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 5v14" />
              <path d="m19 12-7 7-7-7" />
            </svg>
          </button>
          <button
            className="action-btn hover:text-red-500"
            onClick={handleDelete}
            title="删除"
          >
            <TrashIcon size={16} />
          </button>
        </div>
      </div>
    </motion.div>
  );
};
