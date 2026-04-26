# Frontend - DevCost AI

Next.js dashboard application for cloud cost optimization and waste detection.

## Technology Stack

- **Framework**: Next.js 14 with React 18 & TypeScript
- **Styling**: Tailwind CSS
- **HTTP Client**: Fetch API
- **Features**: 4 main pages with minimal, production-ready UI

## 🎯 Features

### Dashboard (`/`)
- Total AWS cost display
- Waste percentage indicator
- Total resources monitored
- Monthly/annual savings projections
- Quick action links

### Waste Detection (`/waste`)
- List of detected waste resources
- Resource details (type, ID, name)
- Waste reason explanation
- Severity levels (low, medium, high, critical)
- Estimated monthly savings per resource
- Summary cards (total items, critical/high count, total potential savings)

### Recommendations (`/recommendations`)
- Cost optimization recommendations
- Priority levels (low, medium, high, critical)
- Status tracking (active, pending, completed)
- Estimated savings per recommendation
- Risk assessment
- Summary cards (total, critical/high priority, total savings)

### Actions (`/actions`)
- Executed automation actions
- Action status tracking (success, failed, pending, in_progress)
- Execution timestamps and duration
- Error messages for failed actions
- Summary cards (total, successful, failed)

## Development Setup

```bash
# Install dependencies
npm install

# Set up environment
cp .env.example .env.local

# Start development server
npm run dev

# Build for production
npm run build

# Start production server
npm start
```

## Environment Variables

```env
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

## Project Structure

```
src/
├── app/
│   ├── layout.tsx                    # Root layout
│   ├── globals.css                   # Global Tailwind styles
│   └── (dashboard)/
│       ├── layout.tsx                # Dashboard layout with navigation
│       ├── page.tsx                  # Dashboard page
│       ├── waste/
│       │   └── page.tsx             # Waste detection page
│       ├── recommendations/
│       │   └── page.tsx             # Recommendations page
│       └── actions/
│           └── page.tsx             # Actions page
└── lib/
    └── api.ts                        # API client & type definitions
```

## API Integration

The frontend connects to a backend API at `NEXT_PUBLIC_API_URL/v1` with the following endpoints:

- `GET /waste` - Fetch waste resources
- `GET /recommendations` - Fetch recommendations
- `GET /actions` - Fetch executed actions
- `POST /actions/execute` - Execute recommendations

## UI Components

- **Card**: Metric cards with title, value, and optional subtext
- **Table**: Responsive data tables with hover effects
- **Badge**: Status/severity indicators with color coding
- **Navigation**: Top header with active page highlighting
- **Loading States**: Skeleton loading indicators
- **Error States**: Error message displays

## Performance

- Minimal bundle size (Next.js optimizations)
- Single API call aggregation (dashboard combines multiple endpoints)
- Responsive tables with overflow handling
- No external component libraries (pure Tailwind)

## Production Checklist

- ✅ TypeScript strict mode enabled
- ✅ ESLint configured
- ✅ Tailwind CSS production optimization
- ✅ Error handling for API calls
- ✅ Loading states for all data fetches
- ✅ Responsive design (mobile-first)
- ✅ Accessibility considerations (semantic HTML, ARIA labels)

## Running with Docker

The frontend is included in the main docker-compose setup:

```bash
docker-compose up
# Frontend runs on http://localhost:3000
# Backend API runs on http://localhost:8080
```
