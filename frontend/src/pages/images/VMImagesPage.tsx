import Drawer from '@/components/ui/Drawer'
import { type VMImageCatalogItem, vmImageService } from '@/services/vmImageService'
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

const providerOptions = ['', 'aws', 'azure', 'gcp', 'vmware']
const statusOptions = ['', 'success', 'running', 'pending', 'failed', 'cancelled']

const VMImagesPage: React.FC = () => {
  const [items, setItems] = useState<VMImageCatalogItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [provider, setProvider] = useState('')
  const [status, setStatus] = useState('')
  const [totalCount, setTotalCount] = useState(0)
  const [selected, setSelected] = useState<VMImageCatalogItem | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await vmImageService.list({
        limit: 50,
        offset: 0,
        provider: provider || undefined,
        status: status || undefined,
        search: search || undefined,
      })
      setItems(response.data || [])
      setTotalCount(response.total_count || 0)
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to load VM image catalog')
      setItems([])
      setTotalCount(0)
    } finally {
      setLoading(false)
    }
  }, [provider, search, status])

  useEffect(() => {
    load()
  }, [load])

  const onSearchSubmit = (event: React.FormEvent) => {
    event.preventDefault()
    void load()
  }

  const openDetail = async (item: VMImageCatalogItem) => {
    setSelected(item)
    setDetailLoading(true)
    try {
      const response = await vmImageService.get(item.execution_id)
      setSelected(response.data)
    } catch {
      setSelected(item)
    } finally {
      setDetailLoading(false)
    }
  }

  const providerIds = useMemo(() => {
    if (!selected) return []
    return Object.entries(selected.provider_artifact_identifiers || {})
  }, [selected])

  return (
    <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">VM Images</h1>
          <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
            Tenant VM artifact catalog with source build traceability and provider-native identifiers.
          </p>
        </div>
        <Link
          to="/builds"
          className="inline-flex items-center rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
        >
          Back to Builds
        </Link>
      </div>

      <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
        <form className="grid gap-3 md:grid-cols-4" onSubmit={onSearchSubmit}>
          <input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search project, provider id, profile id"
            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 placeholder:text-slate-400 focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-200 dark:border-slate-600 dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-sky-400 dark:focus:ring-sky-900/50"
          />
          <select
            value={provider}
            onChange={(event) => setProvider(event.target.value)}
            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-200 dark:border-slate-600 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900/50"
          >
            {providerOptions.map((option) => (
              <option key={option || 'all'} value={option}>
                {option ? option.toUpperCase() : 'All providers'}
              </option>
            ))}
          </select>
          <select
            value={status}
            onChange={(event) => setStatus(event.target.value)}
            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-200 dark:border-slate-600 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900/50"
          >
            {statusOptions.map((option) => (
              <option key={option || 'all'} value={option}>
                {option ? option : 'All statuses'}
              </option>
            ))}
          </select>
          <button
            type="submit"
            className="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 dark:bg-sky-500 dark:hover:bg-sky-600"
          >
            Apply
          </button>
        </form>
        <p className="mt-3 text-xs text-slate-500 dark:text-slate-400">Total VM artifacts: {totalCount}</p>
      </section>

      <section className="rounded-lg border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900">
        {loading ? (
          <p className="p-4 text-sm text-slate-500 dark:text-slate-400">Loading VM image catalog...</p>
        ) : error ? (
          <p className="p-4 text-sm text-rose-700 dark:text-rose-300">{error}</p>
        ) : items.length === 0 ? (
          <p className="p-4 text-sm text-slate-500 dark:text-slate-400">No VM images found for this tenant.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
              <thead className="bg-slate-50 dark:bg-slate-800/60">
                <tr>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Project</th>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Provider</th>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Lifecycle</th>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Completed</th>
                  <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {items.map((item) => (
                  <tr key={item.execution_id} className="align-top">
                    <td className="px-3 py-3 text-xs text-slate-800 dark:text-slate-100">
                      <p className="font-medium">{item.project_name}</p>
                      <p className="mt-1 text-slate-500 dark:text-slate-400">
                        Build #{item.build_number}
                      </p>
                    </td>
                    <td className="px-3 py-3 text-xs text-slate-700 dark:text-slate-200">
                      {item.target_provider || 'unknown'}
                    </td>
                    <td className="px-3 py-3 text-xs text-slate-700 dark:text-slate-200">
                      <span className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 font-semibold text-slate-700 dark:bg-slate-800 dark:text-slate-200">
                        {item.lifecycle_state}
                      </span>
                      <p className="mt-1 text-slate-500 dark:text-slate-400">Exec: {item.execution_status}</p>
                    </td>
                    <td className="px-3 py-3 text-xs text-slate-700 dark:text-slate-200">
                      {item.completed_at ? new Date(item.completed_at).toLocaleString() : '-'}
                    </td>
                    <td className="px-3 py-3 text-right text-xs">
                      <button
                        type="button"
                        onClick={() => void openDetail(item)}
                        className="rounded-md border border-sky-300 bg-sky-50 px-2.5 py-1 font-medium text-sky-800 hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-900/30 dark:text-sky-200 dark:hover:bg-sky-900/50"
                      >
                        View
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <Drawer isOpen={Boolean(selected)} onClose={() => setSelected(null)} title="VM Image Details" size="lg">
        {selected ? (
          <div className="space-y-4">
            {detailLoading ? (
              <p className="text-xs text-slate-500 dark:text-slate-400">Refreshing details...</p>
            ) : null}
            <div className="rounded-md border border-slate-200 bg-slate-50 p-3 text-xs dark:border-slate-700 dark:bg-slate-800/50">
              <p className="font-semibold text-slate-900 dark:text-slate-100">
                {selected.target_provider || 'unknown'} / {selected.lifecycle_state}
              </p>
              <p className="mt-1 text-slate-600 dark:text-slate-300">Target profile: {selected.target_profile_id || '-'}</p>
              <p className="mt-1 text-slate-600 dark:text-slate-300">Execution: {selected.execution_id}</p>
              <p className="mt-1 text-slate-600 dark:text-slate-300">Build status: {selected.build_status}</p>
            </div>

            <div className="rounded-md border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900">
              <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Source Traceability</h3>
              <div className="mt-2 space-y-1 text-xs text-slate-700 dark:text-slate-200">
                <p>Project: {selected.project_name}</p>
                <p>Build: <Link to={`/builds/${selected.build_id}`} className="text-sky-600 hover:text-sky-500 dark:text-sky-300 dark:hover:text-sky-200">{selected.build_id}</Link></p>
                <p>Build Number: #{selected.build_number}</p>
                <p>Created: {new Date(selected.created_at).toLocaleString()}</p>
                <p>Completed: {selected.completed_at ? new Date(selected.completed_at).toLocaleString() : '-'}</p>
              </div>
            </div>

            <div className="rounded-md border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900">
              <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Provider Artifact IDs</h3>
              {providerIds.length === 0 ? (
                <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">No provider identifiers captured.</p>
              ) : (
                <div className="mt-2 space-y-2">
                  {providerIds.map(([providerKey, values]) => (
                    <div key={providerKey}>
                      <p className="text-xs font-medium text-slate-700 dark:text-slate-200">{providerKey}</p>
                      <ul className="mt-1 space-y-1">
                        {values.map((value) => (
                          <li key={value} className="break-all rounded bg-slate-100 px-2 py-1 text-[11px] text-slate-700 dark:bg-slate-800 dark:text-slate-200">
                            {value}
                          </li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded-md border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900">
              <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Execution Artifacts</h3>
              {selected.artifact_values.length === 0 ? (
                <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">No artifact values captured.</p>
              ) : (
                <ul className="mt-2 space-y-1">
                  {selected.artifact_values.map((value) => (
                    <li key={value} className="break-all rounded bg-slate-100 px-2 py-1 text-[11px] text-slate-700 dark:bg-slate-800 dark:text-slate-200">
                      {value}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        ) : null}
      </Drawer>
    </div>
  )
}

export default VMImagesPage

