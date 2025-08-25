"use client";

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  Home,
  MessageSquare,
  FileText,
  History,
  Settings,
  Shield,
  BarChart3,
  Users,
  ChevronRight,
  X
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';
import { NavigationItem } from '@/types';

interface SidebarProps {
  className?: string;
}

export function Sidebar({ className }: SidebarProps) {
  const pathname = usePathname();
  const { user } = useAuthStore();
  const { sidebarOpen, setSidebarOpen } = useUIStore();

  const navigationItems: NavigationItem[] = [
    {
      id: 'dashboard',
      label: 'Dashboard',
      href: '/dashboard',
      icon: 'Home',
    },
    {
      id: 'consultation',
      label: 'New Consultation',
      href: '/consultation',
      icon: 'MessageSquare',
    },
    {
      id: 'documents',
      label: 'Documents',
      href: '/documents',
      icon: 'FileText',
      badge: '12', // This would come from actual data
    },
    {
      id: 'history',
      label: 'History',
      href: '/history',
      icon: 'History',
    },
    {
      id: 'analytics',
      label: 'Analytics',
      href: '/analytics',
      icon: 'BarChart3',
      requiredRole: 'admin',
    },
    {
      id: 'users',
      label: 'User Management',
      href: '/users',
      icon: 'Users',
      requiredRole: 'admin',
    },
    {
      id: 'audit',
      label: 'Audit Trail',
      href: '/audit',
      icon: 'Shield',
      requiredRole: 'admin',
    },
    {
      id: 'settings',
      label: 'Settings',
      href: '/settings',
      icon: 'Settings',
    },
  ];

  const getIcon = (iconName: string) => {
    const icons = {
      Home,
      MessageSquare,
      FileText,
      History,
      BarChart3,
      Users,
      Shield,
      Settings,
    };
    const IconComponent = icons[iconName as keyof typeof icons];
    return IconComponent ? <IconComponent className="h-4 w-4" /> : null;
  };

  const filteredNavigation = navigationItems.filter(item => {
    if (item.requiredRole && user?.role !== item.requiredRole) {
      return false;
    }
    return true;
  });

  const isActive = (href: string) => {
    if (href === '/dashboard') {
      return pathname === '/dashboard' || pathname === '/';
    }
    return pathname.startsWith(href);
  };

  return (
    <>
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed left-0 top-0 z-50 h-full w-64 sm:w-72 transform border-r bg-background transition-transform duration-200 ease-in-out lg:relative lg:translate-x-0",
          sidebarOpen ? "translate-x-0" : "-translate-x-full",
          className
        )}
        role="complementary"
        aria-label="Navigation sidebar"
      >
        <div className="flex h-full flex-col">
          {/* Header */}
          <div className="flex h-14 sm:h-16 items-center justify-between border-b px-4">
            <div className="flex items-center gap-2 min-w-0">
              <div className="h-7 w-7 sm:h-8 sm:w-8 rounded bg-primary flex items-center justify-center shrink-0">
                <span className="text-primary-foreground font-bold text-xs sm:text-sm">AI</span>
              </div>
              <span className="font-semibold text-sm sm:text-base truncate">Gov Consultant</span>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSidebarOpen(false)}
              className="lg:hidden shrink-0"
              aria-label="Close sidebar"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 p-3 sm:p-4 overflow-y-auto" role="navigation" aria-label="Main navigation">
            {filteredNavigation.map((item) => (
              <Link
                key={item.id}
                href={item.href}
                onClick={() => {
                  // Close sidebar on mobile after navigation
                  if (window.innerWidth < 1024) {
                    setSidebarOpen(false);
                  }
                }}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  isActive(item.href)
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground"
                )}
                aria-current={isActive(item.href) ? 'page' : undefined}
              >
                {getIcon(item.icon || '')}
                <span className="flex-1 truncate">{item.label}</span>
                {item.badge && (
                  <Badge variant="secondary" className="ml-auto shrink-0 text-xs">
                    {item.badge}
                  </Badge>
                )}
                {item.children && (
                  <ChevronRight className="h-4 w-4 shrink-0" />
                )}
              </Link>
            ))}
          </nav>

          {/* Footer */}
          <div className="border-t p-3 sm:p-4">
            <div className="flex items-center gap-3 rounded-lg bg-muted/50 p-3">
              <div className="h-7 w-7 sm:h-8 sm:w-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                <span className="text-xs font-medium text-primary">
                  {user?.name?.charAt(0).toUpperCase()}
                </span>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{user?.name}</p>
                <p className="text-xs text-muted-foreground capitalize truncate">
                  {user?.role}{user?.department && ` • ${user.department}`}
                </p>
              </div>
            </div>
          </div>
        </div>
      </aside>
    </>
  );
}