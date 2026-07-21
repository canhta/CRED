import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout';
import { HStack, VStack } from '@astryxdesign/core/Stack';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { Heading, Text } from '@astryxdesign/core/Text';
import { Badge } from '@astryxdesign/core/Badge';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { Spinner } from '@astryxdesign/core/Spinner';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { Divider } from '@astryxdesign/core/Divider';
import { Code } from '@astryxdesign/core/Code';
import { Banner } from '@astryxdesign/core/Banner';
import { List, ListItem } from '@astryxdesign/core/List';
import { Breadcrumbs, BreadcrumbItem } from '@astryxdesign/core/Breadcrumbs';
import { LabeledValue } from '../components/LabeledValue';
import { useClaim, ApiError } from '../api';
import type { ClaimDetail, Evidence } from '../api';

// Navigation is injected, not reached for with a router hook, so the page stays
// a pure component the tests render without a router context — the same seam
// the claims list uses.
export function ClaimDetailPage({
  id,
  onBack,
}: {
  id: string;
  onBack?: () => void;
}) {
  const claim = useClaim(id);

  return (
    <Layout
      header={
        <LayoutHeader hasDivider>
          <HStack vAlign="center" hAlign="between" gap={4}>
            <Breadcrumbs label="Claim location">
              <BreadcrumbItem
                href="/claims"
                onClick={(e) => {
                  e.preventDefault();
                  onBack?.();
                }}
              >
                Claims
              </BreadcrumbItem>
              <BreadcrumbItem isCurrent>{shortId(id)}</BreadcrumbItem>
            </Breadcrumbs>
            {claim.data ? <StatusIndicator status={claim.data.status} /> : null}
          </HStack>
        </LayoutHeader>
      }
      content={
        <LayoutContent>
          <DetailBody
            isLoading={claim.isLoading}
            error={claim.error}
            data={claim.data}
          />
        </LayoutContent>
      }
    />
  );
}

function DetailBody({
  isLoading,
  error,
  data,
}: {
  isLoading: boolean;
  error: unknown;
  data: ClaimDetail | undefined;
}) {
  if (isLoading) {
    return (
      <Center height="100%">
        <Spinner label="Loading claim" />
      </Center>
    );
  }

  if (error) {
    const status = error instanceof ApiError ? error.status : 0;
    return (
      <Center height="100%">
        {status === 404 ? (
          <EmptyState
            title="Claim not found"
            description="This claim does not exist, or the current principal may not read it."
          />
        ) : (
          <EmptyState
            title="Couldn't load claim"
            description="The API request failed. Check that the CRED server is running."
          />
        )}
      </Center>
    );
  }

  if (!data) {
    return null;
  }

  return (
    <VStack gap={4}>
      <Heading level={3}>{data.statement}</Heading>

      {data.status === 'expired' ? (
        <Banner
          status="warning"
          title="This claim has expired"
          description={
            data.expired_reason ||
            'Its evidence no longer supports it, so recall no longer returns it.'
          }
        />
      ) : null}

      <Card variant="muted" padding={4}>
        <VStack gap={4}>
          <HStack gap={6} vAlign="start" wrap="wrap">
            <LabeledValue label="Kind">
              <Badge variant="neutral" label={data.kind} />
            </LabeledValue>
            <LabeledValue label="Scope">
              {data.scope.kind}: {data.scope.value}
            </LabeledValue>
            <LabeledValue label="Confidence">
              {`${Math.round(data.confidence * 100)}%`}
            </LabeledValue>
            <LabeledValue label="Contributed by">{data.contributed_by}</LabeledValue>
            {data.source_repo ? (
              <LabeledValue label="Source repo">{data.source_repo}</LabeledValue>
            ) : null}
          </HStack>
          <Divider />
          <HStack gap={6} vAlign="start" wrap="wrap">
            <LabeledValue label="Recorded">{fmtDate(data.recorded_at)}</LabeledValue>
            <LabeledValue label="Valid from">{fmtDate(data.valid_from)}</LabeledValue>
            <LabeledValue label="Valid until">{fmtDate(data.valid_until)}</LabeledValue>
            <LabeledValue label="Superseded">{fmtDate(data.superseded_at)}</LabeledValue>
          </HStack>
        </VStack>
      </Card>

      <VStack gap={2}>
        <Heading level={5}>Evidence ({data.evidence.length})</Heading>
        <EvidenceList evidence={data.evidence} />
      </VStack>
    </VStack>
  );
}

function EvidenceList({ evidence }: { evidence: Evidence[] }) {
  if (evidence.length === 0) {
    return (
      <Text type="body" color="secondary">
        This claim rests on attestation, not a code span.
      </Text>
    );
  }
  return (
    <List hasDividers density="compact">
      {evidence.map((e) => (
        <ListItem
          key={e.id}
          label={<Code>{location(e)}</Code>}
          description={
            e.symbol_path ? (
              <Text type="supporting" color="secondary">
                {e.symbol_path}
              </Text>
            ) : (
              e.repo
            )
          }
          endContent={
            <Badge
              variant={e.anchor === 'anchored' ? 'teal' : 'neutral'}
              label={e.anchor}
            />
          }
        />
      ))}
    </List>
  );
}

function StatusIndicator({ status }: { status: string }) {
  return (
    <HStack gap={2} vAlign="center">
      <StatusDot
        variant={status === 'live' ? 'success' : 'neutral'}
        label={status}
      />
      <Text type="body">{status}</Text>
    </HStack>
  );
}

// The anchor carries a symbol path but not always a line range; render whatever
// locates the evidence within its file.
function location(e: Evidence): string {
  if (e.line_start > 0) {
    return `${e.path}:${e.line_start}-${e.line_end}`;
  }
  return e.path;
}

function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}…` : id;
}

// Timestamps arrive as RFC3339 or empty; an empty or unparseable value reads as
// an em dash rather than "Invalid Date".
function fmtDate(iso: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString();
}
