import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ setCategoryDefinition: vi.fn() }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn(), success: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn() }));

import CategoryDefinitionModal from './CategoryDefinitionModal.svelte';

describe('CategoryDefinitionModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // jsdom has no modal dialogs; the Modal only needs them to toggle.
    if (!('showModal' in HTMLDialogElement.prototype)) {
      Object.assign(HTMLDialogElement.prototype, {
        showModal(this: HTMLDialogElement) {
          this.open = true;
        },
        close(this: HTMLDialogElement) {
          this.open = false;
        },
      });
    }
  });

  it('shows the saved definition and writes the edits with tags as the engine stores them (WM-CAT-1, WM-CAT-2)', async () => {
    endpoints.setCategoryDefinition.mockResolvedValue({ name: 'Groceries', type: 'expense' });
    const onclose = vi.fn();
    render(CategoryDefinitionModal, {
      category: { name: 'Groceries', type: 'expense', definition: 'Food for home.', splitTags: ['eating-out'] },
      onclose,
    });

    const definition = screen.getByLabelText<HTMLTextAreaElement>('What belongs here');
    const tags = screen.getByLabelText<HTMLInputElement>('Split tags');
    expect(definition.value).toBe('Food for home.');
    expect(tags.value).toBe('eating-out');

    await fireEvent.input(definition, { target: { value: ' Food for home, and everyday meals out. ' } });
    await fireEvent.input(tags, { target: { value: 'Eating-Out, takeaway,' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(endpoints.setCategoryDefinition).toHaveBeenCalledWith({
      category: 'Groceries',
      definition: 'Food for home, and everyday meals out.',
      splitTags: ['eating-out', 'takeaway'],
    });
    await vi.waitFor(() => expect(onclose).toHaveBeenCalled());
  });
});

describe('CategoryDefinitionModal: Describe it (WM-FILL-6)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('fills the definition and split tags, marks them, and saves them as usual', async () => {
    endpoints.fill = vi.fn().mockResolvedValue({
      form: 'category-definition',
      values: { definition: 'Groceries and meals out, not alcohol.', splitTags: ['work', 'family'] },
      filled: ['definition', 'splitTags'],
      notes: [],
    });
    endpoints.setCategoryDefinition.mockResolvedValue({ name: 'Food', type: 'expense' });
    render(CategoryDefinitionModal, { category: { name: 'Food', type: 'expense' }, onclose: vi.fn() });

    await fireEvent.input(screen.getByRole('textbox', { name: 'Describe it' }), {
      target: { value: 'groceries and eating out, not alcohol; split by work and family' },
    });
    await fireEvent.click(screen.getByRole('button', { name: 'Fill' }));

    const definition = await screen.findByDisplayValue('Groceries and meals out, not alcohol.');
    expect(screen.getByLabelText<HTMLInputElement>('Split tags', { exact: false }).value).toBe('work, family');
    expect(screen.getAllByTitle('Filled by AI. Check it before saving.')).toHaveLength(2);
    expect(endpoints.fill).toHaveBeenCalledWith(
      expect.objectContaining({
        form: 'category-definition',
        current: { category: 'Food', definition: '', splitTags: [] },
      }),
      expect.any(AbortSignal),
    );

    await fireEvent.input(definition, { target: { value: 'Groceries only.' } });
    expect(screen.getAllByTitle('Filled by AI. Check it before saving.')).toHaveLength(1);

    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(endpoints.setCategoryDefinition).toHaveBeenCalledWith({
      category: 'Food',
      definition: 'Groceries only.',
      splitTags: ['work', 'family'],
    });
  });
});
