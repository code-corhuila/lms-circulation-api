#!/usr/bin/env node
// Stress test for circulation-api (11-quality/qa-promotion-standard.md, Stage 2/3).
// No external deps on purpose — this repo's CI has no Node runtime to install one for.
//
// Usage:
//   node scripts/stress-test.mjs <url> [concurrency] [durationSeconds]
//   node scripts/stress-test.mjs http://localhost:8099/health 50 15

const url = process.argv[2];
const concurrency = Number(process.argv[3] ?? 20);
const durationSeconds = Number(process.argv[4] ?? 10);

if (!url) {
  console.error('usage: node scripts/stress-test.mjs <url> [concurrency] [durationSeconds]');
  process.exit(1);
}

const latencies = [];
const statusCounts = new Map();
let errors = 0;
let inFlight = 0;

const deadline = Date.now() + durationSeconds * 1000;

async function hit() {
  const start = performance.now();
  try {
    const res = await fetch(url);
    await res.arrayBuffer(); // drain the body, same cost a real client pays
    latencies.push(performance.now() - start);
    statusCounts.set(res.status, (statusCounts.get(res.status) ?? 0) + 1);
  } catch {
    errors++;
    latencies.push(performance.now() - start);
  }
}

async function worker() {
  while (Date.now() < deadline) {
    await hit();
  }
}

console.log(`Stress test: ${concurrency} concurrent workers hitting ${url} for ${durationSeconds}s...`);
const startedAt = Date.now();
inFlight = concurrency;
await Promise.all(Array.from({ length: concurrency }, worker));
const elapsedSeconds = (Date.now() - startedAt) / 1000;

latencies.sort((a, b) => a - b);
const total = latencies.length;
const pct = (p) => latencies[Math.min(total - 1, Math.floor((p / 100) * total))];

console.log('');
console.log(`Total requests : ${total}`);
console.log(`Throughput     : ${(total / elapsedSeconds).toFixed(1)} req/s`);
console.log(`Errors         : ${errors} (${((errors / total) * 100).toFixed(2)}%)`);
console.log(`Status codes   : ${JSON.stringify(Object.fromEntries(statusCounts))}`);
console.log(`Latency p50    : ${pct(50).toFixed(1)} ms`);
console.log(`Latency p95    : ${pct(95).toFixed(1)} ms`);
console.log(`Latency p99    : ${pct(99).toFixed(1)} ms`);
console.log(`Latency max    : ${latencies[total - 1].toFixed(1)} ms`);

// Non-zero exit when the service degrades under load — a CI/manual gate, not
// just a report nobody reads. 1% error budget, 500ms p95: adjust per the
// team's own SLO once one is written down (none exists yet, same gap
// testing-strategy.md's "Performance Tests" section declares).
const errorRate = errors / total;
const p95 = pct(95);
if (errorRate > 0.01 || p95 > 500) {
  console.error(`\nFAILED: error rate ${(errorRate * 100).toFixed(2)}% (budget 1%) or p95 ${p95.toFixed(1)}ms (budget 500ms) exceeded`);
  process.exit(1);
}
console.log('\nPASSED');
