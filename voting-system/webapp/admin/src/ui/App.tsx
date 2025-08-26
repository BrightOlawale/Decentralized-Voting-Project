import React, { useState, useEffect, useMemo } from 'react';
import type { CSSProperties } from 'react';
import { api, BASE_URL_DEFAULT, toEpochSeconds, formatTs } from './api';
import { Json } from './types';

// Replace static styles with theme-aware styles and toggle
type Theme = 'light' | 'dark';
const THEME_KEY = 'admin_theme';
const detectInitialTheme = (): Theme => {
  try {
    const saved = localStorage.getItem(THEME_KEY);
    if (saved === 'light' || saved === 'dark') return saved as Theme;
  } catch (_) {}
  if (typeof window !== 'undefined' && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }
  return 'light';
};

const makeStyles = (theme: Theme): Record<string, CSSProperties> => {
  const isDark = theme === 'dark';
  const colors = {
    bg: isDark ? '#0f172a' : '#f5f7fb',
    text: isDark ? '#e5e7eb' : '#111827',
    card: isDark ? '#111827' : '#ffffff',
    border: isDark ? '#1f2937' : '#e5e7eb',
    muted: isDark ? '#9ca3af' : '#6b7280',
    primary: '#2563eb',
    primaryText: '#ffffff',
    secondary: isDark ? '#475569' : '#6c757d',
    secondaryText: '#ffffff',
    danger: '#dc2626',
    dangerText: '#ffffff',
    inputBg: isDark ? '#0b1220' : '#ffffff',
    inputText: isDark ? '#e5e7eb' : '#111827',
    overlayBg: isDark ? 'rgba(2,6,23,0.65)' : 'rgba(15,23,42,0.35)'
  };
  return {
    page: { padding: 24, maxWidth: 1200, margin: '0 auto', background: colors.bg, color: colors.text, minHeight: '100vh' },
    card: { padding: 24, border: `1px solid ${colors.border}`, borderRadius: 12, marginBottom: 24, background: colors.card, boxShadow: isDark ? '0 2px 10px rgba(0,0,0,0.3)' : '0 2px 12px rgba(16,24,40,0.06)' },
    h1: { fontSize: 24, marginBottom: 4 },
    h2: { fontSize: 20, marginBottom: 12 },
    muted: { color: colors.muted },
    grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 24 },
    section: { border: `1px solid ${colors.border}`, borderRadius: 12, padding: 24, background: colors.card },
    label: { display: 'block', marginBottom: 4 },
    input: { width: '100%', padding: 10, marginBottom: 12, border: `1px solid ${colors.border}`, borderRadius: 8, background: colors.inputBg, color: colors.inputText, outline: 'none' },
    textarea: { width: '100%', padding: 10, marginBottom: 12, border: `1px solid ${colors.border}`, borderRadius: 8, height: 120, background: colors.inputBg, color: colors.inputText, outline: 'none' },
    button: { padding: '8px 12px', background: colors.primary, color: colors.primaryText, border: 'none', borderRadius: 8, cursor: 'pointer' },
    buttonSecondary: { padding: '8px 12px', background: colors.secondary, color: colors.secondaryText, border: 'none', borderRadius: 8, cursor: 'pointer' },
    buttonDanger: { padding: '8px 12px', background: colors.danger, color: colors.dangerText, border: 'none', borderRadius: 8, cursor: 'pointer' },
    smallButton: { padding: '4px 8px', background: colors.primary, color: colors.primaryText, border: 'none', borderRadius: 6, cursor: 'pointer', marginRight: 4 },
    smallDanger: { padding: '4px 8px', background: colors.danger, color: colors.dangerText, border: 'none', borderRadius: 6, cursor: 'pointer', marginRight: 4 },
    row: { display: 'flex', gap: 8 },
    headerRow: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 },
    table: { width: '100%', borderCollapse: 'collapse', color: colors.text },
    th: { padding: 8, textAlign: 'left', borderBottom: `1px solid ${colors.border}` },
    td: { padding: 8, borderBottom: `1px solid ${colors.border}` },
    pre: { background: isDark ? '#0b1220' : '#f8f9fa', padding: 12, borderRadius: 8, overflowX: 'auto', border: `1px solid ${colors.border}` },
    notice: { padding: 12, background: isDark ? '#0b1220' : '#f8f9fa', borderRadius: 8, marginTop: 12, border: `1px solid ${colors.border}` },
    footer: { textAlign: 'center', color: colors.muted, marginTop: 24 },
    themeToggle: { padding: '6px 10px', background: 'transparent', color: colors.text, border: `1px solid ${colors.border}`, borderRadius: 20, cursor: 'pointer' },
    overlay: { position: 'fixed', inset: 0, background: colors.overlayBg, display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 },
    overlayBox: { background: colors.card, color: colors.text, padding: 16, borderRadius: 12, border: `1px solid ${colors.border}`, minWidth: 220, textAlign: 'center', boxShadow: isDark ? '0 8px 30px rgba(0,0,0,0.55)' : '0 8px 24px rgba(16,24,40,0.14)' },
    spinner: { width: 28, height: 28, border: `3px solid ${colors.border}`, borderTopColor: colors.primary, borderRadius: '50%', margin: '0 auto 12px', animation: 'spin 1s linear infinite' }
  };
};

export const App: React.FC = () => {
  const [theme, setTheme] = useState<Theme>(detectInitialTheme());
  const styles = useMemo(() => makeStyles(theme), [theme]);
  const [baseUrl, setBaseUrl] = useState(BASE_URL_DEFAULT);
  const [token, setToken] = useState('');
  const [notice, setNotice] = useState<string>('');

  const [elections, setElections] = useState<any[]>([]);
  const [loadingElections, setLoadingElections] = useState<boolean>(false);
  const [results, setResults] = useState<Json | null>(null);
  const [puInfo, setPuInfo] = useState<any | null>(null);

  // Global progress overlay
  const [busy, setBusy] = useState(false);
  const [busyMsg, setBusyMsg] = useState('');
  const withProgress = async (msg: string, fn: () => Promise<any> | void) => {
    setBusyMsg(msg);
    setBusy(true);
    try { await Promise.resolve(fn()); }
    finally { setBusy(false); setBusyMsg(''); }
  };

  useEffect(() => {
    try { localStorage.setItem(THEME_KEY, theme); } catch (_) {}
    if (typeof document !== 'undefined') {
      (document.body.style as any).background = styles.page.background as string;
      (document.body.style as any).color = styles.page.color as string;
    }
  }, [theme, styles]);

  const apiUrl = useMemo(() => (p: string) => `${baseUrl.replace(/\/$/, '')}${p}`, [baseUrl]);

  const withNotice = async (fn: () => Promise<any> | void) => {
    setNotice('');
    try { await Promise.resolve(fn()); setNotice('Done'); } catch (e: any) { setNotice(e?.message || String(e)); }
  };

  async function refreshElections() {
    setLoadingElections(true);
    setResults(null);
    try {
      // Prefer active on-chain election
      try {
        const cur: any = await api(apiUrl('/api/v1/public/election/current'), { method:'GET' }, undefined);
        const d = cur?.data || cur;
        if (d) {
          const chainId = String(d.id || d.ID || d.election_id || '');
          // Try to map to DB record to get DB id
          let dbId: any = '-';
          try {
            const list: any = await api(apiUrl('/api/v1/admin/elections/?limit=10'), { method:'GET' }, token);
            const arr: any[] = Array.isArray(list?.data) ? list.data : [];
            const match = arr.find((x:any)=> String(x?.blockchain_id||'') === chainId);
            if (match) dbId = match.id;
          } catch(_) {}
          setElections([{
            id: dbId,
            blockchain_id: chainId,
            name: d.name,
            start_time: d.start_time || d.startTime,
            end_time: d.end_time || d.endTime,
            is_active: (d.is_active ?? d.isActive) ?? false,
          }]);
          return;
        }
      } catch (e: any) {
        // If 404 (no active), fall back to latest DB election only
      }
      const list: any = await api(apiUrl('/api/v1/admin/elections/?limit=1'), { method:'GET' }, token);
      setElections(Array.isArray(list?.data) ? list.data : []);
    } finally {
      setLoadingElections(false);
    }
  }

  useEffect(() => {
    // auto-load on baseUrl set
    if (baseUrl) refreshElections().catch(()=>{});
  }, [baseUrl]);

  return (
    <div style={styles.page}>
      {/* simple keyframes for spinner */}
      <style>{`@keyframes spin { from { transform: rotate(0deg) } to { transform: rotate(360deg) } }`}</style>

      {busy && (
        <div style={styles.overlay as any}>
          <div style={styles.overlayBox}>
            <div style={styles.spinner as any} />
            <div>{busyMsg || 'Working...'}</div>
          </div>
        </div>
      )}

      <div style={styles.card}>
        <div style={styles.headerRow}>
          <div />
          <button style={styles.themeToggle} onClick={()=>setTheme(theme === 'dark' ? 'light' : 'dark')}>
            {theme === 'dark' ? '☀️ Light' : '🌙 Dark'}
          </button>
        </div>
        <h1 style={styles.h1}>Blockchain Voting System Dashboard</h1>
        <p style={styles.muted}>Control panel for blockchain voting.</p>

        <div style={styles.grid}>
          <section style={styles.section}>
            <h2 style={styles.h2}>Server</h2>
            <label style={styles.label}>Base URL</label>
            <input style={styles.input} placeholder="http://localhost:8080" value={baseUrl} onChange={e=>setBaseUrl(e.target.value)} />

            <label style={styles.label}>Admin Token (optional)</label>
            <input style={styles.input} placeholder="paste JWT" value={token} onChange={e=>setToken(e.target.value)} />

            <div style={styles.row}>
              <button style={styles.button} disabled={busy} onClick={()=>setNotice('Configured')}>Save</button>
              <button style={styles.buttonSecondary} disabled={busy} onClick={()=>withProgress('Loading elections...', refreshElections)}>Reload Elections</button>
            </div>
          </section>

          <section style={styles.section}>
            <h2 style={styles.h2}>Create Election</h2>
            <input id="e-name" style={styles.input} placeholder="Election name" />
            <input id="e-desc" style={styles.input} placeholder="Description" />
            <label style={styles.label}>Start</label>
            <input id="e-start" type="datetime-local" style={styles.input} />
            <label style={styles.label}>End</label>
            <input id="e-end" type="datetime-local" style={styles.input} />
            <label style={styles.label}>Candidates (one per line)</label>
            <textarea id="e-cands" style={styles.textarea} placeholder="CANDIDATE_001
CANDIDATE_002"></textarea>
            <button style={styles.button} disabled={busy} onClick={()=>withProgress('Creating election...', async()=>{
              const name = (document.getElementById('e-name') as HTMLInputElement).value;
              const description = (document.getElementById('e-desc') as HTMLInputElement).value;
              const start = toEpochSeconds((document.getElementById('e-start') as HTMLInputElement).value);
              const end = toEpochSeconds((document.getElementById('e-end') as HTMLInputElement).value);
              const raw = (document.getElementById('e-cands') as HTMLTextAreaElement).value;
              const candidates = raw.split(/\r?\n|,/).map(s=>s.trim()).filter(Boolean);
              await api(apiUrl('/api/v1/admin/elections/'), { method:'POST', body: JSON.stringify({ name, description, start_time: start, end_time: end, candidates }) }, token);
              await refreshElections();
              setNotice('Election created');
            })}>Create</button>
          </section>

          <section style={styles.section}>
            <h2 style={styles.h2}>Polling Unit</h2>
            <input id="pu-id" style={styles.input} placeholder="PU ID" />
            <input id="pu-name" style={styles.input} placeholder="Name" />
            <input id="pu-loc" style={styles.input} placeholder="Location" />
            <input id="pu-total" style={styles.input} placeholder="Total Voters" />
            <button style={styles.button} disabled={busy} onClick={()=>withProgress('Ensuring polling unit...', async()=>{
              const id = (document.getElementById('pu-id') as HTMLInputElement).value;
              const name = (document.getElementById('pu-name') as HTMLInputElement).value;
              const location = (document.getElementById('pu-loc') as HTMLInputElement).value;
              const total = Number((document.getElementById('pu-total') as HTMLInputElement).value || '0');
              await api(apiUrl('/api/v1/admin/system/polling-unit'), { method:'POST', body: JSON.stringify({ id, name, location, total_voters: total }) }, token);
              const info:any = await api(apiUrl(`/api/v1/public/polling-unit/${encodeURIComponent(id)}`), { method:'GET' }, undefined);
              setPuInfo(info?.data || info);
              setNotice('Polling unit ensured');
            })}>Ensure/Register PU</button>
            {puInfo && (
              <div style={styles.muted}>
                PU: {puInfo.id || ''} • {puInfo.name || ''} • {puInfo.location || ''} • Voters: {puInfo.total_voters || '0'} • Recorded: {puInfo.votes_recorded || '0'}
              </div>
            )}
          </section>

          <section style={styles.section}>
            <h2 style={styles.h2}>Candidates</h2>
            <input id="c-eid" style={styles.input} placeholder="Election blockchain ID" />
            <textarea id="c-names" style={styles.textarea} placeholder="One candidate per line"></textarea>
            <button style={styles.button} disabled={busy} onClick={()=>withProgress('Registering candidates...', async()=>{
              const id = (document.getElementById('c-eid') as HTMLInputElement).value;
              const raw = (document.getElementById('c-names') as HTMLTextAreaElement).value;
              const names = raw.split('\n').map(s=>s.trim()).filter(Boolean);
              await api(apiUrl(`/api/v1/admin/elections/${id}/candidates`), { method:'POST', body: JSON.stringify({ candidates: names }) }, token);
              setNotice('Candidates registered');
            })}>Register</button>
          </section>

          <section style={{...styles.section, gridColumn: '1 / -1'}}>
            <div style={styles.headerRow}>
              <h2 style={styles.h2}>Active Election</h2>
              <div style={styles.row}>
                <button style={styles.buttonSecondary} disabled={busy} onClick={()=>withProgress('Refreshing...', async()=>{ await refreshElections(); })}>Refresh</button>
                <button style={styles.buttonDanger} disabled={busy} onClick={()=>withProgress('Deleting elections...', async()=>{
                  for (const e of elections) {
                    if (e?.id && e.id !== '-') {
                      await api(apiUrl('/api/v1/admin/elections/'), { method:'DELETE', body: JSON.stringify({ id: e.id }) }, token);
                    }
                  }
                  await refreshElections();
                })}>Delete All</button>
              </div>
            </div>
            <div style={styles.muted}>This view prefers the current on-chain election; if none active, it shows the latest DB election.</div>

            {loadingElections ? <div style={styles.muted}>Loading...</div> : (
              <div style={{overflowX:'auto'}}>
                <table style={styles.table as any}>
                  <thead>
                    <tr>
                      <th style={styles.th}>ID (DB)</th>
                      <th style={styles.th}>Blockchain ID</th>
                      <th style={styles.th}>Name</th>
                      <th style={styles.th}>Start</th>
                      <th style={styles.th}>End</th>
                      <th style={styles.th}>Active</th>
                      <th style={styles.th}>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {elections.map((e:any)=> (
                      <tr key={`${e.blockchain_id}-${e.id}`}>
                        <td style={styles.td}>{e.id}</td>
                        <td style={styles.td}>{e.blockchain_id || '-'}</td>
                        <td style={styles.td}>{e.name}</td>
                        <td style={styles.td}>{formatTs(e.start_time)}</td>
                        <td style={styles.td}>{formatTs(e.end_time)}</td>
                        <td style={styles.td}>{String(e.is_active)}</td>
                        <td style={{...styles.td}}>
                          <div style={styles.row}>
                            <button style={styles.smallButton} disabled={busy} onClick={()=>withProgress('Starting election...', async()=>{
                              const id = e.blockchain_id || e.id;
                              await api(apiUrl(`/api/v1/admin/elections/${id}/start`), { method:'POST' }, token);
                              await refreshElections();
                            })}>Start</button>
                            <button style={styles.smallButton} disabled={busy} onClick={()=>withProgress('Ending election...', async()=>{
                              const id = e.blockchain_id || e.id;
                              await api(apiUrl(`/api/v1/admin/elections/${id}/end`), { method:'POST' }, token);
                              await refreshElections();
                            })}>End</button>
                            <button style={styles.smallDanger} disabled={busy} onClick={()=>withProgress('Deleting election...', async()=>{
                              if (e.id && e.id !== '-') {
                                await api(apiUrl('/api/v1/admin/elections/'), { method:'DELETE', body: JSON.stringify({ id: e.id }) }, token);
                                await refreshElections();
                              }
                            })}>Delete</button>
                            <button style={styles.smallButton} disabled={busy} onClick={()=>withProgress('Fetching results...', async()=>{
                              const id = e.blockchain_id || e.id;
                              const j:any = await api(apiUrl(`/api/v1/public/election/${id}/results`), { method:'GET' }, undefined);
                              setResults(j?.data || j);
                            })}>Results</button>
                            <button style={styles.smallButton} disabled={busy} onClick={()=>withProgress('Fetching details...', async()=>{
                              const id = e.blockchain_id || e.id;
                              await api(apiUrl(`/api/v1/public/election/${id}`), { method:'GET' }, undefined);
                            })}>Details</button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {results && (
              <div style={{marginTop:12}}>
                <h3 style={styles.h2}>Results</h3>
                <pre style={styles.pre}>{JSON.stringify(results, null, 2)}</pre>
              </div>
            )}
          </section>

          <section style={styles.section}>
            <h2 style={styles.h2}>Status</h2>
            <button style={styles.button} disabled={busy} onClick={()=>withProgress('Pinging server...', async()=>{
              await api(apiUrl('/api/v1/public/status'), { method:'GET' }, undefined);
            })}>Ping</button>
            <input id="s-eid" style={styles.input} placeholder="Election blockchain ID" />
            <div style={styles.row}>
              <button style={styles.button} disabled={busy} onClick={()=>withProgress('Loading details...', async()=>{
                const id = (document.getElementById('s-eid') as HTMLInputElement).value;
                await api(apiUrl(`/api/v1/public/election/${id}`), { method:'GET' }, undefined);
              })}>Details</button>
              <button style={styles.button} disabled={busy} onClick={()=>withProgress('Loading results...', async()=>{
                const id = (document.getElementById('s-eid') as HTMLInputElement).value;
                const j:any = await api(apiUrl(`/api/v1/public/election/${id}/results`), { method:'GET' }, undefined);
                setResults(j?.data || j);
              })}>Results</button>
              <button style={styles.buttonSecondary} disabled={busy} onClick={()=>withProgress('Loading current...', async()=>{
                const j:any = await api(apiUrl('/api/v1/public/election/current'), { method:'GET' }, undefined);
                setResults(j?.data || j);
              })}>Current</button>
            </div>
          </section>
        </div>

        {notice && <div style={styles.notice}>{notice}</div>}
      </div>

      <footer style={styles.footer}>Built by Bright Olawale and Samson Bolarinde (Supervised by Mr. M.A Akingbade) — connects directly to Our Go API⚙️ and our Blockchain⛓️.</footer>
    </div>
  );
};
