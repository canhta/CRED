import type { ReactNode } from 'react';
import { AppShell } from '@astryxdesign/core/AppShell';
import {
  SideNav,
  SideNavHeading,
  SideNavItem,
  SideNavSection,
} from '@astryxdesign/core/SideNav';
import { Button } from '@astryxdesign/core/Button';
import { useNavigate, useRouterState } from '@tanstack/react-router';
import { useLogout } from '../api';

type NavPath =
  | '/claims'
  | '/recall'
  | '/usage'
  | '/team'
  | '/projects'
  | '/settings';

interface NavEntry {
  label: string;
  to: NavPath;
}

const sections: { title: string; items: NavEntry[] }[] = [
  {
    title: 'Memory',
    items: [
      { label: 'Claims', to: '/claims' },
      { label: 'Recall', to: '/recall' },
      { label: 'Usage', to: '/usage' },
    ],
  },
  {
    title: 'Workspace',
    items: [
      { label: 'Team', to: '/team' },
      { label: 'Projects', to: '/projects' },
      { label: 'Settings', to: '/settings' },
    ],
  },
];

export function Shell({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const logout = useLogout();

  return (
    <AppShell
      contentPadding={0}
      sideNav={
        <SideNav
          collapsible
          header={
            <SideNavHeading
              heading="CRED"
              subheading="Console"
              headingHref="/claims"
            />
          }
          footer={
            <Button
              label="Sign out"
              variant="ghost"
              isLoading={logout.isPending}
              onClick={() => {
                logout.mutate(undefined, {
                  onSuccess: () => navigate({ to: '/login' }),
                });
              }}
            />
          }
        >
          {sections.map((section) => (
            <SideNavSection key={section.title} title={section.title}>
              {section.items.map((item) => (
                <SideNavItem
                  key={item.to}
                  label={item.label}
                  href={item.to}
                  isSelected={pathname.startsWith(item.to)}
                  onClick={(e) => {
                    e.preventDefault();
                    void navigate({ to: item.to });
                  }}
                />
              ))}
            </SideNavSection>
          ))}
        </SideNav>
      }
    >
      {children}
    </AppShell>
  );
}
