import { useState } from 'react';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { VStack } from '@astryxdesign/core/Stack';
import { Heading, Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Button } from '@astryxdesign/core/Button';
import { Banner } from '@astryxdesign/core/Banner';
import { Link } from '@astryxdesign/core/Link';
import { useLogin, ApiError } from '../api';

export function LoginPage({
  onSuccess,
  onNavigateToRegister,
}: {
  onSuccess: () => void;
  onNavigateToRegister: () => void;
}) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const login = useLogin();

  const submit = () => {
    login.mutate({ email, password }, { onSuccess });
  };

  const errorMessage =
    login.error instanceof ApiError
      ? login.error.status === 429
        ? 'Too many login attempts. Wait a moment and try again.'
        : 'Invalid email or password.'
      : null;

  return (
    <Center height="100dvh">
      <Card padding={4} width={360}>
        <VStack gap={4}>
          <VStack gap={1}>
            <Heading level={4}>Welcome back</Heading>
            <Text type="body" color="secondary">
              CRED console
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
            type="password"
            value={password}
            onChange={setPassword}
            onEnter={submit}
          />
          <Button
            label="Sign in"
            variant="primary"
            onClick={submit}
            isLoading={login.isPending}
          />
          <Link
            href="/register"
            onClick={(e) => {
              e.preventDefault();
              onNavigateToRegister();
            }}
          >
            Need an account? Register
          </Link>
        </VStack>
      </Card>
    </Center>
  );
}
