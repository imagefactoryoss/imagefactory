import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { packerTargetProfileService } from '@/services/packerTargetProfileService'
import {
  PackerTargetProfile,
  PackerTargetProvider,
  PackerTargetValidationResult,
} from '@/types'
import { CheckCircle2, RefreshCw, ShieldAlert, TestTube2, Trash2 } from 'lucide-react'
import React, { useMemo, useState } from 'react'
import toast from 'react-hot-toast'

type FormState = {
  id?: string
  name: string
  provider: PackerTargetProvider
  secret_ref: string
  description: string
  options: string
}

const emptyForm = (): FormState => ({
  name: '',
  provider: 'vmware',
  secret_ref: '',
  description: '',
  options: '{\n  \n}',
})

const statusBadgeClasses: Record<string, string> = {
  valid:
    'border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200',
  invalid:
    'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200',
  untested:
    'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200',
}

const formatStatus = (status: string) =>
  status === 'valid' ? 'Valid' : status === 'invalid' ? 'Invalid' : 'Untested'

const AdminPackerTargetProfilesPage: React.FC = () => {
  const [profiles, setProfiles] = useState<PackerTargetProfile[]>([])
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [validatingId, setValidatingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm())
  const [showForm, setShowForm] = useState(false)
  const [validationResult, setValidationResult] = useState<PackerTargetValidationResult | null>(null)

  const sortedProfiles = useMemo(
    () =>
      [...profiles].sort((a, b) => {
        const aTime = new Date(a.updated_at).getTime()
        const bTime = new Date(b.updated_at).getTime()
        return bTime - aTime
      }),
    [profiles],
  )

  const loadProfiles = async () => {
    setLoading(true)
    try {
      const next = await packerTargetProfileService.list()
      setProfiles(next)
    } catch (error: any) {
      toast.error(error.message || 'Failed to load target profiles')
    } finally {
      setLoading(false)
    }
  }

  React.useEffect(() => {
    void loadProfiles()
  }, [])

  const parseOptions = (raw: string): Record<string, any> => {
    const trimmed = raw.trim()
    if (!trimmed) return {}
    return JSON.parse(trimmed)
  }

  const startCreate = () => {
    setValidationResult(null)
    setForm(emptyForm())
    setShowForm(true)
  }

  const startEdit = (profile: PackerTargetProfile) => {
    setValidationResult(null)
    setForm({
      id: profile.id,
      name: profile.name,
      provider: profile.provider,
      secret_ref: profile.secret_ref,
      description: profile.description || '',
      options: JSON.stringify(profile.options || {}, null, 2),
    })
    setShowForm(true)
  }

  const onSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setSubmitting(true)
    try {
      const payload = {
        name: form.name.trim(),
        provider: form.provider,
        secret_ref: form.secret_ref.trim(),
        description: form.description.trim(),
        options: parseOptions(form.options),
      }
      if (form.id) {
        await packerTargetProfileService.update(form.id, payload)
        toast.success('Target profile updated')
      } else {
        await packerTargetProfileService.create(payload)
        toast.success('Target profile created')
      }
      setShowForm(false)
      setForm(emptyForm())
      await loadProfiles()
    } catch (error: any) {
      toast.error(error.message || 'Failed to save target profile')
    } finally {
      setSubmitting(false)
    }
  }

  const onValidate = async (profileId: string) => {
    setValidatingId(profileId)
    try {
      const result = await packerTargetProfileService.validate(profileId)
      setValidationResult(result)
      toast.success(result.status === 'valid' ? 'Validation passed' : 'Validation completed with issues')
      await loadProfiles()
    } catch (error: any) {
      toast.error(error.message || 'Failed to validate target profile')
    } finally {
      setValidatingId(null)
    }
  }

  const onDelete = async (profileId: string) => {
    const confirmed = window.confirm('Delete this target profile?')
    if (!confirmed) return
    setDeletingId(profileId)
    try {
      await packerTargetProfileService.delete(profileId)
      toast.success('Target profile deleted')
      if (validationResult?.profile_id === profileId) {
        setValidationResult(null)
      }
      await loadProfiles()
    } catch (error: any) {
      toast.error(error.message || 'Failed to delete target profile')
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">Packer Target Profiles</h1>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            Configure provider target profiles used by tenant Packer base-image builds.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => void loadProfiles()}
            disabled={loading}
            className="border-slate-300 text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Button
            onClick={startCreate}
            className="bg-slate-900 text-white hover:bg-slate-700 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-white"
          >
            New Profile
          </Button>
        </div>
      </div>

      {showForm ? (
        <Card className="border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-950">
          <CardHeader>
            <CardTitle className="text-slate-900 dark:text-slate-100">
              {form.id ? 'Edit Target Profile' : 'Create Target Profile'}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form className="grid gap-4 md:grid-cols-2" onSubmit={onSubmit}>
              <label className="text-sm font-medium text-slate-700 dark:text-slate-300">
                Name
                <input
                  required
                  value={form.name}
                  onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                  className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                />
              </label>
              <label className="text-sm font-medium text-slate-700 dark:text-slate-300">
                Provider
                <select
                  value={form.provider}
                  onChange={(event) =>
                    setForm((current) => ({ ...current, provider: event.target.value as PackerTargetProvider }))
                  }
                  className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                >
                  <option value="vmware">VMware</option>
                  <option value="aws">AWS</option>
                  <option value="azure">Azure</option>
                  <option value="gcp">GCP</option>
                </select>
              </label>
              <label className="text-sm font-medium text-slate-700 dark:text-slate-300 md:col-span-2">
                Secret Reference
                <input
                  required
                  value={form.secret_ref}
                  onChange={(event) => setForm((current) => ({ ...current, secret_ref: event.target.value }))}
                  placeholder="namespace/secret-name"
                  className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                />
              </label>
              <label className="text-sm font-medium text-slate-700 dark:text-slate-300 md:col-span-2">
                Description
                <input
                  value={form.description}
                  onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
                  className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                />
              </label>
              <label className="text-sm font-medium text-slate-700 dark:text-slate-300 md:col-span-2">
                Options (JSON)
                <textarea
                  required
                  rows={8}
                  value={form.options}
                  onChange={(event) => setForm((current) => ({ ...current, options: event.target.value }))}
                  className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 font-mono text-xs text-slate-900 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                />
              </label>
              <div className="md:col-span-2 flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setShowForm(false)
                    setForm(emptyForm())
                  }}
                  className="border-slate-300 text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  disabled={submitting}
                  className="bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-400"
                >
                  {submitting ? 'Saving...' : form.id ? 'Save Changes' : 'Create Profile'}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      ) : null}

      <Card className="border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-950">
        <CardHeader>
          <CardTitle className="text-slate-900 dark:text-slate-100">Configured Profiles</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center py-12 text-slate-600 dark:text-slate-300">
              <RefreshCw className="mr-2 h-5 w-5 animate-spin" />
              Loading profiles...
            </div>
          ) : sortedProfiles.length === 0 ? (
            <div className="rounded-lg border border-dashed border-slate-300 bg-slate-50 p-8 text-center text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300">
              No target profiles configured yet.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
                <thead className="bg-slate-50 dark:bg-slate-900">
                  <tr>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-400">
                      Name
                    </th>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-400">
                      Provider
                    </th>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-400">
                      Secret Ref
                    </th>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-400">
                      Validation
                    </th>
                    <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-400">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                  {sortedProfiles.map((profile) => (
                    <tr key={profile.id} className="bg-white hover:bg-slate-50 dark:bg-slate-950 dark:hover:bg-slate-900">
                      <td className="px-3 py-3 align-top">
                        <div className="text-sm font-medium text-slate-900 dark:text-slate-100">{profile.name}</div>
                        {profile.description ? (
                          <div className="mt-0.5 text-xs text-slate-600 dark:text-slate-400">{profile.description}</div>
                        ) : null}
                      </td>
                      <td className="px-3 py-3 align-top text-sm uppercase text-slate-700 dark:text-slate-300">{profile.provider}</td>
                      <td className="px-3 py-3 align-top font-mono text-xs text-slate-700 dark:text-slate-300">{profile.secret_ref}</td>
                      <td className="px-3 py-3 align-top">
                        <span
                          className={`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium ${statusBadgeClasses[profile.validation_status] || statusBadgeClasses.untested}`}
                        >
                          {profile.validation_status === 'valid' ? <CheckCircle2 className="mr-1 h-3.5 w-3.5" /> : null}
                          {profile.validation_status === 'invalid' ? <ShieldAlert className="mr-1 h-3.5 w-3.5" /> : null}
                          {formatStatus(profile.validation_status)}
                        </span>
                        {profile.last_validated_at ? (
                          <div className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
                            {new Date(profile.last_validated_at).toLocaleString()}
                          </div>
                        ) : null}
                      </td>
                      <td className="px-3 py-3 align-top">
                        <div className="flex justify-end gap-1">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => startEdit(profile)}
                            className="border-slate-300 text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
                          >
                            Edit
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={validatingId === profile.id}
                            onClick={() => void onValidate(profile.id)}
                            className="border-blue-300 text-blue-700 hover:bg-blue-50 dark:border-blue-700 dark:text-blue-300 dark:hover:bg-blue-900/40"
                          >
                            <TestTube2 className={`mr-1 h-3.5 w-3.5 ${validatingId === profile.id ? 'animate-pulse' : ''}`} />
                            Validate
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={deletingId === profile.id}
                            onClick={() => void onDelete(profile.id)}
                            className="border-red-300 text-red-700 hover:bg-red-50 dark:border-red-700 dark:text-red-300 dark:hover:bg-red-900/40"
                          >
                            <Trash2 className="mr-1 h-3.5 w-3.5" />
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {validationResult ? (
        <Card className="border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-950">
          <CardHeader>
            <CardTitle className="text-slate-900 dark:text-slate-100">Latest Validation Result</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="text-sm text-slate-700 dark:text-slate-300">{validationResult.message}</div>
            <div className="space-y-2">
              {validationResult.checks.map((check) => (
                <div
                  key={`${validationResult.profile_id}-${check.name}`}
                  className={`rounded-md border px-3 py-2 text-sm ${
                    check.ok
                      ? 'border-green-200 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200'
                      : 'border-red-200 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200'
                  }`}
                >
                  <div className="font-medium">{check.name}</div>
                  {check.message ? <div className="text-xs opacity-90">{check.message}</div> : null}
                  {check.remediation_hint ? <div className="text-xs opacity-90">Hint: {check.remediation_hint}</div> : null}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

export default AdminPackerTargetProfilesPage
