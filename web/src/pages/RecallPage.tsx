import { useState } from 'react';
import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout';
import { HStack, VStack } from '@astryxdesign/core/Stack';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { Heading, Text } from '@astryxdesign/core/Text';
import { Icon } from '@astryxdesign/core/Icon';
import { Badge } from '@astryxdesign/core/Badge';
import { Button } from '@astryxdesign/core/Button';
import { Spinner } from '@astryxdesign/core/Spinner';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { ProgressBar } from '@astryxdesign/core/ProgressBar';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { TextInput } from '@astryxdesign/core/TextInput';
import { List, ListItem } from '@astryxdesign/core/List';
import { LabeledValue } from '../components/LabeledValue';
import { useRecall, ApiError } from '../api';
import type {
  RecallParams,
  RecallResponse,
  RecalledClaim,
  ArmContribution,
} from '../api';

// Retrieval arms are categories, not statuses: each gets a distinct tint so a
// claim found by both arms shows two differently-colored badges at a glance.
function armVariant(arm: string): 'purple' | 'teal' | 'neutral' {
  if (arm === 'dense') return 'purple';
  if (arm === 'lexical') return 'teal';
  return 'neutral';
}

export function RecallPage() {
  const [draft, setDraft] = useState('');
  const [query, setQuery] = useState<RecallParams>({ q: '' });
  const submitted = query.q.length > 0;
  const recall = useRecall(query, submitted);

  const submit = () => {
    const q = draft.trim();
    if (q.length === 0) return;
    setQuery({ q });
  };

  return (
    <Layout
      header={
        <LayoutHeader hasDivider>
          <HStack vAlign="center" hAlign="between" gap={4}>
            <Heading level={4}>Recall Inspector</Heading>
            <HStack gap={2} vAlign="center">
              <TextInput
                label="Recall query"
                isLabelHidden
                placeholder="Ask the recall engine…"
                startIcon={<Icon icon="search" />}
                value={draft}
                onChange={setDraft}
                onEnter={submit}
                hasClear
                width={360}
              />
              <Button label="Search" variant="primary" onClick={submit} />
            </HStack>
          </HStack>
        </LayoutHeader>
      }
      content={
        <LayoutContent>
          <RecallBody
            submitted={submitted}
            isLoading={recall.isLoading}
            error={recall.error}
            data={recall.data}
          />
        </LayoutContent>
      }
    />
  );
}

function RecallBody({
  submitted,
  isLoading,
  error,
  data,
}: {
  submitted: boolean;
  isLoading: boolean;
  error: unknown;
  data: RecallResponse | undefined;
}) {
  if (!submitted) {
    return (
      <Center height="100%">
        <EmptyState
          title="Inspect a recall"
          description="Enter a query to see which claims the engine retrieves — and why each one ranked, arm by arm."
        />
      </Center>
    );
  }

  if (isLoading) {
    return (
      <Center height="100%">
        <Spinner label="Running recall" />
      </Center>
    );
  }

  if (error) {
    const status = error instanceof ApiError ? error.status : 0;
    const description =
      status === 503
        ? 'The recall model is not loaded on the server, so semantic retrieval is unavailable.'
        : status === 429
          ? 'The recall endpoint is rate limited. Wait a moment and try again.'
          : 'The recall request failed. Check that the CRED server is running.';
    return (
      <Center height="100%">
        <EmptyState title="Couldn't run recall" description={description} />
      </Center>
    );
  }

  if (!data) {
    return null;
  }

  return (
    <VStack gap={3} height="100%">
      <AccountingBar data={data} />
      {data.claims.length === 0 ? (
        <Center height="100%">
          <EmptyState
            title="No claims recalled"
            description="No claims matched this query within the current access and token budget."
          />
        </Center>
      ) : (
        <List hasDividers density="compact">
          {data.claims.map((claim, index) => (
            <ResultItem key={claim.id} claim={claim} rank={index + 1} />
          ))}
        </List>
      )}
    </VStack>
  );
}

function AccountingBar({ data }: { data: RecallResponse }) {
  const share = Math.round(data.dominant_share * 100);
  return (
    <Card variant="muted" padding={3}>
      <HStack gap={6} vAlign="center" wrap="wrap">
        <LabeledValue label="Candidates" gap={0} weight="semibold" hasTabularNumbers>
          {String(data.candidates)}
        </LabeledValue>
        <LabeledValue label="Authorized" gap={0} weight="semibold" hasTabularNumbers>
          {String(data.authorized)}
        </LabeledValue>
        <LabeledValue label="Dominant arm" gap={0}>
          {data.dominant_arm ? (
            <HStack gap={1} vAlign="center">
              <Badge
                variant={armVariant(data.dominant_arm)}
                label={data.dominant_arm}
              />
              <Text type="body" weight="semibold" hasTabularNumbers>
                {share}%
              </Text>
            </HStack>
          ) : (
            <Text type="body" color="secondary">
              —
            </Text>
          )}
        </LabeledValue>
        <LabeledValue label="Total latency" gap={0} weight="semibold" hasTabularNumbers>
          {`${data.timings.total_ms.toFixed(1)} ms`}
        </LabeledValue>
        {data.omitted_for_budget > 0 ? (
          <LabeledValue label="Omitted for budget" gap={0}>
            <Badge
              variant="warning"
              label={`${data.omitted_for_budget} dropped`}
            />
          </LabeledValue>
        ) : null}
      </HStack>
    </Card>
  );
}

function ResultItem({ claim, rank }: { claim: RecalledClaim; rank: number }) {
  return (
    <ListItem
      label={claim.statement}
      startContent={
        <VStack gap={0} align="center" width={56}>
          <Text type="label" color="secondary">
            #{rank}
          </Text>
          <Text type="body" weight="bold" hasTabularNumbers>
            {claim.score.toFixed(3)}
          </Text>
        </VStack>
      }
      endContent={
        <HStack gap={2} vAlign="center">
          <StatusDot
            variant={claim.status === 'live' ? 'success' : 'neutral'}
            label={claim.status}
          />
          <Text type="supporting">{claim.status}</Text>
        </HStack>
      }
      description={<ResultDetail claim={claim} />}
    />
  );
}

function ResultDetail({ claim }: { claim: RecalledClaim }) {
  return (
    <VStack gap={1}>
      <HStack gap={3} vAlign="center" wrap="wrap">
        <Text type="supporting" color="secondary">
          {claim.scope.kind}: {claim.scope.value}
        </Text>
        {claim.source ? (
          <Text type="supporting" color="secondary">
            {claim.source.path}:{claim.source.line_start}-
            {claim.source.line_end}
            {claim.source.symbol_path ? ` · ${claim.source.symbol_path}` : ''}
          </Text>
        ) : null}
      </HStack>
      <HStack gap={3} vAlign="center" wrap="wrap">
        {claim.contributions.map((c) => (
          <ArmBreakdown
            key={c.arm}
            contribution={c}
            fused={claim.score}
          />
        ))}
      </HStack>
    </VStack>
  );
}

function ArmBreakdown({
  contribution,
  fused,
}: {
  contribution: ArmContribution;
  fused: number;
}) {
  const share = fused > 0 ? (contribution.score / fused) * 100 : 0;
  return (
    <HStack gap={1.5} vAlign="center">
      <Badge
        variant={armVariant(contribution.arm)}
        label={`${contribution.arm} #${contribution.rank}`}
      />
      <VStack width={64} align="center">
        <ProgressBar
          label={`${contribution.arm} share of fused score`}
          isLabelHidden
          value={share}
          variant="accent"
        />
      </VStack>
      <Text type="supporting" color="secondary" hasTabularNumbers>
        +{contribution.score.toFixed(3)}
      </Text>
    </HStack>
  );
}
