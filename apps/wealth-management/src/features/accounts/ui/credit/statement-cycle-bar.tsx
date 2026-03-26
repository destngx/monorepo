interface StatementCycleBarProps {
  reportDay: number;
}

export function StatementCycleBar({ reportDay }: StatementCycleBarProps) {
  const today = new Date().getDate();
  const progress = (today / 30) * 100;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex justify-between text-sm">
        <span className="text-gray-400">Statement Cycle</span>
        <span className="text-gray-300">{today}/30</span>
      </div>
      <div className="w-full bg-gray-800 rounded-full h-2">
        <div
          className="bg-gradient-to-r from-purple-500 to-pink-500 h-2 rounded-full transition-all"
          style={{ width: `${progress}%` }}
        />
      </div>
      <p className="text-xs text-gray-500">Report day: {reportDay}</p>
    </div>
  );
}
