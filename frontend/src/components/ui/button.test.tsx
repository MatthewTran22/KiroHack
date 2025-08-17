import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

// Simple test component to verify setup
function TestButton({ children }: { children: React.ReactNode }) {
  return <button>{children}</button>;
}

describe('TestButton', () => {
  it('renders correctly', () => {
    render(<TestButton>Click me</TestButton>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });
});
