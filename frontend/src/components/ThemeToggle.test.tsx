import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ThemeToggle } from './ThemeToggle';

// Mock the useTheme hook
vi.mock('../contexts/ThemeContext', () => ({
  useTheme: vi.fn(),
}));

import { useTheme } from '../contexts/ThemeContext';

const mockToggleTheme = vi.fn();
const mockUseTheme = vi.mocked(useTheme);

beforeEach(() => {
  vi.clearAllMocks();
  mockUseTheme.mockReturnValue({
    theme: 'light',
    toggleTheme: mockToggleTheme,
    setTheme: vi.fn(),
  });
});

describe('ThemeToggle', () => {
  it('renders without crashing', () => {
    render(<ThemeToggle />);
    expect(screen.getByText('主题:')).toBeInTheDocument();
  });

  it('displays current theme', () => {
    mockUseTheme.mockReturnValue({
      theme: 'light',
      toggleTheme: mockToggleTheme,
      setTheme: vi.fn(),
    });
    render(<ThemeToggle />);
    expect(screen.getByText('Light')).toHaveClass('btn-primary');
  });

  it('calls toggleTheme when button is clicked', () => {
    render(<ThemeToggle />);
    const lightButton = screen.getByText('Light');
    fireEvent.click(lightButton);
    expect(mockToggleTheme).toHaveBeenCalledTimes(1);
  });

  it('has two theme buttons', () => {
    render(<ThemeToggle />);
    expect(screen.getByText('Light')).toBeInTheDocument();
    expect(screen.getByText('Silk')).toBeInTheDocument();
  });
});