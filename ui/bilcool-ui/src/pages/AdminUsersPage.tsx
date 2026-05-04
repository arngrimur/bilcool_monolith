import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useAllUsers, useDeletedUsers, useCreateUser, useDeleteUser, useChangeUserRole, useRestoreUser } from '../hooks/useUsers'
import { useAuthStore } from '../stores/authStore'
import { toast } from '../components/ui/use-toast'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'

const createSchema = z.object({
  username: z.string().min(1),
  email: z.string().email(),
})

type CreateFormValues = z.infer<typeof createSchema>

export default function AdminUsersPage() {
  const { t } = useTranslation('common')
  const role = useAuthStore((s) => s.role)
  const currentUserRef = useAuthStore((s) => s.userRef)
  const currentUsername = useAuthStore((s) => s.username)
  const currentEmail = useAuthStore((s) => s.email)

  const { data: allUsers = [] } = useAllUsers()
  const { data: deletedUsers = [] } = useDeletedUsers()

  const createUser = useCreateUser()
  const deleteUser = useDeleteUser()
  const restoreUser = useRestoreUser()
  const changeUserRole = useChangeUserRole()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateFormValues>({ resolver: zodResolver(createSchema) })

  async function onSubmit(data: CreateFormValues) {
    const user = await createUser.mutateAsync(data)
    toast({ title: t('admin.user_created', { username: user.username }) })
    reset()
  }

  async function handleDelete(id: string) {
    await deleteUser.mutateAsync(id)
    toast({ title: t('admin.user_deleted') })
  }

  async function handleRestore(id: string, username: string) {
    await restoreUser.mutateAsync(id)
    toast({ title: t('admin.user_restored', { username }) })
  }

  async function handleRoleChange(id: string, newRole: 'admin' | 'user') {
    await changeUserRole.mutateAsync({ id, role: newRole })
    toast({ title: t('admin.role_changed') })
  }

  if (role !== 'admin') {
    return (
      <div className="space-y-8 max-w-2xl">
        <h1 className="text-2xl font-bold">{t('nav.admin_users')}</h1>
        <div className="rounded-lg border overflow-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left font-medium">{t('admin.col_username')}</th>
                <th className="px-4 py-3 text-left font-medium">{t('admin.col_email')}</th>
                <th className="px-4 py-3 text-left font-medium">{t('admin.col_role')}</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td className="px-4 py-3">{currentUsername}</td>
                <td className="px-4 py-3">{currentEmail}</td>
                <td className="px-4 py-3">{role}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8 max-w-2xl">
      <h1 className="text-2xl font-bold">{t('admin.title')}</h1>

      <section aria-labelledby="create-user-heading">
        <h2 id="create-user-heading" className="text-lg font-semibold mb-4">{t('admin.create_user')}</h2>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
          <div className="space-y-2">
            <Label htmlFor="new-username">{t('admin.username_label')}</Label>
            <Input
              id="new-username"
              type="text"
              {...register('username')}
              aria-describedby={errors.username ? 'username-error' : undefined}
              disabled={isSubmitting}
            />
            {errors.username && (
              <p id="username-error" className="text-sm text-destructive" role="alert">
                {errors.username.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="new-email">{t('admin.email_label')}</Label>
            <Input
              id="new-email"
              type="email"
              {...register('email')}
              aria-describedby={errors.email ? 'email-error' : undefined}
              disabled={isSubmitting}
            />
            {errors.email && (
              <p id="email-error" className="text-sm text-destructive" role="alert">
                {errors.email.message}
              </p>
            )}
          </div>

          {createUser.error && (
            <p className="text-sm text-destructive" role="alert">
              {(createUser.error as { body?: { message?: string } }).body?.message ?? t('errors.unexpected')}
            </p>
          )}

          <Button type="submit" disabled={isSubmitting} className="min-h-[44px]">
            {isSubmitting ? '...' : t('admin.create_user')}
          </Button>
        </form>
      </section>

      <section aria-labelledby="users-list-heading">
        <h2 id="users-list-heading" className="text-lg font-semibold mb-4">{t('admin.title')}</h2>

        {allUsers.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('admin.no_users')}</p>
        ) : (
          <div className="rounded-lg border overflow-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_username')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_email')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_role')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_actions')}</th>
                </tr>
              </thead>
              <tbody>
                {allUsers.map((user) => (
                  <tr key={user.user_ref} className="border-b last:border-0">
                    <td className="px-4 py-3">{user.username}</td>
                    <td className="px-4 py-3">{user.email}</td>
                    <td className="px-4 py-3">{user.role}</td>
                    <td className="px-4 py-3 flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRoleChange(user.user_ref, user.role === 'admin' ? 'user' : 'admin')}
                        disabled={user.user_ref === currentUserRef || changeUserRole.isPending}
                        className="min-h-[44px]"
                      >
                        {user.role === 'admin' ? t('admin.demote_to_user') : t('admin.promote_to_admin')}
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDelete(user.user_ref)}
                        disabled={user.user_ref === currentUserRef || deleteUser.isPending}
                        aria-label={t('admin.delete_confirm', { username: user.username })}
                        className="min-h-[44px]"
                      >
                        {t('admin.delete_user')}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section aria-labelledby="deleted-users-heading">
        <h2 id="deleted-users-heading" className="text-lg font-semibold mb-4">{t('admin.deleted_users_title')}</h2>

        {deletedUsers.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('admin.no_deleted_users')}</p>
        ) : (
          <div className="rounded-lg border overflow-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_username')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_email')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_role')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_deleted_at')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('admin.col_actions')}</th>
                </tr>
              </thead>
              <tbody>
                {deletedUsers.map((user) => (
                  <tr key={user.user_ref} className="border-b last:border-0 text-muted-foreground">
                    <td className="px-4 py-3">{user.username}</td>
                    <td className="px-4 py-3">{user.email}</td>
                    <td className="px-4 py-3">{user.role}</td>
                    <td className="px-4 py-3">{new Date(user.deleted_at).toLocaleString()}</td>
                    <td className="px-4 py-3">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRestore(user.user_ref, user.username)}
                        disabled={restoreUser.isPending}
                        className="min-h-[44px]"
                      >
                        {t('admin.restore_user')}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}
