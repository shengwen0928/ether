import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './styles/tokens.css';

const rootElement = document.getElementById('root');

// TODO (Phase 01 gap, pending CI): no runtime fallback is invented here.
// If `#root` is missing, this throws rather than silently rendering nothing.
if (rootElement === null) {
  throw new Error('Missing #root element in index.html.');
}

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
