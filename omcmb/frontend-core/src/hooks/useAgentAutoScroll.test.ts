import { describe, expect, it } from 'vitest';

import {
  captureAgentScrollAnchor,
  getAgentScrollAnchorAdjustment,
  isAgentViewportAtBottom,
} from './useAgentAutoScroll';

describe('isAgentViewportAtBottom', () => {
  it('treats an exact bottom position as pinned', () => {
    expect(
      isAgentViewportAtBottom({
        scrollTop: 600,
        scrollHeight: 1000,
        clientHeight: 400,
      })
    ).toBe(true);
  });

  it('keeps following within the default bottom threshold', () => {
    expect(
      isAgentViewportAtBottom({
        scrollTop: 528,
        scrollHeight: 1000,
        clientHeight: 400,
      })
    ).toBe(true);
  });

  it('stops following after the user scrolls beyond the bottom threshold', () => {
    expect(
      isAgentViewportAtBottom({
        scrollTop: 527,
        scrollHeight: 1000,
        clientHeight: 400,
      })
    ).toBe(false);
  });

  it('accepts a custom threshold', () => {
    expect(
      isAgentViewportAtBottom(
        {
          scrollTop: 550,
          scrollHeight: 1000,
          clientHeight: 400,
        },
        40
      )
    ).toBe(false);
  });
});

describe('agent scroll anchors', () => {
  it('captures the first visible message and its viewport offset', () => {
    const container = document.createElement('div');
    const content = document.createElement('div');
    const first = document.createElement('div');
    const second = document.createElement('div');
    content.append(first, second);
    container.append(content);

    container.getBoundingClientRect = () => ({
      top: 100,
      bottom: 500,
      left: 0,
      right: 400,
      width: 400,
      height: 400,
      x: 0,
      y: 100,
      toJSON: () => ({}),
    });
    first.getBoundingClientRect = () => ({
      top: 40,
      bottom: 90,
      left: 0,
      right: 400,
      width: 400,
      height: 50,
      x: 0,
      y: 40,
      toJSON: () => ({}),
    });
    second.getBoundingClientRect = () => ({
      top: 124,
      bottom: 220,
      left: 0,
      right: 400,
      width: 400,
      height: 96,
      x: 0,
      y: 124,
      toJSON: () => ({}),
    });

    const anchor = captureAgentScrollAnchor(container, content);

    expect(anchor?.element).toBe(second);
    expect(anchor?.viewportOffset).toBe(24);
  });

  it('calculates the scroll correction needed after reflow', () => {
    const container = document.createElement('div');
    const element = document.createElement('div');
    container.getBoundingClientRect = () => ({
      top: 100,
      bottom: 500,
      left: 0,
      right: 400,
      width: 400,
      height: 400,
      x: 0,
      y: 100,
      toJSON: () => ({}),
    });
    element.getBoundingClientRect = () => ({
      top: 64,
      bottom: 164,
      left: 0,
      right: 400,
      width: 400,
      height: 100,
      x: 0,
      y: 64,
      toJSON: () => ({}),
    });

    expect(
      getAgentScrollAnchorAdjustment(container, {
        element,
        viewportOffset: 24,
      })
    ).toBe(-60);
  });
});
