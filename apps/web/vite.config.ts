import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// ETHER Web — Phase 01 skeleton only. No API calls, no proxy, no env vars.
// Dev port: Vite default (5173). No port is pinned here; if the coordinator
// later runs web and admin together, port coordination is a TODO for CI docs.
export default defineConfig({
  plugins: [react()],
});
