"use client";

import React from "react";
import { api, toQuery, API_BASE, normalizeLabel, NLabel } from "@/lib/api";
import TagSelector from "@/components/TagSelector";

type Label = NLabel;

type Filter = {
  tags: string[];
  mode: "any" | "all";
  batch?: number;
  limit: number;
  offset: number;
};

export default function LabelsPage() {
  const [filter, setFilter] = React.useState<Filter>({ tags: [], mode: "any", limit: 50, offset: 0 });
  const [labels, setLabels] = React.useState<Label[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const q = toQuery({
        tags: filter.tags.join(","),
        mode: filter.mode,
        batch: filter.batch,
        limit: filter.limit,
        offset: filter.offset,
      });
  const data = await api<any[]>(`/api/labels${q}`);
  setLabels(data.map(normalizeLabel));
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  React.useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [applyTag, setApplyTag] = React.useState<{ id: number; name: string } | null>(null);

  async function doApplyTag() {
    if (!applyTag) return;
    setLoading(true);
    setError(null);
    try {
      const res = await api<{ added: number }>(`/api/labels/tags/apply`, {
        method: "POST",
        body: JSON.stringify({ tagId: applyTag.id, filter }),
      });
      alert(`Added tag to ${res.added} labels`);
      await load();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  const exportHref = `${API_BASE}/api/export${toQuery({
    tags: filter.tags.join(","),
    mode: filter.mode,
    batch: filter.batch,
  })}`;

  return (
    <div className="container">
      <h1 className="text-2xl font-semibold">Labels</h1>
      <div className="card">
        <div className="row flex-wrap">
          <div>
            <label className="label">Tags (comma separated)</label>
            <input
              className="input w-64"
              type="text"
              value={filter.tags.join(",")}
              onChange={(e) => setFilter((f) => ({ ...f, tags: e.target.value.split(",").map(s => s.trim()).filter(Boolean) }))}
              placeholder="tag-a,tag-b"
            />
          </div>
          <div>
            <label className="label">Mode</label>
            <select className="input" value={filter.mode} onChange={(e) => setFilter((f) => ({ ...f, mode: e.target.value as Filter["mode"] }))}>
              <option value="any">ANY</option>
              <option value="all">ALL</option>
            </select>
          </div>
          <div>
            <label className="label">Batch ID (optional)</label>
            <input className="input w-28" type="number" value={filter.batch || ""} onChange={(e) => setFilter((f) => ({ ...f, batch: e.target.value ? Number(e.target.value) : undefined }))} />
          </div>
          <div>
            <label className="label">Limit</label>
            <input className="input w-24" type="number" value={filter.limit} onChange={(e) => setFilter((f) => ({ ...f, limit: Number(e.target.value) }))} />
          </div>
          <div>
            <label className="label">Offset</label>
            <input className="input w-24" type="number" value={filter.offset} onChange={(e) => setFilter((f) => ({ ...f, offset: Number(e.target.value) }))} />
          </div>
          <div className="row items-end">
            <button className="btn-primary" onClick={load} disabled={loading}>Search</button>
            <a className="btn-outline ml-2" href={exportHref} target="_blank" rel="noreferrer">Export CSV</a>
          </div>
        </div>
      </div>
      <div className="card">
        <div className="row items-center gap-3">
          <strong>Bulk apply tag to current filter:</strong>
          <TagSelector value={applyTag as any} onChange={(t) => setApplyTag(t as any)} />
          <button className="btn-outline" onClick={doApplyTag} disabled={!applyTag || loading}>Apply</button>
        </div>
      </div>

      {error && <div className="card text-red-700">{error}</div>}

      <div className="card">
        <h3 className="font-semibold">Results ({labels.length})</h3>
        <div className="overflow-auto">
          <table className="min-w-full text-sm">
            <thead className="bg-gray-50 text-left">
              <tr>
                <th className="py-2 px-2">ID</th>
                <th className="py-2 px-2">ASCII</th>
                <th className="py-2 px-2">Unicode</th>
                <th className="py-2 px-2">Tags</th>
                <th className="py-2 px-2">Created</th>
              </tr>
            </thead>
            <tbody>
              {labels.map((l) => (
                <tr key={l.id} className="border-t border-gray-100">
                  <td className="py-2 px-2">{l.id}</td>
                  <td className="py-2 px-2">{l.label_ascii}</td>
                  <td className="py-2 px-2">{l.label_unicode}</td>
                  <td className="py-2 px-2">
                    <div className="flex flex-wrap gap-1">
                      {(l.tags || []).map((t) => (
                        <span key={`${l.id}-${t.id}`} className="inline-flex items-center rounded-full bg-blue-50 text-blue-700 px-2 py-0.5 border border-blue-200">
                          {t.name}
                          {t.group_name ? <span className="ml-1 text-xs text-blue-500">({t.group_name})</span> : null}
                        </span>
                      ))}
                      {(l.tags || []).length === 0 && <span className="text-gray-400">—</span>}
                    </div>
                  </td>
                  <td className="py-2 px-2">{l.created_at ? new Date(l.created_at).toLocaleString() : ""}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
