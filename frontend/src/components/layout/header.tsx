"use client";

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useTheme } from 'next-themes';
import { 
  Search, 
  Bell, 
  Menu, 
  Sun, 
  Moon, 
  User, 
  Settings, 
  LogOut,
  Plus,
  Upload
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';
import { SearchResult } from '@/types';

interface HeaderProps {
  onSearch?: (query: string) => void;
  onNewConsultation?: () => void;
  searchResults?: SearchResult[];
  isSearching?: boolean;
}

export function Header({ 
  onSearch, 
  onNewConsultation, 
  searchResults = []
}: HeaderProps) {
  const router = useRouter();
  const { theme, setTheme } = useTheme();
  const { user, logout } = useAuthStore();
  const { toggleSidebar, notifications } = useUIStore();
  const [searchQuery, setSearchQuery] = useState('');
  const [showSearchResults, setShowSearchResults] = useState(false);

  const unreadNotifications = notifications.filter(n => !n.read).length;

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchQuery.trim() && onSearch) {
      onSearch(searchQuery.trim());
      setShowSearchResults(true);
    }
  };

  const handleSearchInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setSearchQuery(value);
    if (value.trim() && onSearch) {
      onSearch(value.trim());
      setShowSearchResults(true);
    } else {
      setShowSearchResults(false);
    }
  };

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  const getUserInitials = (name: string) => {
    return name
      .split(' ')
      .map(n => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  return (
    <header className="sticky top-0 z-40 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-14 sm:h-16 items-center justify-between px-3 sm:px-4">
        {/* Left section - Menu toggle and Logo */}
        <div className="flex items-center gap-2 sm:gap-4 min-w-0">
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleSidebar}
            className="lg:hidden shrink-0"
            aria-label="Toggle sidebar"
          >
            <Menu className="h-5 w-5" />
          </Button>
          
          <div className="hidden sm:flex items-center gap-2 min-w-0">
            <div className="h-7 w-7 sm:h-8 sm:w-8 rounded bg-primary flex items-center justify-center shrink-0">
              <span className="text-primary-foreground font-bold text-xs sm:text-sm">AI</span>
            </div>
            <span className="font-semibold text-base sm:text-lg truncate">Gov Consultant</span>
          </div>
          
          {/* Mobile logo */}
          <div className="flex sm:hidden items-center gap-2">
            <div className="h-7 w-7 rounded bg-primary flex items-center justify-center">
              <span className="text-primary-foreground font-bold text-xs">AI</span>
            </div>
          </div>
        </div>

        {/* Center section - Search */}
        <div className="flex-1 max-w-xs sm:max-w-md mx-2 sm:mx-4 relative min-w-0">
          <form onSubmit={handleSearch} className="relative">
            <Search className="absolute left-2 sm:left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              type="search"
              placeholder="Search..."
              value={searchQuery}
              onChange={handleSearchInputChange}
              className="pl-8 sm:pl-10 pr-3 sm:pr-4 h-8 sm:h-9 text-sm placeholder:text-xs sm:placeholder:text-sm"
              aria-label="Search documents and consultations"
            />
          </form>
          
          {/* Search Results Dropdown */}
          {showSearchResults && searchResults.length > 0 && (
            <div className="absolute top-full left-0 right-0 mt-1 bg-popover border rounded-md shadow-lg z-50 max-h-96 overflow-y-auto">
              {searchResults.map((result) => (
                <div
                  key={result.id}
                  className="p-3 hover:bg-accent cursor-pointer border-b last:border-b-0"
                  onClick={() => {
                    router.push(result.url);
                    setShowSearchResults(false);
                    setSearchQuery('');
                  }}
                >
                  <div className="flex items-center justify-between">
                    <h4 className="font-medium text-sm">{result.title}</h4>
                    <Badge variant="outline" className="text-xs">
                      {result.type}
                    </Badge>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                    {result.excerpt}
                  </p>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Right section - Actions and User menu */}
        <div className="flex items-center gap-1 sm:gap-2 shrink-0">
          {/* Quick Actions */}
          <Button
            variant="ghost"
            size="sm"
            onClick={onNewConsultation}
            className="hidden md:flex h-8 px-2 sm:px-3"
            aria-label="New consultation"
          >
            <Plus className="h-4 w-4 sm:mr-2" />
            <span className="hidden lg:inline">New Chat</span>
          </Button>
          
          <Button
            variant="ghost"
            size="sm"
            onClick={() => router.push('/documents/upload')}
            className="hidden md:flex h-8 px-2"
            aria-label="Upload document"
          >
            <Upload className="h-4 w-4" />
          </Button>

          {/* Mobile Quick Action */}
          <Button
            variant="ghost"
            size="sm"
            onClick={onNewConsultation}
            className="md:hidden h-8 px-2"
            aria-label="New consultation"
          >
            <Plus className="h-4 w-4" />
          </Button>

          {/* Theme Toggle */}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            className="h-8 px-2 relative"
            aria-label="Toggle theme"
          >
            <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
            <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          </Button>

          {/* Notifications */}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => router.push('/notifications')}
            className="relative h-8 px-2"
            aria-label={`Notifications ${unreadNotifications > 0 ? `(${unreadNotifications} unread)` : ''}`}
          >
            <Bell className="h-4 w-4" />
            {unreadNotifications > 0 && (
              <Badge 
                variant="destructive" 
                className="absolute -top-1 -right-1 h-4 w-4 sm:h-5 sm:w-5 flex items-center justify-center p-0 text-xs min-w-0"
              >
                {unreadNotifications > 99 ? '99+' : unreadNotifications}
              </Badge>
            )}
          </Button>

          {/* User Menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="relative h-8 w-8 rounded-full ml-1">
                <Avatar className="h-7 w-7 sm:h-8 sm:w-8">
                  <AvatarImage src={`/avatars/${user?.id}.jpg`} alt={user?.name} />
                  <AvatarFallback className="text-xs sm:text-sm">
                    {user?.name ? getUserInitials(user.name) : <User className="h-3 w-3 sm:h-4 sm:w-4" />}
                  </AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-56" align="end" forceMount>
              <DropdownMenuLabel className="font-normal">
                <div className="flex flex-col space-y-1">
                  <p className="text-sm font-medium leading-none truncate">{user?.name}</p>
                  <p className="text-xs leading-none text-muted-foreground truncate">
                    {user?.email}
                  </p>
                  <p className="text-xs leading-none text-muted-foreground capitalize">
                    {user?.role}{user?.department && ` • ${user.department}`}
                  </p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => router.push('/profile')}>
                <User className="mr-2 h-4 w-4" />
                <span>Profile</span>
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => router.push('/settings')}>
                <Settings className="mr-2 h-4 w-4" />
                <span>Settings</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleLogout}>
                <LogOut className="mr-2 h-4 w-4" />
                <span>Log out</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  );
}