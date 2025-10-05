import React from 'react';
import { useTheme } from '../hooks/useTheme';
import { Switch } from '../components/ui/switch';
import { Sun, Moon } from 'lucide-react';
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Link } from 'react-router-dom';

export const LandingPage: React.FC = () => {
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="flex flex-col items-center justify-center h-screen bg-background text-foreground">
      <div className="absolute top-4 right-4 flex items-center space-x-2">
        <Link to="/about" className="text-sm font-medium text-muted-foreground transition-colors hover:text-primary mr-2">
          About
        </Link>
        {theme === 'light' ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
        <Switch
          checked={theme === 'dark'}
          onCheckedChange={toggleTheme}
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="secondary"
              size="icon"
              className="rounded-full"
            >
              <Avatar>
                <AvatarImage src="" alt="" />
                <AvatarFallback>JJ</AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>My Account</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem>Settings</DropdownMenuItem>
            <DropdownMenuItem>Support</DropdownMenuItem>
            <DropdownMenuItem>Logout</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <h1 className="text-5xl font-bold">Welcome to Chimera</h1>
      <p className="mt-4 text-lg">An Analytics and Intelligence Platform for New Eden</p>
      <div className="mt-8">
        <a href="/dashboard" className="px-6 py-3 text-lg font-semibold text-white bg-blue-600 rounded-md hover:bg-blue-700">
          Go to Dashboard
        </a>
      </div>
    </div>
  );
};
