// react-resizable@3 自带 JS 无类型，这里补最小声明（仅用到 Resizable + onResize 回调）。
declare module 'react-resizable' {
  export interface ResizeCallbackData {
    node: HTMLElement;
    size: { width: number; height: number };
    handle: string;
  }
  export interface ResizableProps {
    width: number;
    height: number;
    onResize?: (e: import('react').SyntheticEvent, data: ResizeCallbackData) => void;
    onResizeStart?: (e: import('react').SyntheticEvent, data: ResizeCallbackData) => void;
    onResizeStop?: (e: import('react').SyntheticEvent, data: ResizeCallbackData) => void;
    handle?: import('react').ReactElement;
    draggableOpts?: Record<string, unknown>;
    minConstraints?: [number, number];
    maxConstraints?: [number, number];
    axis?: 'both' | 'x' | 'y' | 'none';
    children?: import('react').ReactNode;
  }
  export const Resizable: import('react').ComponentType<ResizableProps>;
}
