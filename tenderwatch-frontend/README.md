# TenderWatch Dashboard

A modern, real-time React frontend for TinyMuscle — the intelligent procurement tender monitoring system. TenderWatch provides a beautiful dashboard to monitor, filter, and search government procurement tenders across multiple portals with AI-powered relevance scoring.

## Features

### Real-Time Dashboard

- **Live Statistics** - Total tenders, new opportunities, and updated tenders counter
- **Live Feed Integration** - Server-Sent Events (SSE) for instant tender updates
- **Connection Status** - Visual indicator for SSE connection health
- **Toast Notifications** - Non-intrusive alerts for new and updated tenders

### Advanced Search & Filtering

- **Full-text Search** - Search tenders by title, reference number, or issuing entity
- **Status Filters** - Filter by All, New, or Updated tenders
- **Live Mode** - Quick toggle to view only the latest 20 tenders
- **Smart Sorting** - Most recent tenders displayed first

### Portal Management

- **Add Portals** - Register new procurement portals with an intuitive form
- **AI Relevance Configuration** - Optional business profile and relevance threshold settings
- **Portal Directory** - View all active portals with crawl schedules
- **Quick Delete** - Remove portals you no longer monitor

### Rich Tender Cards

Each tender card displays:

- Title and issuing entity
- Deadline with visual urgency indicators (⚠️ Soon)
- Estimated value (when available)
- AI relevance score (when filtering enabled)
- Direct link to source tender
- Reference number, version, and last update timestamp

## Tech Stack

- **React 18** - Modern UI framework with hooks
- **Vite** - Lightning-fast build tool and dev server
- **Tailwind CSS** - Utility-first CSS framework
- **Lucide React** - Beautiful, consistent icon set
- **date-fns** - Lightweight date formatting and manipulation
- **Axios** - Promise-based HTTP client
- **react-hot-toast** - Non-intrusive toast notifications
- **Server-Sent Events (SSE)** - Real-time server push for live updates

## Quick Start

### Prerequisites

- **Node.js 18+** - [Download](https://nodejs.org/)
- **Backend running** - TinyMuscle backend on `http://localhost:8080`

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

The development server starts on `http://localhost:5173` with hot module reloading (HMR) enabled.

### Build for Production

```bash
npm run build
```

Output is generated in the `dist/` directory.

### Preview Production Build

```bash
npm run preview
```

## Configuration

### API Base URL

By default, the frontend connects to `http://localhost:8080`. To change this:

**Option 1: Environment Variable**

```bash
export VITE_API_URL=http://your-backend-url:8080
npm run dev
```

**Option 2: Edit `src/services/api.js`**

```javascript
const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:8080";
```

### Tailwind Customization

Edit `tailwind.config.js` to customize colors, spacing, and other design tokens:

```javascript
theme: {
  extend: {
    colors: {
      primary: {
        50: '#eff6ff',
        // ... custom color palette
      }
    }
  }
}
```

## Project Structure

```
src/
├── components/
│   ├── PortalManager.jsx     # Portal registration and management
│   └── TenderCard.jsx         # Individual tender card display
├── hooks/
│   └── useSSE.js              # Real-time event streaming hook
├── services/
│   └── api.js                 # Backend API client
├── App.jsx                    # Main application component
├── App.css                    # App-level styles
├── main.jsx                   # React entry point
├── index.css                  # Global styles (Tailwind)
└── assets/                    # Static images and icons

public/                        # Static files (favicon, etc.)
├── vite.svg
├── react.svg
etc.

.eslintrc.cjs                  # ESLint configuration
vite.config.js                 # Vite configuration
tailwind.config.js             # Tailwind CSS configuration
postcss.config.js              # PostCSS configuration
index.html                     # HTML template
package.json                   # Dependencies and scripts
```

## Components

### App.jsx

Main dashboard component that:

- Fetches and displays tender statistics
- Manages search, filter, and live mode state
- Creates portal manager and tender cards
- Handles SSE events for real-time updates
- Displays loading states and error messages

### PortalManager.jsx

Portal management interface with:

- Form to register new procurement portals
- Field validation (ID, URL, Goal)
- Advanced AI relevance filtering options
- Portal list display with delete functionality
- Automatic portal list refresh after changes

### TenderCard.jsx

Individual tender display showing:

- Title and issuing entity
- Deadline with urgency warning (< 7 days)
- Estimated value
- AI relevance score (color-coded)
- Direct link to source tender
- Reference number and version metadata
- Last updated timestamp

### useSSE.js Hook

Custom React hook for Server-Sent Events:

- Establishes SSE connection to `/events` endpoint
- Handles reconnection on connection loss
- Parses incoming JSON events
- Manages connection state and error messages
- Automatic retry with 5-second backoff

## API Integration

The frontend connects to the TinyMuscle backend API:

### Available Endpoints

```javascript
// Portal Management
GET    /portals                 # List all portals
POST   /portals                 # Create new portal
DELETE /portals/{id}            # Delete portal

// Tender Queries
GET    /tenders                 # List all tenders
GET    /tenders/{portalId}      # List tenders by portal

// Real-time Events
GET    /events                  # SSE stream of tender events
```

See [backend README](../README.md#api-reference) for detailed API documentation.

## Performance

- **Code Splitting** - Vite automatically optimizes chunks for production
- **CSS Optimization** - Tailwind CSS purges unused styles (production only)
- **Image Optimization** - SVG assets optimized by Vite
- **Bundle Size** - ~150KB gzipped (including dependencies)

## Development Guidelines

### Adding New Components

```javascript
// src/components/NewComponent.jsx
import { SomeIcon } from "lucide-react";

export function NewComponent({ prop1, prop2 }) {
  return (
    <div className="card p-4">
      <SomeIcon className="w-4 h-4" />
      {/* Component JSX */}
    </div>
  );
}
```

### Adding New Styles

Add Tailwind-based styles in `src/index.css`:

```css
@layer components {
  .my-custom-class {
    @apply bg-blue-50 p-4 rounded-lg border border-blue-200;
  }
}
```

### Making API Calls

Use the `api` service from `src/services/api.js`:

```javascript
import { api } from "../services/api";

// In your component:
const data = await api.getTenders();
const portals = await api.getPortals();
```

## Troubleshooting

### "Failed to load tenders"

- Verify backend is running on `http://localhost:8080`
- Check browser console for CORS errors
- Ensure firewall allows localhost connections

### "SSE Connection Lost"

- Check backend is serving on the correct port
- Verify `/events` endpoint is properly configured
- Check for proxy/firewall blocking SSE

### Hot Module Reload (HMR) Not Working

- Ensure you're accessing via `http://localhost:5173`, not `127.0.0.1`
- Check Vite config `server.host` setting

### Build Errors

- Clear node_modules: `rm -rf node_modules && npm install`
- Check Node.js version: `node --version` (should be 18+)
- Review build logs: `npm run build -- --debug`

## Contributing

To add features or fix bugs:

1. Create a new branch: `git checkout -b feature/my-feature`
2. Make changes and test: `npm run dev`
3. Build for production: `npm run build`
4. Commit and push: `git commit -am "Add my feature" && git push`

## Scripts

```bash
npm run dev         # Start development server with HMR
npm run build       # Build for production
npm run preview     # Preview production build locally
npm run lint        # Run ESLint
npm run lint:fix    # Fix ESLint issues
```
