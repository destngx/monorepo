interface UtilizationRingProps {
  used: number;
  limit: number;
}

export function UtilizationRing({ used, limit }: UtilizationRingProps) {
  const percentage = (used / limit) * 100;
  return (
    <div className="flex flex-col items-center gap-2">
      <div
        className="relative w-32 h-32 rounded-full border-4 border-gray-700 flex items-center justify-center"
        style={{
          background: `conic-gradient(from 0deg, hsl(${Math.max(0, 120 - percentage * 1.2)}, 70%, 50%) 0deg ${percentage * 3.6}deg, transparent ${percentage * 3.6}deg)`,
        }}
      >
        <div className="text-center">
          <p className="text-2xl font-bold">{percentage.toFixed(0)}%</p>
          <p className="text-xs text-gray-400">Used</p>
        </div>
      </div>
    </div>
  );
}
