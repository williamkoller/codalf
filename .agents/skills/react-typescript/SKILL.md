---
name: react-typescript-review
description: >
  Comprehensive code review checklist for React/TypeScript projects. Evaluates React patterns,
  TypeScript usage, hooks best practices, component design, and state management.
  Use when reviewing React/TypeScript code, PRs, or before merging changes.
  Do NOT use for general JavaScript (use javascript-review) or backend TypeScript (use typescript-backend-review).
---

# React/TypeScript Code Review

Structured code review process for React/TypeScript projects. Reviews should be constructive,
specific, and cite the relevant principle behind each finding.

## Review Process

Execute these steps in order. For each finding, classify severity:
- 🔴 **BLOCKER** — Must fix before merge. Correctness, data loss, security, runtime errors.
- 🟡 **WARNING** — Should fix. Performance, maintainability, React best practices.
- 🟢 **SUGGESTION** — Consider improving. Style, naming, documentation.

## 1. React Hooks

### Rules of Hooks
- ✅ Hooks called only at the top level (not inside loops, conditions, or nested functions)
- ✅ Hooks called from React functions (components or custom hooks)
- ✅ `useEffect` dependencies array is exhaustive (includes all used values)
- ✅ No stale closures in useEffect/useCallback/useMemo

### Performance
- ✅ `useCallback` / `useMemo` used for referential equality in deps arrays
- ✅ Expensive computations wrapped in `useMemo`
- ✅ Event handlers wrapped in `useCallback` when passed to memoized children
- ✅ `React.memo` used for expensive components
- ✅ Lazy initialization for expensive initial state computation

## 2. TypeScript

### Type Safety
- ✅ No implicit `any` — always explicit types or proper inference
- ✅ Proper interface vs type usage (interface for object shapes, type for unions)
- ✅ Strict null checks — no `null` or `undefined` unless explicitly typed
- ✅ Generic types used for reusable components
- ✅ Enum usage considered (prefer const objects or literal types)

### Advanced Types
- ✅ Discriminated unions for type narrowing
- ✅ Proper type guards for runtime checks
- ✅ Utility types used appropriately (Partial, Required, Pick, Omit, etc.)
- ✅ Template literal types for string patterns

## 3. Component Design

### Props
- ✅ Props have explicit types (interface/type)
- ✅ Props are properly typed as optional when needed
- ✅ Component name matches file name
- ✅ Props destructured for readability

### Component Structure
- ✅ Component extracted when > 200 lines
- ✅ Complex logic moved to custom hooks
- ✅ No inline function definitions in JSX (causes re-renders)
- ✅ Conditional rendering handled properly (not in return statement with &&)

## 4. State Management

### Local State
- ✅ `useState` vs `useReducer` chosen appropriately
- ✅ State co-located when possible
- ✅ Form state managed with controlled components

### Global State
- ✅ Context used appropriately (not overused)
- ✅ State minimal — derived state computed, not stored
- ✅ Consider splitting contexts (prevent unnecessary re-renders)

## 5. Effects & Async

### useEffect
- ✅ Cleanup function returned for subscriptions/timers
- ✅ No missing dependencies
- ✅ No unnecessary effects (data fetching in useEffect, not render)
- ✅ Async operations handled properly (no ignored promises)

### Error Handling
- ✅ Error boundaries for component trees
- ✅ Async operations have try/catch or .catch()
- ✅ Loading states shown during async operations

## 6. Security

- ✅ No sensitive data in component state (use proper auth)
- ✅ Input sanitized for XSS prevention
- ✅ No `dangerouslySetInnerHTML` or properly sanitized if used
- ✅ Environment variables properly typed and accessed

## 7. Testing

- ✅ Components have corresponding tests
- ✅ User interactions tested (fireEvent, userEvent)
- ✅ Async operations tested with waitFor
- ✅ Mocks used appropriately
- ✅ Test coverage targets critical paths

## 8. Accessibility (a11y)

- ✅ Semantic HTML elements used
- ✅ ARIA attributes when needed
- ✅ Keyboard navigation tested
- ✅ Focus management for modals/dialogs
- ✅ Color contrast meets WCAG AA

## Review Output Format

```
## React/TypeScript Code Review Summary

**Files reviewed:** <list>
**Overall assessment:** APPROVE | REQUEST CHANGES | COMMENT

### Findings

#### 🔴 BLOCKER: <title>
- **File:** `path/to/file.tsx:42`
- **Issue:** <what is wrong>
- **Why:** <which principle or guideline>
- **Fix:** <concrete suggestion>

#### 🟡 WARNING: <title>
...

#### 🟢 SUGGESTION: <title>
...

### What's Done Well
<genuine positive observations — always include at least one>
```
