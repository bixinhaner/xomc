import { useState, useCallback, useMemo } from 'react';
import type { MMLCommand } from '@/types/mml';
import { useAllMMLCommands } from '@/hooks/api/useMML';

export function useCommandSelection() {
  const [selectedCommand, setSelectedCommand] = useState<MMLCommand | null>(null);
  const [searchText, setSearchText] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<string>('');

  const { data: allCommands = [] } = useAllMMLCommands();

  // 过滤后的命令列表
  const filteredCommands = useMemo(() => {
    return allCommands.filter((command) => {
      const matchCategory = !categoryFilter || command.category === categoryFilter;
      const matchSearch = !searchText ||
        command.commandName.includes(searchText) ||
        command.commandCode.toLowerCase().includes(searchText.toLowerCase());
      return matchCategory && matchSearch;
    });
  }, [allCommands, searchText, categoryFilter]);

  // 获取所有分类
  const categories = useMemo(() => {
    return [...new Set(allCommands.map((cmd) => cmd.category))];
  }, [allCommands]);

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
    allCommands,

    // 操作
    setSearchText,
    setCategoryFilter,
    selectCommand,
    clearSelection,
  };
}
