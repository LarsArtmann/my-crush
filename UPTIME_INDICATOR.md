# Uptime Indicator Feature

## Overview
Displays the app's running time in the sidebar with a simple clock icon (●) and time format (e.g., "2h 15m").

## Implementation Details

### Components
1. **`event.AppStartTime()`** - Returns the actual app initialization timestamp
   - Called by `event.AppInitialized()` in `cmd/root.go`
   - Ensures uptime tracking starts at the correct moment

2. **`sidebar.startTime`** - Stores the app start time
   - Initialized in `sidebar.New()` using `event.AppStartTime()`
   - Used by `uptimeBlock()` to calculate uptime

3. **`FormatUptime(duration)`** - Formats duration for display
   - Returns `"Xm"` for < 1 hour
   - Returns `"Xh"` for exact hours (e.g., "1h")
   - Returns `"Xh Ym"` for hours + minutes (e.g., "2h 15m")
   - Hides "0m" when unnecessary for cleaner display

4. **`UptimeTickMsg`** - Bubble Tea message for timer updates
   - Struct: `{Time time.Time}`
   - Triggers uptime display refresh

5. **Timer** - Periodic updates
   - Uses `tea.Tick(1*time.Minute)` in `Init()`
   - Updates uptime display every minute

### Position in UI
```
┌─────────────────────────┐
│ ◇ Model Name         │
│   Thinking on         │
│ ● 2h 15m            │  ← Uptime indicator
│                      │
│ 📁 Files             │
└─────────────────────────┘
```

## Files Modified
- `internal/tui/components/chat/sidebar/sidebar.go` - Main implementation
- `internal/tui/components/chat/sidebar/sidebar_test.go` - BDD tests
- `internal/tui/styles/icons.go` - Added ClockIcon constant
- `internal/event/all.go` - Added AppStartTime() getter

## Test Coverage
- ✅ Timer initialization
- ✅ Message handling (UptimeTickMsg)
- ✅ FormatUptime with 17 test cases
- ⏭ Integration tests (skipped - require full config)

## Future Enhancements
1. Config option to enable/disable
2. Persist uptime to session data
3. Better clock icon (🕐 or similar)
4. Tooltip showing exact start time
5. Adaptive update frequency
6. Golden file tests for visual regression

## Design Decisions
- **1-minute tick interval**: Efficient (vs per-second) while providing good UX
- **Clock icon (●)**: Simple and unobtrusive
- **Muted styling**: Subtle presence in UI
- **AppStartTime vs component creation**: Ensures accuracy

Related: #1876
