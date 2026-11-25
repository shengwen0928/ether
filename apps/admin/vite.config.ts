import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// ETHER Admin — Phase 01 skeleton only. No API calls, no proxy, no env vars.
// Dev port 5174: pinned only so web (Vite default 5173) and admin can run
// side by side locally. This is a Phase 01 local convention, pending
// coordinator / CI confirmation; it is not a production deployment decision.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174,
  },
});
