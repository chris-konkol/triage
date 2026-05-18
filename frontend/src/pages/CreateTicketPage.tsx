import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Button, Container, Group, Select, Stack, Text, Textarea, TextInput, Title,
} from '@mantine/core';
import { createTicket } from '../api/tickets';

export default function CreateTicketPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState('2');
  const [category, setCategory] = useState('3');

  const mutation = useMutation({
    mutationFn: createTicket,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
      navigate('/tickets');
    },
  });

  const handleSubmit = (e: { preventDefault(): void }) => {
    e.preventDefault();
    mutation.mutate({
      title,
      description,
      priority: parseInt(priority, 10),
      category: parseInt(category, 10),
    });
  };

  return (
    <Container size="sm" mt="lg">
      <Title order={2} mb="lg">New Ticket</Title>
      <form onSubmit={handleSubmit}>
        <Stack>
          <TextInput
            label="Title"
            value={title}
            onChange={e => setTitle(e.target.value)}
            required
          />
          <Textarea
            label="Description"
            value={description}
            onChange={e => setDescription(e.target.value)}
            rows={4}
          />
          <Select
            label="Priority"
            value={priority}
            onChange={v => setPriority(v ?? '2')}
            data={[
              { value: '1', label: 'Low' },
              { value: '2', label: 'Medium' },
              { value: '3', label: 'High' },
              { value: '4', label: 'Critical' },
            ]}
          />
          <Select
            label="Category"
            value={category}
            onChange={v => setCategory(v ?? '3')}
            data={[
              { value: '1', label: 'Bug' },
              { value: '2', label: 'Feature Request' },
              { value: '3', label: 'Support' },
              { value: '4', label: 'Documentation' },
              { value: '5', label: 'Infrastructure' },
            ]}
          />
          {mutation.isError && (
            <Text c="red" size="sm">{(mutation.error as Error).message}</Text>
          )}
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => navigate('/tickets')}>Cancel</Button>
            <Button type="submit" loading={mutation.isPending}>Create Ticket</Button>
          </Group>
        </Stack>
      </form>
    </Container>
  );
}
