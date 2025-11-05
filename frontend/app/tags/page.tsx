"use client";

import React from "react";
import { api, normalizeTag } from "@/lib/api";

type Tag = { id: number; name: string; group_name?: string | null; created_at?: string };

export default function TagsPage() {
  const [prefix, setPrefix] = React.useState("");
  const [tags, setTags] = React.useState<Tag[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  async function search() {
    setLoading(true);
    setError(null);
    try {
  const data = await api<any[]>(`/api/tags?prefix=${encodeURIComponent(prefix)}&limit=100`);
  setTags(data.map(normalizeTag));
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  async function createTag(name: string, groupName?: string) {
    setLoading(true);
    setError(null);
    try {
      await api<Tag>(`/api/tags`, { method: "POST", body: JSON.stringify({ name, groupName }) });
      setPrefix("");
      await search();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  async function renameTag(id: number, name?: string, groupName?: string) {
    setLoading(true);
    setError(null);
    try {
      await api<Tag>(`/api/tags/${id}`, { method: "PATCH", body: JSON.stringify({ name, groupName }) });
      await search();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  async function deleteTag(id: number) {
    if (!confirm("Delete this tag?")) return;
    setLoading(true);
    setError(null);
    try {
      await api<void>(`/api/tags/${id}`, { method: "DELETE" });
      await search();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  React.useEffect(() => {
    search();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [newName, setNewName] = React.useState("");
  const [newGroup, setNewGroup] = React.useState("");

  return (
    <div className="container">
      <h1 className="text-2xl font-semibold">Tags</h1>
      <div className="card">
        <div className="row flex-wrap">
          <div>
            <label className="label">Prefix</label>
            <input className="input" value={prefix} onChange={(e) => setPrefix(e.target.value)} placeholder="ca" />
          </div>
          <div className="self-end">
            <button className="btn-primary" onClick={search} disabled={loading}>Search</button>
          </div>
        </div>
      </div>
      <div className="card">
        <h3 className="font-semibold">Create Tag</h3>
        <div className="row flex-wrap">
          <div>
            <label className="label">Name</label>
            <input className="input w-64" value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="category-premium" />
          </div>
          <div>
            <label className="label">Group (optional)</label>
            <input className="input w-40" value={newGroup} onChange={(e) => setNewGroup(e.target.value)} placeholder="category" />
          </div>
          <div className="self-end">
            <button className="btn-outline" onClick={() => createTag(newName, newGroup || undefined)} disabled={loading || !newName}>Create</button>
          </div>
        </div>
      </div>

      {error && <div className="card text-red-700">{error}</div>}

      <div className="card">
        <h3 className="font-semibold">Results ({tags.length})</h3>
        <div className="overflow-auto">
          <table className="min-w-full text-sm">
            <thead className="bg-gray-50 text-left">
              <tr>
                <th className="py-2 px-2">ID</th>
                <th className="py-2 px-2">Name</th>
                <th className="py-2 px-2">Group</th>
                <th className="py-2 px-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {tags.map((t) => (
                <tr key={t.id} className="border-t border-gray-100">
                  <td className="py-2 px-2">{t.id}</td>
                  <td className="py-2 px-2">{t.name}</td>
                  <td className="py-2 px-2">{t.group_name || ""}</td>
                  <td className="py-2 px-2">
                    <InlineEditTag tag={t} onSave={renameTag} onDelete={deleteTag} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function InlineEditTag({ tag, onSave, onDelete }: { tag: Tag; onSave: (id: number, name?: string, groupName?: string) => void; onDelete: (id: number) => void }) {
  const [name, setName] = React.useState(tag.name);
  const [group, setGroup] = React.useState(tag.group_name || "");
  return (
    <div className="row">
      <input value={name} onChange={(e) => setName(e.target.value)} style={{ width: 200 }} />
      <input value={group} onChange={(e) => setGroup(e.target.value)} style={{ width: 160 }} />
      <button onClick={() => onSave(tag.id, name, group || undefined)}>Save</button>
      <button onClick={() => onDelete(tag.id)} style={{ marginLeft: 6 }}>Delete</button>
    </div>
  );
}
