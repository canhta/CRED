import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  useNavigate,
  Outlet,
} from '@tanstack/react-router';
import { Shell } from './app/Shell';
import { ClaimsPage } from './pages/ClaimsPage';
import { ClaimDetailPage } from './pages/ClaimDetailPage';
import { RecallPage } from './pages/RecallPage';
import { UsagePage } from './pages/UsagePage';
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
  component: ClaimsRoute,
});

const claimDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/claims/$id',
  component: ClaimDetailRoute,
});

// Navigation is injected into the pages rather than reached for with a router
// hook inside them, so ClaimsPage and ClaimDetailPage stay pure components the
// tests can render without a router context. These wrappers are the only place
// that seam is closed.
function ClaimsRoute() {
  const navigate = useNavigate();
  return (
    <ClaimsPage
      onOpen={(id) => navigate({ to: '/claims/$id', params: { id } })}
    />
  );
}

function ClaimDetailRoute() {
  const { id } = claimDetailRoute.useParams();
  const navigate = useNavigate();
  return (
    <ClaimDetailPage id={id} onBack={() => navigate({ to: '/claims' })} />
  );
}

const recallRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/recall',
  component: RecallPage,
});

const usageRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/usage',
  component: UsagePage,
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
  claimDetailRoute,
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
