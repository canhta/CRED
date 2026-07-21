import { useState } from 'react';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { VStack } from '@astryxdesign/core/Stack';
import { Heading, Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Button } from '@astryxdesign/core/Button';
import { Banner } from '@astryxdesign/core/Banner';
import { Link } from '@astryxdesign/core/Link';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { useHealth, useRegister, ApiError } from '../api';

export function RegisterPage({
  onSuccess,
  onNavigateToLogin,
}: {
  onSuccess: () => void;
  onNavigateToLogin: () => void;
}) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const health = useHealth();
  const register = useRegister();

  if (health.isLoading) {
    return null;
  }

  if (!health.data?.registration_open) {
    return (
      <Center height="100%">
        <EmptyState
          title="Registration is closed"
          description="An account already exists on this instance. Sign in instead."
        />
      </Center>
    );
  }

  const submit = () => {
    register.mutate({ email, password }, { onSuccess });
  };

  const errorMessage =
    register.error instanceof ApiError
      ? register.error.status === 409
        ? 'That email is already registered.'
        : 'Registration failed. Check your details and try again.'
      : null;

  return (
    <Center height="100%">
      <Card padding={4} width={360}>
        <VStack gap={4}>
          <VStack gap={1}>
            <Heading level={4}>Create the first account</Heading>
            <Text type="body" color="secondary">
              This account becomes the console's admin.
            </Text>
          </VStack>
          {errorMessage ? <Banner status="error" title={errorMessage} /> : null}
          <TextInput
            label="Email"
            type="email"
            value={email}
            onChange={setEmail}
            onEnter={submit}
            hasAutoFocus
          />
          <TextInput
            label="Password"
            description="At least 8 characters."
            type="password"
            value={password}
            onChange={setPassword}
            onEnter={submit}
          />
          <Button
            label="Create account"
            variant="primary"
            onClick={submit}
            isLoading={register.isPending}
          />
          <Link
            href="/login"
            onClick={(e) => {
              e.preventDefault();
              onNavigateToLogin();
            }}
          >
            Already have an account? Sign in
          </Link>
        </VStack>
      </Card>
    </Center>
  );
}
