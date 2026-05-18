import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Badge, Button, Container, Group, Loader, Select, Table, Text, TextInput, Title,
} from '@mantine/core';
import { listTickets } from '../api/tickets';
import { clearToken } from '../utils/auth';
import type { Ticket } from '../types/ticket';

const PRIORITY_COLOR: Record<string, string> = {
  PRIORITY_LOW: 'blue',
  PRIORITY_MEDIUM: 'yellow',
  PRIORITY_HIGH: 'orange',
  PRIORITY_CRITICAL: 'red',
};

const STATUS_COLOR: Record<string, string> = {
  STATUS_OPEN: 'green',
  STATUS_IN_PROGRESS: 'blue',
  STATUS_WAITING: 'yellow',
  STATUS_RESOLVED: 'gray',
  STATUS_CLOSED: 'dark',
};

function label(value: string, prefix: string) {
  return value.replace(prefix, '').replaceAll('_', ' ');
}

export default function TicketsPage() {
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<string | null>(null);
  const [priority, setPriority] = useState<string | null>(null);
  const [category, setCategory] = useState<string | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ['tickets', search, status, priority, category],
    queryFn: () => listTickets({
      search: search || undefined,
      status: status ? parseInt(status, 10) : undefined,
      priority: priority ? parseInt(priority, 10) : undefined,
      category: category ? parseInt(category, 10) : undefined,
    }),
  });

  const handleLogout = () => {
    clearToken();
    navigate('/login');
  };

  return (
    <Container size="xl" mt="lg">
      <Group justify="space-between" mb="lg">
        <Title order={2}>Tickets</Title>
        <Group>
          <Button onClick={() => navigate('/tickets/new')}>New Ticket</Button>
          <Button variant="light" onClick={() => navigate('/dashboard')}>Dashboard</Button>
          <Button variant="subtle" onClick={handleLogout}>Logout</Button>
        </Group>
      </Group>

      <Group mb="md" gap="sm">
        <TextInput
          placeholder="Search..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          style={{ width: 200 }}
        />
        <Select
          placeholder="Status"
          clearable
          value={status}
          onChange={setStatus}
          data={[
            { value: '1', label: 'Open' },
            { value: '2', label: 'In Progress' },
            { value: '3', label: 'Waiting' },
            { value: '4', label: 'Resolved' },
            { value: '5', label: 'Closed' },
          ]}
          style={{ width: 140 }}
        />
        <Select
          placeholder="Priority"
          clearable
          value={priority}
          onChange={setPriority}
          data={[
            { value: '1', label: 'Low' },
            { value: '2', label: 'Medium' },
            { value: '3', label: 'High' },
            { value: '4', label: 'Critical' },
          ]}
          style={{ width: 130 }}
        />
        <Select
          placeholder="Category"
          clearable
          value={category}
          onChange={setCategory}
          data={[
            { value: '1', label: 'Bug' },
            { value: '2', label: 'Feature Request' },
            { value: '3', label: 'Support' },
            { value: '4', label: 'Documentation' },
            { value: '5', label: 'Infrastructure' },
          ]}
          style={{ width: 130 }}
        />
      </Group>

      {isLoading && <Loader />}
      {error && <Text c="red">Failed to load tickets: {(error as Error).message}</Text>}

      {data && (
        <>
          <Text c="dimmed" size="sm" mb="sm">{data.totalCount ?? 0} tickets</Text>
          <Table striped highlightOnHover withTableBorder>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Title</Table.Th>
                <Table.Th>Priority</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Category</Table.Th>
                <Table.Th>Created by</Table.Th>
                <Table.Th>Date</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {(data.tickets ?? []).map((t: Ticket) => (
                <Table.Tr key={t.id} onClick={() => navigate(`/tickets/${t.id}`)} style={{ cursor: 'pointer' }}>
                  <Table.Td>{t.title}</Table.Td>
                  <Table.Td>
                    <Badge color={PRIORITY_COLOR[t.priority] ?? 'gray'} size="sm">
                      {label(t.priority, 'PRIORITY_')}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Badge color={STATUS_COLOR[t.status] ?? 'gray'} size="sm" variant="light">
                      {label(t.status, 'STATUS_')}
                    </Badge>
                  </Table.Td>
                  <Table.Td>{label(t.category, 'CATEGORY_')}</Table.Td>
                  <Table.Td>{t.createdBy}</Table.Td>
                  <Table.Td>{new Date(t.createdAt).toLocaleDateString()}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </>
      )}
    </Container>
  );
}
