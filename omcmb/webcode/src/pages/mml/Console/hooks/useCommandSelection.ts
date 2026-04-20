import { useState, useCallback, useMemo } from 'react';
import type { TreeDataNode } from 'antd';
import { useQuery } from '@tanstack/react-query';
import type { MMLCommand, MMLTemplate } from '@/types/mml';
import { mmlApi } from '@/services/api/mmlApi';
import { useDictionary } from '@/hooks/api/useSystem';
import { useT } from '@/hooks/useT';
import { COMMAND_PAGE_SIZE } from '../constants';

export type CustomNodeType = 'custom_root' | 'custom_public' | 'custom_private' | 'user_dir';

export interface CommandTreeNode extends TreeDataNode {
  nodeType: 'category' | 'command' | CustomNodeType;
  label: string;
  category?: string;
  count?: number;
  commandCode?: string;
  command?: MMLCommand;
  template?: MMLTemplate;
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

  // Fetch templates for custom command directory
  const { data: templatesResponse } = useQuery({
    queryKey: ['mml', 'templates', 'all-for-tree'],
    queryFn: () => mmlApi.getTemplates({ page: 1, pageSize: 1000 }),
  });

  const templates = useMemo(() => templatesResponse?.items ?? [], [templatesResponse]);

  // Build the full tree including custom template directory
  const fullTreeData = useMemo((): CommandTreeNode[] => {
    const baseTree = treeData;

    const publicTemplates = templates.filter((t) => t.templateScope === 'public');
    const privateTemplates = templates.filter((t) => t.templateScope === 'private');

    // Group private templates by creator
    const privateByCreator = new Map<string, MMLTemplate[]>();
    for (const t of privateTemplates) {
      const list = privateByCreator.get(t.creator) ?? [];
      list.push(t);
      privateByCreator.set(t.creator, list);
    }

    // Helper: convert template to command-like tree node
    const templateToNode = (t: MMLTemplate): CommandTreeNode => ({
      key: `tmpl-${t.id}`,
      title: `${t.templateName} ${t.commandCode}`,
      label: t.templateName,
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
    commands,
    treeData: fullTreeData,
    categoryOptions,
    isLoading: isLoading || isFetching,
    setSearchText,
    selectCommand,
    clearSelection,
  };
}
