import { useQueries, useMutation, useQueryClient } from '@tanstack/react-query'
import { getUser, createUser, deleteUser } from '../api/auth'
import type { CreateUserRequest } from '../types/api'

export function useUsers(userRefs: string[]) {
  return useQueries({
    queries: userRefs.map((id) => ({
      queryKey: ['user', id] as const,
      queryFn: () => getUser(id),
      staleTime: 5 * 60_000,
    })),
  })
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateUserRequest) => createUser(body),
    onSuccess: (user) => {
      queryClient.setQueryData(['user', user.user_ref], user)
    },
  })
}

export function useDeleteUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteUser(id),
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: ['user', id] })
    },
  })
}
