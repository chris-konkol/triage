export interface CategoryCount {
  category: string;
  count: number;
}

export interface PriorityCount {
  priority: string;
  count: number;
}

export interface DailyCount {
  date: string;
  created: number;
  resolved: number;
}

export interface DashboardStats {
  totalOpen: number;
  totalInProgress: number;
  totalResolved: number;
  totalClosed: number;
  avgResolutionHours: number;
  ticketsByCategory: CategoryCount[];
  ticketsByPriority: PriorityCount[];
  ticketsPerDay: DailyCount[];
}
