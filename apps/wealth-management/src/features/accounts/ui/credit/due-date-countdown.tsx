interface DueDateCountdownProps {
  balance: number;
  dueDay: number;
}

export function DueDateCountdown({ balance, dueDay }: DueDateCountdownProps) {
  const today = new Date().getDate();
  const daysUntilDue = dueDay > today ? dueDay - today : 30 - today + dueDay;

  return (
    <div className="flex flex-col gap-2 p-4 rounded-lg bg-blue-950/20 border border-blue-900/30">
      <p className="text-sm text-blue-300">Due in {daysUntilDue} days</p>
      <p className="text-2xl font-bold">Day {dueDay}</p>
      <p className="text-xs text-gray-400">Balance: ${balance.toFixed(2)}</p>
    </div>
  );
}
