import React from 'react';

export const AboutPage: React.FC = () => {
  return (
    <div className="flex flex-col items-center justify-center h-full bg-background text-foreground">
      <h1 className="text-5xl font-bold">About Chimera</h1>
      <p className="mt-4 text-lg text-center max-w-2xl">
        Chimera is a is an intelligent and powerful platform that provides tools for deep insight and the development of actionable insights.
      </p>
    </div>
  );
};
