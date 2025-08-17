# AI Government Consultant Frontend

This is the frontend application for the AI Government Consultant system, built with Next.js 14, TypeScript, and Shadcn UI.

## 🚀 Tech Stack

- **Framework**: Next.js 14 with App Router
- **Language**: TypeScript (strict mode)
- **Styling**: Tailwind CSS
- **UI Components**: Shadcn UI
- **State Management**: Zustand
- **Data Fetching**: TanStack Query (React Query)
- **Forms**: React Hook Form with Zod validation
- **Testing**: Jest + React Testing Library
- **Code Quality**: ESLint + Prettier

## 📁 Project Structure

```
src/
├── app/                 # Next.js App Router pages
├── components/          # Reusable UI components
│   └── ui/             # Shadcn UI components
├── hooks/              # Custom React hooks
├── stores/             # Zustand state stores
├── types/              # TypeScript type definitions
└── lib/                # Utility functions and constants
```

## 🛠️ Development

### Prerequisites

- Node.js 18+ 
- npm or yarn

### Getting Started

1. Install dependencies:
   ```bash
   npm install
   ```

2. Start the development server:
   ```bash
   npm run dev
   ```

3. Open [http://localhost:3000](http://localhost:3000) in your browser.

### Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run start` - Start production server
- `npm run lint` - Run ESLint
- `npm run lint:fix` - Fix ESLint issues
- `npm run format` - Format code with Prettier
- `npm run format:check` - Check code formatting
- `npm run test` - Run tests
- `npm run test:watch` - Run tests in watch mode
- `npm run test:coverage` - Run tests with coverage
- `npm run type-check` - Run TypeScript type checking

## 🧪 Testing

The project is configured with Jest and React Testing Library for comprehensive testing:

```bash
# Run all tests
npm run test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage
```

## 🎨 UI Components

This project uses Shadcn UI components. To add new components:

```bash
npx shadcn@latest add [component-name]
```

## 📋 Next Steps

This is Task 1 implementation from the project roadmap. The following tasks will build upon this foundation:

- [ ] 2. Implement authentication system and route protection
- [ ] 3. Build core layout components and navigation
- [ ] 4. Create state management with Zustand and TanStack Query
- [ ] 5. Build API client and backend integration

## 🔧 Configuration

### TypeScript

The project uses strict TypeScript configuration with additional strict checks:
- `noUnusedLocals`
- `noUnusedParameters`
- `exactOptionalPropertyTypes`
- `noImplicitReturns`
- `noFallthroughCasesInSwitch`

### ESLint & Prettier

Code quality is enforced through ESLint and Prettier with consistent formatting rules.

## 🏗️ Build

To create a production build:

```bash
npm run build
```

The build will be optimized and ready for deployment.