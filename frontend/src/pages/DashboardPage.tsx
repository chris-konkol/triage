import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Badge, Container, Grid, Group, Loader, Paper, SimpleGrid, Text, Title,
} from '@mantine/core';
import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend,
} from 'recharts';
import { getDashboardStats } from '../api/tickets';

const PRIORITY_COLORS: Record<string, string> = {
  Low: '#4dabf7',
  Medium: '#ffd43b',
  High: '#ff922b',
  Critical: '#fa5252',
};

const CATEGORY_COLORS = ['#7950f2', '#20c997', '#f76707', '#1c7ed6', '#e64980'];

function StatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <Paper withBorder p="md" radius="md">
      <Text size="sm" c="dimmed">{label}</Text>
      <Badge color={color} size="xl" mt="xs" variant="light">{value}</Badge>
    </Paper>
  );
}

export default function DashboardPage() {
  const navigate = useNavigate();
  const { data, isLoading, error } = useQuery({
    queryKey: ['dashboard'],
    queryFn: getDashboardStats,
    refetchInterval: 30000,
  });

  return (
    <Container size="xl" mt="lg">
      <Group justify="space-between" mb="lg">
        <Title order={2}>Dashboard</Title>
        <Text
          component="button"
          c="blue"
          style={{ cursor: 'pointer', background: 'none', border: 'none' }}
          onClick={() => navigate('/tickets')}
        >
          View Tickets
        </Text>
      </Group>

      {isLoading && <Loader />}
      {error && <Text c="red">Failed to load dashboard: {(error as Error).message}</Text>}

      {data && (
        <>
          <SimpleGrid cols={{ base: 2, sm: 4 }} mb="xl">
            <StatCard label="Open" value={data.totalOpen} color="green" />
            <StatCard label="In Progress" value={data.totalInProgress} color="blue" />
            <StatCard label="Resolved" value={data.totalResolved} color="gray" />
            <StatCard label="Closed" value={data.totalClosed} color="dark" />
          </SimpleGrid>

          {data.avgResolutionHours > 0 && (
            <Text c="dimmed" size="sm" mb="xl">
              Avg resolution time: {data.avgResolutionHours.toFixed(1)} hours
            </Text>
          )}

          <Grid mb="xl">
            <Grid.Col span={{ base: 12, md: 6 }}>
              <Paper withBorder p="md" radius="md">
                <Text fw={500} mb="md">Tickets by Priority</Text>
                <ResponsiveContainer width="100%" height={220}>
                  <PieChart>
                    <Pie
                      data={data.ticketsByPriority ?? []}
                      dataKey="count"
                      nameKey="priority"
                      cx="50%"
                      cy="50%"
                      outerRadius={80}
                      label={(p) => `${(p as any).priority} (${(p as any).count})`}
                    >
                      {(data.ticketsByPriority ?? []).map((entry) => (
                        <Cell key={entry.priority} fill={PRIORITY_COLORS[entry.priority] ?? '#adb5bd'} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              </Paper>
            </Grid.Col>

            <Grid.Col span={{ base: 12, md: 6 }}>
              <Paper withBorder p="md" radius="md">
                <Text fw={500} mb="md">Tickets by Category</Text>
                <ResponsiveContainer width="100%" height={220}>
                  <PieChart>
                    <Pie
                      data={data.ticketsByCategory ?? []}
                      dataKey="count"
                      nameKey="category"
                      cx="50%"
                      cy="50%"
                      outerRadius={80}
                      label={(p) => `${(p as any).category} (${(p as any).count})`}
                    >
                      {(data.ticketsByCategory ?? []).map((entry, i) => (
                        <Cell key={entry.category} fill={CATEGORY_COLORS[i % CATEGORY_COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip />
                    <Legend />
                  </PieChart>
                </ResponsiveContainer>
              </Paper>
            </Grid.Col>
          </Grid>

          {(data.ticketsPerDay ?? []).length > 0 && (
            <Paper withBorder p="md" radius="md">
              <Text fw={500} mb="md">Daily Activity (last 14 days)</Text>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={data.ticketsPerDay ?? []}>
                  <XAxis dataKey="date" tick={{ fontSize: 11 }} />
                  <YAxis allowDecimals={false} />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="created" fill="#4dabf7" name="Created" />
                  <Bar dataKey="resolved" fill="#20c997" name="Resolved" />
                </BarChart>
              </ResponsiveContainer>
            </Paper>
          )}
        </>
      )}
    </Container>
  );
}
