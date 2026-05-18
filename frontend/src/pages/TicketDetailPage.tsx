import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Badge, Button, Container, Divider, Group, Loader, Paper, Select,
  Stack, Text, Textarea, TextInput, Title,
} from '@mantine/core';
import { getTicket, updateTicket, addComment, deleteTicket } from '../api/tickets';

const PRIORITY_COLOR: Record<string, string> = {
  PRIORITY_LOW: 'blue', PRIORITY_MEDIUM: 'yellow',
  PRIORITY_HIGH: 'orange', PRIORITY_CRITICAL: 'red',
};
const STATUS_COLOR: Record<string, string> = {
  STATUS_OPEN: 'green', STATUS_IN_PROGRESS: 'blue',
  STATUS_WAITING: 'yellow', STATUS_RESOLVED: 'gray', STATUS_CLOSED: 'dark',
};

function label(value: string, prefix: string) {
  return value.replace(prefix, '').replaceAll('_', ' ');
}

const STATUS_OPTIONS = [
  { value: '1', label: 'Open' },
  { value: '2', label: 'In Progress' },
  { value: '3', label: 'Waiting' },
  { value: '4', label: 'Resolved' },
  { value: '5', label: 'Closed' },
];

const PRIORITY_OPTIONS = [
  { value: '1', label: 'Low' },
  { value: '2', label: 'Medium' },
  { value: '3', label: 'High' },
  { value: '4', label: 'Critical' },
];

export default function TicketDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [commentBody, setCommentBody] = useState('');
  const [assignedTo, setAssignedTo] = useState('');
  const [editingAssignee, setEditingAssignee] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ['ticket', id],
    queryFn: () => getTicket(id!),
    enabled: !!id,
  });

  useEffect(() => {
    if (data) setAssignedTo(data.ticket.assignedTo ?? '');
  }, [data]);

  const updateMutation = useMutation({
    mutationFn: (payload: Parameters<typeof updateTicket>[1]) => updateTicket(id!, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ticket', id] });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      setEditingAssignee(false);
    },
  });

  const commentMutation = useMutation({
    mutationFn: (body: string) => addComment(id!, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ticket', id] });
      setCommentBody('');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteTicket(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      navigate('/tickets');
    },
  });

  if (isLoading) return <Container mt="lg"><Loader /></Container>;
  if (error) return <Container mt="lg"><Text c="red">Failed to load ticket.</Text></Container>;
  if (!data) return null;

  const { ticket, comments } = data;

  return (
    <Container size="md" mt="lg">
      <Group mb="md">
        <Button variant="subtle" onClick={() => navigate('/tickets')}>← Back</Button>
      </Group>

      <Paper withBorder p="lg" radius="md" mb="md">
        <Group justify="space-between" mb="xs">
          <Title order={3}>{ticket.title}</Title>
          <Group>
            <Badge color={PRIORITY_COLOR[ticket.priority] ?? 'gray'} size="sm">
              {label(ticket.priority, 'PRIORITY_')}
            </Badge>
            <Badge color={STATUS_COLOR[ticket.status] ?? 'gray'} size="sm" variant="light">
              {label(ticket.status, 'STATUS_')}
            </Badge>
          </Group>
        </Group>

        <Text c="dimmed" size="sm" mb="md">{ticket.description}</Text>

        <Group gap="xl" mb="md">
          <Stack gap={2}>
            <Text size="xs" c="dimmed">Created by</Text>
            <Text size="sm">{ticket.createdBy}</Text>
          </Stack>
          <Stack gap={2}>
            <Text size="xs" c="dimmed">Category</Text>
            <Text size="sm">{label(ticket.category, 'CATEGORY_')}</Text>
          </Stack>
          <Stack gap={2}>
            <Text size="xs" c="dimmed">Created</Text>
            <Text size="sm">{new Date(ticket.createdAt).toLocaleDateString()}</Text>
          </Stack>
          <Stack gap={2}>
            <Text size="xs" c="dimmed">Assigned to</Text>
            {editingAssignee ? (
              <Group gap="xs">
                <TextInput
                  size="xs"
                  value={assignedTo}
                  onChange={e => setAssignedTo(e.target.value)}
                  style={{ width: 140 }}
                />
                <Button size="xs" onClick={() => updateMutation.mutate({ assignedTo })}
                  loading={updateMutation.isPending}>Save</Button>
                <Button size="xs" variant="subtle" onClick={() => setEditingAssignee(false)}>Cancel</Button>
              </Group>
            ) : (
              <Group gap="xs">
                <Text size="sm">{ticket.assignedTo || 'Unassigned'}</Text>
                <Button size="xs" variant="subtle" onClick={() => setEditingAssignee(true)}>Edit</Button>
              </Group>
            )}
          </Stack>
        </Group>

        <Divider mb="md" />

        <Group mb="md" gap="sm">
          <Select
            size="xs"
            label="Status"
            value={String(Object.values({
              STATUS_OPEN: 1, STATUS_IN_PROGRESS: 2, STATUS_WAITING: 3,
              STATUS_RESOLVED: 4, STATUS_CLOSED: 5,
            })[Object.keys({
              STATUS_OPEN: 1, STATUS_IN_PROGRESS: 2, STATUS_WAITING: 3,
              STATUS_RESOLVED: 4, STATUS_CLOSED: 5,
            }).indexOf(ticket.status)] ?? 1)}
            onChange={v => v && updateMutation.mutate({ status: parseInt(v, 10) })}
            data={STATUS_OPTIONS}
            style={{ width: 140 }}
          />
          <Select
            size="xs"
            label="Priority"
            value={String(Object.values({
              PRIORITY_LOW: 1, PRIORITY_MEDIUM: 2, PRIORITY_HIGH: 3, PRIORITY_CRITICAL: 4,
            })[Object.keys({
              PRIORITY_LOW: 1, PRIORITY_MEDIUM: 2, PRIORITY_HIGH: 3, PRIORITY_CRITICAL: 4,
            }).indexOf(ticket.priority)] ?? 2)}
            onChange={v => v && updateMutation.mutate({ priority: parseInt(v, 10) })}
            data={PRIORITY_OPTIONS}
            style={{ width: 140 }}
          />
          <Button
            size="xs"
            color="red"
            variant="light"
            mt="lg"
            onClick={() => {
              if (confirm('Delete this ticket?')) deleteMutation.mutate();
            }}
            loading={deleteMutation.isPending}
          >
            Delete
          </Button>
        </Group>
      </Paper>

      <Title order={5} mb="sm">Comments ({(comments ?? []).length})</Title>

      <Stack mb="md">
        {(comments ?? []).length === 0 && (
          <Text c="dimmed" size="sm">No comments yet.</Text>
        )}
        {(comments ?? []).map(c => (
          <Paper key={c.id} withBorder p="sm" radius="md">
            <Group justify="space-between" mb={4}>
              <Text size="sm" fw={500}>{c.author}</Text>
              <Text size="xs" c="dimmed">{new Date(c.createdAt).toLocaleString()}</Text>
            </Group>
            <Text size="sm">{c.body}</Text>
          </Paper>
        ))}
      </Stack>

      <Paper withBorder p="md" radius="md">
        <Textarea
          label="Add a comment"
          value={commentBody}
          onChange={e => setCommentBody(e.target.value)}
          rows={3}
          mb="sm"
        />
        <Button
          onClick={() => commentBody.trim() && commentMutation.mutate(commentBody)}
          loading={commentMutation.isPending}
          disabled={!commentBody.trim()}
        >
          Post Comment
        </Button>
        {commentMutation.isError && (
          <Text c="red" size="sm" mt="xs">{(commentMutation.error as Error).message}</Text>
        )}
      </Paper>
    </Container>
  );
}
