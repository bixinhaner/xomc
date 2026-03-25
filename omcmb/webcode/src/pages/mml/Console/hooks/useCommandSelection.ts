import { useState, useCallback, useMemo } from 'react';
import type { MMLCommand } from '@/types/mml';
import { MOCK_COMMANDS } from '../constants';

export function useCommandSelection() {
  const [selectedCommand, setSelectedCommand] = useState<MMLCommand | null>(null);
  const [searchText, setSearchText] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<string>('');

  // 过滤后的命令列表
  const filteredCommands = useMemo(() => {
    return MOCK_COMMANDS.filter((command) => {
      const matchCategory = !categoryFilter || command.category === categoryFilter;
      const matchSearch = !searchText ||
        command.commandName.includes(searchText) ||
        command.commandCode.toLowerCase().includes(searchText.toLowerCase());
      return matchCategory && matchSearch;
    });
  }, [searchText, categoryFilter]);

  // 获取所有分类
  const categories = useMemo(() => {
    return [...new Set(MOCK_COMMANDS.map((cmd) => cmd.category))];
  }, []);

  // 按分类分组的命令
  const commandsByCategory = useMemo(() => {
    const map = new Map<string, MMLCommand[]>();
    filteredCommands.forEach((cmd) => {
      const cmds = map.get(cmd.category) || [];
      cmds.push(cmd);
      map.set(cmd.category, cmds);
    });
    return map;
  }, [filteredCommands]);

  // 选择命令
  const selectCommand = useCallback((command: MMLCommand | null) => {
    setSelectedCommand(command);
  }, []);

  // 清除选择
  const clearSelection = useCallback(() => {
    setSelectedCommand(null);
  }, []);

  return {
    // 状态
    selectedCommand,
    searchText,
    categoryFilter,
    filteredCommands,
    categories,
    commandsByCategory,
    allCommands: MOCK_COMMANDS,

    // 操作
    setSearchText,
    setCategoryFilter,
    selectCommand,
    clearSelection,
  };
}
