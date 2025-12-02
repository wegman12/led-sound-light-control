# LED Sound Light Control - Development Guidelines

## Project Overview
This is a React + TypeScript + Vite web application for managing LED lights and sound/light behaviors. The application uses Material-UI for styling and components.

## Core Development Principles

### 1. Minimize Code
- Write the least amount of code necessary to fulfill each request
- Avoid over-engineering solutions
- Prefer simple, straightforward implementations over complex abstractions
- Don't add features that weren't explicitly requested

### 2. Avoid Code Duplication
- Extract common logic into reusable utilities or hooks
- Create shared components for repeated UI patterns
- Use composition over duplication
- Consolidate similar functionality rather than copying code

### 3. Application Structure: Pages and Routing
- Application structure is defined by pages and routing
- Each page represents a distinct route/view in the application
- Pages should be placed in `src/pages/` directory
- Use React Router (or similar) to handle navigation between pages
- Pages orchestrate components but contain minimal logic themselves

### 4. Page Behavior: Components
- Page behavior and UI logic should be handled by components
- Components should be modular, reusable, and focused on specific functionality
- Place components in `src/components/` directory
- Components should be self-contained with clear props interfaces
- Prefer small, focused components over large monolithic ones

## Project Structure
```
src/
├── pages/          # Application pages/routes
├── components/     # Reusable UI components
├── hooks/          # Custom React hooks
├── utils/          # Utility functions
├── types/          # TypeScript type definitions
└── services/       # API calls and external service integrations
```

## Technology Stack
- React 19
- TypeScript
- Vite (build tool)
- Material-UI (UI framework)
- Vitest (testing)
- React Testing Library

## Development Workflow
- Keep components focused and single-purpose
- Write tests for critical functionality
- Use TypeScript strictly - avoid `any` types
- Follow Material-UI patterns and conventions
- Ensure responsive design principles
