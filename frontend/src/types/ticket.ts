export type Priority =
  | 'PRIORITY_UNSPECIFIED'
  | 'PRIORITY_LOW'
  | 'PRIORITY_MEDIUM'
  | 'PRIORITY_HIGH'
  | 'PRIORITY_CRITICAL';

export type Status =
  | 'STATUS_UNSPECIFIED'
  | 'STATUS_OPEN'
  | 'STATUS_IN_PROGRESS'
  | 'STATUS_WAITING'
  | 'STATUS_RESOLVED'
  | 'STATUS_CLOSED';

export type Category =
  | 'CATEGORY_UNSPECIFIED'
  | 'CATEGORY_BUG'
  | 'CATEGORY_FEATURE_REQUEST'
  | 'CATEGORY_SUPPORT'
  | 'CATEGORY_DOCUMENTATION'
  | 'CATEGORY_INFRASTRUCTURE';

export interface Ticket {
  id: string;
  title: string;
  description: string;
  priority: Priority;
  status: Status;
  category: Category;
  createdBy: string;
  assignedTo: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
  resolvedAt?: string;
}

export interface Comment {
  id: string;
  ticketId: string;
  author: string;
  body: string;
  createdAt: string;
}
