import { describe, expect, it } from 'vitest';
import { readNDJSON } from './ndjson';

function streamOf(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  return new ReadableStream({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(encoder.encode(chunk));
      controller.close();
    },
  });
}

describe('readNDJSON', () => {
  it('parses lines split across chunks, including a final line without newline', async () => {
    const values: unknown[] = [];
    for await (const value of readNDJSON(streamOf(['{"a":', '1}\n\n{"b"', ':"ẞ"}\n{"c":3}']))) values.push(value);
    expect(values).toEqual([{ a: 1 }, { b: 'ẞ' }, { c: 3 }]);
  });
});
