# State Management Refactor - Detailed Task List

## Overview
This document contains the detailed task breakdown for implementing the state management refactor described in `STATE_MANAGEMENT_REFACTOR.md`. Tasks are organized by phase and component for Claude Flow orchestration.

---

## Phase 1: Foundation & Event System (Priority: Critical)

### Task 1.1: Create Event Bus System
**File**: `internal/events/bus.go` (new)
**Dependencies**: None
**Estimated Effort**: 2-3 hours

**Subtasks**:
- [ ] Create `internal/events` package
- [ ] Define `EventType` constants (ServerStateChanged, AppStateChanged, etc.)
- [ ] Define `Event` struct with type, data, timestamp
- [ ] Implement `Bus` struct with pub/sub pattern
- [ ] Add `Subscribe(eventType)` method returning buffered channel
- [ ] Add `Publish(event)` method with non-blocking send
- [ ] Add `Unsubscribe(channel)` for cleanup
- [ ] Write unit tests for pub/sub functionality
- [ ] Test concurrent publishing and subscribing
- [ ] Test channel buffer overflow handling

**Acceptance Criteria**:
- Event bus supports multiple subscribers per event type
- Publishing doesn't block even if channels are full
- Unit tests cover concurrent access patterns
- No goroutine leaks when subscribers disconnect

---

### Task 1.2: Add Migration System
**File**: `internal/config/migration.go` (new)
**Dependencies**: None
**Estimated Effort**: 3-4 hours

**Subtasks**:
- [ ] Create `internal/config/migration.go`
- [ ] Define `MigrateConfig(config *Config)` function
- [ ] Implement `determineStartupMode(server)` conversion logic
- [ ] Handle edge cases (nil values, missing fields)
- [ ] Create config backup before migration
- [ ] Add migration logging with details
- [ ] Write tests for all migration paths:
  - [ ] enabled=true, start_on_boot=true → "active"
  - [ ] enabled=true, start_on_boot=false → "lazy_loading"
  - [ ] enabled=false → "disabled"
  - [ ] quarantined=true → "quarantined"
  - [ ] auto_disabled=true → "auto_disabled"
- [ ] Test migration with real config files

**Acceptance Criteria**:
- All old config formats migrate correctly
- Backup created before migration
- Migration is idempotent (running twice is safe)
- Comprehensive test coverage for all scenarios

---

### Task 1.3: Update ServerConfig Schema
**File**: `internal/config/config.go`
**Dependencies**: Task 1.2 (migration)
**Estimated Effort**: 2-3 hours

**Subtasks**:
- [ ] Add `StartupMode string` field to `ServerConfig`
- [ ] Add JSON tags: `json:"startup_mode" mapstructure:"startup_mode"`
- [ ] Mark old fields as deprecated (comments only, keep for backward compat)
- [ ] Update `DefaultConfig()` to use startup_mode="active"
- [ ] Update `Validate()` to check startup_mode values
- [ ] Update `ConvertFromCursorFormat()` to set startup_mode
- [ ] Add validation for startup_mode enum values
- [ ] Write tests for config validation
- [ ] Update example config files in docs/

**Acceptance Criteria**:
- startup_mode field exists with proper JSON/mapstructure tags
- Validation rejects invalid startup_mode values
- Old fields still work (backward compatibility)
- Example configs updated

---

### Task 1.4: Integrate Migration on Config Load
**File**: `internal/config/loader.go`
**Dependencies**: Task 1.2, Task 1.3
**Estimated Effort**: 2 hours

**Subtasks**:
- [ ] Update `Load()` to call `MigrateConfig()` after loading
- [ ] Save migrated config back to disk
- [ ] Log migration events (which servers migrated)
- [ ] Handle migration errors gracefully
- [ ] Add flag to skip migration (for testing)
- [ ] Write integration test for load → migrate → save flow

**Acceptance Criteria**:
- Old configs automatically migrate on first load
- Migrated config saved to disk
- No data loss during migration
- Integration tests verify end-to-end migration

---

## Phase 2: State Management Core (Priority: Critical)

### Task 2.1: Define Server State Enums
**File**: `internal/upstream/types/types.go`
**Dependencies**: None
**Estimated Effort**: 1-2 hours

**Subtasks**:
- [ ] Define `ServerState` type as string
- [ ] Add state constants (StateDisabled, StateConnected, etc.)
- [ ] Implement `IsStable()` method on ServerState
- [ ] Implement `String()` method for logging
- [ ] Add state validation function
- [ ] Write tests for state helper methods
- [ ] Document state transitions in comments

**Acceptance Criteria**:
- All states defined with clear names
- IsStable() correctly identifies stable states
- Unit tests for all helper methods

---

### Task 2.2: Update StateManager with Event Bus
**File**: `internal/upstream/types/types.go`
**Dependencies**: Task 1.1, Task 2.1
**Estimated Effort**: 3-4 hours

**Subtasks**:
- [ ] Add `eventBus *events.Bus` field to StateManager
- [ ] Update constructor to accept event bus
- [ ] Modify `SetState()` to emit ServerStateChanged events
- [ ] Add `GetState()` method with proper locking
- [ ] Add `ResetFailures()` method for clearing auto-disable
- [ ] Update failure tracking to emit events
- [ ] Write tests for state transitions with events
- [ ] Test concurrent state access

**Acceptance Criteria**:
- StateManager emits events on state changes
- Thread-safe state access
- Events include old/new state information
- Tests verify event emission

---

### Task 2.3: Implement Server State Machine
**File**: `internal/upstream/managed/state_machine.go` (new)
**Dependencies**: Task 2.1, Task 2.2
**Estimated Effort**: 4-5 hours

**Subtasks**:
- [ ] Create `internal/upstream/managed/state_machine.go`
- [ ] Define valid state transitions map
- [ ] Implement `TransitionTo(newState)` with validation
- [ ] Add `CanTransitionTo(newState)` checker
- [ ] Implement auto-disable threshold logic
- [ ] Add `handleConnectionFailure()` method
- [ ] Add `persistAutoDisable()` method
- [ ] Implement lazy loading state logic (sleeping)
- [ ] Write comprehensive state transition tests
- [ ] Test invalid transition rejection

**Acceptance Criteria**:
- State machine enforces valid transitions only
- Auto-disable triggers at correct threshold
- State changes persisted to storage
- Comprehensive test coverage for all transitions

---

### Task 2.4: Add Application State Management
**File**: `internal/server/server.go`
**Dependencies**: Task 2.1
**Estimated Effort**: 3-4 hours

**Subtasks**:
- [ ] Define `AppState` type and constants
- [ ] Add `appState AppState` field to Server struct
- [ ] Add `appStateMu sync.RWMutex` for thread safety
- [ ] Implement `GetAppState()` method
- [ ] Implement `setAppState(newState)` with event emission
- [ ] Implement `checkAndUpdateAppState()` logic
- [ ] Add `StopAllServers()` method
- [ ] Add `StartAllServers()` method
- [ ] Call state check after server state changes
- [ ] Write tests for app state transitions

**Acceptance Criteria**:
- App state correctly reflects server states
- Starting → Running when all servers stable
- Stopping → Stopped when all servers stopped
- Thread-safe state access
- Events emitted on app state changes

---

### Task 2.5: Implement App State Machine
**File**: `internal/server/app_state_machine.go` (new)
**Dependencies**: Task 2.4
**Estimated Effort**: 2-3 hours

**Subtasks**:
- [ ] Create `internal/server/app_state_machine.go`
- [ ] Document valid app state transitions
- [ ] Implement stability checking logic
- [ ] Add transition logging
- [ ] Implement `waitForStableState()` helper
- [ ] Add timeout for state transitions
- [ ] Write tests for each transition
- [ ] Test timeout scenarios

**Acceptance Criteria**:
- App state transitions follow defined rules
- Stability checks work correctly
- Timeouts prevent infinite waiting
- Comprehensive test coverage

---

## Phase 3: Storage & Persistence (Priority: High)

### Task 3.1: Add Transaction-Safe Config Updates
**File**: `internal/config/loader.go`
**Dependencies**: Task 1.3
**Estimated Effort**: 3-4 hours

**Subtasks**:
- [ ] Add `mu sync.Mutex` to Loader struct
- [ ] Add `skipNextReload bool` flag
- [ ] Implement `UpdateConfigAtomic(updateFn)` method
- [ ] Use temp file + atomic rename pattern
- [ ] Set skipNextReload flag before rename
- [ ] Handle file watcher skip logic
- [ ] Add rollback on failure
- [ ] Write tests for atomic updates
- [ ] Test concurrent update handling

**Acceptance Criteria**:
- Config updates are atomic (all or nothing)
- File watcher doesn't reload programmatic changes
- No race conditions during updates
- Rollback works on failure

---

### Task 3.2: Update Storage Manager Methods
**File**: `internal/storage/manager.go`
**Dependencies**: Task 3.1, Task 2.3
**Estimated Effort**: 4-5 hours

**Subtasks**:
- [ ] Add `UpdateServerStartupMode(name, mode, reason)` method
- [ ] Implement two-phase commit (BBolt → config file)
- [ ] Add rollback logic on config file failure
- [ ] Implement `ClearAutoDisable(name)` method
- [ ] Update `EnableUpstreamServer()` to clear auto-disable
- [ ] Add `StopServer(name)` method (sets runtime flag)
- [ ] Add `StartServer(name)` method (triggers connect)
- [ ] Update all server CRUD to use startup_mode
- [ ] Write tests for each new method
- [ ] Test rollback scenarios

**Acceptance Criteria**:
- All storage operations are transactional
- Auto-disable can be cleared via storage API
- Rollback works correctly on failures
- Comprehensive test coverage

---

### Task 3.3: Integrate Storage with State Machine
**File**: `internal/upstream/managed/client.go`
**Dependencies**: Task 2.3, Task 3.2
**Estimated Effort**: 3-4 hours

**Subtasks**:
- [ ] Update connection failure handler to use storage API
- [ ] Call `storage.UpdateServerStartupMode()` on auto-disable
- [ ] Ensure persistence before state transition
- [ ] Add error handling for storage failures
- [ ] Update reconnection logic for cleared auto-disable
- [ ] Write integration tests for auto-disable flow
- [ ] Test restart persistence

**Acceptance Criteria**:
- Auto-disable state persists to storage immediately
- State survives application restart
- Storage errors don't break state machine
- Integration tests verify persistence

---

## Phase 4: WebSocket & Real-Time Updates (Priority: Medium)

### Task 4.1: Implement WebSocket Handler
**File**: `internal/server/websocket.go` (new)
**Dependencies**: Task 1.1
**Estimated Effort**: 3-4 hours

**Subtasks**:
- [ ] Create `internal/server/websocket.go`
- [ ] Add gorilla/websocket dependency
- [ ] Implement `handleWebSocket()` endpoint
- [ ] Add connection upgrade logic
- [ ] Subscribe to event bus on connection
- [ ] Forward events to WebSocket clients
- [ ] Handle client disconnections gracefully
- [ ] Add ping/pong for connection health
- [ ] Write tests for WebSocket communication
- [ ] Test multiple concurrent clients

**Acceptance Criteria**:
- WebSocket endpoint accepts connections
- Events forwarded to all connected clients
- Disconnections handled without leaks
- Multiple clients can connect simultaneously

---

### Task 4.2: Add WebSocket Routes
**File**: `internal/server/server.go`
**Dependencies**: Task 4.1
**Estimated Effort**: 1-2 hours

**Subtasks**:
- [ ] Add `GET /ws/events` route
- [ ] Add `GET /ws/servers` route for server events only
- [ ] Add connection tracking
- [ ] Add endpoint to list active connections (debug)
- [ ] Update server shutdown to close WebSocket connections
- [ ] Write integration tests for routes

**Acceptance Criteria**:
- WebSocket routes registered and accessible
- Connections cleaned up on server shutdown
- Integration tests verify routing

---

### Task 4.3: Update Dashboard with WebSocket Client
**File**: `internal/server/dashboard.go` + templates
**Dependencies**: Task 4.2
**Estimated Effort**: 2-3 hours

**Subtasks**:
- [ ] Add WebSocket client JavaScript to dashboard
- [ ] Implement event handler for server state changes
- [ ] Update server cards in real-time (no refresh)
- [ ] Add connection status indicator
- [ ] Handle WebSocket reconnection on disconnect
- [ ] Add visual feedback for state transitions
- [ ] Test real-time updates in browser
- [ ] Test reconnection behavior

**Acceptance Criteria**:
- Dashboard updates in real-time without refresh
- Visual indicators for state changes
- Reconnects automatically on disconnect
- Works in all major browsers

---

## Phase 5: Tray UI Integration (Priority: High)

### Task 5.1: Update Tray State Manager
**File**: `internal/tray/managers.go`
**Dependencies**: Task 1.1, Task 2.4
**Estimated Effort**: 3-4 hours

**Subtasks**:
- [ ] Add `eventChan <-chan events.Event` to ServerStateManager
- [ ] Remove polling ticker (replace with events)
- [ ] Implement `StartEventListener()` method
- [ ] Update menu items on state change events
- [ ] Add app state indicator to tray tooltip
- [ ] Handle event channel closure gracefully
- [ ] Write tests for event handling
- [ ] Test menu update logic

**Acceptance Criteria**:
- Tray no longer polls for state
- Menu items update within 100ms of state change
- App state visible in tray tooltip
- No goroutine leaks

---

### Task 5.2: Add App State Controls to Tray
**File**: `internal/tray/event_handlers.go`
**Dependencies**: Task 2.4, Task 5.1
**Estimated Effort**: 2-3 hours

**Subtasks**:
- [ ] Add "Stop All Servers" menu item
- [ ] Add "Start All Servers" menu item
- [ ] Add app state indicator to main menu
- [ ] Implement click handlers for app controls
- [ ] Disable controls during transitions (Starting/Stopping)
- [ ] Add "Clear Auto-Disable" button per server
- [ ] Update menu icons based on state
- [ ] Write tests for menu interactions

**Acceptance Criteria**:
- Stop All/Start All buttons work correctly
- App state indicator shows current state
- Controls disabled during transitions
- Clear Auto-Disable button appears for auto-disabled servers

---

## Phase 6: Group Operations Fix (Priority: High)

### Task 6.1: Fix Group Enable Operation
**File**: `internal/server/groups_web.go`
**Dependencies**: Task 3.2
**Estimated Effort**: 2-3 hours

**Subtasks**:
- [ ] Update `EnableGroup()` to check for auto-disabled servers
- [ ] Call `storage.ClearAutoDisable()` for each auto-disabled server
- [ ] Set all servers to startup_mode="active"
- [ ] Trigger reconnection for all servers in group
- [ ] Emit ServerGroupUpdated event
- [ ] Add error handling and partial failure recovery
- [ ] Write tests for group enable with mixed states
- [ ] Test auto-disable clearing in groups

**Acceptance Criteria**:
- Group enable clears auto-disable for all servers
- All servers transition to active mode
- Events emitted for group operations
- Comprehensive error handling

---

### Task 6.2: Fix Group Disable Operation
**File**: `internal/server/groups_web.go`
**Dependencies**: Task 3.2
**Estimated Effort**: 1-2 hours

**Subtasks**:
- [ ] Update `DisableGroup()` to set startup_mode="disabled"
- [ ] Stop all running servers in group
- [ ] Emit events for each server
- [ ] Add atomic operation support
- [ ] Write tests for group disable

**Acceptance Criteria**:
- Group disable sets all servers to disabled mode
- Running servers stopped gracefully
- Atomic operation (all or nothing)

---

## Phase 7: Testing & Documentation (Priority: Medium)

### Task 7.1: Write Unit Tests
**Files**: Various `*_test.go` files
**Dependencies**: All implementation tasks
**Estimated Effort**: 5-6 hours

**Subtasks**:
- [ ] Event bus pub/sub tests
- [ ] State machine transition tests
- [ ] Migration logic tests
- [ ] Config validation tests
- [ ] Storage transaction tests
- [ ] App state logic tests
- [ ] Achieve >80% code coverage

**Acceptance Criteria**:
- All critical paths have unit tests
- Code coverage >80%
- Tests run in <10 seconds
- No flaky tests

---

### Task 7.2: Write Integration Tests
**File**: `internal/server/integration_test.go` (new)
**Dependencies**: All implementation tasks
**Estimated Effort**: 4-5 hours

**Subtasks**:
- [ ] Test auto-disable → restart → state persisted
- [ ] Test group enable clearing auto-disable
- [ ] Test WebSocket event delivery
- [ ] Test app state transitions
- [ ] Test file watcher + programmatic updates
- [ ] Test event bus across components

**Acceptance Criteria**:
- End-to-end flows verified
- State persistence tested
- Event propagation verified
- All integration scenarios covered

---

### Task 7.3: Write E2E Tests
**File**: `internal/server/e2e_state_test.go` (new)
**Dependencies**: All implementation tasks
**Estimated Effort**: 3-4 hours

**Subtasks**:
- [ ] Test tray "Stop All" → restart → servers stopped
- [ ] Test auto-disable threshold → clear → reconnect
- [ ] Test group operations via web UI
- [ ] Test migration with real config files
- [ ] Test backward compatibility with old API

**Acceptance Criteria**:
- UI interactions verified
- Real world scenarios tested
- Migration tested with production-like configs
- Backward compatibility verified

---

### Task 7.4: Update Documentation
**Files**: Various docs
**Dependencies**: All implementation tasks
**Estimated Effort**: 2-3 hours

**Subtasks**:
- [ ] Update CLAUDE.md with new state system
- [ ] Add startup_mode documentation
- [ ] Document event bus usage
- [ ] Add WebSocket API documentation
- [ ] Update configuration examples
- [ ] Add troubleshooting guide for state issues
- [ ] Document migration process

**Acceptance Criteria**:
- All new features documented
- Examples provided for common operations
- Troubleshooting guide complete
- Migration guide clear and detailed

---

## Execution Strategy

### Phase Ordering
1. **Phase 1** (Foundation) - Must complete first
2. **Phase 2** (State Management) - Depends on Phase 1
3. **Phase 3** (Storage) - Can run parallel with Phase 4
4. **Phase 4** (WebSocket) - Can run parallel with Phase 3
5. **Phase 5** (Tray UI) - Depends on Phases 2 & 4
6. **Phase 6** (Groups) - Depends on Phase 3
7. **Phase 7** (Testing) - Final phase, depends on all

### Parallel Execution Opportunities
- **Phase 3 + Phase 4** can run in parallel (different components)
- **Task 1.2 + Task 1.3** can run in parallel (different files)
- **Task 4.1 + Task 4.2** can run sequentially but in same session

### Risk Mitigation
- Complete Phase 1 fully before moving on (foundation is critical)
- Test each phase before proceeding
- Keep migration backward compatible throughout
- Feature flag for event-driven mode (can disable if issues)

### Estimated Total Effort
- **Phase 1**: 9-12 hours
- **Phase 2**: 13-18 hours
- **Phase 3**: 10-13 hours
- **Phase 4**: 6-9 hours
- **Phase 5**: 5-7 hours
- **Phase 6**: 3-5 hours
- **Phase 7**: 14-18 hours

**Total**: 60-82 hours (7-10 working days)

### Success Metrics
- ✅ All tests passing (unit + integration + E2E)
- ✅ Zero polling in UI (100% event-driven)
- ✅ Auto-disable persists across restarts
- ✅ Group operations work correctly
- ✅ Backward compatibility maintained
- ✅ No performance regression
- ✅ Documentation complete

---

## Claude Flow Task Format

For Claude Flow orchestration, tasks should be executed in this order with dependencies respected:

```yaml
swarm:
  topology: hierarchical
  strategy: balanced

phases:
  - name: "Foundation"
    tasks: [1.1, 1.2, 1.3, 1.4]
    parallel: false  # Sequential execution required

  - name: "State Core"
    tasks: [2.1, 2.2, 2.3, 2.4, 2.5]
    parallel: false

  - name: "Storage & WebSocket"
    tasks: [3.1, 3.2, 3.3, 4.1, 4.2, 4.3]
    parallel: true  # 3.x and 4.x can run in parallel

  - name: "UI Integration"
    tasks: [5.1, 5.2, 6.1, 6.2]
    parallel: false

  - name: "Testing"
    tasks: [7.1, 7.2, 7.3, 7.4]
    parallel: true  # Tests can run in parallel
```