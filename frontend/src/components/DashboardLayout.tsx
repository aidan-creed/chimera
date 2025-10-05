import { useState } from "react";
import { Sidebar } from "@/components/Sidebar";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { PanelLeft, Sun, Moon } from "lucide-react";
import { UploadReportModal } from "./UploadReportModal";
import { Switch } from "@/components/ui/switch";
import { useTheme } from "../hooks/useTheme";
import { Link } from 'react-router-dom';
import { AuthenticationButton } from "./AuthenticationButton";

interface DashboardLayoutProps {
  children: React.ReactNode;
  onUploadSuccess: () => void;
}

export function DashboardLayout({ children, onUploadSuccess }: DashboardLayoutProps) {
  const [isUploadModalOpen, setIsUploadModalOpen] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const { theme, toggleTheme } = useTheme();

  const handleUploadReportClick = () => {
    setIsMobileMenuOpen(false);
    setIsUploadModalOpen(true);
  };

  return (
    <div className="h-screen flex flex-col bg-transparent">
      <header className="flex h-16 items-center justify-between border-b bg-transparent px-6 sticky top-0 z-10">
          {/* Mobile Navigation */}
          <Sheet open={isMobileMenuOpen} onOpenChange={setIsMobileMenuOpen}>
            <SheetTrigger asChild>
              <Button size="icon" variant="outline" className="md:hidden">
                <PanelLeft className="h-5 w-5" />
                <span className="sr-only">Toggle Menu</span>
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="p-0 w-64">
              <Sidebar onUploadReportClick={handleUploadReportClick} />
            </SheetContent>
          </Sheet>
          
          {/* Header Actions */}
          <div className="flex w-full items-center gap-4 justify-end">
            <Link to="/about" className="text-sm font-medium text-muted-foreground transition-colors hover:text-primary">
              About
            </Link>
            <div className="flex items-center space-x-2">
              {theme === "light" ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
              <Switch
                checked={theme === "dark"}
                onCheckedChange={toggleTheme}
              />
            </div>
            <AuthenticationButton />
          </div>
        </header>
        <div className="flex-1 flex flex-row min-h-0">
          <div className="hidden:block w-[280px] border-r bg-background/50 dark:bg-black/20 backdrop-blur-lg flex-shrink-0">
            <Sidebar onUploadReportClick={handleUploadReportClick} />
          </div>

          <main className="flex-1 overflow-y-auto p-6">
            <div className="p-6 h-full">
              {children}
            </div>
          </main>
        </div>

        <Sheet open={isUploadModalOpen} onOpenChange={setIsUploadModalOpen}>
              <SheetContent side="bottom" className="w-[280px]">
                <SheetHeader>
                  <SheetTitle>Upload Report</SheetTitle>
                  <SheetDescription>
                    Select a report type and upload your file.
                  </SheetDescription>
                </SheetHeader>
                <UploadReportModal onClose={() => setIsUploadModalOpen(false)} onUploadSuccess={onUploadSuccess} />
              </SheetContent>
            </Sheet>
      </div>
      
  );
}
