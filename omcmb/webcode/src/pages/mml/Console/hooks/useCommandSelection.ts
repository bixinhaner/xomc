import { useState, useCallback, useMemo } from 'react';
import type { MMLCommand } from '@/types/mml';
import { useAllMMLCommands } from '@/hooks/api/useMML';
import { useDictionary } from '@/hooks/api/useSystem';

export function useCommandSelection() {
  const [selectedCommand, setSelectedCommand] = useState<MMLCommand | null>(null);
  const [searchText, setSearchText] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<string>('');

  const { data: allCommands = [] } = useAllMMLCommands();
  const { data: categoryDict } = useDictionary('mml_command_category');

  // 分类选项（字典驱动，fallback 到从命令数据提取）
  const categoryOptions = useMemo(
    () => {
      const dictDetails = categoryDict?.sysDictionaryDetails;
      if (dictDetails && dictDetails.length > 0) {
        return dictDetails.map((d) => ({ label: d.label, value: d.value }));
      }
      return [...new Set(allCommands.map((cmd) => cmd.category))].map((c) => ({ label: c, value: c }));
    },
    [categoryDict, allCommands]
  );

  // 所有分类值（用于树分组）
  const categories = useMemo(
    () => categoryOptions.map((o) => o.value),
    [categoryOptions]
  );

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
    categoryOptions,
    commandsByCategory,
    allCommands,

    // 操作
    setSearchText,
    setCategoryFilter,
    selectCommand,
    clearSelection,
  };
}
