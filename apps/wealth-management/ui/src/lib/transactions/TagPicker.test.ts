import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import TagPicker from './TagPicker.svelte';

async function enter(text: string) {
  const input = screen.getByRole('textbox', { name: 'Add tag' });
  await fireEvent.input(input, { target: { value: text } });
  await fireEvent.keyDown(input, { key: 'Enter' });
}

describe('TagPicker', () => {
  it('Given a tag that contains the typed text When Enter is pressed Then the typed tag is added, not the longer one', async () => {
    render(TagPicker, { options: ['fastfood'], selected: [] });
    await enter('food');
    expect(screen.getByRole('button', { name: 'Remove food' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Remove fastfood' })).toBeNull();
  });

  it('Given an existing tag typed in another case When Enter is pressed Then the saved spelling is added', async () => {
    render(TagPicker, { options: ['Eating_Out'], selected: [] });
    await enter('eating_out');
    expect(screen.getByRole('button', { name: 'Remove Eating_Out' })).toBeTruthy();
  });
});
