import { useState } from 'react';
import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout';
import { HStack, VStack } from '@astryxdesign/core/Stack';
import { Center } from '@astryxdesign/core/Center';
import { Heading, Text } from '@astryxdesign/core/Text';
import { Spinner } from '@astryxdesign/core/Spinner';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import {
  SegmentedControl,
  SegmentedControlItem,
} from '@astryxdesign/core/SegmentedControl';
import { Table, proportional, pixel } from '@astryxdesign/core/Table';
import type { TableColumn } from '@astryxdesign/core/Table';
import { useClaims } from '../api';
import type { ClaimListItem, StatusFilter } from '../api';

// Astryx Table constrains rows to Record<string, unknown>; the API type has no
// index signature, so we widen it locally rather than pollute the API contract.
type ClaimRow = ClaimListItem & Record<string, unknown>;

const columns: TableColumn<ClaimRow>[] = [
  {
    key: 'statement',
    header: 'Statement',
    width: proportional(2),
    renderCell: (claim) => (
      <Text type="body" weight="medium" maxLines={1}>
        {claim.statement}
      </Text>
    ),
  },
  {
    key: 'kind',
    header: 'Kind',
    width: pixel(120),
    renderCell: (claim) => (
      <Text type="body" color="secondary">
        {claim.kind}
      </Text>
    ),
  },
  {
    key: 'scope',
    header: 'Scope',
    width: proportional(1),
    renderCell: (claim) => (
      <Text type="body" color="secondary" maxLines={1}>
        {claim.scope.kind}: {claim.scope.value}
      </Text>
    ),
  },
  {
    key: 'status',
    header: 'Status',
    width: pixel(120),
    renderCell: (claim) => (
      <HStack gap={2} vAlign="center">
        <StatusDot
          variant={claim.status === 'live' ? 'success' : 'neutral'}
          label={claim.status}
        />
        <Text type="body">{claim.status}</Text>
      </HStack>
    ),
  },
  {
    key: 'source',
    header: 'Source',
    width: proportional(2),
    renderCell: (claim) =>
      claim.source ? (
        <VStack gap={0}>
          <Text type="body" maxLines={1}>
            {claim.source.path}:{claim.source.line_start}-{claim.source.line_end}
          </Text>
          {claim.source.symbol_path ? (
            <Text type="supporting" color="secondary" maxLines={1}>
              {claim.source.symbol_path}
            </Text>
          ) : null}
        </VStack>
      ) : (
        <Text type="body" color="secondary">
          —
        </Text>
      ),
  },
  {
    key: 'contributed_by',
    header: 'Contributed by',
    width: pixel(160),
    renderCell: (claim) => (
      <Text type="body" color="secondary">
        {claim.contributed_by}
      </Text>
    ),
  },
];

export function ClaimsPage() {
  const [status, setStatus] = useState<StatusFilter>('all');
  const claims = useClaims({ status });

  return (
    <Layout
      header={
        <LayoutHeader hasDivider>
          <HStack vAlign="center" hAlign="between">
            <Heading level={4}>Claims</Heading>
            <SegmentedControl
              label="Filter claims by status"
              size="sm"
              value={status}
              onChange={(value) => setStatus(value as StatusFilter)}
            >
              <SegmentedControlItem value="all" label="All" />
              <SegmentedControlItem value="live" label="Live" />
              <SegmentedControlItem value="expired" label="Expired" />
            </SegmentedControl>
          </HStack>
        </LayoutHeader>
      }
      content={
        <LayoutContent>
          <ClaimsBody
            isLoading={claims.isLoading}
            isError={claims.isError}
            items={claims.data?.claims ?? []}
          />
        </LayoutContent>
      }
    />
  );
}

function ClaimsBody({
  isLoading,
  isError,
  items,
}: {
  isLoading: boolean;
  isError: boolean;
  items: ClaimListItem[];
}) {
  if (isLoading) {
    return (
      <Center height="100%">
        <Spinner label="Loading claims" />
      </Center>
    );
  }

  if (isError) {
    return (
      <Center height="100%">
        <EmptyState
          title="Couldn't load claims"
          description="The API request failed. Check that the CRED server is running."
        />
      </Center>
    );
  }

  if (items.length === 0) {
    return (
      <Center height="100%">
        <EmptyState
          title="No claims"
          description="No claims match the current filter."
        />
      </Center>
    );
  }

  return (
    <Table<ClaimRow>
      data={items as ClaimRow[]}
      columns={columns}
      idKey="id"
      hasHover
      textOverflow="truncate"
    />
  );
}
