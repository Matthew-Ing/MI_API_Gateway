import { useState } from "react";

type Health = { name: string; url: string; ok: boolean; breaker: string };

export default function App() {
  const [password, setPassword] = useState("admin");
  const [token, setToken] = useState(() => localStorage.getItem("adminToken") ?? "");
  const [key, setKey] = useState("");
  const [hash, setHash] = useState("");
  const [health, setHealth] = useState<Health[]>([]);
  const [err, setErr] = useState("");

  async function api(path: string, init: RequestInit = {}) {
    const res = await fetch("/api" + path, {
      ...init,
      headers: {
        ...(init.body ? { "Content-Type": "application/json" } : {}),
        ...(token ? { Authorization: "Bearer " + token } : {}),
        ...(init.headers ?? {}),
      },
    });
    const text = await res.text();
    if (!res.ok) throw new Error(text.trim() || res.statusText);
    return text ? JSON.parse(text) : null;
  }

  async function login() {
    setErr("");
    try {
      const data = await api("/admin/login", {
        method: "POST",
        body: JSON.stringify({ password }),
      });
      localStorage.setItem("adminToken", data.token);
      setToken(data.token);
    } catch (e) {
      setErr(String(e));
    }
  }

  async function createKey() {
    setErr("");
    try {
      const data = await api("/admin/keys", { method: "POST" });
      setKey(data.key);
      setHash(data.hash);
    } catch (e) {
      setErr(String(e));
    }
  }

  async function revoke() {
    setErr("");
    try {
      await api("/admin/keys/" + hash, { method: "DELETE" });
      setKey("");
      setHash("");
    } catch (e) {
      setErr(String(e));
    }
  }

  async function loadHealth() {
    setErr("");
    try {
      setHealth(await api("/admin/health"));
    } catch (e) {
      setErr(String(e));
    }
  }

  return (
    <main style={{ fontFamily: "sans-serif", maxWidth: 640, margin: "2rem auto" }}>
      <h1>Gateway admin</h1>
      {err && <p style={{ color: "crimson" }}>{err}</p>}

      <section>
        <h2>Login</h2>
        <input value={password} onChange={(e) => setPassword(e.target.value)} />
        <button onClick={login}>Login</button>
        <p>{token ? "signed in" : "not signed in"}</p>
      </section>

      <section>
        <h2>API keys</h2>
        <button onClick={createKey} disabled={!token}>Create key</button>
        <button onClick={revoke} disabled={!hash}>Revoke last</button>
        {key && (
          <p>
            Send as <code>X-API-Key</code>: <code>{key}</code>
            <br />
            hash: <code>{hash}</code>
          </p>
        )}
      </section>

      <section>
        <h2>Upstream health</h2>
        <button onClick={loadHealth} disabled={!token}>Refresh</button>
        <ul>
          {health.map((u) => (
            <li key={u.name}>
              {u.name}: {u.ok ? "up" : "down"} · breaker {u.breaker}
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}