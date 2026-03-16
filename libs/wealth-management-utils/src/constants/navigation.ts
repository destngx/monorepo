import {
  Home,
  CreditCard,
  PieChart,
  Landmark,
  MessageSquare,
  Settings,
  TrendingUp,
  Target,
  Zap,
  Flag,
  HandCoins
} from "lucide-react";

export const NAV_LINKS = [
  { href: "/", label: "Dashboard", icon: Home },
  { href: "/transactions", label: "Transactions", icon: Zap },
  { href: "/budget", label: "Budget", icon: PieChart },
  { href: "/accounts", label: "Accounts", icon: Landmark },
  { href: "/investments", label: "Investments", icon: TrendingUp },
  { href: "/chat", label: "AI Chat", icon: MessageSquare },
  { href: "/settings", label: "Settings", icon: Settings },
];
