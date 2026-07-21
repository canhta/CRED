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
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { Placeholder } from './pages/Placeholder';
import { getHealth } from './api/client';

const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginRoute,
});

const registerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/register',
  component: RegisterRoute,
});

function LoginRoute() {
  const navigate = useNavigate();
  return (
    <LoginPage
      onSuccess={() => navigate({ to: '/claims' })}
      onNavigateToRegister={() => navigate({ to: '/register' })}
    />
  );
}

function RegisterRoute() {
  const navigate = useNavigate();
  return (
    <RegisterPage
      onSuccess={() => navigate({ to: '/claims' })}
      onNavigateToLogin={() => navigate({ to: '/login' })}
    />
  );
}

// The authenticated app: every route inside here requires a session, checked
// once in beforeLoad rather than per-page, so an unauthenticated visitor
// never sees a protected page flash before the redirect. A pathless route
// (id, not path) so it adds a layout without adding a URL segment.
const appRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'app',
  beforeLoad: async () => {
    // A thrown redirect below must propagate untouched -- TanStack Router
    // implements the redirect itself as a thrown value, so only a genuine
    // health-check failure (network error, 500) is caught here. Fail closed
    // to the login screen: an unreachable backend is not evidence a session
    // exists, and the login page can show a clear error if the backend is
    // still down when the user submits.
    let health;
    try {
      health = await getHealth();
    } catch {
      throw redirect({ to: '/login' });
    }
    if (!health.principal) {
      throw redirect({ to: '/login' });
    }
  },
  component: () => (
    <Shell>
      <Outlet />
    </Shell>
  ),
});

const indexRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/claims' });
  },
});

const claimsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/claims',
  component: ClaimsRoute,
});

const claimDetailRoute = createRoute({
  getParentRoute: () => appRoute,
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
  getParentRoute: () => appRoute,
  path: '/recall',
  component: RecallPage,
});

const usageRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/usage',
  component: UsagePage,
});

const teamRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/team',
  component: () => <Placeholder title="Team" />,
});

const projectsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/projects',
  component: () => <Placeholder title="Projects" />,
});

const settingsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/settings',
  component: () => <Placeholder title="Settings" />,
});

const routeTree = rootRoute.addChildren([
  loginRoute,
  registerRoute,
  appRoute.addChildren([
    indexRoute,
    claimsRoute,
    claimDetailRoute,
    recallRoute,
    usageRoute,
    teamRoute,
    projectsRoute,
    settingsRoute,
  ]),
]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
