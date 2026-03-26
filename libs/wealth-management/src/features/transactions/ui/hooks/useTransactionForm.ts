import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { TransactionSchema, TransactionInput } from '@wealth-management/schemas';
import { Account } from '@wealth-management/types';

interface CategoryChip {
  name: string;
  type: 'income' | 'expense' | 'non-budget';
}

export function useTransactionForm(
  open: boolean,
  onOpenChange: (open: boolean) => void,
  onSubmit: (data: TransactionInput) => Promise<void>,
) {
  const [loading, setLoading] = useState(false);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [availableTags, setAvailableTags] = useState<string[]>([]);
  const [budgetCategories, setBudgetCategories] = useState<CategoryChip[]>([]);
  const [isSuggesting, setIsSuggesting] = useState(false);

  const form = useForm<TransactionInput>({
    resolver: zodResolver(TransactionSchema as any),
    defaultValues: {
      accountName: 'Golden Pocket',
      date: new Date().toISOString().split('T')[0] as any,
      payee: '',
      category: '',
      payment: null,
      deposit: null,
      cleared: false,
      tags: [],
    },
  });

  const { setValue, watch, reset, handleSubmit } = form;
  const payee = watch('payee');
  const currentCategory = watch('category');
  const selectedTags = watch('tags') ?? [];

  useEffect(() => {
    if (open) {
      fetch('/api/accounts')
        .then((r) => r.json())
        .then((data) => {
          setAccounts(data);
          if (data.some((a: Account) => a.name === 'Golden Pocket')) {
            setValue('accountName', 'Golden Pocket');
          }
        })
        .catch(console.error);

      fetch('/api/tags')
        .then((r) => r.json())
        .then((data) => (Array.isArray(data) ? setAvailableTags(data) : null))
        .catch(console.error);

      fetch('/api/categories')
        .then((r) => r.json())
        .then((data: CategoryChip[]) => {
          if (Array.isArray(data) && data.length > 0) {
            setBudgetCategories(data);
          }
        })
        .catch(console.error);
    }
  }, [open, setValue]);

  const handleSuggestCategory = async (targetPayee: string) => {
    if (!targetPayee || targetPayee.length < 3 || budgetCategories.length === 0) return;

    setIsSuggesting(true);
    try {
      const res = await fetch('/api/ai/suggest-category', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ payee: targetPayee, categories: budgetCategories.map((c) => c.name) }),
      });
      const data = await res.json();
      if (data.category) {
        setValue('category', data.category);
      }
    } catch (error) {
      console.error('AI Suggestion Error:', error);
    } finally {
      setIsSuggesting(false);
    }
  };

  useEffect(() => {
    if (!payee || payee.length < 3 || currentCategory) return;
    const timer = setTimeout(() => {
      handleSuggestCategory(payee);
    }, 1000);
    return () => clearTimeout(timer);
  }, [payee, budgetCategories, currentCategory]);

  const onFormSubmit = async (data: TransactionInput) => {
    setLoading(true);
    try {
      await onSubmit(data);
      reset();
      onOpenChange(false);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    reset();
    onOpenChange(false);
  };

  const addTag = (value: string) => {
    if (!value || selectedTags.includes(value)) return;
    setValue('tags', [...selectedTags, value]);
  };

  const removeTag = (tag: string) => {
    setValue(
      'tags',
      selectedTags.filter((t: string) => t !== tag),
    );
  };

  return {
    form,
    loading,
    accounts,
    availableTags,
    budgetCategories,
    isSuggesting,
    onFormSubmit,
    handleClose,
    handleSuggestCategory,
    addTag,
    removeTag,
  };
}
