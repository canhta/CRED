import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  Outlet,
} from '@tanstack/react-router';
import { Shell } from './app/Shell';
import { ClaimsPage } from './pages/ClaimsPage';
import { RecallPage } from './pages/RecallPage';
import { Placeholder } from './pages/Placeholder';

const rootRoute = createRootRoute({
  component: () => (
    <Shell>
      <Outlet />
    </Shell>
  ),
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/claims' });
  },
});

const claimsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/claims',
  component: ClaimsPage,
});

const recallRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/recall',
  component: RecallPage,
});

const usageRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/usage',
  component: () => <Placeholder title="Usage" />,
});

const teamRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/team',
  component: () => <Placeholder title="Team" />,
});

const projectsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/projects',
  component: () => <Placeholder title="Projects" />,
});

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  component: () => <Placeholder title="Settings" />,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  claimsRoute,
  recallRoute,
  usageRoute,
  teamRoute,
  projectsRoute,
  settingsRoute,
]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
