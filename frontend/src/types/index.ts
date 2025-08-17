// Common type definitions for the application

export interface User {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'user';
  createdAt: Date;
  updatedAt: Date;
}

export interface Document {
  id: string;
  name: string;
  type: string;
  size: number;
  uploadedAt: Date;
  userId: string;
}

export interface Consultation {
  id: string;
  title: string;
  type: 'policy' | 'research' | 'compliance';
  status: 'active' | 'completed' | 'draft';
  createdAt: Date;
  updatedAt: Date;
  userId: string;
}

export interface ApiResponse<T> {
  data: T;
  message?: string;
  success: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}
