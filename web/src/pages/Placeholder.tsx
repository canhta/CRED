import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout';
import { Center } from '@astryxdesign/core/Center';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { Heading } from '@astryxdesign/core/Text';

export function Placeholder({ title }: { title: string }) {
  return (
    <Layout
      header={
        <LayoutHeader hasDivider>
          <Heading level={4}>{title}</Heading>
        </LayoutHeader>
      }
      content={
        <LayoutContent>
          <Center height="100%">
            <EmptyState
              title="Coming soon"
              description={`${title} is not part of this release yet.`}
            />
          </Center>
        </LayoutContent>
      }
    />
  );
}
