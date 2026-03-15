---
name: radix-ui
description: Radix UI primitive patterns. Use when building accessible, unstyled UI components like dialogs, dropdowns, tooltips, tabs, and selects. Covers Tailwind styling, keyboard navigation, animations, and portal management.
---

# Radix UI

> **Platform:** Web only. For mobile modals/sheets, see the **expo-sdk** and **react-native-patterns** skills.

> Use Context7 MCP (`resolve-library-id` then `query-docs`) for full API reference, all component props, and additional examples.

## Overview

Unstyled, accessible UI primitives for React with built-in keyboard navigation, focus management, and ARIA attributes. Designed to be composed with Tailwind CSS and Framer Motion.

**Version**: Latest (individual packages) or `radix-ui` unified package

**Install (individual packages)**:
```bash
pnpm add @radix-ui/react-dialog @radix-ui/react-dropdown-menu @radix-ui/react-select @radix-ui/react-tooltip @radix-ui/react-tabs
```

**Install (unified package)**:
```bash
pnpm add radix-ui
```

The unified `radix-ui` package bundles all primitives - use this for simpler dependency management.

## Workflows

**Adding a Dialog:**
1. [ ] Install: `pnpm add @radix-ui/react-dialog`
2. [ ] Import Dialog parts: Root, Trigger, Portal, Overlay, Content
3. [ ] Wrap Overlay and Content in Portal for proper stacking
4. [ ] Style with Tailwind and data-[state=] selectors
5. [ ] Test keyboard navigation (Esc to close, Tab trap)
6. [ ] Add Framer Motion animations if needed

**Adding a Select:**
1. [ ] Install: `pnpm add @radix-ui/react-select`
2. [ ] Import Select parts: Root, Trigger, Portal, Content, Item
3. [ ] Add Icon and Value to Trigger for visual feedback
4. [ ] Style open/closed states with data-[state=open]
5. [ ] Test keyboard (Arrow keys, Enter, Type-ahead)
6. [ ] Ensure proper z-index for Portal

**Adding Tooltips:**
1. [ ] Install: `pnpm add @radix-ui/react-tooltip`
2. [ ] Wrap app with TooltipProvider (once, at root)
3. [ ] Compose Trigger and Content for each tooltip
4. [ ] Set delayDuration for hover timing
5. [ ] Verify screen reader announcements

## Dialog (Modal)

```tsx
import * as Dialog from '@radix-ui/react-dialog';
// OR: import { Dialog } from 'radix-ui';

function ModalExample() {
  const [open, setOpen] = useState(false);

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <button className="px-4 py-2 bg-blue-600 text-white rounded">Open Dialog</button>
      </Dialog.Trigger>

      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />

        <Dialog.Content className="fixed left-[50%] top-[50%] translate-x-[-50%] translate-y-[-50%] max-h-[85vh] w-[90vw] max-w-[500px] rounded-lg bg-white p-6 shadow-lg">
          {/* Title and Description are required for accessibility */}
          <Dialog.Title className="text-lg font-semibold mb-2">Dialog Title</Dialog.Title>
          <Dialog.Description className="text-sm text-gray-600 mb-4">
            Screen readers announce this description.
          </Dialog.Description>

          <div className="flex justify-end gap-2 mt-6">
            <Dialog.Close asChild>
              <button className="px-4 py-2 border rounded">Cancel</button>
            </Dialog.Close>
            <button className="px-4 py-2 bg-blue-600 text-white rounded">Save</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
```

**Framer Motion animations**: Use `forceMount` on Portal and wrap with `AnimatePresence`. Pass `asChild` to Overlay/Content and use `motion.div` as the child.

## Select (Dropdown)

```tsx
import * as Select from '@radix-ui/react-select';
import { ChevronDownIcon, CheckIcon } from '@radix-ui/react-icons';

function SelectExample() {
  return (
    <Select.Root defaultValue="apple">
      <Select.Trigger className="inline-flex items-center justify-between rounded px-4 py-2 text-sm bg-white border gap-2 data-[placeholder]:text-gray-400 min-w-[200px]">
        <Select.Value placeholder="Select a fruit..." />
        <Select.Icon><ChevronDownIcon /></Select.Icon>
      </Select.Trigger>

      <Select.Portal>
        <Select.Content className="overflow-hidden bg-white rounded-md shadow-lg border">
          <Select.Viewport className="p-1">
            <Select.Item value="apple" className="relative flex items-center px-8 py-2 rounded text-sm hover:bg-blue-50 outline-none cursor-pointer data-[disabled]:opacity-50">
              <Select.ItemIndicator className="absolute left-2"><CheckIcon /></Select.ItemIndicator>
              <Select.ItemText>Apple</Select.ItemText>
            </Select.Item>
            <Select.Item value="banana" className="relative flex items-center px-8 py-2 rounded text-sm hover:bg-blue-50 outline-none cursor-pointer">
              <Select.ItemIndicator className="absolute left-2"><CheckIcon /></Select.ItemIndicator>
              <Select.ItemText>Banana</Select.ItemText>
            </Select.Item>
          </Select.Viewport>
        </Select.Content>
      </Select.Portal>
    </Select.Root>
  );
}
```

Use `Select.Group` + `Select.Label` for grouped options, `Select.Separator` for dividers.

## Styling with Tailwind

### Key Data Attribute Selectors

```tsx
// Open/closed state
className="data-[state=open]:bg-blue-50 data-[state=closed]:bg-gray-50"

// Disabled
className="data-[disabled]:opacity-50 data-[disabled]:pointer-events-none"

// Checked (checkboxes, radio items)
className="data-[state=checked]:bg-blue-600"

// Side-based positioning
className="data-[side=top]:animate-slide-down data-[side=bottom]:animate-slide-up"
```

### Common Layout Patterns

```tsx
// Focus ring (required for keyboard UX)
className="outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"

// Centered modal
className="fixed left-[50%] top-[50%] translate-x-[-50%] translate-y-[-50%]"

// Backdrop overlay
className="fixed inset-0 bg-black/50 backdrop-blur-sm"
```

## Accessibility

**Keyboard navigation** (handled automatically):
- **Dialog**: Esc to close, Tab trap inside modal
- **Select**: Arrow keys, Enter to select, type-ahead search
- **DropdownMenu**: Arrow keys, Enter to select
- **Tabs**: Arrow keys, Home/End for first/last
- **Tooltip**: Focus trigger shows tooltip

**ARIA** (added automatically):
- Dialog: `role="dialog"`, `aria-labelledby` (Title), `aria-describedby` (Description)
- Select: `role="combobox"`, `aria-expanded`, `aria-controls`

**Always provide** `Dialog.Title` and `Dialog.Description` — screen readers depend on them.

**Custom focus management:**
```tsx
<Dialog.Content
  onOpenAutoFocus={(e) => { e.preventDefault(); customRef.current?.focus(); }}
  onCloseAutoFocus={(e) => { e.preventDefault(); triggerRef.current?.focus(); }}
>
```

## Controlled vs Uncontrolled

```tsx
// Uncontrolled — component manages its own state
<Dialog.Root defaultOpen={false}>

// Controlled — parent manages state (preferred for complex UIs)
const [open, setOpen] = useState(false);
<Dialog.Root open={open} onOpenChange={setOpen}>
```

## Portal Usage

Always portal overlays and dropdowns to avoid z-index conflicts, `overflow: hidden` clipping, and CSS transform issues:

```tsx
<Dialog.Portal>          {/* renders at document.body */}
  <Dialog.Overlay />
  <Dialog.Content />
</Dialog.Portal>

{/* Custom container */}
<Dialog.Portal container={customContainerRef.current}>
```

## Best Practices

- **Use `asChild` prop** to compose with custom elements without wrapper divs
- **Always Portal** overlays and dropdowns to avoid z-index issues
- **Provide Title and Description** for all Dialogs (accessibility requirement)
- **Use data-[state=] selectors** for open/closed state styling
- **Prefer controlled** for complex state management
- **One TooltipProvider** at app root — not per tooltip
- **Test keyboard navigation** for every interactive component

## Anti-Patterns

- ❌ Forgetting `Dialog.Portal` (causes z-index issues)
- ❌ Missing `Dialog.Title` or `Dialog.Description` (fails a11y)
- ❌ Not using `asChild` with custom triggers (creates wrapper divs)
- ❌ Multiple `TooltipProvider` instances (unnecessary overhead)
- ❌ Using controlled without `onOpenChange`/`onValueChange`
- ❌ Mixing controlled and uncontrolled patterns
- ❌ Forgetting `focus:ring` styles (poor keyboard UX)
- ❌ Not testing with keyboard navigation

## Feedback Loops

**Accessibility testing:**
```bash
# Test with keyboard only (no mouse)
# Tab through all interactive elements
# Esc should close Dialogs, Dropdowns, Selects
# Arrow keys should navigate menus and selects
```

**Screen reader testing:**
```bash
# macOS: VoiceOver (Cmd+F5)
# Verify Dialog.Title and Dialog.Description are announced
# Check for proper role attributes
```

**Visual regression**: Test all states: Closed vs Open, Hover vs Focus, Selected vs Unselected, Disabled, different viewport sizes.

**Framer Motion**: Use `forceMount` + `AnimatePresence`. Test that focus management still works with animations.
