import { renderHook, act } from '@testing-library/react';
import { useUIStore } from '../ui';

// Mock crypto.randomUUID
Object.defineProperty(global, 'crypto', {
  value: {
    randomUUID: jest.fn(() => 'mock-uuid'),
  },
});

describe('useUIStore', () => {
  beforeEach(() => {
    // Clear the store before each test
    useUIStore.setState({
      sidebarOpen: true,
      notifications: [],
    });
  });

  it('initializes with default state', () => {
    const { result } = renderHook(() => useUIStore());
    
    expect(result.current.sidebarOpen).toBe(true);
    expect(result.current.notifications).toEqual([]);
  });

  it('sets sidebar open state', () => {
    const { result } = renderHook(() => useUIStore());
    
    act(() => {
      result.current.setSidebarOpen(false);
    });
    
    expect(result.current.sidebarOpen).toBe(false);
  });

  it('toggles sidebar state', () => {
    const { result } = renderHook(() => useUIStore());
    
    // Initial state is true
    expect(result.current.sidebarOpen).toBe(true);
    
    act(() => {
      result.current.toggleSidebar();
    });
    
    expect(result.current.sidebarOpen).toBe(false);
    
    act(() => {
      result.current.toggleSidebar();
    });
    
    expect(result.current.sidebarOpen).toBe(true);
  });

  it('adds notification', () => {
    const { result } = renderHook(() => useUIStore());
    
    const notification = {
      type: 'info' as const,
      title: 'Test Notification',
      message: 'Test message',
      read: false,
    };
    
    act(() => {
      result.current.addNotification(notification);
    });
    
    expect(result.current.notifications).toHaveLength(1);
    expect(result.current.notifications[0]).toMatchObject({
      ...notification,
      id: 'mock-uuid',
      timestamp: expect.any(Date),
    });
  });

  it('removes notification', () => {
    const { result } = renderHook(() => useUIStore());
    
    // Add a notification first
    act(() => {
      result.current.addNotification({
        type: 'info',
        title: 'Test',
        message: 'Test',
        read: false,
      });
    });
    
    const notificationId = result.current.notifications[0].id;
    
    act(() => {
      result.current.removeNotification(notificationId);
    });
    
    expect(result.current.notifications).toHaveLength(0);
  });

  it('marks notification as read', () => {
    const { result } = renderHook(() => useUIStore());
    
    // Add a notification first
    act(() => {
      result.current.addNotification({
        type: 'info',
        title: 'Test',
        message: 'Test',
        read: false,
      });
    });
    
    const notificationId = result.current.notifications[0].id;
    expect(result.current.notifications[0].read).toBe(false);
    
    act(() => {
      result.current.markNotificationRead(notificationId);
    });
    
    expect(result.current.notifications[0].read).toBe(true);
  });

  it('clears all notifications', () => {
    const { result } = renderHook(() => useUIStore());
    
    // Add multiple notifications
    act(() => {
      result.current.addNotification({
        type: 'info',
        title: 'Test 1',
        message: 'Test 1',
        read: false,
      });
      result.current.addNotification({
        type: 'warning',
        title: 'Test 2',
        message: 'Test 2',
        read: false,
      });
    });
    
    expect(result.current.notifications).toHaveLength(2);
    
    act(() => {
      result.current.clearAllNotifications();
    });
    
    expect(result.current.notifications).toHaveLength(0);
  });

  it('limits notifications to 50', () => {
    const { result } = renderHook(() => useUIStore());
    
    // Add 52 notifications
    act(() => {
      for (let i = 0; i < 52; i++) {
        result.current.addNotification({
          type: 'info',
          title: `Test ${i}`,
          message: `Test message ${i}`,
          read: false,
        });
      }
    });
    
    expect(result.current.notifications).toHaveLength(50);
    // Should keep the latest notifications
    expect(result.current.notifications[0].title).toBe('Test 51');
    expect(result.current.notifications[49].title).toBe('Test 2');
  });

  it('adds notifications in correct order (newest first)', () => {
    const { result } = renderHook(() => useUIStore());
    
    act(() => {
      result.current.addNotification({
        type: 'info',
        title: 'First',
        message: 'First message',
        read: false,
      });
    });
    
    act(() => {
      result.current.addNotification({
        type: 'info',
        title: 'Second',
        message: 'Second message',
        read: false,
      });
    });
    
    expect(result.current.notifications[0].title).toBe('Second');
    expect(result.current.notifications[1].title).toBe('First');
  });

  it('handles notification with actions', () => {
    const { result } = renderHook(() => useUIStore());
    
    const mockAction = jest.fn();
    const notification = {
      type: 'info' as const,
      title: 'Test Notification',
      message: 'Test message',
      read: false,
      actions: [
        {
          label: 'Action',
          action: mockAction,
        },
      ],
    };
    
    act(() => {
      result.current.addNotification(notification);
    });
    
    expect(result.current.notifications[0].actions).toHaveLength(1);
    expect(result.current.notifications[0].actions![0].label).toBe('Action');
    
    // Test action execution
    result.current.notifications[0].actions![0].action();
    expect(mockAction).toHaveBeenCalled();
  });

  it('handles different notification types', () => {
    const { result } = renderHook(() => useUIStore());
    
    const types = ['info', 'success', 'warning', 'error'] as const;
    
    act(() => {
      types.forEach((type) => {
        result.current.addNotification({
          type,
          title: `${type} notification`,
          message: `${type} message`,
          read: false,
        });
      });
    });
    
    expect(result.current.notifications).toHaveLength(4);
    types.forEach((type, index) => {
      expect(result.current.notifications[3 - index].type).toBe(type);
    });
  });
});