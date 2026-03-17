"use client";

import { PieChart, Pie, Cell, ResponsiveContainer } from "recharts";

interface UtilizationRingProps {
  used: number;
  limit: number;
}

export function UtilizationRing({ used, limit }: UtilizationRingProps) {
  const percentage = Math.min(100, (used / limit) * 100);
  const remaining = Math.max(0, 100 - percentage);
  
  const data = [
    { name: "Used", value: percentage },
    { name: "Remaining", value: remaining },
  ];

  // Color coding: green <30%, amber 30–70%, red >70%
  const getColor = (p: number) => {
    if (p < 30) return "#10b981"; // Emerald-500
    if (p < 70) return "#f59e0b"; // Amber-500
    return "#f43f5e"; // Rose-500
  };

  const activeColor = getColor(percentage);

  return (
    <div className="relative w-full aspect-square max-w-[240px] mx-auto">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius="75%"
            outerRadius="95%"
            startAngle={90}
            endAngle={450}
            paddingAngle={0}
            dataKey="value"
            stroke="none"
          >
            <Cell key="cell-used" fill={activeColor} />
            <Cell key="cell-remaining" fill="currentColor" className="text-muted/10" />
          </Pie>
        </PieChart>
      </ResponsiveContainer>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-4xl font-black tracking-tighter" style={{ color: activeColor }}>
          {Math.round(percentage)}%
        </span>
        <span className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground mt-1">
          Utilization
        </span>
      </div>
    </div>
  );
}
