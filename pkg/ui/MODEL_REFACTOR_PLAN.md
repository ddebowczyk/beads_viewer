# Model.go Refactoring Plan

## Current State
- **Total lines**: 4,724
- **Total methods**: 48 methods on Model
- **Total functions**: 10 standalone functions
- **Problem**: Single massive file handling all UI concerns

## Proposed File Structure

### 1. `model.go` (Core - keep ~500 lines)
**Purpose**: Core Model type definition and lifecycle methods

**Contents**:
- Type definitions: `Model`, `focus`, message types
- Constants (focus states, thresholds)
- `NewModel()` constructor
- `Init()` - initialization
- `Update()` - main event loop (delegates to handlers)
- `View()` - main view orchestration (delegates to renderers)
- `Stop()` - cleanup
- Helper commands: `WatchFileCmd`, `CheckUpdateCmd`, `LoadHistoryCmd`, etc.

**Rationale**: Keep the core BubbleTea interface implementation together.

---

### 2. `model_keys.go` (Keyboard Handlers - ~800 lines)
**Purpose**: All keyboard input handling

**Contents**:
- `handleBoardKeys()`
- `handleGraphKeys()`
- `handleActionableKeys()`
- `handleHistoryKeys()`
- `handleRecipePickerKeys()`
- `handleRepoPickerKeys()`
- `handleLabelPickerKeys()`
- `handleInsightsKeys()`
- `handleListKeys()` ← **FIX GOES HERE**
- `handleTimeTravelInputKeys()`
- `handleHelpKeys()`
- `handleSprintKeys()` (if exists)

**Rationale**: All keyboard handling in one place. Our bug fix (focus management) goes here.

---

### 3. `model_render.go` (Main Rendering - ~1000 lines)
**Purpose**: Primary view rendering and layout

**Contents**:
- `renderQuitConfirm()`
- `renderListWithHeader()`
- `renderSplitView()` ← main split view layout
- `renderFooter()`
- `renderHelpOverlay()`
- `renderTimeTravelPrompt()`
- `renderAlertsPanel()`

**Rationale**: Core UI rendering separate from specialized views.

---

### 4. `model_render_label.go` (Label Views - ~800 lines)
**Purpose**: Label-specific rendering (already a distinct feature area)

**Contents**:
- `renderLabelHealthDetail()`
- `renderLabelDrilldown()`
- `renderLabelGraphAnalysis()`
- `getCrossFlowsForLabel()`
- `filterIssuesByLabel()`

**Rationale**: Label dashboard is a feature-complete subsystem.

---

### 5. `model_render_history.go` (History Views - ~400 lines)
**Purpose**: History and correlation views

**Contents**:
- `renderBeadHistoryMD()`
- `enterHistoryView()`
- Related history rendering helpers

**Rationale**: History correlation is another distinct feature.

---

### 6. `model_filters.go` (Filtering & Recipes - ~400 lines)
**Purpose**: Issue filtering, recipes, and list manipulation

**Contents**:
- `applyFilter()`
- `applyRecipe()`
- `SetFilter()`
- `FilteredIssues()`
- `updateSemanticIDs()`

**Rationale**: Data transformation layer separate from rendering.

---

### 7. `model_timetravel.go` (Time Travel - ~300 lines)
**Purpose**: Snapshot comparison feature

**Contents**:
- `enterTimeTravelMode()`
- `exitTimeTravelMode()`
- `IsTimeTravelMode()`
- `TimeTravelDiff()`
- `rebuildListWithDiffInfo()`
- `getDiffStatus()`

**Rationale**: Time travel is a self-contained feature.

---

### 8. `model_workspace.go` (Workspace Mode - ~200 lines)
**Purpose**: Multi-repo workspace functionality

**Contents**:
- `EnableWorkspaceMode()`
- `IsWorkspaceMode()`
- Workspace-related helpers

**Rationale**: Workspace mode is optional functionality.

---

### 9. `model_viewport.go` (Detail Panel - ~200 lines)
**Purpose**: Detail view/viewport management

**Contents**:
- `updateViewportContent()` ← **CRITICAL FOR FIX**
- Viewport-related helpers

**Rationale**: Detail panel logic isolated for maintainability.

---

### 10. `model_export.go` (Export Features - ~200 lines)
**Purpose**: Export, clipboard, external editor

**Contents**:
- `exportToMarkdown()`
- `generateExportFilename()`
- `copyIssueToClipboard()`
- `openInEditor()`

**Rationale**: I/O operations separate from core UI.

---

### 11. `model_alerts.go` (Alerts & Drift - ~200 lines)
**Purpose**: Drift detection and alerts

**Contents**:
- `clearAttentionOverlay()`
- `computeAlerts()` (standalone function)
- `alertKey()` (standalone function)

**Rationale**: Feature flag for future removal if needed.

---

## Migration Strategy

### Phase 1: Extract Keyboard Handlers (PRIORITY - needed for fix)
1. Create `model_keys.go`
2. Move all `handle*Keys()` methods
3. **Apply focus management fix** in `handleListKeys()` and other handlers
4. Run tests: `go test ./pkg/ui/...`

### Phase 2: Extract Renderers
1. Create `model_render.go`, `model_render_label.go`, `model_render_history.go`
2. Move rendering methods
3. Run tests

### Phase 3: Extract Feature Modules
1. Create `model_timetravel.go`, `model_workspace.go`, `model_viewport.go`
2. Move feature-specific methods
3. Run tests

### Phase 4: Extract Utilities
1. Create `model_filters.go`, `model_export.go`, `model_alerts.go`
2. Move utility methods
3. Run tests

### Phase 5: Cleanup Core
1. Verify `model.go` only contains core lifecycle
2. Add package documentation
3. Update CLAUDE.md with new structure

---

## Build Integration

**No changes needed!** Go's package system handles multiple files automatically:
- All files in `pkg/ui/*.go` are part of the same package
- Methods defined in any file are accessible on the type
- Private functions/types remain private to the package

**Only requirement**: All files must have `package ui` at the top.

---

## Testing Strategy

```bash
# After each phase:
go test ./pkg/ui/...

# Verify UI still works:
go build ./cmd/bv
./bv

# Check no broken references:
go build ./...
```

---

## Benefits

1. **Maintainability**: Find code by feature, not by scrolling
2. **Parallel work**: Multiple devs can work on different files
3. **Testing**: Easier to write focused unit tests
4. **Review**: Smaller diffs, easier reviews
5. **IDE performance**: Faster autocomplete, navigation
6. **Cognitive load**: Understand one concern at a time

---

## File Size Targets (After Split)

```
model.go              ~500 lines  (core lifecycle)
model_keys.go         ~800 lines  (keyboard handlers) ← FIX HERE
model_render.go       ~1000 lines (main rendering)
model_render_label.go ~800 lines  (label views)
model_render_history.go ~400 lines (history views)
model_filters.go      ~400 lines  (filtering/recipes)
model_timetravel.go   ~300 lines  (snapshot comparison)
model_workspace.go    ~200 lines  (multi-repo)
model_viewport.go     ~200 lines  (detail panel) ← FIX RELATED
model_export.go       ~200 lines  (I/O operations)
model_alerts.go       ~200 lines  (drift detection)
```

**Total**: Same ~4,700 lines, but organized into logical modules

---

## Current Bug Fix Context

**Bug**: Arrow keys/PgUp/PgDown move list cursor when detail view is open (non-split mode)

**Root cause**: `m.showDetails = true` without `m.focused = focusDetail`

**Files affected by fix**:
- `model_keys.go` (after refactor): All `handle*Keys()` methods
- `model_viewport.go` (after refactor): `updateViewportContent()`

**Fix locations** (6 places to add `m.focused = focusDetail`):
- Line 1873: `handleBoardKeys` - enter key
- Line 1914: `handleGraphKeys` - enter key
- Line 1944: `handleActionableKeys` - enter key
- Line 1982: `handleHistoryKeys` - enter key
- Line 2150: `handleInsightsKeys` - enter key
- Line 2163: `handleListKeys` - enter key

**Fix locations** (2 places to add `m.focused = focusList`):
- Line 1343: q key to close detail view
- Line 1365: esc key to close detail view

---

## Recommendation

**Option A - Fix then refactor**:
1. Apply fix to current `model.go`
2. Test and commit fix
3. Refactor in separate PR

**Option B - Refactor then fix**:
1. Extract `model_keys.go` first (Phase 1)
2. Apply fix to new file
3. Test both refactor and fix together

**Suggested**: Option A (fix first) for faster bug resolution.
