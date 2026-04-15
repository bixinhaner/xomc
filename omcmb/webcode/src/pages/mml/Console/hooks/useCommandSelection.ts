import { useState, useCallback, useEffect, useMemo } from 'react';
import type { TreeDataNode } from 'antd';
import { useQuery } from '@tanstack/react-query';
import type { MMLCommand } from '@/types/mml';
import { mmlApi } from '@/services/api/mmlApi';
import { useDictionary } from '@/hooks/api/useSystem';
import { COMMAND_PAGE_SIZE } from '../constants';

export interface CommandTreeNode extends TreeDataNode {
  nodeType: 'category' | 'command';
  label: string;
  category?: string;
  count?: number;
  commandCode?: string;
  command?: MMLCommand;
  children?: CommandTreeNode[];
}

export function useCommandSelection() {
  const [selectedCommand, setSelectedCommand] = useState<MMLCommand | null>(null);
  const [searchText, setSearchText] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');

  const keyword = searchText.trim() || undefined;
  const category = categoryFilter || undefined;

  const { data: commandsResponse, isLoading, isFetching } = useQuery({
    queryKey: ['mml-commands', { category: category ?? '', keyword: keyword ?? '' }],
    queryFn: () =>
      mmlApi.getCommands({
        page: 1,
        pageSize: COMMAND_PAGE_SIZE,
        category,
        keyword,
      }),
  });

  const { data: categoryDict } = useDictionary('mml_command_category');

  const commands = useMemo(() => commandsResponse?.items ?? [], [commandsResponse]);

  const categoryOptions = useMemo(() => {
    const dictDetails = categoryDict?.sysDictionaryDetails;
    if (dictDetails && dictDetails.length > 0) {
      return dictDetails.map((detail) => ({
        label: detail.label,
        value: detail.value,
      }));
    }

    return [...new Set(commands.map((command) => command.category))].map((value) => ({
      label: value,
      value,
    }));
  }, [categoryDict, commands]);

  const treeData = useMemo((): CommandTreeNode[] => {
    if (commands.length === 0) {
      return [];
    }

    const categoryLabelMap = new Map(categoryOptions.map((option) => [option.value, option.label]));
    const groupedCommands = commands.reduce<Map<string, MMLCommand[]>>((map, command) => {
      const current = map.get(command.category) ?? [];
      current.push(command);
      map.set(command.category, current);
      return map;
    }, new Map<string, MMLCommand[]>());

    const orderedCategories = [
      ...categoryOptions.map((option) => option.value).filter((value) => groupedCommands.has(value)),
      ...[...groupedCommands.keys()].filter(
        (value) => !categoryOptions.some((option) => option.value === value)
      ),
    ];

    return orderedCategories.map((categoryValue) => {
      const categoryCommands = groupedCommands.get(categoryValue) ?? [];
      const categoryLabel = categoryLabelMap.get(categoryValue) || categoryValue;

      return {
        key: `cat-${categoryValue}`,
        title: categoryLabel,
        label: categoryLabel,
        category: categoryValue,
        count: categoryCommands.length,
        nodeType: 'category',
        selectable: false,
        children: categoryCommands.map((command) => ({
          key: command.id,
          title: `${command.commandName} ${command.commandCode}`,
          label: command.commandName,
          commandCode: command.commandCode,
          category: command.category,
          command,
          nodeType: 'command',
          isLeaf: true,
        })),
      };
    });
  }, [categoryOptions, commands]);

  useEffect(() => {
    if (selectedCommand && !commands.some((command) => command.id === selectedCommand.id)) {
      setSelectedCommand(null);
    }
  }, [commands, selectedCommand]);

  const selectCommand = useCallback(async (command: MMLCommand | null) => {
    if (!command) {
      setSelectedCommand(null);
      return;
    }

    const detail = await mmlApi.getCommandById(command.id);
    setSelectedCommand(detail ?? command);
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedCommand(null);
  }, []);

  return {
    selectedCommand,
    searchText,
    categoryFilter,
    commands,
    treeData,
    categoryOptions,
    isLoading: isLoading || isFetching,
    setSearchText,
    setCategoryFilter,
    selectCommand,
    clearSelection,
  };
}
