import { apiFetch } from './client';
import type {
  LoginBeginRequest,
  LoginBeginResponse,
  ResetLoginRequest,
  VerifyTokenRequest,
  VerifyTokenResponse,
  LoginCompleteRequest,
  LoginCompleteResponse,
  CreateUserRequest,
  UserResponse,
  ChangeUserRoleRequest,
} from '../types/api';

const BASE = '/api/v1';

export const beginLogin = (body: LoginBeginRequest) =>
  apiFetch<LoginBeginResponse>(`${BASE}/users/login`, { method: 'POST', body: JSON.stringify(body) });

export const requestLoginReset = (body: ResetLoginRequest) =>
  apiFetch<void>(`${BASE}/users/login/reset`, { method: 'POST', body: JSON.stringify(body) });

export const verifyToken = (body: VerifyTokenRequest) =>
  apiFetch<VerifyTokenResponse>(`${BASE}/users/login/token`, { method: 'POST', body: JSON.stringify(body) });

export const completeLogin = (body: LoginCompleteRequest) =>
  apiFetch<LoginCompleteResponse>(`${BASE}/users/login/complete`, { method: 'POST', body: JSON.stringify(body) });

export const createUser = (body: CreateUserRequest) =>
  apiFetch<UserResponse>(`${BASE}/users`, { method: 'POST', body: JSON.stringify(body) });

export const getUser = (id: string) =>
  apiFetch<UserResponse>(`${BASE}/users/${id}`);

export const listUsers = () =>
  apiFetch<UserResponse[]>(`${BASE}/users`);

export const deleteUser = (id: string) =>
  apiFetch<void>(`${BASE}/users/${id}`, { method: 'DELETE' });

export const changeUserRole = (id: string, body: ChangeUserRoleRequest) =>
  apiFetch<void>(`${BASE}/users/${id}/role`, { method: 'PATCH', body: JSON.stringify(body) });
