import type { ReactNode } from 'react';
import { VStack } from '@astryxdesign/core/Stack';
import type { VStackProps } from '@astryxdesign/core/Stack';
import { Text } from '@astryxdesign/core/Text';

// ClaimDetailPage's metadata fields and RecallPage's accounting stats were
// two near-identical label-over-value stacks; this is the one shared shape.
export function LabeledValue({
  label,
  children,
  gap = 1,
  weight = 'medium',
  hasTabularNumbers = false,
}: {
  label: string;
  children: ReactNode;
  gap?: VStackProps['gap'];
  weight?: 'medium' | 'semibold';
  hasTabularNumbers?: boolean;
}) {
  return (
    <VStack gap={gap}>
      <Text type="supporting" color="secondary">
        {label}
      </Text>
      {typeof children === 'string' ? (
        <Text type="body" weight={weight} hasTabularNumbers={hasTabularNumbers}>
          {children}
        </Text>
      ) : (
        children
      )}
    </VStack>
  );
}
