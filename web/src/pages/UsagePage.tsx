import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout';
import { HStack, VStack } from '@astryxdesign/core/Stack';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { Heading, Text } from '@astryxdesign/core/Text';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Badge } from '@astryxdesign/core/Badge';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { ProgressBar } from '@astryxdesign/core/ProgressBar';
import { Table, proportional, pixel } from '@astryxdesign/core/Table';
import type { TableColumn } from '@astryxdesign/core/Table';
import { useQueryClient } from '@tanstack/react-query';
import { useUsage, queryKeys } from '../api';
import type { LimitStatus, ScopeCost, ScopeGrowth, UsageResponse } from '../api';

type ScopeCostRow = ScopeCost & Record<string, unknown>;
type ScopeGrowthRow = ScopeGrowth & Record<string, unknown>;

const scopeCostColumns: TableColumn<ScopeCostRow>[] = [
  {
    key: 'scope',
    header: 'Scope',
    width: proportional(2),
    renderCell: (row) => (
      <Text type="body" maxLines={1}>
        {row.scope.kind}: {row.scope.value}
      </Text>
    ),
  },
  {
    key: 'calls',
    header: 'Calls',
    width: pixel(100),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.calls}
      </Text>
    ),
  },
  {
    key: 'input_tokens',
    header: 'Input tokens',
    width: pixel(140),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.input_tokens}
      </Text>
    ),
  },
  {
    key: 'output_tokens',
    header: 'Output tokens',
    width: pixel(140),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.output_tokens}
      </Text>
    ),
  },
];

const scopeGrowthColumns: TableColumn<ScopeGrowthRow>[] = [
  {
    key: 'scope',
    header: 'Scope',
    width: proportional(2),
    renderCell: (row) => (
      <Text type="body" maxLines={1}>
        {row.scope.kind}: {row.scope.value}
      </Text>
    ),
  },
  {
    key: 'live',
    header: 'Live claims',
    width: pixel(120),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.live}
      </Text>
    ),
  },
  {
    key: 'ceiling',
    header: 'Ceiling',
    width: pixel(100),
    renderCell: (row) => (
      <Text type="body" color="secondary" hasTabularNumbers>
        {row.ceiling <= 0 ? 'off' : row.ceiling}
      </Text>
    ),
  },
  {
    key: 'next_prune',
    header: 'Next prune',
    width: pixel(120),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.next_prune}
      </Text>
    ),
  },
];

export function UsagePage() {
  const usage = useUsage();
  const queryClient = useQueryClient();

  return (
    <Layout
      header={
        <LayoutHeader hasDivider>
          <HStack vAlign="center" hAlign="between">
            <Heading level={4}>Usage</Heading>
            <Button
              label="Refresh"
              variant="secondary"
              onClick={() => {
                void queryClient.invalidateQueries({
                  queryKey: queryKeys.usage({}),
                });
              }}
            />
          </HStack>
        </LayoutHeader>
      }
      content={
        <LayoutContent>
          <UsageBody
            isLoading={usage.isLoading}
            isError={usage.isError}
            data={usage.data}
          />
        </LayoutContent>
      }
    />
  );
}

function UsageBody({
  isLoading,
  isError,
  data,
}: {
  isLoading: boolean;
  isError: boolean;
  data: UsageResponse | undefined;
}) {
  if (isLoading) {
    return (
      <Center height="100%">
        <Spinner label="Loading usage" />
      </Center>
    );
  }

  if (isError || !data) {
    return (
      <Center height="100%">
        <EmptyState
          title="Couldn't load usage"
          description="The API request failed. Check that the CRED server is running."
        />
      </Center>
    );
  }

  return (
    <VStack gap={4}>
      {data.denied > 0 ? (
        <Banner
          status="warning"
          title={`${data.denied} contribution(s) denied in the last ${data.denied_window}`}
          description="Recorded and on the record — nothing was silently dropped."
        />
      ) : null}

      <HStack gap={3} wrap="wrap">
        <LimitCard label="Contribution" status={data.contribution} />
        <LimitCard
          label="Inference cost"
          status={data.cost}
          extra={`${data.input_tokens_used} / ${
            data.input_tokens_ceiling <= 0 ? 'off' : data.input_tokens_ceiling
          } input tokens`}
        />
        <LimitCard label="Recall" status={data.recall} />
      </HStack>

      <VStack gap={2}>
        <Heading level={5}>Cost by scope</Heading>
        {data.cost_by_scope.length === 0 ? (
          <Text type="body" color="secondary">
            No inference cost recorded in this window.
          </Text>
        ) : (
          <Table<ScopeCostRow>
            data={data.cost_by_scope as ScopeCostRow[]}
            columns={scopeCostColumns}
            idKey={(row) => `${row.scope.kind}:${row.scope.value}`}
            textOverflow="truncate"
          />
        )}
      </VStack>

      <VStack gap={2}>
        <Heading level={5}>Scope growth</Heading>
        {data.scope_growth.length === 0 ? (
          <Text type="body" color="secondary">
            No live claims.
          </Text>
        ) : (
          <Table<ScopeGrowthRow>
            data={data.scope_growth as ScopeGrowthRow[]}
            columns={scopeGrowthColumns}
            idKey={(row) => `${row.scope.kind}:${row.scope.value}`}
            textOverflow="truncate"
          />
        )}
      </VStack>
    </VStack>
  );
}

function LimitCard({
  label,
  status,
  extra,
}: {
  label: string;
  status: LimitStatus;
  extra?: string;
}) {
  const hasCeiling = status.ceiling > 0;
  const value = hasCeiling
    ? Math.min((status.used / status.ceiling) * 100, 100)
    : 0;

  return (
    <Card padding={3}>
      <VStack gap={2} width={220}>
        <HStack hAlign="between" vAlign="center">
          <Text type="body" weight="semibold">
            {label}
          </Text>
          {!status.allowed ? (
            <Badge variant="warning" label="exhausted" />
          ) : null}
        </HStack>
        {hasCeiling ? (
          <ProgressBar
            label={`${label} usage`}
            isLabelHidden
            value={value}
            variant={status.allowed ? 'accent' : 'error'}
          />
        ) : (
          <Text type="supporting" color="secondary">
            unlimited
          </Text>
        )}
        <Text type="supporting" color="secondary" hasTabularNumbers>
          {status.used} / {hasCeiling ? status.ceiling : 'off'} per{' '}
          {status.window}
        </Text>
        {extra ? (
          <Text type="supporting" color="secondary" hasTabularNumbers>
            {extra}
          </Text>
        ) : null}
      </VStack>
    </Card>
  );
}
