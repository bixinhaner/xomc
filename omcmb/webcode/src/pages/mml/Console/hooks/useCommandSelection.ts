import { useState, useCallback, useEffect, useMemo } from 'react';
import type { TreeDataNode } from 'antd';
import { useQuery } from '@tanstack/react-query';
import type { MMLCommand, MMLCustomCommand, MMLParamRef } from '@core/types/mml';
import { mmlApi } from '@core/services/api/mmlApi';
import { useDictionary } from '@core/hooks/api/useSystem';
import { useT } from '@/hooks/useT';
import { COMMAND_PAGE_SIZE } from '../constants';

export type CustomNodeType = 'custom_root' | 'custom_public' | 'custom_private' | 'user_dir';

export interface CommandTreeNode extends TreeDataNode {
  nodeType: 'category' | 'command' | 'param' | CustomNodeType;
  label: string;
  category?: string;
  count?: number;
  commandCode?: string;
  command?: MMLCommand;
  /** @deprecated Kept for backward compatibility; always undefined in new tree. */
  paramId?: string;
  template?: MMLCustomCommand;
  children?: CommandTreeNode[];
}

export function useCommandSelection() {
  const t = useT();
  const [selectedCommand, setSelectedCommand] = useState<MMLCommand | null>(null);
  const [searchText, setSearchText] = useState('');

  const keyword = searchText.trim() || undefined;

  const { data: commandsResponse, isLoading, isFetching } = useQuery({
    queryKey: ['mml-commands', { keyword: keyword ?? '' }],
    queryFn: () =>
      mmlApi.getCommands({
        page: 1,
        pageSize: COMMAND_PAGE_SIZE,
        keyword,
      }),
  });

  const { data: categoryDict } = useDictionary('mml_command_category');

  // 后端返回偶发会带重复（同一 id 出现两次），按 id 排重后再展平到分类树，
  // 避免同一命令在树中渲染两次（to-do-list #6）。
  const commands = useMemo(() => {
    const raw = commandsResponse?.items ?? [];
    const seen = new Set<string>();
    return raw.filter((c) => {
      if (seen.has(c.id)) return false;
      seen.add(c.id);
      return true;
    });
  }, [commandsResponse]);

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
          nodeType: 'command' as const,
          isLeaf: true,
        })),
      };
    });
  }, [categoryOptions, commands]);

  // Fetch templates for custom command directory
  const { data: templatesResponse } = useQuery({
    queryKey: ['mml', 'templates', 'all-for-tree'],
    queryFn: () => mmlApi.getTemplates({ page: 1, pageSize: 1000 }),
  });

  // 自定义命令同样按 id 排重（to-do-list #6）。
  const templates = useMemo(() => {
    const raw = templatesResponse?.items ?? [];
    const seen = new Set<string>();
    return raw.filter((t) => {
      if (seen.has(t.id)) return false;
      seen.add(t.id);
      return true;
    });
  }, [templatesResponse]);

  // Build the full tree including custom template directory
  const fullTreeData = useMemo((): CommandTreeNode[] => {
    const baseTree = treeData;

    const publicTemplates = templates.filter((t) => t.commandScope === 'public');
    const privateTemplates = templates.filter((t) => t.commandScope === 'private');

    // Group private templates by creator
    const privateByCreator = new Map<string, MMLCustomCommand[]>();
    for (const t of privateTemplates) {
      const list = privateByCreator.get(t.creator) ?? [];
      list.push(t);
      privateByCreator.set(t.creator, list);
    }

    // Helper: convert custom command to command-like tree node
    const templateToNode = (t: MMLCustomCommand): CommandTreeNode => ({
      key: `tmpl-${t.id}`,
      title: `${t.commandName} ${t.commandCode}`,
      label: t.commandName,
      commandCode: t.commandCode,
      template: t,
      nodeType: 'command',
      isLeaf: true,
    });

    const customRoot: CommandTreeNode = {
      key: 'custom-root',
      title: t('mml.console.customTemplates'),
      label: t('mml.console.customTemplates'),
      nodeType: 'custom_root',
      selectable: false,
      children: [
        {
          key: 'custom-public',
          title: t('mml.console.publicCommands'),
          label: t('mml.console.publicCommands'),
          nodeType: 'custom_public',
          selectable: false,
          count: publicTemplates.length,
          children: publicTemplates.map(templateToNode),
        },
        {
          key: 'custom-private',
          title: t('mml.console.privateCommands'),
          label: t('mml.console.privateCommands'),
          nodeType: 'custom_private',
          selectable: false,
          count: privateTemplates.length,
          children: [...privateByCreator.entries()].map(([creator, creatorTemplates]) => ({
            key: `custom-private-${creator}`,
            title: creator,
            label: creator,
            nodeType: 'user_dir' as const,
            selectable: false,
            count: creatorTemplates.length,
            children: creatorTemplates.map(templateToNode),
          })),
        },
      ],
    };

    return [...baseTree, customRoot];
  }, [treeData, templates]);

  useEffect(() => {
    if (selectedCommand && !commands.some((command) => command.id === selectedCommand.id)) {
      setSelectedCommand(null);
    }
  }, [commands, selectedCommand]);

  const selectCommand = useCallback(async (command: MMLCommand | null, _param?: MMLParamRef | null) => {
    if (!command) {
      setSelectedCommand(null);
      return;
    }

    // Template commands have synthetic IDs (tmpl-*) that are not valid UUIDs;
    // they already carry full parameter data from the tree node, so skip the API call.
    if (command.id.startsWith('tmpl-')) {
      setSelectedCommand(command);
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
    /** @deprecated Always null. Kept for backward compatibility. */
    selectedParam: null as MMLParamRef | null,
    searchText,
    commands,
    treeData: fullTreeData,
    categoryOptions,
    isLoading: isLoading || isFetching,
    setSearchText,
    selectCommand,
    clearSelection,
  };
}
