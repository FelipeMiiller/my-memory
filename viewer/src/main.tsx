import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './i18n'; // side-effect: initializes i18next + react-i18next
import './styles/globals.css';

const rootElement = document.getElementById('root');
if (!rootElement) {
  throw new Error('Root element #root not found in viewer/index.html');
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
