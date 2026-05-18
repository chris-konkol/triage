import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Anchor, Button, Container, Paper, PasswordInput, Text, TextInput, Title,
} from '@mantine/core';
import { post } from '../api/client';
import { setToken } from '../utils/auth';

type Mode = 'login' | 'register';

export default function LoginPage() {
  const navigate = useNavigate();
  const [mode, setMode] = useState<Mode>('login');
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: { preventDefault(): void }) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const body = mode === 'login'
        ? { username, password }
        : { username, email, password };
      const data = await post<{ token: string }>(
        mode === 'login' ? '/auth/login' : '/auth/register',
        body,
      );
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
      <Text c="dimmed" size="sm" ta="center" mt={5}>
        {mode === 'login' ? "Don't have an account? " : 'Already have an account? '}
        <Anchor component="button" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
          {mode === 'login' ? 'Register' : 'Sign in'}
        </Anchor>
      </Text>

      <Paper withBorder shadow="md" p={30} mt={30} radius="md">
        <form onSubmit={handleSubmit}>
          <TextInput
            label="Username"
            value={username}
            onChange={e => setUsername(e.target.value)}
            required
            mb="sm"
          />
          {mode === 'register' && (
            <TextInput
              label="Email"
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              required
              mb="sm"
            />
          )}
          <PasswordInput
            label="Password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            required
            mb="sm"
          />
          {error && <Text c="red" size="sm" mb="sm">{error}</Text>}
          <Button fullWidth type="submit" loading={loading} mt="sm">
            {mode === 'login' ? 'Sign in' : 'Register'}
          </Button>
        </form>
      </Paper>
    </Container>
  );
}
