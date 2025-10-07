# CSS Migration Plan: From Inline to Component Classes

## Overview
This document outlines the migration strategy from inline Tailwind CSS classes to a maintainable, component-based styling system for the Audius Explorer console.

## Phase 1: Component Class Definition

### Core Components to Create

#### 1. Cards & Containers
```css
/* Primary card used for main content sections */
.card-primary {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6;
}

/* Secondary card with border */
.card-secondary {
  @apply bg-white dark:bg-gray-900 rounded-lg shadow-xl border border-gray-200 dark:border-gray-700;
}

/* Overlay card for map overlays */
.card-overlay {
  @apply bg-white/90 dark:bg-gray-800/90 backdrop-blur-md rounded-lg border border-white/20 dark:border-gray-700/50 shadow-lg;
}
```

#### 2. Typography
```css
/* Page titles */
.text-title {
  @apply text-2xl font-light text-gray-900 dark:text-gray-100;
}

/* Section headings */
.text-heading {
  @apply text-xl font-bold text-gray-900 dark:text-gray-100;
}

/* Labels */
.text-label {
  @apply text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide;
}

/* Large values (stats, metrics) */
.text-value-lg {
  @apply text-3xl font-bold text-gray-900 dark:text-gray-100;
}

/* Extra large values */
.text-value-xl {
  @apply text-4xl font-light text-gray-900 dark:text-gray-100;
}

/* Small descriptive text */
.text-description {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

/* Monospace (addresses, hashes) */
.text-mono {
  @apply font-mono text-sm;
}
```

#### 3. Links & Buttons
```css
/* Primary link */
.link-primary {
  @apply text-purple-600 dark:text-purple-400 hover:text-purple-800 dark:hover:text-purple-300 hover:underline;
}

/* Navigation button */
.btn-nav {
  @apply px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300
         bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600
         rounded transition-colors;
}

/* Disabled button */
.btn-disabled {
  @apply px-4 py-2 text-sm font-medium text-gray-400 dark:text-gray-600
         bg-gray-50 dark:bg-gray-700 rounded cursor-not-allowed;
}
```

#### 4. Status Indicators
```css
/* Success status */
.status-success {
  @apply text-green-600 dark:text-green-400;
}

/* Warning status */
.status-warning {
  @apply text-yellow-600 dark:text-yellow-400;
}

/* Error status */
.status-error {
  @apply text-red-600 dark:text-red-400;
}

/* Info status */
.status-info {
  @apply text-blue-600 dark:text-blue-400;
}

/* Status dot */
.status-dot {
  @apply w-2 h-2 rounded-full;
}

.status-dot-success {
  @apply status-dot bg-green-500;
}

.status-dot-warning {
  @apply status-dot bg-yellow-500 animate-pulse;
}
```

#### 5. Layout Helpers
```css
/* Section divider */
.divider-horizontal {
  @apply border-t border-gray-200 dark:border-gray-700;
}

.divider-vertical {
  @apply border-l border-gray-200 dark:border-gray-700;
}

/* Standard section spacing */
.section-spacing {
  @apply space-y-6;
}

/* Grid layouts */
.grid-dashboard {
  @apply grid grid-cols-1 lg:grid-cols-2 gap-6;
}

/* Flex utilities */
.flex-between {
  @apply flex items-center justify-between;
}

.flex-center {
  @apply flex items-center justify-center;
}
```

#### 6. Interactive Elements
```css
/* Hoverable row */
.row-interactive {
  @apply py-3 px-4 border border-gray-200 dark:border-gray-700 rounded
         hover:bg-purple-50 hover:border-purple-200 dark:hover:bg-gray-700
         transition-colors cursor-pointer;
}

/* Transaction type badge */
.badge {
  @apply text-xs px-2 py-1 rounded;
}

.badge-default {
  @apply badge bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400;
}
```

## Phase 2: Theme Configuration

### Create a CSS Variables System
```css
:root {
  /* Colors */
  --color-primary: theme('colors.purple.600');
  --color-primary-hover: theme('colors.purple.800');
  --color-bg-primary: theme('colors.white');
  --color-bg-secondary: theme('colors.gray.100');

  /* Spacing */
  --spacing-card: theme('spacing.6');
  --spacing-section: theme('spacing.6');

  /* Borders */
  --border-radius: theme('borderRadius.lg');
  --border-color: theme('colors.gray.200');
}

.dark {
  --color-bg-primary: theme('colors.gray.800');
  --color-bg-secondary: theme('colors.gray.900');
  --color-primary: theme('colors.purple.400');
  --color-primary-hover: theme('colors.purple.300');
  --border-color: theme('colors.gray.700');
}
```

## Phase 3: Migration Strategy

### Step-by-Step Process

1. **Update input.css** with component classes
2. **Create a utilities file** for common patterns
3. **Migrate templates progressively**:
   - Start with smaller components (buttons, badges)
   - Move to layout components (cards, sections)
   - Finally update complex components (charts, maps)

### Example Migration

**Before:**
```html
<div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6">
  <div class="flex justify-between items-center mb-6">
    <h2 class="text-2xl font-light text-gray-900 dark:text-gray-100">Latest Blocks</h2>
  </div>
</div>
```

**After:**
```html
<div class="card-primary">
  <div class="flex-between mb-6">
    <h2 class="text-title">Latest Blocks</h2>
  </div>
</div>
```

## Phase 4: Implementation Order

1. **Week 1**: Set up component classes in input.css
2. **Week 2**: Migrate common components (cards, typography, buttons)
3. **Week 3**: Migrate dashboard.templ as proof of concept
4. **Week 4**: Migrate remaining templates
5. **Week 5**: Documentation and testing

## Benefits

1. **Maintainability**: Single source of truth for component styles
2. **Consistency**: Enforced design system across all pages
3. **Figma Alignment**: Easy to map Figma components to CSS classes
4. **Performance**: Smaller HTML files with reusable classes
5. **Developer Experience**: Cleaner templates, easier to read and modify

## Figma Integration

### Mapping Figma Components to CSS Classes

| Figma Component | CSS Class | Usage |
|-----------------|-----------|--------|
| Card/Primary | .card-primary | Main content sections |
| Card/Secondary | .card-secondary | Subsections with borders |
| Typography/Heading | .text-heading | Section headings |
| Typography/Label | .text-label | Form labels, meta info |
| Button/Primary | .btn-primary | Main actions |
| Status/Success | .status-success | Positive indicators |

## Testing Strategy

1. **Visual Regression Testing**: Screenshot comparison before/after
2. **Component Testing**: Verify all components render correctly
3. **Dark Mode Testing**: Ensure all themes work properly
4. **Cross-browser Testing**: Test on Chrome, Firefox, Safari

## Rollback Plan

If issues arise:
1. Keep original inline classes commented
2. Use feature flags to toggle between old/new styles
3. Gradual rollout by page/section

## Success Metrics

- [ ] 70% reduction in HTML file size
- [ ] 90% of styles using component classes
- [ ] Zero visual regressions
- [ ] Improved lighthouse scores
- [ ] Faster development of new features

## Next Steps

1. Review this plan with the team
2. Create a proof of concept with one page
3. Get design approval from Figma team
4. Begin implementation following the phases above