/**
 * Goals Feature - React Hooks
 */

'use client';

import { useEffect, useState } from 'react';
import { Goal, GoalProjection } from './types';
import { AppError, isAppError, getErrorMessage } from '../../../utils/errors';
import * as queries from './queries';
import * as mutations from './mutations';

export function useGoals() {
  const [goals, setGoals] = useState<Goal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<AppError | null>(null);

  useEffect(() => {
    queries
      .getGoals()
      .then(setGoals)
      .catch((err: unknown) => {
        const appError = isAppError(err) ? err : new AppError(getErrorMessage(err));
        setError(appError);
      })
      .finally(() => setLoading(false));
  }, []);

  return { goals, loading, error };
}

export function useGoal(id: string) {
  const [goal, setGoal] = useState<Goal | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<AppError | null>(null);

  useEffect(() => {
    queries
      .getGoalById(id)
      .then(setGoal)
      .catch((err: unknown) => {
        const appError = isAppError(err) ? err : new AppError(getErrorMessage(err));
        setError(appError);
      })
      .finally(() => setLoading(false));
  }, [id]);

  return { goal, loading, error };
}

export function useGoalProjection(id: string) {
  const [projection, setProjection] = useState<GoalProjection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<AppError | null>(null);

  useEffect(() => {
    queries
      .getGoalProjection(id)
      .then(setProjection)
      .catch((err: unknown) => {
        const appError = isAppError(err) ? err : new AppError(getErrorMessage(err));
        setError(appError);
      })
      .finally(() => setLoading(false));
  }, [id]);

  return { projection, loading, error };
}
