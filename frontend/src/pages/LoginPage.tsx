import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button, Container, Paper, PasswordInput, Text, TextInput, Title,
} from '@mantine/core';
import { post } from '../api/client';
import { setToken } from '../utils/auth';

export default function LoginPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: { preventDefault(): void }) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await post<{ token: string }>('/auth/login', { username, password });
      setToken(data.token);
      navigate('/tickets');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container size={420} mt={80}>
      <Title ta="center">Triage</Title>
      <Paper withBorder shadow="md" p={30} mt={30} radius="md">
        <form onSubmit={handleSubmit}>
          <TextInput
            label="Username"
            value={username}
            onChange={e => setUsername(e.target.value)}
            required
            mb="sm"
          />
          <PasswordInput
            label="Password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            required
            mb="sm"
          />
          {error && <Text c="red" size="sm" mb="sm">{error}</Text>}
          <Button fullWidth type="submit" loading={loading} mt="sm">
            Sign in
          </Button>
        </form>
      </Paper>
    </Container>
  );
}
