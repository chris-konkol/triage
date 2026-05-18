import { render, type RenderResult } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, type MemoryRouterProps } from 'react-router-dom';

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

interface Options {
  routerProps?: MemoryRouterProps;
  queryClient?: QueryClient;
}

export function renderWithProviders(
  ui: React.ReactElement,
  { routerProps, queryClient }: Options = {},
): RenderResult {
  const client = queryClient ?? makeQueryClient();
  return render(
    <QueryClientProvider client={client}>
      <MantineProvider>
        <MemoryRouter {...routerProps}>{ui}</MemoryRouter>
      </MantineProvider>
    </QueryClientProvider>,
  );
}
