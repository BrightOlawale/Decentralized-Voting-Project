#!/usr/bin/env node

// End-to-end API test (requires server running and blockchain configured)
// Usage:
//   BASE_URL=http://localhost:8080 \
//   ADMIN_TOKEN=eyJ... \
//   node scripts/api-e2e-test.js
//
// Optional:
//   CANDIDATES="CANDIDATE_001,CANDIDATE_002,CANDIDATE_003"

const fetch = (...args) =>
  import("node-fetch").then(({ default: fetch }) => fetch(...args));

const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:8080";
const TOKEN = process.env.ADMIN_TOKEN || "";
const AUTH = TOKEN ? { Authorization: `Bearer ${TOKEN}` } : {};

const CANDIDATES = (
  process.env.CANDIDATES || "CANDIDATE_001,CANDIDATE_002"
).split(",");
const TERMINAL_ID = "TEST_TERMINAL_1";
const TERMINAL_ADDR =
  process.env.TERMINAL_ADDR || "0x0000000000000000000000000000000000000001";
const POLLING_UNIT_ID = "TEST_PU1";

// Per-run uniqueness to avoid DB uniqueness collisions across runs
const RUN_ID = Math.floor(Math.random() * 1e9);
const RUN_NIN_BASE = 10000000000 + Math.floor(Math.random() * 8_000_000_000);
const FP_PREFIX = `FP-${RUN_ID}`;
const PER_VOTE_DELAY_MS = Number(process.env.E2E_DELAY_MS || 300);

const c = {
  reset: "\x1b[0m",
  bold: "\x1b[1m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  cyan: "\x1b[36m",
  gray: "\x1b[90m",
};
const step = (t) => console.log(`\n${c.cyan}${c.bold}▶ ${t}${c.reset}`);
const ok = (t) => console.log(`${c.green}✓ ${t}${c.reset}`);
const warn = (t) => console.log(`${c.yellow}! ${t}${c.reset}`);

async function http(method, path, body, headers = {}) {
  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers: { "Content-Type": "application/json", ...AUTH, ...headers },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let json;
  try {
    json = JSON.parse(text);
  } catch {
    json = { raw: text };
  }
  if (!res.ok) {
    throw new Error(`${method} ${path} -> ${res.status}: ${text}`);
  }
  return json;
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
async function httpRetry(method, path, body, tries = 5) {
  for (let a = 0; a < tries; a++) {
    try {
      return await http(method, path, body);
    } catch (e) {
      if (
        !String(e.message).includes(" 429: ") &&
        !String(e.message).includes("gapped-nonce tx") || a === tries - 1
      ) {
        throw e;
      }
      await sleep(200 * Math.pow(2, a)); // backoff
    }
  }
}

async function endActiveElectionIfAny() {
  try {
    const cur = await http("GET", "/api/v1/public/election/current");
    const activeId = cur?.data?.id || cur?.data?.election_id || cur?.data?.ID;
    if (activeId && String(activeId) !== "0") {
      await http("POST", `/api/v1/admin/elections/${activeId}/end`, {});
      ok(`Ended active election: ${activeId}`);
    }
  } catch (_) {
    // ignore
  }
}

async function endAllActiveElections() {
  // End elections until none remains active, with safety cap
  const maxAttempts = 5;
  for (let i = 0; i < maxAttempts; i++) {
    try {
      const cur = await http("GET", "/api/v1/public/election/current");
      const activeId = cur?.data?.id || cur?.data?.election_id || cur?.data?.ID;
      if (!activeId || String(activeId) === "0") break;
      await httpRetry("POST", `/api/v1/admin/elections/${activeId}/end`, {});
      ok(`Ended active election: ${activeId}`);
      await sleep(2000); // allow chain to finalize
    } catch (e) {
      if (String(e.message).includes(" 404: ")) break;
      throw e;
    }
  }
}

async function ensurePollingUnit() {
  try {
    await httpRetry("POST", "/api/v1/admin/system/polling-unit", {
      id: POLLING_UNIT_ID,
      name: "Test PU",
      location: "Localhost",
      total_voters: 1000,
    });
    ok("Polling unit registered");
  } catch (e) {
    warn(`polling unit warning: ${e.message}`);
  }
}

function trySqliteCleanup() {
  console.log("Cleaning up SQLite database");
  // Terminal table
  try {
    execSync('sqlite3 ./voting.db "DELETE FROM terminals;"', {
      stdio: "ignore",
    });
  } catch (_) {}
  // Election table
  try {
    execSync('sqlite3 ./voting.db "DELETE FROM elections;"', {
      stdio: "ignore",
    });
  } catch (_) {}
  try {
    execSync('sqlite3 ./voting.db "DELETE FROM votes;"', { stdio: "ignore" });
  } catch (_) {}
}

(async () => {
  try {
    if (process.env.E2E_CLEANUP === "1") {
      step("Cleanup environment");
      await endActiveElectionIfAny();
      trySqliteCleanup();
      ok("Cleanup done");
    }

    step("Ensure polling unit exists on-chain");
    await ensurePollingUnit();

    step("End any active on-chain election");
    await endAllActiveElections();

    step("Register terminal (and authorize on-chain)");
    await httpRetry("POST", "/api/v1/terminal/register", {
      terminal_id: TERMINAL_ID,
      name: "Terminal 1",
      location: "Localhost",
      polling_unit_id: POLLING_UNIT_ID,
      address: TERMINAL_ADDR,
      authorize: true,
    });
    ok("Terminal registered");

    step("Create election (starts in ~20s)");
    const now = Math.floor(Date.now() / 1000);
    const createResp = await http("POST", "/api/v1/admin/elections/", {
      name: "API E2E Election",
      description: "End-to-end API test",
      start_time: now + 20,
      end_time: now + 3600,
      candidates: CANDIDATES,
    });
    ok(`Created election: ${JSON.stringify(createResp.data)}`);

    const totalId = createResp.data?.blockchain_id;

    step("Register candidates not already present");
    const det = await http("GET", `/api/v1/public/election/${totalId}`);
    const existing = (det.data && det.data.candidates) || [];
    const toAdd = CANDIDATES.filter((c) => c && !existing.includes(c));
    if (toAdd.length) {
      await http("POST", `/api/v1/admin/elections/${totalId}/candidates`, {
        candidates: toAdd,
      });
      ok(`Registered new candidates: ${toAdd.join(", ")}`);
    } else {
      ok("All candidates already present, skipping");
    }

    step("Wait for start time and start election");
    const details = await http("GET", `/api/v1/public/election/${totalId}`);
    const startTs = Number(
      details.data?.start_time || details.data?.startTime || now + 20
    );
    const waitMs = Math.max(
      0,
      (startTs - Math.floor(Date.now() / 1000) + 1) * 1000
    );
    if (waitMs > 0) await sleep(waitMs);
    await http("POST", `/api/v1/admin/elections/${totalId}/start`, {});
    ok("Election started");

    step("Cast 20 randomized votes via API");
    const totalVotes = 20;
    const tally = Object.fromEntries(CANDIDATES.map((c) => [c, 0]));
    for (let i = 0; i < totalVotes; i++) {
      const candidate =
        CANDIDATES[Math.floor(Math.random() * CANDIDATES.length)];
      const nin = String(RUN_NIN_BASE + i); // unique 11-digit NIN per run
      const fp = `${FP_PREFIX}-${i}`; // unique fingerprint per run

      // Register a voter quickly (idempotent if you wire checks)
      try {
        await httpRetry("POST", "/api/v1/public/voter/register", {
          nin,
          first_name: "Test",
          last_name: `Voter${i}`,
          date_of_birth: new Date("1990-01-01T00:00:00Z").toISOString(),
          gender: "M",
          polling_unit_id: POLLING_UNIT_ID,
          fingerprint_data: fp,
        });
      } catch (e) {
        warn(`voter register warning: ${e.message}`);
      }

      await httpRetry("POST", "/api/v1/voting/cast", {
        nin,
        polling_unit_id: POLLING_UNIT_ID,
        candidate_id: candidate,
        encrypted_vote: `cipher-${i}`,
        fingerprint_data: fp,
      });
      tally[candidate]++;
      if ((i + 1) % 10 === 0)
        console.log(
          `${c.gray}Progress${c.reset} ${i + 1}/${totalVotes}`,
          tally
        );
      if ((i + 1) % 10 === 0)
        await sleep(Number(process.env.E2E_COOLDOWN_MS || 2000));
      await sleep(PER_VOTE_DELAY_MS); // per-cast delay
    }
    ok("Voting complete");

    step("Fetch results");
    const results = await http(
      "GET",
      `/api/v1/public/election/${totalId}/results`
    );
    console.log("Results:", JSON.stringify(results.data, null, 2));

    console.log(`\n${c.bold}${c.green}✅ API E2E test completed.${c.reset}`);
  } catch (err) {
    console.error(`${c.yellow}Test failed:${c.reset}`, err.message);
    process.exit(1);
  }
})();
