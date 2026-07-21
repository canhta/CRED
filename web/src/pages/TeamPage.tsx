import { useState } from 'react';
import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout';
import { HStack, VStack } from '@astryxdesign/core/Stack';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { Heading, Text } from '@astryxdesign/core/Text';
import { Spinner } from '@astryxdesign/core/Spinner';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { Button } from '@astryxdesign/core/Button';
import { Banner } from '@astryxdesign/core/Banner';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Selector } from '@astryxdesign/core/Selector';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { Table, proportional, pixel } from '@astryxdesign/core/Table';
import type { TableColumn } from '@astryxdesign/core/Table';
import {
  useTeamMembers,
  useInvites,
  useCreateInvite,
  useRevokeInvite,
  ApiError,
} from '../api';
import type { TeamMember, Invite } from '../api';

type MemberRow = TeamMember & Record<string, unknown>;
type InviteRow = Invite & Record<string, unknown>;

const memberColumns: TableColumn<MemberRow>[] = [
  {
    key: 'email',
    header: 'Email',
    width: proportional(2),
    renderCell: (row) => <Text type="body">{row.email}</Text>,
  },
  {
    key: 'role',
    header: 'Role',
    width: pixel(120),
    renderCell: (row) => (
      <Text type="body" color="secondary">
        {row.role}
      </Text>
    ),
  },
  {
    key: 'created_at',
    header: 'Joined',
    width: pixel(160),
    renderCell: (row) => (
      <Text type="body" color="secondary">
        {new Date(row.created_at).toLocaleDateString()}
      </Text>
    ),
  },
];

function inviteColumns(onRevoke: (id: string) => void): TableColumn<InviteRow>[] {
  return [
    {
      key: 'email',
      header: 'Email',
      width: proportional(2),
      renderCell: (row) => <Text type="body">{row.email}</Text>,
    },
    {
      key: 'role',
      header: 'Role',
      width: pixel(120),
      renderCell: (row) => (
        <Text type="body" color="secondary">
          {row.role}
        </Text>
      ),
    },
    {
      key: 'expires_at',
      header: 'Expires',
      width: pixel(160),
      renderCell: (row) => (
        <Text type="body" color="secondary">
          {new Date(row.expires_at).toLocaleDateString()}
        </Text>
      ),
    },
    {
      key: 'revoke',
      header: '',
      width: pixel(100),
      renderCell: (row) => (
        <Button
          label="Revoke"
          variant="destructive"
          size="sm"
          onClick={() => onRevoke(row.id)}
        />
      ),
    },
  ];
}

export function TeamPage() {
  const members = useTeamMembers();
  const invites = useInvites();
  const revokeInvite = useRevokeInvite();
  const [isInviting, setIsInviting] = useState(false);

  return (
    <>
    <Layout
      header={
        <LayoutHeader hasDivider>
          <HStack vAlign="center" hAlign="between">
            <Heading level={4}>Team</Heading>
            <Button
              label="Invite"
              variant="primary"
              onClick={() => setIsInviting(true)}
            />
          </HStack>
        </LayoutHeader>
      }
      content={
        <LayoutContent>
          <VStack gap={4}>
            <VStack gap={2}>
              <Heading level={5}>Members</Heading>
              {members.isLoading ? (
                <Center height={120}>
                  <Spinner label="Loading members" />
                </Center>
              ) : members.isError || !members.data ? (
                <EmptyState
                  title="Couldn't load members"
                  description="The API request failed. Check that the CRED server is running."
                />
              ) : (
                <Table<MemberRow>
                  data={members.data as MemberRow[]}
                  columns={memberColumns}
                  idKey="principal_id"
                  textOverflow="truncate"
                />
              )}
            </VStack>

            <VStack gap={2}>
              <Heading level={5}>Pending invites</Heading>
              {invites.isLoading ? (
                <Center height={120}>
                  <Spinner label="Loading invites" />
                </Center>
              ) : invites.isError || !invites.data ? (
                <EmptyState
                  title="Couldn't load invites"
                  description="The API request failed. Check that the CRED server is running."
                />
              ) : invites.data.length === 0 ? (
                <Text type="body" color="secondary">
                  No pending invites.
                </Text>
              ) : (
                <Table<InviteRow>
                  data={invites.data as InviteRow[]}
                  columns={inviteColumns((id) => revokeInvite.mutate(id))}
                  idKey="id"
                  textOverflow="truncate"
                />
              )}
            </VStack>
          </VStack>
        </LayoutContent>
      }
    />
    <InviteDialog isOpen={isInviting} onOpenChange={setIsInviting} />
    </>
  );
}

function InviteDialog({
  isOpen,
  onOpenChange,
}: {
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
}) {
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('member');
  const [link, setLink] = useState<string | null>(null);
  const [isCopied, setIsCopied] = useState(false);
  const createInvite = useCreateInvite();

  const close = () => {
    onOpenChange(false);
    setEmail('');
    setRole('member');
    setLink(null);
    setIsCopied(false);
    createInvite.reset();
  };

  const submit = () => {
    createInvite.mutate(
      { email, role },
      {
        onSuccess: (res) => {
          setLink(`${window.location.origin}/register?invite=${res.token}`);
        },
      },
    );
  };

  const errorMessage =
    createInvite.error instanceof ApiError
      ? 'Could not create the invite. Check the email and try again.'
      : null;

  return (
    <Dialog
      isOpen={isOpen}
      onOpenChange={(open) => {
        if (!open) close();
      }}
      purpose="form"
    >
      <DialogHeader
        title={link ? 'Invite created' : 'Invite a teammate'}
        onOpenChange={() => close()}
      />
      {link ? (
        <VStack gap={2}>
          <Text type="body" color="secondary">
            Share this link with {email} — it works once, and expires in 7
            days.
          </Text>
          <Card padding={2}>
            <Text type="body" hasTabularNumbers>
              {link}
            </Text>
          </Card>
          <Button
            label={isCopied ? 'Copied' : 'Copy link'}
            variant="secondary"
            onClick={() => {
              void navigator.clipboard.writeText(link);
              setIsCopied(true);
            }}
          />
          <Button label="Done" variant="primary" onClick={close} />
        </VStack>
      ) : (
        <VStack gap={4}>
          {errorMessage ? <Banner status="error" title={errorMessage} /> : null}
          <TextInput
            label="Email"
            type="email"
            value={email}
            onChange={setEmail}
            hasAutoFocus
          />
          <Selector
            label="Role"
            value={role}
            onChange={(value) => setRole(value)}
            options={[
              { value: 'member', label: 'Member' },
              { value: 'admin', label: 'Admin' },
            ]}
          />
          <Button
            label="Send invite"
            variant="primary"
            onClick={submit}
            isLoading={createInvite.isPending}
          />
        </VStack>
      )}
    </Dialog>
  );
}
