import { useQuery, useQueries, useMutation, useQueryClient } from '@tanstack/react-query'
import { getUser, listUsers, listDeletedUsers, createUser, deleteUser, changeUserRole, restoreUser, updateUser } from '../api/auth'
import type { CreateUserRequest, ChangeUserRoleRequest, UpdateUserRequest } from '../types/api'

export function useAllUsers() {
  return useQuery({
    queryKey: ['users'],
    queryFn: async () => (await listUsers()) ?? [],
    staleTime: 5 * 60_000,
  })
}

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
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useDeletedUsers() {
  return useQuery({
    queryKey: ['users', 'deleted'],
    queryFn: async () => (await listDeletedUsers()) ?? [],
    staleTime: 5 * 60_000,
  })
}

export function useDeleteUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteUser(id),
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: ['user', id] })
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useRestoreUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => restoreUser(id),
    onSuccess: (user) => {
      queryClient.setQueryData(['user', user.user_ref], user)
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useUpdateUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: string } & UpdateUserRequest) => updateUser(id, body),
    onSuccess: (user) => {
      queryClient.setQueryData(['user', user.user_ref], user)
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

export function useChangeUserRole() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: { id: string } & ChangeUserRoleRequest) => changeUserRole(id, body),
    onSuccess: (_data, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['user', id] })
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}
