import type { ComponentType, ReactNode, SVGProps } from 'react';
import { AppShell } from '@astryxdesign/core/AppShell';
import {
  SideNav,
  SideNavHeading,
  SideNavItem,
  SideNavSection,
} from '@astryxdesign/core/SideNav';
import { Divider } from '@astryxdesign/core/Divider';
import { VStack } from '@astryxdesign/core/Stack';
import { useNavigate, useRouterState } from '@tanstack/react-router';
import {
  Brain,
  Folder,
  Gauge,
  LogOut,
  Settings,
  ShieldCheck,
  Users,
} from 'lucide-react';
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
  icon: ComponentType<SVGProps<SVGSVGElement>>;
}

const sections: { title: string; items: NavEntry[] }[] = [
  {
    title: 'Memory',
    items: [
      { label: 'Claims', to: '/claims', icon: ShieldCheck },
      { label: 'Recall', to: '/recall', icon: Brain },
      { label: 'Usage', to: '/usage', icon: Gauge },
    ],
  },
  {
    title: 'Workspace',
    items: [
      { label: 'Team', to: '/team', icon: Users },
      { label: 'Projects', to: '/projects', icon: Folder },
      { label: 'Settings', to: '/settings', icon: Settings },
    ],
  },
];

export function Shell({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const logout = useLogout();

  return (
    <AppShell
      contentPadding={6}
      variant="section"
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
            <VStack gap={2}>
              <SideNavItem
                label="Sign out"
                icon={LogOut}
                isDisabled={logout.isPending}
                onClick={() => {
                  logout.mutate(undefined, {
                    onSuccess: () => navigate({ to: '/login' }),
                  });
                }}
              />
              <Divider />
            </VStack>
          }
        >
          {sections.map((section) => (
            <SideNavSection key={section.title} title={section.title}>
              {section.items.map((item) => (
                <SideNavItem
                  key={item.to}
                  label={item.label}
                  icon={item.icon}
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
