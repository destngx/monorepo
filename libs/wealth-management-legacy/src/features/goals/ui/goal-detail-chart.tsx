"use client";

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine, AreaChart, Area } from "recharts";
import { Contribution } from "../model/types";

interface GoalChartProps {
  data: { date: string; actual: number; projected?: number; target: number }[];
}

export function GoalDetailChart({ data }: GoalChartProps) {
  return (
    <div className="h-[300px] w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 10, right: 10, left: 10, bottom: 0 }}>
          <defs>
            <linearGradient id="colorActual" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#6366f1" stopOpacity={0.1}/>
              <stop offset="95%" stopColor="#6366f1" stopOpacity={0}/>
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#E2E8F0" opacity={0.5} />
          <XAxis 
            dataKey="date" 
            axisLine={false} 
            tickLine={false} 
            tick={{ fontSize: 10, fill: "#94A3B8" }} 
            dy={10}
          />
          <YAxis hide />
          <Tooltip 
            contentStyle={{ borderRadius: '12px', border: 'none', boxShadow: '0 10px 15px -3px rgb(0 0 0 / 0.1)' }}
          />
          <ReferenceLine y={data[0]?.target} stroke="#94A3B8" strokeDasharray="3 3" label={{ position: 'top', value: 'Target', fill: '#94A3B8', fontSize: 10 }} />
          <Area 
            type="monotone" 
            dataKey="actual" 
            stroke="#6366f1" 
            strokeWidth={3}
            fillOpacity={1} 
            fill="url(#colorActual)" 
            animationDuration={1500}
          />
          {data.some(d => d.projected) && (
            <Line 
              type="monotone" 
              dataKey="projected" 
              stroke="#6366f1" 
              strokeWidth={2} 
              strokeDasharray="5 5" 
              dot={false}
              animationDuration={1500}
            />
          )}
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
